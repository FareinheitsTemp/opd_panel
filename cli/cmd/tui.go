package cmd

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start daemon + launch interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try to start the daemon embedded in this process.
		// If port already taken — existing daemon is running, just connect.
		d, daemonErr := daemon.New()
		if daemonErr == nil {
			go func() {
				_ = d.ListenAndServe()
			}()
			time.Sleep(150 * time.Millisecond)
		}

		// Verify connectivity before launching TUI.
		c := client.New()
		var pingErr error
		for i := 0; i < 5; i++ {
			if pingErr = c.Ping(); pingErr == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if pingErr != nil {
			if daemonErr != nil {
				return fmt.Errorf("daemon unavailable: %w", daemonErr)
			}
			return fmt.Errorf("daemon started but not responding: %w", pingErr)
		}

		m := tui.NewModel(c)
		p := tea.NewProgram(m, tea.WithAltScreen())
		m.SetProgram(p)
		_, err := p.Run()
		if err != nil {
			return fmt.Errorf("tui error: %w", err)
		}

		if d != nil {
			_ = d.Shutdown()
		}
		return nil
	},
}
