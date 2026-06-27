package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

func DefaultServersRoot() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "opd", "servers")
	}
	return "/var/lib/opd/servers"
}

var ServersRoot = DefaultServersRoot()

func SetServersRoot(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", path, err)
	}
	ServersRoot = path
	return saveGlobalConfig()
}

type globalConfig struct {
	ServersRoot string `json:"servers_root"`
}

func globalConfigPath() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "opd", "config.json")
	}
	return "/var/lib/opd/config.json"
}

func LoadGlobalConfig() {
	f, err := os.Open(globalConfigPath())
	if err != nil {
		return
	}
	defer f.Close()
	var gc globalConfig
	if err := json.NewDecoder(f).Decode(&gc); err == nil && gc.ServersRoot != "" {
		ServersRoot = gc.ServersRoot
	}
}

func saveGlobalConfig() error {
	path := globalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(globalConfig{ServersRoot: ServersRoot})
}

type ServerConfig struct {
	ID          string
	Name        string   `json:"name"`
	Port        int      `json:"port"`
	RAMMinMB    int      `json:"ram_min_mb"`
	RAMMaxMB    int      `json:"ram_max_mb"`
	Jar         string   `json:"jar"`
	JavaFlags   []string `json:"java_flags"`
	Motd        string   `json:"motd,omitempty"`
	MaxPlayers  int      `json:"max_players,omitempty"`
	Gamemode    string   `json:"gamemode,omitempty"`
	Difficulty  string   `json:"difficulty,omitempty"`
	AutoRestart bool     `json:"auto_restart,omitempty"`
	Dir         string
	JarPath     string
}

func Load(id string) (*ServerConfig, error) {
	dir := filepath.Join(ServersRoot, id)
	path := filepath.Join(dir, "opd.json")

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var cfg ServerConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg.ID = id
	cfg.Dir = dir
	if cfg.Jar == "" {
		cfg.Jar = "server.jar"
	}
	cfg.JarPath = filepath.Join(dir, cfg.Jar)
	return &cfg, nil
}

func Save(cfg *ServerConfig) error {
	path := filepath.Join(ServersRoot, cfg.ID, "opd.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

// Create creates the server directory and writes opd.json. Returns the dir path.
func Create(cfg *ServerConfig) (string, error) {
	dir := filepath.Join(ServersRoot, cfg.ID)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("server %s already exists", cfg.ID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}
	cfg.Dir = dir
	cfg.JarPath = filepath.Join(dir, cfg.Jar)
	if err := Save(cfg); err != nil {
		return "", err
	}
	return dir, nil
}

func ListAll() ([]ipc.DiskServerInfo, error) {
	entries, err := os.ReadDir(ServersRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []ipc.DiskServerInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfg, err := Load(e.Name())
		if err != nil {
			continue
		}
		out = append(out, ipc.DiskServerInfo{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Port:   cfg.Port,
			RAMMax: cfg.RAMMaxMB,
			Jar:    cfg.Jar,
		})
	}
	return out, nil
}

func Remove(id string) error {
	dir := filepath.Join(ServersRoot, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("server %s not found on disk", id)
	}
	return os.RemoveAll(dir)
}
