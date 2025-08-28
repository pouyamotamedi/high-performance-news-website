package services

import (
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// Mock repositories for testing
type mockWidgetRepository struct {
	widgets    map[uint64]*models.Widget
	placements map[uint64]*models.WidgetPlacement
	nextID     uint64
}

func newMockWidgetRepository() *mockWidgetRepository {
	return &mockWidgetRepository{
		widgets:    make(map[uint64]*models.Widget),
		placements: make(map[uint64]*models.WidgetPlacement),
		nextID:     1,
	}
}

func (m *mockWidgetRepository) Create(widget *models.Widget) (*models.Widget, error) {
	widget.ID = m.nextID
	m.nextID++
	widget.CreatedAt = time.Now()
	widget.UpdatedAt = time.Now()
	m.widgets[widget.ID] = widget
	return widget, nil
}

func (m *mockWidgetRepository) GetByID(id uint64) (*models.Widget, error) {
	widget, exists := m.widgets[id]
	if !exists {
		return nil, fmt.Errorf("widget not found")
	}
	return widget, nil
}

func (m *mockWidgetRepository) GetAll() ([]*models.Widget, error) {
	var widgets []*models.Widget
	for _, widget := range m.widgets {
		widgets = append(widgets, widget)
	}
	return widgets, nil
}

func (m *mockWidgetRepository) GetByType(widgetType models.WidgetType) ([]*models.Widget, error) {
	var widgets []*models.Widget
	for _, widget := range m.widgets {
		if widget.Type == widgetType && widget.IsActive {
			widgets = append(widgets, widget)
		}
	}
	return widgets, nil
}

func (m *mockWidgetRepository) Update(widget *models.Widget) error {
	if _, exists := m.widgets[widget.ID]; !exists {
		return fmt.Errorf("widget not found")
	}
	widget.UpdatedAt = time.Now()
	m.widgets[widget.ID] = widget
	return nil
}

func (m *mockWidgetRepository) Delete(id uint64) error {
	if _, exists := m.widgets[id]; !exists {
		return fmt.Errorf("widget not found")
	}
	delete(m.widgets, id)
	return nil
}

func (m *mockWidgetRepository) CreatePlacement(placement *models.WidgetPlacement) (*models.WidgetPlacement, error) {
	placement.ID = m.nextID
	m.nextID++
	placement.CreatedAt = time.Now()
	placement.UpdatedAt = time.Now()
	m.placements[placement.ID] = placement
	return placement, nil
}

func (m *mockWidgetRepository) GetPlacementsByPage(pageType models.PageType, zone models.WidgetZone) ([]*models.WidgetPlacement, error) {
	var placements []*models.WidgetPlacement
	for _, placement := range m.placements {
		if (placement.PageType == pageType || placement.PageType == models.PageTypeGlobal) && 
		   placement.Zone == zone && placement.IsActive {
			// Add widget data
			if widget, exists := m.widgets[placement.WidgetID]; exists {
				placement.Widget = widget
			}
			placements = append(placements, placement)
		}
	}
	return placements, nil
}

func (m *mockWidgetRepository) UpdatePlacement(placement *models.WidgetPlacement) error {
	if _, exists := m.placements[placement.ID]; !exists {
		return fmt.Errorf("widget placement not found")
	}
	placement.UpdatedAt = time.Now()
	m.placements[placement.ID] = placement
	return nil
}

func (m *mockWidgetRepository) DeletePlacement(id uint64) error {
	if _, exists := m.placements[id]; !exists {
		return fmt.Errorf("widget placement not found")
	}
	delete(m.placements, id)
	return nil
}

func (m *mockWidgetRepository) UpdatePlacementPositions(placements []*models.WidgetPlacement) error {
	for _, placement := range placements {
		if _, exists := m.placements[placement.ID]; !exists {
			return fmt.Errorf("widget placement not found")
		}
		placement.UpdatedAt = time.Now()
		m.placements[placement.ID] = placement
	}
	return nil
}

// Mock article repository
type mockArticleRepository struct{}

func (m *mockArticleRepository) GetLatest(count int) ([]*models.Article, error) {
	return []*models.Article{
		{ID: 1, Title: "Latest Article 1", Slug: "latest-1", Excerpt: "Excerpt 1"},
		{ID: 2, Title: "Latest Article 2", Slug: "latest-2", Excerpt: "Excerpt 2"},
	}, nil
}

func (m *mockArticleRepository) GetPopular(count int) ([]*models.Article, error) {
	return []*models.Article{
		{ID: 1, Title: "Popular Article 1", Slug: "popular-1", ViewCount: 1000},
		{ID: 2, Title: "Popular Article 2", Slug: "popular-2", ViewCount: 800},
	}, nil
}

func (m *mockArticleRepository) GetTrending(count int) ([]*models.Article, error) {
	return []*models.Article{
		{ID: 1, Title: "Trending Article 1", Slug: "trending-1"},
		{ID: 2, Title: "Trending Article 2", Slug: "trending-2"},
	}, nil
}

// Mock category repository
type mockCategoryRepository struct{}

func (m *mockCategoryRepository) GetAll() ([]*models.Category, error) {
	return []*models.Category{
		{ID: 1, Name: "Technology", Slug: "technology"},
		{ID: 2, Name: "Sports", Slug: "sports"},
	}, nil
}

// Mock tag repository
type mockTagRepository struct{}

func (m *mockTagRepository) GetAll() ([]*models.Tag, error) {
	return []*models.Tag{
		{ID: 1, Name: "Tech", Slug: "tech"},
		{ID: 2, Name: "News", Slug: "news"},
	}, nil
}

// Mock cache service
type mockCacheService struct{}

func (m *mockCacheService) Get(key string) ([]byte, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *mockCacheService) Delete(key string) error {
	return nil
}

func (m *mockCacheService) DeletePattern(pattern string) error {
	return nil
}

func (m *mockCacheService) Exists(key string) bool {
	return false
}

func setupWidgetService() *WidgetService {
	return NewWidgetService(
		newMockWidgetRepository(),
		&mockArticleRepository{},
		&mockCategoryRepository{},
		&mockTagRepository{},
		&mockCacheService{},
	)
}

func TestWidgetService_CreateWidget(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		Name:        "Test Widget",
		Type:        models.WidgetTypeLatestArticles,
		Title:       "Latest Articles",
		Description: "Display latest articles",
		Config: map[string]interface{}{
			"article_count": 5,
		},
		IsActive: true,
	}

	createdWidget, err := service.CreateWidget(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	if createdWidget.ID == 0 {
		t.Error("Expected widget ID to be set")
	}

	if createdWidget.SortOrder == 0 {
		t.Error("Expected default sort order to be set")
	}
}

func TestWidgetService_CreateWidget_Validation(t *testing.T) {
	service := setupWidgetService()

	// Test with invalid widget (missing name)
	widget := &models.Widget{
		Type:     models.WidgetTypeLatestArticles,
		IsActive: true,
	}

	_, err := service.CreateWidget(widget)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}

	// Test with invalid widget type
	widget = &models.Widget{
		Name:     "Test Widget",
		Type:     "invalid_type",
		IsActive: true,
	}

	_, err = service.CreateWidget(widget)
	if err == nil {
		t.Error("Expected validation error for invalid type")
	}
}

func TestWidgetService_GetWidgetsByType(t *testing.T) {
	service := setupWidgetService()

	// Create widgets of different types
	widget1 := &models.Widget{
		Name:     "Latest Widget",
		Type:     models.WidgetTypeLatestArticles,
		IsActive: true,
		Config:   map[string]interface{}{},
	}

	widget2 := &models.Widget{
		Name:     "Popular Widget",
		Type:     models.WidgetTypePopularArticles,
		IsActive: true,
		Config:   map[string]interface{}{},
	}

	widget3 := &models.Widget{
		Name:     "Another Latest Widget",
		Type:     models.WidgetTypeLatestArticles,
		IsActive: true,
		Config:   map[string]interface{}{},
	}

	service.CreateWidget(widget1)
	service.CreateWidget(widget2)
	service.CreateWidget(widget3)

	// Get widgets by type
	latestWidgets, err := service.GetWidgetsByType(models.WidgetTypeLatestArticles)
	if err != nil {
		t.Fatalf("Failed to get widgets by type: %v", err)
	}

	if len(latestWidgets) != 2 {
		t.Errorf("Expected 2 latest article widgets, got %d", len(latestWidgets))
	}
}

func TestWidgetService_CreateWidgetPlacement(t *testing.T) {
	service := setupWidgetService()

	// Create a widget first
	widget := &models.Widget{
		Name:     "Test Widget",
		Type:     models.WidgetTypeLatestArticles,
		IsActive: true,
		Config:   map[string]interface{}{},
	}

	createdWidget, err := service.CreateWidget(widget)
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

	createdPlacement, err := service.CreateWidgetPlacement(placement)
	if err != nil {
		t.Fatalf("Failed to create placement: %v", err)
	}

	if createdPlacement.ID == 0 {
		t.Error("Expected placement ID to be set")
	}
}

func TestWidgetService_CreateWidgetPlacement_Validation(t *testing.T) {
	service := setupWidgetService()

	// Test with invalid placement (missing widget ID)
	placement := &models.WidgetPlacement{
		PageType: models.PageTypeHomepage,
		Zone:     models.WidgetZoneSidebar,
		Position: 1,
		IsActive: true,
	}

	_, err := service.CreateWidgetPlacement(placement)
	if err == nil {
		t.Error("Expected validation error for missing widget ID")
	}

	// Test with invalid page type
	placement = &models.WidgetPlacement{
		WidgetID: 1,
		PageType: "invalid_page_type",
		Zone:     models.WidgetZoneSidebar,
		Position: 1,
		IsActive: true,
	}

	_, err = service.CreateWidgetPlacement(placement)
	if err == nil {
		t.Error("Expected validation error for invalid page type")
	}
}

func TestWidgetService_RenderWidget_LatestArticles(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:    1,
		Name:  "Latest Articles",
		Type:  models.WidgetTypeLatestArticles,
		Title: "Latest News",
		Config: map[string]interface{}{
			"article_count": 2,
			"show_excerpt":  true,
			"show_date":     true,
		},
	}

	html, err := service.RenderWidget(widget)
	if err != nil {
		t.Fatalf("Failed to render widget: %v", err)
	}

	htmlStr := string(html)
	if !strings.Contains(htmlStr, "Latest News") {
		t.Error("Expected widget title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "Latest Article 1") {
		t.Error("Expected article title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "widget-latest-articles") {
		t.Error("Expected widget CSS class in rendered HTML")
	}
}

func TestWidgetService_RenderWidget_PopularArticles(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:    1,
		Name:  "Popular Articles",
		Type:  models.WidgetTypePopularArticles,
		Title: "Most Popular",
		Config: map[string]interface{}{
			"article_count": 2,
		},
	}

	html, err := service.RenderWidget(widget)
	if err != nil {
		t.Fatalf("Failed to render widget: %v", err)
	}

	htmlStr := string(html)
	if !strings.Contains(htmlStr, "Most Popular") {
		t.Error("Expected widget title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "Popular Article 1") {
		t.Error("Expected article title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "1000 views") {
		t.Error("Expected view count in rendered HTML")
	}

	if !strings.Contains(htmlStr, "widget-popular-articles") {
		t.Error("Expected widget CSS class in rendered HTML")
	}
}

func TestWidgetService_RenderWidget_Categories(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:    1,
		Name:  "Categories",
		Type:  models.WidgetTypeCategories,
		Title: "Browse Categories",
		Config: map[string]interface{}{
			"show_hierarchy": true,
		},
	}

	html, err := service.RenderWidget(widget)
	if err != nil {
		t.Fatalf("Failed to render widget: %v", err)
	}

	htmlStr := string(html)
	if !strings.Contains(htmlStr, "Browse Categories") {
		t.Error("Expected widget title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "Technology") {
		t.Error("Expected category name in rendered HTML")
	}

	if !strings.Contains(htmlStr, "widget-categories") {
		t.Error("Expected widget CSS class in rendered HTML")
	}
}

func TestWidgetService_RenderWidget_Search(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:    1,
		Name:  "Search",
		Type:  models.WidgetTypeSearch,
		Title: "Search Articles",
		Config: map[string]interface{}{},
	}

	html, err := service.RenderWidget(widget)
	if err != nil {
		t.Fatalf("Failed to render widget: %v", err)
	}

	htmlStr := string(html)
	if !strings.Contains(htmlStr, "Search Articles") {
		t.Error("Expected widget title in rendered HTML")
	}

	if !strings.Contains(htmlStr, `action="/search"`) {
		t.Error("Expected search form action in rendered HTML")
	}

	if !strings.Contains(htmlStr, "widget-search") {
		t.Error("Expected widget CSS class in rendered HTML")
	}
}

func TestWidgetService_RenderWidget_CustomHTML(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:    1,
		Name:  "Custom HTML",
		Type:  models.WidgetTypeCustomHTML,
		Title: "Custom Content",
		Config: map[string]interface{}{
			"html_content": "<p>This is custom HTML content</p>",
		},
	}

	html, err := service.RenderWidget(widget)
	if err != nil {
		t.Fatalf("Failed to render widget: %v", err)
	}

	htmlStr := string(html)
	if !strings.Contains(htmlStr, "Custom Content") {
		t.Error("Expected widget title in rendered HTML")
	}

	if !strings.Contains(htmlStr, "This is custom HTML content") {
		t.Error("Expected custom HTML content in rendered HTML")
	}

	if !strings.Contains(htmlStr, "widget-custom-html") {
		t.Error("Expected widget CSS class in rendered HTML")
	}
}

func TestWidgetService_RenderWidget_UnsupportedType(t *testing.T) {
	service := setupWidgetService()

	widget := &models.Widget{
		ID:     1,
		Name:   "Unsupported Widget",
		Type:   "unsupported_type",
		Config: map[string]interface{}{},
	}

	_, err := service.RenderWidget(widget)
	if err == nil {
		t.Error("Expected error for unsupported widget type")
	}

	if !strings.Contains(err.Error(), "unsupported widget type") {
		t.Errorf("Expected unsupported widget type error, got: %v", err)
	}
}