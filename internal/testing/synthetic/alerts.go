package synthetic

import (
	"fmt"
	"log"
	"time"
)

// AlertManager handles alerting for synthetic monitoring failures
type AlertManager interface {
	SendAlert(level AlertLevel, message string, result MonitoringResult) error
	ConfigureWebhook(url string) error
	ConfigureEmail(config EmailConfig) error
}

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	AlertCritical AlertLevel = "critical"
	AlertHigh     AlertLevel = "high"
	AlertMedium   AlertLevel = "medium"
	AlertLow      AlertLevel = "low"
)

// EmailConfig contains email configuration for alerts
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	FromAddress  string `json:"from_address"`
	ToAddresses  []string `json:"to_addresses"`
}

// SimpleAlertManager implements basic alerting functionality
type SimpleAlertManager struct {
	webhookURL   string
	emailConfig  *EmailConfig
	alertHistory []Alert
	rateLimiter  map[string]time.Time // Prevent alert spam
}

// Alert represents an alert that was sent
type Alert struct {
	Level     AlertLevel       `json:"level"`
	Message   string          `json:"message"`
	TestName  string          `json:"test_name"`
	Timestamp time.Time       `json:"timestamp"`
	Result    MonitoringResult `json:"result"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *SimpleAlertManager {
	return &SimpleAlertManager{
		alertHistory: make([]Alert, 0),
		rateLimiter:  make(map[string]time.Time),
	}
}

// SendAlert sends an alert for a monitoring failure
func (a *SimpleAlertManager) SendAlert(level AlertLevel, message string, result MonitoringResult) error {
	// Rate limiting: don't send same alert more than once per 15 minutes
	rateLimitKey := fmt.Sprintf("%s_%s", result.TestName, level)
	if lastSent, exists := a.rateLimiter[rateLimitKey]; exists {
		if time.Since(lastSent) < 15*time.Minute {
			return nil // Skip this alert due to rate limiting
		}
	}

	alert := Alert{
		Level:     level,
		Message:   message,
		TestName:  result.TestName,
		Timestamp: time.Now(),
		Result:    result,
	}

	// Store alert in history
	a.alertHistory = append(a.alertHistory, alert)

	// Keep only last 1000 alerts
	if len(a.alertHistory) > 1000 {
		a.alertHistory = a.alertHistory[len(a.alertHistory)-1000:]
	}

	// Update rate limiter
	a.rateLimiter[rateLimitKey] = time.Now()

	// Log the alert
	log.Printf("[ALERT %s] %s - Test: %s, Duration: %v, Errors: %v", 
		level, message, result.TestName, result.Duration, result.Errors)

	// Send webhook if configured
	if a.webhookURL != "" {
		go a.sendWebhookAlert(alert)
	}

	// Send email if configured
	if a.emailConfig != nil {
		go a.sendEmailAlert(alert)
	}

	return nil
}

// ConfigureWebhook sets up webhook alerting
func (a *SimpleAlertManager) ConfigureWebhook(url string) error {
	a.webhookURL = url
	log.Printf("Configured webhook alerts to: %s", url)
	return nil
}

// ConfigureEmail sets up email alerting
func (a *SimpleAlertManager) ConfigureEmail(config EmailConfig) error {
	a.emailConfig = &config
	log.Printf("Configured email alerts to: %v", config.ToAddresses)
	return nil
}

// sendWebhookAlert sends an alert via webhook
func (a *SimpleAlertManager) sendWebhookAlert(alert Alert) {
	// Implementation would send HTTP POST to webhook URL
	// For now, just log that we would send it
	log.Printf("Would send webhook alert: %s", alert.Message)
}

// sendEmailAlert sends an alert via email
func (a *SimpleAlertManager) sendEmailAlert(alert Alert) {
	// Implementation would send email using SMTP
	// For now, just log that we would send it
	log.Printf("Would send email alert to %v: %s", a.emailConfig.ToAddresses, alert.Message)
}

// GetAlertHistory returns recent alerts
func (a *SimpleAlertManager) GetAlertHistory(limit int) []Alert {
	if len(a.alertHistory) == 0 {
		return []Alert{}
	}

	start := len(a.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	return a.alertHistory[start:]
}

// GetAlertStats returns alert statistics
func (a *SimpleAlertManager) GetAlertStats(duration time.Duration) AlertStats {
	since := time.Now().Add(-duration)
	
	stats := AlertStats{
		Period: duration,
		Total:  0,
		ByLevel: make(map[AlertLevel]int),
		ByTest:  make(map[string]int),
	}

	for _, alert := range a.alertHistory {
		if alert.Timestamp.After(since) {
			stats.Total++
			stats.ByLevel[alert.Level]++
			stats.ByTest[alert.TestName]++
		}
	}

	return stats
}

// AlertStats provides statistics about alerts
type AlertStats struct {
	Period  time.Duration         `json:"period"`
	Total   int                   `json:"total"`
	ByLevel map[AlertLevel]int    `json:"by_level"`
	ByTest  map[string]int        `json:"by_test"`
}

// ClearAlertHistory clears the alert history
func (a *SimpleAlertManager) ClearAlertHistory() {
	a.alertHistory = make([]Alert, 0)
	a.rateLimiter = make(map[string]time.Time)
}