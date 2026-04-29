package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestResourcePool_NegativeProtection(t *testing.T) {
	pool := &ResourcePool{
		MaxMemory:       1024 * 1024 * 1024, // 1GB
		MaxCPU:          100000,              // 1 CPU core
		MaxEnvironments: 2,
	}

	// Allocate some resources
	allocation := ResourceAllocation{
		Memory:   512 * 1024 * 1024,
		CPUQuota: 50000,
	}
	pool.AllocateResources(allocation)

	// Release more than allocated (should not go negative)
	largerAllocation := ResourceAllocation{
		Memory:   1024 * 1024 * 1024, // 1GB (more than allocated)
		CPUQuota: 100000,             // 100% CPU (more than allocated)
	}
	pool.ReleaseResources(largerAllocation)

	// Utilization should not be negative
	memUtil, cpuUtil, envUtil := pool.GetUtilization()
	assert.GreaterOrEqual(t, memUtil, 0.0)
	assert.GreaterOrEqual(t, cpuUtil, 0.0)
	assert.GreaterOrEqual(t, envUtil, 0.0)
}

func TestResourcePool_ConcurrentAccess(t *testing.T) {
	pool := &ResourcePool{
		MaxMemory:       2048 * 1024 * 1024, // 2GB
		MaxCPU:          200000,              // 2 CPU cores
		MaxEnvironments: 10,
	}

	allocation := ResourceAllocation{
		Memory:   256 * 1024 * 1024, // 256MB
		CPUQuota: 25000,              // 25% CPU
	}

	// Test concurrent allocations
	done := make(chan bool, 4)
	
	for i := 0; i < 4; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Each goroutine allocates and releases resources
			for j := 0; j < 10; j++ {
				if pool.CanAllocate(allocation.Memory, allocation.CPUQuota) {
					pool.AllocateResources(allocation)
					pool.ReleaseResources(allocation)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// Final utilization should be zero
	memUtil, cpuUtil, envUtil := pool.GetUtilization()
	assert.Equal(t, 0.0, memUtil)
	assert.Equal(t, 0.0, cpuUtil)
	assert.Equal(t, 0.0, envUtil)
}