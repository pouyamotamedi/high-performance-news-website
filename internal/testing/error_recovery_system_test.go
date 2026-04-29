package testing

import (
	"context"
	"testing"
	"time"
)

func TestErrorRecoverySystem(t *testing.T) {
	// Create error recovery system
	ers := NewErrorRecoverySystem()
	
	// Start the system
	if err := ers.Start(); err != nil {
		t.Fatalf("Failed to start error recovery system: %v", err)
	}
	defer ers.Stop()
	
	// Test system status
	status := ers.GetSystemStatus()
	if !status.Active {
		t.Error("Error recovery system should be active")
	}
	
	// Test database connection recovery
	t.Run("DatabaseConnectionRecovery", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_db_failure",
			Type:      FailureTypeDatabase,
			Severity:  SeverityCritical,
			Component: "database",
			Message:   "Database connection failed",
			Context: map[string]interface{}{
				"connection_pool": "primary",
			},
			DetectedAt: time.Now(),
		}
		
		err := ers.recoverDatabaseConnection(context.Background(), failure)
		if err != nil {
			t.Errorf("Database connection recovery failed: %v", err)
		}
	})
	
	// Test cache service recovery
	t.Run("CacheServiceRecovery", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_cache_failure",
			Type:      FailureTypeCache,
			Severity:  SeverityHigh,
			Component: "cache",
			Message:   "Cache service unavailable",
			Context: map[string]interface{}{
				"cache_type": "redis",
			},
			DetectedAt: time.Now(),
		}
		
		err := ers.recoverCacheService(context.Background(), failure)
		if err != nil {
			t.Errorf("Cache service recovery failed: %v", err)
		}
	})
	
	// Test test environment recovery
	t.Run("TestEnvironmentRecovery", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_env_failure",
			Type:      FailureTypeEnvironment,
			Severity:  SeverityMedium,
			Component: "test_environment",
			Message:   "Test environment unhealthy",
			Context: map[string]interface{}{
				"environment_id": "test_env_123",
			},
			DetectedAt: time.Now(),
		}
		
		err := ers.recoverTestEnvironment(context.Background(), failure)
		if err != nil {
			t.Errorf("Test environment recovery failed: %v", err)
		}
	})
	
	// Test memory pressure recovery
	t.Run("MemoryPressureRecovery", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_memory_failure",
			Type:      FailureTypeMemory,
			Severity:  SeverityHigh,
			Component: "memory",
			Message:   "High memory usage detected",
			Context: map[string]interface{}{
				"memory_usage": "85%",
			},
			DetectedAt: time.Now(),
		}
		
		err := ers.recoverMemoryPressure(context.Background(), failure)
		if err != nil {
			t.Errorf("Memory pressure recovery failed: %v", err)
		}
	})
}

func TestEmergencyProcedureManager(t *testing.T) {
	// Create emergency procedure manager
	epm := NewEmergencyProcedureManager()
	
	// Test emergency status
	status := epm.GetEmergencyStatus()
	if status.EmergencyMode {
		t.Error("Emergency mode should not be active initially")
	}
	
	// Test entering emergency mode
	t.Run("EnterEmergencyMode", func(t *testing.T) {
		err := epm.EnterEmergencyMode("Test emergency")
		if err != nil {
			t.Errorf("Failed to enter emergency mode: %v", err)
		}
		
		status := epm.GetEmergencyStatus()
		if !status.EmergencyMode {
			t.Error("Emergency mode should be active")
		}
	})
	
	// Test manual override
	t.Run("ManualOverride", func(t *testing.T) {
		override := ManualOverride{
			Component: "database",
			Action:    "disable_component",
			Reason:    "Testing manual override",
			Operator:  "test_operator",
			Duration:  5 * time.Minute,
		}
		
		err := epm.ActivateManualOverride(override)
		if err != nil {
			t.Errorf("Failed to activate manual override: %v", err)
		}
		
		status := epm.GetEmergencyStatus()
		if status.ActiveOverrides == 0 {
			t.Error("Should have active overrides")
		}
	})
	
	// Test emergency procedure trigger
	t.Run("TriggerEmergencyProcedure", func(t *testing.T) {
		event := EmergencyEvent{
			ID:        "test_emergency",
			Type:      EmergencyTriggerCriticalFailure,
			Severity:  EmergencySeverityP0,
			Component: "system",
			Message:   "Critical system failure",
			Context: map[string]interface{}{
				"component": "database",
			},
			Timestamp: time.Now(),
		}
		
		err := epm.TriggerEmergencyProcedure(event)
		if err != nil {
			t.Errorf("Failed to trigger emergency procedure: %v", err)
		}
	})
	
	// Test exiting emergency mode
	t.Run("ExitEmergencyMode", func(t *testing.T) {
		err := epm.ExitEmergencyMode()
		if err != nil {
			t.Errorf("Failed to exit emergency mode: %v", err)
		}
		
		status := epm.GetEmergencyStatus()
		if status.EmergencyMode {
			t.Error("Emergency mode should not be active")
		}
	})
}

func TestDisasterRecoveryManager(t *testing.T) {
	// Create disaster recovery manager
	drm := NewDisasterRecoveryManager()
	
	// Test available scenarios
	scenarios := drm.GetAvailableScenarios()
	if len(scenarios) == 0 {
		t.Error("Should have available disaster scenarios")
	}
	
	// Test backup locations
	locations := drm.GetBackupLocations()
	if len(locations) == 0 {
		t.Error("Should have backup locations")
	}
	
	// Test recovery plan testing
	t.Run("TestRecoveryPlan", func(t *testing.T) {
		err := drm.TestRecoveryPlan("database_recovery_plan")
		if err != nil {
			t.Errorf("Recovery plan test failed: %v", err)
		}
	})
	
	// Test disaster recovery initiation
	t.Run("InitiateRecovery", func(t *testing.T) {
		err := drm.InitiateRecovery("database_failure")
		if err != nil {
			t.Errorf("Failed to initiate disaster recovery: %v", err)
		}
		
		// Check recovery status
		status := drm.GetRecoveryStatus()
		if status == nil {
			t.Error("Recovery status should not be nil")
		} else if status.Status != RecoveryStatusInProgress {
			t.Errorf("Expected recovery status to be in progress, got: %s", status.Status)
		}
		
		// Wait a bit for recovery to progress
		time.Sleep(2 * time.Second)
		
		// Check progress
		status = drm.GetRecoveryStatus()
		if status != nil && status.Progress == 0 {
			t.Error("Recovery should have made some progress")
		}
	})
}

func TestCascadeFailurePreventor(t *testing.T) {
	// Create cascade failure preventor
	cfp := NewCascadeFailurePreventor()
	
	// Start the preventor
	ctx := context.Background()
	if err := cfp.Start(ctx); err != nil {
		t.Fatalf("Failed to start cascade failure preventor: %v", err)
	}
	defer cfp.Stop()
	
	// Test isolation status
	status := cfp.GetIsolationStatus()
	if status.ActiveBulkheads == 0 {
		t.Error("Should have active bulkheads")
	}
	
	// Test should prevent cascade
	t.Run("ShouldPreventCascade", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_cascade_failure",
			Type:      FailureTypeDatabase,
			Severity:  SeverityCritical,
			Component: "database",
			Message:   "Database failure",
			DetectedAt: time.Now(),
		}
		
		shouldPrevent := cfp.ShouldPreventCascade(failure)
		if !shouldPrevent {
			t.Error("Should prevent cascade for critical database failure")
		}
	})
	
	// Test isolate failure
	t.Run("IsolateFailure", func(t *testing.T) {
		failure := FailureEvent{
			ID:        "test_isolate_failure",
			Type:      FailureTypeDatabase,
			Severity:  SeverityCritical,
			Component: "database",
			Message:   "Database failure requiring isolation",
			DetectedAt: time.Now(),
		}
		
		err := cfp.IsolateFailure(failure)
		if err != nil {
			t.Errorf("Failed to isolate failure: %v", err)
		}
		
		// Check isolation status
		status := cfp.GetIsolationStatus()
		if status.IsolatedComponents == 0 {
			t.Error("Should have isolated components")
		}
	})
	
	// Test bulkhead execution
	t.Run("ExecuteWithBulkhead", func(t *testing.T) {
		executed := false
		action := func() error {
			executed = true
			return nil
		}
		
		err := cfp.ExecuteWithBulkhead("database", action)
		if err != nil {
			t.Errorf("Bulkhead execution failed: %v", err)
		}
		
		if !executed {
			t.Error("Action should have been executed")
		}
	})
	
	// Test rate limiting
	t.Run("CheckRateLimit", func(t *testing.T) {
		// First request should be allowed
		allowed := cfp.CheckRateLimit("api")
		if !allowed {
			t.Error("First request should be allowed")
		}
		
		// Test multiple requests to potentially hit rate limit
		allowedCount := 0
		for i := 0; i < 10; i++ {
			if cfp.CheckRateLimit("api") {
				allowedCount++
			}
		}
		
		if allowedCount == 0 {
			t.Error("At least some requests should be allowed")
		}
	})
}

func TestRetryManager(t *testing.T) {
	// Create retry manager
	rm := NewRetryManager()
	
	// Test successful retry
	t.Run("SuccessfulRetry", func(t *testing.T) {
		attempts := 0
		operation := func(ctx context.Context, failure FailureEvent) error {
			attempts++
			if attempts < 2 {
				return fmt.Errorf("temporary failure")
			}
			return nil
		}
		
		failure := FailureEvent{
			ID:   "test_retry",
			Type: FailureTypeNetwork,
		}
		
		err := rm.ExecuteWithRetry(operation, failure, 3, 10*time.Second)
		if err != nil {
			t.Errorf("Retry should have succeeded: %v", err)
		}
		
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})
	
	// Test retry with config
	t.Run("RetryWithConfig", func(t *testing.T) {
		attempts := 0
		operation := func(ctx context.Context, failure FailureEvent) error {
			attempts++
			return fmt.Errorf("persistent failure")
		}
		
		failure := FailureEvent{
			ID:   "test_retry_config",
			Type: FailureTypeNetwork,
		}
		
		config := RetryConfig{
			MaxRetries:    2,
			BaseDelay:     100 * time.Millisecond,
			MaxDelay:      1 * time.Second,
			Multiplier:    2.0,
			JitterEnabled: false,
		}
		
		result := rm.ExecuteWithConfig(operation, failure, config, 5*time.Second)
		if result.Success {
			t.Error("Retry should have failed")
		}
		
		if result.Attempts != 3 { // MaxRetries + 1
			t.Errorf("Expected 3 attempts, got %d", result.Attempts)
		}
		
		if len(result.RetryDelays) != 2 {
			t.Errorf("Expected 2 retry delays, got %d", len(result.RetryDelays))
		}
	})
	
	// Test circuit breaker
	t.Run("CircuitBreaker", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 5*time.Second)
		
		// Should allow execution initially
		if !cb.CanExecute() {
			t.Error("Circuit breaker should allow execution initially")
		}
		
		// Record failures to open circuit
		cb.RecordFailure()
		cb.RecordFailure()
		
		// Should be open now
		if cb.CanExecute() {
			t.Error("Circuit breaker should be open after failures")
		}
		
		// Record success to close circuit
		cb.RecordSuccess()
		
		// Should be closed now
		if !cb.CanExecute() {
			t.Error("Circuit breaker should be closed after success")
		}
	})
}

func TestFailureDetector(t *testing.T) {
	// Create failure detector
	fd := NewFailureDetector()
	
	// Start the detector
	ctx := context.Background()
	if err := fd.Start(ctx); err != nil {
		t.Fatalf("Failed to start failure detector: %v", err)
	}
	defer fd.Stop()
	
	// Test health check status
	status := fd.GetHealthCheckStatus()
	if len(status) == 0 {
		t.Error("Should have health checks configured")
	}
	
	// Test pending failures (initially should be empty)
	failures := fd.GetPendingFailures()
	initialFailureCount := len(failures)
	
	// Add a custom health check that will fail
	failingCheck := SystemHealthCheck{
		Name:      "test_failing_check",
		Type:      FailureTypeTest,
		CheckFunc: func() error { return fmt.Errorf("test failure") },
		Interval:  1 * time.Second,
		Timeout:   2 * time.Second,
		Enabled:   true,
	}
	
	fd.AddCustomHealthCheck(failingCheck)
	
	// Wait for health check to run and detect failure
	time.Sleep(3 * time.Second)
	
	// Check for new failures
	failures = fd.GetPendingFailures()
	if len(failures) <= initialFailureCount {
		t.Error("Should have detected new failures")
	}
	
	// Test marking failure as resolved
	if len(failures) > 0 {
		fd.MarkResolved(failures[0].ID)
		
		resolvedCount := fd.GetResolvedCount()
		if resolvedCount == 0 {
			t.Error("Should have resolved failures")
		}
	}
}

// Benchmark tests
func BenchmarkErrorRecovery(b *testing.B) {
	ers := NewErrorRecoverySystem()
	ers.Start()
	defer ers.Stop()
	
	failure := FailureEvent{
		ID:        "bench_failure",
		Type:      FailureTypeDatabase,
		Severity:  SeverityMedium,
		Component: "database",
		Message:   "Benchmark failure",
		DetectedAt: time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ers.recoverDatabaseConnection(context.Background(), failure)
	}
}

func BenchmarkBulkheadExecution(b *testing.B) {
	cfp := NewCascadeFailurePreventor()
	ctx := context.Background()
	cfp.Start(ctx)
	defer cfp.Stop()
	
	action := func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfp.ExecuteWithBulkhead("database", action)
	}
}

func BenchmarkRateLimit(b *testing.B) {
	cfp := NewCascadeFailurePreventor()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfp.CheckRateLimit("api")
	}
}