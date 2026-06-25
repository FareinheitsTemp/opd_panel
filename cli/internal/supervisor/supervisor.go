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

type Supervisor struct {
	mu      sync.RWMutex
	servers map[string]*process.Handle
}

func New() *Supervisor {
	return &Supervisor{servers: make(map[string]*process.Handle)}
}

// RestoreState starts servers that were running before the daemon was last stopped.
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
	s.persistState()
	go s.watch(id, cfg)

	return nil
}

func (s *Supervisor) Stop(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("server %s not found", id)
	}
	// Stop() blocks until process exits or 30s timeout
	err := h.Stop()
	s.persistState()
	return err
}

func (s *Supervisor) Restart(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("server %s not found", id)
	}
	// Stop blocks, then watchdog auto-restarts
	return h.Stop()
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

func (s *Supervisor) List() []ipc.ServerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ipc.ServerInfo
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

func (s *Supervisor) StopAll() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.servers))
	for id := range s.servers {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	for _, id := range ids {
		if err := s.Stop(id); err != nil {
			fmt.Printf("[opd] stop %s: %v\n", id, err)
		}
	}
	// Clear state on clean shutdown
	_ = state.Save([]string{})
}

// persistState saves currently running server IDs to disk.
// Must be called with no locks held (takes RLock internally).
func (s *Supervisor) persistState() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.servers))
	for id, h := range s.servers {
		if h.IsRunning() {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	_ = state.Save(ids)
}

func (s *Supervisor) watch(id string, cfg *config.ServerConfig) {
	for {
		s.mu.RLock()
		h, ok := s.servers[id]
		s.mu.RUnlock()
		if !ok {
			return
		}

		exitCode := h.Wait()

		if h.IntentionallyStopped() {
			fmt.Printf("[opd] %s stopped\n", id)
			s.mu.Lock()
			delete(s.servers, id)
			s.mu.Unlock()
			s.persistState()
			return
		}

		fmt.Printf("[opd] %s crashed (exit %d), restarting in 5s...\n", id, exitCode)
		time.Sleep(5 * time.Second)

		newHandle, err := process.Spawn(cfg)
		if err != nil {
			fmt.Printf("[opd] failed to restart %s: %v\n", id, err)
			s.mu.Lock()
			delete(s.servers, id)
			s.mu.Unlock()
			s.persistState()
			return
		}

		s.mu.Lock()
		s.servers[id] = newHandle
		s.mu.Unlock()
		s.persistState()

		fmt.Printf("[opd] %s restarted (pid %d)\n", id, newHandle.PID())
	}
}
