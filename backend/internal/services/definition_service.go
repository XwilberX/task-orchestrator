package services

import (
	"context"
	"errors"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/google/uuid"
)

var ErrDefinitionNotFound = errors.New("definición no encontrada")
var ErrDefinitionNameTaken = errors.New("ya existe una definición con ese nombre")

type DefinitionService struct {
	repo *repositories.DefinitionRepository
}

func NewDefinitionService(repo *repositories.DefinitionRepository) *DefinitionService {
	return &DefinitionService{repo: repo}
}

func (s *DefinitionService) Create(ctx context.Context, d models.Definition) (*models.Definition, error) {
	existing, err := s.repo.GetByName(ctx, d.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDefinitionNameTaken
	}

	applyDefaults(&d)
	d.ID = uuid.NewString()
	d.CreatedAt = time.Now().UTC()
	d.UpdatedAt = d.CreatedAt

	if err := s.repo.Create(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DefinitionService) GetByID(ctx context.Context, id string) (*models.Definition, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrDefinitionNotFound
	}
	return d, nil
}

func (s *DefinitionService) List(ctx context.Context) ([]models.Definition, error) {
	return s.repo.List(ctx)
}

func (s *DefinitionService) Update(ctx context.Context, id string, updated models.Definition) (*models.Definition, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrDefinitionNotFound
	}

	// Si cambia el nombre, verificar que no esté tomado
	if updated.Name != existing.Name {
		taken, err := s.repo.GetByName(ctx, updated.Name)
		if err != nil {
			return nil, err
		}
		if taken != nil {
			return nil, ErrDefinitionNameTaken
		}
	}

	applyDefaults(&updated)
	updated.ID = id
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, id, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// UpdateCode reemplaza solo el campo code de una definición existente.
func (s *DefinitionService) UpdateCode(ctx context.Context, id, code string) (*models.Definition, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrDefinitionNotFound
	}
	existing.Code = code
	existing.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *DefinitionService) Delete(ctx context.Context, id string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrDefinitionNotFound
	}
	return s.repo.Delete(ctx, id)
}

func applyDefaults(d *models.Definition) {
	defaults := models.DefaultDefinition()
	if d.TimeoutSeconds == 0 {
		d.TimeoutSeconds = defaults.TimeoutSeconds
	}
	if d.MaxRetries == 0 {
		d.MaxRetries = defaults.MaxRetries
	}
	if d.BackoffMultiplier == 0 {
		d.BackoffMultiplier = defaults.BackoffMultiplier
	}
	if d.MaxConcurrent == 0 {
		d.MaxConcurrent = defaults.MaxConcurrent
	}
	if d.MemoryMB == 0 {
		d.MemoryMB = defaults.MemoryMB
	}
	if d.CPUShares == 0 {
		d.CPUShares = defaults.CPUShares
	}
}
