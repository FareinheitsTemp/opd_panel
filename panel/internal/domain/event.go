package domain

import "time"

type EventType string

const (
	EventServerStart   EventType = "server.start"
	EventServerStop    EventType = "server.stop"
	EventServerCrash   EventType = "server.crash"
	EventBackupCreate  EventType = "backup.create"
	EventConsoleCmd    EventType = "console.command"
)

type Event struct {
	ID        int64
	ServerID  string
	Type      EventType
	Payload   string // JSON
	CreatedAt time.Time
}
