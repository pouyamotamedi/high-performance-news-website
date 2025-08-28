# Image Processing System

This document describes the image processing pipeline implementation for the high-performance news website.

## Overview

The image processing system provides:
- Multi-format image generation (JPEG, WebP, AVIF)
- Responsive image variants (thumbnail, small, medium, large)
- Error recovery for critical vs non-critical image sizes
- Lazy loading and progressive image loading support
- Asynchronous processing with job queues
- Responsive HTML generation

## Architecture

### Components

1. **ImageProcessor** - Core processing engine
2. **ImageHandlers** - HTTP API handlers
3. **Image Models** - Data structures for images and variants
4. **Job Queue** - Asynchronous processing system

### Image Sizes

- **Thumbnail**: 150x150px (critical)
- **Small**: 300x200px (critical)
- **Medium**: 600x400px (non-critical)
- **Large**: 1200x800px (non-critical)
- **Original**: Unchanged dimensions

### Image Formats

- **AVIF**: Best compression, modern browsers
- **WebP**: Good compression, wide support
- **JPEG**: Universal fallback format
- **PNG**: For images requiring transparency

## Usage

### Basic Image Processing

```go
// Create image processor
config := services.ImageProcessorConfig{
    StorageBasePath: "/var/www/static",
    MaxWorkers:      4,
    QueueSize:       100,
}
processor := services.NewImageProcessor(config)

// Process image synchronously
sizes := []models.ImageSize{
    models.ImageSizeThumbnail,
    models.ImageSizeSmall,
    models.ImageSizeMedium,
}

formats := []models.ImageFormat{
    models.ImageFormatJPEG,
    models.ImageFormatWebP,
    models.ImageFormatAVIF,
}

variants, err := processor.ProcessImageSync(imageID, originalPath, sizes, formats)
```

### Asynchronous Processing

```go
// Queue image for processing
err := processor.ProcessImage(imageID, originalPath, sizes, formats)
if err != nil {
    log.Printf("Failed to queue image: %v", err)
}

// Check queue status
status := processor.GetQueueStatus()
fmt.Printf("Queue length: %d/%d\n", 
    status["queue_length"], 
    status["queue_capacity"])
```

### Responsive HTML Generation

```go
// Generate responsive HTML with lazy loading
html := processor.GenerateResponsiveImageHTML(image, variants, true)

// Example output:
// <picture>
//   <source type="image/avif" srcset="/images/2024/01/01/1_small.avif 300w, /images/2024/01/01/1_medium.avif 600w">
//   <source type="image/webp" srcset="/images/2024/01/01/1_small.webp 300w, /images/2024/01/01/1_medium.webp 600w">
//   <source type="image/jpeg" srcset="/images/2024/01/01/1_small.jpeg 300w, /images/2024/01/01/1_medium.jpeg 600w">
//   <img src="/images/2024/01/01/1_medium.jpeg" alt="Image description" width="600" height="400" loading="lazy">
// </picture>
```

## API Endpoints

### Upload Image

```http
POST /api/v1/images/upload
Content-Type: multipart/form-data

image: [file]
alt_text: "Image description"
caption: "Image caption"
article_id: "123"
```

Response:
```json
{
  "image": {
    "id": 1,
    "original_url": "/uploads/originals/2024/01/01/1_original.jpg",
    "filename": "photo.jpg",
    "alt_text": "Image description",
    "caption": "Image caption",
    "width": 1200,
    "height": 800,
    "file_size": 245760,
    "mime_type": "image/jpeg",
    "article_id": 123,
    "created_at": "2024-01-01T12:00:00Z"
  },
  "variants": [
    {
      "id": 1,
      "image_id": 1,
      "size": "small",
      "format": "jpeg",
      "url": "/images/2024/01/01/1_small.jpeg",
      "width": 300,
      "height": 200,
      "file_size": 15360,
      "quality": 85
    }
  ],
  "html": "<picture>...</picture>"
}
```

### Process Image Variants

```http
POST /api/v1/images/123/variants?sizes=small,medium&formats=jpeg,webp
```

### Get Responsive HTML

```http
GET /api/v1/images/123/html?lazy=true
```

### Get Processing Status

```http
GET /api/v1/images/status
```

Response:
```json
{
  "queue_length": 5,
  "queue_capacity": 100,
  "workers": 4
}
```

## Error Handling

### Critical vs Non-Critical Sizes

The system distinguishes between critical and non-critical image sizes:

- **Critical sizes** (thumbnail, small): Must be generated successfully
- **Non-critical sizes** (medium, large): Failures are logged but don't fail the operation

```go
// Critical sizes that must succeed
func (s ImageSize) IsCriticalSize() bool {
    switch s {
    case ImageSizeThumbnail, ImageSizeSmall:
        return true
    default:
        return false
    }
}
```

### Error Recovery

1. **File System Errors**: Automatic directory creation, permission handling
2. **Processing Errors**: Graceful degradation, fallback to original
3. **Queue Overflow**: Backpressure handling, error reporting
4. **Format Failures**: Format-specific fallbacks (AVIF → WebP → JPEG)

## Performance Considerations

### Optimization Strategies

1. **Concurrent Processing**: Multiple worker goroutines
2. **Batch Operations**: Process multiple images simultaneously
3. **Memory Management**: Streaming processing for large images
4. **Cache-Friendly Paths**: Date-based directory structure
5. **Progressive Enhancement**: Critical sizes first

### Benchmarks

Typical performance on modern hardware:
- Single image (4 sizes × 3 formats): ~500ms
- Batch processing (10 images): ~2-3 seconds
- HTML generation: <1ms per image
- Queue throughput: 100+ images/minute

## File Organization

### Storage Structure

```
/var/www/static/
├── uploads/
│   └── originals/
│       └── 2024/01/01/
│           └── 123_original.jpg
└── images/
    └── 2024/01/01/
        ├── 123_thumbnail.jpeg
        ├── 123_thumbnail.webp
        ├── 123_small.jpeg
        ├── 123_small.webp
        ├── 123_medium.jpeg
        └── 123_medium.webp
```

### URL Structure

- Original: `/uploads/originals/2024/01/01/123_original.jpg`
- Variants: `/images/2024/01/01/123_small.webp`

## Configuration

### Environment Variables

```bash
# Image processing settings
IMAGE_STORAGE_PATH=/var/www/static
IMAGE_MAX_WORKERS=4
IMAGE_QUEUE_SIZE=100
IMAGE_MAX_FILE_SIZE=10485760  # 10MB

# Quality settings
IMAGE_JPEG_QUALITY=85
IMAGE_WEBP_QUALITY=80
IMAGE_AVIF_QUALITY=75
```

### Production Recommendations

1. **Storage**: Use SSD storage for better I/O performance
2. **Workers**: Set to number of CPU cores
3. **Queue Size**: 10-20x worker count
4. **Memory**: Monitor memory usage during peak processing
5. **Monitoring**: Track queue length and processing times

## Testing

### Unit Tests

```bash
go test ./internal/services -run TestImageProcessor
```

### Integration Tests

```bash
go test ./internal/integration -run TestImageProcessing
```

### Load Testing

```bash
go test -bench=BenchmarkImageProcessor ./internal/services
```

## Future Enhancements

1. **Advanced Formats**: Support for HEIC, JXL
2. **AI Processing**: Automatic alt-text generation, content-aware cropping
3. **CDN Integration**: Automatic CDN upload and purging
4. **Metadata Extraction**: EXIF data processing, geolocation
5. **Watermarking**: Automatic watermark application
6. **Compression**: Advanced compression algorithms

## Troubleshooting

### Common Issues

1. **Queue Full**: Increase queue size or add more workers
2. **Slow Processing**: Check disk I/O, increase worker count
3. **Memory Issues**: Implement streaming for large images
4. **Format Errors**: Verify image library dependencies

### Monitoring

```go
// Monitor queue status
status := processor.GetQueueStatus()
if status["queue_length"].(int) > status["queue_capacity"].(int) * 0.8 {
    log.Warn("Image processing queue is nearly full")
}
```

### Logs

The system logs important events:
- Image processing start/completion
- Error conditions and recovery
- Performance metrics
- Queue status changes

## Security Considerations

1. **File Validation**: Strict MIME type checking
2. **Size Limits**: Configurable maximum file sizes
3. **Path Traversal**: Secure file path generation
4. **Resource Limits**: Memory and CPU usage monitoring
5. **Access Control**: Authentication required for uploads