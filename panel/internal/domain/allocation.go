package domain

import "time"

type Allocation struct {
	ID        string
	ServerID  string
	IP        string
	Port      int
	Alias     string // optional: "dynmap", "geyser"
	IsPrimary bool
	CreatedAt time.Time
}
