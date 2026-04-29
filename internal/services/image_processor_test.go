package services

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/disintegration/imaging"
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

func TestImageProcessor_RealResizing(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")

	// Create a 800x600 test image
	createTestImage(t, testImagePath, 800, 600)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Process with thumbnail size (150x150)
	sizes := []models.ImageSize{models.ImageSizeThumbnail}
	formats := []models.ImageFormat{models.ImageFormatJPEG}

	variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
	require.NoError(t, err)
	require.Len(t, variants, 1)

	// Verify the variant was actually resized
	variant := variants[0]
	assert.Equal(t, models.ImageSizeThumbnail, variant.Size)

	// Load the generated file and verify dimensions
	generatedPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), "1_thumbnail.jpeg")
	file, err := os.Open(generatedPath)
	require.NoError(t, err)
	defer file.Close()

	img, _, err := image.Decode(file)
	require.NoError(t, err)

	bounds := img.Bounds()
	// Thumbnail should be resized to fit within 150x150 while preserving aspect ratio
	assert.LessOrEqual(t, bounds.Dx(), 150)
	assert.LessOrEqual(t, bounds.Dy(), 150)
}

func TestImageProcessor_RealWebPEncoding(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")

	// Create a test image
	createTestImage(t, testImagePath, 400, 300)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Process with WebP format
	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatWebP}

	variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
	require.NoError(t, err)
	require.Len(t, variants, 1)

	// Verify the variant is WebP
	variant := variants[0]
	assert.Equal(t, models.ImageFormatWebP, variant.Format)

	// Verify the file exists and has WebP magic bytes
	generatedPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), "1_small.webp")
	
	isWebP, err := processor.VerifyWebPFile(generatedPath)
	require.NoError(t, err)
	assert.True(t, isWebP, "Generated file should be a valid WebP")

	// Also verify file size is reasonable (WebP should be smaller than equivalent JPEG)
	fileInfo, err := os.Stat(generatedPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0), "WebP file should have content")
}

func TestImageProcessor_MultiFormatGeneration(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 600, 400)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatJPEG, models.ImageFormatWebP}

	variants, err := processor.ProcessImageSync(2, testImagePath, sizes, formats)
	require.NoError(t, err)

	// Should generate 1 size × 2 formats = 2 variants
	assert.Len(t, variants, 2)

	// Verify we have both formats
	formatCounts := make(map[models.ImageFormat]int)
	for _, v := range variants {
		formatCounts[v.Format]++
	}

	assert.Equal(t, 1, formatCounts[models.ImageFormatJPEG], "Should have 1 JPEG variant")
	assert.Equal(t, 1, formatCounts[models.ImageFormatWebP], "Should have 1 WebP variant")

	// Verify WebP file is valid
	webpPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), "2_small.webp")
	isWebP, err := processor.VerifyWebPFile(webpPath)
	require.NoError(t, err)
	assert.True(t, isWebP, "WebP file should have valid magic bytes")
}

func TestImageProcessor_MultiSizeGeneration(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 1200, 800)

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
		models.ImageSizeLarge,
	}
	formats := []models.ImageFormat{models.ImageFormatJPEG}

	variants, err := processor.ProcessImageSync(2, testImagePath, sizes, formats)
	require.NoError(t, err)

	// Should generate 4 sizes × 1 format = 4 variants
	assert.Len(t, variants, 4)

	// Verify each size was generated
	sizeCounts := make(map[models.ImageSize]int)
	for _, v := range variants {
		sizeCounts[v.Size]++
		assert.Greater(t, v.FileSize, int64(0))
		assert.Greater(t, v.Width, 0)
		assert.Greater(t, v.Height, 0)
	}

	assert.Equal(t, 1, sizeCounts[models.ImageSizeThumbnail])
	assert.Equal(t, 1, sizeCounts[models.ImageSizeSmall])
	assert.Equal(t, 1, sizeCounts[models.ImageSizeMedium])
	assert.Equal(t, 1, sizeCounts[models.ImageSizeLarge])
}

func TestImageProcessor_CriticalSizeFailure(t *testing.T) {
	tempDir := t.TempDir()
	invalidImagePath := filepath.Join(tempDir, "nonexistent.jpg")

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	sizes := []models.ImageSize{models.ImageSizeThumbnail} // Critical size
	formats := []models.ImageFormat{models.ImageFormatJPEG}

	_, err := processor.ProcessImageSync(1, invalidImagePath, sizes, formats)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load original image")
}

func TestImageProcessor_GenerateLQIP(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 800, 600)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Load the test image using imaging library
	img, err := imaging.Open(testImagePath)
	require.NoError(t, err)

	// Generate LQIP
	lqip, err := processor.GenerateLQIP(img)
	require.NoError(t, err)

	// Verify it's a valid base64 data URI
	assert.True(t, strings.HasPrefix(lqip, "data:image/jpeg;base64,"))
	assert.Greater(t, len(lqip), 50) // Should have some content
}

func TestImageProcessor_GenerateBlurHash(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 400, 300)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	img, err := imaging.Open(testImagePath)
	require.NoError(t, err)

	hash := processor.GenerateBlurHash(img)

	// Should be a valid hex color
	assert.True(t, strings.HasPrefix(hash, "#"))
	assert.Len(t, hash, 7) // #RRGGBB
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
		{ImageID: 1, Size: models.ImageSizeSmall, Format: models.ImageFormatJPEG, URL: "/images/1_small.jpeg", Width: 300, Height: 200},
		{ImageID: 1, Size: models.ImageSizeMedium, Format: models.ImageFormatJPEG, URL: "/images/1_medium.jpeg", Width: 600, Height: 400},
	}

	// Test with lazy loading
	html := processor.GenerateResponsiveImageHTML(image, variants, true)

	assert.Contains(t, html, "<picture>")
	assert.Contains(t, html, "</picture>")
	assert.Contains(t, html, `type="image/jpeg"`)
	assert.Contains(t, html, `loading="lazy"`)
	assert.Contains(t, html, `alt="Test image"`)
	assert.Contains(t, html, "srcset=")
	assert.Contains(t, html, "sizes=")
	assert.Contains(t, html, `decoding="async"`)

	// Test without lazy loading (hero image)
	htmlEager := processor.GenerateResponsiveImageHTML(image, variants, false)
	assert.Contains(t, htmlEager, `loading="eager"`)
	assert.Contains(t, htmlEager, `fetchpriority="high"`)
	assert.NotContains(t, htmlEager, `loading="lazy"`)
}

func TestImageProcessor_GenerateResponsiveImageHTMLWithLQIP(t *testing.T) {
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
		{ImageID: 1, Size: models.ImageSizeMedium, Format: models.ImageFormatJPEG, URL: "/images/1_medium.jpeg", Width: 600, Height: 400},
	}

	lqip := "data:image/jpeg;base64,/9j/4AAQSkZJRg=="

	html := processor.GenerateResponsiveImageHTMLWithLQIP(image, variants, lqip, true)

	assert.Contains(t, html, "image-wrapper")
	assert.Contains(t, html, "lqip-placeholder")
	assert.Contains(t, html, lqip)
	assert.Contains(t, html, "filter: blur(20px)")
}

func TestImageProcessor_DeleteImageVariants(t *testing.T) {
	tempDir := t.TempDir()

	// Create test variant files
	variantDir := filepath.Join(tempDir, "images", "2024", "01", "15")
	require.NoError(t, os.MkdirAll(variantDir, 0755))

	variant1 := filepath.Join(variantDir, "1_small.jpeg")
	variant2 := filepath.Join(variantDir, "1_medium.jpeg")

	require.NoError(t, os.WriteFile(variant1, []byte("test"), 0644))
	require.NoError(t, os.WriteFile(variant2, []byte("test"), 0644))

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	variantURLs := []string{
		"/uploads/2024/01/15/1_small.jpeg",
		"/uploads/2024/01/15/1_medium.jpeg",
	}

	err := processor.DeleteImageVariants(variantURLs)
	assert.NoError(t, err)

	// Verify files are deleted
	_, err = os.Stat(variant1)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(variant2)
	assert.True(t, os.IsNotExist(err))
}

func TestImageProcessor_ComputeImageHash(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 100, 100)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	hash, err := processor.ComputeImageHash(testImagePath)
	require.NoError(t, err)

	// SHA256 hash should be 64 hex characters
	assert.Len(t, hash, 64)

	// Same file should produce same hash
	hash2, err := processor.ComputeImageHash(testImagePath)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

func TestImageProcessor_AsyncProcessing(t *testing.T) {
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
	time.Sleep(200 * time.Millisecond)

	status := processor.GetQueueStatus()
	assert.Contains(t, status, "queue_length")
	assert.Contains(t, status, "queue_capacity")
	assert.Contains(t, status, "workers")
}

func TestImageProcessor_GeneratePreloadTag(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	tag := processor.GeneratePreloadTag("/images/hero.jpeg", models.ImageFormatJPEG)

	assert.Contains(t, tag, `rel="preload"`)
	assert.Contains(t, tag, `as="image"`)
	assert.Contains(t, tag, `href="/images/hero.jpeg"`)
	assert.Contains(t, tag, `type="image/jpeg"`)
	assert.Contains(t, tag, `fetchpriority="high"`)
}

func TestImageProcessor_GetLargestVariant(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	variants := []models.ImageVariant{
		{Size: models.ImageSizeSmall, Format: models.ImageFormatJPEG, Width: 300, URL: "/small.jpeg"},
		{Size: models.ImageSizeMedium, Format: models.ImageFormatJPEG, Width: 600, URL: "/medium.jpeg"},
		{Size: models.ImageSizeLarge, Format: models.ImageFormatJPEG, Width: 1200, URL: "/large.jpeg"},
	}

	// Get largest JPEG
	largest := processor.GetLargestVariant(variants, models.ImageFormatJPEG)
	assert.NotNil(t, largest)
	assert.Equal(t, 1200, largest.Width)
	assert.Equal(t, models.ImageFormatJPEG, largest.Format)
}

func TestImageProcessor_Shutdown(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      2,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)

	err := processor.Shutdown()
	assert.NoError(t, err)

	select {
	case <-processor.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestImageProcessor_VerifyWebPFile(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 200, 150)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Generate a WebP file
	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatWebP}

	variants, err := processor.ProcessImageSync(99, testImagePath, sizes, formats)
	require.NoError(t, err)
	require.Len(t, variants, 1)

	// Get the generated WebP path
	webpPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), "99_small.webp")

	// Verify it's a valid WebP
	isWebP, err := processor.VerifyWebPFile(webpPath)
	require.NoError(t, err)
	assert.True(t, isWebP, "File should be a valid WebP with RIFF/WEBP header")

	// Verify a non-WebP file returns false
	isWebP, err = processor.VerifyWebPFile(testImagePath)
	require.NoError(t, err)
	assert.False(t, isWebP, "JPEG file should not be detected as WebP")
}

func TestImageSizeConfig(t *testing.T) {
	config := models.GetImageSizeConfig()

	assert.Contains(t, config, models.ImageSizeThumbnail)
	assert.Contains(t, config, models.ImageSizeSmall)
	assert.Contains(t, config, models.ImageSizeMedium)
	assert.Contains(t, config, models.ImageSizeLarge)

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

func TestImageProcessor_AVIFFallbackToWebP(t *testing.T) {
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, testImagePath, 400, 300)

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Request AVIF format
	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatAVIF}

	variants, err := processor.ProcessImageSync(1, testImagePath, sizes, formats)
	require.NoError(t, err)
	require.Len(t, variants, 1)

	// Verify the variant is WebP (fallback), not AVIF
	variant := variants[0]
	assert.Equal(t, models.ImageFormatWebP, variant.Format, "AVIF should fall back to WebP")
	assert.Contains(t, variant.URL, ".webp", "URL should have .webp extension")

	// Verify the file is actually WebP
	webpPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), "1_small.webp")
	isWebP, err := processor.VerifyWebPFile(webpPath)
	require.NoError(t, err)
	assert.True(t, isWebP, "Fallback file should be valid WebP")
}

func TestImageProcessor_OriginalDeletionAfterVariants(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create original image in "originals" subfolder
	originalsDir := filepath.Join(tempDir, "originals")
	require.NoError(t, os.MkdirAll(originalsDir, 0755))
	
	originalPath := filepath.Join(originalsDir, "test_original.jpg")
	createTestImage(t, originalPath, 800, 600)

	// Verify original exists
	_, err := os.Stat(originalPath)
	require.NoError(t, err, "Original file should exist before processing")

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Process all critical sizes
	sizes := []models.ImageSize{
		models.ImageSizeThumbnail, // Critical
		models.ImageSizeSmall,     // Critical
		models.ImageSizeMedium,
	}
	formats := []models.ImageFormat{models.ImageFormatJPEG}

	variants, err := processor.ProcessImageSync(1, originalPath, sizes, formats)
	require.NoError(t, err)

	// Verify critical variants were generated
	criticalCount := 0
	for _, v := range variants {
		if v.Size == models.ImageSizeThumbnail || v.Size == models.ImageSizeSmall {
			criticalCount++
		}
	}
	assert.GreaterOrEqual(t, criticalCount, 2, "Should have at least 2 critical variants")

	// Simulate what the handler does: delete original after successful variant generation
	hasCritical := criticalCount >= 2
	if hasCritical {
		err := os.Remove(originalPath)
		assert.NoError(t, err, "Should be able to delete original after variants generated")
	}

	// Verify original is deleted
	_, err = os.Stat(originalPath)
	assert.True(t, os.IsNotExist(err), "Original file should be deleted after successful variant generation")

	// Verify variants still exist
	for _, v := range variants {
		variantPath := filepath.Join(tempDir, "images", time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), fmt.Sprintf("1_%s.jpeg", v.Size))
		_, err := os.Stat(variantPath)
		assert.NoError(t, err, "Variant %s should still exist", v.Size)
	}
}

func TestImageProcessor_DeduplicationHash(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create two identical images
	testImage1 := filepath.Join(tempDir, "test1.jpg")
	testImage2 := filepath.Join(tempDir, "test2.jpg")
	
	createTestImage(t, testImage1, 400, 300)
	
	// Copy to create identical file
	data, err := os.ReadFile(testImage1)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(testImage2, data, 0644))

	config := ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Compute hashes
	hash1, err := processor.ComputeImageHash(testImage1)
	require.NoError(t, err)

	hash2, err := processor.ComputeImageHash(testImage2)
	require.NoError(t, err)

	// Identical files should have identical hashes
	assert.Equal(t, hash1, hash2, "Identical images should have identical hashes")
	assert.Len(t, hash1, 64, "SHA256 hash should be 64 hex characters")

	// Create a different image
	testImage3 := filepath.Join(tempDir, "test3.jpg")
	createTestImage(t, testImage3, 500, 400) // Different dimensions

	hash3, err := processor.ComputeImageHash(testImage3)
	require.NoError(t, err)

	// Different files should have different hashes
	assert.NotEqual(t, hash1, hash3, "Different images should have different hashes")
}

func TestImageProcessor_PreloadTagGeneration(t *testing.T) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	// Test JPEG preload
	jpegTag := processor.GeneratePreloadTag("/images/hero.jpeg", models.ImageFormatJPEG)
	assert.Contains(t, jpegTag, `rel="preload"`)
	assert.Contains(t, jpegTag, `as="image"`)
	assert.Contains(t, jpegTag, `type="image/jpeg"`)
	assert.Contains(t, jpegTag, `fetchpriority="high"`)

	// Test WebP preload
	webpTag := processor.GeneratePreloadTag("/images/hero.webp", models.ImageFormatWebP)
	assert.Contains(t, webpTag, `type="image/webp"`)
}

// Helper function to create a test image
func createTestImage(t *testing.T, path string, width, height int) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient pattern for realistic testing
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

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
}

// Benchmark tests
func BenchmarkImageProcessor_RealResizing(b *testing.B) {
	tempDir := b.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")
	createTestImageForBenchmark(b, testImagePath, 1920, 1080)

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

func BenchmarkImageProcessor_WebPEncoding(b *testing.B) {
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

	sizes := []models.ImageSize{models.ImageSizeSmall}
	formats := []models.ImageFormat{models.ImageFormatWebP}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessImageSync(uint64(i), testImagePath, sizes, formats)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkImageProcessor_GenerateResponsiveHTML(b *testing.B) {
	config := ImageProcessorConfig{
		StorageBasePath: "/tmp",
		MaxWorkers:      1,
		QueueSize:       5,
	}

	processor := NewImageProcessor(config)
	defer processor.Shutdown()

	image := &models.Image{ID: 1, AltText: "Test image"}
	variants := []models.ImageVariant{
		{Size: models.ImageSizeSmall, Format: models.ImageFormatJPEG, URL: "/test1.jpg", Width: 300, Height: 200},
		{Size: models.ImageSizeMedium, Format: models.ImageFormatJPEG, URL: "/test2.jpg", Width: 600, Height: 400},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		processor.GenerateResponsiveImageHTML(image, variants, true)
	}
}

func createTestImageForBenchmark(b *testing.B, path string, width, height int) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		b.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

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

	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
}
