package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryJobQueue implements JobQueue interface with priority queues
type InMemoryJobQueue struct {
	queues        map[Priority]chan *Job
	handlers      map[JobType]JobHandler
	memoryMonitor MemoryMonitor
	workerPool    WorkerPool
	mu            sync.RWMutex
	running       bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewInMemoryJobQueue creates a new in-memory job queue
func NewInMemoryJobQueue(memoryMonitor MemoryMonitor, workerPool WorkerPool) *InMemoryJobQueue {
	return &InMemoryJobQueue{
		queues: map[Priority]chan *Job{
			PriorityHigh:   make(chan *Job, 1000),
			PriorityMedium: make(chan *Job, 5000),
			PriorityLow:    make(chan *Job, 10000),
		},
		handlers:      make(map[JobType]JobHandler),
		memoryMonitor: memoryMonitor,
		workerPool:    workerPool,
		stopCh:        make(chan struct{}),
	}
}

// Enqueue adds a job to the appropriate priority queue
func (q *InMemoryJobQueue) Enqueue(job *Job) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.running {
		return fmt.Errorf("queue is not running")
	}

	// Check memory pressure before enqueuing
	if q.memoryMonitor.IsMemoryPressure() {
		// Only allow high priority jobs during memory pressure
		if job.Priority != PriorityHigh {
			return fmt.Errorf("memory pressure detected, only high priority jobs allowed")
		}
	}

	// Set job ID if not provided
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	// Set creation time if not provided
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}

	// Set scheduled time if not provided
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}

	// Set default max attempts if not provided
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	queue, exists := q.queues[job.Priority]
	if !exists {
		return fmt.Errorf("invalid priority: %d", job.Priority)
	}

	select {
	case queue <- job:
		log.Printf("Job %s enqueued with priority %d", job.ID, job.Priority)
		return nil
	default:
		return fmt.Errorf("queue is full for priority %d", job.Priority)
	}
}

// Dequeue removes and returns a job from the specified priority queue
func (q *InMemoryJobQueue) Dequeue(priority Priority) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.running {
		return nil, fmt.Errorf("queue is not running")
	}

	queue, exists := q.queues[priority]
	if !exists {
		return nil, fmt.Errorf("invalid priority: %d", priority)
	}

	select {
	case job := <-queue:
		return job, nil
	default:
		return nil, fmt.Errorf("no jobs available in priority %d queue", priority)
	}
}

// GetQueueSize returns the number of jobs in the specified priority queue
func (q *InMemoryJobQueue) GetQueueSize(priority Priority) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queue, exists := q.queues[priority]
	if !exists {
		return 0
	}

	return len(queue)
}

// GetMemoryUsage returns current memory usage
func (q *InMemoryJobQueue) GetMemoryUsage() (uint64, error) {
	return q.memoryMonitor.GetMemoryUsage()
}

// IsMemoryPressure checks if system is under memory pressure
func (q *InMemoryJobQueue) IsMemoryPressure() bool {
	return q.memoryMonitor.IsMemoryPressure()
}

// RegisterHandler registers a job handler for a specific job type
func (q *InMemoryJobQueue) RegisterHandler(handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.handlers[handler.GetJobType()] = handler
	log.Printf("Registered handler for job type: %s", handler.GetJobType())
}

// Start starts the job queue processing
func (q *InMemoryJobQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.running {
		return fmt.Errorf("queue is already running")
	}

	q.running = true
	log.Println("Starting job queue...")

	// Start worker pool
	if err := q.workerPool.Start(ctx); err != nil {
		q.running = false
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start job dispatcher
	q.wg.Add(1)
	go q.dispatcher(ctx)

	// Start memory monitor
	q.wg.Add(1)
	go q.memoryMonitorLoop(ctx)

	log.Println("Job queue started successfully")
	return nil
}

// Stop stops the job queue processing
func (q *InMemoryJobQueue) Stop() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.running {
		return fmt.Errorf("queue is not running")
	}

	log.Println("Stopping job queue...")
	q.running = false

	// Signal stop
	close(q.stopCh)

	// Stop worker pool
	if err := q.workerPool.Stop(); err != nil {
		log.Printf("Error stopping worker pool: %v", err)
	}

	// Wait for goroutines to finish
	q.wg.Wait()

	log.Println("Job queue stopped successfully")
	return nil
}

// dispatcher handles job distribution to workers based on priority
func (q *InMemoryJobQueue) dispatcher(ctx context.Context) {
	defer q.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopCh:
			return
		case <-ticker.C:
			q.processJobs(ctx)
		}
	}
}

// processJobs processes jobs from queues in priority order
func (q *InMemoryJobQueue) processJobs(ctx context.Context) {
	// Process high priority jobs first
	priorities := []Priority{PriorityHigh, PriorityMedium, PriorityLow}

	for _, priority := range priorities {
		// During memory pressure, only process high priority jobs
		if q.memoryMonitor.IsMemoryPressure() && priority != PriorityHigh {
			continue
		}

		job, err := q.Dequeue(priority)
		if err != nil {
			continue // No jobs in this priority queue
		}

		// Check if job is scheduled for future execution
		if job.ScheduledAt.After(time.Now()) {
			// Re-enqueue for later processing
			if err := q.Enqueue(job); err != nil {
				log.Printf("Failed to re-enqueue scheduled job %s: %v", job.ID, err)
			}
			continue
		}

		// Find handler for job type
		q.mu.RLock()
		handler, exists := q.handlers[job.Type]
		q.mu.RUnlock()

		if !exists {
			log.Printf("No handler found for job type: %s", job.Type)
			continue
		}

		// Process job asynchronously
		go q.processJob(ctx, job, handler)
	}
}

// processJob processes a single job with error handling and retry logic
func (q *InMemoryJobQueue) processJob(ctx context.Context, job *Job, handler JobHandler) {
	log.Printf("Processing job %s of type %s (attempt %d/%d)", job.ID, job.Type, job.Attempts+1, job.MaxAttempts)

	job.Attempts++

	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// Process the job
	err := handler.Handle(jobCtx, job)
	if err != nil {
		job.LastError = err.Error()
		log.Printf("Job %s failed (attempt %d/%d): %v", job.ID, job.Attempts, job.MaxAttempts, err)

		// Retry if attempts remaining
		if job.Attempts < job.MaxAttempts {
			// Exponential backoff for retry
			delay := time.Duration(job.Attempts*job.Attempts) * time.Second
			job.ScheduledAt = time.Now().Add(delay)

			if err := q.Enqueue(job); err != nil {
				log.Printf("Failed to re-enqueue job %s for retry: %v", job.ID, err)
			} else {
				log.Printf("Job %s scheduled for retry in %v", job.ID, delay)
			}
		} else {
			log.Printf("Job %s failed permanently after %d attempts", job.ID, job.Attempts)
		}
	} else {
		log.Printf("Job %s completed successfully", job.ID)
	}
}

// memoryMonitorLoop monitors memory usage and logs warnings
func (q *InMemoryJobQueue) memoryMonitorLoop(ctx context.Context) {
	defer q.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopCh:
			return
		case <-ticker.C:
			if q.memoryMonitor.IsMemoryPressure() {
				usage, _ := q.memoryMonitor.GetMemoryUsage()
				threshold := q.memoryMonitor.GetMemoryThreshold()
				log.Printf("MEMORY PRESSURE WARNING: Usage %s exceeds threshold %s", 
					FormatBytes(usage), FormatBytes(threshold))
			}
		}
	}
}

// GetQueueStats returns statistics about all queues
func (q *InMemoryJobQueue) GetQueueStats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["running"] = q.running
	stats["queues"] = map[string]int{
		"high":   q.GetQueueSize(PriorityHigh),
		"medium": q.GetQueueSize(PriorityMedium),
		"low":    q.GetQueueSize(PriorityLow),
	}

	if memStats, err := q.memoryMonitor.(*MemoryMonitorImpl).GetMemoryStats(); err == nil {
		stats["memory"] = memStats
	}

	stats["handlers"] = len(q.handlers)
	stats["worker_count"] = q.workerPool.GetWorkerCount()
	stats["active_jobs"] = q.workerPool.GetActiveJobs()

	return stats
}