package services

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

func setupVersioningTestDB(t *testing.T) *sql.DB {
	// This would connect to a test database
	// For now, we'll use a mock or skip if no test DB is available
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	
	// Create test tables (simplified for testing)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS article_versions (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	return db
}

func cleanupVersioningTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS article_versions")
	if err != nil {
		t.Errorf("Failed to cleanup test table: %v", err)
	}
	db.Close()
}

func TestVersioningService_CreateVersion(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create test article
	article := &models.Article{
		ID:         1,
		Title:      "Test Article",
		Slug:       "test-article",
		Content:    "This is test content",
		Excerpt:    "Test excerpt",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		AutoLinking: true,
		SEOData: models.SEOData{
			MetaTitle:       "Test Meta Title",
			MetaDescription: "Test meta description",
			SchemaType:      "NewsArticle",
		},
	}
	
	// Test creating first version
	version, err := vs.CreateVersion(article, "Initial version", 1)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}
	
	if version.ID == 0 {
		t.Error("Version ID should not be 0")
	}
	
	if version.VersionNumber != 1 {
		t.Errorf("Expected version number 1, got %d", version.VersionNumber)
	}
	
	if version.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, version.Title)
	}
	
	if version.ChangeSummary != "Initial version" {
		t.Errorf("Expected change summary 'Initial version', got %s", version.ChangeSummary)
	}
	
	// Test creating second version
	article.Title = "Updated Test Article"
	version2, err := vs.CreateVersion(article, "Updated title", 1)
	if err != nil {
		t.Fatalf("Failed to create second version: %v", err)
	}
	
	if version2.VersionNumber != 2 {
		t.Errorf("Expected version number 2, got %d", version2.VersionNumber)
	}
	
	if version2.Title != "Updated Test Article" {
		t.Errorf("Expected updated title, got %s", version2.Title)
	}
}

func TestVersioningService_GetVersionHistory(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create test article with multiple versions
	article := &models.Article{
		ID:         1,
		Title:      "Test Article",
		Slug:       "test-article",
		Content:    "This is test content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}
	
	// Create 3 versions
	for i := 1; i <= 3; i++ {
		article.Title = fmt.Sprintf("Test Article v%d", i)
		_, err := vs.CreateVersion(article, fmt.Sprintf("Version %d", i), 1)
		if err != nil {
			t.Fatalf("Failed to create version %d: %v", i, err)
		}
	}
	
	// Get version history
	versions, err := vs.GetVersionHistory(1)
	if err != nil {
		t.Fatalf("Failed to get version history: %v", err)
	}
	
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
	
	// Versions should be ordered by version number DESC
	if versions[0].VersionNumber != 3 {
		t.Errorf("Expected first version to be 3, got %d", versions[0].VersionNumber)
	}
	
	if versions[2].VersionNumber != 1 {
		t.Errorf("Expected last version to be 1, got %d", versions[2].VersionNumber)
	}
}

func TestVersioningService_GetVersion(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create test version
	article := &models.Article{
		ID:         1,
		Title:      "Test Article",
		Slug:       "test-article",
		Content:    "This is test content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}
	
	createdVersion, err := vs.CreateVersion(article, "Test version", 1)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}
	
	// Get specific version
	version, err := vs.GetVersion(1, 1)
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	
	if version.ID != createdVersion.ID {
		t.Errorf("Expected version ID %d, got %d", createdVersion.ID, version.ID)
	}
	
	if version.Title != article.Title {
		t.Errorf("Expected title %s, got %s", article.Title, version.Title)
	}
	
	// Test getting non-existent version
	_, err = vs.GetVersion(1, 999)
	if err == nil {
		t.Error("Expected error for non-existent version")
	}
}

func TestVersioningService_CompareVersions(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create first version
	article := &models.Article{
		ID:         1,
		Title:      "Original Title",
		Slug:       "original-title",
		Content:    "Original content",
		Excerpt:    "Original excerpt",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle: "Original Meta",
			SchemaType: "NewsArticle",
		},
	}
	
	_, err := vs.CreateVersion(article, "First version", 1)
	if err != nil {
		t.Fatalf("Failed to create first version: %v", err)
	}
	
	// Create second version with changes
	article.Title = "Updated Title"
	article.Content = "Updated content"
	article.SEOData.MetaTitle = "Updated Meta"
	
	_, err = vs.CreateVersion(article, "Second version", 1)
	if err != nil {
		t.Fatalf("Failed to create second version: %v", err)
	}
	
	// Compare versions
	comparison, err := vs.CompareVersions(1, 1, 2)
	if err != nil {
		t.Fatalf("Failed to compare versions: %v", err)
	}
	
	// Check that changes were detected
	if len(comparison.Changes) == 0 {
		t.Error("Expected changes to be detected")
	}
	
	// Check specific changes
	if titleChange, exists := comparison.Changes["title"]; exists {
		if titleChange.OldValue != "Original Title" {
			t.Errorf("Expected old title 'Original Title', got %s", titleChange.OldValue)
		}
		if titleChange.NewValue != "Updated Title" {
			t.Errorf("Expected new title 'Updated Title', got %s", titleChange.NewValue)
		}
	} else {
		t.Error("Expected title change to be detected")
	}
	
	if contentChange, exists := comparison.Changes["content"]; exists {
		if contentChange.OldValue != "Original content" {
			t.Errorf("Expected old content 'Original content', got %s", contentChange.OldValue)
		}
		if contentChange.NewValue != "Updated content" {
			t.Errorf("Expected new content 'Updated content', got %s", contentChange.NewValue)
		}
	} else {
		t.Error("Expected content change to be detected")
	}
}

func TestVersioningService_RestoreVersion(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	// We need to create the articles table for restore functionality
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create articles table: %v", err)
	}
	defer db.Exec("DROP TABLE IF EXISTS articles")
	
	vs := NewVersioningService(db)
	
	// Insert test article
	_, err = db.Exec(`
		INSERT INTO articles (id, title, slug, content, author_id, category_id, status, language_code)
		VALUES (1, 'Current Title', 'current-title', 'Current content', 1, 1, 'published', 'en')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	// Create version to restore to
	article := &models.Article{
		ID:         1,
		Title:      "Original Title",
		Slug:       "original-title",
		Content:    "Original content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}
	
	_, err = vs.CreateVersion(article, "Original version", 1)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}
	
	// Restore to version 1
	err = vs.RestoreVersion(1, 1, 2)
	if err != nil {
		t.Fatalf("Failed to restore version: %v", err)
	}
	
	// Verify article was updated
	var restoredTitle, restoredContent string
	err = db.QueryRow("SELECT title, content FROM articles WHERE id = 1").Scan(&restoredTitle, &restoredContent)
	if err != nil {
		t.Fatalf("Failed to get restored article: %v", err)
	}
	
	if restoredTitle != "Original Title" {
		t.Errorf("Expected restored title 'Original Title', got %s", restoredTitle)
	}
	
	if restoredContent != "Original content" {
		t.Errorf("Expected restored content 'Original content', got %s", restoredContent)
	}
	
	// Verify restoration created a new version
	versions, err := vs.GetVersionHistory(1)
	if err != nil {
		t.Fatalf("Failed to get version history: %v", err)
	}
	
	if len(versions) < 2 {
		t.Error("Expected at least 2 versions after restoration")
	}
	
	// Latest version should be the restoration
	latestVersion := versions[0]
	if !strings.Contains(latestVersion.ChangeSummary, "Restored to version 1") {
		t.Errorf("Expected restoration change summary, got %s", latestVersion.ChangeSummary)
	}
}

func TestVersioningService_GetVersionStats(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create test versions for multiple articles
	for articleID := 1; articleID <= 3; articleID++ {
		for version := 1; version <= 2; version++ {
			article := &models.Article{
				ID:         uint64(articleID),
				Title:      fmt.Sprintf("Article %d v%d", articleID, version),
				Slug:       fmt.Sprintf("article-%d-v%d", articleID, version),
				Content:    fmt.Sprintf("Content for article %d version %d", articleID, version),
				AuthorID:   1,
				CategoryID: 1,
				Status:     "draft",
				LanguageCode: "en",
				SEOData: models.SEOData{
					SchemaType: "NewsArticle",
				},
			}
			
			_, err := vs.CreateVersion(article, fmt.Sprintf("Version %d", version), 1)
			if err != nil {
				t.Fatalf("Failed to create version: %v", err)
			}
		}
	}
	
	// Get stats
	stats, err := vs.GetVersionStats()
	if err != nil {
		t.Fatalf("Failed to get version stats: %v", err)
	}
	
	if stats.TotalVersions != 6 {
		t.Errorf("Expected 6 total versions, got %d", stats.TotalVersions)
	}
	
	if stats.ArticlesWithVersions != 3 {
		t.Errorf("Expected 3 articles with versions, got %d", stats.ArticlesWithVersions)
	}
	
	if stats.AverageVersionsPerArticle != 2.0 {
		t.Errorf("Expected average 2.0 versions per article, got %f", stats.AverageVersionsPerArticle)
	}
	
	if stats.MaxVersionsForArticle != 2 {
		t.Errorf("Expected max 2 versions for article, got %d", stats.MaxVersionsForArticle)
	}
}

func TestVersioningService_DeleteOldVersions(t *testing.T) {
	db := setupVersioningTestDB(t)
	defer cleanupVersioningTestDB(t, db)
	
	vs := NewVersioningService(db)
	
	// Create old version (simulate by inserting directly with old timestamp)
	_, err := db.Exec(`
		INSERT INTO article_versions (
			article_id, version_number, title, slug, content, author_id,
			category_id, status, language_code, schema_type, change_summary,
			created_by, created_at
		) VALUES (
			1, 1, 'Old Version', 'old-version', 'Old content', 1,
			1, 'draft', 'en', 'NewsArticle', 'Old version',
			1, NOW() - INTERVAL '40 days'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to insert old version: %v", err)
	}
	
	// Create recent version (this should be kept as it's the latest)
	article := &models.Article{
		ID:         1,
		Title:      "Recent Version",
		Slug:       "recent-version",
		Content:    "Recent content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}
	
	_, err = vs.CreateVersion(article, "Recent version", 1)
	if err != nil {
		t.Fatalf("Failed to create recent version: %v", err)
	}
	
	// Delete versions older than 30 days
	deletedCount, err := vs.DeleteOldVersions(30)
	if err != nil {
		t.Fatalf("Failed to delete old versions: %v", err)
	}
	
	if deletedCount != 1 {
		t.Errorf("Expected 1 deleted version, got %d", deletedCount)
	}
	
	// Verify only recent version remains
	versions, err := vs.GetVersionHistory(1)
	if err != nil {
		t.Fatalf("Failed to get version history: %v", err)
	}
	
	if len(versions) != 1 {
		t.Errorf("Expected 1 remaining version, got %d", len(versions))
	}
	
	if versions[0].Title != "Recent Version" {
		t.Errorf("Expected recent version to remain, got %s", versions[0].Title)
	}
}