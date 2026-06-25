package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var consoleCmd = &cobra.Command{
	Use:   "console <server-id>",
	Short: "Attach an interactive console to a server's stdin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID := args[0]
		c := client.New()

		// Start log stream in background so we see output
		logCh, err := c.StreamLogs(serverID)
		if err != nil {
			return fmt.Errorf("could not attach log stream: %w", err)
		}
		go func() {
			for line := range logCh {
				fmt.Printf("\r%s\n> ", line)
			}
		}()

		fmt.Printf("[opd] Attached to %s console. Type commands, Ctrl+C to detach.\n", serverID)

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		scanner := bufio.NewScanner(os.Stdin)
		doneCh := make(chan struct{})

		go func() {
			defer close(doneCh)
			for {
				fmt.Print("> ")
				if !scanner.Scan() {
					return
				}
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if _, err := c.SendCommand(serverID, line); err != nil {
					fmt.Fprintf(os.Stderr, "[opd] error: %v\n", err)
				}
			}
		}()

		select {
		case <-sig:
		case <-doneCh:
		}

		fmt.Println("\n[opd] Detached.")
		return nil
	},
}
