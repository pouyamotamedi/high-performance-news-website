package queue

import (
	"context"
	"fmt"
	"runtime"
)

// QueueConfig holds configuration for the job queue system
type QueueConfig struct {
	WorkerCount      int    `json:"worker_count"`
	MemoryThreshold  uint64 `json:"memory_threshold"`  // in bytes
	HighPrioritySize int    `json:"high_priority_size"`
	MediumPrioritySize int  `json:"medium_priority_size"`
	LowPrioritySize    int  `json:"low_priority_size"`
}

// DefaultQueueConfig returns default configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		WorkerCount:        runtime.NumCPU(),
		MemoryThreshold:    28 * 1024 * 1024 * 1024, // 28GB
		HighPrioritySize:   1000,
		MediumPrioritySize: 5000,
		LowPrioritySize:    10000,
	}
}

// QueueSystem represents the complete job queue system
type QueueSystem struct {
	Queue         JobQueue
	WorkerPool    WorkerPool
	MemoryMonitor MemoryMonitor
	Registry      *JobHandlerRegistry
	Config        *QueueConfig
}

// NewQueueSystem creates a new complete job queue system
func NewQueueSystem(config *QueueConfig) (*QueueSystem, error) {
	if config == nil {
		config = DefaultQueueConfig()
	}

	// Create memory monitor
	memoryMonitor := NewMemoryMonitor()
	if config.MemoryThreshold > 0 {
		memoryMonitor.SetMemoryThreshold(config.MemoryThreshold)
	}

	// Create worker pool
	workerPool := NewWorkerPool(config.WorkerCount)

	// Create job queue
	queue := NewInMemoryJobQueue(memoryMonitor, workerPool)

	// Create handler registry and register default handlers
	registry := NewJobHandlerRegistry()
	registry.RegisterDefaultHandlers()

	// Register handlers with queue
	for _, handler := range registry.GetAll() {
		queue.RegisterHandler(handler)
	}

	return &QueueSystem{
		Queue:         queue,
		WorkerPool:    workerPool,
		MemoryMonitor: memoryMonitor,
		Registry:      registry,
		Config:        config,
	}, nil
}

// Start starts the entire queue system
func (qs *QueueSystem) Start(ctx context.Context) error {
	return qs.Queue.Start(ctx)
}

// Stop stops the entire queue system
func (qs *QueueSystem) Stop() error {
	return qs.Queue.Stop()
}

// EnqueueStaticGeneration enqueues a static generation job
func (qs *QueueSystem) EnqueueStaticGeneration(articleID interface{}, pageType string, priority Priority) error {
	job := &Job{
		Type:     JobTypeStaticGeneration,
		Priority: priority,
		Payload: map[string]interface{}{
			"article_id": articleID,
			"page_type":  pageType,
		},
		MaxAttempts: 3,
	}
	return qs.Queue.Enqueue(job)
}

// EnqueueImageProcessing enqueues an image processing job
func (qs *QueueSystem) EnqueueImageProcessing(imagePath string, formats []string, priority Priority) error {
	// Convert formats to interface slice
	formatInterfaces := make([]interface{}, len(formats))
	for i, format := range formats {
		formatInterfaces[i] = format
	}

	job := &Job{
		Type:     JobTypeImageProcessing,
		Priority: priority,
		Payload: map[string]interface{}{
			"image_path": imagePath,
			"formats":    formatInterfaces,
		},
		MaxAttempts: 2, // Image processing failures are less critical
	}
	return qs.Queue.Enqueue(job)
}

// EnqueueSearchIndexing enqueues a search indexing job
func (qs *QueueSystem) EnqueueSearchIndexing(documentID interface{}, action string, priority Priority) error {
	job := &Job{
		Type:     JobTypeSearchIndexing,
		Priority: priority,
		Payload: map[string]interface{}{
			"document_id": documentID,
			"action":      action,
		},
		MaxAttempts: 5, // Search indexing is important for discoverability
	}
	return qs.Queue.Enqueue(job)
}

// EnqueueNotification enqueues a notification job
func (qs *QueueSystem) EnqueueNotification(notificationType string, recipients []interface{}, message string, priority Priority) error {
	job := &Job{
		Type:     JobTypeNotifications,
		Priority: priority,
		Payload: map[string]interface{}{
			"type":       notificationType,
			"recipients": recipients,
			"message":    message,
		},
		MaxAttempts: 3,
	}
	return qs.Queue.Enqueue(job)
}

// GetStats returns comprehensive statistics about the queue system
func (qs *QueueSystem) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Queue stats
	if inMemQueue, ok := qs.Queue.(*InMemoryJobQueue); ok {
		stats["queue"] = inMemQueue.GetQueueStats()
	}

	// Worker stats
	if workerPool, ok := qs.WorkerPool.(*WorkerPoolImpl); ok {
		stats["workers"] = workerPool.GetWorkerStats()
	}

	// Memory stats
	if memMonitor, ok := qs.MemoryMonitor.(*MemoryMonitorImpl); ok {
		memStats, err := memMonitor.GetMemoryStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get memory stats: %w", err)
		}
		stats["memory"] = memStats
	}

	// Configuration
	stats["config"] = qs.Config

	return stats, nil
}

// IsHealthy checks if the queue system is healthy
func (qs *QueueSystem) IsHealthy() bool {
	// Check if memory pressure is too high
	if qs.MemoryMonitor.IsMemoryPressure() {
		return false
	}

	// Check if queues are not completely full
	highQueueSize := qs.Queue.GetQueueSize(PriorityHigh)
	if highQueueSize >= 900 { // 90% of capacity
		return false
	}

	return true
}