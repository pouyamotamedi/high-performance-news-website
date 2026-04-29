package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Mock services for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(req *services.CreateUserRequest, currentUser *models.User) (*models.User, error) {
	args := m.Called(req, currentUser)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetByID(id uint64, currentUser *models.User) (*models.User, error) {
	args := m.Called(id, currentUser)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Update(id uint64, req *services.UpdateUserRequest, currentUser *models.User) (*models.User, error) {
	args := m.Called(id, req, currentUser)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Delete(id uint64, currentUser *models.User) error {
	args := m.Called(id, currentUser)
	return args.Error(0)
}

func (m *MockUserService) List(limit, offset int, currentUser *models.User) ([]*models.User, int, error) {
	args := m.Called(limit, offset, currentUser)
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *MockUserService) Login(req *services.LoginRequest) (*services.LoginResponse, error) {
	args := m.Called(req)
	return args.Get(0).(*services.LoginResponse), args.Error(1)
}

func (m *MockUserService) RefreshToken(refreshToken string) (*auth.TokenPair, error) {
	args := m.Called(refreshToken)
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockUserService) ChangePassword(userID uint64, req *services.ChangePasswordRequest, currentUser *models.User) error {
	args := m.Called(userID, req, currentUser)
	return args.Error(0)
}

type MockArticleService struct {
	mock.Mock
}

func (m *MockArticleService) Create(ctx context.Context, article *models.Article, currentUser *models.User) (*models.Article, error) {
	args := m.Called(ctx, article, currentUser)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	args := m.Called(ctx, slug)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) Update(ctx context.Context, id uint64, req interface{}, currentUser *models.User) (*models.Article, error) {
	args := m.Called(ctx, id, req, currentUser)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) Delete(ctx context.Context, id uint64, currentUser *models.User) error {
	args := m.Called(ctx, id, currentUser)
	return args.Error(0)
}

func (m *MockArticleService) List(ctx context.Context, limit, offset int, filters services.ArticleFilters, sortBy, sortOrder string) ([]models.Article, int, error) {
	args := m.Called(ctx, limit, offset, filters, sortBy, sortOrder)
	return args.Get(0).([]models.Article), args.Int(1), args.Error(2)
}

func (m *MockArticleService) Publish(ctx context.Context, id uint64, currentUser *models.User) (*models.Article, error) {
	args := m.Called(ctx, id, currentUser)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) GetTrending(ctx context.Context, limit, hours int) ([]models.Article, error) {
	args := m.Called(ctx, limit, hours)
	return args.Get(0).([]models.Article), args.Error(1)
}

func (m *MockArticleService) GetPopular(ctx context.Context, limit, days int) ([]models.Article, error) {
	args := m.Called(ctx, limit, days)
	return args.Get(0).([]models.Article), args.Error(1)
}

func (m *MockArticleService) BulkCreate(ctx context.Context, articles []models.Article, currentUser *models.User) ([]models.Article, error) {
	args := m.Called(ctx, articles, currentUser)
	return args.Get(0).([]models.Article), args.Error(1)
}

func (m *MockArticleService) RecordView(ctx context.Context, articleID uint64, ipAddress, userAgent, referer string) error {
	args := m.Called(ctx, articleID, ipAddress, userAgent, referer)
	return args.Error(0)
}

type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SearchArticles(ctx context.Context, filters services.SearchFilters, limit, offset int) ([]services.SearchResult, services.SearchFacets, int, float64, error) {
	args := m.Called(ctx, filters, limit, offset)
	return args.Get(0).([]services.SearchResult), args.Get(1).(services.SearchFacets), args.Int(2), args.Get(3).(float64), args.Error(4)
}

func (m *MockSearchService) GetSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSearchService) GetPopularSearches(ctx context.Context, limit, days int) ([]services.PopularSearch, error) {
	args := m.Called(ctx, limit, days)
	return args.Get(0).([]services.PopularSearch), args.Error(1)
}

// Test setup helpers
func setupTestHandler() (*APIHandler, *MockUserService, *MockArticleService, *MockSearchService) {
	mockUserService := &MockUserService{}
	mockArticleService := &MockArticleService{}
	mockSearchService := &MockSearchService{}
	
	handler := NewAPIHandler(mockUserService, mockArticleService, mockSearchService)
	
	return handler, mockUserService, mockArticleService, mockSearchService
}

func setupTestRouter(handler *APIHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add test user to context middleware
	router.Use(func(c *gin.Context) {
		// Add a test user for authenticated endpoints
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			Role:     models.RoleAdmin,
			IsActive: true,
		}
		c.Set("user", testUser)
		c.Next()
	})
	
	return router
}

// Test cases

func TestHealthCheck(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.GET("/health", handler.HealthCheck)
	
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "version")
}

func TestCreateArticle(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.POST("/articles", handler.CreateArticle)
	
	// Mock the service call
	expectedArticle := &models.Article{
		ID:         1,
		Title:      "Test Article",
		Content:    "Test content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	mockArticleService.On("Create", mock.Anything, mock.AnythingOfType("*models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticle, nil)
	
	// Create request
	requestBody := CreateArticleRequest{
		Title:      "Test Article",
		Content:    "Test content",
		CategoryID: 1,
		Status:     "draft",
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Article created successfully", response.Message)
	
	mockArticleService.AssertExpectations(t)
}

func TestGetArticle(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.GET("/articles/:id", handler.GetArticle)
	
	// Mock the service call
	expectedArticle := &models.Article{
		ID:         1,
		Title:      "Test Article",
		Content:    "Test content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	mockArticleService.On("GetByID", mock.Anything, uint64(1)).Return(expectedArticle, nil)
	
	req, _ := http.NewRequest("GET", "/articles/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	// Verify the article data is returned
	articleData := response.Data.(map[string]interface{})
	assert.Equal(t, float64(1), articleData["id"])
	assert.Equal(t, "Test Article", articleData["title"])
	
	mockArticleService.AssertExpectations(t)
}

func TestListArticles(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.GET("/articles", handler.ListArticles)
	
	// Mock the service call
	expectedArticles := []models.Article{
		{
			ID:         1,
			Title:      "Article 1",
			AuthorID:   1,
			CategoryID: 1,
			Status:     "published",
		},
		{
			ID:         2,
			Title:      "Article 2",
			AuthorID:   1,
			CategoryID: 1,
			Status:     "published",
		},
	}
	
	mockArticleService.On("List", mock.Anything, 20, 0, mock.AnythingOfType("services.ArticleFilters"), "published_at", "desc").Return(expectedArticles, 2, nil)
	
	req, _ := http.NewRequest("GET", "/articles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response ArticleListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Articles, 2)
	assert.Equal(t, 2, response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.Page)
	
	mockArticleService.AssertExpectations(t)
}

func TestBulkCreateArticles(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.POST("/articles/bulk", handler.BulkCreateArticles)
	
	// Mock the service call
	expectedArticles := []models.Article{
		{ID: 1, Title: "Article 1", AuthorID: 1, CategoryID: 1, Status: "draft"},
		{ID: 2, Title: "Article 2", AuthorID: 1, CategoryID: 1, Status: "draft"},
	}
	
	mockArticleService.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticles, nil)
	
	// Create request
	requestBody := BulkCreateArticleRequest{
		Articles: []CreateArticleRequest{
			{Title: "Article 1", Content: "Content 1", CategoryID: 1, Status: "draft"},
			{Title: "Article 2", Content: "Content 2", CategoryID: 1, Status: "draft"},
		},
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/articles/bulk", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Articles created successfully", response.Message)
	
	mockArticleService.AssertExpectations(t)
}

func TestLogin(t *testing.T) {
	handler, mockUserService, _, _ := setupTestHandler()
	router := gin.New() // Don't use setupTestRouter for login test
	
	router.POST("/login", handler.Login)
	
	// Mock the service call
	expectedResponse := &services.LoginResponse{
		User: &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			Role:     models.RoleAdmin,
		},
		Tokens: &auth.TokenPair{
			AccessToken:  "access_token",
			RefreshToken: "refresh_token",
		},
	}
	
	mockUserService.On("Login", mock.AnythingOfType("*services.LoginRequest")).Return(expectedResponse, nil)
	
	// Create request
	requestBody := LoginAPIRequest{
		Username: "testuser",
		Password: "password123",
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Login successful", response.Message)
	
	mockUserService.AssertExpectations(t)
}

func TestCreateUser(t *testing.T) {
	handler, mockUserService, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.POST("/users", handler.CreateUser)
	
	// Mock the service call
	expectedUser := &models.User{
		ID:        2,
		Username:  "newuser",
		Email:     "newuser@example.com",
		Role:      models.RoleReporter,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	mockUserService.On("Create", mock.AnythingOfType("*services.CreateUserRequest"), mock.AnythingOfType("*models.User")).Return(expectedUser, nil)
	
	// Create request
	requestBody := CreateUserAPIRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "password123",
		Role:     models.RoleReporter,
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User created successfully", response.Message)
	
	mockUserService.AssertExpectations(t)
}

func TestSearchArticles(t *testing.T) {
	handler, _, _, mockSearchService := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.GET("/search", handler.SearchArticles)
	
	// Mock the service call
	expectedResults := []services.SearchResult{
		{
			ID:          1,
			Title:       "Test Article",
			Slug:        "test-article",
			Excerpt:     "Test excerpt",
			AuthorID:    1,
			AuthorName:  "Test Author",
			CategoryID:  1,
			Category:    "Test Category",
			Tags:        []string{"test", "article"},
			PublishedAt: time.Now().Format(time.RFC3339),
			ViewCount:   100,
			Score:       0.95,
		},
	}
	
	expectedFacets := services.SearchFacets{
		Categories: []services.FacetItem{{ID: 1, Name: "Test Category", Count: 1}},
		Tags:       []services.FacetItem{{ID: 1, Name: "test", Count: 1}},
		Authors:    []services.FacetItem{{ID: 1, Name: "Test Author", Count: 1}},
	}
	
	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(expectedResults, expectedFacets, 1, 15.5, nil)
	
	req, _ := http.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response SearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Results, 1)
	assert.Equal(t, "test", response.Query)
	assert.Equal(t, 15.5, response.TotalTime)
	assert.Equal(t, 1, response.Pagination.Total)
	
	mockSearchService.AssertExpectations(t)
}