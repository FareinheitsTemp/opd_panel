package domain

import "time"

type BackupStatus string

const (
	BackupStatusDone   BackupStatus = "done"
	BackupStatusFailed BackupStatus = "failed"
)

type Backup struct {
	ID        string
	ServerID  string
	Path      string
	SizeBytes int64
	Status    BackupStatus
	CreatedAt time.Time
}
