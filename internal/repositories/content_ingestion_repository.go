package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/database"
)

// ContentIngestionRepository handles content ingestion data operations
type ContentIngestionRepository struct {
	db *database.DB
}

// NewContentIngestionRepository creates a new content ingestion repository
func NewContentIngestionRepository(db *database.DB) *ContentIngestionRepository {
	return &ContentIngestionRepository{
		db: db,
	}
}

// CreateContentSource creates a new content source
func (r *ContentIngestionRepository) CreateContentSource(ctx context.Context, source *models.ContentSource) (*models.ContentSource, error) {
	query := `
		INSERT INTO content_sources (name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	
	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	
	err = r.db.QueryRowContext(ctx, query,
		source.Name,
		source.Type,
		source.APIKey,
		source.IsActive,
		source.RateLimit,
		source.Priority,
		configJSON,
	).Scan(&source.ID, &source.CreatedAt, &source.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create content source: %w", err)
	}
	
	return source, nil
}

// GetContentSourceByAPIKey retrieves a content source by API key
func (r *ContentIngestionRepository) GetContentSourceByAPIKey(ctx context.Context, apiKey string) (*models.ContentSource, error) {
	query := `
		SELECT id, name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at
		FROM content_sources
		WHERE api_key = $1 AND is_active = true`
	
	var source models.ContentSource
	var configJSON []byte
	
	err := r.db.QueryRowContext(ctx, query, apiKey).Scan(
		&source.ID,
		&source.Name,
		&source.Type,
		&source.APIKey,
		&source.IsActive,
		&source.RateLimit,
		&source.Priority,
		&configJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content source not found")
		}
		return nil, fmt.Errorf("failed to get content source: %w", err)
	}
	
	if err := json.Unmarshal(configJSON, &source.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &source, nil
}

// CreateIngestedContent creates a new ingested content record
func (r *ContentIngestionRepository) CreateIngestedContent(ctx context.Context, content *models.IngestedContent) (*models.IngestedContent, error) {
	query := `
		INSERT INTO ingested_content (
			source_id, external_id, title, content, excerpt, author_name, author_email,
			category_name, tags, published_at, source_url, content_hash, status,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	
	tagsJSON, err := json.Marshal(content.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}
	
	metadataJSON, err := json.Marshal(content.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	err = r.db.QueryRowContext(ctx, query,
		content.SourceID,
		content.ExternalID,
		content.Title,
		content.Content,
		content.Excerpt,
		content.AuthorName,
		content.AuthorEmail,
		content.CategoryName,
		tagsJSON,
		content.PublishedAt,
		content.SourceURL,
		content.ContentHash,
		content.Status,
		metadataJSON,
	).Scan(&content.ID, &content.CreatedAt, &content.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create ingested content: %w", err)
	}
	
	return content, nil
}

// CheckDuplicateContent checks for duplicate content using multiple strategies
func (r *ContentIngestionRepository) CheckDuplicateContent(ctx context.Context, content *models.IngestedContent) (*models.DuplicateCheckResult, error) {
	result := &models.DuplicateCheckResult{
		IsDuplicate: false,
		Similarity:  0.0,
	}
	
	// 1. Check by external ID and source
	if content.ExternalID != "" {
		duplicateID, err := r.checkByExternalID(ctx, content.SourceID, content.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate by external ID: %w", err)
		}
		if duplicateID != nil {
			result.IsDuplicate = true
			result.ExistingID = duplicateID
			result.Similarity = 1.0
			result.MatchType = "exact"
			result.MatchedField = "external_id"
			return result, nil
		}
	}
	
	// 2. Check by content hash (exact content match)
	if content.ContentHash != "" {
		duplicateID, err := r.checkByContentHash(ctx, content.ContentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate by content hash: %w", err)
		}
		if duplicateID != nil {
			result.IsDuplicate = true
			result.ExistingID = duplicateID
			result.Similarity = 1.0
			result.MatchType = "hash"
			result.MatchedField = "content_hash"
			return result, nil
		}
	}
	
	// 3. Check by title similarity (fuzzy match)
	if content.Title != "" {
		duplicateID, similarity, err := r.checkByTitleSimilarity(ctx, content.Title)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate by title: %w", err)
		}
		if duplicateID != nil && similarity > 0.85 { // 85% similarity threshold
			result.IsDuplicate = true
			result.ExistingID = duplicateID
			result.Similarity = similarity
			result.MatchType = "title"
			result.MatchedField = "title"
			return result, nil
		}
	}
	
	// 4. Check by source URL (if provided)
	if content.SourceURL != "" {
		duplicateID, err := r.checkBySourceURL(ctx, content.SourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate by source URL: %w", err)
		}
		if duplicateID != nil {
			result.IsDuplicate = true
			result.ExistingID = duplicateID
			result.Similarity = 1.0
			result.MatchType = "exact"
			result.MatchedField = "source_url"
			return result, nil
		}
	}
	
	return result, nil
}

// UpdateIngestedContentStatus updates the status of ingested content
func (r *ContentIngestionRepository) UpdateIngestedContentStatus(ctx context.Context, id uint64, status string, articleID *uint64, rejectionReason string) error {
	query := `
		UPDATE ingested_content 
		SET status = $1, article_id = $2, rejection_reason = $3, processed_at = NOW(), updated_at = NOW()
		WHERE id = $4`
	
	result, err := r.db.ExecContext(ctx, query, status, articleID, rejectionReason, id)
	if err != nil {
		return fmt.Errorf("failed to update ingested content status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("ingested content not found")
	}
	
	return nil
}

// GetPendingContent retrieves pending content for processing
func (r *ContentIngestionRepository) GetPendingContent(ctx context.Context, limit int) ([]models.IngestedContent, error) {
	query := `
		SELECT id, source_id, external_id, title, content, excerpt, author_name, author_email,
			   category_name, tags, published_at, source_url, content_hash, status,
			   metadata, created_at, updated_at
		FROM ingested_content
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending content: %w", err)
	}
	defer rows.Close()
	
	var contents []models.IngestedContent
	for rows.Next() {
		var content models.IngestedContent
		var tagsJSON, metadataJSON []byte
		
		err := rows.Scan(
			&content.ID,
			&content.SourceID,
			&content.ExternalID,
			&content.Title,
			&content.Content,
			&content.Excerpt,
			&content.AuthorName,
			&content.AuthorEmail,
			&content.CategoryName,
			&tagsJSON,
			&content.PublishedAt,
			&content.SourceURL,
			&content.ContentHash,
			&content.Status,
			&metadataJSON,
			&content.CreatedAt,
			&content.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ingested content: %w", err)
		}
		
		// Unmarshal JSON fields
		if err := json.Unmarshal(tagsJSON, &content.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
		
		if err := json.Unmarshal(metadataJSON, &content.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		
		contents = append(contents, content)
	}
	
	return contents, nil
}

// GetIngestionStats retrieves ingestion statistics
func (r *ContentIngestionRepository) GetIngestionStats(ctx context.Context, sourceID *uint64, hours int) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM ingested_content
		WHERE created_at > NOW() - INTERVAL '%d hours'`
	
	args := []interface{}{}
	if sourceID != nil {
		query += " AND source_id = $1"
		args = append(args, *sourceID)
	}
	
	query += " GROUP BY status"
	formattedQuery := fmt.Sprintf(query, hours)
	
	rows, err := r.db.QueryContext(ctx, formattedQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ingestion stats: %w", err)
	}
	defer rows.Close()
	
	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		
		stats[status] = count
	}
	
	return stats, nil
}

// Private helper methods for duplicate checking

func (r *ContentIngestionRepository) checkByExternalID(ctx context.Context, sourceID uint64, externalID string) (*uint64, error) {
	query := `
		SELECT ic.article_id
		FROM ingested_content ic
		WHERE ic.source_id = $1 AND ic.external_id = $2 AND ic.status = 'processed' AND ic.article_id IS NOT NULL
		LIMIT 1`
	
	var articleID uint64
	err := r.db.QueryRowContext(ctx, query, sourceID, externalID).Scan(&articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &articleID, nil
}

func (r *ContentIngestionRepository) checkByContentHash(ctx context.Context, contentHash string) (*uint64, error) {
	query := `
		SELECT ic.article_id
		FROM ingested_content ic
		WHERE ic.content_hash = $1 AND ic.status = 'processed' AND ic.article_id IS NOT NULL
		LIMIT 1`
	
	var articleID uint64
	err := r.db.QueryRowContext(ctx, query, contentHash).Scan(&articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &articleID, nil
}

func (r *ContentIngestionRepository) checkByTitleSimilarity(ctx context.Context, title string) (*uint64, float64, error) {
	// Use PostgreSQL's similarity function for fuzzy matching
	query := `
		SELECT ic.article_id, similarity(ic.title, $1) as sim
		FROM ingested_content ic
		WHERE ic.status = 'processed' AND ic.article_id IS NOT NULL
		  AND similarity(ic.title, $1) > 0.8
		ORDER BY sim DESC
		LIMIT 1`
	
	var articleID uint64
	var similarity float64
	err := r.db.QueryRowContext(ctx, query, title).Scan(&articleID, &similarity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	
	return &articleID, similarity, nil
}

func (r *ContentIngestionRepository) checkBySourceURL(ctx context.Context, sourceURL string) (*uint64, error) {
	query := `
		SELECT ic.article_id
		FROM ingested_content ic
		WHERE ic.source_url = $1 AND ic.status = 'processed' AND ic.article_id IS NOT NULL
		LIMIT 1`
	
	var articleID uint64
	err := r.db.QueryRowContext(ctx, query, sourceURL).Scan(&articleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &articleID, nil
}

// GetContentSourceByID retrieves a content source by ID
func (r *ContentIngestionRepository) GetContentSourceByID(ctx context.Context, id uint64) (*models.ContentSource, error) {
	query := `
		SELECT id, name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at
		FROM content_sources
		WHERE id = $1`
	
	var source models.ContentSource
	var configJSON []byte
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID,
		&source.Name,
		&source.Type,
		&source.APIKey,
		&source.IsActive,
		&source.RateLimit,
		&source.Priority,
		&configJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content source not found")
		}
		return nil, fmt.Errorf("failed to get content source: %w", err)
	}
	
	if err := json.Unmarshal(configJSON, &source.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &source, nil
}

// ListContentSources retrieves all content sources with pagination
func (r *ContentIngestionRepository) ListContentSources(ctx context.Context, limit, offset int) ([]models.ContentSource, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM content_sources`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count content sources: %w", err)
	}
	
	// Get sources with pagination
	query := `
		SELECT id, name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at
		FROM content_sources
		ORDER BY priority DESC, created_at DESC
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query content sources: %w", err)
	}
	defer rows.Close()
	
	var sources []models.ContentSource
	for rows.Next() {
		var source models.ContentSource
		var configJSON []byte
		
		err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.Type,
			&source.APIKey,
			&source.IsActive,
			&source.RateLimit,
			&source.Priority,
			&configJSON,
			&source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan content source: %w", err)
		}
		
		if err := json.Unmarshal(configJSON, &source.Config); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		
		sources = append(sources, source)
	}
	
	return sources, total, nil
}