package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"high-performance-news-website/pkg/database"
)

// ConsistencyReporter handles reporting and remediation of consistency issues
type ConsistencyReporter struct {
	db                *database.DB
	remediationQueue  *RemediationQueue
	trendTracker      *TrendTracker
	alertManager      *AlertManager
}

// RemediationSuggestion represents an automated fix suggestion
type RemediationSuggestion struct {
	ID          string                 `json:"id"`
	IssueID     string                 `json:"issue_id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	SQL         string                 `json:"sql,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	Confidence  float64                `json:"confidence"` // 0.0 to 1.0
	CreatedAt   time.Time              `json:"created_at"`
}

// ManualReviewItem represents an issue requiring manual review
type ManualReviewItem struct {
	ID          string                 `json:"id"`
	IssueID     string                 `json:"issue_id"`
	Priority    string                 `json:"priority"`
	AssignedTo  *string                `json:"assigned_to,omitempty"`
	Status      string                 `json:"status"`
	Notes       string                 `json:"notes"`
	Context     map[string]interface{} `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ConsistencyTrend represents trend data for consistency issues
type ConsistencyTrend struct {
	Date        time.Time `json:"date"`
	IssueType   string    `json:"issue_type"`
	Count       int       `json:"count"`
	Severity    string    `json:"severity"`
	Resolved    int       `json:"resolved"`
	NewIssues   int       `json:"new_issues"`
}

// RemediationQueue manages automated and manual remediation tasks
type RemediationQueue struct {
	db *database.DB
}

// TrendTracker tracks consistency issue trends over time
type TrendTracker struct {
	db *database.DB
}

// AlertManager handles alerting for consistency issues
type AlertManager struct {
	db *database.DB
}

// NewConsistencyReporter creates a new consistency reporter
func NewConsistencyReporter() *ConsistencyReporter {
	return &ConsistencyReporter{
		remediationQueue: &RemediationQueue{},
		trendTracker:     &TrendTracker{},
		alertManager:     &AlertManager{},
	}
}

// SetDatabase sets the database connection for all components
func (r *ConsistencyReporter) SetDatabase(db *database.DB) {
	r.db = db
	r.remediationQueue.db = db
	r.trendTracker.db = db
	r.alertManager.db = db
}

// ProcessIssues processes consistency issues and generates remediation suggestions
func (r *ConsistencyReporter) ProcessIssues(ctx context.Context, issues []ConsistencyIssue) error {
	log.Printf("Processing %d consistency issues", len(issues))

	for _, issue := range issues {
		// Store the issue
		if err := r.storeIssue(ctx, issue); err != nil {
			log.Printf("Failed to store issue %s: %v", issue.ID, err)
			continue
		}

		// Generate remediation suggestions
		suggestions := r.generateRemediationSuggestions(issue)
		for _, suggestion := range suggestions {
			if err := r.storeRemediationSuggestion(ctx, suggestion); err != nil {
				log.Printf("Failed to store remediation suggestion %s: %v", suggestion.ID, err)
			}
		}

		// Check if manual review is needed
		if r.requiresManualReview(issue) {
			reviewItem := r.createManualReviewItem(issue)
			if err := r.addToManualReviewQueue(ctx, reviewItem); err != nil {
				log.Printf("Failed to add issue %s to manual review queue: %v", issue.ID, err)
			}
		}
	}

	// Update trend tracking
	if err := r.trendTracker.UpdateTrends(ctx, issues); err != nil {
		log.Printf("Failed to update consistency trends: %v", err)
	}

	// Check for alerting conditions
	if err := r.alertManager.CheckAlertConditions(ctx, issues); err != nil {
		log.Printf("Failed to check alert conditions: %v", err)
	}

	return nil
}

// generateRemediationSuggestions creates automated fix suggestions for issues
func (r *ConsistencyReporter) generateRemediationSuggestions(issue ConsistencyIssue) []RemediationSuggestion {
	var suggestions []RemediationSuggestion

	switch issue.Type {
	case "broken_author_reference":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "reassign_author",
			Description: "Reassign article to a default active author",
			Action:      "UPDATE articles SET author_id = $1 WHERE id = $2",
			SQL:         "UPDATE articles SET author_id = (SELECT id FROM users WHERE is_active = true AND role IN ('admin', 'editor') ORDER BY id LIMIT 1) WHERE id = $1",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.8,
			CreatedAt:  time.Now(),
		})

	case "broken_category_reference":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "reassign_category",
			Description: "Reassign article to default 'Uncategorized' category",
			Action:      "UPDATE articles SET category_id = $1 WHERE id = $2",
			SQL:         "UPDATE articles SET category_id = (SELECT id FROM categories WHERE slug = 'uncategorized' OR name ILIKE '%uncategorized%' ORDER BY id LIMIT 1) WHERE id = $1",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.9,
			CreatedAt:  time.Now(),
		})

	case "orphaned_article_tag":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "remove_orphaned_tag",
			Description: "Remove orphaned tag reference from article",
			Action:      "DELETE FROM article_tags WHERE article_id = $1 AND tag_id = $2",
			SQL:         "DELETE FROM article_tags WHERE article_id = $1 AND tag_id = $2",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
				"tag_id":     issue.TagID,
			},
			Confidence: 0.95,
			CreatedAt:  time.Now(),
		})

	case "missing_meta_title":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "generate_meta_title",
			Description: "Generate meta title from article title (truncated to 60 chars)",
			Action:      "UPDATE articles SET meta_title = $1 WHERE id = $2",
			SQL:         "UPDATE articles SET meta_title = LEFT(title, 60) WHERE id = $1 AND (meta_title IS NULL OR meta_title = '')",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.7,
			CreatedAt:  time.Now(),
		})

	case "missing_meta_description":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "generate_meta_description",
			Description: "Generate meta description from article excerpt (truncated to 160 chars)",
			Action:      "UPDATE articles SET meta_description = $1 WHERE id = $2",
			SQL:         "UPDATE articles SET meta_description = LEFT(COALESCE(excerpt, LEFT(content, 200)), 160) WHERE id = $1 AND (meta_description IS NULL OR meta_description = '')",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.6,
			CreatedAt:  time.Now(),
		})

	case "invalid_schema_type":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "fix_schema_type",
			Description: "Set schema type to default 'NewsArticle'",
			Action:      "UPDATE articles SET schema_type = 'NewsArticle' WHERE id = $1",
			SQL:         "UPDATE articles SET schema_type = 'NewsArticle' WHERE id = $1",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.9,
			CreatedAt:  time.Now(),
		})

	case "meta_title_too_long":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "truncate_meta_title",
			Description: "Truncate meta title to 60 characters",
			Action:      "UPDATE articles SET meta_title = LEFT(meta_title, 60) WHERE id = $1",
			SQL:         "UPDATE articles SET meta_title = LEFT(meta_title, 60) WHERE id = $1",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.8,
			CreatedAt:  time.Now(),
		})

	case "meta_description_too_long":
		suggestions = append(suggestions, RemediationSuggestion{
			ID:          generateSuggestionID(),
			IssueID:     issue.ID,
			Type:        "truncate_meta_description",
			Description: "Truncate meta description to 160 characters",
			Action:      "UPDATE articles SET meta_description = LEFT(meta_description, 160) WHERE id = $1",
			SQL:         "UPDATE articles SET meta_description = LEFT(meta_description, 160) WHERE id = $1",
			Parameters: map[string]interface{}{
				"article_id": issue.ArticleID,
			},
			Confidence: 0.8,
			CreatedAt:  time.Now(),
		})
	}

	return suggestions
}

// requiresManualReview determines if an issue needs manual review
func (r *ConsistencyReporter) requiresManualReview(issue ConsistencyIssue) bool {
	// High severity issues always require manual review
	if issue.Severity == "high" {
		return true
	}

	// Complex issues that need human judgment
	complexIssueTypes := map[string]bool{
		"translation_status_inconsistency":           true,
		"duplicate_language_in_translation_group":    true,
		"broken_translation_group_reference":        true,
		"invalid_canonical_url":                      true,
	}

	return complexIssueTypes[issue.Type]
}

// createManualReviewItem creates a manual review item for complex issues
func (r *ConsistencyReporter) createManualReviewItem(issue ConsistencyIssue) ManualReviewItem {
	priority := "medium"
	if issue.Severity == "high" {
		priority = "high"
	} else if issue.Severity == "low" {
		priority = "low"
	}

	return ManualReviewItem{
		ID:        generateReviewItemID(),
		IssueID:   issue.ID,
		Priority:  priority,
		Status:    "pending",
		Context:   issue.Details,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Database operations

func (r *ConsistencyReporter) storeIssue(ctx context.Context, issue ConsistencyIssue) error {
	query := `
		INSERT INTO consistency_issues (
			id, type, description, severity, article_id, category_id, tag_id, user_id,
			details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			description = EXCLUDED.description,
			severity = EXCLUDED.severity,
			details = EXCLUDED.details
	`

	detailsJSON, _ := json.Marshal(issue.Details)

	_, err := r.db.ExecContext(ctx, query,
		issue.ID, issue.Type, issue.Description, issue.Severity,
		issue.ArticleID, issue.CategoryID, issue.TagID, issue.UserID,
		detailsJSON, issue.CreatedAt,
	)

	return err
}

func (r *ConsistencyReporter) storeRemediationSuggestion(ctx context.Context, suggestion RemediationSuggestion) error {
	query := `
		INSERT INTO remediation_suggestions (
			id, issue_id, type, description, action, sql_query, parameters, confidence, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`

	parametersJSON, _ := json.Marshal(suggestion.Parameters)

	_, err := r.db.ExecContext(ctx, query,
		suggestion.ID, suggestion.IssueID, suggestion.Type, suggestion.Description,
		suggestion.Action, suggestion.SQL, parametersJSON, suggestion.Confidence,
		suggestion.CreatedAt,
	)

	return err
}

func (r *ConsistencyReporter) addToManualReviewQueue(ctx context.Context, item ManualReviewItem) error {
	query := `
		INSERT INTO manual_review_queue (
			id, issue_id, priority, assigned_to, status, notes, context, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			priority = EXCLUDED.priority,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	contextJSON, _ := json.Marshal(item.Context)

	_, err := r.db.ExecContext(ctx, query,
		item.ID, item.IssueID, item.Priority, item.AssignedTo, item.Status,
		item.Notes, contextJSON, item.CreatedAt, item.UpdatedAt,
	)

	return err
}

// GetRemediationSuggestions retrieves remediation suggestions for an issue
func (r *ConsistencyReporter) GetRemediationSuggestions(ctx context.Context, issueID string) ([]RemediationSuggestion, error) {
	query := `
		SELECT id, issue_id, type, description, action, sql_query, parameters, confidence, created_at
		FROM remediation_suggestions
		WHERE issue_id = $1
		ORDER BY confidence DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []RemediationSuggestion
	for rows.Next() {
		var suggestion RemediationSuggestion
		var parametersJSON []byte
		var sqlQuery sql.NullString

		err := rows.Scan(
			&suggestion.ID, &suggestion.IssueID, &suggestion.Type, &suggestion.Description,
			&suggestion.Action, &sqlQuery, &parametersJSON, &suggestion.Confidence,
			&suggestion.CreatedAt,
		)
		if err != nil {
			continue
		}

		suggestion.SQL = sqlQuery.String
		json.Unmarshal(parametersJSON, &suggestion.Parameters)
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// ExecuteRemediation executes an automated remediation suggestion
func (r *ConsistencyReporter) ExecuteRemediation(ctx context.Context, suggestionID string) error {
	// Get the suggestion
	query := `
		SELECT id, issue_id, type, sql_query, parameters, confidence
		FROM remediation_suggestions
		WHERE id = $1
	`

	var suggestion RemediationSuggestion
	var parametersJSON []byte
	var sqlQuery sql.NullString

	err := r.db.QueryRowContext(ctx, query, suggestionID).Scan(
		&suggestion.ID, &suggestion.IssueID, &suggestion.Type,
		&sqlQuery, &parametersJSON, &suggestion.Confidence,
	)
	if err != nil {
		return fmt.Errorf("suggestion not found: %w", err)
	}

	suggestion.SQL = sqlQuery.String
	json.Unmarshal(parametersJSON, &suggestion.Parameters)

	// Only execute high-confidence suggestions automatically
	if suggestion.Confidence < 0.8 {
		return fmt.Errorf("suggestion confidence too low for automatic execution: %f", suggestion.Confidence)
	}

	// Execute the remediation SQL
	if suggestion.SQL != "" {
		// Extract parameters for SQL execution
		var args []interface{}
		if articleID, ok := suggestion.Parameters["article_id"]; ok {
			args = append(args, articleID)
		}
		if tagID, ok := suggestion.Parameters["tag_id"]; ok {
			args = append(args, tagID)
		}

		_, err = r.db.ExecContext(ctx, suggestion.SQL, args...)
		if err != nil {
			return fmt.Errorf("failed to execute remediation: %w", err)
		}

		// Mark the issue as resolved
		_, err = r.db.ExecContext(ctx,
			"UPDATE consistency_issues SET status = 'resolved', resolved_at = NOW() WHERE id = $1",
			suggestion.IssueID,
		)
		if err != nil {
			log.Printf("Failed to mark issue as resolved: %v", err)
		}

		log.Printf("Successfully executed remediation %s for issue %s", suggestionID, suggestion.IssueID)
	}

	return nil
}

// TrendTracker methods

func (t *TrendTracker) UpdateTrends(ctx context.Context, issues []ConsistencyIssue) error {
	if len(issues) == 0 {
		return nil
	}

	// Group issues by type and severity
	trendData := make(map[string]map[string]int)
	for _, issue := range issues {
		if trendData[issue.Type] == nil {
			trendData[issue.Type] = make(map[string]int)
		}
		trendData[issue.Type][issue.Severity]++
	}

	// Store trend data
	today := time.Now().Truncate(24 * time.Hour)
	for issueType, severityMap := range trendData {
		for severity, count := range severityMap {
			query := `
				INSERT INTO consistency_trends (date, issue_type, severity, count, new_issues)
				VALUES ($1, $2, $3, $4, $4)
				ON CONFLICT (date, issue_type, severity) DO UPDATE SET
					count = consistency_trends.count + EXCLUDED.count,
					new_issues = consistency_trends.new_issues + EXCLUDED.new_issues
			`

			_, err := t.db.ExecContext(ctx, query, today, issueType, severity, count)
			if err != nil {
				log.Printf("Failed to update trend for %s/%s: %v", issueType, severity, err)
			}
		}
	}

	return nil
}

// AlertManager methods

func (a *AlertManager) CheckAlertConditions(ctx context.Context, issues []ConsistencyIssue) error {
	// Count high severity issues
	highSeverityCount := 0
	for _, issue := range issues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}

	// Alert if too many high severity issues
	if highSeverityCount > 10 {
		return a.sendAlert(ctx, "high_severity_threshold", fmt.Sprintf("Found %d high severity consistency issues", highSeverityCount))
	}

	// Check for specific issue type spikes
	issueTypeCounts := make(map[string]int)
	for _, issue := range issues {
		issueTypeCounts[issue.Type]++
	}

	for issueType, count := range issueTypeCounts {
		if count > 50 { // Threshold for issue type spike
			return a.sendAlert(ctx, "issue_type_spike", fmt.Sprintf("Found %d issues of type %s", count, issueType))
		}
	}

	return nil
}

func (a *AlertManager) sendAlert(ctx context.Context, alertType, message string) error {
	query := `
		INSERT INTO consistency_alerts (type, message, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := a.db.ExecContext(ctx, query, alertType, message, time.Now())
	if err != nil {
		return err
	}

	log.Printf("CONSISTENCY ALERT [%s]: %s", strings.ToUpper(alertType), message)
	return nil
}

// Utility functions

func generateSuggestionID() string {
	return fmt.Sprintf("suggestion_%d", time.Now().UnixNano())
}

func generateReviewItemID() string {
	return fmt.Sprintf("review_%d", time.Now().UnixNano())
}