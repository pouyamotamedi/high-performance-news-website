package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func TestStaticGenerator_NewStaticGenerator(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))

	// Create a simple template
	simpleTemplate := `<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1></body></html>`
	err = os.WriteFile(filepath.Join(templatesDir, "test.html"), []byte(simpleTemplate), 0644)
	require.NoError(t, err)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	// Simple cache implementation for testing
	cacheService := &struct {
		Get           func(ctx context.Context, key string) ([]byte, error)
		Set           func(ctx context.Context, key string, value []byte, ttl time.Duration) error
		Delete        func(ctx context.Context, key string) error
		DeletePattern func(ctx context.Context, pattern string) error
		Exists        func(ctx context.Context, key string) (bool, error)
		Close         func() error
		Health        func(ctx context.Context) error
	}{
		Get:           func(ctx context.Context, key string) ([]byte, error) { return nil, nil },
		Set:           func(ctx context.Context, key string, value []byte, ttl time.Duration) error { return nil },
		Delete:        func(ctx context.Context, key string) error { return nil },
		DeletePattern: func(ctx context.Context, pattern string) error { return nil },
		Exists:        func(ctx context.Context, key string) (bool, error) { return false, nil },
		Close:         func() error { return nil },
		Health:        func(ctx context.Context) error { return nil },
	}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.NotNil(t, sg)
	assert.NotNil(t, sg.templates)
	assert.Equal(t, outputDir, sg.outputPath)
	assert.Equal(t, "https://example.com", sg.baseURL)
	assert.DirExists(t, outputDir) // Should be created automatically
}

func TestStaticGenerator_TextDirection(t *testing.T) {
	tests := []struct {
		language string
		expected string
	}{
		{"fa", "rtl"},
		{"ar", "rtl"},
		{"he", "rtl"},
		{"ur", "rtl"},
		{"en", "ltr"},
		{"fr", "ltr"},
		{"de", "ltr"},
		{"unknown", "ltr"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := getTextDirection(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticGenerator_ConfigValidation(t *testing.T) {
	config := StaticGeneratorConfig{
		OutputPath:   "/tmp/output",
		TemplatesDir: "/tmp/templates",
		BaseURL:      "https://example.com",
	}

	assert.Equal(t, "/tmp/output", config.OutputPath)
	assert.Equal(t, "/tmp/templates", config.TemplatesDir)
	assert.Equal(t, "https://example.com", config.BaseURL)
}

func TestStaticGenerator_DataStructures(t *testing.T) {
	// Test PageData
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

	// Test PaginationData
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
}
