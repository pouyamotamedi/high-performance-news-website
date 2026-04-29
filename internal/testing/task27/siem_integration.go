package task27

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SIEMIntegration provides security automation and SIEM system integration
type SIEMIntegration struct {
	projectRoot string
	config      *SIEMIntegrationConfig
	connectors  map[string]SIEMConnector
	alertManager *SIEMAlertManager
	ruleEngine  *SIEMRuleEngine
}

// SIEMIntegrationConfig holds configuration for SIEM integration
type SIEMIntegrationConfig struct {
	EnabledSIEMs     []string          `yaml:"enabled_siems"`
	AlertThresholds  map[string]int    `yaml:"alert_thresholds"`
	RetentionPeriod  time.Duration     `yaml:"retention_period"`
	CorrelationRules []string          `yaml:"correlation_rules"`
	AutoResponse     bool              `yaml:"auto_response"`
	WebhookURLs      map[string]string `yaml:"webhook_urls"`
}

// SIEMIntegrationResult represents the result of SIEM integration
type SIEMIntegrationResult struct {
	Status           string                 `json:"status"`
	ConnectedSIEMs   []string               `json:"connected_siems"`
	EventsForwarded  int                    `json:"events_forwarded"`
	AlertsGenerated  int                    `json:"alerts_generated"`
	RulesTriggered   []string               `json:"rules_triggered"`
	CorrelationResults []CorrelationResult  `json:"correlation_results"`
	AutoResponses    []AutoResponse         `json:"auto_responses"`
	Metadata         map[string]interface{} `json:"metadata"`
	Timestamp        time.Time              `json:"timestamp"`
}

// SIEMConnector interface for different SIEM systems
type SIEMConnector interface {
	Connect(ctx context.Context) error
	SendEvent(ctx context.Context, event *SecurityEvent) error
	SendAlert(ctx context.Context, alert *SecurityAlert) error
	GetName() string
	IsConnected() bool
	Disconnect() error
}

// SecurityEvent represents a security event for SIEM
type SecurityEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	SourceIP    string                 `json:"source_ip,omitempty"`
	TargetIP    string                 `json:"target_ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Method      string                 `json:"method,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Payload     string                 `json:"payload,omitempty"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityAlert represents a security alert for SIEM
type SecurityAlert struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	AlertType       string                 `json:"alert_type"`
	Severity        string                 `json:"severity"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Source          string                 `json:"source"`
	SourceEvents    []string               `json:"source_events"`
	RiskScore       float64                `json:"risk_score"`
	Confidence      float64                `json:"confidence"`
	Status          string                 `json:"status"`
	AssignedTo      string                 `json:"assigned_to,omitempty"`
	Indicators      []ThreatIndicator      `json:"indicators"`
	Recommendations []string               `json:"recommendations"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ThreatIndicator represents an indicator of compromise
type ThreatIndicator struct {
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Description string    `json:"description"`
}

// CorrelationResult represents the result of event correlation
type CorrelationResult struct {
	RuleID      string          `json:"rule_id"`
	RuleName    string          `json:"rule_name"`
	Events      []SecurityEvent `json:"events"`
	Confidence  float64         `json:"confidence"`
	RiskScore   float64         `json:"risk_score"`
	Description string          `json:"description"`
	Timestamp   time.Time       `json:"timestamp"`
}

// AutoResponse represents an automated response action
type AutoResponse struct {
	ID          string                 `json:"id"`
	TriggerRule string                 `json:"trigger_rule"`
	Action      string                 `json:"action"`
	Target      string                 `json:"target"`
	Status      string                 `json:"status"`
	Result      string                 `json:"result"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SIEMAlertManager manages SIEM alerts
type SIEMAlertManager struct {
	thresholds map[string]int
	rules      []AlertRule
}

// SIEMRuleEngine processes correlation rules
type SIEMRuleEngine struct {
	rules       []CorrelationRule
	eventBuffer []SecurityEvent
	maxBuffer   int
}

// AlertRule represents an alerting rule
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Condition   string            `json:"condition"`
	Threshold   int               `json:"threshold"`
	TimeWindow  time.Duration     `json:"time_window"`
	Severity    string            `json:"severity"`
	Actions     []string          `json:"actions"`
	Metadata    map[string]string `json:"metadata"`
}

// CorrelationRule represents an event correlation rule
type CorrelationRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Pattern     string        `json:"pattern"`
	TimeWindow  time.Duration `json:"time_window"`
	MinEvents   int           `json:"min_events"`
	Confidence  float64       `json:"confidence"`
	Description string        `json:"description"`
}

// Specific SIEM connector implementations

// SplunkConnector implements Splunk integration
type SplunkConnector struct {
	baseURL    string
	token      string
	httpClient *http.Client
	connected  bool
}

// ElasticSIEMConnector implements Elastic SIEM integration
type ElasticSIEMConnector struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	connected  bool
}

// NewSIEMIntegration creates a new SIEM integration
func NewSIEMIntegration(projectRoot string) *SIEMIntegration {
	config := &SIEMIntegrationConfig{
		EnabledSIEMs: []string{"splunk", "elastic"},
		AlertThresholds: map[string]int{
			"failed_login":    10,
			"sql_injection":   1,
			"xss_attempt":     5,
			"brute_force":     20,
		},
		RetentionPeriod: 90 * 24 * time.Hour, // 90 days
		CorrelationRules: []string{
			"multiple_failed_logins",
			"privilege_escalation_sequence",
			"data_exfiltration_pattern",
		},
		AutoResponse: true,
		WebhookURLs: map[string]string{
			"slack":     "",
			"teams":     "",
			"pagerduty": "",
		},
	}

	connectors := make(map[string]SIEMConnector)
	connectors["splunk"] = &SplunkConnector{
		baseURL:    "http://localhost:8089",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	connectors["elastic"] = &ElasticSIEMConnector{
		baseURL:    "http://localhost:9200",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	alertManager := &SIEMAlertManager{
		thresholds: config.AlertThresholds,
		rules:      createDefaultAlertRules(),
	}

	ruleEngine := &SIEMRuleEngine{
		rules:       createDefaultCorrelationRules(),
		eventBuffer: make([]SecurityEvent, 0),
		maxBuffer:   10000,
	}

	return &SIEMIntegration{
		projectRoot:  projectRoot,
		config:       config,
		connectors:   connectors,
		alertManager: alertManager,
		ruleEngine:   ruleEngine,
	}
}

// IntegrateWithSIEM performs SIEM integration for threat model
func (s *SIEMIntegration) IntegrateWithSIEM(ctx context.Context, threatModel *ThreatModel) (*SIEMIntegrationResult, error) {
	result := &SIEMIntegrationResult{
		Status:          "running",
		ConnectedSIEMs:  []string{},
		Timestamp:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	log.Println("Starting SIEM integration...")

	// Connect to enabled SIEM systems
	for _, siemName := range s.config.EnabledSIEMs {
		if connector, exists := s.connectors[siemName]; exists {
			if err := connector.Connect(ctx); err != nil {
				log.Printf("Failed to connect to %s: %v", siemName, err)
				continue
			}
			result.ConnectedSIEMs = append(result.ConnectedSIEMs, siemName)
			log.Printf("Connected to %s SIEM", siemName)
		}
	}

	// Generate security events from threat model
	events := s.generateSecurityEventsFromThreatModel(threatModel)
	result.EventsForwarded = len(events)

	// Forward events to connected SIEMs
	for _, event := range events {
		s.forwardEventToSIEMs(ctx, &event)
		s.ruleEngine.ProcessEvent(&event)
	}

	// Process correlation rules
	correlationResults := s.ruleEngine.ProcessCorrelationRules()
	result.CorrelationResults = correlationResults

	// Generate alerts based on correlation results
	alerts := s.generateAlertsFromCorrelation(correlationResults)
	result.AlertsGenerated = len(alerts)

	// Forward alerts to SIEMs
	for _, alert := range alerts {
		s.forwardAlertToSIEMs(ctx, &alert)
	}

	// Execute auto-responses if enabled
	if s.config.AutoResponse {
		autoResponses := s.executeAutoResponses(ctx, alerts)
		result.AutoResponses = autoResponses
	}

	// Collect triggered rules
	for _, correlation := range correlationResults {
		result.RulesTriggered = append(result.RulesTriggered, correlation.RuleName)
	}

	result.Status = "completed"
	log.Printf("SIEM integration completed: %d events forwarded, %d alerts generated", 
		result.EventsForwarded, result.AlertsGenerated)

	return result, nil
}

// generateSecurityEventsFromThreatModel creates security events from threat model
func (s *SIEMIntegration) generateSecurityEventsFromThreatModel(threatModel *ThreatModel) []SecurityEvent {
	var events []SecurityEvent

	// Generate events for each threat
	for _, threat := range threatModel.Threats {
		event := SecurityEvent{
			ID:          fmt.Sprintf("threat-%s-%d", threat.ID, time.Now().Unix()),
			Timestamp:   time.Now(),
			Source:      "threat_modeling",
			EventType:   "threat_identified",
			Severity:    s.mapThreatSeverity(threat.RiskScore),
			Category:    threat.Category,
			Description: fmt.Sprintf("Threat identified: %s", threat.Name),
			Tags:        []string{"threat_modeling", threat.STRIDECategory},
			Metadata: map[string]interface{}{
				"threat_id":      threat.ID,
				"risk_score":     threat.RiskScore,
				"likelihood":     threat.Likelihood,
				"impact":         threat.Impact,
				"mitre_attack":   threat.MITREAttackID,
				"target_assets":  threat.TargetAssets,
				"attack_vectors": threat.AttackVectors,
			},
		}
		events = append(events, event)
	}

	// Generate events for vulnerabilities
	for _, vuln := range threatModel.Vulnerabilities {
		event := SecurityEvent{
			ID:          fmt.Sprintf("vuln-%s-%d", vuln.ID, time.Now().Unix()),
			Timestamp:   time.Now(),
			Source:      "vulnerability_assessment",
			EventType:   "vulnerability_detected",
			Severity:    vuln.Severity,
			Category:    "vulnerability",
			Description: fmt.Sprintf("Vulnerability detected: %s", vuln.Title),
			Tags:        []string{"vulnerability", vuln.Scanner},
			Metadata: map[string]interface{}{
				"vulnerability_id": vuln.ID,
				"cvss_score":      vuln.CVSS,
				"package":         vuln.Package,
				"version":         vuln.Version,
				"fixed_version":   vuln.FixedVersion,
				"upgradable":      vuln.Upgradable,
			},
		}
		events = append(events, event)
	}

	// Generate events for attack paths
	for _, attackPath := range threatModel.AttackPaths {
		event := SecurityEvent{
			ID:          fmt.Sprintf("attack-path-%s-%d", attackPath.ID, time.Now().Unix()),
			Timestamp:   time.Now(),
			Source:      "attack_path_analysis",
			EventType:   "attack_path_identified",
			Severity:    s.mapAttackPathSeverity(attackPath.Probability, attackPath.Impact),
			Category:    "attack_path",
			Description: fmt.Sprintf("Attack path identified: %s", attackPath.Name),
			Tags:        []string{"attack_path", attackPath.Complexity},
			Metadata: map[string]interface{}{
				"attack_path_id":       attackPath.ID,
				"start_asset":          attackPath.StartAsset,
				"target_asset":         attackPath.TargetAsset,
				"probability":          attackPath.Probability,
				"impact":               attackPath.Impact,
				"detection_difficulty": attackPath.DetectionDifficulty,
			},
		}
		events = append(events, event)
	}

	return events
}

// forwardEventToSIEMs forwards an event to all connected SIEMs
func (s *SIEMIntegration) forwardEventToSIEMs(ctx context.Context, event *SecurityEvent) {
	for _, siemName := range s.config.EnabledSIEMs {
		if connector, exists := s.connectors[siemName]; exists && connector.IsConnected() {
			if err := connector.SendEvent(ctx, event); err != nil {
				log.Printf("Failed to send event to %s: %v", siemName, err)
			}
		}
	}
}

// forwardAlertToSIEMs forwards an alert to all connected SIEMs
func (s *SIEMIntegration) forwardAlertToSIEMs(ctx context.Context, alert *SecurityAlert) {
	for _, siemName := range s.config.EnabledSIEMs {
		if connector, exists := s.connectors[siemName]; exists && connector.IsConnected() {
			if err := connector.SendAlert(ctx, alert); err != nil {
				log.Printf("Failed to send alert to %s: %v", siemName, err)
			}
		}
	}
}

// generateAlertsFromCorrelation generates alerts from correlation results
func (s *SIEMIntegration) generateAlertsFromCorrelation(correlations []CorrelationResult) []SecurityAlert {
	var alerts []SecurityAlert

	for _, correlation := range correlations {
		if correlation.Confidence >= 0.7 { // High confidence threshold
			alert := SecurityAlert{
				ID:          fmt.Sprintf("alert-%s-%d", correlation.RuleID, time.Now().Unix()),
				Timestamp:   time.Now(),
				AlertType:   "correlation",
				Severity:    s.mapRiskScoreToSeverity(correlation.RiskScore),
				Title:       fmt.Sprintf("Security Pattern Detected: %s", correlation.RuleName),
				Description: correlation.Description,
				Source:      "correlation_engine",
				RiskScore:   correlation.RiskScore,
				Confidence:  correlation.Confidence,
				Status:      "open",
				Indicators:  s.extractIndicatorsFromEvents(correlation.Events),
				Recommendations: s.generateAlertRecommendations(correlation),
				Metadata: map[string]interface{}{
					"rule_id":     correlation.RuleID,
					"event_count": len(correlation.Events),
				},
			}

			// Add source event IDs
			for _, event := range correlation.Events {
				alert.SourceEvents = append(alert.SourceEvents, event.ID)
			}

			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// executeAutoResponses executes automated responses for alerts
func (s *SIEMIntegration) executeAutoResponses(ctx context.Context, alerts []SecurityAlert) []AutoResponse {
	var responses []AutoResponse

	for _, alert := range alerts {
		if alert.Severity == "critical" || alert.Severity == "high" {
			// Send notification
			notificationResponse := AutoResponse{
				ID:          fmt.Sprintf("notification-%d", time.Now().Unix()),
				TriggerRule: alert.ID,
				Action:      "send_notification",
				Target:      "security_team",
				Status:      "executed",
				Result:      "Notification sent to security team",
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"alert_id": alert.ID,
					"channel":  "slack",
				},
			}
			responses = append(responses, notificationResponse)
		}
	}

	return responses
}

// Helper methods

func (s *SIEMIntegration) mapThreatSeverity(riskScore float64) string {
	switch {
	case riskScore >= 9.0:
		return "critical"
	case riskScore >= 7.0:
		return "high"
	case riskScore >= 4.0:
		return "medium"
	default:
		return "low"
	}
}

func (s *SIEMIntegration) mapAttackPathSeverity(probability float64, impact string) string {
	if probability >= 0.7 && (impact == "critical" || impact == "high") {
		return "high"
	} else if probability >= 0.5 {
		return "medium"
	}
	return "low"
}

func (s *SIEMIntegration) mapRiskScoreToSeverity(riskScore float64) string {
	switch {
	case riskScore >= 8.0:
		return "critical"
	case riskScore >= 6.0:
		return "high"
	case riskScore >= 4.0:
		return "medium"
	default:
		return "low"
	}
}

func (s *SIEMIntegration) extractIndicatorsFromEvents(events []SecurityEvent) []ThreatIndicator {
	var indicators []ThreatIndicator

	for _, event := range events {
		if event.SourceIP != "" {
			indicators = append(indicators, ThreatIndicator{
				Type:        "ip",
				Value:       event.SourceIP,
				Confidence:  0.8,
				Source:      event.Source,
				FirstSeen:   event.Timestamp,
				LastSeen:    event.Timestamp,
				Description: "Suspicious IP address",
			})
		}

		if event.UserAgent != "" && strings.Contains(event.UserAgent, "bot") {
			indicators = append(indicators, ThreatIndicator{
				Type:        "user_agent",
				Value:       event.UserAgent,
				Confidence:  0.6,
				Source:      event.Source,
				FirstSeen:   event.Timestamp,
				LastSeen:    event.Timestamp,
				Description: "Suspicious user agent",
			})
		}
	}

	return indicators
}

func (s *SIEMIntegration) generateAlertRecommendations(correlation CorrelationResult) []string {
	var recommendations []string

	switch correlation.RuleID {
	case "multiple_failed_logins":
		recommendations = append(recommendations, "Investigate user account for compromise")
		recommendations = append(recommendations, "Consider implementing account lockout")
	case "privilege_escalation_sequence":
		recommendations = append(recommendations, "Investigate privilege escalation attempt")
		recommendations = append(recommendations, "Review user permissions and access controls")
	case "data_exfiltration_pattern":
		recommendations = append(recommendations, "Investigate potential data exfiltration")
		recommendations = append(recommendations, "Review data access logs")
	default:
		recommendations = append(recommendations, "Investigate security event correlation")
	}

	return recommendations
}

// SIEM Rule Engine methods

// ProcessEvent processes a single security event
func (r *SIEMRuleEngine) ProcessEvent(event *SecurityEvent) {
	// Add event to buffer
	r.eventBuffer = append(r.eventBuffer, *event)

	// Maintain buffer size
	if len(r.eventBuffer) > r.maxBuffer {
		r.eventBuffer = r.eventBuffer[1:]
	}
}

// ProcessCorrelationRules processes all correlation rules
func (r *SIEMRuleEngine) ProcessCorrelationRules() []CorrelationResult {
	var results []CorrelationResult

	for _, rule := range r.rules {
		result := r.processCorrelationRule(rule)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}

// processCorrelationRule processes a single correlation rule
func (r *SIEMRuleEngine) processCorrelationRule(rule CorrelationRule) *CorrelationResult {
	// Simple pattern matching - in practice would be more sophisticated
	matchingEvents := r.findMatchingEvents(rule)

	if len(matchingEvents) >= rule.MinEvents {
		return &CorrelationResult{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Events:      matchingEvents,
			Confidence:  rule.Confidence,
			RiskScore:   r.calculateRiskScore(matchingEvents),
			Description: rule.Description,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// findMatchingEvents finds events matching a correlation rule
func (r *SIEMRuleEngine) findMatchingEvents(rule CorrelationRule) []SecurityEvent {
	var matchingEvents []SecurityEvent

	cutoff := time.Now().Add(-rule.TimeWindow)

	for _, event := range r.eventBuffer {
		if event.Timestamp.After(cutoff) && r.eventMatchesPattern(event, rule.Pattern) {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents
}

// eventMatchesPattern checks if an event matches a pattern
func (r *SIEMRuleEngine) eventMatchesPattern(event SecurityEvent, pattern string) bool {
	// Simplified pattern matching
	switch pattern {
	case "failed_login":
		return event.EventType == "authentication_failure"
	case "privilege_escalation":
		return strings.Contains(event.Description, "privilege") || strings.Contains(event.Description, "escalation")
	case "data_access":
		return event.EventType == "data_access" || strings.Contains(event.Description, "data")
	default:
		return false
	}
}

// calculateRiskScore calculates risk score for correlated events
func (r *SIEMRuleEngine) calculateRiskScore(events []SecurityEvent) float64 {
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
		}
	}

	return totalScore / float64(len(events))
}

// Factory functions for default rules

func createDefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			ID:         "failed_login_threshold",
			Name:       "Multiple Failed Logins",
			Condition:  "failed_login_count > 10",
			Threshold:  10,
			TimeWindow: 5 * time.Minute,
			Severity:   "medium",
			Actions:    []string{"alert", "block_ip"},
		},
		{
			ID:         "sql_injection_detected",
			Name:       "SQL Injection Attempt",
			Condition:  "sql_injection_count > 0",
			Threshold:  1,
			TimeWindow: 1 * time.Minute,
			Severity:   "high",
			Actions:    []string{"alert", "block_ip", "notify_security"},
		},
	}
}

func createDefaultCorrelationRules() []CorrelationRule {
	return []CorrelationRule{
		{
			ID:          "multiple_failed_logins",
			Name:        "Multiple Failed Login Attempts",
			Pattern:     "failed_login",
			TimeWindow:  5 * time.Minute,
			MinEvents:   5,
			Confidence:  0.8,
			Description: "Multiple failed login attempts detected from same source",
		},
		{
			ID:          "privilege_escalation_sequence",
			Name:        "Privilege Escalation Sequence",
			Pattern:     "privilege_escalation",
			TimeWindow:  10 * time.Minute,
			MinEvents:   2,
			Confidence:  0.9,
			Description: "Sequence of privilege escalation attempts detected",
		},
		{
			ID:          "data_exfiltration_pattern",
			Name:        "Data Exfiltration Pattern",
			Pattern:     "data_access",
			TimeWindow:  15 * time.Minute,
			MinEvents:   10,
			Confidence:  0.7,
			Description: "Unusual data access pattern suggesting exfiltration",
		},
	}
}

// SIEM Connector implementations (simplified)

// Splunk Connector
func (s *SplunkConnector) Connect(ctx context.Context) error {
	s.connected = true // Simplified
	return nil
}

func (s *SplunkConnector) SendEvent(ctx context.Context, event *SecurityEvent) error {
	if !s.connected {
		return fmt.Errorf("not connected to Splunk")
	}
	// Would send to Splunk HEC endpoint
	log.Printf("Sent event %s to Splunk", event.ID)
	return nil
}

func (s *SplunkConnector) SendAlert(ctx context.Context, alert *SecurityAlert) error {
	if !s.connected {
		return fmt.Errorf("not connected to Splunk")
	}
	log.Printf("Sent alert %s to Splunk", alert.ID)
	return nil
}

func (s *SplunkConnector) GetName() string { return "splunk" }
func (s *SplunkConnector) IsConnected() bool { return s.connected }
func (s *SplunkConnector) Disconnect() error { s.connected = false; return nil }

// Elastic SIEM Connector
func (e *ElasticSIEMConnector) Connect(ctx context.Context) error {
	e.connected = true // Simplified
	return nil
}

func (e *ElasticSIEMConnector) SendEvent(ctx context.Context, event *SecurityEvent) error {
	if !e.connected {
		return fmt.Errorf("not connected to Elastic")
	}
	log.Printf("Sent event %s to Elastic SIEM", event.ID)
	return nil
}

func (e *ElasticSIEMConnector) SendAlert(ctx context.Context, alert *SecurityAlert) error {
	if !e.connected {
		return fmt.Errorf("not connected to Elastic")
	}
	log.Printf("Sent alert %s to Elastic SIEM", alert.ID)
	return nil
}

func (e *ElasticSIEMConnector) GetName() string { return "elastic" }
func (e *ElasticSIEMConnector) IsConnected() bool { return e.connected }
func (e *ElasticSIEMConnector) Disconnect() error { e.connected = false; return nil }