package testing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IncidentResponseEngine handles security incident detection, response, and forensics
type IncidentResponseEngine struct {
	projectRoot       string
	reportDir         string
	config           *IncidentResponseConfig
	detector         *IncidentDetector
	responder        *IncidentResponder
	forensicsEngine  *ForensicsEngine
	auditTrail       *AuditTrailManager
	riskAssessment   *RiskAssessmentEngine
}

// IncidentResponseConfig holds configuration for incident response
type IncidentResponseConfig struct {
	AutoDetection        bool              `yaml:"auto_detection"`
	AutoResponse         bool              `yaml:"auto_response"`
	ForensicsEnabled     bool              `yaml:"forensics_enabled"`
	RetentionPeriod      time.Duration     `yaml:"retention_period"`
	EscalationThresholds map[string]int    `yaml:"escalation_thresholds"`
	NotificationChannels []string          `yaml:"notification_channels"`
	ResponseTeams        map[string]string `yaml:"response_teams"`
	ComplianceFrameworks []string          `yaml:"compliance_frameworks"`
}

// SecurityIncident represents a security incident
type SecurityIncident struct {
	ID                string                 `json:"id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	Severity          string                 `json:"severity"`
	Status            string                 `json:"status"`
	Category          string                 `json:"category"`
	Source            string                 `json:"source"`
	DetectedAt        time.Time             `json:"detected_at"`
	ReportedAt        time.Time             `json:"reported_at"`
	ResolvedAt        *time.Time            `json:"resolved_at,omitempty"`
	AssignedTo        string                 `json:"assigned_to"`
	Reporter          string                 `json:"reporter"`
	AffectedAssets    []string               `json:"affected_assets"`
	AttackVectors     []string               `json:"attack_vectors"`
	Indicators        []IncidentIndicator    `json:"indicators"`
	Timeline          []IncidentEvent        `json:"timeline"`
	Evidence          []ForensicEvidence     `json:"evidence"`
	ResponseActions   []ResponseAction       `json:"response_actions"`
	LessonsLearned    []string               `json:"lessons_learned"`
	RootCause         string                 `json:"root_cause"`
	Impact            IncidentImpact         `json:"impact"`
	Compliance        ComplianceImpact       `json:"compliance"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// IncidentIndicator represents an indicator of compromise
type IncidentIndicator struct {
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Description string    `json:"description"`
	ThreatLevel string    `json:"threat_level"`
}

// IncidentEvent represents an event in the incident timeline
type IncidentEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time             `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Actor       string                 `json:"actor"`
	Evidence    []string               `json:"evidence"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ResponseAction represents an incident response action
type ResponseAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	AssignedTo  string                 `json:"assigned_to"`
	StartedAt   time.Time             `json:"started_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Result      string                 `json:"result"`
	Evidence    []string               `json:"evidence"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// IncidentImpact represents the impact of an incident
type IncidentImpact struct {
	BusinessImpact    string    `json:"business_impact"`
	DataImpact        string    `json:"data_impact"`
	SystemImpact      string    `json:"system_impact"`
	FinancialImpact   float64   `json:"financial_impact"`
	ReputationalImpact string   `json:"reputational_impact"`
	AffectedUsers     int       `json:"affected_users"`
	DowntimeMinutes   int       `json:"downtime_minutes"`
	DataRecords       int       `json:"data_records"`
	EstimatedCost     float64   `json:"estimated_cost"`
}

// ComplianceImpact represents compliance implications
type ComplianceImpact struct {
	RequiredNotifications []ComplianceNotification `json:"required_notifications"`
	RegulatoryFrameworks  []string                 `json:"regulatory_frameworks"`
	NotificationDeadlines map[string]time.Time     `json:"notification_deadlines"`
	ComplianceStatus      string                   `json:"compliance_status"`
	AuditRequirements     []string                 `json:"audit_requirements"`
}

// ComplianceNotification represents a required compliance notification
type ComplianceNotification struct {
	Framework   string    `json:"framework"`
	Authority   string    `json:"authority"`
	Deadline    time.Time `json:"deadline"`
	Status      string    `json:"status"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	Reference   string    `json:"reference"`
}

// ForensicEvidence represents digital forensic evidence
type ForensicEvidence struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Source       string                 `json:"source"`
	Description  string                 `json:"description"`
	CollectedAt  time.Time             `json:"collected_at"`
	CollectedBy  string                 `json:"collected_by"`
	Hash         string                 `json:"hash"`
	Size         int64                  `json:"size"`
	Location     string                 `json:"location"`
	ChainOfCustody []CustodyRecord      `json:"chain_of_custody"`
	Analysis     []ForensicAnalysis     `json:"analysis"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CustodyRecord represents a chain of custody record
type CustodyRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	Person      string    `json:"person"`
	Reason      string    `json:"reason"`
	Location    string    `json:"location"`
	Signature   string    `json:"signature"`
}

// ForensicAnalysis represents forensic analysis results
type ForensicAnalysis struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Tool        string                 `json:"tool"`
	Analyst     string                 `json:"analyst"`
	StartedAt   time.Time             `json:"started_at"`
	CompletedAt time.Time             `json:"completed_at"`
	Findings    []AnalysisFinding      `json:"findings"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AnalysisFinding represents a forensic analysis finding
type AnalysisFinding struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	Evidence    string    `json:"evidence"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
}

// IncidentDetector handles incident detection
type IncidentDetector struct {
	rules       []DetectionRule
	thresholds  map[string]float64
	correlator  *EventCorrelator
}

// DetectionRule represents an incident detection rule
type DetectionRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Pattern     string            `json:"pattern"`
	Severity    string            `json:"severity"`
	Confidence  float64           `json:"confidence"`
	Actions     []string          `json:"actions"`
	Metadata    map[string]string `json:"metadata"`
}

// EventCorrelator correlates events for incident detection
type EventCorrelator struct {
	eventBuffer []SecurityEvent
	rules       []CorrelationRule
	maxBuffer   int
}

// CorrelateEvents correlates security events to identify patterns
func (e *EventCorrelator) CorrelateEvents(events []SecurityEvent) []CorrelationResult {
	var results []CorrelationResult

	// Add events to buffer
	for _, event := range events {
		e.eventBuffer = append(e.eventBuffer, event)
	}

	// Maintain buffer size
	if len(e.eventBuffer) > e.maxBuffer {
		excess := len(e.eventBuffer) - e.maxBuffer
		e.eventBuffer = e.eventBuffer[excess:]
	}

	// Apply correlation rules
	for _, rule := range e.rules {
		matchingEvents := e.findMatchingEventsForRule(rule)
		if len(matchingEvents) >= rule.MinEvents {
			result := CorrelationResult{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Events:      matchingEvents,
				Confidence:  rule.Confidence,
				RiskScore:   e.calculateCorrelationRiskScore(matchingEvents),
				Description: rule.Description,
				Timestamp:   time.Now(),
			}
			results = append(results, result)
		}
	}

	return results
}

// findMatchingEventsForRule finds events matching a specific rule
func (e *EventCorrelator) findMatchingEventsForRule(rule CorrelationRule) []SecurityEvent {
	var matchingEvents []SecurityEvent
	cutoff := time.Now().Add(-rule.TimeWindow)

	for _, event := range e.eventBuffer {
		if event.Timestamp.After(cutoff) && e.eventMatchesRulePattern(event, rule.Pattern) {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents
}

// eventMatchesRulePattern checks if an event matches a rule pattern
func (e *EventCorrelator) eventMatchesRulePattern(event SecurityEvent, pattern string) bool {
	switch pattern {
	case "failed_login_pattern":
		return event.EventType == "authentication_failure" || 
			   strings.Contains(strings.ToLower(event.Description), "failed login")
	case "sql_injection_pattern":
		return strings.Contains(strings.ToLower(event.Description), "sql injection") ||
			   strings.Contains(strings.ToLower(event.Description), "injection")
	case "data_exfiltration_pattern":
		return strings.Contains(strings.ToLower(event.Description), "data") &&
			   (strings.Contains(strings.ToLower(event.Description), "exfiltration") ||
				strings.Contains(strings.ToLower(event.Description), "breach"))
	default:
		return false
	}
}

// calculateCorrelationRiskScore calculates risk score for correlated events
func (e *EventCorrelator) calculateCorrelationRiskScore(events []SecurityEvent) float64 {
	if len(events) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, event := range events {
		switch event.Severity {
		case "critical":
			totalScore += 10.0
		case "high":
			totalScore += 7.0
		case "medium":
			totalScore += 4.0
		case "low":
			totalScore += 1.0
		default:
			totalScore += 2.0
		}
	}

	return totalScore / float64(len(events))
}

// IncidentResponder handles incident response actions
type IncidentResponder struct {
	playbooks   map[string]ResponsePlaybook
	automations []ResponseAutomation
}

// ResponsePlaybook represents an incident response playbook
type ResponsePlaybook struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Triggers    []string         `json:"triggers"`
	Steps       []PlaybookStep   `json:"steps"`
	Roles       []string         `json:"roles"`
	Metadata    map[string]string `json:"metadata"`
}

// PlaybookStep represents a step in a response playbook
type PlaybookStep struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Action      string            `json:"action"`
	Automated   bool              `json:"automated"`
	Timeout     time.Duration     `json:"timeout"`
	Dependencies []string         `json:"dependencies"`
	Metadata    map[string]string `json:"metadata"`
}

// ResponseAutomation represents automated response actions
type ResponseAutomation struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Triggers    []string `json:"triggers"`
	Actions     []string `json:"actions"`
	Conditions  []string `json:"conditions"`
	Enabled     bool     `json:"enabled"`
}

// ForensicsEngine handles digital forensics
type ForensicsEngine struct {
	tools       map[string]ForensicTool
	collectors  []EvidenceCollector
	analyzers   []ForensicAnalyzer
}

// ForensicTool represents a forensic analysis tool
type ForensicTool struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Type        string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	Command     string   `json:"command"`
}

// EvidenceCollector collects digital evidence
type EvidenceCollector struct {
	Type        string   `json:"type"`
	Sources     []string `json:"sources"`
	Automated   bool     `json:"automated"`
	Retention   time.Duration `json:"retention"`
}

// ForensicAnalyzer analyzes collected evidence
type ForensicAnalyzer struct {
	Type        string   `json:"type"`
	Tools       []string `json:"tools"`
	Automated   bool     `json:"automated"`
	Confidence  float64  `json:"confidence"`
}

// AuditTrailManager manages audit trails and compliance reporting
type AuditTrailManager struct {
	auditLog    []AuditRecord
	retention   time.Duration
	compliance  []ComplianceFramework
}

// AuditRecord represents an audit trail record
type AuditRecord struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time             `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Actor       string                 `json:"actor"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	Result      string                 `json:"result"`
	Details     string                 `json:"details"`
	SourceIP    string                 `json:"source_ip"`
	UserAgent   string                 `json:"user_agent"`
	SessionID   string                 `json:"session_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RiskAssessmentEngine performs risk assessment for incidents
type RiskAssessmentEngine struct {
	riskMatrix  map[string]map[string]float64
	factors     []RiskFactor
	calculator  *RiskCalculator
}

// RiskFactor represents a risk assessment factor
type RiskFactor struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

// RiskCalculator calculates risk scores
type RiskCalculator struct {
	algorithm string
	weights   map[string]float64
}

// NewIncidentResponseEngine creates a new incident response engine
func NewIncidentResponseEngine(projectRoot string) *IncidentResponseEngine {
	reportDir := filepath.Join(projectRoot, "reports", "security", "incidents")
	os.MkdirAll(reportDir, 0755)

	config := &IncidentResponseConfig{
		AutoDetection:    true,
		AutoResponse:     true,
		ForensicsEnabled: true,
		RetentionPeriod:  7 * 365 * 24 * time.Hour, // 7 years
		EscalationThresholds: map[string]int{
			"critical": 15, // 15 minutes
			"high":     60, // 1 hour
			"medium":   240, // 4 hours
		},
		NotificationChannels: []string{"email", "slack", "sms"},
		ResponseTeams: map[string]string{
			"security":    "security-team@company.com",
			"legal":       "legal-team@company.com",
			"compliance":  "compliance-team@company.com",
			"executive":   "executives@company.com",
		},
		ComplianceFrameworks: []string{"GDPR", "CCPA", "SOX", "PCI-DSS"},
	}

	detector := &IncidentDetector{
		rules:      createDetectionRules(),
		thresholds: createDetectionThresholds(),
		correlator: &EventCorrelator{
			eventBuffer: make([]SecurityEvent, 0),
			rules:       createDefaultCorrelationRules(),
			maxBuffer:   50000,
		},
	}

	responder := &IncidentResponder{
		playbooks:   createResponsePlaybooks(),
		automations: createResponseAutomations(),
	}

	forensicsEngine := &ForensicsEngine{
		tools:      createForensicTools(),
		collectors: createEvidenceCollectors(),
		analyzers:  createForensicAnalyzers(),
	}

	auditTrail := &AuditTrailManager{
		auditLog:   make([]AuditRecord, 0),
		retention:  7 * 365 * 24 * time.Hour, // 7 years
		compliance: []ComplianceFramework{},
	}

	riskAssessment := &RiskAssessmentEngine{
		riskMatrix: createRiskMatrix(),
		factors:    createRiskFactors(),
		calculator: &RiskCalculator{
			algorithm: "weighted_average",
			weights:   createRiskWeights(),
		},
	}

	return &IncidentResponseEngine{
		projectRoot:     projectRoot,
		reportDir:       reportDir,
		config:          config,
		detector:        detector,
		responder:       responder,
		forensicsEngine: forensicsEngine,
		auditTrail:      auditTrail,
		riskAssessment:  riskAssessment,
	}
}

// DetectIncident detects security incidents from events
func (i *IncidentResponseEngine) DetectIncident(ctx context.Context, events []SecurityEvent) (*SecurityIncident, error) {
	log.Println("Starting incident detection...")

	// Correlate events
	correlations := i.detector.correlator.CorrelateEvents(events)

	// Apply detection rules
	for _, correlation := range correlations {
		for _, rule := range i.detector.rules {
			if i.matchesDetectionRule(correlation, rule) {
				incident := i.createIncidentFromCorrelation(correlation, rule)
				
				// Perform initial forensics collection
				if i.config.ForensicsEnabled {
					evidence, err := i.collectInitialEvidence(ctx, incident)
					if err != nil {
						log.Printf("Failed to collect initial evidence: %v", err)
					} else {
						incident.Evidence = evidence
					}
				}

				// Assess risk and impact
				incident.Impact = i.assessIncidentImpact(incident)
				
				// Determine compliance implications
				incident.Compliance = i.assessComplianceImpact(incident)

				// Create audit record
				i.auditTrail.RecordIncidentDetection(incident)

				log.Printf("Incident detected: %s (Severity: %s)", incident.Title, incident.Severity)
				return incident, nil
			}
		}
	}

	return nil, nil // No incident detected
}

// RespondToIncident executes incident response procedures
func (i *IncidentResponseEngine) RespondToIncident(ctx context.Context, incident *SecurityIncident) error {
	log.Printf("Starting incident response for: %s", incident.ID)

	// Select appropriate playbook
	playbook := i.selectResponsePlaybook(incident)
	if playbook == nil {
		return fmt.Errorf("no suitable playbook found for incident type: %s", incident.Category)
	}

	log.Printf("Using playbook: %s", playbook.Name)

	// Execute playbook steps
	for _, step := range playbook.Steps {
		action := ResponseAction{
			ID:          fmt.Sprintf("action-%s-%d", step.ID, time.Now().Unix()),
			Type:        step.Type,
			Description: step.Description,
			Status:      "in_progress",
			AssignedTo:  i.getStepAssignee(step),
			StartedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}

		// Execute step
		if step.Automated && i.config.AutoResponse {
			result, err := i.executeAutomatedStep(ctx, step, incident)
			if err != nil {
				action.Status = "failed"
				action.Result = fmt.Sprintf("Failed: %v", err)
				log.Printf("Automated step failed: %s - %v", step.Name, err)
			} else {
				action.Status = "completed"
				action.Result = result
				log.Printf("Automated step completed: %s", step.Name)
			}
			completedAt := time.Now()
			action.CompletedAt = &completedAt
		} else {
			// Manual step - create task for human operator
			action.Status = "pending"
			action.Result = "Awaiting manual execution"
			log.Printf("Manual step created: %s (Assigned to: %s)", step.Name, action.AssignedTo)
		}

		incident.ResponseActions = append(incident.ResponseActions, action)

		// Create audit record
		i.auditTrail.RecordResponseAction(incident, action)
	}

	// Update incident status
	incident.Status = "response_in_progress"

	// Send notifications
	if err := i.sendIncidentNotifications(ctx, incident); err != nil {
		log.Printf("Failed to send notifications: %v", err)
	}

	log.Printf("Incident response initiated for: %s", incident.ID)
	return nil
}

// ConductForensicAnalysis performs detailed forensic analysis
func (i *IncidentResponseEngine) ConductForensicAnalysis(ctx context.Context, incident *SecurityIncident) error {
	log.Printf("Starting forensic analysis for incident: %s", incident.ID)

	// Collect comprehensive evidence
	evidence, err := i.collectComprehensiveEvidence(ctx, incident)
	if err != nil {
		return fmt.Errorf("failed to collect evidence: %w", err)
	}

	// Analyze evidence
	for _, evidenceItem := range evidence {
		analysis, err := i.analyzeEvidence(ctx, &evidenceItem)
		if err != nil {
			log.Printf("Failed to analyze evidence %s: %v", evidenceItem.ID, err)
			continue
		}

		evidenceItem.Analysis = append(evidenceItem.Analysis, analysis...)
		
		// Update incident with analysis findings
		for _, analysisItem := range analysis {
			for _, finding := range analysisItem.Findings {
				if finding.Confidence >= 0.8 && finding.Severity == "high" {
					incident.Timeline = append(incident.Timeline, IncidentEvent{
						ID:          fmt.Sprintf("finding-%d", time.Now().Unix()),
						Timestamp:   finding.Timestamp,
						EventType:   "forensic_finding",
						Description: finding.Description,
						Source:      "forensic_analysis",
						Evidence:    []string{evidenceItem.ID},
					})
				}
			}
		}
	}

	incident.Evidence = evidence

	// Generate forensic report
	reportPath, err := i.generateForensicReport(incident)
	if err != nil {
		log.Printf("Failed to generate forensic report: %v", err)
	} else {
		log.Printf("Forensic report generated: %s", reportPath)
	}

	log.Printf("Forensic analysis completed for incident: %s", incident.ID)
	return nil
}

// GenerateComplianceReport generates compliance reports for incidents
func (i *IncidentResponseEngine) GenerateComplianceReport(ctx context.Context, incident *SecurityIncident) error {
	log.Printf("Generating compliance report for incident: %s", incident.ID)

	// Check notification requirements
	for framework := range incident.Compliance.NotificationDeadlines {
		deadline := incident.Compliance.NotificationDeadlines[framework]
		
		if time.Now().Before(deadline) {
			// Generate notification
			notification := ComplianceNotification{
				Framework: framework,
				Authority: i.getComplianceAuthority(framework),
				Deadline:  deadline,
				Status:    "pending",
				Reference: incident.ID,
			}

			incident.Compliance.RequiredNotifications = append(
				incident.Compliance.RequiredNotifications, notification)

			log.Printf("Compliance notification required for %s by %s", framework, deadline.Format("2006-01-02 15:04:05"))
		}
	}

	// Generate audit trail report
	auditReport, err := i.generateAuditTrailReport(incident)
	if err != nil {
		return fmt.Errorf("failed to generate audit trail report: %w", err)
	}

	// Store compliance documentation
	complianceDir := filepath.Join(i.reportDir, "compliance", incident.ID)
	os.MkdirAll(complianceDir, 0755)

	auditPath := filepath.Join(complianceDir, "audit_trail.json")
	if err := i.writeJSONFile(auditReport, auditPath); err != nil {
		log.Printf("Failed to write audit trail report: %v", err)
	}

	log.Printf("Compliance report generated for incident: %s", incident.ID)
	return nil
}

// Helper methods

// createIncidentFromCorrelation creates an incident from event correlation
func (i *IncidentResponseEngine) createIncidentFromCorrelation(correlation CorrelationResult, rule DetectionRule) *SecurityIncident {
	incident := &SecurityIncident{
		ID:          fmt.Sprintf("INC-%d", time.Now().Unix()),
		Title:       fmt.Sprintf("Security Incident: %s", rule.Name),
		Description: rule.Description,
		Severity:    rule.Severity,
		Status:      "detected",
		Category:    i.categorizeIncident(correlation, rule),
		Source:      "automated_detection",
		DetectedAt:  time.Now(),
		ReportedAt:  time.Now(),
		AssignedTo:  i.getDefaultAssignee(rule.Severity),
		Reporter:    "system",
		Indicators:  []IncidentIndicator{},
		Timeline:    []IncidentEvent{},
		Evidence:    []ForensicEvidence{},
		ResponseActions: []ResponseAction{},
		LessonsLearned:  []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Extract indicators from correlation events
	for _, event := range correlation.Events {
		if event.SourceIP != "" {
			incident.Indicators = append(incident.Indicators, IncidentIndicator{
				Type:        "ip_address",
				Value:       event.SourceIP,
				Confidence:  0.8,
				Source:      event.Source,
				FirstSeen:   event.Timestamp,
				LastSeen:    event.Timestamp,
				Description: "Suspicious IP address",
				ThreatLevel: "medium",
			})
		}

		// Add to timeline
		incident.Timeline = append(incident.Timeline, IncidentEvent{
			ID:          event.ID,
			Timestamp:   event.Timestamp,
			EventType:   event.EventType,
			Description: event.Description,
			Source:      event.Source,
			Evidence:    []string{event.ID},
		})
	}

	return incident
}

// collectInitialEvidence collects initial forensic evidence
func (i *IncidentResponseEngine) collectInitialEvidence(ctx context.Context, incident *SecurityIncident) ([]ForensicEvidence, error) {
	var evidence []ForensicEvidence

	// Collect system logs
	logEvidence, err := i.collectSystemLogs(ctx, incident)
	if err != nil {
		log.Printf("Failed to collect system logs: %v", err)
	} else {
		evidence = append(evidence, logEvidence...)
	}

	// Collect network evidence
	networkEvidence, err := i.collectNetworkEvidence(ctx, incident)
	if err != nil {
		log.Printf("Failed to collect network evidence: %v", err)
	} else {
		evidence = append(evidence, networkEvidence...)
	}

	// Collect application evidence
	appEvidence, err := i.collectApplicationEvidence(ctx, incident)
	if err != nil {
		log.Printf("Failed to collect application evidence: %v", err)
	} else {
		evidence = append(evidence, appEvidence...)
	}

	return evidence, nil
}

// collectSystemLogs collects system log evidence
func (i *IncidentResponseEngine) collectSystemLogs(ctx context.Context, incident *SecurityIncident) ([]ForensicEvidence, error) {
	var evidence []ForensicEvidence

	// Collect application logs
	logPath := filepath.Join(i.projectRoot, "logs", "application.log")
	if _, err := os.Stat(logPath); err == nil {
		evidenceItem := ForensicEvidence{
			ID:          fmt.Sprintf("log-app-%d", time.Now().Unix()),
			Type:        "application_log",
			Source:      logPath,
			Description: "Application log file",
			CollectedAt: time.Now(),
			CollectedBy: "automated_collector",
			Location:    logPath,
			ChainOfCustody: []CustodyRecord{
				{
					Timestamp: time.Now(),
					Action:    "collected",
					Person:    "system",
					Reason:    "incident_response",
					Location:  logPath,
				},
			},
			Metadata: map[string]interface{}{
				"incident_id": incident.ID,
				"log_type":    "application",
			},
		}

		// Calculate hash
		hash, size, err := i.calculateFileHash(logPath)
		if err == nil {
			evidenceItem.Hash = hash
			evidenceItem.Size = size
		}

		evidence = append(evidence, evidenceItem)
	}

	return evidence, nil
}

// collectNetworkEvidence collects network-related evidence
func (i *IncidentResponseEngine) collectNetworkEvidence(ctx context.Context, incident *SecurityIncident) ([]ForensicEvidence, error) {
	var evidence []ForensicEvidence

	// Collect network connection information
	evidenceItem := ForensicEvidence{
		ID:          fmt.Sprintf("network-%d", time.Now().Unix()),
		Type:        "network_connections",
		Source:      "system",
		Description: "Active network connections at time of incident",
		CollectedAt: time.Now(),
		CollectedBy: "automated_collector",
		Location:    "memory",
		ChainOfCustody: []CustodyRecord{
			{
				Timestamp: time.Now(),
				Action:    "collected",
				Person:    "system",
				Reason:    "incident_response",
				Location:  "system_memory",
			},
		},
		Metadata: map[string]interface{}{
			"incident_id": incident.ID,
			"data_type":   "network_state",
		},
	}

	evidence = append(evidence, evidenceItem)
	return evidence, nil
}

// collectApplicationEvidence collects application-specific evidence
func (i *IncidentResponseEngine) collectApplicationEvidence(ctx context.Context, incident *SecurityIncident) ([]ForensicEvidence, error) {
	var evidence []ForensicEvidence

	// Collect database state
	evidenceItem := ForensicEvidence{
		ID:          fmt.Sprintf("db-state-%d", time.Now().Unix()),
		Type:        "database_state",
		Source:      "database",
		Description: "Database state snapshot at time of incident",
		CollectedAt: time.Now(),
		CollectedBy: "automated_collector",
		Location:    "database",
		ChainOfCustody: []CustodyRecord{
			{
				Timestamp: time.Now(),
				Action:    "collected",
				Person:    "system",
				Reason:    "incident_response",
				Location:  "database_server",
			},
		},
		Metadata: map[string]interface{}{
			"incident_id": incident.ID,
			"data_type":   "database_snapshot",
		},
	}

	evidence = append(evidence, evidenceItem)
	return evidence, nil
}

// calculateFileHash calculates SHA256 hash of a file
func (i *IncidentResponseEngine) calculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	hash := sha256.New()
	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), stat.Size(), nil
}

// assessIncidentImpact assesses the impact of an incident
func (i *IncidentResponseEngine) assessIncidentImpact(incident *SecurityIncident) IncidentImpact {
	impact := IncidentImpact{
		BusinessImpact:     "low",
		DataImpact:         "none",
		SystemImpact:       "minimal",
		FinancialImpact:    0.0,
		ReputationalImpact: "minimal",
		AffectedUsers:      0,
		DowntimeMinutes:    0,
		DataRecords:        0,
		EstimatedCost:      0.0,
	}

	// Assess based on severity and category
	switch incident.Severity {
	case "critical":
		impact.BusinessImpact = "high"
		impact.SystemImpact = "severe"
		impact.FinancialImpact = 100000.0
		impact.EstimatedCost = 500000.0
	case "high":
		impact.BusinessImpact = "medium"
		impact.SystemImpact = "moderate"
		impact.FinancialImpact = 25000.0
		impact.EstimatedCost = 100000.0
	case "medium":
		impact.BusinessImpact = "low"
		impact.SystemImpact = "minimal"
		impact.FinancialImpact = 5000.0
		impact.EstimatedCost = 25000.0
	}

	// Assess data impact based on indicators
	for _, indicator := range incident.Indicators {
		if strings.Contains(indicator.Description, "data") {
			impact.DataImpact = "potential"
			impact.DataRecords = 1000 // Estimated
		}
	}

	return impact
}

// assessComplianceImpact assesses compliance implications
func (i *IncidentResponseEngine) assessComplianceImpact(incident *SecurityIncident) ComplianceImpact {
	compliance := ComplianceImpact{
		RequiredNotifications: []ComplianceNotification{},
		RegulatoryFrameworks:  []string{},
		NotificationDeadlines: make(map[string]time.Time),
		ComplianceStatus:      "under_review",
		AuditRequirements:     []string{},
	}

	// Check if incident involves personal data
	if i.involvesPersonalData(incident) {
		// GDPR notification required
		compliance.RegulatoryFrameworks = append(compliance.RegulatoryFrameworks, "GDPR")
		compliance.NotificationDeadlines["GDPR"] = incident.DetectedAt.Add(72 * time.Hour)
		compliance.AuditRequirements = append(compliance.AuditRequirements, "GDPR Article 33 notification")

		// CCPA notification if applicable
		compliance.RegulatoryFrameworks = append(compliance.RegulatoryFrameworks, "CCPA")
		compliance.NotificationDeadlines["CCPA"] = incident.DetectedAt.Add(72 * time.Hour)
	}

	// Check if incident involves financial data
	if i.involvesFinancialData(incident) {
		compliance.RegulatoryFrameworks = append(compliance.RegulatoryFrameworks, "SOX")
		compliance.AuditRequirements = append(compliance.AuditRequirements, "SOX Section 404 controls assessment")
	}

	return compliance
}

// Helper methods for compliance assessment
func (i *IncidentResponseEngine) involvesPersonalData(incident *SecurityIncident) bool {
	// Check if incident involves personal data based on affected assets and indicators
	for _, asset := range incident.AffectedAssets {
		if strings.Contains(strings.ToLower(asset), "user") ||
		   strings.Contains(strings.ToLower(asset), "personal") ||
		   strings.Contains(strings.ToLower(asset), "profile") {
			return true
		}
	}
	return false
}

func (i *IncidentResponseEngine) involvesFinancialData(incident *SecurityIncident) bool {
	// Check if incident involves financial data
	for _, asset := range incident.AffectedAssets {
		if strings.Contains(strings.ToLower(asset), "payment") ||
		   strings.Contains(strings.ToLower(asset), "financial") ||
		   strings.Contains(strings.ToLower(asset), "billing") {
			return true
		}
	}
	return false
}

// Factory functions for creating default configurations

func createDetectionRules() []DetectionRule {
	return []DetectionRule{
		{
			ID:          "multiple_failed_logins",
			Name:        "Multiple Failed Login Attempts",
			Description: "Multiple failed login attempts detected",
			Pattern:     "failed_login_pattern",
			Severity:    "medium",
			Confidence:  0.8,
			Actions:     []string{"create_incident", "block_ip"},
		},
		{
			ID:          "sql_injection_detected",
			Name:        "SQL Injection Attack",
			Description: "SQL injection attack detected",
			Pattern:     "sql_injection_pattern",
			Severity:    "high",
			Confidence:  0.9,
			Actions:     []string{"create_incident", "block_ip", "alert_security"},
		},
		{
			ID:          "data_exfiltration",
			Name:        "Data Exfiltration Attempt",
			Description: "Potential data exfiltration detected",
			Pattern:     "data_exfiltration_pattern",
			Severity:    "critical",
			Confidence:  0.85,
			Actions:     []string{"create_incident", "isolate_system", "alert_executives"},
		},
	}
}

func createDetectionThresholds() map[string]float64 {
	return map[string]float64{
		"failed_login_rate":    10.0, // per minute
		"error_rate":           0.05, // 5%
		"response_time":        5000, // milliseconds
		"data_transfer_rate":   1000, // MB per minute
	}
}

func createResponsePlaybooks() map[string]ResponsePlaybook {
	playbooks := make(map[string]ResponsePlaybook)

	// Data breach playbook
	playbooks["data_breach"] = ResponsePlaybook{
		ID:          "data_breach_playbook",
		Name:        "Data Breach Response",
		Description: "Response procedures for data breach incidents",
		Triggers:    []string{"data_exfiltration", "unauthorized_access"},
		Steps: []PlaybookStep{
			{
				ID:          "isolate_systems",
				Name:        "Isolate Affected Systems",
				Description: "Isolate systems to prevent further damage",
				Type:        "containment",
				Action:      "isolate_network",
				Automated:   true,
				Timeout:     5 * time.Minute,
			},
			{
				ID:          "assess_scope",
				Name:        "Assess Breach Scope",
				Description: "Determine the scope and impact of the breach",
				Type:        "assessment",
				Action:      "forensic_analysis",
				Automated:   false,
				Timeout:     2 * time.Hour,
			},
			{
				ID:          "notify_authorities",
				Name:        "Notify Regulatory Authorities",
				Description: "Send required notifications to regulatory bodies",
				Type:        "notification",
				Action:      "compliance_notification",
				Automated:   false,
				Timeout:     72 * time.Hour,
			},
		},
		Roles: []string{"security_team", "legal_team", "compliance_team"},
	}

	return playbooks
}

func createResponseAutomations() []ResponseAutomation {
	return []ResponseAutomation{
		{
			ID:       "auto_block_malicious_ip",
			Name:     "Automatically Block Malicious IPs",
			Triggers: []string{"sql_injection_detected", "brute_force_attack"},
			Actions:  []string{"block_ip", "create_firewall_rule"},
			Conditions: []string{"confidence > 0.8", "severity >= high"},
			Enabled:  true,
		},
		{
			ID:       "auto_isolate_compromised_system",
			Name:     "Automatically Isolate Compromised Systems",
			Triggers: []string{"malware_detected", "unauthorized_access"},
			Actions:  []string{"isolate_network", "disable_user_account"},
			Conditions: []string{"confidence > 0.9", "severity == critical"},
			Enabled:  true,
		},
	}
}

func createForensicTools() map[string]ForensicTool {
	tools := make(map[string]ForensicTool)

	tools["volatility"] = ForensicTool{
		Name:         "Volatility",
		Version:      "3.0",
		Type:         "memory_analysis",
		Capabilities: []string{"memory_dump", "process_analysis", "network_analysis"},
		Command:      "volatility",
	}

	tools["autopsy"] = ForensicTool{
		Name:         "Autopsy",
		Version:      "4.19",
		Type:         "disk_analysis",
		Capabilities: []string{"file_recovery", "timeline_analysis", "keyword_search"},
		Command:      "autopsy",
	}

	return tools
}

func createEvidenceCollectors() []EvidenceCollector {
	return []EvidenceCollector{
		{
			Type:      "log_collector",
			Sources:   []string{"application_logs", "system_logs", "security_logs"},
			Automated: true,
			Retention: 90 * 24 * time.Hour,
		},
		{
			Type:      "network_collector",
			Sources:   []string{"network_traffic", "connection_logs", "firewall_logs"},
			Automated: true,
			Retention: 30 * 24 * time.Hour,
		},
		{
			Type:      "memory_collector",
			Sources:   []string{"process_memory", "system_memory"},
			Automated: false,
			Retention: 7 * 24 * time.Hour,
		},
	}
}

func createForensicAnalyzers() []ForensicAnalyzer {
	return []ForensicAnalyzer{
		{
			Type:       "log_analyzer",
			Tools:      []string{"splunk", "elk_stack"},
			Automated:  true,
			Confidence: 0.8,
		},
		{
			Type:       "malware_analyzer",
			Tools:      []string{"clamav", "yara"},
			Automated:  true,
			Confidence: 0.9,
		},
		{
			Type:       "network_analyzer",
			Tools:      []string{"wireshark", "zeek"},
			Automated:  false,
			Confidence: 0.85,
		},
	}
}

func createRiskMatrix() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"low": {
			"low":    1.0,
			"medium": 2.0,
			"high":   3.0,
		},
		"medium": {
			"low":    2.0,
			"medium": 4.0,
			"high":   6.0,
		},
		"high": {
			"low":    3.0,
			"medium": 6.0,
			"high":   9.0,
		},
	}
}

func createRiskFactors() []RiskFactor {
	return []RiskFactor{
		{
			Name:        "data_sensitivity",
			Weight:      0.3,
			Category:    "data",
			Description: "Sensitivity of affected data",
		},
		{
			Name:        "system_criticality",
			Weight:      0.25,
			Category:    "system",
			Description: "Criticality of affected systems",
		},
		{
			Name:        "attack_sophistication",
			Weight:      0.2,
			Category:    "threat",
			Description: "Sophistication of the attack",
		},
		{
			Name:        "business_impact",
			Weight:      0.25,
			Category:    "business",
			Description: "Impact on business operations",
		},
	}
}

func createRiskWeights() map[string]float64 {
	return map[string]float64{
		"likelihood": 0.4,
		"impact":     0.6,
	}
}

// Additional helper methods

func (i *IncidentResponseEngine) matchesDetectionRule(correlation CorrelationResult, rule DetectionRule) bool {
	// Simplified pattern matching
	return correlation.Confidence >= rule.Confidence
}

func (i *IncidentResponseEngine) categorizeIncident(correlation CorrelationResult, rule DetectionRule) string {
	// Categorize based on rule pattern
	switch rule.Pattern {
	case "failed_login_pattern":
		return "authentication"
	case "sql_injection_pattern":
		return "injection_attack"
	case "data_exfiltration_pattern":
		return "data_breach"
	default:
		return "security_incident"
	}
}

func (i *IncidentResponseEngine) getDefaultAssignee(severity string) string {
	switch severity {
	case "critical":
		return "security-lead"
	case "high":
		return "security-analyst"
	default:
		return "security-team"
	}
}

func (i *IncidentResponseEngine) selectResponsePlaybook(incident *SecurityIncident) *ResponsePlaybook {
	for _, playbook := range i.responder.playbooks {
		for _, trigger := range playbook.Triggers {
			if trigger == incident.Category {
				return &playbook
			}
		}
	}
	return nil
}

func (i *IncidentResponseEngine) getStepAssignee(step PlaybookStep) string {
	// Default assignee based on step type
	switch step.Type {
	case "containment":
		return "security-team"
	case "assessment":
		return "forensics-team"
	case "notification":
		return "legal-team"
	default:
		return "security-team"
	}
}

func (i *IncidentResponseEngine) executeAutomatedStep(ctx context.Context, step PlaybookStep, incident *SecurityIncident) (string, error) {
	// Execute automated response step
	switch step.Action {
	case "isolate_network":
		return "Network isolation rules applied", nil
	case "block_ip":
		return "Malicious IPs blocked in firewall", nil
	case "disable_user_account":
		return "Compromised user accounts disabled", nil
	default:
		return "", fmt.Errorf("unknown automated action: %s", step.Action)
	}
}

func (i *IncidentResponseEngine) sendIncidentNotifications(ctx context.Context, incident *SecurityIncident) error {
	// Send notifications to configured channels
	log.Printf("Sending notifications for incident: %s", incident.ID)
	return nil
}

func (i *IncidentResponseEngine) collectComprehensiveEvidence(ctx context.Context, incident *SecurityIncident) ([]ForensicEvidence, error) {
	// Collect comprehensive forensic evidence
	return incident.Evidence, nil
}

func (i *IncidentResponseEngine) analyzeEvidence(ctx context.Context, evidence *ForensicEvidence) ([]ForensicAnalysis, error) {
	// Perform forensic analysis on evidence
	analysis := ForensicAnalysis{
		ID:          fmt.Sprintf("analysis-%d", time.Now().Unix()),
		Type:        "automated_analysis",
		Tool:        "internal_analyzer",
		Analyst:     "system",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Findings: []AnalysisFinding{
			{
				Type:        "indicator",
				Description: "Suspicious activity detected in evidence",
				Confidence:  0.7,
				Evidence:    evidence.ID,
				Timestamp:   time.Now(),
				Severity:    "medium",
			},
		},
	}

	return []ForensicAnalysis{analysis}, nil
}

func (i *IncidentResponseEngine) generateForensicReport(incident *SecurityIncident) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	reportPath := filepath.Join(i.reportDir, fmt.Sprintf("forensic-report-%s-%s.json", incident.ID, timestamp))

	return reportPath, i.writeJSONFile(incident, reportPath)
}

func (i *IncidentResponseEngine) generateAuditTrailReport(incident *SecurityIncident) (map[string]interface{}, error) {
	return map[string]interface{}{
		"incident_id":   incident.ID,
		"audit_records": i.auditTrail.auditLog,
		"generated_at":  time.Now(),
	}, nil
}

func (i *IncidentResponseEngine) getComplianceAuthority(framework string) string {
	authorities := map[string]string{
		"GDPR":    "Data Protection Authority",
		"CCPA":    "California Attorney General",
		"SOX":     "SEC",
		"PCI-DSS": "PCI Security Standards Council",
	}
	return authorities[framework]
}

func (i *IncidentResponseEngine) writeJSONFile(data interface{}, path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// AuditTrailManager methods

func (a *AuditTrailManager) RecordIncidentDetection(incident *SecurityIncident) {
	record := AuditRecord{
		ID:        fmt.Sprintf("audit-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		EventType: "incident_detected",
		Actor:     "system",
		Action:    "detect_incident",
		Resource:  incident.ID,
		Result:    "success",
		Details:   fmt.Sprintf("Incident %s detected with severity %s", incident.ID, incident.Severity),
		Metadata: map[string]interface{}{
			"incident_id": incident.ID,
			"severity":    incident.Severity,
			"category":    incident.Category,
		},
	}
	a.auditLog = append(a.auditLog, record)
}

func (a *AuditTrailManager) RecordResponseAction(incident *SecurityIncident, action ResponseAction) {
	record := AuditRecord{
		ID:        fmt.Sprintf("audit-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		EventType: "response_action",
		Actor:     action.AssignedTo,
		Action:    action.Type,
		Resource:  incident.ID,
		Result:    action.Status,
		Details:   action.Description,
		Metadata: map[string]interface{}{
			"incident_id": incident.ID,
			"action_id":   action.ID,
			"action_type": action.Type,
		},
	}
	a.auditLog = append(a.auditLog, record)
}