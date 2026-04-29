package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// AIAlertManager manages alerts for AI code anomalies and regressions
type AIAlertManager struct {
	alertChannels   []AlertChannel
	alertHistory    []Alert
	alertRules      []AlertRule
	rateLimiter     *AlertRateLimiter
	escalationRules []EscalationRule
	mu              sync.RWMutex
}

// AlertChannel defines how alerts are sent
type AlertChannel interface {
	SendAlert(alert Alert) error
	GetChannelType() string
	IsEnabled() bool
}

// Alert represents an alert to be sent
type Alert struct {
	ID              string                 `json:"id"`
	Type            AlertType              `json:"type"`
	Severity        AlertSeverity          `json:"severity"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Source          string                 `json:"source"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata"`
	Resolved        bool                   `json:"resolved"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	Escalated       bool                   `json:"escalated"`
	EscalatedAt     *time.Time             `json:"escalated_at,omitempty"`
	Acknowledgments []Acknowledgment       `json:"acknowledgments"`
	Actions         []AlertAction          `json:"actions"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeAnomaly     AlertType = "anomaly"
	AlertTypeRegression  AlertType = "regression"
	AlertTypeThreshold   AlertType = "threshold"
	AlertTypePattern     AlertType = "pattern"
	AlertTypeSystem      AlertType = "system"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   AlertCondition         `json:"condition"`
	Severity    AlertSeverity          `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AlertCondition defines the condition for triggering an alert
type AlertCondition struct {
	Metric      string      `json:"metric"`
	Operator    string      `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold   interface{} `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Aggregation string      `json:"aggregation"` // avg, max, min, sum, count
}

// EscalationRule defines when and how to escalate alerts
type EscalationRule struct {
	ID              string        `json:"id"`
	AlertSeverity   AlertSeverity `json:"alert_severity"`
	EscalateAfter   time.Duration `json:"escalate_after"`
	EscalationLevel int           `json:"escalation_level"`
	Channels        []string      `json:"channels"`
	Enabled         bool          `json:"enabled"`
}

// AlertRateLimiter prevents alert spam
type AlertRateLimiter struct {
	limits      map[string]*RateLimit
	mu          sync.RWMutex
}

// RateLimit defines rate limiting for specific alert types
type RateLimit struct {
	MaxAlerts   int           `json:"max_alerts"`
	TimeWindow  time.Duration `json:"time_window"`
	AlertCount  int           `json:"alert_count"`
	WindowStart time.Time     `json:"window_start"`
}

// Acknowledgment represents an alert acknowledgment
type Acknowledgment struct {
	User        string    `json:"user"`
	Timestamp   time.Time `json:"timestamp"`
	Comment     string    `json:"comment"`
}

// AlertAction represents an action that can be taken on an alert
type AlertAction struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // "webhook", "script", "api_call"
	Config      map[string]interface{} `json:"config"`
	Automatic   bool                   `json:"automatic"`
}

// EmailAlertChannel sends alerts via email
type EmailAlertChannel struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FromEmail    string   `json:"from_email"`
	ToEmails     []string `json:"to_emails"`
	Enabled      bool     `json:"enabled"`
}

// SlackAlertChannel sends alerts to Slack
type SlackAlertChannel struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	Enabled    bool   `json:"enabled"`
}

// WebhookAlertChannel sends alerts to a webhook endpoint
type WebhookAlertChannel struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Enabled bool              `json:"enabled"`
}

// LogAlertChannel logs alerts to the application log
type LogAlertChannel struct {
	LogLevel string `json:"log_level"`
	Enabled  bool   `json:"enabled"`
}

// NewAIAlertManager creates a new AI alert manager
func NewAIAlertManager() *AIAlertManager {
	manager := &AIAlertManager{
		alertChannels: make([]AlertChannel, 0),
		alertHistory:  make([]Alert, 0),
		alertRules:    make([]AlertRule, 0),
		rateLimiter: &AlertRateLimiter{
			limits: make(map[string]*RateLimit),
		},
		escalationRules: make([]EscalationRule, 0),
	}
	
	manager.initializeDefaultRules()
	manager.initializeDefaultChannels()
	manager.initializeRateLimits()
	
	return manager
}

// initializeDefaultRules sets up default alert rules
func (am *AIAlertManager) initializeDefaultRules() {
	am.alertRules = []AlertRule{
		{
			ID:          "query_performance_critical",
			Name:        "Critical Query Performance Degradation",
			Description: "Database query performance is critically degraded",
			Condition: AlertCondition{
				Metric:      "query_time",
				Operator:    ">",
				Threshold:   500.0, // 500ms
				Duration:    5 * time.Minute,
				Aggregation: "avg",
			},
			Severity: AlertSeverityCritical,
			Enabled:  true,
		},
		{
			ID:          "error_rate_high",
			Name:        "High Error Rate",
			Description: "AI-generated code error rate is high",
			Condition: AlertCondition{
				Metric:      "error_rate",
				Operator:    ">",
				Threshold:   10.0, // 10%
				Duration:    3 * time.Minute,
				Aggregation: "avg",
			},
			Severity: AlertSeverityError,
			Enabled:  true,
		},
		{
			ID:          "memory_usage_warning",
			Name:        "High Memory Usage",
			Description: "Memory usage is above normal levels",
			Condition: AlertCondition{
				Metric:      "memory_usage",
				Operator:    ">",
				Threshold:   500 * 1024 * 1024, // 500MB
				Duration:    10 * time.Minute,
				Aggregation: "avg",
			},
			Severity: AlertSeverityWarning,
			Enabled:  true,
		},
		{
			ID:          "consistency_score_low",
			Name:        "Low Business Logic Consistency",
			Description: "Business logic consistency score is below acceptable levels",
			Condition: AlertCondition{
				Metric:      "consistency_score",
				Operator:    "<",
				Threshold:   80.0, // 80%
				Duration:    5 * time.Minute,
				Aggregation: "avg",
			},
			Severity: AlertSeverityWarning,
			Enabled:  true,
		},
		{
			ID:          "performance_regression",
			Name:        "Performance Regression Detected",
			Description: "Significant performance regression in AI-generated code",
			Condition: AlertCondition{
				Metric:      "performance_score",
				Operator:    "<",
				Threshold:   70.0, // 70%
				Duration:    2 * time.Minute,
				Aggregation: "avg",
			},
			Severity: AlertSeverityError,
			Enabled:  true,
		},
	}
}

// initializeDefaultChannels sets up default alert channels
func (am *AIAlertManager) initializeDefaultChannels() {
	// Add log channel as default
	logChannel := &LogAlertChannel{
		LogLevel: "ERROR",
		Enabled:  true,
	}
	am.alertChannels = append(am.alertChannels, logChannel)
}

// initializeRateLimits sets up rate limiting for different alert types
func (am *AIAlertManager) initializeRateLimits() {
	am.rateLimiter.limits = map[string]*RateLimit{
		"anomaly": {
			MaxAlerts:  10,
			TimeWindow: 1 * time.Hour,
		},
		"regression": {
			MaxAlerts:  5,
			TimeWindow: 30 * time.Minute,
		},
		"threshold": {
			MaxAlerts:  20,
			TimeWindow: 1 * time.Hour,
		},
		"pattern": {
			MaxAlerts:  15,
			TimeWindow: 1 * time.Hour,
		},
		"system": {
			MaxAlerts:  5,
			TimeWindow: 15 * time.Minute,
		},
	}
}

// SendAlerts sends alerts for detected anomalies
func (am *AIAlertManager) SendAlerts(anomalies []Anomaly) {
	for _, anomaly := range anomalies {
		alert := am.createAlertFromAnomaly(anomaly)
		am.sendAlert(alert)
	}
}

// SendRegressionAlerts sends alerts for detected regressions
func (am *AIAlertManager) SendRegressionAlerts(regressions []*RegressionAlert) {
	for _, regression := range regressions {
		alert := am.createAlertFromRegression(regression)
		am.sendAlert(alert)
	}
}

// createAlertFromAnomaly creates an alert from an anomaly
func (am *AIAlertManager) createAlertFromAnomaly(anomaly Anomaly) Alert {
	alert := Alert{
		ID:          am.generateAlertID(),
		Type:        AlertTypeAnomaly,
		Severity:    am.mapAnomalySeverity(anomaly.Severity),
		Title:       fmt.Sprintf("AI Code Anomaly: %s", anomaly.Type),
		Description: anomaly.Description,
		Source:      "ai_anomaly_detector",
		Timestamp:   anomaly.DetectedAt,
		Metadata: map[string]interface{}{
			"anomaly_type":    anomaly.Type,
			"current_value":   anomaly.CurrentValue,
			"baseline_value":  anomaly.BaselineValue,
			"threshold":       anomaly.Threshold,
			"confidence":      anomaly.Confidence,
			"recommendation":  anomaly.Recommendation,
			"trend_analysis":  anomaly.TrendAnalysis,
		},
		Actions: am.getActionsForAnomaly(anomaly),
	}
	
	return alert
}

// createAlertFromRegression creates an alert from a regression
func (am *AIAlertManager) createAlertFromRegression(regression *RegressionAlert) Alert {
	alert := Alert{
		ID:          am.generateAlertID(),
		Type:        AlertTypeRegression,
		Severity:    am.mapRegressionSeverity(regression.Severity),
		Title:       fmt.Sprintf("AI Code Regression: %s", regression.RuleName),
		Description: regression.Description,
		Source:      "ai_performance_tracker",
		Timestamp:   regression.DetectedAt,
		Metadata: map[string]interface{}{
			"rule_name":       regression.RuleName,
			"current_value":   regression.CurrentValue,
			"baseline_value":  regression.BaselineValue,
			"percent_change":  regression.PercentChange,
			"confidence":      regression.Confidence,
			"recommendation":  regression.Recommendation,
			"affected_code":   regression.AffectedCode,
		},
		Actions: am.getActionsForRegression(regression),
	}
	
	return alert
}

// sendAlert sends an alert through all enabled channels
func (am *AIAlertManager) sendAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Check rate limiting
	if !am.rateLimiter.allowAlert(string(alert.Type)) {
		log.Printf("Alert rate limited: %s", alert.ID)
		return
	}
	
	// Add to history
	am.alertHistory = append(am.alertHistory, alert)
	
	// Keep only recent alerts (last 1000)
	if len(am.alertHistory) > 1000 {
		am.alertHistory = am.alertHistory[1:]
	}
	
	// Send through all enabled channels
	for _, channel := range am.alertChannels {
		if channel.IsEnabled() {
			go func(ch AlertChannel, a Alert) {
				if err := ch.SendAlert(a); err != nil {
					log.Printf("Failed to send alert through %s: %v", ch.GetChannelType(), err)
				}
			}(channel, alert)
		}
	}
	
	// Check for escalation
	go am.checkEscalation(alert)
}

// checkEscalation checks if an alert should be escalated
func (am *AIAlertManager) checkEscalation(alert Alert) {
	for _, rule := range am.escalationRules {
		if rule.Enabled && rule.AlertSeverity == alert.Severity {
			time.Sleep(rule.EscalateAfter)
			
			// Check if alert is still unresolved
			am.mu.RLock()
			currentAlert := am.findAlert(alert.ID)
			am.mu.RUnlock()
			
			if currentAlert != nil && !currentAlert.Resolved && !currentAlert.Escalated {
				am.escalateAlert(alert.ID, rule)
			}
		}
	}
}

// escalateAlert escalates an alert
func (am *AIAlertManager) escalateAlert(alertID string, rule EscalationRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert := am.findAlert(alertID)
	if alert == nil {
		return
	}
	
	alert.Escalated = true
	now := time.Now()
	alert.EscalatedAt = &now
	
	// Create escalation alert
	escalationAlert := Alert{
		ID:          am.generateAlertID(),
		Type:        alert.Type,
		Severity:    AlertSeverityCritical,
		Title:       fmt.Sprintf("ESCALATED: %s", alert.Title),
		Description: fmt.Sprintf("Alert %s has been escalated due to lack of response", alert.ID),
		Source:      "ai_alert_manager",
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"original_alert_id": alert.ID,
			"escalation_rule":   rule.ID,
			"escalation_level":  rule.EscalationLevel,
		},
	}
	
	// Send escalation alert through specified channels
	for _, channelType := range rule.Channels {
		for _, channel := range am.alertChannels {
			if channel.GetChannelType() == channelType && channel.IsEnabled() {
				go func(ch AlertChannel, a Alert) {
					if err := ch.SendAlert(a); err != nil {
						log.Printf("Failed to send escalation alert through %s: %v", ch.GetChannelType(), err)
					}
				}(channel, escalationAlert)
			}
		}
	}
}

// AddAlertChannel adds a new alert channel
func (am *AIAlertManager) AddAlertChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.alertChannels = append(am.alertChannels, channel)
}

// AcknowledgeAlert acknowledges an alert
func (am *AIAlertManager) AcknowledgeAlert(alertID, user, comment string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert := am.findAlert(alertID)
	if alert == nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	acknowledgment := Acknowledgment{
		User:      user,
		Timestamp: time.Now(),
		Comment:   comment,
	}
	
	alert.Acknowledgments = append(alert.Acknowledgments, acknowledgment)
	
	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AIAlertManager) ResolveAlert(alertID, user string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert := am.findAlert(alertID)
	if alert == nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now
	
	// Add resolution acknowledgment
	acknowledgment := Acknowledgment{
		User:      user,
		Timestamp: now,
		Comment:   "Alert resolved",
	}
	
	alert.Acknowledgments = append(alert.Acknowledgments, acknowledgment)
	
	return nil
}

// GetAlertHistory returns the alert history
func (am *AIAlertManager) GetAlertHistory(limit int) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}
	
	// Return most recent alerts
	start := len(am.alertHistory) - limit
	if start < 0 {
		start = 0
	}
	
	history := make([]Alert, limit)
	copy(history, am.alertHistory[start:])
	
	return history
}

// Helper methods

func (am *AIAlertManager) generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

func (am *AIAlertManager) mapAnomalySeverity(severity AnomalySeverity) AlertSeverity {
	switch severity {
	case SeverityCritical:
		return AlertSeverityCritical
	case SeverityHigh:
		return AlertSeverityError
	case SeverityMedium:
		return AlertSeverityWarning
	case SeverityLow:
		return AlertSeverityInfo
	default:
		return AlertSeverityInfo
	}
}

func (am *AIAlertManager) mapRegressionSeverity(severity string) AlertSeverity {
	switch severity {
	case "critical":
		return AlertSeverityCritical
	case "high":
		return AlertSeverityError
	case "medium":
		return AlertSeverityWarning
	case "low":
		return AlertSeverityInfo
	default:
		return AlertSeverityInfo
	}
}

func (am *AIAlertManager) getActionsForAnomaly(anomaly Anomaly) []AlertAction {
	var actions []AlertAction
	
	switch anomaly.Type {
	case AnomalyTypeQueryPerformance:
		actions = append(actions, AlertAction{
			ID:          "analyze_slow_queries",
			Name:        "Analyze Slow Queries",
			Description: "Analyze slow query log for optimization opportunities",
			Type:        "script",
			Config: map[string]interface{}{
				"script": "analyze_slow_queries.sh",
			},
			Automatic: false,
		})
	case AnomalyTypeMemoryUsage:
		actions = append(actions, AlertAction{
			ID:          "memory_profiling",
			Name:        "Start Memory Profiling",
			Description: "Start memory profiling to identify leaks",
			Type:        "api_call",
			Config: map[string]interface{}{
				"endpoint": "/debug/pprof/heap",
			},
			Automatic: false,
		})
	}
	
	return actions
}

func (am *AIAlertManager) getActionsForRegression(regression *RegressionAlert) []AlertAction {
	var actions []AlertAction
	
	actions = append(actions, AlertAction{
		ID:          "rollback_recent_changes",
		Name:        "Rollback Recent Changes",
		Description: "Consider rolling back recent AI-generated code changes",
		Type:        "webhook",
		Config: map[string]interface{}{
			"url": "/api/rollback",
		},
		Automatic: false,
	})
	
	return actions
}

func (am *AIAlertManager) findAlert(alertID string) *Alert {
	for i := range am.alertHistory {
		if am.alertHistory[i].ID == alertID {
			return &am.alertHistory[i]
		}
	}
	return nil
}

// Rate limiter methods

func (rl *AlertRateLimiter) allowAlert(alertType string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limit, exists := rl.limits[alertType]
	if !exists {
		return true
	}
	
	now := time.Now()
	
	// Reset window if expired
	if now.Sub(limit.WindowStart) > limit.TimeWindow {
		limit.AlertCount = 0
		limit.WindowStart = now
	}
	
	// Check if under limit
	if limit.AlertCount >= limit.MaxAlerts {
		return false
	}
	
	limit.AlertCount++
	return true
}

// Alert channel implementations

func (e *EmailAlertChannel) SendAlert(alert Alert) error {
	if !e.Enabled {
		return nil
	}
	
	// This would implement actual email sending
	log.Printf("EMAIL ALERT: [%s] %s - %s", alert.Severity, alert.Title, alert.Description)
	return nil
}

func (e *EmailAlertChannel) GetChannelType() string {
	return "email"
}

func (e *EmailAlertChannel) IsEnabled() bool {
	return e.Enabled
}

func (s *SlackAlertChannel) SendAlert(alert Alert) error {
	if !s.Enabled {
		return nil
	}
	
	payload := map[string]interface{}{
		"channel":  s.Channel,
		"username": s.Username,
		"text":     fmt.Sprintf("🚨 *%s*\n%s\n_Source: %s_", alert.Title, alert.Description, alert.Source),
		"attachments": []map[string]interface{}{
			{
				"color": s.getSeverityColor(alert.Severity),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
					{
						"title": "Timestamp",
						"value": alert.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

func (s *SlackAlertChannel) GetChannelType() string {
	return "slack"
}

func (s *SlackAlertChannel) IsEnabled() bool {
	return s.Enabled
}

func (s *SlackAlertChannel) getSeverityColor(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityCritical:
		return "danger"
	case AlertSeverityError:
		return "warning"
	case AlertSeverityWarning:
		return "warning"
	default:
		return "good"
	}
}

func (w *WebhookAlertChannel) SendAlert(alert Alert) error {
	if !w.Enabled {
		return nil
	}
	
	jsonPayload, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest(w.Method, w.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

func (w *WebhookAlertChannel) GetChannelType() string {
	return "webhook"
}

func (w *WebhookAlertChannel) IsEnabled() bool {
	return w.Enabled
}

func (l *LogAlertChannel) SendAlert(alert Alert) error {
	if !l.Enabled {
		return nil
	}
	
	logMessage := fmt.Sprintf("AI_ALERT [%s] %s: %s (Source: %s, Time: %s)",
		alert.Severity, alert.Title, alert.Description, alert.Source, alert.Timestamp.Format(time.RFC3339))
	
	switch l.LogLevel {
	case "ERROR":
		log.Printf("ERROR: %s", logMessage)
	case "WARN":
		log.Printf("WARN: %s", logMessage)
	case "INFO":
		log.Printf("INFO: %s", logMessage)
	default:
		log.Printf("%s", logMessage)
	}
	
	return nil
}

func (l *LogAlertChannel) GetChannelType() string {
	return "log"
}

func (l *LogAlertChannel) IsEnabled() bool {
	return l.Enabled
}