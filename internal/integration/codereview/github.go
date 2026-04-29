package codereview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"high-performance-news-website/internal/integration/interfaces"
)

// GitHubIntegration implements GitHub code review integration
type GitHubIntegration struct {
	baseURL    string
	token      string
	owner      string
	repo       string
	client     *http.Client
	connected  bool
}

// GitHubComment represents a GitHub PR comment
type GitHubComment struct {
	Body string `json:"body"`
}

// GitHubStatus represents a GitHub commit status
type GitHubStatus struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// GitHubCheckRun represents a GitHub check run
type GitHubCheckRun struct {
	Name       string                    `json:"name"`
	HeadSHA    string                    `json:"head_sha"`
	Status     string                    `json:"status"`
	Conclusion string                    `json:"conclusion,omitempty"`
	Output     GitHubCheckRunOutput      `json:"output,omitempty"`
}

// GitHubCheckRunOutput represents check run output
type GitHubCheckRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text,omitempty"`
}

// NewGitHubIntegration creates a new GitHub integration
func NewGitHubIntegration() *GitHubIntegration {
	return &GitHubIntegration{
		baseURL: "https://api.github.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the integration name
func (g *GitHubIntegration) Name() string {
	return "github"
}

// Type returns the integration type
func (g *GitHubIntegration) Type() interfaces.IntegrationType {
	return interfaces.IntegrationTypeCodeReview
}

// Connect establishes connection to GitHub
func (g *GitHubIntegration) Connect(ctx context.Context, config interfaces.Config) error {
	settings := config.Settings

	token, ok := settings["token"].(string)
	if !ok {
		return fmt.Errorf("token is required")
	}

	owner, ok := settings["owner"].(string)
	if !ok {
		return fmt.Errorf("owner is required")
	}

	repo, ok := settings["repo"].(string)
	if !ok {
		return fmt.Errorf("repo is required")
	}

	if baseURL, ok := settings["base_url"].(string); ok {
		g.baseURL = baseURL
	}

	g.token = token
	g.owner = owner
	g.repo = repo

	// Test connection
	if err := g.testConnection(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	g.connected = true
	return nil
}

// Disconnect closes the GitHub connection
func (g *GitHubIntegration) Disconnect(ctx context.Context) error {
	g.connected = false
	return nil
}

// IsHealthy checks if the GitHub integration is healthy
func (g *GitHubIntegration) IsHealthy(ctx context.Context) bool {
	if !g.connected {
		return false
	}

	return g.testConnection(ctx) == nil
}

// SendEvent sends an event to GitHub
func (g *GitHubIntegration) SendEvent(ctx context.Context, event interfaces.Event) error {
	if !g.connected {
		return fmt.Errorf("not connected to GitHub")
	}

	switch event.Type {
	case interfaces.EventTypeTestFailure:
		return g.handleTestFailure(ctx, event)
	case interfaces.EventTypeTestSuccess:
		return g.handleTestSuccess(ctx, event)
	case interfaces.EventTypeCodeReview:
		return g.handleCodeReview(ctx, event)
	default:
		return nil // Ignore other event types
	}
}

// testConnection tests the GitHub connection
func (g *GitHubIntegration) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/repos/%s/%s", g.baseURL, g.owner, g.repo)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

// handleTestFailure handles test failure events
func (g *GitHubIntegration) handleTestFailure(ctx context.Context, event interfaces.Event) error {
	sha, ok := event.Data["commit_sha"].(string)
	if !ok {
		return fmt.Errorf("commit_sha is required for test failure events")
	}

	// Create a failing check run
	checkRun := GitHubCheckRun{
		Name:       "Comprehensive Testing",
		HeadSHA:    sha,
		Status:     "completed",
		Conclusion: "failure",
		Output: GitHubCheckRunOutput{
			Title:   "Test Failure Detected",
			Summary: g.getTestFailureSummary(event),
			Text:    g.getTestFailureDetails(event),
		},
	}

	return g.createCheckRun(ctx, checkRun)
}

// handleTestSuccess handles test success events
func (g *GitHubIntegration) handleTestSuccess(ctx context.Context, event interfaces.Event) error {
	sha, ok := event.Data["commit_sha"].(string)
	if !ok {
		return fmt.Errorf("commit_sha is required for test success events")
	}

	// Create a successful check run
	checkRun := GitHubCheckRun{
		Name:       "Comprehensive Testing",
		HeadSHA:    sha,
		Status:     "completed",
		Conclusion: "success",
		Output: GitHubCheckRunOutput{
			Title:   "All Tests Passed",
			Summary: g.getTestSuccessSummary(event),
		},
	}

	return g.createCheckRun(ctx, checkRun)
}

// handleCodeReview handles code review events
func (g *GitHubIntegration) handleCodeReview(ctx context.Context, event interfaces.Event) error {
	prNumber, ok := event.Data["pr_number"].(int)
	if !ok {
		// Try to parse as string
		if prStr, ok := event.Data["pr_number"].(string); ok {
			var err error
			prNumber, err = strconv.Atoi(prStr)
			if err != nil {
				return fmt.Errorf("invalid pr_number: %s", prStr)
			}
		} else {
			return fmt.Errorf("pr_number is required for code review events")
		}
	}

	comment := GitHubComment{
		Body: g.getCodeReviewComment(event),
	}

	return g.createPRComment(ctx, prNumber, comment)
}

// createCheckRun creates a GitHub check run
func (g *GitHubIntegration) createCheckRun(ctx context.Context, checkRun GitHubCheckRun) error {
	url := fmt.Sprintf("%s/repos/%s/%s/check-runs", g.baseURL, g.owner, g.repo)
	
	jsonData, err := json.Marshal(checkRun)
	if err != nil {
		return fmt.Errorf("failed to marshal check run: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create check run, status: %d", resp.StatusCode)
	}

	return nil
}

// createPRComment creates a comment on a pull request
func (g *GitHubIntegration) createPRComment(ctx context.Context, prNumber int, comment GitHubComment) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", g.baseURL, g.owner, g.repo, prNumber)
	
	jsonData, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create PR comment, status: %d", resp.StatusCode)
	}

	return nil
}

// getTestFailureSummary creates a summary for test failures
func (g *GitHubIntegration) getTestFailureSummary(event interfaces.Event) string {
	if testName, ok := event.Data["test_name"].(string); ok {
		return fmt.Sprintf("Test '%s' failed", testName)
	}
	return "One or more tests failed"
}

// getTestFailureDetails creates detailed information for test failures
func (g *GitHubIntegration) getTestFailureDetails(event interfaces.Event) string {
	details := fmt.Sprintf("**Event ID:** %s\n**Timestamp:** %s\n**Priority:** %s\n\n",
		event.ID, event.Timestamp.Format(time.RFC3339), event.Priority)

	if errorMsg, ok := event.Data["error"].(string); ok {
		details += fmt.Sprintf("**Error:** %s\n\n", errorMsg)
	}

	if testFile, ok := event.Data["test_file"].(string); ok {
		details += fmt.Sprintf("**Test File:** %s\n", testFile)
	}

	if stackTrace, ok := event.Data["stack_trace"].(string); ok {
		details += fmt.Sprintf("\n**Stack Trace:**\n```\n%s\n```\n", stackTrace)
	}

	return details
}

// getTestSuccessSummary creates a summary for test success
func (g *GitHubIntegration) getTestSuccessSummary(event interfaces.Event) string {
	if testCount, ok := event.Data["test_count"].(int); ok {
		return fmt.Sprintf("All %d tests passed successfully", testCount)
	}
	return "All tests passed successfully"
}

// getCodeReviewComment creates a code review comment
func (g *GitHubIntegration) getCodeReviewComment(event interfaces.Event) string {
	comment := "## 🤖 Automated Code Review\n\n"
	
	if issues, ok := event.Data["issues"].([]interface{}); ok && len(issues) > 0 {
		comment += "### Issues Found:\n"
		for i, issue := range issues {
			if issueStr, ok := issue.(string); ok {
				comment += fmt.Sprintf("%d. %s\n", i+1, issueStr)
			}
		}
		comment += "\n"
	}

	if suggestions, ok := event.Data["suggestions"].([]interface{}); ok && len(suggestions) > 0 {
		comment += "### Suggestions:\n"
		for i, suggestion := range suggestions {
			if suggestionStr, ok := suggestion.(string); ok {
				comment += fmt.Sprintf("%d. %s\n", i+1, suggestionStr)
			}
		}
		comment += "\n"
	}

	if coverage, ok := event.Data["coverage"].(float64); ok {
		comment += fmt.Sprintf("**Test Coverage:** %.1f%%\n", coverage)
	}

	comment += "\n---\n*This comment was generated automatically by the Comprehensive Testing & QA System*"
	
	return comment
}