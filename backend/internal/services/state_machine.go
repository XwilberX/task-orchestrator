package services

import (
	"errors"

	"github.com/XwilberX/task-orchestrator/internal/models"
)

var ErrInvalidTransition = errors.New("transición de estado inválida")

// validTransitions define las transiciones permitidas por estado.
var validTransitions = map[models.TaskStatus][]models.TaskStatus{
	models.StatusPending: {models.StatusQueued, models.StatusRunning, models.StatusCancelled},
	models.StatusQueued:  {models.StatusRunning, models.StatusCancelled},
	models.StatusRunning: {models.StatusSuccess, models.StatusFailed, models.StatusTimeout},
}

// CanTransition devuelve true si la transición from→to es válida.
func CanTransition(from, to models.TaskStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false // estado terminal
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// MustTransition valida y devuelve error si la transición no es válida.
func MustTransition(from, to models.TaskStatus) error {
	if !CanTransition(from, to) {
		return ErrInvalidTransition
	}
	return nil
}
