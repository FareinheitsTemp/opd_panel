package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	SocketPath  string
	DBPath      string
	AgentURL    string
	AgentSecret string
	ServersDir  string
	CacheDir    string
	LogLevel    string
}

func (c *Config) DBDir() string {
	return filepath.Dir(c.DBPath)
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("OPD")
	viper.AutomaticEnv()

	viper.SetDefault("SOCKET", "/var/run/opd/opd.sock")
	viper.SetDefault("DB_PATH", "/var/lib/opd/opd.db")
	viper.SetDefault("AGENT_URL", "http://127.0.0.1:7070")
	viper.SetDefault("SERVERS_DIR", "/var/lib/opd/servers")
	viper.SetDefault("CACHE_DIR", "/var/lib/opd/cache")
	viper.SetDefault("LOG_LEVEL", "info")

	return &Config{
		SocketPath:  viper.GetString("SOCKET"),
		DBPath:      viper.GetString("DB_PATH"),
		AgentURL:    viper.GetString("AGENT_URL"),
		AgentSecret: viper.GetString("AGENT_SECRET"),
		ServersDir:  viper.GetString("SERVERS_DIR"),
		CacheDir:    viper.GetString("CACHE_DIR"),
		LogLevel:    viper.GetString("LOG_LEVEL"),
	}, nil
}
