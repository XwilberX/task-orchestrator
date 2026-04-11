package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/events"
	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/XwilberX/task-orchestrator/pkg/response"
	"github.com/go-chi/chi/v5"
)

type TaskHandler struct {
	svc    *services.TaskService
	broker *events.Broker
}

func NewTaskHandler(svc *services.TaskService, broker *events.Broker) *TaskHandler {
	return &TaskHandler{svc: svc, broker: broker}
}

func (h *TaskHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Dispatch)
	r.Post("/sync", h.DispatchSync)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Cancel)
	return r
}

func (h *TaskHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	var req models.DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}

	task, err := h.svc.Dispatch(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err, err.Error())
		return
	}
	response.Created(w, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := models.TaskFilter{
		Status:       q.Get("status"),
		DefinitionID: q.Get("definition_id"),
		Runtime:      q.Get("runtime"),
	}

	if from := q.Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err == nil {
			filter.From = &t
		}
	}
	if to := q.Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err == nil {
			filter.To = &t
		}
	}

	tasks, err := h.svc.List(r.Context(), filter)
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, tasks, "")
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, task, "")
}

func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Cancel(r.Context(), id); err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, services.ErrCannotCancel) {
			response.BadRequest(w, err, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, nil, "tarea cancelada")
}

// DispatchSync despacha una tarea y bloquea hasta que llega a estado terminal.
// Devuelve el resultado completo incluyendo output_data.
// Si el timeout de la tarea se agota antes de que responda, retorna 408 con el task_id.
func (h *TaskHandler) DispatchSync(w http.ResponseWriter, r *http.Request) {
	var req models.DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}

	// Suscribirse al broker ANTES de despachar para no perder el evento si la tarea
	// termina muy rápido entre el Dispatch y el Subscribe.
	ch, cancelSub := h.broker.Subscribe()
	defer cancelSub()

	task, err := h.svc.Dispatch(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err, err.Error())
		return
	}

	// Verificar si ya terminó (race: terminó entre Dispatch y Subscribe)
	current, err := h.svc.GetByID(r.Context(), task.ID)
	if err == nil && current.Status.IsTerminal() {
		response.OK(w, current, "")
		return
	}

	// Esperar hasta que la tarea termine. El timeout es el de la tarea + 30s de margen.
	timeout := time.Duration(task.TimeoutSeconds+30) * time.Second
	ctx, cancelTimeout := context.WithTimeout(r.Context(), timeout)
	defer cancelTimeout()

	for {
		select {
		case <-ctx.Done():
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"task_id": task.ID,
				"message": "timeout esperando resultado de la tarea",
			})
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.TaskID == task.ID && ev.Status.IsTerminal() {
				result, err := h.svc.GetByID(r.Context(), task.ID)
				if err != nil {
					response.InternalError(w, err)
					return
				}
				response.OK(w, result, "")
				return
			}
		}
	}
}
