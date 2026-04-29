package testing

import (
	"context"
	"testing"
	"time"
)

func TestThreatModelingEngine(t *testing.T) {
	projectRoot := "../../"
	engine := NewThreatModelingEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("ComprehensiveThreatModeling", func(t *testing.T) {
		result, err := engine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Threat modeling failed: %v", err)
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

		// Verify threat model components
		threatModel := result.ThreatModel
		if len(threatModel.Assets) == 0 {
			t.Error("Expected assets to be identified")
		}

		if len(threatModel.Threats) == 0 {
			t.Error("Expected threats to be identified")
		}

		if len(threatModel.Mitigations) == 0 {
			t.Error("Expected mitigations to be identified")
		}

		// Verify risk assessment
		if threatModel.RiskAssessment.OverallRisk == "" {
			t.Error("Expected overall risk assessment")
		}

		// Verify compliance assessment
		if threatModel.ComplianceStatus.OverallScore == 0 {
			t.Error("Expected compliance score")
		}

		t.Logf("Threat modeling completed in %v", result.ExecutionTime)
		t.Logf("Identified %d assets, %d threats, %d mitigations", 
			len(threatModel.Assets), len(threatModel.Threats), len(threatModel.Mitigations))
		t.Logf("Overall risk: %s (score: %.1f)", 
			threatModel.RiskAssessment.OverallRisk, threatModel.RiskAssessment.RiskScore)
		t.Logf("Compliance score: %.1f%%", threatModel.ComplianceStatus.OverallScore)
	})

	t.Run("AttackSimulation", func(t *testing.T) {
		// First build a threat model
		result, err := engine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Failed to build threat model: %v", err)
		}

		// Test attack simulation
		if len(result.AttackSimulations) == 0 {
			t.Error("Expected attack simulations to be performed")
		}

		for _, simulation := range result.AttackSimulations {
			if simulation.AttackType == "" {
				t.Error("Expected attack type to be specified")
			}

			if simulation.Duration == 0 {
				t.Error("Expected non-zero simulation duration")
			}

			t.Logf("Attack simulation: %s - Success: %v, Severity: %s", 
				simulation.AttackType, simulation.Success, simulation.Severity)
		}
	})

	t.Run("ComplianceChecking", func(t *testing.T) {
		result, err := engine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Failed to run threat modeling: %v", err)
		}

		if len(result.ComplianceResults) == 0 {
			t.Error("Expected compliance results")
		}

		for _, complianceResult := range result.ComplianceResults {
			if complianceResult.Framework == "" {
				t.Error("Expected compliance framework to be specified")
			}

			if complianceResult.OverallScore == 0 {
				t.Error("Expected non-zero compliance score")
			}

			t.Logf("Compliance %s: %.1f%% (%s)", 
				complianceResult.Framework, complianceResult.OverallScore, complianceResult.Status)
		}
	})

	t.Run("SIEMIntegration", func(t *testing.T) {
		result, err := engine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Failed to run threat modeling: %v", err)
		}

		siemResult := result.SIEMIntegration
		if siemResult.Status == "" {
			t.Error("Expected SIEM integration status")
		}

		if siemResult.EventsForwarded == 0 {
			t.Error("Expected events to be forwarded to SIEM")
		}

		t.Logf("SIEM integration: %s, %d events forwarded, %d alerts generated", 
			siemResult.Status, siemResult.EventsForwarded, siemResult.AlertsGenerated)
	})
}

func TestAttackSimulator(t *testing.T) {
	projectRoot := "../../"
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("WebApplicationAttacks", func(t *testing.T) {
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
}

func TestComplianceChecker(t *testing.T) {
	projectRoot := "../../"
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

func TestSIEMIntegration(t *testing.T) {
	projectRoot := "../../"
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

	t.Run("CorrelationRules", func(t *testing.T) {
		result, err := siem.IntegrateWithSIEM(ctx, threatModel)
		if err != nil {
			t.Fatalf("SIEM integration failed: %v", err)
		}

		if len(result.CorrelationResults) > 0 {
			for _, correlation := range result.CorrelationResults {
				if correlation.Confidence == 0 {
					t.Error("Expected non-zero correlation confidence")
				}

				t.Logf("Correlation rule %s: confidence=%.2f, risk=%.1f", 
					correlation.RuleName, correlation.Confidence, correlation.RiskScore)
			}
		}
	})
}

func TestIncidentResponseEngine(t *testing.T) {
	projectRoot := "../../"
	engine := NewIncidentResponseEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("IncidentDetection", func(t *testing.T) {
		// Create mock security events
		events := []SecurityEvent{
			{
				ID:          "event-1",
				Timestamp:   time.Now(),
				Source:      "web_application",
				EventType:   "authentication_failure",
				Severity:    "medium",
				Category:    "authentication",
				Description: "Failed login attempt",
				SourceIP:    "192.168.1.100",
				UserID:      "test_user",
			},
			{
				ID:          "event-2",
				Timestamp:   time.Now(),
				Source:      "web_application",
				EventType:   "authentication_failure",
				Severity:    "medium",
				Category:    "authentication",
				Description: "Failed login attempt",
				SourceIP:    "192.168.1.100",
				UserID:      "test_user",
			},
		}

		incident, err := engine.DetectIncident(ctx, events)
		if err != nil {
			t.Fatalf("Incident detection failed: %v", err)
		}

		// Note: Incident might be nil if no patterns match
		if incident != nil {
			if incident.ID == "" {
				t.Error("Expected incident ID")
			}

			if incident.Severity == "" {
				t.Error("Expected incident severity")
			}

			if len(incident.Timeline) == 0 {
				t.Error("Expected incident timeline")
			}

			t.Logf("Incident detected: %s (Severity: %s)", incident.Title, incident.Severity)
			t.Logf("Timeline events: %d", len(incident.Timeline))
			t.Logf("Evidence collected: %d", len(incident.Evidence))
		} else {
			t.Log("No incident detected from provided events")
		}
	})

	t.Run("IncidentResponse", func(t *testing.T) {
		// Create a mock incident
		incident := &SecurityIncident{
			ID:          "INC-001",
			Title:       "Test Security Incident",
			Description: "Test incident for response testing",
			Severity:    "high",
			Status:      "detected",
			Category:    "data_breach",
			DetectedAt:  time.Now(),
			Timeline:    []IncidentEvent{},
			Evidence:    []ForensicEvidence{},
			ResponseActions: []ResponseAction{},
		}

		err := engine.RespondToIncident(ctx, incident)
		if err != nil {
			t.Fatalf("Incident response failed: %v", err)
		}

		if incident.Status != "response_in_progress" {
			t.Errorf("Expected status 'response_in_progress', got '%s'", incident.Status)
		}

		if len(incident.ResponseActions) == 0 {
			t.Error("Expected response actions to be created")
		}

		t.Logf("Incident response initiated: %d actions created", len(incident.ResponseActions))
		for _, action := range incident.ResponseActions {
			t.Logf("Action: %s (%s) - Status: %s", action.Description, action.Type, action.Status)
		}
	})

	t.Run("ForensicAnalysis", func(t *testing.T) {
		// Create a mock incident with evidence
		incident := &SecurityIncident{
			ID:          "INC-002",
			Title:       "Test Forensic Analysis",
			Description: "Test incident for forensic analysis",
			Severity:    "high",
			Status:      "detected",
			Category:    "malware",
			DetectedAt:  time.Now(),
			Evidence: []ForensicEvidence{
				{
					ID:          "evidence-1",
					Type:        "log_file",
					Source:      "/var/log/application.log",
					Description: "Application log file",
					CollectedAt: time.Now(),
					CollectedBy: "system",
				},
			},
		}

		err := engine.ConductForensicAnalysis(ctx, incident)
		if err != nil {
			t.Fatalf("Forensic analysis failed: %v", err)
		}

		if len(incident.Evidence) == 0 {
			t.Error("Expected forensic evidence")
		}

		t.Logf("Forensic analysis completed: %d evidence items", len(incident.Evidence))
		for _, evidence := range incident.Evidence {
			t.Logf("Evidence: %s (%s) - Analysis: %d", evidence.Description, evidence.Type, len(evidence.Analysis))
		}
	})

	t.Run("ComplianceReporting", func(t *testing.T) {
		// Create a mock incident with compliance implications
		incident := &SecurityIncident{
			ID:          "INC-003",
			Title:       "Data Breach Incident",
			Description: "Personal data breach requiring compliance reporting",
			Severity:    "critical",
			Status:      "detected",
			Category:    "data_breach",
			DetectedAt:  time.Now(),
			AffectedAssets: []string{"user_database", "personal_data"},
			Compliance: ComplianceImpact{
				RequiredNotifications: []ComplianceNotification{},
				RegulatoryFrameworks:  []string{"GDPR"},
				NotificationDeadlines: map[string]time.Time{
					"GDPR": time.Now().Add(72 * time.Hour),
				},
				ComplianceStatus:  "under_review",
				AuditRequirements: []string{},
			},
		}

		err := engine.GenerateComplianceReport(ctx, incident)
		if err != nil {
			t.Fatalf("Compliance reporting failed: %v", err)
		}

		if len(incident.Compliance.RequiredNotifications) == 0 {
			t.Error("Expected compliance notifications to be generated")
		}

		t.Logf("Compliance report generated: %d notifications required", len(incident.Compliance.RequiredNotifications))
		for _, notification := range incident.Compliance.RequiredNotifications {
			t.Logf("Notification: %s to %s by %s", notification.Framework, notification.Authority, notification.Deadline.Format("2006-01-02 15:04:05"))
		}
	})
}

func TestIntegrationWorkflow(t *testing.T) {
	projectRoot := "../../"
	
	// Create all engines
	threatEngine := NewThreatModelingEngine(projectRoot)
	incidentEngine := NewIncidentResponseEngine(projectRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("EndToEndWorkflow", func(t *testing.T) {
		// Step 1: Run threat modeling
		threatResult, err := threatEngine.RunComprehensiveThreatModeling(ctx)
		if err != nil {
			t.Fatalf("Threat modeling failed: %v", err)
		}

		// Step 2: Simulate security events based on threats
		var events []SecurityEvent
		for _, threat := range threatResult.ThreatModel.Threats {
			event := SecurityEvent{
				ID:          fmt.Sprintf("event-%s", threat.ID),
				Timestamp:   time.Now(),
				Source:      "threat_simulation",
				EventType:   "threat_indicator",
				Severity:    threatEngine.mapThreatSeverity(threat.RiskScore),
				Category:    threat.Category,
				Description: fmt.Sprintf("Threat indicator: %s", threat.Name),
			}
			events = append(events, event)
		}

		// Step 3: Detect incidents from events
		incident, err := incidentEngine.DetectIncident(ctx, events)
		if err != nil {
			t.Fatalf("Incident detection failed: %v", err)
		}

		if incident != nil {
			// Step 4: Respond to incident
			err = incidentEngine.RespondToIncident(ctx, incident)
			if err != nil {
				t.Fatalf("Incident response failed: %v", err)
			}

			// Step 5: Conduct forensic analysis
			err = incidentEngine.ConductForensicAnalysis(ctx, incident)
			if err != nil {
				t.Fatalf("Forensic analysis failed: %v", err)
			}

			// Step 6: Generate compliance report
			err = incidentEngine.GenerateComplianceReport(ctx, incident)
			if err != nil {
				t.Fatalf("Compliance reporting failed: %v", err)
			}

			t.Logf("End-to-end workflow completed successfully")
			t.Logf("Threat model: %d threats identified", len(threatResult.ThreatModel.Threats))
			t.Logf("Attack simulations: %d performed", len(threatResult.AttackSimulations))
			t.Logf("Incident: %s (Severity: %s)", incident.Title, incident.Severity)
			t.Logf("Response actions: %d initiated", len(incident.ResponseActions))
			t.Logf("Evidence collected: %d items", len(incident.Evidence))
			t.Logf("Compliance notifications: %d required", len(incident.Compliance.RequiredNotifications))
		} else {
			t.Log("No incidents detected in end-to-end workflow")
		}
	})
}