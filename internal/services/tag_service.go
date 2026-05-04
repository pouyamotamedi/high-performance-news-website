package services

import (
	"context"
	"database/sql"
	"strings"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// TagService handles tag-related operations
type TagService struct {
	db *sql.DB
}

// NewTagService creates a new TagService instance
func NewTagService(db *sql.DB) *TagService {
	return &TagService{
		db: db,
	}
}

// GetDB returns the database connection
func (ts *TagService) GetDB() *sql.DB {
	return ts.db
}

// GetAll returns all tags
func (ts *TagService) GetAll() ([]models.Tag, error) {
	repo := repositories.NewTagRepository(ts.db)
	return repo.GetAll()
}

// GetByID returns a tag by ID
func (ts *TagService) GetByID(id uint64) (*models.Tag, error) {
	repo := repositories.NewTagRepository(ts.db)
	return repo.GetByID(context.Background(), id)
}

// GetBySlug returns a tag by slug
func (ts *TagService) GetBySlug(slug string) (*models.Tag, error) {
	repo := repositories.NewTagRepository(ts.db)
	return repo.GetBySlug(slug, "en") // Default to English
}

// Create creates a new tag
func (ts *TagService) Create(tag *models.Tag) error {
	repo := repositories.NewTagRepository(ts.db)
	_, err := repo.Create(tag)
	return err
}

// Update updates an existing tag
func (ts *TagService) Update(tag *models.Tag) error {
	repo := repositories.NewTagRepository(ts.db)
	return repo.Update(tag)
}

// Delete deletes a tag
func (ts *TagService) Delete(id uint64) error {
	repo := repositories.NewTagRepository(ts.db)
	err := repo.Delete(id)
	if err != nil {
		// Handle foreign key constraint violations
		if strings.Contains(err.Error(), "foreign key constraint") && strings.Contains(err.Error(), "article_tags") {
			return &models.ValidationError{
				Message: "Cannot delete tag",
				Fields:  []string{"This tag is currently used by one or more articles. Please remove the tag from all articles before deleting it."},
			}
		}
		return err
	}
	return nil
}

// GetTotalCount returns the total number of tags
func (ts *TagService) GetTotalCount() (int64, error) {
	tags, err := ts.GetAll()
	if err != nil {
		return 0, err
	}
	return int64(len(tags)), nil
}

// GetPopular returns popular tags (by usage count)
func (ts *TagService) GetPopular(limit int) ([]models.Tag, error) {
	// For now, return all tags limited by count
	// TODO: Implement actual popularity ranking based on article_tags usage
	tags, err := ts.GetAll()
	if err != nil {
		return nil, err
	}
	
	if limit > len(tags) {
		limit = len(tags)
	}
	
	if limit <= 0 {
		return tags, nil
	}
	
	return tags[:limit], nil
}

// GetByTranslationGroupAndLanguage returns a tag by translation_group_id and language_code
// This is used to find the correct localized version of a tag for a specific language
func (ts *TagService) GetByTranslationGroupAndLanguage(translationGroupID uint64, languageCode string) (*models.Tag, error) {
	query := `
		SELECT id, name, slug, description, keywords, color, created_at, updated_at, 
		       language_code, translation_group_id
		FROM tags 
		WHERE (translation_group_id = $1 OR id = $1) AND language_code = $2
		LIMIT 1
	`
	
	var tag models.Tag
	var translationGrpID sql.NullInt64
	var keywords sql.NullString
	
	err := ts.db.QueryRow(query, translationGroupID, languageCode).Scan(
		&tag.ID,
		&tag.Name,
		&tag.Slug,
		&tag.Description,
		&keywords,
		&tag.Color,
		&tag.CreatedAt,
		&tag.UpdatedAt,
		&tag.LanguageCode,
		&translationGrpID,
	)
	
	if err == sql.ErrNoRows {
		// If no translation exists for this language, return the original tag
		return ts.GetByID(translationGroupID)
	}
	if err != nil {
		return nil, err
	}
	
	if translationGrpID.Valid {
		tgid := uint64(translationGrpID.Int64)
		tag.TranslationGroupID = &tgid
	}
	
	// Parse keywords JSON
	if keywords.Valid && keywords.String != "" {
		// Keywords are stored as JSON array
		tag.Keywords = parseKeywordsJSON(keywords.String)
	}
	
	return &tag, nil
}

// parseKeywordsJSON parses a JSON array string into []string
func parseKeywordsJSON(jsonStr string) []string {
	// Simple parsing for JSON array like ["keyword1", "keyword2"]
	jsonStr = strings.TrimPrefix(jsonStr, "[")
	jsonStr = strings.TrimSuffix(jsonStr, "]")
	if jsonStr == "" {
		return nil
	}
	
	var keywords []string
	parts := strings.Split(jsonStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"")
		if part != "" {
			keywords = append(keywords, part)
		}
	}
	return keywords
}

// GetTranslationGroupID returns the translation_group_id for a tag
// If the tag doesn't have a translation_group_id, it returns the tag's own ID
func (ts *TagService) GetTranslationGroupID(tagID uint64) (uint64, error) {
	query := `SELECT COALESCE(translation_group_id, id) FROM tags WHERE id = $1`
	var groupID uint64
	err := ts.db.QueryRow(query, tagID).Scan(&groupID)
	if err != nil {
		return 0, err
	}
	return groupID, nil
}

// GetAllTranslations returns all translations of a tag (all tags in the same translation group)
func (ts *TagService) GetAllTranslations(translationGroupID uint64) ([]models.Tag, error) {
	query := `
		SELECT id, name, slug, description, keywords, color, created_at, updated_at, 
		       language_code, translation_group_id
		FROM tags 
		WHERE translation_group_id = $1 OR id = $1
		ORDER BY language_code
	`
	
	rows, err := ts.db.Query(query, translationGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var translationGrpID sql.NullInt64
		var keywords sql.NullString
		
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Description,
			&keywords,
			&tag.Color,
			&tag.CreatedAt,
			&tag.UpdatedAt,
			&tag.LanguageCode,
			&translationGrpID,
		)
		if err != nil {
			return nil, err
		}
		
		if translationGrpID.Valid {
			tgid := uint64(translationGrpID.Int64)
			tag.TranslationGroupID = &tgid
		}
		
		if keywords.Valid && keywords.String != "" {
			tag.Keywords = parseKeywordsJSON(keywords.String)
		}
		
		tags = append(tags, tag)
	}
	
	return tags, nil
}

// GetCoreTags returns all unique tag concepts (one per translation group)
// This is used in admin UI to show tags without language filtering
func (ts *TagService) GetCoreTags() ([]models.Tag, error) {
	query := `
		WITH ranked_tags AS (
			SELECT *,
				ROW_NUMBER() OVER (
					PARTITION BY COALESCE(translation_group_id, id)
					ORDER BY 
						CASE WHEN translation_group_id IS NULL THEN 0 ELSE 1 END,
						CASE WHEN language_code = 'en' THEN 0 ELSE 1 END,
						id
				) as rn
			FROM tags
		)
		SELECT id, name, slug, description, keywords, color, created_at, updated_at, 
		       language_code, translation_group_id
		FROM ranked_tags
		WHERE rn = 1
		ORDER BY name
	`
	
	rows, err := ts.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var translationGrpID sql.NullInt64
		var keywords sql.NullString
		
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Description,
			&keywords,
			&tag.Color,
			&tag.CreatedAt,
			&tag.UpdatedAt,
			&tag.LanguageCode,
			&translationGrpID,
		)
		if err != nil {
			return nil, err
		}
		
		if translationGrpID.Valid {
			tgid := uint64(translationGrpID.Int64)
			tag.TranslationGroupID = &tgid
		}
		
		if keywords.Valid && keywords.String != "" {
			tag.Keywords = parseKeywordsJSON(keywords.String)
		}
		
		tags = append(tags, tag)
	}
	
	return tags, nil
}
