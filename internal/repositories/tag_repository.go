package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// TagRepository handles database operations for tags
type TagRepository struct {
	db *sql.DB
}

// NewTagRepository creates a new tag repository
func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

// Create creates a new tag
func (r *TagRepository) Create(tag *models.Tag) (*models.Tag, error) {
	tag.PrepareForDB()
	if err := models.ValidateTag(tag); err != nil {
		return nil, err
	}

	// Validate keyword uniqueness across all tags
	if err := r.ValidateKeywordUniqueness(tag.Keywords, 0); err != nil {
		return nil, err
	}

	// Check for duplicate tag name
	if err := r.ValidateTagNameUniqueness(tag.Name, 0); err != nil {
		return nil, err
	}

	// If this is a new tag (not a translation), create a translation group first
	if tag.TranslationGroupID == nil {
		var groupID uint64
		err := r.db.QueryRow(
			"INSERT INTO translation_groups (group_type) VALUES ('tag') RETURNING id",
		).Scan(&groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to create translation group: %w", err)
		}
		tag.TranslationGroupID = &groupID
	}

	keywordsJSON, err := json.Marshal(tag.Keywords)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keywords: %w", err)
	}

	query := `INSERT INTO tags (name, slug, description, keywords, color, language_code, translation_group_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	now := time.Now()
	err = r.db.QueryRow(query, tag.Name, tag.Slug, tag.Description, 
		keywordsJSON, tag.Color, tag.LanguageCode, tag.TranslationGroupID, now).Scan(&tag.ID, &tag.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

// GetByID retrieves a tag by ID
func (r *TagRepository) GetByID(ctx context.Context, id uint64) (*models.Tag, error) {
	query := `SELECT id, name, slug, description, keywords, color, created_at, language_code
		FROM tags WHERE id = $1`

	tag := &models.Tag{}
	var keywordsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &tag.Description,
		&keywordsJSON, &tag.Color, &tag.CreatedAt, &tag.LanguageCode,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewNotFoundError("tag", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	if len(keywordsJSON) > 0 {
		if err := json.Unmarshal(keywordsJSON, &tag.Keywords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
		}
	}

	return tag, nil
}

// GetBySlug retrieves a tag by slug and language code
func (r *TagRepository) GetBySlug(slug, languageCode string) (*models.Tag, error) {
	// First try with the specified language, then try without language constraint
	query := `SELECT id, name, slug, description, keywords, color, created_at, 
		language_code, translation_group_id
		FROM tags WHERE slug = $1 AND language_code = $2`

	tag := &models.Tag{}
	var description sql.NullString
	var keywordsJSON sql.NullString
	var color sql.NullString
	var translationGroupID sql.NullInt64
	
	err := r.db.QueryRow(query, slug, languageCode).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &description,
		&keywordsJSON, &color, &tag.CreatedAt,
		&tag.LanguageCode, &translationGroupID,
	)
	
	if err == sql.ErrNoRows {
		// Try without language constraint
		query = `SELECT id, name, slug, description, keywords, color, created_at, 
			language_code, translation_group_id
			FROM tags WHERE slug = $1 LIMIT 1`
		
		err = r.db.QueryRow(query, slug).Scan(
			&tag.ID, &tag.Name, &tag.Slug, &description,
			&keywordsJSON, &color, &tag.CreatedAt,
			&tag.LanguageCode, &translationGroupID,
		)
	}
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag by slug: %w", err)
	}

	// Handle nullable fields
	if description.Valid {
		tag.Description = description.String
	}
	if color.Valid {
		tag.Color = color.String
	} else {
		tag.Color = "#000000" // Default color
	}
	if translationGroupID.Valid {
		val := uint64(translationGroupID.Int64)
		tag.TranslationGroupID = &val
	}

	// Handle keywords JSON
	if keywordsJSON.Valid && keywordsJSON.String != "" {
		if err := json.Unmarshal([]byte(keywordsJSON.String), &tag.Keywords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
		}
	} else {
		tag.Keywords = []string{}
	}

	return tag, nil
}

// GetAll retrieves all tags
func (r *TagRepository) GetAll() ([]models.Tag, error) {
	query := `SELECT id, name, slug, description, keywords, color, created_at, language_code
		FROM tags ORDER BY name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var description sql.NullString
		var keywordsJSON sql.NullString
		var color sql.NullString
		
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &description,
			&keywordsJSON, &color, &tag.CreatedAt, &tag.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			tag.Description = description.String
		}
		if color.Valid {
			tag.Color = color.String
		} else {
			tag.Color = "#000000" // Default color
		}

		// Handle keywords JSON
		if keywordsJSON.Valid && keywordsJSON.String != "" {
			if err := json.Unmarshal([]byte(keywordsJSON.String), &tag.Keywords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
			}
		} else {
			tag.Keywords = []string{}
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// Update updates an existing tag
func (r *TagRepository) Update(tag *models.Tag) error {
	tag.PrepareForDB()
	if err := models.ValidateTag(tag); err != nil {
		return err
	}

	// Validate keyword uniqueness across all tags (excluding this tag)
	if err := r.ValidateKeywordUniqueness(tag.Keywords, tag.ID); err != nil {
		return err
	}

	// Check for duplicate tag name (excluding this tag)
	if err := r.ValidateTagNameUniqueness(tag.Name, tag.ID); err != nil {
		return err
	}

	keywordsJSON, err := json.Marshal(tag.Keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}

	query := `UPDATE tags 
		SET name = $1, slug = $2, description = $3, keywords = $4, color = $5, language_code = $6
		WHERE id = $7`

	result, err := r.db.Exec(query, tag.Name, tag.Slug, tag.Description,
		keywordsJSON, tag.Color, tag.LanguageCode, tag.ID)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.NewNotFoundError("tag", fmt.Sprintf("%d", tag.ID))
	}

	return nil
}

// Delete deletes a tag
func (r *TagRepository) Delete(id uint64) error {
	query := `DELETE FROM tags WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return models.NewNotFoundError("tag", fmt.Sprintf("%d", id))
	}

	return nil
}

// BulkCreate creates multiple tags in a single transaction
func (r *TagRepository) BulkCreate(tags []models.Tag) ([]models.Tag, error) {
	if len(tags) == 0 {
		return tags, nil
	}

	// Validate all tags first
	for i := range tags {
		tags[i].PrepareForDB()
		if err := models.ValidateTag(&tags[i]); err != nil {
			return nil, fmt.Errorf("validation failed for tag %d: %w", i, err)
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO tags (name, slug, description, keywords, color, language_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	now := time.Now()
	for i := range tags {
		keywordsJSON, err := json.Marshal(tags[i].Keywords)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal keywords for tag %s: %w", tags[i].Name, err)
		}

		err = tx.QueryRow(query, tags[i].Name, tags[i].Slug, tags[i].Description, 
			keywordsJSON, tags[i].Color, tags[i].LanguageCode, now).Scan(&tags[i].ID, &tags[i].CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create tag %s: %w", tags[i].Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tags, nil
}

// GetAllWithKeywords retrieves all tags that have keywords for auto-linking
func (r *TagRepository) GetAllWithKeywords(ctx context.Context) ([]models.Tag, error) {
	query := `SELECT id, name, slug, description, keywords, color, created_at, language_code
		FROM tags 
		WHERE keywords IS NOT NULL AND keywords != '[]' AND keywords != 'null'
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags with keywords: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var keywordsJSON []byte
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Description,
			&keywordsJSON, &tag.Color, &tag.CreatedAt, &tag.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		if len(keywordsJSON) > 0 {
			if err := json.Unmarshal(keywordsJSON, &tag.Keywords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
			}
		}

		// Only include tags that actually have keywords
		if len(tag.Keywords) > 0 {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// SearchTags searches tags by name or keywords
func (r *TagRepository) SearchTags(query string, limit int) ([]models.Tag, error) {
	searchQuery := `SELECT id, name, slug, description, keywords, color, created_at, language_code
		FROM tags 
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY CASE WHEN name ILIKE $1 THEN 1 ELSE 2 END, name
		LIMIT $2`

	searchTerm := "%" + strings.ToLower(query) + "%"
	rows, err := r.db.Query(searchQuery, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var keywordsJSON []byte
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Description,
			&keywordsJSON, &tag.Color, &tag.CreatedAt, &tag.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		if len(keywordsJSON) > 0 {
			if err := json.Unmarshal(keywordsJSON, &tag.Keywords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
			}
		}

		tags = append(tags, tag)
	}

	return tags, nil
}
// ValidateKeywordUniqueness ensures keywords are unique across all tags
func (r *TagRepository) ValidateKeywordUniqueness(keywords []string, excludeTagID uint64) error {
	if len(keywords) == 0 {
		return nil
	}

	// Get all existing keywords from other tags
	query := `SELECT id, name, keywords FROM tags WHERE id != $1`
	rows, err := r.db.Query(query, excludeTagID)
	if err != nil {
		return fmt.Errorf("failed to query existing tags: %w", err)
	}
	defer rows.Close()

	existingKeywords := make(map[string]string) // keyword -> tag name
	for rows.Next() {
		var tagID uint64
		var tagName string
		var keywordsJSON string

		if err := rows.Scan(&tagID, &tagName, &keywordsJSON); err != nil {
			continue
		}

		var tagKeywords []string
		if err := json.Unmarshal([]byte(keywordsJSON), &tagKeywords); err != nil {
			continue
		}

		for _, keyword := range tagKeywords {
			normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
			if normalizedKeyword != "" {
				existingKeywords[normalizedKeyword] = tagName
			}
		}
	}

	// Check for conflicts
	var conflicts []string
	for _, keyword := range keywords {
		normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
		if normalizedKeyword == "" {
			continue
		}

		if existingTagName, exists := existingKeywords[normalizedKeyword]; exists {
			conflicts = append(conflicts, fmt.Sprintf("keyword '%s' is already used by tag '%s'", keyword, existingTagName))
		}
	}

	if len(conflicts) > 0 {
		return &models.ValidationError{
			Message: "Keyword conflicts detected",
			Fields:  conflicts,
		}
	}

	return nil
}

// ValidateTagNameUniqueness ensures tag names are unique
func (r *TagRepository) ValidateTagNameUniqueness(name string, excludeTagID uint64) error {
	query := `SELECT COUNT(*) FROM tags WHERE LOWER(name) = LOWER($1) AND id != $2`
	var count int
	err := r.db.QueryRow(query, strings.TrimSpace(name), excludeTagID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check tag name uniqueness: %w", err)
	}

	if count > 0 {
		return &models.ValidationError{
			Message: "Tag name already exists",
			Fields:  []string{fmt.Sprintf("tag name '%s' is already in use", name)},
		}
	}

	return nil
}

// ValidateCategoryNameUniqueness ensures category names are unique
func ValidateCategoryNameUniqueness(db *sql.DB, name string, excludeCategoryID uint64) error {
	query := `SELECT COUNT(*) FROM categories WHERE LOWER(name) = LOWER($1) AND id != $2`
	var count int
	err := db.QueryRow(query, strings.TrimSpace(name), excludeCategoryID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check category name uniqueness: %w", err)
	}

	if count > 0 {
		return &models.ValidationError{
			Message: "Category name already exists",
			Fields:  []string{fmt.Sprintf("category name '%s' is already in use", name)},
		}
	}

	return nil
}