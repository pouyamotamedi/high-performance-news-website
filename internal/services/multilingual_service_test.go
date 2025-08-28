package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

func setupMultilingualTestDB(t *testing.T) *sql.DB {
	// This would typically connect to a test database
	// For now, we'll use a mock or skip database-dependent tests
	t.Skip("Database tests require test database setup")
	return nil
}

func TestMultilingualService_GetLanguages(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	languages, err := service.GetLanguages()
	require.NoError(t, err)
	
	// Should have at least Persian, English, and Arabic
	assert.GreaterOrEqual(t, len(languages), 3)
	
	// Check for Persian (default language)
	var foundPersian bool
	for _, lang := range languages {
		if lang.Code == "fa" {
			foundPersian = true
			assert.Equal(t, "Persian", lang.Name)
			assert.Equal(t, "فارسی", lang.NativeName)
			assert.Equal(t, "rtl", lang.Direction)
			assert.True(t, lang.IsActive)
			break
		}
	}
	assert.True(t, foundPersian, "Persian language should be present")
}

func TestMultilingualService_GetActiveLanguages(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	languages, err := service.GetActiveLanguages()
	require.NoError(t, err)
	
	// All returned languages should be active
	for _, lang := range languages {
		assert.True(t, lang.IsActive)
	}
}

func TestMultilingualService_GetLanguageConfig(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	config, err := service.GetLanguageConfig()
	require.NoError(t, err)
	
	assert.Equal(t, "fa", config.DefaultLanguage)
	assert.Equal(t, "fa", config.FallbackLanguage)
	assert.Contains(t, config.ActiveLanguages, "fa")
	assert.Contains(t, config.RTLLanguages, "fa")
	
	// Test utility methods
	assert.True(t, config.IsRTL("fa"))
	assert.True(t, config.IsActive("fa"))
	assert.Equal(t, "fa", config.GetFallbackLanguage())
}

func TestMultilingualService_CreateTranslationGroup(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	// Test with valid input
	contentIDs := []uint64{1, 2, 3}
	groupID, err := service.CreateTranslationGroup("article", contentIDs)
	require.NoError(t, err)
	assert.Greater(t, groupID, uint64(0))
	
	// Test with invalid group type
	_, err = service.CreateTranslationGroup("invalid", contentIDs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group type")
	
	// Test with insufficient content IDs
	_, err = service.CreateTranslationGroup("article", []uint64{1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 items")
}

func TestMultilingualService_ValidateLanguageCode(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	// Test valid language codes
	err := service.ValidateLanguageCode("fa")
	assert.NoError(t, err)
	
	// Test invalid language codes
	err = service.ValidateLanguageCode("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
	
	err = service.ValidateLanguageCode("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly 2 characters")
	
	err = service.ValidateLanguageCode("xx")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestMultilingualService_IsRTLLanguage(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	// Test RTL languages
	isRTL, err := service.IsRTLLanguage("fa")
	require.NoError(t, err)
	assert.True(t, isRTL)
	
	isRTL, err = service.IsRTLLanguage("ar")
	require.NoError(t, err)
	assert.True(t, isRTL)
	
	// Test LTR languages
	isRTL, err = service.IsRTLLanguage("en")
	require.NoError(t, err)
	assert.False(t, isRTL)
	
	// Test invalid language
	_, err = service.IsRTLLanguage("xx")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExtractLanguageFromURL(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedLang string
		expectedPath string
	}{
		{
			name:         "URL with language prefix",
			path:         "/en/articles/test-article",
			expectedLang: "en",
			expectedPath: "articles/test-article",
		},
		{
			name:         "URL without language prefix",
			path:         "/articles/test-article",
			expectedLang: "fa",
			expectedPath: "articles/test-article",
		},
		{
			name:         "Root URL with language",
			path:         "/ar",
			expectedLang: "ar",
			expectedPath: "",
		},
		{
			name:         "Root URL without language",
			path:         "/",
			expectedLang: "fa",
			expectedPath: "",
		},
		{
			name:         "Empty path",
			path:         "",
			expectedLang: "fa",
			expectedPath: "",
		},
		{
			name:         "Path with non-language prefix",
			path:         "/api/v1/articles",
			expectedLang: "fa",
			expectedPath: "api/v1/articles",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, path := ExtractLanguageFromURL(tt.path)
			assert.Equal(t, tt.expectedLang, lang)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestMultilingualService_GenerateLanguageRouteInfo(t *testing.T) {
	db := setupMultilingualTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	
	service := NewMultilingualService(db)
	
	routeInfo, err := service.GenerateLanguageRouteInfo("articles", "test-article", "en")
	require.NoError(t, err)
	
	assert.Equal(t, "en", routeInfo.LanguageCode)
	assert.False(t, routeInfo.IsDefault) // Persian is default
	assert.Equal(t, "/en", routeInfo.URLPrefix)
	assert.Equal(t, "ltr", routeInfo.Direction)
	assert.NotEmpty(t, routeInfo.AlternateURLs)
	
	// Test with default language
	routeInfo, err = service.GenerateLanguageRouteInfo("articles", "test-article", "fa")
	require.NoError(t, err)
	
	assert.Equal(t, "fa", routeInfo.LanguageCode)
	assert.True(t, routeInfo.IsDefault)
	assert.Equal(t, "", routeInfo.URLPrefix)
	assert.Equal(t, "rtl", routeInfo.Direction)
}

// Test models
func TestTranslations_ScanAndValue(t *testing.T) {
	translations := models.Translations{
		{
			ID:           1,
			Title:        "Test Article",
			Slug:         "test-article",
			LanguageCode: "en",
		},
		{
			ID:           2,
			Title:        "مقاله آزمایشی",
			Slug:         "test-article-fa",
			LanguageCode: "fa",
		},
	}
	
	// Test Value method
	value, err := translations.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)
	
	// Test Scan method
	var scanned models.Translations
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Equal(t, len(translations), len(scanned))
	assert.Equal(t, translations[0].ID, scanned[0].ID)
	assert.Equal(t, translations[1].LanguageCode, scanned[1].LanguageCode)
	
	// Test Scan with nil
	var nilScanned models.Translations
	err = nilScanned.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, nilScanned)
	
	// Test Scan with invalid data
	var invalidScanned models.Translations
	err = invalidScanned.Scan(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scan")
}

func TestLanguageConfig_Methods(t *testing.T) {
	config := &models.LanguageConfig{
		DefaultLanguage:  "fa",
		FallbackLanguage: "en",
		ActiveLanguages:  []string{"fa", "en", "ar"},
		RTLLanguages:     []string{"fa", "ar"},
	}
	
	// Test IsRTL
	assert.True(t, config.IsRTL("fa"))
	assert.True(t, config.IsRTL("ar"))
	assert.False(t, config.IsRTL("en"))
	assert.False(t, config.IsRTL("de"))
	
	// Test IsActive
	assert.True(t, config.IsActive("fa"))
	assert.True(t, config.IsActive("en"))
	assert.True(t, config.IsActive("ar"))
	assert.False(t, config.IsActive("de"))
	
	// Test GetFallbackLanguage
	assert.Equal(t, "en", config.GetFallbackLanguage())
	
	// Test with empty fallback
	config.FallbackLanguage = ""
	assert.Equal(t, "fa", config.GetFallbackLanguage())
}

func TestMultilingualArticle_ContentWithTranslations(t *testing.T) {
	article := &models.MultilingualArticle{
		Article: models.Article{
			ID:           1,
			Title:        "Test Article",
			LanguageCode: "en",
		},
		TranslationGroupID: func() *uint64 { id := uint64(100); return &id }(),
		Translations: models.Translations{
			{
				ID:           2,
				Title:        "مقاله آزمایشی",
				LanguageCode: "fa",
			},
		},
	}
	
	// Test ContentWithTranslations interface methods
	assert.Equal(t, uint64(1), article.GetID())
	assert.Equal(t, "en", article.GetLanguageCode())
	assert.Equal(t, uint64(100), *article.GetTranslationGroupID())
	assert.Equal(t, 1, len(article.GetTranslations()))
	
	// Test SetTranslations
	newTranslations := models.Translations{
		{
			ID:           3,
			Title:        "مقالة اختبار",
			LanguageCode: "ar",
		},
	}
	article.SetTranslations(newTranslations)
	assert.Equal(t, 1, len(article.GetTranslations()))
	assert.Equal(t, "ar", article.GetTranslations()[0].LanguageCode)
}

// Benchmark tests
func BenchmarkExtractLanguageFromURL(b *testing.B) {
	paths := []string{
		"/en/articles/test-article",
		"/articles/test-article",
		"/ar/categories/technology",
		"/fa/tags/programming",
		"/api/v1/articles",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		ExtractLanguageFromURL(path)
	}
}

func BenchmarkLanguageConfig_IsRTL(b *testing.B) {
	config := &models.LanguageConfig{
		RTLLanguages: []string{"fa", "ar", "he", "ur"},
	}
	
	languages := []string{"fa", "en", "ar", "de", "he"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lang := languages[i%len(languages)]
		config.IsRTL(lang)
	}
}

// Integration test helpers
func createTestArticleWithLanguage(t *testing.T, db *sql.DB, title, slug, languageCode string) uint64 {
	query := `
		INSERT INTO articles (title, slug, content, author_id, category_id, language_code, status, published_at)
		VALUES ($1, $2, 'Test content', 1, 1, $3, 'published', NOW())
		RETURNING id
	`
	
	var articleID uint64
	err := db.QueryRow(query, title, slug, languageCode).Scan(&articleID)
	require.NoError(t, err)
	
	return articleID
}

func createTestTranslationGroup(t *testing.T, db *sql.DB, groupType string, contentIDs []uint64) uint64 {
	service := NewMultilingualService(db)
	groupID, err := service.CreateTranslationGroup(groupType, contentIDs)
	require.NoError(t, err)
	return groupID
}

// Test data cleanup
func cleanupTestData(t *testing.T, db *sql.DB) {
	queries := []string{
		"DELETE FROM article_tags WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM articles WHERE created_at >= NOW() - INTERVAL '1 hour'",
		"DELETE FROM translation_groups WHERE created_at >= NOW() - INTERVAL '1 hour'",
	}
	
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}
}