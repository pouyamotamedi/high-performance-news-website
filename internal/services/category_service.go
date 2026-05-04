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

// GetByTranslationGroupAndLanguage returns a category by translation_group_id and language_code
// This is used to find the correct localized version of a category for a specific language
func (cs *CategoryService) GetByTranslationGroupAndLanguage(translationGroupID uint64, languageCode string) (*models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, 
		       created_at, updated_at, language_code, translation_group_id
		FROM categories 
		WHERE (translation_group_id = $1 OR id = $1) AND language_code = $2
		LIMIT 1
	`
	
	var category models.Category
	var parentID, translationGrpID sql.NullInt64
	var imageURL, imageAltText sql.NullString
	
	err := cs.db.QueryRow(query, translationGroupID, languageCode).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&parentID,
		&category.SortOrder,
		&imageURL,
		&imageAltText,
		&category.CreatedAt,
		&category.UpdatedAt,
		&category.LanguageCode,
		&translationGrpID,
	)
	
	if err == sql.ErrNoRows {
		// If no translation exists for this language, return the original category
		return cs.GetByID(translationGroupID)
	}
	if err != nil {
		return nil, err
	}
	
	if parentID.Valid {
		pid := uint64(parentID.Int64)
		category.ParentID = &pid
	}
	if translationGrpID.Valid {
		tgid := uint64(translationGrpID.Int64)
		category.TranslationGroupID = &tgid
	}
	if imageURL.Valid {
		category.ImageURL = &imageURL.String
	}
	if imageAltText.Valid {
		category.ImageAltText = &imageAltText.String
	}
	
	return &category, nil
}

// GetTranslationGroupID returns the translation_group_id for a category
// If the category doesn't have a translation_group_id, it returns the category's own ID
func (cs *CategoryService) GetTranslationGroupID(categoryID uint64) (uint64, error) {
	query := `SELECT COALESCE(translation_group_id, id) FROM categories WHERE id = $1`
	var groupID uint64
	err := cs.db.QueryRow(query, categoryID).Scan(&groupID)
	if err != nil {
		return 0, err
	}
	return groupID, nil
}

// GetAllTranslations returns all translations of a category (all categories in the same translation group)
func (cs *CategoryService) GetAllTranslations(translationGroupID uint64) ([]models.Category, error) {
	query := `
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, 
		       created_at, updated_at, language_code, translation_group_id
		FROM categories 
		WHERE translation_group_id = $1 OR id = $1
		ORDER BY language_code
	`
	
	rows, err := cs.db.Query(query, translationGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var categories []models.Category
	for rows.Next() {
		var category models.Category
		var parentID, translationGrpID sql.NullInt64
		var imageURL, imageAltText sql.NullString
		
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&parentID,
			&category.SortOrder,
			&imageURL,
			&imageAltText,
			&category.CreatedAt,
			&category.UpdatedAt,
			&category.LanguageCode,
			&translationGrpID,
		)
		if err != nil {
			return nil, err
		}
		
		if parentID.Valid {
			pid := uint64(parentID.Int64)
			category.ParentID = &pid
		}
		if translationGrpID.Valid {
			tgid := uint64(translationGrpID.Int64)
			category.TranslationGroupID = &tgid
		}
		if imageURL.Valid {
			category.ImageURL = &imageURL.String
		}
		if imageAltText.Valid {
			category.ImageAltText = &imageAltText.String
		}
		
		categories = append(categories, category)
	}
	
	return categories, nil
}

// GetCoreCategories returns all unique category concepts (one per translation group)
// This is used in admin UI to show categories without language filtering
func (cs *CategoryService) GetCoreCategories() ([]models.Category, error) {
	// Get categories that are either:
	// 1. The "root" of a translation group (translation_group_id IS NULL)
	// 2. Or the English version if available
	// 3. Or the first one in the group
	query := `
		WITH ranked_categories AS (
			SELECT *,
				ROW_NUMBER() OVER (
					PARTITION BY COALESCE(translation_group_id, id)
					ORDER BY 
						CASE WHEN translation_group_id IS NULL THEN 0 ELSE 1 END,
						CASE WHEN language_code = 'en' THEN 0 ELSE 1 END,
						id
				) as rn
			FROM categories
		)
		SELECT id, name, slug, description, parent_id, sort_order, image_url, image_alt_text, 
		       created_at, updated_at, language_code, translation_group_id
		FROM ranked_categories
		WHERE rn = 1
		ORDER BY sort_order, name
	`
	
	rows, err := cs.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var categories []models.Category
	for rows.Next() {
		var category models.Category
		var parentID, translationGrpID sql.NullInt64
		var imageURL, imageAltText sql.NullString
		
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&parentID,
			&category.SortOrder,
			&imageURL,
			&imageAltText,
			&category.CreatedAt,
			&category.UpdatedAt,
			&category.LanguageCode,
			&translationGrpID,
		)
		if err != nil {
			return nil, err
		}
		
		if parentID.Valid {
			pid := uint64(parentID.Int64)
			category.ParentID = &pid
		}
		if translationGrpID.Valid {
			tgid := uint64(translationGrpID.Int64)
			category.TranslationGroupID = &tgid
		}
		if imageURL.Valid {
			category.ImageURL = &imageURL.String
		}
		if imageAltText.Valid {
			category.ImageAltText = &imageAltText.String
		}
		
		categories = append(categories, category)
	}
	
	return categories, nil
}
