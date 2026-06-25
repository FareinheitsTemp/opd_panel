package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ServersRoot = "/var/lib/opd/servers"

type ServerConfig struct {
	ID         string
	Name       string   `json:"name"`
	Port       int      `json:"port"`
	RAMMinMB   int      `json:"ram_min_mb"`
	RAMMaxMB   int      `json:"ram_max_mb"`
	Jar        string   `json:"jar"`
	JavaFlags  []string `json:"java_flags"`
	Dir        string
	JarPath    string
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
