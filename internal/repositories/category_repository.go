package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"high-performance-news-website/internal/models"
)

// CategoryRepository handles database operations for categories
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create creates a new category
func (r *CategoryRepository) Create(category *models.Category) (*models.Category, error) {
	category.PrepareForDB()
	
	if err := models.ValidateCategory(category); err != nil {
		return nil, err
	}

	query := `
		INSERT INTO categories (name, slug, description, parent_id, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	now := time.Now()
	err := r.db.QueryRow(query, category.Name, category.Slug, category.Description, 
		category.ParentID, category.SortOrder, now).Scan(&category.ID, &category.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id uint64) (*models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, created_at
		FROM categories 
		WHERE id = $1`

	category := &models.Category{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Description,
		&category.ParentID, &category.SortOrder, &category.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &models.NotFoundError{Resource: "category", ID: strconv.FormatUint(id, 10)}
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return category, nil
}

// GetAll retrieves all categories
func (r *CategoryRepository) GetAll() ([]models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, created_at
		FROM categories 
		ORDER BY parent_id NULLS FIRST, sort_order, name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &category.Description,
			&category.ParentID, &category.SortOrder, &category.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// Update updates an existing category
func (r *CategoryRepository) Update(category *models.Category) error {
	category.PrepareForDB()
	
	if err := models.ValidateCategory(category); err != nil {
		return err
	}

	query := `
		UPDATE categories 
		SET name = $1, slug = $2, description = $3, parent_id = $4, sort_order = $5
		WHERE id = $6`

	result, err := r.db.Exec(query, category.Name, category.Slug, category.Description,
		category.ParentID, category.SortOrder, category.ID)
	
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &models.NotFoundError{Resource: "category", ID: strconv.FormatUint(category.ID, 10)}
	}

	return nil
}

// Delete deletes a category
func (r *CategoryRepository) Delete(id uint64) error {
	query := `DELETE FROM categories WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &models.NotFoundError{Resource: "category", ID: strconv.FormatUint(id, 10)}
	}

	return nil
}

// GetBySlug retrieves a category by slug and language code
func (r *CategoryRepository) GetBySlug(slug, languageCode string) (*models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, created_at, updated_at, 
			   language_code, translation_group_id
		FROM categories
		WHERE slug = $1 AND language_code = $2`

	var category models.Category
	err := r.db.QueryRow(query, slug, languageCode).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Description,
		&category.ParentID, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt,
		&category.LanguageCode, &category.TranslationGroupID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}

	return &category, nil
}