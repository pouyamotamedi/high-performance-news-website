package services

import (
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// MockAdvertisementRepository for testing
type MockAdvertisementRepository struct {
	campaigns   map[uint64]*models.AdvertisementCampaign
	slots       map[uint64]*models.AdvertisementSlot
	creatives   map[uint64]*models.AdvertisementCreative
	placements  []models.AdvertisementPlacement
	impressions []models.AdvertisementImpression
	clicks      []models.AdvertisementClick
	nextID      uint64
}

func NewMockAdvertisementRepository() *MockAdvertisementRepository {
	return &MockAdvertisementRepository{
		campaigns:  make(map[uint64]*models.AdvertisementCampaign),
		slots:      make(map[uint64]*models.AdvertisementSlot),
		creatives:  make(map[uint64]*models.AdvertisementCreative),
		placements: []models.AdvertisementPlacement{},
		nextID:     1,
	}
}

func (m *MockAdvertisementRepository) CreateCampaign(campaign *models.AdvertisementCampaign) error {
	campaign.ID = m.nextID
	m.nextID++
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockAdvertisementRepository) GetCampaignByID(id uint64) (*models.AdvertisementCampaign, error) {
	if campaign, exists := m.campaigns[id]; exists {
		return campaign, nil
	}
	return nil, models.ErrNotFound
}

func (m *MockAdvertisementRepository) GetActiveCampaigns() ([]models.AdvertisementCampaign, error) {
	var active []models.AdvertisementCampaign
	for _, campaign := range m.campaigns {
		if campaign.Status == "active" {
			active = append(active, *campaign)
		}
	}
	return active, nil
}

func (m *MockAdvertisementRepository) UpdateCampaign(campaign *models.AdvertisementCampaign) error {
	if _, exists := m.campaigns[campaign.ID]; !exists {
		return models.ErrNotFound
	}
	campaign.UpdatedAt = time.Now()
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockAdvertisementRepository) DeleteCampaign(id uint64) error {
	if _, exists := m.campaigns[id]; !exists {
		return models.ErrNotFound
	}
	delete(m.campaigns, id)
	return nil
}

func (m *MockAdvertisementRepository) CreateSlot(slot *models.AdvertisementSlot) error {
	slot.ID = m.nextID
	m.nextID++
	slot.CreatedAt = time.Now()
	slot.UpdatedAt = time.Now()
	m.slots[slot.ID] = slot
	return nil
}

func (m *MockAdvertisementRepository) GetSlotByID(id uint64) (*models.AdvertisementSlot, error) {
	if slot, exists := m.slots[id]; exists {
		return slot, nil
	}
	return nil, models.ErrNotFound
}

func (m *MockAdvertisementRepository) GetSlotsByPageType(pageType, position string) ([]models.AdvertisementSlot, error) {
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

func (m *MockAdvertisementRepository) GetAllSlots() ([]models.AdvertisementSlot, error) {
	var slots []models.AdvertisementSlot
	for _, slot := range m.slots {
		slots = append(slots, *slot)
	}
	return slots, nil
}

func (m *MockAdvertisementRepository) CreateCreative(creative *models.AdvertisementCreative) error {
	creative.ID = m.nextID
	m.nextID++
	creative.CreatedAt = time.Now()
	creative.UpdatedAt = time.Now()
	m.creatives[creative.ID] = creative
	return nil
}

func (m *MockAdvertisementRepository) GetCreativesByCampaign(campaignID uint64) ([]models.AdvertisementCreative, error) {
	var creatives []models.AdvertisementCreative
	for _, creative := range m.creatives {
		if creative.CampaignID == campaignID && creative.IsActive {
			creatives = append(creatives, *creative)
		}
	}
	return creatives, nil
}

func (m *MockAdvertisementRepository) CreateTargeting(targeting *models.AdvertisementTargeting) error {
	targeting.ID = m.nextID
	m.nextID++
	targeting.CreatedAt = time.Now()
	return nil
}

func (m *MockAdvertisementRepository) GetTargetingByCampaign(campaignID uint64) ([]models.AdvertisementTargeting, error) {
	return []models.AdvertisementTargeting{}, nil
}

func (m *MockAdvertisementRepository) CreatePlacement(placement *models.AdvertisementPlacement) error {
	placement.ID = m.nextID
	m.nextID++
	placement.CreatedAt = time.Now()
	m.placements = append(m.placements, *placement)
	return nil
}

func (m *MockAdvertisementRepository) GetPlacementsBySlot(slotID uint64, categoryID *uint64, tagIDs []uint64) ([]models.AdvertisementPlacement, error) {
	var placements []models.AdvertisementPlacement
	for _, placement := range m.placements {
		if placement.SlotID == slotID && placement.IsActive {
			// Add campaign and creative data
			if campaign, exists := m.campaigns[placement.CampaignID]; exists {
				placement.Campaign = campaign
			}
			if creative, exists := m.creatives[placement.CreativeID]; exists {
				placement.Creative = creative
			}
			if slot, exists := m.slots[placement.SlotID]; exists {
				placement.Slot = slot
			}
			placements = append(placements, placement)
		}
	}
	return placements, nil
}

func (m *MockAdvertisementRepository) RecordImpression(impression *models.AdvertisementImpression) error {
	impression.ID = m.nextID
	m.nextID++
	impression.CreatedAt = time.Now()
	m.impressions = append(m.impressions, *impression)
	return nil
}

func (m *MockAdvertisementRepository) RecordClick(click *models.AdvertisementClick) error {
	click.ID = m.nextID
	m.nextID++
	click.CreatedAt = time.Now()
	m.clicks = append(m.clicks, *click)
	return nil
}

func (m *MockAdvertisementRepository) GetCampaignStats(campaignID uint64, startDate, endDate time.Time) (*models.AdvertisementStats, error) {
	impressions := uint64(0)
	clicks := uint64(0)
	
	for _, imp := range m.impressions {
		if imp.CampaignID == campaignID && imp.CreatedAt.After(startDate) && imp.CreatedAt.Before(endDate) {
			impressions++
		}
	}
	
	for _, click := range m.clicks {
		if click.CampaignID == campaignID && click.CreatedAt.After(startDate) && click.CreatedAt.Before(endDate) {
			clicks++
		}
	}
	
	ctr := float64(0)
	if impressions > 0 {
		ctr = (float64(clicks) / float64(impressions)) * 100
	}
	
	return &models.AdvertisementStats{
		CampaignID:  campaignID,
		Impressions: impressions,
		Clicks:      clicks,
		CTR:         ctr,
		StartDate:   startDate,
		EndDate:     endDate,
	}, nil
}

func (m *MockAdvertisementRepository) GetPerformanceReport(startDate, endDate time.Time, campaignIDs []uint64) ([]models.AdvertisementPerformanceReport, error) {
	return []models.AdvertisementPerformanceReport{}, nil
}

// MockCacheService for testing
type MockCacheService struct {
	data map[string][]byte
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
	}
}

func (m *MockCacheService) Get(key string) ([]byte, error) {
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, models.ErrNotFound
}

func (m *MockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCacheService) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCacheService) DeletePattern(pattern string) error {
	// Simple pattern matching for testing
	for key := range m.data {
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				delete(m.data, key)
			}
		}
	}
	return nil
}

func (m *MockCacheService) Exists(key string) bool {
	_, exists := m.data[key]
	return exists
}

func TestAdvertisementService_CreateCampaign(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}

	err := service.CreateCampaign(campaign)
	if err != nil {
		t.Errorf("CreateCampaign() error = %v", err)
		return
	}

	if campaign.ID == 0 {
		t.Error("Campaign ID should be set after creation")
	}

	// Verify campaign was stored
	stored, err := service.GetCampaign(campaign.ID)
	if err != nil {
		t.Errorf("GetCampaign() error = %v", err)
		return
	}

	if stored.Name != campaign.Name {
		t.Errorf("Campaign name mismatch: got %v, want %v", stored.Name, campaign.Name)
	}
}

func TestAdvertisementService_CreateSlot(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	slot := &models.AdvertisementSlot{
		Name:     "Test Slot",
		Slug:     "test-slot",
		PageType: "homepage",
		Position: "header",
		IsActive: true,
		LazyLoad: false,
	}

	err := service.CreateSlot(slot)
	if err != nil {
		t.Errorf("CreateSlot() error = %v", err)
		return
	}

	if slot.ID == 0 {
		t.Error("Slot ID should be set after creation")
	}

	// Verify slot was stored
	stored, err := service.GetSlot(slot.ID)
	if err != nil {
		t.Errorf("GetSlot() error = %v", err)
		return
	}

	if stored.Name != slot.Name {
		t.Errorf("Slot name mismatch: got %v, want %v", stored.Name, slot.Name)
	}
}

func TestAdvertisementService_CreateCreative(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	// First create a campaign
	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}
	service.CreateCampaign(campaign)

	creative := &models.AdvertisementCreative{
		CampaignID: campaign.ID,
		Name:       "Test Creative",
		Type:       "image",
		Content:    "https://example.com/image.jpg",
		IsActive:   true,
	}

	err := service.CreateCreative(creative)
	if err != nil {
		t.Errorf("CreateCreative() error = %v", err)
		return
	}

	if creative.ID == 0 {
		t.Error("Creative ID should be set after creation")
	}

	// Verify creative was stored
	creatives, err := service.GetCreativesByCampaign(campaign.ID)
	if err != nil {
		t.Errorf("GetCreativesByCampaign() error = %v", err)
		return
	}

	if len(creatives) != 1 {
		t.Errorf("Expected 1 creative, got %d", len(creatives))
		return
	}

	if creatives[0].Name != creative.Name {
		t.Errorf("Creative name mismatch: got %v, want %v", creatives[0].Name, creative.Name)
	}
}

func TestAdvertisementService_GetAdvertisements(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	// Create test data
	campaign := &models.AdvertisementCampaign{
		Name:           "Test Campaign",
		AdvertiserName: "Test Advertiser",
		StartDate:      time.Now(),
		Priority:       5,
		Status:         "active",
	}
	service.CreateCampaign(campaign)

	slot := &models.AdvertisementSlot{
		Name:     "Test Slot",
		Slug:     "test-slot",
		PageType: "homepage",
		Position: "header",
		IsActive: true,
		LazyLoad: false,
	}
	service.CreateSlot(slot)

	creative := &models.AdvertisementCreative{
		CampaignID: campaign.ID,
		Name:       "Test Creative",
		Type:       "image",
		Content:    "https://example.com/image.jpg",
		IsActive:   true,
	}
	service.CreateCreative(creative)

	placement := &models.AdvertisementPlacement{
		CampaignID: campaign.ID,
		SlotID:     slot.ID,
		CreativeID: creative.ID,
		Weight:     1,
		IsActive:   true,
	}
	service.CreatePlacement(placement)

	// Test getting advertisements
	request := &models.AdvertisementRequest{
		PageType:   "homepage",
		Position:   "header",
		MaxAds:     3,
		DeviceType: "desktop",
		IPAddress:  "127.0.0.1",
		UserAgent:  "Test User Agent",
	}

	response, err := service.GetAdvertisements(request)
	if err != nil {
		t.Errorf("GetAdvertisements() error = %v", err)
		return
	}

	if response.TotalAds != 1 {
		t.Errorf("Expected 1 ad, got %d", response.TotalAds)
		return
	}

	ad := response.Ads[0]
	if ad.Type != "image" {
		t.Errorf("Expected ad type 'image', got %v", ad.Type)
	}

	if ad.Content != "https://example.com/image.jpg" {
		t.Errorf("Expected ad content 'https://example.com/image.jpg', got %v", ad.Content)
	}
}

func TestAdvertisementService_ValidateAdPerformance(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	tests := []struct {
		name     string
		creative models.AdvertisementCreative
		wantErr  bool
	}{
		{
			name: "valid image creative",
			creative: models.AdvertisementCreative{
				Type:     "image",
				Content:  "https://example.com/image.jpg",
				Width:    &[]int{300}[0],
				Height:   &[]int{250}[0],
				FileSize: &[]int{50000}[0], // 50KB
			},
			wantErr: false,
		},
		{
			name: "image creative too large",
			creative: models.AdvertisementCreative{
				Type:     "image",
				Content:  "https://example.com/image.jpg",
				Width:    &[]int{300}[0],
				Height:   &[]int{250}[0],
				FileSize: &[]int{200000}[0], // 200KB
			},
			wantErr: true,
		},
		{
			name: "image creative without dimensions",
			creative: models.AdvertisementCreative{
				Type:    "image",
				Content: "https://example.com/image.jpg",
			},
			wantErr: true,
		},
		{
			name: "script creative with document.write",
			creative: models.AdvertisementCreative{
				Type:    "script",
				Content: "document.write('<div>Ad</div>');",
			},
			wantErr: true,
		},
		{
			name: "valid script creative",
			creative: models.AdvertisementCreative{
				Type:    "script",
				Content: "console.log('Ad loaded');",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateAdPerformance(&tt.creative)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAdPerformance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdvertisementService_GetDeviceType(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{
			name:      "mobile iPhone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
			want:      "mobile",
		},
		{
			name:      "mobile Android",
			userAgent: "Mozilla/5.0 (Linux; Android 10; SM-G975F)",
			want:      "mobile",
		},
		{
			name:      "tablet iPad",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)",
			want:      "tablet",
		},
		{
			name:      "desktop Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			want:      "desktop",
		},
		{
			name:      "desktop Firefox",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0",
			want:      "desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GetDeviceType(tt.userAgent)
			if got != tt.want {
				t.Errorf("GetDeviceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdvertisementService_RecordImpression(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	request := &models.AdvertisementRequest{
		IPAddress: "127.0.0.1",
		UserAgent: "Test User Agent",
		Referer:   "https://example.com",
		PageURL:   "https://example.com/article",
	}

	// Test recording impression
	err := service.RecordImpression("test-ad-id", request)
	if err != nil {
		t.Errorf("RecordImpression() error = %v", err)
	}

	// Give some time for async operation
	time.Sleep(100 * time.Millisecond)

	// Verify impression was recorded
	if len(repo.impressions) != 1 {
		t.Errorf("Expected 1 impression, got %d", len(repo.impressions))
	}
}

func TestAdvertisementService_RecordClick(t *testing.T) {
	repo := NewMockAdvertisementRepository()
	cache := NewMockCacheService()
	service := NewAdvertisementService(repo, cache, "http://localhost")

	request := &models.AdvertisementRequest{
		IPAddress: "127.0.0.1",
		UserAgent: "Test User Agent",
		Referer:   "https://example.com",
		PageURL:   "https://example.com/article",
	}

	// Test recording click
	err := service.RecordClick("test-ad-id", request)
	if err != nil {
		t.Errorf("RecordClick() error = %v", err)
	}

	// Give some time for async operation
	time.Sleep(100 * time.Millisecond)

	// Verify click was recorded
	if len(repo.clicks) != 1 {
		t.Errorf("Expected 1 click, got %d", len(repo.clicks))
	}
}