package cli

import (
	"github.com/spf13/cobra"
)

var (
	socketPath string
	outputFmt  string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "opd",
		Short: "OPD Panel — Minecraft server manager CLI",
		Long:  "Manage Minecraft servers via the opdd daemon.",
	}

	root.PersistentFlags().StringVar(&socketPath, "socket", "/var/run/opd/opd.sock", "Unix socket path")
	root.PersistentFlags().StringVar(&outputFmt, "output", "table", "Output format: table|json")

	root.AddCommand(
		newServerCmd(),
		newBackupCmd(),
		newVersionsCmd(),
		newTunnelCmd(),
	)

	return root
}
