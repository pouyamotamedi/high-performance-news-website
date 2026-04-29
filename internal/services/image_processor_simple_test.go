package services

import (
	"testing"

	"high-performance-news-website/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestImageProcessorBasicFunctionality tests basic image processor functionality
func TestImageProcessorBasicFunctionality(t *testing.T) {
	tempDir := t.TempDir()
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	// Test processor creation
	assert.NotNil(t, processor)
	assert.Equal(t, tempDir, processor.storageBasePath)
	assert.Equal(t, 1, processor.maxWorkers)
	
	// Test queue status
	status := processor.GetQueueStatus()
	assert.Contains(t, status, "queue_length")
	assert.Contains(t, status, "queue_capacity")
	assert.Contains(t, status, "workers")
	assert.Equal(t, 5, status["queue_capacity"])
	assert.Equal(t, 1, status["workers"])
}

// TestImageModels tests the image model functionality
func TestImageModels(t *testing.T) {
	// Test image size configuration
	config := models.GetImageSizeConfig()
	assert.Contains(t, config, models.ImageSizeThumbnail)
	assert.Contains(t, config, models.ImageSizeSmall)
	assert.Contains(t, config, models.ImageSizeMedium)
	assert.Contains(t, config, models.ImageSizeLarge)
	
	// Test thumbnail config
	thumbnail := config[models.ImageSizeThumbnail]
	assert.Equal(t, 150, thumbnail.Width)
	assert.Equal(t, 150, thumbnail.Height)
	assert.Equal(t, 85, thumbnail.Quality)
	
	// Test format priority
	priority := models.GetFormatPriority()
	expected := []models.ImageFormat{
		models.ImageFormatAVIF,
		models.ImageFormatWebP,
		models.ImageFormatJPEG,
	}
	assert.Equal(t, expected, priority)
	
	// Test critical size detection
	assert.True(t, models.ImageSizeThumbnail.IsCriticalSize())
	assert.True(t, models.ImageSizeSmall.IsCriticalSize())
	assert.False(t, models.ImageSizeMedium.IsCriticalSize())
	assert.False(t, models.ImageSizeLarge.IsCriticalSize())
	assert.False(t, models.ImageSizeOriginal.IsCriticalSize())
}

// TestResponsiveImageHTML tests HTML generation
func TestResponsiveImageHTML(t *testing.T) {
	tempDir := t.TempDir()
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	image := &models.Image{
		ID:      1,
		AltText: "Test image",
		Width:   800,
		Height:  600,
	}
	
	variants := []models.ImageVariant{
		{
			ImageID: 1,
			Size:    models.ImageSizeSmall,
			Format:  models.ImageFormatJPEG,
			URL:     "/images/2024/01/01/1_small.jpeg",
			Width:   300,
			Height:  200,
		},
		{
			ImageID: 1,
			Size:    models.ImageSizeMedium,
			Format:  models.ImageFormatWebP,
			URL:     "/images/2024/01/01/1_medium.webp",
			Width:   600,
			Height:  400,
		},
	}
	
	// Test with lazy loading
	htmlLazy := processor.GenerateResponsiveImageHTML(image, variants, true)
	assert.Contains(t, htmlLazy, "<picture>")
	assert.Contains(t, htmlLazy, "</picture>")
	assert.Contains(t, htmlLazy, `type="image/webp"`)
	assert.Contains(t, htmlLazy, `type="image/jpeg"`)
	assert.Contains(t, htmlLazy, `loading="lazy"`)
	assert.Contains(t, htmlLazy, `alt="Test image"`)
	
	// Test without lazy loading
	htmlNoLazy := processor.GenerateResponsiveImageHTML(image, variants, false)
	assert.Contains(t, htmlNoLazy, "<picture>")
	assert.NotContains(t, htmlNoLazy, `loading="lazy"`)
}

// TestImageProcessorHelpers tests helper methods
func TestImageProcessorHelpers(t *testing.T) {
	tempDir := t.TempDir()
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	// Test MIME type mapping
	tests := []struct {
		format   models.ImageFormat
		expected string
	}{
		{models.ImageFormatJPEG, "image/jpeg"},
		{models.ImageFormatPNG, "image/png"},
		{models.ImageFormatWebP, "image/webp"},
		{models.ImageFormatAVIF, "image/avif"},
	}
	
	for _, test := range tests {
		result := processor.getMimeType(test.format)
		assert.Equal(t, test.expected, result)
	}
	
	// Test path generation
	path := processor.generateVariantPath(123, models.ImageSizeSmall, models.ImageFormatJPEG)
	assert.Contains(t, path, tempDir)
	assert.Contains(t, path, "123_small.jpeg")
	
	// Test URL generation
	url := processor.generateVariantURL(456, models.ImageSizeLarge, models.ImageFormatWebP)
	assert.Contains(t, url, "/images/")
	assert.Contains(t, url, "456_large.webp")
}

// TestJobStatus tests job status constants
func TestJobStatus(t *testing.T) {
	assert.Equal(t, models.JobStatus("pending"), models.JobStatusPending)
	assert.Equal(t, models.JobStatus("processing"), models.JobStatusProcessing)
	assert.Equal(t, models.JobStatus("completed"), models.JobStatusCompleted)
	assert.Equal(t, models.JobStatus("failed"), models.JobStatusFailed)
}