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
	"high-performance-news-website/internal/models"
)

// MockCDNService implements CDNServiceInterface for testing
type MockCDNService struct {
	config       *models.CDNConfig
	failoverMode bool
	shouldError  bool
}

func NewMockCDNService() *MockCDNService {
	return &MockCDNService{
		config: &models.CDNConfig{
			ID:        1,
			Provider:  "cloudflare",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		failoverMode: false,
		shouldError:  false,
	}
}

func (m *MockCDNService) GetConfig() (*models.CDNConfig, error) {
	if m.shouldError {
		return nil, fmt.Errorf("configuration not found")
	}
	return m.config, nil
}

func (m *MockCDNService) UpdateConfig(config *models.CDNConfig) error {
	if m.shouldError {
		return fmt.Errorf("invalid configuration")
	}
	m.config = config
	return nil
}

func (m *MockCDNService) TestConnection() error {
	if m.shouldError {
		return fmt.Errorf("connection failed")
	}
	return nil
}

func (m *MockCDNService) PurgeCache(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	if m.shouldError {
		return nil, fmt.Errorf("purge failed")
	}
	return &models.CDNPurgeResponse{
		Success:   true,
		RequestID: "test-request-id",
		Message:   "Cache purged successfully",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockCDNService) PurgeURL(url string) error {
	if m.shouldError {
		return fmt.Errorf("purge URL failed")
	}
	return nil
}

func (m *MockCDNService) PurgeURLs(urls []string) error {
	if m.shouldError {
		return fmt.Errorf("purge URLs failed")
	}
	return nil
}

func (m *MockCDNService) PurgeAll() error {
	if m.shouldError {
		return fmt.Errorf("purge all failed")
	}
	return nil
}

func (m *MockCDNService) GetStats() (*models.CDNStats, error) {
	if m.shouldError {
		return nil, fmt.Errorf("get stats failed")
	}
	return &models.CDNStats{
		CacheHitRatio:  85.5,
		BandwidthSaved: 1024000,
		RequestsServed: 50000,
		OriginRequests: 7500,
		ResponseTime:   150,
		LastUpdated:    time.Now(),
	}, nil
}

func (m *MockCDNService) GetHealthStatus() (*models.CDNHealthCheck, error) {
	if m.shouldError {
		return nil, fmt.Errorf("get health failed")
	}
	
	status := "healthy"
	if m.failoverMode {
		status = "degraded"
	}
	
	return &models.CDNHealthCheck{
		Provider:     "cloudflare",
		Status:       status,
		ResponseTime: 150,
		LastCheck:    time.Now(),
		ErrorCount:   0,
		Message:      "CDN is operating normally",
	}, nil
}

func (m *MockCDNService) EnableFailover() error {
	if m.shouldError {
		return fmt.Errorf("enable failover failed")
	}
	m.failoverMode = true
	return nil
}

func (m *MockCDNService) DisableFailover() error {
	if m.shouldError {
		return fmt.Errorf("disable failover failed")
	}
	m.failoverMode = false
	return nil
}

func (m *MockCDNService) IsFailoverActive() bool {
	return m.failoverMode
}

func TestGetCDNConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := NewMockCDNService()
	handlers := NewCDNHandlers(mockService)
	
	router := gin.New()
	router.GET("/cdn/config", handlers.GetCDNConfig)
	
	req, _ := http.NewRequest("GET", "/cdn/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "cloudflare", data["provider"])
	assert.Equal(t, "example.com", data["domain"])
	assert.True(t, data["enabled"].(bool))
}

func TestTestCDNConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := NewMockCDNService()
	handlers := NewCDNHandlers(mockService)
	
	router := gin.New()
	router.POST("/cdn/test", handlers.TestCDNConnection)
	
	req, _ := http.NewRequest("POST", "/cdn/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}