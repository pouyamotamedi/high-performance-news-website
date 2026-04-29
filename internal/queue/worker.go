package queue

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPoolImpl implements WorkerPool interface
type WorkerPoolImpl struct {
	workerCount int
	activeJobs  int64
	workers     []*Worker
	jobCh       chan *JobTask
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// Worker represents a single worker goroutine
type Worker struct {
	id       int
	pool     *WorkerPoolImpl
	jobCh    chan *JobTask
	stopCh   chan struct{}
	wg       *sync.WaitGroup
	active   bool
	lastJob  time.Time
}

// JobTask represents a job with its handler
type JobTask struct {
	Job     *Job
	Handler JobHandler
	Ctx     context.Context
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount int) *WorkerPoolImpl {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	return &WorkerPoolImpl{
		workerCount: workerCount,
		jobCh:       make(chan *JobTask, workerCount*2),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the worker pool
func (wp *WorkerPoolImpl) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return nil
	}

	wp.running = true
	wp.workers = make([]*Worker, wp.workerCount)

	log.Printf("Starting worker pool with %d workers", wp.workerCount)

	// Start workers
	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			id:     i,
			pool:   wp,
			jobCh:  wp.jobCh,
			stopCh: wp.stopCh,
			wg:     &wp.wg,
		}
		wp.workers[i] = worker

		wp.wg.Add(1)
		go worker.start(ctx)
	}

	log.Printf("Worker pool started with %d workers", wp.workerCount)
	return nil
}

// Stop stops the worker pool gracefully
func (wp *WorkerPoolImpl) Stop() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return nil
	}

	log.Println("Stopping worker pool...")
	wp.running = false

	// Signal all workers to stop
	close(wp.stopCh)

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Worker pool stop timeout, some workers may still be running")
	}

	// Close job channel
	close(wp.jobCh)

	log.Println("Worker pool stopped")
	return nil
}

// GetWorkerCount returns the number of workers
func (wp *WorkerPoolImpl) GetWorkerCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.workerCount
}

// GetActiveJobs returns the number of currently active jobs
func (wp *WorkerPoolImpl) GetActiveJobs() int {
	return int(atomic.LoadInt64(&wp.activeJobs))
}

// SubmitJob submits a job to the worker pool
func (wp *WorkerPoolImpl) SubmitJob(ctx context.Context, job *Job, handler JobHandler) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.running {
		return ErrWorkerPoolStopped
	}

	task := &JobTask{
		Job:     job,
		Handler: handler,
		Ctx:     ctx,
	}

	select {
	case wp.jobCh <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrWorkerPoolFull
	}
}

// GetWorkerStats returns statistics about workers
func (wp *WorkerPoolImpl) GetWorkerStats() []WorkerStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	stats := make([]WorkerStats, len(wp.workers))
	for i, worker := range wp.workers {
		stats[i] = WorkerStats{
			ID:      worker.id,
			Active:  worker.active,
			LastJob: worker.lastJob,
		}
	}
	return stats
}

// WorkerStats contains statistics about a worker
type WorkerStats struct {
	ID      int       `json:"id"`
	Active  bool      `json:"active"`
	LastJob time.Time `json:"last_job"`
}

// start starts the worker goroutine
func (w *Worker) start(ctx context.Context) {
	defer w.wg.Done()
	log.Printf("Worker %d started", w.id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopped due to context cancellation", w.id)
			return
		case <-w.stopCh:
			log.Printf("Worker %d stopped", w.id)
			return
		case task := <-w.jobCh:
			if task == nil {
				log.Printf("Worker %d received nil task, stopping", w.id)
				return
			}
			w.processJob(task)
		}
	}
}

// processJob processes a single job
func (w *Worker) processJob(task *JobTask) {
	w.active = true
	w.lastJob = time.Now()
	atomic.AddInt64(&w.pool.activeJobs, 1)

	defer func() {
		w.active = false
		atomic.AddInt64(&w.pool.activeJobs, -1)

		// Recover from panics
		if r := recover(); r != nil {
			log.Printf("Worker %d recovered from panic while processing job %s: %v", 
				w.id, task.Job.ID, r)
		}
	}()

	log.Printf("Worker %d processing job %s of type %s", w.id, task.Job.ID, task.Job.Type)

	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(task.Ctx, 30*time.Minute)
	defer cancel()

	// Process the job
	err := task.Handler.Handle(jobCtx, task.Job)
	if err != nil {
		log.Printf("Worker %d: Job %s failed: %v", w.id, task.Job.ID, err)
	} else {
		log.Printf("Worker %d: Job %s completed successfully", w.id, task.Job.ID)
	}
}

// Custom errors
var (
	ErrWorkerPoolStopped = fmt.Errorf("worker pool is stopped")
	ErrWorkerPoolFull    = fmt.Errorf("worker pool is full")
)