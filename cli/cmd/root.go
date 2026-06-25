package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "opd",
	Short: "OPD — Minecraft server process manager",
	Long:  `opd manages Minecraft server processes on your host.\nRun 'opd daemon' first, then use other commands to control servers.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		addCmd,
		daemonCmd,
		startCmd,
		stopCmd,
		restartCmd,
		statusCmd,
		logsCmd,
		consoleCmd,
		metricsCmd,
		tuiCmd,
	)
}
