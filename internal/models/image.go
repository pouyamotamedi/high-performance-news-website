package models

import (
	"time"
)

// ImageSize represents different image size variants
type ImageSize string

const (
	ImageSizeThumbnail ImageSize = "thumbnail" // 150x150
	ImageSizeSmall     ImageSize = "small"     // 300x200
	ImageSizeMedium    ImageSize = "medium"    // 600x400
	ImageSizeLarge     ImageSize = "large"     // 1200x800
	ImageSizeOriginal  ImageSize = "original"  // Original size
)

// ImageFormat represents supported image formats
type ImageFormat string

const (
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatWebP ImageFormat = "webp"
	ImageFormatAVIF ImageFormat = "avif"
	ImageFormatPNG  ImageFormat = "png"
)

// Image represents an image in the system
type Image struct {
	ID          uint64    `json:"id" db:"id"`
	OriginalURL string    `json:"original_url" db:"original_url"`
	Filename    string    `json:"filename" db:"filename"`
	AltText     string    `json:"alt_text" db:"alt_text"`
	Caption     string    `json:"caption" db:"caption"`
	Width       int       `json:"width" db:"width"`
	Height      int       `json:"height" db:"height"`
	FileSize    int64     `json:"file_size" db:"file_size"`
	MimeType    string    `json:"mime_type" db:"mime_type"`
	Hash        string    `json:"hash,omitempty" db:"hash"` // SHA256 hash for deduplication
	ArticleID   *uint64   `json:"article_id,omitempty" db:"article_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ImageVariant represents different size/format variants of an image
type ImageVariant struct {
	ID       uint64      `json:"id" db:"id"`
	ImageID  uint64      `json:"image_id" db:"image_id"`
	Size     ImageSize   `json:"size" db:"size"`
	Format   ImageFormat `json:"format" db:"format"`
	URL      string      `json:"url" db:"url"`
	Width    int         `json:"width" db:"width"`
	Height   int         `json:"height" db:"height"`
	FileSize int64       `json:"file_size" db:"file_size"`
	Quality  int         `json:"quality" db:"quality"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ImageProcessingJob represents a job for processing images
type ImageProcessingJob struct {
	ID        uint64                 `json:"id"`
	ImageID   uint64                 `json:"image_id"`
	Sizes     []ImageSize            `json:"sizes"`
	Formats   []ImageFormat          `json:"formats"`
	Priority  JobPriority            `json:"priority"`
	Status    JobStatus              `json:"status"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ImageSizeConfig defines dimensions and quality for each size
type ImageSizeConfig struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Quality int `json:"quality"`
}

// GetImageSizeConfig returns configuration for each image size
func GetImageSizeConfig() map[ImageSize]ImageSizeConfig {
	return map[ImageSize]ImageSizeConfig{
		ImageSizeThumbnail: {Width: 150, Height: 150, Quality: 85},
		ImageSizeSmall:     {Width: 300, Height: 200, Quality: 85},
		ImageSizeMedium:    {Width: 600, Height: 400, Quality: 90},
		ImageSizeLarge:     {Width: 1200, Height: 800, Quality: 95},
	}
}

// IsCriticalSize returns true if the image size is critical for the application
func (s ImageSize) IsCriticalSize() bool {
	switch s {
	case ImageSizeThumbnail, ImageSizeSmall:
		return true
	default:
		return false
	}
}

// GetFormatPriority returns the priority order for image formats
func GetFormatPriority() []ImageFormat {
	return []ImageFormat{
		ImageFormatAVIF, // Best compression
		ImageFormatWebP, // Good compression, wide support
		ImageFormatJPEG, // Universal fallback
	}
}