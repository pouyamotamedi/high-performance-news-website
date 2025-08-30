package models

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestArticleValidationComprehensive provides comprehensive validation testing
func TestArticleValidationComprehensive(t *testing.T) {
	tests := []struct {
		name        string
		setupArticle func() *Article
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_article_all_fields",
			setupArticle: func() *Article {
				now := time.Now()
				return &Article{
					Title:      "Test Article",
					Content:    "This is test content",
					Excerpt:    "Test excerpt",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
					PublishedAt: &now,
					SEOData: SEOData{
						MetaTitle:       "Test Meta Title",
						MetaDescription: "Test meta description",
						SchemaType:      "NewsArticle",
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty_title",
			setupArticle: func() *Article {
				return &Article{
					Title:      "",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "title is required",
		},
		{
			name: "title_too_long",
			setupArticle: func() *Article {
				return &Article{
					Title:      strings.Repeat("a", 256),
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "title must be less than 255 characters",
		},
		{
			name: "empty_content",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "content is required",
		},
		{
			name: "missing_author_id",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   0,
					CategoryID: 1,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "author_id is required",
		},
		{
			name: "missing_category_id",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 0,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "category_id is required",
		},
		{
			name: "invalid_status",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "invalid_status",
				}
			},
			wantErr:     true,
			errContains: "status must be one of",
		},
		{
			name: "excerpt_too_long",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					Excerpt:    strings.Repeat("a", 501),
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
				}
			},
			wantErr:     true,
			errContains: "excerpt must be less than 500 characters",
		},
		{
			name: "valid_draft_status",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "draft",
				}
			},
			wantErr: false,
		},
		{
			name: "valid_archived_status",
			setupArticle: func() *Article {
				return &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "archived",
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := tt.setupArticle()
			err := ValidateArticle(article)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Validate required fields are set
				assert.NotEmpty(t, article.Title)
				assert.NotEmpty(t, article.Content)
				assert.NotZero(t, article.AuthorID)
				assert.NotZero(t, article.CategoryID)
				assert.NotEmpty(t, article.Status)
			}
		})
	}
}

// TestSEODataValidationComprehensive provides comprehensive SEO data validation testing
func TestSEODataValidationComprehensive(t *testing.T) {
	tests := []struct {
		name        string
		setupSEO    func() *SEOData
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_seo_data",
			setupSEO: func() *SEOData {
				return &SEOData{
					MetaTitle:       "Valid Meta Title",
					MetaDescription: "Valid meta description under 160 characters",
					CanonicalURL:    "https://example.com/article",
					SchemaType:      "NewsArticle",
					Keywords:        []string{"test", "article", "seo"},
				}
			},
			wantErr: false,
		},
		{
			name: "meta_title_too_long",
			setupSEO: func() *SEOData {
				return &SEOData{
					MetaTitle: strings.Repeat("a", 61),
				}
			},
			wantErr:     true,
			errContains: "meta_title must be less than 60 characters",
		},
		{
			name: "meta_description_too_long",
			setupSEO: func() *SEOData {
				return &SEOData{
					MetaDescription: strings.Repeat("a", 161),
				}
			},
			wantErr:     true,
			errContains: "meta_description must be less than 160 characters",
		},
		{
			name: "invalid_canonical_url",
			setupSEO: func() *SEOData {
				return &SEOData{
					CanonicalURL: "not-a-valid-url",
				}
			},
			wantErr:     true,
			errContains: "canonical_url must be a valid URL",
		},
		{
			name: "invalid_schema_type",
			setupSEO: func() *SEOData {
				return &SEOData{
					SchemaType: "InvalidSchemaType",
				}
			},
			wantErr:     true,
			errContains: "schema_type must be one of",
		},
		{
			name: "valid_article_schema",
			setupSEO: func() *SEOData {
				return &SEOData{
					SchemaType: "Article",
				}
			},
			wantErr: false,
		},
		{
			name: "valid_blog_posting_schema",
			setupSEO: func() *SEOData {
				return &SEOData{
					SchemaType: "BlogPosting",
				}
			},
			wantErr: false,
		},
		{
			name: "empty_schema_type_defaults_to_news_article",
			setupSEO: func() *SEOData {
				return &SEOData{}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seo := tt.setupSEO()
			err := ValidateSEOData(seo)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Check default schema type is set
				if seo.SchemaType == "" {
					assert.Equal(t, "NewsArticle", seo.SchemaType)
				}
			}
		})
	}
}

// TestSlugGenerationComprehensive provides comprehensive slug generation testing
func TestSlugGenerationComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple_title",
			title:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "title_with_special_characters",
			title:    "Hello, World! How are you?",
			expected: "hello-world-how-are-you",
		},
		{
			name:     "title_with_numbers",
			title:    "Top 10 Programming Languages 2024",
			expected: "top-10-programming-languages-2024",
		},
		{
			name:     "title_with_multiple_spaces",
			title:    "Hello    World    Test",
			expected: "hello-world-test",
		},
		{
			name:     "title_with_leading_trailing_spaces",
			title:    "  Hello World  ",
			expected: "hello-world",
		},
		{
			name:     "title_with_ampersand",
			title:    "News & Updates",
			expected: "news-updates",
		},
		{
			name:     "title_with_parentheses",
			title:    "Test (Updated Version)",
			expected: "test-updated-version",
		},
		{
			name:     "title_with_quotes",
			title:    `"Breaking News" Today`,
			expected: "breaking-news-today",
		},
		{
			name:     "title_with_unicode_should_be_empty",
			title:    "مقاله تست فارسی",
			expected: "",
		},
		{
			name:     "empty_title",
			title:    "",
			expected: "",
		},
		{
			name:     "only_special_characters",
			title:    "!@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSlug(tt.title)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// BenchmarkArticleValidationComprehensive benchmarks article validation performance
func BenchmarkArticleValidationComprehensive(b *testing.B) {
	now := time.Now()
	article := &Article{
		Title:      "Test Article",
		Content:    "This is test content for benchmarking",
		Excerpt:    "Test excerpt",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		PublishedAt: &now,
		SEOData: SEOData{
			MetaTitle:       "Test Meta Title",
			MetaDescription: "Test meta description",
			SchemaType:      "NewsArticle",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateArticle(article)
	}
}

// BenchmarkSlugGenerationComprehensive benchmarks slug generation performance
func BenchmarkSlugGenerationComprehensive(b *testing.B) {
	title := "This is a Test Article Title with Special Characters & Numbers 123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSlug(title)
	}
}

// TestArticleValidationConcurrency tests validation under concurrent access
func TestArticleValidationConcurrency(t *testing.T) {
	const numGoroutines = 100
	const numValidationsPerGoroutine = 10

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*numValidationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < numValidationsPerGoroutine; j++ {
				now := time.Now()
				article := &Article{
					Title:      "Test Article",
					Content:    "Test content",
					AuthorID:   1,
					CategoryID: 1,
					Status:     "published",
					PublishedAt: &now,
				}
				
				if err := ValidateArticle(article); err != nil {
					errors <- err
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(errors)

	// Check for any errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Concurrent validation failed with %d errors: %v", len(errorList), errorList[0])
	}
}