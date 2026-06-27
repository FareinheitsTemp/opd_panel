package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/usecase"
)

type HTTPServer struct {
	serverUC   *usecase.ServerUseCase
	scheduleUC *usecase.ScheduleUseCase
	subuserUC  *usecase.SubuserUseCase
	networkUC  *usecase.NetworkUseCase
	databaseUC *usecase.DatabaseUseCase
	addr       string
	serversDir string
}

func NewHTTPServer(
	addr string,
	serversDir string,
	serverUC *usecase.ServerUseCase,
	scheduleUC *usecase.ScheduleUseCase,
	subuserUC *usecase.SubuserUseCase,
	networkUC *usecase.NetworkUseCase,
	databaseUC *usecase.DatabaseUseCase,
) *HTTPServer {
	return &HTTPServer{
		addr:       addr,
		serversDir: serversDir,
		serverUC:   serverUC,
		scheduleUC: scheduleUC,
		subuserUC:  subuserUC,
		networkUC:  networkUC,
		databaseUC: databaseUC,
	}
}

func (s *HTTPServer) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/disks", s.cors(s.handleDisks))
	mux.HandleFunc("/api/servers", s.cors(s.handleServers))
	mux.HandleFunc("/api/servers/", s.cors(s.handleServer))
	mux.HandleFunc("/api/schedules", s.cors(s.handleSchedules))
	mux.HandleFunc("/api/schedules/", s.cors(s.handleSchedule))
	mux.HandleFunc("/api/subusers", s.cors(s.handleSubusers))
	mux.HandleFunc("/api/subusers/", s.cors(s.handleSubuser))
	mux.HandleFunc("/api/databases", s.cors(s.handleDatabases))
	mux.HandleFunc("/api/databases/", s.cors(s.handleDatabase))
	mux.HandleFunc("/api/allocations", s.cors(s.handleAllocations))
	mux.HandleFunc("/api/allocations/", s.cors(s.handleAllocation))

	srv := &http.Server{Addr: s.addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("HTTP API listening on %s", s.addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *HTTPServer) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func pathSegment(r *http.Request, index int) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func serverIDFromPath(r *http.Request) string { return pathSegment(r, 2) }
func subPath(r *http.Request) string          { return pathSegment(r, 3) }

// ── Disks (platform-specific impl in disks_windows.go / disks_linux.go) ────

func (s *HTTPServer) handleDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	disks := getDiskList()
	writeJSON(w, 200, map[string]any{
		"disks":        disks,
		"current_root": s.serversDir,
	})
}

// ── Servers ────────────────────────────────────────────────────────────

func (s *HTTPServer) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.serverUC.List(r.Context())
		if err != nil {
			writeErr(w, 500, err)
			return
		}
		if list == nil {
			list = []*domain.Server{}
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var in usecase.CreateServerInput
		if err := decodeJSON(r, &in); err != nil {
			writeErr(w, 400, err)
			return
		}
		srv, err := s.serverUC.Create(r.Context(), in)
		if err != nil {
			writeErr(w, 500, err)
			return
		}
		writeJSON(w, 201, srv)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleServer(w http.ResponseWriter, r *http.Request) {
	id := serverIDFromPath(r)
	sub := subPath(r)

	switch sub {
	case "start":
		if err := s.serverUC.Start(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, map[string]string{"status": "starting"})
		return
	case "stop":
		if err := s.serverUC.Stop(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, map[string]string{"status": "stopping"})
		return
	case "restart":
		if err := s.serverUC.Restart(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, map[string]string{"status": "restarting"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		srv, metrics, err := s.serverUC.Info(r.Context(), id)
		if err != nil {
			writeErr(w, 404, err); return
		}
		writeJSON(w, 200, map[string]any{"server": srv, "metrics": metrics})
	case http.MethodDelete:
		if err := s.serverUC.Delete(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ── Schedules ────────────────────────────────────────────────────────────

func (s *HTTPServer) handleSchedules(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	switch r.Method {
	case http.MethodGet:
		list, err := s.scheduleUC.List(r.Context(), serverID)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var in usecase.CreateScheduleInput
		if err := decodeJSON(r, &in); err != nil {
			writeErr(w, 400, err); return
		}
		sc, err := s.scheduleUC.Create(r.Context(), in)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 201, sc)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleSchedule(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r, 2)
	switch r.Method {
	case http.MethodPut:
		var in usecase.CreateScheduleInput
		if err := decodeJSON(r, &in); err != nil {
			writeErr(w, 400, err); return
		}
		sc, err := s.scheduleUC.Update(r.Context(), id, in)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, sc)
	case http.MethodDelete:
		if err := s.scheduleUC.Delete(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ── Subusers ─────────────────────────────────────────────────────────────

func (s *HTTPServer) handleSubusers(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	switch r.Method {
	case http.MethodGet:
		list, err := s.subuserUC.List(r.Context(), serverID)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var body struct {
			ServerID    string   `json:"server_id"`
			Email       string   `json:"email"`
			Permissions []string `json:"permissions"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, 400, err); return
		}
		su, err := s.subuserUC.Add(r.Context(), body.ServerID, body.Email, body.Permissions)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 201, su)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleSubuser(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r, 2)
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, 400, err); return
		}
		su, err := s.subuserUC.UpdatePermissions(r.Context(), id, body.Permissions)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, su)
	case http.MethodDelete:
		if err := s.subuserUC.Remove(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ── Databases ────────────────────────────────────────────────────────────

func (s *HTTPServer) handleDatabases(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	switch r.Method {
	case http.MethodGet:
		list, err := s.databaseUC.List(r.Context(), serverID)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var body struct {
			ServerID string `json:"server_id"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, 400, err); return
		}
		db, pass, err := s.databaseUC.Create(r.Context(), body.ServerID)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 201, map[string]any{"database": db, "password": pass})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleDatabase(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r, 2)
	switch r.Method {
	case http.MethodDelete:
		if err := s.databaseUC.Delete(r.Context(), id); err != nil {
			writeErr(w, 500, err); return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ── Allocations ───────────────────────────────────────────────────────────

func (s *HTTPServer) handleAllocations(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	switch r.Method {
	case http.MethodGet:
		list, err := s.networkUC.List(r.Context(), serverID)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var body struct {
			ServerID string `json:"server_id"`
			IP       string `json:"ip"`
			Alias    string `json:"alias"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, 400, err); return
		}
		a, err := s.networkUC.Assign(r.Context(), body.ServerID, body.IP, body.Alias)
		if err != nil {
			writeErr(w, 500, err); return
		}
		writeJSON(w, 201, a)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleAllocation(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r, 2)
	switch r.Method {
	case http.MethodDelete:
		if err := s.networkUC.Free(r.Context(), id); err != nil {
			writeErr(w, 500, fmt.Errorf("free allocation: %w", err)); return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
