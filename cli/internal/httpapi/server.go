// Package httpapi provides the HTTP REST + WebSocket API for the OPD web UI.
// It listens on 127.0.0.1:51201 (separate from the IPC port 51200).
package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
)

const HTTPAddr = "127.0.0.1:51201"

type Server struct {
	sup *supervisor.Supervisor
	httpServer *http.Server
}

func New(sup *supervisor.Supervisor) *Server {
	s := &Server{sup: sup}

	mux := http.NewServeMux()

	// CORS middleware wrapper
	withCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h(w, r)
		}
	}

	mux.HandleFunc("/api/servers", withCORS(s.handleListServers))
	mux.HandleFunc("/api/servers/", withCORS(s.handleServerAction))
	mux.HandleFunc("/ws/logs/", s.handleWsLogs)
	mux.HandleFunc("/api/health", withCORS(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	s.httpServer = &http.Server{
		Addr:    HTTPAddr,
		Handler: mux,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("[opd-http] listening on http://%s", HTTPAddr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown() {
	s.httpServer.Close()
}

// GET /api/servers — list all servers (running + disk)
func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	running := s.sup.List()
	disk, _ := config.ListAll()

	type ServerEntry struct {
		ID      string  `json:"id"`
		Name    string  `json:"name"`
		Status  string  `json:"status"`
		Port    int     `json:"port"`
		PID     int     `json:"pid"`
		RAMUsed uint64  `json:"ram_used"`
		RAMMax  int     `json:"ram_max"`
		CPU     float64 `json:"cpu"`
		Uptime  string  `json:"uptime"`
		Jar     string  `json:"jar"`
	}

	// Index running servers for quick lookup
	runningMap := make(map[string]ipc.ServerInfo)
	for _, srv := range running {
		runningMap[srv.ID] = srv
	}

	// Merge disk list with running info
	var entries []ServerEntry
	seen := make(map[string]bool)

	for _, d := range disk {
		e := ServerEntry{
			ID:     d.ID,
			Name:   d.Name,
			Status: "stopped",
			Port:   d.Port,
			RAMMax: d.RAMMax,
			Jar:    d.Jar,
		}
		if r, ok := runningMap[d.ID]; ok {
			e.Status = r.Status
			e.PID = r.PID
			e.RAMUsed = r.RAMUsed
			e.CPU = r.CPU
			if r.Uptime > 0 {
				e.Uptime = formatDuration(r.Uptime)
			}
		}
		entries = append(entries, e)
		seen[d.ID] = true
	}

	// Add running servers that might not be on disk list (edge case)
	for _, r := range running {
		if !seen[r.ID] {
			entries = append(entries, ServerEntry{
				ID:      r.ID,
				Name:    r.Name,
				Status:  r.Status,
				Port:    r.Port,
				PID:     r.PID,
				RAMUsed: r.RAMUsed,
				RAMMax:  r.RAMMax,
				CPU:     r.CPU,
				Uptime:  formatDuration(r.Uptime),
			})
		}
	}

	if entries == nil {
		entries = []ServerEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

// POST /api/servers/:id/start|stop|restart|command
func (s *Server) handleServerAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/servers/:id/:action
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/servers/"), "/")
	if len(parts) < 2 {
		http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
		return
	}
	id := parts[0]
	action := parts[1]

	var errMsg string
	switch action {
	case "start":
		if err := s.sup.Start(id); err != nil {
			errMsg = err.Error()
		}
	case "stop":
		if err := s.sup.Stop(id); err != nil {
			errMsg = err.Error()
		}
	case "restart":
		if err := s.sup.Restart(id); err != nil {
			errMsg = err.Error()
		}
	case "command":
		var body struct {
			Cmd string `json:"cmd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cmd == "" {
			http.Error(w, `{"error":"missing cmd"}`, http.StatusBadRequest)
			return
		}
		if err := s.sup.SendCommand(id, body.Cmd); err != nil {
			errMsg = err.Error()
		}
	default:
		http.Error(w, fmt.Sprintf(`{"error":"unknown action: %s"}`, action), http.StatusBadRequest)
		return
	}

	if errMsg != "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// WebSocket log streaming — /ws/logs/:id
// Uses a simple line-based protocol over raw HTTP upgrade (no external ws lib needed).
func (s *Server) handleWsLogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/ws/logs/")
	if id == "" {
		http.Error(w, "missing server id", http.StatusBadRequest)
		return
	}

	ch, err := s.sup.SubscribeLogs(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Upgrade to SSE (Server-Sent Events) — simpler than WebSocket, works everywhere
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Use a done channel to detect client disconnect
	var once sync.Once
	done := make(chan struct{})
	go func() {
		<-r.Context().Done()
		once.Do(func() { close(done) })
	}()

	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return
			}
			// SSE format: "data: <line>\n\n"
			fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", " "))
			flusher.Flush()
		case <-done:
			return
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
