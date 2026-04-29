package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// TestFullArticleWorkflow tests the complete article lifecycle
func TestFullArticleWorkflow(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	// Setup routes
	router.POST("/articles", handler.CreateArticle)
	router.GET("/articles/:id", handler.GetArticle)
	router.PUT("/articles/:id", handler.UpdateArticle)
	router.POST("/articles/:id/publish", handler.PublishArticle)
	router.DELETE("/articles/:id", handler.DeleteArticle)

	// Step 1: Create a draft article
	createReq := CreateArticleRequest{
		Title:      "Integration Test Article",
		Content:    "This is a test article for integration testing",
		Excerpt:    "Test excerpt",
		CategoryID: 1,
		Status:     "draft",
		SEOData: models.SEOData{
			MetaTitle:       "Test Meta Title",
			MetaDescription: "Test meta description",
			Keywords:        []string{"test", "integration"},
		},
	}

	createdArticle := &models.Article{
		ID:         1,
		Title:      createReq.Title,
		Content:    createReq.Content,
		Excerpt:    createReq.Excerpt,
		AuthorID:   1,
		CategoryID: createReq.CategoryID,
		Status:     createReq.Status,
		SEOData:    createReq.SEOData,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockArticleService.On("Create", mock.Anything, mock.AnythingOfType("*models.Article"), mock.AnythingOfType("*models.User")).Return(createdArticle, nil).Once()

	jsonBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var createResponse SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Article created successfully", createResponse.Message)

	// Step 2: Retrieve the created article
	mockArticleService.On("GetByID", mock.Anything, uint64(1)).Return(createdArticle, nil).Once()

	req, _ = http.NewRequest("GET", "/articles/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var getResponse SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.NoError(t, err)

	// Step 3: Update the article
	updateReq := UpdateArticleRequest{
		Title:   stringPtr("Updated Integration Test Article"),
		Content: stringPtr("Updated content for integration testing"),
	}

	updatedArticle := *createdArticle
	updatedArticle.Title = *updateReq.Title
	updatedArticle.Content = *updateReq.Content
	updatedArticle.UpdatedAt = time.Now()

	mockArticleService.On("Update", mock.Anything, uint64(1), mock.AnythingOfType("*api.UpdateArticleRequest"), mock.AnythingOfType("*models.User")).Return(&updatedArticle, nil).Once()

	jsonBody, _ = json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", "/articles/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var updateResponse SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &updateResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Article updated successfully", updateResponse.Message)

	// Step 4: Publish the article
	publishedArticle := updatedArticle
	publishedArticle.Status = "published"
	now := time.Now()
	publishedArticle.PublishedAt = &now

	mockArticleService.On("Publish", mock.Anything, uint64(1), mock.AnythingOfType("*models.User")).Return(&publishedArticle, nil).Once()

	req, _ = http.NewRequest("POST", "/articles/1/publish", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var publishResponse SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &publishResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Article published successfully", publishResponse.Message)

	// Step 5: Delete the article
	mockArticleService.On("Delete", mock.Anything, uint64(1), mock.AnythingOfType("*models.User")).Return(nil).Once()

	req, _ = http.NewRequest("DELETE", "/articles/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var deleteResponse SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &deleteResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Article deleted successfully", deleteResponse.Message)

	mockArticleService.AssertExpectations(t)
}

// TestUserManagementWorkflow tests the complete user management lifecycle
func TestUserManagementWorkflow(t *testing.T) {
	handler, mockUserService, _, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	// Setup routes
	router.POST("/users", handler.CreateUser)
	router.GET("/users/:id", handler.GetUser)
	router.PUT("/users/:id", handler.UpdateUser)
	router.POST("/users/:id/change-password", handler.ChangePassword)
	router.DELETE("/users/:id", handler.DeleteUser)

	// Step 1: Create a new user
	createReq := CreateUserAPIRequest{
		Username:  "integrationtest",
		Email:     "integration@test.com",
		Password:  "password123",
		Role:      models.RoleReporter,
		FirstName: "Integration",
		LastName:  "Test",
		Bio:       "User for integration testing",
	}

	createdUser := &models.User{
		ID:        2,
		Username:  createReq.Username,
		Email:     createReq.Email,
		Role:      createReq.Role,
		FirstName: createReq.FirstName,
		LastName:  createReq.LastName,
		Bio:       createReq.Bio,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockUserService.On("Create", mock.AnythingOfType("*services.CreateUserRequest"), mock.AnythingOfType("*models.User")).Return(createdUser, nil).Once()

	jsonBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Step 2: Retrieve the created user
	mockUserService.On("GetByID", uint64(2), mock.AnythingOfType("*models.User")).Return(createdUser, nil).Once()

	req, _ = http.NewRequest("GET", "/users/2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Step 3: Update the user
	updateReq := UpdateUserAPIRequest{
		FirstName: stringPtr("Updated"),
		LastName:  stringPtr("Name"),
		Bio:       stringPtr("Updated bio for integration testing"),
	}

	updatedUser := *createdUser
	updatedUser.FirstName = *updateReq.FirstName
	updatedUser.LastName = *updateReq.LastName
	updatedUser.Bio = *updateReq.Bio
	updatedUser.UpdatedAt = time.Now()

	mockUserService.On("Update", uint64(2), mock.AnythingOfType("*services.UpdateUserRequest"), mock.AnythingOfType("*models.User")).Return(&updatedUser, nil).Once()

	jsonBody, _ = json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", "/users/2", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Step 4: Change password
	changePasswordReq := ChangePasswordAPIRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	}

	mockUserService.On("ChangePassword", uint64(2), mock.AnythingOfType("*services.ChangePasswordRequest"), mock.AnythingOfType("*models.User")).Return(nil).Once()

	jsonBody, _ = json.Marshal(changePasswordReq)
	req, _ = http.NewRequest("POST", "/users/2/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Step 5: Delete the user
	mockUserService.On("Delete", uint64(2), mock.AnythingOfType("*models.User")).Return(nil).Once()

	req, _ = http.NewRequest("DELETE", "/users/2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockUserService.AssertExpectations(t)
}

// TestSearchWorkflow tests the complete search functionality
func TestSearchWorkflow(t *testing.T) {
	handler, _, _, mockSearchService := setupTestHandler()
	router := setupTestRouter(handler)
	
	// Setup routes
	router.GET("/search", handler.SearchArticles)
	router.GET("/search/suggestions", handler.GetSearchSuggestions)
	router.GET("/search/popular", handler.GetPopularSearches)
	router.GET("/search/category/:category_id", handler.SearchByCategory)
	router.GET("/search/tag/:tag_id", handler.SearchByTag)

	// Step 1: Basic search
	searchResults := []services.SearchResult{
		{
			ID:          1,
			Title:       "Integration Test Article",
			Slug:        "integration-test-article",
			Excerpt:     "Test excerpt",
			AuthorID:    1,
			AuthorName:  "Test Author",
			CategoryID:  1,
			Category:    "Test Category",
			Tags:        []string{"integration", "test"},
			PublishedAt: time.Now().Format(time.RFC3339),
			ViewCount:   100,
			Score:       0.95,
			Highlights:  []string{"Integration Test Article"},
		},
	}

	facets := services.SearchFacets{
		Categories: []services.FacetItem{{ID: 1, Name: "Test Category", Count: 1}},
		Tags:       []services.FacetItem{{ID: 1, Name: "integration", Count: 1}},
		Authors:    []services.FacetItem{{ID: 1, Name: "Test Author", Count: 1}},
	}

	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(searchResults, facets, 1, 15.5, nil).Once()

	req, _ := http.NewRequest("GET", "/search?q=integration", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var searchResponse SearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &searchResponse)
	assert.NoError(t, err)
	assert.Len(t, searchResponse.Results, 1)
	assert.Equal(t, "integration", searchResponse.Query)

	// Step 2: Search suggestions
	suggestions := []string{"integration test", "integration testing", "integration guide"}
	mockSearchService.On("GetSuggestions", mock.Anything, "integ", 10).Return(suggestions, nil).Once()

	req, _ = http.NewRequest("GET", "/search/suggestions?q=integ", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var suggestionsResponse SearchSuggestionResponse
	err = json.Unmarshal(w.Body.Bytes(), &suggestionsResponse)
	assert.NoError(t, err)
	assert.Len(t, suggestionsResponse.Suggestions, 3)

	// Step 3: Popular searches
	popularSearches := []services.PopularSearch{
		{Query: "integration", Count: 100},
		{Query: "testing", Count: 80},
	}
	mockSearchService.On("GetPopularSearches", mock.Anything, 10, 7).Return(popularSearches, nil).Once()

	req, _ = http.NewRequest("GET", "/search/popular", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Step 4: Category search
	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(searchResults, facets, 1, 12.3, nil).Once()

	req, _ = http.NewRequest("GET", "/search/category/1?q=test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Step 5: Tag search
	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(searchResults, facets, 1, 18.7, nil).Once()

	req, _ = http.NewRequest("GET", "/search/tag/1?q=test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockSearchService.AssertExpectations(t)
}

// TestBulkOperationsWorkflow tests bulk operations
func TestBulkOperationsWorkflow(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles/bulk", handler.BulkCreateArticles)

	// Test different bulk sizes
	bulkSizes := []int{1, 10, 50, 100}

	for _, size := range bulkSizes {
		t.Run(fmt.Sprintf("BulkSize_%d", size), func(t *testing.T) {
			// Create bulk request
			articles := make([]CreateArticleRequest, size)
			expectedArticles := make([]models.Article, size)

			for i := 0; i < size; i++ {
				articles[i] = CreateArticleRequest{
					Title:      fmt.Sprintf("Bulk Article %d", i+1),
					Content:    fmt.Sprintf("Content for bulk article %d", i+1),
					CategoryID: 1,
					Status:     "draft",
				}
				expectedArticles[i] = models.Article{
					ID:         uint64(i + 1),
					Title:      articles[i].Title,
					Content:    articles[i].Content,
					AuthorID:   1,
					CategoryID: articles[i].CategoryID,
					Status:     articles[i].Status,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
			}

			mockArticleService.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticles, nil).Once()

			requestBody := BulkCreateArticleRequest{Articles: articles}
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

			// Verify response data
			responseData, ok := response.Data.([]interface{})
			assert.True(t, ok)
			assert.Len(t, responseData, size)
		})
	}

	mockArticleService.AssertExpectations(t)
}

// TestPaginationWorkflow tests pagination across different endpoints
func TestPaginationWorkflow(t *testing.T) {
	handler, mockUserService, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	
	router.GET("/articles", handler.ListArticles)
	router.GET("/users", handler.ListUsers)

	// Test article pagination
	articles := make([]models.Article, 5)
	for i := 0; i < 5; i++ {
		articles[i] = models.Article{
			ID:         uint64(i + 1),
			Title:      fmt.Sprintf("Article %d", i+1),
			AuthorID:   1,
			CategoryID: 1,
			Status:     "published",
		}
	}

	mockArticleService.On("List", mock.Anything, 3, 0, mock.AnythingOfType("services.ArticleFilters"), "published_at", "desc").Return(articles[:3], 5, nil).Once()
	mockArticleService.On("List", mock.Anything, 3, 3, mock.AnythingOfType("services.ArticleFilters"), "published_at", "desc").Return(articles[3:], 5, nil).Once()

	// Page 1
	req, _ := http.NewRequest("GET", "/articles?page=1&limit=3", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var page1Response ArticleListResponse
	err := json.Unmarshal(w.Body.Bytes(), &page1Response)
	assert.NoError(t, err)
	assert.Len(t, page1Response.Articles, 3)
	assert.Equal(t, 1, page1Response.Pagination.Page)
	assert.Equal(t, 3, page1Response.Pagination.Limit)
	assert.Equal(t, 5, page1Response.Pagination.Total)
	assert.Equal(t, 2, page1Response.Pagination.TotalPages)

	// Page 2
	req, _ = http.NewRequest("GET", "/articles?page=2&limit=3", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var page2Response ArticleListResponse
	err = json.Unmarshal(w.Body.Bytes(), &page2Response)
	assert.NoError(t, err)
	assert.Len(t, page2Response.Articles, 2)
	assert.Equal(t, 2, page2Response.Pagination.Page)

	// Test user pagination
	users := make([]*models.User, 3)
	for i := 0; i < 3; i++ {
		users[i] = &models.User{
			ID:       uint64(i + 1),
			Username: fmt.Sprintf("user%d", i+1),
			Email:    fmt.Sprintf("user%d@test.com", i+1),
			Role:     models.RoleReporter,
			IsActive: true,
		}
	}

	mockUserService.On("List", 10, 0, mock.AnythingOfType("*models.User")).Return(users, 3, nil).Once()

	req, _ = http.NewRequest("GET", "/users?limit=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var userResponse UserListResponse
	err = json.Unmarshal(w.Body.Bytes(), &userResponse)
	assert.NoError(t, err)
	assert.Len(t, userResponse.Users, 3)
	assert.Equal(t, 1, userResponse.Pagination.Page)
	assert.Equal(t, 10, userResponse.Pagination.Limit)

	mockArticleService.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

// TestFilteringAndSortingWorkflow tests filtering and sorting functionality
func TestFilteringAndSortingWorkflow(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles", handler.ListArticles)

	articles := []models.Article{
		{ID: 1, Title: "Article 1", Status: "published", CategoryID: 1},
		{ID: 2, Title: "Article 2", Status: "published", CategoryID: 2},
	}

	// Test status filtering
	mockArticleService.On("List", mock.Anything, 20, 0, mock.MatchedBy(func(filters services.ArticleFilters) bool {
		return filters.Status == "published"
	}), "published_at", "desc").Return(articles, 2, nil).Once()

	req, _ := http.NewRequest("GET", "/articles?status=published", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test category filtering
	mockArticleService.On("List", mock.Anything, 20, 0, mock.MatchedBy(func(filters services.ArticleFilters) bool {
		return filters.CategoryID != nil && *filters.CategoryID == 1
	}), "published_at", "desc").Return(articles[:1], 1, nil).Once()

	req, _ = http.NewRequest("GET", "/articles?category_id=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test sorting
	mockArticleService.On("List", mock.Anything, 20, 0, mock.AnythingOfType("services.ArticleFilters"), "title", "asc").Return(articles, 2, nil).Once()

	req, _ = http.NewRequest("GET", "/articles?sort_by=title&sort_order=asc", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockArticleService.AssertExpectations(t)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}

func boolPtr(b bool) *bool {
	return &b
}