package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/cli/client"
)

func newTunnelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tunnel",
		Short: "Manage tunnels (Playit.gg, DuckDNS)",
	}
	cmd.AddCommand(
		tunnelAttachCmd(),
		tunnelDetachCmd(),
		tunnelStatusCmd(),
	)
	return cmd
}

func tunnelAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <server_id>",
		Short: "Attach a public tunnel to a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("tunnel.attach", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Tunnel attached to %s\n", args[0])
			return nil
		},
	}
}

func tunnelDetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detach <server_id>",
		Short: "Detach tunnel from a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("tunnel.detach", map[string]any{"server_id": args[0]})
			return err
		},
	}
}

func tunnelStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active tunnels",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			resp, err := c.Send("tunnel.status", nil)
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
}
