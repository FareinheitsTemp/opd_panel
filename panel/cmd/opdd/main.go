package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/config"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/socket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv, err := socket.NewServer(cfg)
	if err != nil {
		log.Fatalf("socket server: %v", err)
	}

	log.Printf("opdd listening on %s", cfg.SocketPath)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}
