package socket

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	var ln net.Listener
	var err error

	addr := s.cfg.SocketPath

	// On Windows or if addr looks like host:port — use TCP
	if runtime.GOOS == "windows" || isTCP(addr) {
		if isTCP(addr) {
			ln, err = net.Listen("tcp", addr)
		} else {
			// default Windows TCP fallback
			ln, err = net.Listen("tcp", "127.0.0.1:7071")
		}
	} else {
		if err2 := os.MkdirAll(filepath.Dir(addr), 0750); err2 != nil {
			return err2
		}
		os.Remove(addr)
		ln, err = net.Listen("unix", addr)
	}
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

func isTCP(addr string) bool {
	return strings.Contains(addr, ":")
}
