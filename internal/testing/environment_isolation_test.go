package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceOptimizer_ScaleThresholds(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	
	// Test initial thresholds
	assert.Equal(t, 0.8, optimizer.scaleThresholds.MemoryScaleUp)
	assert.Equal(t, 0.3, optimizer.scaleThresholds.MemoryScaleDown)
	assert.Equal(t, 0.8, optimizer.scaleThresholds.CPUScaleUp)
	assert.Equal(t, 0.3, optimizer.scaleThresholds.CPUScaleDown)
	assert.Equal(t, 8, optimizer.scaleThresholds.EnvironmentLimit)
}

func TestResourceOptimizer_OptimizationRules(t *testing.T) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	
	// Test default optimization rules
	assert.Len(t, optimizer.optimizationRules, 3)
	
	// Check cleanup rule
	cleanupRule := optimizer.optimizationRules[0]
	assert.Equal(t, "cleanup_idle_environments", cleanupRule.Name)
	assert.Equal(t, "cleanup_idle_environments", cleanupRule.Action)
	assert.True(t, cleanupRule.Enabled)
	assert.Equal(t, 1, cleanupRule.Priority)
	
	// Check optimization rule
	optimizeRule := optimizer.optimizationRules[1]
	assert.Equal(t, "optimize_resource_allocation", optimizeRule.Name)
	assert.Equal(t, "optimize_resource_allocation", optimizeRule.Action)
	assert.True(t, optimizeRule.Enabled)
	assert.Equal(t, 2, optimizeRule.Priority)
	
	// Check consolidation rule (should be disabled by default)
	consolidateRule := optimizer.optimizationRules[2]
	assert.Equal(t, "consolidate_environments", consolidateRule.Name)
	assert.Equal(t, "consolidate_environments", consolidateRule.Action)
	assert.False(t, consolidateRule.Enabled)
	assert.Equal(t, 3, consolidateRule.Priority)
}

func TestEnvironmentQueue_BasicQueueing(t *testing.T) {
	queue := NewEnvironmentQueue(2)
	
	// Create test requests
	request1 := EnvironmentRequest{
		TestSuite: "test-suite-1",
		Priority:  1,
		Resources: ResourceAllocation{
			Memory:   512 * 1024 * 1024,
			CPUQuota: 50000,
		},
		Timeout: 5 * time.Minute,
	}
	
	request2 := EnvironmentRequest{
		TestSuite: "test-suite-2",
		Priority:  2, // Higher priority
		Resources: ResourceAllocation{
			Memory:   256 * 1024 * 1024,
			CPUQuota: 25000,
		},
		Timeout: 3 * time.Minute,
	}
	
	// Queue requests
	id1 := queue.QueueEnvironmentRequest(request1)
	id2 := queue.QueueEnvironmentRequest(request2)
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	
	// Check request status
	req1, result1, err1 := queue.GetRequestStatus(id1)
	assert.NoError(t, err1)
	assert.NotNil(t, req1)
	assert.Nil(t, result1) // Should still be processing or pending
	
	req2, result2, err2 := queue.GetRequestStatus(id2)
	assert.NoError(t, err2)
	assert.NotNil(t, req2)
	assert.Nil(t, result2) // Should still be processing or pending
}

func TestEnvironmentQueue_PriorityOrdering(t *testing.T) {
	queue := NewEnvironmentQueue(1) // Only 1 concurrent to test queuing
	
	// Create requests with different priorities
	lowPriorityRequest := EnvironmentRequest{
		TestSuite: "low-priority",
		Priority:  1,
		Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
		Timeout:   1 * time.Minute,
	}
	
	highPriorityRequest := EnvironmentRequest{
		TestSuite: "high-priority",
		Priority:  5,
		Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
		Timeout:   1 * time.Minute,
	}
	
	mediumPriorityRequest := EnvironmentRequest{
		TestSuite: "medium-priority",
		Priority:  3,
		Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
		Timeout:   1 * time.Minute,
	}
	
	// Queue in non-priority order
	queue.QueueEnvironmentRequest(lowPriorityRequest)
	queue.QueueEnvironmentRequest(highPriorityRequest)
	queue.QueueEnvironmentRequest(mediumPriorityRequest)
	
	// Verify queue is ordered by priority (high to low)
	queue.mutex.RLock()
	assert.Len(t, queue.queue, 2) // One should be processing, two in queue
	if len(queue.queue) >= 2 {
		assert.Equal(t, 3, queue.queue[0].Priority) // Medium priority should be first in queue
		assert.Equal(t, 1, queue.queue[1].Priority) // Low priority should be last
	}
	queue.mutex.RUnlock()
}

func TestCostTracker_BasicTracking(t *testing.T) {
	tracker := NewCostTracker()
	
	envID := "test-env-1"
	
	// Start tracking
	tracker.StartTracking(envID)
	
	// Verify tracking started
	cost := tracker.GetEnvironmentCost(envID)
	require.NotNil(t, cost)
	assert.Equal(t, envID, cost.EnvironmentID)
	assert.False(t, cost.StartTime.IsZero())
	
	// Update costs
	tracker.UpdateCost(envID, 50.0, 1024.0, 10.0, 5.0) // 50% CPU, 1GB memory, 10GB storage, 5GB network
	
	// Verify costs calculated
	cost = tracker.GetEnvironmentCost(envID)
	require.NotNil(t, cost)
	assert.Equal(t, 50.0*0.01, cost.CPUCost)      // $0.50
	assert.Equal(t, 1024.0*0.005, cost.MemoryCost) // $5.12
	assert.Equal(t, 10.0*0.001, cost.StorageCost)  // $0.01
	assert.Equal(t, 5.0*0.002, cost.NetworkCost)   // $0.01
	
	expectedTotal := cost.CPUCost + cost.MemoryCost + cost.StorageCost + cost.NetworkCost
	assert.Equal(t, expectedTotal, cost.TotalCost)
	
	// Stop tracking
	finalCost := tracker.StopTracking(envID)
	require.NotNil(t, finalCost)
	assert.Equal(t, expectedTotal, finalCost.TotalCost)
	assert.Greater(t, finalCost.Duration, time.Duration(0))
	
	// Verify environment removed from tracking
	cost = tracker.GetEnvironmentCost(envID)
	assert.Nil(t, cost)
}

func TestCostTracker_MultipleEnvironments(t *testing.T) {
	tracker := NewCostTracker()
	
	env1 := "test-env-1"
	env2 := "test-env-2"
	
	// Start tracking multiple environments
	tracker.StartTracking(env1)
	tracker.StartTracking(env2)
	
	// Update costs for both
	tracker.UpdateCost(env1, 25.0, 512.0, 5.0, 2.0)
	tracker.UpdateCost(env2, 75.0, 2048.0, 20.0, 10.0)
	
	// Verify individual costs
	cost1 := tracker.GetEnvironmentCost(env1)
	cost2 := tracker.GetEnvironmentCost(env2)
	
	require.NotNil(t, cost1)
	require.NotNil(t, cost2)
	
	assert.NotEqual(t, cost1.TotalCost, cost2.TotalCost)
	assert.Greater(t, cost2.TotalCost, cost1.TotalCost) // env2 should cost more
	
	// Verify total cost
	expectedTotal := cost1.TotalCost + cost2.TotalCost
	assert.Equal(t, expectedTotal, tracker.GetTotalCost())
}

func TestRecoveryStrategies_DefaultStrategies(t *testing.T) {
	strategies := getDefaultRecoveryStrategies()
	
	// Should have default strategies for common failures
	assert.Contains(t, strategies, "insufficient resources available")
	assert.Contains(t, strategies, "failed to create container")
	assert.Contains(t, strategies, "port already in use")
	
	// Check resource cleanup strategy
	resourceStrategy := strategies["insufficient resources available"]
	assert.Equal(t, "resource_cleanup", resourceStrategy.Name)
	assert.Equal(t, "cleanup_failed_containers", resourceStrategy.RecoveryAction)
	assert.Equal(t, 2, resourceStrategy.MaxRetries)
	assert.Equal(t, 30*time.Second, resourceStrategy.RetryDelay)
	assert.True(t, resourceStrategy.Enabled)
	
	// Check docker cleanup strategy
	dockerStrategy := strategies["failed to create container"]
	assert.Equal(t, "docker_cleanup", dockerStrategy.Name)
	assert.Equal(t, "free_resources", dockerStrategy.RecoveryAction)
	assert.Equal(t, 3, dockerStrategy.MaxRetries)
	assert.Equal(t, 15*time.Second, dockerStrategy.RetryDelay)
	assert.True(t, dockerStrategy.Enabled)
	
	// Check port conflict strategy
	portStrategy := strategies["port already in use"]
	assert.Equal(t, "port_conflict", portStrategy.Name)
	assert.Equal(t, "cleanup_failed_containers", portStrategy.RecoveryAction)
	assert.Equal(t, 5, portStrategy.MaxRetries)
	assert.Equal(t, 10*time.Second, portStrategy.RetryDelay)
	assert.True(t, portStrategy.Enabled)
}

func TestAlertRules_DefaultRules(t *testing.T) {
	rules := getDefaultAlertRules()
	
	// Should have default alert rules
	assert.Len(t, rules, 4)
	
	ruleNames := make([]string, len(rules))
	for i, rule := range rules {
		ruleNames[i] = rule.Name
	}
	
	assert.Contains(t, ruleNames, "high_memory_usage")
	assert.Contains(t, ruleNames, "high_cpu_usage")
	assert.Contains(t, ruleNames, "environment_creation_failure")
	assert.Contains(t, ruleNames, "low_performance_score")
	
	// Check high memory usage rule
	var memoryRule AlertRule
	for _, rule := range rules {
		if rule.Name == "high_memory_usage" {
			memoryRule = rule
			break
		}
	}
	
	assert.Equal(t, "memory_usage > 0.9", memoryRule.Condition)
	assert.Equal(t, 0.9, memoryRule.Threshold)
	assert.Equal(t, "critical", memoryRule.Severity)
	assert.True(t, memoryRule.Enabled)
	assert.Equal(t, 5*time.Minute, memoryRule.Cooldown)
}

func TestLogAlertHandler_HandleAlert(t *testing.T) {
	handler := &LogAlertHandler{}
	
	alert := EnvironmentAlert{
		ID:            "test-alert-1",
		EnvironmentID: "test-env-1",
		RuleName:      "high_memory_usage",
		Severity:      "critical",
		Message:       "Memory usage exceeded 90%",
		Timestamp:     time.Now(),
	}
	
	// Should not return error
	err := handler.HandleAlert(alert)
	assert.NoError(t, err)
}

func TestEmailAlertHandler_HandleAlert(t *testing.T) {
	handler := &EmailAlertHandler{
		SMTPServer: "smtp.example.com",
		From:       "alerts@example.com",
		To:         []string{"admin@example.com"},
	}
	
	alert := EnvironmentAlert{
		ID:            "test-alert-1",
		EnvironmentID: "test-env-1",
		RuleName:      "high_cpu_usage",
		Severity:      "warning",
		Message:       "CPU usage exceeded 80%",
		Timestamp:     time.Now(),
	}
	
	// Should not return error (mock implementation)
	err := handler.HandleAlert(alert)
	assert.NoError(t, err)
}

func TestSlackAlertHandler_HandleAlert(t *testing.T) {
	handler := &SlackAlertHandler{
		WebhookURL: "https://hooks.slack.com/services/test",
		Channel:    "#alerts",
	}
	
	alert := EnvironmentAlert{
		ID:            "test-alert-1",
		EnvironmentID: "test-env-1",
		RuleName:      "environment_creation_failure",
		Severity:      "critical",
		Message:       "Environment creation failure rate exceeded 10%",
		Timestamp:     time.Now(),
	}
	
	// Should not return error (mock implementation)
	err := handler.HandleAlert(alert)
	assert.NoError(t, err)
}

// Benchmark tests for performance validation

func BenchmarkResourceOptimizer_OptimizeResources(b *testing.B) {
	manager, err := NewTestEnvironmentManager()
	require.NoError(b, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.optimizeResources()
	}
}

func BenchmarkEnvironmentQueue_QueueRequest(b *testing.B) {
	queue := NewEnvironmentQueue(10)
	
	request := EnvironmentRequest{
		TestSuite: "benchmark-test",
		Priority:  1,
		Resources: ResourceAllocation{
			Memory:   256 * 1024 * 1024,
			CPUQuota: 25000,
		},
		Timeout: 1 * time.Minute,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.QueueEnvironmentRequest(request)
	}
}

func BenchmarkCostTracker_UpdateCost(b *testing.B) {
	tracker := NewCostTracker()
	envID := "benchmark-env"
	tracker.StartTracking(envID)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.UpdateCost(envID, 50.0, 1024.0, 10.0, 5.0)
	}
}