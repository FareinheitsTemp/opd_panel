package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var (
	metricsWatch    bool
	metricsInterval int
)

var metricsCmd = &cobra.Command{
	Use:   "metrics <server-id>",
	Short: "Show live CPU/RAM metrics for a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID := args[0]
		c := client.New()

		printMetrics := func() error {
			m, err := c.Metrics(serverID)
			if err != nil {
				return err
			}
			fmt.Printf(
				"[%s] CPU: \033[36m%.1f%%\033[0m  RAM: \033[36m%s\033[0m / %s  Uptime: %s\n",
				m.ServerID,
				m.CPU,
				formatBytes(m.RAMUsed),
				formatBytes(m.RAMMax),
				formatUptime(m.Uptime),
			)
			return nil
		}

		if !metricsWatch {
			return printMetrics()
		}

		// Watch mode: refresh every N seconds
		ticker := time.NewTicker(time.Duration(metricsInterval) * time.Second)
		defer ticker.Stop()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		_ = printMetrics()
		for {
			select {
			case <-ticker.C:
				_ = printMetrics()
			case <-sig:
				return nil
			}
		}
	},
}

func init() {
	metricsCmd.Flags().BoolVarP(&metricsWatch, "watch", "w", false, "Continuously refresh metrics")
	metricsCmd.Flags().IntVarP(&metricsInterval, "interval", "i", 2, "Refresh interval in seconds (used with --watch)")
}
