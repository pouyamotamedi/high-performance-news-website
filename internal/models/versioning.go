package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ArticleVersion represents a version of an article for history tracking
type ArticleVersion struct {
	ID                 uint64     `json:"id" db:"id"`
	ArticleID          uint64     `json:"article_id" db:"article_id"`
	VersionNumber      int        `json:"version_number" db:"version_number"`
	Title              string     `json:"title" db:"title"`
	Slug               string     `json:"slug" db:"slug"`
	Content            string     `json:"content" db:"content"`
	Excerpt            string     `json:"excerpt" db:"excerpt"`
	AuthorID           uint64     `json:"author_id" db:"author_id"`
	CategoryID         uint64     `json:"category_id" db:"category_id"`
	Status             string     `json:"status" db:"status"`
	PublishedAt        *time.Time `json:"published_at" db:"published_at"`
	MetaTitle          string     `json:"meta_title" db:"meta_title"`
	MetaDescription    string     `json:"meta_description" db:"meta_description"`
	CanonicalURL       string     `json:"canonical_url" db:"canonical_url"`
	SchemaType         string     `json:"schema_type" db:"schema_type"`
	LanguageCode       string     `json:"language_code" db:"language_code"`
	TranslationGroupID *uint64    `json:"translation_group_id" db:"translation_group_id"`
	AutoLinking        bool       `json:"auto_linking" db:"auto_linking"`
	ChangeSummary      string     `json:"change_summary" db:"change_summary"`
	CreatedBy          uint64     `json:"created_by" db:"created_by"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

// ModerationQueue represents an item in the content moderation queue
type ModerationQueue struct {
	ID               uint64                 `json:"id" db:"id"`
	ArticleID        uint64                 `json:"article_id" db:"article_id"`
	ArticleVersionID *uint64                `json:"article_version_id" db:"article_version_id"`
	ContentType      string                 `json:"content_type" db:"content_type"`
	Status           ModerationStatus       `json:"status" db:"status"`
	Priority         int                    `json:"priority" db:"priority"`
	SubmittedBy      uint64                 `json:"submitted_by" db:"submitted_by"`
	AssignedTo       *uint64                `json:"assigned_to" db:"assigned_to"`
	AIQualityScore   *float64               `json:"ai_quality_score" db:"ai_quality_score"`
	AIFeedback       *AIFeedback            `json:"ai_feedback" db:"ai_feedback"`
	ModeratorNotes   string                 `json:"moderator_notes" db:"moderator_notes"`
	RejectionReason  string                 `json:"rejection_reason" db:"rejection_reason"`
	AutoApproved     bool                   `json:"auto_approved" db:"auto_approved"`
	SubmittedAt      time.Time              `json:"submitted_at" db:"submitted_at"`
	ReviewedAt       *time.Time             `json:"reviewed_at" db:"reviewed_at"`
	ReviewedBy       *uint64                `json:"reviewed_by" db:"reviewed_by"`
}

// ModerationStatus represents the status of content in moderation
type ModerationStatus string

const (
	ModerationStatusPending  ModerationStatus = "pending"
	ModerationStatusApproved ModerationStatus = "approved"
	ModerationStatusRejected ModerationStatus = "rejected"
	ModerationStatusFlagged  ModerationStatus = "flagged"
)

// AIFeedback represents AI analysis results
type AIFeedback struct {
	Provider             string                 `json:"provider"`
	QualityScore         float64                `json:"quality_score"`
	GrammarScore         *float64               `json:"grammar_score,omitempty"`
	ReadabilityScore     *float64               `json:"readability_score,omitempty"`
	AppropriatenessScore *float64               `json:"appropriateness_score,omitempty"`
	Issues               []AIIssue              `json:"issues,omitempty"`
	Suggestions          []AISuggestion         `json:"suggestions,omitempty"`
	FlaggedContent       []AIFlaggedContent     `json:"flagged_content,omitempty"`
	ProcessingTimeMs     int                    `json:"processing_time_ms"`
	Confidence           float64                `json:"confidence"`
}

// AIIssue represents an issue found by AI analysis
type AIIssue struct {
	Type        string `json:"type"`        // "grammar", "spelling", "readability", "inappropriate"
	Severity    string `json:"severity"`    // "low", "medium", "high"
	Description string `json:"description"`
	Location    string `json:"location,omitempty"` // Position in content
	Suggestion  string `json:"suggestion,omitempty"`
}

// AISuggestion represents an improvement suggestion from AI
type AISuggestion struct {
	Type        string `json:"type"`        // "title", "meta_description", "content", "seo"
	Priority    string `json:"priority"`    // "low", "medium", "high"
	Description string `json:"description"`
	Original    string `json:"original,omitempty"`
	Suggested   string `json:"suggested,omitempty"`
}

// AIFlaggedContent represents content flagged for review
type AIFlaggedContent struct {
	Type        string `json:"type"`        // "inappropriate", "spam", "low_quality"
	Content     string `json:"content"`     // The flagged content snippet
	Reason      string `json:"reason"`      // Why it was flagged
	Confidence  float64 `json:"confidence"` // AI confidence in flagging (0.0-1.0)
}

// Scan implements the sql.Scanner interface for AIFeedback
func (a *AIFeedback) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AIFeedback", value)
	}
}

// Value implements the driver.Valuer interface for AIFeedback
func (a AIFeedback) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// ContentQualityCheck represents AI analysis results stored in database
type ContentQualityCheck struct {
	ID                   uint64     `json:"id" db:"id"`
	ArticleID            uint64     `json:"article_id" db:"article_id"`
	ArticleVersionID     *uint64    `json:"article_version_id" db:"article_version_id"`
	AIProvider           string     `json:"ai_provider" db:"ai_provider"`
	QualityScore         float64    `json:"quality_score" db:"quality_score"`
	GrammarScore         *float64   `json:"grammar_score" db:"grammar_score"`
	ReadabilityScore     *float64   `json:"readability_score" db:"readability_score"`
	AppropriatenessScore *float64   `json:"appropriateness_score" db:"appropriateness_score"`
	IssuesFound          *AIIssues  `json:"issues_found" db:"issues_found"`
	Suggestions          *AISuggestions `json:"suggestions" db:"suggestions"`
	FlaggedContent       *AIFlaggedContents `json:"flagged_content" db:"flagged_content"`
	ProcessingTimeMs     int        `json:"processing_time_ms" db:"processing_time_ms"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// AIIssues is a wrapper for JSON serialization
type AIIssues []AIIssue

// Scan implements the sql.Scanner interface for AIIssues
func (a *AIIssues) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AIIssues", value)
	}
}

// Value implements the driver.Valuer interface for AIIssues
func (a AIIssues) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// AISuggestions is a wrapper for JSON serialization
type AISuggestions []AISuggestion

// Scan implements the sql.Scanner interface for AISuggestions
func (a *AISuggestions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AISuggestions", value)
	}
}

// Value implements the driver.Valuer interface for AISuggestions
func (a AISuggestions) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// AIFlaggedContents is a wrapper for JSON serialization
type AIFlaggedContents []AIFlaggedContent

// Scan implements the sql.Scanner interface for AIFlaggedContents
func (a *AIFlaggedContents) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AIFlaggedContents", value)
	}
}

// Value implements the driver.Valuer interface for AIFlaggedContents
func (a AIFlaggedContents) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// ModerationAction represents an action taken in the moderation process
type ModerationAction struct {
	ID                 uint64    `json:"id" db:"id"`
	ModerationQueueID  uint64    `json:"moderation_queue_id" db:"moderation_queue_id"`
	Action             string    `json:"action" db:"action"`
	PerformedBy        uint64    `json:"performed_by" db:"performed_by"`
	Notes              string    `json:"notes" db:"notes"`
	PreviousStatus     string    `json:"previous_status" db:"previous_status"`
	NewStatus          string    `json:"new_status" db:"new_status"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// BulkModerationJob represents a bulk moderation operation
type BulkModerationJob struct {
	ID               uint64                 `json:"id" db:"id"`
	JobName          string                 `json:"job_name" db:"job_name"`
	JobType          string                 `json:"job_type" db:"job_type"`
	Criteria         *BulkModerationCriteria `json:"criteria" db:"criteria"`
	TotalItems       int                    `json:"total_items" db:"total_items"`
	ProcessedItems   int                    `json:"processed_items" db:"processed_items"`
	SuccessfulItems  int                    `json:"successful_items" db:"successful_items"`
	FailedItems      int                    `json:"failed_items" db:"failed_items"`
	Status           string                 `json:"status" db:"status"`
	CreatedBy        uint64                 `json:"created_by" db:"created_by"`
	StartedAt        *time.Time             `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at" db:"completed_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	ErrorLog         string                 `json:"error_log" db:"error_log"`
}

// BulkModerationCriteria represents selection criteria for bulk operations
type BulkModerationCriteria struct {
	Status           []string   `json:"status,omitempty"`
	Priority         []int      `json:"priority,omitempty"`
	SubmittedAfter   *time.Time `json:"submitted_after,omitempty"`
	SubmittedBefore  *time.Time `json:"submitted_before,omitempty"`
	AIQualityMin     *float64   `json:"ai_quality_min,omitempty"`
	AIQualityMax     *float64   `json:"ai_quality_max,omitempty"`
	ContentType      []string   `json:"content_type,omitempty"`
	SubmittedBy      []uint64   `json:"submitted_by,omitempty"`
	AssignedTo       []uint64   `json:"assigned_to,omitempty"`
}

// Scan implements the sql.Scanner interface for BulkModerationCriteria
func (b *BulkModerationCriteria) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, b)
	case string:
		return json.Unmarshal([]byte(v), b)
	default:
		return fmt.Errorf("cannot scan %T into BulkModerationCriteria", value)
	}
}

// Value implements the driver.Valuer interface for BulkModerationCriteria
func (b BulkModerationCriteria) Value() (driver.Value, error) {
	return json.Marshal(b)
}

// ValidateModerationQueue validates a ModerationQueue struct
func ValidateModerationQueue(mq *ModerationQueue) error {
	var errors []string

	// Article ID validation
	if mq.ArticleID == 0 {
		errors = append(errors, "article_id is required")
	}

	// Content type validation
	validContentTypes := map[string]bool{
		"article": true,
		"comment": true,
	}
	if !validContentTypes[mq.ContentType] {
		errors = append(errors, "content_type must be one of: article, comment")
	}

	// Status validation
	validStatuses := map[ModerationStatus]bool{
		ModerationStatusPending:  true,
		ModerationStatusApproved: true,
		ModerationStatusRejected: true,
		ModerationStatusFlagged:  true,
	}
	if !validStatuses[mq.Status] {
		errors = append(errors, "status must be one of: pending, approved, rejected, flagged")
	}

	// Priority validation
	if mq.Priority < 1 || mq.Priority > 4 {
		errors = append(errors, "priority must be between 1 and 4")
	}

	// Submitted by validation
	if mq.SubmittedBy == 0 {
		errors = append(errors, "submitted_by is required")
	}

	// AI quality score validation
	if mq.AIQualityScore != nil && (*mq.AIQualityScore < 0.0 || *mq.AIQualityScore > 1.0) {
		errors = append(errors, "ai_quality_score must be between 0.0 and 1.0")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "ModerationQueue validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// IsHighPriority returns true if the moderation item is high priority
func (mq *ModerationQueue) IsHighPriority() bool {
	return mq.Priority >= 3
}

// IsAutoApprovable returns true if the item can be auto-approved based on AI score
func (mq *ModerationQueue) IsAutoApprovable(threshold float64) bool {
	return mq.AIQualityScore != nil && *mq.AIQualityScore >= threshold
}

// HasIssues returns true if AI found issues with the content
func (mq *ModerationQueue) HasIssues() bool {
	return mq.AIFeedback != nil && len(mq.AIFeedback.Issues) > 0
}

// GetHighSeverityIssues returns issues marked as high severity
func (mq *ModerationQueue) GetHighSeverityIssues() []AIIssue {
	if mq.AIFeedback == nil {
		return nil
	}
	
	var highSeverityIssues []AIIssue
	for _, issue := range mq.AIFeedback.Issues {
		if issue.Severity == "high" {
			highSeverityIssues = append(highSeverityIssues, issue)
		}
	}
	
	return highSeverityIssues
}