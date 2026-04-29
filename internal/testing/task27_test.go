package testing

import (
	"context"
	"testing"
	"time"
)

// Test the core threat modeling functionality
func TestTask27ThreatModeling(t *testing.T) {
	projectRoot := "../../"
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

	t.Run("AttackSimulatorCreation", func(t *testing.T) {
		simulator := NewAttackSimulator(projectRoot)
		if simulator == nil {
			t.Fatal("Attack simulator is nil")
		}

		if simulator.config == nil {
			t.Error("Expected attack simulator config")
		}

		if simulator.exploitDB == nil {
			t.Error("Expected exploit database")
		}

		if simulator.payloadGen == nil {
			t.Error("Expected payload generator")
		}

		t.Log("Attack simulator created successfully")
	})

	t.Run("ComplianceCheckerCreation", func(t *testing.T) {
		checker := NewComplianceChecker(projectRoot)
		if checker == nil {
			t.Fatal("Compliance checker is nil")
		}

		if checker.config == nil {
			t.Error("Expected compliance checker config")
		}

		if len(checker.frameworks) == 0 {
			t.Error("Expected compliance frameworks")
		}

		t.Logf("Compliance checker created with %d frameworks", len(checker.frameworks))
	})

	t.Run("SIEMIntegrationCreation", func(t *testing.T) {
		siem := NewSIEMIntegration(projectRoot)
		if siem == nil {
			t.Fatal("SIEM integration is nil")
		}

		if siem.config == nil {
			t.Error("Expected SIEM config")
		}

		if len(siem.connectors) == 0 {
			t.Error("Expected SIEM connectors")
		}

		t.Logf("SIEM integration created with %d connectors", len(siem.connectors))
	})

	t.Run("IncidentResponseCreation", func(t *testing.T) {
		incidentEngine := NewIncidentResponseEngine(projectRoot)
		if incidentEngine == nil {
			t.Fatal("Incident response engine is nil")
		}

		if incidentEngine.config == nil {
			t.Error("Expected incident response config")
		}

		if incidentEngine.detector == nil {
			t.Error("Expected incident detector")
		}

		if incidentEngine.responder == nil {
			t.Error("Expected incident responder")
		}

		if incidentEngine.forensicsEngine == nil {
			t.Error("Expected forensics engine")
		}

		t.Log("Incident response engine created successfully")
	})
}

// Test attack simulation functionality
func TestTask27AttackSimulation(t *testing.T) {
	projectRoot := "../../"
	simulator := NewAttackSimulator(projectRoot)

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
func TestTask27ComplianceChecking(t *testing.T) {
	projectRoot := "../../"
	checker := NewComplianceChecker(projectRoot)

	t.Run("GDPRFramework", func(t *testing.T) {
		gdprChecker, exists := checker.frameworks["GDPR"]
		if !exists {
			t.Fatal("GDPR framework not found")
		}

		if gdprChecker.GetFrameworkName() != "GDPR" {
			t.Error("Expected GDPR framework name")
		}

		controls := gdprChecker.GetControls()
		if len(controls) == 0 {
			t.Error("Expected GDPR controls")
		}

		t.Logf("GDPR framework has %d controls", len(controls))
		for _, control := range controls {
			t.Logf("Control %s: %s", control.ID, control.Name)
		}
	})

	t.Run("MultipleFrameworks", func(t *testing.T) {
		expectedFrameworks := []string{"GDPR", "CCPA", "SOX", "ISO27001", "PCIDSS"}
		for _, framework := range expectedFrameworks {
			if _, exists := checker.frameworks[framework]; !exists {
				t.Errorf("Framework %s not found", framework)
			}
		}

		t.Logf("Compliance checker supports %d frameworks", len(checker.frameworks))
	})
}

// Test SIEM integration functionality
func TestTask27SIEMIntegration(t *testing.T) {
	projectRoot := "../../"
	siem := NewSIEMIntegration(projectRoot)

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

// Test incident response functionality
func TestTask27IncidentResponse(t *testing.T) {
	projectRoot := "../../"
	engine := NewIncidentResponseEngine(projectRoot)

	t.Run("DetectionRules", func(t *testing.T) {
		if len(engine.detector.rules) == 0 {
			t.Error("Expected detection rules")
		}

		for _, rule := range engine.detector.rules {
			if rule.ID == "" {
				t.Error("Expected rule ID")
			}
			if rule.Name == "" {
				t.Error("Expected rule name")
			}
		}

		t.Logf("Incident detector has %d rules", len(engine.detector.rules))
	})

	t.Run("ResponsePlaybooks", func(t *testing.T) {
		if len(engine.responder.playbooks) == 0 {
			t.Error("Expected response playbooks")
		}

		for _, playbook := range engine.responder.playbooks {
			if playbook.ID == "" {
				t.Error("Expected playbook ID")
			}
			if playbook.Name == "" {
				t.Error("Expected playbook name")
			}
			if len(playbook.Steps) == 0 {
				t.Error("Expected playbook steps")
			}
		}

		t.Logf("Incident responder has %d playbooks", len(engine.responder.playbooks))
	})

	t.Run("ForensicTools", func(t *testing.T) {
		if len(engine.forensicsEngine.tools) == 0 {
			t.Error("Expected forensic tools")
		}

		for _, tool := range engine.forensicsEngine.tools {
			if tool.Name == "" {
				t.Error("Expected tool name")
			}
			if tool.Type == "" {
				t.Error("Expected tool type")
			}
		}

		t.Logf("Forensics engine has %d tools", len(engine.forensicsEngine.tools))
	})

	t.Run("EvidenceCollectors", func(t *testing.T) {
		if len(engine.forensicsEngine.collectors) == 0 {
			t.Error("Expected evidence collectors")
		}

		for _, collector := range engine.forensicsEngine.collectors {
			if collector.Type == "" {
				t.Error("Expected collector type")
			}
			if len(collector.Sources) == 0 {
				t.Error("Expected collector sources")
			}
		}

		t.Logf("Forensics engine has %d evidence collectors", len(engine.forensicsEngine.collectors))
	})
}

// Test integration between components
func TestTask27Integration(t *testing.T) {
	projectRoot := "../../"
	
	threatEngine := NewThreatModelingEngine(projectRoot)
	incidentEngine := NewIncidentResponseEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("ThreatToIncidentFlow", func(t *testing.T) {
		// Build a simple threat model
		threatModel, err := threatEngine.buildThreatModel(ctx)
		if err != nil {
			t.Fatalf("Failed to build threat model: %v", err)
		}

		// Create security events from threats
		var events []SecurityEvent
		for i, threat := range threatModel.Threats {
			if i >= 3 { // Limit to first 3 threats
				break
			}
			
			event := SecurityEvent{
				ID:          "event-" + threat.ID,
				Timestamp:   time.Now(),
				Source:      "threat_modeling",
				EventType:   "threat_indicator",
				Severity:    "medium",
				Category:    threat.Category,
				Description: "Threat indicator: " + threat.Name,
			}
			events = append(events, event)
		}

		// Test event correlation
		correlations := incidentEngine.detector.correlator.CorrelateEvents(events)
		
		t.Logf("Created %d events from threats", len(events))
		t.Logf("Found %d correlations", len(correlations))

		if len(events) > 0 && len(correlations) == 0 {
			t.Log("No correlations found - this is expected for simple test events")
		}
	})
}