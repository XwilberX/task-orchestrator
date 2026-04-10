package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// LogEntry representa una línea de log enviada/recibida de Victoria Logs.
type LogEntry struct {
	Msg            string    `json:"_msg"`
	Time           time.Time `json:"_time"`
	TaskID         string    `json:"task_id"`
	DefinitionName string    `json:"definition_name,omitempty"`
	Runtime        string    `json:"runtime"`
	Attempt        int       `json:"attempt"`
	Stream         string    `json:"stream"` // "stdout" | "stderr"
}

// Client es el cliente asíncrono de Victoria Logs.
type Client struct {
	baseURL string
	http    *http.Client
	buf     chan LogEntry
}

// New crea un Client e inicia el goroutine de flush.
func New(baseURL string) *Client {
	c := &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
		buf:     make(chan LogEntry, 2048),
	}
	go c.flushLoop()
	return c
}

// Write encola una entrada de log (no bloquea).
func (c *Client) Write(entry LogEntry) {
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	}
	select {
	case c.buf <- entry:
	default:
		// buffer lleno — se descarta para no bloquear la ejecución de la tarea
	}
}

// Query obtiene los logs de una tarea desde Victoria Logs.
func (c *Client) Query(ctx context.Context, taskID string, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 1000
	}
	query := fmt.Sprintf(`task_id:"%s"`, taskID)
	endpoint := fmt.Sprintf("%s/select/logsql/query?query=%s&limit=%d",
		c.baseURL,
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("victoria logs query error %d: %s", resp.StatusCode, body)
	}

	// Victoria Logs devuelve JSON lines (un objeto por línea)
	var entries []LogEntry
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var e LogEntry
		if err := dec.Decode(&e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// flushLoop envía el buffer a Victoria Logs cada 500ms.
func (c *Client) flushLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]LogEntry, 0, 64)

	for {
		select {
		case entry := <-c.buf:
			batch = append(batch, entry)
			// drain lo que haya acumulado sin bloquear
			for len(batch) < 256 {
				select {
				case e := <-c.buf:
					batch = append(batch, e)
				default:
					goto flush
				}
			}
		flush:
			if len(batch) > 0 {
				c.send(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				c.send(batch)
				batch = batch[:0]
			}
		}
	}
}

// send envía un batch de entradas a /insert/jsonline.
func (c *Client) send(entries []LogEntry) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			continue
		}
	}

	resp, err := c.http.Post(
		c.baseURL+"/insert/jsonline",
		"application/stream+json",
		&buf,
	)
	if err != nil {
		log.Printf("vlogs: error enviando logs: %v", err)
		return
	}
	resp.Body.Close()
}
