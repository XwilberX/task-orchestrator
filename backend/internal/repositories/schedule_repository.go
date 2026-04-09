package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ScheduleRepository struct {
	col *mongo.Collection
}

func NewScheduleRepository(db *mongo.Database) *ScheduleRepository {
	return &ScheduleRepository{col: db.Collection("schedules")}
}

func (r *ScheduleRepository) Create(ctx context.Context, s *models.Schedule) error {
	_, err := r.col.InsertOne(ctx, s)
	return err
}

func (r *ScheduleRepository) GetByID(ctx context.Context, id string) (*models.Schedule, error) {
	var s models.Schedule
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &s, err
}

func (r *ScheduleRepository) List(ctx context.Context) ([]models.Schedule, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Schedule
	return results, cursor.All(ctx, &results)
}

func (r *ScheduleRepository) Update(ctx context.Context, id string, s *models.Schedule) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": id}, s)
	return err
}

func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *ScheduleRepository) UpdateNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"last_run_at": lastRun,
			"next_run_at": nextRun,
		},
	})
	return err
}

func (r *ScheduleRepository) ListActive(ctx context.Context) ([]models.Schedule, error) {
	opts := options.Find().SetSort(bson.D{{Key: "next_run_at", Value: 1}})
	cursor, err := r.col.Find(ctx, bson.M{"status": models.ScheduleActive}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Schedule
	return results, cursor.All(ctx, &results)
}
