package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/services"
)

// CanonicalHandlers handles HTTP requests for canonicalization
type CanonicalHandlers struct {
	canonicalManager *services.CanonicalManager
}

// NewCanonicalHandlers creates a new canonical handlers instance
func NewCanonicalHandlers(canonicalManager *services.CanonicalManager) *CanonicalHandlers {
	return &CanonicalHandlers{
		canonicalManager: canonicalManager,
	}
}

// ScheduleCanonicalJobRequest represents the request to schedule a canonical job
type ScheduleCanonicalJobRequest struct {
	ArticleID     uint64                      `json:"article_id" binding:"required"`
	Target        services.CanonicalTarget    `json:"target" binding:"required"`
	AdminOverride bool                        `json:"admin_override"`
}

// ScheduleCanonicalJobResponse represents the response after scheduling a canonical job
type ScheduleCanonicalJobResponse struct {
	JobID   uint64 `json:"job_id"`
	Message string `json:"message"`
}

// ScheduleCanonicalJob schedules a new canonicalization job
// POST /api/v1/canonical/jobs
func (h *CanonicalHandlers) ScheduleCanonicalJob(c *gin.Context) {
	var req ScheduleCanonicalJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
		})
		return
	}

	userIDUint64, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal error",
			"message": "Invalid user ID format",
		})
		return
	}

	// Check if user has permission for admin override
	if req.AdminOverride {
		userRole, exists := c.Get("user_role")
		if !exists || (userRole != "admin" && userRole != "editor") {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Admin override requires admin or editor role",
			})
			return
		}
	}

	// Schedule the job
	jobID, err := h.canonicalManager.ScheduleCanonicalJob(
		req.ArticleID,
		req.Target,
		&userIDUint64,
		req.AdminOverride,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to schedule canonical job",
			"message": err.Error(),
		})
		return
	}

	message := "Canonical job scheduled with 48-hour delay"
	if req.AdminOverride {
		message = "Canonical job scheduled for immediate processing"
	}

	c.JSON(http.StatusCreated, ScheduleCanonicalJobResponse{
		JobID:   jobID,
		Message: message,
	})
}

// GetPendingJobs returns all pending canonical jobs
// GET /api/v1/canonical/jobs/pending
func (h *CanonicalHandlers) GetPendingJobs(c *gin.Context) {
	jobs, err := h.canonicalManager.GetPendingJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get pending jobs",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// GetJobsByArticle returns all canonical jobs for a specific article
// GET /api/v1/canonical/jobs/article/:article_id
func (h *CanonicalHandlers) GetJobsByArticle(c *gin.Context) {
	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}

	jobs, err := h.canonicalManager.GetJobsByArticle(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get jobs for article",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":       jobs,
		"count":      len(jobs),
		"article_id": articleID,
	})
}

// ProcessPendingJobs manually processes all pending canonical jobs
// POST /api/v1/canonical/jobs/process
func (h *CanonicalHandlers) ProcessPendingJobs(c *gin.Context) {
	// Check if user has admin/editor role
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Processing jobs requires admin or editor role",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
		})
		return
	}

	userIDUint64, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal error",
			"message": "Invalid user ID format",
		})
		return
	}

	// Process pending jobs
	processed, err := h.canonicalManager.ProcessPendingJobs(&userIDUint64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process pending jobs",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"processed": processed,
		"message":   "Pending jobs processed successfully",
	})
}

// CancelJob cancels a pending canonical job
// DELETE /api/v1/canonical/jobs/:job_id
func (h *CanonicalHandlers) CancelJob(c *gin.Context) {
	jobIDStr := c.Param("job_id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid job ID",
			"message": "Job ID must be a valid number",
		})
		return
	}

	// Check if user has admin/editor role
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Cancelling jobs requires admin or editor role",
		})
		return
	}

	err = h.canonicalManager.CancelJob(jobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to cancel job",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancelled successfully",
		"job_id":  jobID,
	})
}

// RetryFailedJobRequest represents the request to retry a failed job
type RetryFailedJobRequest struct {
	AdminOverride bool `json:"admin_override"`
}

// RetryFailedJob retries a failed canonical job
// POST /api/v1/canonical/jobs/:job_id/retry
func (h *CanonicalHandlers) RetryFailedJob(c *gin.Context) {
	jobIDStr := c.Param("job_id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid job ID",
			"message": "Job ID must be a valid number",
		})
		return
	}

	var req RetryFailedJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to no admin override if request body is empty
		req.AdminOverride = false
	}

	// Check if user has admin/editor role
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Retrying jobs requires admin or editor role",
		})
		return
	}

	// Check if user has permission for admin override
	if req.AdminOverride && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Admin override requires admin role",
		})
		return
	}

	err = h.canonicalManager.RetryFailedJob(jobID, req.AdminOverride)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to retry job",
			"message": err.Error(),
		})
		return
	}

	message := "Job scheduled for retry with 48-hour delay"
	if req.AdminOverride {
		message = "Job scheduled for immediate retry"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"job_id":  jobID,
	})
}

// GenerateCanonicalURLRequest represents the request to generate a canonical URL
type GenerateCanonicalURLRequest struct {
	Target services.CanonicalTarget `json:"target" binding:"required"`
}

// GenerateCanonicalURLResponse represents the response with generated canonical URL
type GenerateCanonicalURLResponse struct {
	CanonicalURL string `json:"canonical_url"`
}

// GenerateCanonicalURL generates a canonical URL for preview purposes
// POST /api/v1/canonical/generate-url
func (h *CanonicalHandlers) GenerateCanonicalURL(c *gin.Context) {
	var req GenerateCanonicalURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	canonicalURL, err := h.canonicalManager.GenerateCanonicalURL(req.Target)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to generate canonical URL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateCanonicalURLResponse{
		CanonicalURL: canonicalURL,
	})
}

// GetJobStats returns statistics about canonical jobs
// GET /api/v1/canonical/stats
func (h *CanonicalHandlers) GetJobStats(c *gin.Context) {
	stats, err := h.canonicalManager.GetJobStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get job statistics",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// CleanupOldJobs removes old processed/cancelled/failed jobs
// POST /api/v1/canonical/cleanup
func (h *CanonicalHandlers) CleanupOldJobs(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Cleanup requires admin role",
		})
		return
	}

	deletedCount, err := h.canonicalManager.CleanupOldJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cleanup old jobs",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted_count": deletedCount,
		"message":       "Old jobs cleaned up successfully",
	})
}

// RegisterCanonicalRoutes registers all canonical-related routes
func RegisterCanonicalRoutes(router *gin.RouterGroup, handlers *CanonicalHandlers) {
	canonical := router.Group("/canonical")
	{
		// Job management
		canonical.POST("/jobs", handlers.ScheduleCanonicalJob)
		canonical.GET("/jobs/pending", handlers.GetPendingJobs)
		canonical.GET("/jobs/article/:article_id", handlers.GetJobsByArticle)
		canonical.POST("/jobs/process", handlers.ProcessPendingJobs)
		canonical.DELETE("/jobs/:job_id", handlers.CancelJob)
		canonical.POST("/jobs/:job_id/retry", handlers.RetryFailedJob)

		// Utility endpoints
		canonical.POST("/generate-url", handlers.GenerateCanonicalURL)
		canonical.GET("/stats", handlers.GetJobStats)
		canonical.POST("/cleanup", handlers.CleanupOldJobs)
	}
}