package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"high-performance-news-website/internal/models"
)

// MediaService handles media-related operations
type MediaService struct {
	db *sql.DB
}

// NewMediaService creates a new MediaService instance
func NewMediaService(db *sql.DB) *MediaService {
	return &MediaService{
		db: db,
	}
}

// GetImageByHash retrieves an image by its content hash for deduplication
func (ms *MediaService) GetImageByHash(hash string) (*models.Image, []models.ImageVariant, error) {
	if hash == "" {
		return nil, nil, fmt.Errorf("empty hash provided")
	}

	query := `
		SELECT id, original_url, filename, alt_text, caption, width, height, 
			   file_size, mime_type, hash, article_id, created_at, updated_at
		FROM images 
		WHERE hash = $1
		LIMIT 1
	`

	var img models.Image
	var articleID sql.NullInt64
	var altText, caption, imgHash sql.NullString

	err := ms.db.QueryRow(query, hash).Scan(
		&img.ID, &img.OriginalURL, &img.Filename, &altText, &caption,
		&img.Width, &img.Height, &img.FileSize, &img.MimeType,
		&imgHash, &articleID, &img.CreatedAt, &img.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil, nil // No existing image found
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query image by hash: %w", err)
	}

	if altText.Valid {
		img.AltText = altText.String
	}
	if caption.Valid {
		img.Caption = caption.String
	}
	if imgHash.Valid {
		img.Hash = imgHash.String
	}
	if articleID.Valid {
		articleIDUint := uint64(articleID.Int64)
		img.ArticleID = &articleIDUint
	}

	// Get variants for this image
	variants, err := ms.GetImageVariants(img.ID)
	if err != nil {
		log.Printf("Warning: Failed to get variants for deduplicated image %d: %v", img.ID, err)
		variants = []models.ImageVariant{}
	}

	return &img, variants, nil
}

// GetImageByID retrieves an image by its ID
func (ms *MediaService) GetImageByID(imageID uint64) (*models.Image, error) {
	query := `
		SELECT id, original_url, filename, alt_text, caption, width, height, 
			   file_size, mime_type, hash, article_id, created_at, updated_at
		FROM images 
		WHERE id = $1
	`

	var img models.Image
	var articleID sql.NullInt64
	var altText, caption, imgHash sql.NullString

	err := ms.db.QueryRow(query, imageID).Scan(
		&img.ID, &img.OriginalURL, &img.Filename, &altText, &caption,
		&img.Width, &img.Height, &img.FileSize, &img.MimeType,
		&imgHash, &articleID, &img.CreatedAt, &img.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("image not found: %d", imageID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query image by ID: %w", err)
	}

	if altText.Valid {
		img.AltText = altText.String
	}
	if caption.Valid {
		img.Caption = caption.String
	}
	if imgHash.Valid {
		img.Hash = imgHash.String
	}
	if articleID.Valid {
		articleIDUint := uint64(articleID.Int64)
		img.ArticleID = &articleIDUint
	}

	return &img, nil
}

// CreateImage saves a new image to the database
func (ms *MediaService) CreateImage(image *models.Image) error {
	query := `
		INSERT INTO images (id, original_url, filename, alt_text, caption, width, height, 
			file_size, mime_type, article_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := ms.db.Exec(query,
		image.ID, image.OriginalURL, image.Filename, image.AltText, image.Caption,
		image.Width, image.Height, image.FileSize, image.MimeType, image.ArticleID,
		image.CreatedAt, image.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}
	
	return nil
}

// CreateImageVariant saves a new image variant to the database
func (ms *MediaService) CreateImageVariant(variant *models.ImageVariant) error {
	query := `
		INSERT INTO image_variants (image_id, size, format, url, width, height, 
			file_size, quality)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := ms.db.Exec(query,
		variant.ImageID, variant.Size, variant.Format, variant.URL,
		variant.Width, variant.Height, variant.FileSize, variant.Quality,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create image variant: %w", err)
	}
	
	return nil
}

// CreateImageWithVariants saves an image and its variants in a transaction
func (ms *MediaService) CreateImageWithVariants(image *models.Image, variants []models.ImageVariant) error {
	log.Printf("CreateImageWithVariants called for image ID: %d, variants: %d", image.ID, len(variants))

	// Start transaction
	tx, err := ms.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert main image (including hash for deduplication)
	imageQuery := `
		INSERT INTO images (id, original_url, filename, alt_text, caption, width, height, 
			file_size, mime_type, hash, article_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = tx.Exec(imageQuery,
		image.ID, image.OriginalURL, image.Filename, image.AltText, image.Caption,
		image.Width, image.Height, image.FileSize, image.MimeType, image.Hash,
		image.ArticleID, image.CreatedAt, image.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert image: %w", err)
	}

	// Insert variants
	if len(variants) > 0 {
		variantQuery := `
			INSERT INTO image_variants (image_id, size, format, url, width, height, 
				file_size, quality)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		for _, variant := range variants {
			_, err = tx.Exec(variantQuery,
				variant.ImageID, variant.Size, variant.Format, variant.URL,
				variant.Width, variant.Height, variant.FileSize, variant.Quality,
			)

			if err != nil {
				return fmt.Errorf("failed to insert image variant: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully saved image ID %d with %d variants to database", image.ID, len(variants))
	return nil
}

// GetAllMedia returns all media files with their variants
func (ms *MediaService) GetAllMedia() ([]models.Image, error) {
	query := `
		SELECT id, original_url, filename, alt_text, caption, width, height, 
			   file_size, mime_type, article_id, created_at, updated_at
		FROM images 
		ORDER BY created_at DESC
	`
	
	rows, err := ms.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()
	
	var images []models.Image
	for rows.Next() {
		var img models.Image
		var articleID sql.NullInt64
		var altText sql.NullString
		var caption sql.NullString
		
		err := rows.Scan(
			&img.ID, &img.OriginalURL, &img.Filename, &altText, &caption,
			&img.Width, &img.Height, &img.FileSize, &img.MimeType,
			&articleID, &img.CreatedAt, &img.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		
		if altText.Valid {
			img.AltText = altText.String
		}
		if caption.Valid {
			img.Caption = caption.String
		}
		if articleID.Valid {
			articleIDUint := uint64(articleID.Int64)
			img.ArticleID = &articleIDUint
		}
		
		images = append(images, img)
	}
	
	return images, nil
}

// GetImageVariants returns all variants for a specific image
func (ms *MediaService) GetImageVariants(imageID uint64) ([]models.ImageVariant, error) {
	query := `
		SELECT id, image_id, size, format, url, width, height, file_size, quality
		FROM image_variants 
		WHERE image_id = $1
		ORDER BY size, format
	`
	
	rows, err := ms.db.Query(query, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query image variants: %w", err)
	}
	defer rows.Close()
	
	var variants []models.ImageVariant
	for rows.Next() {
		var variant models.ImageVariant
		
		err := rows.Scan(
			&variant.ID, &variant.ImageID, &variant.Size, &variant.Format,
			&variant.URL, &variant.Width, &variant.Height, &variant.FileSize, &variant.Quality,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image variant: %w", err)
		}
		
		variants = append(variants, variant)
	}
	
	return variants, nil
}

// DeleteMedia deletes an image and all its variants from database and disk
func (ms *MediaService) DeleteMedia(imageID uint64) error {
	// First, get the image and variant URLs before deleting from database
	imageQuery := "SELECT original_url FROM images WHERE id = $1"
	var originalURL string
	err := ms.db.QueryRow(imageQuery, imageID).Scan(&originalURL)
	if err != nil {
		return fmt.Errorf("failed to get image URL: %w", err)
	}
	
	// Get all variant URLs
	variantQuery := "SELECT url FROM image_variants WHERE image_id = $1"
	rows, err := ms.db.Query(variantQuery, imageID)
	if err != nil {
		return fmt.Errorf("failed to get variant URLs: %w", err)
	}
	defer rows.Close()
	
	var variantURLs []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return fmt.Errorf("failed to scan variant URL: %w", err)
		}
		variantURLs = append(variantURLs, url)
	}
	
	// Start transaction for database deletion
	tx, err := ms.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete variants from database first (due to foreign key constraint)
	_, err = tx.Exec("DELETE FROM image_variants WHERE image_id = $1", imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image variants: %w", err)
	}
	
	// Delete main image from database
	result, err := tx.Exec("DELETE FROM images WHERE id = $1", imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("image not found")
	}
	
	// Commit database transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Delete physical files (after successful database deletion)
	// Convert URL paths to file system paths
	basePath := "web/static/uploads/images"
	
	// Delete original file
	if originalURL != "" {
		originalPath := basePath + originalURL[8:] // Remove "/uploads" prefix
		if err := os.Remove(originalPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: Failed to delete original file %s: %v\n", originalPath, err)
		}
	}
	
	// Delete variant files
	for _, variantURL := range variantURLs {
		if variantURL != "" {
			variantPath := basePath + variantURL[8:] // Remove "/uploads" prefix
			if err := os.Remove(variantPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: Failed to delete variant file %s: %v\n", variantPath, err)
			}
		}
	}
	
	return nil
}
