package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	gops "github.com/shirou/gopsutil/v3/process"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
)

// Handle owns a single running Java process.
type Handle struct {
	cfg          *config.ServerConfig
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	logBroadcast *broadcast

	// doneCh is closed exactly once when the process exits.
	doneCh   chan struct{}
	exitCode int

	// stopped is set to true ONLY by an intentional MarkStopped() call.
	stopped atomic.Bool

	statusMu sync.RWMutex
	status   string
	pid      int
}

func Spawn(cfg *config.ServerConfig) (*Handle, error) {
	args := buildJavaArgs(cfg)
	cmd := exec.Command("java", args...)
	cmd.Dir = cfg.Dir
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start java: %w", err)
	}

	h := &Handle{
		cfg:          cfg,
		cmd:          cmd,
		stdin:        stdin,
		logBroadcast: newBroadcast(),
		doneCh:       make(chan struct{}),
		pid:          cmd.Process.Pid,
		// BUG FIX #1: set status to "running" HERE, before goroutines are
		// launched, not after. The previous code set it after — creating a
		// window where IsRunning() returned false for a live process.
		status: "running",
	}

	go h.pipeReader(stdout)
	go h.pipeReader(stderr)

	go func() {
		err := cmd.Wait()
		code := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			}
		}
		h.statusMu.Lock()
		if h.stopped.Load() {
			h.status = "stopped"
		} else {
			h.status = "crashed"
		}
		h.exitCode = code
		h.statusMu.Unlock()

		h.logBroadcast.close()
		close(h.doneCh)
	}()

	return h, nil
}

func (h *Handle) pipeReader(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		h.logBroadcast.send(scanner.Text())
	}
}

func (h *Handle) SendCommand(cmd string) error {
	_, err := fmt.Fprintf(h.stdin, "%s\n", cmd)
	return err
}

// Stop sends 'stop' to stdin and waits up to 30 s for graceful exit,
// then kills the process. Does NOT set h.stopped — caller's responsibility.
func (h *Handle) Stop() error {
	h.statusMu.Lock()
	h.status = "stopping"
	h.statusMu.Unlock()

	_ = h.SendCommand("stop")

	select {
	case <-h.doneCh:
		return nil
	case <-time.After(30 * time.Second):
		if h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
		<-h.doneCh
		return nil
	}
}

// Done returns a channel closed when the process exits.
func (h *Handle) Done() <-chan struct{} { return h.doneCh }

// ExitCode is safe to read only after Done() is closed.
func (h *Handle) ExitCode() int { return h.exitCode }

func (h *Handle) MarkStopped()              { h.stopped.Store(true) }
func (h *Handle) IntentionallyStopped() bool { return h.stopped.Load() }

func (h *Handle) IsRunning() bool {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status == "running" || h.status == "starting"
}

func (h *Handle) Status() string {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status
}

func (h *Handle) PID() int                    { return h.pid }
func (h *Handle) Port() int                   { return h.cfg.Port }
func (h *Handle) Name() string                { return h.cfg.Name }
func (h *Handle) SubscribeLogs() <-chan string { return h.logBroadcast.subscribe() }

func (h *Handle) Metrics() *ipc.MetricsInfo {
	// BUG FIX #2: if the process is already stopped, gopsutil will query
	// a dead PID which can match a recycled OS process — return zeros instead.
	if !h.IsRunning() {
		return &ipc.MetricsInfo{
			ServerID: h.cfg.ID,
			RAMMax:   uint64(h.cfg.RAMMaxMB) * 1024 * 1024,
		}
	}

	p, err := gops.NewProcess(int32(h.pid))
	if err != nil {
		return &ipc.MetricsInfo{
			ServerID: h.cfg.ID,
			RAMMax:   uint64(h.cfg.RAMMaxMB) * 1024 * 1024,
		}
	}
	cpu, _ := p.CPUPercent()
	mem, _ := p.MemoryInfo()
	created, _ := p.CreateTime()
	var ramUsed uint64
	if mem != nil {
		ramUsed = mem.RSS
	}
	var uptime uint64
	if created > 0 {
		uptime = uint64(time.Now().UnixMilli()/1000) - uint64(created/1000)
	}
	return &ipc.MetricsInfo{
		ServerID: h.cfg.ID,
		PID:      uint32(h.pid),
		RAMUsed:  ramUsed,
		RAMMax:   uint64(h.cfg.RAMMaxMB) * 1024 * 1024,
		CPU:      float32(cpu),
		Uptime:   uptime,
	}
}

func buildJavaArgs(cfg *config.ServerConfig) []string {
	args := []string{
		fmt.Sprintf("-Xms%dM", cfg.RAMMinMB),
		fmt.Sprintf("-Xmx%dM", cfg.RAMMaxMB),
		"-XX:+UseG1GC",
		"-XX:+ParallelRefProcEnabled",
		"-XX:MaxGCPauseMillis=200",
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+DisableExplicitGC",
		"-XX:G1NewSizePercent=30",
		"-XX:G1MaxNewSizePercent=40",
		"-XX:G1HeapRegionSize=8M",
		"-XX:G1ReservePercent=20",
		"-XX:G1HeapWastePercent=5",
		"-XX:G1MixedGCCountTarget=4",
		"-XX:InitiatingHeapOccupancyPercent=15",
		"-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:G1RSetUpdatingPauseTimePercent=5",
		"-XX:SurvivorRatio=32",
		"-XX:+PerfDisableSharedMem",
		"-XX:MaxTenuringThreshold=1",
	}
	args = append(args, cfg.JavaFlags...)
	args = append(args, "-jar", cfg.JarPath, "nogui")
	return args
}
