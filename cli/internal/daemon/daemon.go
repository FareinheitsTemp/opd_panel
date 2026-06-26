package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/supervisor/config"
)

// TCPAddr is the address the daemon listens on.
// Using TCP instead of Unix sockets for cross-platform compatibility (Windows).
const TCPAddr = "127.0.0.1:51200"

type Daemon struct {
	listener   net.Listener
	supervisor *supervisor.Supervisor
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
