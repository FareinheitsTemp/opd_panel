package socket

// Request is sent by the CLI to the daemon over Unix socket.
type Request struct {
	ID      string         `json:"id"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Response is sent by the daemon back to the CLI.
type Response struct {
	ID     string         `json:"id"`
	OK     bool           `json:"ok"`
	Data   map[string]any `json:"data,omitempty"`
	Error  string         `json:"error,omitempty"`
	Stream bool           `json:"stream,omitempty"`
	Line   string         `json:"line,omitempty"`
}
