package models

import "time"

type Definition struct {
	ID                string    `json:"id" bson:"_id"`
	Name              string    `json:"name" bson:"name" validate:"required,min=1,max=100"`
	Description       string    `json:"description" bson:"description"`
	Runtime           string    `json:"runtime" bson:"runtime" validate:"required,oneof=python nodejs typescript go java"`
	Code              string    `json:"code" bson:"code" validate:"required"`
	Packages          string    `json:"packages" bson:"packages"`
	TimeoutSeconds    int       `json:"timeout_seconds" bson:"timeout_seconds" validate:"min=1,max=3600"`
	MaxRetries        int       `json:"max_retries" bson:"max_retries" validate:"min=0,max=10"`
	BackoffMultiplier int       `json:"backoff_multiplier" bson:"backoff_multiplier" validate:"min=1"`
	MaxConcurrent     int       `json:"max_concurrent" bson:"max_concurrent" validate:"min=1"`
	MemoryMB          int       `json:"memory_mb" bson:"memory_mb" validate:"min=64"`
	CPUShares         int       `json:"cpu_shares" bson:"cpu_shares" validate:"min=0"`
	NetworkEnabled    bool      `json:"network_enabled" bson:"network_enabled"`
	CreatedAt         time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" bson:"updated_at"`
}

func DefaultDefinition() Definition {
	return Definition{
		TimeoutSeconds:    60,
		MaxRetries:        3,
		BackoffMultiplier: 5,
		MaxConcurrent:     1,
		MemoryMB:          256,
		CPUShares:         512,
		NetworkEnabled:    false,
	}
}
