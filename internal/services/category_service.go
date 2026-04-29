package services

import (
	"context"
	"database/sql"
	"strings"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// CategoryService handles category-related operations
type CategoryService struct {
	db *sql.DB
}

// NewCategoryService creates a new CategoryService instance
func NewCategoryService(db *sql.DB) *CategoryService {
	return &CategoryService{
		db: db,
	}
}

// GetAll returns all categories
func (cs *CategoryService) GetAll() ([]models.Category, error) {
	repo := repositories.NewCategoryRepository(cs.db)
	return repo.GetAll()
}

// GetByID returns a category by ID
func (cs *CategoryService) GetByID(id uint64) (*models.Category, error) {
	repo := repositories.NewCategoryRepository(cs.db)
	return repo.GetByID(context.Background(), id)
}

// GetBySlug returns a category by slug
func (cs *CategoryService) GetBySlug(slug string) (*models.Category, error) {
	repo := repositories.NewCategoryRepository(cs.db)
	return repo.GetBySlug(slug, "en") // Default to English language
}

// Create creates a new category
func (cs *CategoryService) Create(category *models.Category) error {
	repo := repositories.NewCategoryRepository(cs.db)
	_, err := repo.Create(category)
	return err
}

// Update updates an existing category
func (cs *CategoryService) Update(category *models.Category) error {
	repo := repositories.NewCategoryRepository(cs.db)
	return repo.Update(category)
}

// Delete deletes a category
func (cs *CategoryService) Delete(id uint64) error {
	repo := repositories.NewCategoryRepository(cs.db)
	err := repo.Delete(id)
	if err != nil {
		// Handle foreign key constraint violations
		if strings.Contains(err.Error(), "foreign key constraint") && strings.Contains(err.Error(), "articles") {
			return &models.ValidationError{
				Message: "Cannot delete category",
				Fields:  []string{"This category is currently used by one or more articles. Please move all articles to another category before deleting this one."},
			}
		}
		if strings.Contains(err.Error(), "foreign key constraint") && strings.Contains(err.Error(), "categories") {
			return &models.ValidationError{
				Message: "Cannot delete category",
				Fields:  []string{"This category has child categories. Please delete or move all child categories first."},
			}
		}
		return err
	}
	return nil
}

// GetTotalCount returns the total number of categories
func (cs *CategoryService) GetTotalCount() (int64, error) {
	categories, err := cs.GetAll()
	if err != nil {
		return 0, err
	}
	return int64(len(categories)), nil
}
