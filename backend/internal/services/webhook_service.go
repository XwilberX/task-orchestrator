package services

import (
	"context"
	"errors"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/google/uuid"
)

var ErrWebhookNotFound = errors.New("webhook no encontrado")

type WebhookService struct {
	repo *repositories.WebhookRepository
}

func NewWebhookService(repo *repositories.WebhookRepository) *WebhookService {
	return &WebhookService{repo: repo}
}

func (s *WebhookService) Create(ctx context.Context, url string) (*models.Webhook, error) {
	wh := &models.Webhook{
		ID:        uuid.NewString(),
		URL:       url,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, wh); err != nil {
		return nil, err
	}
	return wh, nil
}

func (s *WebhookService) List(ctx context.Context) ([]models.Webhook, error) {
	return s.repo.List(ctx)
}

func (s *WebhookService) Delete(ctx context.Context, id string) error {
	wh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if wh == nil {
		return ErrWebhookNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *WebhookService) ListDeliveries(ctx context.Context, id string) ([]models.WebhookDelivery, error) {
	wh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wh == nil {
		return nil, ErrWebhookNotFound
	}
	return s.repo.ListDeliveries(ctx, id, 20)
}
