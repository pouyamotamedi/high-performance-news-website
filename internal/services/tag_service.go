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
