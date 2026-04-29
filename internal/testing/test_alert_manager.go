package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// TestAlertManager manages alerts for test execution monitoring
type TestAlertManager struct {
	alerts          map[string]*TestAlert
	alertRules      []AlertRule
	notifiers       []AlertNotifier
	escalationRules []EscalationRule
	mu              sync.RWMutex
	isRunning       bool
	stopChan        chan struct{}
	alertHistory    []TestAlert
}

// TestAlert represents an alert for test execution issues
type TestAlert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      AlertStatus            `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	AckedBy     string                 `json:"acked_by,omitempty"`
	EscalatedAt *time.Time             `json:"escalated_at,omitempty"`
	NotifiedAt  *time.Time             `json:"notified_at,omitempty"`
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeTestFailure     AlertType = "test_failure"
	AlertTypeTestTimeout     AlertType = "test_timeout"
	AlertTypeResourceLimit   AlertType = "resource_limit"
	AlertTypePerformance     AlertType = "performance"
	AlertTypeInfrastructure  AlertType = "infrastructure"
	AlertTypeQuality         AlertType = "quality"
	AlertTypeReliability     AlertType = "reliability"
	AlertTypeSecurity        AlertType = "security"
	AlertTypeCapacity        AlertType = "capacity"
)

// AlertSeverity defines the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus defines the status of an alert
type AlertStatus string

const (
	StatusActive     AlertStatus = "active"
	StatusAcknowledged AlertStatus = "acknowledged"
	StatusResolved   AlertStatus = "resolved"
	StatusSuppressed AlertStatus = "suppressed"
)

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Condition   AlertCondition         `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`
	Enabled     bool                   `json:"enabled"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertCondition defines the condition for triggering an alert
type AlertCondition struct {
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"` // >, <, >=, <=, ==, !=
	Value     float64 `json:"value"`
	Timeframe string  `json:"timeframe"` // 1m, 5m, 15m, 1h, etc.
}

// EscalationRule defines escalation behavior for alerts
type EscalationRule struct {
	ID           string        `json:"id"`
	AlertType    AlertType     `json:"alert_type"`
	Severity     AlertSeverity `json:"severity"`
	EscalateAfter time.Duration `json:"escalate_after"`
	EscalateTo   []string      `json:"escalate_to"`
	Enabled      bool          `json:"enabled"`
}

// AlertNotifier interface for different notification channels
type AlertNotifier interface {
	SendAlert(alert *TestAlert) error
	GetName() string
	IsEnabled() bool
}

// AlertSummary provides a summary of alert status
type AlertSummary struct {
	ActiveAlerts    int `json:"active_alerts"`
	CriticalAlerts  int `json:"critical_alerts"`
	WarningAlerts   int `json:"warning_alerts"`
	ResolvedToday   int `json:"resolved_today"`
}

// NewTestAlertManager creates a new test alert manager
func NewTestAlertManager() *TestAlertManager {
	return &TestAlertManager{
		alerts:          make(map[string]*TestAlert),
		alertRules:      make([]AlertRule, 0),
		notifiers:       make([]AlertNotifier, 0),
		escalationRules: make([]EscalationRule, 0),
		stopChan:        make(chan struct{}),
		alertHistory:    make([]TestAlert, 0),
	}
}

// Start begins alert management
func (a *TestAlertManager) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isRunning {
		return fmt.Errorf("test alert manager is already running")
	}

	// Initialize default alert rules
	a.initializeDefaultRules()

	// Initialize default notifiers
	a.initializeDefaultNotifiers()

	a.isRunning = true

	// Start management goroutines
	go a.processAlerts(ctx)
	go a.checkEscalations(ctx)
	go a.cleanupHistory(ctx)

	log.Printf("Test alert manager started with %d rules and %d notifiers", 
		len(a.alertRules), len(a.notifiers))
	return nil
}

// Stop stops alert management
func (a *TestAlertManager) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isRunning {
		return
	}

	close(a.stopChan)
	a.isRunning = false

	log.Printf("Test alert manager stopped")
}

// CreateAlert creates a new alert
func (a *TestAlertManager) CreateAlert(alertType AlertType, severity AlertSeverity, title, message, source string, tags map[string]string, metadata map[string]interface{}) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	alert := &TestAlert{
		ID:        generateAlertID(),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Source:    source,
		Tags:      tags,
		Metadata:  metadata,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	a.alerts[alert.ID] = alert

	// Send notifications
	go a.sendNotifications(alert)

	log.Printf("Alert created: %s - %s (%s)", severity, title, alert.ID)
	return alert.ID
}

// ResolveAlert resolves an active alert
func (a *TestAlertManager) ResolveAlert(alertID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	alert, exists := a.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	if alert.Status == StatusResolved {
		return fmt.Errorf("alert %s is already resolved", alertID)
	}

	now := time.Now()
	alert.Status = StatusResolved
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	// Move to history
	a.alertHistory = append(a.alertHistory, *alert)
	delete(a.alerts, alertID)

	log.Printf("Alert resolved: %s", alertID)
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (a *TestAlertManager) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	alert, exists := a.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	if alert.Status == StatusResolved {
		return fmt.Errorf("cannot acknowledge resolved alert %s", alertID)
	}

	now := time.Now()
	alert.Status = StatusAcknowledged
	alert.AckedAt = &now
	alert.AckedBy = acknowledgedBy
	alert.UpdatedAt = now

	log.Printf("Alert acknowledged: %s by %s", alertID, acknowledgedBy)
	return nil
}

// GetActiveAlerts returns all active alerts
func (a *TestAlertManager) GetActiveAlerts() map[string]*TestAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	alerts := make(map[string]*TestAlert)
	for id, alert := range a.alerts {
		alertCopy := *alert
		alerts[id] = &alertCopy
	}
	return alerts
}

// GetAlertHistory returns recent alert history
func (a *TestAlertManager) GetAlertHistory(limit int) []TestAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.alertHistory) {
		limit = len(a.alertHistory)
	}

	// Return most recent alerts
	start := len(a.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]TestAlert, limit)
	copy(history, a.alertHistory[start:])
	return history
}

// GetAlertSummary returns alert summary statistics
func (a *TestAlertManager) GetAlertSummary() AlertSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()

	summary := AlertSummary{}

	// Count active alerts by severity
	for _, alert := range a.alerts {
		summary.ActiveAlerts++
		if alert.Severity == SeverityCritical {
			summary.CriticalAlerts++
		} else if alert.Severity == SeverityWarning {
			summary.WarningAlerts++
		}
	}

	// Count resolved alerts today
	today := time.Now().Truncate(24 * time.Hour)
	for _, alert := range a.alertHistory {
		if alert.ResolvedAt != nil && alert.ResolvedAt.After(today) {
			summary.ResolvedToday++
		}
	}

	return summary
}

// NotifyTestStart sends notification when a test starts
func (a *TestAlertManager) NotifyTestStart(execution *TestExecution) {
	// Only notify for long-running or critical tests
	if len(execution.Tags) > 0 {
		for _, tag := range execution.Tags {
			if tag == "critical" || tag == "long-running" {
				a.CreateAlert(
					AlertTypeTestFailure,
					SeverityInfo,
					"Test Started",
					fmt.Sprintf("Test %s/%s started", execution.TestSuite, execution.TestName),
					"test_monitor",
					map[string]string{
						"test_suite": execution.TestSuite,
						"test_name":  execution.TestName,
						"environment": execution.Environment,
					},
					map[string]interface{}{
						"execution_id": execution.ID,
						"start_time":   execution.StartTime,
					},
				)
				break
			}
		}
	}
}

// NotifyTestCompletion sends notification when a test completes
func (a *TestAlertManager) NotifyTestCompletion(execution *TestExecution) {
	if execution.Status == StatusFailed {
		a.CreateAlert(
			AlertTypeTestFailure,
			SeverityError,
			"Test Failed",
			fmt.Sprintf("Test %s/%s failed: %s", execution.TestSuite, execution.TestName, execution.ErrorMessage),
			"test_monitor",
			map[string]string{
				"test_suite": execution.TestSuite,
				"test_name":  execution.TestName,
				"environment": execution.Environment,
				"status":     string(execution.Status),
			},
			map[string]interface{}{
				"execution_id": execution.ID,
				"duration":     execution.Duration.String(),
				"error":        execution.ErrorMessage,
			},
		)
	}
}

// NotifyTestTimeout sends notification when a test times out
func (a *TestAlertManager) NotifyTestTimeout(execution *TestExecution) {
	a.CreateAlert(
		AlertTypeTestTimeout,
		SeverityCritical,
		"Test Timeout",
		fmt.Sprintf("Test %s/%s timed out after %v", execution.TestSuite, execution.TestName, execution.Duration),
		"test_monitor",
		map[string]string{
			"test_suite": execution.TestSuite,
			"test_name":  execution.TestName,
			"environment": execution.Environment,
		},
		map[string]interface{}{
			"execution_id": execution.ID,
			"duration":     execution.Duration.String(),
		},
	)
}

// processAlerts processes active alerts
func (a *TestAlertManager) processAlerts(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.evaluateAlertRules()
		}
	}
}

// evaluateAlertRules evaluates alert rules against current metrics
func (a *TestAlertManager) evaluateAlertRules() {
	// This would evaluate alert rules against current metrics
	// For now, this is a placeholder implementation
	log.Printf("Evaluating alert rules...")
}

// checkEscalations checks for alerts that need escalation
func (a *TestAlertManager) checkEscalations(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.processEscalations()
		}
	}
}

// processEscalations processes alert escalations
func (a *TestAlertManager) processEscalations() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	for _, alert := range a.alerts {
		if alert.Status != StatusActive || alert.EscalatedAt != nil {
			continue
		}

		// Check if alert should be escalated
		for _, rule := range a.escalationRules {
			if !rule.Enabled {
				continue
			}

			if rule.AlertType == alert.Type && rule.Severity == alert.Severity {
				if now.Sub(alert.CreatedAt) >= rule.EscalateAfter {
					alert.EscalatedAt = &now
					alert.UpdatedAt = now

					// Send escalation notifications
					go a.sendEscalationNotifications(alert, rule)

					log.Printf("Alert escalated: %s after %v", alert.ID, rule.EscalateAfter)
				}
			}
		}
	}
}

// sendNotifications sends notifications for an alert
func (a *TestAlertManager) sendNotifications(alert *TestAlert) {
	for _, notifier := range a.notifiers {
		if notifier.IsEnabled() {
			if err := notifier.SendAlert(alert); err != nil {
				log.Printf("Failed to send alert via %s: %v", notifier.GetName(), err)
			}
		}
	}

	now := time.Now()
	alert.NotifiedAt = &now
}

// sendEscalationNotifications sends escalation notifications
func (a *TestAlertManager) sendEscalationNotifications(alert *TestAlert, rule EscalationRule) {
	// This would send escalation notifications to specific recipients
	log.Printf("Sending escalation notifications for alert %s to %v", alert.ID, rule.EscalateTo)
}

// cleanupHistory periodically cleans up old alert history
func (a *TestAlertManager) cleanupHistory(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.performHistoryCleanup()
		}
	}
}

// performHistoryCleanup removes old alert history
func (a *TestAlertManager) performHistoryCleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Keep only last 30 days of history
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	newHistory := make([]TestAlert, 0)

	for _, alert := range a.alertHistory {
		if alert.CreatedAt.After(cutoff) {
			newHistory = append(newHistory, alert)
		}
	}

	oldCount := len(a.alertHistory)
	a.alertHistory = newHistory
	newCount := len(a.alertHistory)

	if oldCount != newCount {
		log.Printf("Cleaned up alert history: removed %d old alerts, kept %d", 
			oldCount-newCount, newCount)
	}
}

// initializeDefaultRules initializes default alert rules
func (a *TestAlertManager) initializeDefaultRules() {
	defaultRules := []AlertRule{
		{
			ID:       "test_failure_rate",
			Name:     "High Test Failure Rate",
			Type:     AlertTypeTestFailure,
			Severity: SeverityWarning,
			Condition: AlertCondition{
				Metric:    "test.failure_rate",
				Operator:  ">",
				Value:     0.1, // 10%
				Timeframe: "15m",
			},
			Threshold: 0.1,
			Duration:  15 * time.Minute,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "resource_exhaustion",
			Name:     "Resource Exhaustion",
			Type:     AlertTypeResourceLimit,
			Severity: SeverityCritical,
			Condition: AlertCondition{
				Metric:    "system.memory_percent",
				Operator:  ">",
				Value:     90.0,
				Timeframe: "5m",
			},
			Threshold: 90.0,
			Duration:  5 * time.Minute,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "performance_degradation",
			Name:     "Performance Degradation",
			Type:     AlertTypePerformance,
			Severity: SeverityWarning,
			Condition: AlertCondition{
				Metric:    "test.avg_duration",
				Operator:  ">",
				Value:     300000, // 5 minutes in milliseconds
				Timeframe: "10m",
			},
			Threshold: 300000,
			Duration:  10 * time.Minute,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	a.alertRules = append(a.alertRules, defaultRules...)
}

// initializeDefaultNotifiers initializes default notifiers
func (a *TestAlertManager) initializeDefaultNotifiers() {
	// Add console notifier
	a.notifiers = append(a.notifiers, &ConsoleNotifier{enabled: true})
	
	// Add log notifier
	a.notifiers = append(a.notifiers, &LogNotifier{enabled: true})
}

// Built-in notifiers

// ConsoleNotifier sends alerts to console
type ConsoleNotifier struct {
	enabled bool
}

func (c *ConsoleNotifier) SendAlert(alert *TestAlert) error {
	fmt.Printf("[ALERT] %s: %s - %s\n", alert.Severity, alert.Title, alert.Message)
	return nil
}

func (c *ConsoleNotifier) GetName() string {
	return "console"
}

func (c *ConsoleNotifier) IsEnabled() bool {
	return c.enabled
}

// LogNotifier sends alerts to log
type LogNotifier struct {
	enabled bool
}

func (l *LogNotifier) SendAlert(alert *TestAlert) error {
	alertJSON, _ := json.Marshal(alert)
	log.Printf("ALERT: %s", string(alertJSON))
	return nil
}

func (l *LogNotifier) GetName() string {
	return "log"
}

func (l *LogNotifier) IsEnabled() bool {
	return l.enabled
}

// Utility functions

func generateAlertID() string {
	return fmt.Sprintf("alert_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}