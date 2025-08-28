package services

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupCanonicalTestDB(t *testing.T) *sql.DB {
	// Use test database connection
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
	}

	// Ping to ensure connection works
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: could not ping test database: %v", err)
	}

	return db
}

func setupCanonicalTestData(t *testing.T, db *sql.DB) (uint64, uint64, uint64) {
	// Create test user
	var userID uint64
	err := db.QueryRow(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ('testuser', 'test@example.com', 'hash', 'admin')
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test category
	var categoryID uint64
	err = db.QueryRow(`
		INSERT INTO categories (name, slug, description)
		VALUES ('Test Category', 'test-category', 'Test category description')
		RETURNING id
	`).Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Create test tag
	var tagID uint64
	err = db.QueryRow(`
		INSERT INTO tags (name, slug, description, keywords)
		VALUES ('Test Tag', 'test-tag', 'Test tag description', '["test", "keyword"]')
		RETURNING id
	`).Scan(&tagID)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	// Create test article partition for current month
	currentMonth := time.Now().Format("2006_01")
	partitionName := "articles_" + currentMonth
	
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + partitionName + ` PARTITION OF articles
		FOR VALUES FROM ($1) TO ($2)
	`, time.Now().Truncate(24*time.Hour).Format("2006-01-02"), 
		time.Now().AddDate(0, 1, 0).Truncate(24*time.Hour).Format("2006-01-02"))
	if err != nil {
		t.Logf("Partition may already exist: %v", err)
	}

	// Create test article
	var articleID uint64
	err = db.QueryRow(`
		INSERT INTO articles (title, slug, content, author_id, category_id, status, published_at)
		VALUES ('Test Article', 'test-article', 'Test content', $1, $2, 'published', NOW())
		RETURNING id
	`, userID, categoryID).Scan(&articleID)
	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	return userID, categoryID, tagID
}

func cleanupCanonicalTestData(t *testing.T, db *sql.DB) {
	// Clean up in reverse order of dependencies
	_, err := db.Exec("DELETE FROM canonical_jobs")
	if err != nil {
		t.Logf("Failed to clean up canonical_jobs: %v", err)
	}

	_, err = db.Exec("DELETE FROM articles")
	if err != nil {
		t.Logf("Failed to clean up articles: %v", err)
	}

	_, err = db.Exec("DELETE FROM tags")
	if err != nil {
		t.Logf("Failed to clean up tags: %v", err)
	}

	_, err = db.Exec("DELETE FROM categories")
	if err != nil {
		t.Logf("Failed to clean up categories: %v", err)
	}

	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		t.Logf("Failed to clean up users: %v", err)
	}
}

func TestCanonicalManager_ScheduleCanonicalJob(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, categoryID, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	tests := []struct {
		name          string
		articleID     uint64
		target        CanonicalTarget
		createdBy     *uint64
		adminOverride bool
		expectError   bool
	}{
		{
			name:          "Schedule tag canonical job",
			articleID:     1,
			target:        NewTagTarget(tagID),
			createdBy:     &userID,
			adminOverride: false,
			expectError:   false,
		},
		{
			name:          "Schedule category canonical job",
			articleID:     1,
			target:        NewCategoryTarget(categoryID),
			createdBy:     &userID,
			adminOverride: false,
			expectError:   false,
		},
		{
			name:          "Schedule URL canonical job",
			articleID:     1,
			target:        NewURLTarget("/custom/canonical/url"),
			createdBy:     &userID,
			adminOverride: false,
			expectError:   false,
		},
		{
			name:          "Schedule with admin override",
			articleID:     1,
			target:        NewTagTarget(tagID),
			createdBy:     &userID,
			adminOverride: true,
			expectError:   false,
		},
		{
			name:          "Invalid target type",
			articleID:     1,
			target:        CanonicalTarget{Type: "invalid"},
			createdBy:     &userID,
			adminOverride: false,
			expectError:   true,
		},
		{
			name:          "Tag target without ID",
			articleID:     1,
			target:        CanonicalTarget{Type: "tag"},
			createdBy:     &userID,
			adminOverride: false,
			expectError:   true,
		},
		{
			name:          "URL target without URL",
			articleID:     1,
			target:        CanonicalTarget{Type: "url"},
			createdBy:     &userID,
			adminOverride: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID, err := cm.ScheduleCanonicalJob(tt.articleID, tt.target, tt.createdBy, tt.adminOverride)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if jobID == 0 {
				t.Errorf("Expected non-zero job ID")
				return
			}

			// Verify job was created correctly
			var job CanonicalJob
			err = db.QueryRow(`
				SELECT id, article_id, target_type, target_id, target_url, 
				       admin_override, status, created_by
				FROM canonical_jobs WHERE id = $1
			`, jobID).Scan(
				&job.ID, &job.ArticleID, &job.TargetType, &job.TargetID, 
				&job.TargetURL, &job.AdminOverride, &job.Status, &job.CreatedBy,
			)
			if err != nil {
				t.Errorf("Failed to fetch created job: %v", err)
				return
			}

			if job.ArticleID != tt.articleID {
				t.Errorf("Expected article_id %d, got %d", tt.articleID, job.ArticleID)
			}

			if job.TargetType != tt.target.Type {
				t.Errorf("Expected target_type %s, got %s", tt.target.Type, job.TargetType)
			}

			if job.AdminOverride != tt.adminOverride {
				t.Errorf("Expected admin_override %t, got %t", tt.adminOverride, job.AdminOverride)
			}

			if job.Status != "pending" {
				t.Errorf("Expected status 'pending', got %s", job.Status)
			}
		})
	}
}

func TestCanonicalManager_GenerateCanonicalURL(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	_, categoryID, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	tests := []struct {
		name        string
		target      CanonicalTarget
		expectedURL string
		expectError bool
	}{
		{
			name:        "Generate tag URL",
			target:      NewTagTarget(tagID),
			expectedURL: "/tag/test-tag",
			expectError: false,
		},
		{
			name:        "Generate category URL",
			target:      NewCategoryTarget(categoryID),
			expectedURL: "/category/test-category",
			expectError: false,
		},
		{
			name:        "Generate custom URL",
			target:      NewURLTarget("/custom/path"),
			expectedURL: "/custom/path",
			expectError: false,
		},
		{
			name:        "Invalid tag ID",
			target:      NewTagTarget(99999),
			expectedURL: "",
			expectError: true,
		},
		{
			name:        "Invalid category ID",
			target:      NewCategoryTarget(99999),
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := cm.GenerateCanonicalURL(tt.target)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if url != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, url)
			}
		})
	}
}

func TestCanonicalManager_ProcessJob(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create a test job
	jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, true) // admin override for immediate processing
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Process the job
	success, err := cm.ProcessJob(jobID, &userID)
	if err != nil {
		t.Fatalf("Failed to process job: %v", err)
	}

	if !success {
		t.Errorf("Expected job processing to succeed")
	}

	// Verify job status was updated
	var status string
	var processedBy *uint64
	err = db.QueryRow(`
		SELECT status, processed_by 
		FROM canonical_jobs 
		WHERE id = $1
	`, jobID).Scan(&status, &processedBy)
	if err != nil {
		t.Fatalf("Failed to fetch job status: %v", err)
	}

	if status != "processed" {
		t.Errorf("Expected status 'processed', got %s", status)
	}

	if processedBy == nil || *processedBy != userID {
		t.Errorf("Expected processed_by to be %d, got %v", userID, processedBy)
	}

	// Verify article canonical URL was updated
	var canonicalURL sql.NullString
	err = db.QueryRow(`
		SELECT canonical_url 
		FROM articles 
		WHERE id = 1
	`).Scan(&canonicalURL)
	if err != nil {
		t.Fatalf("Failed to fetch article canonical URL: %v", err)
	}

	if !canonicalURL.Valid {
		t.Errorf("Expected canonical URL to be set")
	}

	expectedURL := "/tag/test-tag"
	if canonicalURL.String != expectedURL {
		t.Errorf("Expected canonical URL %s, got %s", expectedURL, canonicalURL.String)
	}
}

func TestCanonicalManager_GetPendingJobs(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create test jobs
	jobID1, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, true) // admin override
	if err != nil {
		t.Fatalf("Failed to create test job 1: %v", err)
	}

	_, err = cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false) // normal delay
	if err != nil {
		t.Fatalf("Failed to create test job 2: %v", err)
	}

	// Get pending jobs
	jobs, err := cm.GetPendingJobs()
	if err != nil {
		t.Fatalf("Failed to get pending jobs: %v", err)
	}

	// Should have at least the admin override job (ready for processing)
	if len(jobs) == 0 {
		t.Errorf("Expected at least one pending job")
	}

	// Find our admin override job
	var adminJob *CanonicalJob
	for _, job := range jobs {
		if job.ID == jobID1 {
			adminJob = &job
			break
		}
	}

	if adminJob == nil {
		t.Errorf("Admin override job not found in pending jobs")
	} else {
		if !adminJob.AdminOverride {
			t.Errorf("Expected admin override job to have admin_override = true")
		}
	}

	// The normal delay job should not be ready yet (unless we're testing at exactly the right time)
	// We don't check for it specifically since it might not be in the pending list
}

func TestCanonicalManager_CancelJob(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create a test job
	jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Cancel the job
	err = cm.CancelJob(jobID)
	if err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}

	// Verify job status
	var status string
	err = db.QueryRow(`
		SELECT status FROM canonical_jobs WHERE id = $1
	`, jobID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to fetch job status: %v", err)
	}

	if status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %s", status)
	}

	// Try to cancel again (should fail)
	err = cm.CancelJob(jobID)
	if err == nil {
		t.Errorf("Expected error when cancelling already cancelled job")
	}
}

func TestCanonicalManager_RetryFailedJob(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create a test job and manually set it to failed
	jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Manually set job to failed status
	_, err = db.Exec(`
		UPDATE canonical_jobs 
		SET status = 'failed', error_message = 'Test error', retry_count = 1
		WHERE id = $1
	`, jobID)
	if err != nil {
		t.Fatalf("Failed to set job to failed: %v", err)
	}

	// Retry the job
	err = cm.RetryFailedJob(jobID, false)
	if err != nil {
		t.Fatalf("Failed to retry job: %v", err)
	}

	// Verify job status
	var status string
	var errorMessage sql.NullString
	err = db.QueryRow(`
		SELECT status, error_message FROM canonical_jobs WHERE id = $1
	`, jobID).Scan(&status, &errorMessage)
	if err != nil {
		t.Fatalf("Failed to fetch job status: %v", err)
	}

	if status != "pending" {
		t.Errorf("Expected status 'pending', got %s", status)
	}

	if errorMessage.Valid {
		t.Errorf("Expected error_message to be cleared, got %s", errorMessage.String)
	}
}

func TestCanonicalManager_GetJobStats(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create test jobs with different statuses
	jobID1, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job 1: %v", err)
	}

	_, err = cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job 2: %v", err)
	}

	// Set one job to processed
	_, err = db.Exec(`
		UPDATE canonical_jobs 
		SET status = 'processed', processed_at = NOW()
		WHERE id = $1
	`, jobID1)
	if err != nil {
		t.Fatalf("Failed to set job to processed: %v", err)
	}

	// Get stats
	stats, err := cm.GetJobStats()
	if err != nil {
		t.Fatalf("Failed to get job stats: %v", err)
	}

	if stats["pending"] < 1 {
		t.Errorf("Expected at least 1 pending job, got %d", stats["pending"])
	}

	if stats["processed"] < 1 {
		t.Errorf("Expected at least 1 processed job, got %d", stats["processed"])
	}
}

func TestCanonicalManager_CleanupOldJobs(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	// Create an old processed job
	jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Set job to processed and make it old
	_, err = db.Exec(`
		UPDATE canonical_jobs 
		SET status = 'processed', 
		    processed_at = NOW() - INTERVAL '31 days',
		    updated_at = NOW() - INTERVAL '31 days'
		WHERE id = $1
	`, jobID)
	if err != nil {
		t.Fatalf("Failed to set job to old processed: %v", err)
	}

	// Run cleanup
	deletedCount, err := cm.CleanupOldJobs()
	if err != nil {
		t.Fatalf("Failed to cleanup old jobs: %v", err)
	}

	if deletedCount < 1 {
		t.Errorf("Expected at least 1 job to be deleted, got %d", deletedCount)
	}

	// Verify job was deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM canonical_jobs WHERE id = $1`, jobID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check if job was deleted: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected job to be deleted, but it still exists")
	}
}

func TestCanonicalTarget_Helpers(t *testing.T) {
	// Test helper functions
	tagTarget := NewTagTarget(123)
	if tagTarget.Type != "tag" {
		t.Errorf("Expected tag target type 'tag', got %s", tagTarget.Type)
	}
	if tagTarget.ID == nil || *tagTarget.ID != 123 {
		t.Errorf("Expected tag target ID 123, got %v", tagTarget.ID)
	}
	if tagTarget.URL != nil {
		t.Errorf("Expected tag target URL to be nil, got %v", tagTarget.URL)
	}

	categoryTarget := NewCategoryTarget(456)
	if categoryTarget.Type != "category" {
		t.Errorf("Expected category target type 'category', got %s", categoryTarget.Type)
	}
	if categoryTarget.ID == nil || *categoryTarget.ID != 456 {
		t.Errorf("Expected category target ID 456, got %v", categoryTarget.ID)
	}
	if categoryTarget.URL != nil {
		t.Errorf("Expected category target URL to be nil, got %v", categoryTarget.URL)
	}

	urlTarget := NewURLTarget("/test/url")
	if urlTarget.Type != "url" {
		t.Errorf("Expected URL target type 'url', got %s", urlTarget.Type)
	}
	if urlTarget.URL == nil || *urlTarget.URL != "/test/url" {
		t.Errorf("Expected URL target URL '/test/url', got %v", urlTarget.URL)
	}
	if urlTarget.ID != nil {
		t.Errorf("Expected URL target ID to be nil, got %v", urlTarget.ID)
	}
}