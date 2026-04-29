package queue

import (
	"context"
	"testing"
	"time"
)

func TestQueueSystem(t *testing.T) {
	t.Run("NewQueueSystemWithDefaultConfig", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		if system.Config.WorkerCount <= 0 {
			t.Error("Expected positive worker count")
		}
		if system.Config.MemoryThreshold == 0 {
			t.Error("Expected non-zero memory threshold")
		}
	})

	t.Run("NewQueueSystemWithCustomConfig", func(t *testing.T) {
		config := &QueueConfig{
			WorkerCount:     8,
			MemoryThreshold: 16 * 1024 * 1024 * 1024, // 16GB
		}

		system, err := NewQueueSystem(config)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		if system.Config.WorkerCount != 8 {
			t.Errorf("Expected worker count 8, got %d", system.Config.WorkerCount)
		}
	})

	t.Run("StartAndStopSystem", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		err = system.Stop()
		if err != nil {
			t.Fatalf("Failed to stop system: %v", err)
		}
	})

	t.Run("EnqueueStaticGeneration", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		err = system.EnqueueStaticGeneration(123, "article", PriorityHigh)
		if err != nil {
			t.Errorf("Failed to enqueue static generation job: %v", err)
		}

		// Check queue size
		size := system.Queue.GetQueueSize(PriorityHigh)
		if size != 1 {
			t.Errorf("Expected queue size 1, got %d", size)
		}
	})

	t.Run("EnqueueImageProcessing", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		formats := []string{"webp", "avif", "jpeg"}
		err = system.EnqueueImageProcessing("/path/to/image.jpg", formats, PriorityMedium)
		if err != nil {
			t.Errorf("Failed to enqueue image processing job: %v", err)
		}

		size := system.Queue.GetQueueSize(PriorityMedium)
		if size != 1 {
			t.Errorf("Expected queue size 1, got %d", size)
		}
	})

	t.Run("EnqueueSearchIndexing", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		err = system.EnqueueSearchIndexing(456, "index", PriorityMedium)
		if err != nil {
			t.Errorf("Failed to enqueue search indexing job: %v", err)
		}

		size := system.Queue.GetQueueSize(PriorityMedium)
		if size != 1 {
			t.Errorf("Expected queue size 1, got %d", size)
		}
	})

	t.Run("EnqueueNotification", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		recipients := []interface{}{"user1@example.com", "user2@example.com"}
		err = system.EnqueueNotification("email", recipients, "Test message", PriorityLow)
		if err != nil {
			t.Errorf("Failed to enqueue notification job: %v", err)
		}

		size := system.Queue.GetQueueSize(PriorityLow)
		if size != 1 {
			t.Errorf("Expected queue size 1, got %d", size)
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		stats, err := system.GetStats()
		if err != nil {
			t.Errorf("Failed to get stats: %v", err)
		}

		if stats == nil {
			t.Error("Expected non-nil stats")
		}

		// Check for expected keys
		expectedKeys := []string{"queue", "workers", "memory", "config"}
		for _, key := range expectedKeys {
			if _, exists := stats[key]; !exists {
				t.Errorf("Expected stats to contain key: %s", key)
			}
		}
	})

	t.Run("IsHealthy", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		// Should be healthy initially
		if !system.IsHealthy() {
			t.Error("Expected system to be healthy initially")
		}
	})

	t.Run("MemoryPressureHealth", func(t *testing.T) {
		config := &QueueConfig{
			WorkerCount:     2,
			MemoryThreshold: 1, // 1 byte - always under pressure
		}

		system, err := NewQueueSystem(config)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		// Should be unhealthy due to memory pressure
		if system.IsHealthy() {
			t.Error("Expected system to be unhealthy due to memory pressure")
		}
	})

	t.Run("FullQueueHealth", func(t *testing.T) {
		system, err := NewQueueSystem(nil)
		if err != nil {
			t.Fatalf("Failed to create queue system: %v", err)
		}

		ctx := context.Background()
		err = system.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer system.Stop()

		// Fill high priority queue to near capacity
		for i := 0; i < 950; i++ {
			err = system.EnqueueStaticGeneration(i, "article", PriorityHigh)
			if err != nil {
				break // Queue might be full
			}
		}

		// Should be unhealthy due to full queue
		if system.IsHealthy() {
			t.Error("Expected system to be unhealthy due to full queue")
		}
	})
}

func TestDefaultQueueConfig(t *testing.T) {
	config := DefaultQueueConfig()

	if config.WorkerCount <= 0 {
		t.Error("Expected positive worker count")
	}
	if config.MemoryThreshold == 0 {
		t.Error("Expected non-zero memory threshold")
	}
	if config.HighPrioritySize <= 0 {
		t.Error("Expected positive high priority queue size")
	}
	if config.MediumPrioritySize <= 0 {
		t.Error("Expected positive medium priority queue size")
	}
	if config.LowPrioritySize <= 0 {
		t.Error("Expected positive low priority queue size")
	}

	// Check that medium and low priority queues are larger than high priority
	if config.MediumPrioritySize <= config.HighPrioritySize {
		t.Error("Expected medium priority queue to be larger than high priority")
	}
	if config.LowPrioritySize <= config.MediumPrioritySize {
		t.Error("Expected low priority queue to be larger than medium priority")
	}
}

func TestQueueSystemIntegration(t *testing.T) {
	system, err := NewQueueSystem(nil)
	if err != nil {
		t.Fatalf("Failed to create queue system: %v", err)
	}

	ctx := context.Background()
	err = system.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start system: %v", err)
	}
	defer system.Stop()

	// Enqueue jobs of different types and priorities
	err = system.EnqueueStaticGeneration(1, "homepage", PriorityHigh)
	if err != nil {
		t.Errorf("Failed to enqueue static generation: %v", err)
	}

	err = system.EnqueueImageProcessing("/test.jpg", []string{"webp"}, PriorityMedium)
	if err != nil {
		t.Errorf("Failed to enqueue image processing: %v", err)
	}

	err = system.EnqueueSearchIndexing(1, "index", PriorityMedium)
	if err != nil {
		t.Errorf("Failed to enqueue search indexing: %v", err)
	}

	recipients := []interface{}{"test@example.com"}
	err = system.EnqueueNotification("push", recipients, "Test", PriorityLow)
	if err != nil {
		t.Errorf("Failed to enqueue notification: %v", err)
	}

	// Wait for jobs to be processed
	time.Sleep(1 * time.Second)

	// Check that queues are being processed
	stats, err := system.GetStats()
	if err != nil {
		t.Errorf("Failed to get stats: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}