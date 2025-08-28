package queue

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryJobQueue(t *testing.T) {
	memMonitor := NewMemoryMonitor()
	workerPool := NewWorkerPool(2)
	queue := NewInMemoryJobQueue(memMonitor, workerPool)

	// Register test handler
	handler := &TestJobHandler{}
	queue.RegisterHandler(handler)

	ctx := context.Background()

	t.Run("StartAndStop", func(t *testing.T) {
		err := queue.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start queue: %v", err)
		}

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		err = queue.Stop()
		if err != nil {
			t.Fatalf("Failed to stop queue: %v", err)
		}
	})

	t.Run("EnqueueAndDequeue", func(t *testing.T) {
		err := queue.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start queue: %v", err)
		}
		defer queue.Stop()

		job := &Job{
			ID:       "test-job-1",
			Type:     JobTypeStaticGeneration,
			Priority: PriorityHigh,
			Payload:  map[string]interface{}{"test": "data"},
		}

		// Enqueue job
		err = queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}

		// Check queue size
		size := queue.GetQueueSize(PriorityHigh)
		if size != 1 {
			t.Errorf("Expected queue size 1, got %d", size)
		}

		// Dequeue job
		dequeuedJob, err := queue.Dequeue(PriorityHigh)
		if err != nil {
			t.Fatalf("Failed to dequeue job: %v", err)
		}

		if dequeuedJob.ID != job.ID {
			t.Errorf("Expected job ID %s, got %s", job.ID, dequeuedJob.ID)
		}
	})

	t.Run("PriorityQueues", func(t *testing.T) {
		err := queue.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start queue: %v", err)
		}
		defer queue.Stop()

		// Enqueue jobs with different priorities
		jobs := []*Job{
			{ID: "low", Type: JobTypeStaticGeneration, Priority: PriorityLow},
			{ID: "high", Type: JobTypeStaticGeneration, Priority: PriorityHigh},
			{ID: "medium", Type: JobTypeStaticGeneration, Priority: PriorityMedium},
		}

		for _, job := range jobs {
			err := queue.Enqueue(job)
			if err != nil {
				t.Fatalf("Failed to enqueue job %s: %v", job.ID, err)
			}
		}

		// Check queue sizes
		if queue.GetQueueSize(PriorityHigh) != 1 {
			t.Error("Expected 1 high priority job")
		}
		if queue.GetQueueSize(PriorityMedium) != 1 {
			t.Error("Expected 1 medium priority job")
		}
		if queue.GetQueueSize(PriorityLow) != 1 {
			t.Error("Expected 1 low priority job")
		}
	})

	t.Run("MemoryPressureBlocking", func(t *testing.T) {
		// Create a monitor with very low threshold
		lowMemMonitor := NewMemoryMonitor()
		lowMemMonitor.SetMemoryThreshold(1) // 1 byte - always under pressure
		
		testQueue := NewInMemoryJobQueue(lowMemMonitor, workerPool)
		testQueue.RegisterHandler(handler)

		err := testQueue.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start queue: %v", err)
		}
		defer testQueue.Stop()

		// High priority job should be allowed
		highPriorityJob := &Job{
			ID:       "high-priority",
			Type:     JobTypeStaticGeneration,
			Priority: PriorityHigh,
		}
		err = testQueue.Enqueue(highPriorityJob)
		if err != nil {
			t.Errorf("High priority job should be allowed during memory pressure: %v", err)
		}

		// Low priority job should be blocked
		lowPriorityJob := &Job{
			ID:       "low-priority",
			Type:     JobTypeStaticGeneration,
			Priority: PriorityLow,
		}
		err = testQueue.Enqueue(lowPriorityJob)
		if err == nil {
			t.Error("Low priority job should be blocked during memory pressure")
		}
	})
}

// TestJobHandler is a simple test implementation of JobHandler
type TestJobHandler struct {
	processedJobs []string
}

func (h *TestJobHandler) Handle(ctx context.Context, job *Job) error {
	h.processedJobs = append(h.processedJobs, job.ID)
	return nil
}

func (h *TestJobHandler) GetJobType() JobType {
	return JobTypeStaticGeneration
}

func TestJobQueueIntegration(t *testing.T) {
	memMonitor := NewMemoryMonitor()
	workerPool := NewWorkerPool(1)
	queue := NewInMemoryJobQueue(memMonitor, workerPool)

	handler := &TestJobHandler{}
	queue.RegisterHandler(handler)

	ctx := context.Background()
	err := queue.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	// Enqueue a job
	job := &Job{
		ID:       "integration-test",
		Type:     JobTypeStaticGeneration,
		Priority: PriorityHigh,
		Payload:  map[string]interface{}{"test": "integration"},
	}

	err = queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Check if job was processed
	if len(handler.processedJobs) != 1 {
		t.Errorf("Expected 1 processed job, got %d", len(handler.processedJobs))
	}
	if len(handler.processedJobs) > 0 && handler.processedJobs[0] != job.ID {
		t.Errorf("Expected processed job ID %s, got %s", job.ID, handler.processedJobs[0])
	}
}