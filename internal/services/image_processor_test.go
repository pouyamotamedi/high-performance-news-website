package services

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageProcessor(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp/test-images",
		MaxWorkers:      2,
		QueueSize:       10,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	assert.NotNil(t, processor)
	assert.Equal(t, "/tmp/test-images", processor.storageBasePath)
	assert.Equal(t, 2, processor.maxWorkers)
	assert.Equal(t, 10, cap(processor.jobQueue))
}

func TestImageProcessor_ProcessImageSync(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create test image
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 800, 600)
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	sizes := []models.ImageSize{
		models.ImageSizeThumbnail,
		models.ImageSizeSmall,
		models.ImageSizeMedium,
	}
	
	formats := []models.ImageFormat{
		models.ImageFormatJPEG,
		models.ImageFormatWebP,
	}
	
	variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
	
	assert.NoError(t, err)
	assert.Len(t, variants, 6) // 3 sizes × 2 formats
	
	// Verify variants have correct properties
	for _, variant := range variants {
		assert.Equal(t, uint64(1), variant.ImageID)
		assert.Contains(t, []models.ImageSize{
			models.ImageSizeThumbnail,
			models.ImageSizeSmall,
			models.ImageSizeMedium,
		}, variant.Size)
		assert.Contains(t, []models.ImageFormat{
			models.ImageFormatJPEG,
			models.ImageFormatWebP,
		}, variant.Format)
		assert.NotEmpty(t, variant.URL)
		assert.Greater(t, variant.Width, 0)
		assert.Greater(t, variant.Height, 0)
		assert.Greater(t, variant.FileSize, int64(0))
	}
}

func TestImageProcessor_ProcessImageSync_CriticalSizeFailure(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create invalid image path to trigger error
	invalidImagePath := filepath.Join(tempDir, "nonexistent.jpg")
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	sizes := []models.ImageSize{
		models.ImageSizeThumbnail, // Critical size
		models.ImageSizeLarge,     // Non-critical size
	}
	
	formats := []models.ImageFormat{models.ImageFormatJPEG}
	
	_, err := processor.ProcessImageSync(1, invalidImagePath, sizes, formats)
	
	// Should fail because critical size failed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load original image")
}

func TestImageProcessor_ProcessImage_AsyncProcessing(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 400, 300)
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      2,
		QueueSize:       10,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatJPEG}
	
	err := processor.ProcessImage(1, testImagePath, sizes, formats)
	assert.NoError(t, err)
	
	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)
	
	status := processor.GetQueueStatus()
	assert.Contains(t, status, "queue_length")
	assert.Contains(t, status, "queue_capacity")
	assert.Contains(t, status, "workers")
}

func TestImageProcessor_GenerateResponsiveImageHTML(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
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
			Format:  models.ImageFormatJPEG,
			URL:     "/images/2024/01/01/1_medium.jpeg",
			Width:   600,
			Height:  400,
		},
		{
			ImageID: 1,
			Size:    models.ImageSizeSmall,
			Format:  models.ImageFormatWebP,
			URL:     "/images/2024/01/01/1_small.webp",
			Width:   300,
			Height:  200,
		},
	}
	
	html := processor.GenerateResponsiveImageHTML(image, variants, true)
	
	assert.Contains(t, html, "<picture>")
	assert.Contains(t, html, "</picture>")
	assert.Contains(t, html, `type="image/webp"`)
	assert.Contains(t, html, `type="image/jpeg"`)
	assert.Contains(t, html, `loading="lazy"`)
	assert.Contains(t, html, `alt="Test image"`)
	assert.Contains(t, html, "srcset=")
}

func TestImageProcessor_GenerateResponsiveImageHTML_NoLazyLoad(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	image := &models.Image{
		ID:      1,
		AltText: "Test image",
	}
	
	variants := []models.ImageVariant{
		{
			ImageID: 1,
			Size:    models.ImageSizeMedium,
			Format:  models.ImageFormatJPEG,
			URL:     "/images/2024/01/01/1_medium.jpeg",
			Width:   600,
			Height:  400,
		},
	}
	
	html := processor.GenerateResponsiveImageHTML(image, variants, false)
	
	assert.Contains(t, html, "<picture>")
	assert.NotContains(t, html, `loading="lazy"`)
}

func TestImageProcessor_GenerateVariantPath(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/var/www/static",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	path := processor.generateVariantPath(123, models.ImageSizeSmall, models.ImageFormatJPEG)
	
	assert.Contains(t, path, "/var/www/static/images/")
	assert.Contains(t, path, "123_small.jpeg")
	
	// Should contain year/month/day structure
	now := time.Now()
	assert.Contains(t, path, now.Format("2006"))
	assert.Contains(t, path, now.Format("01"))
	assert.Contains(t, path, now.Format("02"))
}

func TestImageProcessor_GenerateVariantURL(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	url := processor.generateVariantURL(456, models.ImageSizeLarge, models.ImageFormatWebP)
	
	assert.Contains(t, url, "/images/")
	assert.Contains(t, url, "456_large.webp")
	
	// Should contain year/month/day structure
	now := time.Now()
	assert.Contains(t, url, now.Format("2006"))
	assert.Contains(t, url, now.Format("01"))
	assert.Contains(t, url, now.Format("02"))
}

func TestImageProcessor_GetMimeType(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
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
}

func TestImageProcessor_GetFallbackVariant(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	variants := []models.ImageVariant{
		{
			Size:   models.ImageSizeSmall,
			Format: models.ImageFormatWebP,
			URL:    "/test1.webp",
		},
		{
			Size:   models.ImageSizeMedium,
			Format: models.ImageFormatJPEG,
			URL:    "/test2.jpeg",
		},
		{
			Size:   models.ImageSizeLarge,
			Format: models.ImageFormatJPEG,
			URL:    "/test3.jpeg",
		},
	}
	
	fallback := processor.getFallbackVariant(variants)
	
	// Should prefer JPEG medium size
	assert.Equal(t, models.ImageFormatJPEG, fallback.Format)
	assert.Equal(t, models.ImageSizeMedium, fallback.Size)
	assert.Equal(t, "/test2.jpeg", fallback.URL)
}

func TestImageProcessor_GetFallbackVariant_NoMediumJPEG(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	variants := []models.ImageVariant{
		{
			Size:   models.ImageSizeSmall,
			Format: models.ImageFormatWebP,
			URL:    "/test1.webp",
		},
		{
			Size:   models.ImageSizeLarge,
			Format: models.ImageFormatJPEG,
			URL:    "/test2.jpeg",
		},
	}
	
	fallback := processor.getFallbackVariant(variants)
	
	// Should fallback to any JPEG
	assert.Equal(t, models.ImageFormatJPEG, fallback.Format)
	assert.Equal(t, "/test2.jpeg", fallback.URL)
}

func TestImageProcessor_Shutdown(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      2,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	
	// Test graceful shutdown
	err := processor.Shutdown()
	assert.NoError(t, err)
	
	// Context should be cancelled
	select {
	case <-processor.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestImageSizeConfig(t *testing.T) {
	config := models.GetImageSizeConfig()
	
	assert.Contains(t, config, models.ImageSizeThumbnail)
	assert.Contains(t, config, models.ImageSizeSmall)
	assert.Contains(t, config, models.ImageSizeMedium)
	assert.Contains(t, config, models.ImageSizeLarge)
	
	// Verify thumbnail config
	thumbnail := config[models.ImageSizeThumbnail]
	assert.Equal(t, 150, thumbnail.Width)
	assert.Equal(t, 150, thumbnail.Height)
	assert.Equal(t, 85, thumbnail.Quality)
}

func TestImageSize_IsCriticalSize(t *testing.T) {
	tests := []struct {
		size     models.ImageSize
		critical bool
	}{
		{models.ImageSizeThumbnail, true},
		{models.ImageSizeSmall, true},
		{models.ImageSizeMedium, false},
		{models.ImageSizeLarge, false},
		{models.ImageSizeOriginal, false},
	}
	
	for _, test := range tests {
		result := test.size.IsCriticalSize()
		assert.Equal(t, test.critical, result, "Size %s critical check failed", test.size)
	}
}

func TestGetFormatPriority(t *testing.T) {
	priority := models.GetFormatPriority()
	
	expected := []models.ImageFormat{
		models.ImageFormatAVIF,
		models.ImageFormatWebP,
		models.ImageFormatJPEG,
	}
	
	assert.Equal(t, expected, priority)
}

// Helper function to create a test image
func createTestImage(t *testing.T, path string, width, height int) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}
	
	// Save as JPEG
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()
	
	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
}

// Helper function to create a test image for benchmarks
func createTestImageForBenchmark(b *testing.B, path string, width, height int) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		b.Fatal(err)
	}
	
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}
	
	// Save as JPEG
	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()
	
	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	if err != nil {
		b.Fatal(err)
	}
}

// Benchmark tests
func BenchmarkImageProcessor_ProcessImageSync(b *testing.B) {
	tempDir := b.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImageForBenchmark(b, testImagePath, 800, 600)
	
	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	sizes := []models.ImageSize{models.ImageSizeSmall, models.ImageSizeMedium}
	formats := []models.ImageFormat{models.ImageFormatJPEG}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessImageSync(uint64(i), testImagePath, sizes, formats)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkImageProcessor_GenerateResponsiveImageHTML(b *testing.B) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}
	
	processor := NewImageProcessor(config)
	defer processor.Shutdown()
	
	image := &models.Image{
		ID:      1,
		AltText: "Test image",
	}
	
	variants := []models.ImageVariant{
		{Size: models.ImageSizeSmall, Format: models.ImageFormatJPEG, URL: "/test1.jpg", Width: 300, Height: 200},
		{Size: models.ImageSizeMedium, Format: models.ImageFormatJPEG, URL: "/test2.jpg", Width: 600, Height: 400},
		{Size: models.ImageSizeSmall, Format: models.ImageFormatWebP, URL: "/test1.webp", Width: 300, Height: 200},
		{Size: models.ImageSizeMedium, Format: models.ImageFormatWebP, URL: "/test2.webp", Width: 600, Height: 400},
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		processor.GenerateResponsiveImageHTML(image, variants, true)
	}
}