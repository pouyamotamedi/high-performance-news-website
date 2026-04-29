package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
)

func TestNewAlertingService(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		AlertCooldownPeriod: 15 * time.Minute,
	}
	
	emailService := NewMockEmailService()
	
	service := NewAlertingService(config, emailService)
	
	if service == nil {
		t.Fatal("Expected AlertingService to be created")
	}
	
	if service.config != config {
		t.Error("Config not properly set")
	}
	
	if service.emailService != emailService {
		t.Error("EmailService not properly set")
	}
}

func TestSendAlert(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:        true,
		EmailAlerting:         true,
		AlertEmailRecipients:  "admin@example.com,ops@example.com",
		AlertCooldownPeriod:   5 * time.Minute,
	}
	
	emailService := NewMockEmailService()
	service := NewAlertingService(config, emailService)
	
	alert := &models.Alert{
		Name:         "test_alert",
		Description:  "Test alert description",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    80.0,
		CurrentValue: 85.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send alert: %v", err)
	}
	
	// Check that emails were sent
	sentEmails := emailService.GetSentEmails()
	if len(sentEmails) != 2 {
		t.Errorf("Expected 2 emails to be sent, got %d", len(sentEmails))
	}
	
	// Check email content
	for _, email := range sentEmails {
		if !strings.Contains(email.Subject, "WARNING") {
			t.Errorf("Expected WARNING in subject, got %s", email.Subject)
		}
		
		if !strings.Contains(email.Body, "test_alert") {
			t.Errorf("Expected alert name in body, got %s", email.Body)
		}
	}
	
	// Check that alert is stored in history
	history := service.GetAlertHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 alert in history, got %d", len(history))
	}
}

func TestRateLimiting(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:        true,
		EmailAlerting:         true,
		AlertEmailRecipients:  "admin@example.com",
		AlertCooldownPeriod:   1 * time.Second,
	}
	
	emailService := NewMockEmailService()
	service := NewAlertingService(config, emailService)
	
	alert := &models.Alert{
		Name:         "rate_limited_alert",
		Description:  "Test rate limiting",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    80.0,
		CurrentValue: 85.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	// Send first alert
	err := service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send first alert: %v", err)
	}
	
	// Send second alert immediately (should be rate limited)
	err = service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send second alert: %v", err)
	}
	
	// Check that only one email was sent
	sentEmails := emailService.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Errorf("Expected 1 email due to rate limiting, got %d", len(sentEmails))
	}
	
	// Wait for cooldown period
	time.Sleep(1100 * time.Millisecond)
	
	// Send third alert (should not be rate limited)
	err = service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send third alert: %v", err)
	}
	
	// Check that second email was sent
	sentEmails = emailService.GetSentEmails()
	if len(sentEmails) != 2 {
		t.Errorf("Expected 2 emails after cooldown, got %d", len(sentEmails))
	}
}

func TestSlackAlert(t *testing.T) {
	// Create a test server to simulate Slack webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}
		
		var message SlackMessage
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			t.Errorf("Failed to decode Slack message: %v", err)
		}
		
		if !strings.Contains(message.Text, "CRITICAL") {
			t.Errorf("Expected CRITICAL in message text, got %s", message.Text)
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		SlackAlerting:       true,
		SlackWebhookURL:     server.URL,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	service := NewAlertingService(config, nil)
	
	alert := &models.Alert{
		Name:         "slack_test_alert",
		Description:  "Test Slack alert",
		Severity:     models.AlertSeverityCritical,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    90.0,
		CurrentValue: 95.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send Slack alert: %v", err)
	}
}

func TestWebhookAlert(t *testing.T) {
	// Create a test server to simulate webhook endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}
		
		var webhookAlert WebhookAlert
		err := json.NewDecoder(r.Body).Decode(&webhookAlert)
		if err != nil {
			t.Errorf("Failed to decode webhook alert: %v", err)
		}
		
		if webhookAlert.Name != "webhook_test_alert" {
			t.Errorf("Expected alert name 'webhook_test_alert', got %s", webhookAlert.Name)
		}
		
		if webhookAlert.Severity != "warning" {
			t.Errorf("Expected severity 'warning', got %s", webhookAlert.Severity)
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		WebhookAlerting:     true,
		AlertWebhookURL:     server.URL,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	service := NewAlertingService(config, nil)
	
	alert := &models.Alert{
		Name:         "webhook_test_alert",
		Description:  "Test webhook alert",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    75.0,
		CurrentValue: 80.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := service.SendAlert(alert)
	if err != nil {
		t.Fatalf("Failed to send webhook alert: %v", err)
	}
}

func TestTestAlert(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:        true,
		EmailAlerting:         true,
		AlertEmailRecipients:  "test@example.com",
		AlertCooldownPeriod:   5 * time.Minute,
	}
	
	emailService := NewMockEmailService()
	service := NewAlertingService(config, emailService)
	
	err := service.TestAlert()
	if err != nil {
		t.Fatalf("Failed to send test alert: %v", err)
	}
	
	// Check that test email was sent
	sentEmails := emailService.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Errorf("Expected 1 test email, got %d", len(sentEmails))
	}
	
	email := sentEmails[0]
	if !strings.Contains(email.Subject, "INFO") {
		t.Errorf("Expected INFO in test alert subject, got %s", email.Subject)
	}
	
	if !strings.Contains(email.Body, "test alert") {
		t.Errorf("Expected 'test alert' in body, got %s", email.Body)
	}
}

func TestEvaluateAlertRule(t *testing.T) {
	service := NewAlertingService(nil, nil)
	
	rule := &models.AlertRule{
		Operator:  ">",
		Threshold: 80.0,
	}
	
	// Test greater than
	if !service.EvaluateAlertRule(rule, 85.0) {
		t.Error("Expected true for 85.0 > 80.0")
	}
	
	if service.EvaluateAlertRule(rule, 75.0) {
		t.Error("Expected false for 75.0 > 80.0")
	}
	
	// Test less than
	rule.Operator = "<"
	if !service.EvaluateAlertRule(rule, 75.0) {
		t.Error("Expected true for 75.0 < 80.0")
	}
	
	if service.EvaluateAlertRule(rule, 85.0) {
		t.Error("Expected false for 85.0 < 80.0")
	}
	
	// Test equals
	rule.Operator = "=="
	if !service.EvaluateAlertRule(rule, 80.0) {
		t.Error("Expected true for 80.0 == 80.0")
	}
	
	if service.EvaluateAlertRule(rule, 85.0) {
		t.Error("Expected false for 85.0 == 80.0")
	}
	
	// Test unknown operator
	rule.Operator = "unknown"
	if service.EvaluateAlertRule(rule, 85.0) {
		t.Error("Expected false for unknown operator")
	}
}

func TestProcessAlertRule(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:        true,
		EmailAlerting:         true,
		AlertEmailRecipients:  "test@example.com",
		AlertCooldownPeriod:   5 * time.Minute,
	}
	
	emailService := NewMockEmailService()
	service := NewAlertingService(config, emailService)
	
	rule := &models.AlertRule{
		Name:        "test_rule",
		Description: "Test alert rule",
		Component:   "test",
		Metric:      "test_metric",
		Operator:    ">",
		Threshold:   80.0,
		Severity:    models.AlertSeverityWarning,
		Enabled:     true,
	}
	
	// Test rule that should trigger
	err := service.ProcessAlertRule(rule, 85.0)
	if err != nil {
		t.Fatalf("Failed to process alert rule: %v", err)
	}
	
	// Check that alert was sent
	sentEmails := emailService.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Errorf("Expected 1 email from triggered rule, got %d", len(sentEmails))
	}
	
	// Test rule that should not trigger
	err = service.ProcessAlertRule(rule, 75.0)
	if err != nil {
		t.Fatalf("Failed to process alert rule: %v", err)
	}
	
	// Check that no additional alert was sent
	sentEmails = emailService.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Errorf("Expected still 1 email, got %d", len(sentEmails))
	}
	
	// Test disabled rule
	rule.Enabled = false
	err = service.ProcessAlertRule(rule, 90.0)
	if err != nil {
		t.Fatalf("Failed to process disabled alert rule: %v", err)
	}
	
	// Check that no additional alert was sent
	sentEmails = emailService.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Errorf("Expected still 1 email from disabled rule, got %d", len(sentEmails))
	}
}

func TestGetSlackColor(t *testing.T) {
	service := NewAlertingService(nil, nil)
	
	tests := []struct {
		severity models.AlertSeverity
		expected string
	}{
		{models.AlertSeverityCritical, "danger"},
		{models.AlertSeverityWarning, "warning"},
		{models.AlertSeverityInfo, "good"},
		{"unknown", "#808080"},
	}
	
	for _, test := range tests {
		color := service.getSlackColor(test.severity)
		if color != test.expected {
			t.Errorf("Expected color %s for severity %s, got %s", test.expected, test.severity, color)
		}
	}
}

func TestAlertHistory(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	service := NewAlertingService(config, nil)
	
	// Initially empty
	history := service.GetAlertHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history initially, got %d alerts", len(history))
	}
	
	// Add some alerts
	alert1 := &models.Alert{Name: "alert1", Severity: models.AlertSeverityWarning}
	alert2 := &models.Alert{Name: "alert2", Severity: models.AlertSeverityCritical}
	
	service.storeAlert(alert1)
	service.storeAlert(alert2)
	
	history = service.GetAlertHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 alerts in history, got %d", len(history))
	}
	
	// Clear history
	service.ClearAlertHistory()
	
	history = service.GetAlertHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d alerts", len(history))
	}
}

// Benchmark tests
func BenchmarkSendAlert(b *testing.B) {
	config := &config.MonitoringConfig{
		EnableAlerting:        true,
		EmailAlerting:         true,
		AlertEmailRecipients:  "test@example.com",
		AlertCooldownPeriod:   1 * time.Hour, // Long cooldown to avoid rate limiting
	}
	
	emailService := NewMockEmailService()
	service := NewAlertingService(config, emailService)
	
	alert := &models.Alert{
		Name:         "benchmark_alert",
		Description:  "Benchmark test alert",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    80.0,
		CurrentValue: 85.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alert.Name = fmt.Sprintf("benchmark_alert_%d", i) // Unique names to avoid rate limiting
		service.SendAlert(alert)
	}
}

func BenchmarkEvaluateAlertRule(b *testing.B) {
	service := NewAlertingService(nil, nil)
	
	rule := &models.AlertRule{
		Operator:  ">",
		Threshold: 80.0,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.EvaluateAlertRule(rule, 85.0)
	}
}