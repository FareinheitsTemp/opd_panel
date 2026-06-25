package supervisor

import (
	"fmt"
	"sync"
	"time"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/process"
)

type Supervisor struct {
	mu      sync.RWMutex
	servers map[string]*process.Handle
}

func New() *Supervisor {
	return &Supervisor{servers: make(map[string]*process.Handle)}
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
	return h.Stop()
}

func (s *Supervisor) Restart(id string) error {
	s.mu.RLock()
	h, ok := s.servers[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("server %s not found", id)
	}
	if err := h.Stop(); err != nil {
		return err
	}
	// watchdog буде чекати виходу і перезапустить
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
	defer s.mu.RUnlock()
	for _, h := range s.servers {
		_ = h.Stop()
	}
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
			return
		}

		s.mu.Lock()
		s.servers[id] = newHandle
		s.mu.Unlock()

		fmt.Printf("[opd] %s restarted (pid %d)\n", id, newHandle.PID())
	}
}
