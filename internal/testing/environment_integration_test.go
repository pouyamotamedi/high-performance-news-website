package testing

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentManager_BasicIntegration(t *testing.T) {
	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Test creating an isolated environment
	env, err := manager.CreateIsolatedEnvironment("basic-integration-test")
	require.NoError(t, err)
	assert.NotEmpty(t, env.ID)
	assert.Equal(t, "basic-integration-test", env.TestSuite)
	assert.Equal(t, EnvironmentStatusCreating, env.Status)

	// Wait for environment to be ready (with timeout)
	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var ready bool
	for !ready {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for environment to be ready")
		case <-ticker.C:
			currentEnv, err := manager.GetEnvironment(env.ID)
			require.NoError(t, err)
			
			if currentEnv.Status == EnvironmentStatusReady {
				ready = true
				
				// Test database connectivity
				db, err := sql.Open("postgres", currentEnv.DatabaseURL)
				require.NoError(t, err)
				defer db.Close()

				err = db.Ping()
				assert.NoError(t, err, "Should be able to connect to database")

				// Test basic database operations
				_, err = db.Exec("CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(100))")
				assert.NoError(t, err, "Should be able to create table")

				_, err = db.Exec("INSERT INTO test_table (name) VALUES ('test')")
				assert.NoError(t, err, "Should be able to insert data")

				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
				assert.NoError(t, err, "Should be able to query data")
				assert.Equal(t, 1, count, "Should have one record")

			} else if currentEnv.Status == EnvironmentStatusFailed {
				t.Fatalf("Environment creation failed: %s", currentEnv.ErrorMessage)
			}
		}
	}

	// Test cleanup
	err = manager.CleanupEnvironment(env.ID)
	require.NoError(t, err)

	// Verify environment is removed
	_, err = manager.GetEnvironment(env.ID)
	assert.Error(t, err, "Environment should be removed after cleanup")
}

func TestEnvironmentManager_ResourceManagement(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Check initial resource utilization
	memUtil, cpuUtil, envUtil := manager.resourcePool.GetUtilization()
	assert.Equal(t, 0.0, memUtil)
	assert.Equal(t, 0.0, cpuUtil)
	assert.Equal(t, 0.0, envUtil)

	// Create an environment
	env, err := manager.CreateIsolatedEnvironment("resource-test")
	require.NoError(t, err)

	// Wait for environment to be ready
	waitForEnvironmentReady(t, manager, env.ID, 3*time.Minute)

	// Check resource utilization after creation
	memUtil, cpuUtil, envUtil = manager.resourcePool.GetUtilization()
	assert.Greater(t, memUtil, 0.0, "Memory should be allocated")
	assert.Greater(t, cpuUtil, 0.0, "CPU should be allocated")
	assert.Greater(t, envUtil, 0.0, "Environment slot should be used")

	// Cleanup environment
	err = manager.CleanupEnvironment(env.ID)
	require.NoError(t, err)

	// Check resource utilization after cleanup
	memUtil, cpuUtil, envUtil = manager.resourcePool.GetUtilization()
	assert.Equal(t, 0.0, memUtil, "Memory should be released")
	assert.Equal(t, 0.0, cpuUtil, "CPU should be released")
	assert.Equal(t, 0.0, envUtil, "Environment slot should be released")
}

func TestEnvironmentManager_MultipleEnvironments(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Create multiple environments
	env1, err := manager.CreateIsolatedEnvironment("multi-test-1")
	require.NoError(t, err)

	env2, err := manager.CreateIsolatedEnvironment("multi-test-2")
	require.NoError(t, err)

	// Verify they have different IDs and URLs
	assert.NotEqual(t, env1.ID, env2.ID)
	assert.NotEqual(t, env1.TestSuite, env2.TestSuite)

	// Wait for both to be ready
	waitForEnvironmentReady(t, manager, env1.ID, 3*time.Minute)
	waitForEnvironmentReady(t, manager, env2.ID, 3*time.Minute)

	// Verify isolation - each should have its own database
	db1, err := sql.Open("postgres", env1.DatabaseURL)
	require.NoError(t, err)
	defer db1.Close()

	db2, err := sql.Open("postgres", env2.DatabaseURL)
	require.NoError(t, err)
	defer db2.Close()

	// Create different tables in each database
	_, err = db1.Exec("CREATE TABLE env1_table (id SERIAL PRIMARY KEY, data TEXT)")
	require.NoError(t, err)

	_, err = db2.Exec("CREATE TABLE env2_table (id SERIAL PRIMARY KEY, info TEXT)")
	require.NoError(t, err)

	// Verify isolation - env1 shouldn't see env2's table
	_, err = db1.Exec("SELECT * FROM env2_table")
	assert.Error(t, err, "env1 should not see env2's table")

	_, err = db2.Exec("SELECT * FROM env1_table")
	assert.Error(t, err, "env2 should not see env1's table")

	// Cleanup both environments
	err = manager.CleanupEnvironment(env1.ID)
	require.NoError(t, err)

	err = manager.CleanupEnvironment(env2.ID)
	require.NoError(t, err)
}

func TestEnhancedEnvironmentManager_WithRecovery(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	enhanced, err := NewEnhancedEnvironmentManager()
	require.NoError(t, err)
	defer enhanced.Shutdown()

	// Test environment creation with recovery
	env, err := enhanced.CreateIsolatedEnvironmentWithRecovery("recovery-test")
	require.NoError(t, err)
	assert.NotEmpty(t, env.ID)
	assert.Equal(t, "recovery-test", env.TestSuite)

	// Wait for environment to be ready
	waitForEnvironmentReady(t, enhanced.TestEnvironmentManager, env.ID, 3*time.Minute)

	// Verify performance monitoring started
	metrics, err := enhanced.GetEnvironmentPerformanceMetrics(env.ID)
	require.NoError(t, err)
	assert.Equal(t, env.ID, metrics.EnvironmentID)

	// Cleanup
	err = enhanced.CleanupEnvironment(env.ID)
	require.NoError(t, err)
}

func TestEnvironmentQueue_Integration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	queue := NewEnvironmentQueue(2)

	// Create multiple requests
	requests := []EnvironmentRequest{
		{
			TestSuite: "queue-test-1",
			Priority:  1,
			Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
			Timeout:   2 * time.Minute,
		},
		{
			TestSuite: "queue-test-2",
			Priority:  3,
			Resources: ResourceAllocation{Memory: 512 * 1024 * 1024, CPUQuota: 50000},
			Timeout:   2 * time.Minute,
		},
		{
			TestSuite: "queue-test-3",
			Priority:  2,
			Resources: ResourceAllocation{Memory: 256 * 1024 * 1024, CPUQuota: 25000},
			Timeout:   2 * time.Minute,
		},
	}

	// Queue all requests
	requestIDs := make([]string, len(requests))
	for i, req := range requests {
		requestIDs[i] = queue.QueueEnvironmentRequest(req)
		assert.NotEmpty(t, requestIDs[i])
	}

	// Wait a bit for processing
	time.Sleep(1 * time.Second)

	// Check status of all requests
	for _, id := range requestIDs {
		req, result, err := queue.GetRequestStatus(id)
		assert.NoError(t, err)
		
		if req != nil {
			assert.Contains(t, []RequestStatus{RequestStatusPending, RequestStatusProcessing}, req.Status)
		}
		
		if result != nil {
			assert.Contains(t, []RequestStatus{RequestStatusCompleted, RequestStatusFailed, RequestStatusTimeout}, req.Status)
		}
	}
}

func TestResourceOptimizer_Integration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	optimizer := NewResourceOptimizer(manager)

	// Create some environments to test optimization
	env1, err := manager.CreateIsolatedEnvironment("optimizer-test-1")
	require.NoError(t, err)

	env2, err := manager.CreateIsolatedEnvironment("optimizer-test-2")
	require.NoError(t, err)

	// Wait for environments to be ready
	waitForEnvironmentReady(t, manager, env1.ID, 2*time.Minute)
	waitForEnvironmentReady(t, manager, env2.ID, 2*time.Minute)

	// Run optimization
	optimizer.optimizeResources()

	// Verify environments are still running
	env1Updated, err := manager.GetEnvironment(env1.ID)
	require.NoError(t, err)
	assert.Equal(t, EnvironmentStatusReady, env1Updated.Status)

	env2Updated, err := manager.GetEnvironment(env2.ID)
	require.NoError(t, err)
	assert.Equal(t, EnvironmentStatusReady, env2Updated.Status)

	// Cleanup
	err = manager.CleanupEnvironment(env1.ID)
	require.NoError(t, err)

	err = manager.CleanupEnvironment(env2.ID)
	require.NoError(t, err)
}

func TestCostTracker_Integration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	tracker := NewCostTracker()

	// Create environment
	env, err := manager.CreateIsolatedEnvironment("cost-test")
	require.NoError(t, err)

	// Start cost tracking
	tracker.StartTracking(env.ID)

	// Wait for environment to be ready
	waitForEnvironmentReady(t, manager, env.ID, 2*time.Minute)

	// Simulate some usage and update costs
	tracker.UpdateCost(env.ID, 30.0, 512.0, 5.0, 2.0)

	// Verify cost tracking
	cost := tracker.GetEnvironmentCost(env.ID)
	require.NotNil(t, cost)
	assert.Greater(t, cost.TotalCost, 0.0)
	assert.Greater(t, cost.Duration, time.Duration(0))

	// Stop tracking
	finalCost := tracker.StopTracking(env.ID)
	require.NotNil(t, finalCost)
	assert.Greater(t, finalCost.TotalCost, 0.0)

	// Cleanup
	err = manager.CleanupEnvironment(env.ID)
	require.NoError(t, err)
}

// Helper functions

func isDockerAvailable() bool {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer client.Close()

	_, err = client.Ping(context.Background())
	return err == nil
}

func waitForEnvironmentReady(t *testing.T, manager *TestEnvironmentManager, envID string, timeout time.Duration) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("Timeout waiting for environment %s to be ready", envID)
		case <-ticker.C:
			env, err := manager.GetEnvironment(envID)
			require.NoError(t, err)

			if env.Status == EnvironmentStatusReady {
				return
			} else if env.Status == EnvironmentStatusFailed {
				t.Fatalf("Environment %s creation failed: %s", envID, env.ErrorMessage)
			}
		}
	}
}