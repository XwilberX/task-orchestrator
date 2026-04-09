package services

import (
	"context"
	"errors"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/google/uuid"
)

var ErrTaskNotFound = errors.New("tarea no encontrada")
var ErrCannotCancel = errors.New("solo se pueden cancelar tareas en estado PENDING o QUEUED")

// Submitter es la interfaz que implementa el WorkerPool.
// Se usa para evitar import circular entre services y scheduler.
type Submitter interface {
	Submit(task *models.Task)
}

type TaskService struct {
	repo    *repositories.TaskRepository
	defRepo *repositories.DefinitionRepository
	pool    Submitter
}

func NewTaskService(
	repo *repositories.TaskRepository,
	defRepo *repositories.DefinitionRepository,
	pool Submitter,
) *TaskService {
	return &TaskService{repo: repo, defRepo: defRepo, pool: pool}
}

// Dispatch crea la tarea y la envía al worker pool.
func (s *TaskService) Dispatch(ctx context.Context, req models.DispatchRequest) (*models.Task, error) {
	task, err := s.buildTask(ctx, req)
	if err != nil {
		return nil, err
	}

	task.ID = uuid.NewString()
	task.Status = models.StatusPending
	task.Attempt = 1
	task.CreatedAt = time.Now().UTC()

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	s.pool.Submit(task)
	return task, nil
}

// Cancel cancela una tarea en PENDING o QUEUED.
func (s *TaskService) Cancel(ctx context.Context, id string) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}
	if task.Status != models.StatusPending && task.Status != models.StatusQueued {
		return ErrCannotCancel
	}
	return s.repo.UpdateStatus(ctx, id, models.StatusCancelled)
}

// GetByID devuelve una tarea por ID.
func (s *TaskService) GetByID(ctx context.Context, id string) (*models.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// List devuelve tareas con filtros.
func (s *TaskService) List(ctx context.Context, filter models.TaskFilter) ([]models.Task, error) {
	return s.repo.List(ctx, filter, 100)
}

// buildTask construye un Task a partir del DispatchRequest.
func (s *TaskService) buildTask(ctx context.Context, req models.DispatchRequest) (*models.Task, error) {
	task := &models.Task{
		MaxRetries:        3,
		BackoffMultiplier: 5,
		TimeoutSeconds:    60,
		MemoryMB:          256,
		CPUShares:         512,
	}

	if req.Definition != "" {
		def, err := s.defRepo.GetByName(ctx, req.Definition)
		if err != nil {
			return nil, err
		}
		if def == nil {
			return nil, ErrDefinitionNotFound
		}
		task.DefinitionID = def.ID
		task.DefinitionName = def.Name
		task.Runtime = def.Runtime
		task.Code = def.Code
		task.Packages = def.Packages
		task.MaxRetries = def.MaxRetries
		task.BackoffMultiplier = def.BackoffMultiplier
		task.MemoryMB = def.MemoryMB
		task.CPUShares = def.CPUShares
		task.NetworkEnabled = def.NetworkEnabled
		task.Input = req.Input
		if req.TimeoutSeconds > 0 {
			task.TimeoutSeconds = req.TimeoutSeconds
		} else {
			task.TimeoutSeconds = def.TimeoutSeconds
		}
	} else {
		if req.Runtime == "" || req.Code == "" {
			return nil, errors.New("runtime y code son requeridos para tareas ad-hoc")
		}
		task.Runtime = req.Runtime
		task.Code = req.Code
		task.Args = req.Args
		task.Packages = req.Packages
		if req.TimeoutSeconds > 0 {
			task.TimeoutSeconds = req.TimeoutSeconds
		}
		if req.MemoryMB > 0 {
			task.MemoryMB = req.MemoryMB
		}
	}

	return task, nil
}
