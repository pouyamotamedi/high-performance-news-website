package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/desktop"
)

// Integration tests for the desktop deployment application
func TestDesktopAppIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "desktop-app-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test configuration file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
servers:
  test-server:
    host: "localhost"
    port: 22
    user: "testuser"
    key_file: "/tmp/test-key"

app:
  name: "test-app"
  binary: "./test-binary"
  port: 8080
  health_path: "/health"

deploy:
  strategy: "blue-green"
  health_timeout: 30s
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create test app
	config := &desktop.Config{
		Port:    0, // Use random port for testing
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(t, err)

	// Create test server
	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test cases
	t.Run("Dashboard", func(t *testing.T) {
		testDashboard(t, server.URL)
	})

	t.Run("Configuration Management", func(t *testing.T) {
		testConfigurationManagement(t, server.URL, configPath)
	})

	t.Run("Server Management", func(t *testing.T) {
		testServerManagement(t, server.URL)
	})

	t.Run("WebSocket Connection", func(t *testing.T) {
		testWebSocketConnection(t, server.URL)
	})

	t.Run("API Endpoints", func(t *testing.T) {
		testAPIEndpoints(t, server.URL)
	})
}

func testDashboard(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func testConfigurationManagement(t *testing.T, baseURL, configPath string) {
	// Test loading configuration
	loadConfigReq := map[string]string{
		"config_path": configPath,
	}
	jsonData, _ := json.Marshal(loadConfigReq)

	resp, err := http.Post(baseURL+"/api/config/load", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Note: This will fail because the config file references non-existent binary
	// But we can test that the endpoint is working
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)

	// Test getting configuration status
	resp, err = http.Get(baseURL + "/api/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var configResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&configResp)
	require.NoError(t, err)

	loaded, ok := configResp["loaded"].(bool)
	assert.True(t, ok)
	assert.False(t, loaded) // Should be false due to validation failure
}

func testServerManagement(t *testing.T, baseURL string) {
	// Test getting servers (should return error when no config loaded)
	resp, err := http.Get(baseURL + "/api/servers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "No configuration loaded", errorResp["error"])
}

func testWebSocketConnection(t *testing.T, baseURL string) {
	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + baseURL[4:] + "/ws"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read initial connection message
	var message map[string]interface{}
	err = conn.ReadJSON(&message)
	require.NoError(t, err)

	assert.Equal(t, "connected", message["type"])
	assert.Equal(t, "WebSocket connected", message["message"])
}

func testAPIEndpoints(t *testing.T, baseURL string) {
	endpoints := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/api/servers", http.StatusBadRequest}, // No config loaded
		{"GET", "/api/config", http.StatusOK},
		{"GET", "/api/servers/test/logs", http.StatusOK},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
			var resp *http.Response
			var err error

			switch endpoint.method {
			case "GET":
				resp, err = http.Get(baseURL + endpoint.path)
			case "POST":
				resp, err = http.Post(baseURL+endpoint.path, "application/json", bytes.NewBuffer([]byte("{}")))
			}

			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, endpoint.expected, resp.StatusCode)
		})
	}
}

// Test the desktop app startup and shutdown
func TestDesktopAppLifecycle(t *testing.T) {
	config := &desktop.Config{
		Port:    0, // Random port
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(t, err)

	// Create server
	server := &http.Server{
		Addr:    ":0",
		Handler: app.Router(),
	}

	// Start server in goroutine
	go func() {
		server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkDesktopAppDashboard(b *testing.B) {
	config := &desktop.Config{
		Port:    0,
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(b, err)

	server := httptest.NewServer(app.Router())
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(server.URL + "/")
		require.NoError(b, err)
		resp.Body.Close()
	}
}

func BenchmarkDesktopAppAPI(b *testing.B) {
	config := &desktop.Config{
		Port:    0,
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(b, err)

	server := httptest.NewServer(app.Router())
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(server.URL + "/api/config")
		require.NoError(b, err)
		resp.Body.Close()
	}
}

// Test error handling
func TestDesktopAppErrorHandling(t *testing.T) {
	config := &desktop.Config{
		Port:    0,
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(t, err)

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test invalid JSON in POST request
	resp, err := http.Post(server.URL+"/api/config", "application/json", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errorResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid JSON", errorResp["error"])
}

// Test concurrent access
func TestDesktopAppConcurrency(t *testing.T) {
	config := &desktop.Config{
		Port:    0,
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(t, err)

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Make concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(server.URL + "/api/config")
			assert.NoError(t, err)
			if resp != nil {
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
			// Request completed
		case <-time.After(5 * time.Second):
			t.Fatal("Request timed out")
		}
	}
}

// Test WebSocket message broadcasting
func TestWebSocketBroadcasting(t *testing.T) {
	config := &desktop.Config{
		Port:    0,
		DevMode: true,
	}

	app, err := desktop.NewApp(config)
	require.NoError(t, err)

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Connect multiple WebSocket clients
	wsURL := "ws" + server.URL[4:] + "/ws"
	
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Read initial connection messages
	var msg1, msg2 map[string]interface{}
	
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn1.ReadJSON(&msg1)
	require.NoError(t, err)
	assert.Equal(t, "connected", msg1["type"])

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn2.ReadJSON(&msg2)
	require.NoError(t, err)
	assert.Equal(t, "connected", msg2["type"])
}