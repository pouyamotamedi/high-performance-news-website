package services

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

// Integration tests for versioning and moderation system
func TestVersioningModerationIntegration(t *testing.T) {
	// Skip if no test database available
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	// Setup test tables
	setupIntegrationTables(t, db)
	defer cleanupIntegrationTables(t, db)

	// Create services
	mockAI := &MockAIService{
		feedback: &models.AIFeedback{
			Provider:             "mock",
			QualityScore:         0.75, // Below auto-approve threshold
			GrammarScore:         &[]float64{0.80}[0],
			ReadabilityScore:     &[]float64{0.70}[0],
			AppropriatenessScore: &[]float64{0.90}[0],
			Issues: []models.AIIssue{
				{
					Type:        "grammar",
					Severity:    "medium",
					Description: "Minor grammar issue",
					Suggestion:  "Fix grammar",
				},
			},
			Suggestions:      []models.AISuggestion{},
			FlaggedContent:   []models.AIFlaggedContent{},
			ProcessingTimeMs: 150,
			Confidence:       0.85,
		},
	}

	versioningService := NewVersioningService(db)
	moderationService := NewModerationService(db, mockAI)
	bulkModerationService := NewBulkModerationService(db, moderationService)

	// Test scenario: Create article, submit for moderation, create versions
	t.Run("CompleteWorkflow", func(t *testing.T) {
		// Insert test article
		_, err := db.Exec(`
			INSERT INTO articles (id, title, slug, content, excerpt, author_id, category_id, status, language_code)
			VALUES (1, 'Test Article', 'test-article', 'Original content', 'Original excerpt', 1, 1, 'draft', 'en')
		`)
		if err != nil {
			t.Fatalf("Failed to insert test article: %v", err)
		}

		// Create initial version
		article := &models.Article{
			ID:           1,
			Title:        "Test Article",
			Slug:         "test-article",
			Content:      "Original content",
			Excerpt:      "Original excerpt",
			AuthorID:     1,
			CategoryID:   1,
			Status:       "draft",
			LanguageCode: "en",
			SEOData: models.SEOData{
				SchemaType: "NewsArticle",
			},
		}

		version1, err := versioningService.CreateVersion(article, "Initial version", 1)
		if err != nil {
			t.Fatalf("Failed to create initial version: %v", err)
		}

		// Submit for moderation
		moderationItem, err := moderationService.SubmitForModeration(1, "article", 2, 1)
		if err != nil {
			t.Fatalf("Failed to submit for moderation: %v", err)
		}

		// Wait for AI processing
		time.Sleep(200 * time.Millisecond)

		// Check moderation status (should be pending due to low score)
		var status string
		var aiScore sql.NullFloat64
		err = db.QueryRow(`
			SELECT status, ai_quality_score FROM moderation_queue WHERE id = $1
		`, moderationItem.ID).Scan(&status, &aiScore)
		if err != nil {
			t.Fatalf("Failed to get moderation status: %v", err)
		}

		if status != "pending" {
			t.Errorf("Expected status 'pending', got %s", status)
		}

		if !aiScore.Valid || aiScore.Float64 != 0.75 {
			t.Errorf("Expected AI score 0.75, got %v", aiScore)
		}

		// Approve the content
		err = moderationService.ApproveContent(moderationItem.ID, 2, "Content approved after review", false)
		if err != nil {
			t.Fatalf("Failed to approve content: %v", err)
		}

		// Update article and create new version
		article.Title = "Updated Test Article"
		article.Content = "Updated content with improvements"
		
		// Update in database first
		_, err = db.Exec(`
			UPDATE articles SET title = $2, content = $3, updated_at = NOW() WHERE id = $1
		`, article.ID, article.Title, article.Content)
		if err != nil {
			t.Fatalf("Failed to update article: %v", err)
		}

		version2, err := versioningService.CreateVersion(article, "Updated content", 1)
		if err != nil {
			t.Fatalf("Failed to create second version: %v", err)
		}

		// Verify version history
		versions, err := versioningService.GetVersionHistory(1)
		if err != nil {
			t.Fatalf("Failed to get version history: %v", err)
		}

		if len(versions) != 2 {
			t.Errorf("Expected 2 versions, got %d", len(versions))
		}

		// Compare versions
		comparison, err := versioningService.CompareVersions(1, 1, 2)
		if err != nil {
			t.Fatalf("Failed to compare versions: %v", err)
		}

		if len(comparison.Changes) == 0 {
			t.Error("Expected changes between versions")
		}

		// Test version restoration
		err = versioningService.RestoreVersion(1, 1, 2)
		if err != nil {
			t.Fatalf("Failed to restore version: %v", err)
		}

		// Verify restoration created new version
		versionsAfterRestore, err := versioningService.GetVersionHistory(1)
		if err != nil {
			t.Fatalf("Failed to get version history after restore: %v", err)
		}

		if len(versionsAfterRestore) != 3 {
			t.Errorf("Expected 3 versions after restore, got %d", len(versionsAfterRestore))
		}

		// Verify article was restored
		var restoredTitle string
		err = db.QueryRow("SELECT title FROM articles WHERE id = 1").Scan(&restoredTitle)
		if err != nil {
			t.Fatalf("Failed to get restored article: %v", err)
		}

		if restoredTitle != "Test Article" {
			t.Errorf("Expected restored title 'Test Article', got %s", restoredTitle)
		}

		t.Logf("Integration test completed successfully:")
		t.Logf("- Created %d versions", len(versionsAfterRestore))
		t.Logf("- Version 1 ID: %d", version1.ID)
		t.Logf("- Version 2 ID: %d", version2.ID)
		t.Logf("- Moderation item ID: %d", moderationItem.ID)
		t.Logf("- Final article title: %s", restoredTitle)
	})

	// Test bulk moderation
	t.Run("BulkModerationWorkflow", func(t *testing.T) {
		// Create multiple articles and moderation items
		for i := 2; i <= 5; i++ {
			_, err := db.Exec(`
				INSERT INTO articles (id, title, content, author_id, category_id, status, language_code)
				VALUES ($1, $2, $3, 1, 1, 'draft', 'en')
			`, i, fmt.Sprintf("Article %d", i), fmt.Sprintf("Content %d", i))
			if err != nil {
				t.Fatalf("Failed to insert article %d: %v", i, err)
			}

			_, err = moderationService.SubmitForModeration(uint64(i), "article", 1, 1)
			if err != nil {
				t.Fatalf("Failed to submit article %d for moderation: %v", i, err)
			}
		}

		// Wait for AI processing
		time.Sleep(500 * time.Millisecond)

		// Create bulk approval job
		criteria := &models.BulkModerationCriteria{
			Status: []string{"pending"},
		}

		job, err := bulkModerationService.CreateBulkJob("Test Bulk Approval", "bulk_approve", criteria, 2)
		if err != nil {
			t.Fatalf("Failed to create bulk job: %v", err)
		}

		if job.TotalItems != 4 {
			t.Errorf("Expected 4 items in bulk job, got %d", job.TotalItems)
		}

		// Execute bulk job
		err = bulkModerationService.ExecuteBulkJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to execute bulk job: %v", err)
		}

		// Verify job completion
		completedJob, err := bulkModerationService.GetBulkJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to get completed job: %v", err)
		}

		if completedJob.Status != "completed" {
			t.Errorf("Expected job status 'completed', got %s", completedJob.Status)
		}

		if completedJob.SuccessfulItems != 4 {
			t.Errorf("Expected 4 successful items, got %d", completedJob.SuccessfulItems)
		}

		// Verify all items were approved
		var approvedCount int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM moderation_queue 
			WHERE article_id IN (2, 3, 4, 5) AND status = 'approved'
		`).Scan(&approvedCount)
		if err != nil {
			t.Fatalf("Failed to count approved items: %v", err)
		}

		if approvedCount != 4 {
			t.Errorf("Expected 4 approved items, got %d", approvedCount)
		}

		t.Logf("Bulk moderation test completed successfully:")
		t.Logf("- Bulk job ID: %d", job.ID)
		t.Logf("- Total items processed: %d", completedJob.ProcessedItems)
		t.Logf("- Successful items: %d", completedJob.SuccessfulItems)
		t.Logf("- Failed items: %d", completedJob.FailedItems)
	})
}

func TestHighVolumeVersioning(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	setupIntegrationTables(t, db)
	defer cleanupIntegrationTables(t, db)

	versioningService := NewVersioningService(db)

	// Test creating many versions for performance
	t.Run("HighVolumeVersionCreation", func(t *testing.T) {
		// Insert test article
		_, err := db.Exec(`
			INSERT INTO articles (id, title, content, author_id, category_id, status, language_code)
			VALUES (1, 'High Volume Test', 'Original content', 1, 1, 'draft', 'en')
		`)
		if err != nil {
			t.Fatalf("Failed to insert test article: %v", err)
		}

		startTime := time.Now()
		versionCount := 100

		// Create many versions
		for i := 1; i <= versionCount; i++ {
			article := &models.Article{
				ID:           1,
				Title:        fmt.Sprintf("Version %d Title", i),
				Slug:         fmt.Sprintf("version-%d-title", i),
				Content:      fmt.Sprintf("Content for version %d", i),
				AuthorID:     1,
				CategoryID:   1,
				Status:       "draft",
				LanguageCode: "en",
				SEOData: models.SEOData{
					SchemaType: "NewsArticle",
				},
			}

			_, err := versioningService.CreateVersion(article, fmt.Sprintf("Version %d", i), 1)
			if err != nil {
				t.Fatalf("Failed to create version %d: %v", i, err)
			}

			// Log progress every 25 versions
			if i%25 == 0 {
				elapsed := time.Since(startTime)
				t.Logf("Created %d versions in %v (avg: %v per version)", 
					i, elapsed, elapsed/time.Duration(i))
			}
		}

		totalTime := time.Since(startTime)
		avgTime := totalTime / time.Duration(versionCount)

		t.Logf("High volume test completed:")
		t.Logf("- Created %d versions in %v", versionCount, totalTime)
		t.Logf("- Average time per version: %v", avgTime)
		t.Logf("- Versions per second: %.2f", float64(versionCount)/totalTime.Seconds())

		// Verify all versions were created
		versions, err := versioningService.GetVersionHistory(1)
		if err != nil {
			t.Fatalf("Failed to get version history: %v", err)
		}

		if len(versions) != versionCount {
			t.Errorf("Expected %d versions, got %d", versionCount, len(versions))
		}

		// Test version stats
		stats, err := versioningService.GetVersionStats()
		if err != nil {
			t.Fatalf("Failed to get version stats: %v", err)
		}

		t.Logf("Version stats:")
		t.Logf("- Total versions: %d", stats.TotalVersions)
		t.Logf("- Articles with versions: %d", stats.ArticlesWithVersions)
		t.Logf("- Average versions per article: %.2f", stats.AverageVersionsPerArticle)
		t.Logf("- Max versions for article: %d", stats.MaxVersionsForArticle)

		// Performance check: getting version history should be fast
		historyStartTime := time.Now()
		_, err = versioningService.GetVersionHistory(1)
		historyTime := time.Since(historyStartTime)

		if historyTime > 100*time.Millisecond {
			t.Errorf("Version history query too slow: %v (expected < 100ms)", historyTime)
		}

		t.Logf("Version history query time: %v", historyTime)
	})
}

func setupIntegrationTables(t *testing.T, db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			excerpt TEXT,
			author_id BIGINT NOT NULL,
			category_id BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL,
			published_at TIMESTAMP WITH TIME ZONE,
			language_code VARCHAR(2) NOT NULL DEFAULT 'fa',
			translation_group_id BIGINT,
			auto_linking BOOLEAN DEFAULT true,
			moderation_status VARCHAR(20) DEFAULT 'approved',
			moderation_notes TEXT,
			last_moderated_at TIMESTAMP WITH TIME ZONE,
			last_moderated_by BIGINT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS article_versions (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT NOT NULL,
			version_number INTEGER NOT NULL,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			excerpt TEXT,
			author_id BIGINT NOT NULL,
			category_id BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL,
			published_at TIMESTAMP WITH TIME ZONE,
			meta_title VARCHAR(60),
			meta_description VARCHAR(160),
			canonical_url VARCHAR(500),
			schema_type VARCHAR(50) DEFAULT 'NewsArticle',
			language_code VARCHAR(2) NOT NULL DEFAULT 'fa',
			translation_group_id BIGINT,
			auto_linking BOOLEAN DEFAULT true,
			change_summary TEXT,
			created_by BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(article_id, version_number)
		)`,
		`CREATE TABLE IF NOT EXISTS moderation_queue (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT NOT NULL,
			article_version_id BIGINT,
			content_type VARCHAR(20) NOT NULL DEFAULT 'article',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 1,
			submitted_by BIGINT NOT NULL,
			assigned_to BIGINT,
			ai_quality_score DECIMAL(3,2),
			ai_feedback JSONB,
			moderator_notes TEXT,
			rejection_reason TEXT,
			auto_approved BOOLEAN DEFAULT false,
			submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			reviewed_at TIMESTAMP WITH TIME ZONE,
			reviewed_by BIGINT
		)`,
		`CREATE TABLE IF NOT EXISTS content_quality_checks (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT NOT NULL,
			article_version_id BIGINT,
			ai_provider VARCHAR(20) NOT NULL,
			quality_score DECIMAL(3,2) NOT NULL,
			grammar_score DECIMAL(3,2),
			readability_score DECIMAL(3,2),
			appropriateness_score DECIMAL(3,2),
			issues_found JSONB,
			suggestions JSONB,
			flagged_content JSONB,
			processing_time_ms INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS moderation_actions (
			id BIGSERIAL PRIMARY KEY,
			moderation_queue_id BIGINT NOT NULL,
			action VARCHAR(20) NOT NULL,
			performed_by BIGINT NOT NULL,
			notes TEXT,
			previous_status VARCHAR(20),
			new_status VARCHAR(20),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS bulk_moderation_jobs (
			id BIGSERIAL PRIMARY KEY,
			job_name VARCHAR(255) NOT NULL,
			job_type VARCHAR(50) NOT NULL,
			criteria JSONB NOT NULL,
			total_items INTEGER DEFAULT 0,
			processed_items INTEGER DEFAULT 0,
			successful_items INTEGER DEFAULT 0,
			failed_items INTEGER DEFAULT 0,
			status VARCHAR(20) DEFAULT 'pending',
			created_by BIGINT NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			error_log TEXT
		)`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	// Create indexes for performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_article_versions_article_id ON article_versions (article_id, version_number DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_queue_status ON moderation_queue (status, priority DESC, submitted_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_content_quality_checks_article ON content_quality_checks (article_id, created_at DESC)`,
	}

	for _, index := range indexes {
		_, err := db.Exec(index)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}
	}
}

func cleanupIntegrationTables(t *testing.T, db *sql.DB) {
	tables := []string{
		"bulk_moderation_jobs",
		"moderation_actions", 
		"content_quality_checks",
		"moderation_queue",
		"article_versions",
		"articles",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Errorf("Failed to cleanup table %s: %v", table, err)
		}
	}
}