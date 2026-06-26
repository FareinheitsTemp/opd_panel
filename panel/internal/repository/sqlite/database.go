package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type DatabaseRepo struct{ db *sql.DB }

func NewDatabaseRepo(db *sql.DB) *DatabaseRepo { return &DatabaseRepo{db: db} }

func (r *DatabaseRepo) Create(ctx context.Context, d *domain.ServerDatabase) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO server_databases (id, server_id, db_name, db_user, db_pass_enc, host, port, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.ServerID, d.DBName, d.DBUser, d.DBPassEnc, d.Host, d.Port, d.CreatedAt,
	)
	return err
}

func (r *DatabaseRepo) ListByServer(ctx context.Context, serverID string) ([]*domain.ServerDatabase, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, server_id, db_name, db_user, db_pass_enc, host, port, created_at FROM server_databases WHERE server_id=? ORDER BY created_at`,
		serverID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []*domain.ServerDatabase
	for rows.Next() {
		var d domain.ServerDatabase
		if err := rows.Scan(&d.ID, &d.ServerID, &d.DBName, &d.DBUser, &d.DBPassEnc, &d.Host, &d.Port, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *DatabaseRepo) GetByID(ctx context.Context, id string) (*domain.ServerDatabase, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, server_id, db_name, db_user, db_pass_enc, host, port, created_at FROM server_databases WHERE id=?`, id)
	var d domain.ServerDatabase
	err := row.Scan(&d.ID, &d.ServerID, &d.DBName, &d.DBUser, &d.DBPassEnc, &d.Host, &d.Port, &d.CreatedAt)
	if err != nil { return nil, err }
	return &d, nil
}

func (r *DatabaseRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM server_databases WHERE id=?`, id)
	return err
}

var _ = time.Now
