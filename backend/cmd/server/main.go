package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/config"
	"github.com/XwilberX/task-orchestrator/internal/executor"
	"github.com/XwilberX/task-orchestrator/internal/handlers"
	apimw "github.com/XwilberX/task-orchestrator/internal/middleware"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/XwilberX/task-orchestrator/internal/scheduler"
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
	mongoClient, err := repositories.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.MongoDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repositories.CreateIndexes(ctx, db); err != nil {
		log.Fatalf("indexes: %v", err)
	}
	log.Printf("conectado a MongoDB: %s", cfg.MongoDB)

	// Executor
	exec, err := executor.New(cfg.GVisorRuntime)
	if err != nil {
		log.Fatalf("executor: %v", err)
	}

	// Repositorios
	defRepo := repositories.NewDefinitionRepository(db)
	taskRepo := repositories.NewTaskRepository(db)

	// Worker pool
	maxConcurrent, _ := strconv.Atoi(cfg.MaxConcurrentTasks)
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	pool := scheduler.New(maxConcurrent, taskRepo, defRepo, exec)

	// Recovery: re-encolar tareas RUNNING de ejecuciones previas
	recCtx, recCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer recCancel()
	if err := pool.Recover(recCtx); err != nil {
		log.Printf("recovery warning: %v", err)
	}

	// Iniciar polling de tareas QUEUED
	poolCtx, poolCancel := context.WithCancel(context.Background())
	defer poolCancel()
	pool.Start(poolCtx)

	// Servicios
	defSvc := services.NewDefinitionService(defRepo)
	taskSvc := services.NewTaskService(taskRepo, defRepo, pool)

	// Handlers
	defHandler := handlers.NewDefinitionHandler(defSvc)
	taskHandler := handlers.NewTaskHandler(taskSvc)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apimw.APIKey(cfg.APIKey))
		r.Mount("/definitions", defHandler.Routes())
		r.Mount("/tasks", taskHandler.Routes())
	})

	log.Printf("servidor iniciado en :%s (max_concurrent=%d)", cfg.Port, maxConcurrent)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
