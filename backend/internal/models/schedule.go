package models

import "time"

type ScheduleStatus string

const (
	ScheduleActive ScheduleStatus = "active"
	SchedulePaused ScheduleStatus = "paused"
)

type Schedule struct {
	ID           string         `json:"id" bson:"_id"`
	DefinitionID string         `json:"definition_id" bson:"definition_id" validate:"required"`
	Cron         string         `json:"cron" bson:"cron" validate:"required"`
	Status       ScheduleStatus `json:"status" bson:"status"`
	LastRunAt    *time.Time     `json:"last_run_at,omitempty" bson:"last_run_at,omitempty"`
	NextRunAt    *time.Time     `json:"next_run_at,omitempty" bson:"next_run_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at" bson:"created_at"`
}
