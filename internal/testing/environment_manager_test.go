package testing

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestEnvironmentManager_CreateIsolatedEnvironment(t *testing.T) {
	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Test creating an isolated environment
	env, err := manager.CreateIsolatedEnvironment("unit-tests")
	require.NoError(t, err)
	assert.NotEmpty(t, env.ID)
	assert.Equal(t, "unit-tests", env.TestSuite)
	assert.Equal(t, EnvironmentStatusCreating, env.Status)

	// Wait for environment to be ready
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for environment to be ready")
		case <-ticker.C:
			currentEnv, err := manager.GetEnvironment(env.ID)
			require.NoError(t, err)
			
			if currentEnv.Status == EnvironmentStatusReady {
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

				return
			} else if currentEnv.Status == EnvironmentStatusFailed {
				t.Fatalf("Environment creation failed: %s", currentEnv.ErrorMessage)
			}
		}
	}
}

func TestTestEnvironmentManager_MultipleEnvironments(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Create multiple environments
	env1, err := manager.CreateIsolatedEnvironment("integration-tests")
	require.NoError(t, err)

	env2, err := manager.CreateIsolatedEnvironment("performance-tests")
	require.NoError(t, err)

	// Verify they have different IDs and URLs
	assert.NotEqual(t, env1.ID, env2.ID)
	assert.NotEqual(t, env1.TestSuite, env2.TestSuite)

	// Wait for both to be ready
	waitForEnvironmentReady(t, manager, env1.ID, 2*time.Minute)
	waitForEnvironmentReady(t, manager, env2.ID, 2*time.Minute)

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
}

func TestTestEnvironmentManager_ResourceManagement(t *testing.T) {
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

	waitForEnvironmentReady(t, manager, env.ID, 2*time.Minute)

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

func TestTestEnvironmentManager_HealthMonitoring(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	env, err := manager.CreateIsolatedEnvironment("health-test")
	require.NoError(t, err)

	waitForEnvironmentReady(t, manager, env.ID, 2*time.Minute)

	// Wait for at least one health check
	time.Sleep(35 * time.Second)

	currentEnv, err := manager.GetEnvironment(env.ID)
	require.NoError(t, err)

	assert.NotZero(t, currentEnv.LastHealthCheck, "Health check should have been performed")
	assert.Equal(t, "healthy", currentEnv.HealthStatus, "Environment should be healthy")
}

func TestTestEnvironmentManager_CleanupEnvironment(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	env, err := manager.CreateIsolatedEnvironment("cleanup-test")
	require.NoError(t, err)

	waitForEnvironmentReady(t, manager, env.ID, 2*time.Minute)

	// Verify environment exists
	_, err = manager.GetEnvironment(env.ID)
	require.NoError(t, err)

	// Cleanup environment
	err = manager.CleanupEnvironment(env.ID)
	require.NoError(t, err)

	// Verify environment is removed
	_, err = manager.GetEnvironment(env.ID)
	assert.Error(t, err, "Environment should be removed after cleanup")

	// Verify containers are stopped
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer dockerClient.Close()

	if env.ContainerID != "" {
		_, err = dockerClient.ContainerInspect(context.Background(), env.ContainerID)
		assert.Error(t, err, "Database container should be removed")
	}

	if env.CacheContainerID != "" {
		_, err = dockerClient.ContainerInspect(context.Background(), env.CacheContainerID)
		assert.Error(t, err, "Cache container should be removed")
	}
}

func TestTestEnvironmentManager_ListEnvironments(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	manager, err := NewTestEnvironmentManager()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Initially should be empty
	environments := manager.ListEnvironments()
	assert.Empty(t, environments)

	// Create environments
	env1, err := manager.CreateIsolatedEnvironment("list-test-1")
	require.NoError(t, err)

	env2, err := manager.CreateIsolatedEnvironment("list-test-2")
	require.NoError(t, err)

	// Should list both environments
	environments = manager.ListEnvironments()
	assert.Len(t, environments, 2)

	envIDs := make([]string, len(environments))
	for i, env := range environments {
		envIDs[i] = env.ID
	}

	assert.Contains(t, envIDs, env1.ID)
	assert.Contains(t, envIDs, env2.ID)
}

func TestResourcePool_AllocationLimits(t *testing.T) {
	pool := &ResourcePool{
		MaxMemory:       1024 * 1024 * 1024, // 1GB
		MaxCPU:          100000,              // 1 CPU core
		MaxEnvironments: 2,
	}

	// Should be able to allocate within limits
	assert.True(t, pool.CanAllocate(512*1024*1024, 50000)) // 512MB, 50% CPU

	// Allocate resources
	allocation := ResourceAllocation{
		Memory:   512 * 1024 * 1024,
		CPUQuota: 50000,
	}
	pool.AllocateResources(allocation)

	// Should still be able to allocate more
	assert.True(t, pool.CanAllocate(512*1024*1024, 50000))

	// Allocate more resources
	pool.AllocateResources(allocation)

	// Should not be able to allocate more (would exceed limits)
	assert.False(t, pool.CanAllocate(1, 1))

	// Release resources
	pool.ReleaseResources(allocation)

	// Should be able to allocate again
	assert.True(t, pool.CanAllocate(512*1024*1024, 50000))
}

func TestResourcePool_Utilization(t *testing.T) {
	pool := &ResourcePool{
		MaxMemory:       1024 * 1024 * 1024, // 1GB
		MaxCPU:          100000,              // 1 CPU core
		MaxEnvironments: 4,
	}

	// Initial utilization should be zero
	memUtil, cpuUtil, envUtil := pool.GetUtilization()
	assert.Equal(t, 0.0, memUtil)
	assert.Equal(t, 0.0, cpuUtil)
	assert.Equal(t, 0.0, envUtil)

	// Allocate 50% of resources
	allocation := ResourceAllocation{
		Memory:   512 * 1024 * 1024, // 512MB
		CPUQuota: 50000,              // 50% CPU
	}
	pool.AllocateResources(allocation)

	memUtil, cpuUtil, envUtil = pool.GetUtilization()
	assert.Equal(t, 0.5, memUtil)
	assert.Equal(t, 0.5, cpuUtil)
	assert.Equal(t, 0.25, envUtil) // 1 out of 4 environments

	// Allocate more resources
	pool.AllocateResources(allocation)

	memUtil, cpuUtil, envUtil = pool.GetUtilization()
	assert.Equal(t, 1.0, memUtil)
	assert.Equal(t, 1.0, cpuUtil)
	assert.Equal(t, 0.5, envUtil) // 2 out of 4 environments
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

// Benchmark tests

func BenchmarkEnvironmentCreation(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker not available, skipping benchmark")
	}

	manager, err := NewTestEnvironmentManager()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Shutdown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		env, err := manager.CreateIsolatedEnvironment(fmt.Sprintf("benchmark-test-%d", i))
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

func waitForEnvironmentReadyBench(b *testing.B, manager *TestEnvironmentManager, envID string, timeout time.Duration) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			b.Fatalf("Timeout waiting for environment %s to be ready", envID)
		case <-ticker.C:
			env, err := manager.GetEnvironment(envID)
			if err != nil {
				b.Fatal(err)
			}

			if env.Status == EnvironmentStatusReady {
				return
			} else if env.Status == EnvironmentStatusFailed {
				b.Fatalf("Environment %s creation failed: %s", envID, env.ErrorMessage)
			}
		}
	}
}