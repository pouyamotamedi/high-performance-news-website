package services

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// AdvertisementService handles advertisement business logic
type AdvertisementService struct {
	repo         *repositories.AdvertisementRepository
	cacheService CacheService
	baseURL      string
}

// NewAdvertisementService creates a new advertisement service
func NewAdvertisementService(repo *repositories.AdvertisementRepository, cache CacheService, baseURL string) *AdvertisementService {
	return &AdvertisementService{
		repo:         repo,
		cacheService: cache,
		baseURL:      baseURL,
	}
}

// Campaign management

// CreateCampaign creates a new advertisement campaign
func (s *AdvertisementService) CreateCampaign(campaign *models.AdvertisementCampaign) error {
	// Validate campaign
	if err := campaign.IsValidCampaign(); err != nil {
		return err
	}
	
	// Set defaults
	if campaign.Status == "" {
		campaign.Status = "draft"
	}
	if campaign.Priority == 0 {
		campaign.Priority = 1
	}
	
	// Create campaign
	if err := s.repo.CreateCampaign(campaign); err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// GetCampaign retrieves a campaign by ID
func (s *AdvertisementService) GetCampaign(id uint64) (*models.AdvertisementCampaign, error) {
	campaign, err := s.repo.GetCampaignByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	
	// Load related data
	creatives, _ := s.repo.GetCreativesByCampaign(id)
	campaign.Creatives = creatives
	
	targeting, _ := s.repo.GetTargetingByCampaign(id)
	campaign.Targeting = targeting
	
	return campaign, nil
}

// UpdateCampaign updates an existing campaign
func (s *AdvertisementService) UpdateCampaign(campaign *models.AdvertisementCampaign) error {
	// Validate campaign
	if err := campaign.IsValidCampaign(); err != nil {
		return err
	}
	
	// Update campaign
	if err := s.repo.UpdateCampaign(campaign); err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// DeleteCampaign deletes a campaign
func (s *AdvertisementService) DeleteCampaign(id uint64) error {
	if err := s.repo.DeleteCampaign(id); err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// Slot management

// CreateSlot creates a new advertisement slot
func (s *AdvertisementService) CreateSlot(slot *models.AdvertisementSlot) error {
	// Validate slot
	if err := slot.IsValidSlot(); err != nil {
		return err
	}
	
	// Generate slug if not provided
	if slot.Slug == "" {
		slot.Slug = s.generateSlug(slot.Name)
	}
	
	// Set defaults
	if slot.LazyLoad == false && slot.Position != "header" && slot.Position != "content-top" {
		slot.LazyLoad = true // Default to lazy load for below-fold ads
	}
	
	// Create slot
	if err := s.repo.CreateSlot(slot); err != nil {
		return fmt.Errorf("failed to create slot: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// GetSlot retrieves a slot by ID
func (s *AdvertisementService) GetSlot(id uint64) (*models.AdvertisementSlot, error) {
	return s.repo.GetSlotByID(id)
}

// GetSlotsByPageType retrieves slots for a specific page type
func (s *AdvertisementService) GetSlotsByPageType(pageType, position string) ([]models.AdvertisementSlot, error) {
	return s.repo.GetSlotsByPageType(pageType, position)
}

// GetAllSlots retrieves all slots
func (s *AdvertisementService) GetAllSlots() ([]models.AdvertisementSlot, error) {
	return s.repo.GetAllSlots()
}

// Creative management

// CreateCreative creates a new advertisement creative
func (s *AdvertisementService) CreateCreative(creative *models.AdvertisementCreative) error {
	// Validate creative
	if err := creative.IsValidCreative(); err != nil {
		return err
	}
	
	// Set defaults
	if creative.IsActive == false {
		creative.IsActive = true
	}
	
	// Calculate file size for performance tracking (estimate for HTML/script)
	if creative.FileSize == nil {
		size := len(creative.Content)
		creative.FileSize = &size
	}
	
	// Create creative
	if err := s.repo.CreateCreative(creative); err != nil {
		return fmt.Errorf("failed to create creative: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// GetCreativesByCampaign retrieves creatives for a campaign
func (s *AdvertisementService) GetCreativesByCampaign(campaignID uint64) ([]models.AdvertisementCreative, error) {
	return s.repo.GetCreativesByCampaign(campaignID)
}

// Targeting management

// CreateTargeting creates targeting rules for a campaign
func (s *AdvertisementService) CreateTargeting(targeting *models.AdvertisementTargeting) error {
	if err := s.repo.CreateTargeting(targeting); err != nil {
		return fmt.Errorf("failed to create targeting: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// GetTargetingByCampaign retrieves targeting rules for a campaign
func (s *AdvertisementService) GetTargetingByCampaign(campaignID uint64) ([]models.AdvertisementTargeting, error) {
	return s.repo.GetTargetingByCampaign(campaignID)
}

// Placement management

// CreatePlacement creates a new advertisement placement
func (s *AdvertisementService) CreatePlacement(placement *models.AdvertisementPlacement) error {
	// Set defaults
	if placement.Weight == 0 {
		placement.Weight = 1
	}
	if placement.IsActive == false {
		placement.IsActive = true
	}
	
	if err := s.repo.CreatePlacement(placement); err != nil {
		return fmt.Errorf("failed to create placement: %w", err)
	}
	
	// Clear cache
	s.clearAdvertisementCache()
	
	return nil
}

// Ad serving and targeting

// GetAdvertisements retrieves advertisements for a specific request
func (s *AdvertisementService) GetAdvertisements(request *models.AdvertisementRequest) (*models.AdvertisementResponse, error) {
	startTime := time.Now()
	
	// Generate request ID for tracking
	requestID := s.generateRequestID(request)
	
	// Check cache first
	cacheKey := fmt.Sprintf("ads:%s:%s:%s", request.PageType, request.Position, s.hashRequest(request))
	if cached, err := s.cacheService.Get(cacheKey); err == nil && len(cached) > 0 {
		// Return cached response (implement cache deserialization if needed)
		// For now, skip cache and always generate fresh ads for accuracy
	}
	
	// Get slots for the page type and position
	slots, err := s.repo.GetSlotsByPageType(request.PageType, request.Position)
	if err != nil {
		return nil, fmt.Errorf("failed to get slots: %w", err)
	}
	
	var allAds []models.AdvertisementAd
	
	// Get advertisements for each slot
	for _, slot := range slots {
		// Get placements for this slot with targeting
		placements, err := s.repo.GetPlacementsBySlot(slot.ID, request.CategoryID, request.TagIDs)
		if err != nil {
			continue // Skip this slot on error
		}
		
		// Filter placements based on additional targeting and A/B testing
		filteredPlacements := s.filterPlacements(placements, request)
		
		// Select ads based on weight and rotation
		selectedPlacements := s.selectPlacements(filteredPlacements, request.MaxAds)
		
		// Convert placements to ads
		for _, placement := range selectedPlacements {
			ad := s.placementToAd(placement, requestID, &slot)
			allAds = append(allAds, ad)
		}
	}
	
	// Limit total ads
	if len(allAds) > request.MaxAds {
		allAds = allAds[:request.MaxAds]
	}
	
	// Create response
	response := &models.AdvertisementResponse{
		Ads:           allAds,
		TotalAds:      len(allAds),
		RequestID:     requestID,
		CacheTTL:      300, // 5 minutes cache
		PerformanceMs: int(time.Since(startTime).Milliseconds()),
	}
	
	// Cache response (implement cache serialization if needed)
	// s.cacheService.Set(cacheKey, responseBytes, 5*time.Minute)
	
	return response, nil
}

// filterPlacements filters placements based on targeting rules and device type
func (s *AdvertisementService) filterPlacements(placements []models.AdvertisementPlacement, request *models.AdvertisementRequest) []models.AdvertisementPlacement {
	var filtered []models.AdvertisementPlacement
	
	for _, placement := range placements {
		// Skip excluded campaigns
		if s.isExcluded(placement.CampaignID, request.ExcludeAds) {
			continue
		}
		
		// Check device targeting (if implemented)
		if request.DeviceType != "" {
			// Add device targeting logic here if needed
		}
		
		// Check time-based targeting (if implemented)
		// Add time targeting logic here if needed
		
		// Check budget constraints (if implemented)
		// Add budget checking logic here if needed
		
		filtered = append(filtered, placement)
	}
	
	return filtered
}

// selectPlacements selects placements based on weight and A/B testing
func (s *AdvertisementService) selectPlacements(placements []models.AdvertisementPlacement, maxAds int) []models.AdvertisementPlacement {
	if len(placements) == 0 {
		return placements
	}
	
	// Sort by priority and weight
	// For simplicity, we'll use a weighted random selection
	totalWeight := 0
	for _, p := range placements {
		totalWeight += p.Weight * p.Campaign.Priority
	}
	
	if totalWeight == 0 {
		// If no weights, return first maxAds placements
		if len(placements) > maxAds {
			return placements[:maxAds]
		}
		return placements
	}
	
	var selected []models.AdvertisementPlacement
	usedCampaigns := make(map[uint64]bool)
	
	for len(selected) < maxAds && len(selected) < len(placements) {
		// Weighted random selection
		randWeight := rand.Intn(totalWeight)
		currentWeight := 0
		
		for _, placement := range placements {
			// Skip if campaign already selected (for diversity)
			if usedCampaigns[placement.CampaignID] {
				continue
			}
			
			currentWeight += placement.Weight * placement.Campaign.Priority
			if randWeight < currentWeight {
				selected = append(selected, placement)
				usedCampaigns[placement.CampaignID] = true
				break
			}
		}
		
		// If all campaigns are used, allow duplicates
		if len(selected) == 0 {
			selected = append(selected, placements[0])
			break
		}
	}
	
	return selected
}

// placementToAd converts a placement to an advertisement ad
func (s *AdvertisementService) placementToAd(placement models.AdvertisementPlacement, requestID string, slot *models.AdvertisementSlot) models.AdvertisementAd {
	adID := fmt.Sprintf("%s-%d-%d", requestID, placement.ID, time.Now().UnixNano())
	
	return models.AdvertisementAd{
		ID:            adID,
		PlacementID:   placement.ID,
		CampaignID:    placement.CampaignID,
		SlotID:        placement.SlotID,
		CreativeID:    placement.CreativeID,
		Type:          placement.Creative.Type,
		Content:       placement.Creative.Content,
		AltText:       placement.Creative.AltText,
		ClickURL:      placement.Creative.ClickURL,
		Width:         placement.Creative.Width,
		Height:        placement.Creative.Height,
		LazyLoad:      slot.LazyLoad,
		TrackingURL:   fmt.Sprintf("%s/api/v1/ads/impression/%s", s.baseURL, adID),
		ClickTrackURL: fmt.Sprintf("%s/api/v1/ads/click/%s", s.baseURL, adID),
		Weight:        placement.Weight,
		Priority:      placement.Campaign.Priority,
	}
}

// Tracking

// RecordImpression records an advertisement impression
func (s *AdvertisementService) RecordImpression(adID string, request *models.AdvertisementRequest) error {
	// Parse ad ID to get placement info
	placementID, campaignID, slotID, creativeID, err := s.parseAdID(adID)
	if err != nil {
		return fmt.Errorf("invalid ad ID: %w", err)
	}
	
	impression := &models.AdvertisementImpression{
		PlacementID: placementID,
		CampaignID:  campaignID,
		SlotID:      slotID,
		CreativeID:  creativeID,
		IPAddress:   request.IPAddress,
		UserAgent:   request.UserAgent,
		Referer:     request.Referer,
		PageURL:     request.PageURL,
		DeviceType:  request.DeviceType,
	}
	
	// Record impression asynchronously for performance
	go func() {
		if err := s.repo.RecordImpression(impression); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to record impression: %v\n", err)
		}
	}()
	
	return nil
}

// RecordClick records an advertisement click
func (s *AdvertisementService) RecordClick(adID string, request *models.AdvertisementRequest) error {
	// Parse ad ID to get placement info
	placementID, campaignID, slotID, creativeID, err := s.parseAdID(adID)
	if err != nil {
		return fmt.Errorf("invalid ad ID: %w", err)
	}
	
	click := &models.AdvertisementClick{
		PlacementID: placementID,
		CampaignID:  campaignID,
		SlotID:      slotID,
		CreativeID:  creativeID,
		IPAddress:   request.IPAddress,
		UserAgent:   request.UserAgent,
		Referer:     request.Referer,
		PageURL:     request.PageURL,
		DeviceType:  request.DeviceType,
	}
	
	// Record click asynchronously for performance
	go func() {
		if err := s.repo.RecordClick(click); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to record click: %v\n", err)
		}
	}()
	
	return nil
}

// Analytics and reporting

// GetCampaignStats retrieves statistics for a campaign
func (s *AdvertisementService) GetCampaignStats(campaignID uint64, startDate, endDate time.Time) (*models.AdvertisementStats, error) {
	return s.repo.GetCampaignStats(campaignID, startDate, endDate)
}

// GetPerformanceReport generates a performance report
func (s *AdvertisementService) GetPerformanceReport(startDate, endDate time.Time, campaignIDs []uint64) ([]models.AdvertisementPerformanceReport, error) {
	return s.repo.GetPerformanceReport(startDate, endDate, campaignIDs)
}

// Helper methods

// generateRequestID generates a unique request ID
func (s *AdvertisementService) generateRequestID(request *models.AdvertisementRequest) string {
	data := fmt.Sprintf("%s-%s-%s-%d", request.PageType, request.Position, request.IPAddress, time.Now().UnixNano())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)[:16]
}

// hashRequest creates a hash of the request for caching
func (s *AdvertisementService) hashRequest(request *models.AdvertisementRequest) string {
	data := fmt.Sprintf("%s-%s-%v-%v-%s", 
		request.PageType, request.Position, request.CategoryID, request.TagIDs, request.DeviceType)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)[:8]
}

// parseAdID parses an ad ID to extract placement information
func (s *AdvertisementService) parseAdID(adID string) (placementID, campaignID, slotID, creativeID uint64, err error) {
	// This is a simplified implementation
	// In a real system, you'd want to encode/decode this more securely
	parts := strings.Split(adID, "-")
	if len(parts) < 3 {
		return 0, 0, 0, 0, fmt.Errorf("invalid ad ID format")
	}
	
	// For now, return dummy values - implement proper parsing based on your ad ID format
	placementID = 1
	campaignID = 1
	slotID = 1
	creativeID = 1
	
	return placementID, campaignID, slotID, creativeID, nil
}

// generateSlug generates a URL-friendly slug from a name
func (s *AdvertisementService) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove special characters (simplified)
	return slug
}

// isExcluded checks if a campaign ID is in the exclude list
func (s *AdvertisementService) isExcluded(campaignID uint64, excludeAds []uint64) bool {
	for _, id := range excludeAds {
		if id == campaignID {
			return true
		}
	}
	return false
}

// clearAdvertisementCache clears advertisement-related cache
func (s *AdvertisementService) clearAdvertisementCache() {
	// Clear all advertisement cache patterns
	s.cacheService.DeletePattern("ads:*")
}

// GetDeviceType determines device type from user agent
func (s *AdvertisementService) GetDeviceType(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	
	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") || strings.Contains(userAgent, "iphone") {
		return "mobile"
	}
	if strings.Contains(userAgent, "tablet") || strings.Contains(userAgent, "ipad") {
		return "tablet"
	}
	return "desktop"
}

// GetClientIP extracts client IP from request
func (s *AdvertisementService) GetClientIP(remoteAddr, xForwardedFor, xRealIP string) string {
	// Check X-Forwarded-For header first
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	
	// Check X-Real-IP header
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}
	
	// Fall back to remote address
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	
	return remoteAddr
}

// ValidateAdPerformance checks if ads meet Core Web Vitals requirements
func (s *AdvertisementService) ValidateAdPerformance(creative *models.AdvertisementCreative) error {
	// Check file size for performance (requirement 28.1)
	if creative.FileSize != nil && *creative.FileSize > 100*1024 { // 100KB limit
		return fmt.Errorf("creative file size exceeds performance limit (100KB)")
	}
	
	// Check dimensions for layout shift prevention
	if creative.Type == "image" && (creative.Width == nil || creative.Height == nil) {
		return fmt.Errorf("image creatives must have width and height specified to prevent layout shift")
	}
	
	// Validate script content for performance
	if creative.Type == "script" {
		if strings.Contains(creative.Content, "document.write") {
			return fmt.Errorf("script creatives cannot use document.write (blocks rendering)")
		}
	}
	
	return nil
}