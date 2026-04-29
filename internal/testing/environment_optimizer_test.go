package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceOptimizer_RequestEnvironment(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	ctx := context.Background()

	// Test immediate allocation
	env, err := optimizer.RequestEnvironment(ctx, "unit-tests", 1)
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.Equal(t, "unit-tests", env.TestSuite)

	// Wait for environment to be ready
	waitForEnvironmentReady(t, manager, env.ID, 2*time.Minute)

	// Cleanup
	err = manager.CleanupEnvironment(env.ID)
	require.NoError(t, err)
}

func TestResourceOptimizer_QueueManagement(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Set very low resource limits to force queuing
	manager.resourcePool.MaxMemory = 256 * 1024 * 1024 // 256MB total
	manager.resourcePool.MaxCPU = 25000                // 25% CPU total
	manager.resourcePool.MaxEnvironments = 1

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	ctx := context.Background()

	// First request should succeed immediately
	env1, err := optimizer.RequestEnvironment(ctx, "test-1", 1)
	require.NoError(t, err)

	// Second request should be queued
	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()

	start := time.Now()
	env2, err := optimizer.RequestEnvironment(ctx2, "test-2", 2) // Higher priority
	
	// Should either succeed after env1 is cleaned up, or timeout
	if err != nil {
		assert.Contains(t, err.Error(), "timeout")
		duration := time.Since(start)
		assert.GreaterOrEqual(t, duration, 10*time.Second)
	} else {
		assert.NotNil(t, env2)
		// Cleanup
		manager.CleanupEnvironment(env2.ID)
	}

	// Cleanup first environment
	manager.CleanupEnvironment(env1.ID)
}

func TestResourceOptimizer_PriorityOrdering(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Create requests with different priorities
	req1 := &EnvironmentRequest{
		ID:          "req1",
		TestSuite:   "test1",
		Priority:    1,
		RequestedAt: time.Now(),
		ResultChan:  make(chan *EnvironmentResult, 1),
		Context:     context.Background(),
	}

	req2 := &EnvironmentRequest{
		ID:          "req2",
		TestSuite:   "test2",
		Priority:    3, // Higher priority
		RequestedAt: time.Now().Add(1 * time.Second), // Later request time
		ResultChan:  make(chan *EnvironmentResult, 1),
		Context:     context.Background(),
	}

	req3 := &EnvironmentRequest{
		ID:          "req3",
		TestSuite:   "test3",
		Priority:    2,
		RequestedAt: time.Now().Add(-1 * time.Second), // Earlier request time
		ResultChan:  make(chan *EnvironmentResult, 1),
		Context:     context.Background(),
	}

	// Add requests to queue
	optimizer.queue.requests = []*EnvironmentRequest{req1, req2, req3}
	optimizer.sortQueueByPriority()

	// Should be sorted by priority (highest first), then by time for same priority
	assert.Equal(t, "req2", optimizer.queue.requests[0].ID) // Priority 3
	assert.Equal(t, "req3", optimizer.queue.requests[1].ID) // Priority 2
	assert.Equal(t, "req1", optimizer.queue.requests[2].ID) // Priority 1
}

func TestResourceOptimizer_OptimalResourceCalculation(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Test different test suite types
	testCases := []struct {
		testSuite      string
		expectedMemory int64
		expectedCPU    int64
	}{
		{"unit-tests", 256 * 1024 * 1024, 25000},
		{"integration-tests", 768 * 1024 * 1024, 75000},
		{"performance-tests", 1024 * 1024 * 1024, 100000},
		{"load-tests", 1024 * 1024 * 1024, 100000},
		{"unknown-tests", 512 * 1024 * 1024, 50000}, // Default
	}

	for _, tc := range testCases {
		t.Run(tc.testSuite, func(t *testing.T) {
			allocation := optimizer.calculateOptimalResources(tc.testSuite)
			assert.Equal(t, tc.expectedMemory, allocation.Memory)
			assert.Equal(t, tc.expectedCPU, allocation.CPUQuota)
		})
	}
}

func TestResourceOptimizer_MetricsTracking(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Initial metrics should be zero
	metrics := optimizer.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRequests)
	assert.Equal(t, int64(0), metrics.SuccessfulCreations)
	assert.Equal(t, int64(0), metrics.FailedCreations)

	// Simulate some metrics updates
	optimizer.metrics.TotalRequests = 10
	optimizer.metrics.SuccessfulCreations = 8
	optimizer.metrics.FailedCreations = 2
	optimizer.metrics.ResourceEfficiency = 0.8

	updatedMetrics := optimizer.GetMetrics()
	assert.Equal(t, int64(10), updatedMetrics.TotalRequests)
	assert.Equal(t, int64(8), updatedMetrics.SuccessfulCreations)
	assert.Equal(t, int64(2), updatedMetrics.FailedCreations)
	assert.Equal(t, 0.8, updatedMetrics.ResourceEfficiency)
}

func TestResourceOptimizer_CostTracking(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Test cost calculation
	resources := ResourceAllocation{
		Memory:    512 * 1024 * 1024, // 512MB
		CPUQuota:  50000,             // 50% CPU (0.5 cores)
		DiskSpace: 1024 * 1024 * 1024, // 1GB
	}

	hourlyRate := optimizer.calculateHourlyRate(resources)
	
	// Expected: (512 * 0.001) + (0.5 * 0.05) + (1 * 0.01) = 0.512 + 0.025 + 0.01 = 0.547
	expectedRate := 0.547
	assert.InDelta(t, expectedRate, hourlyRate, 0.001)

	// Test cost summary
	costSummary := optimizer.GetCostSummary()
	assert.NotNil(t, costSummary.ResourceCosts)
	assert.Equal(t, 0.001, costSummary.ResourceCosts.MemoryPerMBPerHour)
	assert.Equal(t, 0.05, costSummary.ResourceCosts.CPUPerCorePerHour)
}

func TestResourceOptimizer_ConflictResolution(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Test strategy condition checking
	strategy := ConflictStrategy{
		Name:     "test_strategy",
		Priority: 1,
		Conditions: map[string]interface{}{
			"memory_utilization_above": 0.5,
			"cpu_utilization_above":    0.3,
		},
		Actions: []string{"cleanup_idle_environments"},
		Enabled: true,
	}

	// Set resource pool to simulate high utilization
	manager.resourcePool.allocatedMemory = manager.resourcePool.MaxMemory / 2   // 50% memory
	manager.resourcePool.allocatedCPU = manager.resourcePool.MaxCPU * 4 / 10    // 40% CPU

	shouldApply := optimizer.shouldApplyStrategy(strategy)
	assert.True(t, shouldApply, "Strategy should apply when conditions are met")

	// Test with conditions not met
	manager.resourcePool.allocatedMemory = manager.resourcePool.MaxMemory / 4   // 25% memory
	manager.resourcePool.allocatedCPU = manager.resourcePool.MaxCPU / 10        // 10% CPU

	shouldApply = optimizer.shouldApplyStrategy(strategy)
	assert.False(t, shouldApply, "Strategy should not apply when conditions are not met")
}

func TestResourceOptimizer_ScalingDecisions(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Record initial limits
	initialMemory := manager.resourcePool.MaxMemory
	initialCPU := manager.resourcePool.MaxCPU
	initialEnvs := manager.resourcePool.MaxEnvironments

	// Simulate high utilization to trigger scale up
	manager.resourcePool.allocatedMemory = int64(float64(manager.resourcePool.MaxMemory) * 0.85)
	manager.resourcePool.allocatedCPU = int64(float64(manager.resourcePool.MaxCPU) * 0.85)

	optimizer.scaleUp()

	// Verify resources were scaled up
	assert.Greater(t, manager.resourcePool.MaxMemory, initialMemory)
	assert.Greater(t, manager.resourcePool.MaxCPU, initialCPU)
	assert.Greater(t, manager.resourcePool.MaxEnvironments, initialEnvs)

	// Test scale down
	currentMemory := manager.resourcePool.MaxMemory
	currentCPU := manager.resourcePool.MaxCPU
	currentEnvs := manager.resourcePool.MaxEnvironments

	optimizer.scaleDown()

	// Verify resources were scaled down
	assert.Less(t, manager.resourcePool.MaxMemory, currentMemory)
	assert.Less(t, manager.resourcePool.MaxCPU, currentCPU)
	assert.Less(t, manager.resourcePool.MaxEnvironments, currentEnvs)
}

func TestResourceOptimizer_QueueStatus(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Add some test requests to queue
	now := time.Now()
	requests := []*EnvironmentRequest{
		{
			ID:          "req1",
			Priority:    1,
			RequestedAt: now.Add(-5 * time.Minute),
		},
		{
			ID:          "req2",
			Priority:    2,
			RequestedAt: now.Add(-2 * time.Minute),
		},
		{
			ID:          "req3",
			Priority:    1,
			RequestedAt: now.Add(-1 * time.Minute),
		},
	}

	optimizer.queue.requests = requests

	status := optimizer.GetQueueStatus()
	
	assert.Equal(t, 3, status["queue_length"])
	assert.Equal(t, 100, status["max_queue_size"])
	
	oldestAge, ok := status["oldest_request"].(time.Duration)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, oldestAge, 5*time.Minute)

	priorityDist, ok := status["priority_distribution"].(map[int]int)
	assert.True(t, ok)
	assert.Equal(t, 2, priorityDist[1]) // Two priority 1 requests
	assert.Equal(t, 1, priorityDist[2]) // One priority 2 request
}

func TestResourceOptimizer_ContextCancellation(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Set very low limits to force queuing
	manager.resourcePool.MaxMemory = 256 * 1024 * 1024
	manager.resourcePool.MaxCPU = 25000
	manager.resourcePool.MaxEnvironments = 1

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Create first environment to fill capacity
	ctx1 := context.Background()
	env1, err := optimizer.RequestEnvironment(ctx1, "test-1", 1)
	require.NoError(t, err)

	// Create context that will be cancelled
	ctx2, cancel := context.WithCancel(context.Background())

	// Start second request (should be queued)
	done := make(chan struct{})
	var env2 *IsolatedEnvironment
	var err2 error

	go func() {
		defer close(done)
		env2, err2 = optimizer.RequestEnvironment(ctx2, "test-2", 1)
	}()

	// Cancel the context after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for the request to complete
	<-done

	// Should have been cancelled
	assert.Nil(t, env2)
	assert.Equal(t, context.Canceled, err2)

	// Cleanup
	manager.CleanupEnvironment(env1.ID)
}

func TestResourceOptimizer_TimeoutHandling(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Set very low limits to force queuing
	manager.resourcePool.MaxMemory = 256 * 1024 * 1024
	manager.resourcePool.MaxCPU = 25000
	manager.resourcePool.MaxEnvironments = 1

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Create first environment to fill capacity
	ctx1 := context.Background()
	env1, err := optimizer.RequestEnvironment(ctx1, "test-1", 1)
	require.NoError(t, err)

	// Create context with short timeout
	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	env2, err := optimizer.RequestEnvironment(ctx2, "test-2", 1)

	// Should timeout
	assert.Nil(t, env2)
	assert.Equal(t, context.DeadlineExceeded, err)
	
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 2*time.Second)
	assert.Less(t, duration, 3*time.Second) // Should not take much longer than timeout

	// Cleanup
	manager.CleanupEnvironment(env1.ID)
}

// Benchmark tests for performance optimization

func BenchmarkResourceOptimizer_RequestEnvironment(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker not available, skipping benchmark")
	}

	manager, err := NewTestEnvironmentManager()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		env, err := optimizer.RequestEnvironment(ctx, "benchmark-test", 1)
		if err != nil {
			b.Fatal(err)
		}

		// Wait for environment to be ready
		waitForEnvironmentReadyBench(b, manager, env.ID, 2*time.Minute)

		// Cleanup immediately
		err = manager.CleanupEnvironment(env.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResourceOptimizer_QueueProcessing(b *testing.B) {
	manager, err := NewTestEnvironmentManager()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	defer optimizer.Stop()

	// Create many requests
	requests := make([]*EnvironmentRequest, b.N)
	for i := 0; i < b.N; i++ {
		requests[i] = &EnvironmentRequest{
			ID:          generateUniqueID(),
			TestSuite:   "benchmark-test",
			Priority:    i % 5, // Vary priorities
			RequestedAt: time.Now(),
			ResultChan:  make(chan *EnvironmentResult, 1),
			Context:     context.Background(),
		}
	}

	b.ResetTimer()

	// Benchmark queue operations
	for i := 0; i < b.N; i++ {
		optimizer.queue.requests = append(optimizer.queue.requests, requests[i])
		optimizer.sortQueueByPriority()
	}
}