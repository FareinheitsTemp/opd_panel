package domain

import "time"

type Plugin struct {
	ID          string
	ServerID    string
	Name        string
	FileName    string
	Version     string
	Enabled     bool
	InstalledAt time.Time
}
