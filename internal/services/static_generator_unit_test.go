package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTextDirection(t *testing.T) {
	tests := []struct {
		language string
		expected string
		name     string
	}{
		{"fa", "rtl", "Persian should be RTL"},
		{"ar", "rtl", "Arabic should be RTL"},
		{"he", "rtl", "Hebrew should be RTL"},
		{"ur", "rtl", "Urdu should be RTL"},
		{"en", "ltr", "English should be LTR"},
		{"fr", "ltr", "French should be LTR"},
		{"de", "ltr", "German should be LTR"},
		{"unknown", "ltr", "Unknown language should default to LTR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTextDirection(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticGeneratorConfig(t *testing.T) {
	config := StaticGeneratorConfig{
		OutputPath:   "/tmp/output",
		TemplatesDir: "/tmp/templates",
		BaseURL:      "https://example.com",
	}

	assert.Equal(t, "/tmp/output", config.OutputPath)
	assert.Equal(t, "/tmp/templates", config.TemplatesDir)
	assert.Equal(t, "https://example.com", config.BaseURL)
}

func TestPageData(t *testing.T) {
	pageData := PageData{
		Title:        "Test Title",
		Description:  "Test Description",
		Keywords:     []string{"test", "keywords"},
		CanonicalURL: "https://example.com/test",
		Language:     "en",
		Direction:    "ltr",
		BaseURL:      "https://example.com",
	}

	assert.Equal(t, "Test Title", pageData.Title)
	assert.Equal(t, "Test Description", pageData.Description)
	assert.Equal(t, []string{"test", "keywords"}, pageData.Keywords)
	assert.Equal(t, "https://example.com/test", pageData.CanonicalURL)
	assert.Equal(t, "en", pageData.Language)
	assert.Equal(t, "ltr", pageData.Direction)
	assert.Equal(t, "https://example.com", pageData.BaseURL)
}

func TestPaginationData(t *testing.T) {
	pagination := PaginationData{
		CurrentPage:  2,
		TotalPages:   5,
		HasPrevious:  true,
		HasNext:      true,
		PreviousPage: 1,
		NextPage:     3,
		TotalItems:   100,
	}

	assert.Equal(t, 2, pagination.CurrentPage)
	assert.Equal(t, 5, pagination.TotalPages)
	assert.True(t, pagination.HasPrevious)
	assert.True(t, pagination.HasNext)
	assert.Equal(t, 1, pagination.PreviousPage)
	assert.Equal(t, 3, pagination.NextPage)
	assert.Equal(t, 100, pagination.TotalItems)
}