package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ImageProcessor handles image processing operations with real encoding
type ImageProcessor struct {
	storageBasePath string
	maxWorkers      int
	jobQueue        chan *models.ImageProcessingJob
	workers         sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
}

// ImageProcessorConfig contains configuration for the image processor
type ImageProcessorConfig struct {
	StorageBasePath string
	MaxWorkers      int
	QueueSize       int
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(config ImageProcessorConfig) *ImageProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &ImageProcessor{
		storageBasePath: config.StorageBasePath,
		maxWorkers:      config.MaxWorkers,
		jobQueue:        make(chan *models.ImageProcessingJob, config.QueueSize),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start worker goroutines
	for i := 0; i < config.MaxWorkers; i++ {
		processor.workers.Add(1)
		go processor.worker(i)
	}

	return processor
}

// ProcessImage processes an image with multiple formats and sizes asynchronously
func (ip *ImageProcessor) ProcessImage(imageID uint64, originalPath string, sizes []models.ImageSize, formats []models.ImageFormat) error {
	job := &models.ImageProcessingJob{
		ID:        uint64(time.Now().UnixNano()),
		ImageID:   imageID,
		Sizes:     sizes,
		Formats:   formats,
		Priority:  models.JobPriorityMedium,
		Status:    models.JobStatusPending,
		CreatedAt: time.Now(),
	}

	job.Metadata = map[string]interface{}{
		"original_path": originalPath,
	}

	select {
	case ip.jobQueue <- job:
		return nil
	case <-ip.ctx.Done():
		return fmt.Errorf("image processor is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// ProcessImageSync processes an image synchronously with real resizing and encoding
func (ip *ImageProcessor) ProcessImageSync(imageID uint64, originalPath string, sizes []models.ImageSize, formats []models.ImageFormat) ([]models.ImageVariant, error) {
	originalImg, err := imaging.Open(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load original image: %w", err)
	}

	var variants []models.ImageVariant
	var criticalErrors []error
	var nonCriticalErrors []error

	sizeConfigs := models.GetImageSizeConfig()

	for _, size := range sizes {
		config, exists := sizeConfigs[size]
		if !exists && size != models.ImageSizeOriginal {
			continue
		}

		if size == models.ImageSizeOriginal {
			bounds := originalImg.Bounds()
			config = models.ImageSizeConfig{
				Width:   bounds.Dx(),
				Height:  bounds.Dy(),
				Quality: 95,
			}
		}

		// Real image resizing using imaging library
		resizedImg := ip.resizeImage(originalImg, config.Width, config.Height)

		for _, targetFormat := range formats {
			variant, err := ip.processImageVariant(imageID, resizedImg, size, targetFormat, config.Quality)
			if err != nil {
				if size.IsCriticalSize() {
					criticalErrors = append(criticalErrors, fmt.Errorf("failed to process %s/%s: %w", size, targetFormat, err))
				} else {
					nonCriticalErrors = append(nonCriticalErrors, fmt.Errorf("failed to process %s/%s: %w", size, targetFormat, err))
				}
				continue
			}
			variants = append(variants, *variant)
		}
	}

	if len(nonCriticalErrors) > 0 {
		log.Printf("Non-critical image processing errors: %v", nonCriticalErrors)
	}

	if len(criticalErrors) > 0 {
		return variants, fmt.Errorf("critical image processing errors: %v", criticalErrors)
	}

	return variants, nil
}

// worker processes jobs from the queue
func (ip *ImageProcessor) worker(workerID int) {
	defer ip.workers.Done()

	for {
		select {
		case job := <-ip.jobQueue:
			ip.processJob(workerID, job)
		case <-ip.ctx.Done():
			log.Printf("Image processor worker %d shutting down", workerID)
			return
		}
	}
}

// processJob processes a single image processing job
func (ip *ImageProcessor) processJob(workerID int, job *models.ImageProcessingJob) {
	log.Printf("Worker %d processing job %d for image %d", workerID, job.ID, job.ImageID)

	job.Status = models.JobStatusProcessing
	job.UpdatedAt = time.Now()

	originalPath, ok := job.Metadata["original_path"].(string)
	if !ok {
		job.Status = models.JobStatusFailed
		job.Error = "missing original_path in metadata"
		return
	}

	variants, err := ip.ProcessImageSync(job.ImageID, originalPath, job.Sizes, job.Formats)
	if err != nil {
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		log.Printf("Worker %d failed to process job %d: %v", workerID, job.ID, err)
		return
	}

	job.Status = models.JobStatusCompleted
	job.UpdatedAt = time.Now()
	job.Metadata["variants_count"] = len(variants)

	log.Printf("Worker %d completed job %d, generated %d variants", workerID, job.ID, len(variants))
}

// resizeImage resizes an image using the imaging library (REAL implementation)
func (ip *ImageProcessor) resizeImage(img image.Image, width, height int) image.Image {
	// Use imaging.Fit to resize while preserving aspect ratio
	// Lanczos is high-quality resampling filter
	return imaging.Fit(img, width, height, imaging.Lanczos)
}

// convertToRGBA converts any image to RGBA format for consistent encoding
func (ip *ImageProcessor) convertToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// processImageVariant processes a single image variant with real encoding
func (ip *ImageProcessor) processImageVariant(imageID uint64, img image.Image, size models.ImageSize, format models.ImageFormat, quality int) (*models.ImageVariant, error) {
	outputPath := ip.generateVariantPath(imageID, size, format)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var buf bytes.Buffer
	bounds := img.Bounds()

	switch format {
	case models.ImageFormatJPEG:
		// High-quality JPEG encoding
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case models.ImageFormatPNG:
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	case models.ImageFormatWebP:
		// REAL WebP encoding using chai2010/webp with CGO
		rgba := ip.convertToRGBA(img)
		err := webp.Encode(&buf, rgba, &webp.Options{
			Lossless: false,
			Quality:  float32(quality),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode WebP: %w", err)
		}
	case models.ImageFormatAVIF:
		// AVIF ENCODING LIMITATION:
		// AVIF encoding requires native libavif with CGO bindings.
		// No stable pure-Go or CGO AVIF encoder exists for Go as of 2024.
		// 
		// FALLBACK BEHAVIOR:
		// When AVIF is requested, we produce a WebP file instead.
		// WebP offers similar compression benefits and has broader browser support.
		// The output file will have .webp extension to avoid client confusion.
		//
		// RECOMMENDATION:
		// Do not request AVIF format in production. Use WebP instead.
		// If native AVIF is required, consider using an external image service
		// (e.g., Cloudinary, imgix) or a command-line tool (e.g., avifenc).
		log.Printf("AVIF requested but not supported - falling back to WebP for image %d", imageID)
		rgba := ip.convertToRGBA(img)
		err := webp.Encode(&buf, rgba, &webp.Options{
			Lossless: false,
			Quality:  float32(quality),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode AVIF fallback (WebP): %w", err)
		}
		// Update format and path to reflect actual output
		format = models.ImageFormatWebP
		outputPath = ip.generateVariantPath(imageID, size, format)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Write to file
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write image file: %w", err)
	}

	url := ip.generateVariantURL(imageID, size, format)

	variant := &models.ImageVariant{
		ImageID:  imageID,
		Size:     size,
		Format:   format,
		URL:      url,
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		FileSize: int64(buf.Len()),
		Quality:  quality,
	}

	return variant, nil
}

// generateVariantPath generates the file system path for an image variant
func (ip *ImageProcessor) generateVariantPath(imageID uint64, size models.ImageSize, format models.ImageFormat) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	filename := fmt.Sprintf("%d_%s.%s", imageID, size, format)
	// Path matches URL: /uploads/YYYY/MM/DD/filename
	return filepath.Join(ip.storageBasePath, year, month, day, filename)
}

// generateVariantURL generates the web URL for an image variant
func (ip *ImageProcessor) generateVariantURL(imageID uint64, size models.ImageSize, format models.ImageFormat) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	filename := fmt.Sprintf("%d_%s.%s", imageID, size, format)
	return fmt.Sprintf("/uploads/%s/%s/%s/%s", year, month, day, filename)
}

// GenerateLQIP generates a Low Quality Image Placeholder (base64 encoded tiny image)
func (ip *ImageProcessor) GenerateLQIP(img image.Image) (string, error) {
	// Create a tiny 20x20 version for LQIP
	tiny := imaging.Resize(img, 20, 0, imaging.Box)

	// Apply blur for smooth placeholder effect
	blurred := imaging.Blur(tiny, 2.0)

	// Encode as low-quality JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, blurred, &jpeg.Options{Quality: 20}); err != nil {
		return "", fmt.Errorf("failed to encode LQIP: %w", err)
	}

	// Return as base64 data URI
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/jpeg;base64,%s", base64Str), nil
}

// GenerateBlurHash generates a blur placeholder hash (simplified implementation)
func (ip *ImageProcessor) GenerateBlurHash(img image.Image) string {
	// Create tiny version and get average color for simple blur effect
	tiny := imaging.Resize(img, 4, 4, imaging.Box)
	bounds := tiny.Bounds()

	var r, g, b uint32
	var count uint32

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pr, pg, pb, _ := tiny.At(x, y).RGBA()
			r += pr >> 8
			g += pg >> 8
			b += pb >> 8
			count++
		}
	}

	if count > 0 {
		r /= count
		g /= count
		b /= count
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// GenerateResponsiveImageHTML generates HTML for responsive images
func (ip *ImageProcessor) GenerateResponsiveImageHTML(image *models.Image, variants []models.ImageVariant, lazyLoad bool) string {
	if len(variants) == 0 {
		return ""
	}

	formatGroups := make(map[models.ImageFormat][]models.ImageVariant)
	for _, variant := range variants {
		formatGroups[variant.Format] = append(formatGroups[variant.Format], variant)
	}

	var html strings.Builder
	html.WriteString("<picture>")

	formatPriority := models.GetFormatPriority()
	for _, format := range formatPriority {
		variants, exists := formatGroups[format]
		if !exists {
			continue
		}

		var srcset []string
		for _, variant := range variants {
			srcset = append(srcset, fmt.Sprintf("%s %dw", variant.URL, variant.Width))
		}

		if len(srcset) > 0 {
			mimeType := ip.getMimeType(format)
			html.WriteString(fmt.Sprintf(`<source type="%s" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`,
				mimeType, strings.Join(srcset, ", ")))
		}
	}

	fallbackVariant := ip.getFallbackVariant(variants)
	imgAttrs := []string{
		fmt.Sprintf(`src="%s"`, fallbackVariant.URL),
		fmt.Sprintf(`alt="%s"`, image.AltText),
		fmt.Sprintf(`width="%d"`, fallbackVariant.Width),
		fmt.Sprintf(`height="%d"`, fallbackVariant.Height),
		`sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw"`,
		`decoding="async"`,
	}

	if lazyLoad {
		imgAttrs = append(imgAttrs, `loading="lazy"`)
	} else {
		// For above-the-fold images
		imgAttrs = append(imgAttrs, `loading="eager"`, `fetchpriority="high"`)
	}

	html.WriteString(fmt.Sprintf("<img %s>", strings.Join(imgAttrs, " ")))
	html.WriteString("</picture>")

	return html.String()
}

// GenerateResponsiveImageHTMLWithLQIP generates HTML with LQIP placeholder
func (ip *ImageProcessor) GenerateResponsiveImageHTMLWithLQIP(image *models.Image, variants []models.ImageVariant, lqip string, lazyLoad bool) string {
	if len(variants) == 0 {
		return ""
	}

	formatGroups := make(map[models.ImageFormat][]models.ImageVariant)
	for _, variant := range variants {
		formatGroups[variant.Format] = append(formatGroups[variant.Format], variant)
	}

	var html strings.Builder

	// Wrapper div for LQIP effect
	html.WriteString(`<div class="image-wrapper" style="position: relative; overflow: hidden;">`)

	// LQIP background
	if lqip != "" && lazyLoad {
		html.WriteString(fmt.Sprintf(`<div class="lqip-placeholder" style="position: absolute; inset: 0; background-image: url('%s'); background-size: cover; filter: blur(20px); transform: scale(1.1);"></div>`, lqip))
	}

	html.WriteString("<picture>")

	formatPriority := models.GetFormatPriority()
	for _, format := range formatPriority {
		variants, exists := formatGroups[format]
		if !exists {
			continue
		}

		var srcset []string
		for _, variant := range variants {
			srcset = append(srcset, fmt.Sprintf("%s %dw", variant.URL, variant.Width))
		}

		if len(srcset) > 0 {
			mimeType := ip.getMimeType(format)
			html.WriteString(fmt.Sprintf(`<source type="%s" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`,
				mimeType, strings.Join(srcset, ", ")))
		}
	}

	fallbackVariant := ip.getFallbackVariant(variants)
	imgAttrs := []string{
		fmt.Sprintf(`src="%s"`, fallbackVariant.URL),
		fmt.Sprintf(`alt="%s"`, image.AltText),
		fmt.Sprintf(`width="%d"`, fallbackVariant.Width),
		fmt.Sprintf(`height="%d"`, fallbackVariant.Height),
		`sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw"`,
		`decoding="async"`,
		`style="position: relative; z-index: 1;"`,
		`onload="this.parentElement.parentElement.querySelector('.lqip-placeholder')?.remove()"`,
	}

	if lazyLoad {
		imgAttrs = append(imgAttrs, `loading="lazy"`)
	} else {
		imgAttrs = append(imgAttrs, `loading="eager"`, `fetchpriority="high"`)
	}

	html.WriteString(fmt.Sprintf("<img %s>", strings.Join(imgAttrs, " ")))
	html.WriteString("</picture>")
	html.WriteString("</div>")

	return html.String()
}

// getMimeType returns the MIME type for an image format
func (ip *ImageProcessor) getMimeType(format models.ImageFormat) string {
	switch format {
	case models.ImageFormatJPEG:
		return "image/jpeg"
	case models.ImageFormatPNG:
		return "image/png"
	case models.ImageFormatWebP:
		return "image/webp"
	case models.ImageFormatAVIF:
		return "image/avif"
	default:
		return "image/jpeg"
	}
}

// getFallbackVariant returns the best fallback variant (JPEG, medium size preferred)
func (ip *ImageProcessor) getFallbackVariant(variants []models.ImageVariant) models.ImageVariant {
	for _, variant := range variants {
		if variant.Format == models.ImageFormatJPEG && variant.Size == models.ImageSizeMedium {
			return variant
		}
	}

	for _, variant := range variants {
		if variant.Format == models.ImageFormatJPEG {
			return variant
		}
	}

	if len(variants) > 0 {
		return variants[0]
	}

	return models.ImageVariant{}
}

// ComputeImageHash computes SHA256 hash of image for deduplication
func (ip *ImageProcessor) ComputeImageHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// DeleteImageVariants deletes all variant files for an image
func (ip *ImageProcessor) DeleteImageVariants(variantURLs []string) error {
	var errors []error

	for _, url := range variantURLs {
		// Convert URL to file path
		// URL format: /uploads/YYYY/MM/DD/filename
		if strings.HasPrefix(url, "/uploads/") {
			relativePath := strings.TrimPrefix(url, "/uploads/")
			fullPath := filepath.Join(ip.storageBasePath, relativePath)

			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				errors = append(errors, fmt.Errorf("failed to delete %s: %w", fullPath, err))
			} else {
				log.Printf("Deleted image variant: %s", fullPath)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors deleting variants: %v", errors)
	}

	return nil
}

// DeleteOriginalImage deletes the original image file
func (ip *ImageProcessor) DeleteOriginalImage(originalURL string) error {
	if strings.HasPrefix(originalURL, "/uploads/") {
		relativePath := strings.TrimPrefix(originalURL, "/uploads/")
		fullPath := filepath.Join(ip.storageBasePath, relativePath)

		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete original image %s: %w", fullPath, err)
		}
		log.Printf("Deleted original image: %s", fullPath)
	}

	return nil
}

// Shutdown gracefully shuts down the image processor
func (ip *ImageProcessor) Shutdown() error {
	log.Println("Shutting down image processor...")
	ip.cancel()
	ip.workers.Wait()
	close(ip.jobQueue)
	log.Println("Image processor shutdown complete")
	return nil
}

// GetQueueStatus returns the current status of the job queue
func (ip *ImageProcessor) GetQueueStatus() map[string]interface{} {
	return map[string]interface{}{
		"queue_length":   len(ip.jobQueue),
		"queue_capacity": cap(ip.jobQueue),
		"workers":        ip.maxWorkers,
	}
}

// GeneratePreloadTag generates a preload link tag for LCP images
func (ip *ImageProcessor) GeneratePreloadTag(imageURL string, format models.ImageFormat) string {
	mimeType := ip.getMimeType(format)
	return fmt.Sprintf(`<link rel="preload" as="image" href="%s" type="%s" fetchpriority="high">`, imageURL, mimeType)
}

// GetLargestVariant returns the largest variant for preloading
func (ip *ImageProcessor) GetLargestVariant(variants []models.ImageVariant, preferredFormat models.ImageFormat) *models.ImageVariant {
	var largest *models.ImageVariant

	for i := range variants {
		v := &variants[i]
		if v.Format != preferredFormat {
			continue
		}
		if largest == nil || v.Width > largest.Width {
			largest = v
		}
	}

	// Fallback to any format if preferred not found
	if largest == nil {
		for i := range variants {
			v := &variants[i]
			if largest == nil || v.Width > largest.Width {
				largest = v
			}
		}
	}

	return largest
}

// CreatePlaceholderColor creates a solid color placeholder image
func (ip *ImageProcessor) CreatePlaceholderColor(width, height int, hexColor string) image.Image {
	// Parse hex color
	var r, g, b uint8
	fmt.Sscanf(hexColor, "#%02x%02x%02x", &r, &g, &b)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := color.RGBA{r, g, b, 255}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}

	return img
}

// VerifyWebPFile checks if a file is a valid WebP by checking magic bytes
func (ip *ImageProcessor) VerifyWebPFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// WebP files start with "RIFF" followed by file size, then "WEBP"
	header := make([]byte, 12)
	_, err = file.Read(header)
	if err != nil {
		return false, err
	}

	// Check RIFF header and WEBP signature
	isWebP := string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP"
	return isWebP, nil
}
