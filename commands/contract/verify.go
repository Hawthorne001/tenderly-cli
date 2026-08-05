package contract

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/tenderly/tenderly-cli/commands"
	"github.com/tenderly/tenderly-cli/config"
	"github.com/tenderly/tenderly-cli/providers"
	"github.com/tenderly/tenderly-cli/rest"
	"github.com/tenderly/tenderly-cli/rest/client"
	"github.com/tenderly/tenderly-cli/rest/etherscan"
	"github.com/tenderly/tenderly-cli/rest/payloads"
	"github.com/tenderly/tenderly-cli/userError"
)

var (
	verifyNetworks string
	verifyMode     string
	verifyRPC      string
	verifyProject  string
	verifyLegacy   bool
)

func init() {
	verifyCmd.PersistentFlags().StringVar(&verifyNetworks, "networks", "", "A comma separated list of networks to verify")
	verifyCmd.PersistentFlags().StringVar(&verifyMode, "mode", "public", "Verification visibility: \"public\" publishes to the shared contract registry, \"private\" keeps it project-scoped.")
	verifyCmd.PersistentFlags().StringVar(&verifyRPC, "rpc", "", "Virtual TestNet RPC URL. Verifies contracts against the TestNet's state instead of a real network; no --mode applies.")
	verifyCmd.PersistentFlags().StringVar(&verifyProject, "project", "", "Project slug (or account/project) to verify into. Defaults to the configured project.")
	verifyCmd.PersistentFlags().BoolVar(&verifyLegacy, "legacy", false, "Use the legacy bulk verification endpoint.")

	ContractsCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verifies all project contracts on Tenderly",
	Run: func(cmd *cobra.Command, args []string) {
		commands.InitProvider()
		commands.CheckProvider(commands.DeploymentProvider)

		if !providers.ValidProviderStructure(
			config.ProjectDirectory,
			commands.DeploymentProvider.GetDirectoryStructure(),
		) && !commands.ForceInit {
			commands.WrongFolderMessage("verify", "cd %s; tenderly verify")
			os.Exit(1)
		}
		logrus.Info("Verifying your contracts...")

		var err error
		if verifyLegacy {
			commands.CheckLogin()
			err = verifyContractsLegacy(commands.NewRest())
		} else {
			if verifyRPC == "" {
				commands.CheckLogin()
			}
			err = verifyContractsEtherscan()
		}
		if err != nil {
			userError.LogErrorf("unable to verify contracts: %s", err)
			os.Exit(1)
		}

		logrus.Info("Smart Contracts successfully verified.")
	},
}

// verifyContractsEtherscan verifies each deployed contract through Tenderly's
// Etherscan-compatible verification API using the full solc standard-JSON
// input, either against real networks (dashboard API) or against a Virtual
// TestNet's state (--rpc).
func verifyContractsEtherscan() error {
	if verifyRPC == "" && verifyMode != "public" && verifyMode != "private" {
		return userError.NewUserError(
			fmt.Errorf("invalid verification mode: %s", verifyMode),
			commands.Colorizer.Sprintf("Invalid %s value %s. Supported values are %s and %s.",
				commands.Colorizer.Bold(commands.Colorizer.Green("--mode")),
				commands.Colorizer.Bold(commands.Colorizer.Red(verifyMode)),
				commands.Colorizer.Bold(commands.Colorizer.Green("public")),
				commands.Colorizer.Bold(commands.Colorizer.Green("private")),
			),
		)
	}

	contracts, providerConfig, err := getProviderContracts()
	if err != nil {
		return err
	}

	configPayload := commands.GetConfigPayload(providerConfig)

	sourceJSON, err := etherscan.BuildStandardJSONInput(contracts, configPayload)
	if err != nil {
		return userError.NewUserError(
			errors.Wrap(err, "unable to build compilation input"),
			"Couldn't assemble the compiler input from the build artifacts. Try recompiling your project.",
		)
	}

	accountID, projectSlug := "", ""
	if verifyRPC == "" {
		accountID, projectSlug, err = resolveAccountAndProject()
		if err != nil {
			return err
		}
	}

	var verified, failed int
	for _, contract := range contracts {
		if len(contract.Networks) == 0 {
			continue
		}

		compilerVersion := contract.Compiler.Version
		if compilerVersion == "" && configPayload != nil {
			compilerVersion = configPayload.CompilerVersion
		}
		if compilerVersion == "" {
			logrus.Error(commands.Colorizer.Sprintf(
				"✗ %s: no compiler version found in the build artifact or project configuration",
				commands.Colorizer.Bold(commands.Colorizer.Red(contract.Name)),
			))
			failed++
			continue
		}

		for networkID, deployment := range contract.Networks {
			verifierURL := buildVerifierURL(accountID, projectSlug, networkID)
			verifierClient := etherscan.NewClient(verifierURL, config.GetAccessKey(), config.GetToken(), client.UserAgent())

			target := fmt.Sprintf("network %s", networkID)
			if verifyRPC != "" {
				target = "Virtual TestNet"
			}

			s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
			s.Start()

			response, err := verifierClient.VerifySourceCode(&etherscan.VerifyRequest{
				ContractAddress: deployment.Address,
				ContractName:    fmt.Sprintf("%s:%s", contract.SourcePath, contract.Name),
				CompilerVersion: compilerVersion,
				SourceCode:      sourceJSON,
			})

			s.Stop()

			if err != nil {
				logrus.Error(commands.Colorizer.Sprintf(
					"✗ %s at %s on %s: %s",
					commands.Colorizer.Bold(commands.Colorizer.Red(contract.Name)),
					deployment.Address, target, err,
				))
				failed++
				continue
			}

			if !response.IsOK() {
				logrus.Error(commands.Colorizer.Sprintf(
					"✗ %s at %s on %s: %s",
					commands.Colorizer.Bold(commands.Colorizer.Red(contract.Name)),
					deployment.Address, target, response.Result,
				))
				failed++
				continue
			}

			logrus.Info(commands.Colorizer.Sprintf(
				"✓ %s at %s on %s",
				commands.Colorizer.Bold(commands.Colorizer.Green(contract.Name)),
				deployment.Address, target,
			))
			verified++
		}
	}

	if failed > 0 {
		return userError.NewUserError(
			fmt.Errorf("%d of %d verifications failed", failed, failed+verified),
			fmt.Sprintf("%d of %d contract verifications failed. See the list above for details.", failed, failed+verified),
		)
	}

	return nil
}

// getProviderContracts reads the build artifacts and applies the same guards
// as the legacy flow.
func getProviderContracts() ([]providers.Contract, *providers.Config, error) {
	logrus.Info("Analyzing provider configuration...")

	providerConfig, err := commands.DeploymentProvider.MustGetConfig()
	if err != nil {
		return nil, nil, err
	}

	networkIDs := commands.ExtractNetworkIDs(verifyNetworks)

	contracts, numberOfContractsWithANetwork, err := commands.DeploymentProvider.GetContracts(providerConfig.AbsoluteBuildDirectoryPath(), networkIDs)
	if err != nil {
		return nil, nil, userError.NewUserError(
			errors.Wrap(err, "unable to get provider contracts"),
			fmt.Sprintf("Couldn't read provider build files at: %s", providerConfig.AbsoluteBuildDirectoryPath()),
		)
	}

	if len(contracts) == 0 {
		return nil, nil, userError.NewUserError(
			fmt.Errorf("no contracts found in build dir: %s", providerConfig.AbsoluteBuildDirectoryPath()),
			commands.Colorizer.Sprintf("No contracts detected in build directory: %s. "+
				"This can happen when no contracts have been compiled yet.",
				commands.Colorizer.Bold(commands.Colorizer.Red(providerConfig.AbsoluteBuildDirectoryPath())),
			),
		)
	}
	if numberOfContractsWithANetwork == 0 {
		return nil, nil, userError.NewUserError(
			fmt.Errorf("no contracts with a network found in build dir: %s", providerConfig.AbsoluteBuildDirectoryPath()),
			commands.Colorizer.Sprintf("No deployed contracts detected in build directory: %s. This can happen when no contracts have been deployed yet.",
				commands.Colorizer.Bold(commands.Colorizer.Red(providerConfig.AbsoluteBuildDirectoryPath())),
			),
		)
	}

	logrus.Info("We have detected the following Smart Contracts:")
	for _, contract := range contracts {
		if len(contract.Networks) > 0 {
			logrus.Info(fmt.Sprintf("• %s", contract.Name))
		} else {
			logrus.Info(fmt.Sprintf("• %s (not deployed to any network, will be used as a library contract)", contract.Name))
		}
	}

	return contracts, providerConfig, nil
}

// resolveAccountAndProject figures out the dashboard verification target from
// the --project flag or the configured project.
func resolveAccountAndProject() (accountID string, projectSlug string, err error) {
	accountID = config.GetGlobalString(config.AccountID)

	projectSlug = verifyProject
	if projectSlug == "" {
		projectSlug = config.MaybeGetString(config.ProjectSlug)
	}
	if projectSlug == "" {
		if projectConfigurations, configErr := commands.GetProjectConfiguration(); configErr == nil && len(projectConfigurations) == 1 {
			for slug := range projectConfigurations {
				projectSlug = slug
			}
		}
	}
	if strings.Contains(projectSlug, "/") {
		projectInfo := strings.Split(projectSlug, "/")
		accountID = projectInfo[0]
		projectSlug = projectInfo[1]
	}

	if projectSlug == "" {
		return "", "", userError.NewUserError(
			fmt.Errorf("no project found"),
			commands.Colorizer.Sprintf("No project configured. Run %s, or pass the %s flag.",
				commands.Colorizer.Bold(commands.Colorizer.Green("tenderly init")),
				commands.Colorizer.Bold(commands.Colorizer.Green("--project")),
			),
		)
	}

	return accountID, projectSlug, nil
}

// buildVerifierURL picks the verification endpoint: the Virtual TestNet's
// {rpc}/verify when --rpc is set, otherwise the dashboard API endpoint for
// the deployment's network with the requested visibility.
func buildVerifierURL(accountID, projectSlug, networkID string) string {
	if verifyRPC != "" {
		return strings.TrimSuffix(verifyRPC, "/") + "/verify"
	}

	verifierURL := fmt.Sprintf(
		"%s/api/v1/account/%s/project/%s/etherscan/verify/network/%s",
		client.ApiBaseURL(), accountID, projectSlug, networkID,
	)
	if verifyMode == "public" {
		verifierURL += "/public"
	}
	return verifierURL
}

// verifyContractsLegacy is the pre-etherscan flow: bulk upload of truffle-style
// artifacts to the legacy verification endpoint.
func verifyContractsLegacy(rest *rest.Rest) error {
	contracts, providerConfig, err := getProviderContracts()
	if err != nil {
		return err
	}

	numberOfContractsWithANetwork := 0
	for _, contract := range contracts {
		numberOfContractsWithANetwork += len(contract.Networks)
	}

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)

	s.Start()

	configPayload := commands.GetConfigPayload(providerConfig)

	response, err := rest.Contract.VerifyContracts(payloads.UploadContractsRequest{
		Contracts: contracts,
		Config:    configPayload,
	})

	s.Stop()

	if err != nil {
		return userError.NewUserError(
			fmt.Errorf("failed uploading contracts: %s", err),
			"Couldn't verify contracts to the Tenderly servers",
		)
	}

	if response.Error != nil {
		return userError.NewUserError(
			fmt.Errorf("api error uploading contracts: %s", response.Error.Slug),
			commands.ContractErrorMessage(response.Error),
		)
	}

	if len(response.Contracts) != numberOfContractsWithANetwork {
		var nonPushedContracts []string

		for _, contract := range contracts {
			if len(contract.Networks) == 0 {
				continue
			}
			for networkId, network := range contract.Networks {
				var found bool
				for _, pushedContract := range response.Contracts {
					if pushedContract.Address == strings.ToLower(network.Address) && pushedContract.NetworkID == strings.ToLower(networkId) {
						found = true
						break
					}
				}
				if !found {
					nonPushedContracts = append(nonPushedContracts, commands.Colorizer.Sprintf(
						"• %s on network %s with address %s",
						commands.Colorizer.Bold(commands.Colorizer.Red(contract.Name)),
						commands.Colorizer.Bold(commands.Colorizer.Red(networkId)),
						commands.Colorizer.Bold(commands.Colorizer.Red(network.Address)),
					))
				}
			}
		}

		return userError.NewUserError(
			fmt.Errorf("unexpected number of verified contracts. Got: %d expected: %d", len(response.Contracts), numberOfContractsWithANetwork),
			fmt.Sprintf("Some of the contracts haven't been verified. This can happen when the contract isn't deployed to a supported network or some other error might have occurred. "+
				"Below is the list with all the contracts that weren't verified successfully:\n%s",
				strings.Join(nonPushedContracts, "\n"),
			),
		)
	}

	return nil
}
