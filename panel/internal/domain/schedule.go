package domain

import "time"

type TaskAction string

const (
	TaskCommand TaskAction = "command"
	TaskPower   TaskAction = "power"
	TaskBackup  TaskAction = "backup"
)

type ScheduleTask struct {
	Order   int
	Action  TaskAction
	Payload string
	DelayMs int
}

type Schedule struct {
	ID         string
	ServerID   string
	Name       string
	CronExpr   string
	Enabled    bool
	Tasks      []ScheduleTask
	LastRunAt  *time.Time
	CreatedAt  time.Time
}
