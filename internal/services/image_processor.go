package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
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
)

// ImageProcessor handles image processing operations
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

// ProcessImage processes an image with multiple formats and sizes
func (ip *ImageProcessor) ProcessImage(imageID uint64, originalPath string, sizes []models.ImageSize, formats []models.ImageFormat) error {
	job := &models.ImageProcessingJob{
		ID:        uint64(time.Now().UnixNano()), // Simple ID generation
		ImageID:   imageID,
		Sizes:     sizes,
		Formats:   formats,
		Priority:  models.JobPriorityMedium,
		Status:    models.JobStatusPending,
		CreatedAt: time.Now(),
	}
	
	// Add metadata
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

// ProcessImageSync processes an image synchronously
func (ip *ImageProcessor) ProcessImageSync(imageID uint64, originalPath string, sizes []models.ImageSize, formats []models.ImageFormat) ([]models.ImageVariant, error) {
	// Load original image
	originalImg, _, err := ip.loadImage(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load original image: %w", err)
	}
	
	var variants []models.ImageVariant
	var criticalErrors []error
	var nonCriticalErrors []error
	
	sizeConfigs := models.GetImageSizeConfig()
	
	// Process each size and format combination
	for _, size := range sizes {
		config, exists := sizeConfigs[size]
		if !exists && size != models.ImageSizeOriginal {
			continue
		}
		
		// For original size, use original dimensions
		if size == models.ImageSizeOriginal {
			bounds := originalImg.Bounds()
			config = models.ImageSizeConfig{
				Width:   bounds.Dx(),
				Height:  bounds.Dy(),
				Quality: 95,
			}
		}
		
		// Resize image
		resizedImg, err := ip.resizeImage(originalImg, config.Width, config.Height)
		if err != nil {
			if size.IsCriticalSize() {
				criticalErrors = append(criticalErrors, fmt.Errorf("failed to resize to %s: %w", size, err))
			} else {
				nonCriticalErrors = append(nonCriticalErrors, fmt.Errorf("failed to resize to %s: %w", size, err))
			}
			continue
		}
		
		// Process each format
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
	
	// Log non-critical errors but don't fail the operation
	if len(nonCriticalErrors) > 0 {
		log.Printf("Non-critical image processing errors: %v", nonCriticalErrors)
	}
	
	// Fail only if critical sizes failed
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

// loadImage loads an image from file
func (ip *ImageProcessor) loadImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", err
	}
	
	return img, format, nil
}

// resizeImage resizes an image to the specified dimensions
func (ip *ImageProcessor) resizeImage(img image.Image, width, height int) (image.Image, error) {
	// Simple resize implementation - in production, use a library like imaging or resize
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	// Calculate aspect ratio preserving dimensions
	aspectRatio := float64(originalWidth) / float64(originalHeight)
	
	if float64(width)/float64(height) > aspectRatio {
		// Height is the limiting factor
		width = int(float64(height) * aspectRatio)
	} else {
		// Width is the limiting factor
		height = int(float64(width) / aspectRatio)
	}
	
	// For now, return original image (in production, implement actual resizing)
	// This is a placeholder - you would use a proper image resizing library
	return img, nil
}

// processImageVariant processes a single image variant
func (ip *ImageProcessor) processImageVariant(imageID uint64, img image.Image, size models.ImageSize, format models.ImageFormat, quality int) (*models.ImageVariant, error) {
	// Generate output path
	outputPath := ip.generateVariantPath(imageID, size, format)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()
	
	// Encode image in the specified format
	var buf bytes.Buffer
	bounds := img.Bounds()
	
	switch format {
	case models.ImageFormatJPEG:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	case models.ImageFormatPNG:
		err = png.Encode(&buf, img)
	case models.ImageFormatWebP:
		// WebP encoding would require a WebP library
		// For now, fallback to JPEG
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	case models.ImageFormatAVIF:
		// AVIF encoding would require an AVIF library
		// For now, fallback to JPEG
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	
	// Write to file
	if _, err := io.Copy(outputFile, &buf); err != nil {
		return nil, fmt.Errorf("failed to write image file: %w", err)
	}
	
	// Generate URL (relative to web root)
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
	// Organize by date for better file system performance
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	
	filename := fmt.Sprintf("%d_%s.%s", imageID, size, format)
	return filepath.Join(ip.storageBasePath, "images", year, month, day, filename)
}

// generateVariantURL generates the web URL for an image variant
func (ip *ImageProcessor) generateVariantURL(imageID uint64, size models.ImageSize, format models.ImageFormat) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	
	filename := fmt.Sprintf("%d_%s.%s", imageID, size, format)
	return fmt.Sprintf("/images/%s/%s/%s/%s", year, month, day, filename)
}

// GenerateResponsiveImageHTML generates HTML for responsive images with lazy loading
func (ip *ImageProcessor) GenerateResponsiveImageHTML(image *models.Image, variants []models.ImageVariant, lazyLoad bool) string {
	if len(variants) == 0 {
		return ""
	}
	
	// Group variants by format
	formatGroups := make(map[models.ImageFormat][]models.ImageVariant)
	for _, variant := range variants {
		formatGroups[variant.Format] = append(formatGroups[variant.Format], variant)
	}
	
	var html strings.Builder
	html.WriteString("<picture>")
	
	// Generate source elements for each format in priority order
	formatPriority := models.GetFormatPriority()
	for _, format := range formatPriority {
		variants, exists := formatGroups[format]
		if !exists {
			continue
		}
		
		// Generate srcset for this format
		var srcset []string
		for _, variant := range variants {
			srcset = append(srcset, fmt.Sprintf("%s %dw", variant.URL, variant.Width))
		}
		
		if len(srcset) > 0 {
			mimeType := ip.getMimeType(format)
			html.WriteString(fmt.Sprintf(`<source type="%s" srcset="%s">`, 
				mimeType, strings.Join(srcset, ", ")))
		}
	}
	
	// Fallback img element
	fallbackVariant := ip.getFallbackVariant(variants)
	imgAttrs := []string{
		fmt.Sprintf(`src="%s"`, fallbackVariant.URL),
		fmt.Sprintf(`alt="%s"`, image.AltText),
		fmt.Sprintf(`width="%d"`, fallbackVariant.Width),
		fmt.Sprintf(`height="%d"`, fallbackVariant.Height),
	}
	
	if lazyLoad {
		imgAttrs = append(imgAttrs, `loading="lazy"`)
	}
	
	html.WriteString(fmt.Sprintf("<img %s>", strings.Join(imgAttrs, " ")))
	html.WriteString("</picture>")
	
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
	// Prefer JPEG medium size
	for _, variant := range variants {
		if variant.Format == models.ImageFormatJPEG && variant.Size == models.ImageSizeMedium {
			return variant
		}
	}
	
	// Fallback to any JPEG
	for _, variant := range variants {
		if variant.Format == models.ImageFormatJPEG {
			return variant
		}
	}
	
	// Last resort: return first variant
	if len(variants) > 0 {
		return variants[0]
	}
	
	// This shouldn't happen, but return empty variant
	return models.ImageVariant{}
}

// Shutdown gracefully shuts down the image processor
func (ip *ImageProcessor) Shutdown() error {
	log.Println("Shutting down image processor...")
	
	// Cancel context to stop workers
	ip.cancel()
	
	// Wait for all workers to finish
	ip.workers.Wait()
	
	// Close job queue
	close(ip.jobQueue)
	
	log.Println("Image processor shutdown complete")
	return nil
}

// GetQueueStatus returns the current status of the job queue
func (ip *ImageProcessor) GetQueueStatus() map[string]interface{} {
	return map[string]interface{}{
		"queue_length": len(ip.jobQueue),
		"queue_capacity": cap(ip.jobQueue),
		"workers": ip.maxWorkers,
	}
}