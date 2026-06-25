package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var restartCmd = &cobra.Command{
	Use:   "restart <server-id>",
	Short: "Restart a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		resp, err := c.Restart(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("[opd] %s\n", resp.Message)
		return nil
	},
}
