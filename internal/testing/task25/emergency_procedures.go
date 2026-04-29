package task25

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// EmergencyProcedureManager handles emergency procedures and manual overrides
type EmergencyProcedureManager struct {
	procedures       map[string]EmergencyProcedure
	manualOverrides  map[string]ManualOverride
	disasterRecovery *DisasterRecoveryManager
	mu               sync.RWMutex
	emergencyMode    bool
	emergencyStarted time.Time
}

// EmergencyProcedure defines an emergency response procedure
type EmergencyProcedure struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	TriggerType EmergencyTriggerType  `json:"trigger_type"`
	Severity    EmergencySeverity     `json:"severity"`
	Action      func(context.Context, EmergencyEvent) error
	Timeout     time.Duration         `json:"timeout"`
	Priority    int                   `json:"priority"`
	RequiresApproval bool             `json:"requires_approval"`
	Description string               `json:"description"`
}

// ManualOverride allows manual intervention in automated systems
type ManualOverride struct {
	ID          string                 `json:"id"`
	Component   string                 `json:"component"`
	Action      string                 `json:"action"`
	Reason      string                 `json:"reason"`
	Operator    string                 `json:"operator"`
	Timestamp   time.Time             `json:"timestamp"`
	Duration    time.Duration         `json:"duration"`
	Active      bool                  `json:"active"`
	Context     map[string]interface{} `json:"context"`
}

// EmergencyEvent represents an emergency situation
type EmergencyEvent struct {
	ID          string                 `json:"id"`
	Type        EmergencyTriggerType  `json:"type"`
	Severity    EmergencySeverity     `json:"severity"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   time.Time             `json:"timestamp"`
	Resolved    bool                  `json:"resolved"`
	ResolvedAt  *time.Time            `json:"resolved_at,omitempty"`
}

type EmergencyTriggerType string

const (
	EmergencyTriggerCriticalFailure    EmergencyTriggerType = "critical_failure"
	EmergencyTriggerCascadeFailure     EmergencyTriggerType = "cascade_failure"
	EmergencyTriggerDataCorruption     EmergencyTriggerType = "data_corruption"
	EmergencyTriggerSecurityBreach     EmergencyTriggerType = "security_breach"
	EmergencyTriggerInfrastructureDown EmergencyTriggerType = "infrastructure_down"
	EmergencyTriggerManualTrigger      EmergencyTriggerType = "manual_trigger"
)

type EmergencySeverity string

const (
	EmergencySeverityP0 EmergencySeverity = "P0" // Critical - System down
	EmergencySeverityP1 EmergencySeverity = "P1" // High - Major functionality impacted
	EmergencySeverityP2 EmergencySeverity = "P2" // Medium - Some functionality impacted
	EmergencySeverityP3 EmergencySeverity = "P3" // Low - Minor impact
)

// NewEmergencyProcedureManager creates a new emergency procedure manager
func NewEmergencyProcedureManager() *EmergencyProcedureManager {
	epm := &EmergencyProcedureManager{
		procedures:       make(map[string]EmergencyProcedure),
		manualOverrides:  make(map[string]ManualOverride),
		disasterRecovery: NewDisasterRecoveryManager(),
		emergencyMode:    false,
	}
	
	// Register default emergency procedures
	epm.registerDefaultEmergencyProcedures()
	
	return epm
}

// TriggerEmergencyProcedure triggers an emergency procedure
func (epm *EmergencyProcedureManager) TriggerEmergencyProcedure(event EmergencyEvent) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	
	log.Printf("Emergency procedure triggered: %s (Type: %s, Severity: %s)", 
		event.ID, event.Type, event.Severity)
	
	// Find appropriate emergency procedure
	procedure := epm.findEmergencyProcedure(event)
	if procedure == nil {
		return fmt.Errorf("no emergency procedure found for event type: %s", event.Type)
	}
	
	// Check if manual approval is required
	if procedure.RequiresApproval && event.Severity != EmergencySeverityP0 {
		log.Printf("Emergency procedure %s requires manual approval", procedure.ID)
		return epm.requestManualApproval(event, *procedure)
	}
	
	// Execute emergency procedure
	ctx, cancel := context.WithTimeout(context.Background(), procedure.Timeout)
	defer cancel()
	
	log.Printf("Executing emergency procedure: %s", procedure.Name)
	
	if err := procedure.Action(ctx, event); err != nil {
		log.Printf("Emergency procedure failed: %s - %v", procedure.ID, err)
		return fmt.Errorf("emergency procedure failed: %w", err)
	}
	
	log.Printf("Emergency procedure completed successfully: %s", procedure.ID)
	return nil
}

// ActivateManualOverride activates a manual override
func (epm *EmergencyProcedureManager) ActivateManualOverride(override ManualOverride) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	
	override.ID = fmt.Sprintf("override_%d", time.Now().UnixNano())
	override.Timestamp = time.Now()
	override.Active = true
	
	epm.manualOverrides[override.ID] = override
	
	log.Printf("Manual override activated: %s for component %s (Reason: %s)", 
		override.ID, override.Component, override.Reason)
	
	// Execute the override action
	if err := epm.executeManualOverride(override); err != nil {
		return fmt.Errorf("failed to execute manual override: %w", err)
	}
	
	// Schedule automatic deactivation if duration is set
	if override.Duration > 0 {
		go epm.scheduleOverrideDeactivation(override.ID, override.Duration)
	}
	
	return nil
}

// DeactivateManualOverride deactivates a manual override
func (epm *EmergencyProcedureManager) DeactivateManualOverride(overrideID string) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	
	override, exists := epm.manualOverrides[overrideID]
	if !exists {
		return fmt.Errorf("manual override not found: %s", overrideID)
	}
	
	if !override.Active {
		return fmt.Errorf("manual override already inactive: %s", overrideID)
	}
	
	override.Active = false
	epm.manualOverrides[overrideID] = override
	
	log.Printf("Manual override deactivated: %s", overrideID)
	
	// Execute deactivation logic
	return epm.executeOverrideDeactivation(override)
}

// EnterEmergencyMode puts the system into emergency mode
func (epm *EmergencyProcedureManager) EnterEmergencyMode(reason string) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	
	if epm.emergencyMode {
		return fmt.Errorf("system already in emergency mode")
	}
	
	epm.emergencyMode = true
	epm.emergencyStarted = time.Now()
	
	log.Printf("EMERGENCY MODE ACTIVATED: %s", reason)
	
	// Activate emergency protocols
	if err := epm.activateEmergencyProtocols(); err != nil {
		return fmt.Errorf("failed to activate emergency protocols: %w", err)
	}
	
	return nil
}

// ExitEmergencyMode exits emergency mode
func (epm *EmergencyProcedureManager) ExitEmergencyMode() error {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	
	if !epm.emergencyMode {
		return fmt.Errorf("system not in emergency mode")
	}
	
	epm.emergencyMode = false
	duration := time.Since(epm.emergencyStarted)
	
	log.Printf("EMERGENCY MODE DEACTIVATED (Duration: %v)", duration)
	
	// Deactivate emergency protocols
	if err := epm.deactivateEmergencyProtocols(); err != nil {
		return fmt.Errorf("failed to deactivate emergency protocols: %w", err)
	}
	
	return nil
}

// InitiateDisasterRecovery initiates disaster recovery procedures
func (epm *EmergencyProcedureManager) InitiateDisasterRecovery(scenario string) error {
	log.Printf("DISASTER RECOVERY INITIATED: %s", scenario)
	
	return epm.disasterRecovery.InitiateRecovery(scenario)
}

// registerDefaultEmergencyProcedures sets up built-in emergency procedures
func (epm *EmergencyProcedureManager) registerDefaultEmergencyProcedures() {
	// Critical system failure procedure
	epm.procedures["critical_system_failure"] = EmergencyProcedure{
		ID:          "critical_system_failure",
		Name:        "Critical System Failure Response",
		TriggerType: EmergencyTriggerCriticalFailure,
		Severity:    EmergencySeverityP0,
		Action:      epm.handleCriticalSystemFailure,
		Timeout:     5 * time.Minute,
		Priority:    100,
		RequiresApproval: false,
		Description: "Immediate response to critical system failures",
	}
	
	// Cascade failure procedure
	epm.procedures["cascade_failure"] = EmergencyProcedure{
		ID:          "cascade_failure",
		Name:        "Cascade Failure Containment",
		TriggerType: EmergencyTriggerCascadeFailure,
		Severity:    EmergencySeverityP1,
		Action:      epm.handleCascadeFailure,
		Timeout:     10 * time.Minute,
		Priority:    90,
		RequiresApproval: false,
		Description: "Contain and isolate cascade failures",
	}
	
	// Data corruption procedure
	epm.procedures["data_corruption"] = EmergencyProcedure{
		ID:          "data_corruption",
		Name:        "Data Corruption Response",
		TriggerType: EmergencyTriggerDataCorruption,
		Severity:    EmergencySeverityP0,
		Action:      epm.handleDataCorruption,
		Timeout:     30 * time.Minute,
		Priority:    95,
		RequiresApproval: true,
		Description: "Respond to data corruption incidents",
	}
	
	// Security breach procedure
	epm.procedures["security_breach"] = EmergencyProcedure{
		ID:          "security_breach",
		Name:        "Security Breach Response",
		TriggerType: EmergencyTriggerSecurityBreach,
		Severity:    EmergencySeverityP0,
		Action:      epm.handleSecurityBreach,
		Timeout:     15 * time.Minute,
		Priority:    100,
		RequiresApproval: false,
		Description: "Immediate response to security breaches",
	}
	
	// Infrastructure down procedure
	epm.procedures["infrastructure_down"] = EmergencyProcedure{
		ID:          "infrastructure_down",
		Name:        "Infrastructure Failure Response",
		TriggerType: EmergencyTriggerInfrastructureDown,
		Severity:    EmergencySeverityP1,
		Action:      epm.handleInfrastructureDown,
		Timeout:     20 * time.Minute,
		Priority:    85,
		RequiresApproval: false,
		Description: "Respond to infrastructure failures",
	}
}

// Emergency procedure implementations
func (epm *EmergencyProcedureManager) handleCriticalSystemFailure(ctx context.Context, event EmergencyEvent) error {
	log.Printf("Handling critical system failure: %s", event.Message)
	
	// Step 1: Enter emergency mode
	if err := epm.EnterEmergencyMode("Critical system failure detected"); err != nil {
		log.Printf("Warning: failed to enter emergency mode: %v", err)
	}
	
	// Step 2: Isolate failed components
	if component, exists := event.Context["component"].(string); exists {
		if err := epm.isolateComponent(component); err != nil {
			return fmt.Errorf("failed to isolate component %s: %w", component, err)
		}
	}
	
	// Step 3: Activate backup systems
	if err := epm.activateBackupSystems(); err != nil {
		return fmt.Errorf("failed to activate backup systems: %w", err)
	}
	
	// Step 4: Notify stakeholders
	if err := epm.notifyStakeholders(event); err != nil {
		log.Printf("Warning: failed to notify stakeholders: %v", err)
	}
	
	return nil
}

func (epm *EmergencyProcedureManager) handleCascadeFailure(ctx context.Context, event EmergencyEvent) error {
	log.Printf("Handling cascade failure: %s", event.Message)
	
	// Step 1: Identify cascade pattern
	pattern := epm.identifyCascadePattern(event)
	log.Printf("Cascade pattern identified: %s", pattern)
	
	// Step 2: Break cascade chain
	if err := epm.breakCascadeChain(pattern); err != nil {
		return fmt.Errorf("failed to break cascade chain: %w", err)
	}
	
	// Step 3: Isolate affected components
	if err := epm.isolateAffectedComponents(event); err != nil {
		return fmt.Errorf("failed to isolate affected components: %w", err)
	}
	
	return nil
}

func (epm *EmergencyProcedureManager) handleDataCorruption(ctx context.Context, event EmergencyEvent) error {
	log.Printf("Handling data corruption: %s", event.Message)
	
	// Step 1: Stop all write operations
	if err := epm.stopWriteOperations(); err != nil {
		return fmt.Errorf("failed to stop write operations: %w", err)
	}
	
	// Step 2: Assess corruption scope
	scope, err := epm.assessCorruptionScope(event)
	if err != nil {
		return fmt.Errorf("failed to assess corruption scope: %w", err)
	}
	
	// Step 3: Initiate data recovery
	if err := epm.initiateDataRecovery(scope); err != nil {
		return fmt.Errorf("failed to initiate data recovery: %w", err)
	}
	
	return nil
}

func (epm *EmergencyProcedureManager) handleSecurityBreach(ctx context.Context, event EmergencyEvent) error {
	log.Printf("Handling security breach: %s", event.Message)
	
	// Step 1: Isolate compromised systems
	if err := epm.isolateCompromisedSystems(event); err != nil {
		return fmt.Errorf("failed to isolate compromised systems: %w", err)
	}
	
	// Step 2: Revoke access tokens
	if err := epm.revokeAccessTokens(); err != nil {
		return fmt.Errorf("failed to revoke access tokens: %w", err)
	}
	
	// Step 3: Enable enhanced monitoring
	if err := epm.enableEnhancedMonitoring(); err != nil {
		return fmt.Errorf("failed to enable enhanced monitoring: %w", err)
	}
	
	return nil
}

func (epm *EmergencyProcedureManager) handleInfrastructureDown(ctx context.Context, event EmergencyEvent) error {
	log.Printf("Handling infrastructure failure: %s", event.Message)
	
	// Step 1: Activate failover systems
	if err := epm.activateFailoverSystems(); err != nil {
		return fmt.Errorf("failed to activate failover systems: %w", err)
	}
	
	// Step 2: Reroute traffic
	if err := epm.rerouteTraffic(); err != nil {
		return fmt.Errorf("failed to reroute traffic: %w", err)
	}
	
	return nil
}

// Helper methods for emergency procedures
func (epm *EmergencyProcedureManager) findEmergencyProcedure(event EmergencyEvent) *EmergencyProcedure {
	var bestProcedure *EmergencyProcedure
	highestPriority := -1
	
	for _, procedure := range epm.procedures {
		if procedure.TriggerType == event.Type && procedure.Priority > highestPriority {
			procedureCopy := procedure
			bestProcedure = &procedureCopy
			highestPriority = procedure.Priority
		}
	}
	
	return bestProcedure
}

func (epm *EmergencyProcedureManager) requestManualApproval(event EmergencyEvent, procedure EmergencyProcedure) error {
	log.Printf("Manual approval required for procedure: %s", procedure.ID)
	// This would integrate with approval systems
	return fmt.Errorf("manual approval required")
}

func (epm *EmergencyProcedureManager) executeManualOverride(override ManualOverride) error {
	log.Printf("Executing manual override: %s on component %s", override.Action, override.Component)
	
	switch override.Action {
	case "disable_component":
		return epm.disableComponent(override.Component)
	case "enable_component":
		return epm.enableComponent(override.Component)
	case "restart_component":
		return epm.restartComponent(override.Component)
	case "isolate_component":
		return epm.isolateComponent(override.Component)
	default:
		return fmt.Errorf("unknown override action: %s", override.Action)
	}
}

func (epm *EmergencyProcedureManager) executeOverrideDeactivation(override ManualOverride) error {
	log.Printf("Deactivating override for component: %s", override.Component)
	// This would reverse the override action
	return nil
}

func (epm *EmergencyProcedureManager) scheduleOverrideDeactivation(overrideID string, duration time.Duration) {
	time.Sleep(duration)
	if err := epm.DeactivateManualOverride(overrideID); err != nil {
		log.Printf("Failed to auto-deactivate override %s: %v", overrideID, err)
	}
}

// Component control methods
func (epm *EmergencyProcedureManager) disableComponent(component string) error {
	log.Printf("Disabling component: %s", component)
	return nil
}

func (epm *EmergencyProcedureManager) enableComponent(component string) error {
	log.Printf("Enabling component: %s", component)
	return nil
}

func (epm *EmergencyProcedureManager) restartComponent(component string) error {
	log.Printf("Restarting component: %s", component)
	return nil
}

func (epm *EmergencyProcedureManager) isolateComponent(component string) error {
	log.Printf("Isolating component: %s", component)
	return nil
}

// Emergency protocol methods
func (epm *EmergencyProcedureManager) activateEmergencyProtocols() error {
	log.Printf("Activating emergency protocols")
	return nil
}

func (epm *EmergencyProcedureManager) deactivateEmergencyProtocols() error {
	log.Printf("Deactivating emergency protocols")
	return nil
}

// Additional helper methods
func (epm *EmergencyProcedureManager) identifyCascadePattern(event EmergencyEvent) string {
	// This would analyze the cascade pattern
	return "database_to_cache_to_api"
}

func (epm *EmergencyProcedureManager) breakCascadeChain(pattern string) error {
	log.Printf("Breaking cascade chain: %s", pattern)
	return nil
}

func (epm *EmergencyProcedureManager) isolateAffectedComponents(event EmergencyEvent) error {
	log.Printf("Isolating affected components")
	return nil
}

func (epm *EmergencyProcedureManager) stopWriteOperations() error {
	log.Printf("Stopping all write operations")
	return nil
}

func (epm *EmergencyProcedureManager) assessCorruptionScope(event EmergencyEvent) (string, error) {
	log.Printf("Assessing corruption scope")
	return "limited", nil
}

func (epm *EmergencyProcedureManager) initiateDataRecovery(scope string) error {
	log.Printf("Initiating data recovery for scope: %s", scope)
	return nil
}

func (epm *EmergencyProcedureManager) isolateCompromisedSystems(event EmergencyEvent) error {
	log.Printf("Isolating compromised systems")
	return nil
}

func (epm *EmergencyProcedureManager) revokeAccessTokens() error {
	log.Printf("Revoking all access tokens")
	return nil
}

func (epm *EmergencyProcedureManager) enableEnhancedMonitoring() error {
	log.Printf("Enabling enhanced security monitoring")
	return nil
}

func (epm *EmergencyProcedureManager) activateFailoverSystems() error {
	log.Printf("Activating failover systems")
	return nil
}

func (epm *EmergencyProcedureManager) rerouteTraffic() error {
	log.Printf("Rerouting traffic to backup systems")
	return nil
}

func (epm *EmergencyProcedureManager) activateBackupSystems() error {
	log.Printf("Activating backup systems")
	return nil
}

func (epm *EmergencyProcedureManager) notifyStakeholders(event EmergencyEvent) error {
	log.Printf("Notifying stakeholders of emergency: %s", event.Type)
	return nil
}

// GetEmergencyStatus returns the current emergency status
func (epm *EmergencyProcedureManager) GetEmergencyStatus() EmergencyStatus {
	epm.mu.RLock()
	defer epm.mu.RUnlock()
	
	activeOverrides := 0
	for _, override := range epm.manualOverrides {
		if override.Active {
			activeOverrides++
		}
	}
	
	return EmergencyStatus{
		EmergencyMode:       epm.emergencyMode,
		EmergencyStarted:    epm.emergencyStarted,
		ActiveOverrides:     activeOverrides,
		AvailableProcedures: len(epm.procedures),
	}
}

type EmergencyStatus struct {
	EmergencyMode       bool      `json:"emergency_mode"`
	EmergencyStarted    time.Time `json:"emergency_started,omitempty"`
	ActiveOverrides     int       `json:"active_overrides"`
	AvailableProcedures int       `json:"available_procedures"`
}