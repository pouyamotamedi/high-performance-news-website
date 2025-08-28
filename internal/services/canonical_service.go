package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// CanonicalJob represents a canonicalization job
type CanonicalJob struct {
	ID            uint64     `json:"id" db:"id"`
	ArticleID     uint64     `json:"article_id" db:"article_id"`
	TargetType    string     `json:"target_type" db:"target_type"`
	TargetID      *uint64    `json:"target_id" db:"target_id"`
	TargetURL     *string    `json:"target_url" db:"target_url"`
	ScheduledAt   time.Time  `json:"scheduled_at" db:"scheduled_at"`
	ProcessedAt   *time.Time `json:"processed_at" db:"processed_at"`
	Status        string     `json:"status" db:"status"`
	AdminOverride bool       `json:"admin_override" db:"admin_override"`
	CreatedBy     *uint64    `json:"created_by" db:"created_by"`
	ProcessedBy   *uint64    `json:"processed_by" db:"processed_by"`
	ErrorMessage  *string    `json:"error_message" db:"error_message"`
	RetryCount    int        `json:"retry_count" db:"retry_count"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`

	// Computed fields from view
	ArticleTitle *string `json:"article_title,omitempty"`
	ArticleSlug  *string `json:"article_slug,omitempty"`
	TargetName   *string `json:"target_name,omitempty"`
	TargetSlug   *string `json:"target_slug,omitempty"`
}

// CanonicalTarget represents a canonicalization target
type CanonicalTarget struct {
	Type string  `json:"type"` // "tag", "category", "url"
	ID   *uint64 `json:"id,omitempty"`
	URL  *string `json:"url,omitempty"`
}

// CanonicalManager handles delayed canonicalization processing
type CanonicalManager struct {
	db *sql.DB
}

// NewCanonicalManager creates a new canonical manager instance
func NewCanonicalManager(db *sql.DB) *CanonicalManager {
	return &CanonicalManager{
		db: db,
	}
}

// ScheduleCanonicalJob schedules a canonicalization job with 48-hour delay
func (cm *CanonicalManager) ScheduleCanonicalJob(articleID uint64, target CanonicalTarget, createdBy *uint64, adminOverride bool) (uint64, error) {
	// Validate target
	if err := cm.validateTarget(target); err != nil {
		return 0, fmt.Errorf("invalid target: %w", err)
	}

	// Call database function to schedule the job
	var jobID uint64
	err := cm.db.QueryRow(`
		SELECT schedule_canonical_job($1, $2, $3, $4, $5, $6)
	`, articleID, target.Type, target.ID, target.URL, createdBy, adminOverride).Scan(&jobID)

	if err != nil {
		return 0, fmt.Errorf("failed to schedule canonical job: %w", err)
	}

	log.Printf("Scheduled canonical job %d for article %d (type: %s, admin_override: %t)", 
		jobID, articleID, target.Type, adminOverride)

	return jobID, nil
}

// ProcessPendingJobs processes all pending canonical jobs that are ready
func (cm *CanonicalManager) ProcessPendingJobs(processorUserID *uint64) (int, error) {
	// Get all pending jobs ready for processing
	jobs, err := cm.GetPendingJobs()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	processed := 0
	for _, job := range jobs {
		success, err := cm.ProcessJob(job.ID, processorUserID)
		if err != nil {
			log.Printf("Error processing canonical job %d: %v", job.ID, err)
			continue
		}
		if success {
			processed++
			log.Printf("Successfully processed canonical job %d for article %d", job.ID, job.ArticleID)
		}
	}

	return processed, nil
}

// ProcessJob processes a single canonical job
func (cm *CanonicalManager) ProcessJob(jobID uint64, processorUserID *uint64) (bool, error) {
	var success bool
	err := cm.db.QueryRow(`
		SELECT process_canonical_job($1, $2)
	`, jobID, processorUserID).Scan(&success)

	if err != nil {
		return false, fmt.Errorf("failed to process canonical job %d: %w", jobID, err)
	}

	return success, nil
}

// GetPendingJobs returns all pending canonical jobs ready for processing
func (cm *CanonicalManager) GetPendingJobs() ([]CanonicalJob, error) {
	rows, err := cm.db.Query(`
		SELECT id, article_id, target_type, target_id, target_url, 
		       scheduled_at, processed_at, status, admin_override, 
		       created_by, processed_by, error_message, retry_count,
		       created_at, updated_at, article_title, article_slug,
		       target_name, target_slug
		FROM pending_canonical_jobs
		ORDER BY admin_override DESC, scheduled_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []CanonicalJob
	for rows.Next() {
		var job CanonicalJob
		err := rows.Scan(
			&job.ID, &job.ArticleID, &job.TargetType, &job.TargetID, &job.TargetURL,
			&job.ScheduledAt, &job.ProcessedAt, &job.Status, &job.AdminOverride,
			&job.CreatedBy, &job.ProcessedBy, &job.ErrorMessage, &job.RetryCount,
			&job.CreatedAt, &job.UpdatedAt, &job.ArticleTitle, &job.ArticleSlug,
			&job.TargetName, &job.TargetSlug,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// GetJobsByArticle returns all canonical jobs for a specific article
func (cm *CanonicalManager) GetJobsByArticle(articleID uint64) ([]CanonicalJob, error) {
	rows, err := cm.db.Query(`
		SELECT id, article_id, target_type, target_id, target_url, 
		       scheduled_at, processed_at, status, admin_override, 
		       created_by, processed_by, error_message, retry_count,
		       created_at, updated_at
		FROM canonical_jobs
		WHERE article_id = $1
		ORDER BY created_at DESC
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs for article %d: %w", articleID, err)
	}
	defer rows.Close()

	var jobs []CanonicalJob
	for rows.Next() {
		var job CanonicalJob
		err := rows.Scan(
			&job.ID, &job.ArticleID, &job.TargetType, &job.TargetID, &job.TargetURL,
			&job.ScheduledAt, &job.ProcessedAt, &job.Status, &job.AdminOverride,
			&job.CreatedBy, &job.ProcessedBy, &job.ErrorMessage, &job.RetryCount,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// CancelJob cancels a pending canonical job
func (cm *CanonicalManager) CancelJob(jobID uint64) error {
	result, err := cm.db.Exec(`
		UPDATE canonical_jobs 
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, jobID)
	if err != nil {
		return fmt.Errorf("failed to cancel job %d: %w", jobID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job %d not found or not in pending status", jobID)
	}

	log.Printf("Cancelled canonical job %d", jobID)
	return nil
}

// RetryFailedJob retries a failed canonical job
func (cm *CanonicalManager) RetryFailedJob(jobID uint64, adminOverride bool) error {
	// Check if job exists and is in failed status
	var currentStatus string
	var retryCount int
	err := cm.db.QueryRow(`
		SELECT status, retry_count 
		FROM canonical_jobs 
		WHERE id = $1
	`, jobID).Scan(&currentStatus, &retryCount)

	if err == sql.ErrNoRows {
		return fmt.Errorf("job %d not found", jobID)
	}
	if err != nil {
		return fmt.Errorf("failed to check job status: %w", err)
	}

	if currentStatus != "failed" {
		return fmt.Errorf("job %d is not in failed status (current: %s)", jobID, currentStatus)
	}

	if retryCount >= 3 && !adminOverride {
		return fmt.Errorf("job %d has exceeded maximum retry attempts (3)", jobID)
	}

	// Reset job to pending status
	scheduledAt := time.Now()
	if !adminOverride {
		scheduledAt = scheduledAt.Add(48 * time.Hour)
	}

	_, err = cm.db.Exec(`
		UPDATE canonical_jobs 
		SET status = 'pending', 
		    scheduled_at = $2,
		    admin_override = $3,
		    error_message = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, jobID, scheduledAt, adminOverride)

	if err != nil {
		return fmt.Errorf("failed to retry job %d: %w", jobID, err)
	}

	log.Printf("Retried canonical job %d (admin_override: %t)", jobID, adminOverride)
	return nil
}

// GenerateCanonicalURL generates a canonical URL for the given target
func (cm *CanonicalManager) GenerateCanonicalURL(target CanonicalTarget) (string, error) {
	if err := cm.validateTarget(target); err != nil {
		return "", fmt.Errorf("invalid target: %w", err)
	}

	var canonicalURL string
	err := cm.db.QueryRow(`
		SELECT generate_canonical_url($1, $2, $3)
	`, target.Type, target.ID, target.URL).Scan(&canonicalURL)

	if err != nil {
		return "", fmt.Errorf("failed to generate canonical URL: %w", err)
	}

	return canonicalURL, nil
}

// CleanupOldJobs removes old processed/cancelled/failed jobs
func (cm *CanonicalManager) CleanupOldJobs() (int, error) {
	var deletedCount int
	err := cm.db.QueryRow(`SELECT cleanup_old_canonical_jobs()`).Scan(&deletedCount)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old jobs: %w", err)
	}

	log.Printf("Cleaned up %d old canonical jobs", deletedCount)
	return deletedCount, nil
}

// GetJobStats returns statistics about canonical jobs
func (cm *CanonicalManager) GetJobStats() (map[string]int, error) {
	rows, err := cm.db.Query(`
		SELECT status, COUNT(*) as count
		FROM canonical_jobs
		WHERE created_at >= NOW() - INTERVAL '30 days'
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats row: %w", err)
		}
		stats[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stats rows: %w", err)
	}

	return stats, nil
}

// validateTarget validates a canonical target
func (cm *CanonicalManager) validateTarget(target CanonicalTarget) error {
	switch target.Type {
	case "tag":
		if target.ID == nil {
			return fmt.Errorf("target_id is required for tag type")
		}
		if target.URL != nil {
			return fmt.Errorf("target_url should not be set for tag type")
		}
	case "category":
		if target.ID == nil {
			return fmt.Errorf("target_id is required for category type")
		}
		if target.URL != nil {
			return fmt.Errorf("target_url should not be set for category type")
		}
	case "url":
		if target.URL == nil || strings.TrimSpace(*target.URL) == "" {
			return fmt.Errorf("target_url is required for url type")
		}
		if target.ID != nil {
			return fmt.Errorf("target_id should not be set for url type")
		}
	default:
		return fmt.Errorf("invalid target type: %s (must be tag, category, or url)", target.Type)
	}
	return nil
}

// StartJobProcessor starts a background job processor
func (cm *CanonicalManager) StartJobProcessor(interval time.Duration, processorUserID *uint64) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			processed, err := cm.ProcessPendingJobs(processorUserID)
			if err != nil {
				log.Printf("Error processing canonical jobs: %v", err)
			} else if processed > 0 {
				log.Printf("Processed %d canonical jobs", processed)
			}
		}
	}()
	log.Printf("Started canonical job processor with %v interval", interval)
}

// Helper functions for creating targets

// NewTagTarget creates a canonical target for a tag
func NewTagTarget(tagID uint64) CanonicalTarget {
	return CanonicalTarget{
		Type: "tag",
		ID:   &tagID,
	}
}

// NewCategoryTarget creates a canonical target for a category
func NewCategoryTarget(categoryID uint64) CanonicalTarget {
	return CanonicalTarget{
		Type: "category",
		ID:   &categoryID,
	}
}

// NewURLTarget creates a canonical target for a custom URL
func NewURLTarget(url string) CanonicalTarget {
	return CanonicalTarget{
		Type: "url",
		URL:  &url,
	}
}