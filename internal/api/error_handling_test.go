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
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
)

// TestErrorHandling tests various error scenarios

func TestCreateArticleValidationErrors(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles", handler.CreateArticle)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Empty request body",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Missing title",
			requestBody: CreateArticleRequest{
				Content:    "Test content",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Missing content",
			requestBody: CreateArticleRequest{
				Title:      "Test Article",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Invalid status",
			requestBody: CreateArticleRequest{
				Title:      "Test Article",
				Content:    "Test content",
				CategoryID: 1,
				Status:     "invalid_status",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Title too long",
			requestBody: CreateArticleRequest{
				Title:      string(make([]byte, 300)), // 300 characters, exceeds 255 limit
				Content:    "Test content",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestGetArticleNotFound(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles/:id", handler.GetArticle)

	// Mock service to return not found error
	mockArticleService.On("GetByID", mock.Anything, uint64(999)).Return((*models.Article)(nil), auth.ErrUserNotFound)

	req, _ := http.NewRequest("GET", "/articles/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, ErrCodeNotFound, response.Code)
}

func TestGetArticleInvalidID(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles/:id", handler.GetArticle)

	req, _ := http.NewRequest("GET", "/articles/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, ErrCodeValidation, response.Code)
}

func TestBulkCreateArticlesValidationErrors(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles/bulk", handler.BulkCreateArticles)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "Empty articles array",
			requestBody: BulkCreateArticleRequest{
				Articles: []CreateArticleRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Too many articles",
			requestBody: BulkCreateArticleRequest{
				Articles: make([]CreateArticleRequest, 1001), // Exceeds 1000 limit
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
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
				assert.NoError(t, err)
			}

			req, _ := http.NewRequest("POST", "/articles/bulk", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestSearchValidationErrors(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/search", handler.SearchArticles)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Missing query parameter",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name:           "Empty query parameter",
			queryParams:    "?q=",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name:           "Query too long",
			queryParams:    "?q=" + string(make([]byte, 201)), // Exceeds 200 character limit
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestUserManagementErrors(t *testing.T) {
	handler, mockUserService, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/users", handler.CreateUser)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockError      error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "Duplicate username",
			requestBody: CreateUserAPIRequest{
				Username: "duplicate",
				Email:    "test@example.com",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			mockError:      &models.ValidationError{Message: "Username already exists", Fields: []string{"username"}},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Invalid email format",
			requestBody: CreateUserAPIRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			mockError:      &models.ValidationError{Message: "Invalid email format", Fields: []string{"email"}},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Insufficient permissions",
			requestBody: CreateUserAPIRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Role:     models.RoleAdmin,
			},
			mockError:      auth.ErrInsufficientPermissions,
			expectedStatus: http.StatusForbidden,
			expectedCode:   ErrCodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserService.On("Create", mock.AnythingOfType("*services.CreateUserRequest"), mock.AnythingOfType("*models.User")).Return((*models.User)(nil), tt.mockError).Once()

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestLoginErrors(t *testing.T) {
	handler, mockUserService, _, _ := setupTestHandler()
	router := gin.New() // Don't use setupTestRouter for login test
	router.POST("/login", handler.Login)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockError      error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "Invalid credentials",
			requestBody: LoginAPIRequest{
				Username: "wronguser",
				Password: "wrongpassword",
			},
			mockError:      auth.ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   ErrCodeUnauthorized,
		},
		{
			name: "User not found",
			requestBody: LoginAPIRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			mockError:      auth.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrCodeNotFound,
		},
		{
			name: "Missing username",
			requestBody: LoginAPIRequest{
				Password: "password123",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
		{
			name: "Missing password",
			requestBody: LoginAPIRequest{
				Username: "testuser",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockError != nil {
				mockUserService.On("Login", mock.AnythingOfType("*services.LoginRequest")).Return((*services.LoginResponse)(nil), tt.mockError).Once()
			}

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles", handler.ListArticles)

	// Mock empty results
	mockArticleService.On("List", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("services.ArticleFilters"), mock.Anything, mock.Anything).Return([]models.Article{}, 0, nil)

	tests := []struct {
		name        string
		queryParams string
		expectPage  int
		expectLimit int
	}{
		{
			name:        "Default pagination",
			queryParams: "",
			expectPage:  1,
			expectLimit: 20,
		},
		{
			name:        "Custom page and limit",
			queryParams: "?page=2&limit=50",
			expectPage:  2,
			expectLimit: 50,
		},
		{
			name:        "Invalid page (negative)",
			queryParams: "?page=-1",
			expectPage:  1, // Should default to 1
			expectLimit: 20,
		},
		{
			name:        "Invalid limit (too high)",
			queryParams: "?limit=2000",
			expectPage:  1,
			expectLimit: 20, // Should default to 20 when exceeding max
		},
		{
			name:        "Invalid page (zero)",
			queryParams: "?page=0",
			expectPage:  1, // Should default to 1
			expectLimit: 20,
		},
		{
			name:        "Non-numeric page",
			queryParams: "?page=abc",
			expectPage:  1, // Should default to 1
			expectLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/articles"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response ArticleListResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectPage, response.Pagination.Page)
			assert.Equal(t, tt.expectLimit, response.Pagination.Limit)
		})
	}
}

func TestContentTypeValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles", handler.CreateArticle)

	tests := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Valid JSON content type",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest, // Will fail validation but content type is accepted
		},
		{
			name:           "Missing content type",
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid content type",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
	}

	requestBody := CreateArticleRequest{
		Title:      "Test Article",
		Content:    "Test content",
		CategoryID: 1,
		Status:     "draft",
	}
	jsonBody, _ := json.Marshal(requestBody)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}