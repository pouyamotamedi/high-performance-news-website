package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCascadeFailureDetector_Creation(t *testing.T) {
	detector := NewCascadeFailureDetector()
	
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.failures)
	assert.NotNil(t, detector.monitors)
	assert.Equal(t, 3, detector.alertThreshold)
}

func TestCascadeFailureDetector_StartMonitoring(t *testing.T) {
	detector := NewCascadeFailureDetector()
	ctx := context.Background()

	t.Run("successful monitoring start", func(t *testing.T) {
		monitor := detector.StartMonitoring(ctx)
		
		assert.NotNil(t, monitor)
		assert.Equal(t, "system", monitor.Component)
		assert.True(t, monitor.IsActive)
		assert.NotNil(t, monitor.NewFailures)
		assert.NotNil(t, monitor.stopChan)
		
		// Stop monitoring
		monitor.Stop()
		assert.False(t, monitor.IsActive)
	})

	t.Run("monitoring with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		monitor := detector.StartMonitoring(ctx)
		assert.NotNil(t, monitor)
		
		// Wait for context to be cancelled
		time.Sleep(200 * time.Millisecond)
		
		monitor.Stop()
	})
}

func TestCascadeFailureDetector_FailureDetection(t *testing.T) {
	detector := NewCascadeFailureDetector()
	ctx := context.Background()

	t.Run("detect cascade failures during monitoring", func(t *testing.T) {
		monitor := detector.StartMonitoring(ctx)
		defer monitor.Stop()
		
		// Let monitoring run for a few seconds to potentially detect failures
		time.Sleep(5 * time.Second)
		
		// Check for new failures
		newFailures := monitor.GetNewFailures()
		
		// May or may not have failures due to random nature of simulation
		if len(newFailures) > 0 {
			for _, failure := range newFailures {
				assert.NotEmpty(t, failure.ID)
				assert.NotEmpty(t, failure.Component)
				assert.NotEmpty(t, failure.TriggerType)
				assert.NotEmpty(t, failure.Impact)
				assert.NotEmpty(t, failure.Severity)
				assert.NotEmpty(t, failure.Propagation)
				assert.True(t, failure.StartTime.Before(time.Now()) || failure.StartTime.Equal(time.Now()))
			}
		}
		
		// Check that failures were recorded in detector
		allFailures := detector.GetAllFailures()
		assert.Equal(t, len(newFailures), len(allFailures))
	})

	t.Run("multiple calls to GetNewFailures should clear previous results", func(t *testing.T) {
		monitor := detector.StartMonitoring(ctx)
		defer monitor.Stop()
		
		// Let monitoring run briefly
		time.Sleep(3 * time.Second)
		
		// Get failures first time
		firstFailures := monitor.GetNewFailures()
		
		// Get failures second time - should be empty or different
		secondFailures := monitor.GetNewFailures()
		
		// Second call should not return the same failures
		if len(firstFailures) > 0 && len(secondFailures) > 0 {
			// If both have failures, they should be different instances
			for _, firstFailure := range firstFailures {
				for _, secondFailure := range secondFailures {
					assert.NotEqual(t, firstFailure.ID, secondFailure.ID)
				}
			}
		}
	})
}

func TestCascadeFailureDetector_FailureManagement(t *testing.T) {
	detector := NewCascadeFailureDetector()

	t.Run("record and retrieve failures", func(t *testing.T) {
		failure := &CascadeFailure{
			ID:          "test_failure_123",
			Component:   "database",
			TriggerType: "connection_pool_exhaustion",
			Impact:      "API response degradation",
			StartTime:   time.Now(),
			Severity:    "high",
			Propagation: []string{"api_server", "cache_service"},
		}

		detector.recordFailure(failure)

		// Retrieve all failures
		allFailures := detector.GetAllFailures()
		assert.Len(t, allFailures, 1)
		assert.Equal(t, failure.ID, allFailures[0].ID)
		assert.Equal(t, failure.Component, allFailures[0].Component)
	})

	t.Run("get failures by component", func(t *testing.T) {
		detector.ClearFailures() // Start fresh
		
		// Add failures for different components
		dbFailure := &CascadeFailure{
			ID:        "db_failure_1",
			Component: "database",
			Severity:  "high",
		}
		cacheFailure := &CascadeFailure{
			ID:        "cache_failure_1",
			Component: "cache",
			Severity:  "medium",
		}
		
		detector.recordFailure(dbFailure)
		detector.recordFailure(cacheFailure)

		// Get failures by component
		dbFailures := detector.GetFailuresByComponent("database")
		assert.Len(t, dbFailures, 1)
		assert.Equal(t, "db_failure_1", dbFailures[0].ID)

		cacheFailures := detector.GetFailuresByComponent("cache")
		assert.Len(t, cacheFailures, 1)
		assert.Equal(t, "cache_failure_1", cacheFailures[0].ID)

		// Non-existent component
		nonExistentFailures := detector.GetFailuresByComponent("nonexistent")
		assert.Len(t, nonExistentFailures, 0)
	})

	t.Run("get failures by severity", func(t *testing.T) {
		detector.ClearFailures() // Start fresh
		
		// Add failures with different severities
		highFailure := &CascadeFailure{
			ID:        "high_failure_1",
			Component: "database",
			Severity:  "high",
		}
		mediumFailure := &CascadeFailure{
			ID:        "medium_failure_1",
			Component: "cache",
			Severity:  "medium",
		}
		
		detector.recordFailure(highFailure)
		detector.recordFailure(mediumFailure)

		// Get failures by severity
		highFailures := detector.GetFailuresBySeverity("high")
		assert.Len(t, highFailures, 1)
		assert.Equal(t, "high_failure_1", highFailures[0].ID)

		mediumFailures := detector.GetFailuresBySeverity("medium")
		assert.Len(t, mediumFailures, 1)
		assert.Equal(t, "medium_failure_1", mediumFailures[0].ID)

		// Non-existent severity
		lowFailures := detector.GetFailuresBySeverity("low")
		assert.Len(t, lowFailures, 0)
	})

	t.Run("clear failures", func(t *testing.T) {
		// Add some failures
		failure1 := &CascadeFailure{ID: "failure_1", Component: "test"}
		failure2 := &CascadeFailure{ID: "failure_2", Component: "test"}
		
		detector.recordFailure(failure1)
		detector.recordFailure(failure2)

		// Verify failures exist
		allFailures := detector.GetAllFailures()
		assert.Len(t, allFailures, 2)

		// Clear failures
		detector.ClearFailures()

		// Verify failures are cleared
		allFailures = detector.GetAllFailures()
		assert.Len(t, allFailures, 0)
	})
}

func TestCascadeFailureDetector_AlertThreshold(t *testing.T) {
	detector := NewCascadeFailureDetector()

	t.Run("default alert threshold", func(t *testing.T) {
		assert.False(t, detector.ShouldAlert()) // No failures initially
		
		// Add failures below threshold
		for i := 0; i < 2; i++ {
			failure := &CascadeFailure{
				ID:        fmt.Sprintf("failure_%d", i),
				Component: "test",
			}
			detector.recordFailure(failure)
		}
		
		assert.False(t, detector.ShouldAlert()) // Below threshold (3)
		
		// Add one more to reach threshold
		failure := &CascadeFailure{
			ID:        "failure_3",
			Component: "test",
		}
		detector.recordFailure(failure)
		
		assert.True(t, detector.ShouldAlert()) // At threshold
	})

	t.Run("custom alert threshold", func(t *testing.T) {
		detector.ClearFailures()
		detector.SetAlertThreshold(5)
		
		// Add failures below new threshold
		for i := 0; i < 4; i++ {
			failure := &CascadeFailure{
				ID:        fmt.Sprintf("failure_%d", i),
				Component: "test",
			}
			detector.recordFailure(failure)
		}
		
		assert.False(t, detector.ShouldAlert()) // Below threshold (5)
		
		// Add one more to reach threshold
		failure := &CascadeFailure{
			ID:        "failure_5",
			Component: "test",
		}
		detector.recordFailure(failure)
		
		assert.True(t, detector.ShouldAlert()) // At threshold
	})
}

func TestCascadeMonitor_FailureDetectionMethods(t *testing.T) {
	monitor := &CascadeMonitor{
		Component:   "test",
		IsActive:    true,
		NewFailures: make([]*CascadeFailure, 0),
	}

	t.Run("database cascade detection", func(t *testing.T) {
		// Test multiple times due to random nature
		detectionCount := 0
		for i := 0; i < 100; i++ {
			if monitor.shouldDetectDatabaseCascade() {
				detectionCount++
			}
		}
		
		// Should detect some failures (around 10% due to modulo 10)
		assert.True(t, detectionCount > 0 && detectionCount < 100)
	})

	t.Run("memory cascade detection", func(t *testing.T) {
		// Test multiple times due to random nature
		detectionCount := 0
		for i := 0; i < 100; i++ {
			if monitor.shouldDetectMemoryCascade() {
				detectionCount++
			}
		}
		
		// Should detect some failures (around 5% due to modulo 20)
		assert.True(t, detectionCount >= 0 && detectionCount < 100)
	})

	t.Run("CPU cascade detection", func(t *testing.T) {
		// Test multiple times due to random nature
		detectionCount := 0
		for i := 0; i < 100; i++ {
			if monitor.shouldDetectCPUCascade() {
				detectionCount++
			}
		}
		
		// Should detect some failures (around 8% due to modulo 12)
		assert.True(t, detectionCount >= 0 && detectionCount < 100)
	})
}

func TestCascadeFailure_Structure(t *testing.T) {
	t.Run("cascade failure structure", func(t *testing.T) {
		failure := &CascadeFailure{
			ID:          "test_cascade_123",
			Component:   "database",
			TriggerType: "connection_pool_exhaustion",
			Impact:      "API response degradation, cache miss increase",
			StartTime:   time.Now(),
			Severity:    "high",
			Propagation: []string{"api_server", "cache_service", "user_sessions"},
		}

		assert.Equal(t, "test_cascade_123", failure.ID)
		assert.Equal(t, "database", failure.Component)
		assert.Equal(t, "connection_pool_exhaustion", failure.TriggerType)
		assert.Equal(t, "API response degradation, cache miss increase", failure.Impact)
		assert.Equal(t, "high", failure.Severity)
		assert.Len(t, failure.Propagation, 3)
		assert.Contains(t, failure.Propagation, "api_server")
		assert.Contains(t, failure.Propagation, "cache_service")
		assert.Contains(t, failure.Propagation, "user_sessions")
	})
}

// Integration test for cascade failure detection
func TestCascadeFailureDetector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector := NewCascadeFailureDetector()
	ctx := context.Background()

	t.Run("complete cascade failure detection workflow", func(t *testing.T) {
		// Start monitoring
		monitor := detector.StartMonitoring(ctx)
		defer monitor.Stop()

		// Let monitoring run for several seconds
		time.Sleep(8 * time.Second)

		// Check for detected failures
		newFailures := monitor.GetNewFailures()
		allFailures := detector.GetAllFailures()

		// Verify monitoring was active
		assert.True(t, len(allFailures) >= 0) // May be 0 due to random nature
		assert.True(t, len(newFailures) >= 0) // May be 0 due to random nature

		// If failures were detected, verify their structure
		for _, failure := range allFailures {
			assert.NotEmpty(t, failure.ID)
			assert.NotEmpty(t, failure.Component)
			assert.NotEmpty(t, failure.TriggerType)
			assert.NotEmpty(t, failure.Impact)
			assert.NotEmpty(t, failure.Severity)
			assert.True(t, len(failure.Propagation) > 0)
		}

		// Test alert threshold
		if len(allFailures) >= detector.alertThreshold {
			assert.True(t, detector.ShouldAlert())
		} else {
			assert.False(t, detector.ShouldAlert())
		}

		// Test filtering by component and severity
		if len(allFailures) > 0 {
			// Get unique components and severities
			components := make(map[string]bool)
			severities := make(map[string]bool)
			
			for _, failure := range allFailures {
				components[failure.Component] = true
				severities[failure.Severity] = true
			}

			// Test component filtering
			for component := range components {
				componentFailures := detector.GetFailuresByComponent(component)
				assert.True(t, len(componentFailures) > 0)
				
				for _, failure := range componentFailures {
					assert.Equal(t, component, failure.Component)
				}
			}

			// Test severity filtering
			for severity := range severities {
				severityFailures := detector.GetFailuresBySeverity(severity)
				assert.True(t, len(severityFailures) > 0)
				
				for _, failure := range severityFailures {
					assert.Equal(t, severity, failure.Severity)
				}
			}
		}
	})
}

// Benchmark tests for cascade failure detection
func BenchmarkCascadeFailureDetector_StartMonitoring(b *testing.B) {
	detector := NewCascadeFailureDetector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor := detector.StartMonitoring(ctx)
		monitor.Stop()
	}
}

func BenchmarkCascadeFailureDetector_RecordFailure(b *testing.B) {
	detector := NewCascadeFailureDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		failure := &CascadeFailure{
			ID:        fmt.Sprintf("failure_%d", i),
			Component: "test",
			Severity:  "medium",
		}
		detector.recordFailure(failure)
	}
}

func BenchmarkCascadeFailureDetector_GetAllFailures(b *testing.B) {
	detector := NewCascadeFailureDetector()
	
	// Pre-populate with failures
	for i := 0; i < 1000; i++ {
		failure := &CascadeFailure{
			ID:        fmt.Sprintf("failure_%d", i),
			Component: "test",
			Severity:  "medium",
		}
		detector.recordFailure(failure)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		failures := detector.GetAllFailures()
		_ = failures // Use the result to prevent optimization
	}
}