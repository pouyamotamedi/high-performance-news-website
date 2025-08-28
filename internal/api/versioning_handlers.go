package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// VersioningHandlers handles versioning-related API endpoints
type VersioningHandlers struct {
	versioningService *services.VersioningService
}

// NewVersioningHandlers creates new versioning handlers
func NewVersioningHandlers(versioningService *services.VersioningService) *VersioningHandlers {
	return &VersioningHandlers{
		versioningService: versioningService,
	}
}

// CreateVersionRequest represents the request to create a new version
type CreateVersionRequest struct {
	ChangeSummary string `json:"change_summary" binding:"required"`
}

// CreateVersion creates a new version of an article
// POST /api/v1/articles/:id/versions
func (vh *VersioningHandlers) CreateVersion(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get current article data (this would typically come from article service)
	// For now, we'll create a mock article - in real implementation, 
	// this would fetch from the database
	article := &models.Article{
		ID:           articleID,
		Title:        "Sample Article", // This should be fetched from DB
		Slug:         "sample-article",
		Content:      "Sample content",
		AuthorID:     userID.(uint64),
		CategoryID:   1,
		Status:       "draft",
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	version, err := vh.versioningService.CreateVersion(article, req.ChangeSummary, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Version created successfully",
		"version": version,
	})
}

// GetVersionHistory returns the version history for an article
// GET /api/v1/articles/:id/versions
func (vh *VersioningHandlers) GetVersionHistory(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	versions, err := vh.versioningService.GetVersionHistory(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
		"count":    len(versions),
	})
}

// GetVersion returns a specific version of an article
// GET /api/v1/articles/:id/versions/:version
func (vh *VersioningHandlers) GetVersion(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	versionNumber, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	version, err := vh.versioningService.GetVersion(articleID, versionNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"version": version})
}

// CompareVersions compares two versions of an article
// GET /api/v1/articles/:id/versions/:version1/compare/:version2
func (vh *VersioningHandlers) CompareVersions(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	version1, err := strconv.Atoi(c.Param("version1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version1 number"})
		return
	}

	version2, err := strconv.Atoi(c.Param("version2"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version2 number"})
		return
	}

	comparison, err := vh.versioningService.CompareVersions(articleID, version1, version2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compare versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comparison": comparison})
}

// RestoreVersionRequest represents the request to restore a version
type RestoreVersionRequest struct {
	VersionNumber int `json:"version_number" binding:"required"`
}

// RestoreVersion restores an article to a specific version
// POST /api/v1/articles/:id/versions/restore
func (vh *VersioningHandlers) RestoreVersion(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req RestoreVersionRequest
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

	// Check user permissions (admin or editor)
	userRole, exists := c.Get("user_role")
	if !exists || (userRole != "admin" && userRole != "editor") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	err = vh.versioningService.RestoreVersion(articleID, req.VersionNumber, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Version restored successfully",
		"restored_to_version": req.VersionNumber,
	})
}

// GetVersionStats returns statistics about article versions
// GET /api/v1/admin/versions/stats
func (vh *VersioningHandlers) GetVersionStats(c *gin.Context) {
	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	stats, err := vh.versioningService.GetVersionStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// DeleteOldVersionsRequest represents the request to delete old versions
type DeleteOldVersionsRequest struct {
	DaysToKeep int `json:"days_to_keep" binding:"required,min=1"`
}

// DeleteOldVersions deletes versions older than specified days
// DELETE /api/v1/admin/versions/cleanup
func (vh *VersioningHandlers) DeleteOldVersions(c *gin.Context) {
	// Check admin permissions
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	var req DeleteOldVersionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deletedCount, err := vh.versioningService.DeleteOldVersions(req.DaysToKeep)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Old versions deleted successfully",
		"deleted_count": deletedCount,
		"days_kept": req.DaysToKeep,
	})
}

// RegisterVersioningRoutes registers all versioning routes
func (vh *VersioningHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Article version routes
	articles := router.Group("/articles")
	{
		articles.POST("/:id/versions", vh.CreateVersion)
		articles.GET("/:id/versions", vh.GetVersionHistory)
		articles.GET("/:id/versions/:version", vh.GetVersion)
		articles.GET("/:id/versions/:version1/compare/:version2", vh.CompareVersions)
		articles.POST("/:id/versions/restore", vh.RestoreVersion)
	}

	// Admin version routes
	admin := router.Group("/admin")
	{
		admin.GET("/versions/stats", vh.GetVersionStats)
		admin.DELETE("/versions/cleanup", vh.DeleteOldVersions)
	}
}