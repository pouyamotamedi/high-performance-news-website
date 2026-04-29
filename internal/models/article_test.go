package models

import (
	"testing"
)

func TestValidateArticle(t *testing.T) {
	tests := []struct {
		name    string
		article Article
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid article",
			article: Article{
				Title:      "Test Article",
				Content:    "This is test content",
				AuthorID:   1,
				CategoryID: 1,
				Status:     "published",
				SEOData: SEOData{
					MetaTitle:       "Test Meta Title",
					MetaDescription: "Test meta description",
					SchemaType:      "NewsArticle",
				},
			},
			wantErr: false,
		},
		{
			name: "missing title",
			article: Article{
				Content:    "This is test content",
				AuthorID:   1,
				CategoryID: 1,
				Status:     "published",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "title too long",
			article: Article{
				Title:      string(make([]byte, 256)), // 256 characters
				Content:    "This is test content",
				AuthorID:   1,
				CategoryID: 1,
				Status:     "published",
			},
			wantErr: true,
			errMsg:  "title must be less than 255 characters",
		},
		{
			name: "missing content",
			article: Article{
				Title:      "Test Article",
				AuthorID:   1,
				CategoryID: 1,
				Status:     "published",
			},
			wantErr: true,
			errMsg:  "content is required",
		},
		{
			name: "missing author ID",
			article: Article{
				Title:      "Test Article",
				Content:    "This is test content",
				CategoryID: 1,
				Status:     "published",
			},
			wantErr: true,
			errMsg:  "author_id is required",
		},
		{
			name: "missing category ID",
			article: Article{
				Title:    "Test Article",
				Content:  "This is test content",
				AuthorID: 1,
				Status:   "published",
			},
			wantErr: true,
			errMsg:  "category_id is required",
		},
		{
			name: "invalid status",
			article: Article{
				Title:      "Test Article",
				Content:    "This is test content",
				AuthorID:   1,
				CategoryID: 1,
				Status:     "invalid",
			},
			wantErr: true,
			errMsg:  "status must be one of: draft, published, archived",
		},
		{
			name: "excerpt too long",
			article: Article{
				Title:      "Test Article",
				Content:    "This is test content",
				Excerpt:    string(make([]byte, 501)), // 501 characters
				AuthorID:   1,
				CategoryID: 1,
				Status:     "published",
			},
			wantErr: true,
			errMsg:  "excerpt must be less than 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArticle(&tt.article)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateArticle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateArticle() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateSEOData(t *testing.T) {
	tests := []struct {
		name    string
		seo     SEOData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SEO data",
			seo: SEOData{
				MetaTitle:       "Test Title",
				MetaDescription: "Test description",
				CanonicalURL:    "https://example.com/test",
				SchemaType:      "NewsArticle",
			},
			wantErr: false,
		},
		{
			name: "meta title too long",
			seo: SEOData{
				MetaTitle: string(make([]byte, 61)), // 61 characters
			},
			wantErr: true,
			errMsg:  "meta_title must be less than 60 characters",
		},
		{
			name: "meta description too long",
			seo: SEOData{
				MetaDescription: string(make([]byte, 161)), // 161 characters
			},
			wantErr: true,
			errMsg:  "meta_description must be less than 160 characters",
		},
		{
			name: "invalid canonical URL",
			seo: SEOData{
				CanonicalURL: "not-a-url",
			},
			wantErr: true,
			errMsg:  "canonical_url must be a valid URL",
		},
		{
			name: "invalid schema type",
			seo: SEOData{
				SchemaType: "InvalidType",
			},
			wantErr: true,
			errMsg:  "schema_type must be one of: NewsArticle, Article, BlogPosting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSEOData(&tt.seo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSEOData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSEOData() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple title",
			title: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "title with special characters",
			title: "Hello, World! How are you?",
			want:  "hello-world-how-are-you",
		},
		{
			name:  "title with numbers",
			title: "Top 10 Programming Languages 2024",
			want:  "top-10-programming-languages-2024",
		},
		{
			name:  "title with unicode",
			title: "مقاله تست فارسی",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.title)
			if got != tt.want {
				t.Errorf("GenerateSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{
			name: "valid slug",
			slug: "hello-world",
			want: true,
		},
		{
			name: "valid slug with numbers",
			slug: "hello-world-123",
			want: true,
		},
		{
			name: "invalid slug with uppercase",
			slug: "Hello-World",
			want: false,
		},
		{
			name: "invalid slug with special characters",
			slug: "hello_world",
			want: false,
		},
		{
			name: "invalid slug starting with hyphen",
			slug: "-hello-world",
			want: false,
		},
		{
			name: "invalid slug ending with hyphen",
			slug: "hello-world-",
			want: false,
		},
		{
			name: "empty slug",
			slug: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidSlug(tt.slug)
			if got != tt.want {
				t.Errorf("IsValidSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid HTTP URL",
			url:  "http://example.com",
			want: true,
		},
		{
			name: "valid HTTPS URL",
			url:  "https://example.com/path",
			want: true,
		},
		{
			name: "invalid URL without protocol",
			url:  "example.com",
			want: false,
		},
		{
			name: "invalid URL with spaces",
			url:  "https://example .com",
			want: false,
		},
		{
			name: "empty URL",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidURL(tt.url)
			if got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "normal title",
			title: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "title with multiple spaces",
			title: "Hello    World",
			want:  "Hello World",
		},
		{
			name:  "title with leading/trailing spaces",
			title: "  Hello World  ",
			want:  "Hello World",
		},
		{
			name:  "title with control characters",
			title: "Hello\x00\x01World",
			want:  "HelloWorld",
		},
		{
			name:  "title with tabs and newlines",
			title: "Hello\t\nWorld",
			want:  "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTitle(tt.title)
			if got != tt.want {
				t.Errorf("SanitizeTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArticlePrepareForDB(t *testing.T) {
	article := &Article{
		Title:   "  Test Article  ",
		Status:  "",
		SEOData: SEOData{},
	}

	article.PrepareForDB()

	if article.Title != "Test Article" {
		t.Errorf("PrepareForDB() title = %v, want %v", article.Title, "Test Article")
	}
	if article.Status != "draft" {
		t.Errorf("PrepareForDB() status = %v, want %v", article.Status, "draft")
	}
	if article.SEOData.SchemaType != "NewsArticle" {
		t.Errorf("PrepareForDB() schema_type = %v, want %v", article.SEOData.SchemaType, "NewsArticle")
	}
	if article.Slug == "" {
		t.Errorf("PrepareForDB() should generate slug")
	}
}

// contains function is imported from configuration.go	
article.PrepareForDB()

	if strings.TrimSpace(article.Title) != "Test Article" {
		t.Errorf("PrepareForDB() title = %v, want %v", article.Title, "Test Article")
	}
	if article.Status != "draft" {
		t.Errorf("PrepareForDB() status = %v, want %v", article.Status, "draft")
	}
	if article.SEOData.SchemaType != "NewsArticle" {
		t.Errorf("PrepareForDB() schema_type = %v, want %v", article.SEOData.SchemaType, "NewsArticle")
	}
	if article.Slug == "" {
		t.Error("PrepareForDB() should generate slug")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}