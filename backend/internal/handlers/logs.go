package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/XwilberX/task-orchestrator/internal/logger"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/XwilberX/task-orchestrator/pkg/response"
	"github.com/go-chi/chi/v5"
)

type LogHandler struct {
	taskSvc *services.TaskService
	vlogs   *logger.Client
}

func NewLogHandler(taskSvc *services.TaskService, vlogs *logger.Client) *LogHandler {
	return &LogHandler{taskSvc: taskSvc, vlogs: vlogs}
}

// GetTaskLogs devuelve los logs históricos de una tarea desde Victoria Logs.
func (h *LogHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verificar que la tarea existe
	_, err := h.taskSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}

	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	entries, err := h.vlogs.Query(r.Context(), id, limit)
	if err != nil {
		response.InternalError(w, err)
		return
	}

	response.OK(w, entries, "")
}
