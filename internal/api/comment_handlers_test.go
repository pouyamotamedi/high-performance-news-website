package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommentRepository is a mock implementation of the comment repository
type MockCommentRepository struct {
	mock.Mock
}

func (m *MockCommentRepository) Create(comment *models.Comment) (*models.Comment, error) {
	args := m.Called(comment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) GetByID(id uint64) (*models.Comment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) GetByArticleID(articleID uint64, status models.CommentStatus) ([]models.Comment, error) {
	args := m.Called(articleID, status)
	return args.Get(0).([]models.Comment), args.Error(1)
}

func (m *MockCommentRepository) GetPendingComments(limit, offset int) ([]models.Comment, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]models.Comment), args.Error(1)
}

func (m *MockCommentRepository) UpdateStatus(commentID uint64, status models.CommentStatus, moderatorID uint64, reason string) error {
	args := m.Called(commentID, status, moderatorID, reason)
	return args.Error(0)
}

func (m *MockCommentRepository) BulkUpdateStatus(commentIDs []uint64, status models.CommentStatus, moderatorID uint64, reason string) error {
	args := m.Called(commentIDs, status, moderatorID, reason)
	return args.Error(0)
}

func (m *MockCommentRepository) Delete(commentID uint64) error {
	args := m.Called(commentID)
	return args.Error(0)
}

func (m *MockCommentRepository) GetCommentCount(articleID uint64) (int, error) {
	args := m.Called(articleID)
	return args.Int(0), args.Error(1)
}

func (m *MockCommentRepository) GetModerationStats() (map[string]int, error) {
	args := m.Called()
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockCommentRepository) GetRecentComments(limit int) ([]models.Comment, error) {
	args := m.Called(limit)
	return args.Get(0).([]models.Comment), args.Error(1)
}

func (m *MockCommentRepository) SearchComments(query string, status models.CommentStatus, limit, offset int) ([]models.Comment, error) {
	args := m.Called(query, status, limit, offset)
	return args.Get(0).([]models.Comment), args.Error(1)
}

// MockUserRepository is a mock implementation of the user repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(id uint64) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func setupCommentHandlersTest() (*CommentHandlers, *MockCommentRepository, *MockUserRepository) {
	gin.SetMode(gin.TestMode)
	
	mockCommentRepo := &MockCommentRepository{}
	mockUserRepo := &MockUserRepository{}
	rateLimiter := NewRateLimiter()
	
	handlers := NewCommentHandlers(mockCommentRepo, mockUserRepo, rateLimiter)
	
	return handlers, mockCommentRepo, mockUserRepo
}

func TestCommentHandlers_CreateComment(t *testing.T) {
	handlers, mockCommentRepo, _ := setupCommentHandlersTest()

	t.Run("successful comment creation", func(t *testing.T) {
		// Setup
		req := CreateCommentRequest{
			ArticleID:   1,
			Content:     "This is a test comment",
			AuthorName:  "John Doe",
			AuthorEmail: "john@example.com",
		}

		createdComment := &models.Comment{
			ID:          1,
			ArticleID:   1,
			Content:     "This is a test comment",
			AuthorName:  "John Doe",
			AuthorEmail: "john@example.com",
			Status:      models.CommentStatusApproved,
			CreatedAt:   time.Now(),
		}

		mockCommentRepo.On("Create", mock.AnythingOfType("*models.Comment")).Return(createdComment, nil)

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/comments", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		// Execute
		handlers.CreateComment(c)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "message")
		assert.Contains(t, response, "comment")

		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		// Setup - invalid request (missing required fields)
		req := CreateCommentRequest{
			Content: "", // Empty content should fail validation
		}

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/comments", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		// Execute
		handlers.CreateComment(c)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})

	t.Run("spam detection", func(t *testing.T) {
		// Setup - comment that should be flagged as spam
		req := CreateCommentRequest{
			ArticleID:   1,
			Content:     "CONGRATULATIONS! You won the lottery! Visit https://spam.com and https://more-spam.com and https://even-more-spam.com NOW!!!!",
			AuthorName:  "Spammer",
			AuthorEmail: "spam@example.com",
		}

		createdComment := &models.Comment{
			ID:          1,
			ArticleID:   1,
			Content:     req.Content,
			AuthorName:  req.AuthorName,
			AuthorEmail: req.AuthorEmail,
			Status:      models.CommentStatusSpam,
			SpamScore:   0.9,
			CreatedAt:   time.Now(),
		}

		mockCommentRepo.On("Create", mock.AnythingOfType("*models.Comment")).Return(createdComment, nil)

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/comments", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		// Execute
		handlers.CreateComment(c)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"].(string), "spam")

		mockCommentRepo.AssertExpectations(t)
	})
}

func TestCommentHandlers_GetCommentsByArticle(t *testing.T) {
	handlers, mockCommentRepo, _ := setupCommentHandlersTest()

	t.Run("successful retrieval", func(t *testing.T) {
		// Setup
		articleID := uint64(1)
		comments := []models.Comment{
			{
				ID:          1,
				ArticleID:   articleID,
				Content:     "First comment",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      models.CommentStatusApproved,
			},
			{
				ID:          2,
				ArticleID:   articleID,
				Content:     "Second comment",
				AuthorName:  "Jane Doe",
				AuthorEmail: "jane@example.com",
				Status:      models.CommentStatusApproved,
			},
		}

		mockCommentRepo.On("GetByArticleID", articleID, models.CommentStatusApproved).Return(comments, nil)
		mockCommentRepo.On("GetCommentCount", articleID).Return(2, nil)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/articles/1/comments", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		// Execute
		handlers.GetCommentsByArticle(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "comments")
		assert.Contains(t, response, "total_count")
		assert.Equal(t, float64(2), response["total_count"])

		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("invalid article ID", func(t *testing.T) {
		// Create request with invalid ID
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/articles/invalid/comments", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		// Execute
		handlers.GetCommentsByArticle(c)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})
}

func TestCommentHandlers_ModerateComment(t *testing.T) {
	handlers, mockCommentRepo, _ := setupCommentHandlersTest()

	t.Run("successful moderation", func(t *testing.T) {
		// Setup
		commentID := uint64(1)
		moderatorID := uint64(2)
		req := ModerationRequest{
			Action: "approve",
			Reason: "Looks good",
		}

		mockCommentRepo.On("UpdateStatus", commentID, models.CommentStatusApproved, moderatorID, req.Reason).Return(nil)

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/admin/comments/1/moderate", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		
		// Set user context (simulate authenticated moderator)
		c.Set("user_role", models.RoleEditor)
		c.Set("user_id", moderatorID)

		// Execute
		handlers.ModerateComment(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "message")
		assert.Equal(t, "approve", response["action"])

		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("access denied for non-moderator", func(t *testing.T) {
		// Setup
		req := ModerationRequest{
			Action: "approve",
			Reason: "Looks good",
		}

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/admin/comments/1/moderate", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		
		// Set user context (simulate non-moderator user)
		c.Set("user_role", models.RoleReporter)
		c.Set("user_id", uint64(2))

		// Execute
		handlers.ModerateComment(c)

		// Assert
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})

	t.Run("invalid action", func(t *testing.T) {
		// Setup
		req := ModerationRequest{
			Action: "invalid_action",
			Reason: "Test",
		}

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/admin/comments/1/moderate", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		
		// Set user context (simulate authenticated moderator)
		c.Set("user_role", models.RoleEditor)
		c.Set("user_id", uint64(2))

		// Execute
		handlers.ModerateComment(c)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})
}

func TestCommentHandlers_BulkModerateComments(t *testing.T) {
	handlers, mockCommentRepo, _ := setupCommentHandlersTest()

	t.Run("successful bulk moderation", func(t *testing.T) {
		// Setup
		moderatorID := uint64(2)
		req := BulkModerationRequest{
			CommentIDs: []uint64{1, 2, 3},
			Action:     "approve",
			Reason:     "Bulk approval",
		}

		mockCommentRepo.On("BulkUpdateStatus", req.CommentIDs, models.CommentStatusApproved, moderatorID, req.Reason).Return(nil)

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/admin/comments/bulk-moderate", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		
		// Set user context (simulate authenticated moderator)
		c.Set("user_role", models.RoleEditor)
		c.Set("user_id", moderatorID)

		// Execute
		handlers.BulkModerateComments(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "message")
		assert.Equal(t, "approve", response["action"])
		assert.Equal(t, float64(3), response["count"])

		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("too many comments", func(t *testing.T) {
		// Setup - more than 100 comments
		commentIDs := make([]uint64, 101)
		for i := range commentIDs {
			commentIDs[i] = uint64(i + 1)
		}

		req := BulkModerationRequest{
			CommentIDs: commentIDs,
			Action:     "approve",
			Reason:     "Bulk approval",
		}

		// Create request
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/admin/comments/bulk-moderate", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		
		// Set user context (simulate authenticated moderator)
		c.Set("user_role", models.RoleEditor)
		c.Set("user_id", uint64(2))

		// Execute
		handlers.BulkModerateComments(c)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["message"].(string), "100 comments")
	})
}

func TestCommentHandlers_GetPendingComments(t *testing.T) {
	handlers, mockCommentRepo, _ := setupCommentHandlersTest()

	t.Run("successful retrieval", func(t *testing.T) {
		// Setup
		pendingComments := []models.Comment{
			{
				ID:          1,
				ArticleID:   1,
				Content:     "Pending comment 1",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      models.CommentStatusPending,
			},
			{
				ID:          2,
				ArticleID:   2,
				Content:     "Pending comment 2",
				AuthorName:  "Jane Doe",
				AuthorEmail: "jane@example.com",
				Status:      models.CommentStatusPending,
			},
		}

		mockCommentRepo.On("GetPendingComments", 20, 0).Return(pendingComments, nil)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/admin/comments/pending", nil)
		
		// Set user context (simulate authenticated moderator)
		c.Set("user_role", models.RoleEditor)
		c.Set("user_id", uint64(2))

		// Execute
		handlers.GetPendingComments(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "comments")
		assert.Equal(t, float64(20), response["limit"])
		assert.Equal(t, float64(0), response["offset"])

		mockCommentRepo.AssertExpectations(t)
	})

	t.Run("access denied for non-moderator", func(t *testing.T) {
		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/admin/comments/pending", nil)
		
		// Set user context (simulate non-moderator user)
		c.Set("user_role", models.RoleReporter)
		c.Set("user_id", uint64(2))

		// Execute
		handlers.GetPendingComments(c)

		// Assert
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})
}