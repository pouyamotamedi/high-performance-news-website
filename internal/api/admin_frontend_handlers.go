package api

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// AdminFrontendHandlers handles admin panel frontend routes
type AdminFrontendHandlers struct {
	templatePath string
}

// NewAdminFrontendHandlers creates a new AdminFrontendHandlers instance
func NewAdminFrontendHandlers(templatePath string) *AdminFrontendHandlers {
	return &AdminFrontendHandlers{
		templatePath: templatePath,
	}
}

// RegisterAdminFrontendRoutes registers admin frontend routes
func (h *AdminFrontendHandlers) RegisterAdminFrontendRoutes(router *gin.Engine) {
	// Admin login page (no auth required)
	router.GET("/admin/login", h.AdminLoginPage)
	
	// Admin dashboard and pages (auth required - will be handled by middleware)
	adminGroup := router.Group("/admin")
	{
		adminGroup.GET("/", h.AdminDashboard)
		adminGroup.GET("/dashboard", h.AdminDashboard)
		adminGroup.GET("/analytics", h.AdminAnalytics)
		adminGroup.GET("/users", h.AdminUsers)
		adminGroup.GET("/content", h.AdminContent)
		adminGroup.GET("/settings", h.AdminSettings)
		adminGroup.GET("/system", h.AdminSystem)
	}
}

// AdminLoginPage serves the admin login page
func (h *AdminFrontendHandlers) AdminLoginPage(c *gin.Context) {
	// Check if user is already logged in via cookie/session
	// For now, just serve the login page
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Admin Login",
	})
}

// AdminDashboard serves the main admin dashboard
func (h *AdminFrontendHandlers) AdminDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Admin Dashboard",
		"user":  h.getCurrentUser(c),
	})
}

// AdminAnalytics serves the analytics page
func (h *AdminFrontendHandlers) AdminAnalytics(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Analytics",
		"user":  h.getCurrentUser(c),
		"page":  "analytics",
	})
}

// AdminUsers serves the user management page
func (h *AdminFrontendHandlers) AdminUsers(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "User Management",
		"user":  h.getCurrentUser(c),
		"page":  "users",
	})
}

// AdminContent serves the content management page
func (h *AdminFrontendHandlers) AdminContent(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Content Management",
		"user":  h.getCurrentUser(c),
		"page":  "content",
	})
}

// AdminSettings serves the settings page
func (h *AdminFrontendHandlers) AdminSettings(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Settings",
		"user":  h.getCurrentUser(c),
		"page":  "settings",
	})
}

// AdminSystem serves the system monitoring page
func (h *AdminFrontendHandlers) AdminSystem(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "System Monitoring",
		"user":  h.getCurrentUser(c),
		"page":  "system",
	})
}

// Helper method to get current user from context
func (h *AdminFrontendHandlers) getCurrentUser(c *gin.Context) gin.H {
	// This would typically get user info from JWT token or session
	// For now, return mock user data
	return gin.H{
		"id":       1,
		"username": "admin",
		"email":    "admin@example.com",
		"role":     "admin",
	}
}

// SetupTemplates configures HTML templates for admin panel
func SetupAdminTemplates(router *gin.Engine, templatePath string) {
	// Load HTML templates
	router.LoadHTMLGlob(filepath.Join(templatePath, "**/*"))
	
	// Serve static assets for admin panel
	router.Static("/admin/assets", filepath.Join(templatePath, "../assets"))
}