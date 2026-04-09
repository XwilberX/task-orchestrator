package services

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/executor"
	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var ErrTaskNotFound = errors.New("tarea no encontrada")
var ErrCannotCancel = errors.New("solo se pueden cancelar tareas en estado PENDING o QUEUED")

type TaskService struct {
	repo    *repositories.TaskRepository
	defRepo *repositories.DefinitionRepository
	exec    *executor.Executor
}

func NewTaskService(
	repo *repositories.TaskRepository,
	defRepo *repositories.DefinitionRepository,
	exec *executor.Executor,
) *TaskService {
	return &TaskService{repo: repo, defRepo: defRepo, exec: exec}
}

// Dispatch crea la tarea y la ejecuta en background.
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

	// Ejecutar en background — el worker pool (paso 6) reemplazará esto
	go s.runWithRetries(task)

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

// GetByID devuelve una tarea por su ID.
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

// List devuelve tareas aplicando filtros.
func (s *TaskService) List(ctx context.Context, filter models.TaskFilter) ([]models.Task, error) {
	return s.repo.List(ctx, filter, 100)
}

// runWithRetries ejecuta la tarea y aplica la política de reintentos.
func (s *TaskService) runWithRetries(task *models.Task) {
	for {
		result := s.runOnce(task)

		finalStatus := models.StatusSuccess
		if result.TimedOut {
			finalStatus = models.StatusTimeout
		} else if result.ExitCode != 0 {
			finalStatus = models.StatusFailed
		}

		now := time.Now().UTC()
		fields := bson.M{
			"status":      finalStatus,
			"finished_at": now,
			"exit_code":   result.ExitCode,
		}
		s.repo.UpdateFields(context.Background(), task.ID, fields)

		// ¿Reintentamos?
		if (finalStatus == models.StatusFailed || finalStatus == models.StatusTimeout) &&
			task.Attempt < task.MaxRetries {

			backoff := time.Duration(math.Pow(float64(task.BackoffMultiplier), float64(task.Attempt))) * time.Second
			time.Sleep(backoff)

			task.Attempt++
			s.repo.UpdateFields(context.Background(), task.ID, bson.M{
				"status":  models.StatusPending,
				"attempt": task.Attempt,
			})
			continue
		}

		break
	}
}

// runOnce ejecuta el contenedor una vez y devuelve el resultado.
func (s *TaskService) runOnce(task *models.Task) *executor.ExecResult {
	now := time.Now().UTC()
	s.repo.UpdateFields(context.Background(), task.ID, bson.M{
		"status":     models.StatusRunning,
		"started_at": now,
	})

	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := s.exec.Run(ctx, executor.RunConfig{
		TaskID:         task.ID,
		DefinitionName: task.DefinitionName,
		Attempt:        task.Attempt,
		Runtime:        task.Runtime,
		Code:           task.Code,
		Args:           task.Args,
		Packages:       task.Packages,
		TimeoutSeconds: task.TimeoutSeconds,
		MemoryMB:       task.MemoryMB,
		CPUShares:      task.CPUShares,
		NetworkEnabled: task.NetworkEnabled,
	})
	if err != nil {
		// Error interno del executor → FAILED
		return &executor.ExecResult{ExitCode: -1}
	}
	return result
}

// buildTask construye un Task a partir del DispatchRequest.
// Si hay definition_name, carga la definición y copia su config.
// Si es ad-hoc, usa los campos del request directamente.
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
		// Ad-hoc
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
