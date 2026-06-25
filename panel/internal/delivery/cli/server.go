package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/cli/client"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage Minecraft servers",
	}

	cmd.AddCommand(
		serverListCmd(),
		serverStartCmd(),
		serverStopCmd(),
		serverRestartCmd(),
		serverInfoCmd(),
		serverCreateCmd(),
		serverDeleteCmd(),
		serverLogsCmd(),
	)

	return cmd
}

func serverListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			resp, err := c.Send("server.list", nil)
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
}

func serverStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Start a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.start", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Starting %s\n", args[0])
			return nil
		},
	}
}

func serverStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.stop", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Stopping %s\n", args[0])
			return nil
		},
	}
}

func serverRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.restart", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Restarting %s\n", args[0])
			return nil
		},
	}
}

func serverInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <id>",
		Short: "Get server info and live metrics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			resp, err := c.Send("server.info", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
}

func serverCreateCmd() *cobra.Command {
	var (
		name    string
		srvType string
		version string
		ram     int
		port    int
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new server",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.create", map[string]any{
				"name":    name,
				"type":    srvType,
				"version": version,
				"ram":     ram,
				"port":    port,
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Server %s created\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Server name (required)")
	cmd.Flags().StringVar(&srvType, "type", "paper", "Server type: paper|vanilla|purpur|fabric")
	cmd.Flags().StringVar(&version, "version", "", "Minecraft version (required)")
	cmd.Flags().IntVar(&ram, "ram", 1024, "Max RAM in MB")
	cmd.Flags().IntVar(&port, "port", 25565, "Server port")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("version")
	return cmd
}

func serverDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a server (must be stopped)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.delete", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Server %s deleted\n", args[0])
			return nil
		},
	}
}

func serverLogsCmd() *cobra.Command {
	var follow bool
	var lines int
	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "View server logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("server.logs", map[string]any{
				"server_id": args[0],
				"follow":    follow,
				"lines":     lines,
			})
			return err
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVar(&lines, "lines", 50, "Number of lines to show")
	return cmd
}
