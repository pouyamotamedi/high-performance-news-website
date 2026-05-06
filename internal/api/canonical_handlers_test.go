package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/services"
	_ "github.com/lib/pq"
)

func setupCanonicalHandlerTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: could not ping test database: %v", err)
	}

	return db
}

func setupCanonicalHandlerTestData(t *testing.T, db *sql.DB) (uint64, uint64, uint64) {
	// Create test user
	var userID uint64
	err := db.QueryRow(`
		INSERT INTO users (username, email, password_hash, role)
		VALUES ('testuser', 'test@example.com', 'hash', 'admin')
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test category
	var categoryID uint64
	err = db.QueryRow(`
		INSERT INTO categories (name, slug, description)
		VALUES ('Test Category', 'test-category', 'Test category description')
		RETURNING id
	`).Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Create test tag
	var tagID uint64
	err = db.QueryRow(`
		INSERT INTO tags (name, slug, description, keywords)
		VALUES ('Test Tag', 'test-tag', 'Test tag description', '["test", "keyword"]')
		RETURNING id
	`).Scan(&tagID)
	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	return userID, categoryID, tagID
}

func cleanupCanonicalHandlerTestData(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DELETE FROM canonical_jobs")
	if err != nil {
		t.Logf("Failed to clean up canonical_jobs: %v", err)
	}

	_, err = db.Exec("DELETE FROM tags")
	if err != nil {
		t.Logf("Failed to clean up tags: %v", err)
	}

	_, err = db.Exec("DELETE FROM categories")
	if err != nil {
		t.Logf("Failed to clean up categories: %v", err)
	}

	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		t.Logf("Failed to clean up users: %v", err)
	}
}

func setupCanonicalRouter(handlers *CanonicalHandlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("user_role", "admin")
		c.Next()
	})

	api := router.Group("/api/v1")
	RegisterCanonicalRoutes(api, handlers)

	return router
}

func TestCanonicalHandlers_ScheduleCanonicalJob(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, categoryID, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	tests := []struct {
		name           string
		request        ScheduleCanonicalJobRequest
		expectedStatus int
		expectJobID    bool
	}{
		{
			name: "Schedule tag canonical job",
			request: ScheduleCanonicalJobRequest{
				ArticleID:     1,
				Target:        services.NewTagTarget(tagID),
				AdminOverride: false,
			},
			expectedStatus: http.StatusCreated,
			expectJobID:    true,
		},
		{
			name: "Schedule category canonical job with admin override",
			request: ScheduleCanonicalJobRequest{
				ArticleID:     1,
				Target:        services.NewCategoryTarget(categoryID),
				AdminOverride: true,
			},
			expectedStatus: http.StatusCreated,
			expectJobID:    true,
		},
		{
			name: "Schedule URL canonical job",
			request: ScheduleCanonicalJobRequest{
				ArticleID:     1,
				Target:        services.NewURLTarget("/custom/canonical/url"),
				AdminOverride: false,
			},
			expectedStatus: http.StatusCreated,
			expectJobID:    true,
		},
		{
			name: "Invalid request - missing article ID",
			request: ScheduleCanonicalJobRequest{
				Target:        services.NewTagTarget(tagID),
				AdminOverride: false,
			},
			expectedStatus: http.StatusBadRequest,
			expectJobID:    false,
		},
		{
			name: "Invalid request - invalid target",
			request: ScheduleCanonicalJobRequest{
				ArticleID:     1,
				Target:        services.CanonicalTarget{Type: "invalid"},
				AdminOverride: false,
			},
			expectedStatus: http.StatusBadRequest,
			expectJobID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/api/v1/canonical/jobs", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectJobID && w.Code == http.StatusCreated {
				var response ScheduleCanonicalJobResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if response.JobID == 0 {
					t.Errorf("Expected non-zero job ID")
				}

				if response.Message == "" {
					t.Errorf("Expected non-empty message")
				}
			}
		})
	}

	// Clean up created jobs
	_, _ = db.Exec("DELETE FROM canonical_jobs WHERE created_by = $1", userID)
}

func TestCanonicalHandlers_GetPendingJobs(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	// Create a test job with admin override (ready for processing)
	_, err := cm.ScheduleCanonicalJob(1, services.NewTagTarget(tagID), &userID, true)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/canonical/jobs/pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	jobs, ok := response["jobs"].([]interface{})
	if !ok {
		t.Errorf("Expected jobs array in response")
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Errorf("Expected count in response")
	}

	if len(jobs) != int(count) {
		t.Errorf("Jobs array length (%d) doesn't match count (%d)", len(jobs), int(count))
	}

	if len(jobs) == 0 {
		t.Errorf("Expected at least one pending job")
	}
}

func TestCanonicalHandlers_GetJobsByArticle(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	// Create test jobs for article 1
	_, err := cm.ScheduleCanonicalJob(1, services.NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/canonical/jobs/article/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	jobs, ok := response["jobs"].([]interface{})
	if !ok {
		t.Errorf("Expected jobs array in response")
	}

	articleID, ok := response["article_id"].(float64)
	if !ok || articleID != 1 {
		t.Errorf("Expected article_id 1 in response, got %v", articleID)
	}

	if len(jobs) == 0 {
		t.Errorf("Expected at least one job for article 1")
	}
}

func TestCanonicalHandlers_ProcessPendingJobs(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	// Create test jobs with admin override (ready for processing)
	_, err := cm.ScheduleCanonicalJob(1, services.NewTagTarget(tagID), &userID, true)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	req, _ := http.NewRequest("POST", "/api/v1/canonical/jobs/process", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	processed, ok := response["processed"].(float64)
	if !ok {
		t.Errorf("Expected processed count in response")
	}

	if processed < 1 {
		t.Errorf("Expected at least 1 job to be processed, got %v", processed)
	}

	message, ok := response["message"].(string)
	if !ok || message == "" {
		t.Errorf("Expected non-empty message in response")
	}
}

func TestCanonicalHandlers_CancelJob(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	// Create a test job
	jobID, err := cm.ScheduleCanonicalJob(1, services.NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	req, _ := http.NewRequest("DELETE", "/api/v1/canonical/jobs/"+string(rune(jobID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	message, ok := response["message"].(string)
	if !ok || message == "" {
		t.Errorf("Expected non-empty message in response")
	}

	returnedJobID, ok := response["job_id"].(float64)
	if !ok || uint64(returnedJobID) != jobID {
		t.Errorf("Expected job_id %d in response, got %v", jobID, returnedJobID)
	}
}

func TestCanonicalHandlers_GenerateCanonicalURL(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	_, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	tests := []struct {
		name           string
		request        GenerateCanonicalURLRequest
		expectedStatus int
		expectedURL    string
	}{
		{
			name: "Generate tag URL",
			request: GenerateCanonicalURLRequest{
				Target: services.NewTagTarget(tagID),
			},
			expectedStatus: http.StatusOK,
			expectedURL:    "/en/tag/test-tag",
		},
		{
			name: "Generate custom URL",
			request: GenerateCanonicalURLRequest{
				Target: services.NewURLTarget("/custom/path"),
			},
			expectedStatus: http.StatusOK,
			expectedURL:    "/custom/path",
		},
		{
			name: "Invalid target",
			request: GenerateCanonicalURLRequest{
				Target: services.CanonicalTarget{Type: "invalid"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/api/v1/canonical/generate-url", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response GenerateCanonicalURLResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if response.CanonicalURL != tt.expectedURL {
					t.Errorf("Expected canonical URL %s, got %s", tt.expectedURL, response.CanonicalURL)
				}
			}
		})
	}
}

func TestCanonicalHandlers_GetJobStats(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	userID, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)
	router := setupCanonicalRouter(handlers)

	// Create test jobs
	_, err := cm.ScheduleCanonicalJob(1, services.NewTagTarget(tagID), &userID, false)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/canonical/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	stats, ok := response["stats"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected stats object in response")
	}

	// Should have at least pending jobs
	if len(stats) == 0 {
		t.Errorf("Expected at least one stat entry")
	}
}

func TestCanonicalHandlers_AuthorizationChecks(t *testing.T) {
	db := setupCanonicalHandlerTestDB(t)
	defer db.Close()

	_, _, tagID := setupCanonicalHandlerTestData(t, db)
	defer cleanupCanonicalHandlerTestData(t, db)

	cm := services.NewCanonicalManager(db)
	handlers := NewCanonicalHandlers(cm)

	// Setup router with non-admin user
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Set("user_role", "reporter") // Non-admin role
		c.Next()
	})

	api := router.Group("/api/v1")
	RegisterCanonicalRoutes(api, handlers)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:   "Admin override forbidden for non-admin",
			method: "POST",
			path:   "/api/v1/canonical/jobs",
			body: ScheduleCanonicalJobRequest{
				ArticleID:     1,
				Target:        services.NewTagTarget(tagID),
				AdminOverride: true,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Process jobs forbidden for non-admin",
			method:         "POST",
			path:           "/api/v1/canonical/jobs/process",
			body:           nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Cancel job forbidden for non-admin",
			method:         "DELETE",
			path:           "/api/v1/canonical/jobs/1",
			body:           nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Cleanup forbidden for non-admin",
			method:         "POST",
			path:           "/api/v1/canonical/cleanup",
			body:           nil,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				requestBody, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(requestBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}