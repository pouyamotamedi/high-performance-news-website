package models

import (
	"testing"
	"time"
)

// TestModelsIntegration tests that all models work together properly
func TestModelsIntegration(t *testing.T) {
	// Create a user
	user := &User{
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      RoleReporter,
		FirstName: "Test",
		LastName:  "User",
	}

	if err := ValidateUser(user); err != nil {
		t.Fatalf("User validation failed: %v", err)
	}

	// Create a category
	category := &Category{
		Name:        "Technology",
		Description: "Technology related articles",
	}

	if err := ValidateCategory(category); err != nil {
		t.Fatalf("Category validation failed: %v", err)
	}

	// Create tags
	tag1 := &Tag{
		Name:     "Programming",
		Keywords: []string{"programming", "code", "development"},
		Color:    "#FF0000",
	}

	tag2 := &Tag{
		Name:     "Go Language",
		Keywords: []string{"golang", "go programming", "go lang"},
		Color:    "#00ADD8",
	}

	if err := ValidateTag(tag1); err != nil {
		t.Fatalf("Tag1 validation failed: %v", err)
	}

	if err := ValidateTag(tag2); err != nil {
		t.Fatalf("Tag2 validation failed: %v", err)
	}

	// Create an article
	now := time.Now()
	article := &Article{
		Title:       "Introduction to Go Programming",
		Content:     "This is a comprehensive guide to Go programming language...",
		Excerpt:     "Learn the basics of Go programming",
		AuthorID:    1, // Would be user.ID in real scenario
		CategoryID:  1, // Would be category.ID in real scenario
		Tags:        []Tag{*tag1, *tag2},
		Status:      "published",
		PublishedAt: &now,
		SEOData: SEOData{
			MetaTitle:       "Go Programming Guide",
			MetaDescription: "Complete guide to learning Go programming language",
			Keywords:        []string{"go", "programming", "tutorial"},
			SchemaType:      "NewsArticle",
		},
	}

	if err := ValidateArticle(article); err != nil {
		t.Fatalf("Article validation failed: %v", err)
	}

	// Test slug generation
	if article.Slug == "" {
		t.Error("Article slug should be generated")
	}

	expectedSlug := "introduction-to-go-programming"
	if article.Slug != expectedSlug {
		t.Errorf("Generated slug = %v, want %v", article.Slug, expectedSlug)
	}

	// Test PrepareForDB methods
	user.PrepareForDB()
	category.PrepareForDB()
	tag1.PrepareForDB()
	tag2.PrepareForDB()
	article.PrepareForDB()

	// Verify all fields are properly prepared
	if user.Username != "testuser" {
		t.Errorf("User username not properly prepared: %v", user.Username)
	}

	if category.Slug == "" {
		t.Error("Category slug should be generated")
	}

	if tag1.Slug == "" {
		t.Error("Tag1 slug should be generated")
	}

	if article.Status != "published" {
		t.Errorf("Article status = %v, want published", article.Status)
	}

	if article.SEOData.SchemaType != "NewsArticle" {
		t.Errorf("Article schema type = %v, want NewsArticle", article.SEOData.SchemaType)
	}
}

// TestSlugUniqueness tests that slug generation produces unique results for different titles
func TestSlugUniqueness(t *testing.T) {
	titles := []string{
		"Hello World",
		"Hello, World!",
		"Hello - World",
		"Hello_World",
		"Hello   World",
	}

	slugs := make(map[string]bool)
	
	for _, title := range titles {
		slug := GenerateSlug(title)
		if slugs[slug] {
			t.Errorf("Duplicate slug generated: %v for title: %v", slug, title)
		}
		slugs[slug] = true
		
		// Verify slug is valid
		if !IsValidSlug(slug) {
			t.Errorf("Invalid slug generated: %v for title: %v", slug, title)
		}
	}
}

// TestErrorTypes tests that all custom error types work correctly
func TestErrorTypes(t *testing.T) {
	// Test ValidationError
	validationErr := NewValidationError("Test validation error", "field1", "field2")
	if !validationErr.HasFields() {
		t.Error("ValidationError should have fields")
	}

	validationErr.AddField("field3")
	if len(validationErr.Fields) != 3 {
		t.Errorf("ValidationError fields count = %v, want 3", len(validationErr.Fields))
	}

	// Test NotFoundError
	notFoundErr := NewNotFoundError("Article", "123")
	expectedMsg := "Article with ID 123 not found"
	if notFoundErr.Error() != expectedMsg {
		t.Errorf("NotFoundError message = %v, want %v", notFoundErr.Error(), expectedMsg)
	}

	// Test DuplicateError
	duplicateErr := NewDuplicateError("User", "email", "test@example.com")
	expectedMsg = "User with email 'test@example.com' already exists"
	if duplicateErr.Error() != expectedMsg {
		t.Errorf("DuplicateError message = %v, want %v", duplicateErr.Error(), expectedMsg)
	}

	// Test UnauthorizedError
	unauthorizedErr := NewUnauthorizedError("delete", "article")
	expectedMsg = "unauthorized to delete article"
	if unauthorizedErr.Error() != expectedMsg {
		t.Errorf("UnauthorizedError message = %v, want %v", unauthorizedErr.Error(), expectedMsg)
	}
}

// TestUserPermissions tests the user permission system
func TestUserPermissions(t *testing.T) {
	admin := &User{Role: RoleAdmin}
	editor := &User{Role: RoleEditor}
	reporter := &User{Role: RoleReporter}
	contributor := &User{Role: RoleContributor}

	// Test admin permissions
	if !admin.HasPermission("manage_system") {
		t.Error("Admin should have manage_system permission")
	}

	// Test editor permissions
	if !editor.HasPermission("publish") {
		t.Error("Editor should have publish permission")
	}
	if editor.HasPermission("manage_system") {
		t.Error("Editor should not have manage_system permission")
	}

	// Test reporter permissions
	if !reporter.HasPermission("create") {
		t.Error("Reporter should have create permission")
	}
	if reporter.HasPermission("delete") {
		t.Error("Reporter should not have delete permission")
	}

	// Test contributor permissions
	if !contributor.HasPermission("read") {
		t.Error("Contributor should have read permission")
	}
	if contributor.HasPermission("update") {
		t.Error("Contributor should not have update permission")
	}
}

// TestCategoryHierarchy tests category hierarchy functionality
func TestCategoryHierarchy(t *testing.T) {
	// Create root category
	tech := &Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	// Create child category
	programming := &Category{
		ID:       2,
		Name:     "Programming",
		Slug:     "programming",
		ParentID: &tech.ID,
		Parent:   tech,
	}

	// Create grandchild category
	golang := &Category{
		ID:       3,
		Name:     "Go Language",
		Slug:     "go-language",
		ParentID: &programming.ID,
		Parent:   programming,
	}

	// Test hierarchy methods
	if !tech.IsRoot() {
		t.Error("Tech category should be root")
	}

	if programming.IsRoot() {
		t.Error("Programming category should not be root")
	}

	if tech.GetDepth() != 0 {
		t.Errorf("Tech category depth = %v, want 0", tech.GetDepth())
	}

	if programming.GetDepth() != 1 {
		t.Errorf("Programming category depth = %v, want 1", programming.GetDepth())
	}

	if golang.GetDepth() != 2 {
		t.Errorf("Go category depth = %v, want 2", golang.GetDepth())
	}

	expectedPath := "Technology/Programming/Go Language"
	if golang.GetPath() != expectedPath {
		t.Errorf("Go category path = %v, want %v", golang.GetPath(), expectedPath)
	}

	expectedSlugPath := "technology/programming/go-language"
	if golang.GetSlugPath() != expectedSlugPath {
		t.Errorf("Go category slug path = %v, want %v", golang.GetSlugPath(), expectedSlugPath)
	}
}

// TestTagKeywordManagement tests tag keyword functionality
func TestTagKeywordManagement(t *testing.T) {
	tag := &Tag{
		Name:     "Programming",
		Keywords: []string{"programming", "code"},
	}

	// Test adding keywords
	tag.AddKeyword("development")
	if !tag.HasKeyword("development") {
		t.Error("Tag should have 'development' keyword after adding")
	}

	// Test adding duplicate keyword (should not add)
	initialCount := len(tag.Keywords)
	tag.AddKeyword("Programming") // Different case
	if len(tag.Keywords) != initialCount {
		t.Error("Tag should not add duplicate keyword")
	}

	// Test removing keyword
	tag.RemoveKeyword("code")
	if tag.HasKeyword("code") {
		t.Error("Tag should not have 'code' keyword after removing")
	}

	// Test longest keyword
	tag.Keywords = []string{"go", "programming", "web development"}
	longest := tag.GetLongestKeyword()
	if longest != "web development" {
		t.Errorf("Longest keyword = %v, want 'web development'", longest)
	}
}

// BenchmarkValidation benchmarks the validation functions
func BenchmarkValidation(b *testing.B) {
	article := &Article{
		Title:      "Test Article",
		Content:    "This is test content for benchmarking validation performance",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
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

// BenchmarkSlugGeneration benchmarks slug generation
func BenchmarkSlugGeneration(b *testing.B) {
	title := "This is a Test Article Title with Multiple Words and Numbers 123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSlug(title)
	}
}