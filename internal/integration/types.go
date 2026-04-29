package integration

import (
	"high-performance-news-website/internal/integration/interfaces"
)

// Re-export types from interfaces package to maintain compatibility
type IntegrationType = interfaces.IntegrationType
type EventType = interfaces.EventType
type EventPriority = interfaces.EventPriority
type Config = interfaces.Config
type Event = interfaces.Event
type Integration = interfaces.Integration
type IntegrationStatus = interfaces.IntegrationStatus
type IntegrationMetrics = interfaces.IntegrationMetrics

// Re-export constants
const (
	IntegrationTypeBugTracking     = interfaces.IntegrationTypeBugTracking
	IntegrationTypeProjectMgmt     = interfaces.IntegrationTypeProjectMgmt
	IntegrationTypeCodeReview      = interfaces.IntegrationTypeCodeReview
	IntegrationTypeMonitoring      = interfaces.IntegrationTypeMonitoring
	IntegrationTypeCommunication   = interfaces.IntegrationTypeCommunication
	IntegrationTypeCI              = interfaces.IntegrationTypeCI
	IntegrationTypeWorkflow        = interfaces.IntegrationTypeWorkflow
	IntegrationTypeFailover        = interfaces.IntegrationTypeFailover

	EventTypeTestFailure     = interfaces.EventTypeTestFailure
	EventTypeTestSuccess     = interfaces.EventTypeTestSuccess
	EventTypeDeployment      = interfaces.EventTypeDeployment
	EventTypeSecurityAlert   = interfaces.EventTypeSecurityAlert
	EventTypePerformanceIssue = interfaces.EventTypePerformanceIssue
	EventTypeCodeReview      = interfaces.EventTypeCodeReview

	PriorityLow      = interfaces.PriorityLow
	PriorityMedium   = interfaces.PriorityMedium
	PriorityHigh     = interfaces.PriorityHigh
	PriorityCritical = interfaces.PriorityCritical
)