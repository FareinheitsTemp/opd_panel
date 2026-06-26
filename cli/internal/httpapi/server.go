package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
)

const HTTPAddr = "127.0.0.1:51201"

type Server struct {
	sup        *supervisor.Supervisor
	httpServer *http.Server
}

func New(sup *supervisor.Supervisor) *Server {
	s := &Server{sup: sup}

	mux := http.NewServeMux()

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
	mux.HandleFunc("/api/servers/create", withCORS(s.handleCreateServer))
	mux.HandleFunc("/api/servers/", withCORS(s.handleServerAction))
	mux.HandleFunc("/ws/logs/", s.handleSSELogs)
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

type ServerEntry struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	Port    int     `json:"port"`
	PID     int     `json:"pid"`
	RAMUsed uint64  `json:"ram_used"`
	RAMMax  uint64  `json:"ram_max"`
	CPU     float64 `json:"cpu"`
	Uptime  string  `json:"uptime"`
	Jar     string  `json:"jar"`
}

// GET /api/servers
func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	running := s.sup.List()
	disk, _ := config.ListAll()

	runningMap := make(map[string]ipc.ServerInfo)
	for _, srv := range running {
		runningMap[srv.ID] = srv
	}

	var entries []ServerEntry
	seen := make(map[string]bool)

	for _, d := range disk {
		e := ServerEntry{
			ID:     d.ID,
			Name:   d.Name,
			Status: "stopped",
			Port:   d.Port,
			RAMMax: uint64(d.RAMMax),
			Jar:    d.Jar,
		}
		if rv, ok := runningMap[d.ID]; ok {
			e.Status  = rv.Status
			e.PID     = rv.PID
			e.RAMUsed = rv.RAMUsed
			e.RAMMax  = rv.RAMMax
			e.CPU     = float64(rv.CPU)
			e.Uptime  = formatDuration(time.Duration(rv.Uptime))
		}
		entries = append(entries, e)
		seen[d.ID] = true
	}

	for _, rv := range running {
		if !seen[rv.ID] {
			entries = append(entries, ServerEntry{
				ID:      rv.ID,
				Name:    rv.Name,
				Status:  rv.Status,
				Port:    rv.Port,
				PID:     rv.PID,
				RAMUsed: rv.RAMUsed,
				RAMMax:  rv.RAMMax,
				CPU:     float64(rv.CPU),
				Uptime:  formatDuration(time.Duration(rv.Uptime)),
			})
		}
	}

	if entries == nil {
		entries = []ServerEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

// POST /api/servers/create
type createServerRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Port     int    `json:"port"`
	RAMMinMB int    `json:"ram_min_mb"`
	RAMMaxMB int    `json:"ram_max_mb"`
	Jar      string `json:"jar"`
}

type serverConfigJSON struct {
	Name      string   `json:"name"`
	Port      int      `json:"port"`
	RAMMinMB  int      `json:"ram_min_mb"`
	RAMMaxMB  int      `json:"ram_max_mb"`
	Jar       string   `json:"jar"`
	JavaFlags []string `json:"java_flags"`
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.ID == "" || strings.ContainsAny(req.ID, " /\\:*?\"<>|") {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		http.Error(w, `{"error":"invalid port"}`, http.StatusBadRequest)
		return
	}
	if req.RAMMinMB <= 0 { req.RAMMinMB = 1024 }
	if req.RAMMaxMB <= 0 { req.RAMMaxMB = 4096 }
	if req.Jar == "" { req.Jar = "server.jar" }
	if req.Name == "" { req.Name = req.ID }

	dir := filepath.Join(config.ServersRoot, req.ID)
	configPath := filepath.Join(dir, "opd.json")

	if _, err := os.Stat(configPath); err == nil {
		http.Error(w, `{"error":"server already exists"}`, http.StatusConflict)
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	cfg := serverConfigJSON{
		Name:      req.Name,
		Port:      req.Port,
		RAMMinMB:  req.RAMMinMB,
		RAMMaxMB:  req.RAMMaxMB,
		Jar:       req.Jar,
		JavaFlags: []string{},
	}

	f, err := os.Create(configPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"ok":  "true",
		"dir": dir,
	})
}

// POST /api/servers/:id/start|stop|restart|command
func (s *Server) handleServerAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

// GET /ws/logs/:id — SSE
func (s *Server) handleSSELogs(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

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
