package handlers

import (
	"net/http"

	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/XwilberX/task-orchestrator/pkg/response"
)

type MetricsHandler struct {
	taskRepo *repositories.TaskRepository
}

func NewMetricsHandler(taskRepo *repositories.TaskRepository) *MetricsHandler {
	return &MetricsHandler{taskRepo: taskRepo}
}

// Summary devuelve las métricas del día actual.
// GET /api/v1/metrics/summary
func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.taskRepo.MetricsSummary(r.Context())
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, summary, "")
}
