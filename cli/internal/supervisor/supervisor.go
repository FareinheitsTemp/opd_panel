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
	s.saveState()
	go s.watch(id, cfg, 0)

	return nil
}

// Stop gracefully stops a server.
func (s *Supervisor) Stop(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server %s not found", id)
	}

	h.MarkStopped()
	if err := h.Stop(); err != nil {
		return err
	}

	s.mu.Lock()
	s.saveState()
	delete(s.servers, id)
	s.mu.Unlock()

	return nil
}

// Restart gracefully stops then restarts a server.
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

	h.MarkStopped()
	_ = h.Stop()

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
	// BUG FIX #10: launch watch() while mu is still held, matching Start().
	// Launching it after Unlock() left a window where a concurrent Restart()
	// or Stop() could register a second watchdog for the same server ID.
	go s.watch(id, cfg, 0)
	s.mu.Unlock()

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

// saveState persists currently running IDs.
// MUST be called while mu is already held — does NOT lock itself.
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
// BUG FIX #3: the respawn attempt counter was never reset after a successful
// run. A server that crashed → restarted → ran for hours → crashed again
// would still be on attempt N+1 of 5. Fix: reset counter after a
// configurable "stable" duration (30s), meaning the process is considered
// healthy and subsequent crashes start fresh.
func (s *Supervisor) watch(id string, cfg *config.ServerConfig, attempts int) {
	const stableAfter = 30 * time.Second

	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return
	}

	// Reset attempt counter if the process survives stableAfter.
	if attempts > 0 {
		select {
		case <-time.After(stableAfter):
			attempts = 0
		case <-h.Done():
			// Exited before stable — fall through with current attempts
		}
	}

	// Block until process exits (if not already done from above).
	select {
	case <-h.Done():
	default:
		<-h.Done()
	}

	if h.IntentionallyStopped() {
		return
	}

	if attempts >= maxRespawnAttempts {
		fmt.Printf("[opd] %s crashed %d times consecutively, giving up\n", id, attempts)
		s.mu.Lock()
		delete(s.servers, id)
		s.saveState()
		s.mu.Unlock()
		return
	}

	delay := time.Duration(1<<uint(attempts)) * time.Second
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
	go s.watch(id, cfg, attempts+1)
}
