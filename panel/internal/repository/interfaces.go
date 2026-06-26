package repository

import (
	"context"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type ServerRepo interface {
	Create(ctx context.Context, s *domain.Server) error
	GetByID(ctx context.Context, id string) (*domain.Server, error)
	GetByName(ctx context.Context, name string) (*domain.Server, error)
	List(ctx context.Context) ([]*domain.Server, error)
	Update(ctx context.Context, s *domain.Server) error
	Delete(ctx context.Context, id string) error
}

type BackupRepo interface {
	Create(ctx context.Context, b *domain.Backup) error
	ListByServer(ctx context.Context, serverID string) ([]*domain.Backup, error)
	GetByID(ctx context.Context, id string) (*domain.Backup, error)
	Delete(ctx context.Context, id string) error
}

type EventRepo interface {
	Append(ctx context.Context, e *domain.Event) error
	ListByServer(ctx context.Context, serverID string, limit int) ([]*domain.Event, error)
}

type ScheduleRepo interface {
	Create(ctx context.Context, s *domain.Schedule) error
	GetByID(ctx context.Context, id string) (*domain.Schedule, error)
	ListByServer(ctx context.Context, serverID string) ([]*domain.Schedule, error)
	Update(ctx context.Context, s *domain.Schedule) error
	Delete(ctx context.Context, id string) error
}

type SubuserRepo interface {
	Create(ctx context.Context, s *domain.Subuser) error
	GetByID(ctx context.Context, id string) (*domain.Subuser, error)
	ListByServer(ctx context.Context, serverID string) ([]*domain.Subuser, error)
	Update(ctx context.Context, s *domain.Subuser) error
	Delete(ctx context.Context, id string) error
	GetByEmailAndServer(ctx context.Context, email, serverID string) (*domain.Subuser, error)
}

type AllocationRepo interface {
	Create(ctx context.Context, a *domain.Allocation) error
	ListByServer(ctx context.Context, serverID string) ([]*domain.Allocation, error)
	GetFreePort(ctx context.Context, startPort, endPort int) (int, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Allocation, error)
}

type DatabaseRepo interface {
	Create(ctx context.Context, d *domain.ServerDatabase) error
	ListByServer(ctx context.Context, serverID string) ([]*domain.ServerDatabase, error)
	GetByID(ctx context.Context, id string) (*domain.ServerDatabase, error)
	Delete(ctx context.Context, id string) error
}
