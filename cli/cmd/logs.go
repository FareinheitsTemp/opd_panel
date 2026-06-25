package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var logsCmd = &cobra.Command{
	Use:   "logs <server-id>",
	Short: "Stream server logs to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		ch, err := c.StreamLogs(args[0])
		if err != nil {
			return err
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case line, ok := <-ch:
				if !ok {
					fmt.Println("[opd] log stream closed")
					return nil
				}
				fmt.Println(line)
			case <-sig:
				return nil
			}
		}
	},
}
