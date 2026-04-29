package task25

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ErrorRecoveryExample demonstrates the comprehensive error recovery system
type ErrorRecoveryExample struct {
	errorRecovery    *ErrorRecoverySystem
	emergencyManager *EmergencyProcedureManager
	disasterRecovery *DisasterRecoveryManager
}

// NewErrorRecoveryExample creates a new example instance
func NewErrorRecoveryExample() *ErrorRecoveryExample {
	return &ErrorRecoveryExample{
		errorRecovery:    NewErrorRecoverySystem(),
		emergencyManager: NewEmergencyProcedureManager(),
		disasterRecovery: NewDisasterRecoveryManager(),
	}
}

// RunExample demonstrates the error recovery system capabilities
func (ere *ErrorRecoveryExample) RunExample() error {
	log.Println("=== Error Recovery System Example ===")
	
	// Start the error recovery system
	if err := ere.errorRecovery.Start(); err != nil {
		return fmt.Errorf("failed to start error recovery system: %w", err)
	}
	defer ere.errorRecovery.Stop()
	
	// Demonstrate different failure scenarios
	if err := ere.demonstrateFailureScenarios(); err != nil {
		return fmt.Errorf("failure scenario demonstration failed: %w", err)
	}
	
	// Demonstrate emergency procedures
	if err := ere.demonstrateEmergencyProcedures(); err != nil {
		return fmt.Errorf("emergency procedure demonstration failed: %w", err)
	}
	
	// Demonstrate disaster recovery
	if err := ere.demonstrateDisasterRecovery(); err != nil {
		return fmt.Errorf("disaster recovery demonstration failed: %w", err)
	}
	
	log.Println("=== Error Recovery System Example Completed ===")
	return nil
}

// demonstrateFailureScenarios shows different types of failure recovery
func (ere *ErrorRecoveryExample) demonstrateFailureScenarios() error {
	log.Println("\n--- Demonstrating Failure Recovery Scenarios ---")
	
	// Scenario 1: Database connection failure
	log.Println("Scenario 1: Database Connection Failure")
	dbFailure := FailureEvent{
		ID:        "demo_db_failure",
		Type:      FailureTypeDatabase,
		Severity:  SeverityCritical,
		Component: "database",
		Message:   "Database connection pool exhausted",
		Context: map[string]interface{}{
			"connection_count": 100,
			"max_connections": 100,
			"error_rate":      "95%",
		},
		DetectedAt: time.Now(),
	}
	
	// Register a custom recovery procedure for this demo
	ere.errorRecovery.RegisterRecoveryProcedure(RecoveryProcedure{
		Name:        "demo_database_recovery",
		FailureType: string(FailureTypeDatabase),
		Priority:    110, // Higher priority than default
		Action:      ere.customDatabaseRecovery,
		Timeout:     30 * time.Second,
		MaxRetries:  2,
	})
	
	// Simulate the failure and recovery
	ere.errorRecovery.handleFailure(dbFailure)
	time.Sleep(2 * time.Second) // Allow recovery to complete
	
	// Scenario 2: Memory pressure
	log.Println("\nScenario 2: Memory Pressure Recovery")
	memoryFailure := FailureEvent{
		ID:        "demo_memory_failure",
		Type:      FailureTypeMemory,
		Severity:  SeverityHigh,
		Component: "application",
		Message:   "Memory usage above 90%",
		Context: map[string]interface{}{
			"memory_usage": "92%",
			"gc_frequency": "high",
		},
		DetectedAt: time.Now(),
	}
	
	ere.errorRecovery.handleFailure(memoryFailure)
	time.Sleep(2 * time.Second) // Allow recovery to complete
	
	// Scenario 3: Cache service failure with fallback
	log.Println("\nScenario 3: Cache Service Failure with Fallback")
	cacheFailure := FailureEvent{
		ID:        "demo_cache_failure",
		Type:      FailureTypeCache,
		Severity:  SeverityHigh,
		Component: "cache",
		Message:   "Cache service unavailable",
		Context: map[string]interface{}{
			"service_status": "down",
			"last_response": "5 minutes ago",
		},
		DetectedAt: time.Now(),
	}
	
	ere.errorRecovery.handleFailure(cacheFailure)
	time.Sleep(2 * time.Second) // Allow recovery to complete
	
	return nil
}

// demonstrateEmergencyProcedures shows emergency response capabilities
func (ere *ErrorRecoveryExample) demonstrateEmergencyProcedures() error {
	log.Println("\n--- Demonstrating Emergency Procedures ---")
	
	// Scenario 1: Critical system failure requiring emergency mode
	log.Println("Scenario 1: Critical System Failure")
	criticalEvent := EmergencyEvent{
		ID:        "demo_critical_failure",
		Type:      EmergencyTriggerCriticalFailure,
		Severity:  EmergencySeverityP0,
		Component: "core_system",
		Message:   "Multiple system components failing simultaneously",
		Context: map[string]interface{}{
			"failed_components": []string{"database", "cache", "api"},
			"error_rate":        "100%",
			"user_impact":       "complete_outage",
		},
		Timestamp: time.Now(),
	}
	
	if err := ere.emergencyManager.TriggerEmergencyProcedure(criticalEvent); err != nil {
		log.Printf("Emergency procedure failed: %v", err)
	}
	
	time.Sleep(1 * time.Second)
	
	// Scenario 2: Manual override for maintenance
	log.Println("\nScenario 2: Manual Override for Maintenance")
	maintenanceOverride := ManualOverride{
		Component: "api_gateway",
		Action:    "disable_component",
		Reason:    "Scheduled maintenance window",
		Operator:  "ops_team",
		Duration:  10 * time.Minute,
	}
	
	if err := ere.emergencyManager.ActivateManualOverride(maintenanceOverride); err != nil {
		log.Printf("Manual override failed: %v", err)
	}
	
	// Show emergency status
	status := ere.emergencyManager.GetEmergencyStatus()
	log.Printf("Emergency Status: Mode=%v, Overrides=%d", 
		status.EmergencyMode, status.ActiveOverrides)
	
	// Exit emergency mode
	if err := ere.emergencyManager.ExitEmergencyMode(); err != nil {
		log.Printf("Failed to exit emergency mode: %v", err)
	}
	
	return nil
}

// demonstrateDisasterRecovery shows disaster recovery capabilities
func (ere *ErrorRecoveryExample) demonstrateDisasterRecovery() error {
	log.Println("\n--- Demonstrating Disaster Recovery ---")
	
	// Show available scenarios
	scenarios := ere.disasterRecovery.GetAvailableScenarios()
	log.Printf("Available disaster scenarios: %d", len(scenarios))
	for _, scenario := range scenarios {
		log.Printf("  - %s: %s (Severity: %s)", 
			scenario.ID, scenario.Name, scenario.Severity)
	}
	
	// Show backup locations
	locations := ere.disasterRecovery.GetBackupLocations()
	log.Printf("\nBackup locations: %d", len(locations))
	for _, location := range locations {
		log.Printf("  - %s: %s (Status: %s, Used: %.1f%%)", 
			location.ID, location.Name, location.Status, 
			float64(location.Used)/float64(location.Capacity)*100)
	}
	
	// Test a recovery plan
	log.Println("\nTesting database recovery plan...")
	if err := ere.disasterRecovery.TestRecoveryPlan("database_recovery_plan"); err != nil {
		log.Printf("Recovery plan test failed: %v", err)
	} else {
		log.Println("Recovery plan test passed")
	}
	
	// Simulate a disaster recovery scenario
	log.Println("\nSimulating database failure disaster recovery...")
	if err := ere.disasterRecovery.InitiateRecovery("database_failure"); err != nil {
		log.Printf("Failed to initiate disaster recovery: %v", err)
		return err
	}
	
	// Monitor recovery progress
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		status := ere.disasterRecovery.GetRecoveryStatus()
		if status != nil {
			log.Printf("Recovery Progress: %.1f%% (Step %d, Status: %s)", 
				status.Progress, status.CurrentStep+1, status.Status)
			
			if status.Status == RecoveryStatusCompleted {
				log.Println("Disaster recovery completed successfully!")
				break
			} else if status.Status == RecoveryStatusFailed {
				log.Printf("Disaster recovery failed with errors: %v", status.Errors)
				break
			}
		}
	}
	
	return nil
}

// customDatabaseRecovery demonstrates a custom recovery procedure
func (ere *ErrorRecoveryExample) customDatabaseRecovery(ctx context.Context, failure FailureEvent) error {
	log.Printf("Executing custom database recovery for failure: %s", failure.ID)
	
	// Step 1: Analyze failure context
	connectionCount, _ := failure.Context["connection_count"].(int)
	errorRate, _ := failure.Context["error_rate"].(string)
	
	log.Printf("Failure analysis: Connections=%d, ErrorRate=%s", connectionCount, errorRate)
	
	// Step 2: Implement custom recovery logic
	log.Println("Step 1: Killing long-running queries")
	time.Sleep(500 * time.Millisecond) // Simulate query termination
	
	log.Println("Step 2: Resetting connection pool with increased limits")
	time.Sleep(500 * time.Millisecond) // Simulate pool reset
	
	log.Println("Step 3: Enabling read-only mode temporarily")
	time.Sleep(500 * time.Millisecond) // Simulate read-only mode
	
	log.Println("Step 4: Gradually restoring write operations")
	time.Sleep(500 * time.Millisecond) // Simulate gradual restoration
	
	log.Printf("Custom database recovery completed for failure: %s", failure.ID)
	return nil
}

// GetSystemStatus returns comprehensive system status
func (ere *ErrorRecoveryExample) GetSystemStatus() SystemStatus {
	return ere.errorRecovery.GetSystemStatus()
}

// DemonstrateAdvancedFeatures shows advanced error recovery features
func (ere *ErrorRecoveryExample) DemonstrateAdvancedFeatures() error {
	log.Println("\n--- Demonstrating Advanced Features ---")
	
	// Demonstrate cascade failure prevention
	log.Println("Feature 1: Cascade Failure Prevention")
	cascadeFailure := FailureEvent{
		ID:        "demo_cascade_failure",
		Type:      FailureTypeDatabase,
		Severity:  SeverityCritical,
		Component: "primary_database",
		Message:   "Primary database cluster failure",
		Context: map[string]interface{}{
			"cluster_nodes": []string{"db1", "db2", "db3"},
			"failure_type":  "network_partition",
		},
		DetectedAt: time.Now(),
	}
	
	// Check if cascade prevention should be triggered
	cfp := ere.errorRecovery.cascadePreventor
	if cfp.ShouldPreventCascade(cascadeFailure) {
		log.Println("Cascade prevention triggered - isolating failed component")
		if err := cfp.IsolateFailure(cascadeFailure); err != nil {
			log.Printf("Cascade isolation failed: %v", err)
		} else {
			log.Println("Component successfully isolated to prevent cascade")
		}
	}
	
	// Demonstrate intelligent retry mechanisms
	log.Println("\nFeature 2: Intelligent Retry Mechanisms")
	retryManager := ere.errorRecovery.retryManager
	
	// Create a failing operation that succeeds after 2 attempts
	attempts := 0
	failingOperation := func(ctx context.Context, failure FailureEvent) error {
		attempts++
		log.Printf("Retry attempt %d for operation", attempts)
		if attempts < 3 {
			return fmt.Errorf("temporary failure (attempt %d)", attempts)
		}
		return nil
	}
	
	config := RetryConfig{
		MaxRetries:    5,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		Multiplier:    2.0,
		JitterEnabled: true,
	}
	
	result := retryManager.ExecuteWithConfig(failingOperation, cascadeFailure, config, 10*time.Second)
	log.Printf("Retry result: Success=%v, Attempts=%d, Duration=%v", 
		result.Success, result.Attempts, result.TotalDuration)
	
	// Demonstrate health monitoring
	log.Println("\nFeature 3: Infrastructure Health Monitoring")
	healthMonitor := ere.errorRecovery.healthMonitor
	healthScore := healthMonitor.GetOverallHealthScore()
	log.Printf("Overall system health score: %.1f%%", healthScore)
	
	return nil
}