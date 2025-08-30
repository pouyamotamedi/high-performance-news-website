package testhelpers

import (
	"fmt"
	"time"
)

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	counter int64
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{counter: 1}
}

// Article represents a test article (avoiding import cycle)
type Article struct {
	ID          uint64
	Title       string
	Slug        string
	Content     string
	Excerpt     string
	AuthorID    uint64
	CategoryID  uint64
	Status      string
	PublishedAt *time.Time
	SEOData     SEOData
}

// SEOData represents SEO metadata for testing
type SEOData struct {
	MetaTitle       string
	MetaDescription string
	SchemaType      string
	Keywords        []string
	CanonicalURL    string
}

// User represents a test user
type User struct {
	ID        uint64
	Username  string
	Email     string
	FirstName string
	LastName  string
	Role      string
	IsActive  bool
}

// Category represents a test category
type Category struct {
	ID          uint64
	Name        string
	Slug        string
	Description string
	SortOrder   int
}

// Tag represents a test tag
type Tag struct {
	ID          uint64
	Name        string
	Slug        string
	Description string
}

// GenerateTestArticle creates a test article with realistic data
func (g *TestDataGenerator) GenerateTestArticle() *Article {
	g.counter++
	now := time.Now()
	
	return &Article{
		Title:      fmt.Sprintf("Test Article %d", g.counter),
		Slug:       fmt.Sprintf("test-article-%d", g.counter),
		Content:    fmt.Sprintf("This is test content for article %d with sufficient length to test various functionality.", g.counter),
		Excerpt:    fmt.Sprintf("Test excerpt for article %d", g.counter),
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		PublishedAt: &now,
		SEOData: SEOData{
			MetaTitle:       fmt.Sprintf("Test Article %d - Meta Title", g.counter),
			MetaDescription: fmt.Sprintf("Meta description for test article %d", g.counter),
			SchemaType:      "NewsArticle",
		},
	}
}

// GenerateTestUser creates a test user
func (g *TestDataGenerator) GenerateTestUser() *User {
	g.counter++
	
	return &User{
		Username:  fmt.Sprintf("testuser%d", g.counter),
		Email:     fmt.Sprintf("test%d@example.com", g.counter),
		FirstName: fmt.Sprintf("Test%d", g.counter),
		LastName:  "User",
		Role:      "author",
		IsActive:  true,
	}
}

// GenerateTestCategory creates a test category
func (g *TestDataGenerator) GenerateTestCategory() *Category {
	g.counter++
	
	return &Category{
		Name:        fmt.Sprintf("Test Category %d", g.counter),
		Slug:        fmt.Sprintf("test-category-%d", g.counter),
		Description: fmt.Sprintf("Description for test category %d", g.counter),
		SortOrder:   int(g.counter),
	}
}

// GenerateTestTag creates a test tag
func (g *TestDataGenerator) GenerateTestTag() *Tag {
	g.counter++
	
	return &Tag{
		Name:        fmt.Sprintf("test-tag-%d", g.counter),
		Slug:        fmt.Sprintf("test-tag-%d", g.counter),
		Description: fmt.Sprintf("Description for test tag %d", g.counter),
	}
}