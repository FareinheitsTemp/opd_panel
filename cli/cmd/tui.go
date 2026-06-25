package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		m := tui.NewModel(c)
		// Create program first so we can pass it to the model
		// before Init() is called — this is the correct bubbletea pattern
		// for goroutine → program communication via p.Send()
		p := tea.NewProgram(m, tea.WithAltScreen())
		m.SetProgram(p)
		_, err := p.Run()
		if err != nil {
			return fmt.Errorf("tui error: %w", err)
		}
		return nil
	},
}
