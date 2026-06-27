package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "opd",
	Short: "OPD — Minecraft server process manager",
	Long:  `opd manages Minecraft server processes.\nRun 'opd tui' or just 'opd' to launch the full interactive dashboard.`,
	RunE:  tuiCmd.RunE,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		tuiCmd,
		daemonCmd,
		addCmd,
		listCmd,
		removeCmd,
		startCmd,
		stopCmd,
		restartCmd,
		statusCmd,
		logsCmd,
		consoleCmd,
		metricsCmd,
	)
}
