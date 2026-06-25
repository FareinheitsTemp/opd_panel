package ipc

// Message types sent over the Unix socket.
// Each message is a newline-delimited JSON object.

const (
	// Client → Daemon
	CmdList        = "list"
	CmdStart       = "start"
	CmdStop        = "stop"
	CmdRestart     = "restart"
	CmdSendCommand = "console"
	CmdMetrics     = "metrics"
	CmdStreamLogs  = "stream_logs"

	// Daemon → Client
	RespOK    = "ok"
	RespError = "error"
	RespData  = "data"
	RespLog   = "log"
)

type Request struct {
	Cmd      string `json:"cmd"`
	ServerID string `json:"server_id,omitempty"`
	Payload  string `json:"payload,omitempty"`
}

type Response struct {
	Type    string      `json:"type"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ServerInfo struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	PID     int     `json:"pid"`
	Port    int     `json:"port"`
	RAMUsed uint64  `json:"ram_used"`
	RAMMax  uint64  `json:"ram_max"`
	CPU     float32 `json:"cpu"`
	Uptime  uint64  `json:"uptime"`
}

type MetricsInfo struct {
	ServerID string  `json:"server_id"`
	PID      uint32  `json:"pid"`
	RAMUsed  uint64  `json:"ram_used"`
	RAMMax   uint64  `json:"ram_max"`
	CPU      float32 `json:"cpu"`
	Uptime   uint64  `json:"uptime"`
}
