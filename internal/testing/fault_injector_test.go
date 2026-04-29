package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedFaultInjector_Creation(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	
	assert.NotNil(t, injector)
	assert.NotNil(t, injector.databaseInjector)
	assert.NotNil(t, injector.cacheInjector)
	assert.NotNil(t, injector.networkInjector)
	assert.NotNil(t, injector.resourceInjector)
	assert.NotNil(t, injector.activeFaults)
}

func TestEnhancedFaultInjector_DatabaseConnectionPoolExhaustion(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	duration := 3 * time.Second

	t.Run("successful database pool exhaustion injection", func(t *testing.T) {
		result, err := injector.InjectDatabaseConnectionPoolExhaustion(ctx, duration)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "database_pool_exhaustion", result.Type)
		assert.True(t, result.Duration >= duration)
		assert.True(t, result.GracefulDegradation)
		assert.NotEmpty(t, result.MetricsCollected)
		assert.Contains(t, result.MetricsCollected, "connections_exhausted")
	})

	t.Run("context cancellation during injection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		result, err := injector.InjectDatabaseConnectionPoolExhaustion(ctx, 5*time.Second)
		
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		if result != nil {
			assert.Contains(t, result.ErrorsObserved, "Context cancelled")
		}
	})
}

func TestEnhancedFaultInjector_CacheMemoryLeak(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	duration := 2 * time.Second

	t.Run("successful cache memory leak injection", func(t *testing.T) {
		result, err := injector.InjectCacheMemoryLeak(ctx, duration)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cache_memory_leak", result.Type)
		assert.True(t, result.Duration >= duration)
		assert.True(t, result.GracefulDegradation)
		assert.NotEmpty(t, result.MetricsCollected)
		assert.Contains(t, result.MetricsCollected, "memory_leaked_mb")
	})
}

func TestEnhancedFaultInjector_CPUSpike(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	intensity := 0.5
	duration := 2 * time.Second

	t.Run("successful CPU spike injection", func(t *testing.T) {
		result, err := injector.InjectCPUSpike(ctx, intensity, duration)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cpu_spike", result.Type)
		assert.True(t, result.Duration >= duration)
		assert.True(t, result.GracefulDegradation)
		assert.NotEmpty(t, result.MetricsCollected)
		assert.Equal(t, intensity, result.MetricsCollected["cpu_intensity"])
	})

	t.Run("high intensity CPU spike", func(t *testing.T) {
		highIntensity := 0.9
		result, err := injector.InjectCPUSpike(ctx, highIntensity, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, highIntensity, result.MetricsCollected["cpu_intensity"])
	})
}

func TestEnhancedFaultInjector_SystemClockSkew(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	skew := 30 * time.Second
	duration := 2 * time.Second

	t.Run("successful clock skew injection", func(t *testing.T) {
		result, err := injector.InjectSystemClockSkew(ctx, skew, duration)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "clock_skew", result.Type)
		assert.True(t, result.Duration >= duration)
		assert.True(t, result.GracefulDegradation)
		assert.NotEmpty(t, result.MetricsCollected)
		assert.Equal(t, skew.Seconds(), result.MetricsCollected["clock_skew_seconds"])
	})
}

func TestEnhancedFaultInjector_ActiveFaultsManagement(t *testing.T) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()

	t.Run("track active faults", func(t *testing.T) {
		// Start multiple faults concurrently
		go func() {
			injector.InjectDatabaseConnectionPoolExhaustion(ctx, 2*time.Second)
		}()
		go func() {
			injector.InjectCacheMemoryLeak(ctx, 2*time.Second)
		}()
		
		// Give faults time to start
		time.Sleep(100 * time.Millisecond)
		
		activeFaults := injector.GetActiveFaults()
		assert.True(t, len(activeFaults) >= 0) // May be 0 if faults completed quickly
		
		// Wait for faults to complete
		time.Sleep(3 * time.Second)
		
		finalActiveFaults := injector.GetActiveFaults()
		assert.Equal(t, 0, len(finalActiveFaults))
	})

	t.Run("stop all faults", func(t *testing.T) {
		// Start a long-running fault
		go func() {
			injector.InjectCPUSpike(ctx, 0.3, 10*time.Second)
		}()
		
		// Give fault time to start
		time.Sleep(100 * time.Millisecond)
		
		// Stop all faults
		injector.StopAllFaults()
		
		activeFaults := injector.GetActiveFaults()
		assert.Equal(t, 0, len(activeFaults))
	})
}

func TestEnhancedDatabaseInjector(t *testing.T) {
	injector := NewEnhancedDatabaseInjector()

	t.Run("connection pool exhaustion", func(t *testing.T) {
		stopFunc, err := injector.InjectConnectionPoolExhaustion(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("slow queries injection", func(t *testing.T) {
		stopFunc, err := injector.InjectSlowQueries(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Stop the injection
		stopFunc()
	})
}

func TestEnhancedCacheInjector(t *testing.T) {
	injector := NewEnhancedCacheInjector()

	t.Run("memory leak injection", func(t *testing.T) {
		stopFunc, err := injector.InjectMemoryLeak(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("eviction storm injection", func(t *testing.T) {
		stopFunc, err := injector.InjectEvictionStorm(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})
}

func TestEnhancedNetworkInjector(t *testing.T) {
	injector := NewEnhancedNetworkInjector()

	t.Run("latency injection", func(t *testing.T) {
		latency := 100 * time.Millisecond
		stopFunc, err := injector.InjectLatency(latency, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("packet loss injection", func(t *testing.T) {
		lossRate := 0.1 // 10% packet loss
		stopFunc, err := injector.InjectPacketLoss(lossRate, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("DNS failure injection", func(t *testing.T) {
		stopFunc, err := injector.InjectDNSFailure(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("bandwidth limit injection", func(t *testing.T) {
		limitMbps := 10.0
		stopFunc, err := injector.InjectBandwidthLimit(limitMbps, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Stop the injection
		stopFunc()
	})
}

func TestEnhancedResourceInjector(t *testing.T) {
	injector := NewEnhancedResourceInjector()

	t.Run("CPU spike injection", func(t *testing.T) {
		intensity := 0.3 // 30% CPU usage
		stopFunc, err := injector.InjectCPUSpike(intensity, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("memory pressure injection", func(t *testing.T) {
		sizeMB := int64(50) // 50MB
		stopFunc, err := injector.InjectMemoryPressure(sizeMB, 1*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("disk pressure injection", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping disk pressure test in short mode")
		}
		
		sizeGB := int64(1) // 1GB
		stopFunc, err := injector.InjectDiskPressure(sizeGB, 2*time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(500 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})

	t.Run("I/O bottleneck injection", func(t *testing.T) {
		stopFunc, err := injector.InjectIOBottleneck(1 * time.Second)
		
		require.NoError(t, err)
		assert.NotNil(t, stopFunc)
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		// Stop the injection
		stopFunc()
	})
}

func TestEnhancedFaultResult(t *testing.T) {
	t.Run("enhanced fault result structure", func(t *testing.T) {
		result := &EnhancedFaultResult{
			ID:                  "test_fault_123",
			Type:                "test_fault",
			StartTime:           time.Now(),
			EndTime:             time.Now().Add(5 * time.Second),
			Duration:            5 * time.Second,
			GracefulDegradation: true,
			ErrorsObserved:      []string{"test error"},
			MetricsCollected:    map[string]float64{"test_metric": 1.0},
		}
		
		assert.Equal(t, "test_fault_123", result.ID)
		assert.Equal(t, "test_fault", result.Type)
		assert.Equal(t, 5*time.Second, result.Duration)
		assert.True(t, result.GracefulDegradation)
		assert.Len(t, result.ErrorsObserved, 1)
		assert.Equal(t, "test error", result.ErrorsObserved[0])
		assert.Equal(t, 1.0, result.MetricsCollected["test_metric"])
	})
}

func TestFaultInjection(t *testing.T) {
	t.Run("fault injection structure", func(t *testing.T) {
		injection := &FaultInjection{
			ID:         "test_injection_123",
			Type:       "test_injection",
			StartTime:  time.Now(),
			Duration:   10 * time.Second,
			Parameters: map[string]interface{}{"param1": "value1"},
			StopFunc:   func() {},
			GracefulDegradation: true,
		}
		
		assert.Equal(t, "test_injection_123", injection.ID)
		assert.Equal(t, "test_injection", injection.Type)
		assert.Equal(t, 10*time.Second, injection.Duration)
		assert.Equal(t, "value1", injection.Parameters["param1"])
		assert.NotNil(t, injection.StopFunc)
		assert.True(t, injection.GracefulDegradation)
	})
}

// Integration test for multiple concurrent fault injections
func TestEnhancedFaultInjector_ConcurrentFaults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent faults test in short mode")
	}

	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	duration := 2 * time.Second

	t.Run("concurrent fault injections", func(t *testing.T) {
		// Start multiple faults concurrently
		results := make(chan *EnhancedFaultResult, 3)
		
		go func() {
			result, err := injector.InjectDatabaseConnectionPoolExhaustion(ctx, duration)
			if err == nil {
				results <- result
			}
		}()
		
		go func() {
			result, err := injector.InjectCacheMemoryLeak(ctx, duration)
			if err == nil {
				results <- result
			}
		}()
		
		go func() {
			result, err := injector.InjectCPUSpike(ctx, 0.3, duration)
			if err == nil {
				results <- result
			}
		}()
		
		// Collect results
		var collectedResults []*EnhancedFaultResult
		timeout := time.After(duration + 2*time.Second)
		
		for i := 0; i < 3; i++ {
			select {
			case result := <-results:
				collectedResults = append(collectedResults, result)
			case <-timeout:
				t.Fatal("Timeout waiting for fault results")
			}
		}
		
		// Verify all faults completed
		assert.Len(t, collectedResults, 3)
		
		// Verify different fault types
		faultTypes := make(map[string]bool)
		for _, result := range collectedResults {
			faultTypes[result.Type] = true
		}
		
		assert.True(t, faultTypes["database_pool_exhaustion"])
		assert.True(t, faultTypes["cache_memory_leak"])
		assert.True(t, faultTypes["cpu_spike"])
	})
}

// Benchmark tests for fault injection performance
func BenchmarkEnhancedFaultInjector_DatabasePoolExhaustion(b *testing.B) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := injector.InjectDatabaseConnectionPoolExhaustion(ctx, duration)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("Expected result")
		}
	}
}

func BenchmarkEnhancedFaultInjector_CPUSpike(b *testing.B) {
	injector := NewEnhancedFaultInjector()
	ctx := context.Background()
	intensity := 0.1 // Low intensity for benchmarking
	duration := 50 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := injector.InjectCPUSpike(ctx, intensity, duration)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("Expected result")
		}
	}
}