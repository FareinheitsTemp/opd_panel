package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type AllocationRepo struct{ db *sql.DB }

func NewAllocationRepo(db *sql.DB) *AllocationRepo { return &AllocationRepo{db: db} }

func (r *AllocationRepo) Create(ctx context.Context, a *domain.Allocation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO allocations (id, server_id, ip, port, alias, is_primary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ServerID, a.IP, a.Port, a.Alias, a.IsPrimary, a.CreatedAt,
	)
	return err
}

func (r *AllocationRepo) ListByServer(ctx context.Context, serverID string) ([]*domain.Allocation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, server_id, ip, port, alias, is_primary, created_at FROM allocations WHERE server_id=? ORDER BY is_primary DESC, created_at`,
		serverID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []*domain.Allocation
	for rows.Next() {
		var a domain.Allocation
		if err := rows.Scan(&a.ID, &a.ServerID, &a.IP, &a.Port, &a.Alias, &a.IsPrimary, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *AllocationRepo) GetFreePort(ctx context.Context, startPort, endPort int) (int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT port FROM allocations WHERE port BETWEEN ? AND ?`, startPort, endPort)
	if err != nil { return 0, err }
	defer rows.Close()
	used := make(map[int]bool)
	for rows.Next() {
		var p int
		_ = rows.Scan(&p)
		used[p] = true
	}
	for p := startPort; p <= endPort; p++ {
		if !used[p] { return p, nil }
	}
	return 0, fmt.Errorf("no free ports in range %d-%d", startPort, endPort)
}

func (r *AllocationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM allocations WHERE id=?`, id)
	return err
}

func (r *AllocationRepo) GetByID(ctx context.Context, id string) (*domain.Allocation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, server_id, ip, port, alias, is_primary, created_at FROM allocations WHERE id=?`, id)
	var a domain.Allocation
	err := row.Scan(&a.ID, &a.ServerID, &a.IP, &a.Port, &a.Alias, &a.IsPrimary, &a.CreatedAt)
	if err != nil { return nil, err }
	return &a, nil
}

var _ = time.Now
