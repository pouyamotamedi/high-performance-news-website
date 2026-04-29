package models

import (
	"strings"
	"time"
)

// CommentStatus represents the moderation status of a comment
type CommentStatus string

const (
	CommentStatusPending  CommentStatus = "pending"
	CommentStatusApproved CommentStatus = "approved"
	CommentStatusRejected CommentStatus = "rejected"
	CommentStatusSpam     CommentStatus = "spam"
)

// Comment represents a user comment with nested thread support
type Comment struct {
	ID           uint64        `json:"id" db:"id"`
	ArticleID    uint64        `json:"article_id" db:"article_id" validate:"required"`
	UserID       *uint64       `json:"user_id" db:"user_id"` // Nullable for anonymous comments
	ParentID     *uint64       `json:"parent_id" db:"parent_id"` // For nested threading
	Content      string        `json:"content" db:"content" validate:"required,max=2000"`
	AuthorName   string        `json:"author_name" db:"author_name" validate:"required,max=100"`
	AuthorEmail  string        `json:"author_email" db:"author_email" validate:"required,email,max=255"`
	AuthorIP     string        `json:"-" db:"author_ip"` // Hidden from JSON for privacy
	UserAgent    string        `json:"-" db:"user_agent"` // Hidden from JSON for privacy
	Status       CommentStatus `json:"status" db:"status" validate:"required"`
	SpamScore    float64       `json:"spam_score" db:"spam_score"` // Spam detection score
	ModeratedBy  *uint64       `json:"moderated_by" db:"moderated_by"` // User who moderated
	ModeratedAt  *time.Time    `json:"moderated_at" db:"moderated_at"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
	
	// Nested comments for threading (populated by repository)
	Replies []Comment `json:"replies,omitempty" db:"-"`
	
	// User information (populated by repository)
	User *User `json:"user,omitempty" db:"-"`
	
	// Moderated by user information (populated by repository)
	Moderator *User `json:"moderator,omitempty" db:"-"`
	
	// Article information (populated by repository for admin views)
	ArticleSlug  string `json:"article_slug,omitempty" db:"-"`
	ArticleTitle string `json:"article_title,omitempty" db:"-"`
}

// ValidateComment validates a Comment struct with comprehensive error checking
func ValidateComment(comment *Comment) error {
	var errors []string

	// Article ID validation
	if comment.ArticleID == 0 {
		errors = append(errors, "article_id is required")
	}

	// Content validation
	if strings.TrimSpace(comment.Content) == "" {
		errors = append(errors, "content is required")
	}
	if len(comment.Content) > 2000 {
		errors = append(errors, "content must be less than 2000 characters")
	}

	// Author name validation
	if strings.TrimSpace(comment.AuthorName) == "" {
		errors = append(errors, "author_name is required")
	}
	if len(comment.AuthorName) > 100 {
		errors = append(errors, "author_name must be less than 100 characters")
	}

	// Author email validation
	if strings.TrimSpace(comment.AuthorEmail) == "" {
		errors = append(errors, "author_email is required")
	}
	if len(comment.AuthorEmail) > 255 {
		errors = append(errors, "author_email must be less than 255 characters")
	}
	if !IsValidEmail(comment.AuthorEmail) {
		errors = append(errors, "author_email must be a valid email address")
	}

	// Status validation
	if !IsValidCommentStatus(comment.Status) {
		errors = append(errors, "status must be one of: pending, approved, rejected, spam")
	}

	// Parent ID validation (if provided, ensure it's not the same as the comment ID)
	if comment.ParentID != nil && comment.ID != 0 && *comment.ParentID == comment.ID {
		errors = append(errors, "parent_id cannot be the same as comment id")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Comment validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// IsValidCommentStatus checks if the comment status is valid
func IsValidCommentStatus(status CommentStatus) bool {
	validStatuses := map[CommentStatus]bool{
		CommentStatusPending:  true,
		CommentStatusApproved: true,
		CommentStatusRejected: true,
		CommentStatusSpam:     true,
	}
	return validStatuses[status]
}

// PrepareForDB prepares the comment for database insertion
func (c *Comment) PrepareForDB() {
	c.Content = strings.TrimSpace(c.Content)
	c.AuthorName = strings.TrimSpace(c.AuthorName)
	c.AuthorEmail = strings.TrimSpace(strings.ToLower(c.AuthorEmail))
	
	if c.Status == "" {
		c.Status = CommentStatusPending // Default status
	}
}

// IsApproved returns true if the comment is approved
func (c *Comment) IsApproved() bool {
	return c.Status == CommentStatusApproved
}

// IsSpam returns true if the comment is marked as spam
func (c *Comment) IsSpam() bool {
	return c.Status == CommentStatusSpam
}

// CanBeRepliedTo returns true if the comment can receive replies
func (c *Comment) CanBeRepliedTo() bool {
	return c.Status == CommentStatusApproved
}

// GetThreadDepth calculates the depth of the comment in the thread
func (c *Comment) GetThreadDepth() int {
	if c.ParentID == nil {
		return 0
	}
	// This would need to be calculated by the repository layer
	// by traversing up the parent chain
	return 1 // Simplified for now
}

// SanitizeContent removes potentially harmful content from the comment
func (c *Comment) SanitizeContent() {
	// Remove HTML tags and normalize whitespace
	c.Content = SanitizeCommentContent(c.Content)
}

// SanitizeCommentContent removes HTML and normalizes whitespace
func SanitizeCommentContent(content string) string {
	// Remove HTML tags (simple approach)
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	
	// Normalize whitespace
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	return strings.Join(cleanLines, "\n")
}

// CommentModerationAction represents an action taken during moderation
type CommentModerationAction struct {
	CommentID   uint64        `json:"comment_id" db:"comment_id"`
	Action      CommentStatus `json:"action" db:"action"`
	ModeratorID uint64        `json:"moderator_id" db:"moderator_id"`
	Reason      string        `json:"reason" db:"reason"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// CommentSpamDetection contains spam detection results
type CommentSpamDetection struct {
	Score       float64           `json:"score"`
	Reasons     []string          `json:"reasons"`
	IsSpam      bool              `json:"is_spam"`
	Confidence  float64           `json:"confidence"`
	Checks      map[string]bool   `json:"checks"`
}

// DetectSpam performs basic spam detection on a comment with custom settings
func DetectSpam(comment *Comment) *CommentSpamDetection {
	return DetectSpamWithSettings(comment, nil, 0.5)
}

// DetectSpamWithSettings performs spam detection with custom keywords and threshold
func DetectSpamWithSettings(comment *Comment, customKeywords []string, threshold float64) *CommentSpamDetection {
	detection := &CommentSpamDetection{
		Score:      0.0,
		Reasons:    []string{},
		IsSpam:     false,
		Confidence: 0.0,
		Checks:     make(map[string]bool),
	}
	
	content := strings.ToLower(comment.Content)
	
	// Check for links (both raw URLs and HTML-encoded)
	linkCount := strings.Count(content, "http://") + strings.Count(content, "https://")
	// Also check for HTML-encoded links (href= indicates a link)
	linkCount += strings.Count(content, "href=")
	
	if linkCount >= 1 {
		if linkCount == 1 {
			detection.Score += 0.2 // 20% for single link
			detection.Reasons = append(detection.Reasons, "contains link")
			detection.Checks["has_links"] = true
		} else if linkCount >= 2 {
			detection.Score += 0.4 // 40% for multiple links
			detection.Reasons = append(detection.Reasons, "multiple links")
			detection.Checks["multiple_links"] = true
		}
	}
	
	// Check for spam keywords (use custom keywords if provided)
	spamKeywords := []string{"viagra", "casino", "lottery", "winner", "congratulations", "click here", "free money"}
	if customKeywords != nil && len(customKeywords) > 0 {
		spamKeywords = customKeywords
	}
	
	for _, keyword := range spamKeywords {
		if strings.Contains(content, keyword) {
			detection.Score += 0.4
			detection.Reasons = append(detection.Reasons, "spam keywords")
			detection.Checks["spam_keywords"] = true
			break
		}
	}
	
	// Check for excessive capitalization
	upperCount := 0
	for _, r := range comment.Content {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		}
	}
	if len(comment.Content) > 0 && float64(upperCount)/float64(len(comment.Content)) > 0.5 {
		detection.Score += 0.2
		detection.Reasons = append(detection.Reasons, "excessive capitalization")
		detection.Checks["excessive_caps"] = true
	}
	
	// Check for very short content (potential spam)
	if len(strings.TrimSpace(comment.Content)) < 10 {
		detection.Score += 0.1
		detection.Reasons = append(detection.Reasons, "very short content")
		detection.Checks["short_content"] = true
	}
	
	// Check for repetitive characters
	if hasRepetitiveChars(comment.Content) {
		detection.Score += 0.2
		detection.Reasons = append(detection.Reasons, "repetitive characters")
		detection.Checks["repetitive_chars"] = true
	}
	
	// Determine if it's spam based on threshold
	detection.IsSpam = detection.Score >= threshold
	detection.Confidence = detection.Score
	
	return detection
}

// hasRepetitiveChars checks for repetitive character patterns
func hasRepetitiveChars(content string) bool {
	if len(content) < 4 {
		return false
	}
	
	for i := 0; i < len(content)-3; i++ {
		if content[i] == content[i+1] && content[i+1] == content[i+2] && content[i+2] == content[i+3] {
			return true
		}
	}
	return false
}