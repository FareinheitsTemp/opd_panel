package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers known on disk (running or not)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		servers, err := c.ListDisk()
		if err != nil {
			return err
		}

		if len(servers) == 0 {
			fmt.Println("No servers found. Run 'opd add <id>' to create one.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Port", "Max RAM", "Jar"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, s := range servers {
			table.Append([]string{
				s.ID,
				s.Name,
				fmt.Sprintf("%d", s.Port),
				fmt.Sprintf("%dMB", s.RAMMax),
				s.Jar,
			})
		}
		table.Render()
		return nil
	},
}
