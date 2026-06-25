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

		// BUG FIX #6: os.Exit(1) inside the goroutine bypassed the deferred
		// Shutdown() call, leaving all managed Java processes orphaned.
		// Fix: use an error channel so the main goroutine handles the error
		// and Shutdown() is always called via the defer path.
		errCh := make(chan error, 1)
		go func() {
			errCh <- d.ListenAndServe()
		}()

		fmt.Printf("opd daemon running (socket: %s)\n", daemon.SocketPath)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(quit)

		select {
		case <-quit:
			fmt.Println("\nShutting down daemon...")
			return d.Shutdown()
		case err := <-errCh:
			if err != nil {
				fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
				_ = d.Shutdown()
				return err
			}
			return nil
		}
	},
}
