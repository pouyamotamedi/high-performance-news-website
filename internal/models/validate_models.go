package models

import (
	"fmt"
	"time"
)

// ValidateAllModels performs a comprehensive validation of all model types
// This function is used to ensure all models work correctly together
func ValidateAllModels() error {
	fmt.Println("Validating all models...")

	// Test User model
	user := &User{
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      RoleReporter,
		FirstName: "Test",
		LastName:  "User",
		Bio:       "Test user bio",
	}

	if err := ValidateUser(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}
	user.PrepareForDB()
	fmt.Println("✓ User model validation passed")

	// Test Category model
	category := &Category{
		Name:        "Technology",
		Description: "Technology related articles",
	}

	if err := ValidateCategory(category); err != nil {
		return fmt.Errorf("category validation failed: %w", err)
	}
	category.PrepareForDB()
	fmt.Println("✓ Category model validation passed")

	// Test Tag model
	tag := &Tag{
		Name:        "Programming",
		Description: "Programming related content",
		Keywords:    []string{"programming", "code", "development"},
		Color:       "#FF0000",
	}

	if err := ValidateTag(tag); err != nil {
		return fmt.Errorf("tag validation failed: %w", err)
	}
	tag.PrepareForDB()
	fmt.Println("✓ Tag model validation passed")

	// Test Article model
	now := time.Now()
	article := &Article{
		Title:       "Test Article",
		Content:     "This is test content for validation",
		Excerpt:     "Test excerpt",
		AuthorID:    1,
		CategoryID:  1,
		Status:      "published",
		PublishedAt: &now,
		MetaTitle:       "Test Meta Title",
		MetaDescription: "Test meta description",
		SchemaType:      "NewsArticle",
	}

	if err := ValidateArticle(article); err != nil {
		return fmt.Errorf("article validation failed: %w", err)
	}
	article.PrepareForDB()
	fmt.Println("✓ Article model validation passed")

	// Test SEO Data - now using individual fields in Article struct
	fmt.Println("✓ SEO Data validation (now handled in Article validation)")

	// Test error types
	validationErr := NewValidationError("Test error", "field1", "field2")
	if !validationErr.HasFields() {
		return fmt.Errorf("validation error should have fields")
	}

	notFoundErr := NewNotFoundError("Article", "123")
	if notFoundErr.Resource != "Article" {
		return fmt.Errorf("not found error resource mismatch")
	}

	duplicateErr := NewDuplicateError("User", "email", "test@example.com")
	if duplicateErr.Field != "email" {
		return fmt.Errorf("duplicate error field mismatch")
	}

	unauthorizedErr := NewUnauthorizedError("delete", "article")
	if unauthorizedErr.Action != "delete" {
		return fmt.Errorf("unauthorized error action mismatch")
	}
	fmt.Println("✓ Error types validation passed")

	// Test utility functions
	slug := GenerateSlug("Test Article Title")
	if !IsValidSlug(slug) {
		return fmt.Errorf("generated slug is invalid: %s", slug)
	}

	if !IsValidEmail("test@example.com") {
		return fmt.Errorf("valid email rejected")
	}

	if !IsValidURL("https://example.com") {
		return fmt.Errorf("valid URL rejected")
	}

	if !IsValidHexColor("#FF0000") {
		return fmt.Errorf("valid hex color rejected")
	}

	if !IsValidUsername("testuser123") {
		return fmt.Errorf("valid username rejected")
	}

	if !IsValidRole(RoleAdmin) {
		return fmt.Errorf("valid role rejected")
	}

	if !IsValidKeyword("programming") {
		return fmt.Errorf("valid keyword rejected")
	}
	fmt.Println("✓ Utility functions validation passed")

	fmt.Println("✅ All models validation completed successfully!")
	return nil
}

// TestModelInteractions tests how models work together
func TestModelInteractions() error {
	fmt.Println("Testing model interactions...")

	// Create a complete content scenario
	admin := &User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		Role:     RoleAdmin,
	}

	editor := &User{
		ID:       2,
		Username: "editor",
		Email:    "editor@example.com",
		Role:     RoleEditor,
	}

	reporter := &User{
		ID:       3,
		Username: "reporter",
		Email:    "reporter@example.com",
		Role:     RoleReporter,
	}

	// Test user management permissions
	if !admin.CanManageUser(editor) {
		return fmt.Errorf("admin should be able to manage editor")
	}

	if !editor.CanManageUser(reporter) {
		return fmt.Errorf("editor should be able to manage reporter")
	}

	if reporter.CanManageUser(editor) {
		return fmt.Errorf("reporter should not be able to manage editor")
	}

	// Create category hierarchy
	tech := &Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	programming := &Category{
		ID:       2,
		Name:     "Programming",
		Slug:     "programming",
		ParentID: &tech.ID,
		Parent:   tech,
	}

	if programming.GetDepth() != 1 {
		return fmt.Errorf("programming category should have depth 1")
	}

	if programming.GetPath() != "Technology/Programming" {
		return fmt.Errorf("incorrect category path: %s", programming.GetPath())
	}

	// Create tags with keywords
	goTag := &Tag{
		Name:     "Go Language",
		Keywords: []string{"golang", "go programming", "go"},
		Color:    "#00ADD8",
	}

	webTag := &Tag{
		Name:     "Web Development",
		Keywords: []string{"web development", "frontend", "backend"},
		Color:    "#FF6B35",
	}

	// Test keyword management
	goTag.AddKeyword("go lang")
	if !goTag.HasKeyword("go lang") {
		return fmt.Errorf("tag should have added keyword")
	}

	longest := webTag.GetLongestKeyword()
	if longest != "web development" {
		return fmt.Errorf("longest keyword should be 'web development', got: %s", longest)
	}

	// Create article with all relationships
	now := time.Now()
	article := &Article{
		Title:       "Introduction to Go Web Development",
		Content:     "This article covers Go web development...",
		Excerpt:     "Learn Go web development basics",
		AuthorID:    reporter.ID,
		CategoryID:  programming.ID,
		Tags:        []Tag{*goTag, *webTag},
		Status:      "published",
		PublishedAt: &now,
		MetaTitle:       "Go Web Development Guide",
		MetaDescription: "Complete guide to Go web development",
		SchemaType:      "NewsArticle",
	}

	if err := ValidateArticle(article); err != nil {
		return fmt.Errorf("article validation failed: %w", err)
	}

	// Test slug generation
	expectedSlug := "introduction-to-go-web-development"
	if article.Slug != expectedSlug {
		return fmt.Errorf("expected slug %s, got %s", expectedSlug, article.Slug)
	}

	fmt.Println("✅ Model interactions test completed successfully!")
	return nil
}