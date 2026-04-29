package validation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/pkg/database"
	"high-performance-news-website/internal/models"
)

func TestConsistencyChecker_ValidateDataConsistency(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create test data with known issues
	setupTestDataWithIssues(t, db)

	// Run consistency check
	check, err := checker.ValidateDataConsistency(ctx)
	require.NoError(t, err)
	require.NotNil(t, check)

	// Verify check structure
	assert.Equal(t, "Sample-Based Data Consistency Check", check.Name)
	assert.Equal(t, CheckTypeSample, check.Type)
	assert.Equal(t, 1000, check.SampleSize)
	assert.True(t, check.Duration > 0)
	assert.NotEmpty(t, check.ID)

	// Should find some issues in our test data
	assert.Greater(t, len(check.Issues), 0)

	// Verify metadata
	assert.Contains(t, check.Metadata, "actual_sample_size")
	assert.Contains(t, check.Metadata, "referential_issues")
	assert.Contains(t, check.Metadata, "multilingual_issues")
	assert.Contains(t, check.Metadata, "seo_issues")
}

func TestConsistencyChecker_getSampleArticles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create test articles
	createTestArticles(t, db, 50)

	// Test sampling
	articles, err := checker.getSampleArticles(ctx, 10)
	require.NoError(t, err)

	// Should return articles (may be less than requested due to sampling)
	assert.LessOrEqual(t, len(articles), 10)

	// Verify article structure
	if len(articles) > 0 {
		article := articles[0]
		assert.NotZero(t, article.ID)
		assert.NotEmpty(t, article.Title)
		assert.NotEmpty(t, article.Slug)
		assert.NotZero(t, article.AuthorID)
		assert.NotZero(t, article.CategoryID)
		assert.Equal(t, "published", article.Status)
	}
}

func TestConsistencyChecker_validateReferentialIntegrity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create test data with referential integrity issues
	articles := []SampleArticle{
		{
			ID:         1,
			Title:      "Test Article 1",
			Slug:       "test-article-1",
			AuthorID:   999, // Non-existent author
			CategoryID: 1,   // Valid category
			Status:     "published",
		},
		{
			ID:         2,
			Title:      "Test Article 2",
			Slug:       "test-article-2",
			AuthorID:   1,   // Valid author
			CategoryID: 999, // Non-existent category
			Status:     "published",
		},
	}

	// Create valid author and category
	createTestUser(t, db, 1, "testuser", "test@example.com")
	createTestCategory(t, db, 1, "Test Category", "test-category")

	issues := checker.validateReferentialIntegrity(ctx, articles)

	// Should find 2 issues: broken author reference and broken category reference
	assert.Len(t, issues, 2)

	// Check for broken author reference
	foundAuthorIssue := false
	foundCategoryIssue := false
	for _, issue := range issues {
		if issue.Type == "broken_author_reference" {
			foundAuthorIssue = true
			assert.Equal(t, "high", issue.Severity)
			assert.Equal(t, uint64(1), *issue.ArticleID)
			assert.Equal(t, uint64(999), *issue.UserID)
		}
		if issue.Type == "broken_category_reference" {
			foundCategoryIssue = true
			assert.Equal(t, "high", issue.Severity)
			assert.Equal(t, uint64(2), *issue.ArticleID)
			assert.Equal(t, uint64(999), *issue.CategoryID)
		}
	}

	assert.True(t, foundAuthorIssue, "Should find broken author reference")
	assert.True(t, foundCategoryIssue, "Should find broken category reference")
}

func TestConsistencyChecker_validateMultilingualConsistency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create translation group
	groupID := createTestTranslationGroup(t, db, "article")

	// Create articles with multilingual consistency issues
	articles := []SampleArticle{
		{
			ID:                 1,
			Title:              "English Article",
			LanguageCode:       "en",
			Status:             "published",
			TranslationGroupID: &groupID,
		},
		{
			ID:                 2,
			Title:              "Persian Article",
			LanguageCode:       "fa",
			Status:             "draft", // Inconsistent status
			TranslationGroupID: &groupID,
		},
		{
			ID:                 3,
			Title:              "Another English Article",
			LanguageCode:       "en", // Duplicate language in group
			Status:             "published",
			TranslationGroupID: &groupID,
		},
	}

	issues := checker.validateMultilingualConsistency(ctx, articles)

	// Should find issues for status inconsistency and duplicate language
	assert.Greater(t, len(issues), 0)

	foundStatusIssue := false
	foundDuplicateLanguage := false
	for _, issue := range issues {
		if issue.Type == "translation_status_inconsistency" {
			foundStatusIssue = true
			assert.Equal(t, "medium", issue.Severity)
		}
		if issue.Type == "duplicate_language_in_translation_group" {
			foundDuplicateLanguage = true
			assert.Equal(t, "high", issue.Severity)
		}
	}

	assert.True(t, foundStatusIssue, "Should find translation status inconsistency")
	assert.True(t, foundDuplicateLanguage, "Should find duplicate language in translation group")
}

func TestConsistencyChecker_validateSEOMetadataConsistency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create articles with SEO metadata issues
	articles := []SampleArticle{
		{
			ID:              1,
			Title:           "Article with Missing Meta Title",
			MetaTitle:       "", // Missing meta title
			MetaDescription: "Valid description",
			SchemaType:      "NewsArticle",
		},
		{
			ID:              2,
			Title:           "Article with Long Meta Title",
			MetaTitle:       "This is a very long meta title that exceeds the recommended 60 character limit for SEO optimization", // Too long
			MetaDescription: "",                                                                                                    // Missing meta description
			SchemaType:      "NewsArticle",
		},
		{
			ID:              3,
			Title:           "Article with Invalid Schema",
			MetaTitle:       "Valid Title",
			MetaDescription: "Valid description",
			SchemaType:      "InvalidSchema", // Invalid schema type
			CanonicalURL:    "not-a-valid-url", // Invalid URL
		},
	}

	issues := checker.validateSEOMetadataConsistency(ctx, articles)

	// Should find multiple SEO issues
	assert.Greater(t, len(issues), 0)

	issueTypes := make(map[string]bool)
	for _, issue := range issues {
		issueTypes[issue.Type] = true
	}

	assert.True(t, issueTypes["missing_meta_title"], "Should find missing meta title")
	assert.True(t, issueTypes["missing_meta_description"], "Should find missing meta description")
	assert.True(t, issueTypes["meta_title_too_long"], "Should find meta title too long")
	assert.True(t, issueTypes["invalid_schema_type"], "Should find invalid schema type")
	assert.True(t, issueTypes["invalid_canonical_url"], "Should find invalid canonical URL")
}

func TestConsistencyChecker_determineCheckStatus(t *testing.T) {
	checker := NewConsistencyChecker(nil)

	tests := []struct {
		name     string
		issues   []ConsistencyIssue
		expected CheckStatus
	}{
		{
			name:     "No issues",
			issues:   []ConsistencyIssue{},
			expected: CheckStatusPassed,
		},
		{
			name: "Only low severity issues",
			issues: []ConsistencyIssue{
				{Severity: "low"},
				{Severity: "low"},
			},
			expected: CheckStatusWarning,
		},
		{
			name: "Medium severity issues",
			issues: []ConsistencyIssue{
				{Severity: "medium"},
				{Severity: "low"},
			},
			expected: CheckStatusWarning,
		},
		{
			name: "High severity issues",
			issues: []ConsistencyIssue{
				{Severity: "high"},
				{Severity: "medium"},
				{Severity: "low"},
			},
			expected: CheckStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := checker.determineCheckStatus(tt.issues)
			assert.Equal(t, tt.expected, status)
		})
	}
}

// Helper functions for testing

func setupTestDB(t *testing.T) *database.DB {
	// This would typically connect to a test database
	// For this example, we'll mock the database connection
	// In a real implementation, you'd use a test database
	db, err := database.NewTestDB()
	require.NoError(t, err)
	return db
}

func setupTestDataWithIssues(t *testing.T, db *database.DB) {
	// Create test users, categories, and articles with known consistency issues
	createTestUser(t, db, 1, "validuser", "valid@example.com")
	createTestCategory(t, db, 1, "Valid Category", "valid-category")
	
	// Create articles with various issues
	createTestArticleWithIssues(t, db)
}

func createTestArticles(t *testing.T, db *database.DB, count int) {
	// Create valid test data
	createTestUser(t, db, 1, "testuser", "test@example.com")
	createTestCategory(t, db, 1, "Test Category", "test-category")
	
	for i := 1; i <= count; i++ {
		query := `
			INSERT INTO articles (id, title, slug, content, author_id, category_id, status, published_at, language_code)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err := db.Exec(query, i, 
			fmt.Sprintf("Test Article %d", i),
			fmt.Sprintf("test-article-%d", i),
			"Test content",
			1, 1, "published", time.Now(), "fa")
		require.NoError(t, err)
	}
}

func createTestUser(t *testing.T, db *database.DB, id uint64, username, email string) {
	query := `
		INSERT INTO users (id, username, email, password_hash, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Exec(query, id, username, email, "hash", true)
	require.NoError(t, err)
}

func createTestCategory(t *testing.T, db *database.DB, id uint64, name, slug string) {
	query := `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Exec(query, id, name, slug)
	require.NoError(t, err)
}

func createTestTranslationGroup(t *testing.T, db *database.DB, groupType string) uint64 {
	query := `
		INSERT INTO translation_groups (group_type)
		VALUES ($1)
		RETURNING id
	`
	var id uint64
	err := db.QueryRow(query, groupType).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestArticleWithIssues(t *testing.T, db *database.DB) {
	// Create article with missing author (referential integrity issue)
	query := `
		INSERT INTO articles (id, title, slug, content, author_id, category_id, status, published_at, language_code, meta_title)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.Exec(query, 999, 
		"Article with Issues",
		"article-with-issues",
		"Content",
		999, // Non-existent author
		1,   // Valid category
		"published", 
		time.Now(), 
		"fa",
		"") // Missing meta title
	require.NoError(t, err)
}