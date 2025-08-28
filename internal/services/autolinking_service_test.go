package services

import (
	"context"
	"testing"

	"high-performance-news-website/internal/models"
)

// MockTagRepository implements TagRepository interface for testing
type MockTagRepository struct {
	tags []models.Tag
}

func (m *MockTagRepository) GetAllWithKeywords(ctx context.Context) ([]models.Tag, error) {
	return m.tags, nil
}

func TestTrie_Insert(t *testing.T) {
	trie := NewTrie()
	tag := &models.Tag{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	trie.Insert("artificial intelligence", tag)
	trie.Insert("machine learning", tag)
	trie.Insert("AI", tag)

	// Test that keywords were inserted correctly
	matches := trie.FindLongestMatches("artificial intelligence and machine learning")
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	// Test longest match priority
	matches = trie.FindLongestMatches("AI and artificial intelligence")
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	// Find the "artificial intelligence" match
	var aiMatch *KeywordMatch
	for _, match := range matches {
		if match.Keyword == "artificial intelligence" {
			aiMatch = &match
			break
		}
	}

	if aiMatch == nil {
		t.Error("Expected to find 'artificial intelligence' match")
	} else if aiMatch.Length != len("artificial intelligence") {
		t.Errorf("Expected match length %d, got %d", len("artificial intelligence"), aiMatch.Length)
	}
}

func TestTrie_FindLongestMatches(t *testing.T) {
	trie := NewTrie()
	
	techTag := &models.Tag{ID: 1, Name: "Technology", Slug: "technology"}
	aiTag := &models.Tag{ID: 2, Name: "AI", Slug: "ai"}
	
	// Insert overlapping keywords to test longest match priority
	trie.Insert("machine", techTag)
	trie.Insert("machine learning", aiTag)
	trie.Insert("learning", techTag)

	tests := []struct {
		name     string
		text     string
		expected int
		keywords []string
	}{
		{
			name:     "Longest match wins",
			text:     "machine learning is important",
			expected: 1,
			keywords: []string{"machine learning"},
		},
		{
			name:     "Multiple non-overlapping matches",
			text:     "machine and learning are different",
			expected: 2,
			keywords: []string{"machine", "learning"},
		},
		{
			name:     "Case insensitive matching",
			text:     "Machine Learning is important",
			expected: 1,
			keywords: []string{"machine learning"},
		},
		{
			name:     "Word boundary respect",
			text:     "machinelearning is one word",
			expected: 0,
			keywords: []string{},
		},
		{
			name:     "Partial word no match",
			text:     "supermachine and machinegun",
			expected: 0, // Should not match "machine" inside other words
			keywords: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := trie.FindLongestMatches(tt.text)
			
			if len(matches) != tt.expected {
				t.Errorf("Expected %d matches, got %d", tt.expected, len(matches))
			}

			for i, expectedKeyword := range tt.keywords {
				if i >= len(matches) {
					t.Errorf("Expected keyword '%s' not found", expectedKeyword)
					continue
				}
				if matches[i].Keyword != expectedKeyword {
					t.Errorf("Expected keyword '%s', got '%s'", expectedKeyword, matches[i].Keyword)
				}
			}
		})
	}
}

func TestAutoLinkingService_ProcessArticleLinks(t *testing.T) {
	// Setup mock repository with test tags
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"artificial intelligence", "machine learning", "AI"},
			},
			{
				ID:       2,
				Name:     "Programming",
				Slug:     "programming",
				Keywords: []string{"Python", "JavaScript", "Go"},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	// Load keywords
	err := service.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	tests := []struct {
		name           string
		article        *models.Article
		expectedLinks  int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name: "Basic auto-linking",
			article: &models.Article{
				Content:     "Artificial intelligence and machine learning are transforming technology.",
				AutoLinking: true,
			},
			expectedLinks: 2,
			shouldContain: []string{
				`<a href="/tags/technology" title="">Artificial intelligence</a>`,
				`<a href="/tags/technology" title="">machine learning</a>`,
			},
		},
		{
			name: "Auto-linking disabled",
			article: &models.Article{
				Content:     "Artificial intelligence and machine learning are transforming technology.",
				AutoLinking: false,
			},
			expectedLinks: 0,
			shouldNotContain: []string{
				`<a href="/tags/technology"`,
			},
		},
		{
			name: "One link per keyword per article",
			article: &models.Article{
				Content:     "Python is great. Python is awesome. Python programming is fun.",
				AutoLinking: true,
			},
			expectedLinks: 1,
			shouldContain: []string{
				`<a href="/tags/programming" title="">Python</a>`,
			},
		},
		{
			name: "Longest match priority",
			article: &models.Article{
				Content:     "Machine learning and AI are related to artificial intelligence.",
				AutoLinking: true,
			},
			expectedLinks: 3,
			shouldContain: []string{
				`<a href="/tags/technology" title="">Machine learning</a>`,
				`<a href="/tags/technology" title="">AI</a>`,
				`<a href="/tags/technology" title="">artificial intelligence</a>`,
			},
		},
		{
			name: "Case preservation",
			article: &models.Article{
				Content:     "PYTHON and JavaScript are programming languages.",
				AutoLinking: true,
			},
			expectedLinks: 2,
			shouldContain: []string{
				`<a href="/tags/programming" title="">PYTHON</a>`,
				`<a href="/tags/programming" title="">JavaScript</a>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ProcessArticleLinks(ctx, tt.article)
			if err != nil {
				t.Fatalf("ProcessArticleLinks failed: %v", err)
			}



			// Count actual links
			linkCount := 0
			for _, shouldContain := range tt.shouldContain {
				if contains(result, shouldContain) {
					linkCount++
				} else {
					t.Errorf("Expected result to contain: %s", shouldContain)
				}
			}

			if linkCount != tt.expectedLinks {
				t.Errorf("Expected %d links, got %d", tt.expectedLinks, linkCount)
			}

			// Check that unwanted content is not present
			for _, shouldNotContain := range tt.shouldNotContain {
				if contains(result, shouldNotContain) {
					t.Errorf("Expected result to NOT contain: %s", shouldNotContain)
				}
			}
		})
	}
}

func TestAutoLinkingService_ProcessArticleLinksWithExclusions(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"artificial intelligence", "machine learning", "AI"},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	err := service.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	article := &models.Article{
		Content:     "Artificial intelligence and machine learning are related to AI.",
		AutoLinking: true,
	}

	// Test with exclusions
	result, err := service.ProcessArticleLinksWithExclusions(ctx, article, []string{"AI"})
	if err != nil {
		t.Fatalf("ProcessArticleLinksWithExclusions failed: %v", err)
	}

	// Should contain links for non-excluded keywords
	if !contains(result, `<a href="/tags/technology" title="">Artificial intelligence</a>`) {
		t.Error("Expected link for 'artificial intelligence'")
	}
	if !contains(result, `<a href="/tags/technology" title="">machine learning</a>`) {
		t.Error("Expected link for 'machine learning'")
	}

	// Should NOT contain link for excluded keyword
	if contains(result, `<a href="/tags/technology" title="">AI</a>`) {
		t.Error("Expected NO link for excluded keyword 'AI'")
	}
}

func TestAutoLinkingService_ProcessHTMLContent(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"Python", "JavaScript"},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	err := service.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:    "Preserve existing links",
			content: `<p>Python is great. <a href="/existing">JavaScript</a> is also good.</p>`,
			expected: `<p><a href="/tags/technology" title="">Python</a> is great. <a href="/existing">JavaScript</a> is also good.</p>`,
		},
		{
			name:    "Preserve HTML tags",
			content: `<h1>Python Tutorial</h1><p>Learn JavaScript programming.</p>`,
			expected: `<h1><a href="/tags/technology" title="">Python</a> Tutorial</h1><p>Learn <a href="/tags/technology" title="">JavaScript</a> programming.</p>`,
		},
		{
			name:    "Don't link inside HTML attributes",
			content: `<img src="python.jpg" alt="Python logo">`,
			expected: `<img src="python.jpg" alt="Python logo">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := &models.Article{
				Content:     tt.content,
				AutoLinking: true,
			}

			result, err := service.ProcessHTMLContent(ctx, article)
			if err != nil {
				t.Fatalf("ProcessHTMLContent failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestAutoLinkingService_ValidateKeywordConflicts(t *testing.T) {
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
				Keywords: []string{"AI", "neural networks"}, // Conflict: "AI" is in both tags
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	conflicts, err := service.ValidateKeywordConflicts(ctx)
	if err != nil {
		t.Fatalf("ValidateKeywordConflicts failed: %v", err)
	}

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}

	expectedConflict := "Keyword 'ai' is used in tags: Technology, Artificial Intelligence"
	if len(conflicts) > 0 && conflicts[0] != expectedConflict {
		t.Errorf("Expected conflict: %s\nGot: %s", expectedConflict, conflicts[0])
	}
}

func TestAutoLinkingService_GetTrieStats(t *testing.T) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:       1,
				Name:     "Technology",
				Slug:     "technology",
				Keywords: []string{"AI", "machine learning", "Python"},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()

	// Test empty trie
	stats := service.GetTrieStats()
	if stats["total_nodes"] != 0 || stats["total_keywords"] != 0 {
		t.Error("Expected empty trie stats")
	}

	// Load keywords and test again
	err := service.LoadKeywords(ctx)
	if err != nil {
		t.Fatalf("Failed to load keywords: %v", err)
	}

	stats = service.GetTrieStats()
	if stats["total_keywords"] != 3 {
		t.Errorf("Expected 3 keywords, got %d", stats["total_keywords"])
	}
	if stats["total_nodes"] <= 3 {
		t.Errorf("Expected more nodes than keywords, got %d nodes", stats["total_nodes"])
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'5', true},
		{'\'', true},
		{'-', true},
		{' ', false},
		{'.', false},
		{'!', false},
		{'@', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isWordChar(tt.char)
			if result != tt.expected {
				t.Errorf("isWordChar(%c) = %v, expected %v", tt.char, result, tt.expected)
			}
		})
	}
}

func TestKeywordMatch_EdgeCases(t *testing.T) {
	trie := NewTrie()
	tag := &models.Tag{ID: 1, Name: "Test", Slug: "test"}

	// Test edge cases
	trie.Insert("test", tag)
	trie.Insert("testing", tag)

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
		},
		{
			name:     "Only whitespace",
			text:     "   \n\t  ",
			expected: 0,
		},
		{
			name:     "Keyword at start",
			text:     "test is good",
			expected: 1,
		},
		{
			name:     "Keyword at end",
			text:     "this is a test",
			expected: 1,
		},
		{
			name:     "Keyword with punctuation",
			text:     "test, testing, and more",
			expected: 2,
		},
		{
			name:     "Overlapping keywords - longest wins",
			text:     "testing is important",
			expected: 1, // Should match "testing", not "test"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := trie.FindLongestMatches(tt.text)
			if len(matches) != tt.expected {
				t.Errorf("Expected %d matches, got %d", tt.expected, len(matches))
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests for performance
func BenchmarkTrie_Insert(b *testing.B) {
	trie := NewTrie()
	tag := &models.Tag{ID: 1, Name: "Test", Slug: "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Insert("test keyword", tag)
	}
}

func BenchmarkTrie_FindLongestMatches(b *testing.B) {
	trie := NewTrie()
	tag := &models.Tag{ID: 1, Name: "Test", Slug: "test"}
	
	// Setup trie with many keywords
	keywords := []string{
		"artificial intelligence", "machine learning", "deep learning",
		"neural networks", "computer vision", "natural language processing",
		"data science", "big data", "cloud computing", "cybersecurity",
	}
	
	for _, keyword := range keywords {
		trie.Insert(keyword, tag)
	}
	
	text := "Artificial intelligence and machine learning are transforming technology through deep learning and neural networks."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.FindLongestMatches(text)
	}
}

func BenchmarkAutoLinkingService_ProcessArticleLinks(b *testing.B) {
	mockRepo := &MockTagRepository{
		tags: []models.Tag{
			{
				ID:   1,
				Name: "Technology",
				Slug: "technology",
				Keywords: []string{
					"artificial intelligence", "machine learning", "deep learning",
					"neural networks", "computer vision", "natural language processing",
				},
			},
		},
	}

	service := NewAutoLinkingService(mockRepo)
	ctx := context.Background()
	service.LoadKeywords(ctx)

	article := &models.Article{
		Content: `
			Artificial intelligence and machine learning are revolutionizing technology.
			Deep learning and neural networks enable computer vision and natural language processing.
			These technologies are transforming how we interact with data and systems.
		`,
		AutoLinking: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ProcessArticleLinks(ctx, article)
	}
}