package domain

import "time"

// All available granular permissions.
const (
	PermConsoleRead    = "console.read"
	PermConsoleWrite   = "console.write"
	PermFilesRead      = "files.read"
	PermFilesWrite     = "files.write"
	PermFilesDelete    = "files.delete"
	PermBackupsRead    = "backups.read"
	PermBackupsCreate  = "backups.create"
	PermBackupsDelete  = "backups.delete"
	PermPowerStart     = "power.start"
	PermPowerStop      = "power.stop"
	PermPowerRestart   = "power.restart"
	PermDatabasesRead  = "databases.read"
	PermDatabasesCreate = "databases.create"
	PermDatabasesDelete = "databases.delete"
	PermNetworkRead    = "network.read"
	PermNetworkManage  = "network.manage"
	PermSchedulesRead  = "schedules.read"
	PermSchedulesManage = "schedules.manage"
)

type Subuser struct {
	ID          string
	ServerID    string
	Email       string       // invited by email
	UserID      string       // resolved after acceptance
	Permissions []string
	CreatedAt   time.Time
}

func (s *Subuser) HasPermission(perm string) bool {
	for _, p := range s.Permissions {
		if p == perm { return true }
	}
	return false
}
