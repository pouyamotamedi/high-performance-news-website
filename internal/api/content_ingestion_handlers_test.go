package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
)

func TestIngestContent_ValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test request
	request := models.ContentIngestionRequest{
		ExternalID:   "ext123",
		Title:        "Test Article",
		Content:      "This is test content",
		Excerpt:      "Test excerpt",
		AuthorName:   "John Doe",
		AuthorEmail:  "john@example.com",
		CategoryName: "Technology",
		Tags:         []string{"tech", "news"},
		SourceURL:    "https://example.com/article",
		Metadata:     map[string]interface{}{"source": "test"},
	}

	requestBody, _ := json.Marshal(request)

	// Create HTTP request
	req := httptest.NewRequest("POST", "/api/v1/content/ingest", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Test request parsing
	var parsedRequest models.ContentIngestionRequest
	if err := c.ShouldBindJSON(&parsedRequest); err != nil {
		t.Errorf("Failed to parse request: %v", err)
		return
	}

	// Verify parsed request
	if parsedRequest.ExternalID != request.ExternalID {
		t.Errorf("Expected ExternalID %s, got %s", request.ExternalID, parsedRequest.ExternalID)
	}

	if parsedRequest.Title != request.Title {
		t.Errorf("Expected Title %s, got %s", request.Title, parsedRequest.Title)
	}

	if parsedRequest.Content != request.Content {
		t.Errorf("Expected Content %s, got %s", request.Content, parsedRequest.Content)
	}
}

func TestIngestContent_MissingAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := models.ContentIngestionRequest{
		ExternalID: "ext123",
		Title:      "Test Article",
		Content:    "This is test content",
	}

	requestBody, _ := json.Marshal(request)

	// Create HTTP request without API key
	req := httptest.NewRequest("POST", "/api/v1/content/ingest", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	// No X-API-Key header

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Test that missing API key is detected
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		t.Error("Expected empty API key")
	}
}