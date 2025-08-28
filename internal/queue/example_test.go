package queue

import (
	"context"
	"fmt"
	"log"
	"time"
)

// DemoQueueSystem demonstrates how to use the job queue system
func DemoQueueSystem() {
	// Create a queue system with default configuration
	system, err := NewQueueSystem(nil)
	if err != nil {
		log.Fatalf("Failed to create queue system: %v", err)
	}

	// Start the system
	ctx := context.Background()
	if err := system.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue system: %v", err)
	}
	defer system.Stop()

	// Enqueue different types of jobs

	// 1. Static generation job (high priority)
	err = system.EnqueueStaticGeneration(123, "article", PriorityHigh)
	if err != nil {
		log.Printf("Failed to enqueue static generation: %v", err)
	}

	// 2. Image processing job (medium priority)
	formats := []string{"webp", "avif", "jpeg"}
	err = system.EnqueueImageProcessing("/uploads/2024/01/image.jpg", formats, PriorityMedium)
	if err != nil {
		log.Printf("Failed to enqueue image processing: %v", err)
	}

	// 3. Search indexing job (medium priority)
	err = system.EnqueueSearchIndexing(123, "index", PriorityMedium)
	if err != nil {
		log.Printf("Failed to enqueue search indexing: %v", err)
	}

	// 4. Notification job (low priority)
	recipients := []interface{}{"user1@example.com", "user2@example.com"}
	err = system.EnqueueNotification("email", recipients, "New article published!", PriorityLow)
	if err != nil {
		log.Printf("Failed to enqueue notification: %v", err)
	}

	// Wait for jobs to be processed
	time.Sleep(5 * time.Second)

	// Get system statistics
	stats, err := system.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Queue system stats: %+v\n", stats)
	}

	// Check system health
	if system.IsHealthy() {
		fmt.Println("Queue system is healthy")
	} else {
		fmt.Println("Queue system is experiencing issues")
	}

	// Output: Queue system is healthy
}

// DemoCustomConfiguration demonstrates custom configuration
func DemoCustomConfiguration() {
	// Create custom configuration
	config := &QueueConfig{
		WorkerCount:        8,                            // 8 workers
		MemoryThreshold:    16 * 1024 * 1024 * 1024,     // 16GB threshold
		HighPrioritySize:   500,                          // Smaller high priority queue
		MediumPrioritySize: 2000,                         // Medium priority queue
		LowPrioritySize:    5000,                         // Large low priority queue
	}

	system, err := NewQueueSystem(config)
	if err != nil {
		log.Fatalf("Failed to create queue system: %v", err)
	}

	ctx := context.Background()
	if err := system.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue system: %v", err)
	}
	defer system.Stop()

	fmt.Printf("Started queue system with %d workers\n", config.WorkerCount)
	fmt.Printf("Memory threshold: %s\n", FormatBytes(config.MemoryThreshold))

	// Output: Started queue system with 8 workers
	// Memory threshold: 16.0 GB
}

// DemoMemoryPressureHandling demonstrates memory pressure handling
func DemoMemoryPressureHandling() {
	// Create system with low memory threshold for demonstration
	config := &QueueConfig{
		WorkerCount:     4,
		MemoryThreshold: 1024 * 1024, // 1MB threshold (very low for demo)
	}

	system, err := NewQueueSystem(config)
	if err != nil {
		log.Fatalf("Failed to create queue system: %v", err)
	}

	ctx := context.Background()
	if err := system.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue system: %v", err)
	}
	defer system.Stop()

	// Try to enqueue high priority job (should succeed even under memory pressure)
	err = system.EnqueueStaticGeneration(1, "article", PriorityHigh)
	if err != nil {
		fmt.Printf("High priority job rejected: %v\n", err)
	} else {
		fmt.Println("High priority job accepted")
	}

	// Try to enqueue low priority job (might be rejected under memory pressure)
	err = system.EnqueueNotification("email", []interface{}{"test@example.com"}, "Test", PriorityLow)
	if err != nil {
		fmt.Printf("Low priority job rejected: %v\n", err)
	} else {
		fmt.Println("Low priority job accepted")
	}

	// Output: High priority job accepted
	// Low priority job rejected: memory pressure detected, only high priority jobs allowed
}

// DemoJobRetryLogic demonstrates job retry behavior
func DemoJobRetryLogic() {
	system, err := NewQueueSystem(nil)
	if err != nil {
		log.Fatalf("Failed to create queue system: %v", err)
	}

	// Register a custom handler that fails initially
	failingHandler := &FailingJobHandler{failCount: 2}
	system.Registry.Register(failingHandler)
	system.Queue.RegisterHandler(failingHandler)

	ctx := context.Background()
	if err := system.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue system: %v", err)
	}
	defer system.Stop()

	// Enqueue a job that will fail initially but succeed on retry
	job := &Job{
		ID:          "retry-test",
		Type:        JobType("failing"),
		Priority:    PriorityHigh,
		MaxAttempts: 3,
		Payload:     map[string]interface{}{},
	}

	err = system.Queue.Enqueue(job)
	if err != nil {
		log.Printf("Failed to enqueue job: %v", err)
	}

	// Wait for job processing and retries
	time.Sleep(10 * time.Second)

	fmt.Printf("Job processed successfully after %d attempts\n", failingHandler.attempts)

	// Output: Job processed successfully after 3 attempts
}

// FailingJobHandler is a test handler that fails a specified number of times
type FailingJobHandler struct {
	failCount int
	attempts  int
}

func (h *FailingJobHandler) Handle(ctx context.Context, job *Job) error {
	h.attempts++
	if h.attempts <= h.failCount {
		return fmt.Errorf("simulated failure (attempt %d)", h.attempts)
	}
	return nil
}

func (h *FailingJobHandler) GetJobType() JobType {
	return JobType("failing")
}