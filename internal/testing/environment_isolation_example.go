package testing

import (
	"fmt"
	"log"
	"time"
)

// ExampleDockerEnvironmentIsolation demonstrates the complete Docker-based environment isolation system
func ExampleDockerEnvironmentIsolation() {
	// Create enhanced environment manager with all features
	manager, err := NewEnhancedEnvironmentManager()
	if err != nil {
		log.Fatalf("Failed to create enhanced environment manager: %v", err)
	}
	defer manager.Shutdown()

	// Add alert handlers
	manager.alertManager.AddAlertHandler(&LogAlertHandler{})
	manager.alertManager.AddAlertHandler(&EmailAlertHandler{
		SMTPServer: "smtp.example.com",
		From:       "alerts@example.com",
		To:         []string{"admin@example.com"},
	})

	// Create environment queue for managing concurrent requests
	queue := NewEnvironmentQueue(3) // Max 3 concurrent environments

	// Create cost tracker
	costTracker := NewCostTracker()

	fmt.Println("=== Docker-based Environment Isolation System Demo ===")

	// Example 1: Create isolated environment with automatic recovery
	fmt.Println("\n1. Creating isolated environment with recovery...")
	env1, err := manager.CreateIsolatedEnvironmentWithRecovery("integration-test-suite")
	if err != nil {
		log.Printf("Failed to create environment: %v", err)
		return
	}

	fmt.Printf("Environment created: ID=%s, TestSuite=%s, Status=%s\n", 
		env1.ID, env1.TestSuite, env1.Status)

	// Start cost tracking
	costTracker.StartTracking(env1.ID)

	// Example 2: Queue multiple environment requests with priorities
	fmt.Println("\n2. Queuing multiple environment requests...")
	
	requests := []EnvironmentRequest{
		{
			TestSuite: "high-priority-tests",
			Priority:  5,
			Resources: ResourceAllocation{Memory: 1024 * 1024 * 1024, CPUQuota: 100000},
			Timeout:   5 * time.Minute,
		},
		{
			TestSuite: "medium-priority-tests",
			Priority:  3,
			Resources: ResourceAllocation{Memory: 512 * 1024 * 1024, CPUQuota: 50000},
			Timeout:   3 * time.Minute,
		},
		{
			TestSuite: "low-priority-tests",
			Priority:  1,
			Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
			Timeout:   2 * time.Minute,
		},
	}

	requestIDs := make([]string, len(requests))
	for i, req := range requests {
		requestIDs[i] = queue.QueueEnvironmentRequest(req)
		fmt.Printf("Queued request: ID=%s, TestSuite=%s, Priority=%d\n", 
			requestIDs[i], req.TestSuite, req.Priority)
	}

	// Example 3: Monitor resource utilization
	fmt.Println("\n3. Monitoring resource utilization...")
	memUtil, cpuUtil, envUtil := manager.resourcePool.GetUtilization()
	fmt.Printf("Resource Utilization - Memory: %.1f%%, CPU: %.1f%%, Environments: %.1f%%\n",
		memUtil*100, cpuUtil*100, envUtil*100)

	// Example 4: Check performance metrics
	fmt.Println("\n4. Checking performance metrics...")
	time.Sleep(2 * time.Second) // Allow some time for metrics collection

	if metrics, err := manager.GetEnvironmentPerformanceMetrics(env1.ID); err == nil {
		fmt.Printf("Performance Metrics for %s:\n", env1.ID)
		fmt.Printf("  CPU Usage: %.2f%%\n", metrics.CPUUsage)
		fmt.Printf("  Memory Usage: %d MB / %d MB\n", 
			metrics.MemoryUsage/(1024*1024), metrics.MemoryLimit/(1024*1024))
		fmt.Printf("  Performance Score: %.1f/100\n", metrics.PerformanceScore)
		
		if len(metrics.Alerts) > 0 {
			fmt.Printf("  Active Alerts: %d\n", len(metrics.Alerts))
			for _, alert := range metrics.Alerts {
				fmt.Printf("    - %s: %s\n", alert.Type, alert.Message)
			}
		}
	}

	// Example 5: Update and check costs
	fmt.Println("\n5. Tracking environment costs...")
	costTracker.UpdateCost(env1.ID, 45.0, 512.0, 8.0, 3.0)
	
	if cost := costTracker.GetEnvironmentCost(env1.ID); cost != nil {
		fmt.Printf("Cost Breakdown for %s:\n", env1.ID)
		fmt.Printf("  CPU Cost: $%.4f\n", cost.CPUCost)
		fmt.Printf("  Memory Cost: $%.4f\n", cost.MemoryCost)
		fmt.Printf("  Storage Cost: $%.4f\n", cost.StorageCost)
		fmt.Printf("  Network Cost: $%.4f\n", cost.NetworkCost)
		fmt.Printf("  Total Cost: $%.4f\n", cost.TotalCost)
		fmt.Printf("  Duration: %v\n", cost.Duration)
	}

	// Example 6: Check queue status
	fmt.Println("\n6. Checking queue status...")
	for i, requestID := range requestIDs {
		req, result, err := queue.GetRequestStatus(requestID)
		if err != nil {
			fmt.Printf("Request %d: Error - %v\n", i+1, err)
			continue
		}

		if req != nil {
			fmt.Printf("Request %d: Status=%s, TestSuite=%s\n", 
				i+1, req.Status, req.TestSuite)
		}

		if result != nil {
			fmt.Printf("Request %d: Completed in %v\n", i+1, result.Duration)
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
		}
	}

	// Example 7: Demonstrate resource optimization
	fmt.Println("\n7. Running resource optimization...")
	manager.resourceOptimizer.optimizeResources()

	// Example 8: List all environments
	fmt.Println("\n8. Listing all environments...")
	environments := manager.ListEnvironments()
	fmt.Printf("Total environments: %d\n", len(environments))
	for i, env := range environments {
		fmt.Printf("  %d. ID=%s, TestSuite=%s, Status=%s, Health=%s\n",
			i+1, env.ID, env.TestSuite, env.Status, env.HealthStatus)
	}

	// Example 9: Trigger a test alert
	fmt.Println("\n9. Triggering test alert...")
	testAlert := EnvironmentAlert{
		ID:            "demo-alert-1",
		EnvironmentID: env1.ID,
		RuleName:      "high_memory_usage",
		Severity:      "warning",
		Message:       "Demo alert: Memory usage simulation exceeded threshold",
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"threshold": 85.0,
			"current":   92.5,
		},
	}
	manager.alertManager.TriggerAlert(testAlert)

	// Example 10: Final cost summary
	fmt.Println("\n10. Final cost summary...")
	finalCost := costTracker.StopTracking(env1.ID)
	if finalCost != nil {
		fmt.Printf("Final cost for %s: $%.4f over %v\n", 
			env1.ID, finalCost.TotalCost, finalCost.Duration)
	}
	fmt.Printf("Total system cost: $%.4f\n", costTracker.GetTotalCost())

	// Cleanup
	fmt.Println("\n11. Cleaning up environments...")
	for _, env := range manager.ListEnvironments() {
		if err := manager.CleanupEnvironment(env.ID); err != nil {
			log.Printf("Failed to cleanup environment %s: %v", env.ID, err)
		} else {
			fmt.Printf("Cleaned up environment: %s\n", env.ID)
		}
	}

	fmt.Println("\n=== Demo completed successfully ===")
}

// ExampleResourceOptimization demonstrates resource optimization features
func ExampleResourceOptimization() {
	manager, err := NewTestEnvironmentManager()
	if err != nil {
		log.Fatalf("Failed to create environment manager: %v", err)
	}
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)

	fmt.Println("=== Resource Optimization Demo ===")

	// Show initial resource state
	memUtil, cpuUtil, envUtil := manager.resourcePool.GetUtilization()
	fmt.Printf("Initial utilization - Memory: %.1f%%, CPU: %.1f%%, Environments: %.1f%%\n",
		memUtil*100, cpuUtil*100, envUtil*100)

	// Create several environments to increase utilization
	fmt.Println("\nCreating environments to increase utilization...")
	environments := make([]*IsolatedEnvironment, 3)
	for i := 0; i < 3; i++ {
		env, err := manager.CreateIsolatedEnvironment(fmt.Sprintf("optimization-test-%d", i+1))
		if err != nil {
			log.Printf("Failed to create environment %d: %v", i+1, err)
			continue
		}
		environments[i] = env
		fmt.Printf("Created environment %d: %s\n", i+1, env.ID)
	}

	// Check utilization after creating environments
	memUtil, cpuUtil, envUtil = manager.resourcePool.GetUtilization()
	fmt.Printf("\nUtilization after creating environments - Memory: %.1f%%, CPU: %.1f%%, Environments: %.1f%%\n",
		memUtil*100, cpuUtil*100, envUtil*100)

	// Run optimization
	fmt.Println("\nRunning resource optimization...")
	optimizer.optimizeResources()

	// Show optimization rules
	fmt.Println("\nOptimization rules:")
	for i, rule := range optimizer.optimizationRules {
		status := "disabled"
		if rule.Enabled {
			status = "enabled"
		}
		fmt.Printf("  %d. %s (%s) - Priority: %d, Status: %s\n",
			i+1, rule.Name, rule.Action, rule.Priority, status)
	}

	// Show scale thresholds
	fmt.Printf("\nScale thresholds:\n")
	fmt.Printf("  Memory Scale Up: %.1f%%, Scale Down: %.1f%%\n",
		optimizer.scaleThresholds.MemoryScaleUp*100, optimizer.scaleThresholds.MemoryScaleDown*100)
	fmt.Printf("  CPU Scale Up: %.1f%%, Scale Down: %.1f%%\n",
		optimizer.scaleThresholds.CPUScaleUp*100, optimizer.scaleThresholds.CPUScaleDown*100)
	fmt.Printf("  Environment Limit: %d\n", optimizer.scaleThresholds.EnvironmentLimit)

	// Cleanup
	fmt.Println("\nCleaning up environments...")
	for i, env := range environments {
		if env != nil {
			if err := manager.CleanupEnvironment(env.ID); err != nil {
				log.Printf("Failed to cleanup environment %d: %v", i+1, err)
			} else {
				fmt.Printf("Cleaned up environment %d: %s\n", i+1, env.ID)
			}
		}
	}

	// Final utilization check
	memUtil, cpuUtil, envUtil = manager.resourcePool.GetUtilization()
	fmt.Printf("\nFinal utilization - Memory: %.1f%%, CPU: %.1f%%, Environments: %.1f%%\n",
		memUtil*100, cpuUtil*100, envUtil*100)

	fmt.Println("\n=== Resource Optimization Demo completed ===")
}

// ExampleEnvironmentQueue demonstrates environment queuing and conflict resolution
func ExampleEnvironmentQueue() {
	fmt.Println("=== Environment Queue Demo ===")

	queue := NewEnvironmentQueue(2) // Max 2 concurrent environments

	// Create requests with different priorities
	requests := []EnvironmentRequest{
		{
			TestSuite: "critical-security-tests",
			Priority:  10, // Highest priority
			Resources: ResourceAllocation{Memory: 2048 * 1024 * 1024, CPUQuota: 200000},
			Timeout:   10 * time.Minute,
			Metadata:  map[string]interface{}{"type": "security", "critical": true},
		},
		{
			TestSuite: "performance-regression-tests",
			Priority:  7,
			Resources: ResourceAllocation{Memory: 1024 * 1024 * 1024, CPUQuota: 100000},
			Timeout:   15 * time.Minute,
			Metadata:  map[string]interface{}{"type": "performance", "baseline": "v1.2.0"},
		},
		{
			TestSuite: "unit-tests",
			Priority:  3,
			Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
			Timeout:   5 * time.Minute,
			Metadata:  map[string]interface{}{"type": "unit", "fast": true},
		},
		{
			TestSuite: "integration-tests",
			Priority:  5,
			Resources: ResourceAllocation{Memory: 512 * 1024 * 1024, CPUQuota: 50000},
			Timeout:   8 * time.Minute,
			Metadata:  map[string]interface{}{"type": "integration", "database": true},
		},
		{
			TestSuite: "ui-automation-tests",
			Priority:  2, // Lowest priority
			Resources: ResourceAllocation{Memory: 1024 * 1024 * 1024, CPUQuota: 75000},
			Timeout:   20 * time.Minute,
			Metadata:  map[string]interface{}{"type": "ui", "browser": "chrome"},
		},
	}

	// Queue all requests
	fmt.Printf("Queuing %d environment requests...\n", len(requests))
	requestIDs := make([]string, len(requests))
	for i, req := range requests {
		requestIDs[i] = queue.QueueEnvironmentRequest(req)
		fmt.Printf("  Queued: %s (Priority: %d, TestSuite: %s)\n", 
			requestIDs[i], req.Priority, req.TestSuite)
	}

	// Monitor queue status over time
	fmt.Println("\nMonitoring queue status...")
	for iteration := 0; iteration < 5; iteration++ {
		time.Sleep(1 * time.Second)
		
		fmt.Printf("\n--- Status Check %d ---\n", iteration+1)
		
		pendingCount := 0
		processingCount := 0
		completedCount := 0
		failedCount := 0
		
		for i, requestID := range requestIDs {
			req, result, err := queue.GetRequestStatus(requestID)
			if err != nil {
				fmt.Printf("  Request %d: Error - %v\n", i+1, err)
				continue
			}

			if req != nil {
				switch req.Status {
				case RequestStatusPending:
					pendingCount++
					fmt.Printf("  Request %d: PENDING - %s\n", i+1, req.TestSuite)
				case RequestStatusProcessing:
					processingCount++
					duration := time.Since(*req.StartedAt)
					fmt.Printf("  Request %d: PROCESSING - %s (for %v)\n", 
						i+1, req.TestSuite, duration)
				}
			}

			if result != nil {
				if result.Error != nil {
					failedCount++
					fmt.Printf("  Request %d: FAILED - %s (Error: %v)\n", 
						i+1, requests[i].TestSuite, result.Error)
				} else {
					completedCount++
					fmt.Printf("  Request %d: COMPLETED - %s (Duration: %v)\n", 
						i+1, requests[i].TestSuite, result.Duration)
				}
			}
		}
		
		fmt.Printf("Summary: %d pending, %d processing, %d completed, %d failed\n",
			pendingCount, processingCount, completedCount, failedCount)
		
		if pendingCount == 0 && processingCount == 0 {
			fmt.Println("All requests processed!")
			break
		}
	}

	fmt.Println("\n=== Environment Queue Demo completed ===")
}

// RunAllExamples runs all the example demonstrations
func RunAllExamples() {
	fmt.Println("Starting Docker-based Environment Isolation System Examples...")
	fmt.Println("================================================================")

	// Run examples in sequence
	ExampleEnvironmentQueue()
	fmt.Println()
	
	ExampleResourceOptimization()
	fmt.Println()
	
	// Note: ExampleDockerEnvironmentIsolation requires Docker to be available
	// and may take longer to run, so it's commented out for quick demos
	// ExampleDockerEnvironmentIsolation()

	fmt.Println("All examples completed successfully!")
}