package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// CommentHandlers handles HTTP requests for comments
type CommentHandlers struct {
	commentRepo *repositories.CommentRepository
	userRepo    *repositories.UserRepository
	rateLimiter *RateLimiter
}

// NewCommentHandlers creates a new comment handlers instance
func NewCommentHandlers(commentRepo *repositories.CommentRepository, userRepo *repositories.UserRepository, rateLimiter *RateLimiter) *CommentHandlers {
	return &CommentHandlers{
		commentRepo: commentRepo,
		userRepo:    userRepo,
		rateLimiter: rateLimiter,
	}
}

// CreateCommentRequest represents the request body for creating a comment
type CreateCommentRequest struct {
	ArticleID   uint64  `json:"article_id" binding:"required"`
	ParentID    *uint64 `json:"parent_id,omitempty"`
	Content     string  `json:"content" binding:"required,max=2000"`
	AuthorName  string  `json:"author_name" binding:"required,max=100"`
	AuthorEmail string  `json:"author_email" binding:"required,email,max=255"`
}

// CreateComment handles POST /api/v1/comments
func (h *CommentHandlers) CreateComment(c *gin.Context) {
	// Rate limiting for comment creation (5 comments per minute per IP)
	clientIP := c.ClientIP()
	if !h.rateLimiter.Allow("comment_create:"+clientIP, 5, time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"message": "Too many comments created. Please wait before creating another comment.",
		})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Get user ID if authenticated
	var userID *uint64
	if userIDInterface, exists := c.Get("user_id"); exists {
		if uid, ok := userIDInterface.(uint64); ok {
			userID = &uid
		}
	}

	// Create comment model
	comment := &models.Comment{
		ArticleID:   req.ArticleID,
		UserID:      userID,
		ParentID:    req.ParentID,
		Content:     req.Content,
		AuthorName:  req.AuthorName,
		AuthorEmail: req.AuthorEmail,
		AuthorIP:    clientIP,
		UserAgent:   c.GetHeader("User-Agent"),
		Status:      models.CommentStatusPending,
	}

	// Sanitize content
	comment.SanitizeContent()

	// Perform spam detection
	spamDetection := models.DetectSpam(comment)
	comment.SpamScore = spamDetection.Score

	// Auto-approve if spam score is low and user is authenticated
	if spamDetection.Score < 0.3 && userID != nil {
		comment.Status = models.CommentStatusApproved
	} else if spamDetection.IsSpam {
		comment.Status = models.CommentStatusSpam
	}

	// Create comment in database
	createdComment, err := h.commentRepo.Create(comment)
	if err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Validation failed",
				"message": validationErr.Message,
				"details": validationErr.Fields,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create comment",
			"message": "An error occurred while creating the comment",
		})
		return
	}

	// Return response based on status
	if createdComment.Status == models.CommentStatusApproved {
		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment created and approved",
			"comment": createdComment,
		})
	} else if createdComment.Status == models.CommentStatusSpam {
		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment flagged as spam and will be reviewed",
			"comment_id": createdComment.ID,
		})
	} else {
		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment created and pending moderation",
			"comment_id": createdComment.ID,
		})
	}
}

// GetCommentsByArticle handles GET /api/v1/articles/:id/comments
func (h *CommentHandlers) GetCommentsByArticle(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}

	// Only show approved comments to public
	comments, err := h.commentRepo.GetByArticleID(articleID, models.CommentStatusApproved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve comments",
			"message": "An error occurred while retrieving comments",
		})
		return
	}

	// Get comment count
	count, err := h.commentRepo.GetCommentCount(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get comment count",
			"message": "An error occurred while counting comments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total_count": count,
	})
}

// GetComment handles GET /api/v1/comments/:id
func (h *CommentHandlers) GetComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid comment ID",
			"message": "Comment ID must be a valid number",
		})
		return
	}

	comment, err := h.commentRepo.GetByID(commentID)
	if err != nil {
		if _, ok := err.(*models.NotFoundError); ok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Comment not found",
				"message": "The requested comment does not exist",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve comment",
			"message": "An error occurred while retrieving the comment",
		})
		return
	}

	// Only show approved comments to public, unless user is moderator
	userRole, _ := c.Get("user_role")
	if comment.Status != models.CommentStatusApproved && userRole != models.RoleAdmin && userRole != models.RoleEditor {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comment not found",
			"message": "The requested comment does not exist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comment": comment,
	})
}

// GetPendingComments handles GET /api/v1/admin/comments/pending
func (h *CommentHandlers) GetPendingComments(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to access moderation features",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	comments, err := h.commentRepo.GetPendingComments(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve pending comments",
			"message": "An error occurred while retrieving pending comments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"limit": limit,
		"offset": offset,
	})
}

// ModerationRequest represents the request body for comment moderation
type ModerationRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject spam"`
	Reason string `json:"reason,omitempty"`
}

// ModerateComment handles PUT /api/v1/admin/comments/:id/moderate
func (h *CommentHandlers) ModerateComment(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to moderate comments",
		})
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid comment ID",
			"message": "Comment ID must be a valid number",
		})
		return
	}

	var req ModerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Get moderator ID
	moderatorID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to moderate comments",
		})
		return
	}

	// Convert action to status
	var status models.CommentStatus
	switch req.Action {
	case "approve":
		status = models.CommentStatusApproved
	case "reject":
		status = models.CommentStatusRejected
	case "spam":
		status = models.CommentStatusSpam
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid action",
			"message": "Action must be one of: approve, reject, spam",
		})
		return
	}

	err = h.commentRepo.UpdateStatus(commentID, status, moderatorID.(uint64), req.Reason)
	if err != nil {
		if _, ok := err.(*models.NotFoundError); ok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Comment not found",
				"message": "The requested comment does not exist",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to moderate comment",
			"message": "An error occurred while moderating the comment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Comment moderated successfully",
		"action": req.Action,
	})
}

// BulkModerationRequest represents the request body for bulk moderation
type BulkModerationRequest struct {
	CommentIDs []uint64 `json:"comment_ids" binding:"required,min=1"`
	Action     string   `json:"action" binding:"required,oneof=approve reject spam"`
	Reason     string   `json:"reason,omitempty"`
}

// BulkModerateComments handles PUT /api/v1/admin/comments/bulk-moderate
func (h *CommentHandlers) BulkModerateComments(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to moderate comments",
		})
		return
	}

	var req BulkModerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Limit bulk operations to prevent abuse
	if len(req.CommentIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Too many comments",
			"message": "Bulk moderation is limited to 100 comments at a time",
		})
		return
	}

	// Get moderator ID
	moderatorID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to moderate comments",
		})
		return
	}

	// Convert action to status
	var status models.CommentStatus
	switch req.Action {
	case "approve":
		status = models.CommentStatusApproved
	case "reject":
		status = models.CommentStatusRejected
	case "spam":
		status = models.CommentStatusSpam
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid action",
			"message": "Action must be one of: approve, reject, spam",
		})
		return
	}

	err := h.commentRepo.BulkUpdateStatus(req.CommentIDs, status, moderatorID.(uint64), req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to moderate comments",
			"message": "An error occurred while moderating the comments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Comments moderated successfully",
		"action": req.Action,
		"count": len(req.CommentIDs),
	})
}

// DeleteComment handles DELETE /api/v1/admin/comments/:id
func (h *CommentHandlers) DeleteComment(c *gin.Context) {
	// Check if user has admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to delete comments",
		})
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid comment ID",
			"message": "Comment ID must be a valid number",
		})
		return
	}

	err = h.commentRepo.Delete(commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete comment",
			"message": "An error occurred while deleting the comment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Comment and its replies deleted successfully",
	})
}

// GetModerationStats handles GET /api/v1/admin/comments/stats
func (h *CommentHandlers) GetModerationStats(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to access moderation statistics",
		})
		return
	}

	stats, err := h.commentRepo.GetModerationStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
			"message": "An error occurred while retrieving moderation statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// SearchComments handles GET /api/v1/admin/comments/search
func (h *CommentHandlers) SearchComments(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to search comments",
		})
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing query",
			"message": "Search query is required",
		})
		return
	}

	statusStr := c.Query("status")
	var status models.CommentStatus
	if statusStr != "" {
		status = models.CommentStatus(statusStr)
		if !models.IsValidCommentStatus(status) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid status",
				"message": "Status must be one of: pending, approved, rejected, spam",
			})
			return
		}
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	comments, err := h.commentRepo.SearchComments(query, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search comments",
			"message": "An error occurred while searching comments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"query": query,
		"status": statusStr,
		"limit": limit,
		"offset": offset,
	})
}

// GetRecentComments handles GET /api/v1/admin/comments/recent
func (h *CommentHandlers) GetRecentComments(c *gin.Context) {
	// Check if user has moderation permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != models.RoleAdmin && userRole != models.RoleEditor) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You don't have permission to view recent comments",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	comments, err := h.commentRepo.GetRecentComments(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve recent comments",
			"message": "An error occurred while retrieving recent comments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"limit": limit,
	})
}