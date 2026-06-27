package config

import (
	"path/filepath"
	"runtime"

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

	defaultSocket := "/var/run/opd/opd.sock"
	defaultDB := "/var/lib/opd/opd.db"
	defaultServers := "/var/lib/opd/servers"
	defaultCache := "/var/lib/opd/cache"

	if runtime.GOOS == "windows" {
		defaultSocket = "127.0.0.1:7071"
		defaultDB = "C:/opd/opd.db"
		defaultServers = "C:/opd/servers"
		defaultCache = "C:/opd/cache"
	}

	viper.SetDefault("SOCKET", defaultSocket)
	viper.SetDefault("DB_PATH", defaultDB)
	viper.SetDefault("AGENT_URL", "http://127.0.0.1:7070")
	viper.SetDefault("SERVERS_DIR", defaultServers)
	viper.SetDefault("CACHE_DIR", defaultCache)
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
