package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the OPD background daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := daemon.New()
		if err != nil {
			return fmt.Errorf("init daemon: %w", err)
		}

		fmt.Println("[opd] daemon started")
		fmt.Println("[opd] IPC listening on 127.0.0.1:51200")
		fmt.Println("[opd] Web UI API listening on http://127.0.0.1:51201")
		fmt.Println("[opd] Open http://localhost:3000 in your browser for the web UI")

		openBrowser("http://localhost:3000")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit

		fmt.Println("[opd] shutting down...")
		return d.Shutdown()
	},
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		fmt.Printf("[opd] could not open browser: %v\n", err)
		fmt.Printf("[opd] open manually: %s\n", url)
	}
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
