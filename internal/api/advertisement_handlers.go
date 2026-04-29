package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// AdvertisementHandlers handles advertisement-related HTTP requests
type AdvertisementHandlers struct {
	adService *services.AdvertisementService
}

// NewAdvertisementHandlers creates new advertisement handlers
func NewAdvertisementHandlers(adService *services.AdvertisementService) *AdvertisementHandlers {
	return &AdvertisementHandlers{
		adService: adService,
	}
}

// Campaign handlers

// CreateCampaign creates a new advertisement campaign
// @Summary Create advertisement campaign
// @Description Create a new advertisement campaign
// @Tags advertisements
// @Accept json
// @Produce json
// @Param campaign body models.AdvertisementCampaign true "Campaign data"
// @Success 201 {object} models.AdvertisementCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns [post]
func (h *AdvertisementHandlers) CreateCampaign(c *gin.Context) {
	var campaign models.AdvertisementCampaign
	if err := c.ShouldBindJSON(&campaign); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.adService.CreateCampaign(&campaign); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create campaign",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, campaign)
}

// GetCampaign retrieves a campaign by ID
// @Summary Get advertisement campaign
// @Description Get advertisement campaign by ID
// @Tags advertisements
// @Produce json
// @Param id path int true "Campaign ID"
// @Success 200 {object} models.AdvertisementCampaign
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{id} [get]
func (h *AdvertisementHandlers) GetCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	campaign, err := h.adService.GetCampaign(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Campaign not found",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, campaign)
}

// UpdateCampaign updates an existing campaign
// @Summary Update advertisement campaign
// @Description Update an existing advertisement campaign
// @Tags advertisements
// @Accept json
// @Produce json
// @Param id path int true "Campaign ID"
// @Param campaign body models.AdvertisementCampaign true "Campaign data"
// @Success 200 {object} models.AdvertisementCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{id} [put]
func (h *AdvertisementHandlers) UpdateCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	var campaign models.AdvertisementCampaign
	if err := c.ShouldBindJSON(&campaign); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	campaign.ID = id
	if err := h.adService.UpdateCampaign(&campaign); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update campaign",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, campaign)
}

// DeleteCampaign deletes a campaign
// @Summary Delete advertisement campaign
// @Description Delete an advertisement campaign
// @Tags advertisements
// @Param id path int true "Campaign ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{id} [delete]
func (h *AdvertisementHandlers) DeleteCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	if err := h.adService.DeleteCampaign(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete campaign",
			Message: err.Error(),
		})
		return
	}
	
	c.Status(http.StatusNoContent)
}

// Slot handlers

// CreateSlot creates a new advertisement slot
// @Summary Create advertisement slot
// @Description Create a new advertisement slot
// @Tags advertisements
// @Accept json
// @Produce json
// @Param slot body models.AdvertisementSlot true "Slot data"
// @Success 201 {object} models.AdvertisementSlot
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/slots [post]
func (h *AdvertisementHandlers) CreateSlot(c *gin.Context) {
	var slot models.AdvertisementSlot
	if err := c.ShouldBindJSON(&slot); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.adService.CreateSlot(&slot); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create slot",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, slot)
}

// GetSlots retrieves advertisement slots
// @Summary Get advertisement slots
// @Description Get advertisement slots, optionally filtered by page type and position
// @Tags advertisements
// @Produce json
// @Param page_type query string false "Page type filter"
// @Param position query string false "Position filter"
// @Success 200 {array} models.AdvertisementSlot
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/slots [get]
func (h *AdvertisementHandlers) GetSlots(c *gin.Context) {
	pageType := c.Query("page_type")
	position := c.Query("position")
	
	var slots []models.AdvertisementSlot
	var err error
	
	if pageType != "" {
		slots, err = h.adService.GetSlotsByPageType(pageType, position)
	} else {
		slots, err = h.adService.GetAllSlots()
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get slots",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, slots)
}

// GetSlot retrieves a slot by ID
// @Summary Get advertisement slot
// @Description Get advertisement slot by ID
// @Tags advertisements
// @Produce json
// @Param id path int true "Slot ID"
// @Success 200 {object} models.AdvertisementSlot
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/slots/{id} [get]
func (h *AdvertisementHandlers) GetSlot(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid slot ID",
			Message: "Slot ID must be a valid number",
		})
		return
	}
	
	slot, err := h.adService.GetSlot(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Slot not found",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, slot)
}

// Creative handlers

// CreateCreative creates a new advertisement creative
// @Summary Create advertisement creative
// @Description Create a new advertisement creative
// @Tags advertisements
// @Accept json
// @Produce json
// @Param creative body models.AdvertisementCreative true "Creative data"
// @Success 201 {object} models.AdvertisementCreative
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/creatives [post]
func (h *AdvertisementHandlers) CreateCreative(c *gin.Context) {
	var creative models.AdvertisementCreative
	if err := c.ShouldBindJSON(&creative); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	// Validate performance requirements (Core Web Vitals)
	if err := h.adService.ValidateAdPerformance(&creative); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Performance validation failed",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.adService.CreateCreative(&creative); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create creative",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, creative)
}

// GetCreativesByCampaign retrieves creatives for a campaign
// @Summary Get campaign creatives
// @Description Get advertisement creatives for a specific campaign
// @Tags advertisements
// @Produce json
// @Param campaign_id path int true "Campaign ID"
// @Success 200 {array} models.AdvertisementCreative
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{campaign_id}/creatives [get]
func (h *AdvertisementHandlers) GetCreativesByCampaign(c *gin.Context) {
	campaignID, err := strconv.ParseUint(c.Param("campaign_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	creatives, err := h.adService.GetCreativesByCampaign(campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get creatives",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, creatives)
}

// Targeting handlers

// CreateTargeting creates targeting rules for a campaign
// @Summary Create campaign targeting
// @Description Create targeting rules for an advertisement campaign
// @Tags advertisements
// @Accept json
// @Produce json
// @Param targeting body models.AdvertisementTargeting true "Targeting data"
// @Success 201 {object} models.AdvertisementTargeting
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/targeting [post]
func (h *AdvertisementHandlers) CreateTargeting(c *gin.Context) {
	var targeting models.AdvertisementTargeting
	if err := c.ShouldBindJSON(&targeting); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.adService.CreateTargeting(&targeting); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create targeting",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, targeting)
}

// GetTargetingByCampaign retrieves targeting rules for a campaign
// @Summary Get campaign targeting
// @Description Get targeting rules for a specific campaign
// @Tags advertisements
// @Produce json
// @Param campaign_id path int true "Campaign ID"
// @Success 200 {array} models.AdvertisementTargeting
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{campaign_id}/targeting [get]
func (h *AdvertisementHandlers) GetTargetingByCampaign(c *gin.Context) {
	campaignID, err := strconv.ParseUint(c.Param("campaign_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	targeting, err := h.adService.GetTargetingByCampaign(campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get targeting",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, targeting)
}

// Placement handlers

// CreatePlacement creates a new advertisement placement
// @Summary Create advertisement placement
// @Description Create a new advertisement placement
// @Tags advertisements
// @Accept json
// @Produce json
// @Param placement body models.AdvertisementPlacement true "Placement data"
// @Success 201 {object} models.AdvertisementPlacement
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/placements [post]
func (h *AdvertisementHandlers) CreatePlacement(c *gin.Context) {
	var placement models.AdvertisementPlacement
	if err := c.ShouldBindJSON(&placement); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.adService.CreatePlacement(&placement); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create placement",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, placement)
}

// Ad serving handlers

// GetAdvertisements retrieves advertisements for a specific context
// @Summary Get advertisements
// @Description Get advertisements for a specific page context with targeting
// @Tags advertisements
// @Accept json
// @Produce json
// @Param request body models.AdvertisementRequest true "Advertisement request"
// @Success 200 {object} models.AdvertisementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/serve [post]
func (h *AdvertisementHandlers) GetAdvertisements(c *gin.Context) {
	var request models.AdvertisementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Message: err.Error(),
		})
		return
	}
	
	// Set defaults
	if request.MaxAds == 0 {
		request.MaxAds = 3
	}
	
	// Extract client information
	request.IPAddress = h.getClientIP(c)
	request.UserAgent = c.GetHeader("User-Agent")
	request.Referer = c.GetHeader("Referer")
	
	// Determine device type if not provided
	if request.DeviceType == "" {
		request.DeviceType = h.adService.GetDeviceType(request.UserAgent)
	}
	
	response, err := h.adService.GetAdvertisements(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get advertisements",
			Message: err.Error(),
		})
		return
	}
	
	// Set cache headers for performance
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.JSON(http.StatusOK, response)
}

// Tracking handlers

// RecordImpression records an advertisement impression
// @Summary Record advertisement impression
// @Description Record an advertisement impression for tracking
// @Tags advertisements
// @Param ad_id path string true "Advertisement ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/impression/{ad_id} [post]
func (h *AdvertisementHandlers) RecordImpression(c *gin.Context) {
	adID := c.Param("ad_id")
	if adID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ad ID",
			Message: "Ad ID is required",
		})
		return
	}
	
	request := &models.AdvertisementRequest{
		IPAddress: h.getClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
		Referer:   c.GetHeader("Referer"),
		PageURL:   c.GetHeader("X-Page-URL"),
	}
	
	if err := h.adService.RecordImpression(adID, request); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to record impression",
			Message: err.Error(),
		})
		return
	}
	
	// Return 1x1 transparent pixel for tracking
	c.Header("Content-Type", "image/gif")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	
	// 1x1 transparent GIF
	pixel := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
		0x0C, 0x0A, 0x00, 0x3B,
	}
	
	c.Data(http.StatusOK, "image/gif", pixel)
}

// RecordClick records an advertisement click
// @Summary Record advertisement click
// @Description Record an advertisement click for tracking
// @Tags advertisements
// @Param ad_id path string true "Advertisement ID"
// @Success 302
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/click/{ad_id} [get]
func (h *AdvertisementHandlers) RecordClick(c *gin.Context) {
	adID := c.Param("ad_id")
	if adID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ad ID",
			Message: "Ad ID is required",
		})
		return
	}
	
	request := &models.AdvertisementRequest{
		IPAddress: h.getClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
		Referer:   c.GetHeader("Referer"),
		PageURL:   c.GetHeader("X-Page-URL"),
	}
	
	// Record click asynchronously
	go h.adService.RecordClick(adID, request)
	
	// Get click URL from query parameter
	clickURL := c.Query("url")
	if clickURL == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing click URL",
			Message: "Click URL is required",
		})
		return
	}
	
	// Redirect to the actual URL
	c.Redirect(http.StatusFound, clickURL)
}

// Analytics handlers

// GetCampaignStats retrieves campaign statistics
// @Summary Get campaign statistics
// @Description Get performance statistics for a campaign
// @Tags advertisements
// @Produce json
// @Param campaign_id path int true "Campaign ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} models.AdvertisementStats
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/campaigns/{campaign_id}/stats [get]
func (h *AdvertisementHandlers) GetCampaignStats(c *gin.Context) {
	campaignID, err := strconv.ParseUint(c.Param("campaign_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid campaign ID",
			Message: "Campaign ID must be a valid number",
		})
		return
	}
	
	// Parse date range
	startDate := time.Now().AddDate(0, 0, -30) // Default to last 30 days
	endDate := time.Now()
	
	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}
	
	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}
	
	stats, err := h.adService.GetCampaignStats(campaignID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get campaign stats",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// GetPerformanceReport generates a performance report
// @Summary Get performance report
// @Description Get performance report for campaigns
// @Tags advertisements
// @Produce json
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param campaign_ids query string false "Comma-separated campaign IDs"
// @Success 200 {array} models.AdvertisementPerformanceReport
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ads/reports/performance [get]
func (h *AdvertisementHandlers) GetPerformanceReport(c *gin.Context) {
	// Parse date range
	startDate := time.Now().AddDate(0, 0, -30) // Default to last 30 days
	endDate := time.Now()
	
	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}
	
	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}
	
	// Parse campaign IDs
	var campaignIDs []uint64
	if idsStr := c.Query("campaign_ids"); idsStr != "" {
		for _, idStr := range strings.Split(idsStr, ",") {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil {
				campaignIDs = append(campaignIDs, id)
			}
		}
	}
	
	reports, err := h.adService.GetPerformanceReport(startDate, endDate, campaignIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get performance report",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, reports)
}

// Helper methods

// getClientIP extracts the client IP address from the request
func (h *AdvertisementHandlers) getClientIP(c *gin.Context) string {
	return h.adService.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)
}