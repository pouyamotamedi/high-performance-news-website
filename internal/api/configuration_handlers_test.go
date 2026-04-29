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
	"high-performance-news-website/internal/services"
)

func setupConfigurationHandlersTest() (*ConfigurationHandlers, *services.ConfigService) {
	gin.SetMode(gin.TestMode)
	
	// Create mock cache service
	cache := &MockCacheService{
		data: make(map[string][]byte),
	}
	
	// Create config service
	configService := &services.ConfigService{}
	configService = services.NewConfigService(nil, cache) // Using nil DB for testing
	
	// Create handlers
	handlers := NewConfigurationHandlers(configService)
	
	return handlers, configService
}

func TestGetAllConfigurations(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	router.GET("/config", handlers.GetAllConfigurations)
	
	req, _ := http.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Configurations retrieved successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
}

func TestGetConfiguration(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	router.GET("/config/:key", handlers.GetConfiguration)
	
	tests := []struct {
		name           string
		key            string
		expectedStatus int
	}{
		{
			name:           "Get existing configuration",
			key:            "site_name",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent configuration",
			key:            "non_existent_key",
			expectedStatus: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/config/"+tt.key, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestUpdateConfiguration(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	// Add middleware to set user_id for testing
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	router.PUT("/config/:key", handlers.UpdateConfiguration)
	
	tests := []struct {
		name           string
		key            string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Update existing configuration",
			key:  "site_name",
			requestBody: map[string]interface{}{
				"value":  "Updated Site Name",
				"reason": "Testing update",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Update non-existent configuration",
			key:  "non_existent_key",
			requestBody: map[string]interface{}{
				"value": "Some value",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid request body",
			key:            "site_name",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/config/"+tt.key, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestValidateConfigurations(t *testing.T) {
	handlers, configService := setupConfigurationHandlersTest()
	
	// Add an invalid configuration for testing
	invalidConfig := &models.Configuration{
		Key:         "invalid_config",
		Value:       "", // Empty but required
		Type:        models.ConfigTypeString,
		Category:    "test",
		Description: "Invalid configuration",
		Validation: &models.ConfigValidationRules{
			Required: true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Access the internal config map for testing
	allConfigs := configService.GetAll()
	allConfigs["invalid_config"] = invalidConfig
	
	router := gin.New()
	router.POST("/config/validate", handlers.ValidateConfigurations)
	
	req, _ := http.NewRequest("POST", "/config/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return validation errors
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetConfigurationsByCategory(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	router.GET("/config/categories/:category", handlers.GetConfigurationsByCategory)
	
	req, _ := http.NewRequest("GET", "/config/categories/site", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	// Check that we got configurations
	if response.Data == nil {
		t.Error("Expected configuration data but got nil")
	}
}

func TestCreateSnapshot(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	// Add middleware to set user_id for testing
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	router.POST("/config/snapshots", handlers.CreateSnapshot)
	
	requestBody := map[string]interface{}{
		"name":        "test_snapshot",
		"description": "Test snapshot for unit testing",
	}
	
	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/config/snapshots", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Snapshot created successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
}

func TestCreateFeatureFlag(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	router.POST("/feature-flags", handlers.CreateFeatureFlag)
	
	featureFlag := models.FeatureFlag{
		Key:         "test_feature",
		Name:        "Test Feature",
		Description: "A test feature flag",
		Enabled:     true,
		Rollout: &models.FeatureFlagRollout{
			Percentage: 50,
		},
	}
	
	body, _ := json.Marshal(featureFlag)
	req, _ := http.NewRequest("POST", "/feature-flags", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Feature flag created successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
}

func TestGetAllFeatureFlags(t *testing.T) {
	handlers, configService := setupConfigurationHandlersTest()
	
	// Add a test feature flag
	featureFlag := &models.FeatureFlag{
		Key:         "test_feature",
		Name:        "Test Feature",
		Description: "A test feature flag",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	configService.SetFeatureFlag(featureFlag)
	
	router := gin.New()
	router.GET("/feature-flags", handlers.GetAllFeatureFlags)
	
	req, _ := http.NewRequest("GET", "/feature-flags", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Feature flags retrieved successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
}

func TestCheckFeatureFlag(t *testing.T) {
	handlers, configService := setupConfigurationHandlersTest()
	
	// Add a test feature flag with rollout
	featureFlag := &models.FeatureFlag{
		Key:         "test_feature",
		Name:        "Test Feature",
		Description: "A test feature flag",
		Enabled:     true,
		Rollout: &models.FeatureFlagRollout{
			Percentage: 50,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	configService.SetFeatureFlag(featureFlag)
	
	router := gin.New()
	// Add middleware to set user_id for testing
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	router.POST("/feature-flags/:key/check", handlers.CheckFeatureFlag)
	
	requestBody := map[string]interface{}{
		"context": map[string]interface{}{
			"user_role": "admin",
		},
	}
	
	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/feature-flags/test_feature/check", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Feature flag checked successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
	
	// Check that the response contains the enabled status
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Error("Expected response data to be a map")
	}
	
	if _, exists := data["enabled"]; !exists {
		t.Error("Expected response to contain 'enabled' field")
	}
}

func TestReloadConfiguration(t *testing.T) {
	handlers, _ := setupConfigurationHandlersTest()
	
	router := gin.New()
	router.POST("/config/reload", handlers.ReloadConfiguration)
	
	req, _ := http.NewRequest("POST", "/config/reload", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	
	if response.Message != "Configuration reloaded successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
}