package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/cli/client"
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage server backups",
	}
	cmd.AddCommand(
		backupCreateCmd(),
		backupListCmd(),
		backupRestoreCmd(),
	)
	return cmd
}

func backupCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <server_id>",
		Short: "Create a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("backup.create", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Backup started for %s\n", args[0])
			return nil
		},
	}
}

func backupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <server_id>",
		Short: "List backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			resp, err := c.Send("backup.list", map[string]any{"server_id": args[0]})
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
}

func backupRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <server_id> <backup_id>",
		Short: "Restore a backup",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(socketPath)
			_, err := c.Send("backup.restore", map[string]any{
				"server_id": args[0],
				"backup_id": args[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Restoring backup %s for %s\n", args[1], args[0])
			return nil
		},
	}
}
