package services

import (
	"database/sql"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// ModerationService handles content moderation workflow
type ModerationService struct {
	db        *sql.DB
	aiService AIService
}

// NewModerationService creates a new moderation service
func NewModerationService(db *sql.DB, aiService AIService) *ModerationService {
	return &ModerationService{
		db:        db,
		aiService: aiService,
	}
}

// SubmitForModeration submits content for moderation review
func (ms *ModerationService) SubmitForModeration(articleID uint64, contentType string, priority int, submittedBy uint64) (*models.ModerationQueue, error) {
	// Check if already in moderation queue
	var existingID uint64
	err := ms.db.QueryRow(`
		SELECT id FROM moderation_queue 
		WHERE article_id = $1 AND status = 'pending'
	`, articleID).Scan(&existingID)
	
	if err == nil {
		return nil, fmt.Errorf("article %d is already in moderation queue", articleID)
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing moderation: %w", err)
	}

	// Create moderation queue entry
	moderationItem := &models.ModerationQueue{
		ArticleID:   articleID,
		ContentType: contentType,
		Status:      models.ModerationStatusPending,
		Priority:    priority,
		SubmittedBy: submittedBy,
		SubmittedAt: time.Now(),
	}

	// Validate the moderation item
	if err := models.ValidateModerationQueue(moderationItem); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Insert into database
	err = ms.db.QueryRow(`
		INSERT INTO moderation_queue (
			article_id, content_type, status, priority, submitted_by, submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`,
		moderationItem.ArticleID, moderationItem.ContentType, moderationItem.Status,
		moderationItem.Priority, moderationItem.SubmittedBy, moderationItem.SubmittedAt,
	).Scan(&moderationItem.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to submit for moderation: %w", err)
	}

	// Trigger AI quality check asynchronously
	go func() {
		if err := ms.RunAIQualityCheck(moderationItem.ID); err != nil {
			// Log error but don't fail the submission
			fmt.Printf("AI quality check failed for moderation item %d: %v\n", moderationItem.ID, err)
		}
	}()

	return moderationItem, nil
}

// RunAIQualityCheck performs AI quality analysis on content
func (ms *ModerationService) RunAIQualityCheck(moderationID uint64) error {
	// Get moderation item and article content
	var articleID uint64
	var contentType string
	err := ms.db.QueryRow(`
		SELECT article_id, content_type FROM moderation_queue WHERE id = $1
	`, moderationID).Scan(&articleID, &contentType)
	if err != nil {
		return fmt.Errorf("failed to get moderation item: %w", err)
	}

	// Get article content
	var title, content string
	err = ms.db.QueryRow(`
		SELECT title, content FROM articles WHERE id = $1
	`, articleID).Scan(&title, &content)
	if err != nil {
		return fmt.Errorf("failed to get article content: %w", err)
	}

	// Run AI analysis
	startTime := time.Now()
	feedback, err := ms.aiService.AnalyzeContent(title, content)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}
	processingTime := int(time.Since(startTime).Milliseconds())

	// Store quality check results
	_, err = ms.db.Exec(`
		INSERT INTO content_quality_checks (
			article_id, ai_provider, quality_score, grammar_score,
			readability_score, appropriateness_score, issues_found,
			suggestions, flagged_content, processing_time_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		articleID, feedback.Provider, feedback.QualityScore, feedback.GrammarScore,
		feedback.ReadabilityScore, feedback.AppropriatenessScore,
		models.AIIssues(feedback.Issues), models.AISuggestions(feedback.Suggestions),
		models.AIFlaggedContents(feedback.FlaggedContent), processingTime,
	)
	if err != nil {
		return fmt.Errorf("failed to store quality check: %w", err)
	}

	// Update moderation queue with AI feedback
	_, err = ms.db.Exec(`
		UPDATE moderation_queue SET
			ai_quality_score = $2,
			ai_feedback = $3
		WHERE id = $1
	`, moderationID, feedback.QualityScore, feedback)
	if err != nil {
		return fmt.Errorf("failed to update moderation queue: %w", err)
	}

	// Auto-approve if quality score is high enough and no high-severity issues
	if ms.shouldAutoApprove(feedback) {
		return ms.ApproveContent(moderationID, 0, "Auto-approved by AI", true)
	}

	return nil
}

// shouldAutoApprove determines if content should be auto-approved
func (ms *ModerationService) shouldAutoApprove(feedback *models.AIFeedback) bool {
	// Auto-approve threshold
	const autoApproveThreshold = 0.85

	// Don't auto-approve if quality score is too low
	if feedback.QualityScore < autoApproveThreshold {
		return false
	}

	// Don't auto-approve if there are high-severity issues
	for _, issue := range feedback.Issues {
		if issue.Severity == "high" {
			return false
		}
	}

	// Don't auto-approve if content is flagged
	if len(feedback.FlaggedContent) > 0 {
		return false
	}

	return true
}

// GetModerationQueue returns items in the moderation queue with filtering
func (ms *ModerationService) GetModerationQueue(filters ModerationFilters, limit, offset int) ([]models.ModerationQueue, int, error) {
	// Build WHERE clause
	whereClause, args := ms.buildModerationFilters(filters)
	
	// Get total count
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM moderation_queue mq
		LEFT JOIN users u ON mq.submitted_by = u.id
		%s
	`, whereClause)
	
	err := ms.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get items
	query := fmt.Sprintf(`
		SELECT mq.id, mq.article_id, mq.article_version_id, mq.content_type,
			   mq.status, mq.priority, mq.submitted_by, mq.assigned_to,
			   mq.ai_quality_score, mq.ai_feedback, mq.moderator_notes,
			   mq.rejection_reason, mq.auto_approved, mq.submitted_at,
			   mq.reviewed_at, mq.reviewed_by
		FROM moderation_queue mq
		LEFT JOIN users u ON mq.submitted_by = u.id
		%s
		ORDER BY mq.priority DESC, mq.submitted_at ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)
	
	args = append(args, limit, offset)
	
	rows, err := ms.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get moderation queue: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan moderation item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating moderation queue: %w", err)
	}

	return items, totalCount, nil
}

// buildModerationFilters builds WHERE clause for moderation queue filtering
func (ms *ModerationService) buildModerationFilters(filters ModerationFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if len(filters.Status) > 0 {
		placeholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.status IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if len(filters.Priority) > 0 {
		placeholders := make([]string, len(filters.Priority))
		for i, priority := range filters.Priority {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, priority)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("mq.priority IN (%s)", 
			fmt.Sprintf("%s", placeholders)))
	}

	if filters.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("mq.assigned_to = $%d", argIndex))
		args = append(args, *filters.AssignedTo)
		argIndex++
	}

	if filters.SubmittedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("mq.submitted_at >= $%d", argIndex))
		args = append(args, *filters.SubmittedAfter)
		argIndex++
	}

	if filters.SubmittedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("mq.submitted_at <= $%d", argIndex))
		args = append(args, *filters.SubmittedBefore)
		argIndex++
	}

	if filters.AIQualityMin != nil {
		conditions = append(conditions, fmt.Sprintf("mq.ai_quality_score >= $%d", argIndex))
		args = append(args, *filters.AIQualityMin)
		argIndex++
	}

	if filters.AIQualityMax != nil {
		conditions = append(conditions, fmt.Sprintf("mq.ai_quality_score <= $%d", argIndex))
		args = append(args, *filters.AIQualityMax)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("%s", conditions)
	}

	return whereClause, args
}

// AssignModerator assigns a moderator to a moderation item
func (ms *ModerationService) AssignModerator(moderationID, moderatorID, assignedBy uint64) error {
	// Update assignment
	_, err := ms.db.Exec(`
		UPDATE moderation_queue SET assigned_to = $2 WHERE id = $1
	`, moderationID, moderatorID)
	if err != nil {
		return fmt.Errorf("failed to assign moderator: %w", err)
	}

	// Log the action
	return ms.logModerationAction(moderationID, "assign", assignedBy, 
		fmt.Sprintf("Assigned to moderator %d", moderatorID), "", "pending")
}

// ApproveContent approves content in moderation
func (ms *ModerationService) ApproveContent(moderationID, reviewedBy uint64, notes string, autoApproved bool) error {
	now := time.Now()
	
	// Update moderation status
	_, err := ms.db.Exec(`
		UPDATE moderation_queue SET
			status = 'approved',
			moderator_notes = $2,
			auto_approved = $3,
			reviewed_at = $4,
			reviewed_by = $5
		WHERE id = $1
	`, moderationID, notes, autoApproved, now, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to approve content: %w", err)
	}

	// Update article moderation status
	_, err = ms.db.Exec(`
		UPDATE articles SET
			moderation_status = 'approved',
			moderation_notes = $2,
			last_moderated_at = $3,
			last_moderated_by = $4
		WHERE id = (SELECT article_id FROM moderation_queue WHERE id = $1)
	`, moderationID, notes, now, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to update article moderation status: %w", err)
	}

	// Log the action
	action := "approve"
	if autoApproved {
		action = "auto_approve"
	}
	return ms.logModerationAction(moderationID, action, reviewedBy, notes, "pending", "approved")
}

// RejectContent rejects content in moderation
func (ms *ModerationService) RejectContent(moderationID, reviewedBy uint64, reason, notes string) error {
	now := time.Now()
	
	// Update moderation status
	_, err := ms.db.Exec(`
		UPDATE moderation_queue SET
			status = 'rejected',
			rejection_reason = $2,
			moderator_notes = $3,
			reviewed_at = $4,
			reviewed_by = $5
		WHERE id = $1
	`, moderationID, reason, notes, now, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to reject content: %w", err)
	}

	// Update article moderation status
	_, err = ms.db.Exec(`
		UPDATE articles SET
			moderation_status = 'rejected',
			moderation_notes = $2,
			last_moderated_at = $3,
			last_moderated_by = $4
		WHERE id = (SELECT article_id FROM moderation_queue WHERE id = $1)
	`, moderationID, fmt.Sprintf("%s: %s", reason, notes), now, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to update article moderation status: %w", err)
	}

	// Log the action
	return ms.logModerationAction(moderationID, "reject", reviewedBy, 
		fmt.Sprintf("%s: %s", reason, notes), "pending", "rejected")
}

// FlagContent flags content for additional review
func (ms *ModerationService) FlagContent(moderationID, reviewedBy uint64, reason string) error {
	// Update moderation status
	_, err := ms.db.Exec(`
		UPDATE moderation_queue SET
			status = 'flagged',
			moderator_notes = $2,
			priority = GREATEST(priority, 3)
		WHERE id = $1
	`, moderationID, reason)
	if err != nil {
		return fmt.Errorf("failed to flag content: %w", err)
	}

	// Log the action
	return ms.logModerationAction(moderationID, "flag", reviewedBy, reason, "pending", "flagged")
}

// logModerationAction logs a moderation action for audit trail
func (ms *ModerationService) logModerationAction(moderationID uint64, action string, performedBy uint64, notes, previousStatus, newStatus string) error {
	_, err := ms.db.Exec(`
		INSERT INTO moderation_actions (
			moderation_queue_id, action, performed_by, notes,
			previous_status, new_status
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, moderationID, action, performedBy, notes, previousStatus, newStatus)
	
	if err != nil {
		return fmt.Errorf("failed to log moderation action: %w", err)
	}
	
	return nil
}

// GetModerationStats returns moderation statistics
func (ms *ModerationService) GetModerationStats() (*ModerationStats, error) {
	var stats ModerationStats

	// Get pending count
	err := ms.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_queue WHERE status = 'pending'
	`).Scan(&stats.PendingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	// Get approved count (last 24 hours)
	err = ms.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_queue 
		WHERE status = 'approved' AND reviewed_at >= NOW() - INTERVAL '24 hours'
	`).Scan(&stats.ApprovedLast24h)
	if err != nil {
		return nil, fmt.Errorf("failed to get approved count: %w", err)
	}

	// Get rejected count (last 24 hours)
	err = ms.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_queue 
		WHERE status = 'rejected' AND reviewed_at >= NOW() - INTERVAL '24 hours'
	`).Scan(&stats.RejectedLast24h)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected count: %w", err)
	}

	// Get auto-approved count (last 24 hours)
	err = ms.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_queue 
		WHERE auto_approved = true AND reviewed_at >= NOW() - INTERVAL '24 hours'
	`).Scan(&stats.AutoApprovedLast24h)
	if err != nil {
		return nil, fmt.Errorf("failed to get auto-approved count: %w", err)
	}

	// Get average processing time
	err = ms.db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (reviewed_at - submitted_at))/60)::DECIMAL(10,2)
		FROM moderation_queue 
		WHERE reviewed_at IS NOT NULL AND reviewed_at >= NOW() - INTERVAL '7 days'
	`).Scan(&stats.AvgProcessingTimeMinutes)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get average processing time: %w", err)
	}

	// Get high priority count
	err = ms.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_queue WHERE priority >= 3 AND status = 'pending'
	`).Scan(&stats.HighPriorityCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get high priority count: %w", err)
	}

	return &stats, nil
}

// ModerationFilters represents filters for moderation queue queries
type ModerationFilters struct {
	Status           []string   `json:"status,omitempty"`
	Priority         []int      `json:"priority,omitempty"`
	AssignedTo       *uint64    `json:"assigned_to,omitempty"`
	SubmittedAfter   *time.Time `json:"submitted_after,omitempty"`
	SubmittedBefore  *time.Time `json:"submitted_before,omitempty"`
	AIQualityMin     *float64   `json:"ai_quality_min,omitempty"`
	AIQualityMax     *float64   `json:"ai_quality_max,omitempty"`
}

// ModerationStats represents moderation statistics
type ModerationStats struct {
	PendingCount              int     `json:"pending_count"`
	ApprovedLast24h           int     `json:"approved_last_24h"`
	RejectedLast24h           int     `json:"rejected_last_24h"`
	AutoApprovedLast24h       int     `json:"auto_approved_last_24h"`
	HighPriorityCount         int     `json:"high_priority_count"`
	AvgProcessingTimeMinutes  float64 `json:"avg_processing_time_minutes"`
}