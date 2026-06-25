package supervisor

import (
	"fmt"
	"sync"
	"time"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/process"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/state"
)

const maxRespawnAttempts = 5

type Supervisor struct {
	mu      sync.RWMutex
	servers map[string]*process.Handle
}

func New() *Supervisor {
	return &Supervisor{servers: make(map[string]*process.Handle)}
}

// RestoreState starts servers that were running before the daemon last stopped.
func (s *Supervisor) RestoreState() {
	ids, err := state.Load()
	if err != nil || len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if err := s.Start(id); err != nil {
			fmt.Printf("[opd] restore %s: %v\n", id, err)
		} else {
			fmt.Printf("[opd] restored %s\n", id)
		}
	}
}

func (s *Supervisor) Start(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if h, ok := s.servers[id]; ok && h.IsRunning() {
		return fmt.Errorf("server %s is already running", id)
	}

	cfg, err := config.Load(id)
	if err != nil {
		return fmt.Errorf("load config for %s: %w", id, err)
	}

	h, err := process.Spawn(cfg)
	if err != nil {
		return fmt.Errorf("spawn %s: %w", id, err)
	}

	s.servers[id] = h
	s.saveState() // safe: called under mu.Lock() — saveState does NOT lock
	go s.watch(id, cfg, 0)

	return nil
}

// Stop gracefully stops a server. Marks it as intentionally stopped so
// watch() does not attempt a respawn.
func (s *Supervisor) Stop(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server %s not found", id)
	}

	// Mark stopped BEFORE calling h.Stop() so the goroutine in watch()
	// sees the flag when doneCh fires.
	h.MarkStopped()
	if err := h.Stop(); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.servers, id)
	s.saveState()
	s.mu.Unlock()

	return nil
}

// Restart gracefully stops then restarts a server.
// Critically: we do NOT set h.MarkStopped() so watch() will respawn,
// but we also don't rely on the old watch() — we stop it cleanly and
// start a fresh one to avoid double-watch races.
func (s *Supervisor) Restart(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server %s not found", id)
	}

	cfg, err := config.Load(id)
	if err != nil {
		return err
	}

	// Stop old process cleanly — mark it intentionally stopped
	// so the OLD watch() goroutine exits.
	h.MarkStopped()
	_ = h.Stop()

	// Spawn fresh process
	newH, err := process.Spawn(cfg)
	if err != nil {
		s.mu.Lock()
		delete(s.servers, id)
		s.saveState()
		s.mu.Unlock()
		return fmt.Errorf("restart %s: %w", id, err)
	}

	s.mu.Lock()
	s.servers[id] = newH
	s.saveState()
	s.mu.Unlock()

	go s.watch(id, cfg, 0)
	return nil
}

func (s *Supervisor) SendCommand(id, cmd string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("server %s not found", id)
	}
	return h.SendCommand(cmd)
}

func (s *Supervisor) Metrics(id string) (*ipc.MetricsInfo, error) {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("server %s not found", id)
	}
	return h.Metrics(), nil
}

func (s *Supervisor) SubscribeLogs(id string) (<-chan string, error) {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("server %s not found", id)
	}
	return h.SubscribeLogs(), nil
}

func (s *Supervisor) IsRunning(id string) bool {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	return ok && h.IsRunning()
}

func (s *Supervisor) List() []ipc.ServerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ipc.ServerInfo, 0, len(s.servers))
	for id, h := range s.servers {
		m := h.Metrics()
		out = append(out, ipc.ServerInfo{
			ID:      id,
			Name:    h.Name(),
			Status:  h.Status(),
			PID:     h.PID(),
			Port:    h.Port(),
			RAMUsed: m.RAMUsed,
			RAMMax:  m.RAMMax,
			CPU:     m.CPU,
			Uptime:  m.Uptime,
		})
	}
	return out
}

// StopAll stops every running server concurrently and clears state.
func (s *Supervisor) StopAll() {
	// Snapshot IDs under lock, then release before blocking Stop() calls.
	s.mu.RLock()
	ids := make([]string, 0, len(s.servers))
	for id := range s.servers {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := s.Stop(id); err != nil {
				fmt.Printf("[opd] stop %s: %v\n", id, err)
			}
		}(id)
	}
	wg.Wait()

	_ = state.Save([]string{})
}

// saveState persists currently running IDs to disk.
// MUST be called while mu is already held (Lock or RLock) — does NOT lock itself.
func (s *Supervisor) saveState() {
	ids := make([]string, 0, len(s.servers))
	for id, h := range s.servers {
		if h.IsRunning() {
			ids = append(ids, id)
		}
	}
	_ = state.Save(ids)
}

// watch is the watchdog goroutine for a single server.
// attempts tracks consecutive respawn failures for exponential backoff.
func (s *Supervisor) watch(id string, cfg *config.ServerConfig, attempts int) {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return
	}

	// Block until the process exits.
	<-h.Done()

	if h.IntentionallyStopped() {
		// Clean stop — supervisor.Stop() already cleaned up.
		return
	}

	// Crash — attempt respawn with capped exponential backoff.
	if attempts >= maxRespawnAttempts {
		fmt.Printf("[opd] %s crashed %d times consecutively, giving up\n", id, attempts)
		s.mu.Lock()
		delete(s.servers, id)
		s.saveState()
		s.mu.Unlock()
		return
	}

	delay := time.Duration(1<<uint(attempts)) * time.Second // 1s, 2s, 4s, 8s, 16s
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	fmt.Printf("[opd] %s crashed (exit %d), restarting in %s (attempt %d/%d)...\n",
		id, h.ExitCode(), delay, attempts+1, maxRespawnAttempts)
	time.Sleep(delay)

	newHandle, err := process.Spawn(cfg)
	if err != nil {
		fmt.Printf("[opd] failed to respawn %s: %v\n", id, err)
		s.mu.Lock()
		delete(s.servers, id)
		s.saveState()
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.servers[id] = newHandle
	s.saveState()
	s.mu.Unlock()

	fmt.Printf("[opd] %s restarted (pid %d)\n", id, newHandle.PID())
	go s.watch(id, cfg, attempts+1) // new goroutine for new handle
}
