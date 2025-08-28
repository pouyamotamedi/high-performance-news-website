package api

import (
	"fmt"
	"io"
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
	uploadPath     string
	maxFileSize    int64
}

// NewImageHandlers creates new image handlers
func NewImageHandlers(imageProcessor *services.ImageProcessor, uploadPath string, maxFileSize int64) *ImageHandlers {
	return &ImageHandlers{
		imageProcessor: imageProcessor,
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
	Image    *models.Image          `json:"image"`
	Variants []models.ImageVariant  `json:"variants"`
	HTML     string                 `json:"html"`
}

// UploadImage handles image upload and processing
func (h *ImageHandlers) UploadImage(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(h.maxFileSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse multipart form",
			"details": err.Error(),
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No image file provided",
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
			"error": "Invalid file type. Only JPEG, PNG, and WebP images are allowed",
		})
		return
	}

	// Parse request data
	var req UploadImageRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Generate unique filename
	imageID := uint64(time.Now().UnixNano())
	originalFilename := h.generateOriginalFilename(imageID, header.Filename)
	originalPath := filepath.Join(h.uploadPath, "originals", originalFilename)

	// Ensure upload directory exists
	if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create upload directory",
			"details": err.Error(),
		})
		return
	}

	// Save original file
	if err := h.saveUploadedFile(file, originalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save uploaded file",
			"details": err.Error(),
		})
		return
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Parse article ID if provided
	if req.ArticleID != "" {
		if articleID, err := strconv.ParseUint(req.ArticleID, 10, 64); err == nil {
			image.ArticleID = &articleID
		}
	}

	// Process image variants
	sizes := []models.ImageSize{
		models.ImageSizeThumbnail,
		models.ImageSizeSmall,
		models.ImageSizeMedium,
		models.ImageSizeLarge,
	}

	formats := []models.ImageFormat{
		models.ImageFormatJPEG,
		models.ImageFormatWebP,
		models.ImageFormatAVIF,
	}

	variants, err := h.imageProcessor.ProcessImageSync(imageID, originalPath, sizes, formats)
	if err != nil {
		// Log error but don't fail the upload - we have the original
		fmt.Printf("Warning: Failed to process image variants: %v\n", err)
		variants = []models.ImageVariant{} // Empty variants
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
	formatsParam := c.DefaultQuery("formats", "jpeg,webp,avif")

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
			"error": "Failed to process image variants",
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
		"image_id": imageID,
		"html":     html,
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
		case "avif":
			formats = append(formats, models.ImageFormatAVIF)
		}
	}
	
	return formats
}