package bugtracking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// JiraIntegration implements JIRA bug tracking integration
type JiraIntegration struct {
	baseURL    string
	username   string
	apiToken   string
	projectKey string
	client     *http.Client
	connected  bool
}

// JiraIssue represents a JIRA issue
type JiraIssue struct {
	Fields JiraFields `json:"fields"`
}

// JiraFields represents JIRA issue fields
type JiraFields struct {
	Project     JiraProject   `json:"project"`
	Summary     string        `json:"summary"`
	Description string        `json:"description"`
	IssueType   JiraIssueType `json:"issuetype"`
	Priority    JiraPriority  `json:"priority"`
	Labels      []string      `json:"labels"`
}

// JiraProject represents a JIRA project
type JiraProject struct {
	Key string `json:"key"`
}

// JiraIssueType represents a JIRA issue type
type JiraIssueType struct {
	Name string `json:"name"`
}

// JiraPriority represents a JIRA priority
type JiraPriority struct {
	Name string `json:"name"`
}

// NewJiraIntegration creates a new JIRA integration
func NewJiraIntegration() *JiraIntegration {
	return &JiraIntegration{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the integration name
func (j *JiraIntegration) Name() string {
	return "jira"
}

// Type returns the integration type
func (j *JiraIntegration) Type() interfaces.IntegrationType {
	return interfaces.IntegrationTypeBugTracking
}

// Connect establishes connection to JIRA
func (j *JiraIntegration) Connect(ctx context.Context, config interfaces.Config) error {
	settings := config.Settings

	baseURL, ok := settings["base_url"].(string)
	if !ok {
		return fmt.Errorf("base_url is required")
	}

	username, ok := settings["username"].(string)
	if !ok {
		return fmt.Errorf("username is required")
	}

	apiToken, ok := settings["api_token"].(string)
	if !ok {
		return fmt.Errorf("api_token is required")
	}

	projectKey, ok := settings["project_key"].(string)
	if !ok {
		return fmt.Errorf("project_key is required")
	}

	j.baseURL = baseURL
	j.username = username
	j.apiToken = apiToken
	j.projectKey = projectKey

	// Test connection
	if err := j.testConnection(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	j.connected = true
	return nil
}

// Disconnect closes the JIRA connection
func (j *JiraIntegration) Disconnect(ctx context.Context) error {
	j.connected = false
	return nil
}

// IsHealthy checks if the JIRA integration is healthy
func (j *JiraIntegration) IsHealthy(ctx context.Context) bool {
	if !j.connected {
		return false
	}

	return j.testConnection(ctx) == nil
}

// SendEvent sends an event to JIRA (creates an issue)
func (j *JiraIntegration) SendEvent(ctx context.Context, event interfaces.Event) error {
	if !j.connected {
		return fmt.Errorf("not connected to JIRA")
	}

	// Only handle test failures and security alerts
	if event.Type != interfaces.EventTypeTestFailure && event.Type != interfaces.EventTypeSecurityAlert {
		return nil
	}

	issue := j.createIssueFromEvent(event)
	return j.createIssue(ctx, issue)
}

// testConnection tests the JIRA connection
func (j *JiraIntegration) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/rest/api/2/myself", j.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(j.username, j.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JIRA API returned status %d", resp.StatusCode)
	}

	return nil
}

// createIssueFromEvent creates a JIRA issue from an event
func (j *JiraIntegration) createIssueFromEvent(event interfaces.Event) JiraIssue {
	summary := fmt.Sprintf("[%s] %s", event.Type, j.getEventSummary(event))
	description := j.getEventDescription(event)
	
	priority := j.mapPriorityToJira(event.Priority)
	issueType := j.mapEventTypeToJiraIssueType(event.Type)

	return JiraIssue{
		Fields: JiraFields{
			Project: JiraProject{
				Key: j.projectKey,
			},
			Summary:     summary,
			Description: description,
			IssueType: JiraIssueType{
				Name: issueType,
			},
			Priority: JiraPriority{
				Name: priority,
			},
			Labels: []string{"automated", "testing", string(event.Type)},
		},
	}
}

// createIssue creates a JIRA issue
func (j *JiraIntegration) createIssue(ctx context.Context, issue JiraIssue) error {
	url := fmt.Sprintf("%s/rest/api/2/issue", j.baseURL)
	
	jsonData, err := json.Marshal(issue)
	if err != nil {
		return fmt.Errorf("failed to marshal issue: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.SetBasicAuth(j.username, j.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create JIRA issue, status: %d", resp.StatusCode)
	}

	return nil
}

// getEventSummary extracts a summary from the event
func (j *JiraIntegration) getEventSummary(event interfaces.Event) string {
	if summary, ok := event.Data["summary"].(string); ok {
		return summary
	}
	if testName, ok := event.Data["test_name"].(string); ok {
		return fmt.Sprintf("Test failure: %s", testName)
	}
	return fmt.Sprintf("Event from %s", event.Source)
}

// getEventDescription creates a detailed description from the event
func (j *JiraIntegration) getEventDescription(event interfaces.Event) string {
	description := fmt.Sprintf("Event ID: %s\nSource: %s\nTimestamp: %s\nPriority: %s\n\n",
		event.ID, event.Source, event.Timestamp.Format(time.RFC3339), event.Priority)

	if errorMsg, ok := event.Data["error"].(string); ok {
		description += fmt.Sprintf("Error: %s\n\n", errorMsg)
	}

	if stackTrace, ok := event.Data["stack_trace"].(string); ok {
		description += fmt.Sprintf("Stack Trace:\n{code}\n%s\n{code}\n\n", stackTrace)
	}

	if details, ok := event.Data["details"].(string); ok {
		description += fmt.Sprintf("Details: %s\n", details)
	}

	return description
}

// mapPriorityToJira maps event priority to JIRA priority
func (j *JiraIntegration) mapPriorityToJira(priority interfaces.EventPriority) string {
	switch priority {
	case interfaces.PriorityCritical:
		return "Highest"
	case interfaces.PriorityHigh:
		return "High"
	case interfaces.PriorityMedium:
		return "Medium"
	case interfaces.PriorityLow:
		return "Low"
	default:
		return "Medium"
	}
}

// mapEventTypeToJiraIssueType maps event type to JIRA issue type
func (j *JiraIntegration) mapEventTypeToJiraIssueType(eventType interfaces.EventType) string {
	switch eventType {
	case interfaces.EventTypeTestFailure:
		return "Bug"
	case interfaces.EventTypeSecurityAlert:
		return "Security Issue"
	default:
		return "Task"
	}
}