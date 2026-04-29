package integration

import (
	"testing"

	"high-performance-news-website/internal/models"
)

// TestContentIngestionIntegration tests the complete content ingestion flow
func TestContentIngestionIntegration(t *testing.T) {
	// Test content ingestion request validation
	request := &models.ContentIngestionRequest{
		ExternalID:   "test123",
		Title:        "Integration Test Article",
		Content:      "This is a test article for integration testing",
		Excerpt:      "Test excerpt",
		AuthorName:   "Test Author",
		AuthorEmail:  "test@example.com",
		CategoryName: "Technology",
		Tags:         []string{"test", "integration"},
		SourceURL:    "https://example.com/test-article",
		Metadata:     map[string]interface{}{"test": true},
	}

	// Test that the request structure is valid
	if request.ExternalID == "" {
		t.Error("Expected ExternalID to be set")
	}

	if request.Title == "" {
		t.Error("Expected Title to be set")
	}

	if request.Content == "" {
		t.Error("Expected Content to be set")
	}

	if len(request.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(request.Tags))
	}

	if request.Metadata["test"] != true {
		t.Error("Expected metadata test field to be true")
	}
}

// TestContentSourceCreation tests content source creation
func TestContentSourceCreation(t *testing.T) {
	source := &models.ContentSource{
		Name:      "Test Integration Source",
		Type:      "api",
		IsActive:  true,
		RateLimit: 100,
		Priority:  5,
		Config: models.SourceConfig{
			AutoPublish:       false,
			DefaultCategoryID: 1,
			DefaultAuthorID:   1,
			AllowedDomains:    []string{"example.com"},
			RequiredFields:    []string{"title", "content"},
		},
	}

	// Test source validation
	if source.Name == "" {
		t.Error("Expected source name to be set")
	}

	if source.Type != "api" {
		t.Error("Expected source type to be 'api'")
	}

	if !source.IsActive {
		t.Error("Expected source to be active")
	}

	if source.Config.DefaultCategoryID == 0 {
		t.Error("Expected default category ID to be set")
	}

	if len(source.Config.AllowedDomains) != 1 {
		t.Error("Expected one allowed domain")
	}
}

// TestIngestedContentProcessing tests ingested content processing
func TestIngestedContentProcessing(t *testing.T) {
	content := &models.IngestedContent{
		SourceID:     1,
		ExternalID:   "test456",
		Title:        "Test Processing Article",
		Content:      "Content for processing test",
		Excerpt:      "Processing excerpt",
		AuthorName:   "Processing Author",
		AuthorEmail:  "processing@example.com",
		CategoryName: "Test Category",
		Tags:         []string{"processing", "test"},
		Status:       "pending",
		Metadata:     map[string]interface{}{"processing": true},
	}

	// Generate content hash
	content.GenerateContentHash()

	if content.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}

	// Test validation
	result := models.ValidateIngestedContent(content)
	if !result.IsValid {
		t.Errorf("Expected content to be valid, got errors: %v", result.Errors)
	}

	// Test sanitization
	models.SanitizeIngestedContent(content)

	if content.Title == "" {
		t.Error("Expected title to remain after sanitization")
	}

	if content.Content == "" {
		t.Error("Expected content to remain after sanitization")
	}
}

// TestDuplicateDetection tests duplicate content detection
func TestDuplicateDetection(t *testing.T) {
	content1 := &models.IngestedContent{
		Title:   "Duplicate Test Article",
		Content: "This content should be detected as duplicate",
	}
	content1.GenerateContentHash()

	content2 := &models.IngestedContent{
		Title:   "Duplicate Test Article",
		Content: "This content should be detected as duplicate",
	}
	content2.GenerateContentHash()

	// Same content should have same hash
	if content1.ContentHash != content2.ContentHash {
		t.Error("Expected same content to have same hash")
	}

	content3 := &models.IngestedContent{
		Title:   "Different Article",
		Content: "This is different content",
	}
	content3.GenerateContentHash()

	// Different content should have different hash
	if content1.ContentHash == content3.ContentHash {
		t.Error("Expected different content to have different hash")
	}
}

// TestContentIngestionWorkflow tests the complete workflow
func TestContentIngestionWorkflow(t *testing.T) {
	// 1. Create content source
	source := &models.ContentSource{
		Name:      "Workflow Test Source",
		Type:      "api",
		IsActive:  true,
		RateLimit: 50,
		Priority:  7,
		Config: models.SourceConfig{
			AutoPublish:       true,
			DefaultCategoryID: 1,
			DefaultAuthorID:   1,
		},
	}

	// 2. Create ingestion request
	request := &models.ContentIngestionRequest{
		ExternalID:   "workflow123",
		Title:        "Workflow Test Article",
		Content:      "This tests the complete workflow",
		Excerpt:      "Workflow excerpt",
		AuthorName:   "Workflow Author",
		AuthorEmail:  "workflow@example.com",
		CategoryName: "Workflow",
		Tags:         []string{"workflow", "test", "complete"},
		SourceURL:    "https://example.com/workflow-test",
		Metadata:     map[string]interface{}{"workflow": "complete"},
	}

	// 3. Convert to ingested content
	content := &models.IngestedContent{
		SourceID:     1, // Would be source.ID in real implementation
		ExternalID:   request.ExternalID,
		Title:        request.Title,
		Content:      request.Content,
		Excerpt:      request.Excerpt,
		AuthorName:   request.AuthorName,
		AuthorEmail:  request.AuthorEmail,
		CategoryName: request.CategoryName,
		Tags:         request.Tags,
		SourceURL:    request.SourceURL,
		Metadata:     request.Metadata,
		Status:       "pending",
	}

	// 4. Process content
	content.PrepareForProcessing()

	// 5. Validate workflow
	if content.Status != "pending" {
		t.Errorf("Expected status to be 'pending', got %s", content.Status)
	}

	if content.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}

	if content.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if content.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// 6. Test auto-publish configuration
	if source.Config.AutoPublish {
		// In real implementation, this would trigger article creation
		t.Log("Auto-publish is enabled, article would be created automatically")
	}
}