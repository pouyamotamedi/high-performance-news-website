package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

// GetDB returns the database connection
func (r *ContentIngestionRepository) GetDB() *database.DB {
	return r.db
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



// GetIngestionStats retrieves ingestion statistics
func (r *ContentIngestionRepository) GetIngestionStats(ctx context.Context, sourceID *uint64, hours int) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM ingested_content
		WHERE created_at > NOW() - $1 * INTERVAL '1 hour'`
	
	args := []interface{}{hours}
	argIndex := 2
	
	if sourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *sourceID)
		argIndex++
	}
	
	query += " GROUP BY status"
	
	rows, err := r.db.QueryContext(ctx, query, args...)
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

// GetPendingContent retrieves pending content from ingested_content table
func (r *ContentIngestionRepository) GetPendingContent(ctx context.Context, limit, offset int) ([]*models.IngestedContent, error) {
	query := `
		SELECT ic.id, ic.source_id, ic.external_id, ic.title, ic.content, ic.excerpt, 
		       ic.author_name, ic.author_email, ic.category_name, ic.tags, 
		       ic.published_at, ic.source_url, ic.status, ic.metadata, ic.created_at, ic.updated_at,
		       cs.name as source_name
		FROM ingested_content ic
		LEFT JOIN content_sources cs ON ic.source_id = cs.id
		WHERE ic.status = 'pending'
		ORDER BY ic.created_at DESC
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending content: %w", err)
	}
	defer rows.Close()
	
	var contents []*models.IngestedContent
	for rows.Next() {
		content := &models.IngestedContent{}
		var sourceName sql.NullString
		var tagsJSON []byte
		var metadataJSON []byte
		
		err := rows.Scan(
			&content.ID, &content.SourceID, &content.ExternalID, &content.Title,
			&content.Content, &content.Excerpt, &content.AuthorName, &content.AuthorEmail,
			&content.CategoryName, &tagsJSON, &content.PublishedAt, &content.SourceURL,
			&content.Status, &metadataJSON, &content.CreatedAt, &content.UpdatedAt, &sourceName,
		)
		if err != nil {
			continue // Skip problematic rows
		}
		
		// Parse tags JSON
		if len(tagsJSON) > 0 {
			fmt.Printf("DEBUG GetPendingContent: Content ID %d, tagsJSON: %s\n", content.ID, string(tagsJSON))
			if err := json.Unmarshal(tagsJSON, &content.Tags); err != nil {
				fmt.Printf("ERROR GetPendingContent: Failed to unmarshal tags for content %d: %v\n", content.ID, err)
			} else {
				fmt.Printf("DEBUG GetPendingContent: Content ID %d, parsed tags: %v (count: %d)\n", content.ID, content.Tags, len(content.Tags))
			}
		} else {
			fmt.Printf("DEBUG GetPendingContent: Content ID %d has no tags\n", content.ID)
		}
		
		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			fmt.Printf("DEBUG GetPendingContent: Content ID %d, metadataJSON: %s\n", content.ID, string(metadataJSON))
			if err := json.Unmarshal(metadataJSON, &content.Metadata); err != nil {
				fmt.Printf("ERROR GetPendingContent: Failed to unmarshal metadata for content %d: %v\n", content.ID, err)
			} else {
				fmt.Printf("DEBUG GetPendingContent: Content ID %d, parsed metadata: %+v\n", content.ID, content.Metadata)
			}
		} else {
			fmt.Printf("DEBUG GetPendingContent: Content ID %d has no metadata\n", content.ID)
		}
		
		contents = append(contents, content)
	}
	
	return contents, nil
}

// GetPendingContentCount returns the total count of pending content
func (r *ContentIngestionRepository) GetPendingContentCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM ingested_content WHERE status = 'pending'`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending content: %w", err)
	}
	
	return count, nil
}

// GetProcessedContent retrieves processed content from ingested_content table
func (r *ContentIngestionRepository) GetProcessedContent(ctx context.Context, limit, offset int, status string) ([]*models.IngestedContent, error) {
	var query string
	var args []interface{}
	
	if status != "" {
		query = `
			SELECT ic.id, ic.source_id, ic.external_id, ic.title, ic.content, ic.excerpt,
			       ic.author_name, ic.author_email, ic.category_name, ic.tags,
			       ic.published_at, ic.source_url, ic.status, ic.processed_at, 
			       ic.article_id, ic.created_at, ic.updated_at,
			       cs.name as source_name, a.slug as article_slug
			FROM ingested_content ic
			LEFT JOIN content_sources cs ON ic.source_id = cs.id
			LEFT JOIN articles a ON ic.article_id = a.id
			WHERE ic.status = $1 AND ic.status != 'pending'
			ORDER BY 
				CASE WHEN ic.processed_at IS NOT NULL THEN ic.processed_at ELSE ic.created_at END DESC,
				ic.created_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{status, limit, offset}
	} else {
		query = `
			SELECT ic.id, ic.source_id, ic.external_id, ic.title, ic.content, ic.excerpt,
			       ic.author_name, ic.author_email, ic.category_name, ic.tags,
			       ic.published_at, ic.source_url, ic.status, ic.processed_at,
			       ic.article_id, ic.created_at, ic.updated_at,
			       cs.name as source_name, a.slug as article_slug
			FROM ingested_content ic
			LEFT JOIN content_sources cs ON ic.source_id = cs.id
			LEFT JOIN articles a ON ic.article_id = a.id
			WHERE ic.status != 'pending'
			ORDER BY 
				CASE WHEN ic.processed_at IS NOT NULL THEN ic.processed_at ELSE ic.created_at END DESC,
				ic.created_at DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query processed content: %w", err)
	}
	defer rows.Close()
	
	var contents []*models.IngestedContent
	for rows.Next() {
		content := &models.IngestedContent{}
		var sourceName sql.NullString
		var articleSlug sql.NullString
		var tagsJSON []byte
		
		err := rows.Scan(
			&content.ID, &content.SourceID, &content.ExternalID, &content.Title,
			&content.Content, &content.Excerpt, &content.AuthorName, &content.AuthorEmail,
			&content.CategoryName, &tagsJSON, &content.PublishedAt, &content.SourceURL,
			&content.Status, &content.ProcessedAt, &content.ArticleID,
			&content.CreatedAt, &content.UpdatedAt, &sourceName, &articleSlug,
		)
		if err != nil {
			continue // Skip problematic rows
		}
		
		// Parse tags JSON
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &content.Tags)
		}
		
		// Store article slug in metadata for access in service layer
		if articleSlug.Valid {
			if content.Metadata == nil {
				content.Metadata = make(map[string]interface{})
			}
			content.Metadata["article_slug"] = articleSlug.String
		}
		
		contents = append(contents, content)
	}
	
	return contents, nil
}

// GetProcessedContentCount returns the total count of processed content
func (r *ContentIngestionRepository) GetProcessedContentCount(ctx context.Context, status string) (int, error) {
	var query string
	var args []interface{}
	
	if status != "" {
		query = `SELECT COUNT(*) FROM ingested_content WHERE status = $1`
		args = []interface{}{status}
	} else {
		query = `SELECT COUNT(*) FROM ingested_content WHERE status != 'pending'`
		args = []interface{}{}
	}
	
	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count processed content: %w", err)
	}
	
	return count, nil
}

// GetContentSources retrieves content sources from database
func (r *ContentIngestionRepository) GetContentSources(ctx context.Context, limit, offset int) ([]*models.ContentSource, int, error) {
	query := `
		SELECT id, name, type, is_active, api_key, rate_limit, priority, created_at, updated_at
		FROM content_sources
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query content sources: %w", err)
	}
	defer rows.Close()
	
	var sources []*models.ContentSource
	for rows.Next() {
		source := &models.ContentSource{}
		
		err := rows.Scan(
			&source.ID, &source.Name, &source.Type, &source.IsActive, 
			&source.APIKey, &source.RateLimit, &source.Priority,
			&source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			continue // Skip problematic rows
		}
		
		sources = append(sources, source)
	}
	
	// Get total count
	countQuery := `SELECT COUNT(*) FROM content_sources`
	var total int
	err = r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		total = len(sources) // Fallback to current result count
	}
	
	return sources, total, nil
}

// UpdateContentSource updates an existing content source
func (r *ContentIngestionRepository) UpdateContentSource(ctx context.Context, source *models.ContentSource) (*models.ContentSource, error) {
	// First, get the existing source to check if API key is changing
	existingSource, err := r.GetContentSourceByID(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing source: %w", err)
	}

	var query string
	var args []interface{}
	
	// Marshal config to JSON
	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Check if API key is changing
	if source.APIKey != existingSource.APIKey {
		// API key is changing, include it in the update
		query = `
			UPDATE content_sources 
			SET name = $1, type = $2, api_key = $3, is_active = $4, rate_limit = $5, priority = $6, config = $7, updated_at = $8
			WHERE id = $9
			RETURNING id, name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at`
		args = []interface{}{
			source.Name,
			source.Type,
			source.APIKey,
			source.IsActive,
			source.RateLimit,
			source.Priority,
			configJSON,
			time.Now(),
			source.ID,
		}
	} else {
		// API key is not changing, exclude it from the update
		query = `
			UPDATE content_sources 
			SET name = $1, type = $2, is_active = $3, rate_limit = $4, priority = $5, config = $6, updated_at = $7
			WHERE id = $8
			RETURNING id, name, type, api_key, is_active, rate_limit, priority, config, created_at, updated_at`
		args = []interface{}{
			source.Name,
			source.Type,
			source.IsActive,
			source.RateLimit,
			source.Priority,
			configJSON,
			time.Now(),
			source.ID,
		}
	}

	var updatedSource models.ContentSource
	var returnedConfigJSON []byte
	
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&updatedSource.ID,
		&updatedSource.Name,
		&updatedSource.Type,
		&updatedSource.APIKey,
		&updatedSource.IsActive,
		&updatedSource.RateLimit,
		&updatedSource.Priority,
		&returnedConfigJSON,
		&updatedSource.CreatedAt,
		&updatedSource.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content source not found")
		}
		return nil, fmt.Errorf("failed to update content source: %w", err)
	}

	// Unmarshal config JSON
	if err := json.Unmarshal(returnedConfigJSON, &updatedSource.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &updatedSource, nil
}

// GetPendingContentByID retrieves a specific pending content item by ID
func (r *ContentIngestionRepository) GetPendingContentByID(ctx context.Context, contentID uint64) (*models.IngestedContent, error) {
	query := `
		SELECT id, source_id, external_id, title, content, excerpt, author_name, author_email,
			   category_name, tags, source_url, status, rejection_reason, article_id, 
			   created_at, processed_at
		FROM ingested_content 
		WHERE id = $1 AND status = 'pending'`

	var content models.IngestedContent
	err := r.db.QueryRowContext(ctx, query, contentID).Scan(
		&content.ID,
		&content.SourceID,
		&content.ExternalID,
		&content.Title,
		&content.Content,
		&content.Excerpt,
		&content.AuthorName,
		&content.AuthorEmail,
		&content.CategoryName,
		&content.Tags,
		&content.SourceURL,
		&content.Status,
		&content.RejectionReason,
		&content.ArticleID,
		&content.CreatedAt,
		&content.ProcessedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending content by ID: %w", err)
	}

	return &content, nil
}

// RejectPendingContent rejects a pending content item
func (r *ContentIngestionRepository) RejectPendingContent(ctx context.Context, contentID uint64, reason string) error {
	query := `
		UPDATE ingested_content 
		SET status = 'rejected', rejection_reason = $1, processed_at = $2
		WHERE id = $3 AND status = 'pending'`

	result, err := r.db.ExecContext(ctx, query, reason, time.Now(), contentID)
	if err != nil {
		return fmt.Errorf("failed to reject content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no pending content found with ID %d", contentID)
	}

	return nil
}

// DeleteContentSource deletes a content source
func (r *ContentIngestionRepository) DeleteContentSource(ctx context.Context, sourceID uint64) error {
	query := `DELETE FROM content_sources WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete content source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("content source not found")
	}

	return nil
}

// GetContentByID retrieves a specific content item by ID (any status)
func (r *ContentIngestionRepository) GetContentByID(ctx context.Context, contentID uint64) (*models.IngestedContent, error) {
	query := `
		SELECT id, source_id, external_id, title, content, excerpt, author_name, author_email,
			   category_name, tags, published_at, source_url, content_hash, status, 
			   processed_at, article_id, rejection_reason, created_at
		FROM ingested_content 
		WHERE id = $1`

	var content models.IngestedContent
	var tagsJSON []byte
	var rejectionReason sql.NullString
	err := r.db.QueryRowContext(ctx, query, contentID).Scan(
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
		&content.ProcessedAt,
		&content.ArticleID,
		&rejectionReason,
		&content.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get content by ID: %w", err)
	}

	// Parse tags JSON
	if len(tagsJSON) > 0 {
		err = json.Unmarshal(tagsJSON, &content.Tags)
		if err != nil {
			// If JSON parsing fails, set empty tags
			content.Tags = []string{}
		}
	} else {
		content.Tags = []string{}
	}

	// Handle NULL rejection reason
	if rejectionReason.Valid {
		content.RejectionReason = rejectionReason.String
	} else {
		content.RejectionReason = ""
	}

	return &content, nil
}

// UpdateContentStatus updates the status of a content item
func (r *ContentIngestionRepository) UpdateContentStatus(ctx context.Context, contentID uint64, status string) error {
	query := `UPDATE ingested_content SET status = $1, processed_at = CURRENT_TIMESTAMP WHERE id = $2`
	
	result, err := r.db.ExecContext(ctx, query, status, contentID)
	if err != nil {
		return fmt.Errorf("failed to update content status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("content not found")
	}

	return nil
}

// GetIngestionStatsByAction retrieves ingestion statistics based on when actions were taken (updated_at)
func (r *ContentIngestionRepository) GetIngestionStatsByAction(ctx context.Context, sourceID *uint64, hours int) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM ingested_content
		WHERE updated_at > NOW() - $1 * INTERVAL '1 hour'`
	
	args := []interface{}{hours}
	argIndex := 2
	
	if sourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *sourceID)
		argIndex++
	}
	
	query += " GROUP BY status"
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ingestion stats by action: %w", err)
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