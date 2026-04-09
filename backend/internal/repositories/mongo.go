package repositories

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func CreateIndexes(ctx context.Context, db *mongo.Database) error {
	// tasks: índices para filtrado y cola
	_, err := db.Collection("tasks").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "definition_id", Value: 1}}},
		{Keys: bson.D{{Key: "runtime", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
	if err != nil {
		return err
	}

	// definitions: nombre único
	_, err = db.Collection("definitions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// schedules: índice para el scheduler
	_, err = db.Collection("schedules").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "next_run_at", Value: 1}},
	})
	return err
}
