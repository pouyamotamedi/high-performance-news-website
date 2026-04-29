package queue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	t.Run("CreateWorkerPool", func(t *testing.T) {
		pool := NewWorkerPool(4)
		if pool.GetWorkerCount() != 4 {
			t.Errorf("Expected 4 workers, got %d", pool.GetWorkerCount())
		}
		if pool.GetActiveJobs() != 0 {
			t.Errorf("Expected 0 active jobs, got %d", pool.GetActiveJobs())
		}
	})

	t.Run("StartAndStopWorkerPool", func(t *testing.T) {
		pool := NewWorkerPool(2)
		ctx := context.Background()

		err := pool.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start worker pool: %v", err)
		}

		// Give workers time to start
		time.Sleep(100 * time.Millisecond)

		err = pool.Stop()
		if err != nil {
			t.Fatalf("Failed to stop worker pool: %v", err)
		}
	})

	t.Run("SubmitJob", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx := context.Background()

		err := pool.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start worker pool: %v", err)
		}
		defer pool.Stop()

		handler := &TestJobHandler{}
		job := &Job{
			ID:   "worker-test",
			Type: JobTypeStaticGeneration,
		}

		err = pool.SubmitJob(ctx, job, handler)
		if err != nil {
			t.Fatalf("Failed to submit job: %v", err)
		}

		// Wait for job to be processed
		time.Sleep(200 * time.Millisecond)

		if len(handler.processedJobs) != 1 {
			t.Errorf("Expected 1 processed job, got %d", len(handler.processedJobs))
		}
	})

	t.Run("WorkerStats", func(t *testing.T) {
		pool := NewWorkerPool(3)
		ctx := context.Background()

		err := pool.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start worker pool: %v", err)
		}
		defer pool.Stop()

		stats := pool.GetWorkerStats()
		if len(stats) != 3 {
			t.Errorf("Expected 3 worker stats, got %d", len(stats))
		}

		for i, stat := range stats {
			if stat.ID != i {
				t.Errorf("Expected worker ID %d, got %d", i, stat.ID)
			}
			if stat.Active {
				t.Errorf("Expected worker %d to be inactive initially", i)
			}
		}
	})

	t.Run("GracefulShutdown", func(t *testing.T) {
		pool := NewWorkerPool(2)
		ctx := context.Background()

		err := pool.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start worker pool: %v", err)
		}

		// Submit a long-running job
		handler := &SlowJobHandler{duration: 500 * time.Millisecond}
		job := &Job{
			ID:   "slow-job",
			Type: JobTypeStaticGeneration,
		}

		err = pool.SubmitJob(ctx, job, handler)
		if err != nil {
			t.Fatalf("Failed to submit job: %v", err)
		}

		// Give job time to start
		time.Sleep(100 * time.Millisecond)

		// Stop should wait for job to complete
		start := time.Now()
		err = pool.Stop()
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to stop worker pool: %v", err)
		}

		// Should have waited for the job to complete (with some tolerance for timing)
		if duration < 350*time.Millisecond {
			t.Errorf("Expected graceful shutdown to wait for job completion, took %v", duration)
		}
	})

	t.Run("WorkerPoolFull", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx := context.Background()

		err := pool.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start worker pool: %v", err)
		}
		defer pool.Stop()

		handler := &SlowJobHandler{duration: 200 * time.Millisecond}

		// Fill the worker pool and job channel
		for i := 0; i < 10; i++ {
			job := &Job{
				ID:   fmt.Sprintf("job-%d", i),
				Type: JobTypeStaticGeneration,
			}
			err = pool.SubmitJob(ctx, job, handler)
			if err != nil && err != ErrWorkerPoolFull {
				t.Fatalf("Unexpected error submitting job %d: %v", i, err)
			}
		}

		// Additional job should fail
		job := &Job{
			ID:   "overflow-job",
			Type: JobTypeStaticGeneration,
		}
		err = pool.SubmitJob(ctx, job, handler)
		if err != ErrWorkerPoolFull {
			t.Errorf("Expected ErrWorkerPoolFull, got %v", err)
		}
	})
}

// SlowJobHandler simulates a slow job for testing
type SlowJobHandler struct {
	duration      time.Duration
	processedJobs []string
}

func (h *SlowJobHandler) Handle(ctx context.Context, job *Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(h.duration):
		h.processedJobs = append(h.processedJobs, job.ID)
		return nil
	}
}

func (h *SlowJobHandler) GetJobType() JobType {
	return JobTypeStaticGeneration
}