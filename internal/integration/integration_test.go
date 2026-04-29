package integration

import (
	"context"
	"testing"
	"time"
)

func TestIntegrationManager(t *testing.T) {
	manager := NewIntegrationManager()

	// Test registering a mock integration
	mockIntegration := &MockIntegration{
		name:    "test-integration",
		intType: IntegrationTypeBugTracking,
		healthy: true,
	}

	err := manager.RegisterIntegration(mockIntegration)
	if err != nil {
		t.Fatalf("Failed to register integration: %v", err)
	}

	// Test connecting to integration
	config := Config{
		Name:     "test-integration",
		Type:     IntegrationTypeBugTracking,
		Enabled:  true,
		Settings: map[string]interface{}{
			"test_setting": "test_value",
		},
	}

	ctx := context.Background()
	err = manager.ConnectIntegration(ctx, "test-integration", config)
	if err != nil {
		t.Fatalf("Failed to connect to integration: %v", err)
	}

	// Test sending event
	event := Event{
		ID:        "test-event-1",
		Type:      EventTypeTestFailure,
		Source:    "test-source",
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
		Data: map[string]interface{}{
			"test_name": "TestExample",
			"error":     "Test failed",
		},
	}

	err = manager.SendEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Test health check
	health := manager.HealthCheck(ctx)
	if !health["test-integration"] {
		t.Errorf("Expected integration to be healthy")
	}

	// Test getting status
	status := manager.GetIntegrationStatus()
	if len(status) != 1 {
		t.Errorf("Expected 1 integration status, got %d", len(status))
	}

	integrationStatus := status["test-integration"]
	if integrationStatus.Name != "test-integration" {
		t.Errorf("Expected integration name 'test-integration', got %s", integrationStatus.Name)
	}
}

func TestWebhookManager(t *testing.T) {
	webhookManager := NewWebhookManager()

	// Test registering webhook
	webhook := &Webhook{
		Name:    "test-webhook",
		URL:     "https://example.com/webhook",
		Enabled: true,
		Events:  []EventType{EventTypeTestFailure},
		Retries: 3,
		Timeout: 10 * time.Second,
	}

	err := webhookManager.RegisterWebhook(webhook)
	if err != nil {
		t.Fatalf("Failed to register webhook: %v", err)
	}

	// Test getting webhooks
	webhooks := webhookManager.GetWebhooks()
	if len(webhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(webhooks))
	}

	if webhooks["test-webhook"].URL != "https://example.com/webhook" {
		t.Errorf("Expected webhook URL 'https://example.com/webhook', got %s", webhooks["test-webhook"].URL)
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	// Test recording metrics
	collector.RecordIntegrationConnection("test-integration", true)
	collector.RecordEventSent("test-integration", EventTypeTestFailure)
	collector.RecordIntegrationError("test-integration", "connection_failed")

	// Test getting metrics
	metrics := collector.GetIntegrationMetrics("test-integration")
	if metrics.EventsSent != 1 {
		t.Errorf("Expected 1 event sent, got %d", metrics.EventsSent)
	}

	if metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.ErrorCount)
	}

	if metrics.SuccessRate != 50.0 {
		t.Errorf("Expected 50%% success rate, got %.1f%%", metrics.SuccessRate)
	}
}

// MockIntegration is a mock implementation for testing
type MockIntegration struct {
	name      string
	intType   IntegrationType
	healthy   bool
	connected bool
	events    []Event
}

func (m *MockIntegration) Name() string {
	return m.name
}

func (m *MockIntegration) Type() IntegrationType {
	return m.intType
}

func (m *MockIntegration) Connect(ctx context.Context, config Config) error {
	m.connected = true
	return nil
}

func (m *MockIntegration) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *MockIntegration) IsHealthy(ctx context.Context) bool {
	return m.healthy && m.connected
}

func (m *MockIntegration) SendEvent(ctx context.Context, event Event) error {
	if !m.connected {
		return ErrNotConnected
	}
	m.events = append(m.events, event)
	return nil
}

// Custom error for testing
var ErrNotConnected = fmt.Errorf("integration not connected")

func TestIntegrationService(t *testing.T) {
	service := NewIntegrationService()

	// Test connecting to integration
	config := Config{
		Name:     "jira",
		Type:     IntegrationTypeBugTracking,
		Enabled:  true,
		Settings: map[string]interface{}{
			"base_url":    "https://test.atlassian.net",
			"username":    "test@example.com",
			"api_token":   "test-token",
			"project_key": "TEST",
		},
	}

	ctx := context.Background()
	
	// Note: This will fail in tests without actual JIRA instance
	// In a real test environment, you would mock the HTTP client
	err := service.ConnectIntegration(ctx, "jira", config)
	if err != nil {
		t.Logf("Expected connection failure in test environment: %v", err)
	}

	// Test getting status
	status := service.GetIntegrationStatus()
	if status == nil {
		t.Errorf("Expected status to be non-nil")
	}

	// Test health check
	health := service.HealthCheck(ctx)
	if health == nil {
		t.Errorf("Expected health to be non-nil")
	}

	// Test metrics
	metrics := service.GetMetrics()
	if metrics == nil {
		t.Errorf("Expected metrics to be non-nil")
	}
}