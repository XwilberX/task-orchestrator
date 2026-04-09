package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/config"
	"github.com/XwilberX/task-orchestrator/internal/handlers"
	apimw "github.com/XwilberX/task-orchestrator/internal/middleware"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// MongoDB
	client, err := repositories.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(cfg.MongoDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repositories.CreateIndexes(ctx, db); err != nil {
		log.Fatalf("indexes: %v", err)
	}
	log.Printf("conectado a MongoDB: %s", cfg.MongoDB)

	// Repositorios
	defRepo := repositories.NewDefinitionRepository(db)

	// Servicios
	defSvc := services.NewDefinitionService(defRepo)

	// Handlers
	defHandler := handlers.NewDefinitionHandler(defSvc)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1 — protegida por API key
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apimw.APIKey(cfg.APIKey))
		r.Mount("/definitions", defHandler.Routes())
	})

	log.Printf("servidor iniciado en :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
