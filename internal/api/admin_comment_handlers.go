package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminCommentHandlers handles admin comment management frontend routes
type AdminCommentHandlers struct {
	templatePath string
}

// NewAdminCommentHandlers creates a new AdminCommentHandlers instance
func NewAdminCommentHandlers(templatePath string) *AdminCommentHandlers {
	return &AdminCommentHandlers{
		templatePath: templatePath,
	}
}

// RegisterAdminCommentRoutes registers admin comment management routes
func (h *AdminCommentHandlers) RegisterAdminCommentRoutes(router *gin.RouterGroup) {
	// Comment management page
	router.GET("/comments", h.CommentManagementPage)
	
	// Additional comment-related admin pages can be added here
	// router.GET("/comments/analytics", h.CommentAnalyticsPage)
	// router.GET("/comments/settings", h.CommentSettingsPage)
}

// CommentManagementPage serves the comment management interface
func (h *AdminCommentHandlers) CommentManagementPage(c *gin.Context) {
	c.HTML(http.StatusOK, "comment_management.html", gin.H{
		"title": "Comment Management",
		"user":  h.getCurrentUser(c),
		"page":  "comments",
		"breadcrumbs": []gin.H{
			{"name": "Dashboard", "url": "/admin/dashboard"},
			{"name": "Comment Management", "url": "/admin/comments"},
		},
	})
}

// CommentAnalyticsPage serves comment analytics and insights
func (h *AdminCommentHandlers) CommentAnalyticsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "comment_analytics.html", gin.H{
		"title": "Comment Analytics",
		"user":  h.getCurrentUser(c),
		"page":  "comment-analytics",
		"breadcrumbs": []gin.H{
			{"name": "Dashboard", "url": "/admin/dashboard"},
			{"name": "Comments", "url": "/admin/comments"},
			{"name": "Analytics", "url": "/admin/comments/analytics"},
		},
	})
}

// CommentSettingsPage serves comment system configuration
func (h *AdminCommentHandlers) CommentSettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "comment_settings.html", gin.H{
		"title": "Comment Settings",
		"user":  h.getCurrentUser(c),
		"page":  "comment-settings",
		"breadcrumbs": []gin.H{
			{"name": "Dashboard", "url": "/admin/dashboard"},
			{"name": "Comments", "url": "/admin/comments"},
			{"name": "Settings", "url": "/admin/comments/settings"},
		},
	})
}

// Helper method to get current user from context
func (h *AdminCommentHandlers) getCurrentUser(c *gin.Context) gin.H {
	// This would typically get user info from JWT token or session
	// For now, return mock user data
	return gin.H{
		"id":       1,
		"username": "admin",
		"email":    "admin@example.com",
		"role":     "admin",
		"avatar":   "/static/images/default-avatar.png",
	}
}