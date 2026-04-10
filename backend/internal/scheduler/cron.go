package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/repositories"
	"github.com/robfig/cron/v3"
)

// Dispatcher es la interfaz que implementa TaskService para despachar tareas.
type Dispatcher interface {
	Dispatch(ctx context.Context, req models.DispatchRequest) (*models.Task, error)
}

// CronScheduler gestiona los schedules activos en memoria.
type CronScheduler struct {
	c         *cron.Cron
	entries   map[string]cron.EntryID // scheduleID → entryID en robfig/cron
	mu        sync.Mutex
	dispatcher Dispatcher
	schedRepo *repositories.ScheduleRepository
	defRepo   *repositories.DefinitionRepository
}

func NewCronScheduler(
	dispatcher Dispatcher,
	schedRepo *repositories.ScheduleRepository,
	defRepo *repositories.DefinitionRepository,
) *CronScheduler {
	return &CronScheduler{
		c:          cron.New(),
		entries:    make(map[string]cron.EntryID),
		dispatcher: dispatcher,
		schedRepo:  schedRepo,
		defRepo:    defRepo,
	}
}

// Load carga todos los schedules activos desde MongoDB al arrancar.
func (cs *CronScheduler) Load(ctx context.Context) error {
	schedules, err := cs.schedRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	for _, s := range schedules {
		s := s
		if err := cs.register(&s); err != nil {
			log.Printf("cron: no se pudo registrar schedule %s (%s): %v", s.ID, s.Cron, err)
		} else {
			log.Printf("cron: schedule %s registrado (%s)", s.ID, s.Cron)
		}
	}
	return nil
}

// Add registra un nuevo schedule en el cron.
func (cs *CronScheduler) Add(s *models.Schedule) error {
	return cs.register(s)
}

// Remove elimina un schedule del cron.
func (cs *CronScheduler) Remove(scheduleID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if id, ok := cs.entries[scheduleID]; ok {
		cs.c.Remove(id)
		delete(cs.entries, scheduleID)
	}
}

// Start arranca el cron.
func (cs *CronScheduler) Start() {
	cs.c.Start()
}

// Stop detiene el cron.
func (cs *CronScheduler) Stop() {
	cs.c.Stop()
}

// NextRun calcula la próxima ejecución de una expresión cron.
func NextRun(cronExpr string) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(time.Now().UTC()), nil
}

// register añade el schedule al cron interno.
func (cs *CronScheduler) register(s *models.Schedule) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Si ya existe, eliminarlo antes de re-registrar
	if id, ok := cs.entries[s.ID]; ok {
		cs.c.Remove(id)
	}

	scheduleID := s.ID
	definitionID := s.DefinitionID

	entryID, err := cs.c.AddFunc(s.Cron, func() {
		cs.fire(scheduleID, definitionID)
	})
	if err != nil {
		return err
	}
	cs.entries[scheduleID] = entryID
	return nil
}

// fire se ejecuta cuando llega el momento de un schedule.
func (cs *CronScheduler) fire(scheduleID, definitionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	def, err := cs.defRepo.GetByID(ctx, definitionID)
	if err != nil || def == nil {
		log.Printf("cron: definición %s no encontrada para schedule %s", definitionID, scheduleID)
		return
	}

	_, err = cs.dispatcher.Dispatch(ctx, models.DispatchRequest{
		Definition: def.Name,
	})
	if err != nil {
		log.Printf("cron: error despachando tarea para schedule %s: %v", scheduleID, err)
		return
	}

	// Actualizar last_run_at y next_run_at en MongoDB
	now := time.Now().UTC()
	next, _ := NextRun(def.Name) // ignoramos error, ya validado al crear
	cs.schedRepo.UpdateNextRun(ctx, scheduleID, now, next)
	log.Printf("cron: schedule %s ejecutado, próxima ejecución: %s", scheduleID, next)
}
