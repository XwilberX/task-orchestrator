package repositories

import (
	"context"
	"errors"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type WebhookRepository struct {
	col         *mongo.Collection
	deliveryCol *mongo.Collection
}

func NewWebhookRepository(db *mongo.Database) *WebhookRepository {
	return &WebhookRepository{
		col:         db.Collection("webhooks"),
		deliveryCol: db.Collection("webhook_deliveries"),
	}
}

func (r *WebhookRepository) Create(ctx context.Context, w *models.Webhook) error {
	_, err := r.col.InsertOne(ctx, w)
	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*models.Webhook, error) {
	var w models.Webhook
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&w)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &w, err
}

func (r *WebhookRepository) List(ctx context.Context) ([]models.Webhook, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Webhook
	return results, cursor.All(ctx, &results)
}

func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *WebhookRepository) CreateDelivery(ctx context.Context, d *models.WebhookDelivery) error {
	_, err := r.deliveryCol.InsertOne(ctx, d)
	return err
}

func (r *WebhookRepository) ListDeliveries(ctx context.Context, webhookID string, limit int) ([]models.WebhookDelivery, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "attempt_at", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.deliveryCol.Find(ctx, bson.M{"webhook_id": webhookID}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.WebhookDelivery
	return results, cursor.All(ctx, &results)
}

func (r *WebhookRepository) ListAll(ctx context.Context) ([]models.Webhook, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var results []models.Webhook
	return results, cursor.All(ctx, &results)
}
