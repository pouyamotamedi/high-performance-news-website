package integration

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// IntegrationManager manages all external tool integrations
type IntegrationManager struct {
	integrations map[string]interfaces.Integration
	webhooks     *WebhookManager
	metrics      *MetricsCollector
	mu           sync.RWMutex
}



// NewIntegrationManager creates a new integration manager
func NewIntegrationManager() *IntegrationManager {
	return &IntegrationManager{
		integrations: make(map[string]interfaces.Integration),
		webhooks:     NewWebhookManager(),
		metrics:      NewMetricsCollector(),
	}
}

// RegisterIntegration registers a new integration
func (im *IntegrationManager) RegisterIntegration(integration interfaces.Integration) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	name := integration.Name()
	if _, exists := im.integrations[name]; exists {
		return fmt.Errorf("integration %s already registered", name)
	}

	im.integrations[name] = integration
	log.Printf("Registered integration: %s (type: %s)", name, integration.Type())
	return nil
}

// ConnectIntegration connects to an external tool
func (im *IntegrationManager) ConnectIntegration(ctx context.Context, name string, config interfaces.Config) error {
	im.mu.RLock()
	integration, exists := im.integrations[name]
	im.mu.RUnlock()

	if !exists {
		return fmt.Errorf("integration %s not found", name)
	}

	if err := integration.Connect(ctx, config); err != nil {
		im.metrics.RecordIntegrationError(name, "connection_failed")
		return fmt.Errorf("failed to connect to %s: %w", name, err)
	}

	im.metrics.RecordIntegrationConnection(name, true)
	log.Printf("Connected to integration: %s", name)
	return nil
}

// SendEvent sends an event to all relevant integrations
func (im *IntegrationManager) SendEvent(ctx context.Context, event interfaces.Event) error {
	im.mu.RLock()
	defer im.mu.RUnlock()

	var errors []error
	successCount := 0

	for name, integration := range im.integrations {
		if im.shouldSendToIntegration(integration, event) {
			if err := integration.SendEvent(ctx, event); err != nil {
				errors = append(errors, fmt.Errorf("failed to send event to %s: %w", name, err))
				im.metrics.RecordIntegrationError(name, "event_send_failed")
			} else {
				successCount++
				im.metrics.RecordEventSent(name, event.Type)
			}
		}
	}

	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("failed to send event to any integration: %v", errors)
	}

	return nil
}

// shouldSendToIntegration determines if an event should be sent to a specific integration
func (im *IntegrationManager) shouldSendToIntegration(integration interfaces.Integration, event interfaces.Event) bool {
	switch integration.Type() {
	case interfaces.IntegrationTypeBugTracking:
		return event.Type == interfaces.EventTypeTestFailure || event.Type == interfaces.EventTypeSecurityAlert
	case interfaces.IntegrationTypeProjectMgmt:
		return event.Priority == interfaces.PriorityHigh || event.Priority == interfaces.PriorityCritical
	case interfaces.IntegrationTypeCodeReview:
		return event.Type == interfaces.EventTypeCodeReview || event.Type == interfaces.EventTypeTestFailure
	case interfaces.IntegrationTypeMonitoring:
		return true // Send all events to monitoring
	case interfaces.IntegrationTypeCommunication:
		return event.Priority == interfaces.PriorityCritical
	default:
		return false
	}
}

// HealthCheck checks the health of all integrations
func (im *IntegrationManager) HealthCheck(ctx context.Context) map[string]bool {
	im.mu.RLock()
	defer im.mu.RUnlock()

	health := make(map[string]bool)
	for name, integration := range im.integrations {
		health[name] = integration.IsHealthy(ctx)
		im.metrics.RecordIntegrationHealth(name, health[name])
	}

	return health
}

// GetIntegrationStatus returns the status of all integrations
func (im *IntegrationManager) GetIntegrationStatus() map[string]interfaces.IntegrationStatus {
	im.mu.RLock()
	defer im.mu.RUnlock()

	status := make(map[string]interfaces.IntegrationStatus)
	for name, integration := range im.integrations {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		healthy := integration.IsHealthy(ctx)
		cancel()

		status[name] = interfaces.IntegrationStatus{
			Name:    name,
			Type:    integration.Type(),
			Healthy: healthy,
			Metrics: im.metrics.GetIntegrationMetrics(name),
		}
	}

	return status
}

