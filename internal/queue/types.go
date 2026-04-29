package queue

import (
	"context"
	"time"
)

// Priority levels for job processing
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

// JobType defines the type of job to be processed
type JobType string

const (
	JobTypeStaticGeneration JobType = "static_generation"
	JobTypeImageProcessing  JobType = "image_processing"
	JobTypeSearchIndexing   JobType = "search_indexing"
	JobTypeNotifications    JobType = "notifications"
)

// Job represents a unit of work to be processed
type Job struct {
	ID          string                 `json:"id"`
	Type        JobType                `json:"type"`
	Priority    Priority               `json:"priority"`
	Payload     map[string]interface{} `json:"payload"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	LastError   string                 `json:"last_error,omitempty"`
}

// JobHandler defines the interface for processing jobs
type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
	GetJobType() JobType
}

// JobQueue defines the interface for job queue operations
type JobQueue interface {
	Enqueue(job *Job) error
	Dequeue(priority Priority) (*Job, error)
	GetQueueSize(priority Priority) int
	GetMemoryUsage() (uint64, error)
	IsMemoryPressure() bool
	Start(ctx context.Context) error
	Stop() error
	RegisterHandler(handler JobHandler)
}

// MemoryMonitor defines the interface for memory monitoring
type MemoryMonitor interface {
	GetMemoryUsage() (uint64, error)
	IsMemoryPressure() bool
	GetMemoryThreshold() uint64
}

// WorkerPool defines the interface for managing workers
type WorkerPool interface {
	Start(ctx context.Context) error
	Stop() error
	GetWorkerCount() int
	GetActiveJobs() int
}