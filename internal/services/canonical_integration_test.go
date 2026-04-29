package services

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestCanonicalWorkflowIntegration tests the complete canonicalization workflow
func TestCanonicalWorkflowIntegration(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, categoryID, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	t.Run("Complete Canonicalization Workflow", func(t *testing.T) {
		// Step 1: Schedule a canonical job with 48-hour delay
		jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, false)
		if err != nil {
			t.Fatalf("Failed to schedule canonical job: %v", err)
		}

		// Step 2: Verify job is scheduled but not ready for processing
		jobs, err := cm.GetPendingJobs()
		if err != nil {
			t.Fatalf("Failed to get pending jobs: %v", err)
		}

		// The job should not be in the pending list because it's not ready yet
		jobFound := false
		for _, job := range jobs {
			if job.ID == jobID {
				jobFound = true
				break
			}
		}

		if jobFound {
			t.Errorf("Job should not be ready for processing yet (48-hour delay)")
		}

		// Step 3: Test admin override - schedule immediate processing
		adminJobID, err := cm.ScheduleCanonicalJob(1, NewCategoryTarget(categoryID), &userID, true)
		if err != nil {
			t.Fatalf("Failed to schedule admin override job: %v", err)
		}

		// Step 4: Verify admin override job is ready for processing
		jobs, err = cm.GetPendingJobs()
		if err != nil {
			t.Fatalf("Failed to get pending jobs: %v", err)
		}

		adminJobFound := false
		for _, job := range jobs {
			if job.ID == adminJobID && job.AdminOverride {
				adminJobFound = true
				break
			}
		}

		if !adminJobFound {
			t.Errorf("Admin override job should be ready for processing immediately")
		}

		// Step 5: Process the admin override job
		success, err := cm.ProcessJob(adminJobID, &userID)
		if err != nil {
			t.Fatalf("Failed to process admin job: %v", err)
		}

		if !success {
			t.Errorf("Admin job processing should succeed")
		}

		// Step 6: Verify article canonical URL was updated
		var canonicalURL sql.NullString
		err = db.QueryRow(`
			SELECT canonical_url FROM articles WHERE id = 1
		`).Scan(&canonicalURL)
		if err != nil {
			t.Fatalf("Failed to fetch article canonical URL: %v", err)
		}

		if !canonicalURL.Valid {
			t.Errorf("Canonical URL should be set after processing")
		}

		expectedURL := "/category/test-category"
		if canonicalURL.String != expectedURL {
			t.Errorf("Expected canonical URL %s, got %s", expectedURL, canonicalURL.String)
		}

		// Step 7: Test job cancellation
		err = cm.CancelJob(jobID)
		if err != nil {
			t.Fatalf("Failed to cancel job: %v", err)
		}

		// Step 8: Verify job was cancelled
		var status string
		err = db.QueryRow(`
			SELECT status FROM canonical_jobs WHERE id = $1
		`, jobID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to fetch job status: %v", err)
		}

		if status != "cancelled" {
			t.Errorf("Expected job status 'cancelled', got %s", status)
		}

		// Step 9: Test job statistics
		stats, err := cm.GetJobStats()
		if err != nil {
			t.Fatalf("Failed to get job stats: %v", err)
		}

		if stats["processed"] < 1 {
			t.Errorf("Expected at least 1 processed job in stats")
		}

		if stats["cancelled"] < 1 {
			t.Errorf("Expected at least 1 cancelled job in stats")
		}
	})
}

// TestCanonicalJobProcessorIntegration tests the background job processor
func TestCanonicalJobProcessorIntegration(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	t.Run("Background Job Processor", func(t *testing.T) {
		// Create multiple jobs with admin override for immediate processing
		jobIDs := make([]uint64, 3)
		for i := 0; i < 3; i++ {
			jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, true)
			if err != nil {
				t.Fatalf("Failed to schedule job %d: %v", i, err)
			}
			jobIDs[i] = jobID
		}

		// Process all pending jobs
		processed, err := cm.ProcessPendingJobs(&userID)
		if err != nil {
			t.Fatalf("Failed to process pending jobs: %v", err)
		}

		if processed < 3 {
			t.Errorf("Expected to process at least 3 jobs, got %d", processed)
		}

		// Verify all jobs were processed
		for _, jobID := range jobIDs {
			var status string
			err = db.QueryRow(`
				SELECT status FROM canonical_jobs WHERE id = $1
			`, jobID).Scan(&status)
			if err != nil {
				t.Fatalf("Failed to fetch job %d status: %v", jobID, err)
			}

			if status != "processed" {
				t.Errorf("Expected job %d status 'processed', got %s", jobID, status)
			}
		}
	})
}

// TestCanonicalErrorHandlingIntegration tests error handling scenarios
func TestCanonicalErrorHandlingIntegration(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, _ := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	t.Run("Error Handling Scenarios", func(t *testing.T) {
		// Test invalid tag ID
		jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(99999), &userID, true)
		if err != nil {
			t.Fatalf("Failed to schedule job with invalid tag: %v", err)
		}

		// Try to process the job - should fail
		success, err := cm.ProcessJob(jobID, &userID)
		if err != nil {
			t.Fatalf("Unexpected error processing invalid job: %v", err)
		}

		if success {
			t.Errorf("Job with invalid tag should fail processing")
		}

		// Verify job status is failed
		var status string
		var errorMessage sql.NullString
		var retryCount int
		err = db.QueryRow(`
			SELECT status, error_message, retry_count 
			FROM canonical_jobs WHERE id = $1
		`, jobID).Scan(&status, &errorMessage, &retryCount)
		if err != nil {
			t.Fatalf("Failed to fetch failed job details: %v", err)
		}

		if status != "failed" {
			t.Errorf("Expected job status 'failed', got %s", status)
		}

		if !errorMessage.Valid {
			t.Errorf("Expected error message to be set for failed job")
		}

		if retryCount != 1 {
			t.Errorf("Expected retry count 1, got %d", retryCount)
		}

		// Test job retry
		err = cm.RetryFailedJob(jobID, false)
		if err != nil {
			t.Fatalf("Failed to retry job: %v", err)
		}

		// Verify job is back to pending
		err = db.QueryRow(`
			SELECT status FROM canonical_jobs WHERE id = $1
		`, jobID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to fetch retried job status: %v", err)
		}

		if status != "pending" {
			t.Errorf("Expected retried job status 'pending', got %s", status)
		}
	})
}

// TestCanonicalHierarchicalCategoriesIntegration tests canonical URLs for hierarchical categories
func TestCanonicalHierarchicalCategoriesIntegration(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, parentCategoryID, _ := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	// Create child category
	var childCategoryID uint64
	err := db.QueryRow(`
		INSERT INTO categories (name, slug, description, parent_id)
		VALUES ('Child Category', 'child-category', 'Child category description', $1)
		RETURNING id
	`, parentCategoryID).Scan(&childCategoryID)
	if err != nil {
		t.Fatalf("Failed to create child category: %v", err)
	}

	cm := NewCanonicalManager(db)

	t.Run("Hierarchical Category Canonical URLs", func(t *testing.T) {
		// Test canonical URL generation for child category
		target := NewCategoryTarget(childCategoryID)
		canonicalURL, err := cm.GenerateCanonicalURL(target)
		if err != nil {
			t.Fatalf("Failed to generate canonical URL for child category: %v", err)
		}

		expectedURL := "/category/test-category/child-category"
		if canonicalURL != expectedURL {
			t.Errorf("Expected hierarchical canonical URL %s, got %s", expectedURL, canonicalURL)
		}

		// Schedule and process a job for the child category
		jobID, err := cm.ScheduleCanonicalJob(1, target, &userID, true)
		if err != nil {
			t.Fatalf("Failed to schedule job for child category: %v", err)
		}

		success, err := cm.ProcessJob(jobID, &userID)
		if err != nil {
			t.Fatalf("Failed to process child category job: %v", err)
		}

		if !success {
			t.Errorf("Child category job processing should succeed")
		}

		// Verify article canonical URL was updated with hierarchical path
		var articleCanonicalURL sql.NullString
		err = db.QueryRow(`
			SELECT canonical_url FROM articles WHERE id = 1
		`).Scan(&articleCanonicalURL)
		if err != nil {
			t.Fatalf("Failed to fetch article canonical URL: %v", err)
		}

		if !articleCanonicalURL.Valid {
			t.Errorf("Canonical URL should be set after processing")
		}

		if articleCanonicalURL.String != expectedURL {
			t.Errorf("Expected article canonical URL %s, got %s", expectedURL, articleCanonicalURL.String)
		}
	})
}

// TestCanonicalJobCleanupIntegration tests the cleanup functionality
func TestCanonicalJobCleanupIntegration(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	t.Run("Job Cleanup Integration", func(t *testing.T) {
		// Create and process several jobs
		jobIDs := make([]uint64, 5)
		for i := 0; i < 5; i++ {
			jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, true)
			if err != nil {
				t.Fatalf("Failed to schedule job %d: %v", i, err)
			}
			jobIDs[i] = jobID

			// Process the job
			success, err := cm.ProcessJob(jobID, &userID)
			if err != nil {
				t.Fatalf("Failed to process job %d: %v", i, err)
			}
			if !success {
				t.Errorf("Job %d processing should succeed", i)
			}
		}

		// Make jobs old by updating their timestamps
		for _, jobID := range jobIDs {
			_, err := db.Exec(`
				UPDATE canonical_jobs 
				SET updated_at = NOW() - INTERVAL '31 days'
				WHERE id = $1
			`, jobID)
			if err != nil {
				t.Fatalf("Failed to make job %d old: %v", jobID, err)
			}
		}

		// Run cleanup
		deletedCount, err := cm.CleanupOldJobs()
		if err != nil {
			t.Fatalf("Failed to cleanup old jobs: %v", err)
		}

		if deletedCount < 5 {
			t.Errorf("Expected to delete at least 5 jobs, got %d", deletedCount)
		}

		// Verify jobs were deleted
		for _, jobID := range jobIDs {
			var count int
			err = db.QueryRow(`SELECT COUNT(*) FROM canonical_jobs WHERE id = $1`, jobID).Scan(&count)
			if err != nil {
				t.Fatalf("Failed to check if job %d was deleted: %v", jobID, err)
			}

			if count != 0 {
				t.Errorf("Expected job %d to be deleted, but it still exists", jobID)
			}
		}
	})
}

// TestCanonicalJobProcessorStartup tests the background processor startup
func TestCanonicalJobProcessorStartup(t *testing.T) {
	db := setupCanonicalTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalTestData(t, db)
	defer cleanupCanonicalTestData(t, db)

	cm := NewCanonicalManager(db)

	t.Run("Background Processor Startup", func(t *testing.T) {
		// Create a job that will be processed by the background processor
		jobID, err := cm.ScheduleCanonicalJob(1, NewTagTarget(tagID), &userID, true)
		if err != nil {
			t.Fatalf("Failed to schedule job: %v", err)
		}

		// Start the background processor with a short interval
		cm.StartJobProcessor(100*time.Millisecond, &userID)

		// Wait a bit for the processor to run
		time.Sleep(200 * time.Millisecond)

		// Check if the job was processed
		var status string
		err = db.QueryRow(`
			SELECT status FROM canonical_jobs WHERE id = $1
		`, jobID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to fetch job status: %v", err)
		}

		if status != "processed" {
			t.Errorf("Expected job to be processed by background processor, got status %s", status)
		}
	})
}