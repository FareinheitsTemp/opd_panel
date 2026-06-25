package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/cli/client"
)

func newVersionsCmd() *cobra.Command {
	var srvType string
	var all bool

	cmd := &cobra.Command{
		Use:   "versions",
		Short: "List available server versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			payload := map[string]any{"type": srvType, "all": all}
			resp, err := c.Send("versions.list", payload)
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&srvType, "type", "paper", "Server type: paper|vanilla|purpur|fabric")
	cmd.Flags().BoolVar(&all, "all", false, "Show all types")
	return cmd
}
