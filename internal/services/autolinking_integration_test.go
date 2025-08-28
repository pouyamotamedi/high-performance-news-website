package services

import (
	"context"
	"testing"

	"high-performance-news-website/internal/models"
)

func TestAutoLinkingIntegration(t *testing.T) {
	// Setup mock repository
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"artificial intelligence", "machine learning"},
			},
		},
	}

	// Create auto-linking service
	autoLinkService := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	// Load keywords
	err := autoLinkService.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	// Test article processing
	article := &models.Article{
		Title:       "AI Technology",
		Content:     "Artificial intelligence is transforming machine learning applications.",
		AutoLinking: true,
	}

	processedContent, err := autoLinkService.ProcessHTMLContent(ctx, article)
	if err != nil {
		t.Fatalf("Failed to process article: %v", err)
	}

	// Verify links were created
	expectedLinks := []string{
		`<a href="/tags/technology" title="">Artificial intelligence</a>`,
		`<a href="/tags/technology" title="">machine learning</a>`,
	}

	for _, expectedLink := range expectedLinks {
		if !contains(processedContent, expectedLink) {
			t.Errorf("Expected processed content to contain: %s\nGot: %s", expectedLink, processedContent)
		}
	}
}

func TestAutoLinkingServiceSetup(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"AI", "machine learning"},
			},
			{
				ID:       2,
				Name:     "Programming",
				Slug:     "programming",
				Keywords: []string{"Python", "JavaScript"},
			},
		},
	}

	service, err := SetupAutoLinkingService(mockRepo)
	if err != nil {
		t.Fatalf("Failed to setup auto-linking service: %v", err)
	}

	// Verify service is properly initialized
	stats := service.GetTrieStats()
	if stats["total_keywords"] != 4 {
		t.Errorf("Expected 4 keywords, got %d", stats["total_keywords"])
	}

	if stats["total_nodes"] == 0 {
		t.Error("Expected non-zero nodes")
	}
}

func TestKeywordConflictDetection(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"AI", "machine learning"},
			},
			{
				ID:       2,
				Name:     "Artificial Intelligence",
				Slug:     "ai",
				Keywords: []string{"AI", "neural networks"}, // Conflict: "AI" appears in both
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	conflicts, err := service.ValidateKeywordConflicts(ctx)
	if err != nil {
		t.Fatalf("Failed to validate conflicts: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}

	expectedConflict := "Keyword 'ai' is used in tags: Technology, Artificial Intelligence"
	if len(conflicts) > 0 && conflicts[0] != expectedConflict {
		t.Errorf("Expected conflict: %s\nGot: %s", expectedConflict, conflicts[0])
	}
}

func TestRefreshKeywords(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"AI"},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	// Initial load
	err := service.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	initialStats := service.GetTrieStats()

	// Update mock data
	mockRepo.tags[0].Keywords = append(mockRepo.tags[0].Keywords, "machine learning")

	// Refresh keywords
	err = RefreshAutoLinkingKeywords(service)
	if err != nil {
		t.Fatalf("Failed to refresh keywords: %v", err)
	}

	newStats := service.GetTrieStats()

	// Should have more keywords now
	if newStats["total_keywords"] <= initialStats["total_keywords"] {
		t.Errorf("Expected more keywords after refresh, got %d vs %d", 
			newStats["total_keywords"], initialStats["total_keywords"])
	}
}