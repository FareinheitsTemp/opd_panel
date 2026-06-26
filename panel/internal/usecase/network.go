package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository"
)

const (
	PortRangeStart = 25565
	PortRangeEnd   = 30000
)

type NetworkUseCase struct {
	repo repository.AllocationRepo
}

func NewNetworkUseCase(repo repository.AllocationRepo) *NetworkUseCase {
	return &NetworkUseCase{repo: repo}
}

func (uc *NetworkUseCase) List(ctx context.Context, serverID string) ([]*domain.Allocation, error) {
	return uc.repo.ListByServer(ctx, serverID)
}

func (uc *NetworkUseCase) Assign(ctx context.Context, serverID, ip, alias string) (*domain.Allocation, error) {
	port, err := uc.repo.GetFreePort(ctx, PortRangeStart, PortRangeEnd)
	if err != nil { return nil, fmt.Errorf("no free ports: %w", err) }
	a := &domain.Allocation{
		ID:        uuid.NewString(),
		ServerID:  serverID,
		IP:        ip,
		Port:      port,
		Alias:     alias,
		IsPrimary: false,
		CreatedAt: time.Now(),
	}
	if err := uc.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create allocation: %w", err)
	}
	return a, nil
}

func (uc *NetworkUseCase) Free(ctx context.Context, id string) error {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil { return fmt.Errorf("allocation not found: %w", err) }
	if a.IsPrimary { return fmt.Errorf("cannot remove primary port") }
	return uc.repo.Delete(ctx, id)
}
