package models

import "time"

type TaskStatus string

const (
	StatusPending   TaskStatus = "PENDING"
	StatusQueued    TaskStatus = "QUEUED"
	StatusRunning   TaskStatus = "RUNNING"
	StatusSuccess   TaskStatus = "SUCCESS"
	StatusFailed    TaskStatus = "FAILED"
	StatusTimeout   TaskStatus = "TIMEOUT"
	StatusCancelled TaskStatus = "CANCELLED"
)

func (s TaskStatus) IsTerminal() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusTimeout || s == StatusCancelled
}

type Task struct {
	ID                string                 `json:"id" bson:"_id"`
	DefinitionID      string                 `json:"definition_id,omitempty" bson:"definition_id,omitempty"`
	DefinitionName    string                 `json:"definition,omitempty" bson:"definition,omitempty"`
	Runtime           string                 `json:"runtime" bson:"runtime"`
	RuntimeVersion    string                 `json:"runtime_version,omitempty" bson:"runtime_version,omitempty"`
	Code              string                 `json:"code" bson:"code"`
	Args              []string               `json:"args,omitempty" bson:"args,omitempty"`
	Packages          string                 `json:"packages,omitempty" bson:"packages,omitempty"`
	Input             map[string]interface{} `json:"input,omitempty" bson:"input,omitempty"`
	Status            TaskStatus             `json:"status" bson:"status"`
	Attempt           int                    `json:"attempt" bson:"attempt"`
	MaxRetries        int                    `json:"max_retries" bson:"max_retries"`
	BackoffMultiplier int                    `json:"backoff_multiplier" bson:"backoff_multiplier"`
	TimeoutSeconds    int                    `json:"timeout_seconds" bson:"timeout_seconds"`
	MemoryMB          int                    `json:"memory_mb" bson:"memory_mb"`
	CPUShares         int                    `json:"cpu_shares" bson:"cpu_shares"`
	NetworkEnabled    bool                   `json:"network_enabled" bson:"network_enabled"`
	ContainerID       string                 `json:"container_id,omitempty" bson:"container_id,omitempty"`
	ExitCode          *int                   `json:"exit_code,omitempty" bson:"exit_code,omitempty"`
	OutputData        interface{}            `json:"output_data,omitempty" bson:"output_data,omitempty"`
	CreatedAt         time.Time              `json:"created_at" bson:"created_at"`
	StartedAt         *time.Time             `json:"started_at,omitempty" bson:"started_at,omitempty"`
	FinishedAt        *time.Time             `json:"finished_at,omitempty" bson:"finished_at,omitempty"`
}

// DispatchRequest es el payload para despachar una tarea
type DispatchRequest struct {
	// Tarea registrada
	Definition string                 `json:"definition"`
	Input      map[string]interface{} `json:"input"`

	// Tarea ad-hoc
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtime_version"`
	Code           string   `json:"code"`
	Args     []string `json:"args"`
	Packages string   `json:"packages"`

	// Común
	TimeoutSeconds int `json:"timeout_seconds"`
	MemoryMB       int `json:"memory_mb"`
}

// TaskFilter son los filtros para listar tareas
type TaskFilter struct {
	Status       string
	DefinitionID string
	Runtime      string
	From         *time.Time
	To           *time.Time
}
