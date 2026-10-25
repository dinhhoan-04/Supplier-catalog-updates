package domain

import (
	"time"
)

type JobStatus string

const (
	JobStatusPending     JobStatus = "PENDING"
	JobStatusRunning     JobStatus = "RUNNING"
	JobStatusInterrupted JobStatus = "INTERRUPTED"
	JobStatusFailed      JobStatus = "FAILED"
	JobStatusCompleted   JobStatus = "COMPLETED"
)

type SyncJob struct {
	ID               string     `json:"id"`
	Status           JobStatus  `json:"status"`
	Scenario         string     `json:"scenario"`
	TotalBatches     int        `json:"total_batches"`
	ProcessedBatches int        `json:"processed_batches"`
	TotalItems       int        `json:"total_items"`
	CreatedCount     int        `json:"created_count"`
	UpdatedCount     int        `json:"updated_count"`
	WithdrawnCount   int        `json:"withdrawn_count"`
	SkippedCount     int        `json:"skipped_count"`
	ErrorCount       int        `json:"error_count"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	InterruptedAt    *time.Time `json:"interrupted_at,omitempty"`
	ErrorSummary     *string    `json:"error_summary,omitempty"`
}

type BatchCheckpoint struct {
	ID          int64     `json:"id"`
	JobID       string    `json:"job_id"`
	BatchIndex  int       `json:"batch_index"`
	ItemsCount  int       `json:"items_count"`
	ProcessedAt time.Time `json:"processed_at"`
	Status      string    `json:"status"`
}
