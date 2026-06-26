package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type ScheduleRepo struct{ db *sql.DB }

func NewScheduleRepo(db *sql.DB) *ScheduleRepo { return &ScheduleRepo{db: db} }

func (r *ScheduleRepo) Create(ctx context.Context, s *domain.Schedule) error {
	tasksJSON, _ := json.Marshal(s.Tasks)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO schedules (id, server_id, name, cron_expr, enabled, tasks, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ServerID, s.Name, s.CronExpr, s.Enabled, string(tasksJSON), s.CreatedAt,
	)
	return err
}

func (r *ScheduleRepo) GetByID(ctx context.Context, id string) (*domain.Schedule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, server_id, name, cron_expr, enabled, tasks, last_run_at, created_at FROM schedules WHERE id = ?`, id)
	return scanSchedule(row)
}

func (r *ScheduleRepo) ListByServer(ctx context.Context, serverID string) ([]*domain.Schedule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, server_id, name, cron_expr, enabled, tasks, last_run_at, created_at FROM schedules WHERE server_id = ? ORDER BY created_at`, serverID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []*domain.Schedule
	for rows.Next() {
		s, err := scanScheduleRows(rows)
		if err != nil { return nil, err }
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ScheduleRepo) Update(ctx context.Context, s *domain.Schedule) error {
	tasksJSON, _ := json.Marshal(s.Tasks)
	_, err := r.db.ExecContext(ctx,
		`UPDATE schedules SET name=?, cron_expr=?, enabled=?, tasks=?, last_run_at=? WHERE id=?`,
		s.Name, s.CronExpr, s.Enabled, string(tasksJSON), s.LastRunAt, s.ID,
	)
	return err
}

func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	return err
}

func scanSchedule(row *sql.Row) (*domain.Schedule, error) {
	var s domain.Schedule
	var tasksJSON string
	var lastRun sql.NullTime
	err := row.Scan(&s.ID, &s.ServerID, &s.Name, &s.CronExpr, &s.Enabled, &tasksJSON, &lastRun, &s.CreatedAt)
	if err != nil { return nil, err }
	_ = json.Unmarshal([]byte(tasksJSON), &s.Tasks)
	if lastRun.Valid { t := lastRun.Time; s.LastRunAt = &t }
	return &s, nil
}

func scanScheduleRows(rows *sql.Rows) (*domain.Schedule, error) {
	var s domain.Schedule
	var tasksJSON string
	var lastRun sql.NullTime
	err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.CronExpr, &s.Enabled, &tasksJSON, &lastRun, &s.CreatedAt)
	if err != nil { return nil, err }
	_ = json.Unmarshal([]byte(tasksJSON), &s.Tasks)
	if lastRun.Valid { t := lastRun.Time; s.LastRunAt = &t }
	return &s, nil
}

// ensure time import used
var _ = time.Now
