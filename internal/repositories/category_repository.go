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

	// Check for duplicate category name
	if err := ValidateCategoryNameUniqueness(r.db, category.Name, 0); err != nil {
		return nil, err
	}

	// If this is a new category (not a translation), create a translation group first
	if category.TranslationGroupID == nil {
		var groupID uint64
		err := r.db.QueryRow(
			"INSERT INTO translation_groups (group_type) VALUES ('category') RETURNING id",
		).Scan(&groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to create translation group: %w", err)
		}
		category.TranslationGroupID = &groupID
	}

	query := `
		INSERT INTO categories (name, slug, description, parent_id, sort_order, language_code, translation_group_id, image_url, image_alt_text, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	now := time.Now()
	err := r.db.QueryRow(query, category.Name, category.Slug, category.Description,
		category.ParentID, category.SortOrder, category.LanguageCode, category.TranslationGroupID, category.ImageURL, category.ImageAltText, now).Scan(&category.ID, &category.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id uint64) (*models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code, translation_group_id
		FROM categories 
		WHERE id = $1`

	category := &models.Category{}
	var translationGroupID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Description,
		&category.ParentID, &category.SortOrder, &category.ImageURL, &category.ImageAltText, &category.CreatedAt,
		&category.LanguageCode, &translationGroupID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &models.NotFoundError{Resource: "category", ID: strconv.FormatUint(id, 10)}
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if translationGroupID.Valid {
		val := uint64(translationGroupID.Int64)
		category.TranslationGroupID = &val
	}

	return category, nil
}

// GetAll retrieves all categories
func (r *CategoryRepository) GetAll() ([]models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code, translation_group_id
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
		var translationGroupID sql.NullInt64
		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &category.Description,
			&category.ParentID, &category.SortOrder, &category.ImageURL, &category.ImageAltText, &category.CreatedAt, &category.LanguageCode, &translationGroupID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		if translationGroupID.Valid {
			val := uint64(translationGroupID.Int64)
			category.TranslationGroupID = &val
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

	// Check for duplicate category name (excluding this category)
	if err := ValidateCategoryNameUniqueness(r.db, category.Name, category.ID); err != nil {
		return err
	}

	query := `
		UPDATE categories
		SET name = $1, slug = $2, description = $3, parent_id = $4, sort_order = $5, language_code = $6, image_url = $7, image_alt_text = $8
		WHERE id = $9`

		result, err := r.db.Exec(query, category.Name, category.Slug, category.Description,
			category.ParentID, category.SortOrder, category.LanguageCode, category.ImageURL, category.ImageAltText, category.ID)
	
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
	// First try with the specified language, then try without language constraint
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, 
			   language_code, translation_group_id
		FROM categories
		WHERE slug = $1 AND language_code = $2`

	var category models.Category
	var description sql.NullString
	var parentID sql.NullInt64
	var imageURL sql.NullString
	var imageAltText sql.NullString
	var translationGroupID sql.NullInt64
	
	err := r.db.QueryRow(query, slug, languageCode).Scan(
		&category.ID, &category.Name, &category.Slug, &description,
		&parentID, &category.SortOrder, &imageURL, &imageAltText, &category.CreatedAt,
		&category.LanguageCode, &translationGroupID,
	)

	if err == sql.ErrNoRows {
		// Try without language constraint
		query = `
			SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, 
				   language_code, translation_group_id
			FROM categories
			WHERE slug = $1 LIMIT 1`
		
		err = r.db.QueryRow(query, slug).Scan(
			&category.ID, &category.Name, &category.Slug, &description,
			&parentID, &category.SortOrder, &imageURL, &imageAltText, &category.CreatedAt,
			&category.LanguageCode, &translationGroupID,
		)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}

	// Handle nullable fields
	if description.Valid {
		category.Description = description.String
	}
	if parentID.Valid {
		val := uint64(parentID.Int64)
		category.ParentID = &val
	}
	if imageURL.Valid {
		category.ImageURL = &imageURL.String
	}
	if imageAltText.Valid {
		category.ImageAltText = &imageAltText.String
	}
	if translationGroupID.Valid {
		val := uint64(translationGroupID.Int64)
		category.TranslationGroupID = &val
	}

	return &category, nil
}

// BulkCreate creates multiple categories in a single transaction
func (r *CategoryRepository) BulkCreate(categories []models.Category) ([]models.Category, error) {
	if len(categories) == 0 {
		return categories, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO categories (name, slug, description, parent_id, sort_order, language_code, image_url, image_alt_text, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	results := make([]models.Category, len(categories))
	now := time.Now()

	for i, category := range categories {
		category.PrepareForDB()
		if err := models.ValidateCategory(&category); err != nil {
			return nil, fmt.Errorf("validation failed for category %d: %w", i, err)
		}

		err := tx.QueryRow(query, category.Name, category.Slug, category.Description,
			category.ParentID, category.SortOrder, category.LanguageCode, 
			category.ImageURL, category.ImageAltText, now).Scan(&category.ID, &category.CreatedAt)
		
		if err != nil {
			return nil, fmt.Errorf("failed to create category %d: %w", i, err)
		}

		results[i] = category
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
}

// GetChildren retrieves all child categories of a parent category
func (r *CategoryRepository) GetChildren(ctx context.Context, parentID uint64) ([]models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code
		FROM categories
		WHERE parent_id = $1
		ORDER BY sort_order, name`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query child categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		var description, imageURL, imageAltText sql.NullString
		var parentID sql.NullInt64

		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &description,
			&parentID, &category.SortOrder, &imageURL, &imageAltText,
			&category.CreatedAt, &category.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			category.Description = description.String
		}
		if imageURL.Valid {
			category.ImageURL = &imageURL.String
		}
		if imageAltText.Valid {
			category.ImageAltText = &imageAltText.String
		}
		if parentID.Valid {
			pid := uint64(parentID.Int64)
			category.ParentID = &pid
		}

		categories = append(categories, category)
	}

	return categories, nil
}

// GetRootCategories retrieves all root categories (categories without parent)
func (r *CategoryRepository) GetRootCategories(ctx context.Context) ([]models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code
		FROM categories
		WHERE parent_id IS NULL
		ORDER BY sort_order, name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query root categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		var description, imageURL, imageAltText sql.NullString
		var parentID sql.NullInt64

		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &description,
			&parentID, &category.SortOrder, &imageURL, &imageAltText,
			&category.CreatedAt, &category.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			category.Description = description.String
		}
		if imageURL.Valid {
			category.ImageURL = &imageURL.String
		}
		if imageAltText.Valid {
			category.ImageAltText = &imageAltText.String
		}
		if parentID.Valid {
			pid := uint64(parentID.Int64)
			category.ParentID = &pid
		}

		categories = append(categories, category)
	}

	return categories, nil
}

// GetCategoryPath retrieves the full path from root to the specified category
func (r *CategoryRepository) GetCategoryPath(ctx context.Context, categoryID uint64) ([]models.Category, error) {
	query := `
		WITH RECURSIVE category_path AS (
			-- Base case: start with the target category
			SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code, 0 as level
			FROM categories
			WHERE id = $1
			
			UNION ALL
			
			-- Recursive case: get parent categories
			SELECT c.id, c.name, c.slug, c.description, c.parent_id, c.sort_order, c.image_url, c.image_alt_text, c.created_at, c.language_code, cp.level + 1
			FROM categories c
			INNER JOIN category_path cp ON c.id = cp.parent_id
		)
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, created_at, language_code
		FROM category_path
		ORDER BY level DESC`

	rows, err := r.db.QueryContext(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query category path: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		var description, imageURL, imageAltText sql.NullString
		var parentID sql.NullInt64

		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &description,
			&parentID, &category.SortOrder, &imageURL, &imageAltText,
			&category.CreatedAt, &category.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			category.Description = description.String
		}
		if imageURL.Valid {
			category.ImageURL = &imageURL.String
		}
		if imageAltText.Valid {
			category.ImageAltText = &imageAltText.String
		}
		if parentID.Valid {
			pid := uint64(parentID.Int64)
			category.ParentID = &pid
		}

		categories = append(categories, category)
	}

	return categories, nil
}