package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/google/uuid"
)

const (
	maxAttempts  = 3
	retryBackoff = 10 * time.Second
)

// Job es una tarea que llegó a estado terminal y debe notificarse.
type Job struct {
	Task *models.Task
}

// Dispatcher gestiona la entrega de webhooks de forma asíncrona.
type Dispatcher struct {
	repo   *repositories.WebhookRepository
	queue  chan Job
	client *http.Client
	secret string // clave HMAC-SHA256
}

// New crea un Dispatcher. secret se usa para firmar los payloads.
func New(repo *repositories.WebhookRepository, secret string) *Dispatcher {
	return &Dispatcher{
		repo:   repo,
		queue:  make(chan Job, 256),
		client: &http.Client{Timeout: 15 * time.Second},
		secret: secret,
	}
}

// Notify encola la notificación para una tarea en estado terminal.
// No bloquea — si la cola está llena, se descarta silenciosamente.
func (d *Dispatcher) Notify(task *models.Task) {
	select {
	case d.queue <- Job{Task: task}:
	default:
		log.Printf("webhook: cola llena, descartando notificación para tarea %s", task.ID)
	}
}

// Start procesa la cola de jobs en background.
func (d *Dispatcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-d.queue:
				d.process(job)
			}
		}
	}()
}

// process envía el webhook a todos los endpoints registrados.
func (d *Dispatcher) process(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	webhooks, err := d.repo.ListAll(ctx)
	if err != nil {
		log.Printf("webhook: error obteniendo webhooks: %v", err)
		return
	}

	payload := buildPayload(job.Task)
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, wh := range webhooks {
		d.deliverWithRetries(ctx, wh.ID, job.Task.ID, wh.URL, body)
	}
}

// deliverWithRetries intenta entregar el webhook hasta maxAttempts veces.
func (d *Dispatcher) deliverWithRetries(ctx context.Context, webhookID, taskID, url string, body []byte) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		statusCode, err := d.deliver(ctx, url, body)
		success := err == nil && statusCode >= 200 && statusCode < 300

		delivery := &models.WebhookDelivery{
			ID:         uuid.NewString(),
			WebhookID:  webhookID,
			TaskID:     taskID,
			StatusCode: statusCode,
			Success:    success,
			AttemptAt:  time.Now().UTC(),
		}
		if err != nil {
			delivery.Response = err.Error()
		}
		d.repo.CreateDelivery(context.Background(), delivery)

		if success {
			return
		}

		lastErr = err
		if attempt < maxAttempts {
			log.Printf("webhook: intento %d/%d fallido para %s (task %s), reintentando en %s",
				attempt, maxAttempts, url, taskID, retryBackoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryBackoff):
			}
		}
	}
	log.Printf("webhook: todos los intentos fallidos para %s (task %s): %v", url, taskID, lastErr)
}

// deliver hace el POST al endpoint del cliente.
func (d *Dispatcher) deliver(ctx context.Context, url string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", d.sign(body))

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// sign genera la firma HMAC-SHA256 del payload.
func (d *Dispatcher) sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(d.secret))
	mac.Write(body)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}

// buildPayload construye el WebhookPayload desde una tarea.
func buildPayload(t *models.Task) models.WebhookPayload {
	p := models.WebhookPayload{
		Event:      "task.completed",
		TaskID:     t.ID,
		Definition: t.DefinitionName,
		Status:     t.Status,
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
		Attempt:    t.Attempt,
	}
	p.Output.ExitCode = t.ExitCode
	return p
}
