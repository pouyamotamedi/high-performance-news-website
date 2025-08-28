package repositories

import (
	"strings"
	"testing"

	"high-performance-news-website/internal/models"
)

func TestTagRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	tests := []struct {
		name    string
		tag     models.Tag
		wantErr bool
	}{
		{
			name: "valid tag",
			tag: models.Tag{
				Name:        "Go Programming",
				Description: "Articles about Go programming language",
				Keywords:    []string{"go", "golang", "programming"},
				Color:       "#00ADD8",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			tag: models.Tag{
				Name:        "",
				Description: "Invalid tag",
			},
			wantErr: true,
		},
		{
			name: "invalid color",
			tag: models.Tag{
				Name:  "Test",
				Color: "invalid-color",
			},
			wantErr: true,
		},
		{
			name: "duplicate keywords",
			tag: models.Tag{
				Name:     "Test",
				Keywords: []string{"go", "Go", "GO"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.Create(&tt.tag)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Create() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Create() unexpected error: %v", err)
				return
			}

			if result.ID == 0 {
				t.Errorf("Create() expected ID to be set")
			}

			if result.CreatedAt.IsZero() {
				t.Errorf("Create() expected CreatedAt to be set")
			}

			if result.Slug == "" {
				t.Errorf("Create() expected Slug to be generated")
			}

			if result.Color == "" {
				t.Errorf("Create() expected Color to be set")
			}
		})
	}
}

func TestTagRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tag
	tag := &models.Tag{
		Name:        "Go Programming",
		Description: "Go language articles",
		Keywords:    []string{"go", "golang"},
		Color:       "#00ADD8",
	}
	created, err := repo.Create(tag)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	tests := []struct {
		name    string
		id      uint64
		wantErr bool
	}{
		{
			name:    "existing tag",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existent tag",
			id:      99999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetByID(tt.id)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetByID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetByID() unexpected error: %v", err)
				return
			}

			if result.ID != tt.id {
				t.Errorf("GetByID() expected ID %d, got %d", tt.id, result.ID)
			}

			if len(result.Keywords) != len(created.Keywords) {
				t.Errorf("GetByID() expected %d keywords, got %d", len(created.Keywords), len(result.Keywords))
			}

			if result.ArticleCount < 0 {
				t.Errorf("GetByID() expected ArticleCount to be loaded")
			}
		})
	}
}

func TestTagRepository_KeywordUniqueness(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create first tag with keywords
	tag1 := &models.Tag{
		Name:     "Go Programming",
		Keywords: []string{"go", "golang", "programming"},
		Color:    "#00ADD8",
	}
	_, err := repo.Create(tag1)
	if err != nil {
		t.Fatalf("Failed to create first tag: %v", err)
	}

	// Try to create second tag with conflicting keywords
	tag2 := &models.Tag{
		Name:     "Python Programming",
		Keywords: []string{"python", "go"}, // "go" conflicts
		Color:    "#3776AB",
	}
	_, err = repo.Create(tag2)
	if err == nil {
		t.Errorf("Create() expected error for conflicting keywords, got nil")
	}

	// Create tag with non-conflicting keywords
	tag3 := &models.Tag{
		Name:     "Python Programming",
		Keywords: []string{"python", "django", "flask"},
		Color:    "#3776AB",
	}
	_, err = repo.Create(tag3)
	if err != nil {
		t.Errorf("Create() unexpected error for non-conflicting keywords: %v", err)
	}
}

func TestTagRepository_BulkCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	tags := []models.Tag{
		{
			Name:     "Go Programming",
			Keywords: []string{"go", "golang"},
			Color:    "#00ADD8",
		},
		{
			Name:     "Python Programming",
			Keywords: []string{"python", "django"},
			Color:    "#3776AB",
		},
		{
			Name:     "JavaScript",
			Keywords: []string{"javascript", "js", "node"},
			Color:    "#F7DF1E",
		},
	}

	results, err := repo.BulkCreate(tags)
	if err != nil {
		t.Errorf("BulkCreate() unexpected error: %v", err)
		return
	}

	if len(results) != len(tags) {
		t.Errorf("BulkCreate() expected %d results, got %d", len(tags), len(results))
	}

	for i, result := range results {
		if result.ID == 0 {
			t.Errorf("BulkCreate() result[%d] expected ID to be set", i)
		}
		if result.CreatedAt.IsZero() {
			t.Errorf("BulkCreate() result[%d] expected CreatedAt to be set", i)
		}
		if result.Slug == "" {
			t.Errorf("BulkCreate() result[%d] expected Slug to be generated", i)
		}
	}
}

func TestTagRepository_BulkCreateWithConflicts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create tags with conflicting keywords within the batch
	tags := []models.Tag{
		{
			Name:     "Go Programming",
			Keywords: []string{"go", "golang"},
			Color:    "#00ADD8",
		},
		{
			Name:     "Go Web Development",
			Keywords: []string{"go", "web"}, // "go" conflicts with first tag
			Color:    "#00ADD8",
		},
	}

	_, err := repo.BulkCreate(tags)
	if err == nil {
		t.Errorf("BulkCreate() expected error for conflicting keywords within batch, got nil")
	}
}

func TestTagRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tag
	tag := &models.Tag{
		Name:        "Go Programming",
		Description: "Go language articles",
		Keywords:    []string{"go", "golang"},
		Color:       "#00ADD8",
	}
	created, err := repo.Create(tag)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	// Update the tag
	created.Name = "Updated Go Programming"
	created.Description = "Updated description"
	created.Keywords = []string{"go", "golang", "programming"}
	created.Color = "#FF0000"

	err = repo.Update(created)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
		return
	}

	// Verify the update
	updated, err := repo.GetByID(created.ID)
	if err != nil {
		t.Errorf("Failed to get updated tag: %v", err)
		return
	}

	if updated.Name != "Updated Go Programming" {
		t.Errorf("Update() expected name 'Updated Go Programming', got '%s'", updated.Name)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Update() expected description 'Updated description', got '%s'", updated.Description)
	}

	if len(updated.Keywords) != 3 {
		t.Errorf("Update() expected 3 keywords, got %d", len(updated.Keywords))
	}

	if updated.Color != "#FF0000" {
		t.Errorf("Update() expected color '#FF0000', got '%s'", updated.Color)
	}
}

func TestTagRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tag
	tag := &models.Tag{
		Name:     "Go Programming",
		Keywords: []string{"go", "golang"},
		Color:    "#00ADD8",
	}
	created, err := repo.Create(tag)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	// Delete the tag
	err = repo.Delete(created.ID)
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
		return
	}

	// Verify deletion
	_, err = repo.GetByID(created.ID)
	if err == nil {
		t.Errorf("Delete() expected tag to be deleted")
	}

	// Test deleting non-existent tag
	err = repo.Delete(99999)
	if err == nil {
		t.Errorf("Delete() expected error for non-existent tag")
	}
}

func TestTagRepository_ArticleTagOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)
	articleRepo := NewArticleRepository(db) // Assuming this exists

	// Create test tag
	tag := &models.Tag{
		Name:     "Go Programming",
		Keywords: []string{"go", "golang"},
		Color:    "#00ADD8",
	}
	createdTag, err := repo.Create(tag)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	// For this test, we'll assume we have an article with ID 1
	// In a real test, you'd create the article first
	articleID := uint64(1)

	// Test AddTagToArticle
	err = repo.AddTagToArticle(articleID, createdTag.ID)
	if err != nil {
		t.Errorf("AddTagToArticle() unexpected error: %v", err)
	}

	// Test GetTagsByArticle
	tags, err := repo.GetTagsByArticle(articleID)
	if err != nil {
		t.Errorf("GetTagsByArticle() unexpected error: %v", err)
		return
	}

	if len(tags) != 1 {
		t.Errorf("GetTagsByArticle() expected 1 tag, got %d", len(tags))
	}

	if len(tags) > 0 && tags[0].ID != createdTag.ID {
		t.Errorf("GetTagsByArticle() expected tag ID %d, got %d", createdTag.ID, tags[0].ID)
	}

	// Test RemoveTagFromArticle
	err = repo.RemoveTagFromArticle(articleID, createdTag.ID)
	if err != nil {
		t.Errorf("RemoveTagFromArticle() unexpected error: %v", err)
	}

	// Verify removal
	tags, err = repo.GetTagsByArticle(articleID)
	if err != nil {
		t.Errorf("GetTagsByArticle() unexpected error after removal: %v", err)
		return
	}

	if len(tags) != 0 {
		t.Errorf("GetTagsByArticle() expected 0 tags after removal, got %d", len(tags))
	}
}

func TestTagRepository_BulkAddTagsToArticle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tags
	tags := []models.Tag{
		{Name: "Go", Keywords: []string{"go"}, Color: "#00ADD8"},
		{Name: "Programming", Keywords: []string{"programming"}, Color: "#FF0000"},
		{Name: "Web", Keywords: []string{"web"}, Color: "#00FF00"},
	}

	var tagIDs []uint64
	for i := range tags {
		created, err := repo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create test tag: %v", err)
		}
		tagIDs = append(tagIDs, created.ID)
	}

	// Test bulk add
	articleID := uint64(1)
	err := repo.BulkAddTagsToArticle(articleID, tagIDs)
	if err != nil {
		t.Errorf("BulkAddTagsToArticle() unexpected error: %v", err)
		return
	}

	// Verify all tags were added
	articleTags, err := repo.GetTagsByArticle(articleID)
	if err != nil {
		t.Errorf("GetTagsByArticle() unexpected error: %v", err)
		return
	}

	if len(articleTags) != len(tagIDs) {
		t.Errorf("BulkAddTagsToArticle() expected %d tags, got %d", len(tagIDs), len(articleTags))
	}
}

func TestTagRepository_SearchTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tags
	tags := []models.Tag{
		{Name: "Go Programming", Description: "Go language", Keywords: []string{"go", "golang"}},
		{Name: "Python", Description: "Python programming", Keywords: []string{"python"}},
		{Name: "JavaScript", Description: "JS development", Keywords: []string{"javascript", "js"}},
	}

	for i := range tags {
		_, err := repo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create test tag: %v", err)
		}
	}

	// Search for "go"
	results, err := repo.SearchTags("go", 10)
	if err != nil {
		t.Errorf("SearchTags() unexpected error: %v", err)
		return
	}

	if len(results) < 1 {
		t.Errorf("SearchTags() expected at least 1 result, got %d", len(results))
	}

	// Verify results contain "go" in name, description, or keywords
	for _, result := range results {
		found := false
		if strings.Contains(strings.ToLower(result.Name), "go") ||
		   strings.Contains(strings.ToLower(result.Description), "go") {
			found = true
		}
		
		for _, keyword := range result.Keywords {
			if strings.Contains(strings.ToLower(keyword), "go") {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("SearchTags() result '%s' does not contain 'go'", result.Name)
		}
	}
}

func TestTagRepository_GetAllKeywords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tags with keywords
	tags := []models.Tag{
		{Name: "Go", Keywords: []string{"go", "golang"}},
		{Name: "Python", Keywords: []string{"python", "django"}},
	}

	for i := range tags {
		_, err := repo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create test tag: %v", err)
		}
	}

	// Get all keywords
	keywords, err := repo.GetAllKeywords()
	if err != nil {
		t.Errorf("GetAllKeywords() unexpected error: %v", err)
		return
	}

	expectedKeywords := []string{"go", "golang", "python", "django"}
	if len(keywords) < len(expectedKeywords) {
		t.Errorf("GetAllKeywords() expected at least %d keywords, got %d", len(expectedKeywords), len(keywords))
	}

	// Check that expected keywords exist
	for _, expected := range expectedKeywords {
		if _, exists := keywords[expected]; !exists {
			t.Errorf("GetAllKeywords() missing expected keyword: %s", expected)
		}
	}
}

func TestTagRepository_GetPopularTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTagRepository(db)

	// Create test tags
	tags := []models.Tag{
		{Name: "Go", Keywords: []string{"go"}},
		{Name: "Python", Keywords: []string{"python"}},
		{Name: "JavaScript", Keywords: []string{"javascript"}},
	}

	for i := range tags {
		_, err := repo.Create(&tags[i])
		if err != nil {
			t.Fatalf("Failed to create test tag: %v", err)
		}
	}

	// Get popular tags
	popular, err := repo.GetPopularTags(5)
	if err != nil {
		t.Errorf("GetPopularTags() unexpected error: %v", err)
		return
	}

	if len(popular) == 0 {
		t.Errorf("GetPopularTags() expected at least some tags")
	}

	// Verify tags are ordered by article count (descending)
	for i := 1; i < len(popular); i++ {
		if popular[i-1].ArticleCount < popular[i].ArticleCount {
			t.Errorf("GetPopularTags() tags not ordered by article count")
		}
	}
}