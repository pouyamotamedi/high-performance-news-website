package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
)

// AlertingService handles alert notifications and management
type AlertingService struct {
	config       *config.MonitoringConfig
	emailService EmailService
	httpClient   *http.Client
	
	// Alert state management
	alertHistory map[string]*models.Alert
	alertMutex   sync.RWMutex
	
	// Rate limiting for alerts
	lastAlertTime map[string]time.Time
	rateMutex     sync.RWMutex
}

// NewAlertingService creates a new AlertingService instance
func NewAlertingService(config *config.MonitoringConfig, emailService EmailService) *AlertingService {
	return &AlertingService{
		config:        config,
		emailService:  emailService,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		alertHistory:  make(map[string]*models.Alert),
		lastAlertTime: make(map[string]time.Time),
	}
}

// SendAlert sends an alert through configured channels
func (as *AlertingService) SendAlert(alert *models.Alert) error {
	if as.config == nil || !as.config.EnableAlerting {
		return nil
	}
	
	// Check rate limiting
	if as.isRateLimited(alert.Name) {
		log.Printf("Alert %s is rate limited, skipping", alert.Name)
		return nil
	}
	
	// Store alert in history
	as.storeAlert(alert)
	
	var errors []error
	
	// Send email alert
	if as.config.EmailAlerting && as.config.AlertEmailRecipients != "" {
		if err := as.sendEmailAlert(alert); err != nil {
			errors = append(errors, fmt.Errorf("email alert failed: %v", err))
		}
	}
	
	// Send Slack alert
	if as.config.SlackAlerting && as.config.SlackWebhookURL != "" {
		if err := as.sendSlackAlert(alert); err != nil {
			errors = append(errors, fmt.Errorf("slack alert failed: %v", err))
		}
	}
	
	// Send webhook alert
	if as.config.WebhookAlerting && as.config.AlertWebhookURL != "" {
		if err := as.sendWebhookAlert(alert); err != nil {
			errors = append(errors, fmt.Errorf("webhook alert failed: %v", err))
		}
	}
	
	// Update rate limiting
	as.updateRateLimit(alert.Name)
	
	if len(errors) > 0 {
		return fmt.Errorf("alert sending failed: %v", errors)
	}
	
	log.Printf("Alert sent successfully: %s", alert.Name)
	return nil
}

// isRateLimited checks if an alert is rate limited
func (as *AlertingService) isRateLimited(alertName string) bool {
	as.rateMutex.RLock()
	defer as.rateMutex.RUnlock()
	
	lastTime, exists := as.lastAlertTime[alertName]
	if !exists {
		return false
	}
	
	return time.Since(lastTime) < as.config.AlertCooldownPeriod
}

// updateRateLimit updates the rate limit timestamp for an alert
func (as *AlertingService) updateRateLimit(alertName string) {
	as.rateMutex.Lock()
	defer as.rateMutex.Unlock()
	
	as.lastAlertTime[alertName] = time.Now()
}

// storeAlert stores an alert in the history
func (as *AlertingService) storeAlert(alert *models.Alert) {
	as.alertMutex.Lock()
	defer as.alertMutex.Unlock()
	
	as.alertHistory[alert.Name] = alert
}

// sendEmailAlert sends an alert via email
func (as *AlertingService) sendEmailAlert(alert *models.Alert) error {
	if as.emailService == nil {
		return fmt.Errorf("email service not available")
	}
	
	recipients := strings.Split(as.config.AlertEmailRecipients, ",")
	for i, recipient := range recipients {
		recipients[i] = strings.TrimSpace(recipient)
	}
	
	subject := fmt.Sprintf("[%s] %s Alert: %s", 
		strings.ToUpper(string(alert.Severity)), 
		alert.Component, 
		alert.Name)
	
	body := as.formatEmailBody(alert)
	
	for _, recipient := range recipients {
		err := as.emailService.SendEmail(context.Background(), recipient, subject, body)
		if err != nil {
			log.Printf("Failed to send alert email to %s: %v", recipient, err)
			return err
		}
	}
	
	return nil
}

// formatEmailBody formats the email body for an alert
func (as *AlertingService) formatEmailBody(alert *models.Alert) string {
	return fmt.Sprintf(`
Alert Details:
--------------
Name: %s
Severity: %s
Component: %s
Metric: %s
Current Value: %.2f
Threshold: %.2f
Description: %s
Triggered At: %s

This is an automated alert from the High-Performance News Website monitoring system.
`, 
		alert.Name,
		alert.Severity,
		alert.Component,
		alert.Metric,
		alert.CurrentValue,
		alert.Threshold,
		alert.Description,
		alert.TriggeredAt.Format(time.RFC3339))
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title"`
	Text      string       `json:"text"`
	Fields    []SlackField `json:"fields"`
	Timestamp int64        `json:"ts"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// sendSlackAlert sends an alert to Slack
func (as *AlertingService) sendSlackAlert(alert *models.Alert) error {
	color := as.getSlackColor(alert.Severity)
	
	attachment := SlackAttachment{
		Color: color,
		Title: fmt.Sprintf("%s Alert: %s", strings.Title(string(alert.Severity)), alert.Name),
		Text:  alert.Description,
		Fields: []SlackField{
			{Title: "Component", Value: alert.Component, Short: true},
			{Title: "Metric", Value: alert.Metric, Short: true},
			{Title: "Current Value", Value: fmt.Sprintf("%.2f", alert.CurrentValue), Short: true},
			{Title: "Threshold", Value: fmt.Sprintf("%.2f", alert.Threshold), Short: true},
		},
		Timestamp: alert.TriggeredAt.Unix(),
	}
	
	message := SlackMessage{
		Text:        fmt.Sprintf("🚨 %s Alert Triggered", strings.ToUpper(string(alert.Severity))),
		Username:    "Monitoring Bot",
		IconEmoji:   ":warning:",
		Attachments: []SlackAttachment{attachment},
	}
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %v", err)
	}
	
	resp, err := as.httpClient.Post(as.config.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack webhook: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// getSlackColor returns the appropriate color for a Slack message based on severity
func (as *AlertingService) getSlackColor(severity models.AlertSeverity) string {
	switch severity {
	case models.AlertSeverityCritical:
		return "danger"
	case models.AlertSeverityWarning:
		return "warning"
	case models.AlertSeverityInfo:
		return "good"
	default:
		return "#808080"
	}
}

// WebhookAlert represents a webhook alert payload
type WebhookAlert struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Severity     string                 `json:"severity"`
	Status       string                 `json:"status"`
	Component    string                 `json:"component"`
	Metric       string                 `json:"metric"`
	Threshold    float64                `json:"threshold"`
	CurrentValue float64                `json:"current_value"`
	TriggeredAt  time.Time              `json:"triggered_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// sendWebhookAlert sends an alert to a webhook endpoint
func (as *AlertingService) sendWebhookAlert(alert *models.Alert) error {
	webhookAlert := WebhookAlert{
		Name:         alert.Name,
		Description:  alert.Description,
		Severity:     string(alert.Severity),
		Status:       string(alert.Status),
		Component:    alert.Component,
		Metric:       alert.Metric,
		Threshold:    alert.Threshold,
		CurrentValue: alert.CurrentValue,
		TriggeredAt:  alert.TriggeredAt,
		Metadata:     alert.Metadata,
	}
	
	jsonData, err := json.Marshal(webhookAlert)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook alert: %v", err)
	}
	
	resp, err := as.httpClient.Post(as.config.AlertWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook alert: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// GetAlertHistory returns the alert history
func (as *AlertingService) GetAlertHistory() map[string]*models.Alert {
	as.alertMutex.RLock()
	defer as.alertMutex.RUnlock()
	
	history := make(map[string]*models.Alert)
	for k, v := range as.alertHistory {
		history[k] = v
	}
	
	return history
}

// ClearAlertHistory clears the alert history
func (as *AlertingService) ClearAlertHistory() {
	as.alertMutex.Lock()
	defer as.alertMutex.Unlock()
	
	as.alertHistory = make(map[string]*models.Alert)
}

// TestAlert sends a test alert to verify alert channels are working
func (as *AlertingService) TestAlert() error {
	testAlert := &models.Alert{
		Name:         "test_alert",
		Description:  "This is a test alert to verify monitoring system functionality",
		Severity:     models.AlertSeverityInfo,
		Status:       models.AlertStatusActive,
		Component:    "monitoring",
		Metric:       "test",
		Threshold:    1.0,
		CurrentValue: 1.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	return as.SendAlert(testAlert)
}

// CreateAlertRule creates a new alert rule
func (as *AlertingService) CreateAlertRule(rule *models.AlertRule) error {
	// Validate the alert rule
	if err := models.ValidateAlertRule(rule); err != nil {
		return err
	}
	
	// TODO: Store alert rule in database
	// This would typically involve saving to a database table
	
	log.Printf("Alert rule created: %s", rule.Name)
	return nil
}

// EvaluateAlertRule evaluates an alert rule against current metrics
func (as *AlertingService) EvaluateAlertRule(rule *models.AlertRule, currentValue float64) bool {
	switch rule.Operator {
	case ">":
		return currentValue > rule.Threshold
	case "<":
		return currentValue < rule.Threshold
	case ">=":
		return currentValue >= rule.Threshold
	case "<=":
		return currentValue <= rule.Threshold
	case "==":
		return currentValue == rule.Threshold
	case "!=":
		return currentValue != rule.Threshold
	default:
		log.Printf("Unknown operator in alert rule %s: %s", rule.Name, rule.Operator)
		return false
	}
}

// ProcessAlertRule processes an alert rule and triggers alerts if conditions are met
func (as *AlertingService) ProcessAlertRule(rule *models.AlertRule, currentValue float64) error {
	if !rule.Enabled {
		return nil
	}
	
	if as.EvaluateAlertRule(rule, currentValue) {
		alert := &models.Alert{
			Name:         rule.Name,
			Description:  rule.Description,
			Severity:     rule.Severity,
			Status:       models.AlertStatusActive,
			Component:    rule.Component,
			Metric:       rule.Metric,
			Threshold:    rule.Threshold,
			CurrentValue: currentValue,
			Metadata:     rule.Conditions,
			TriggeredAt:  time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		return as.SendAlert(alert)
	}
	
	return nil
}

// GetAlertingConfig returns the current alerting configuration
func (as *AlertingService) GetAlertingConfig() *config.MonitoringConfig {
	return as.config
}

// UpdateAlertingConfig updates the alerting configuration
func (as *AlertingService) UpdateAlertingConfig(config *config.MonitoringConfig) {
	as.config = config
}