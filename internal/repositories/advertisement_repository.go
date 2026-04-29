package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// AdvertisementRepository handles advertisement data operations
type AdvertisementRepository struct {
	db *sql.DB
}

// NewAdvertisementRepository creates a new advertisement repository
func NewAdvertisementRepository(db *sql.DB) *AdvertisementRepository {
	return &AdvertisementRepository{db: db}
}

// Campaign operations

// CreateCampaign creates a new advertisement campaign
func (r *AdvertisementRepository) CreateCampaign(campaign *models.AdvertisementCampaign) error {
	query := `
		INSERT INTO advertisement_campaigns 
		(name, description, advertiser_name, advertiser_email, start_date, end_date, 
		 budget_total, budget_daily, status, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		campaign.Name, campaign.Description, campaign.AdvertiserName, campaign.AdvertiserEmail,
		campaign.StartDate, campaign.EndDate, campaign.BudgetTotal, campaign.BudgetDaily,
		campaign.Status, campaign.Priority,
	).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
}

// GetCampaignByID retrieves a campaign by ID
func (r *AdvertisementRepository) GetCampaignByID(id uint64) (*models.AdvertisementCampaign, error) {
	campaign := &models.AdvertisementCampaign{}
	query := `
		SELECT id, name, description, advertiser_name, advertiser_email, start_date, end_date,
		       budget_total, budget_daily, status, priority, created_at, updated_at
		FROM advertisement_campaigns WHERE id = $1`
	
	err := r.db.QueryRow(query, id).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.AdvertiserName,
		&campaign.AdvertiserEmail, &campaign.StartDate, &campaign.EndDate,
		&campaign.BudgetTotal, &campaign.BudgetDaily, &campaign.Status, &campaign.Priority,
		&campaign.CreatedAt, &campaign.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return campaign, nil
}

// GetActiveCampaigns retrieves all active campaigns
func (r *AdvertisementRepository) GetActiveCampaigns() ([]models.AdvertisementCampaign, error) {
	query := `
		SELECT id, name, description, advertiser_name, advertiser_email, start_date, end_date,
		       budget_total, budget_daily, status, priority, created_at, updated_at
		FROM advertisement_campaigns 
		WHERE status = 'active' AND start_date <= NOW() 
		AND (end_date IS NULL OR end_date >= NOW())
		ORDER BY priority DESC, created_at DESC`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.AdvertisementCampaign
	for rows.Next() {
		var campaign models.AdvertisementCampaign
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description, &campaign.AdvertiserName,
			&campaign.AdvertiserEmail, &campaign.StartDate, &campaign.EndDate,
			&campaign.BudgetTotal, &campaign.BudgetDaily, &campaign.Status, &campaign.Priority,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}

// UpdateCampaign updates an existing campaign
func (r *AdvertisementRepository) UpdateCampaign(campaign *models.AdvertisementCampaign) error {
	query := `
		UPDATE advertisement_campaigns 
		SET name = $2, description = $3, advertiser_name = $4, advertiser_email = $5,
		    start_date = $6, end_date = $7, budget_total = $8, budget_daily = $9,
		    status = $10, priority = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	
	return r.db.QueryRow(query,
		campaign.ID, campaign.Name, campaign.Description, campaign.AdvertiserName,
		campaign.AdvertiserEmail, campaign.StartDate, campaign.EndDate,
		campaign.BudgetTotal, campaign.BudgetDaily, campaign.Status, campaign.Priority,
	).Scan(&campaign.UpdatedAt)
}

// DeleteCampaign deletes a campaign
func (r *AdvertisementRepository) DeleteCampaign(id uint64) error {
	query := `DELETE FROM advertisement_campaigns WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// Slot operations

// CreateSlot creates a new advertisement slot
func (r *AdvertisementRepository) CreateSlot(slot *models.AdvertisementSlot) error {
	query := `
		INSERT INTO advertisement_slots 
		(name, slug, description, page_type, position, width, height, is_active, lazy_load)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		slot.Name, slot.Slug, slot.Description, slot.PageType, slot.Position,
		slot.Width, slot.Height, slot.IsActive, slot.LazyLoad,
	).Scan(&slot.ID, &slot.CreatedAt, &slot.UpdatedAt)
}

// GetSlotByID retrieves a slot by ID
func (r *AdvertisementRepository) GetSlotByID(id uint64) (*models.AdvertisementSlot, error) {
	slot := &models.AdvertisementSlot{}
	query := `
		SELECT id, name, slug, description, page_type, position, width, height,
		       is_active, lazy_load, created_at, updated_at
		FROM advertisement_slots WHERE id = $1`
	
	err := r.db.QueryRow(query, id).Scan(
		&slot.ID, &slot.Name, &slot.Slug, &slot.Description, &slot.PageType,
		&slot.Position, &slot.Width, &slot.Height, &slot.IsActive, &slot.LazyLoad,
		&slot.CreatedAt, &slot.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return slot, nil
}

// GetSlotsByPageType retrieves slots by page type and position
func (r *AdvertisementRepository) GetSlotsByPageType(pageType, position string) ([]models.AdvertisementSlot, error) {
	query := `
		SELECT id, name, slug, description, page_type, position, width, height,
		       is_active, lazy_load, created_at, updated_at
		FROM advertisement_slots 
		WHERE is_active = true AND (page_type = $1 OR page_type = 'all')`
	
	args := []interface{}{pageType}
	
	if position != "" {
		query += ` AND position = $2`
		args = append(args, position)
	}
	
	query += ` ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slots []models.AdvertisementSlot
	for rows.Next() {
		var slot models.AdvertisementSlot
		err := rows.Scan(
			&slot.ID, &slot.Name, &slot.Slug, &slot.Description, &slot.PageType,
			&slot.Position, &slot.Width, &slot.Height, &slot.IsActive, &slot.LazyLoad,
			&slot.CreatedAt, &slot.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	
	return slots, nil
}

// GetAllSlots retrieves all slots
func (r *AdvertisementRepository) GetAllSlots() ([]models.AdvertisementSlot, error) {
	query := `
		SELECT id, name, slug, description, page_type, position, width, height,
		       is_active, lazy_load, created_at, updated_at
		FROM advertisement_slots 
		ORDER BY page_type, position, created_at DESC`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slots []models.AdvertisementSlot
	for rows.Next() {
		var slot models.AdvertisementSlot
		err := rows.Scan(
			&slot.ID, &slot.Name, &slot.Slug, &slot.Description, &slot.PageType,
			&slot.Position, &slot.Width, &slot.Height, &slot.IsActive, &slot.LazyLoad,
			&slot.CreatedAt, &slot.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	
	return slots, nil
}

// Creative operations

// CreateCreative creates a new advertisement creative
func (r *AdvertisementRepository) CreateCreative(creative *models.AdvertisementCreative) error {
	query := `
		INSERT INTO advertisement_creatives 
		(campaign_id, name, type, content, alt_text, click_url, width, height, file_size, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		creative.CampaignID, creative.Name, creative.Type, creative.Content,
		creative.AltText, creative.ClickURL, creative.Width, creative.Height,
		creative.FileSize, creative.IsActive,
	).Scan(&creative.ID, &creative.CreatedAt, &creative.UpdatedAt)
}

// GetCreativesByCampaign retrieves creatives by campaign ID
func (r *AdvertisementRepository) GetCreativesByCampaign(campaignID uint64) ([]models.AdvertisementCreative, error) {
	query := `
		SELECT id, campaign_id, name, type, content, alt_text, click_url, width, height,
		       file_size, is_active, created_at, updated_at
		FROM advertisement_creatives 
		WHERE campaign_id = $1 AND is_active = true
		ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var creatives []models.AdvertisementCreative
	for rows.Next() {
		var creative models.AdvertisementCreative
		err := rows.Scan(
			&creative.ID, &creative.CampaignID, &creative.Name, &creative.Type,
			&creative.Content, &creative.AltText, &creative.ClickURL, &creative.Width,
			&creative.Height, &creative.FileSize, &creative.IsActive,
			&creative.CreatedAt, &creative.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		creatives = append(creatives, creative)
	}
	
	return creatives, nil
}

// Targeting operations

// CreateTargeting creates targeting rules for a campaign
func (r *AdvertisementRepository) CreateTargeting(targeting *models.AdvertisementTargeting) error {
	query := `
		INSERT INTO advertisement_targeting 
		(campaign_id, target_type, target_value, is_include)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	
	return r.db.QueryRow(query,
		targeting.CampaignID, targeting.TargetType, targeting.TargetValue, targeting.IsInclude,
	).Scan(&targeting.ID, &targeting.CreatedAt)
}

// GetTargetingByCampaign retrieves targeting rules by campaign ID
func (r *AdvertisementRepository) GetTargetingByCampaign(campaignID uint64) ([]models.AdvertisementTargeting, error) {
	query := `
		SELECT id, campaign_id, target_type, target_value, is_include, created_at
		FROM advertisement_targeting 
		WHERE campaign_id = $1
		ORDER BY target_type, created_at`
	
	rows, err := r.db.Query(query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var targeting []models.AdvertisementTargeting
	for rows.Next() {
		var t models.AdvertisementTargeting
		err := rows.Scan(
			&t.ID, &t.CampaignID, &t.TargetType, &t.TargetValue, &t.IsInclude, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		targeting = append(targeting, t)
	}
	
	return targeting, nil
}

// Placement operations

// CreatePlacement creates a new advertisement placement
func (r *AdvertisementRepository) CreatePlacement(placement *models.AdvertisementPlacement) error {
	query := `
		INSERT INTO advertisement_placements 
		(campaign_id, slot_id, creative_id, weight, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	
	return r.db.QueryRow(query,
		placement.CampaignID, placement.SlotID, placement.CreativeID,
		placement.Weight, placement.IsActive,
	).Scan(&placement.ID, &placement.CreatedAt)
}

// GetPlacementsBySlot retrieves active placements for a slot with targeting
func (r *AdvertisementRepository) GetPlacementsBySlot(slotID uint64, categoryID *uint64, tagIDs []uint64) ([]models.AdvertisementPlacement, error) {
	// Base query for active placements
	query := `
		SELECT DISTINCT p.id, p.campaign_id, p.slot_id, p.creative_id, p.weight, p.is_active, p.created_at,
		       c.name as campaign_name, c.priority, c.status, c.start_date, c.end_date,
		       cr.type, cr.content, cr.alt_text, cr.click_url, cr.width, cr.height,
		       s.lazy_load
		FROM advertisement_placements p
		JOIN advertisement_campaigns c ON p.campaign_id = c.id
		JOIN advertisement_creatives cr ON p.creative_id = cr.id
		JOIN advertisement_slots s ON p.slot_id = s.id
		WHERE p.slot_id = $1 AND p.is_active = true AND c.status = 'active'
		AND c.start_date <= NOW() AND (c.end_date IS NULL OR c.end_date >= NOW())
		AND cr.is_active = true AND s.is_active = true`
	
	args := []interface{}{slotID}
	argIndex := 2
	
	// Add targeting conditions if provided
	if categoryID != nil || len(tagIDs) > 0 {
		query += ` AND (`
		conditions := []string{}
		
		// Check if campaign has no targeting (show to all)
		conditions = append(conditions, `
			NOT EXISTS (
				SELECT 1 FROM advertisement_targeting t 
				WHERE t.campaign_id = c.id
			)`)
		
		// Category targeting
		if categoryID != nil {
			conditions = append(conditions, fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM advertisement_targeting t 
					WHERE t.campaign_id = c.id AND t.target_type = 'category' 
					AND t.target_value = $%d AND t.is_include = true
				)`, argIndex))
			args = append(args, fmt.Sprintf("%d", *categoryID))
			argIndex++
		}
		
		// Tag targeting
		if len(tagIDs) > 0 {
			tagIDStrings := make([]string, len(tagIDs))
			for i, tagID := range tagIDs {
				tagIDStrings[i] = fmt.Sprintf("%d", tagID)
			}
			tagIDsStr := strings.Join(tagIDStrings, ",")
			
			conditions = append(conditions, fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM advertisement_targeting t 
					WHERE t.campaign_id = c.id AND t.target_type = 'tag' 
					AND t.target_value IN (%s) AND t.is_include = true
				)`, tagIDsStr))
		}
		
		query += strings.Join(conditions, " OR ") + `)`
	}
	
	query += ` ORDER BY c.priority DESC, p.weight DESC, p.created_at DESC`
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var placements []models.AdvertisementPlacement
	for rows.Next() {
		var p models.AdvertisementPlacement
		var campaignName string
		var priority int
		var status string
		var startDate, endDate time.Time
		var creativeType, content, altText, clickURL string
		var width, height *int
		var lazyLoad bool
		
		err := rows.Scan(
			&p.ID, &p.CampaignID, &p.SlotID, &p.CreativeID, &p.Weight, &p.IsActive, &p.CreatedAt,
			&campaignName, &priority, &status, &startDate, &endDate,
			&creativeType, &content, &altText, &clickURL, &width, &height,
			&lazyLoad,
		)
		if err != nil {
			return nil, err
		}
		
		// Populate related data
		p.Campaign = &models.AdvertisementCampaign{
			ID:        p.CampaignID,
			Name:      campaignName,
			Priority:  priority,
			Status:    status,
			StartDate: startDate,
			EndDate:   &endDate,
		}
		
		p.Creative = &models.AdvertisementCreative{
			ID:       p.CreativeID,
			Type:     creativeType,
			Content:  content,
			AltText:  altText,
			ClickURL: clickURL,
			Width:    width,
			Height:   height,
		}
		
		p.Slot = &models.AdvertisementSlot{
			ID:       p.SlotID,
			LazyLoad: lazyLoad,
		}
		
		placements = append(placements, p)
	}
	
	return placements, nil
}

// Tracking operations

// RecordImpression records an advertisement impression
func (r *AdvertisementRepository) RecordImpression(impression *models.AdvertisementImpression) error {
	query := `
		INSERT INTO advertisement_impressions 
		(placement_id, campaign_id, slot_id, creative_id, ip_address, user_agent, referer, page_url, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	
	return r.db.QueryRow(query,
		impression.PlacementID, impression.CampaignID, impression.SlotID, impression.CreativeID,
		impression.IPAddress, impression.UserAgent, impression.Referer, impression.PageURL,
		impression.DeviceType,
	).Scan(&impression.ID, &impression.CreatedAt)
}

// RecordClick records an advertisement click
func (r *AdvertisementRepository) RecordClick(click *models.AdvertisementClick) error {
	query := `
		INSERT INTO advertisement_clicks 
		(placement_id, campaign_id, slot_id, creative_id, impression_id, ip_address, user_agent, referer, page_url, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	
	return r.db.QueryRow(query,
		click.PlacementID, click.CampaignID, click.SlotID, click.CreativeID, click.ImpressionID,
		click.IPAddress, click.UserAgent, click.Referer, click.PageURL, click.DeviceType,
	).Scan(&click.ID, &click.CreatedAt)
}

// GetCampaignStats retrieves statistics for a campaign
func (r *AdvertisementRepository) GetCampaignStats(campaignID uint64, startDate, endDate time.Time) (*models.AdvertisementStats, error) {
	query := `
		SELECT 
			COALESCE(i.impressions, 0) as impressions,
			COALESCE(c.clicks, 0) as clicks,
			CASE 
				WHEN COALESCE(i.impressions, 0) > 0 
				THEN (COALESCE(c.clicks, 0)::float / i.impressions::float) * 100 
				ELSE 0 
			END as ctr
		FROM (
			SELECT COUNT(*) as impressions
			FROM advertisement_impressions
			WHERE campaign_id = $1 AND created_at BETWEEN $2 AND $3
		) i
		CROSS JOIN (
			SELECT COUNT(*) as clicks
			FROM advertisement_clicks
			WHERE campaign_id = $1 AND created_at BETWEEN $2 AND $3
		) c`
	
	stats := &models.AdvertisementStats{
		CampaignID: campaignID,
		StartDate:  startDate,
		EndDate:    endDate,
	}
	
	err := r.db.QueryRow(query, campaignID, startDate, endDate).Scan(
		&stats.Impressions, &stats.Clicks, &stats.CTR,
	)
	
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// GetPerformanceReport generates a performance report for campaigns
func (r *AdvertisementRepository) GetPerformanceReport(startDate, endDate time.Time, campaignIDs []uint64) ([]models.AdvertisementPerformanceReport, error) {
	whereClause := ""
	args := []interface{}{startDate, endDate}
	
	if len(campaignIDs) > 0 {
		placeholders := make([]string, len(campaignIDs))
		for i, id := range campaignIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+3)
			args = append(args, id)
		}
		whereClause = fmt.Sprintf("AND c.id IN (%s)", strings.Join(placeholders, ","))
	}
	
	query := fmt.Sprintf(`
		SELECT 
			c.id as campaign_id,
			c.name as campaign_name,
			s.id as slot_id,
			s.name as slot_name,
			cr.id as creative_id,
			cr.name as creative_name,
			COALESCE(i.impressions, 0) as impressions,
			COALESCE(cl.clicks, 0) as clicks,
			CASE 
				WHEN COALESCE(i.impressions, 0) > 0 
				THEN (COALESCE(cl.clicks, 0)::float / i.impressions::float) * 100 
				ELSE 0 
			END as ctr,
			0 as spend,
			0 as revenue,
			0 as cpm,
			0 as cpc,
			0 as conversion_rate,
			0 as viewability_rate,
			0 as load_time_ms,
			0 as error_rate
		FROM advertisement_campaigns c
		JOIN advertisement_placements p ON c.id = p.campaign_id
		JOIN advertisement_slots s ON p.slot_id = s.id
		JOIN advertisement_creatives cr ON p.creative_id = cr.id
		LEFT JOIN (
			SELECT campaign_id, slot_id, creative_id, COUNT(*) as impressions
			FROM advertisement_impressions
			WHERE created_at BETWEEN $1 AND $2
			GROUP BY campaign_id, slot_id, creative_id
		) i ON c.id = i.campaign_id AND s.id = i.slot_id AND cr.id = i.creative_id
		LEFT JOIN (
			SELECT campaign_id, slot_id, creative_id, COUNT(*) as clicks
			FROM advertisement_clicks
			WHERE created_at BETWEEN $1 AND $2
			GROUP BY campaign_id, slot_id, creative_id
		) cl ON c.id = cl.campaign_id AND s.id = cl.slot_id AND cr.id = cl.creative_id
		WHERE c.status = 'active' %s
		ORDER BY c.priority DESC, impressions DESC`, whereClause)
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var reports []models.AdvertisementPerformanceReport
	for rows.Next() {
		var report models.AdvertisementPerformanceReport
		err := rows.Scan(
			&report.CampaignID, &report.CampaignName, &report.SlotID, &report.SlotName,
			&report.CreativeID, &report.CreativeName, &report.Impressions, &report.Clicks,
			&report.CTR, &report.Spend, &report.Revenue, &report.CPM, &report.CPC,
			&report.ConversionRate, &report.ViewabilityRate, &report.LoadTime,
			&report.ErrorRate,
		)
		if err != nil {
			return nil, err
		}
		
		report.Date = startDate
		report.Period = "custom"
		reports = append(reports, report)
	}
	
	return reports, nil
}