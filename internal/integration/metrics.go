package integration

import (
	"sync"
	"time"
)

// MetricsCollector collects integration metrics
type MetricsCollector struct {
	metrics map[string]*IntegrationMetrics
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*IntegrationMetrics),
	}
}

// RecordIntegrationConnection records a connection event
func (mc *MetricsCollector) RecordIntegrationConnection(name string, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.metrics[name] == nil {
		mc.metrics[name] = &IntegrationMetrics{}
	}

	if !success {
		mc.metrics[name].ErrorCount++
		mc.metrics[name].LastErrorTime = time.Now()
	}

	mc.updateSuccessRate(name)
}

// RecordEventSent records an event being sent
func (mc *MetricsCollector) RecordEventSent(name string, eventType EventType) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.metrics[name] == nil {
		mc.metrics[name] = &IntegrationMetrics{}
	}

	mc.metrics[name].EventsSent++
	mc.metrics[name].LastEventTime = time.Now()
	mc.updateSuccessRate(name)
}

// RecordIntegrationError records an integration error
func (mc *MetricsCollector) RecordIntegrationError(name, errorType string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.metrics[name] == nil {
		mc.metrics[name] = &IntegrationMetrics{}
	}

	mc.metrics[name].ErrorCount++
	mc.metrics[name].LastErrorTime = time.Now()
	mc.updateSuccessRate(name)
}

// RecordIntegrationHealth records health check result
func (mc *MetricsCollector) RecordIntegrationHealth(name string, healthy bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.metrics[name] == nil {
		mc.metrics[name] = &IntegrationMetrics{}
	}

	// Health checks don't affect success rate directly
	// but we can track them for monitoring
}

// GetIntegrationMetrics returns metrics for a specific integration
func (mc *MetricsCollector) GetIntegrationMetrics(name string) IntegrationMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, exists := mc.metrics[name]; exists {
		return *metrics
	}

	return IntegrationMetrics{}
}

// GetAllMetrics returns all integration metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]IntegrationMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]IntegrationMetrics)
	for name, metrics := range mc.metrics {
		result[name] = *metrics
	}

	return result
}

// updateSuccessRate calculates and updates the success rate
func (mc *MetricsCollector) updateSuccessRate(name string) {
	metrics := mc.metrics[name]
	if metrics == nil {
		return
	}

	total := metrics.EventsSent + metrics.ErrorCount
	if total > 0 {
		metrics.SuccessRate = float64(metrics.EventsSent) / float64(total) * 100
	}
}

// Reset resets metrics for a specific integration
func (mc *MetricsCollector) Reset(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics[name] = &IntegrationMetrics{}
}

// ResetAll resets all metrics
func (mc *MetricsCollector) ResetAll() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = make(map[string]*IntegrationMetrics)
}