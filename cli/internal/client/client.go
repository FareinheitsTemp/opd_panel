package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

// Client sends IPC requests over the Unix socket to the daemon.
type Client struct {
	socketPath string
}

func New() *Client {
	return &Client{socketPath: daemon.SocketPath}
}

func (c *Client) send(req ipc.Request) (*ipc.Response, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to opd daemon (is it running?): %w", err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		return nil, err
	}

	var resp ipc.Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}

	if resp.Type == ipc.RespError {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func (c *Client) Start(id string) (*ipc.Response, error) {
	return c.send(ipc.Request{Cmd: ipc.CmdStart, ServerID: id})
}

func (c *Client) Stop(id string) (*ipc.Response, error) {
	return c.send(ipc.Request{Cmd: ipc.CmdStop, ServerID: id})
}

func (c *Client) Restart(id string) (*ipc.Response, error) {
	return c.send(ipc.Request{Cmd: ipc.CmdRestart, ServerID: id})
}

func (c *Client) SendCommand(id, command string) (*ipc.Response, error) {
	return c.send(ipc.Request{Cmd: ipc.CmdSendCommand, ServerID: id, Payload: command})
}

func (c *Client) List() ([]ipc.ServerInfo, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdList})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var servers []ipc.ServerInfo
	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func (c *Client) Metrics(id string) (*ipc.MetricsInfo, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdMetrics, ServerID: id})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var m ipc.MetricsInfo
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// StreamLogs opens a persistent connection and streams log lines over a channel.
func (c *Client) StreamLogs(id string) (<-chan string, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to opd daemon: %w", err)
	}

	req := ipc.Request{Cmd: ipc.CmdStreamLogs, ServerID: id}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		conn.Close()
		return nil, err
	}

	ch := make(chan string, 256)
	go func() {
		defer conn.Close()
		defer close(ch)
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			var resp ipc.Response
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				continue
			}
			if resp.Type == ipc.RespLog {
				ch <- resp.Message
			}
		}
	}()

	return ch, nil
}
