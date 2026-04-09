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

type DefinitionRepository struct {
	col *mongo.Collection
}

func NewDefinitionRepository(db *mongo.Database) *DefinitionRepository {
	return &DefinitionRepository{col: db.Collection("definitions")}
}

func (r *DefinitionRepository) Create(ctx context.Context, d *models.Definition) error {
	_, err := r.col.InsertOne(ctx, d)
	return err
}

func (r *DefinitionRepository) GetByID(ctx context.Context, id string) (*models.Definition, error) {
	var d models.Definition
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &d, err
}

func (r *DefinitionRepository) GetByName(ctx context.Context, name string) (*models.Definition, error) {
	var d models.Definition
	err := r.col.FindOne(ctx, bson.M{"name": name}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &d, err
}

func (r *DefinitionRepository) List(ctx context.Context) ([]models.Definition, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var results []models.Definition
	return results, cursor.All(ctx, &results)
}

func (r *DefinitionRepository) Update(ctx context.Context, id string, d *models.Definition) error {
	d.UpdatedAt = time.Now().UTC()
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": id}, d)
	return err
}

func (r *DefinitionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
