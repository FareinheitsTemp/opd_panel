package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		servers, err := c.List()
		if err != nil {
			return err
		}

		if len(servers) == 0 {
			fmt.Println("No servers registered.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Status", "PID", "Port", "RAM", "CPU", "Uptime"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, s := range servers {
			table.Append([]string{
				s.ID,
				s.Name,
				statusColour(s.Status),
				fmt.Sprintf("%d", s.PID),
				fmt.Sprintf("%d", s.Port),
				fmt.Sprintf("%s / %s", formatBytes(s.RAMUsed), formatBytes(s.RAMMax)),
				fmt.Sprintf("%.1f%%", s.CPU),
				formatUptime(s.Uptime),
			})
		}

		table.Render()
		return nil
	},
}

func statusColour(status string) string {
	switch status {
	case "running":
		return "\033[32m" + status + "\033[0m"
	case "stopped":
		return "\033[90m" + status + "\033[0m"
	case "starting":
		return "\033[33m" + status + "\033[0m"
	case "stopping":
		return "\033[33m" + status + "\033[0m"
	case "crashed":
		return "\033[31m" + status + "\033[0m"
	}
	return status
}

func formatBytes(b uint64) string {
	const mb = 1024 * 1024
	if b < mb {
		return fmt.Sprintf("%dB", b)
	}
	return fmt.Sprintf("%dMB", b/mb)
}

func formatUptime(secs uint64) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm%ds", secs/60, secs%60)
	}
	return fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
}
