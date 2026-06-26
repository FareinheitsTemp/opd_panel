package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type SubuserRepo struct{ db *sql.DB }

func NewSubuserRepo(db *sql.DB) *SubuserRepo { return &SubuserRepo{db: db} }

func (r *SubuserRepo) Create(ctx context.Context, s *domain.Subuser) error {
	perms := strings.Join(s.Permissions, ",")
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO subusers (id, server_id, email, user_id, permissions, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.ServerID, s.Email, s.UserID, perms, s.CreatedAt,
	)
	return err
}

func (r *SubuserRepo) GetByID(ctx context.Context, id string) (*domain.Subuser, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, server_id, email, user_id, permissions, created_at FROM subusers WHERE id=?`, id)
	return scanSubuser(row)
}

func (r *SubuserRepo) GetByEmailAndServer(ctx context.Context, email, serverID string) (*domain.Subuser, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, server_id, email, user_id, permissions, created_at FROM subusers WHERE email=? AND server_id=?`, email, serverID)
	return scanSubuser(row)
}

func (r *SubuserRepo) ListByServer(ctx context.Context, serverID string) ([]*domain.Subuser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, server_id, email, user_id, permissions, created_at FROM subusers WHERE server_id=? ORDER BY created_at`, serverID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []*domain.Subuser
	for rows.Next() {
		var s domain.Subuser
		var perms string
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Email, &s.UserID, &perms, &s.CreatedAt); err != nil {
			return nil, err
		}
		if perms != "" { s.Permissions = strings.Split(perms, ",") }
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *SubuserRepo) Update(ctx context.Context, s *domain.Subuser) error {
	perms := strings.Join(s.Permissions, ",")
	_, err := r.db.ExecContext(ctx,
		`UPDATE subusers SET permissions=? WHERE id=?`, perms, s.ID)
	return err
}

func (r *SubuserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM subusers WHERE id=?`, id)
	return err
}

func scanSubuser(row *sql.Row) (*domain.Subuser, error) {
	var s domain.Subuser
	var perms string
	err := row.Scan(&s.ID, &s.ServerID, &s.Email, &s.UserID, &perms, &s.CreatedAt)
	if err != nil { return nil, err }
	if perms != "" { s.Permissions = strings.Split(perms, ",") }
	return &s, nil
}

var _ = time.Now
