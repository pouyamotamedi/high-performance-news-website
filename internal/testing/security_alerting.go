package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// SecurityAlerting handles security alert notifications
type SecurityAlerting struct {
	webhookURL     string
	slackToken     string
	emailConfig    EmailConfig
	alertThreshold AlertThreshold
	httpClient     *http.Client
}

// EmailConfig contains email notification settings
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	ToEmails     []string
}

// AlertThreshold defines when to send alerts
type AlertThreshold struct {
	CriticalIssues int
	HighIssues     int
	TotalIssues    int
}

// AlertPayload represents the structure of security alerts
type AlertPayload struct {
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	ScanResult  *SecurityScanResult    `json:"scan_result"`
	Summary     SecuritySummary        `json:"summary"`
	TopIssues   []SecurityIssue        `json:"top_issues"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SlackMessage represents a Slack notification message
type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title"`
	Text      string       `json:"text"`
	Fields    []SlackField `json:"fields"`
	Timestamp int64        `json:"ts"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSecurityAlerting creates a new security alerting instance
func NewSecurityAlerting() *SecurityAlerting {
	return &SecurityAlerting{
		webhookURL: os.Getenv("SECURITY_WEBHOOK_URL"),
		slackToken: os.Getenv("SLACK_TOKEN"),
		emailConfig: EmailConfig{
			SMTPHost:     getEnvOrDefault("SMTP_HOST", "localhost"),
			SMTPPort:     587,
			SMTPUsername: os.Getenv("SMTP_USERNAME"),
			SMTPPassword: os.Getenv("SMTP_PASSWORD"),
			FromEmail:    getEnvOrDefault("SECURITY_FROM_EMAIL", "security@example.com"),
			ToEmails:     strings.Split(os.Getenv("SECURITY_TO_EMAILS"), ","),
		},
		alertThreshold: AlertThreshold{
			CriticalIssues: 1,  // Alert on any critical issue
			HighIssues:     5,  // Alert on 5+ high issues
			TotalIssues:    20, // Alert on 20+ total issues
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessSecurityScanResult processes scan results and sends alerts if needed
func (s *SecurityAlerting) ProcessSecurityScanResult(ctx context.Context, result *SecurityScanResult) error {
	if !s.shouldAlert(result) {
		log.Println("Security scan completed - no alerts needed")
		return nil
	}

	alert := s.createAlert(result)

	// Send alerts through all configured channels
	var errors []string

	// Send webhook alert
	if s.webhookURL != "" {
		if err := s.sendWebhookAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Sprintf("webhook: %v", err))
		}
	}

	// Send Slack alert
	if s.slackToken != "" {
		if err := s.sendSlackAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Sprintf("slack: %v", err))
		}
	}

	// Send email alert
	if len(s.emailConfig.ToEmails) > 0 && s.emailConfig.ToEmails[0] != "" {
		if err := s.sendEmailAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Sprintf("email: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("alert sending failed: %s", strings.Join(errors, ", "))
	}

	log.Printf("Security alerts sent successfully for scan with %d issues", result.Summary.TotalIssues)
	return nil
}

// shouldAlert determines if an alert should be sent based on scan results
func (s *SecurityAlerting) shouldAlert(result *SecurityScanResult) bool {
	summary := result.Summary

	// Always alert on critical issues
	if summary.CriticalIssues >= s.alertThreshold.CriticalIssues {
		return true
	}

	// Alert on high number of high-severity issues
	if summary.HighIssues >= s.alertThreshold.HighIssues {
		return true
	}

	// Alert on high total number of issues
	if summary.TotalIssues >= s.alertThreshold.TotalIssues {
		return true
	}

	return false
}

// createAlert creates an alert payload from scan results
func (s *SecurityAlerting) createAlert(result *SecurityScanResult) *AlertPayload {
	severity := s.determineSeverity(result.Summary)
	
	// Get top 5 most critical issues
	topIssues := s.getTopIssues(result.Issues, 5)

	alert := &AlertPayload{
		Title:      fmt.Sprintf("Security Scan Alert - %s", strings.Title(severity)),
		Message:    s.createAlertMessage(result),
		Severity:   severity,
		Timestamp:  time.Now(),
		ScanResult: result,
		Summary:    result.Summary,
		TopIssues:  topIssues,
		Metadata: map[string]interface{}{
			"scan_duration": result.Duration.String(),
			"scan_type":     result.ScanType,
		},
	}

	return alert
}

// determineSeverity determines the overall severity of the scan results
func (s *SecurityAlerting) determineSeverity(summary SecuritySummary) string {
	if summary.CriticalIssues > 0 {
		return "critical"
	}
	if summary.HighIssues >= 5 {
		return "high"
	}
	if summary.HighIssues > 0 || summary.MediumIssues >= 10 {
		return "medium"
	}
	return "low"
}

// createAlertMessage creates a human-readable alert message
func (s *SecurityAlerting) createAlertMessage(result *SecurityScanResult) string {
	summary := result.Summary
	
	var message strings.Builder
	message.WriteString(fmt.Sprintf("Security scan completed with %d issues found:\n", summary.TotalIssues))
	
	if summary.CriticalIssues > 0 {
		message.WriteString(fmt.Sprintf("🔴 Critical: %d\n", summary.CriticalIssues))
	}
	if summary.HighIssues > 0 {
		message.WriteString(fmt.Sprintf("🟠 High: %d\n", summary.HighIssues))
	}
	if summary.MediumIssues > 0 {
		message.WriteString(fmt.Sprintf("🟡 Medium: %d\n", summary.MediumIssues))
	}
	if summary.LowIssues > 0 {
		message.WriteString(fmt.Sprintf("🟢 Low: %d\n", summary.LowIssues))
	}
	if summary.InfoIssues > 0 {
		message.WriteString(fmt.Sprintf("ℹ️ Info: %d\n", summary.InfoIssues))
	}

	message.WriteString(fmt.Sprintf("\nScan Duration: %s", result.Duration.String()))
	
	if result.ReportPath != "" {
		message.WriteString(fmt.Sprintf("\nDetailed report: %s", result.ReportPath))
	}

	return message.String()
}

// getTopIssues returns the most critical issues
func (s *SecurityAlerting) getTopIssues(issues []SecurityIssue, limit int) []SecurityIssue {
	// Simple sorting by severity (critical > high > medium > low > info)
	var sortedIssues []SecurityIssue
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		for _, issue := range issues {
			if issue.Severity == severity {
				sortedIssues = append(sortedIssues, issue)
				if len(sortedIssues) >= limit {
					return sortedIssues
				}
			}
		}
	}

	return sortedIssues
}

// sendWebhookAlert sends alert via webhook
func (s *SecurityAlerting) sendWebhookAlert(ctx context.Context, alert *AlertPayload) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// sendSlackAlert sends alert to Slack
func (s *SecurityAlerting) sendSlackAlert(ctx context.Context, alert *AlertPayload) error {
	color := s.getSeverityColor(alert.Severity)
	
	slackMsg := SlackMessage{
		Text:     "Security Scan Alert",
		Username: "Security Scanner",
		IconEmoji: ":warning:",
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: alert.Title,
				Text:  alert.Message,
				Fields: []SlackField{
					{
						Title: "Critical Issues",
						Value: fmt.Sprintf("%d", alert.Summary.CriticalIssues),
						Short: true,
					},
					{
						Title: "High Issues",
						Value: fmt.Sprintf("%d", alert.Summary.HighIssues),
						Short: true,
					},
					{
						Title: "Total Issues",
						Value: fmt.Sprintf("%d", alert.Summary.TotalIssues),
						Short: true,
					},
					{
						Title: "Scan Duration",
						Value: alert.ScanResult.Duration.String(),
						Short: true,
					},
				},
				Timestamp: alert.Timestamp.Unix(),
			},
		},
	}

	// Add top issues as additional fields
	if len(alert.TopIssues) > 0 {
		var topIssuesText strings.Builder
		for i, issue := range alert.TopIssues {
			if i >= 3 { // Limit to top 3 for Slack
				break
			}
			topIssuesText.WriteString(fmt.Sprintf("• %s (%s)\n", issue.Title, issue.Severity))
		}

		slackMsg.Attachments[0].Fields = append(slackMsg.Attachments[0].Fields, SlackField{
			Title: "Top Issues",
			Value: topIssuesText.String(),
			Short: false,
		})
	}

	payload, err := json.Marshal(slackMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	// Use webhook URL for Slack (assuming it's a Slack webhook)
	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendEmailAlert sends alert via email
func (s *SecurityAlerting) sendEmailAlert(ctx context.Context, alert *AlertPayload) error {
	// For simplicity, we'll log the email content
	// In a real implementation, you'd use a proper email library like gomail
	
	subject := fmt.Sprintf("[SECURITY ALERT] %s", alert.Title)
	body := s.createEmailBody(alert)
	
	log.Printf("EMAIL ALERT would be sent to %v:\nSubject: %s\nBody:\n%s", 
		s.emailConfig.ToEmails, subject, body)
	
	// TODO: Implement actual email sending using SMTP
	// This is a placeholder for the actual email implementation
	
	return nil
}

// createEmailBody creates the email body for security alerts
func (s *SecurityAlerting) createEmailBody(alert *AlertPayload) string {
	var body strings.Builder
	
	body.WriteString(fmt.Sprintf("Security Scan Alert\n"))
	body.WriteString(fmt.Sprintf("==================\n\n"))
	body.WriteString(fmt.Sprintf("Timestamp: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05")))
	body.WriteString(fmt.Sprintf("Severity: %s\n", strings.ToUpper(alert.Severity)))
	body.WriteString(fmt.Sprintf("Scan Type: %s\n", alert.ScanResult.ScanType))
	body.WriteString(fmt.Sprintf("Duration: %s\n\n", alert.ScanResult.Duration.String()))
	
	body.WriteString("Summary:\n")
	body.WriteString(fmt.Sprintf("- Critical Issues: %d\n", alert.Summary.CriticalIssues))
	body.WriteString(fmt.Sprintf("- High Issues: %d\n", alert.Summary.HighIssues))
	body.WriteString(fmt.Sprintf("- Medium Issues: %d\n", alert.Summary.MediumIssues))
	body.WriteString(fmt.Sprintf("- Low Issues: %d\n", alert.Summary.LowIssues))
	body.WriteString(fmt.Sprintf("- Info Issues: %d\n", alert.Summary.InfoIssues))
	body.WriteString(fmt.Sprintf("- Total Issues: %d\n\n", alert.Summary.TotalIssues))
	
	if len(alert.TopIssues) > 0 {
		body.WriteString("Top Issues:\n")
		for i, issue := range alert.TopIssues {
			body.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, issue.Title, strings.ToUpper(issue.Severity)))
			body.WriteString(fmt.Sprintf("   Description: %s\n", issue.Description))
			if issue.File != "" {
				body.WriteString(fmt.Sprintf("   File: %s:%d\n", issue.File, issue.Line))
			}
			body.WriteString("\n")
		}
	}
	
	if alert.ScanResult.ReportPath != "" {
		body.WriteString(fmt.Sprintf("\nDetailed report available at: %s\n", alert.ScanResult.ReportPath))
	}
	
	body.WriteString("\nPlease review and address these security issues promptly.\n")
	
	return body.String()
}

// getSeverityColor returns color code for Slack attachments based on severity
func (s *SecurityAlerting) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "danger"
	case "high":
		return "warning"
	case "medium":
		return "#fbc02d"
	case "low":
		return "good"
	default:
		return "#1976d2"
	}
}