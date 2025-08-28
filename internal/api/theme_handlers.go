package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// ThemeHandlers handles theme-related HTTP requests
type ThemeHandlers struct {
	themeService *services.ThemeService
}

// NewThemeHandlers creates new theme handlers
func NewThemeHandlers(themeService *services.ThemeService) *ThemeHandlers {
	return &ThemeHandlers{
		themeService: themeService,
	}
}

// CreateTheme creates a new theme
func (h *ThemeHandlers) CreateTheme(c *gin.Context) {
	var theme models.Theme
	if err := c.ShouldBindJSON(&theme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	createdTheme, err := h.themeService.CreateTheme(&theme)
	if err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdTheme)
}

// GetTheme retrieves a theme by ID
func (h *ThemeHandlers) GetTheme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme ID"})
		return
	}

	theme, err := h.themeService.GetTheme(id)
	if err != nil {
		if err.Error() == "theme not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, theme)
}

// GetActiveTheme retrieves the active theme
func (h *ThemeHandlers) GetActiveTheme(c *gin.Context) {
	theme, err := h.themeService.GetActiveTheme()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, theme)
}

// GetAllThemes retrieves all themes
func (h *ThemeHandlers) GetAllThemes(c *gin.Context) {
	themes, err := h.themeService.GetAllThemes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get themes", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"themes": themes})
}

// UpdateTheme updates a theme
func (h *ThemeHandlers) UpdateTheme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme ID"})
		return
	}

	var theme models.Theme
	if err := c.ShouldBindJSON(&theme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	theme.ID = id
	if err := h.themeService.UpdateTheme(&theme); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		if err.Error() == "theme not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme updated successfully"})
}

// SetActiveTheme sets a theme as active
func (h *ThemeHandlers) SetActiveTheme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme ID"})
		return
	}

	if err := h.themeService.SetActiveTheme(id); err != nil {
		if err.Error() == "theme not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set active theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme activated successfully"})
}

// DeleteTheme deletes a theme
func (h *ThemeHandlers) DeleteTheme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme ID"})
		return
	}

	if err := h.themeService.DeleteTheme(id); err != nil {
		if err.Error() == "theme not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
			return
		}
		if err.Error() == "cannot delete active theme" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete active theme"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete theme", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme deleted successfully"})
}

// GenerateThemeCSS generates CSS for a theme
func (h *ThemeHandlers) GenerateThemeCSS(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme ID"})
		return
	}

	theme, err := h.themeService.GetTheme(id)
	if err != nil {
		if err.Error() == "theme not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get theme", "details": err.Error()})
		return
	}

	css, err := h.themeService.GenerateCSS(theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSS", "details": err.Error()})
		return
	}

	c.Header("Content-Type", "text/css")
	c.String(http.StatusOK, css)
}

// GetActiveThemeCSS generates CSS for the active theme
func (h *ThemeHandlers) GetActiveThemeCSS(c *gin.Context) {
	theme, err := h.themeService.GetActiveTheme()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active theme", "details": err.Error()})
		return
	}

	css, err := h.themeService.GenerateCSS(theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSS", "details": err.Error()})
		return
	}

	c.Header("Content-Type", "text/css")
	c.String(http.StatusOK, css)
}

// CreateTemplateOverride creates a new template override
func (h *ThemeHandlers) CreateTemplateOverride(c *gin.Context) {
	var override models.TemplateOverride
	if err := c.ShouldBindJSON(&override); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	createdOverride, err := h.themeService.CreateTemplateOverride(&override)
	if err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template override", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdOverride)
}

// GetTemplateOverride retrieves a template override by path
func (h *ThemeHandlers) GetTemplateOverride(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template path is required"})
		return
	}

	override, err := h.themeService.GetTemplateOverride(path)
	if err != nil {
		if err.Error() == "template override not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template override not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template override", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, override)
}

// GetAllTemplateOverrides retrieves all template overrides
func (h *ThemeHandlers) GetAllTemplateOverrides(c *gin.Context) {
	overrides, err := h.themeService.GetAllTemplateOverrides()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template overrides", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"overrides": overrides})
}

// UpdateTemplateOverride updates a template override
func (h *ThemeHandlers) UpdateTemplateOverride(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid override ID"})
		return
	}

	var override models.TemplateOverride
	if err := c.ShouldBindJSON(&override); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	override.ID = id
	if err := h.themeService.UpdateTemplateOverride(&override); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		if err.Error() == "template override not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template override not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template override", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template override updated successfully"})
}

// DeleteTemplateOverride deletes a template override
func (h *ThemeHandlers) DeleteTemplateOverride(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid override ID"})
		return
	}

	if err := h.themeService.DeleteTemplateOverride(id); err != nil {
		if err.Error() == "template override not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template override not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template override", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template override deleted successfully"})
}

// PreviewTemplate generates a preview of a template
func (h *ThemeHandlers) PreviewTemplate(c *gin.Context) {
	var request struct {
		Content      string `json:"content"`
		TemplatePath string `json:"template_path"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if request.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template content is required"})
		return
	}

	html, err := h.themeService.PreviewTemplate(request.Content, request.TemplatePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to preview template", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"html": string(html)})
}

// GetTemplateContent gets template content (with override support)
func (h *ThemeHandlers) GetTemplateContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template path is required"})
		return
	}

	content, err := h.themeService.GetTemplateContent(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template content", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content, "path": path})
}

// GetDefaultThemeConfig returns the default theme configuration
func (h *ThemeHandlers) GetDefaultThemeConfig(c *gin.Context) {
	config := models.GetDefaultThemeConfig()
	c.JSON(http.StatusOK, gin.H{"config": config})
}