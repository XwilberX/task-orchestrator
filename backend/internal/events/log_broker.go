package events

import "sync"

// LogBroker distribuye líneas de log en tiempo real por task_id.
type LogBroker struct {
	mu       sync.RWMutex
	channels map[string][]chan string
}

func NewLogBroker() *LogBroker {
	return &LogBroker{channels: make(map[string][]chan string)}
}

// Subscribe suscribe a los logs de una tarea específica.
func (b *LogBroker) Subscribe(taskID string) (chan string, func()) {
	ch := make(chan string, 256)
	b.mu.Lock()
	b.channels[taskID] = append(b.channels[taskID], ch)
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.channels[taskID]
		for i, s := range subs {
			if s == ch {
				b.channels[taskID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
		if len(b.channels[taskID]) == 0 {
			delete(b.channels, taskID)
		}
	}
	return ch, cancel
}

// Publish envía una línea de log a todos los suscriptores de la tarea.
func (b *LogBroker) Publish(taskID, line string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.channels[taskID] {
		select {
		case ch <- line:
		default:
		}
	}
}

// Close cierra todos los canales de una tarea (cuando termina de ejecutar).
func (b *LogBroker) Close(taskID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.channels[taskID] {
		close(ch)
	}
	delete(b.channels, taskID)
}
