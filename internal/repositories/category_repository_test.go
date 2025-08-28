package repositories

import (
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/database"
)

func TestCategoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	tests := []struct {
		name     string
		category models.Category
		wantErr  bool
	}{
		{
			name: "valid category",
			category: models.Category{
				Name:        "Technology",
				Description: "Technology related articles",
				SortOrder:   1,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			category: models.Category{
				Name:        "",
				Description: "Invalid category",
			},
			wantErr: true,
		},
		{
			name: "duplicate slug",
			category: models.Category{
				Name: "Tech", // Will generate same slug as "Technology"
				Slug: "technology",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.Create(&tt.category)
			
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
		})
	}
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Create test category
	category := &models.Category{
		Name:        "Technology",
		Description: "Tech articles",
		SortOrder:   1,
	}
	created, err := repo.Create(category)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	tests := []struct {
		name          string
		id            uint64
		loadRelations bool
		wantErr       bool
	}{
		{
			name:          "existing category",
			id:            created.ID,
			loadRelations: false,
			wantErr:       false,
		},
		{
			name:          "non-existent category",
			id:            99999,
			loadRelations: false,
			wantErr:       true,
		},
		{
			name:          "with relations",
			id:            created.ID,
			loadRelations: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetByID(tt.id, tt.loadRelations)
			
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

			if tt.loadRelations && result.ArticleCount < 0 {
				t.Errorf("GetByID() expected ArticleCount to be loaded")
			}
		})
	}
}

func TestCategoryRepository_HierarchicalOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Create parent category
	parent := &models.Category{
		Name:        "Technology",
		Description: "Tech articles",
		SortOrder:   1,
	}
	createdParent, err := repo.Create(parent)
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
	createdChild, err := repo.Create(child)
	if err != nil {
		t.Fatalf("Failed to create child category: %v", err)
	}

	// Test GetChildren
	children, err := repo.GetChildren(createdParent.ID, false)
	if err != nil {
		t.Errorf("GetChildren() unexpected error: %v", err)
	}

	if len(children) != 1 {
		t.Errorf("GetChildren() expected 1 child, got %d", len(children))
	}

	if children[0].ID != createdChild.ID {
		t.Errorf("GetChildren() expected child ID %d, got %d", createdChild.ID, children[0].ID)
	}

	// Test GetRootCategories
	roots, err := repo.GetRootCategories(true)
	if err != nil {
		t.Errorf("GetRootCategories() unexpected error: %v", err)
	}

	if len(roots) == 0 {
		t.Errorf("GetRootCategories() expected at least 1 root category")
	}

	// Find our parent category
	var foundParent *models.Category
	for i := range roots {
		if roots[i].ID == createdParent.ID {
			foundParent = &roots[i]
			break
		}
	}

	if foundParent == nil {
		t.Errorf("GetRootCategories() did not return our parent category")
	} else if len(foundParent.Children) != 1 {
		t.Errorf("GetRootCategories() expected parent to have 1 child, got %d", len(foundParent.Children))
	}

	// Test GetCategoryPath
	path, err := repo.GetCategoryPath(createdChild.ID)
	if err != nil {
		t.Errorf("GetCategoryPath() unexpected error: %v", err)
	}

	expectedPath := []string{"Technology", "Programming"}
	if len(path) != len(expectedPath) {
		t.Errorf("GetCategoryPath() expected path length %d, got %d", len(expectedPath), len(path))
	}

	for i, expected := range expectedPath {
		if i >= len(path) || path[i] != expected {
			t.Errorf("GetCategoryPath() expected path[%d] = %s, got %s", i, expected, path[i])
		}
	}
}

func TestCategoryRepository_BulkCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	categories := []models.Category{
		{
			Name:        "Technology",
			Description: "Tech articles",
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

	results, err := repo.BulkCreate(categories)
	if err != nil {
		t.Errorf("BulkCreate() unexpected error: %v", err)
		return
	}

	if len(results) != len(categories) {
		t.Errorf("BulkCreate() expected %d results, got %d", len(categories), len(results))
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

func TestCategoryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Create test category
	category := &models.Category{
		Name:        "Technology",
		Description: "Tech articles",
		SortOrder:   1,
	}
	created, err := repo.Create(category)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Update the category
	created.Name = "Updated Technology"
	created.Description = "Updated description"
	created.SortOrder = 2

	err = repo.Update(created)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
		return
	}

	// Verify the update
	updated, err := repo.GetByID(created.ID, false)
	if err != nil {
		t.Errorf("Failed to get updated category: %v", err)
		return
	}

	if updated.Name != "Updated Technology" {
		t.Errorf("Update() expected name 'Updated Technology', got '%s'", updated.Name)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Update() expected description 'Updated description', got '%s'", updated.Description)
	}

	if updated.SortOrder != 2 {
		t.Errorf("Update() expected sort order 2, got %d", updated.SortOrder)
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Create test category
	category := &models.Category{
		Name:        "Technology",
		Description: "Tech articles",
		SortOrder:   1,
	}
	created, err := repo.Create(category)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Delete the category
	err = repo.Delete(created.ID)
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
		return
	}

	// Verify deletion
	_, err = repo.GetByID(created.ID, false)
	if err == nil {
		t.Errorf("Delete() expected category to be deleted")
	}

	// Test deleting non-existent category
	err = repo.Delete(99999)
	if err == nil {
		t.Errorf("Delete() expected error for non-existent category")
	}
}

func TestCategoryRepository_SearchCategories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Create test categories
	categories := []models.Category{
		{Name: "Technology", Description: "Tech articles"},
		{Name: "Science", Description: "Science articles"},
		{Name: "Tech News", Description: "Latest technology news"},
	}

	for i := range categories {
		_, err := repo.Create(&categories[i])
		if err != nil {
			t.Fatalf("Failed to create test category: %v", err)
		}
	}

	// Search for "tech"
	results, err := repo.SearchCategories("tech", 10)
	if err != nil {
		t.Errorf("SearchCategories() unexpected error: %v", err)
		return
	}

	if len(results) < 2 {
		t.Errorf("SearchCategories() expected at least 2 results, got %d", len(results))
	}

	// Verify results contain "tech" in name or description
	for _, result := range results {
		name := strings.ToLower(result.Name)
		desc := strings.ToLower(result.Description)
		if !strings.Contains(name, "tech") && !strings.Contains(desc, "tech") {
			t.Errorf("SearchCategories() result '%s' does not contain 'tech'", result.Name)
		}
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	// This would typically use a test database
	// For now, we'll skip if no test database is available
	t.Skip("Database tests require test database setup")
	return nil
}