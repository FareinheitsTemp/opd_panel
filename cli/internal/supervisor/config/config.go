package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

const ServersRoot = "/var/lib/opd/servers"

type ServerConfig struct {
	ID        string
	Name      string   `json:"name"`
	Port      int      `json:"port"`
	RAMMinMB  int      `json:"ram_min_mb"`
	RAMMaxMB  int      `json:"ram_max_mb"`
	Jar       string   `json:"jar"`
	JavaFlags []string `json:"java_flags"`
	Dir       string
	JarPath   string
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

// ListAll scans ServersRoot and returns info for every server that has an opd.json.
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
			continue // skip dirs without opd.json
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

// Remove deletes the server config directory.
// Refuses if the directory contains a running process (caller should stop first).
func Remove(id string) error {
	dir := filepath.Join(ServersRoot, id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("server %s not found on disk", id)
	}
	return os.RemoveAll(dir)
}
