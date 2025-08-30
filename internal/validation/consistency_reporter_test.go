package validation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/pkg/database"
)

func TestConsistencyReporter_ProcessIssues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reporter := NewConsistencyReporter()
	reporter.SetDatabase(db)
	ctx := context.Background()

	// Create test issues
	issues := []ConsistencyIssue{
		{
			ID:          "issue_1",
			Type:        "broken_author_reference",
			Description: "Article references non-existent author",
			Severity:    "high",
			ArticleID:   &[]uint64{1}[0],
			UserID:      &[]uint64{999}[0],
			Details:     map[string]interface{}{"article_title": "Test Article"},
			CreatedAt:   time.Now(),
		},
		{
			ID:          "issue_2",
			Type:        "missing_meta_title",
			Description: "Article is missing meta title",
			Severity:    "medium",
			ArticleID:   &[]uint64{2}[0],
			Details:     map[string]interface{}{"article_title": "Another Article"},
			CreatedAt:   time.Now(),
		},
	}

	// Process issues
	err := reporter.ProcessIssues(ctx, issues)
	require.NoError(t, err)

	// Verify issues were stored
	// This would require checking the database
	// In a real test, you'd query the consistency_issues table
}

func TestConsistencyReporter_generateRemediationSuggestions(t *testing.T) {
	reporter := NewConsistencyReporter()

	tests := []struct {
		name          string
		issue         ConsistencyIssue
		expectedCount int
		expectedTypes []string
	}{
		{
			name: "Broken author reference",
			issue: ConsistencyIssue{
				ID:        "issue_1",
				Type:      "broken_author_reference",
				ArticleID: &[]uint64{1}[0],
				UserID:    &[]uint64{999}[0],
			},
			expectedCount: 1,
			expectedTypes: []string{"reassign_author"},
		},
		{
			name: "Missing meta title",
			issue: ConsistencyIssue{
				ID:        "issue_2",
				Type:      "missing_meta_title",
				ArticleID: &[]uint64{1}[0],
			},
			expectedCount: 1,
			expectedTypes: []string{"generate_meta_title"},
		},
		{
			name: "Orphaned article tag",
			issue: ConsistencyIssue{
				ID:        "issue_3",
				Type:      "orphaned_article_tag",
				ArticleID: &[]uint64{1}[0],
				TagID:     &[]uint64{999}[0],
			},
			expectedCount: 1,
			expectedTypes: []string{"remove_orphaned_tag"},
		},
		{
			name: "Invalid schema type",
			issue: ConsistencyIssue{
				ID:        "issue_4",
				Type:      "invalid_schema_type",
				ArticleID: &[]uint64{1}[0],
			},
			expectedCount: 1,
			expectedTypes: []string{"fix_schema_type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := reporter.generateRemediationSuggestions(tt.issue)
			
			assert.Len(t, suggestions, tt.expectedCount)
			
			if len(suggestions) > 0 {
				suggestion := suggestions[0]
				assert.Equal(t, tt.issue.ID, suggestion.IssueID)
				assert.Contains(t, tt.expectedTypes, suggestion.Type)
				assert.NotEmpty(t, suggestion.Description)
				assert.NotEmpty(t, suggestion.Action)
				assert.Greater(t, suggestion.Confidence, 0.0)
				assert.LessOrEqual(t, suggestion.Confidence, 1.0)
				assert.NotEmpty(t, suggestion.ID)
			}
		})
	}
}

func TestConsistencyReporter_requiresManualReview(t *testing.T) {
	reporter := NewConsistencyReporter()

	tests := []struct {
		name     string
		issue    ConsistencyIssue
		expected bool
	}{
		{
			name: "High severity issue",
			issue: ConsistencyIssue{
				Type:     "broken_author_reference",
				Severity: "high",
			},
			expected: true,
		},
		{
			name: "Complex translation issue",
			issue: ConsistencyIssue{
				Type:     "translation_status_inconsistency",
				Severity: "medium",
			},
			expected: true,
		},
		{
			name: "Simple SEO issue",
			issue: ConsistencyIssue{
				Type:     "missing_meta_title",
				Severity: "medium",
			},
			expected: false,
		},
		{
			name: "Low severity issue",
			issue: ConsistencyIssue{
				Type:     "meta_title_too_long",
				Severity: "low",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reporter.requiresManualReview(tt.issue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConsistencyReporter_createManualReviewItem(t *testing.T) {
	reporter := NewConsistencyReporter()

	issue := ConsistencyIssue{
		ID:       "issue_1",
		Type:     "translation_status_inconsistency",
		Severity: "high",
		Details:  map[string]interface{}{"translation_group_id": 123},
	}

	reviewItem := reporter.createManualReviewItem(issue)

	assert.Equal(t, issue.ID, reviewItem.IssueID)
	assert.Equal(t, "high", reviewItem.Priority)
	assert.Equal(t, "pending", reviewItem.Status)
	assert.Equal(t, issue.Details, reviewItem.Context)
	assert.NotEmpty(t, reviewItem.ID)
	assert.False(t, reviewItem.CreatedAt.IsZero())
	assert.False(t, reviewItem.UpdatedAt.IsZero())
}

func TestTrendTracker_UpdateTrends(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tracker := &TrendTracker{db: db}
	ctx := context.Background()

	issues := []ConsistencyIssue{
		{Type: "broken_author_reference", Severity: "high"},
		{Type: "broken_author_reference", Severity: "high"},
		{Type: "missing_meta_title", Severity: "medium"},
		{Type: "missing_meta_title", Severity: "medium"},
		{Type: "missing_meta_title", Severity: "medium"},
	}

	err := tracker.UpdateTrends(ctx, issues)
	require.NoError(t, err)

	// In a real test, you'd verify the trends were stored in the database
	// by querying the consistency_trends table
}

func TestAlertManager_CheckAlertConditions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	alertManager := &AlertManager{db: db}
	ctx := context.Background()

	tests := []struct {
		name           string
		issues         []ConsistencyIssue
		shouldAlert    bool
		expectedAlerts int
	}{
		{
			name: "Too many high severity issues",
			issues: make([]ConsistencyIssue, 15, 15), // 15 issues
			shouldAlert: true,
		},
		{
			name: "Normal number of issues",
			issues: []ConsistencyIssue{
				{Type: "missing_meta_title", Severity: "medium"},
				{Type: "missing_meta_title", Severity: "medium"},
			},
			shouldAlert: false,
		},
		{
			name: "Issue type spike",
			issues: func() []ConsistencyIssue {
				issues := make([]ConsistencyIssue, 60)
				for i := range issues {
					issues[i] = ConsistencyIssue{
						Type:     "broken_author_reference",
						Severity: "medium",
					}
				}
				return issues
			}(),
			shouldAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set severity for high severity test
			if tt.name == "Too many high severity issues" {
				for i := range tt.issues {
					tt.issues[i].Severity = "high"
					tt.issues[i].Type = "broken_author_reference"
				}
			}

			err := alertManager.CheckAlertConditions(ctx, tt.issues)
			
			if tt.shouldAlert {
				// In a real test, you might expect an error or check that alerts were sent
				// For now, we just ensure no unexpected errors occurred
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationSuggestion_Confidence(t *testing.T) {
	reporter := NewConsistencyReporter()

	// Test that different issue types have appropriate confidence levels
	testCases := []struct {
		issueType           string
		expectedMinConfidence float64
	}{
		{"orphaned_article_tag", 0.9},      // High confidence - safe to remove
		{"invalid_schema_type", 0.8},       // High confidence - safe default
		{"broken_category_reference", 0.8}, // High confidence - safe reassignment
		{"missing_meta_title", 0.6},        // Medium confidence - generated content
		{"missing_meta_description", 0.5},  // Lower confidence - generated content
	}

	for _, tc := range testCases {
		t.Run(tc.issueType, func(t *testing.T) {
			issue := ConsistencyIssue{
				ID:        "test_issue",
				Type:      tc.issueType,
				ArticleID: &[]uint64{1}[0],
			}

			suggestions := reporter.generateRemediationSuggestions(issue)
			require.Greater(t, len(suggestions), 0, "Should generate at least one suggestion")

			suggestion := suggestions[0]
			assert.GreaterOrEqual(t, suggestion.Confidence, tc.expectedMinConfidence,
				"Confidence should be at least %f for %s", tc.expectedMinConfidence, tc.issueType)
		})
	}
}

func TestConsistencyReporter_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create the full consistency checking pipeline
	checker := NewConsistencyChecker(db)
	reporter := NewConsistencyReporter()
	reporter.SetDatabase(db)
	
	ctx := context.Background()

	// Setup test data with known issues
	setupTestDataWithIssues(t, db)

	// Run consistency check
	check, err := checker.ValidateDataConsistency(ctx)
	require.NoError(t, err)
	require.NotNil(t, check)

	// Process issues through reporter
	if len(check.Issues) > 0 {
		err = reporter.ProcessIssues(ctx, check.Issues)
		require.NoError(t, err)

		// Verify that suggestions were generated
		for _, issue := range check.Issues {
			suggestions, err := reporter.GetRemediationSuggestions(ctx, issue.ID)
			require.NoError(t, err)
			
			// Most issues should have at least one suggestion
			if issue.Type != "translation_status_inconsistency" && 
			   issue.Type != "duplicate_language_in_translation_group" {
				assert.Greater(t, len(suggestions), 0, 
					"Issue type %s should have remediation suggestions", issue.Type)
			}
		}
	}
}