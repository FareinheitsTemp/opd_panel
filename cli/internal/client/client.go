package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/daemon"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

type Client struct {
	addr string
}

func New() *Client {
	return &Client{addr: daemon.TCPAddr}
}

func (c *Client) send(req ipc.Request) (*ipc.Response, error) {
	conn, err := net.DialTimeout("tcp", c.addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to opd daemon (is it running?): %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}

	var resp ipc.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	if resp.Type == ipc.RespError {
		return nil, fmt.Errorf("%s", resp.Message)
	}
	return &resp, nil
}

func (c *Client) Ping() error {
	_, err := c.send(ipc.Request{Cmd: ipc.CmdPing})
	return err
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
func (c *Client) Remove(id string) (*ipc.Response, error) {
	return c.send(ipc.Request{Cmd: ipc.CmdRemove, ServerID: id})
}

func (c *Client) List() ([]ipc.ServerInfo, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdList})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var servers []ipc.ServerInfo
	return servers, json.Unmarshal(b, &servers)
}

func (c *Client) ListDisk() ([]ipc.DiskServerInfo, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdListDisk})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var servers []ipc.DiskServerInfo
	return servers, json.Unmarshal(b, &servers)
}

func (c *Client) Metrics(id string) (*ipc.MetricsInfo, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdMetrics, ServerID: id})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var m ipc.MetricsInfo
	return &m, json.Unmarshal(b, &m)
}

func (c *Client) Create(cr ipc.CreateRequest) (string, error) {
	b, err := json.Marshal(cr)
	if err != nil {
		return "", err
	}
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdCreate, Payload: string(b)})
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}

func (c *Client) UpdateSettings(us ipc.UpdateSettingsRequest) error {
	b, err := json.Marshal(us)
	if err != nil {
		return err
	}
	_, err = c.send(ipc.Request{Cmd: ipc.CmdUpdateSettings, Payload: string(b)})
	return err
}

func (c *Client) SysStats() (*ipc.SysStats, error) {
	resp, err := c.send(ipc.Request{Cmd: ipc.CmdSysStats})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp.Data)
	var s ipc.SysStats
	return &s, json.Unmarshal(b, &s)
}

// StreamLogs opens a persistent TCP connection and streams log lines.
func (c *Client) StreamLogs(id string) (<-chan string, error) {
	conn, err := net.DialTimeout("tcp", c.addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to opd daemon: %w", err)
	}

	if err := json.NewEncoder(conn).Encode(ipc.Request{Cmd: ipc.CmdStreamLogs, ServerID: id}); err != nil {
		conn.Close()
		return nil, err
	}

	ch := make(chan string, 512)
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
