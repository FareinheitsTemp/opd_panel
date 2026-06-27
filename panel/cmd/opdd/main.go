package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "modernc.org/sqlite"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/agent"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/config"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/socket"
	httpdelivery "github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/http"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository/sqlite"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/usecase"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/versions"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := os.MkdirAll(cfg.DBDir(), 0750); err != nil {
		log.Fatalf("mkdir db: %v", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	serverRepo := sqlite.NewServerRepo(db)
	scheduleRepo := sqlite.NewScheduleRepo(db)
	subuserRepo := sqlite.NewSubuserRepo(db)
	allocationRepo := sqlite.NewAllocationRepo(db)
	databaseRepo := sqlite.NewDatabaseRepo(db)

	agentClient := agent.NewClient(cfg.AgentURL, cfg.AgentSecret)
	vm := versions.NewManager(cfg.CacheDir)

	mysqlDSN := getEnv("OPD_MYSQL_DSN", "root:@tcp(127.0.0.1:3306)/")
	mysqlHost := getEnv("OPD_MYSQL_HOST", "127.0.0.1")
	mysqlPort := getEnvInt("OPD_MYSQL_PORT", 3306)
	encKey := getEncryptKey()

	serverUC := usecase.NewServerUseCase(serverRepo, agentClient, vm)
	scheduleUC := usecase.NewScheduleUseCase(scheduleRepo, agentClient)
	subuserUC := usecase.NewSubuserUseCase(subuserRepo)
	networkUC := usecase.NewNetworkUseCase(allocationRepo)
	databaseUC := usecase.NewDatabaseUseCase(databaseRepo, mysqlDSN, mysqlHost, mysqlPort, encKey)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TCP socket server (internal)
	srv, err := socket.NewServer(cfg, serverUC, scheduleUC, subuserUC, networkUC, databaseUC)
	if err != nil {
		log.Fatalf("socket server: %v", err)
	}

	// HTTP REST API server (for web frontend)
	httpAddr := getEnv("OPD_HTTP_ADDR", "127.0.0.1:51201")
	httpSrv := httpdelivery.NewHTTPServer(httpAddr, cfg.ServersDir, serverUC, scheduleUC, subuserUC, networkUC, databaseUC)

	go func() {
		if err := httpSrv.Run(ctx); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("opdd socket listening on %s", cfg.SocketPath)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			version TEXT NOT NULL,
			port INTEGER NOT NULL,
			ram_min INTEGER NOT NULL,
			ram_max INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'stopped',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			name TEXT NOT NULL,
			cron_expr TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			tasks TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL,
			last_run_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS subusers (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			email TEXT NOT NULL,
			permissions TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL,
			UNIQUE(server_id, email)
		);
		CREATE TABLE IF NOT EXISTS allocations (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL UNIQUE,
			alias TEXT,
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS server_databases (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			db_name TEXT NOT NULL UNIQUE,
			db_user TEXT NOT NULL UNIQUE,
			password_enc TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
	`)
	return err
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEncryptKey() []byte {
	hexKey := os.Getenv("OPD_DB_ENCRYPT_KEY")
	if hexKey != "" {
		key, err := hex.DecodeString(hexKey)
		if err == nil && len(key) == 32 {
			return key
		}
	}
	log.Println("WARNING: OPD_DB_ENCRYPT_KEY not set, using insecure dev key")
	key := make([]byte, 32)
	for i := range key { key[i] = byte(i + 1) }
	return key
}
