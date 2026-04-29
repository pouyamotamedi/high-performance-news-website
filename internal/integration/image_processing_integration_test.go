package integration

import (
	"context"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Setup test environment
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "original.jpg")
	createTestImage(t, testImagePath, 1200, 800)
	
	// Configure image processor
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      2,
		QueueSize:       10,
	}
	
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	t.Run("ProcessFullImagePipeline", func(t *testing.T) {
		// Test processing all sizes and formats
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
		
		variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
		
		assert.NoError(t, err)
		assert.Len(t, variants, 12) // 4 sizes × 3 formats
		
		// Verify all variants were created
		for _, variant := range variants {
			assert.NotEmpty(t, variant.URL)
			assert.Greater(t, variant.Width, 0)
			assert.Greater(t, variant.Height, 0)
			assert.Greater(t, variant.FileSize, int64(0))
			
			// Verify file exists (note: actual file creation depends on implementation)
			// In a real implementation, you would check if the file exists on disk
		}
	})
	
	t.Run("ErrorRecoveryForCriticalSizes", func(t *testing.T) {
		// Test with a corrupted image path to trigger errors
		corruptedPath := filepath.Join(tempDir, "corrupted.jpg")
		createCorruptedImage(t, corruptedPath)
		
		sizes := []models.ImageSize{
			models.ImageSizeThumbnail, // Critical
			models.ImageSizeSmall,     // Critical
			models.ImageSizeLarge,     // Non-critical
		}
		
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		// This should fail because critical sizes can't be processed
		_, err := processor.ProcessImageSync(2, corruptedPath, sizes, formats)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load original image")
	})
	
	t.Run("AsyncProcessingWithQueue", func(t *testing.T) {
		// Test async processing
		sizes := []models.ImageSize{models.ImageSizeSmall}
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		// Submit multiple jobs
		for i := 0; i < 5; i++ {
			err := processor.ProcessImage(uint64(100+i), testImagePath, sizes, formats)
			assert.NoError(t, err)
		}
		
		// Check queue status
		status := processor.GetQueueStatus()
		assert.Contains(t, status, "queue_length")
		assert.Contains(t, status, "workers")
		
		// Wait for processing to complete
		time.Sleep(500 * time.Millisecond)
	})
	
	t.Run("ResponsiveImageHTMLGeneration", func(t *testing.T) {
		image := &models.Image{
			ID:      1,
			AltText: "Integration test image",
			Caption: "Test caption",
			Width:   1200,
			Height:  800,
		}
		
		variants := []models.ImageVariant{
			{
				ImageID: 1,
				Size:    models.ImageSizeSmall,
				Format:  models.ImageFormatJPEG,
				URL:     "/images/2024/01/01/1_small.jpeg",
				Width:   300,
				Height:  200,
				Quality: 85,
			},
			{
				ImageID: 1,
				Size:    models.ImageSizeMedium,
				Format:  models.ImageFormatJPEG,
				URL:     "/images/2024/01/01/1_medium.jpeg",
				Width:   600,
				Height:  400,
				Quality: 90,
			},
			{
				ImageID: 1,
				Size:    models.ImageSizeSmall,
				Format:  models.ImageFormatWebP,
				URL:     "/images/2024/01/01/1_small.webp",
				Width:   300,
				Height:  200,
				Quality: 85,
			},
			{
				ImageID: 1,
				Size:    models.ImageSizeMedium,
				Format:  models.ImageFormatWebP,
				URL:     "/images/2024/01/01/1_medium.webp",
				Width:   600,
				Height:  400,
				Quality: 90,
			},
		}
		
		// Test with lazy loading
		htmlLazy := processor.GenerateResponsiveImageHTML(image, variants, true)
		
		assert.Contains(t, htmlLazy, "<picture>")
		assert.Contains(t, htmlLazy, "</picture>")
		assert.Contains(t, htmlLazy, `type="image/webp"`)
		assert.Contains(t, htmlLazy, `type="image/jpeg"`)
		assert.Contains(t, htmlLazy, `loading="lazy"`)
		assert.Contains(t, htmlLazy, `alt="Integration test image"`)
		assert.Contains(t, htmlLazy, "srcset=")
		
		// Test without lazy loading
		htmlNoLazy := processor.GenerateResponsiveImageHTML(image, variants, false)
		
		assert.Contains(t, htmlNoLazy, "<picture>")
		assert.NotContains(t, htmlNoLazy, `loading="lazy"`)
	})
	
	t.Run("HighVolumeProcessing", func(t *testing.T) {
		// Test processing multiple images concurrently
		const numImages = 10
		
		sizes := []models.ImageSize{models.ImageSizeSmall, models.ImageSizeMedium}
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		// Create multiple test images
		imagePaths := make([]string, numImages)
		for i := 0; i < numImages; i++ {
			imagePath := filepath.Join(tempDir, fmt.Sprintf("bulk_%d.jpg", i))
			createTestImage(t, imagePath, 600, 400)
			imagePaths[i] = imagePath
		}
		
		// Process all images
		start := time.Now()
		for i, imagePath := range imagePaths {
			err := processor.ProcessImage(uint64(200+i), imagePath, sizes, formats)
			assert.NoError(t, err)
		}
		
		// Wait for all processing to complete
		time.Sleep(2 * time.Second)
		
		duration := time.Since(start)
		t.Logf("Processed %d images in %v", numImages, duration)
		
		// Verify queue is empty or nearly empty
		status := processor.GetQueueStatus()
		queueLength := status["queue_length"].(int)
		assert.LessOrEqual(t, queueLength, 2, "Queue should be mostly empty after processing")
	})
}

func TestImageProcessingErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	tempDir := t.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	t.Run("NonExistentImageFile", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "does_not_exist.jpg")
		
		sizes := []models.ImageSize{models.ImageSizeSmall}
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		_, err := processor.ProcessImageSync(1, nonExistentPath, sizes, formats)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load original image")
	})
	
	t.Run("InvalidImageFormat", func(t *testing.T) {
		// Create a text file with .jpg extension
		invalidImagePath := filepath.Join(tempDir, "invalid.jpg")
		err := os.WriteFile(invalidImagePath, []byte("This is not an image"), 0644)
		require.NoError(t, err)
		
		sizes := []models.ImageSize{models.ImageSizeSmall}
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		_, err = processor.ProcessImageSync(1, invalidImagePath, sizes, formats)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load original image")
	})
	
	t.Run("QueueOverflow", func(t *testing.T) {
		// Fill up the queue
		testImagePath := filepath.Join(tempDir, "test.jpg")
		createTestImage(t, testImagePath, 400, 300)
		
		sizes := []models.ImageSize{models.ImageSizeSmall}
		formats := []models.ImageFormat{models.ImageFormatJPEG}
		
		// Fill the queue beyond capacity
		var lastErr error
		for i := 0; i < 10; i++ {
			err := processor.ProcessImage(uint64(i), testImagePath, sizes, formats)
			if err != nil {
				lastErr = err
				break
			}
		}
		
		// Should eventually get a queue full error
		if lastErr != nil {
			assert.Contains(t, lastErr.Error(), "queue is full")
		}
	})
}

func TestImageProcessingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "perf_test.jpg")
	createTestImage(t, testImagePath, 1920, 1080) // Large image
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      4,
		QueueSize:       20,
	}
	
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	t.Run("SingleImageProcessingTime", func(t *testing.T) {
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
		
		start := time.Now()
		variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Len(t, variants, 8) // 4 sizes × 2 formats
		
		t.Logf("Processed 1 image into 8 variants in %v", duration)
		
		// Performance assertion - should complete within reasonable time
		assert.Less(t, duration, 5*time.Second, "Image processing took too long")
	})
	
	t.Run("HTMLGenerationPerformance", func(t *testing.T) {
		image := &models.Image{
			ID:      1,
			AltText: "Performance test image",
		}
		
		// Create many variants to test HTML generation performance
		variants := make([]models.ImageVariant, 0, 12)
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
		
		for _, size := range sizes {
			for _, format := range formats {
				variants = append(variants, models.ImageVariant{
					ImageID: 1,
					Size:    size,
					Format:  format,
					URL:     fmt.Sprintf("/images/2024/01/01/1_%s.%s", size, format),
					Width:   600,
					Height:  400,
				})
			}
		}
		
		// Measure HTML generation time
		start := time.Now()
		for i := 0; i < 1000; i++ {
			html := processor.GenerateResponsiveImageHTML(image, variants, true)
			assert.NotEmpty(t, html)
		}
		duration := time.Since(start)
		
		t.Logf("Generated 1000 responsive image HTML snippets in %v", duration)
		
		// Should be very fast
		assert.Less(t, duration, 100*time.Millisecond, "HTML generation too slow")
	})
}

// Helper functions

func createTestImage(t *testing.T, path string, width, height int) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Create a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8(((x + y) * 255) / (width + height)),
				A: 255,
			})
		}
	}
	
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()
	
	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
}

func createCorruptedImage(t *testing.T, path string) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	
	// Create a file that looks like an image but isn't
	err = os.WriteFile(path, []byte("Not a real image file"), 0644)
	require.NoError(t, err)
}