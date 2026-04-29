package interfaces

import (
	"context"
	"time"
)

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeBugTracking     IntegrationType = "bug_tracking"
	IntegrationTypeProjectMgmt     IntegrationType = "project_management"
	IntegrationTypeCodeReview      IntegrationType = "code_review"
	IntegrationTypeMonitoring      IntegrationType = "monitoring"
	IntegrationTypeCommunication   IntegrationType = "communication"
	IntegrationTypeCI              IntegrationType = "ci_cd"
	IntegrationTypeWorkflow        IntegrationType = "workflow"
	IntegrationTypeFailover        IntegrationType = "failover"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeTestFailure     EventType = "test_failure"
	EventTypeTestSuccess     EventType = "test_success"
	EventTypeDeployment      EventType = "deployment"
	EventTypeSecurityAlert   EventType = "security_alert"
	EventTypePerformanceIssue EventType = "performance_issue"
	EventTypeCodeReview      EventType = "code_review"
)

// EventPriority represents the priority of an event
type EventPriority string

const (
	PriorityLow      EventPriority = "low"
	PriorityMedium   EventPriority = "medium"
	PriorityHigh     EventPriority = "high"
	PriorityCritical EventPriority = "critical"
)

// Config represents integration configuration
type Config struct {
	Name     string                 `json:"name"`
	Type     IntegrationType        `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}

// Event represents an integration event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  EventPriority          `json:"priority"`
	Data      map[string]interface{} `json:"data"`
}

// Integration represents a generic integration interface
type Integration interface {
	Name() string
	Type() IntegrationType
	Connect(ctx context.Context, config Config) error
	Disconnect(ctx context.Context) error
	IsHealthy(ctx context.Context) bool
	SendEvent(ctx context.Context, event Event) error
}

// IntegrationStatus represents the status of an integration
type IntegrationStatus struct {
	Name    string            `json:"name"`
	Type    IntegrationType   `json:"type"`
	Healthy bool              `json:"healthy"`
	Metrics IntegrationMetrics `json:"metrics"`
}

// IntegrationMetrics holds metrics for an integration
type IntegrationMetrics struct {
	EventsSent     int64     `json:"events_sent"`
	ErrorCount     int64     `json:"error_count"`
	LastEventTime  time.Time `json:"last_event_time"`
	LastErrorTime  time.Time `json:"last_error_time"`
	SuccessRate    float64   `json:"success_rate"`
}