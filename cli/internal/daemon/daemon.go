package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
)

const SocketPath = "/run/opd.sock"

type Daemon struct {
	listener   net.Listener
	supervisor *supervisor.Supervisor
	mu         sync.Mutex
}

func New() (*Daemon, error) {
	// Remove stale socket
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
	return &Daemon{listener: l, supervisor: sup}, nil
}

func (d *Daemon) ListenAndServe() error {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			return err
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) Shutdown() error {
	d.supervisor.StopAll()
	return d.listener.Close()
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	var req ipc.Request
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&req); err != nil {
		writeError(conn, "invalid request")
		return
	}

	enc := json.NewEncoder(conn)

	switch req.Cmd {
	case ipc.CmdList:
		servers := d.supervisor.List()
		_ = enc.Encode(ipc.Response{Type: ipc.RespData, Data: servers})

	case ipc.CmdStart:
		if err := d.supervisor.Start(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		_ = enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("starting %s", req.ServerID)})

	case ipc.CmdStop:
		if err := d.supervisor.Stop(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		_ = enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("stopping %s", req.ServerID)})

	case ipc.CmdRestart:
		if err := d.supervisor.Restart(req.ServerID); err != nil {
			writeError(conn, err.Error())
			return
		}
		_ = enc.Encode(ipc.Response{Type: ipc.RespOK, Message: fmt.Sprintf("restarting %s", req.ServerID)})

	case ipc.CmdSendCommand:
		if err := d.supervisor.SendCommand(req.ServerID, req.Payload); err != nil {
			writeError(conn, err.Error())
			return
		}
		_ = enc.Encode(ipc.Response{Type: ipc.RespOK, Message: "sent"})

	case ipc.CmdMetrics:
		m, err := d.supervisor.Metrics(req.ServerID)
		if err != nil {
			writeError(conn, err.Error())
			return
		}
		_ = enc.Encode(ipc.Response{Type: ipc.RespData, Data: m})

	case ipc.CmdStreamLogs:
		// Keep connection open and stream log lines
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
	enc := json.NewEncoder(conn)
	_ = enc.Encode(ipc.Response{Type: ipc.RespError, Message: msg})
}
