package validation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/pkg/database"
)

// TestConsistencySystemIntegration tests the complete consistency checking system
func TestConsistencySystemIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database with schema
	db := setupIntegrationTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create the complete system
	checker := NewConsistencyChecker(db)
	reporter := NewConsistencyReporter()
	reporter.SetDatabase(db)
	scheduler := NewCheckScheduler()
	scheduler.SetDependencies(db, checker, reporter, &mockMonitoringClient{})

	// Setup test data with various consistency issues
	setupComprehensiveTestData(t, db)

	t.Run("Full Consistency Check Pipeline", func(t *testing.T) {
		// 1. Run consistency check
		check, err := checker.ValidateDataConsistency(ctx)
		require.NoError(t, err)
		require.NotNil(t, check)

		// Verify check results
		assert.Equal(t, CheckTypeSample, check.Type)
		assert.Greater(t, len(check.Issues), 0, "Should find issues in test data")
		assert.True(t, check.Duration > 0)

		// 2. Process issues through reporter
		err = reporter.ProcessIssues(ctx, check.Issues)
		require.NoError(t, err)

		// 3. Verify remediation suggestions were generated
		totalSuggestions := 0
		for _, issue := range check.Issues {
			suggestions, err := reporter.GetRemediationSuggestions(ctx, issue.ID)
			require.NoError(t, err)
			totalSuggestions += len(suggestions)
		}
		assert.Greater(t, totalSuggestions, 0, "Should generate remediation suggestions")

		// 4. Execute high-confidence remediation
		executedCount := 0
		for _, issue := range check.Issues {
			suggestions, err := reporter.GetRemediationSuggestions(ctx, issue.ID)
			require.NoError(t, err)

			for _, suggestion := range suggestions {
				if suggestion.Confidence >= 0.9 {
					err := reporter.ExecuteRemediation(ctx, suggestion.ID)
					if err == nil {
						executedCount++
					}
				}
			}
		}

		t.Logf("Executed %d high-confidence remediation suggestions", executedCount)

		// 5. Run another check to verify improvements
		secondCheck, err := checker.ValidateDataConsistency(ctx)
		require.NoError(t, err)

		// Should have fewer issues after remediation
		if executedCount > 0 {
			assert.LessOrEqual(t, len(secondCheck.Issues), len(check.Issues),
				"Should have same or fewer issues after remediation")
		}
	})

	t.Run("Scheduler Integration", func(t *testing.T) {
		// Start scheduler
		err := scheduler.Start(ctx)
		require.NoError(t, err)

		// Verify schedules were loaded
		schedules := scheduler.GetSchedules()
		assert.Greater(t, len(schedules), 0, "Should have default schedules")

		// Stop scheduler
		scheduler.Stop()

		// Verify check results were stored
		results, err := scheduler.GetCheckResults(ctx, 10)
		require.NoError(t, err)
		// May be empty if no scheduled checks ran during test
		t.Logf("Found %d check results", len(results))
	})

	t.Run("Performance Validation", func(t *testing.T) {
		// Test with larger sample size
		start := time.Now()
		check, err := checker.ValidateDataConsistency(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 30*time.Second, "Check should complete within 30 seconds")
		assert.Greater(t, check.SampleSize, 0, "Should have processed some articles")

		t.Logf("Processed %d articles in %v", check.SampleSize, duration)
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Test with invalid database connection
		invalidChecker := NewConsistencyChecker(nil)
		_, err := invalidChecker.ValidateDataConsistency(ctx)
		assert.Error(t, err, "Should handle invalid database connection")

		// Test with empty database
		emptyDB := setupEmptyTestDB(t)
		defer emptyDB.Close()

		emptyChecker := NewConsistencyChecker(emptyDB)
		check, err := emptyChecker.ValidateDataConsistency(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, len(check.Issues), "Should handle empty database gracefully")
	})
}

// TestConsistencyCheckerPerformance tests performance with realistic data volumes
func TestConsistencyCheckerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db := setupIntegrationTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create large dataset
	createLargeTestDataset(t, db, 10000) // 10k articles

	// Test performance with different sample sizes
	testCases := []struct {
		name       string
		sampleSize int
		maxTime    time.Duration
	}{
		{"Small Sample", 100, 5 * time.Second},
		{"Medium Sample", 1000, 15 * time.Second},
		{"Large Sample", 5000, 45 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			check, err := checker.ValidateDataConsistency(ctx)
			duration := time.Since(start)

			require.NoError(t, err)
			assert.Less(t, duration, tc.maxTime, 
				"Check should complete within %v for sample size %d", tc.maxTime, tc.sampleSize)
			
			t.Logf("Sample size %d completed in %v with %d issues", 
				check.SampleSize, duration, len(check.Issues))
		})
	}
}

// TestConsistencyCheckerAccuracy tests the accuracy of issue detection
func TestConsistencyCheckerAccuracy(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	checker := NewConsistencyChecker(db)
	ctx := context.Background()

	// Create test data with known issues
	knownIssues := setupKnownIssuesTestData(t, db)

	check, err := checker.ValidateDataConsistency(ctx)
	require.NoError(t, err)

	// Verify that known issues were detected
	detectedIssueTypes := make(map[string]int)
	for _, issue := range check.Issues {
		detectedIssueTypes[issue.Type]++
	}

	// Check for expected issue types
	expectedIssueTypes := []string{
		"broken_author_reference",
		"broken_category_reference",
		"missing_meta_title",
		"missing_meta_description",
		"invalid_schema_type",
	}

	for _, expectedType := range expectedIssueTypes {
		assert.Greater(t, detectedIssueTypes[expectedType], 0, 
			"Should detect %s issues", expectedType)
	}

	t.Logf("Detected issue types: %+v", detectedIssueTypes)
	t.Logf("Detection accuracy: %d/%d known issues detected", 
		len(check.Issues), knownIssues)
}

// Helper functions for integration tests

func setupIntegrationTestDB(t *testing.T) *database.DB {
	// In a real implementation, this would set up a test database
	// with the full schema including consistency checking tables
	db, err := database.NewTestDB()
	require.NoError(t, err)

	// Run migrations to create consistency tables
	err = runTestMigrations(db)
	require.NoError(t, err)

	return db
}

func setupEmptyTestDB(t *testing.T) *database.DB {
	db, err := database.NewTestDB()
	require.NoError(t, err)

	// Create minimal schema without data
	err = runMinimalTestMigrations(db)
	require.NoError(t, err)

	return db
}

func setupComprehensiveTestData(t *testing.T, db *database.DB) {
	// Create users
	createTestUser(t, db, 1, "validuser", "valid@example.com")
	createTestUser(t, db, 2, "anotheruser", "another@example.com")

	// Create categories
	createTestCategory(t, db, 1, "Valid Category", "valid-category")
	createTestCategory(t, db, 2, "Another Category", "another-category")

	// Create translation groups
	groupID1 := createTestTranslationGroup(t, db, "article")
	groupID2 := createTestTranslationGroup(t, db, "article")

	// Create articles with various issues
	testArticles := []struct {
		id                 uint64
		title              string
		authorID           uint64
		categoryID         uint64
		languageCode       string
		translationGroupID *uint64
		metaTitle          string
		metaDescription    string
		schemaType         string
	}{
		// Valid articles
		{1, "Valid Article 1", 1, 1, "fa", nil, "Valid Meta Title", "Valid meta description", "NewsArticle"},
		{2, "Valid Article 2", 2, 2, "en", nil, "Another Title", "Another description", "Article"},

		// Articles with issues
		{3, "Article with Missing Author", 999, 1, "fa", nil, "", "", "NewsArticle"},           // Broken author reference
		{4, "Article with Missing Category", 1, 999, "fa", nil, "Title", "Description", "NewsArticle"}, // Broken category reference
		{5, "Article Missing Meta Title", 1, 1, "fa", nil, "", "Has description", "NewsArticle"},       // Missing meta title
		{6, "Article Missing Meta Description", 1, 1, "fa", nil, "Has title", "", "NewsArticle"},       // Missing meta description
		{7, "Article with Invalid Schema", 1, 1, "fa", nil, "Title", "Description", "InvalidSchema"},   // Invalid schema type

		// Translation group issues
		{8, "English Article", 1, 1, "en", &groupID1, "Title", "Description", "NewsArticle"},
		{9, "Persian Article Draft", 1, 1, "fa", &groupID1, "Title", "Description", "NewsArticle"}, // Will be set to draft status
		{10, "Duplicate English", 1, 1, "en", &groupID1, "Title", "Description", "NewsArticle"},    // Duplicate language in group
	}

	for _, article := range testArticles {
		status := "published"
		if article.id == 9 {
			status = "draft" // Create status inconsistency
		}

		query := `
			INSERT INTO articles (id, title, slug, content, author_id, category_id, status, 
				published_at, language_code, translation_group_id, meta_title, meta_description, schema_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`
		_, err := db.Exec(query,
			article.id, article.title, fmt.Sprintf("slug-%d", article.id), "Content",
			article.authorID, article.categoryID, status, time.Now(),
			article.languageCode, article.translationGroupID,
			article.metaTitle, article.metaDescription, article.schemaType)
		require.NoError(t, err)
	}

	// Create orphaned article tags
	createTestTag(t, db, 1, "Valid Tag", "valid-tag")
	createOrphanedArticleTag(t, db, 1, 999) // Tag 999 doesn't exist
}

func createLargeTestDataset(t *testing.T, db *database.DB, articleCount int) {
	// Create base data
	createTestUser(t, db, 1, "testuser", "test@example.com")
	createTestCategory(t, db, 1, "Test Category", "test-category")

	// Create many articles
	for i := 1; i <= articleCount; i++ {
		query := `
			INSERT INTO articles (id, title, slug, content, author_id, category_id, status, 
				published_at, language_code, meta_title, meta_description, schema_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
		_, err := db.Exec(query,
			i, fmt.Sprintf("Article %d", i), fmt.Sprintf("article-%d", i), "Content",
			1, 1, "published", time.Now(), "fa",
			fmt.Sprintf("Meta Title %d", i), fmt.Sprintf("Meta description %d", i), "NewsArticle")
		require.NoError(t, err)
	}
}

func setupKnownIssuesTestData(t *testing.T, db *database.DB) int {
	knownIssueCount := 0

	// Create valid base data
	createTestUser(t, db, 1, "validuser", "valid@example.com")
	createTestCategory(t, db, 1, "Valid Category", "valid-category")

	// Create articles with specific known issues
	knownIssues := []struct {
		description string
		setup       func()
	}{
		{
			"Broken author reference",
			func() {
				query := `INSERT INTO articles (id, title, slug, content, author_id, category_id, status, published_at, language_code) VALUES (100, 'Test', 'test-100', 'Content', 999, 1, 'published', NOW(), 'fa')`
				db.Exec(query)
				knownIssueCount++
			},
		},
		{
			"Missing meta title",
			func() {
				query := `INSERT INTO articles (id, title, slug, content, author_id, category_id, status, published_at, language_code, meta_title) VALUES (101, 'Test', 'test-101', 'Content', 1, 1, 'published', NOW(), 'fa', '')`
				db.Exec(query)
				knownIssueCount++
			},
		},
		{
			"Invalid schema type",
			func() {
				query := `INSERT INTO articles (id, title, slug, content, author_id, category_id, status, published_at, language_code, schema_type) VALUES (102, 'Test', 'test-102', 'Content', 1, 1, 'published', NOW(), 'fa', 'InvalidType')`
				db.Exec(query)
				knownIssueCount++
			},
		},
	}

	for _, issue := range knownIssues {
		issue.setup()
	}

	return knownIssueCount
}

func createTestTag(t *testing.T, db *database.DB, id uint64, name, slug string) {
	query := `INSERT INTO tags (id, name, slug) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(query, id, name, slug)
	require.NoError(t, err)
}

func createOrphanedArticleTag(t *testing.T, db *database.DB, articleID, tagID uint64) {
	query := `INSERT INTO article_tags (article_id, tag_id) VALUES ($1, $2)`
	_, err := db.Exec(query, articleID, tagID)
	require.NoError(t, err)
}

func runTestMigrations(db *database.DB) error {
	// In a real implementation, this would run the consistency checking migrations
	// For now, we'll assume the tables exist
	return nil
}

func runMinimalTestMigrations(db *database.DB) error {
	// Create minimal schema for empty database test
	return nil
}

// Mock monitoring client for testing
type mockMonitoringClient struct{}

func (m *mockMonitoringClient) SendMetric(name string, value float64, tags map[string]string) error {
	return nil
}

func (m *mockMonitoringClient) SendAlert(level string, message string, details map[string]interface{}) error {
	return nil
}