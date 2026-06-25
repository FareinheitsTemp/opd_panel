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
}

type EventRepo interface {
	Append(ctx context.Context, e *domain.Event) error
	ListByServer(ctx context.Context, serverID string, limit int) ([]*domain.Event, error)
}
