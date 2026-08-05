package commands

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/tenderly/tenderly-cli/providers"
	"github.com/tenderly/tenderly-cli/rest"
	"github.com/tenderly/tenderly-cli/userError"
)

// CheckUnknownPushNetworks warns about contract deployments on networks
// Tenderly doesn't recognize — the API silently drops those, and when nothing
// remains it fails with an opaque internal server error. Returns a user error
// when no deployment targets a recognized network. If the public networks
// can't be fetched, validation is skipped.
func CheckUnknownPushNetworks(rest *rest.Rest, contracts []providers.Contract) *userError.UserError {
	networks, err := rest.Networks.GetPublicNetworks()
	if err != nil || networks == nil {
		logrus.Debugf("skipping network validation, failed fetching public networks: %s", err)
		return nil
	}

	knownNetworkIDs := make(map[string]bool)
	for _, network := range *networks {
		knownNetworkIDs[network.EthereumNetworkID] = true
	}
	if len(knownNetworkIDs) == 0 {
		return nil
	}

	recognized := 0
	var unknown []string
	for _, contract := range contracts {
		for networkID, deployment := range contract.Networks {
			if knownNetworkIDs[networkID] {
				recognized++
				continue
			}
			unknown = append(unknown, Colorizer.Sprintf(
				"• %s on network %s (%s)",
				Colorizer.Bold(contract.Name),
				Colorizer.Bold(Colorizer.Red(networkID)),
				deployment.Address,
			))
		}
	}

	if len(unknown) == 0 {
		return nil
	}

	logrus.Warn(Colorizer.Sprintf(
		"The following deployments target networks that are not supported Tenderly public networks "+
			"(e.g. local nodes like Hardhat 31337 or Ganache 5777) and will be ignored:\n%s",
		strings.Join(unknown, "\n"),
	))

	if recognized == 0 {
		return userError.NewUserError(
			fmt.Errorf("no contracts deployed to a supported network"),
			"None of the detected deployments target a supported Tenderly network, so there is nothing to push. "+
				"Deploy your contracts to a supported network first, or use a Virtual TestNet.",
		)
	}

	return nil
}
