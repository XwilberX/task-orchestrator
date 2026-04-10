package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/events"
	"github.com/XwilberX/task-orchestrator/internal/logger"
	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/go-chi/chi/v5"
)

const heartbeatInterval = 30 * time.Second

type SSEHandler struct {
	taskSvc   *services.TaskService
	broker    *events.Broker
	logBroker *events.LogBroker
	vlogs     *logger.Client
}

func NewSSEHandler(
	taskSvc *services.TaskService,
	broker *events.Broker,
	logBroker *events.LogBroker,
	vlogs *logger.Client,
) *SSEHandler {
	return &SSEHandler{taskSvc: taskSvc, broker: broker, logBroker: logBroker, vlogs: vlogs}
}

// StreamEvents emite eventos globales de cambio de estado de tareas.
// GET /api/v1/events
func (h *SSEHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming no soportado", http.StatusInternalServerError)
		return
	}

	sseHeaders(w)

	ch, cancel := h.broker.Subscribe()
	defer cancel()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// StreamTaskLogs emite logs en vivo de una tarea RUNNING.
// Si la tarea está en estado terminal, devuelve los logs históricos de Victoria Logs.
// GET /api/v1/tasks/:id/stream
func (h *SSEHandler) StreamTaskLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	task, err := h.taskSvc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "tarea no encontrada", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming no soportado", http.StatusInternalServerError)
		return
	}

	sseHeaders(w)

	// Tarea en estado terminal — devolver logs históricos y cerrar
	if task.Status.IsTerminal() {
		entries, err := h.vlogs.Query(r.Context(), id, 1000)
		if err == nil {
			for _, e := range entries {
				fmt.Fprintf(w, "data: %s\n\n", e.Msg)
				flusher.Flush()
			}
		}
		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		flusher.Flush()
		return
	}

	// Tarea RUNNING o QUEUED — suscribir al log broker en tiempo real
	ch, cancel := h.logBroker.Subscribe(id)
	defer cancel()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	// También observar cambios de estado para cerrar el stream cuando termine
	eventCh, eventCancel := h.broker.Subscribe()
	defer eventCancel()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-ch:
			if !ok {
				// Log broker cerrado — tarea terminó
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case ev := <-eventCh:
			if ev.TaskID == id && ev.Status.IsTerminal() {
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(ev.Status))
				flusher.Flush()
				return
			}
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

// sseEvent emite un evento SSE tipado — usado para test/debug.
func sseEvent(w http.ResponseWriter, eventType string, status models.TaskStatus) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, status)
}

var _ = sseEvent // evitar "unused"
