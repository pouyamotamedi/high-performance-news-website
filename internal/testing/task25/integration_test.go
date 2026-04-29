package task25

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestErrorRecoverySystemIntegration(t *testing.T) {
	// Create the example system
	example := NewErrorRecoveryExample()
	
	// Test basic system startup
	t.Run("SystemStartup", func(t *testing.T) {
		if err := example.errorRecovery.Start(); err != nil {
			t.Fatalf("Failed to start error recovery system: %v", err)
		}
		defer example.errorRecovery.Stop()
		
		// Verify system is active
		status := example.GetSystemStatus()
		if !status.Active {
			t.Error("Error recovery system should be active")
		}
		
		// Wait a moment for components to initialize
		time.Sleep(1 * time.Second)
		
		// Check health score
		if status.HealthScore < 0 || status.HealthScore > 100 {
			t.Errorf("Health score should be 0-100, got: %.2f", status.HealthScore)
		}
	})
	
	// Test emergency procedures
	t.Run("EmergencyProcedures", func(t *testing.T) {
		epm := example.emergencyManager
		
		// Test emergency mode activation
		err := epm.EnterEmergencyMode("Integration test emergency")
		if err != nil {
			t.Errorf("Failed to enter emergency mode: %v", err)
		}
		
		status := epm.GetEmergencyStatus()
		if !status.EmergencyMode {
			t.Error("Emergency mode should be active")
		}
		
		// Test manual override
		override := ManualOverride{
			Component: "test_component",
			Action:    "disable_component",
			Reason:    "Integration test",
			Operator:  "test_system",
			Duration:  1 * time.Minute,
		}
		
		err = epm.ActivateManualOverride(override)
		if err != nil {
			t.Errorf("Failed to activate manual override: %v", err)
		}
		
		// Exit emergency mode
		err = epm.ExitEmergencyMode()
		if err != nil {
			t.Errorf("Failed to exit emergency mode: %v", err)
		}
	})
	
	// Test disaster recovery
	t.Run("DisasterRecovery", func(t *testing.T) {
		drm := example.disasterRecovery
		
		// Test available scenarios
		scenarios := drm.GetAvailableScenarios()
		if len(scenarios) == 0 {
			t.Error("Should have disaster recovery scenarios")
		}
		
		// Test backup locations
		locations := drm.GetBackupLocations()
		if len(locations) == 0 {
			t.Error("Should have backup locations")
		}
		
		// Test recovery plan
		err := drm.TestRecoveryPlan("database_recovery_plan")
		if err != nil {
			t.Errorf("Recovery plan test failed: %v", err)
		}
	})
}

func TestAdvancedFeatures(t *testing.T) {
	example := NewErrorRecoveryExample()
	
	// Start the system
	if err := example.errorRecovery.Start(); err != nil {
		t.Fatalf("Failed to start error recovery system: %v", err)
	}
	defer example.errorRecovery.Stop()
	
	// Test cascade failure prevention
	t.Run("CascadeFailurePrevention", func(t *testing.T) {
		cfp := example.errorRecovery.cascadePreventor
		
		// Start cascade preventor
		if err := cfp.Start(example.errorRecovery.ctx); err != nil {
			t.Fatalf("Failed to start cascade preventor: %v", err)
		}
		defer cfp.Stop()
		
		// Test critical failure that should trigger cascade prevention
		failure := FailureEvent{
			ID:        "test_cascade",
			Type:      FailureTypeDatabase,
			Severity:  SeverityCritical,
			Component: "database",
			Message:   "Critical database failure",
			DetectedAt: time.Now(),
		}
		
		shouldPrevent := cfp.ShouldPreventCascade(failure)
		if !shouldPrevent {
			t.Error("Should prevent cascade for critical database failure")
		}
		
		// Test isolation
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
	
	// Test retry mechanisms
	t.Run("RetryMechanisms", func(t *testing.T) {
		rm := example.errorRecovery.retryManager
		
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
		
		err := rm.ExecuteWithRetry(operation, failure, 3, 5*time.Second)
		if err != nil {
			t.Errorf("Retry should have succeeded: %v", err)
		}
		
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})
	
	// Test health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		ihm := example.errorRecovery.healthMonitor
		
		// Wait for initial metrics collection
		time.Sleep(1 * time.Second)
		
		score := ihm.GetOverallHealthScore()
		if score < 0 || score > 100 {
			t.Errorf("Health score should be 0-100, got: %.2f", score)
		}
	})
}

// TestQuickIntegration runs a fast integration test
func TestQuickIntegration(t *testing.T) {
	// Create components
	ers := NewErrorRecoverySystem()
	epm := NewEmergencyProcedureManager()
	
	// Test basic functionality without long delays
	t.Run("QuickSystemTest", func(t *testing.T) {
		// Start system
		if err := ers.Start(); err != nil {
			t.Fatalf("Failed to start system: %v", err)
		}
		defer ers.Stop()
		
		// Check status
		status := ers.GetSystemStatus()
		if !status.Active {
			t.Error("System should be active")
		}
		
		// Test emergency mode
		err := epm.EnterEmergencyMode("Quick test")
		if err != nil {
			t.Errorf("Failed to enter emergency mode: %v", err)
		}
		
		emergencyStatus := epm.GetEmergencyStatus()
		if !emergencyStatus.EmergencyMode {
			t.Error("Emergency mode should be active")
		}
		
		// Exit emergency mode
		err = epm.ExitEmergencyMode()
		if err != nil {
			t.Errorf("Failed to exit emergency mode: %v", err)
		}
	})
}