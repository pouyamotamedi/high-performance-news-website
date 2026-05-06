package templates

import (
	"testing"
	"time"
)

func TestNewBaseTemplateData(t *testing.T) {
	data := NewBaseTemplateData()
	
	if data == nil {
		t.Fatal("Expected BaseTemplateData to be created")
	}
	
	// Check default values
	if data.CurrentYear != time.Now().Year() {
		t.Errorf("Expected CurrentYear to be %d, got %d", time.Now().Year(), data.CurrentYear)
	}
	
	if data.LanguageCode != "en" {
		t.Errorf("Expected LanguageCode to be 'en', got %s", data.LanguageCode)
	}
	
	if data.LanguageDirection != "ltr" {
		t.Errorf("Expected LanguageDirection to be 'ltr', got %s", data.LanguageDirection)
	}
	
	if data.ThemeMode != "auto" {
		t.Errorf("Expected ThemeMode to be 'auto', got %s", data.ThemeMode)
	}
	
	if data.OGType != "website" {
		t.Errorf("Expected OGType to be 'website', got %s", data.OGType)
	}
	
	if data.TwitterCard != "summary_large_image" {
		t.Errorf("Expected TwitterCard to be 'summary_large_image', got %s", data.TwitterCard)
	}
	
	if data.CustomData == nil {
		t.Error("Expected CustomData to be initialized")
	}
	
	// Check that CurrentTime is recent
	if time.Since(data.CurrentTime) > time.Minute {
		t.Error("Expected CurrentTime to be recent")
	}
}

func TestBaseTemplateDataFields(t *testing.T) {
	data := NewBaseTemplateData()
	
	// Test setting various fields
	data.SiteName = "Test Site"
	data.Title = "Test Title"
	data.Description = "Test Description"
	data.Keywords = []string{"test", "keywords"}
	data.CanonicalURL = "https://example.com/test"
	
	if data.SiteName != "Test Site" {
		t.Errorf("Expected SiteName to be 'Test Site', got %s", data.SiteName)
	}
	
	if data.Title != "Test Title" {
		t.Errorf("Expected Title to be 'Test Title', got %s", data.Title)
	}
	
	if data.Description != "Test Description" {
		t.Errorf("Expected Description to be 'Test Description', got %s", data.Description)
	}
	
	if len(data.Keywords) != 2 || data.Keywords[0] != "test" || data.Keywords[1] != "keywords" {
		t.Errorf("Expected Keywords to be ['test', 'keywords'], got %v", data.Keywords)
	}
	
	if data.CanonicalURL != "https://example.com/test" {
		t.Errorf("Expected CanonicalURL to be 'https://example.com/test', got %s", data.CanonicalURL)
	}
}

func TestNavigationItem(t *testing.T) {
	nav := NavigationItem{
		Name:   "Home",
		URL:    "/",
		Icon:   "home",
		Active: true,
	}
	
	if nav.Name != "Home" {
		t.Errorf("Expected Name to be 'Home', got %s", nav.Name)
	}
	
	if nav.URL != "/" {
		t.Errorf("Expected URL to be '/', got %s", nav.URL)
	}
	
	if nav.Icon != "home" {
		t.Errorf("Expected Icon to be 'home', got %s", nav.Icon)
	}
	
	if !nav.Active {
		t.Error("Expected Active to be true")
	}
	
	// Test with children
	nav.Children = []NavigationItem{
		{Name: "Child 1", URL: "/child1"},
		{Name: "Child 2", URL: "/child2"},
	}
	
	if len(nav.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(nav.Children))
	}
	
	if nav.Children[0].Name != "Child 1" {
		t.Errorf("Expected first child name to be 'Child 1', got %s", nav.Children[0].Name)
	}
}

func TestBreadcrumbItem(t *testing.T) {
	breadcrumb := BreadcrumbItem{
		Name: "Category",
		URL:  "/categories/tech",
	}
	
	if breadcrumb.Name != "Category" {
		t.Errorf("Expected Name to be 'Category', got %s", breadcrumb.Name)
	}
	
	if breadcrumb.URL != "/categories/tech" {
		t.Errorf("Expected URL to be '/categories/tech', got %s", breadcrumb.URL)
	}
}

func TestPreloadResource(t *testing.T) {
	resource := PreloadResource{
		URL:  "/static/css/main.css",
		Type: "style",
		As:   "style",
	}
	
	if resource.URL != "/static/css/main.css" {
		t.Errorf("Expected URL to be '/static/css/main.css', got %s", resource.URL)
	}
	
	if resource.Type != "style" {
		t.Errorf("Expected Type to be 'style', got %s", resource.Type)
	}
	
	if resource.As != "style" {
		t.Errorf("Expected As to be 'style', got %s", resource.As)
	}
}

func TestHomepageData(t *testing.T) {
	data := HomepageData{
		BaseTemplateData: *NewBaseTemplateData(),
		HeroTitle:        "Welcome to News Site",
		HeroSubtitle:     "Latest news and updates",
		TotalArticles:    1000,
		TotalViews:       50000,
	}
	
	if data.HeroTitle != "Welcome to News Site" {
		t.Errorf("Expected HeroTitle to be 'Welcome to News Site', got %s", data.HeroTitle)
	}
	
	if data.HeroSubtitle != "Latest news and updates" {
		t.Errorf("Expected HeroSubtitle to be 'Latest news and updates', got %s", data.HeroSubtitle)
	}
	
	if data.TotalArticles != 1000 {
		t.Errorf("Expected TotalArticles to be 1000, got %d", data.TotalArticles)
	}
	
	if data.TotalViews != 50000 {
		t.Errorf("Expected TotalViews to be 50000, got %d", data.TotalViews)
	}
}

func TestArticlePageData(t *testing.T) {
	now := time.Now()
	
	article := Article{
		ID:          1,
		Title:       "Test Article",
		Slug:        "test-article",
		Content:     "This is test content",
		Excerpt:     "Test excerpt",
		AuthorID:    1,
		CategoryID:  1,
		Status:      "published",
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
		ViewCount:   100,
		LikeCount:   10,
	}
	
	data := ArticlePageData{
		BaseTemplateData: *NewBaseTemplateData(),
		Article:          article,
		ReadingTime:      5,
		CommentCount:     3,
	}
	
	if data.Article.Title != "Test Article" {
		t.Errorf("Expected Article.Title to be 'Test Article', got %s", data.Article.Title)
	}
	
	if data.ReadingTime != 5 {
		t.Errorf("Expected ReadingTime to be 5, got %d", data.ReadingTime)
	}
	
	if data.CommentCount != 3 {
		t.Errorf("Expected CommentCount to be 3, got %d", data.CommentCount)
	}
}

func TestCategoryPageData(t *testing.T) {
	category := Category{
		ID:           1,
		Name:         "Technology",
		Slug:         "technology",
		Description:  "Tech news and updates",
		ArticleCount: 50,
	}
	
	pagination := PaginationData{
		CurrentPage:  1,
		TotalPages:   5,
		TotalItems:   100,
		ItemsPerPage: 20,
		HasPrevious:  false,
		HasNext:      true,
		NextURL:      "?page=2",
	}
	
	data := CategoryPageData{
		BaseTemplateData: *NewBaseTemplateData(),
		Category:         category,
		Pagination:       pagination,
	}
	
	if data.Category.Name != "Technology" {
		t.Errorf("Expected Category.Name to be 'Technology', got %s", data.Category.Name)
	}
	
	if data.Pagination.CurrentPage != 1 {
		t.Errorf("Expected Pagination.CurrentPage to be 1, got %d", data.Pagination.CurrentPage)
	}
	
	if data.Pagination.HasNext != true {
		t.Error("Expected Pagination.HasNext to be true")
	}
}

func TestTagPageData(t *testing.T) {
	tag := Tag{
		ID:           1,
		Name:         "JavaScript",
		Slug:         "javascript",
		Description:  "JavaScript programming",
		Color:        "#f7df1e",
		ArticleCount: 25,
		Keywords:     []string{"js", "javascript", "programming"},
	}
	
	data := TagPageData{
		BaseTemplateData: *NewBaseTemplateData(),
		Tag:              tag,
	}
	
	if data.Tag.Name != "JavaScript" {
		t.Errorf("Expected Tag.Name to be 'JavaScript', got %s", data.Tag.Name)
	}
	
	if data.Tag.Color != "#f7df1e" {
		t.Errorf("Expected Tag.Color to be '#f7df1e', got %s", data.Tag.Color)
	}
	
	if len(data.Tag.Keywords) != 3 {
		t.Errorf("Expected 3 keywords, got %d", len(data.Tag.Keywords))
	}
}

func TestSearchPageData(t *testing.T) {
	searchTime := 50 * time.Millisecond
	
	filters := SearchFilters{
		SortBy:    "relevance",
		SortOrder: "desc",
	}
	
	data := SearchPageData{
		BaseTemplateData: *NewBaseTemplateData(),
		Query:            "test query",
		TotalResults:     42,
		SearchTime:       searchTime,
		Suggestions:      []string{"test", "testing", "tests"},
		Filters:          filters,
	}
	
	if data.Query != "test query" {
		t.Errorf("Expected Query to be 'test query', got %s", data.Query)
	}
	
	if data.TotalResults != 42 {
		t.Errorf("Expected TotalResults to be 42, got %d", data.TotalResults)
	}
	
	if data.SearchTime != searchTime {
		t.Errorf("Expected SearchTime to be %v, got %v", searchTime, data.SearchTime)
	}
	
	if len(data.Suggestions) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(data.Suggestions))
	}
	
	if data.Filters.SortBy != "relevance" {
		t.Errorf("Expected Filters.SortBy to be 'relevance', got %s", data.Filters.SortBy)
	}
}

func TestPaginationData(t *testing.T) {
	pagination := PaginationData{
		CurrentPage:  3,
		TotalPages:   10,
		TotalItems:   200,
		ItemsPerPage: 20,
		HasPrevious:  true,
		HasNext:      true,
		PreviousURL:  "?page=2",
		NextURL:      "?page=4",
	}
	
	if pagination.CurrentPage != 3 {
		t.Errorf("Expected CurrentPage to be 3, got %d", pagination.CurrentPage)
	}
	
	if pagination.TotalPages != 10 {
		t.Errorf("Expected TotalPages to be 10, got %d", pagination.TotalPages)
	}
	
	if pagination.TotalItems != 200 {
		t.Errorf("Expected TotalItems to be 200, got %d", pagination.TotalItems)
	}
	
	if !pagination.HasPrevious {
		t.Error("Expected HasPrevious to be true")
	}
	
	if !pagination.HasNext {
		t.Error("Expected HasNext to be true")
	}
	
	if pagination.PreviousURL != "?page=2" {
		t.Errorf("Expected PreviousURL to be '?page=2', got %s", pagination.PreviousURL)
	}
	
	if pagination.NextURL != "?page=4" {
		t.Errorf("Expected NextURL to be '?page=4', got %s", pagination.NextURL)
	}
}

func TestArticleStruct(t *testing.T) {
	now := time.Now()
	
	article := Article{
		ID:             1,
		Title:          "Test Article",
		Slug:           "test-article",
		Content:        "This is the article content",
		Excerpt:        "This is the excerpt",
		AuthorID:       1,
		CategoryID:     1,
		Status:         "published",
		PublishedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
		ViewCount:      100,
		LikeCount:      10,
		DislikeCount:   2,
		SEOTitle:       "SEO Title",
		SEODescription: "SEO Description",
		SEOKeywords:    []string{"test", "article"},
		CanonicalURL:   "https://example.com/en/article/test-article",
		FeaturedImage:  "/images/test.jpg",
		ImageAlt:       "Test image",
	}
	
	if article.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", article.ID)
	}
	
	if article.Title != "Test Article" {
		t.Errorf("Expected Title to be 'Test Article', got %s", article.Title)
	}
	
	if article.ViewCount != 100 {
		t.Errorf("Expected ViewCount to be 100, got %d", article.ViewCount)
	}
	
	if len(article.SEOKeywords) != 2 {
		t.Errorf("Expected 2 SEO keywords, got %d", len(article.SEOKeywords))
	}
}

func TestCategoryStruct(t *testing.T) {
	now := time.Now()
	parentID := uint64(1)
	
	category := Category{
		ID:           2,
		Name:         "Subcategory",
		Slug:         "subcategory",
		Description:  "A subcategory",
		ParentID:     &parentID,
		SortOrder:    1,
		CreatedAt:    now,
		ArticleCount: 25,
	}
	
	if category.ID != 2 {
		t.Errorf("Expected ID to be 2, got %d", category.ID)
	}
	
	if category.Name != "Subcategory" {
		t.Errorf("Expected Name to be 'Subcategory', got %s", category.Name)
	}
	
	if category.ParentID == nil || *category.ParentID != 1 {
		t.Errorf("Expected ParentID to be 1, got %v", category.ParentID)
	}
	
	if category.ArticleCount != 25 {
		t.Errorf("Expected ArticleCount to be 25, got %d", category.ArticleCount)
	}
}

func TestAuthorStruct(t *testing.T) {
	now := time.Now()
	
	author := Author{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		FirstName:    "John",
		LastName:     "Doe",
		Bio:          "Test author bio",
		Avatar:       "/avatars/john.jpg",
		CreatedAt:    now,
		ArticleCount: 15,
		SocialLinks: map[string]string{
			"twitter":  "https://twitter.com/johndoe",
			"linkedin": "https://linkedin.com/in/johndoe",
		},
	}
	
	if author.Username != "testuser" {
		t.Errorf("Expected Username to be 'testuser', got %s", author.Username)
	}
	
	if author.FirstName != "John" {
		t.Errorf("Expected FirstName to be 'John', got %s", author.FirstName)
	}
	
	if author.ArticleCount != 15 {
		t.Errorf("Expected ArticleCount to be 15, got %d", author.ArticleCount)
	}
	
	if len(author.SocialLinks) != 2 {
		t.Errorf("Expected 2 social links, got %d", len(author.SocialLinks))
	}
	
	if author.SocialLinks["twitter"] != "https://twitter.com/johndoe" {
		t.Errorf("Expected Twitter link to be 'https://twitter.com/johndoe', got %s", author.SocialLinks["twitter"])
	}
}

func TestCommentStruct(t *testing.T) {
	now := time.Now()
	authorID := uint64(1)
	parentID := uint64(1)
	
	comment := Comment{
		ID:        2,
		ArticleID: 1,
		ParentID:  &parentID,
		AuthorID:  &authorID,
		Name:      "John Doe",
		Email:     "john@example.com",
		Content:   "This is a test comment",
		Status:    "approved",
		CreatedAt: now,
	}
	
	if comment.ID != 2 {
		t.Errorf("Expected ID to be 2, got %d", comment.ID)
	}
	
	if comment.ArticleID != 1 {
		t.Errorf("Expected ArticleID to be 1, got %d", comment.ArticleID)
	}
	
	if comment.ParentID == nil || *comment.ParentID != 1 {
		t.Errorf("Expected ParentID to be 1, got %v", comment.ParentID)
	}
	
	if comment.AuthorID == nil || *comment.AuthorID != 1 {
		t.Errorf("Expected AuthorID to be 1, got %v", comment.AuthorID)
	}
	
	if comment.Content != "This is a test comment" {
		t.Errorf("Expected Content to be 'This is a test comment', got %s", comment.Content)
	}
}

func TestDateRange(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	
	dateRange := DateRange{
		From: from,
		To:   to,
	}
	
	if !dateRange.From.Equal(from) {
		t.Errorf("Expected From to be %v, got %v", from, dateRange.From)
	}
	
	if !dateRange.To.Equal(to) {
		t.Errorf("Expected To to be %v, got %v", to, dateRange.To)
	}
}