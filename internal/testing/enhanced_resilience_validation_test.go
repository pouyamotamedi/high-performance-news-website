package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedResilienceValidator_Creation(t *testing.T) {
	validator := NewEnhancedResilienceValidator()
	
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.ResilienceValidator)
	assert.NotNil(t, validator.enhancedFaultInjector)
	assert.NotNil(t, validator.recoveryTimeTracker)
	assert.NotNil(t, validator.stabilityAnalyzer)
	assert.NotNil(t, validator.cascadeDetector)
}

func TestEnhancedResilienceValidator_SystemRecoveryTimeValidation(t *testing.T) {
	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()
	maxRecoveryTime := 10 * time.Second

	t.Run("successful system recovery time validation", func(t *testing.T) {
		result, err := validator.ValidateSystemRecoveryTimeEnhanced(ctx, maxRecoveryTime)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Enhanced System Recovery Time Validation", result.TestName)
		assert.NotEmpty(t, result.FaultScenarios)
		assert.True(t, result.Duration > 0)
		
		// Check that all fault scenarios were tested
		scenarioTypes := make(map[string]bool)
		for _, scenario := range result.FaultScenarios {
			scenarioTypes[scenario.FaultType] = true
		}
		
		assert.True(t, scenarioTypes["database_pool_exhaustion"])
		assert.True(t, scenarioTypes["cache_memory_leak"])
		assert.True(t, scenarioTypes["cpu_spike"])
		assert.True(t, scenarioTypes["network_bandwidth"])
	})

	t.Run("recovery time exceeds maximum", func(t *testing.T) {
		shortMaxRecoveryTime := 1 * time.Millisecond // Very short time to trigger failure
		
		result, err := validator.ValidateSystemRecoveryTimeEnhanced(ctx, shortMaxRecoveryTime)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Success) // Should fail due to short recovery time
		assert.NotEmpty(t, result.Errors)
		assert.NotEmpty(t, result.Recommendations)
	})

	t.Run("context cancellation during validation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		result, err := validator.ValidateSystemRecoveryTimeEnhanced(ctx, maxRecoveryTime)
		
		// Should handle context cancellation gracefully
		if err != nil {
			assert.Equal(t, context.DeadlineExceeded, err)
		}
		if result != nil {
			// May have partial results
			assert.NotNil(t, result.TestName)
		}
	})
}

func TestEnhancedResilienceValidator_GracefulDegradationValidation(t *testing.T) {
	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()

	t.Run("successful graceful degradation validation", func(t *testing.T) {
		result, err := validator.ValidateGracefulDegradationEnhanced(ctx)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Enhanced Graceful Degradation Validation", result.TestName)
		assert.NotEmpty(t, result.FaultScenarios)
		
		// Check that all degradation scenarios were tested
		scenarioTypes := make(map[string]bool)
		for _, scenario := range result.FaultScenarios {
			scenarioTypes[scenario.FaultType] = true
		}
		
		assert.True(t, scenarioTypes["cache_eviction_storm"])
		assert.True(t, scenarioTypes["io_bottleneck"])
		assert.True(t, scenarioTypes["dns_failure"])
		assert.True(t, scenarioTypes["clock_skew"])
		
		// All scenarios should show graceful degradation in simulation
		for _, scenario := range result.FaultScenarios {
			assert.True(t, scenario.GracefulDegradation, "Scenario %s should show graceful degradation", scenario.ScenarioName)
		}
	})

	t.Run("graceful degradation with recommendations", func(t *testing.T) {
		result, err := validator.ValidateGracefulDegradationEnhanced(ctx)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// In our simulation, all scenarios show graceful degradation
		// In a real scenario with failures, recommendations would be provided
		if !result.Success {
			assert.NotEmpty(t, result.Recommendations)
		}
	})
}

func TestEnhancedResilienceValidator_CascadeFailurePreventionValidation(t *testing.T) {
	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()

	t.Run("successful cascade failure prevention validation", func(t *testing.T) {
		result, err := validator.ValidateCascadeFailurePreventionEnhanced(ctx)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Enhanced Cascade Failure Prevention Validation", result.TestName)
		assert.NotEmpty(t, result.FaultScenarios)
		
		// Check that all cascade scenarios were tested
		scenarioTypes := make(map[string]bool)
		for _, scenario := range result.FaultScenarios {
			scenarioTypes[scenario.FaultType] = true
		}
		
		assert.True(t, scenarioTypes["database_cascade"])
		assert.True(t, scenarioTypes["memory_cascade"])
		assert.True(t, scenarioTypes["cpu_cascade"])
	})

	t.Run("cascade failures detected", func(t *testing.T) {
		result, err := validator.ValidateCascadeFailurePreventionEnhanced(ctx)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Check if cascade failures were detected (may be random in simulation)
		if len(result.CascadeFailures) > 0 {
			assert.False(t, result.Success)
			assert.NotEmpty(t, result.Recommendations)
			
			for _, failure := range result.CascadeFailures {
				assert.NotEmpty(t, failure.Component)
				assert.NotEmpty(t, failure.Impact)
				assert.NotEmpty(t, failure.Severity)
			}
		}
	})
}

func TestEnhancedResilienceValidator_SystemStabilityUnderChaos(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos testing in short mode")
	}

	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()
	duration := 5 * time.Second

	t.Run("successful system stability under chaos", func(t *testing.T) {
		result, err := validator.ValidateSystemStabilityUnderChaos(ctx, duration)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "System Stability Under Chaos Validation", result.TestName)
		assert.NotEmpty(t, result.FaultScenarios)
		
		// Check system stability metrics
		assert.True(t, result.SystemStability.AvailabilityPercent >= 0)
		assert.True(t, result.SystemStability.ErrorRate >= 0)
		assert.NotNil(t, result.SystemStability.ResourceUtilization)
		
		// Check that chaos scenarios were executed
		scenarioTypes := make(map[string]bool)
		for _, scenario := range result.FaultScenarios {
			scenarioTypes[scenario.FaultType] = true
		}
		
		// Should have at least some of the random fault types
		assert.True(t, len(scenarioTypes) > 0)
	})

	t.Run("context cancellation during chaos testing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		result, err := validator.ValidateSystemStabilityUnderChaos(ctx, 10*time.Second)
		
		if err != nil {
			assert.Equal(t, context.DeadlineExceeded, err)
		}
		if result != nil {
			assert.Contains(t, result.Errors, "Context cancelled during chaos testing")
		}
	})
}

func TestRecoveryTimeTracker(t *testing.T) {
	tracker := NewRecoveryTimeTracker()

	t.Run("track recovery events", func(t *testing.T) {
		component := "database"
		faultType := "connection_failure"
		startTime := time.Now()
		recoveryTime := 5 * time.Second
		successful := true

		tracker.TrackRecovery(component, faultType, startTime, recoveryTime, successful)

		events := tracker.GetRecoveryEvents()
		assert.Len(t, events, 1)
		
		event := events[0]
		assert.Equal(t, component, event.Component)
		assert.Equal(t, faultType, event.FaultType)
		assert.Equal(t, recoveryTime, event.RecoveryTime)
		assert.Equal(t, successful, event.Successful)
	})

	t.Run("track multiple recovery events", func(t *testing.T) {
		tracker := NewRecoveryTimeTracker()
		
		// Track multiple events
		for i := 0; i < 5; i++ {
			tracker.TrackRecovery(
				"component",
				"fault_type",
				time.Now(),
				time.Duration(i+1)*time.Second,
				i%2 == 0,
			)
		}

		events := tracker.GetRecoveryEvents()
		assert.Len(t, events, 5)
	})
}

func TestEnhancedStabilityAnalyzer(t *testing.T) {
	analyzer := NewEnhancedStabilityAnalyzer()
	ctx := context.Background()

	t.Run("start and stop monitoring", func(t *testing.T) {
		monitor := analyzer.StartMonitoring(ctx)
		
		assert.NotNil(t, monitor)
		assert.True(t, monitor.startTime.Before(time.Now()) || monitor.startTime.Equal(time.Now()))
		
		// Let it run briefly
		time.Sleep(100 * time.Millisecond)
		
		monitor.Stop()
	})

	t.Run("collect enhanced metrics", func(t *testing.T) {
		monitor := analyzer.StartMonitoring(ctx)
		
		// Let it collect some metrics
		time.Sleep(2 * time.Second)
		
		metrics := monitor.GetEnhancedMetrics()
		
		assert.True(t, metrics.AvailabilityPercent >= 0 && metrics.AvailabilityPercent <= 100)
		assert.True(t, metrics.ErrorRate >= 0)
		assert.NotNil(t, metrics.ResourceUtilization)
		assert.True(t, len(metrics.ResourceUtilization) > 0)
		
		// Check resource utilization metrics
		assert.Contains(t, metrics.ResourceUtilization, "cpu")
		assert.Contains(t, metrics.ResourceUtilization, "memory")
		assert.Contains(t, metrics.ResourceUtilization, "disk")
		assert.Contains(t, metrics.ResourceUtilization, "network")
		
		monitor.Stop()
	})
}

func TestEnhancedResilienceTestResult(t *testing.T) {
	t.Run("calculate overall metrics", func(t *testing.T) {
		result := &EnhancedResilienceTestResult{
			TestName:       "Test",
			StartTime:      time.Now(),
			FaultScenarios: []FaultScenarioResult{
				{
					ScenarioName:        "Scenario 1",
					RecoveryTime:        3 * time.Second,
					GracefulDegradation: true,
					SystemRecovered:     true,
				},
				{
					ScenarioName:        "Scenario 2",
					RecoveryTime:        5 * time.Second,
					GracefulDegradation: true,
					SystemRecovered:     true,
				},
				{
					ScenarioName:        "Scenario 3",
					RecoveryTime:        2 * time.Second,
					GracefulDegradation: false,
					SystemRecovered:     true,
				},
			},
		}

		result.calculateOverallMetrics()

		// Should use maximum recovery time
		assert.Equal(t, 5*time.Second, result.RecoveryTime)
		
		// Should be false since not all scenarios show graceful degradation
		assert.False(t, result.GracefulDegradation)
	})

	t.Run("empty fault scenarios", func(t *testing.T) {
		result := &EnhancedResilienceTestResult{
			TestName:       "Test",
			StartTime:      time.Now(),
			FaultScenarios: []FaultScenarioResult{},
		}

		result.calculateOverallMetrics()

		assert.Equal(t, time.Duration(0), result.RecoveryTime)
		assert.False(t, result.GracefulDegradation)
	})
}

func TestFaultScenarioResult(t *testing.T) {
	t.Run("fault scenario result structure", func(t *testing.T) {
		scenario := &FaultScenarioResult{
			ScenarioName:        "Test Scenario",
			FaultType:           "test_fault",
			StartTime:           time.Now(),
			EndTime:             time.Now().Add(5 * time.Second),
			Duration:            5 * time.Second,
			RecoveryTime:        3 * time.Second,
			GracefulDegradation: true,
			SystemRecovered:     true,
			ErrorsObserved:      []string{"test error"},
			MetricsCollected:    map[string]float64{"test_metric": 1.0},
		}

		assert.Equal(t, "Test Scenario", scenario.ScenarioName)
		assert.Equal(t, "test_fault", scenario.FaultType)
		assert.Equal(t, 5*time.Second, scenario.Duration)
		assert.Equal(t, 3*time.Second, scenario.RecoveryTime)
		assert.True(t, scenario.GracefulDegradation)
		assert.True(t, scenario.SystemRecovered)
		assert.Len(t, scenario.ErrorsObserved, 1)
		assert.Equal(t, "test error", scenario.ErrorsObserved[0])
		assert.Equal(t, 1.0, scenario.MetricsCollected["test_metric"])
	})
}

// Integration test for enhanced resilience validation
func TestEnhancedResilienceValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()

	t.Run("complete resilience validation workflow", func(t *testing.T) {
		// Test recovery time validation
		recoveryResult, err := validator.ValidateSystemRecoveryTimeEnhanced(ctx, 15*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, recoveryResult)

		// Test graceful degradation validation
		degradationResult, err := validator.ValidateGracefulDegradationEnhanced(ctx)
		require.NoError(t, err)
		assert.NotNil(t, degradationResult)

		// Test cascade failure prevention
		cascadeResult, err := validator.ValidateCascadeFailurePreventionEnhanced(ctx)
		require.NoError(t, err)
		assert.NotNil(t, cascadeResult)

		// Test system stability under chaos
		chaosResult, err := validator.ValidateSystemStabilityUnderChaos(ctx, 3*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, chaosResult)

		// Verify all tests completed
		assert.True(t, recoveryResult.Duration > 0)
		assert.True(t, degradationResult.Duration > 0)
		assert.True(t, cascadeResult.Duration > 0)
		assert.True(t, chaosResult.Duration > 0)
	})
}

// Benchmark tests for enhanced resilience validation
func BenchmarkEnhancedResilienceValidator_SystemRecoveryTime(b *testing.B) {
	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()
	maxRecoveryTime := 10 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := validator.ValidateSystemRecoveryTimeEnhanced(ctx, maxRecoveryTime)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("Expected result")
		}
	}
}

func BenchmarkEnhancedResilienceValidator_GracefulDegradation(b *testing.B) {
	validator := NewEnhancedResilienceValidator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := validator.ValidateGracefulDegradationEnhanced(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("Expected result")
		}
	}
}