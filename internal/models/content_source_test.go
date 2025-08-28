package models

import (
	"testing"
	"time"
)

func TestGenerateContentHash(t *testing.T) {
	content := &IngestedContent{
		Title:   "Test Article Title",
		Content: "This is the content of the test article.",
	}

	content.GenerateContentHash()

	if content.ContentHash == "" {
		t.Error("Expected content hash to be generated, got empty string")
	}

	// Test that same content generates same hash
	content2 := &IngestedContent{
		Title:   "Test Article Title",
		Content: "This is the content of the test article.",
	}
	content2.GenerateContentHash()

	if content.ContentHash != content2.ContentHash {
		t.Error("Expected same content to generate same hash")
	}

	// Test that different content generates different hash
	content3 := &IngestedContent{
		Title:   "Different Title",
		Content: "This is different content.",
	}
	content3.GenerateContentHash()

	if content.ContentHash == content3.ContentHash {
		t.Error("Expected different content to generate different hash")
	}
}

func TestValidateIngestedContent(t *testing.T) {
	tests := []struct {
		name     string
		content  *IngestedContent
		isValid  bool
		hasError string
	}{
		{
			name: "valid content",
			content: &IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      "Valid Title",
				Content:    "Valid content here",
				AuthorEmail: "test@example.com",
				SourceURL:  "https://example.com/article",
			},
			isValid: true,
		},
		{
			name: "missing title",
			content: &IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Content:    "Valid content here",
			},
			isValid:  false,
			hasError: "title is required",
		},
		{
			name: "missing content",
			content: &IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      "Valid Title",
			},
			isValid:  false,
			hasError: "content is required",
		},
		{
			name: "missing source_id",
			content: &IngestedContent{
				ExternalID: "ext123",
				Title:      "Valid Title",
				Content:    "Valid content here",
			},
			isValid:  false,
			hasError: "source_id is required",
		},
		{
			name: "missing external_id",
			content: &IngestedContent{
				SourceID: 1,
				Title:    "Valid Title",
				Content:  "Valid content here",
			},
			isValid:  false,
			hasError: "external_id is required",
		},
		{
			name: "title too long",
			content: &IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      string(make([]byte, 256)), // 256 characters
				Content:    "Valid content here",
			},
			isValid:  false,
			hasError: "title must be less than 255 characters",
		},
		{
			name: "invalid email",
			content: &IngestedContent{
				SourceID:    1,
				ExternalID:  "ext123",
				Title:       "Valid Title",
				Content:     "Valid content here",
				AuthorEmail: "invalid-email",
			},
			isValid:  false,
			hasError: "author_email must be a valid email address",
		},
		{
			name: "invalid URL",
			content: &IngestedContent{
				SourceID:   1,
				ExternalID: "ext123",
				Title:      "Valid Title",
				Content:    "Valid content here",
				SourceURL:  "not-a-url",
			},
			isValid:  false,
			hasError: "source_url must be a valid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateIngestedContent(tt.content)

			if result.IsValid != tt.isValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.isValid, result.IsValid)
			}

			if !tt.isValid && tt.hasError != "" {
				found := false
				for _, err := range result.Errors {
					if err == tt.hasError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error '%s' not found in errors: %v", tt.hasError, result.Errors)
				}
			}
		})
	}
}

func TestSanitizeIngestedContent(t *testing.T) {
	content := &IngestedContent{
		Title:        "  Test Title with   Extra Spaces  ",
		Content:      "<script>alert('xss')</script><p>Safe content</p>",
		Excerpt:      "  Test excerpt  ",
		AuthorName:   "  John Doe  ",
		AuthorEmail:  "  JOHN@EXAMPLE.COM  ",
		CategoryName: "  Technology  ",
		Tags:         []string{"  tag1  ", "  tag2  "},
	}

	SanitizeIngestedContent(content)

	if content.Title != "Test Title with Extra Spaces" {
		t.Errorf("Expected sanitized title, got: %s", content.Title)
	}

	if content.Content == "<script>alert('xss')</script><p>Safe content</p>" {
		t.Error("Expected content to be sanitized")
	}

	if content.Excerpt != "Test excerpt" {
		t.Errorf("Expected sanitized excerpt, got: %s", content.Excerpt)
	}

	if content.AuthorName != "John Doe" {
		t.Errorf("Expected sanitized author name, got: %s", content.AuthorName)
	}

	if content.AuthorEmail != "john@example.com" {
		t.Errorf("Expected sanitized author email, got: %s", content.AuthorEmail)
	}

	if content.CategoryName != "Technology" {
		t.Errorf("Expected sanitized category name, got: %s", content.CategoryName)
	}

	if len(content.Tags) != 2 || content.Tags[0] != "tag1" || content.Tags[1] != "tag2" {
		t.Errorf("Expected sanitized tags, got: %v", content.Tags)
	}

	if content.ContentHash == "" {
		t.Error("Expected content hash to be generated after sanitization")
	}
}

func TestPrepareForProcessing(t *testing.T) {
	content := &IngestedContent{
		Title:   "Test Title",
		Content: "Test content",
	}

	content.PrepareForProcessing()

	if content.Status != "pending" {
		t.Errorf("Expected status to be 'pending', got: %s", content.Status)
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
}



func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "script tag",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert('xss')&lt;/script&gt;",
		},
		{
			name:     "iframe tag",
			input:    "<iframe src='evil.com'></iframe>",
			expected: "&lt;iframe src='evil.com'&gt;&lt;/iframe&gt;",
		},
		{
			name:     "safe content",
			input:    "<p>This is safe content</p>",
			expected: "<p>This is safe content</p>",
		},
		{
			name:     "mixed content",
			input:    "<p>Safe</p><script>evil()</script><div>More safe</div>",
			expected: "<p>Safe</p>&lt;script&gt;evil()&lt;/script&gt;<div>More safe</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestContentIngestionRequest(t *testing.T) {
	now := time.Now()
	
	request := &ContentIngestionRequest{
		ExternalID:   "ext123",
		Title:        "Test Article",
		Content:      "This is test content",
		Excerpt:      "Test excerpt",
		AuthorName:   "John Doe",
		AuthorEmail:  "john@example.com",
		CategoryName: "Technology",
		Tags:         []string{"tech", "news"},
		PublishedAt:  &now,
		SourceURL:    "https://example.com/article",
		Metadata:     map[string]interface{}{"source": "test"},
	}

	// Test that all fields are properly set
	if request.ExternalID != "ext123" {
		t.Errorf("Expected ExternalID to be 'ext123', got %s", request.ExternalID)
	}

	if request.Title != "Test Article" {
		t.Errorf("Expected Title to be 'Test Article', got %s", request.Title)
	}

	if len(request.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(request.Tags))
	}

	if request.Metadata["source"] != "test" {
		t.Errorf("Expected metadata source to be 'test', got %v", request.Metadata["source"])
	}
}

func TestDuplicateCheckResult(t *testing.T) {
	result := &DuplicateCheckResult{
		IsDuplicate:  true,
		ExistingID:   func() *uint64 { id := uint64(123); return &id }(),
		Similarity:   0.95,
		MatchType:    "hash",
		MatchedField: "content_hash",
	}

	if !result.IsDuplicate {
		t.Error("Expected IsDuplicate to be true")
	}

	if result.ExistingID == nil || *result.ExistingID != 123 {
		t.Error("Expected ExistingID to be 123")
	}

	if result.Similarity != 0.95 {
		t.Errorf("Expected Similarity to be 0.95, got %f", result.Similarity)
	}

	if result.MatchType != "hash" {
		t.Errorf("Expected MatchType to be 'hash', got %s", result.MatchType)
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		IsValid:  false,
		Errors:   []string{"title is required", "content is required"},
		Warnings: []string{"excerpt is recommended"},
	}

	if result.IsValid {
		t.Error("Expected IsValid to be false")
	}

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
}