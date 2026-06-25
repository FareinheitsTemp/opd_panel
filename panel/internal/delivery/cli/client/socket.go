package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

type SocketClient struct {
	socketPath string
}

func New(socketPath string) *SocketClient {
	return &SocketClient{socketPath: socketPath}
}

type response struct {
	ID    string         `json:"id"`
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data,omitempty"`
	Error string         `json:"error,omitempty"`
}

// Send sends a single request and returns the parsed response data.
func (c *SocketClient) Send(action string, payload map[string]any) (map[string]any, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to opdd (is it running?): %w", err)
	}
	defer conn.Close()

	req := map[string]any{
		"id":      uuid.NewString(),
		"action":  action,
		"payload": payload,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}

	var resp response
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no response from daemon")
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}
	return resp.Data, nil
}

// Stream opens a persistent connection and calls fn for each streamed line.
func (c *SocketClient) Stream(action string, payload map[string]any, fn func(line string) bool) error {
	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to opdd: %w", err)
	}
	defer conn.Close()

	req := map[string]any{
		"id":      uuid.NewString(),
		"action":  action,
		"payload": payload,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var resp struct {
			Stream bool   `json:"stream"`
			Line   string `json:"line"`
			OK     bool   `json:"ok"`
			Error  string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}
		if !resp.OK && resp.Error != "" {
			return fmt.Errorf(resp.Error)
		}
		if resp.Stream {
			if !fn(resp.Line) {
				return nil
			}
		}
	}
	return scanner.Err()
}
