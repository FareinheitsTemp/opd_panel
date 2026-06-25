package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/agent"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/versions"
)

type ServerUseCase struct {
	repo     repository.ServerRepo
	agent    *agent.Client
	versions *versions.Manager
}

func NewServerUseCase(repo repository.ServerRepo, agent *agent.Client, vm *versions.Manager) *ServerUseCase {
	return &ServerUseCase{repo: repo, agent: agent, versions: vm}
}

type CreateServerInput struct {
	Name    string
	Type    domain.ServerType
	Version string
	Port    int
	RAMMin  int
	RAMMax  int
}

func (uc *ServerUseCase) Create(ctx context.Context, in CreateServerInput) (*domain.Server, error) {
	// Check name uniqueness
	if existing, _ := uc.repo.GetByName(ctx, in.Name); existing != nil {
		return nil, fmt.Errorf("server with name %q already exists", in.Name)
	}

	s := &domain.Server{
		ID:        uuid.NewString(),
		Name:      in.Name,
		Type:      in.Type,
		Version:   in.Version,
		Port:      in.Port,
		RAMMin:    in.RAMMin,
		RAMMax:    in.RAMMax,
		Status:    domain.StatusStopped,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// TODO: resolve & download jar via uc.versions.Resolve(ctx, s.Type, s.Version, serverDir)

	if err := uc.repo.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}
	return s, nil
}

func (uc *ServerUseCase) Start(ctx context.Context, id string) error {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if s.Status == domain.StatusRunning {
		return fmt.Errorf("server %s is already running", id)
	}
	if err := uc.agent.StartServer(ctx, id); err != nil {
		return fmt.Errorf("agent start: %w", err)
	}
	s.Status = domain.StatusStarting
	return uc.repo.Update(ctx, s)
}

func (uc *ServerUseCase) Stop(ctx context.Context, id string) error {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if s.Status == domain.StatusStopped {
		return fmt.Errorf("server %s is already stopped", id)
	}
	if err := uc.agent.StopServer(ctx, id); err != nil {
		return fmt.Errorf("agent stop: %w", err)
	}
	s.Status = domain.StatusStopping
	return uc.repo.Update(ctx, s)
}

func (uc *ServerUseCase) Restart(ctx context.Context, id string) error {
	if err := uc.Stop(ctx, id); err != nil {
		return err
	}
	return uc.Start(ctx, id)
}

func (uc *ServerUseCase) List(ctx context.Context) ([]*domain.Server, error) {
	return uc.repo.List(ctx)
}

func (uc *ServerUseCase) Info(ctx context.Context, id string) (*domain.Server, *domain.ServerMetrics, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	metrics, err := uc.agent.GetStatus(ctx, id)
	if err != nil {
		// metrics are optional — return server without metrics
		return s, nil, nil
	}
	return s, metrics, nil
}

func (uc *ServerUseCase) Delete(ctx context.Context, id string) error {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if s.Status == domain.StatusRunning {
		return fmt.Errorf("stop server before deleting")
	}
	return uc.repo.Delete(ctx, id)
}
