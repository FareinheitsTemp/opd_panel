package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the OPD supervisor daemon",
	Long:  `Starts the background daemon that manages server processes.\nListens on a Unix socket for CLI commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting opd daemon...")

		d, err := daemon.New()
		if err != nil {
			return fmt.Errorf("failed to init daemon: %w", err)
		}

		go func() {
			if err := d.ListenAndServe(); err != nil {
				fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
				os.Exit(1)
			}
		}()

		fmt.Printf("opd daemon running (socket: %s)\n", daemon.SocketPath)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\nShutting down daemon...")
		return d.Shutdown()
	},
}
