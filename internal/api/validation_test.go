package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"high-performance-news-website/internal/models"
)

// TestInputValidation tests comprehensive input validation

func TestArticleValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles", handler.CreateArticle)

	tests := []struct {
		name           string
		input          CreateArticleRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid article",
			input: CreateArticleRequest{
				Title:      "Valid Article Title",
				Content:    "Valid article content with sufficient length",
				Excerpt:    "Valid excerpt",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusCreated,
			expectedError:  "",
		},
		{
			name: "Empty title",
			input: CreateArticleRequest{
				Title:      "",
				Content:    "Valid content",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
		{
			name: "Title too long",
			input: CreateArticleRequest{
				Title:      strings.Repeat("a", 256), // Exceeds 255 character limit
				Content:    "Valid content",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title must be less than 255 characters",
		},
		{
			name: "Empty content",
			input: CreateArticleRequest{
				Title:      "Valid Title",
				Content:    "",
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "content is required",
		},
		{
			name: "Invalid status",
			input: CreateArticleRequest{
				Title:      "Valid Title",
				Content:    "Valid content",
				CategoryID: 1,
				Status:     "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "status must be one of: draft, published",
		},
		{
			name: "Missing category",
			input: CreateArticleRequest{
				Title:   "Valid Title",
				Content: "Valid content",
				Status:  "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "category_id is required",
		},
		{
			name: "Excerpt too long",
			input: CreateArticleRequest{
				Title:      "Valid Title",
				Content:    "Valid content",
				Excerpt:    strings.Repeat("a", 501), // Exceeds 500 character limit
				CategoryID: 1,
				Status:     "draft",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "excerpt must be less than 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.input)
			req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Message, tt.expectedError)
			}
		})
	}
}

func TestUserValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/users", handler.CreateUser)

	tests := []struct {
		name           string
		input          CreateUserAPIRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid user",
			input: CreateUserAPIRequest{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			expectedStatus: http.StatusCreated,
			expectedError:  "",
		},
		{
			name: "Username too short",
			input: CreateUserAPIRequest{
				Username: "ab", // Less than 3 characters
				Email:    "valid@example.com",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "username must be at least 3 characters",
		},
		{
			name: "Username too long",
			input: CreateUserAPIRequest{
				Username: strings.Repeat("a", 51), // Exceeds 50 characters
				Email:    "valid@example.com",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "username must be less than 50 characters",
		},
		{
			name: "Invalid email format",
			input: CreateUserAPIRequest{
				Username: "validuser",
				Email:    "invalid-email",
				Password: "password123",
				Role:     models.RoleReporter,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email must be a valid email address",
		},
		{
			name: "Password too short",
			input: CreateUserAPIRequest{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "short", // Less than 8 characters
				Role:     models.RoleReporter,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
		{
			name: "Invalid role",
			input: CreateUserAPIRequest{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "password123",
				Role:     "invalid_role",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "role must be one of: admin, editor, reporter, contributor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.input)
			req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Message, tt.expectedError)
			}
		})
	}
}

func TestSearchValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/search", handler.SearchArticles)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid search query",
			queryParams:    "?q=test",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Empty query",
			queryParams:    "?q=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query parameter 'q' cannot be empty",
		},
		{
			name:           "Missing query parameter",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query parameter 'q' cannot be empty",
		},
		{
			name:           "Query too long",
			queryParams:    "?q=" + strings.Repeat("a", 201), // Exceeds 200 characters
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query must be less than 200 characters",
		},
		{
			name:           "Valid query with filters",
			queryParams:    "?q=test&categories=1,2&tags=3,4&authors=5,6",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Valid pagination",
			queryParams:    "?q=test&page=2&limit=50",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Invalid page number",
			queryParams:    "?q=test&page=0",
			expectedStatus: http.StatusOK, // Should default to page 1
			expectedError:  "",
		},
		{
			name:           "Invalid limit",
			queryParams:    "?q=test&limit=0",
			expectedStatus: http.StatusOK, // Should default to 20
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Message, tt.expectedError)
			}
		})
	}
}

func TestBulkOperationValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles/bulk", handler.BulkCreateArticles)

	tests := []struct {
		name           string
		input          BulkCreateArticleRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid bulk request",
			input: BulkCreateArticleRequest{
				Articles: []CreateArticleRequest{
					{
						Title:      "Article 1",
						Content:    "Content 1",
						CategoryID: 1,
						Status:     "draft",
					},
					{
						Title:      "Article 2",
						Content:    "Content 2",
						CategoryID: 1,
						Status:     "draft",
					},
				},
			},
			expectedStatus: http.StatusCreated,
			expectedError:  "",
		},
		{
			name: "Empty articles array",
			input: BulkCreateArticleRequest{
				Articles: []CreateArticleRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "articles array cannot be empty",
		},
		{
			name: "Too many articles",
			input: BulkCreateArticleRequest{
				Articles: make([]CreateArticleRequest, 1001), // Exceeds 1000 limit
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "maximum 1000 articles per request",
		},
		{
			name: "Invalid article in bulk",
			input: BulkCreateArticleRequest{
				Articles: []CreateArticleRequest{
					{
						Title:      "", // Invalid: empty title
						Content:    "Content 1",
						CategoryID: 1,
						Status:     "draft",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill in the articles for the "too many" test case
			if len(tt.input.Articles) == 1001 {
				for i := range tt.input.Articles {
					tt.input.Articles[i] = CreateArticleRequest{
						Title:      "Article " + string(rune(i)),
						Content:    "Content " + string(rune(i)),
						CategoryID: 1,
						Status:     "draft",
					}
				}
			}

			jsonBody, _ := json.Marshal(tt.input)
			req, _ := http.NewRequest("POST", "/articles/bulk", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response.Message, tt.expectedError)
			}
		})
	}
}

func TestPaginationValidation(t *testing.T) {
	handler, _, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles", handler.ListArticles)

	tests := []struct {
		name           string
		queryParams    string
		expectedPage   int
		expectedLimit  int
		expectedStatus int
	}{
		{
			name:           "Default pagination",
			queryParams:    "",
			expectedPage:   1,
			expectedLimit:  20,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid custom pagination",
			queryParams:    "?page=2&limit=50",
			expectedPage:   2,
			expectedLimit:  50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid page (negative)",
			queryParams:    "?page=-1",
			expectedPage:   1, // Should default to 1
			expectedLimit:  20,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid page (zero)",
			queryParams:    "?page=0",
			expectedPage:   1, // Should default to 1
			expectedLimit:  20,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid limit (too high)",
			queryParams:    "?limit=2000",
			expectedPage:   1,
			expectedLimit:  20, // Should default to 20 when exceeding max
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid limit (negative)",
			queryParams:    "?limit=-10",
			expectedPage:   1,
			expectedLimit:  20, // Should default to 20
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-numeric values",
			queryParams:    "?page=abc&limit=xyz",
			expectedPage:   1, // Should default to 1
			expectedLimit:  20, // Should default to 20
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Maximum valid limit",
			queryParams:    "?limit=1000",
			expectedPage:   1,
			expectedLimit:  1000,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/articles"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response ArticleListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPage, response.Pagination.Page)
				assert.Equal(t, tt.expectedLimit, response.Pagination.Limit)
			}
		})
	}
}  