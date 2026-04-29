package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// SlackIntegration implements Slack communication integration
type SlackIntegration struct {
	webhookURL string
	channel    string
	username   string
	client     *http.Client
	connected  bool
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackIntegration creates a new Slack integration
func NewSlackIntegration() *SlackIntegration {
	return &SlackIntegration{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: "Testing Bot",
	}
}

// Name returns the integration name
func (s *SlackIntegration) Name() string {
	return "slack"
}

// Type returns the integration type
func (s *SlackIntegration) Type() interfaces.IntegrationType {
	return interfaces.IntegrationTypeCommunication
}

// Connect establishes connection to Slack
func (s *SlackIntegration) Connect(ctx context.Context, config interfaces.Config) error {
	settings := config.Settings

	webhookURL, ok := settings["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("webhook_url is required")
	}

	if channel, ok := settings["channel"].(string); ok {
		s.channel = channel
	}

	if username, ok := settings["username"].(string); ok {
		s.username = username
	}

	s.webhookURL = webhookURL

	// Test connection
	if err := s.testConnection(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	s.connected = true
	return nil
}

// Disconnect closes the Slack connection
func (s *SlackIntegration) Disconnect(ctx context.Context) error {
	s.connected = false
	return nil
}

// IsHealthy checks if the Slack integration is healthy
func (s *SlackIntegration) IsHealthy(ctx context.Context) bool {
	if !s.connected {
		return false
	}

	return s.testConnection(ctx) == nil
}

// SendEvent sends an event to Slack
func (s *SlackIntegration) SendEvent(ctx context.Context, event interfaces.Event) error {
	if !s.connected {
		return fmt.Errorf("not connected to Slack")
	}

	// Only send critical events to avoid spam
	if event.Priority != interfaces.PriorityCritical {
		return nil
	}

	message := s.createMessageFromEvent(event)
	return s.sendMessage(ctx, message)
}

// testConnection tests the Slack connection by sending a test message
func (s *SlackIntegration) testConnection(ctx context.Context) error {
	testMessage := SlackMessage{
		Channel:   s.channel,
		Username:  s.username,
		Text:      "Testing connection to Comprehensive Testing & QA System",
		IconEmoji: ":robot_face:",
	}

	return s.sendMessage(ctx, testMessage)
}

// createMessageFromEvent creates a Slack message from an event
func (s *SlackIntegration) createMessageFromEvent(event interfaces.Event) SlackMessage {
	message := SlackMessage{
		Channel:   s.channel,
		Username:  s.username,
		IconEmoji: s.getEmojiForEvent(event),
	}

	switch event.Type {
	case interfaces.EventTypeTestFailure:
		message.Text = ":x: Critical Test Failure Detected"
		message.Attachments = []SlackAttachment{s.createTestFailureAttachment(event)}
	case interfaces.EventTypeSecurityAlert:
		message.Text = ":warning: Security Alert"
		message.Attachments = []SlackAttachment{s.createSecurityAttachment(event)}
	case interfaces.EventTypePerformanceIssue:
		message.Text = ":chart_with_downwards_trend: Performance Issue Detected"
		message.Attachments = []SlackAttachment{s.createPerformanceAttachment(event)}
	case interfaces.EventTypeDeployment:
		message.Text = ":rocket: Deployment Event"
		message.Attachments = []SlackAttachment{s.createDeploymentAttachment(event)}
	default:
		message.Text = fmt.Sprintf(":information_source: %s Event", event.Type)
		message.Attachments = []SlackAttachment{s.createGenericAttachment(event)}
	}

	return message
}

// createTestFailureAttachment creates an attachment for test failure events
func (s *SlackIntegration) createTestFailureAttachment(event interfaces.Event) SlackAttachment {
	attachment := SlackAttachment{
		Color:     "danger",
		Title:     "Test Failure Details",
		Timestamp: event.Timestamp.Unix(),
		Footer:    "Comprehensive Testing & QA System",
	}

	var fields []SlackField

	if testName, ok := event.Data["test_name"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Test Name",
			Value: testName,
			Short: true,
		})
	}

	if testSuite, ok := event.Data["test_suite"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Test Suite",
			Value: testSuite,
			Short: true,
		})
	}

	if errorMsg, ok := event.Data["error"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Error",
			Value: fmt.Sprintf("```%s```", errorMsg),
			Short: false,
		})
	}

	fields = append(fields, SlackField{
		Title: "Priority",
		Value: string(event.Priority),
		Short: true,
	})

	fields = append(fields, SlackField{
		Title: "Source",
		Value: event.Source,
		Short: true,
	})

	attachment.Fields = fields
	return attachment
}

// createSecurityAttachment creates an attachment for security events
func (s *SlackIntegration) createSecurityAttachment(event interfaces.Event) SlackAttachment {
	attachment := SlackAttachment{
		Color:     "danger",
		Title:     "Security Alert Details",
		Timestamp: event.Timestamp.Unix(),
		Footer:    "Comprehensive Testing & QA System",
	}

	var fields []SlackField

	if vulnerability, ok := event.Data["vulnerability_type"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Vulnerability Type",
			Value: vulnerability,
			Short: true,
		})
	}

	if severity, ok := event.Data["severity"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Severity",
			Value: severity,
			Short: true,
		})
	}

	if description, ok := event.Data["description"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Description",
			Value: description,
			Short: false,
		})
	}

	if file, ok := event.Data["file"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Affected File",
			Value: file,
			Short: true,
		})
	}

	attachment.Fields = fields
	return attachment
}

// createPerformanceAttachment creates an attachment for performance events
func (s *SlackIntegration) createPerformanceAttachment(event interfaces.Event) SlackAttachment {
	attachment := SlackAttachment{
		Color:     "warning",
		Title:     "Performance Issue Details",
		Timestamp: event.Timestamp.Unix(),
		Footer:    "Comprehensive Testing & QA System",
	}

	var fields []SlackField

	if component, ok := event.Data["component"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Component",
			Value: component,
			Short: true,
		})
	}

	if responseTime, ok := event.Data["response_time"].(float64); ok {
		fields = append(fields, SlackField{
			Title: "Response Time",
			Value: fmt.Sprintf("%.2fs", responseTime),
			Short: true,
		})
	}

	if threshold, ok := event.Data["threshold"].(float64); ok {
		fields = append(fields, SlackField{
			Title: "Threshold",
			Value: fmt.Sprintf("%.2fs", threshold),
			Short: true,
		})
	}

	if metric, ok := event.Data["metric"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Metric",
			Value: metric,
			Short: true,
		})
	}

	attachment.Fields = fields
	return attachment
}

// createDeploymentAttachment creates an attachment for deployment events
func (s *SlackIntegration) createDeploymentAttachment(event interfaces.Event) SlackAttachment {
	attachment := SlackAttachment{
		Color:     "good",
		Title:     "Deployment Details",
		Timestamp: event.Timestamp.Unix(),
		Footer:    "Comprehensive Testing & QA System",
	}

	var fields []SlackField

	if environment, ok := event.Data["environment"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Environment",
			Value: environment,
			Short: true,
		})
	}

	if version, ok := event.Data["version"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Version",
			Value: version,
			Short: true,
		})
	}

	if status, ok := event.Data["status"].(string); ok {
		fields = append(fields, SlackField{
			Title: "Status",
			Value: status,
			Short: true,
		})
	}

	attachment.Fields = fields
	return attachment
}

// createGenericAttachment creates a generic attachment for other events
func (s *SlackIntegration) createGenericAttachment(event interfaces.Event) SlackAttachment {
	attachment := SlackAttachment{
		Color:     "good",
		Title:     fmt.Sprintf("%s Event", event.Type),
		Timestamp: event.Timestamp.Unix(),
		Footer:    "Comprehensive Testing & QA System",
	}

	var fields []SlackField

	fields = append(fields, SlackField{
		Title: "Event ID",
		Value: event.ID,
		Short: true,
	})

	fields = append(fields, SlackField{
		Title: "Source",
		Value: event.Source,
		Short: true,
	})

	fields = append(fields, SlackField{
		Title: "Priority",
		Value: string(event.Priority),
		Short: true,
	})

	attachment.Fields = fields
	return attachment
}

// getEmojiForEvent returns an appropriate emoji for the event type
func (s *SlackIntegration) getEmojiForEvent(event interfaces.Event) string {
	switch event.Type {
	case interfaces.EventTypeTestFailure:
		return ":x:"
	case interfaces.EventTypeTestSuccess:
		return ":white_check_mark:"
	case interfaces.EventTypeSecurityAlert:
		return ":warning:"
	case interfaces.EventTypePerformanceIssue:
		return ":chart_with_downwards_trend:"
	case interfaces.EventTypeDeployment:
		return ":rocket:"
	default:
		return ":robot_face:"
	}
}

// sendMessage sends a message to Slack
func (s *SlackIntegration) sendMessage(ctx context.Context, message SlackMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}