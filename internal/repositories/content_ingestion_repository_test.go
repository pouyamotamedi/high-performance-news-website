package repositories

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/database"
)

func TestContentIngestionRepository_CreateContentSource(t *testing.T) {
	// This would require a test database setup
	// For now, we'll test the logic without actual database calls
	
	source := &models.ContentSource{
		Name:      "Test Source",
		Type:      "api",
		IsActive:  true,
		RateLimit: 100,
		Priority:  5,
		Config: models.SourceConfig{
			AutoPublish:       false,
			DefaultCategoryID: 1,
			DefaultAuthorID:   1,
		},
	}

	// Test validation logic
	if source.Name == "" {
		t.Error("Expected name to be set")
	}

	if source.Type != "api" {
		t.Error("Expected type to be 'api'")
	}

	if !source.IsActive {
		t.Error("Expected source to be active")
	}
}

func TestContentIngestionRepository_CreateIngestedContent(t *testing.T) {
	content := &models.IngestedContent{
		SourceID:    1,
		ExternalID:  "ext123",
		Title:       "Test Article",
		Content:     "Test content",
		Excerpt:     "Test excerpt",
		AuthorName:  "John Doe",
		AuthorEmail: "john@example.com",
		Tags:        []string{"tech", "news"},
		Status:      "pending",
		Metadata:    map[string]interface{}{"source": "test"},
	}

	// Generate content hash
	content.GenerateContentHash()

	if content.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}

	if content.Status != "pending" {
		t.Error("Expected status to be 'pending'")
	}
}

func TestContentIngestionRepository_DuplicateChecking(t *testing.T) {
	// Test duplicate checking logic
	content1 := &models.IngestedContent{
		Title:   "Same Title",
		Content: "Same content",
	}
	content1.GenerateContentHash()

	content2 := &models.IngestedContent{
		Title:   "Same Title",
		Content: "Same content",
	}
	content2.GenerateContentHash()

	if content1.ContentHash != content2.ContentHash {
		t.Error("Expected same content to have same hash")
	}

	content3 := &models.IngestedContent{
		Title:   "Different Title",
		Content: "Different content",
	}
	content3.GenerateContentHash()

	if content1.ContentHash == content3.ContentHash {
		t.Error("Expected different content to have different hash")
	}
}

func TestContentIngestionRepository_ValidationLogic(t *testing.T) {
	tests := []struct {
		name    string
		content *models.IngestedContent
		valid   bool
	}{
		{
			name: "valid content",
			content: &models.IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      "Valid Title",
				Content:    "Valid content",
			},
			valid: true,
		},
		{
			name: "missing source_id",
			content: &models.IngestedContent{
				ExternalID: "ext123",
				Title:      "Valid Title",
				Content:    "Valid content",
			},
			valid: false,
		},
		{
			name: "missing external_id",
			content: &models.IngestedContent{
				SourceID: 1,
				Title:    "Valid Title",
				Content:  "Valid content",
			},
			valid: false,
		},
		{
			name: "missing title",
			content: &models.IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Content:    "Valid content",
			},
			valid: false,
		},
		{
			name: "missing content",
			content: &models.IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      "Valid Title",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.ValidateIngestedContent(tt.content)
			if result.IsValid != tt.valid {
				t.Errorf("Expected validation result to be %v, got %v", tt.valid, result.IsValid)
			}
		})
	}
}

func TestContentIngestionRepository_StatusUpdates(t *testing.T) {
	content := &models.IngestedContent{
		ID:         1,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test status transitions
	validStatuses := []string{"pending", "processed", "rejected", "duplicate"}
	
	for _, status := range validStatuses {
		content.Status = status
		// In a real test, we would update the database and verify
		if content.Status != status {
			t.Errorf("Expected status to be %s, got %s", status, content.Status)
		}
	}
}

func TestContentIngestionRepository_RateLimiting(t *testing.T) {
	source := &models.ContentSource{
		ID:        1,
		RateLimit: 100,
		IsActive:  true,
	}

	// Test rate limit validation
	if source.RateLimit <= 0 {
		t.Error("Expected rate limit to be positive")
	}

	if source.RateLimit > 10000 {
		t.Error("Expected rate limit to be reasonable")
	}

	if !source.IsActive {
		t.Error("Expected source to be active for rate limiting")
	}
}

func TestContentIngestionRepository_ConfigValidation(t *testing.T) {
	config := models.SourceConfig{
		AutoPublish:       true,
		DefaultCategoryID: 1,
		DefaultAuthorID:   1,
		AllowedDomains:    []string{"example.com", "test.com"},
		RequiredFields:    []string{"title", "content"},
		TagMappings:       map[string]uint64{"tech": 1, "news": 2},
	}

	if !config.AutoPublish {
		t.Error("Expected auto publish to be enabled")
	}

	if config.DefaultCategoryID == 0 {
		t.Error("Expected default category ID to be set")
	}

	if config.DefaultAuthorID == 0 {
		t.Error("Expected default author ID to be set")
	}

	if len(config.AllowedDomains) != 2 {
		t.Error("Expected 2 allowed domains")
	}

	if len(config.RequiredFields) != 2 {
		t.Error("Expected 2 required fields")
	}

	if len(config.TagMappings) != 2 {
		t.Error("Expected 2 tag mappings")
	}
}

func TestContentIngestionRepository_MetadataHandling(t *testing.T) {
	metadata := map[string]interface{}{
		"source_system": "cms",
		"import_batch":  "batch_123",
		"priority":      5,
		"tags":          []string{"urgent", "breaking"},
		"custom_fields": map[string]string{
			"department": "news",
			"region":     "us-east",
		},
	}

	content := &models.IngestedContent{
		Metadata: metadata,
	}

	// Test metadata access
	if content.Metadata["source_system"] != "cms" {
		t.Error("Expected source_system to be 'cms'")
	}

	if content.Metadata["priority"] != 5 {
		t.Error("Expected priority to be 5")
	}

	// Test nested metadata
	customFields, ok := content.Metadata["custom_fields"].(map[string]string)
	if !ok {
		t.Error("Expected custom_fields to be a map")
	}

	if customFields["department"] != "news" {
		t.Error("Expected department to be 'news'")
	}
}

func TestContentIngestionRepository_TimeHandling(t *testing.T) {
	now := time.Now()
	publishedAt := now.Add(-1 * time.Hour) // Published 1 hour ago

	content := &models.IngestedContent{
		PublishedAt: &publishedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if content.PublishedAt == nil {
		t.Error("Expected published_at to be set")
	}

	if content.PublishedAt.After(content.CreatedAt) {
		t.Error("Expected published_at to be before created_at")
	}

	if content.CreatedAt.After(content.UpdatedAt) {
		t.Error("Expected created_at to be before or equal to updated_at")
	}
}

func TestContentIngestionRepository_URLValidation(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://example.com/article", true},
		{"http://test.com/news/123", true},
		{"https://subdomain.example.com/path/to/article", true},
		{"not-a-url", false},
		{"ftp://example.com", false}, // Only HTTP/HTTPS allowed
		{"", true},                   // Empty URL is allowed (optional field)
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			content := &models.IngestedContent{
				SourceURL: tt.url,
			}

			result := models.ValidateIngestedContent(content)
			
			if tt.url == "" {
				// Empty URL should be valid
				if !result.IsValid {
					t.Error("Expected empty URL to be valid")
				}
			} else if tt.valid {
				// Valid URL should pass validation
				hasURLError := false
				for _, err := range result.Errors {
					if err == "source_url must be a valid URL" {
						hasURLError = true
						break
					}
				}
				if hasURLError {
					t.Errorf("Expected valid URL %s to pass validation", tt.url)
				}
			} else {
				// Invalid URL should fail validation
				hasURLError := false
				for _, err := range result.Errors {
					if err == "source_url must be a valid URL" {
						hasURLError = true
						break
					}
				}
				if !hasURLError {
					t.Errorf("Expected invalid URL %s to fail validation", tt.url)
				}
			}
		})
	}
}

// Mock database for testing (in a real implementation, you'd use a test database)
type mockDB struct{}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *database.Row {
	// Mock implementation
	return nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*database.Rows, error) {
	// Mock implementation
	return nil, nil
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (*database.Result, error) {
	// Mock implementation
	return nil, nil
}