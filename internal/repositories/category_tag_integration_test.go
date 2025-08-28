package repositories

import (
	"testing"

	"high-performance-news-website/internal/models"
)

func TestCategoryTagIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryRepo := NewCategoryRepository(db)
	tagRepo := NewTagRepository(db)
	articleRepo := NewArticleRepository(db) // Assuming this exists

	// Create hierarchical categories
	parentCategory := &models.Category{
		Name:        "Technology",
		Description: "Technology articles",
		SortOrder:   1,
	}
	createdParent, err := categoryRepo.Create(parentCategory)
	if err != nil {
		t.Fatalf("Failed to create parent category: %v", err)
	}

	childCategory := &models.Category{
		Name:        "Programming",
		Description: "Programming articles",
		ParentID:    &createdParent.ID,
		SortOrder:   1,
	}
	createdChild, err := categoryRepo.Create(childCategory)
	if err != nil {
		t.Fatalf("Failed to create child category: %v", err)
	}

	// Create tags with keyword banks
	tags := []models.Tag{
		{
			Name:        "Go Programming",
			Description: "Go language articles",
			Keywords:    []string{"go", "golang", "programming"},
			Color:       "#00ADD8",
		},
		{
			Name:        "Web Development",
			Description: "Web development articles",
			Keywords:    []string{"web", "development", "frontend", "backend"},
			Color:       "#FF5722",
		},
		{
			Name:        "Database",
			Description: "Database related articles",
			Keywords:    []string{"database", "sql", "postgresql", "mysql"},
			Color:       "#4CAF50",
		},
	}

	var createdTags []models.Tag
	for i := range tags {
		created, err := tagRepo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create tag %s: %v", tags[i].Name, err)
		}
		createdTags = append(createdTags, *created)
	}

	// Test keyword uniqueness across tags
	conflictingTag := &models.Tag{
		Name:     "Another Go Tag",
		Keywords: []string{"go", "another"}, // "go" conflicts with first tag
		Color:    "#FF0000",
	}
	_, err = tagRepo.Create(conflictingTag)
	if err == nil {
		t.Errorf("Expected error for conflicting keywords, got nil")
	}

	// Create an article in the child category
	article := &models.Article{
		Title:      "Building Web APIs with Go",
		Slug:       "building-web-apis-with-go",
		Content:    "This article covers building web APIs using Go programming language...",
		Excerpt:    "Learn how to build web APIs with Go",
		AuthorID:   1, // Assuming user with ID 1 exists
		CategoryID: createdChild.ID,
		Status:     "published",
	}
	
	// For this test, we'll simulate article creation
	// In a real scenario, you'd use the article repository
	articleID := uint64(1) // Simulated article ID

	// Associate multiple tags with the article
	tagIDs := []uint64{createdTags[0].ID, createdTags[1].ID}
	err = tagRepo.BulkAddTagsToArticle(articleID, tagIDs)
	if err != nil {
		t.Errorf("Failed to add tags to article: %v", err)
	}

	// Verify tag associations
	articleTags, err := tagRepo.GetTagsByArticle(articleID)
	if err != nil {
		t.Errorf("Failed to get tags by article: %v", err)
	}

	if len(articleTags) != 2 {
		t.Errorf("Expected 2 tags for article, got %d", len(articleTags))
	}

	// Test category hierarchy with article counts
	rootCategories, err := categoryRepo.GetRootCategories(true)
	if err != nil {
		t.Errorf("Failed to get root categories: %v", err)
	}

	// Find our technology category
	var techCategory *models.Category
	for i := range rootCategories {
		if rootCategories[i].ID == createdParent.ID {
			techCategory = &rootCategories[i]
			break
		}
	}

	if techCategory == nil {
		t.Errorf("Technology category not found in root categories")
	} else {
		if len(techCategory.Children) != 1 {
			t.Errorf("Expected 1 child category, got %d", len(techCategory.Children))
		}

		if len(techCategory.Children) > 0 {
			programmingCategory := techCategory.Children[0]
			if programmingCategory.ID != createdChild.ID {
				t.Errorf("Expected child category ID %d, got %d", createdChild.ID, programmingCategory.ID)
			}
		}
	}

	// Test category path
	path, err := categoryRepo.GetCategoryPath(createdChild.ID)
	if err != nil {
		t.Errorf("Failed to get category path: %v", err)
	}

	expectedPath := []string{"Technology", "Programming"}
	if len(path) != len(expectedPath) {
		t.Errorf("Expected path length %d, got %d", len(expectedPath), len(path))
	}

	for i, expected := range expectedPath {
		if i >= len(path) || path[i] != expected {
			t.Errorf("Expected path[%d] = %s, got %s", i, expected, path[i])
		}
	}

	// Test keyword bank functionality
	allKeywords, err := tagRepo.GetAllKeywords()
	if err != nil {
		t.Errorf("Failed to get all keywords: %v", err)
	}

	expectedKeywords := []string{"go", "golang", "programming", "web", "development", "frontend", "backend", "database", "sql", "postgresql", "mysql"}
	for _, expected := range expectedKeywords {
		if _, exists := allKeywords[expected]; !exists {
			t.Errorf("Expected keyword '%s' not found in keyword bank", expected)
		}
	}

	// Test popular tags (should include our created tags)
	popularTags, err := tagRepo.GetPopularTags(10)
	if err != nil {
		t.Errorf("Failed to get popular tags: %v", err)
	}

	if len(popularTags) < len(createdTags) {
		t.Errorf("Expected at least %d popular tags, got %d", len(createdTags), len(popularTags))
	}
}

func TestBulkOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryRepo := NewCategoryRepository(db)
	tagRepo := NewTagRepository(db)

	// Test bulk category creation with hierarchy
	categories := []models.Category{
		{
			Name:        "Technology",
			Description: "Technology articles",
			SortOrder:   1,
		},
		{
			Name:        "Science",
			Description: "Science articles",
			SortOrder:   2,
		},
		{
			Name:        "Sports",
			Description: "Sports articles",
			SortOrder:   3,
		},
	}

	createdCategories, err := categoryRepo.BulkCreate(categories)
	if err != nil {
		t.Errorf("Failed to bulk create categories: %v", err)
	}

	if len(createdCategories) != len(categories) {
		t.Errorf("Expected %d created categories, got %d", len(categories), len(createdCategories))
	}

	// Test bulk tag creation with keyword validation
	tags := []models.Tag{
		{
			Name:        "Go Programming",
			Description: "Go language",
			Keywords:    []string{"go", "golang", "programming"},
			Color:       "#00ADD8",
		},
		{
			Name:        "Python Programming",
			Description: "Python language",
			Keywords:    []string{"python", "django", "flask"},
			Color:       "#3776AB",
		},
		{
			Name:        "JavaScript",
			Description: "JavaScript language",
			Keywords:    []string{"javascript", "js", "node", "react"},
			Color:       "#F7DF1E",
		},
	}

	createdTags, err := tagRepo.BulkCreate(tags)
	if err != nil {
		t.Errorf("Failed to bulk create tags: %v", err)
	}

	if len(createdTags) != len(tags) {
		t.Errorf("Expected %d created tags, got %d", len(tags), len(createdTags))
	}

	// Verify all keywords are unique across tags
	allKeywords, err := tagRepo.GetAllKeywords()
	if err != nil {
		t.Errorf("Failed to get all keywords: %v", err)
	}

	expectedKeywordCount := 0
	for _, tag := range tags {
		expectedKeywordCount += len(tag.Keywords)
	}

	if len(allKeywords) < expectedKeywordCount {
		t.Errorf("Expected at least %d keywords, got %d", expectedKeywordCount, len(allKeywords))
	}

	// Test bulk operations with conflicts
	conflictingTags := []models.Tag{
		{
			Name:     "Another Go Tag",
			Keywords: []string{"go", "new"}, // "go" conflicts
			Color:    "#FF0000",
		},
		{
			Name:     "Valid Tag",
			Keywords: []string{"valid", "unique"},
			Color:    "#00FF00",
		},
	}

	_, err = tagRepo.BulkCreate(conflictingTags)
	if err == nil {
		t.Errorf("Expected error for bulk create with conflicting keywords, got nil")
	}
}

func TestCategoryHierarchyValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryRepo := NewCategoryRepository(db)

	// Create parent category
	parent := &models.Category{
		Name:        "Technology",
		Description: "Tech articles",
		SortOrder:   1,
	}
	createdParent, err := categoryRepo.Create(parent)
	if err != nil {
		t.Fatalf("Failed to create parent category: %v", err)
	}

	// Create child category
	child := &models.Category{
		Name:        "Programming",
		Description: "Programming articles",
		ParentID:    &createdParent.ID,
		SortOrder:   1,
	}
	createdChild, err := categoryRepo.Create(child)
	if err != nil {
		t.Fatalf("Failed to create child category: %v", err)
	}

	// Try to create circular reference (child as parent of its parent)
	createdParent.ParentID = &createdChild.ID
	err = categoryRepo.Update(createdParent)
	if err == nil {
		t.Errorf("Expected error for circular reference, got nil")
	}

	// Try to delete parent category with children
	err = categoryRepo.Delete(createdParent.ID)
	if err == nil {
		t.Errorf("Expected error when deleting category with children, got nil")
	}

	// Delete child first, then parent should work
	err = categoryRepo.Delete(createdChild.ID)
	if err != nil {
		t.Errorf("Failed to delete child category: %v", err)
	}

	err = categoryRepo.Delete(createdParent.ID)
	if err != nil {
		t.Errorf("Failed to delete parent category after removing children: %v", err)
	}
}

func TestSearchFunctionality(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryRepo := NewCategoryRepository(db)
	tagRepo := NewTagRepository(db)

	// Create test data
	categories := []models.Category{
		{Name: "Technology", Description: "Tech articles and news"},
		{Name: "Programming", Description: "Programming tutorials"},
		{Name: "Science", Description: "Scientific research and discoveries"},
		{Name: "Tech News", Description: "Latest technology news"},
	}

	for i := range categories {
		_, err := categoryRepo.Create(&categories[i])
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	tags := []models.Tag{
		{Name: "Go Programming", Description: "Go language", Keywords: []string{"go", "golang"}},
		{Name: "Web Development", Description: "Web dev", Keywords: []string{"web", "html", "css"}},
		{Name: "Database", Description: "Database systems", Keywords: []string{"database", "sql"}},
		{Name: "Go Web", Description: "Go web frameworks", Keywords: []string{"gin", "echo"}},
	}

	for i := range tags {
		_, err := tagRepo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create tag: %v", err)
		}
	}

	// Test category search
	techCategories, err := categoryRepo.SearchCategories("tech", 10)
	if err != nil {
		t.Errorf("Failed to search categories: %v", err)
	}

	if len(techCategories) < 2 {
		t.Errorf("Expected at least 2 categories with 'tech', got %d", len(techCategories))
	}

	// Test tag search
	goTags, err := tagRepo.SearchTags("go", 10)
	if err != nil {
		t.Errorf("Failed to search tags: %v", err)
	}

	if len(goTags) < 2 {
		t.Errorf("Expected at least 2 tags with 'go', got %d", len(goTags))
	}

	// Test keyword search in tags
	webTags, err := tagRepo.SearchTags("web", 10)
	if err != nil {
		t.Errorf("Failed to search tags by keyword: %v", err)
	}

	if len(webTags) < 1 {
		t.Errorf("Expected at least 1 tag with 'web' keyword, got %d", len(webTags))
	}
}