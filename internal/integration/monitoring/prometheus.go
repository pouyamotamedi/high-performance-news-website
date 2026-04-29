package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// PrometheusIntegration implements Prometheus monitoring integration
type PrometheusIntegration struct {
	pushGatewayURL string
	jobName        string
	instance       string
	client         *http.Client
	connected      bool
}

// PrometheusMetric represents a Prometheus metric
type PrometheusMetric struct {
	Name   string            `json:"name"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
	Help   string            `json:"help"`
	Type   string            `json:"type"`
}

// NewPrometheusIntegration creates a new Prometheus integration
func NewPrometheusIntegration() *PrometheusIntegration {
	return &PrometheusIntegration{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		jobName:  "comprehensive_testing_qa",
		instance: "localhost",
	}
}

// Name returns the integration name
func (p *PrometheusIntegration) Name() string {
	return "prometheus"
}

// Type returns the integration type
func (p *PrometheusIntegration) Type() interfaces.IntegrationType {
	return interfaces.IntegrationTypeMonitoring
}

// Connect establishes connection to Prometheus
func (p *PrometheusIntegration) Connect(ctx context.Context, config interfaces.Config) error {
	settings := config.Settings

	pushGatewayURL, ok := settings["push_gateway_url"].(string)
	if !ok {
		return fmt.Errorf("push_gateway_url is required")
	}

	if jobName, ok := settings["job_name"].(string); ok {
		p.jobName = jobName
	}

	if instance, ok := settings["instance"].(string); ok {
		p.instance = instance
	}

	p.pushGatewayURL = strings.TrimSuffix(pushGatewayURL, "/")

	// Test connection
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	p.connected = true
	return nil
}

// Disconnect closes the Prometheus connection
func (p *PrometheusIntegration) Disconnect(ctx context.Context) error {
	p.connected = false
	return nil
}

// IsHealthy checks if the Prometheus integration is healthy
func (p *PrometheusIntegration) IsHealthy(ctx context.Context) bool {
	if !p.connected {
		return false
	}

	return p.testConnection(ctx) == nil
}

// SendEvent sends an event to Prometheus as metrics
func (p *PrometheusIntegration) SendEvent(ctx context.Context, event interfaces.Event) error {
	if !p.connected {
		return fmt.Errorf("not connected to Prometheus")
	}

	metrics := p.convertEventToMetrics(event)
	return p.pushMetrics(ctx, metrics)
}

// testConnection tests the Prometheus push gateway connection
func (p *PrometheusIntegration) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/metrics", p.pushGatewayURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Prometheus push gateway returned status %d", resp.StatusCode)
	}

	return nil
}

// convertEventToMetrics converts an event to Prometheus metrics
func (p *PrometheusIntegration) convertEventToMetrics(event interfaces.Event) []PrometheusMetric {
	var metrics []PrometheusMetric

	baseLabels := map[string]string{
		"event_type": string(event.Type),
		"source":     event.Source,
		"priority":   string(event.Priority),
	}

	// Event counter metric
	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_events_total",
		Value:  1,
		Labels: baseLabels,
		Help:   "Total number of testing events",
		Type:   "counter",
	})

	// Event timestamp metric
	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_event_timestamp",
		Value:  float64(event.Timestamp.Unix()),
		Labels: baseLabels,
		Help:   "Timestamp of the last testing event",
		Type:   "gauge",
	})

	// Event-specific metrics
	switch event.Type {
	case interfaces.EventTypeTestFailure:
		metrics = append(metrics, p.getTestFailureMetrics(event, baseLabels)...)
	case interfaces.EventTypeTestSuccess:
		metrics = append(metrics, p.getTestSuccessMetrics(event, baseLabels)...)
	case interfaces.EventTypePerformanceIssue:
		metrics = append(metrics, p.getPerformanceMetrics(event, baseLabels)...)
	case interfaces.EventTypeSecurityAlert:
		metrics = append(metrics, p.getSecurityMetrics(event, baseLabels)...)
	}

	return metrics
}

// getTestFailureMetrics creates metrics for test failures
func (p *PrometheusIntegration) getTestFailureMetrics(event interfaces.Event, baseLabels map[string]string) []PrometheusMetric {
	var metrics []PrometheusMetric

	labels := make(map[string]string)
	for k, v := range baseLabels {
		labels[k] = v
	}

	if testName, ok := event.Data["test_name"].(string); ok {
		labels["test_name"] = testName
	}

	if testSuite, ok := event.Data["test_suite"].(string); ok {
		labels["test_suite"] = testSuite
	}

	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_failures_total",
		Value:  1,
		Labels: labels,
		Help:   "Total number of test failures",
		Type:   "counter",
	})

	if duration, ok := event.Data["duration"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_failure_duration_seconds",
			Value:  duration,
			Labels: labels,
			Help:   "Duration of failed test in seconds",
			Type:   "gauge",
		})
	}

	return metrics
}

// getTestSuccessMetrics creates metrics for test successes
func (p *PrometheusIntegration) getTestSuccessMetrics(event interfaces.Event, baseLabels map[string]string) []PrometheusMetric {
	var metrics []PrometheusMetric

	labels := make(map[string]string)
	for k, v := range baseLabels {
		labels[k] = v
	}

	if testSuite, ok := event.Data["test_suite"].(string); ok {
		labels["test_suite"] = testSuite
	}

	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_successes_total",
		Value:  1,
		Labels: labels,
		Help:   "Total number of test successes",
		Type:   "counter",
	})

	if testCount, ok := event.Data["test_count"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_tests_executed_total",
			Value:  testCount,
			Labels: labels,
			Help:   "Total number of tests executed",
			Type:   "gauge",
		})
	}

	if coverage, ok := event.Data["coverage"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_coverage_percentage",
			Value:  coverage,
			Labels: labels,
			Help:   "Test coverage percentage",
			Type:   "gauge",
		})
	}

	if duration, ok := event.Data["duration"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_suite_duration_seconds",
			Value:  duration,
			Labels: labels,
			Help:   "Test suite execution duration in seconds",
			Type:   "gauge",
		})
	}

	return metrics
}

// getPerformanceMetrics creates metrics for performance issues
func (p *PrometheusIntegration) getPerformanceMetrics(event interfaces.Event, baseLabels map[string]string) []PrometheusMetric {
	var metrics []PrometheusMetric

	labels := make(map[string]string)
	for k, v := range baseLabels {
		labels[k] = v
	}

	if component, ok := event.Data["component"].(string); ok {
		labels["component"] = component
	}

	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_performance_issues_total",
		Value:  1,
		Labels: labels,
		Help:   "Total number of performance issues detected",
		Type:   "counter",
	})

	if responseTime, ok := event.Data["response_time"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_response_time_seconds",
			Value:  responseTime,
			Labels: labels,
			Help:   "Response time in seconds",
			Type:   "gauge",
		})
	}

	if threshold, ok := event.Data["threshold"].(float64); ok {
		metrics = append(metrics, PrometheusMetric{
			Name:   "testing_performance_threshold_seconds",
			Value:  threshold,
			Labels: labels,
			Help:   "Performance threshold in seconds",
			Type:   "gauge",
		})
	}

	return metrics
}

// getSecurityMetrics creates metrics for security alerts
func (p *PrometheusIntegration) getSecurityMetrics(event interfaces.Event, baseLabels map[string]string) []PrometheusMetric {
	var metrics []PrometheusMetric

	labels := make(map[string]string)
	for k, v := range baseLabels {
		labels[k] = v
	}

	if severity, ok := event.Data["severity"].(string); ok {
		labels["severity"] = severity
	}

	if vulnerability, ok := event.Data["vulnerability_type"].(string); ok {
		labels["vulnerability_type"] = vulnerability
	}

	metrics = append(metrics, PrometheusMetric{
		Name:   "testing_security_alerts_total",
		Value:  1,
		Labels: labels,
		Help:   "Total number of security alerts",
		Type:   "counter",
	})

	return metrics
}

// pushMetrics pushes metrics to Prometheus push gateway
func (p *PrometheusIntegration) pushMetrics(ctx context.Context, metrics []PrometheusMetric) error {
	url := fmt.Sprintf("%s/metrics/job/%s/instance/%s", p.pushGatewayURL, p.jobName, p.instance)
	
	body := p.formatMetricsForPushGateway(metrics)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to push metrics, status: %d", resp.StatusCode)
	}

	return nil
}

// formatMetricsForPushGateway formats metrics in Prometheus exposition format
func (p *PrometheusIntegration) formatMetricsForPushGateway(metrics []PrometheusMetric) string {
	var lines []string

	for _, metric := range metrics {
		// Add help comment
		if metric.Help != "" {
			lines = append(lines, fmt.Sprintf("# HELP %s %s", metric.Name, metric.Help))
		}

		// Add type comment
		if metric.Type != "" {
			lines = append(lines, fmt.Sprintf("# TYPE %s %s", metric.Name, metric.Type))
		}

		// Format labels
		var labelPairs []string
		for key, value := range metric.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, key, value))
		}

		// Create metric line
		if len(labelPairs) > 0 {
			lines = append(lines, fmt.Sprintf("%s{%s} %s", 
				metric.Name, 
				strings.Join(labelPairs, ","), 
				strconv.FormatFloat(metric.Value, 'f', -1, 64)))
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", 
				metric.Name, 
				strconv.FormatFloat(metric.Value, 'f', -1, 64)))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}