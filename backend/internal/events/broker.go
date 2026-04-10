package events

import (
	"sync"

	"github.com/XwilberX/task-orchestrator/internal/models"
)

// TaskEvent representa un cambio de estado de una tarea.
type TaskEvent struct {
	TaskID string           `json:"task_id"`
	Status models.TaskStatus `json:"status"`
}

// Broker es el pub/sub global de eventos de estado de tareas.
type Broker struct {
	mu      sync.RWMutex
	clients map[chan TaskEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan TaskEvent]struct{})}
}

// Subscribe devuelve un canal de eventos y una función de cancelación.
func (b *Broker) Subscribe() (chan TaskEvent, func()) {
	ch := make(chan TaskEvent, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}
	return ch, cancel
}

// Publish envía un evento a todos los suscriptores.
func (b *Broker) Publish(event TaskEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
			// cliente lento — se descarta para no bloquear
		}
	}
}
