package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/config"
)

func TestServerWithCache(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Mode: "test",
		},
		Cache: config.CacheConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1, // Use different DB for tests
		},
		App: config.AppConfig{
			Name:    "Test News Website",
			Version: "1.0.0",
		},
	}

	// Create server (this will test cache initialization)
	srv, err := New(cfg)
	if err != nil {
		t.Skip("Skipping test: cache not available")
	}
	defer srv.cache.Close()

	srv.setupRoutes()

	t.Run("Health check includes cache status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "ok", response["status"])
		assert.Equal(t, cfg.App.Name, response["app"])
		assert.Contains(t, response, "cache")
	})

	t.Run("Cache clear endpoint", func(t *testing.T) {
		// Test cache clear with pattern
		reqBody := map[string]string{
			"pattern": "test:*",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/cache/clear", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["message"], "Cache cleared successfully")
		assert.Equal(t, "test:*", response["pattern"])
	})

	t.Run("Cache stats endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Contains(t, response["message"], "operational")
	})

	t.Run("Invalid cache clear request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/cache/clear", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}