package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the OPD supervisor daemon",
	Long:  `Starts the background daemon that manages server processes.\nListens on TCP 127.0.0.1:51200 for CLI commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting opd daemon...")

		d, err := daemon.New()
		if err != nil {
			return fmt.Errorf("failed to init daemon: %w", err)
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- d.ListenAndServe()
		}()

		fmt.Printf("opd daemon running (tcp: %s)\n", daemon.TCPAddr)

		// os.Interrupt works on both Windows and Unix (unlike syscall.SIGTERM)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
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
