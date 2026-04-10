package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/XwilberX/task-orchestrator/pkg/response"
	"github.com/go-chi/chi/v5"
)

type ScheduleHandler struct {
	svc *services.ScheduleService
}

func NewScheduleHandler(svc *services.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

func (h *ScheduleHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}/toggle", h.Toggle)
	return r
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.Schedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}
	created, err := h.svc.Create(r.Context(), req)
	if err != nil {
		handleScheduleError(w, err)
		return
	}
	response.Created(w, created)
}

func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, list, "")
}

func (h *ScheduleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sched, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleScheduleError(w, err)
		return
	}
	response.OK(w, sched, "")
}

func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.Schedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}
	updated, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		handleScheduleError(w, err)
		return
	}
	response.OK(w, updated, "")
}

func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleScheduleError(w, err)
		return
	}
	response.OK(w, nil, "schedule eliminado")
}

func (h *ScheduleHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sched, err := h.svc.Toggle(r.Context(), id)
	if err != nil {
		handleScheduleError(w, err)
		return
	}
	response.OK(w, sched, "")
}

func handleScheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrScheduleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, services.ErrDefinitionNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, services.ErrInvalidCron):
		response.BadRequest(w, err, err.Error())
	default:
		response.InternalError(w, err)
	}
}
