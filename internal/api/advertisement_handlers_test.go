package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MockAdvertisementService for testing
type MockAdvertisementService struct {
	campaigns   map[uint64]*models.AdvertisementCampaign
	slots       map[uint64]*models.AdvertisementSlot
	creatives   map[uint64]*models.AdvertisementCreative
	nextID      uint64
}

func NewMockAdvertisementService() *MockAdvertisementService {
	return &MockAdvertisementService{
		campaigns: make(map[uint64]*models.AdvertisementCampaign),
		slots:     make(map[uint64]*models.AdvertisementSlot),
		creatives: make(map[uint64]*models.AdvertisementCreative),
		nextID:    1,
	}
}

func (m *MockAdvertisementService) CreateCampaign(campaign *models.AdvertisementCampaign) error {
	campaign.ID = m.nextID
	m.nextID++
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockAdvertisementService) GetCampaign(id uint64) (*models.AdvertisementCampaign, error) {
	if campaign, exists := m.campaigns[id]; exists {
		return campaign, nil
	}
	return nil, models.ErrNotFound
}

func (m *MockAdvertisementService) UpdateCampaign(campaign *models.AdvertisementCampaign) error {
	if _, exists := m.campaigns[campaign.ID]; !exists {
		return models.ErrNotFound
	}
	campaign.UpdatedAt = time.Now()
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockAdvertisementService) DeleteCampaign(id uint64) error {
	if _, exists := m.campaigns[id]; !exists {
		return models.ErrNotFound
	}
	delete(m.campaigns, id)
	return nil
}

func (m *MockAdvertisementService) CreateSlot(slot *models.AdvertisementSlot) error {
	slot.ID = m.nextID
	m.nextID++
	slot.CreatedAt = time.Now()
	slot.UpdatedAt = time.Now()
	m.slots[slot.ID] = slot
	return nil
}

func (m *MockAdvertisementService) GetSlot(id uint64) (*models.AdvertisementSlot, error) {
	if slot, exists := m.slots[id]; exists {
		return slot, nil
	}
	return nil, models.ErrNotFound
}

func (m *MockAdvertisementService) GetSlotsByPageType(pageType, position string) ([]models.AdvertisementSlot, error) {
	var slots []models.AdvertisementSlot
	for _, slot := range m.slots {
		if slot.IsActive && (slot.PageType == pageType || slot.PageType == "all") {
			if position == "" || slot.Position == position {
				slots = append(slots, *slot)
			}
		}
	}
	return slots, nil
}

func (m *MockAdvertisementService) GetAllSlots() ([]models.AdvertisementSlot, error) {
	var slots []models.AdvertisementSlot
	for _, slot := range m.slots {
		slots = append(slots, *slot)
	}
	return slots, nil
}

func (m *MockAdvertisementService) CreateCreative(creative *models.AdvertisementCreative) error {
	creative.ID = m.nextID
	m.nextID++
	creative.CreatedAt = time.Now()
	creative.UpdatedAt = time.Now()
	m.creatives[creative.ID] = creative
	return nil
}

func (m *MockAdvertisementService) GetCreativesByCampaign(campaignID uint64) ([]models.AdvertisementCreative, error) {
	var creatives []models.AdvertisementCreative
	for _, creative := range m.creatives {
		if creative.CampaignID == campaignID && creative.IsActive {
			creatives = append(creatives, *creative)
		}
	}
	return creatives, nil
}

func (m *MockAdvertisementService) CreateTargeting(targeting *models.AdvertisementTargeting) error {
	return nil
}

func (m *MockAdvertisementService) GetTargetingByCampaign(campaignID uint64) ([]models.AdvertisementTargeting, error) {
	return []models.AdvertisementTargeting{}, nil
}

func (m *MockAdvertisementService) CreatePlacement(placement *models.AdvertisementPlacement) error {
	return nil
}

func (m *MockAdvertisementService) GetAdvertisements(request *models.AdvertisementRequest) (*models.AdvertisementResponse, error) {
	return &models.AdvertisementResponse{
		Ads: []models.AdvertisementAd{
			{
				ID:            "test-ad-1",
				PlacementID:   1,
				CampaignID:    1,
				SlotID:        1,
				CreativeID:    1,
				Type:          "image",
				Content:       "https://example.com/image.jpg",
				AltText:       "Test Ad",
				ClickURL:      "https://example.com",
				Width:         &[]int{300}[0],
				Height:        &[]int{250}[0],
				LazyLoad:      true,
				TrackingURL:   "http://localhost/api/v1/ads/impression/test-ad-1",
				ClickTrackURL: "http://localhost/api/v1/ads/click/test-ad-1",
				Weight:        1,
				Priority:      5,
			},
		},
		TotalAds:      1,
		RequestID:     "test-request-id",
		CacheTTL:      300,
		PerformanceMs: 10,
	}, nil
}

func (m *MockAdvertisementService) RecordImpression(adID string, request *models.AdvertisementRequest) error {
	return nil
}

func (m *MockAdvertisementService) RecordClick(adID string, request *models.AdvertisementRequest) error {
	return nil
}

func (m *MockAdvertisementService) GetCampaignStats(campaignID uint64, startDate, endDate time.Time) (*models.AdvertisementStats, error) {
	return &models.AdvertisementStats{
		CampaignID:  campaignID,
		Impressions: 1000,
		Clicks:      50,
		CTR:         5.0,
		StartDate:   startDate,
		EndDate:     endDate,
	}, nil
}

func (m *MockAdvertisementService) GetPerformanceReport(startDate, endDate time.Time, campaignIDs []uint64) ([]models.AdvertisementPerformanceReport, error) {
	return []models.AdvertisementPerformanceReport{
		{
			CampaignID:   1,
			CampaignName: "Test Campaign",
			Impressions:  1000,
			Clicks:       50,
			CTR:          5.0,
			Date:         startDate,
		},
	}, nil
}

func (m *MockAdvertisementService) ValidateAdPerformance(creative *models.AdvertisementCreative) error {
	if creative.FileSize != nil && *creative.FileSize > 100*1024 {
		return models.NewValidationError("file_size", "File size exceeds limit")
	}
	return nil
}

func (m *MockAdvertisementService) GetDeviceType(userAgent string) string {
	return "desktop"
}

func (m *MockAdvertisementService) GetClientIP(remoteAddr, xForwardedFor, xRealIP string) string {
	return "127.0.0.1"
}

func setupTestRouter() (*gin.Engine, *MockAdvertisementService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockService := NewMockAdvertisementService()
	handlers := NewAdvertisementHandlers(mockService)
	
	// Add routes
	api := router.Group("/api/v1/ads")
	{
		// Campaign routes
		api.POST("/campaigns", handlers.CreateCampaign)
		api.GET("/campaigns/:id", handlers.GetCampaign)
		api.PUT("/campaigns/:id", handlers.UpdateCampaign)
		api.DELETE("/campaigns/:id", handlers.DeleteCampaign)
		
		// Slot routes
		api.POST("/slots", handlers.CreateSlot)
		api.GET("/slots", handlers.GetSlots)
		api.GET("/slots/:id", handlers.GetSlot)
		
		// Creative routes
		api.POST("/creatives", handlers.CreateCreative)
		api.GET("/campaigns/:campaign_id/creatives", handlers.GetCreativesByCampaign)
		
		// Targeting routes
		api.POST("/targeting", handlers.CreateTargeting)
		api.GET("/campaigns/:campaign_id/targeting", handlers.GetTargetingByCampaign)
		
		// Placement routes
		api.POST("/placements", handlers.CreatePlacement)
		
		// Ad serving routes
		api.POST("/serve", handlers.GetAdvertisements)
		
		// Tracking routes
		api.POST("/impression/:ad_id", handlers.RecordImpression)
		api.GET("/click/:ad_id", handlers.RecordClick)
		
		// Analytics routes
		api.GET("/campaigns/:campaign_id/stats", handlers.GetCampaignStats)
		api.GET("/reports/performance", handlers.GetPerformanceReport)
	}
	
	return router, mockService
}

func TestAdvertisementHandlers_CreateCampaign(t *testing.T) {
	router, _ := setupTestRouter()

	campaign := models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}

	jsonData, _ := json.Marshal(campaign)
	req, _ := http.NewRequest("POST", "/api/v1/ads/campaigns", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.AdvertisementCampaign
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Name != campaign.Name {
		t.Errorf("Expected campaign name %s, got %s", campaign.Name, response.Name)
	}

	if response.ID == 0 {
		t.Error("Expected campaign ID to be set")
	}
}

func TestAdvertisementHandlers_GetCampaign(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create a test campaign
	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}
	mockService.CreateCampaign(campaign)

	req, _ := http.NewRequest("GET", "/api/v1/ads/campaigns/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AdvertisementCampaign
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Name != campaign.Name {
		t.Errorf("Expected campaign name %s, got %s", campaign.Name, response.Name)
	}
}

func TestAdvertisementHandlers_CreateSlot(t *testing.T) {
	router, _ := setupTestRouter()

	slot := models.AdvertisementSlot{
		Name:     "Test Slot",
		Slug:     "test-slot",
		PageType: "homepage",
		Position: "header",
		IsActive: true,
		LazyLoad: false,
	}

	jsonData, _ := json.Marshal(slot)
	req, _ := http.NewRequest("POST", "/api/v1/ads/slots", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.AdvertisementSlot
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Name != slot.Name {
		t.Errorf("Expected slot name %s, got %s", slot.Name, response.Name)
	}
}

func TestAdvertisementHandlers_GetSlots(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create test slots
	slot1 := &models.AdvertisementSlot{
		Name:     "Homepage Header",
		PageType: "homepage",
		Position: "header",
		IsActive: true,
	}
	slot2 := &models.AdvertisementSlot{
		Name:     "Article Sidebar",
		PageType: "article",
		Position: "sidebar",
		IsActive: true,
	}
	mockService.CreateSlot(slot1)
	mockService.CreateSlot(slot2)

	// Test getting all slots
	req, _ := http.NewRequest("GET", "/api/v1/ads/slots", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []models.AdvertisementSlot
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 slots, got %d", len(response))
	}

	// Test filtering by page type
	req, _ = http.NewRequest("GET", "/api/v1/ads/slots?page_type=homepage", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 slot for homepage, got %d", len(response))
	}

	if response[0].PageType != "homepage" {
		t.Errorf("Expected homepage slot, got %s", response[0].PageType)
	}
}

func TestAdvertisementHandlers_CreateCreative(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create a campaign first
	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}
	mockService.CreateCampaign(campaign)

	creative := models.AdvertisementCreative{
		CampaignID: campaign.ID,
		Name:       "Test Creative",
		Type:       "image",
		Content:    "https://example.com/image.jpg",
		AltText:    "Test Ad",
		Width:      &[]int{300}[0],
		Height:     &[]int{250}[0],
		FileSize:   &[]int{50000}[0], // 50KB
		IsActive:   true,
	}

	jsonData, _ := json.Marshal(creative)
	req, _ := http.NewRequest("POST", "/api/v1/ads/creatives", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.AdvertisementCreative
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Name != creative.Name {
		t.Errorf("Expected creative name %s, got %s", creative.Name, response.Name)
	}
}

func TestAdvertisementHandlers_GetAdvertisements(t *testing.T) {
	router, _ := setupTestRouter()

	request := models.AdvertisementRequest{
		PageType:   "homepage",
		Position:   "header",
		MaxAds:     3,
		DeviceType: "desktop",
	}

	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/ads/serve", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Test User Agent")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AdvertisementResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.TotalAds != 1 {
		t.Errorf("Expected 1 ad, got %d", response.TotalAds)
	}

	if len(response.Ads) != 1 {
		t.Errorf("Expected 1 ad in response, got %d", len(response.Ads))
	}

	ad := response.Ads[0]
	if ad.Type != "image" {
		t.Errorf("Expected ad type 'image', got %s", ad.Type)
	}

	// Check cache headers
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=300" {
		t.Errorf("Expected cache control header 'public, max-age=300', got %s", cacheControl)
	}
}

func TestAdvertisementHandlers_RecordImpression(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("POST", "/api/v1/ads/impression/test-ad-id", nil)
	req.Header.Set("User-Agent", "Test User Agent")
	req.Header.Set("Referer", "https://example.com")
	req.Header.Set("X-Page-URL", "https://example.com/article")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/gif" {
		t.Errorf("Expected content type 'image/gif', got %s", contentType)
	}

	// Check cache headers
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache, no-store, must-revalidate" {
		t.Errorf("Expected no-cache header, got %s", cacheControl)
	}

	// Check that response is a 1x1 pixel GIF
	if len(w.Body.Bytes()) == 0 {
		t.Error("Expected pixel data in response body")
	}
}

func TestAdvertisementHandlers_RecordClick(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/ads/click/test-ad-id?url=https://example.com", nil)
	req.Header.Set("User-Agent", "Test User Agent")
	req.Header.Set("Referer", "https://example.com")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}

	// Check redirect location
	location := w.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("Expected redirect to 'https://example.com', got %s", location)
	}
}

func TestAdvertisementHandlers_GetCampaignStats(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create a test campaign
	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}
	mockService.CreateCampaign(campaign)

	req, _ := http.NewRequest("GET", "/api/v1/ads/campaigns/1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.AdvertisementStats
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.CampaignID != 1 {
		t.Errorf("Expected campaign ID 1, got %d", response.CampaignID)
	}

	if response.Impressions != 1000 {
		t.Errorf("Expected 1000 impressions, got %d", response.Impressions)
	}

	if response.Clicks != 50 {
		t.Errorf("Expected 50 clicks, got %d", response.Clicks)
	}

	if response.CTR != 5.0 {
		t.Errorf("Expected CTR 5.0, got %f", response.CTR)
	}
}

func TestAdvertisementHandlers_GetPerformanceReport(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/ads/reports/performance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []models.AdvertisementPerformanceReport
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 report, got %d", len(response))
	}

	report := response[0]
	if report.CampaignID != 1 {
		t.Errorf("Expected campaign ID 1, got %d", report.CampaignID)
	}

	if report.CampaignName != "Test Campaign" {
		t.Errorf("Expected campaign name 'Test Campaign', got %s", report.CampaignName)
	}
}

func TestAdvertisementHandlers_ValidationErrors(t *testing.T) {
	router, _ := setupTestRouter()

	// Test invalid campaign creation
	invalidCampaign := models.AdvertisementCampaign{
		// Missing required fields
		Priority: 15, // Invalid priority
	}

	jsonData, _ := json.Marshal(invalidCampaign)
	req, _ := http.NewRequest("POST", "/api/v1/ads/campaigns", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for invalid campaign, got %d", http.StatusInternalServerError, w.Code)
	}

	// Test invalid creative with large file size
	creative := models.AdvertisementCreative{
		CampaignID: 1,
		Name:       "Large Creative",
		Type:       "image",
		Content:    "https://example.com/large-image.jpg",
		FileSize:   &[]int{200000}[0], // 200KB - exceeds limit
		IsActive:   true,
	}

	jsonData, _ = json.Marshal(creative)
	req, _ = http.NewRequest("POST", "/api/v1/ads/creatives", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for oversized creative, got %d", http.StatusBadRequest, w.Code)
	}
}