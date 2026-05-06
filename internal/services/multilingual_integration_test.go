package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

// Integration tests for multilingual functionality
// These tests require a test database with the multilingual schema

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	// Skip if no test database is configured
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// This would connect to a real test database
	// For now, we'll skip these tests
	t.Skip("Integration tests require test database setup")
	return nil
}

func TestMultilingualIntegration_CompleteWorkflow(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupIntegrationTestData(t, db)
	
	service := NewMultilingualService(db)
	
	// Test 1: Verify default languages are present
	languages, err := service.GetLanguages()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(languages), 3)
	
	// Find Persian, English, and Arabic
	var persian, english, arabic *models.Language
	for i, lang := range languages {
		switch lang.Code {
		case "fa":
			persian = &languages[i]
		case "en":
			english = &languages[i]
		case "ar":
			arabic = &languages[i]
		}
	}
	
	require.NotNil(t, persian, "Persian language should be present")
	require.NotNil(t, english, "English language should be present")
	require.NotNil(t, arabic, "Arabic language should be present")
	
	// Verify language properties
	assert.Equal(t, "rtl", persian.Direction)
	assert.Equal(t, "ltr", english.Direction)
	assert.Equal(t, "rtl", arabic.Direction)
	assert.True(t, persian.IsActive)
	
	// Test 2: Create multilingual articles
	articleIDs := createTestMultilingualArticles(t, db)
	require.Equal(t, 3, len(articleIDs))
	
	// Test 3: Create translation group
	groupID, err := service.CreateTranslationGroup("article", articleIDs)
	require.NoError(t, err)
	assert.Greater(t, groupID, uint64(0))
	
	// Test 4: Verify translation relationships
	for _, articleID := range articleIDs {
		article, err := service.GetArticleTranslations(articleID)
		require.NoError(t, err)
		
		assert.Equal(t, groupID, *article.TranslationGroupID)
		assert.Equal(t, 2, len(article.Translations)) // Should have 2 other translations
		
		// Verify translations don't include the article itself
		for _, translation := range article.Translations {
			assert.NotEqual(t, articleID, translation.ID)
		}
	}
	
	// Test 5: Test language-specific content retrieval
	persianArticles, err := service.GetArticlesByLanguage("fa", "fa", 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(persianArticles), 1)
	
	englishArticles, err := service.GetArticlesByLanguage("en", "fa", 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(englishArticles), 1)
	
	// Test 6: Test fallback functionality
	// Create an article only in Persian
	persianOnlyID := createTestArticle(t, db, "مقاله فقط فارسی", "persian-only-article", "fa")
	
	// Request in English with Persian fallback
	mixedArticles, err := service.GetArticlesByLanguage("en", "fa", 10, 0)
	require.NoError(t, err)
	
	// Should contain both English articles and Persian fallbacks
	var foundPersianFallback bool
	for _, article := range mixedArticles {
		if article.ID == persianOnlyID && article.IsFallback {
			foundPersianFallback = true
			break
		}
	}
	assert.True(t, foundPersianFallback, "Should find Persian article as fallback")
	
	// Test 7: Test route generation
	routeInfo, err := service.GenerateLanguageRouteInfo("articles", "test-article", "en")
	require.NoError(t, err)
	
	assert.Equal(t, "en", routeInfo.LanguageCode)
	assert.False(t, routeInfo.IsDefault)
	assert.Equal(t, "/en", routeInfo.URLPrefix)
	assert.Equal(t, "ltr", routeInfo.Direction)
	assert.NotEmpty(t, routeInfo.AlternateURLs)
	
	// Test with default language
	routeInfoDefault, err := service.GenerateLanguageRouteInfo("articles", "test-article", "fa")
	require.NoError(t, err)
	
	assert.True(t, routeInfoDefault.IsDefault)
	assert.Equal(t, "", routeInfoDefault.URLPrefix)
	assert.Equal(t, "rtl", routeInfoDefault.Direction)
}

func TestMultilingualIntegration_CategoryTranslations(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupIntegrationTestData(t, db)
	
	service := NewMultilingualService(db)
	
	// Create multilingual categories
	categoryIDs := createTestMultilingualCategories(t, db)
	require.Equal(t, 3, len(categoryIDs))
	
	// Create translation group
	groupID, err := service.CreateTranslationGroup("category", categoryIDs)
	require.NoError(t, err)
	
	// Test language-specific category retrieval
	persianCategories, err := service.GetCategoriesByLanguage("fa")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(persianCategories), 1)
	
	// Find our test category
	var testCategory *models.MultilingualCategory
	for i, cat := range persianCategories {
		if *cat.TranslationGroupID == groupID {
			testCategory = &persianCategories[i]
			break
		}
	}
	
	require.NotNil(t, testCategory, "Should find test category")
	assert.Equal(t, "fa", testCategory.LanguageCode)
	assert.Equal(t, "rtl", testCategory.LanguageDirection)
	assert.Equal(t, 2, len(testCategory.Translations))
}

func TestMultilingualIntegration_TagTranslations(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupIntegrationTestData(t, db)
	
	service := NewMultilingualService(db)
	
	// Create multilingual tags
	tagIDs := createTestMultilingualTags(t, db)
	require.Equal(t, 3, len(tagIDs))
	
	// Create translation group
	groupID, err := service.CreateTranslationGroup("tag", tagIDs)
	require.NoError(t, err)
	
	// Test language-specific tag retrieval
	englishTags, err := service.GetTagsByLanguage("en")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(englishTags), 1)
	
	// Find our test tag
	var testTag *models.MultilingualTag
	for i, tag := range englishTags {
		if *tag.TranslationGroupID == groupID {
			testTag = &englishTags[i]
			break
		}
	}
	
	require.NotNil(t, testTag, "Should find test tag")
	assert.Equal(t, "en", testTag.LanguageCode)
	assert.Equal(t, "ltr", testTag.LanguageDirection)
	assert.Equal(t, 2, len(testTag.Translations))
	assert.NotEmpty(t, testTag.Keywords)
}

func TestMultilingualIntegration_URLExtraction(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedLang string
		expectedPath string
	}{
		{
			name:         "Persian article (default)",
			url:          "/article/test-article",
			expectedLang: "en",
			expectedPath: "article/test-article",
		},
		{
			name:         "English article",
			url:          "/en/article/test-article",
			expectedLang: "en",
			expectedPath: "article/test-article",
		},
		{
			name:         "Arabic category",
			url:          "/ar/categories/technology",
			expectedLang: "ar",
			expectedPath: "categories/technology",
		},
		{
			name:         "Root with language",
			url:          "/en",
			expectedLang: "en",
			expectedPath: "",
		},
		{
			name:         "API endpoint (no language)",
			url:          "/api/v1/articles",
			expectedLang: "fa",
			expectedPath: "api/v1/articles",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, path := ExtractLanguageFromURL(tt.url)
			assert.Equal(t, tt.expectedLang, lang)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestMultilingualIntegration_LanguageValidation(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	// Test valid languages
	validLanguages := []string{"fa", "en", "ar"}
	for _, lang := range validLanguages {
		err := service.ValidateLanguageCode(lang)
		assert.NoError(t, err, "Language %s should be valid", lang)
		
		isRTL, err := service.IsRTLLanguage(lang)
		require.NoError(t, err)
		
		if lang == "fa" || lang == "ar" {
			assert.True(t, isRTL, "Language %s should be RTL", lang)
		} else {
			assert.False(t, isRTL, "Language %s should be LTR", lang)
		}
	}
	
	// Test invalid languages
	invalidLanguages := []string{"", "x", "xxx", "zz"}
	for _, lang := range invalidLanguages {
		err := service.ValidateLanguageCode(lang)
		assert.Error(t, err, "Language %s should be invalid", lang)
	}
}

func TestMultilingualIntegration_PerformanceWithLargeDataset(t *testing.T) {
	db := setupIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupIntegrationTestData(t, db)
	
	service := NewMultilingualService(db)
	
	// Create a large number of articles in different languages
	const numArticles = 100
	var articleIDs []uint64
	
	languages := []string{"fa", "en", "ar"}
	for i := 0; i < numArticles; i++ {
		lang := languages[i%len(languages)]
		title := fmt.Sprintf("Test Article %d", i)
		slug := fmt.Sprintf("test-article-%d", i)
		
		articleID := createTestArticle(t, db, title, slug, lang)
		articleIDs = append(articleIDs, articleID)
	}
	
	// Measure performance of language-specific queries
	start := time.Now()
	
	persianArticles, err := service.GetArticlesByLanguage("fa", "fa", 50, 0)
	require.NoError(t, err)
	
	duration := time.Since(start)
	t.Logf("Retrieved %d Persian articles in %v", len(persianArticles), duration)
	
	// Should be fast (under 100ms for this dataset)
	assert.Less(t, duration, 100*time.Millisecond, "Query should be fast")
	assert.GreaterOrEqual(t, len(persianArticles), numArticles/3-5) // Allow some variance
	
	// Test pagination performance
	start = time.Now()
	
	paginatedArticles, err := service.GetArticlesByLanguage("en", "fa", 10, 20)
	require.NoError(t, err)
	
	paginationDuration := time.Since(start)
	t.Logf("Retrieved paginated articles in %v", paginationDuration)
	
	assert.Less(t, paginationDuration, 50*time.Millisecond, "Pagination should be fast")
	assert.LessOrEqual(t, len(paginatedArticles), 10, "Should respect limit")
}

// Helper functions for integration tests

func createTestMultilingualArticles(t *testing.T, db *sql.DB) []uint64 {
	articles := []struct {
		title    string
		slug     string
		language string
	}{
		{"Test Article", "test-article", "en"},
		{"مقاله آزمایشی", "test-article-fa", "fa"},
		{"مقالة اختبار", "test-article-ar", "ar"},
	}
	
	var articleIDs []uint64
	for _, article := range articles {
		articleID := createTestArticle(t, db, article.title, article.slug, article.language)
		articleIDs = append(articleIDs, articleID)
	}
	
	return articleIDs
}

func createTestMultilingualCategories(t *testing.T, db *sql.DB) []uint64 {
	categories := []struct {
		name     string
		slug     string
		language string
	}{
		{"Technology", "technology", "en"},
		{"فناوری", "technology-fa", "fa"},
		{"التكنولوجيا", "technology-ar", "ar"},
	}
	
	var categoryIDs []uint64
	for _, category := range categories {
		categoryID := createTestCategory(t, db, category.name, category.slug, category.language)
		categoryIDs = append(categoryIDs, categoryID)
	}
	
	return categoryIDs
}

func createTestMultilingualTags(t *testing.T, db *sql.DB) []uint64 {
	tags := []struct {
		name     string
		slug     string
		language string
		keywords []string
	}{
		{"Programming", "programming", "en", []string{"programming", "coding", "development"}},
		{"برنامه‌نویسی", "programming-fa", "fa", []string{"برنامه‌نویسی", "کدنویسی", "توسعه"}},
		{"البرمجة", "programming-ar", "ar", []string{"البرمجة", "الترميز", "التطوير"}},
	}
	
	var tagIDs []uint64
	for _, tag := range tags {
		tagID := createTestTag(t, db, tag.name, tag.slug, tag.language, tag.keywords)
		tagIDs = append(tagIDs, tagID)
	}
	
	return tagIDs
}

func createTestArticle(t *testing.T, db *sql.DB, title, slug, languageCode string) uint64 {
	query := `
		INSERT INTO articles (title, slug, content, excerpt, author_id, category_id, language_code, status, published_at)
		VALUES ($1, $2, 'Test content for ' || $1, 'Test excerpt', 1, 1, $3, 'published', NOW())
		RETURNING id
	`
	
	var articleID uint64
	err := db.QueryRow(query, title, slug, languageCode).Scan(&articleID)
	require.NoError(t, err)
	
	return articleID
}

func createTestCategory(t *testing.T, db *sql.DB, name, slug, languageCode string) uint64 {
	query := `
		INSERT INTO categories (name, slug, description, language_code, sort_order)
		VALUES ($1, $2, 'Test category: ' || $1, $3, 0)
		RETURNING id
	`
	
	var categoryID uint64
	err := db.QueryRow(query, name, slug, languageCode).Scan(&categoryID)
	require.NoError(t, err)
	
	return categoryID
}

func createTestTag(t *testing.T, db *sql.DB, name, slug, languageCode string, keywords []string) uint64 {
	keywordsJSON, err := json.Marshal(keywords)
	require.NoError(t, err)
	
	query := `
		INSERT INTO tags (name, slug, description, keywords, language_code, color)
		VALUES ($1, $2, 'Test tag: ' || $1, $3, $4, '#007bff')
		RETURNING id
	`
	
	var tagID uint64
	err = db.QueryRow(query, name, slug, keywordsJSON, languageCode).Scan(&tagID)
	require.NoError(t, err)
	
	return tagID
}

func cleanupIntegrationTestData(t *testing.T, db *sql.DB) {
	queries := []string{
		"DELETE FROM article_tags WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM articles WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM categories WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM tags WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM translation_groups WHERE created_at >= NOW() - INTERVAL '1 hour'",
	}
	
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}
}