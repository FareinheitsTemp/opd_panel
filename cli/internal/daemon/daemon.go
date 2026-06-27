package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/httpapi"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// TCPAddr is the address the IPC daemon listens on.
const TCPAddr = "127.0.0.1:51200"

type Daemon struct {
	listener   net.Listener
	supervisor *supervisor.Supervisor
	httpSrv    *httpapi.Server
	mu         sync.Mutex
	shutdown   bool
	connWg     sync.WaitGroup
}

func New() (*Daemon, error) {
	l, err := net.Listen("tcp", TCPAddr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", TCPAddr, err)
	}

	sup := supervisor.New()
	sup.RestoreState()

	httpSrv := httpapi.New(sup)

	return &Daemon{
		listener:   l,
		supervisor: sup,
		httpSrv:    httpSrv,
	}, nil
}

func (d *Daemon) ListenAndServe() error {
	go func() {
		if err := d.httpSrv.Start(); err != nil {
			fmt.Printf("[opd-http] error: %v\n", err)
		}
	}()

	for {
		conn, err := d.listener.Accept()
		if err != nil {
			d.mu.Lock()
			shutdown := d.shutdown
			d.mu.Unlock()
			if shutdown {
				return nil
			}
			return err
		}
		d.connWg.Add(1)
		go func() {
			defer d.connWg.Done()
			d.handleConn(conn)
		}()
	}
}

func (d *Daemon) Shutdown() error {
	d.mu.Lock()
	d.shutdown = true
	d.mu.Unlock()

	d.httpSrv.Shutdown()
	listenerErr := d.listener.Close()
	d.supervisor.StopAll()
	d.connWg.Wait()

	return listenerErr
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	var req ipc.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		writeError(conn, "invalid request: "+err.Error())
		return
	}

	enc := json.NewEncoder(conn)

	switch req.Cmd {
	case ipc.CmdPing:
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: "pong"})

	case ipc.CmdList:
		enc.Encode(ipc.Response{Type: ipc.RespData, Data: d.supervisor.List()})

	case ipc.CmdListDisk:
		servers, err := config.ListAll()
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespData, Data: servers})

	case ipc.CmdStart:
		if err := d.supervisor.Start(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("starting %s", req.ServerID)})

	case ipc.CmdStop:
		if err := d.supervisor.Stop(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("%s stopped", req.ServerID)})

	case ipc.CmdRestart:
		if err := d.supervisor.Restart(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("restarting %s", req.ServerID)})

	case ipc.CmdSendCommand:
		if err := d.supervisor.SendCommand(req.ServerID, req.Payload); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: "sent"})

	case ipc.CmdMetrics:
		m, err := d.supervisor.Metrics(req.ServerID)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespData, Data: m})

	case ipc.CmdRemove:
		if d.supervisor.IsRunning(req.ServerID) {
			writeError(conn, fmt.Sprintf("server %s is running — stop it first", req.ServerID))
			return
		}
		if err := config.Remove(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("%s removed", req.ServerID)})

	case ipc.CmdCreate:
		var cr ipc.CreateRequest
		if err := json.Unmarshal([]byte(req.Payload), &cr); err != nil {
			writeError(conn, "invalid create payload: "+err.Error())
			return
		}
		if cr.ID == "" {
			writeError(conn, "server id is required")
			return
		}
		cfg := &config.ServerConfig{
			ID:       cr.ID,
			Name:     cr.Name,
			Port:     cr.Port,
			RAMMinMB: cr.RAMMinMB,
			RAMMaxMB: cr.RAMMaxMB,
			Jar:      cr.Jar,
		}
		if cfg.Name == "" {
			cfg.Name = cfg.ID
		}
		if cfg.Jar == "" {
			cfg.Jar = "server.jar"
		}
		if cfg.Port == 0 {
			cfg.Port = 25565
		}
		if cfg.RAMMinMB == 0 {
			cfg.RAMMinMB = 512
		}
		if cfg.RAMMaxMB == 0 {
			cfg.RAMMaxMB = 2048
		}
		dir, err := config.Create(cfg)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: dir})

	case ipc.CmdUpdateSettings:
		var us ipc.UpdateSettingsRequest
		if err := json.Unmarshal([]byte(req.Payload), &us); err != nil {
			writeError(conn, "invalid settings payload: "+err.Error())
			return
		}
		cfg, err := config.Load(us.ServerID)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		if us.Name != "" {
			cfg.Name = us.Name
		}
		if us.Port != 0 {
			cfg.Port = us.Port
		}
		if us.RAMMaxMB != 0 {
			cfg.RAMMaxMB = us.RAMMaxMB
		}
		if us.Jar != "" {
			cfg.Jar = us.Jar
		}
		cfg.JavaFlags = us.JavaFlags
		cfg.AutoRestart = us.AutoRestart
		if err := config.Save(cfg); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: "settings saved"})

	case ipc.CmdSysStats:
		var stats ipc.SysStats
		if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
			stats.CPUPercent = percents[0]
		}
		if vm, err := mem.VirtualMemory(); err == nil {
			stats.RAMUsedMB = vm.Used / 1024 / 1024
			stats.RAMTotalMB = vm.Total / 1024 / 1024
		}
		if du, err := disk.Usage(config.ServersRoot); err == nil {
			stats.DiskUsedGB = float64(du.Used) / 1024 / 1024 / 1024
			stats.DiskTotalGB = float64(du.Total) / 1024 / 1024 / 1024
		}
		enc.Encode(ipc.Response{Type: ipc.RespData, Data: stats})

	case ipc.CmdStreamLogs:
		ch, err := d.supervisor.SubscribeLogs(req.ServerID)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		for line := range ch {
			if err := enc.Encode(ipc.Response{Type: ipc.RespLog, Message: line}); err != nil {
				return
			}
		}

	default:
		writeError(conn, "unknown command: "+req.Cmd)
	}
}

func writeError(conn net.Conn, msg string) {
	json.NewEncoder(conn).Encode(ipc.Response{Type: ipc.RespError, Message: msg})
}
