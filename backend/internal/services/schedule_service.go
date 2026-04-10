package services

import (
	"context"
	"errors"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/XwilberX/task-orchestrator/internal/scheduler"
	"github.com/google/uuid"
)

var ErrScheduleNotFound = errors.New("schedule no encontrado")
var ErrInvalidCron = errors.New("expresión cron inválida")

type ScheduleService struct {
	repo      *repositories.ScheduleRepository
	defRepo   *repositories.DefinitionRepository
	cronSched *scheduler.CronScheduler
}

func NewScheduleService(
	repo *repositories.ScheduleRepository,
	defRepo *repositories.DefinitionRepository,
	cronSched *scheduler.CronScheduler,
) *ScheduleService {
	return &ScheduleService{repo: repo, defRepo: defRepo, cronSched: cronSched}
}

func (s *ScheduleService) Create(ctx context.Context, req models.Schedule) (*models.Schedule, error) {
	// Validar que la definición existe
	def, err := s.defRepo.GetByID(ctx, req.DefinitionID)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, ErrDefinitionNotFound
	}

	// Validar expresión cron y calcular próxima ejecución
	nextRun, err := scheduler.NextRun(req.Cron)
	if err != nil {
		return nil, ErrInvalidCron
	}

	req.ID = uuid.NewString()
	req.Status = models.ScheduleActive
	req.CreatedAt = time.Now().UTC()
	req.NextRunAt = &nextRun

	if err := s.repo.Create(ctx, &req); err != nil {
		return nil, err
	}

	if err := s.cronSched.Add(&req); err != nil {
		return nil, err
	}

	return &req, nil
}

func (s *ScheduleService) List(ctx context.Context) ([]models.Schedule, error) {
	return s.repo.List(ctx)
}

func (s *ScheduleService) GetByID(ctx context.Context, id string) (*models.Schedule, error) {
	sched, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sched == nil {
		return nil, ErrScheduleNotFound
	}
	return sched, nil
}

func (s *ScheduleService) Update(ctx context.Context, id string, req models.Schedule) (*models.Schedule, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrScheduleNotFound
	}

	nextRun, err := scheduler.NextRun(req.Cron)
	if err != nil {
		return nil, ErrInvalidCron
	}

	req.ID = id
	req.Status = existing.Status
	req.CreatedAt = existing.CreatedAt
	req.NextRunAt = &nextRun

	if err := s.repo.Update(ctx, id, &req); err != nil {
		return nil, err
	}

	// Re-registrar en el cron si está activo
	if req.Status == models.ScheduleActive {
		s.cronSched.Remove(id)
		s.cronSched.Add(&req)
	}

	return &req, nil
}

func (s *ScheduleService) Delete(ctx context.Context, id string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrScheduleNotFound
	}
	s.cronSched.Remove(id)
	return s.repo.Delete(ctx, id)
}

func (s *ScheduleService) Toggle(ctx context.Context, id string) (*models.Schedule, error) {
	sched, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sched == nil {
		return nil, ErrScheduleNotFound
	}

	if sched.Status == models.ScheduleActive {
		sched.Status = models.SchedulePaused
		s.cronSched.Remove(id)
	} else {
		sched.Status = models.ScheduleActive
		nextRun, _ := scheduler.NextRun(sched.Cron)
		sched.NextRunAt = &nextRun
		s.cronSched.Add(sched)
	}

	if err := s.repo.Update(ctx, id, sched); err != nil {
		return nil, err
	}
	return sched, nil
}
