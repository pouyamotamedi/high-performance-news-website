package services

import (
	"database/sql"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// BulkModerationService handles bulk moderation operations
type BulkModerationService struct {
	db                *sql.DB
	moderationService *ModerationService
}

// NewBulkModerationService creates a new bulk moderation service
func NewBulkModerationService(db *sql.DB, moderationService *ModerationService) *BulkModerationService {
	return &BulkModerationService{
		db:                db,
		moderationService: moderationService,
	}
}

// CreateBulkJob creates a new bulk moderation job
func (bms *BulkModerationService) CreateBulkJob(jobName, jobType string, criteria *models.BulkModerationCriteria, createdBy uint64) (*models.BulkModerationJob, error) {
	// Validate job type
	validJobTypes := map[string]bool{
		"bulk_approve":   true,
		"bulk_reject":    true,
		"bulk_ai_check":  true,
		"bulk_assign":    true,
		"bulk_flag":      true,
	}
	
	if !validJobTypes[jobType] {
		return nil, fmt.Errorf("invalid job type: %s", jobType)
	}

	// Count items that match criteria
	totalItems, err := bms.countMatchingItems(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to count matching items: %w", err)
	}

	if totalItems == 0 {
		return nil, fmt.Errorf("no items match the specified criteria")
	}

	// Create job record
	job := &models.BulkModerationJob{
		JobName:    jobName,
		JobType:    jobType,
		Criteria:   criteria,
		TotalItems: totalItems,
		Status:     "pending",
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}

	// Insert job
	err = bms.db.QueryRow(`
		INSERT INTO bulk_moderation_jobs (
			job_name, job_type, criteria, total_items, status, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		job.JobName, job.JobType, job.Criteria, job.TotalItems,
		job.Status, job.CreatedBy, job.CreatedAt,
	).Scan(&job.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create bulk job: %w", err)
	}

	return job, nil
}

// ExecuteBulkJob executes a bulk moderation job
func (bms *BulkModerationService) ExecuteBulkJob(jobID uint64) error {
	// Get job details
	job, err := bms.GetBulkJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status != "pending" {
		return fmt.Errorf("job %d is not in pending status", jobID)
	}

	// Update job status to running
	now := time.Now()
	_, err = bms.db.Exec(`
		UPDATE bulk_moderation_jobs SET status = 'running', started_at = $2 WHERE id = $1
	`, jobID, now)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Execute the job based on type
	var executionErr error
	switch job.JobType {
	case "bulk_approve":
		executionErr = bms.executeBulkApprove(job)
	case "bulk_reject":
		executionErr = bms.executeBulkReject(job)
	case "bulk_ai_check":
		executionErr = bms.executeBulkAICheck(job)
	case "bulk_assign":
		executionErr = bms.executeBulkAssign(job)
	case "bulk_flag":
		executionErr = bms.executeBulkFlag(job)
	default:
		executionErr = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	// Update job completion status
	completedAt := time.Now()
	status := "completed"
	errorLog := ""
	
	if executionErr != nil {
		status = "failed"
		errorLog = executionErr.Error()
	}

	_, err = bms.db.Exec(`
		UPDATE bulk_moderation_jobs SET 
			status = $2, completed_at = $3, error_log = $4
		WHERE id = $1
	`, jobID, status, completedAt, errorLog)
	
	if err != nil {
		return fmt.Errorf("failed to update job completion: %w", err)
	}

	return executionErr
}

// executeBulkApprove approves all items matching criteria
func (bms *BulkModerationService) executeBulkApprove(job *models.BulkModerationJob) error {
	items, err := bms.getMatchingItems(job.Criteria)
	if err != nil {
		return fmt.Errorf("failed to get matching items: %w", err)
	}

	processed := 0
	successful := 0
	failed := 0

	for _, item := range items {
		processed++
		
		err := bms.moderationService.ApproveContent(item.ID, job.CreatedBy, "Bulk approved", false)
		if err != nil {
			failed++
			// Log error but continue processing
			fmt.Printf("Failed to approve item %d: %v\n", item.ID, err)
		} else {
			successful++
		}

		// Update progress every 100 items
		if processed%100 == 0 {
			bms.updateJobProgress(job.ID, processed, successful, failed)
		}
	}

	// Final progress update
	return bms.updateJobProgress(job.ID, processed, successful, failed)
}

// executeBulkReject rejects all items matching criteria
func (bms *BulkModerationService) executeBulkReject(job *models.BulkModerationJob) error {
	items, err := bms.getMatchingItems(job.Criteria)
	if err != nil {
		return fmt.Errorf("failed to get matching items: %w", err)
	}

	processed := 0
	successful := 0
	failed := 0

	for _, item := range items {
		processed++
		
		err := bms.moderationService.RejectContent(item.ID, job.CreatedBy, "Bulk rejected", "Bulk rejection")
		if err != nil {
			failed++
			fmt.Printf("Failed to reject item %d: %v\n", item.ID, err)
		} else {
			successful++
		}

		// Update progress every 100 items
		if processed%100 == 0 {
			bms.updateJobProgress(job.ID, processed, successful, failed)
		}
	}

	// Final progress update
	return bms.updateJobProgress(job.ID, processed, successful, failed)
}

// executeBulkAICheck runs AI quality checks on all matching items
func (bms *BulkModerationService) executeBulkAICheck(job *models.BulkModerationJob) error {
	items, err := bms.getMatchingItems(job.Criteria)
	if err != nil {
		return fmt.Errorf("failed to get matching items: %w", err)
	}

	processed := 0
	successful := 0
	failed := 0

	for _, item := range items {
		processed++
		
		err := bms.moderationService.RunAIQualityCheck(item.ID)
		if err != nil {
			failed++
			fmt.Printf("Failed to run AI check on item %d: %v\n", item.ID, err)
		} else {
			successful++
		}

		// Update progress every 50 items (AI calls are slower)
		if processed%50 == 0 {
			bms.updateJobProgress(job.ID, processed, successful, failed)
		}
	}

	// Final progress update
	return bms.updateJobProgress(job.ID, processed, successful, failed)
}

// executeBulkAssign assigns moderator to all matching items
func (bms *BulkModerationService) executeBulkAssign(job *models.BulkModerationJob) error {
	// For bulk assign, we need the moderator ID from criteria
	// This would be passed in a custom field in the criteria
	items, err := bms.getMatchingItems(job.Criteria)
	if err != nil {
		return fmt.Errorf("failed to get matching items: %w", err)
	}

	processed := 0
	successful := 0
	failed := 0

	// Extract moderator ID from criteria (this would be a custom implementation)
	// For now, we'll assign to the job creator
	moderatorID := job.CreatedBy

	for _, item := range items {
		processed++
		
		err := bms.moderationService.AssignModerator(item.ID, moderatorID, job.CreatedBy)
		if err != nil {
			failed++
			fmt.Printf("Failed to assign moderator to item %d: %v\n", item.ID, err)
		} else {
			successful++
		}

		// Update progress every 100 items
		if processed%100 == 0 {
			bms.updateJobProgress(job.ID, processed, successful, failed)
		}
	}

	// Final progress update
	return bms.updateJobProgress(job.ID, processed, successful, failed)
}

// executeBulkFlag flags all matching items
func (bms *BulkModerationService) executeBulkFlag(job *models.BulkModerationJob) error {
	items, err := bms.getMatchingItems(job.Criteria)
	if err != nil {
		return fmt.Errorf("failed to get matching items: %w", err)
	}

	processed := 0
	successful := 0
	failed := 0

	for _, item := range items {
		processed++
		
		err := bms.moderationService.FlagContent(item.ID, job.CreatedBy, "Bulk flagged for review")
		if err != nil {
			failed++
			fmt.Printf("Failed to flag item %d: %v\n", item.ID, err)
		} else {
			successful++
		}

		// Update progress every 100 items
		if processed%100 == 0 {
			bms.updateJobProgress(job.ID, processed, successful, failed)
		}
	}

	// Final progress update
	return bms.updateJobProgress(job.ID, processed, successful, failed)
}

// updateJobProgress updates job progress in database
func (bms *BulkModerationService) updateJobProgress(jobID uint64, processed, successful, failed int) error {
	_, err := bms.db.Exec(`
		UPDATE bulk_moderation_jobs SET
			processed_items = $2,
			successful_items = $3,
			failed_items = $4
		WHERE id = $1
	`, jobID, processed, successful, failed)
	
	return err
}

// countMatchingItems counts items that match the criteria
func (bms *BulkModerationService) countMatchingItems(criteria *models.BulkModerationCriteria) (int, error) {
	whereClause, args := bms.buildCriteriaQuery(criteria)
	
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM moderation_queue mq %s
	`, whereClause)
	
	var count int
	err := bms.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count items: %w", err)
	}
	
	return count, nil
}

// getMatchingItems gets items that match the criteria
func (bms *BulkModerationService) getMatchingItems(criteria *models.BulkModerationCriteria) ([]models.ModerationQueue, error) {
	whereClause, args := bms.buildCriteriaQuery(criteria)
	
	query := fmt.Sprintf(`
		SELECT id, article_id, article_version_id, content_type, status,
			   priority, submitted_by, assigned_to, ai_quality_score,
			   ai_feedback, moderator_notes, rejection_reason,
			   auto_approved, submitted_at, reviewed_at, reviewed_by
		FROM moderation_queue mq
		%s
		ORDER BY priority DESC, submitted_at ASC
	`, whereClause)
	
	rows, err := bms.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get matching items: %w", err)
	}
	defer rows.Close()

	var items []models.ModerationQueue
	for rows.Next() {
		var item models.ModerationQueue
		err := rows.Scan(
			&item.ID, &item.ArticleID, &item.ArticleVersionID, &item.ContentType,
			&item.Status, &item.Priority, &item.SubmittedBy, &item.AssignedTo,
			&item.AIQualityScore, &item.AIFeedback, &item.ModeratorNotes,
			&item.RejectionReason, &item.AutoApproved, &item.SubmittedAt,
			&item.ReviewedAt, &item.ReviewedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	return items, nil
}

// buildCriteriaQuery builds WHERE clause for criteria
func (bms *BulkModerationService) buildCriteriaQuery(criteria *models.BulkModerationCriteria) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if len(criteria.Status) > 0 {
		placeholders := make([]string, len(criteria.Status))
		for i, status := range criteria.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.status IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if len(criteria.Priority) > 0 {
		placeholders := make([]string, len(criteria.Priority))
		for i, priority := range criteria.Priority {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, priority)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.priority IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if criteria.SubmittedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("mq.submitted_at >= $%d", argIndex))
		args = append(args, *criteria.SubmittedAfter)
		argIndex++
	}

	if criteria.SubmittedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("mq.submitted_at <= $%d", argIndex))
		args = append(args, *criteria.SubmittedBefore)
		argIndex++
	}

	if criteria.AIQualityMin != nil {
		conditions = append(conditions, fmt.Sprintf("mq.ai_quality_score >= $%d", argIndex))
		args = append(args, *criteria.AIQualityMin)
		argIndex++
	}

	if criteria.AIQualityMax != nil {
		conditions = append(conditions, fmt.Sprintf("mq.ai_quality_score <= $%d", argIndex))
		args = append(args, *criteria.AIQualityMax)
		argIndex++
	}

	if len(criteria.ContentType) > 0 {
		placeholders := make([]string, len(criteria.ContentType))
		for i, contentType := range criteria.ContentType {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, contentType)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.content_type IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if len(criteria.SubmittedBy) > 0 {
		placeholders := make([]string, len(criteria.SubmittedBy))
		for i, userID := range criteria.SubmittedBy {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, userID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.submitted_by IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if len(criteria.AssignedTo) > 0 {
		placeholders := make([]string, len(criteria.AssignedTo))
		for i, userID := range criteria.AssignedTo {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, userID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.assigned_to IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("%s", conditions)
	}

	return whereClause, args
}

// GetBulkJob returns a bulk moderation job by ID
func (bms *BulkModerationService) GetBulkJob(jobID uint64) (*models.BulkModerationJob, error) {
	var job models.BulkModerationJob
	err := bms.db.QueryRow(`
		SELECT id, job_name, job_type, criteria, total_items, processed_items,
			   successful_items, failed_items, status, created_by,
			   started_at, completed_at, created_at, error_log
		FROM bulk_moderation_jobs
		WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.JobName, &job.JobType, &job.Criteria, &job.TotalItems,
		&job.ProcessedItems, &job.SuccessfulItems, &job.FailedItems,
		&job.Status, &job.CreatedBy, &job.StartedAt, &job.CompletedAt,
		&job.CreatedAt, &job.ErrorLog,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bulk job %d not found", jobID)
		}
		return nil, fmt.Errorf("failed to get bulk job: %w", err)
	}

	return &job, nil
}

// GetBulkJobs returns a list of bulk moderation jobs
func (bms *BulkModerationService) GetBulkJobs(limit, offset int) ([]models.BulkModerationJob, int, error) {
	// Get total count
	var totalCount int
	err := bms.db.QueryRow(`SELECT COUNT(*) FROM bulk_moderation_jobs`).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get jobs
	rows, err := bms.db.Query(`
		SELECT id, job_name, job_type, criteria, total_items, processed_items,
			   successful_items, failed_items, status, created_by,
			   started_at, completed_at, created_at, error_log
		FROM bulk_moderation_jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get bulk jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.BulkModerationJob
	for rows.Next() {
		var job models.BulkModerationJob
		err := rows.Scan(
			&job.ID, &job.JobName, &job.JobType, &job.Criteria, &job.TotalItems,
			&job.ProcessedItems, &job.SuccessfulItems, &job.FailedItems,
			&job.Status, &job.CreatedBy, &job.StartedAt, &job.CompletedAt,
			&job.CreatedAt, &job.ErrorLog,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan bulk job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating bulk jobs: %w", err)
	}

	return jobs, totalCount, nil
}

// CancelBulkJob cancels a running bulk job
func (bms *BulkModerationService) CancelBulkJob(jobID uint64) error {
	_, err := bms.db.Exec(`
		UPDATE bulk_moderation_jobs SET 
			status = 'cancelled', 
			completed_at = NOW(),
			error_log = 'Job cancelled by user'
		WHERE id = $1 AND status = 'running'
	`, jobID)
	
	if err != nil {
		return fmt.Errorf("failed to cancel bulk job: %w", err)
	}
	
	return nil
}