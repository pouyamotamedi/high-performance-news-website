package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// NotificationManager handles sending notifications for test results
type NotificationManager struct {
	rules []NotificationRule
}

// NotificationRule defines when and how to send notifications
type NotificationRule struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Triggers    []NotificationTrigger `json:"triggers"`
	Channels    []NotificationChannel `json:"channels"`
	Conditions  []NotificationCondition `json:"conditions"`
	Enabled     bool                `json:"enabled"`
	Throttle    time.Duration       `json:"throttle"`
	LastSent    time.Time           `json:"last_sent"`
}

// NotificationTrigger defines what triggers a notification
type NotificationTrigger struct {
	Type      TriggerType `json:"type"`
	Event     string      `json:"event"`
	Severity  string      `json:"severity"`
}

// NotificationCondition defines conditions for sending notifications
type NotificationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(rules ...NotificationRule) *NotificationManager {
	return &NotificationManager{
		rules: rules,
	}
}

// NotifyTestQuarantined sends notification when a test is quarantined
func (n *NotificationManager) NotifyTestQuarantined(testName, testSuite, reason string) {
	log.Printf("Test quarantined: %s.%s - %s", testSuite, testName, reason)
	// Implementation would send actual notifications
}

// NotifyTestReintegrated sends notification when a test is reintegrated
func (n *NotificationManager) NotifyTestReintegrated(testName, testSuite string) {
	log.Printf("Test reintegrated: %s.%s", testSuite, testName)
	// Implementation would send actual notifications
}

// NotifyTestsQuarantined sends notification about multiple tests being quarantined
func (n *NotificationManager) NotifyTestsQuarantined(count int) {
	log.Printf("Multiple tests quarantined: %d tests", count)
	// Implementation would send actual notifications
}

// NotifyHighPriorityRecommendations sends notification about high priority recommendations
func (n *NotificationManager) NotifyHighPriorityRecommendations(count int) {
	log.Printf("High priority recommendations available: %d recommendations", count)
	// Implementation would send actual notifications
}

// SendReportNotifications sends notifications based on report results
func (n *NotificationManager) SendReportNotifications(report *TestReport) {
	log.Printf("Evaluating notification rules for report: %s", report.ID)
	
	for _, rule := range n.rules {
		if !rule.Enabled {
			continue
		}
		
		// Check throttling
		if time.Since(rule.LastSent) < rule.Throttle {
			continue
		}
		
		// Check if rule should trigger
		if n.shouldTriggerRule(rule, report) {
			n.sendNotification(rule, report)
			rule.LastSent = time.Now()
		}
	}
}

// SendPipelineNotifications sends notifications for pipeline results
func (n *NotificationManager) SendPipelineNotifications(result *PipelineResult) {
	log.Printf("Evaluating pipeline notifications for: %s", result.ID)
	
	for _, rule := range n.rules {
		if !rule.Enabled {
			continue
		}
		
		// Check throttling
		if time.Since(rule.LastSent) < rule.Throttle {
			continue
		}
		
		// Check if rule should trigger for pipeline
		if n.shouldTriggerPipelineRule(rule, result) {
			n.sendPipelineNotification(rule, result)
			rule.LastSent = time.Now()
		}
	}
}

// shouldTriggerRule determines if a notification rule should trigger
func (n *NotificationManager) shouldTriggerRule(rule NotificationRule, report *TestReport) bool {
	// Check triggers
	for _, trigger := range rule.Triggers {
		if n.matchesTrigger(trigger, report) {
			// Check conditions
			if n.matchesConditions(rule.Conditions, report) {
				return true
			}
		}
	}
	return false
}

// shouldTriggerPipelineRule determines if a rule should trigger for pipeline
func (n *NotificationManager) shouldTriggerPipelineRule(rule NotificationRule, result *PipelineResult) bool {
	// Check triggers
	for _, trigger := range rule.Triggers {
		if n.matchesPipelineTrigger(trigger, result) {
			// Check conditions
			if n.matchesPipelineConditions(rule.Conditions, result) {
				return true
			}
		}
	}
	return false
}

// matchesTrigger checks if a trigger matches the report
func (n *NotificationManager) matchesTrigger(trigger NotificationTrigger, report *TestReport) bool {
	switch trigger.Event {
	case "coverage_low":
		return report.QualityMetrics.CodeCoverage.Overall < 90.0
	case "security_critical":
		return report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities > 0
	case "performance_regression":
		return report.QualityMetrics.PerformanceMetrics.RegressionPercentage > 15.0
	case "pipeline_failure":
		return report.Summary.SuccessRate < 100.0
	case "quality_degradation":
		// Calculate overall quality score and check if it's declining
		return n.calculateQualityScore(report) < 80.0
	default:
		return false
	}
}

// matchesPipelineTrigger checks if a trigger matches the pipeline result
func (n *NotificationManager) matchesPipelineTrigger(trigger NotificationTrigger, result *PipelineResult) bool {
	switch trigger.Event {
	case "pipeline_failed":
		return result.Status == PipelineStatusFailed
	case "stage_failed":
		for _, stage := range result.Stages {
			if stage.Status == StageStatusFailed {
				return true
			}
		}
		return false
	case "quality_gate_failed":
		return !result.QualityGates.Passed
	default:
		return false
	}
}

// matchesConditions checks if all conditions are met
func (n *NotificationManager) matchesConditions(conditions []NotificationCondition, report *TestReport) bool {
	for _, condition := range conditions {
		if !n.evaluateCondition(condition, report) {
			return false
		}
	}
	return true
}

// matchesPipelineConditions checks if pipeline conditions are met
func (n *NotificationManager) matchesPipelineConditions(conditions []NotificationCondition, result *PipelineResult) bool {
	for _, condition := range conditions {
		if !n.evaluatePipelineCondition(condition, result) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (n *NotificationManager) evaluateCondition(condition NotificationCondition, report *TestReport) bool {
	var fieldValue interface{}
	
	switch condition.Field {
	case "coverage":
		fieldValue = report.QualityMetrics.CodeCoverage.Overall
	case "security_score":
		fieldValue = report.QualityMetrics.SecurityMetrics.SecurityScoreAvg
	case "critical_vulnerabilities":
		fieldValue = report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities
	case "performance_regression":
		fieldValue = report.QualityMetrics.PerformanceMetrics.RegressionPercentage
	case "success_rate":
		fieldValue = report.Summary.SuccessRate
	default:
		return false
	}
	
	return n.compareValues(fieldValue, condition.Operator, condition.Value)
}

// evaluatePipelineCondition evaluates a pipeline condition
func (n *NotificationManager) evaluatePipelineCondition(condition NotificationCondition, result *PipelineResult) bool {
	var fieldValue interface{}
	
	switch condition.Field {
	case "duration":
		fieldValue = result.Duration.Minutes()
	case "failed_stages":
		count := 0
		for _, stage := range result.Stages {
			if stage.Status == StageStatusFailed {
				count++
			}
		}
		fieldValue = count
	case "trigger_type":
		fieldValue = string(result.Trigger.Type)
	default:
		return false
	}
	
	return n.compareValues(fieldValue, condition.Operator, condition.Value)
}

// compareValues compares two values using the specified operator
func (n *NotificationManager) compareValues(fieldValue interface{}, operator string, conditionValue interface{}) bool {
	switch operator {
	case "eq":
		return fieldValue == conditionValue
	case "ne":
		return fieldValue != conditionValue
	case "gt":
		return n.toFloat64(fieldValue) > n.toFloat64(conditionValue)
	case "gte":
		return n.toFloat64(fieldValue) >= n.toFloat64(conditionValue)
	case "lt":
		return n.toFloat64(fieldValue) < n.toFloat64(conditionValue)
	case "lte":
		return n.toFloat64(fieldValue) <= n.toFloat64(conditionValue)
	case "contains":
		return n.contains(fieldValue, conditionValue)
	default:
		return false
	}
}

// toFloat64 converts interface{} to float64
func (n *NotificationManager) toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// contains checks if a value contains another value
func (n *NotificationManager) contains(haystack, needle interface{}) bool {
	haystackStr := fmt.Sprintf("%v", haystack)
	needleStr := fmt.Sprintf("%v", needle)
	return len(haystackStr) > 0 && len(needleStr) > 0 && 
		   haystackStr != needleStr && 
		   len(haystackStr) >= len(needleStr)
}

// sendNotification sends a notification for a report
func (n *NotificationManager) sendNotification(rule NotificationRule, report *TestReport) {
	log.Printf("Sending notification for rule: %s", rule.Name)
	
	message := n.buildReportMessage(rule, report)
	
	for _, channel := range rule.Channels {
		if err := n.sendToChannel(channel, message); err != nil {
			log.Printf("Failed to send notification to %s channel: %v", channel.Type, err)
		}
	}
}

// sendPipelineNotification sends a notification for a pipeline result
func (n *NotificationManager) sendPipelineNotification(rule NotificationRule, result *PipelineResult) {
	log.Printf("Sending pipeline notification for rule: %s", rule.Name)
	
	message := n.buildPipelineMessage(rule, result)
	
	for _, channel := range rule.Channels {
		if err := n.sendToChannel(channel, message); err != nil {
			log.Printf("Failed to send pipeline notification to %s channel: %v", channel.Type, err)
		}
	}
}

// buildReportMessage builds a notification message for a report
func (n *NotificationManager) buildReportMessage(rule NotificationRule, report *TestReport) NotificationMessage {
	return NotificationMessage{
		Title:     fmt.Sprintf("Test Report Alert: %s", rule.Name),
		Body:      n.buildReportBody(report),
		Priority:  n.determinePriority(report),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"report_id":     report.ID,
			"success_rate":  report.Summary.SuccessRate,
			"coverage":      report.QualityMetrics.CodeCoverage.Overall,
			"security_issues": report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities,
		},
	}
}

// buildPipelineMessage builds a notification message for a pipeline result
func (n *NotificationManager) buildPipelineMessage(rule NotificationRule, result *PipelineResult) NotificationMessage {
	return NotificationMessage{
		Title:     fmt.Sprintf("Pipeline Alert: %s", rule.Name),
		Body:      n.buildPipelineBody(result),
		Priority:  n.determinePipelinePriority(result),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"pipeline_id": result.ID,
			"status":      result.Status,
			"duration":    result.Duration.String(),
			"trigger":     result.Trigger.Type,
		},
	}
}

// buildReportBody builds the message body for a report
func (n *NotificationManager) buildReportBody(report *TestReport) string {
	body := fmt.Sprintf("Test Report Summary:\n")
	body += fmt.Sprintf("• Success Rate: %.1f%%\n", report.Summary.SuccessRate)
	body += fmt.Sprintf("• Code Coverage: %.1f%%\n", report.QualityMetrics.CodeCoverage.Overall)
	body += fmt.Sprintf("• Security Issues: %d critical\n", report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities)
	body += fmt.Sprintf("• Performance Regression: %.1f%%\n", report.QualityMetrics.PerformanceMetrics.RegressionPercentage)
	
	if len(report.Recommendations) > 0 {
		body += "\nRecommendations:\n"
		for i, rec := range report.Recommendations {
			if i < 3 { // Limit to top 3 recommendations
				body += fmt.Sprintf("• %s (%s priority)\n", rec.Title, rec.Priority)
			}
		}
	}
	
	return body
}

// buildPipelineBody builds the message body for a pipeline result
func (n *NotificationManager) buildPipelineBody(result *PipelineResult) string {
	body := fmt.Sprintf("Pipeline %s:\n", result.ID)
	body += fmt.Sprintf("• Status: %s\n", result.Status)
	body += fmt.Sprintf("• Duration: %s\n", result.Duration.String())
	body += fmt.Sprintf("• Trigger: %s\n", result.Trigger.Type)
	
	// Add failed stages
	var failedStages []string
	for _, stage := range result.Stages {
		if stage.Status == StageStatusFailed {
			failedStages = append(failedStages, stage.Name)
		}
	}
	
	if len(failedStages) > 0 {
		body += fmt.Sprintf("• Failed Stages: %v\n", failedStages)
	}
	
	if !result.QualityGates.Passed {
		body += fmt.Sprintf("• Quality Gate: FAILED (Score: %.1f)\n", result.QualityGates.Score)
	}
	
	return body
}

// sendToChannel sends a message to a specific notification channel
func (n *NotificationManager) sendToChannel(channel NotificationChannel, message NotificationMessage) error {
	switch channel.Type {
	case "slack":
		return n.sendSlackNotification(channel, message)
	case "email":
		return n.sendEmailNotification(channel, message)
	case "webhook":
		return n.sendWebhookNotification(channel, message)
	case "teams":
		return n.sendTeamsNotification(channel, message)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// sendSlackNotification sends a notification to Slack
func (n *NotificationManager) sendSlackNotification(channel NotificationChannel, message NotificationMessage) error {
	webhookURL := channel.Config["webhook_url"]
	if webhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}
	
	slackMessage := map[string]interface{}{
		"text": message.Title,
		"attachments": []map[string]interface{}{
			{
				"color":     n.getSlackColor(message.Priority),
				"text":      message.Body,
				"timestamp": message.Timestamp.Unix(),
			},
		},
	}
	
	return n.sendHTTPNotification(webhookURL, slackMessage)
}

// sendTeamsNotification sends a notification to Microsoft Teams
func (n *NotificationManager) sendTeamsNotification(channel NotificationChannel, message NotificationMessage) error {
	webhookURL := channel.Config["webhook_url"]
	if webhookURL == "" {
		return fmt.Errorf("teams webhook URL not configured")
	}
	
	teamsMessage := map[string]interface{}{
		"@type":    "MessageCard",
		"@context": "http://schema.org/extensions",
		"summary":  message.Title,
		"themeColor": n.getTeamsColor(message.Priority),
		"sections": []map[string]interface{}{
			{
				"activityTitle": message.Title,
				"text":         message.Body,
			},
		},
	}
	
	return n.sendHTTPNotification(webhookURL, teamsMessage)
}

// sendWebhookNotification sends a generic webhook notification
func (n *NotificationManager) sendWebhookNotification(channel NotificationChannel, message NotificationMessage) error {
	webhookURL := channel.Config["url"]
	if webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}
	
	return n.sendHTTPNotification(webhookURL, message)
}

// sendEmailNotification sends an email notification (placeholder)
func (n *NotificationManager) sendEmailNotification(channel NotificationChannel, message NotificationMessage) error {
	// This would integrate with an email service
	log.Printf("Email notification: %s - %s", message.Title, message.Body)
	return nil
}

// sendHTTPNotification sends an HTTP notification
func (n *NotificationManager) sendHTTPNotification(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send HTTP notification: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP notification failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// Helper functions

// calculateQualityScore calculates an overall quality score
func (n *NotificationManager) calculateQualityScore(report *TestReport) float64 {
	score := 100.0
	
	// Deduct for low coverage
	if report.QualityMetrics.CodeCoverage.Overall < 95 {
		score -= (95 - report.QualityMetrics.CodeCoverage.Overall)
	}
	
	// Deduct for security issues
	score -= float64(report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities) * 10
	
	// Deduct for performance regression
	if report.QualityMetrics.PerformanceMetrics.RegressionPercentage > 5 {
		score -= (report.QualityMetrics.PerformanceMetrics.RegressionPercentage - 5) * 2
	}
	
	// Deduct for failed pipelines
	if report.Summary.SuccessRate < 100 {
		score -= (100 - report.Summary.SuccessRate) * 0.5
	}
	
	return score
}

// determinePriority determines notification priority for a report
func (n *NotificationManager) determinePriority(report *TestReport) string {
	if report.QualityMetrics.SecurityMetrics.CriticalVulnerabilities > 0 {
		return "critical"
	}
	if report.Summary.SuccessRate < 80 {
		return "high"
	}
	if report.QualityMetrics.CodeCoverage.Overall < 80 {
		return "medium"
	}
	return "low"
}

// determinePipelinePriority determines notification priority for a pipeline
func (n *NotificationManager) determinePipelinePriority(result *PipelineResult) string {
	if result.Status == PipelineStatusFailed {
		// Check if it's a production deployment
		if result.Trigger.Type == TriggerDeployment {
			return "critical"
		}
		return "high"
	}
	if !result.QualityGates.Passed {
		return "medium"
	}
	return "low"
}

// getSlackColor returns appropriate Slack color for priority
func (n *NotificationManager) getSlackColor(priority string) string {
	switch priority {
	case "critical":
		return "danger"
	case "high":
		return "warning"
	case "medium":
		return "warning"
	default:
		return "good"
	}
}

// getTeamsColor returns appropriate Teams color for priority
func (n *NotificationManager) getTeamsColor(priority string) string {
	switch priority {
	case "critical":
		return "FF0000"
	case "high":
		return "FF6600"
	case "medium":
		return "FFCC00"
	default:
		return "00CC00"
	}
}

// NotificationMessage represents a notification message
type NotificationMessage struct {
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Priority  string                 `json:"priority"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}