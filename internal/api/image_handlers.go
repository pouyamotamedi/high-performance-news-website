package api

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"

	"github.com/gin-gonic/gin"
)

// ImageHandlers contains handlers for image-related operations
type ImageHandlers struct {
	imageProcessor *services.ImageProcessor
	mediaService   *services.MediaService
	uploadPath     string
	maxFileSize    int64
}

// NewImageHandlers creates new image handlers
func NewImageHandlers(imageProcessor *services.ImageProcessor, mediaService *services.MediaService, uploadPath string, maxFileSize int64) *ImageHandlers {
	return &ImageHandlers{
		imageProcessor: imageProcessor,
		mediaService:   mediaService,
		uploadPath:     uploadPath,
		maxFileSize:    maxFileSize,
	}
}

// UploadImageRequest represents an image upload request
type UploadImageRequest struct {
	AltText   string `form:"alt_text"`
	Caption   string `form:"caption"`
	ArticleID string `form:"article_id"`
}

// UploadImageResponse represents an image upload response
type UploadImageResponse struct {
	Image      *models.Image         `json:"image"`
	Variants   []models.ImageVariant `json:"variants"`
	HTML       string                `json:"html"`
	Deduplicated bool                `json:"deduplicated,omitempty"`
}

// UploadImage handles image upload and processing
// It includes deduplication via content hash and atomic deletion of originals
func (h *ImageHandlers) UploadImage(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(h.maxFileSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to parse multipart form",
			"details": err.Error(),
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No image file provided",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > h.maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", h.maxFileSize),
		})
		return
	}

	// Validate file type
	if !h.isValidImageType(header.Header.Get("Content-Type")) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Only JPEG, PNG, WebP, and SVG images are allowed",
		})
		return
	}

	// Parse request data
	var req UploadImageRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Generate unique filename for temporary storage
	imageID := uint64(time.Now().UnixNano())
	originalFilename := h.generateOriginalFilename(imageID, header.Filename)
	originalPath := filepath.Join(h.uploadPath, "originals", originalFilename)

	// Ensure upload directory exists
	if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create upload directory",
			"details": err.Error(),
		})
		return
	}

	// Save original file temporarily
	if err := h.saveUploadedFile(file, originalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save uploaded file",
			"details": err.Error(),
		})
		return
	}

	// DEDUPLICATION: Compute hash and check for existing image
	imageHash, err := h.imageProcessor.ComputeImageHash(originalPath)
	if err != nil {
		log.Printf("Warning: Failed to compute image hash: %v", err)
	} else if h.mediaService != nil {
		// Check if image with same hash already exists
		existingImage, existingVariants, err := h.mediaService.GetImageByHash(imageHash)
		if err == nil && existingImage != nil {
			// Image already exists - delete the temporary upload and return existing
			os.Remove(originalPath)
			log.Printf("Deduplication: Image with hash %s already exists (ID: %d)", imageHash, existingImage.ID)

			// Update article association if provided
			if req.ArticleID != "" {
				if articleID, err := strconv.ParseUint(req.ArticleID, 10, 64); err == nil {
					existingImage.ArticleID = &articleID
				}
			}

			html := h.imageProcessor.GenerateResponsiveImageHTML(existingImage, existingVariants, true)

			response := UploadImageResponse{
				Image:        existingImage,
				Variants:     existingVariants,
				HTML:         html,
				Deduplicated: true,
			}

			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Create image model
	image := &models.Image{
		ID:          imageID,
		OriginalURL: fmt.Sprintf("/uploads/originals/%s", originalFilename),
		Filename:    header.Filename,
		AltText:     req.AltText,
		Caption:     req.Caption,
		FileSize:    header.Size,
		MimeType:    header.Header.Get("Content-Type"),
		Hash:        imageHash,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Parse article ID if provided
	if req.ArticleID != "" {
		if articleID, err := strconv.ParseUint(req.ArticleID, 10, 64); err == nil {
			image.ArticleID = &articleID
		}
	}

	// Check if this is an SVG file - SVGs don't need processing
	isSVG := header.Header.Get("Content-Type") == "image/svg+xml"
	
	var variants []models.ImageVariant
	var criticalVariantsGenerated bool

	if isSVG {
		// SVG files are vector graphics - no processing needed
		// Just keep the original file and return it
		log.Printf("SVG file uploaded - skipping image processing: %s", header.Filename)
		criticalVariantsGenerated = false // Keep the original
		variants = []models.ImageVariant{} // No variants for SVG
	} else {
		// Process image variants for raster images
		// Note: We request JPEG and WebP. AVIF is not natively supported,
		// so we skip it to avoid confusion (AVIF requests would produce WebP files).
		sizes := []models.ImageSize{
			models.ImageSizeThumbnail,
			models.ImageSizeSmall,
			models.ImageSizeMedium,
			models.ImageSizeLarge,
		}

		formats := []models.ImageFormat{
			models.ImageFormatJPEG,
			models.ImageFormatWebP,
		}

		variants, err = h.imageProcessor.ProcessImageSync(imageID, originalPath, sizes, formats)

		// Check if all critical variants were generated successfully
		criticalVariantsGenerated = h.hasCriticalVariants(variants)

		if err != nil {
			log.Printf("Warning: Image processing had errors: %v", err)
			// If critical variants failed, keep the original and report error
			if !criticalVariantsGenerated {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to generate critical image variants",
					"details": err.Error(),
				})
				return
			}
			// Non-critical errors - continue with what we have
		}
	}

	// ATOMIC DELETION: Delete original only if all critical variants succeeded
	if criticalVariantsGenerated {
		if err := os.Remove(originalPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to delete original file %s: %v", originalPath, err)
		} else {
			log.Printf("Deleted original file after successful variant generation: %s", originalPath)
			// Update image URL to point to the largest variant instead
			if largestVariant := h.imageProcessor.GetLargestVariant(variants, models.ImageFormatJPEG); largestVariant != nil {
				image.OriginalURL = largestVariant.URL
			}
		}
	} else {
		log.Printf("Keeping original file - critical variants not generated: %s", originalPath)
	}

	// Save image and variants to database
	if h.mediaService != nil {
		if err := h.mediaService.CreateImageWithVariants(image, variants); err != nil {
			log.Printf("Error: Failed to save image to database: %v", err)
		}
	}

	// Generate responsive HTML
	html := h.imageProcessor.GenerateResponsiveImageHTML(image, variants, true)

	response := UploadImageResponse{
		Image:    image,
		Variants: variants,
		HTML:     html,
	}

	c.JSON(http.StatusCreated, response)
}

// hasCriticalVariants checks if all critical size variants were generated
func (h *ImageHandlers) hasCriticalVariants(variants []models.ImageVariant) bool {
	criticalSizes := map[models.ImageSize]bool{
		models.ImageSizeThumbnail: false,
		models.ImageSizeSmall:     false,
	}

	for _, v := range variants {
		if _, isCritical := criticalSizes[v.Size]; isCritical {
			criticalSizes[v.Size] = true
		}
	}

	// Check all critical sizes have at least one variant
	for _, found := range criticalSizes {
		if !found {
			return false
		}
	}

	return true
}

// ProcessImageVariants handles processing variants for an existing image
func (h *ImageHandlers) ProcessImageVariants(c *gin.Context) {
	imageIDStr := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	// Get processing options from query parameters
	sizesParam := c.DefaultQuery("sizes", "thumbnail,small,medium,large")
	formatsParam := c.DefaultQuery("formats", "jpeg,webp")

	sizes := h.parseSizes(sizesParam)
	formats := h.parseFormats(formatsParam)

	if len(sizes) == 0 || len(formats) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid sizes or formats specified",
		})
		return
	}

	// For this example, we'll assume the original file path
	// In a real implementation, you'd look this up from the database
	originalPath := filepath.Join(h.uploadPath, "originals", fmt.Sprintf("%d_original.jpg", imageID))

	// Check if original file exists
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Original image file not found",
		})
		return
	}

	// Process variants
	variants, err := h.imageProcessor.ProcessImageSync(imageID, originalPath, sizes, formats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process image variants",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image_id": imageID,
		"variants": variants,
		"count":    len(variants),
	})
}

// GetImageHTML generates responsive HTML for an image
func (h *ImageHandlers) GetImageHTML(c *gin.Context) {
	imageIDStr := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image ID",
		})
		return
	}

	lazyLoad := c.DefaultQuery("lazy", "true") == "true"

	// In a real implementation, you'd fetch the image and variants from the database
	// For this example, we'll create mock data
	image := &models.Image{
		ID:      imageID,
		AltText: "Sample image",
		Width:   800,
		Height:  600,
	}

	// Mock variants - in reality, these would come from the database
	variants := []models.ImageVariant{
		{
			ImageID: imageID,
			Size:    models.ImageSizeSmall,
			Format:  models.ImageFormatJPEG,
			URL:     fmt.Sprintf("/images/2024/01/01/%d_small.jpeg", imageID),
			Width:   300,
			Height:  200,
		},
		{
			ImageID: imageID,
			Size:    models.ImageSizeMedium,
			Format:  models.ImageFormatJPEG,
			URL:     fmt.Sprintf("/images/2024/01/01/%d_medium.jpeg", imageID),
			Width:   600,
			Height:  400,
		},
	}

	html := h.imageProcessor.GenerateResponsiveImageHTML(image, variants, lazyLoad)

	c.JSON(http.StatusOK, gin.H{
		"image_id":  imageID,
		"html":      html,
		"lazy_load": lazyLoad,
	})
}

// GetProcessingStatus returns the status of the image processing queue
func (h *ImageHandlers) GetProcessingStatus(c *gin.Context) {
	status := h.imageProcessor.GetQueueStatus()
	c.JSON(http.StatusOK, status)
}

// Helper methods

func (h *ImageHandlers) isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/svg+xml",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (h *ImageHandlers) generateOriginalFilename(imageID uint64, originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".jpg" // Default extension
	}

	now := time.Now()
	return fmt.Sprintf("%s/%s/%s/%d_original%s",
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
		imageID,
		ext,
	)
}

func (h *ImageHandlers) saveUploadedFile(file multipart.File, dst string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Create destination file
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy file contents
	_, err = io.Copy(out, file)
	return err
}

func (h *ImageHandlers) parseSizes(sizesParam string) []models.ImageSize {
	var sizes []models.ImageSize

	sizeStrings := strings.Split(sizesParam, ",")
	for _, sizeStr := range sizeStrings {
		sizeStr = strings.TrimSpace(sizeStr)
		switch sizeStr {
		case "thumbnail":
			sizes = append(sizes, models.ImageSizeThumbnail)
		case "small":
			sizes = append(sizes, models.ImageSizeSmall)
		case "medium":
			sizes = append(sizes, models.ImageSizeMedium)
		case "large":
			sizes = append(sizes, models.ImageSizeLarge)
		case "original":
			sizes = append(sizes, models.ImageSizeOriginal)
		}
	}

	return sizes
}

func (h *ImageHandlers) parseFormats(formatsParam string) []models.ImageFormat {
	var formats []models.ImageFormat

	formatStrings := strings.Split(formatsParam, ",")
	for _, formatStr := range formatStrings {
		formatStr = strings.TrimSpace(formatStr)
		switch formatStr {
		case "jpeg", "jpg":
			formats = append(formats, models.ImageFormatJPEG)
		case "png":
			formats = append(formats, models.ImageFormatPNG)
		case "webp":
			formats = append(formats, models.ImageFormatWebP)
		// Note: AVIF is intentionally not included here.
		// AVIF encoding requires native libraries not available in pure Go.
		// Requesting AVIF would produce WebP files, which is confusing.
		// Use WebP instead - it has excellent browser support and compression.
		}
	}

	return formats
}
