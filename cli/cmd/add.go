package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const serversRoot = "/var/lib/opd/servers"

type serverConfigJSON struct {
	Name      string   `json:"name"`
	Port      int      `json:"port"`
	RAMMinMB  int      `json:"ram_min_mb"`
	RAMMaxMB  int      `json:"ram_max_mb"`
	Jar       string   `json:"jar"`
	JavaFlags []string `json:"java_flags"`
}

var addCmd = &cobra.Command{
	Use:   "add <server-id>",
	Short: "Interactively create a new server config",
	Long: `Creates /var/lib/opd/servers/{id}/opd.json by asking a few questions.
Put your server.jar in that directory and run 'opd start <id>'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		dir := filepath.Join(serversRoot, id)

		// Check if already exists
		configPath := filepath.Join(dir, "opd.json")
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config already exists at %s", configPath)
		}

		sc := bufio.NewScanner(os.Stdin)

		fmt.Printf("\n\033[36m◈ Creating server:\033[0m %s\n\n", id)

		name := prompt(sc, "Server name", id)
		portStr := prompt(sc, "Port", "25565")
		ramMinStr := prompt(sc, "Min RAM (MB)", "1024")
		ramMaxStr := prompt(sc, "Max RAM (MB)", "4096")
		jar := prompt(sc, "Jar filename", "server.jar")

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		ramMin, err := strconv.Atoi(ramMinStr)
		if err != nil {
			return fmt.Errorf("invalid RAM min: %s", ramMinStr)
		}
		ramMax, err := strconv.Atoi(ramMaxStr)
		if err != nil {
			return fmt.Errorf("invalid RAM max: %s", ramMaxStr)
		}

		cfg := serverConfigJSON{
			Name:      name,
			Port:      port,
			RAMMinMB:  ramMin,
			RAMMaxMB:  ramMax,
			Jar:       jar,
			JavaFlags: []string{},
		}

		// Create directory
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}

		// Write opd.json
		f, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("create config: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf("\n\033[32m✔\033[0m Config written to %s\n", configPath)
		fmt.Printf("\033[90m  Put your %s in %s\033[0m\n", jar, dir)
		fmt.Printf("\033[90m  Then run: opd start %s\033[0m\n\n", id)
		return nil
	},
}

func prompt(sc *bufio.Scanner, label, defaultVal string) string {
	fmt.Printf("  \033[33m%s\033[0m [%s]: ", label, defaultVal)
	if !sc.Scan() {
		return defaultVal
	}
	v := strings.TrimSpace(sc.Text())
	if v == "" {
		return defaultVal
	}
	return v
}
