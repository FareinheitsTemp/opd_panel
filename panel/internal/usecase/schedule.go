package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/agent"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository"
)

type ScheduleUseCase struct {
	repo   repository.ScheduleRepo
	agent  *agent.Client
	cron   *cron.Cron
	entries map[string]cron.EntryID
	mu      sync.Mutex
}

func NewScheduleUseCase(repo repository.ScheduleRepo, ag *agent.Client) *ScheduleUseCase {
	uc := &ScheduleUseCase{
		repo:    repo,
		agent:   ag,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
	}
	uc.cron.Start()
	return uc
}

type CreateScheduleInput struct {
	ServerID string
	Name     string
	CronExpr string
	Enabled  bool
	Tasks    []domain.ScheduleTask
}

func (uc *ScheduleUseCase) Create(ctx context.Context, in CreateScheduleInput) (*domain.Schedule, error) {
	s := &domain.Schedule{
		ID:       uuid.NewString(),
		ServerID: in.ServerID,
		Name:     in.Name,
		CronExpr: in.CronExpr,
		Enabled:  in.Enabled,
		Tasks:    in.Tasks,
		CreatedAt: time.Now(),
	}
	if err := uc.repo.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	if s.Enabled {
		uc.register(s)
	}
	return s, nil
}

func (uc *ScheduleUseCase) List(ctx context.Context, serverID string) ([]*domain.Schedule, error) {
	return uc.repo.ListByServer(ctx, serverID)
}

func (uc *ScheduleUseCase) Update(ctx context.Context, id string, in CreateScheduleInput) (*domain.Schedule, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil { return nil, fmt.Errorf("not found: %w", err) }
	s.Name = in.Name
	s.CronExpr = in.CronExpr
	s.Enabled = in.Enabled
	s.Tasks = in.Tasks
	if err := uc.repo.Update(ctx, s); err != nil {
		return nil, fmt.Errorf("update schedule: %w", err)
	}
	uc.unregister(s.ID)
	if s.Enabled {
		uc.register(s)
	}
	return s, nil
}

func (uc *ScheduleUseCase) Delete(ctx context.Context, id string) error {
	uc.unregister(id)
	return uc.repo.Delete(ctx, id)
}

func (uc *ScheduleUseCase) RunNow(ctx context.Context, id string) error {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil { return fmt.Errorf("not found: %w", err) }
	go uc.execute(s)
	return nil
}

func (uc *ScheduleUseCase) register(s *domain.Schedule) {
	sc := *s
	entryID, err := uc.cron.AddFunc(sc.CronExpr, func() {
		uc.execute(&sc)
	})
	if err == nil {
		uc.mu.Lock()
		uc.entries[sc.ID] = entryID
		uc.mu.Unlock()
	}
}

func (uc *ScheduleUseCase) unregister(id string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if eid, ok := uc.entries[id]; ok {
		uc.cron.Remove(eid)
		delete(uc.entries, id)
	}
}

func (uc *ScheduleUseCase) execute(s *domain.Schedule) {
	ctx := context.Background()
	for _, task := range s.Tasks {
		if task.DelayMs > 0 {
			time.Sleep(time.Duration(task.DelayMs) * time.Millisecond)
		}
		switch task.Action {
		case domain.TaskCommand:
			_ = uc.agent.SendConsoleCommand(ctx, s.ServerID, task.Payload)
		case domain.TaskPower:
			switch task.Payload {
			case "start":
				_ = uc.agent.StartServer(ctx, s.ServerID)
			case "stop":
				_ = uc.agent.StopServer(ctx, s.ServerID)
			case "restart":
				_ = uc.agent.StopServer(ctx, s.ServerID)
				time.Sleep(5 * time.Second)
				_ = uc.agent.StartServer(ctx, s.ServerID)
			}
		case domain.TaskBackup:
			_ = uc.agent.CreateBackup(ctx, s.ServerID)
		}
	}
	now := time.Now()
	s.LastRunAt = &now
	_ = uc.repo.Update(ctx, s)
}
