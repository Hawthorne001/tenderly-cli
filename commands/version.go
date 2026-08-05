package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tenderly/tenderly-cli/rest/client"
)

var CurrentCLIVersion string

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version of the CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current CLI version: %s\n\n"+
			"To report a bug or give feedback send us an email at support@tenderly.co\n",
			CurrentCLIVersion,
		)
	},
}

func SetCurrentCLIVersion(version string) {
	CurrentCLIVersion = version
	if !strings.HasPrefix(CurrentCLIVersion, "v") {
		CurrentCLIVersion = fmt.Sprintf("v%s", CurrentCLIVersion)
	}

	userAgentVersion := CurrentCLIVersion
	if version == "" {
		userAgentVersion = "dev"
	}
	client.SetUserAgent(fmt.Sprintf("tenderly-cli/%s", userAgentVersion))

	CheckVersion(false, false)
}
