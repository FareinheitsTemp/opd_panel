package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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

	cors := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h(w, r)
		}
	}

	// Core
	mux.HandleFunc("/api/health", cors(s.handleHealth))
	mux.HandleFunc("/api/disks", cors(s.handleListDisks))
	mux.HandleFunc("/api/settings", cors(s.handleGlobalSettings))

	// Servers
	mux.HandleFunc("/api/servers", cors(s.handleListServers))
	mux.HandleFunc("/api/servers/create", cors(s.handleCreateServer))
	mux.HandleFunc("/api/servers/", cors(s.handleServerRouter))

	// Logs SSE
	mux.HandleFunc("/ws/logs/", s.handleSSELogs)

	s.httpServer = &http.Server{Addr: HTTPAddr, Handler: mux}
	return s
}

func (s *Server) Start() error {
	log.Printf("[opd-http] listening on http://%s", HTTPAddr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown() { s.httpServer.Close() }

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

// GET /api/disks
func (s *Server) handleListDisks(w http.ResponseWriter, r *http.Request) {
	type DiskInfo struct {
		Path  string `json:"path"`
		Label string `json:"label"`
		Free  uint64 `json:"free_gb"`
	}

	var disks []DiskInfo

	if runtime.GOOS == "windows" {
		// Scan drive letters A-Z
		for c := 'A'; c <= 'Z'; c++ {
			root := fmt.Sprintf("%c:\\", c)
			if _, err := os.Stat(root); err == nil {
				free := getDiskFreeGB(root)
				disks = append(disks, DiskInfo{
					Path:  root,
					Label: fmt.Sprintf("Drive %c:", c),
					Free:  free,
				})
			}
		}
	} else {
		// Linux/macOS — return mount points from /proc/mounts or just /
		common := []string{"/", "/home", "/mnt", "/media"}
		for _, p := range common {
			if _, err := os.Stat(p); err == nil {
				disks = append(disks, DiskInfo{
					Path:  p,
					Label: p,
					Free:  getDiskFreeGB(p),
				})
			}
		}
	}

	// Also include current ServersRoot disk
	jsonOK(w, map[string]any{
		"disks":        disks,
		"current_root": config.ServersRoot,
	})
}

// GET/POST /api/settings
func (s *Server) handleGlobalSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jsonOK(w, map[string]string{"servers_root": config.ServersRoot})
		return
	}
	var body struct {
		ServersRoot string `json:"servers_root"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ServersRoot == "" {
		jsonErr(w, 400, "invalid body")
		return
	}
	if err := config.SetServersRoot(body.ServersRoot); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"ok": "true", "servers_root": config.ServersRoot})
}

// --- Server list / create ---

type ServerEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Port        int      `json:"port"`
	PID         int      `json:"pid"`
	RAMUsed     uint64   `json:"ram_used"`
	RAMMax      uint64   `json:"ram_max"`
	CPU         float64  `json:"cpu"`
	Uptime      string   `json:"uptime"`
	Jar         string   `json:"jar"`
	Dir         string   `json:"dir"`
	Motd        string   `json:"motd"`
	MaxPlayers  int      `json:"max_players"`
	Gamemode    string   `json:"gamemode"`
	Difficulty  string   `json:"difficulty"`
	AutoRestart bool     `json:"auto_restart"`
	JavaFlags   []string `json:"java_flags"`
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	running := s.sup.List()
	disk, _ := config.ListAll()

	runningMap := make(map[string]ipc.ServerInfo)
	for _, srv := range running {
		runningMap[srv.ID] = srv
	}

	var entries []ServerEntry
	seen := make(map[string]bool)

	for _, d := range disk {
		cfg, _ := config.Load(d.ID)
		e := ServerEntry{
			ID:     d.ID,
			Name:   d.Name,
			Status: "stopped",
			Port:   d.Port,
			RAMMax: uint64(d.RAMMax),
			Jar:    d.Jar,
			Dir:    filepath.Join(config.ServersRoot, d.ID),
		}
		if cfg != nil {
			e.Motd        = cfg.Motd
			e.MaxPlayers  = cfg.MaxPlayers
			e.Gamemode    = cfg.Gamemode
			e.Difficulty  = cfg.Difficulty
			e.AutoRestart = cfg.AutoRestart
			e.JavaFlags   = cfg.JavaFlags
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
				ID: rv.ID, Name: rv.Name, Status: rv.Status,
				Port: rv.Port, PID: rv.PID, RAMUsed: rv.RAMUsed,
				RAMMax: rv.RAMMax, CPU: float64(rv.CPU),
				Uptime: formatDuration(time.Duration(rv.Uptime)),
				Dir: filepath.Join(config.ServersRoot, rv.ID),
			})
		}
	}

	if entries == nil {
		entries = []ServerEntry{}
	}
	jsonOK(w, entries)
}

type createServerRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Port        int    `json:"port"`
	RAMMinMB    int    `json:"ram_min_mb"`
	RAMMaxMB    int    `json:"ram_max_mb"`
	Jar         string `json:"jar"`
	ServersRoot string `json:"servers_root"` // optional override
}

type serverConfigJSON struct {
	Name        string   `json:"name"`
	Port        int      `json:"port"`
	RAMMinMB    int      `json:"ram_min_mb"`
	RAMMaxMB    int      `json:"ram_max_mb"`
	Jar         string   `json:"jar"`
	JavaFlags   []string `json:"java_flags"`
	Motd        string   `json:"motd"`
	MaxPlayers  int      `json:"max_players"`
	Gamemode    string   `json:"gamemode"`
	Difficulty  string   `json:"difficulty"`
	AutoRestart bool     `json:"auto_restart"`
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, 405, "method not allowed")
		return
	}
	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	if req.ID == "" || strings.ContainsAny(req.ID, " /\\:*?\"<>|") {
		jsonErr(w, 400, "invalid id (alphanumeric, dash, underscore only)")
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		jsonErr(w, 400, "invalid port")
		return
	}
	if req.RAMMinMB <= 0 { req.RAMMinMB = 1024 }
	if req.RAMMaxMB <= 0 { req.RAMMaxMB = 4096 }
	if req.Jar == "" { req.Jar = "server.jar" }
	if req.Name == "" { req.Name = req.ID }

	root := config.ServersRoot
	if req.ServersRoot != "" {
		root = req.ServersRoot
	}

	dir := filepath.Join(root, req.ID)
	configPath := filepath.Join(dir, "opd.json")
	if _, err := os.Stat(configPath); err == nil {
		jsonErr(w, 409, "server already exists")
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}

	// Also create plugins folder
	os.MkdirAll(filepath.Join(dir, "plugins"), 0755)

	cfg := serverConfigJSON{
		Name: req.Name, Port: req.Port,
		RAMMinMB: req.RAMMinMB, RAMMaxMB: req.RAMMaxMB,
		Jar: req.Jar, JavaFlags: []string{},
		MaxPlayers: 20, Gamemode: "survival", Difficulty: "normal",
	}
	f, err := os.Create(configPath)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}

	// If custom root, update global config
	if req.ServersRoot != "" && req.ServersRoot != config.ServersRoot {
		config.SetServersRoot(req.ServersRoot)
	}

	jsonOK(w, map[string]string{"ok": "true", "dir": dir})
}

// --- Server router: /api/servers/:id/:action ---

func (s *Server) handleServerRouter(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/servers/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		jsonErr(w, 400, "missing server id")
		return
	}
	id := parts[0]
	action := ""
	if len(parts) >= 2 {
		action = parts[1]
	}

	switch action {
	case "start", "stop", "restart":
		s.handleServerAction(w, r, id, action)
	case "command":
		s.handleServerCommand(w, r, id)
	case "settings":
		s.handleServerSettings(w, r, id)
	case "plugins":
		s.handlePlugins(w, r, id, parts)
	case "files":
		s.handleFiles(w, r, id, parts)
	case "":
		// GET /api/servers/:id — return single server detail
		s.handleGetServer(w, r, id)
	default:
		jsonErr(w, 404, "unknown action: "+action)
	}
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request, id string) {
	cfg, err := config.Load(id)
	if err != nil {
		jsonErr(w, 404, "server not found")
		return
	}
	jsonOK(w, cfg)
}

func (s *Server) handleServerAction(w http.ResponseWriter, r *http.Request, id, action string) {
	var err error
	switch action {
	case "start":
		err = s.sup.Start(id)
	case "stop":
		err = s.sup.Stop(id)
	case "restart":
		err = s.sup.Restart(id)
	}
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"ok": "true"})
}

func (s *Server) handleServerCommand(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cmd == "" {
		jsonErr(w, 400, "missing cmd")
		return
	}
	if err := s.sup.SendCommand(id, body.Cmd); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"ok": "true"})
}

// GET /api/servers/:id/settings  — returns current opd.json
// PUT /api/servers/:id/settings  — updates opd.json
func (s *Server) handleServerSettings(w http.ResponseWriter, r *http.Request, id string) {
	cfg, err := config.Load(id)
	if err != nil {
		jsonErr(w, 404, "server not found")
		return
	}
	if r.Method == http.MethodGet {
		jsonOK(w, cfg)
		return
	}
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		jsonErr(w, 405, "method not allowed")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(cfg); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	cfg.ID = id
	if err := config.Save(cfg); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"ok": "true"})
}

// --- Plugins CRUD ---
// GET    /api/servers/:id/plugins           — list plugins
// POST   /api/servers/:id/plugins           — upload plugin (multipart)
// DELETE /api/servers/:id/plugins/:filename — delete plugin

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request, id string, parts []string) {
	cfg, err := config.Load(id)
	if err != nil {
		jsonErr(w, 404, "server not found")
		return
	}
	pluginsDir := filepath.Join(cfg.Dir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	switch r.Method {
	case http.MethodGet:
		// List plugins
		entries, err := os.ReadDir(pluginsDir)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		type PluginInfo struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		}
		var plugins []PluginInfo
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jar") {
				continue
			}
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			plugins = append(plugins, PluginInfo{Name: e.Name(), Size: size})
		}
		if plugins == nil {
			plugins = []PluginInfo{}
		}
		jsonOK(w, plugins)

	case http.MethodPost:
		// Upload plugin
		r.ParseMultipartForm(128 << 20) // 128MB max
		file, header, err := r.FormFile("plugin")
		if err != nil {
			jsonErr(w, 400, "missing file field 'plugin'")
			return
		}
		defer file.Close()
		if !strings.HasSuffix(header.Filename, ".jar") {
			jsonErr(w, 400, "only .jar files are allowed")
			return
		}
		dest := filepath.Join(pluginsDir, filepath.Base(header.Filename))
		out, err := os.Create(dest)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		defer out.Close()
		if _, err := io.Copy(out, file); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		jsonOK(w, map[string]string{"ok": "true", "file": header.Filename})

	case http.MethodDelete:
		// DELETE /api/servers/:id/plugins/:filename
		if len(parts) < 3 || parts[2] == "" {
			jsonErr(w, 400, "missing filename")
			return
		}
		filename := filepath.Base(parts[2])
		if !strings.HasSuffix(filename, ".jar") {
			jsonErr(w, 400, "only .jar files can be deleted")
			return
		}
		path := filepath.Join(pluginsDir, filename)
		if err := os.Remove(path); err != nil {
			jsonErr(w, 404, "plugin not found")
			return
		}
		jsonOK(w, map[string]string{"ok": "true"})

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

// --- Files browser ---
// GET /api/servers/:id/files[/:subpath] — list directory or get file content

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request, id string, parts []string) {
	cfg, err := config.Load(id)
	if err != nil {
		jsonErr(w, 404, "server not found")
		return
	}

	sub := ""
	if len(parts) >= 3 {
		sub = strings.Join(parts[2:], "/")
	}

	clean := filepath.Join(cfg.Dir, filepath.Clean("/"+sub))
	if !strings.HasPrefix(clean, cfg.Dir) {
		jsonErr(w, 403, "path traversal not allowed")
		return
	}

	info, err := os.Stat(clean)
	if err != nil {
		jsonErr(w, 404, "not found")
		return
	}

	if info.IsDir() {
		entries, _ := os.ReadDir(clean)
		type FileEntry struct {
			Name  string `json:"name"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size"`
		}
		var files []FileEntry
		for _, e := range entries {
			fi, _ := e.Info()
			sz := int64(0)
			if fi != nil {
				sz = fi.Size()
			}
			files = append(files, FileEntry{Name: e.Name(), IsDir: e.IsDir(), Size: sz})
		}
		if files == nil {
			files = []FileEntry{}
		}
		jsonOK(w, files)
	} else {
		http.ServeFile(w, r, clean)
	}
}

// --- SSE Logs ---

func (s *Server) handleSSELogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/ws/logs/")
	if id == "" {
		http.Error(w, "missing server id", 400)
		return
	}
	ch, err := s.sup.SubscribeLogs(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
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
