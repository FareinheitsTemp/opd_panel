package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository"
)

type SubuserUseCase struct {
	repo repository.SubuserRepo
}

func NewSubuserUseCase(repo repository.SubuserRepo) *SubuserUseCase {
	return &SubuserUseCase{repo: repo}
}

func (uc *SubuserUseCase) Add(ctx context.Context, serverID, email string, perms []string) (*domain.Subuser, error) {
	if existing, _ := uc.repo.GetByEmailAndServer(ctx, email, serverID); existing != nil {
		return nil, fmt.Errorf("user %q already has access to this server", email)
	}
	s := &domain.Subuser{
		ID:          uuid.NewString(),
		ServerID:    serverID,
		Email:       email,
		Permissions: perms,
		CreatedAt:   time.Now(),
	}
	if err := uc.repo.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("create subuser: %w", err)
	}
	return s, nil
}

func (uc *SubuserUseCase) List(ctx context.Context, serverID string) ([]*domain.Subuser, error) {
	return uc.repo.ListByServer(ctx, serverID)
}

func (uc *SubuserUseCase) UpdatePermissions(ctx context.Context, id string, perms []string) (*domain.Subuser, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil { return nil, fmt.Errorf("subuser not found: %w", err) }
	s.Permissions = perms
	if err := uc.repo.Update(ctx, s); err != nil {
		return nil, fmt.Errorf("update subuser: %w", err)
	}
	return s, nil
}

func (uc *SubuserUseCase) Remove(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *SubuserUseCase) CheckPermission(ctx context.Context, serverID, email, perm string) bool {
	sub, err := uc.repo.GetByEmailAndServer(ctx, email, serverID)
	if err != nil { return false }
	return sub.HasPermission(perm)
}
