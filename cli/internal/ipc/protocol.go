package ipc

const (
	// Client → Daemon
	CmdList           = "list"
	CmdListDisk       = "list_disk"
	CmdStart          = "start"
	CmdStop           = "stop"
	CmdRestart        = "restart"
	CmdSendCommand    = "console"
	CmdMetrics        = "metrics"
	CmdStreamLogs     = "stream_logs"
	CmdRemove         = "remove"
	CmdCreate         = "create"
	CmdUpdateSettings = "update_settings"
	CmdSysStats       = "sys_stats"
	CmdPing           = "ping"

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

type DiskServerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Port   int    `json:"port"`
	RAMMax int    `json:"ram_max_mb"`
	Jar    string `json:"jar"`
}

type CreateRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Port     int    `json:"port"`
	RAMMinMB int    `json:"ram_min_mb"`
	RAMMaxMB int    `json:"ram_max_mb"`
	Jar      string `json:"jar"`
}

type UpdateSettingsRequest struct {
	ServerID    string   `json:"server_id"`
	Name        string   `json:"name"`
	Port        int      `json:"port"`
	RAMMaxMB    int      `json:"ram_max_mb"`
	Jar         string   `json:"jar"`
	JavaFlags   []string `json:"java_flags"`
	AutoRestart bool     `json:"auto_restart"`
}

type SysStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	RAMUsedMB   uint64  `json:"ram_used_mb"`
	RAMTotalMB  uint64  `json:"ram_total_mb"`
	DiskUsedGB  float64 `json:"disk_used_gb"`
	DiskTotalGB float64 `json:"disk_total_gb"`
}
