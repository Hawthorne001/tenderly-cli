package commands

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	exportCmd.AddCommand(exportInitCmd)
	RootCmd.AddCommand(exportCmd)
}

func printExportDeprecationMessage() {
	logrus.Info(Colorizer.Sprintf(
		"The export feature has been deprecated in favor of %s.\n\n"+
			"Virtual TestNets provide on-demand JSON-RPC infrastructure with state sync, "+
			"an unlimited faucet and full debugging tooling at %s.\n"+
			"You can read more about them here: %s.",
		Colorizer.Bold(Colorizer.Green("Virtual TestNets")),
		Colorizer.Bold(Colorizer.Green("https://dashboard.tenderly.co")),
		Colorizer.Bold(Colorizer.Green("https://docs.tenderly.co/virtual-environments/overview")),
	))
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "The export feature has been deprecated in favor of Virtual TestNets",
	Run: func(cmd *cobra.Command, args []string) {
		printExportDeprecationMessage()
	},
}

var exportInitCmd = &cobra.Command{
	Use:   "init",
	Short: "The export feature has been deprecated in favor of Virtual TestNets",
	Run: func(cmd *cobra.Command, args []string) {
		printExportDeprecationMessage()
	},
}
