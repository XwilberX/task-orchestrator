package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/XwilberX/task-orchestrator/pkg/response"
	"github.com/go-chi/chi/v5"
)

type WebhookHandler struct {
	svc *services.WebhookService
}

func NewWebhookHandler(svc *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/logs", h.ListDeliveries)
	return r
}

func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		response.BadRequest(w, errors.New("url requerida"), "url requerida")
		return
	}
	wh, err := h.svc.Create(r.Context(), body.URL)
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.Created(w, wh)
}

func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, list, "")
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, services.ErrWebhookNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, nil, "webhook eliminado")
}

func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deliveries, err := h.svc.ListDeliveries(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrWebhookNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, deliveries, "")
}
