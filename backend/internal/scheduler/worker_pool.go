package scheduler

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/events"
	"github.com/XwilberX/task-orchestrator/internal/executor"
	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const pollInterval = 3 * time.Second

// WebhookNotifier es la interfaz que implementa webhook.Dispatcher.
type WebhookNotifier interface {
	Notify(task *models.Task)
}

// WorkerPool controla la concurrencia global de ejecución de tareas.
type WorkerPool struct {
	slots     chan struct{} // semáforo global
	taskRepo  *repositories.TaskRepository
	defRepo   *repositories.DefinitionRepository
	exec      *executor.Executor
	broker    *events.Broker
	logBroker *events.LogBroker
	webhooks  WebhookNotifier
}

// New crea un WorkerPool con el límite de concurrencia dado.
func New(
	maxConcurrent int,
	taskRepo *repositories.TaskRepository,
	defRepo *repositories.DefinitionRepository,
	exec *executor.Executor,
	broker *events.Broker,
	logBroker *events.LogBroker,
	webhooks WebhookNotifier,
) *WorkerPool {
	return &WorkerPool{
		slots:     make(chan struct{}, maxConcurrent),
		taskRepo:  taskRepo,
		defRepo:   defRepo,
		exec:      exec,
		broker:    broker,
		logBroker: logBroker,
		webhooks:  webhooks,
	}
}

// Submit intenta ejecutar la tarea inmediatamente.
// Si no hay slot disponible o se excede max_concurrent de la definición,
// la tarea pasa a QUEUED y el scheduler la recogerá.
func (p *WorkerPool) Submit(task *models.Task) {
	select {
	case p.slots <- struct{}{}:
		if p.exceedsDefLimit(task) {
			<-p.slots
			p.queue(task)
			return
		}
		go p.runWithRetries(task)
	default:
		p.queue(task)
	}
}

// Start lanza el goroutine de polling que recoge tareas QUEUED.
func (p *WorkerPool) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.drainQueue(ctx)
			}
		}
	}()
}

// Recover re-encola tareas que quedaron en RUNNING al reiniciar el servidor.
func (p *WorkerPool) Recover(ctx context.Context) error {
	tasks, err := p.taskRepo.List(ctx, models.TaskFilter{Status: string(models.StatusRunning)}, 0)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := p.taskRepo.UpdateStatus(ctx, t.ID, models.StatusQueued); err != nil {
			log.Printf("recovery: no se pudo re-encolar tarea %s: %v", t.ID, err)
		} else {
			log.Printf("recovery: tarea %s re-encolada", t.ID)
		}
	}
	return nil
}

// drainQueue intenta despachar todas las tareas QUEUED que quepan en el pool.
func (p *WorkerPool) drainQueue(ctx context.Context) {
	tasks, err := p.taskRepo.FindQueued(ctx, cap(p.slots))
	if err != nil {
		return
	}
	for _, t := range tasks {
		t := t // captura
		select {
		case p.slots <- struct{}{}:
			if p.exceedsDefLimit(&t) {
				<-p.slots
				continue
			}
			go p.runWithRetries(&t)
		default:
			return // pool lleno
		}
	}
}

// queue transiciona la tarea a QUEUED.
func (p *WorkerPool) queue(task *models.Task) {
	if err := p.taskRepo.UpdateStatus(context.Background(), task.ID, models.StatusQueued); err != nil {
		log.Printf("queue: error al encolar tarea %s: %v", task.ID, err)
	}
}

// runWithRetries ejecuta la tarea y aplica reintentos con backoff exponencial.
// Siempre libera el slot al terminar.
func (p *WorkerPool) runWithRetries(task *models.Task) {
	defer func() { <-p.slots }()

	for {
		result := p.runOnce(task)

		finalStatus := models.StatusSuccess
		if result.TimedOut {
			finalStatus = models.StatusTimeout
		} else if result.ExitCode != 0 {
			finalStatus = models.StatusFailed
		}

		now := time.Now().UTC()
		p.taskRepo.UpdateFields(context.Background(), task.ID, bson.M{
			"status":      finalStatus,
			"finished_at": now,
			"exit_code":   result.ExitCode,
		})
		p.publish(task.ID, finalStatus)

		// Cerrar el log broker para esta tarea
		if p.logBroker != nil {
			p.logBroker.Close(task.ID)
		}

		// ¿Reintentamos?
		willRetry := (finalStatus == models.StatusFailed || finalStatus == models.StatusTimeout) &&
			task.Attempt < task.MaxRetries
		if !willRetry && p.webhooks != nil {
			// Notificar webhooks solo en el estado terminal definitivo
			task.Status = finalStatus
			task.FinishedAt = &now
			p.webhooks.Notify(task)
		}

		if (finalStatus == models.StatusFailed || finalStatus == models.StatusTimeout) &&
			task.Attempt < task.MaxRetries {

			backoff := time.Duration(math.Pow(float64(task.BackoffMultiplier), float64(task.Attempt))) * time.Second
			log.Printf("tarea %s fallida (intento %d/%d), reintentando en %s", task.ID, task.Attempt, task.MaxRetries, backoff)
			time.Sleep(backoff)

			task.Attempt++
			p.taskRepo.UpdateFields(context.Background(), task.ID, bson.M{
				"status":  models.StatusPending,
				"attempt": task.Attempt,
			})
			p.publish(task.ID, models.StatusPending)
			continue
		}

		break
	}
}

func (p *WorkerPool) publish(taskID string, status models.TaskStatus) {
	if p.broker != nil {
		p.broker.Publish(events.TaskEvent{TaskID: taskID, Status: status})
	}
}

// runOnce ejecuta el contenedor una sola vez.
func (p *WorkerPool) runOnce(task *models.Task) *executor.ExecResult {
	now := time.Now().UTC()
	p.taskRepo.UpdateFields(context.Background(), task.ID, bson.M{
		"status":     models.StatusRunning,
		"started_at": now,
	})
	p.publish(task.ID, models.StatusRunning)

	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := p.exec.Run(ctx, executor.RunConfig{
		TaskID:         task.ID,
		DefinitionName: task.DefinitionName,
		Attempt:        task.Attempt,
		Runtime:        task.Runtime,
		Code:           task.Code,
		Args:           task.Args,
		Packages:       task.Packages,
		TimeoutSeconds: task.TimeoutSeconds,
		MemoryMB:       task.MemoryMB,
		CPUShares:      task.CPUShares,
		NetworkEnabled: task.NetworkEnabled,
	})
	if err != nil {
		log.Printf("executor error en tarea %s: %v", task.ID, err)
		return &executor.ExecResult{ExitCode: -1}
	}
	return result
}

// exceedsDefLimit comprueba si la definición ya alcanzó su max_concurrent.
func (p *WorkerPool) exceedsDefLimit(task *models.Task) bool {
	if task.DefinitionID == "" {
		return false
	}
	def, err := p.defRepo.GetByID(context.Background(), task.DefinitionID)
	if err != nil || def == nil {
		return false
	}
	running, err := p.taskRepo.CountRunningByDefinition(context.Background(), task.DefinitionID)
	if err != nil {
		return false
	}
	return running >= int64(def.MaxConcurrent)
}
