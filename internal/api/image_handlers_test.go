package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageHandlers_UploadImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tempDir := t.TempDir()
	
	// Setup image processor
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	// Setup handlers
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024) // 10MB max
	
	router := gin.New()
	router.POST("/upload", handlers.UploadImage)
	
	t.Run("SuccessfulUpload", func(t *testing.T) {
		// Create test image
		imageData := createTestImageData(t, 400, 300)
		
		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		// Add image file
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		// Add form fields
		writer.WriteField("alt_text", "Test image")
		writer.WriteField("caption", "Test caption")
		writer.WriteField("article_id", "123")
		
		writer.Close()
		
		// Make request
		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response UploadImageResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.NotNil(t, response.Image)
		assert.Equal(t, "Test image", response.Image.AltText)
		assert.Equal(t, "Test caption", response.Image.Caption)
		assert.NotNil(t, response.Image.ArticleID)
		assert.Equal(t, uint64(123), *response.Image.ArticleID)
		assert.NotEmpty(t, response.HTML)
	})
	
	t.Run("NoImageFile", func(t *testing.T) {
		// Create multipart form without image
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("alt_text", "Test image")
		writer.Close()
		
		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "No image file provided")
	})
	
	t.Run("FileSizeExceeded", func(t *testing.T) {
		// Create handlers with very small max file size
		smallHandlers := NewImageHandlers(processor, tempDir, 100) // 100 bytes max
		
		smallRouter := gin.New()
		smallRouter.POST("/upload", smallHandlers.UploadImage)
		
		// Create test image (will be larger than 100 bytes)
		imageData := createTestImageData(t, 400, 300)
		
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		writer.Close()
		
		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		w := httptest.NewRecorder()
		smallRouter.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "File size exceeds maximum")
	})
	
	t.Run("InvalidFileType", func(t *testing.T) {
		// Create multipart form with text file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		part, err := writer.CreateFormFile("image", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("This is not an image"))
		require.NoError(t, err)
		
		writer.Close()
		
		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid file type")
	})
}

func TestImageHandlers_ProcessImageVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tempDir := t.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024)
	
	router := gin.New()
	router.POST("/images/:id/variants", handlers.ProcessImageVariants)
	
	t.Run("SuccessfulProcessing", func(t *testing.T) {
		// Create original image file
		imageID := uint64(123)
		originalPath := filepath.Join(tempDir, "originals", fmt.Sprintf("%d_original.jpg", imageID))
		createTestImageFile(t, originalPath, 800, 600)
		
		req := httptest.NewRequest("POST", "/images/123/variants?sizes=small,medium&formats=jpeg,webp", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, float64(123), response["image_id"])
		assert.Contains(t, response, "variants")
		assert.Contains(t, response, "count")
	})
	
	t.Run("InvalidImageID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/images/invalid/variants", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid image ID")
	})
	
	t.Run("OriginalFileNotFound", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/images/999/variants", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Original image file not found")
	})
}

func TestImageHandlers_GetImageHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tempDir := t.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024)
	
	router := gin.New()
	router.GET("/images/:id/html", handlers.GetImageHTML)
	
	t.Run("GenerateHTML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/images/123/html", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, float64(123), response["image_id"])
		assert.Contains(t, response, "html")
		assert.Equal(t, true, response["lazy_load"])
		
		html := response["html"].(string)
		assert.Contains(t, html, "<picture>")
		assert.Contains(t, html, `loading="lazy"`)
	})
	
	t.Run("GenerateHTMLNoLazy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/images/123/html?lazy=false", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, false, response["lazy_load"])
		
		html := response["html"].(string)
		assert.NotContains(t, html, `loading="lazy"`)
	})
	
	t.Run("InvalidImageID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/images/invalid/html", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid image ID")
	})
}

func TestImageHandlers_GetProcessingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tempDir := t.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      2,
		QueueSize:       10,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024)
	
	router := gin.New()
	router.GET("/status", handlers.GetProcessingStatus)
	
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "queue_length")
	assert.Contains(t, response, "queue_capacity")
	assert.Contains(t, response, "workers")
	assert.Equal(t, float64(10), response["queue_capacity"])
	assert.Equal(t, float64(2), response["workers"])
}

func TestImageHandlers_HelperMethods(t *testing.T) {
	tempDir := t.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      1,
		QueueSize:       5,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024)
	
	t.Run("IsValidImageType", func(t *testing.T) {
		tests := []struct {
			contentType string
			valid       bool
		}{
			{"image/jpeg", true},
			{"image/jpg", true},
			{"image/png", true},
			{"image/webp", true},
			{"image/gif", false},
			{"text/plain", false},
			{"application/pdf", false},
		}
		
		for _, test := range tests {
			result := handlers.isValidImageType(test.contentType)
			assert.Equal(t, test.valid, result, "Content type: %s", test.contentType)
		}
	})
	
	t.Run("ParseSizes", func(t *testing.T) {
		tests := []struct {
			input    string
			expected []models.ImageSize
		}{
			{
				"thumbnail,small",
				[]models.ImageSize{models.ImageSizeThumbnail, models.ImageSizeSmall},
			},
			{
				"medium,large,original",
				[]models.ImageSize{models.ImageSizeMedium, models.ImageSizeLarge, models.ImageSizeOriginal},
			},
			{
				"invalid,thumbnail",
				[]models.ImageSize{models.ImageSizeThumbnail},
			},
			{
				"",
				[]models.ImageSize{},
			},
		}
		
		for _, test := range tests {
			result := handlers.parseSizes(test.input)
			assert.Equal(t, test.expected, result, "Input: %s", test.input)
		}
	})
	
	t.Run("ParseFormats", func(t *testing.T) {
		tests := []struct {
			input    string
			expected []models.ImageFormat
		}{
			{
				"jpeg,webp",
				[]models.ImageFormat{models.ImageFormatJPEG, models.ImageFormatWebP},
			},
			{
				"jpg,png,avif",
				[]models.ImageFormat{models.ImageFormatJPEG, models.ImageFormatPNG, models.ImageFormatAVIF},
			},
			{
				"invalid,jpeg",
				[]models.ImageFormat{models.ImageFormatJPEG},
			},
			{
				"",
				[]models.ImageFormat{},
			},
		}
		
		for _, test := range tests {
			result := handlers.parseFormats(test.input)
			assert.Equal(t, test.expected, result, "Input: %s", test.input)
		}
	})
	
	t.Run("GenerateOriginalFilename", func(t *testing.T) {
		filename := handlers.generateOriginalFilename(123, "test.jpg")
		
		assert.Contains(t, filename, "123_original.jpg")
		assert.Contains(t, filename, "/") // Should contain date structure
	})
}

// Helper functions

func createTestImageData(t *testing.T, width, height int) []byte {
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
	
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
	
	return buf.Bytes()
}

func createTestImageFile(t *testing.T, path string, width, height int) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	
	imageData := createTestImageData(t, width, height)
	
	err = os.WriteFile(path, imageData, 0644)
	require.NoError(t, err)
}

// Benchmark tests

func BenchmarkImageHandlers_UploadImage(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	tempDir := b.TempDir()
	
	config := services.ImageProcessorConfig{
		StorageBasePath: tempDir,
		MaxWorkers:      2,
		QueueSize:       10,
	}
	processor := services.NewImageProcessor(config)
	defer processor.Shutdown()
	
	handlers := NewImageHandlers(processor, tempDir, 10*1024*1024)
	
	router := gin.New()
	router.POST("/upload", handlers.UploadImage)
	
	// Pre-create image data
	imageData := createTestImageData(b, 400, 300)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		part, _ := writer.CreateFormFile("image", "test.jpg")
		part.Write(imageData)
		writer.WriteField("alt_text", "Benchmark image")
		writer.Close()
		
		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusCreated {
			b.Fatalf("Expected 201, got %d", w.Code)
		}
	}
}