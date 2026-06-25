package domain

import "time"

type ServerType string

const (
	ServerTypePaper   ServerType = "paper"
	ServerTypeVanilla ServerType = "vanilla"
	ServerTypePurpur  ServerType = "purpur"
	ServerTypeFabric  ServerType = "fabric"
)

type ServerStatus string

const (
	StatusRunning  ServerStatus = "running"
	StatusStopped  ServerStatus = "stopped"
	StatusStarting ServerStatus = "starting"
	StatusStopping ServerStatus = "stopping"
	StatusCrashed  ServerStatus = "crashed"
)

type Server struct {
	ID         string
	Name       string
	Type       ServerType
	Version    string
	Port       int
	RAMMin     int // MB
	RAMMax     int // MB
	JavaFlags  []string
	Status     ServerStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ServerMetrics struct {
	ServerID string
	PID      int
	RAMUsed  int64 // bytes
	CPU      float64
	Uptime   int64 // seconds
	Players  int
}
