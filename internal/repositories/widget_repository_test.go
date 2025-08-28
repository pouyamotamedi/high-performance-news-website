package repositories

import (
	"database/sql"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

func setupWidgetTestDB(t *testing.T) *sql.DB {
	// This would typically connect to a test database
	// For now, we'll use a mock or in-memory database
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Database not available for testing")
	}
	return db
}

func TestWidgetRepository_Create(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	widget := &models.Widget{
		Name:        "Test Widget",
		Type:        models.WidgetTypeLatestArticles,
		Title:       "Latest Articles",
		Description: "Display latest articles",
		Config: map[string]interface{}{
			"article_count": 5,
			"show_excerpt": true,
		},
		IsActive:  true,
		SortOrder: 10,
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	if createdWidget.ID == 0 {
		t.Error("Expected widget ID to be set")
	}

	if createdWidget.Name != widget.Name {
		t.Errorf("Expected name %s, got %s", widget.Name, createdWidget.Name)
	}

	if createdWidget.Type != widget.Type {
		t.Errorf("Expected type %s, got %s", widget.Type, createdWidget.Type)
	}
}

func TestWidgetRepository_GetByID(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create a widget first
	widget := &models.Widget{
		Name:        "Test Widget",
		Type:        models.WidgetTypeLatestArticles,
		Title:       "Latest Articles",
		Description: "Display latest articles",
		Config: map[string]interface{}{
			"article_count": 5,
		},
		IsActive:  true,
		SortOrder: 10,
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Retrieve the widget
	retrievedWidget, err := repo.GetByID(createdWidget.ID)
	if err != nil {
		t.Fatalf("Failed to get widget: %v", err)
	}

	if retrievedWidget.ID != createdWidget.ID {
		t.Errorf("Expected ID %d, got %d", createdWidget.ID, retrievedWidget.ID)
	}

	if retrievedWidget.Name != widget.Name {
		t.Errorf("Expected name %s, got %s", widget.Name, retrievedWidget.Name)
	}

	// Check config
	if articleCount, ok := retrievedWidget.Config["article_count"]; !ok || articleCount != float64(5) {
		t.Error("Expected article_count config to be 5")
	}
}

func TestWidgetRepository_GetByType(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create widgets of different types
	widgets := []*models.Widget{
		{
			Name:      "Latest Articles Widget",
			Type:      models.WidgetTypeLatestArticles,
			IsActive:  true,
			SortOrder: 10,
			Config:    map[string]interface{}{},
		},
		{
			Name:      "Popular Articles Widget",
			Type:      models.WidgetTypePopularArticles,
			IsActive:  true,
			SortOrder: 20,
			Config:    map[string]interface{}{},
		},
		{
			Name:      "Another Latest Widget",
			Type:      models.WidgetTypeLatestArticles,
			IsActive:  true,
			SortOrder: 30,
			Config:    map[string]interface{}{},
		},
	}

	for _, widget := range widgets {
		_, err := repo.Create(widget)
		if err != nil {
			t.Fatalf("Failed to create widget: %v", err)
		}
	}

	// Get widgets by type
	latestWidgets, err := repo.GetByType(models.WidgetTypeLatestArticles)
	if err != nil {
		t.Fatalf("Failed to get widgets by type: %v", err)
	}

	if len(latestWidgets) != 2 {
		t.Errorf("Expected 2 latest article widgets, got %d", len(latestWidgets))
	}

	// Check sorting
	if latestWidgets[0].SortOrder > latestWidgets[1].SortOrder {
		t.Error("Expected widgets to be sorted by sort_order")
	}
}

func TestWidgetRepository_Update(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create a widget
	widget := &models.Widget{
		Name:        "Test Widget",
		Type:        models.WidgetTypeLatestArticles,
		Title:       "Original Title",
		Description: "Original description",
		Config:      map[string]interface{}{"article_count": 5},
		IsActive:    true,
		SortOrder:   10,
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Update the widget
	createdWidget.Title = "Updated Title"
	createdWidget.Description = "Updated description"
	createdWidget.Config["article_count"] = 10
	createdWidget.IsActive = false

	err = repo.Update(createdWidget)
	if err != nil {
		t.Fatalf("Failed to update widget: %v", err)
	}

	// Retrieve and verify
	updatedWidget, err := repo.GetByID(createdWidget.ID)
	if err != nil {
		t.Fatalf("Failed to get updated widget: %v", err)
	}

	if updatedWidget.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", updatedWidget.Title)
	}

	if updatedWidget.IsActive {
		t.Error("Expected widget to be inactive")
	}

	if articleCount, ok := updatedWidget.Config["article_count"]; !ok || articleCount != float64(10) {
		t.Error("Expected article_count config to be 10")
	}
}

func TestWidgetRepository_Delete(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create a widget
	widget := &models.Widget{
		Name:      "Test Widget",
		Type:      models.WidgetTypeLatestArticles,
		IsActive:  true,
		SortOrder: 10,
		Config:    map[string]interface{}{},
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Delete the widget
	err = repo.Delete(createdWidget.ID)
	if err != nil {
		t.Fatalf("Failed to delete widget: %v", err)
	}

	// Try to retrieve the deleted widget
	_, err = repo.GetByID(createdWidget.ID)
	if err == nil {
		t.Error("Expected error when getting deleted widget")
	}
}

func TestWidgetRepository_CreatePlacement(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create a widget first
	widget := &models.Widget{
		Name:      "Test Widget",
		Type:      models.WidgetTypeLatestArticles,
		IsActive:  true,
		SortOrder: 10,
		Config:    map[string]interface{}{},
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Create a placement
	placement := &models.WidgetPlacement{
		WidgetID: createdWidget.ID,
		PageType: models.PageTypeHomepage,
		Zone:     models.WidgetZoneSidebar,
		Position: 1,
		IsActive: true,
	}

	createdPlacement, err := repo.CreatePlacement(placement)
	if err != nil {
		t.Fatalf("Failed to create placement: %v", err)
	}

	if createdPlacement.ID == 0 {
		t.Error("Expected placement ID to be set")
	}

	if createdPlacement.WidgetID != createdWidget.ID {
		t.Errorf("Expected widget ID %d, got %d", createdWidget.ID, createdPlacement.WidgetID)
	}
}

func TestWidgetRepository_GetPlacementsByPage(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create widgets
	widget1 := &models.Widget{
		Name:      "Widget 1",
		Type:      models.WidgetTypeLatestArticles,
		IsActive:  true,
		SortOrder: 10,
		Config:    map[string]interface{}{},
	}

	widget2 := &models.Widget{
		Name:      "Widget 2",
		Type:      models.WidgetTypePopularArticles,
		IsActive:  true,
		SortOrder: 20,
		Config:    map[string]interface{}{},
	}

	createdWidget1, err := repo.Create(widget1)
	if err != nil {
		t.Fatalf("Failed to create widget 1: %v", err)
	}

	createdWidget2, err := repo.Create(widget2)
	if err != nil {
		t.Fatalf("Failed to create widget 2: %v", err)
	}

	// Create placements
	placements := []*models.WidgetPlacement{
		{
			WidgetID: createdWidget1.ID,
			PageType: models.PageTypeHomepage,
			Zone:     models.WidgetZoneSidebar,
			Position: 1,
			IsActive: true,
		},
		{
			WidgetID: createdWidget2.ID,
			PageType: models.PageTypeHomepage,
			Zone:     models.WidgetZoneSidebar,
			Position: 2,
			IsActive: true,
		},
		{
			WidgetID: createdWidget1.ID,
			PageType: models.PageTypeArticle,
			Zone:     models.WidgetZoneSidebar,
			Position: 1,
			IsActive: true,
		},
	}

	for _, placement := range placements {
		_, err := repo.CreatePlacement(placement)
		if err != nil {
			t.Fatalf("Failed to create placement: %v", err)
		}
	}

	// Get placements for homepage sidebar
	homepagePlacements, err := repo.GetPlacementsByPage(models.PageTypeHomepage, models.WidgetZoneSidebar)
	if err != nil {
		t.Fatalf("Failed to get placements: %v", err)
	}

	if len(homepagePlacements) != 2 {
		t.Errorf("Expected 2 homepage sidebar placements, got %d", len(homepagePlacements))
	}

	// Check that widgets are included
	if homepagePlacements[0].Widget == nil {
		t.Error("Expected widget to be included in placement")
	}

	// Check sorting by position
	if homepagePlacements[0].Position > homepagePlacements[1].Position {
		t.Error("Expected placements to be sorted by position")
	}
}

func TestWidgetRepository_UpdatePlacementPositions(t *testing.T) {
	db := setupWidgetTestDB(t)
	defer db.Close()

	repo := NewWidgetRepository(db)

	// Create a widget
	widget := &models.Widget{
		Name:      "Test Widget",
		Type:      models.WidgetTypeLatestArticles,
		IsActive:  true,
		SortOrder: 10,
		Config:    map[string]interface{}{},
	}

	createdWidget, err := repo.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Create placements
	placement1 := &models.WidgetPlacement{
		WidgetID: createdWidget.ID,
		PageType: models.PageTypeHomepage,
		Zone:     models.WidgetZoneSidebar,
		Position: 1,
		IsActive: true,
	}

	placement2 := &models.WidgetPlacement{
		WidgetID: createdWidget.ID,
		PageType: models.PageTypeHomepage,
		Zone:     models.WidgetZoneSidebar,
		Position: 2,
		IsActive: true,
	}

	createdPlacement1, err := repo.CreatePlacement(placement1)
	if err != nil {
		t.Fatalf("Failed to create placement 1: %v", err)
	}

	createdPlacement2, err := repo.CreatePlacement(placement2)
	if err != nil {
		t.Fatalf("Failed to create placement 2: %v", err)
	}

	// Update positions (swap them)
	createdPlacement1.Position = 2
	createdPlacement2.Position = 1

	err = repo.UpdatePlacementPositions([]*models.WidgetPlacement{
		createdPlacement1,
		createdPlacement2,
	})
	if err != nil {
		t.Fatalf("Failed to update placement positions: %v", err)
	}

	// Verify positions were updated
	placements, err := repo.GetPlacementsByPage(models.PageTypeHomepage, models.WidgetZoneSidebar)
	if err != nil {
		t.Fatalf("Failed to get placements: %v", err)
	}

	if len(placements) != 2 {
		t.Fatalf("Expected 2 placements, got %d", len(placements))
	}

	// First placement should now be the one that was originally second
	if placements[0].ID != createdPlacement2.ID {
		t.Error("Expected placement positions to be swapped")
	}
}