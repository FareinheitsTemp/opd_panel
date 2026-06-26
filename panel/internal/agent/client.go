package agent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func NewClient(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) sign(body []byte, ts string) string {
	mac := hmac.New(sha256.New, []byte(c.secret))
	mac.Write(body)
	mac.Write([]byte(ts))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf []byte
	var err error
	if body != nil {
		buf, err = json.Marshal(body)
		if err != nil { return nil, err }
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(buf))
	if err != nil { return nil, err }
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Timestamp", ts)
	req.Header.Set("X-Agent-Token", c.sign(buf, ts))
	return c.http.Do(req)
}

func (c *Client) StartServer(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodPost, "/servers/"+id+"/start", nil)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("agent returned %d", resp.StatusCode) }
	return nil
}

func (c *Client) StopServer(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodPost, "/servers/"+id+"/stop", nil)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("agent returned %d", resp.StatusCode) }
	return nil
}

func (c *Client) RestartServer(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodPost, "/servers/"+id+"/restart", nil)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("agent returned %d", resp.StatusCode) }
	return nil
}

func (c *Client) GetStatus(ctx context.Context, id string) (*domain.ServerMetrics, error) {
	resp, err := c.do(ctx, http.MethodGet, "/servers/"+id+"/status", nil)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var m domain.ServerMetrics
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil { return nil, err }
	return &m, nil
}

func (c *Client) SendConsoleCommand(ctx context.Context, id, command string) error {
	body := map[string]string{"command": command}
	resp, err := c.do(ctx, http.MethodPost, "/servers/"+id+"/console", body)
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}

func (c *Client) CreateBackup(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodPost, "/servers/"+id+"/backups", nil)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("agent backup returned %d", resp.StatusCode) }
	return nil
}

func (c *Client) ListBackups(ctx context.Context, id string) ([]map[string]any, error) {
	resp, err := c.do(ctx, http.MethodGet, "/servers/"+id+"/backups", nil)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
	return result, nil
}

func (c *Client) DeleteBackup(ctx context.Context, id, filename string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/servers/"+id+"/backups/"+filename, nil)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("agent delete backup returned %d", resp.StatusCode) }
	return nil
}

func (c *Client) ListDisks(ctx context.Context) ([]map[string]any, error) {
	resp, err := c.do(ctx, http.MethodGet, "/disks", nil)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
	return result, nil
}
