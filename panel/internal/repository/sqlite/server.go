package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type ServerRepo struct{ db *sql.DB }

func NewServerRepo(db *sql.DB) *ServerRepo { return &ServerRepo{db: db} }

func (r *ServerRepo) Create(ctx context.Context, s *domain.Server) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO servers (id, name, type, version, port, ram_min, ram_max, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Type, s.Version, s.Port, s.RAMMin, s.RAMMax, s.Status,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *ServerRepo) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, type, version, port, ram_min, ram_max, status, created_at, updated_at
		 FROM servers WHERE id = ?`, id)
	return scanServer(row)
}

func (r *ServerRepo) GetByName(ctx context.Context, name string) (*domain.Server, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, type, version, port, ram_min, ram_max, status, created_at, updated_at
		 FROM servers WHERE name = ?`, name)
	return scanServer(row)
}

func (r *ServerRepo) List(ctx context.Context) ([]*domain.Server, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, type, version, port, ram_min, ram_max, status, created_at, updated_at
		 FROM servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Server
	for rows.Next() {
		s := &domain.Server{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Version, &s.Port,
			&s.RAMMin, &s.RAMMax, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ServerRepo) Update(ctx context.Context, s *domain.Server) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE servers SET status=?, updated_at=? WHERE id=?`,
		s.Status, time.Now(), s.ID)
	return err
}

func (r *ServerRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM servers WHERE id=?`, id)
	return err
}

func scanServer(row *sql.Row) (*domain.Server, error) {
	s := &domain.Server{}
	err := row.Scan(&s.ID, &s.Name, &s.Type, &s.Version, &s.Port,
		&s.RAMMin, &s.RAMMax, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
