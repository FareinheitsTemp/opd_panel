package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
)

const SocketPath = "/run/opd.sock"

type Daemon struct {
	listener   net.Listener
	supervisor *supervisor.Supervisor
	mu         sync.Mutex
	shutdown   bool
}

func New() (*Daemon, error) {
	_ = os.Remove(SocketPath)

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", SocketPath, err)
	}
	if err := os.Chmod(SocketPath, 0660); err != nil {
		l.Close()
		return nil, err
	}

	sup := supervisor.New()
	sup.RestoreState()

	return &Daemon{listener: l, supervisor: sup}, nil
}

func (d *Daemon) ListenAndServe() error {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			d.mu.Lock()
			shutdown := d.shutdown
			d.mu.Unlock()
			if shutdown {
				// Clean shutdown — not an error
				return nil
			}
			return err
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) Shutdown() error {
	d.mu.Lock()
	d.shutdown = true
	d.mu.Unlock()

	d.supervisor.StopAll()
	return d.listener.Close()
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
		// Safety check: refuse if server is currently running.
		if d.supervisor.IsRunning(req.ServerID) {
			writeError(conn, fmt.Sprintf("server %s is running — stop it first", req.ServerID))
			return
		}
		if err := config.Remove(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("%s removed", req.ServerID)})

	case ipc.CmdStreamLogs:
		ch, err := d.supervisor.SubscribeLogs(req.ServerID)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		for line := range ch {
			if err := enc.Encode(ipc.Response{Type: ipc.RespLog, Message: line}); err != nil {
				return // client disconnected
			}
		}

	default:
		writeError(conn, "unknown command: "+req.Cmd)
	}
}

func writeError(conn net.Conn, msg string) {
	json.NewEncoder(conn).Encode(ipc.Response{Type: ipc.RespError, Message: msg})
}
