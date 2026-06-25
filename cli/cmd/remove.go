package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <server-id>",
	Short: "Remove a server config from disk",
	Long:  `Deletes /var/lib/opd/servers/{id}/ including the jar and all data.\nStop the server first.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if !removeForce {
			fmt.Printf("\033[31mWarning:\033[0m This will delete ALL data for server '%s', including the jar.\n", id)
			fmt.Print("Type the server ID to confirm: ")
			sc := bufio.NewScanner(os.Stdin)
			if !sc.Scan() {
				return fmt.Errorf("aborted")
			}
			if strings.TrimSpace(sc.Text()) != id {
				return fmt.Errorf("aborted: input did not match")
			}
		}

		c := client.New()
		resp, err := c.Remove(id)
		if err != nil {
			return err
		}
		fmt.Printf("[opd] %s\n", resp.Message)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
}
