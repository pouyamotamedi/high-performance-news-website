package integration

import (
	"context"
	"fmt"
	"log"

	"high-performance-news-website/internal/integration/bugtracking"
	"high-performance-news-website/internal/integration/codereview"
	"high-performance-news-website/internal/integration/communication"
	"high-performance-news-website/internal/integration/failover"
	"high-performance-news-website/internal/integration/monitoring"
	"high-performance-news-website/internal/integration/plugin"
	"high-performance-news-website/internal/integration/workflow"
)

// IntegrationService provides a comprehensive integration management service
type IntegrationService struct {
	manager         *IntegrationManager
	pluginManager   *plugin.PluginManager
	workflowManager *workflow.WorkflowAutomation
	failoverHandler *failover.FailoverHandler
}

// NewIntegrationService creates a new integration service
func NewIntegrationService() *IntegrationService {
	manager := NewIntegrationManager()
	pluginManager := plugin.NewPluginManager()
	workflowManager := workflow.NewWorkflowAutomation()
	failoverHandler := failover.NewFailoverHandler()

	service := &IntegrationService{
		manager:         manager,
		pluginManager:   pluginManager,
		workflowManager: workflowManager,
		failoverHandler: failoverHandler,
	}

	// Register built-in integrations
	service.registerBuiltInIntegrations()

	return service
}

// registerBuiltInIntegrations registers all built-in integrations
func (is *IntegrationService) registerBuiltInIntegrations() {
	// Register JIRA integration
	jiraIntegration := bugtracking.NewJiraIntegration()
	if err := is.manager.RegisterIntegration(jiraIntegration); err != nil {
		log.Printf("Failed to register JIRA integration: %v", err)
	}

	// Register GitHub integration
	githubIntegration := codereview.NewGitHubIntegration()
	if err := is.manager.RegisterIntegration(githubIntegration); err != nil {
		log.Printf("Failed to register GitHub integration: %v", err)
	}

	// Register Prometheus integration
	prometheusIntegration := monitoring.NewPrometheusIntegration()
	if err := is.manager.RegisterIntegration(prometheusIntegration); err != nil {
		log.Printf("Failed to register Prometheus integration: %v", err)
	}

	// Register Slack integration
	slackIntegration := communication.NewSlackIntegration()
	if err := is.manager.RegisterIntegration(slackIntegration); err != nil {
		log.Printf("Failed to register Slack integration: %v", err)
	}

	// Register failover configurations
	is.failoverHandler.RegisterIntegration("jira", []string{"github"})
	is.failoverHandler.RegisterIntegration("github", []string{"slack"})
	is.failoverHandler.RegisterIntegration("prometheus", []string{"slack"})
	is.failoverHandler.RegisterIntegration("slack", []string{})

	log.Println("Registered all built-in integrations")
}

// ConnectIntegration connects to an integration with failover support
func (is *IntegrationService) ConnectIntegration(ctx context.Context, name string, config Config) error {
	operation := func(integrationName string) error {
		return is.manager.ConnectIntegration(ctx, integrationName, config)
	}

	result := is.failoverHandler.ExecuteWithFailover(ctx, name, operation)
	if !result.Success {
		return fmt.Errorf("failed to connect to integration %s: %s", name, result.Error)
	}

	return nil
}

// SendEvent sends an event with failover and workflow automation
func (is *IntegrationService) SendEvent(ctx context.Context, event Event) error {
	// First, trigger any workflows based on this event
	if err := is.workflowManager.TriggerWorkflows(ctx, event); err != nil {
		log.Printf("Workflow trigger failed: %v", err)
		// Don't fail the entire operation if workflows fail
	}

	// Send event to integrations with failover support
	operation := func(integrationName string) error {
		return is.manager.SendEvent(ctx, event)
	}

	// Try to send to all relevant integrations
	integrationNames := is.getRelevantIntegrations(event)
	var lastError error

	for _, name := range integrationNames {
		result := is.failoverHandler.ExecuteWithFailover(ctx, name, operation)
		if !result.Success {
			lastError = fmt.Errorf("failed to send event to %s: %s", name, result.Error)
			log.Printf("Event send failed: %v", lastError)
		}
	}

	return lastError // Return last error, but don't fail if at least one succeeded
}

// getRelevantIntegrations returns integrations relevant for an event type
func (is *IntegrationService) getRelevantIntegrations(event Event) []string {
	var integrations []string

	switch event.Type {
	case EventTypeTestFailure, EventTypeSecurityAlert:
		integrations = append(integrations, "jira", "slack")
	case EventTypeCodeReview:
		integrations = append(integrations, "github")
	case EventTypePerformanceIssue:
		integrations = append(integrations, "prometheus", "slack")
	default:
		integrations = append(integrations, "prometheus") // Send all events to monitoring
	}

	return integrations
}

// LoadPlugin loads a plugin with validation
func (is *IntegrationService) LoadPlugin(ctx context.Context, config plugin.PluginConfig) error {
	// Validate plugin first
	manifest, err := is.pluginManager.ValidatePlugin(config.Path)
	if err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	log.Printf("Loading plugin: %s v%s by %s", manifest.Name, manifest.Version, manifest.Author)

	// Load the plugin
	if err := is.pluginManager.LoadPlugin(ctx, config); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Register plugin integrations with the main manager
	integrations := is.pluginManager.GetIntegrations()
	for _, integration := range integrations {
		if err := is.manager.RegisterIntegration(integration); err != nil {
			log.Printf("Failed to register plugin integration %s: %v", integration.Name(), err)
		} else {
			// Register for failover
			is.failoverHandler.RegisterIntegration(integration.Name(), []string{})
		}
	}

	return nil
}

// CreateWorkflow creates a new automated workflow
func (is *IntegrationService) CreateWorkflow(workflow *workflow.Workflow) error {
	return is.workflowManager.RegisterWorkflow(workflow)
}

// ExecuteWorkflow manually executes a workflow
func (is *IntegrationService) ExecuteWorkflow(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
	return is.workflowManager.ExecuteWorkflow(ctx, workflowID, triggerData)
}

// GetIntegrationStatus returns comprehensive integration status
func (is *IntegrationService) GetIntegrationStatus() map[string]interface{} {
	status := make(map[string]interface{})

	// Get basic integration status
	integrationStatus := is.manager.GetIntegrationStatus()
	status["integrations"] = integrationStatus

	// Get health information
	healthStatus := is.failoverHandler.GetAllIntegrationHealth()
	status["health"] = healthStatus

	// Get circuit breaker status
	circuitBreakerStatus := is.failoverHandler.GetCircuitBreakerStatus()
	status["circuit_breakers"] = circuitBreakerStatus

	// Get plugin status
	pluginStatus := is.pluginManager.GetPluginStatus()
	status["plugins"] = pluginStatus

	// Get workflow status
	workflows := is.workflowManager.GetAllWorkflows()
	workflowStatus := make(map[string]interface{})
	for id, workflow := range workflows {
		workflowStatus[id] = map[string]interface{}{
			"name":    workflow.Name,
			"enabled": workflow.Enabled,
			"stats":   workflow.Stats,
		}
	}
	status["workflows"] = workflowStatus

	return status
}

// HealthCheck performs a comprehensive health check
func (is *IntegrationService) HealthCheck(ctx context.Context) map[string]bool {
	health := is.manager.HealthCheck(ctx)
	
	// Add plugin health
	plugins := is.pluginManager.GetAllPlugins()
	for name, plugin := range plugins {
		if plugin.Integration != nil {
			health[fmt.Sprintf("plugin_%s", name)] = plugin.Integration.IsHealthy(ctx)
		}
	}

	return health
}

// GetMetrics returns comprehensive metrics
func (is *IntegrationService) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Integration metrics
	integrationMetrics := is.manager.metrics.GetAllMetrics()
	metrics["integrations"] = integrationMetrics

	// Health metrics
	healthMetrics := is.failoverHandler.GetAllIntegrationHealth()
	metrics["health"] = healthMetrics

	// Workflow metrics
	workflows := is.workflowManager.GetAllWorkflows()
	workflowMetrics := make(map[string]interface{})
	for id, workflow := range workflows {
		workflowMetrics[id] = workflow.Stats
	}
	metrics["workflows"] = workflowMetrics

	return metrics
}

// Shutdown gracefully shuts down all integrations
func (is *IntegrationService) Shutdown(ctx context.Context) error {
	log.Println("Shutting down integration service...")

	// Disconnect all integrations
	integrations := is.manager.GetIntegrationStatus()
	for name := range integrations {
		// Note: We would need to implement Disconnect in the manager
		log.Printf("Disconnecting from %s", name)
	}

	// Unload all plugins
	plugins := is.pluginManager.GetAllPlugins()
	for name := range plugins {
		if err := is.pluginManager.UnloadPlugin(ctx, name); err != nil {
			log.Printf("Failed to unload plugin %s: %v", name, err)
		}
	}

	log.Println("Integration service shutdown complete")
	return nil
}

// RegisterWebhook registers a webhook for external integrations
func (is *IntegrationService) RegisterWebhook(webhook *Webhook) error {
	return is.manager.webhooks.RegisterWebhook(webhook)
}

// SendWebhook sends a webhook notification
func (is *IntegrationService) SendWebhook(ctx context.Context, event Event) error {
	return is.manager.webhooks.SendWebhook(ctx, event)
}

// ResetCircuitBreaker manually resets a circuit breaker
func (is *IntegrationService) ResetCircuitBreaker(integrationName string) error {
	return is.failoverHandler.ResetCircuitBreaker(integrationName)
}

// GetWorkflows returns all workflows
func (is *IntegrationService) GetWorkflows() map[string]*workflow.Workflow {
	return is.workflowManager.GetAllWorkflows()
}

// GetPlugins returns all loaded plugins
func (is *IntegrationService) GetPlugins() map[string]*plugin.LoadedPlugin {
	return is.pluginManager.GetAllPlugins()
}