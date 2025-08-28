package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"high-performance-news-website/internal/queue"
)

func main() {
	fmt.Println("=== Memory-Aware Job Queue System Demo ===")

	// Create queue system with custom configuration
	config := &queue.QueueConfig{
		WorkerCount:        4,                            // 4 workers
		MemoryThreshold:    28 * 1024 * 1024 * 1024,     // 28GB threshold
		HighPrioritySize:   100,
		MediumPrioritySize: 500,
		LowPrioritySize:    1000,
	}

	system, err := queue.NewQueueSystem(config)
	if err != nil {
		log.Fatalf("Failed to create queue system: %v", err)
	}

	// Start the system
	ctx := context.Background()
	if err := system.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue system: %v", err)
	}
	defer system.Stop()

	fmt.Printf("Started queue system with %d workers\n", config.WorkerCount)
	fmt.Printf("Memory threshold: %s\n", queue.FormatBytes(config.MemoryThreshold))

	// Demonstrate different job types and priorities
	fmt.Println("\n=== Enqueueing Jobs ===")

	// 1. High priority static generation job
	err = system.EnqueueStaticGeneration(123, "homepage", queue.PriorityHigh)
	if err != nil {
		log.Printf("Failed to enqueue static generation: %v", err)
	} else {
		fmt.Println("✓ Enqueued high priority static generation job")
	}

	// 2. Medium priority image processing job
	formats := []string{"webp", "avif", "jpeg"}
	err = system.EnqueueImageProcessing("/uploads/2024/01/news-image.jpg", formats, queue.PriorityMedium)
	if err != nil {
		log.Printf("Failed to enqueue image processing: %v", err)
	} else {
		fmt.Println("✓ Enqueued medium priority image processing job")
	}

	// 3. Medium priority search indexing job
	err = system.EnqueueSearchIndexing(123, "index", queue.PriorityMedium)
	if err != nil {
		log.Printf("Failed to enqueue search indexing: %v", err)
	} else {
		fmt.Println("✓ Enqueued medium priority search indexing job")
	}

	// 4. Low priority notification job
	recipients := []interface{}{"editor@news.com", "admin@news.com"}
	err = system.EnqueueNotification("email", recipients, "New article published: Breaking News!", queue.PriorityLow)
	if err != nil {
		log.Printf("Failed to enqueue notification: %v", err)
	} else {
		fmt.Println("✓ Enqueued low priority notification job")
	}

	// Wait for jobs to be processed
	fmt.Println("\n=== Processing Jobs ===")
	time.Sleep(8 * time.Second)

	// Get system statistics
	fmt.Println("\n=== System Statistics ===")
	stats, err := system.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("System Health: %v\n", system.IsHealthy())
		
		if queueStats, ok := stats["queue"].(map[string]interface{}); ok {
			if queues, ok := queueStats["queues"].(map[string]int); ok {
				fmt.Printf("Queue Sizes - High: %d, Medium: %d, Low: %d\n", 
					queues["high"], queues["medium"], queues["low"])
			}
			if handlers, ok := queueStats["handlers"].(int); ok {
				fmt.Printf("Registered Handlers: %d\n", handlers)
			}
		}

		if memStats, ok := stats["memory"]; ok {
			if memData, ok := memStats.(*queue.MemoryStats); ok {
				fmt.Printf("Memory Usage: %s (Threshold: %s)\n", 
					queue.FormatBytes(memData.HeapInuse), 
					queue.FormatBytes(memData.Threshold))
				fmt.Printf("Memory Pressure: %v\n", memData.IsPressure)
			}
		}
	}

	// Demonstrate memory pressure handling
	fmt.Println("\n=== Memory Pressure Demo ===")
	
	// Create a system with very low memory threshold for demonstration
	lowMemConfig := &queue.QueueConfig{
		WorkerCount:     2,
		MemoryThreshold: 1024 * 1024, // 1MB threshold (very low for demo)
	}

	lowMemSystem, err := queue.NewQueueSystem(lowMemConfig)
	if err != nil {
		log.Printf("Failed to create low memory system: %v", err)
		return
	}

	if err := lowMemSystem.Start(ctx); err != nil {
		log.Printf("Failed to start low memory system: %v", err)
		return
	}
	defer lowMemSystem.Stop()

	// Try to enqueue high priority job (should succeed)
	err = lowMemSystem.EnqueueStaticGeneration(1, "article", queue.PriorityHigh)
	if err != nil {
		fmt.Printf("✗ High priority job rejected: %v\n", err)
	} else {
		fmt.Println("✓ High priority job accepted despite memory pressure")
	}

	// Try to enqueue low priority job (should be rejected)
	err = lowMemSystem.EnqueueNotification("email", []interface{}{"test@example.com"}, "Test", queue.PriorityLow)
	if err != nil {
		fmt.Printf("✓ Low priority job correctly rejected: %v\n", err)
	} else {
		fmt.Println("✗ Low priority job unexpectedly accepted")
	}

	fmt.Println("\n=== Demo Complete ===")
}