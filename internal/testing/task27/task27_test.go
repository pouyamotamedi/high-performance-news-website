package task27

import (
	"context"
	"testing"
	"time"
)

// Test the core threat modeling functionality
func TestThreatModelingEngine(t *testing.T) {
	projectRoot := "../../../"
	engine := NewThreatModelingEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("ThreatModelCreation", func(t *testing.T) {
		// Test building a threat model
		threatModel, err := engine.buildThreatModel(ctx)
		if err != nil {
			t.Fatalf("Failed to build threat model: %v", err)
		}

		if threatModel == nil {
			t.Fatal("Threat model is nil")
		}

		if threatModel.ID == "" {
			t.Error("Expected threat model ID")
		}

		if len(threatModel.Assets) == 0 {
			t.Error("Expected assets to be identified")
		}

		if len(threatModel.Threats) == 0 {
			t.Error("Expected threats to be identified")
		}

		if len(threatModel.Mitigations) == 0 {
			t.Error("Expected mitigations to be identified")
		}

		t.Logf("Threat model created with %d assets, %d threats, %d mitigations", 
			len(threatModel.Assets), len(threatModel.Threats), len(threatModel.Mitigations))
		t.Logf("Overall risk: %s (score: %.1f)", 
			threatModel.RiskAssessment.OverallRisk, threatModel.RiskAssessment.RiskScore)
	})

	t.Run("ComprehensiveThreatModeling", func(t *testing.T) {
		result, err := engine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Comprehensive threat modeling failed: %v", err)
		}

		if result == nil {
			t.Fatal("Threat modeling result is nil")
		}

		if result.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", result.Status)
		}

		if result.ExecutionTime == 0 {
			t.Error("Expected non-zero execution time")
		}

		// Verify attack simulations
		if len(result.AttackSimulations) == 0 {
			t.Error("Expected attack simulations to be performed")
		}

		// Verify compliance results
		if len(result.ComplianceResults) == 0 {
			t.Error("Expected compliance results")
		}

		// Verify SIEM integration
		if result.SIEMIntegration.Status == "" {
			t.Error("Expected SIEM integration status")
		}

		t.Logf("Threat modeling completed in %v", result.ExecutionTime)
		t.Logf("Attack simulations: %d", len(result.AttackSimulations))
		t.Logf("Compliance results: %d", len(result.ComplianceResults))
		t.Logf("SIEM events forwarded: %d", result.SIEMIntegration.EventsForwarded)
	})
}

// Test attack simulation functionality
func TestAttackSimulator(t *testing.T) {
	projectRoot := "../../../"
	simulator := NewAttackSimulator(projectRoot)

	// Create a mock threat model
	threatModel := &ThreatModel{
		ID:   "test-model",
		Name: "Test Threat Model",
		Assets: []Asset{
			{
				ID:   "web-app",
				Name: "Web Application",
				Type: "application",
			},
		},
		Threats: []Threat{
			{
				ID:       "T001",
				Name:     "SQL Injection",
				Category: "injection",
				RiskScore: 8.5,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("AttackSimulations", func(t *testing.T) {
		simulations, err := simulator.RunAttackSimulations(ctx, threatModel)
		if err != nil {
			t.Fatalf("Attack simulation failed: %v", err)
		}

		if len(simulations) == 0 {
			t.Error("Expected attack simulations to be performed")
		}

		// Check for specific attack types
		attackTypes := make(map[string]bool)
		for _, sim := range simulations {
			attackTypes[sim.AttackType] = true

			if sim.Duration == 0 {
				t.Errorf("Simulation %s has zero duration", sim.AttackType)
			}

			if sim.Status != "completed" {
				t.Errorf("Simulation %s not completed: %s", sim.AttackType, sim.Status)
			}

			t.Logf("Attack simulation %s: Success=%v, Severity=%s, CVSS=%.1f", 
				sim.AttackType, sim.Success, sim.Severity, sim.CVSS)
		}

		// Verify expected attack types
		expectedAttacks := []string{"sql_injection", "cross_site_scripting", "authentication_bypass", "cross_site_request_forgery", "directory_traversal"}
		for _, expected := range expectedAttacks {
			if !attackTypes[expected] {
				t.Errorf("Expected attack type %s not found", expected)
			}
		}
	})

	t.Run("PayloadGeneration", func(t *testing.T) {
		if len(simulator.payloadGen.SQLInjectionPayloads) == 0 {
			t.Error("Expected SQL injection payloads")
		}

		if len(simulator.payloadGen.XSSPayloads) == 0 {
			t.Error("Expected XSS payloads")
		}

		t.Logf("Generated %d SQL injection payloads", len(simulator.payloadGen.SQLInjectionPayloads))
		t.Logf("Generated %d XSS payloads", len(simulator.payloadGen.XSSPayloads))
	})

	t.Run("ExploitDatabase", func(t *testing.T) {
		if len(simulator.exploitDB.WebExploits) == 0 {
			t.Error("Expected web exploits in database")
		}

		for _, exploit := range simulator.exploitDB.WebExploits {
			if exploit.ID == "" {
				t.Error("Expected exploit ID")
			}
			if exploit.Name == "" {
				t.Error("Expected exploit name")
			}
			if len(exploit.Payloads) == 0 {
				t.Error("Expected exploit payloads")
			}
		}

		t.Logf("Exploit database contains %d web exploits", len(simulator.exploitDB.WebExploits))
	})
}

// Test compliance checking functionality
func TestComplianceChecker(t *testing.T) {
	projectRoot := "../../../"
	checker := NewComplianceChecker(projectRoot)

	// Create a mock threat model
	threatModel := &ThreatModel{
		ID:   "test-model",
		Name: "Test Threat Model",
		Assets: []Asset{
			{
				ID:   "database",
				Name: "User Database",
				Type: "database",
				DataTypes: []string{"pii", "personal_data"},
				AccessControls: []AccessControl{
					{
						Type:        "authentication",
						Principals:  []string{"app_user"},
						Permissions: []string{"read", "write"},
					},
				},
			},
		},
		Threats: []Threat{
			{
				ID:       "T001",
				Name:     "Data Breach",
				Category: "information_disclosure",
				RiskScore: 8.0,
			},
		},
		Mitigations: []Mitigation{
			{
				ID:   "M001",
				Name: "Encryption at Rest",
				Type: "preventive",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("GDPRCompliance", func(t *testing.T) {
		results, err := checker.CheckCompliance(ctx, threatModel)
		if err != nil {
			t.Fatalf("Compliance check failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected compliance results")
		}

		// Find GDPR result
		var gdprResult *ComplianceResult
		for _, result := range results {
			if result.Framework == "GDPR" {
				gdprResult = &result
				break
			}
		}

		if gdprResult == nil {
			t.Fatal("GDPR compliance result not found")
		}

		if gdprResult.OverallScore == 0 {
			t.Error("Expected non-zero GDPR compliance score")
		}

		if len(gdprResult.ControlResults) == 0 {
			t.Error("Expected GDPR control results")
		}

		t.Logf("GDPR compliance: %.1f%% (%s)", gdprResult.OverallScore, gdprResult.Status)
		t.Logf("GDPR controls assessed: %d", len(gdprResult.ControlResults))

		// Check specific controls
		for _, control := range gdprResult.ControlResults {
			t.Logf("Control %s: %.1f%% (%s)", control.ControlID, control.Score, control.Status)
		}
	})

	t.Run("MultipleFrameworks", func(t *testing.T) {
		results, err := checker.CheckCompliance(ctx, threatModel)
		if err != nil {
			t.Fatalf("Compliance check failed: %v", err)
		}

		frameworks := make(map[string]bool)
		for _, result := range results {
			frameworks[result.Framework] = true
		}

		expectedFrameworks := []string{"GDPR", "CCPA", "ISO27001"}
		for _, expected := range expectedFrameworks {
			if !frameworks[expected] {
				t.Errorf("Expected compliance framework %s not found", expected)
			}
		}
	})
}

// Test SIEM integration functionality
func TestSIEMIntegration(t *testing.T) {
	projectRoot := "../../../"
	siem := NewSIEMIntegration(projectRoot)

	// Create a mock threat model
	threatModel := &ThreatModel{
		ID:   "test-model",
		Name: "Test Threat Model",
		Threats: []Threat{
			{
				ID:            "T001",
				Name:          "SQL Injection",
				Category:      "injection",
				RiskScore:     8.5,
				STRIDECategory: "tampering",
				MITREAttackID: "T1190",
			},
		},
		Vulnerabilities: []Vulnerability{
			{
				ID:       "V001",
				Title:    "Outdated Library",
				Severity: "high",
				CVSS:     7.5,
				Package:  "test-package",
			},
		},
		AttackPaths: []AttackPath{
			{
				ID:          "AP001",
				Name:        "Web to Database",
				Probability: 0.6,
				Impact:      "high",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("SIEMIntegration", func(t *testing.T) {
		result, err := siem.IntegrateWithSIEM(ctx, threatModel)
		if err != nil {
			t.Fatalf("SIEM integration failed: %v", err)
		}

		if result == nil {
			t.Fatal("SIEM integration result is nil")
		}

		if result.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", result.Status)
		}

		if result.EventsForwarded == 0 {
			t.Error("Expected events to be forwarded")
		}

		t.Logf("SIEM integration: %d events forwarded, %d alerts generated", 
			result.EventsForwarded, result.AlertsGenerated)
		t.Logf("Connected SIEMs: %v", result.ConnectedSIEMs)
		t.Logf("Rules triggered: %v", result.RulesTriggered)
	})

	t.Run("EventGeneration", func(t *testing.T) {
		result, err := siem.IntegrateWithSIEM(ctx, threatModel)
		if err != nil {
			t.Fatalf("SIEM integration failed: %v", err)
		}

		// Should generate events for threats, vulnerabilities, and attack paths
		expectedMinEvents := len(threatModel.Threats) + len(threatModel.Vulnerabilities) + len(threatModel.AttackPaths)
		if result.EventsForwarded < expectedMinEvents {
			t.Errorf("Expected at least %d events, got %d", expectedMinEvents, result.EventsForwarded)
		}
	})

	t.Run("SIEMConnectors", func(t *testing.T) {
		expectedConnectors := []string{"splunk", "elastic"}
		for _, connectorName := range expectedConnectors {
			if _, exists := siem.connectors[connectorName]; !exists {
				t.Errorf("SIEM connector %s not found", connectorName)
			}
		}

		t.Logf("SIEM integration supports %d connectors", len(siem.connectors))
	})

	t.Run("AlertRules", func(t *testing.T) {
		if len(siem.alertManager.rules) == 0 {
			t.Error("Expected alert rules")
		}

		for _, rule := range siem.alertManager.rules {
			if rule.ID == "" {
				t.Error("Expected rule ID")
			}
			if rule.Name == "" {
				t.Error("Expected rule name")
			}
		}

		t.Logf("Alert manager has %d rules", len(siem.alertManager.rules))
	})

	t.Run("CorrelationRules", func(t *testing.T) {
		if len(siem.ruleEngine.rules) == 0 {
			t.Error("Expected correlation rules")
		}

		for _, rule := range siem.ruleEngine.rules {
			if rule.ID == "" {
				t.Error("Expected rule ID")
			}
			if rule.Name == "" {
				t.Error("Expected rule name")
			}
		}

		t.Logf("Rule engine has %d correlation rules", len(siem.ruleEngine.rules))
	})
}

// Test integration between components
func TestTask27Integration(t *testing.T) {
	projectRoot := "../../../"
	
	threatEngine := NewThreatModelingEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("EndToEndWorkflow", func(t *testing.T) {
		// Step 1: Run comprehensive threat modeling
		result, err := threatEngine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Threat modeling failed: %v", err)
		}

		// Verify all components ran
		if len(result.AttackSimulations) == 0 {
			t.Error("Expected attack simulations")
		}

		if len(result.ComplianceResults) == 0 {
			t.Error("Expected compliance results")
		}

		if result.SIEMIntegration.EventsForwarded == 0 {
			t.Error("Expected SIEM events")
		}

		t.Logf("End-to-end workflow completed successfully")
		t.Logf("Threat model: %d threats identified", len(result.ThreatModel.Threats))
		t.Logf("Attack simulations: %d performed", len(result.AttackSimulations))
		t.Logf("Compliance frameworks: %d assessed", len(result.ComplianceResults))
		t.Logf("SIEM events: %d forwarded", result.SIEMIntegration.EventsForwarded)
		t.Logf("Overall execution time: %v", result.ExecutionTime)
	})

	t.Run("ComponentCreation", func(t *testing.T) {
		// Test that all components can be created successfully
		simulator := NewAttackSimulator(projectRoot)
		if simulator == nil {
			t.Fatal("Attack simulator is nil")
		}

		checker := NewComplianceChecker(projectRoot)
		if checker == nil {
			t.Fatal("Compliance checker is nil")
		}

		siem := NewSIEMIntegration(projectRoot)
		if siem == nil {
			t.Fatal("SIEM integration is nil")
		}

		t.Log("All components created successfully")
	})
}