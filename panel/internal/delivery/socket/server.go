package socket

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/config"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/usecase"
)

type Server struct {
	cfg    *config.Config
	router *Router
}

func NewServer(
	cfg *config.Config,
	serverUC *usecase.ServerUseCase,
	scheduleUC *usecase.ScheduleUseCase,
	subuserUC *usecase.SubuserUseCase,
	networkUC *usecase.NetworkUseCase,
	databaseUC *usecase.DatabaseUseCase,
) (*Server, error) {
	return &Server{
		cfg: cfg,
		router: NewRouter(cfg, serverUC, scheduleUC, subuserUC, networkUC, databaseUC),
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.cfg.SocketPath), 0750); err != nil {
		return err
	}
	os.Remove(s.cfg.SocketPath)

	ln, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		return err
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		}
		go s.handle(ctx, conn)
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	enc := json.NewEncoder(conn)

	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = enc.Encode(Response{ID: req.ID, OK: false, Error: "invalid json"})
			continue
		}
		s.router.Handle(ctx, &req, enc)
	}
}
