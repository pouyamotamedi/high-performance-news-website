package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// WidgetHandlers handles widget-related HTTP requests
type WidgetHandlers struct {
	widgetService *services.WidgetService
}

// NewWidgetHandlers creates new widget handlers
func NewWidgetHandlers(widgetService *services.WidgetService) *WidgetHandlers {
	return &WidgetHandlers{
		widgetService: widgetService,
	}
}

// CreateWidget creates a new widget
func (h *WidgetHandlers) CreateWidget(c *gin.Context) {
	var widget models.Widget
	if err := c.ShouldBindJSON(&widget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	createdWidget, err := h.widgetService.CreateWidget(&widget)
	if err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create widget", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdWidget)
}

// GetWidget retrieves a widget by ID
func (h *WidgetHandlers) GetWidget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid widget ID"})
		return
	}

	widget, err := h.widgetService.GetWidget(id)
	if err != nil {
		if err.Error() == "widget not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widget", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, widget)
}

// GetAllWidgets retrieves all widgets
func (h *WidgetHandlers) GetAllWidgets(c *gin.Context) {
	widgets, err := h.widgetService.GetAllWidgets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widgets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"widgets": widgets})
}

// GetWidgetsByType retrieves widgets by type
func (h *WidgetHandlers) GetWidgetsByType(c *gin.Context) {
	widgetType := models.WidgetType(c.Param("type"))
	
	widgets, err := h.widgetService.GetWidgetsByType(widgetType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widgets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"widgets": widgets})
}

// UpdateWidget updates a widget
func (h *WidgetHandlers) UpdateWidget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid widget ID"})
		return
	}

	var widget models.Widget
	if err := c.ShouldBindJSON(&widget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	widget.ID = id
	if err := h.widgetService.UpdateWidget(&widget); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		if err.Error() == "widget not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update widget", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget updated successfully"})
}

// DeleteWidget deletes a widget
func (h *WidgetHandlers) DeleteWidget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid widget ID"})
		return
	}

	if err := h.widgetService.DeleteWidget(id); err != nil {
		if err.Error() == "widget not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete widget", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget deleted successfully"})
}

// CreateWidgetPlacement creates a new widget placement
func (h *WidgetHandlers) CreateWidgetPlacement(c *gin.Context) {
	var placement models.WidgetPlacement
	if err := c.ShouldBindJSON(&placement); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	createdPlacement, err := h.widgetService.CreateWidgetPlacement(&placement)
	if err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create widget placement", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdPlacement)
}

// GetWidgetPlacements retrieves widget placements for a page and zone
func (h *WidgetHandlers) GetWidgetPlacements(c *gin.Context) {
	pageType := models.PageType(c.Query("page_type"))
	zone := models.WidgetZone(c.Query("zone"))

	if pageType == "" || zone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page_type and zone parameters are required"})
		return
	}

	placements, err := h.widgetService.GetWidgetPlacements(pageType, zone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widget placements", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"placements": placements})
}

// UpdateWidgetPlacement updates a widget placement
func (h *WidgetHandlers) UpdateWidgetPlacement(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid placement ID"})
		return
	}

	var placement models.WidgetPlacement
	if err := c.ShouldBindJSON(&placement); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	placement.ID = id
	if err := h.widgetService.UpdateWidgetPlacement(&placement); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		if err.Error() == "widget placement not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget placement not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update widget placement", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget placement updated successfully"})
}

// DeleteWidgetPlacement deletes a widget placement
func (h *WidgetHandlers) DeleteWidgetPlacement(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid placement ID"})
		return
	}

	if err := h.widgetService.DeleteWidgetPlacement(id); err != nil {
		if err.Error() == "widget placement not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget placement not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete widget placement", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget placement deleted successfully"})
}

// UpdatePlacementPositions updates the positions of multiple widget placements (for drag-and-drop)
func (h *WidgetHandlers) UpdatePlacementPositions(c *gin.Context) {
	var request struct {
		Placements []models.WidgetPlacement `json:"placements"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Convert to pointers
	var placements []*models.WidgetPlacement
	for i := range request.Placements {
		placements = append(placements, &request.Placements[i])
	}

	if err := h.widgetService.UpdatePlacementPositions(placements); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": validationErr.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update placement positions", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Placement positions updated successfully"})
}

// RenderWidget renders a widget's HTML content
func (h *WidgetHandlers) RenderWidget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid widget ID"})
		return
	}

	widget, err := h.widgetService.GetWidget(id)
	if err != nil {
		if err.Error() == "widget not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widget", "details": err.Error()})
		return
	}

	html, err := h.widgetService.RenderWidget(widget)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render widget", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"html": string(html)})
}

// GetWidgetTypes returns available widget types
func (h *WidgetHandlers) GetWidgetTypes(c *gin.Context) {
	types := []map[string]interface{}{
		{
			"type":        models.WidgetTypeLatestArticles,
			"name":        "Latest Articles",
			"description": "Display the most recent articles",
			"configurable": []string{"article_count", "category_ids", "tag_ids", "show_excerpt", "show_date", "show_author", "show_image"},
		},
		{
			"type":        models.WidgetTypePopularArticles,
			"name":        "Popular Articles",
			"description": "Display the most viewed articles",
			"configurable": []string{"article_count", "category_ids", "tag_ids"},
		},
		{
			"type":        models.WidgetTypeTrendingArticles,
			"name":        "Trending Articles",
			"description": "Display trending articles",
			"configurable": []string{"article_count", "category_ids", "tag_ids"},
		},
		{
			"type":        models.WidgetTypeCategories,
			"name":        "Categories",
			"description": "Display article categories",
			"configurable": []string{"show_hierarchy", "max_depth", "show_count"},
		},
		{
			"type":        models.WidgetTypeTags,
			"name":        "Tags",
			"description": "Display article tags as a tag cloud",
			"configurable": []string{},
		},
		{
			"type":        models.WidgetTypeSearch,
			"name":        "Search",
			"description": "Search form widget",
			"configurable": []string{},
		},
		{
			"type":        models.WidgetTypeNewsletter,
			"name":        "Newsletter",
			"description": "Newsletter subscription form",
			"configurable": []string{},
		},
		{
			"type":        models.WidgetTypeCustomHTML,
			"name":        "Custom HTML",
			"description": "Custom HTML content",
			"configurable": []string{"html_content"},
		},
		{
			"type":        models.WidgetTypeAdvertisement,
			"name":        "Advertisement",
			"description": "Advertisement slot",
			"configurable": []string{"ad_slot_id", "ad_size"},
		},
		{
			"type":        models.WidgetTypeSocialMedia,
			"name":        "Social Media",
			"description": "Social media links",
			"configurable": []string{"platforms", "show_icons"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// GetPageTypes returns available page types for widget placement
func (h *WidgetHandlers) GetPageTypes(c *gin.Context) {
	types := []map[string]interface{}{
		{"type": models.PageTypeHomepage, "name": "Homepage"},
		{"type": models.PageTypeArticle, "name": "Article Pages"},
		{"type": models.PageTypeCategory, "name": "Category Pages"},
		{"type": models.PageTypeTag, "name": "Tag Pages"},
		{"type": models.PageTypeSearch, "name": "Search Pages"},
		{"type": models.PageTypeAuthor, "name": "Author Pages"},
		{"type": models.PageTypeGlobal, "name": "All Pages"},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// GetWidgetZones returns available widget zones
func (h *WidgetHandlers) GetWidgetZones(c *gin.Context) {
	zones := []map[string]interface{}{
		{"zone": models.WidgetZoneHeader, "name": "Header"},
		{"zone": models.WidgetZoneSidebar, "name": "Sidebar"},
		{"zone": models.WidgetZoneFooter, "name": "Footer"},
		{"zone": models.WidgetZoneContent, "name": "Content Area"},
		{"zone": models.WidgetZoneAfterTitle, "name": "After Title"},
		{"zone": models.WidgetZoneBeforeContent, "name": "Before Content"},
		{"zone": models.WidgetZoneAfterContent, "name": "After Content"},
	}

	c.JSON(http.StatusOK, gin.H{"zones": zones})
}