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

type Handle struct {
	cfg          *config.ServerConfig
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	logBroadcast *broadcast
	exitCh       chan int
	stopped      atomic.Bool
	statusMu     sync.RWMutex
	status       string
	pid          int
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
		exitCh:       make(chan int, 1),
		pid:          cmd.Process.Pid,
		status:       "starting",
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
		h.statusMu.Unlock()
		h.logBroadcast.close()
		h.exitCh <- code
	}()

	// Mark running only after pipes are set up
	h.statusMu.Lock()
	h.status = "running"
	h.statusMu.Unlock()

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

// Stop sends 'stop' to stdin and waits up to 30s for graceful exit,
// then kills the process forcefully.
func (h *Handle) Stop() error {
	h.stopped.Store(true)
	h.statusMu.Lock()
	h.status = "stopping"
	h.statusMu.Unlock()

	// Send stop command
	_ = h.SendCommand("stop")

	// Wait up to 30s for graceful shutdown
	select {
	case <-h.exitCh:
		// Process exited cleanly — put the code back so Watch() can read it
		// Actually we consumed it, so send 0 back
		h.exitCh <- 0
		return nil
	case <-time.After(30 * time.Second):
		// Force kill
		if h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
		return nil
	}
}

func (h *Handle) Wait() int {
	return <-h.exitCh
}

func (h *Handle) IsRunning() bool {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status == "running" || h.status == "starting"
}

func (h *Handle) IntentionallyStopped() bool { return h.stopped.Load() }
func (h *Handle) Status() string {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status
}
func (h *Handle) PID() int        { return h.pid }
func (h *Handle) Port() int       { return h.cfg.Port }
func (h *Handle) Name() string    { return h.cfg.Name }
func (h *Handle) SubscribeLogs() <-chan string { return h.logBroadcast.subscribe() }

func (h *Handle) Metrics() *ipc.MetricsInfo {
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
