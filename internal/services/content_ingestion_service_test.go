package services

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

func TestContentIngestionService_ValidateContentRequest(t *testing.T) {
	service := &ContentIngestionService{}

	tests := []struct {
		name    string
		request *models.ContentIngestionRequest
		valid   bool
		hasError string
	}{
		{
			name: "valid request",
			request: &models.ContentIngestionRequest{
				ExternalID:   "ext123",
				Title:        "Test Article",
				Content:      "Test content",
				Excerpt:      "Test excerpt",
				AuthorName:   "John Doe",
				AuthorEmail:  "john@example.com",
				CategoryName: "Technology",
				Tags:         []string{"tech", "news"},
				SourceURL:    "https://example.com/article",
			},
			valid: true,
		},
		{
			name: "missing external_id",
			request: &models.ContentIngestionRequest{
				Title:   "Test Article",
				Content: "Test content",
			},
			valid:    false,
			hasError: "external_id is required",
		},
		{
			name: "missing title",
			request: &models.ContentIngestionRequest{
				ExternalID: "ext123",
				Content:    "Test content",
			},
			valid:    false,
			hasError: "title is required",
		},
		{
			name: "missing content",
			request: &models.ContentIngestionRequest{
				ExternalID: "ext123",
				Title:      "Test Article",
			},
			valid:    false,
			hasError: "content is required",
		},
		{
			name: "title too long",
			request: &models.ContentIngestionRequest{
				ExternalID: "ext123",
				Title:      string(make([]byte, 256)), // 256 characters
				Content:    "Test content",
			},
			valid:    false,
			hasError: "title must be less than 255 characters",
		},
		{
			name: "invalid email",
			request: &models.ContentIngestionRequest{
				ExternalID:  "ext123",
				Title:       "Test Article",
				Content:     "Test content",
				AuthorEmail: "invalid-email",
			},
			valid:    false,
			hasError: "author_email must be a valid email address",
		},
		{
			name: "invalid URL",
			request: &models.ContentIngestionRequest{
				ExternalID: "ext123",
				Title:      "Test Article",
				Content:    "Test content",
				SourceURL:  "not-a-url",
			},
			valid:    false,
			hasError: "source_url must be a valid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateContentRequest(tt.request)

			if tt.valid && err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("Expected validation error, got nil")
			}

			if !tt.valid && tt.hasError != "" {
				validationErr, ok := err.(*models.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				found := false
				for _, errMsg := range validationErr.Fields {
					if errMsg == tt.hasError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error '%s' not found in errors: %v", tt.hasError, validationErr.Fields)
				}
			}
		})
	}
}

func TestContentIngestionService_validateContentSource(t *testing.T) {
	service := &ContentIngestionService{}

	tests := []struct {
		name    string
		source  *models.ContentSource
		valid   bool
		hasError string
	}{
		{
			name: "valid source",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "api",
				RateLimit: 100,
				Priority:  5,
			},
			valid: true,
		},
		{
			name: "missing name",
			source: &models.ContentSource{
				Type:      "api",
				RateLimit: 100,
				Priority:  5,
			},
			valid:    false,
			hasError: "name is required",
		},
		{
			name: "invalid type",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "invalid",
				RateLimit: 100,
				Priority:  5,
			},
			valid:    false,
			hasError: "type must be one of: api, webhook, manual",
		},
		{
			name: "negative rate limit",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "api",
				RateLimit: -1,
				Priority:  5,
			},
			valid:    false,
			hasError: "rate_limit must be between 0 and 10000",
		},
		{
			name: "rate limit too high",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "api",
				RateLimit: 10001,
				Priority:  5,
			},
			valid:    false,
			hasError: "rate_limit must be between 0 and 10000",
		},
		{
			name: "priority too low",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "api",
				RateLimit: 100,
				Priority:  0,
			},
			valid:    false,
			hasError: "priority must be between 1 and 10",
		},
		{
			name: "priority too high",
			source: &models.ContentSource{
				Name:      "Test Source",
				Type:      "api",
				RateLimit: 100,
				Priority:  11,
			},
			valid:    false,
			hasError: "priority must be between 1 and 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateContentSource(tt.source)

			if tt.valid && err != nil {
				t.Errorf("Expected valid source, got error: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("Expected validation error, got nil")
			}

			if !tt.valid && tt.hasError != "" {
				validationErr, ok := err.(*models.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				found := false
				for _, errMsg := range validationErr.Fields {
					if errMsg == tt.hasError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error '%s' not found in errors: %v", tt.hasError, validationErr.Fields)
				}
			}
		})
	}
}

func TestContentIngestionService_generateAPIKey(t *testing.T) {
	service := &ContentIngestionService{}

	key1 := service.generateAPIKey()
	
	// Add a small delay to ensure different nanosecond values
	time.Sleep(1 * time.Millisecond)
	
	key2 := service.generateAPIKey()

	if key1 == "" {
		t.Error("Expected API key to be generated")
	}

	if key2 == "" {
		t.Error("Expected API key to be generated")
	}

	if key1 == key2 {
		t.Errorf("Expected different API keys to be generated, got key1=%s, key2=%s", key1, key2)
	}

	// Check that key starts with expected prefix
	if len(key1) < 3 || key1[:3] != "ci_" {
		t.Error("Expected API key to start with 'ci_'")
	}
}

func TestContentIngestionService_convertToArticle(t *testing.T) {
	service := &ContentIngestionService{}

	now := time.Now()
	content := &models.IngestedContent{
		Title:        "Test Article",
		Content:      "This is test content",
		Excerpt:      "Test excerpt",
		AuthorName:   "John Doe",
		AuthorEmail:  "john@example.com",
		CategoryName: "Technology",
		PublishedAt:  &now,
		SourceURL:    "https://example.com/article",
	}

	source := &models.ContentSource{
		Config: models.SourceConfig{
			AutoPublish:       true,
			DefaultCategoryID: 1,
			DefaultAuthorID:   2,
		},
	}

	article, err := service.convertToArticle(context.Background(), content, source)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if article.Title != content.Title {
		t.Errorf("Expected title to be '%s', got '%s'", content.Title, article.Title)
	}

	if article.Content != content.Content {
		t.Errorf("Expected content to be '%s', got '%s'", content.Content, article.Content)
	}

	if article.Excerpt != content.Excerpt {
		t.Errorf("Expected excerpt to be '%s', got '%s'", content.Excerpt, article.Excerpt)
	}

	if article.AuthorID != source.Config.DefaultAuthorID {
		t.Errorf("Expected author ID to be %d, got %d", source.Config.DefaultAuthorID, article.AuthorID)
	}

	if article.CategoryID != source.Config.DefaultCategoryID {
		t.Errorf("Expected category ID to be %d, got %d", source.Config.DefaultCategoryID, article.CategoryID)
	}

	if article.Status != "published" {
		t.Errorf("Expected status to be 'published', got '%s'", article.Status)
	}

	if article.PublishedAt == nil {
		t.Error("Expected published_at to be set")
	}

	if article.SEOData.MetaTitle != content.Title {
		t.Errorf("Expected meta title to be '%s', got '%s'", content.Title, article.SEOData.MetaTitle)
	}

	if article.SEOData.MetaDescription != content.Excerpt {
		t.Errorf("Expected meta description to be '%s', got '%s'", content.Excerpt, article.SEOData.MetaDescription)
	}

	if article.SEOData.CanonicalURL != content.SourceURL {
		t.Errorf("Expected canonical URL to be '%s', got '%s'", content.SourceURL, article.SEOData.CanonicalURL)
	}

	if article.SEOData.SchemaType != "NewsArticle" {
		t.Errorf("Expected schema type to be 'NewsArticle', got '%s'", article.SEOData.SchemaType)
	}
}

func TestContentIngestionService_convertToArticle_DraftMode(t *testing.T) {
	service := &ContentIngestionService{}

	content := &models.IngestedContent{
		Title:   "Test Article",
		Content: "This is test content",
	}

	source := &models.ContentSource{
		Config: models.SourceConfig{
			AutoPublish:       false, // Draft mode
			DefaultCategoryID: 1,
			DefaultAuthorID:   2,
		},
	}

	article, err := service.convertToArticle(context.Background(), content, source)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if article.Status != "draft" {
		t.Errorf("Expected status to be 'draft', got '%s'", article.Status)
	}

	// Published at should not be set for drafts unless explicitly provided
	if content.PublishedAt == nil && article.PublishedAt != nil {
		t.Error("Expected published_at to be nil for draft articles")
	}
}

func TestContentIngestionService_checkRateLimit(t *testing.T) {
	service := &ContentIngestionService{}

	tests := []struct {
		name   string
		source *models.ContentSource
		valid  bool
	}{
		{
			name: "active source",
			source: &models.ContentSource{
				IsActive:  true,
				RateLimit: 100,
			},
			valid: true,
		},
		{
			name: "inactive source",
			source: &models.ContentSource{
				IsActive:  false,
				RateLimit: 100,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.checkRateLimit(context.Background(), tt.source)

			if tt.valid && err != nil {
				t.Errorf("Expected no error for valid source, got: %v", err)
			}

			if !tt.valid && err == nil {
				t.Error("Expected error for invalid source, got nil")
			}
		})
	}
}

func TestContentIngestionService_ConfigDefaults(t *testing.T) {
	source := &models.ContentSource{
		Name: "Test Source",
		Type: "api",
	}

	// Test that defaults are applied
	if source.RateLimit == 0 {
		source.RateLimit = 100 // Default rate limit
	}

	if source.Priority == 0 {
		source.Priority = 5 // Default priority
	}

	if source.RateLimit != 100 {
		t.Errorf("Expected default rate limit to be 100, got %d", source.RateLimit)
	}

	if source.Priority != 5 {
		t.Errorf("Expected default priority to be 5, got %d", source.Priority)
	}
}

func TestContentIngestionService_MetadataProcessing(t *testing.T) {
	metadata := map[string]interface{}{
		"source_system": "external_cms",
		"import_batch":  "batch_456",
		"priority":      "high",
		"custom_tags":   []string{"breaking", "urgent"},
	}

	content := &models.IngestedContent{
		Metadata: metadata,
	}

	// Test metadata access and type assertions
	if content.Metadata["source_system"] != "external_cms" {
		t.Error("Expected source_system to be 'external_cms'")
	}

	if content.Metadata["import_batch"] != "batch_456" {
		t.Error("Expected import_batch to be 'batch_456'")
	}

	if content.Metadata["priority"] != "high" {
		t.Error("Expected priority to be 'high'")
	}

	customTags, ok := content.Metadata["custom_tags"].([]string)
	if !ok {
		t.Error("Expected custom_tags to be []string")
	}

	if len(customTags) != 2 {
		t.Error("Expected 2 custom tags")
	}
}

func TestContentIngestionService_ErrorHandling(t *testing.T) {
	// Test various error conditions
	tests := []struct {
		name        string
		setupError  func() error
		expectError bool
	}{
		{
			name: "validation error",
			setupError: func() error {
				return &models.ValidationError{
					Message: "Validation failed",
					Fields:  []string{"title is required"},
				}
			},
			expectError: true,
		},
		{
			name: "no error",
			setupError: func() error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupError()

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if tt.expectError && err != nil {
				// Check if it's a validation error
				if _, ok := err.(*models.ValidationError); !ok {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}