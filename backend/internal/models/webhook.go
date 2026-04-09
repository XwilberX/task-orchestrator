package models

import "time"

type Webhook struct {
	ID        string    `json:"id" bson:"_id"`
	URL       string    `json:"url" bson:"url" validate:"required,url"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type WebhookDelivery struct {
	ID         string    `json:"id" bson:"_id"`
	WebhookID  string    `json:"webhook_id" bson:"webhook_id"`
	TaskID     string    `json:"task_id" bson:"task_id"`
	StatusCode int       `json:"status_code" bson:"status_code"`
	Success    bool      `json:"success" bson:"success"`
	Response   string    `json:"response" bson:"response"`
	AttemptAt  time.Time `json:"attempt_at" bson:"attempt_at"`
}

// WebhookPayload es lo que se envía al endpoint del cliente
type WebhookPayload struct {
	Event      string     `json:"event"`
	TaskID     string     `json:"task_id"`
	Definition string     `json:"definition,omitempty"`
	Status     TaskStatus `json:"status"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Attempt    int        `json:"attempt"`
	Output     struct {
		ExitCode *int `json:"exit_code,omitempty"`
	} `json:"output"`
}
