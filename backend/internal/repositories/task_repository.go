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

type TaskRepository struct {
	col *mongo.Collection
}

func NewTaskRepository(db *mongo.Database) *TaskRepository {
	return &TaskRepository{col: db.Collection("tasks")}
}

func (r *TaskRepository) Create(ctx context.Context, t *models.Task) error {
	_, err := r.col.InsertOne(ctx, t)
	return err
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	var t models.Task
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &t, err
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error {
	update := bson.M{"$set": bson.M{"status": status}}
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *TaskRepository) UpdateFields(ctx context.Context, id string, fields bson.M) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": fields})
	return err
}

func (r *TaskRepository) List(ctx context.Context, filter models.TaskFilter, limit int) ([]models.Task, error) {
	query := bson.M{}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.DefinitionID != "" {
		query["definition_id"] = filter.DefinitionID
	}
	if filter.Runtime != "" {
		query["runtime"] = filter.Runtime
	}
	if filter.From != nil || filter.To != nil {
		rangeFilter := bson.M{}
		if filter.From != nil {
			rangeFilter["$gte"] = filter.From
		}
		if filter.To != nil {
			rangeFilter["$lte"] = filter.To
		}
		query["created_at"] = rangeFilter
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Task
	return results, cursor.All(ctx, &results)
}

// FindQueued devuelve tareas en estado QUEUED ordenadas por created_at
func (r *TaskRepository) FindQueued(ctx context.Context, limit int) ([]models.Task, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(int64(limit))
	cursor, err := r.col.Find(ctx, bson.M{"status": models.StatusQueued}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Task
	return results, cursor.All(ctx, &results)
}

// CountByDefinition cuenta tareas RUNNING para un definition_id dado
func (r *TaskRepository) CountRunningByDefinition(ctx context.Context, definitionID string) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{
		"definition_id": definitionID,
		"status":        models.StatusRunning,
	})
}

// MetricsSummary devuelve métricas agregadas del día
func (r *TaskRepository) MetricsSummary(ctx context.Context) (map[string]interface{}, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)

	pipeline := mongo.Pipeline{
		{{Key: "$facet", Value: bson.M{
			"tasks_today": bson.A{
				bson.M{"$match": bson.M{"created_at": bson.M{"$gte": startOfDay}}},
				bson.M{"$count": "count"},
			},
			"tasks_failed": bson.A{
				bson.M{"$match": bson.M{
					"created_at": bson.M{"$gte": startOfDay},
					"status":     models.StatusFailed,
				}},
				bson.M{"$count": "count"},
			},
			"tasks_queued": bson.A{
				bson.M{"$match": bson.M{"status": models.StatusQueued}},
				bson.M{"$count": "count"},
			},
			"tasks_running": bson.A{
				bson.M{"$match": bson.M{"status": models.StatusRunning}},
				bson.M{"$count": "count"},
			},
			"avg_duration": bson.A{
				bson.M{"$match": bson.M{
					"created_at": bson.M{"$gte": startOfDay},
					"started_at": bson.M{"$exists": true},
					"finished_at": bson.M{"$exists": true},
				}},
				bson.M{"$group": bson.M{
					"_id": nil,
					"avg": bson.M{"$avg": bson.M{
						"$divide": bson.A{
							bson.M{"$subtract": bson.A{"$finished_at", "$started_at"}},
							1000,
						},
					}},
				}},
			},
		}}},
	}

	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	extract := func(key string) int {
		if arr, ok := results[0][key].(bson.A); ok && len(arr) > 0 {
			if doc, ok := arr[0].(bson.M); ok {
				if v, ok := doc["count"].(int32); ok {
					return int(v)
				}
			}
		}
		return 0
	}

	avgDuration := 0.0
	if arr, ok := results[0]["avg_duration"].(bson.A); ok && len(arr) > 0 {
		if doc, ok := arr[0].(bson.M); ok {
			if v, ok := doc["avg"].(float64); ok {
				avgDuration = v
			}
		}
	}

	return map[string]interface{}{
		"tasks_today":          extract("tasks_today"),
		"tasks_failed":         extract("tasks_failed"),
		"tasks_queued":         extract("tasks_queued"),
		"tasks_running":        extract("tasks_running"),
		"avg_duration_seconds": avgDuration,
	}, nil
}
