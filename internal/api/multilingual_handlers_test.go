package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
)

// MockMultilingualService is a mock implementation of MultilingualService
type MockMultilingualService struct {
	mock.Mock
}

func (m *MockMultilingualService) GetLanguages() ([]models.Language, error) {
	args := m.Called()
	return args.Get(0).([]models.Language), args.Error(1)
}

func (m *MockMultilingualService) GetActiveLanguages() ([]models.Language, error) {
	args := m.Called()
	return args.Get(0).([]models.Language), args.Error(1)
}

func (m *MockMultilingualService) GetLanguageConfig() (*models.LanguageConfig, error) {
	args := m.Called()
	return args.Get(0).(*models.LanguageConfig), args.Error(1)
}

func (m *MockMultilingualService) CreateTranslationGroup(groupType string, contentIDs []uint64) (uint64, error) {
	args := m.Called(groupType, contentIDs)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockMultilingualService) GetArticleTranslations(articleID uint64) (*models.MultilingualArticle, error) {
	args := m.Called(articleID)
	return args.Get(0).(*models.MultilingualArticle), args.Error(1)
}

func (m *MockMultilingualService) GetArticlesByLanguage(languageCode, fallbackLanguage string, limit, offset int) ([]models.MultilingualArticle, error) {
	args := m.Called(languageCode, fallbackLanguage, limit, offset)
	return args.Get(0).([]models.MultilingualArticle), args.Error(1)
}

func (m *MockMultilingualService) GetCategoriesByLanguage(languageCode string) ([]models.MultilingualCategory, error) {
	args := m.Called(languageCode)
	return args.Get(0).([]models.MultilingualCategory), args.Error(1)
}

func (m *MockMultilingualService) GetTagsByLanguage(languageCode string) ([]models.MultilingualTag, error) {
	args := m.Called(languageCode)
	return args.Get(0).([]models.MultilingualTag), args.Error(1)
}

func (m *MockMultilingualService) GenerateLanguageRouteInfo(contentType, slug, languageCode string) (*models.LanguageRouteInfo, error) {
	args := m.Called(contentType, slug, languageCode)
	return args.Get(0).(*models.LanguageRouteInfo), args.Error(1)
}

func (m *MockMultilingualService) ValidateLanguageCode(languageCode string) error {
	args := m.Called(languageCode)
	return args.Error(0)
}

func (m *MockMultilingualService) GetDefaultLanguage() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMultilingualService) IsRTLLanguage(languageCode string) (bool, error) {
	args := m.Called(languageCode)
	return args.Bool(0), args.Error(1)
}

func setupMultilingualTestRouter() (*gin.Engine, *MockMultilingualService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockService := &MockMultilingualService{}
	handlers := NewMultilingualHandlers(mockService)
	
	api := router.Group("/api/v1")
	RegisterMultilingualRoutes(api, handlers)
	
	return router, mockService
}

func TestMultilingualHandlers_GetLanguages(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedLanguages := []models.Language{
		{
			Code:       "fa",
			Name:       "Persian",
			NativeName: "فارسی",
			Direction:  "rtl",
			IsActive:   true,
			SortOrder:  1,
		},
		{
			Code:       "en",
			Name:       "English",
			NativeName: "English",
			Direction:  "ltr",
			IsActive:   true,
			SortOrder:  2,
		},
	}
	
	mockService.On("GetLanguages").Return(expectedLanguages, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/languages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	languages, ok := response["languages"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(languages))
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetActiveLanguages(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedLanguages := []models.Language{
		{
			Code:       "fa",
			Name:       "Persian",
			NativeName: "فارسی",
			Direction:  "rtl",
			IsActive:   true,
			SortOrder:  1,
		},
	}
	
	mockService.On("GetActiveLanguages").Return(expectedLanguages, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/languages/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	languages, ok := response["languages"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(languages))
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetLanguageConfig(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedConfig := &models.LanguageConfig{
		DefaultLanguage:  "fa",
		FallbackLanguage: "fa",
		ActiveLanguages:  []string{"fa", "en"},
		RTLLanguages:     []string{"fa"},
	}
	
	mockService.On("GetLanguageConfig").Return(expectedConfig, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/languages/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	config, ok := response["config"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "fa", config["default_language"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_CreateTranslationGroup(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	// Test successful creation
	mockService.On("CreateTranslationGroup", "article", []uint64{1, 2}).Return(uint64(100), nil)
	
	requestBody := models.TranslationRequest{
		GroupType:  "article",
		ContentIDs: []uint64{1, 2},
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/translations/groups", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, float64(100), response["group_id"])
	assert.Contains(t, response["message"], "successfully")
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_CreateTranslationGroup_ValidationErrors(t *testing.T) {
	router, _ := setupMultilingualTestRouter()
	
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
		{
			name: "Missing group type",
			requestBody: models.TranslationRequest{
				ContentIDs: []uint64{1, 2},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "group_type is required",
		},
		{
			name: "Insufficient content IDs",
			requestBody: models.TranslationRequest{
				GroupType:  "article",
				ContentIDs: []uint64{1},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "at least 2 content IDs are required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonBody []byte
			var err error
			
			if str, ok := tt.requestBody.(string); ok {
				jsonBody = []byte(str)
			} else {
				jsonBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}
			
			req, _ := http.NewRequest("POST", "/api/v1/translations/groups", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Contains(t, response["message"], tt.expectedError)
		})
	}
}

func TestMultilingualHandlers_GetArticleTranslations(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedArticle := &models.MultilingualArticle{
		Article: models.Article{
			ID:           1,
			Title:        "Test Article",
			Slug:         "test-article",
			LanguageCode: "en",
		},
		LanguageName:      "English",
		LanguageDirection: "ltr",
		Translations: models.Translations{
			{
				ID:           2,
				Title:        "مقاله آزمایشی",
				Slug:         "test-article-fa",
				LanguageCode: "fa",
			},
		},
	}
	
	mockService.On("GetArticleTranslations", uint64(1)).Return(expectedArticle, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/articles/1/translations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	article, ok := response["article"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test Article", article["title"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetArticleTranslations_NotFound(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	mockService.On("GetArticleTranslations", uint64(999)).Return((*models.MultilingualArticle)(nil), assert.AnError)
	
	req, _ := http.NewRequest("GET", "/api/v1/articles/999/translations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetArticlesByLanguage(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedArticles := []models.MultilingualArticle{
		{
			Article: models.Article{
				ID:           1,
				Title:        "Test Article",
				LanguageCode: "en",
			},
			IsFallback: false,
		},
	}
	
	mockService.On("ValidateLanguageCode", "en").Return(nil)
	mockService.On("GetDefaultLanguage").Return("fa")
	mockService.On("GetArticlesByLanguage", "en", "fa", 20, 0).Return(expectedArticles, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/articles/language/en", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	articles, ok := response["articles"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(articles))
	assert.Equal(t, "en", response["language_code"])
	assert.Equal(t, float64(1), response["count"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetArticlesByLanguage_WithPagination(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedArticles := []models.MultilingualArticle{}
	
	mockService.On("ValidateLanguageCode", "fa").Return(nil)
	mockService.On("GetDefaultLanguage").Return("fa")
	mockService.On("GetArticlesByLanguage", "fa", "fa", 10, 20).Return(expectedArticles, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/articles/language/fa?limit=10&offset=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, float64(10), response["limit"])
	assert.Equal(t, float64(20), response["offset"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_ValidateLanguageCode(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	// Test valid language code
	mockService.On("ValidateLanguageCode", "fa").Return(nil)
	mockService.On("IsRTLLanguage", "fa").Return(true, nil)
	mockService.On("GetDefaultLanguage").Return("fa")
	
	req, _ := http.NewRequest("GET", "/api/v1/languages/fa/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["valid"].(bool))
	assert.Equal(t, "fa", response["code"])
	assert.True(t, response["is_rtl"].(bool))
	assert.True(t, response["is_default"].(bool))
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_ValidateLanguageCode_Invalid(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	mockService.On("ValidateLanguageCode", "xx").Return(assert.AnError)
	
	req, _ := http.NewRequest("GET", "/api/v1/languages/xx/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["valid"].(bool))
	assert.NotEmpty(t, response["error"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetLanguageRouteInfo(t *testing.T) {
	router, mockService := setupMultilingualTestRouter()
	
	expectedRouteInfo := &models.LanguageRouteInfo{
		LanguageCode:  "en",
		IsDefault:     false,
		URLPrefix:     "/en",
		Direction:     "ltr",
		AlternateURLs: map[string]string{
			"fa": "/articles/test-article",
			"ar": "/ar/articles/test-article",
		},
	}
	
	mockService.On("ValidateLanguageCode", "en").Return(nil)
	mockService.On("GenerateLanguageRouteInfo", "articles", "test-article", "en").Return(expectedRouteInfo, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/routes/articles/test-article/language/en", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	routeInfo, ok := response["route_info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "en", routeInfo["language_code"])
	assert.Equal(t, "/en", routeInfo["url_prefix"])
	
	mockService.AssertExpectations(t)
}

func TestMultilingualHandlers_GetLanguageRouteInfo_InvalidContentType(t *testing.T) {
	router, _ := setupMultilingualTestRouter()
	
	req, _ := http.NewRequest("GET", "/api/v1/routes/invalid/test-article/language/en", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response["message"], "Invalid content type")
}

// Test middleware functions
func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		expected       string
	}{
		{
			name:           "English preference",
			acceptLanguage: "en-US,en;q=0.9,fa;q=0.8",
			expected:       "en",
		},
		{
			name:           "Persian preference",
			acceptLanguage: "fa,en-US;q=0.9",
			expected:       "fa",
		},
		{
			name:           "Empty header",
			acceptLanguage: "",
			expected:       "fa",
		},
		{
			name:           "Complex header",
			acceptLanguage: "fr-CH, fr;q=0.9, en;q=0.8, de;q=0.7, *;q=0.5",
			expected:       "fr",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAcceptLanguage(tt.acceptLanguage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLanguageFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test with language in context
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("language_code", "en")
	
	lang := GetLanguageFromContext(c)
	assert.Equal(t, "en", lang)
	
	// Test without language in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	lang2 := GetLanguageFromContext(c2)
	assert.Equal(t, "fa", lang2) // Default
	
	// Test with invalid type in context
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Set("language_code", 123)
	lang3 := GetLanguageFromContext(c3)
	assert.Equal(t, "fa", lang3) // Default
}

func TestIsRTLFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test with RTL in context
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("is_rtl", true)
	
	isRTL := IsRTLFromContext(c)
	assert.True(t, isRTL)
	
	// Test without RTL in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	isRTL2 := IsRTLFromContext(c2)
	assert.True(t, isRTL2) // Default to true (Persian)
	
	// Test with LTR
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Set("is_rtl", false)
	isRTL3 := IsRTLFromContext(c3)
	assert.False(t, isRTL3)
}

// Benchmark tests
func BenchmarkMultilingualHandlers_GetLanguages(b *testing.B) {
	router, mockService := setupMultilingualTestRouter()
	
	languages := []models.Language{
		{Code: "fa", Name: "Persian", NativeName: "فارسی", Direction: "rtl", IsActive: true},
		{Code: "en", Name: "English", NativeName: "English", Direction: "ltr", IsActive: true},
		{Code: "ar", Name: "Arabic", NativeName: "العربية", Direction: "rtl", IsActive: true},
	}
	
	mockService.On("GetLanguages").Return(languages, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/languages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}