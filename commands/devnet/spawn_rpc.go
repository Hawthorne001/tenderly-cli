package devnet

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/tenderly/tenderly-cli/commands"
)

func init() {
	// The flags are kept so existing invocations still parse and reach the
	// deprecation message instead of failing with a flag error.
	cmdSpawnRpc.PersistentFlags().String("account", "", "Deprecated.")
	cmdSpawnRpc.PersistentFlags().String("project", "", "Deprecated.")
	cmdSpawnRpc.PersistentFlags().String("template", "", "Deprecated.")
	cmdSpawnRpc.PersistentFlags().String("access_key", "", "Deprecated.")
	cmdSpawnRpc.PersistentFlags().String("token", "", "Deprecated.")
	cmdSpawnRpc.PersistentFlags().Bool("return-url", false, "Deprecated.")
	CmdDevNet.AddCommand(cmdSpawnRpc)
}

var cmdSpawnRpc = &cobra.Command{
	Use:        "spawn-rpc",
	Short:      "DevNets have been discontinued in favor of Virtual TestNets",
	Deprecated: "DevNets have been discontinued in favor of Virtual TestNets.",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Info(commands.Colorizer.Sprintf(
			"DevNets have been discontinued and %s no longer works.\n\n"+
				"Use %s instead — they provide the same on-demand JSON-RPC infrastructure with unlimited faucet, "+
				"state sync and debugging tooling.\n\n"+
				"Read more: %s\n",
			commands.Colorizer.Bold(commands.Colorizer.Red("tenderly devnet spawn-rpc")),
			commands.Colorizer.Bold(commands.Colorizer.Green("Virtual TestNets")),
			commands.Colorizer.Bold(commands.Colorizer.Green("https://docs.tenderly.co/virtual-environments/overview")),
		))
		os.Exit(1)
	},
}
