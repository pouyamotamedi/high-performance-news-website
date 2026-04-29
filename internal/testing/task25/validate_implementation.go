package task25

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ValidateImplementation performs a quick validation of the error recovery system
func ValidateImplementation() error {
	log.Println("=== Validating Error Recovery Implementation ===")
	
	// Test 1: Error Recovery System
	log.Println("Test 1: Error Recovery System")
	ers := NewErrorRecoverySystem()
	if err := ers.Start(); err != nil {
		return fmt.Errorf("failed to start error recovery system: %w", err)
	}
	
	// Quick status check
	status := ers.GetSystemStatus()
	if !status.Active {
		ers.Stop()
		return fmt.Errorf("error recovery system not active")
	}
	log.Printf("✓ Error Recovery System active (Health: %.1f%%)", status.HealthScore)
	ers.Stop()
	
	// Test 2: Emergency Procedure Manager
	log.Println("Test 2: Emergency Procedure Manager")
	epm := NewEmergencyProcedureManager()
	
	// Test emergency mode
	if err := epm.EnterEmergencyMode("Validation test"); err != nil {
		return fmt.Errorf("failed to enter emergency mode: %w", err)
	}
	
	emergencyStatus := epm.GetEmergencyStatus()
	if !emergencyStatus.EmergencyMode {
		return fmt.Errorf("emergency mode not activated")
	}
	
	if err := epm.ExitEmergencyMode(); err != nil {
		return fmt.Errorf("failed to exit emergency mode: %w", err)
	}
	log.Println("✓ Emergency Procedure Manager working")
	
	// Test 3: Disaster Recovery Manager
	log.Println("Test 3: Disaster Recovery Manager")
	drm := NewDisasterRecoveryManager()
	
	scenarios := drm.GetAvailableScenarios()
	if len(scenarios) == 0 {
		return fmt.Errorf("no disaster recovery scenarios available")
	}
	
	locations := drm.GetBackupLocations()
	if len(locations) == 0 {
		return fmt.Errorf("no backup locations configured")
	}
	
	// Test recovery plan
	if err := drm.TestRecoveryPlan("database_recovery_plan"); err != nil {
		return fmt.Errorf("recovery plan test failed: %w", err)
	}
	log.Printf("✓ Disaster Recovery Manager working (%d scenarios, %d backup locations)", 
		len(scenarios), len(locations))
	
	// Test 4: Cascade Failure Preventor
	log.Println("Test 4: Cascade Failure Preventor")
	cfp := NewCascadeFailurePreventor()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := cfp.Start(ctx); err != nil {
		return fmt.Errorf("failed to start cascade failure preventor: %w", err)
	}
	
	// Test cascade prevention logic
	failure := FailureEvent{
		ID:        "validation_failure",
		Type:      FailureTypeDatabase,
		Severity:  SeverityCritical,
		Component: "database",
		Message:   "Validation test failure",
		DetectedAt: time.Now(),
	}
	
	shouldPrevent := cfp.ShouldPreventCascade(failure)
	if !shouldPrevent {
		cfp.Stop()
		return fmt.Errorf("cascade prevention not triggered for critical failure")
	}
	
	isolationStatus := cfp.GetIsolationStatus()
	log.Printf("✓ Cascade Failure Preventor working (%d bulkheads, %d rate limiters)", 
		isolationStatus.ActiveBulkheads, isolationStatus.ActiveRateLimiters)
	cfp.Stop()
	
	// Test 5: Retry Manager
	log.Println("Test 5: Retry Manager")
	rm := NewRetryManager()
	
	attempts := 0
	testOperation := func(ctx context.Context, failure FailureEvent) error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	}
	
	err := rm.ExecuteWithRetry(testOperation, failure, 5, 10*time.Second)
	if err != nil {
		return fmt.Errorf("retry mechanism failed: %w", err)
	}
	
	if attempts != 3 {
		return fmt.Errorf("expected 3 attempts, got %d", attempts)
	}
	log.Println("✓ Retry Manager working")
	
	// Test 6: Failure Detector
	log.Println("Test 6: Failure Detector")
	fd := NewFailureDetector()
	
	if err := fd.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start failure detector: %w", err)
	}
	
	healthStatus := fd.GetHealthCheckStatus()
	if len(healthStatus) == 0 {
		fd.Stop()
		return fmt.Errorf("no health checks configured")
	}
	
	log.Printf("✓ Failure Detector working (%d health checks)", len(healthStatus))
	fd.Stop()
	
	// Test 7: Infrastructure Health Monitor
	log.Println("Test 7: Infrastructure Health Monitor")
	ihm := NewInfrastructureHealthMonitor()
	
	if err := ihm.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}
	
	// Give it a moment to collect initial metrics
	time.Sleep(500 * time.Millisecond)
	
	healthScore := ihm.GetOverallHealthScore()
	if healthScore < 0 || healthScore > 100 {
		ihm.Stop()
		return fmt.Errorf("invalid health score: %.2f", healthScore)
	}
	
	log.Printf("✓ Infrastructure Health Monitor working (Score: %.1f%%)", healthScore)
	ihm.Stop()
	
	log.Println("=== All Components Validated Successfully ===")
	return nil
}

// ValidateAdvancedFeatures tests advanced error recovery features
func ValidateAdvancedFeatures() error {
	log.Println("=== Validating Advanced Features ===")
	
	// Test bulkhead execution
	log.Println("Test 1: Bulkhead Execution")
	cfp := NewCascadeFailurePreventor()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := cfp.Start(ctx); err != nil {
		return fmt.Errorf("failed to start cascade preventor: %w", err)
	}
	defer cfp.Stop()
	
	executed := false
	testAction := func() error {
		executed = true
		return nil
	}
	
	if err := cfp.ExecuteWithBulkhead("database", testAction); err != nil {
		return fmt.Errorf("bulkhead execution failed: %w", err)
	}
	
	if !executed {
		return fmt.Errorf("action not executed in bulkhead")
	}
	log.Println("✓ Bulkhead execution working")
	
	// Test rate limiting
	log.Println("Test 2: Rate Limiting")
	allowed := cfp.CheckRateLimit("api")
	if !allowed {
		return fmt.Errorf("rate limit should allow first request")
	}
	log.Println("✓ Rate limiting working")
	
	// Test circuit breaker
	log.Println("Test 3: Circuit Breaker")
	cb := NewCircuitBreaker(2, 5*time.Second)
	
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker should be closed initially")
	}
	
	// Trigger failures to open circuit
	cb.RecordFailure()
	cb.RecordFailure()
	
	if cb.CanExecute() {
		return fmt.Errorf("circuit breaker should be open after failures")
	}
	
	// Record success to close circuit
	cb.RecordSuccess()
	
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker should be closed after success")
	}
	log.Println("✓ Circuit breaker working")
	
	log.Println("=== Advanced Features Validated Successfully ===")
	return nil
}

// ValidateIntegration tests component integration
func ValidateIntegration() error {
	log.Println("=== Validating Component Integration ===")
	
	// Create integrated system
	ers := NewErrorRecoverySystem()
	
	// Start system
	if err := ers.Start(); err != nil {
		return fmt.Errorf("failed to start integrated system: %w", err)
	}
	defer ers.Stop()
	
	// Test failure handling integration
	log.Println("Test 1: Integrated Failure Handling")
	
	// Create a test failure
	failure := FailureEvent{
		ID:        "integration_test_failure",
		Type:      FailureTypeCache,
		Severity:  SeverityHigh,
		Component: "cache",
		Message:   "Integration test cache failure",
		Context: map[string]interface{}{
			"test_mode": true,
		},
		DetectedAt: time.Now(),
	}
	
	// Test recovery procedure execution
	err := ers.recoverCacheService(context.Background(), failure)
	if err != nil {
		return fmt.Errorf("integrated cache recovery failed: %w", err)
	}
	log.Println("✓ Integrated failure handling working")
	
	// Test system status reporting
	log.Println("Test 2: System Status Reporting")
	status := ers.GetSystemStatus()
	
	if !status.Active {
		return fmt.Errorf("system should be active")
	}
	
	if status.HealthScore < 0 || status.HealthScore > 100 {
		return fmt.Errorf("invalid health score: %.2f", status.HealthScore)
	}
	
	log.Printf("✓ System status reporting working (Health: %.1f%%)", status.HealthScore)
	
	log.Println("=== Component Integration Validated Successfully ===")
	return nil
}