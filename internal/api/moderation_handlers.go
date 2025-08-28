package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// ModerationHandlers handles moderation-related API endpoints
type ModerationHandlers struct {
	moderationService     *services.ModerationService
	bulkModerationService *services.BulkModerationService
}

// NewModerationHandlers creates new moderation handlers
func NewModerationHandlers(moderationService *services.ModerationService, bulkModerationService *services.BulkModerationService) *ModerationHandlers {
	return &ModerationHandlers{
		moderationService:     moderationService,
		bulkModerationService: bulkModerationService,
	}
}

// SubmitModerationRequest represents the request to submit content for moderation
type SubmitModerationRequest struct {
	ArticleID   uint64 `json:"article_id" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Priority    int    `json:"priority" binding:"min=1,max=4"`
}

// SubmitForModeration submits content for moderation review
// POST /api/v1/moderation/submit
func (mh *ModerationHandlers) SubmitForModeration(c *gin.Context) {
	var req SubmitModerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Set default priority if not provided
	if req.Priority == 0 {
		req.Priority = 1
	}

	moderationItem, err := mh.moderationService.SubmitForModeration(
		req.ArticleID, req.ContentType, req.Priority, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit for moderation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Content submitted for moderation",
		"moderation_item": moderationItem,
	})
}

// GetModerationQueueRequest represents query parameters for moderation queue
type GetModerationQueueRequest struct {
	Status           []string  `form:"status"`
	Priority         []int     `form:"priority"`
	AssignedTo       *uint64   `form:"assigned_to"`
	SubmittedAfter   *string   `form:"submitted_after"`  // ISO 8601 format
	SubmittedBefore  *string   `form:"submitted_before"` // ISO 8601 format
	AIQualityMin     *float64  `form:"ai_quality_min"`
	AIQualityMax     *float64  `form:"ai_quality_max"`
	Limit            int       `form:"limit,default=20"`
	Offset           int       `form:"offset,default=0"`
}

// GetModerationQueue returns items in the moderation queue
// GET /api/v1/moderation/queue
func (mh *ModerationHandlers) GetModerationQueue(c *gin.Context) {
	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	var req GetModerationQueueRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse time filters
	filters := services.ModerationFilters{
		Status:       req.Status,
		Priority:     req.Priority,
		AssignedTo:   req.AssignedTo,
		AIQualityMin: req.AIQualityMin,
		AIQualityMax: req.AIQualityMax,
	}

	if req.SubmittedAfter != nil {
		if t, err := time.Parse(time.RFC3339, *req.SubmittedAfter); err == nil {
			filters.SubmittedAfter = &t
		}
	}

	if req.SubmittedBefore != nil {
		if t, err := time.Parse(time.RFC3339, *req.SubmittedBefore); err == nil {
			filters.SubmittedBefore = &t
		}
	}

	// Set reasonable limits
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	items, total, err := mh.moderationService.GetModerationQueue(filters, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get moderation queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// AssignModeratorRequest represents the request to assign a moderator
type AssignModeratorRequest struct {
	ModeratorID uint64 `json:"moderator_id" binding:"required"`
}

// AssignModerator assigns a moderator to a moderation item
// POST /api/v1/moderation/:id/assign
func (mh *ModerationHandlers) AssignModerator(c *gin.Context) {
	moderationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderation ID"})
		return
	}

	var req AssignModeratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err = mh.moderationService.AssignModerator(moderationID, req.ModeratorID, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign moderator"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Moderator assigned successfully",
		"moderator_id": req.ModeratorID,
	})
}

// ApproveContentRequest represents the request to approve content
type ApproveContentRequest struct {
	Notes string `json:"notes"`
}

// ApproveContent approves content in moderation
// POST /api/v1/moderation/:id/approve
func (mh *ModerationHandlers) ApproveContent(c *gin.Context) {
	moderationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderation ID"})
		return
	}

	var req ApproveContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err = mh.moderationService.ApproveContent(moderationID, userID.(uint64), req.Notes, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content approved successfully"})
}

// RejectContentRequest represents the request to reject content
type RejectContentRequest struct {
	Reason string `json:"reason" binding:"required"`
	Notes  string `json:"notes"`
}

// RejectContent rejects content in moderation
// POST /api/v1/moderation/:id/reject
func (mh *ModerationHandlers) RejectContent(c *gin.Context) {
	moderationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderation ID"})
		return
	}

	var req RejectContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err = mh.moderationService.RejectContent(moderationID, userID.(uint64), req.Reason, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content rejected successfully"})
}

// FlagContentRequest represents the request to flag content
type FlagContentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// FlagContent flags content for additional review
// POST /api/v1/moderation/:id/flag
func (mh *ModerationHandlers) FlagContent(c *gin.Context) {
	moderationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderation ID"})
		return
	}

	var req FlagContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err = mh.moderationService.FlagContent(moderationID, userID.(uint64), req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content flagged successfully"})
}

// RunAICheck triggers AI quality check for a moderation item
// POST /api/v1/moderation/:id/ai-check
func (mh *ModerationHandlers) RunAICheck(c *gin.Context) {
	moderationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderation ID"})
		return
	}

	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	err = mh.moderationService.RunAIQualityCheck(moderationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run AI quality check"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI quality check initiated"})
}

// GetModerationStats returns moderation statistics
// GET /api/v1/moderation/stats
func (mh *ModerationHandlers) GetModerationStats(c *gin.Context) {
	// Check moderator permissions
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
		return
	}

	stats, err := mh.moderationService.GetModerationStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get moderation stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// CreateBulkJobRequest represents the request to create a bulk moderation job
type CreateBulkJobRequest struct {
	JobName  string                         `json:"job_name" binding:"required"`
	JobType  string                         `json:"job_type" binding:"required"`
	Criteria *models.BulkModerationCriteria `json:"criteria" binding:"required"`
}

// CreateBulkJob creates a new bulk moderation job
// POST /api/v1/moderation/bulk-jobs
func (mh *ModerationHandlers) CreateBulkJob(c *gin.Context) {
	var req CreateBulkJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	job, err := mh.bulkModerationService.CreateBulkJob(req.JobName, req.JobType, req.Criteria, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bulk job"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bulk job created successfully",
		"job": job,
	})
}

// ExecuteBulkJob executes a bulk moderation job
// POST /api/v1/moderation/bulk-jobs/:id/execute
func (mh *ModerationHandlers) ExecuteBulkJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Execute job asynchronously
	go func() {
		if err := mh.bulkModerationService.ExecuteBulkJob(jobID); err != nil {
			// Log error - in production, this would use proper logging
			fmt.Printf("Bulk job %d execution failed: %v\n", jobID, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Bulk job execution started"})
}

// GetBulkJobs returns a list of bulk moderation jobs
// GET /api/v1/moderation/bulk-jobs
func (mh *ModerationHandlers) GetBulkJobs(c *gin.Context) {
	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	jobs, total, err := mh.bulkModerationService.GetBulkJobs(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bulk jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetBulkJob returns a specific bulk moderation job
// GET /api/v1/moderation/bulk-jobs/:id
func (mh *ModerationHandlers) GetBulkJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	job, err := mh.bulkModerationService.GetBulkJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bulk job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

// CancelBulkJob cancels a running bulk job
// POST /api/v1/moderation/bulk-jobs/:id/cancel
func (mh *ModerationHandlers) CancelBulkJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	err = mh.bulkModerationService.CancelBulkJob(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel bulk job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bulk job cancelled successfully"})
}

// RegisterModerationRoutes registers all moderation routes
func (mh *ModerationHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Moderation routes
	moderation := router.Group("/moderation")
	{
		moderation.POST("/submit", mh.SubmitForModeration)
		moderation.GET("/queue", mh.GetModerationQueue)
		moderation.GET("/stats", mh.GetModerationStats)
		
		moderation.POST("/:id/assign", mh.AssignModerator)
		moderation.POST("/:id/approve", mh.ApproveContent)
		moderation.POST("/:id/reject", mh.RejectContent)
		moderation.POST("/:id/flag", mh.FlagContent)
		moderation.POST("/:id/ai-check", mh.RunAICheck)
		
		// Bulk moderation routes
		moderation.POST("/bulk-jobs", mh.CreateBulkJob)
		moderation.GET("/bulk-jobs", mh.GetBulkJobs)
		moderation.GET("/bulk-jobs/:id", mh.GetBulkJob)
		moderation.POST("/bulk-jobs/:id/execute", mh.ExecuteBulkJob)
		moderation.POST("/bulk-jobs/:id/cancel", mh.CancelBulkJob)
	}
}