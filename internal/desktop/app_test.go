package desktop

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, config, app.config)
	assert.NotNil(t, app.wsConnections)
}

func TestAppRouter(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	router := app.Router()
	assert.NotNil(t, router)

	// Test that router is a mux.Router
	_, ok := router.(*mux.Router)
	assert.True(t, ok)
}

func TestHandleDashboard(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	app.handleDashboard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestHandleGetServers_NoConfig(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/servers", nil)
	w := httptest.NewRecorder()

	app.handleGetServers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "No configuration loaded", response["error"])
}

func TestHandleGetServerStatus_NoAgent(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/servers/test/status", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test"})
	w := httptest.NewRecorder()

	app.handleGetServerStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "No deployment agent available", response["error"])
}

func TestHandleLoadConfig(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	// Test with invalid JSON
	req := httptest.NewRequest("POST", "/api/config/load", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	app.handleLoadConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid JSON", response["error"])
}

func TestHandleLoadConfig_ValidRequest(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	// Test with valid JSON but non-existent config file
	requestBody := map[string]string{
		"config_path": "/non/existent/config.yaml",
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/config/load", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.handleLoadConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "Failed to load config")
}

func TestSendJSON(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"test": "value",
		"number": 42,
	}

	app.sendJSON(w, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "value", response["test"])
	assert.Equal(t, float64(42), response["number"])
}

func TestSendError(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	app.sendError(w, "Test error message", http.StatusInternalServerError)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Test error message", response["error"])
}

func TestBroadcastMessage(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	// Test broadcasting with no connections (should not panic)
	message := map[string]interface{}{
		"type": "test",
		"message": "test message",
	}

	// This should not panic even with no connections
	app.broadcastMessage(message)

	// Verify no connections exist
	assert.Equal(t, 0, len(app.wsConnections))
}

func TestHandleGetConfig_NoConfig(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()

	app.handleGetConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, false, response["loaded"])
}

func TestHandleUpdateConfig(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	// Test with valid JSON
	configData := map[string]interface{}{
		"app": map[string]interface{}{
			"name": "test-app",
			"port": 8080,
		},
	}
	jsonBody, _ := json.Marshal(configData)

	req := httptest.NewRequest("POST", "/api/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.handleUpdateConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "saved", response["status"])
}

func TestHandleUpdateConfig_InvalidJSON(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/config", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	app.handleUpdateConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid JSON", response["error"])
}

func TestHandleGetHistory_NoAgent(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/servers/test/history", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test"})
	w := httptest.NewRecorder()

	app.handleGetHistory(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "No deployment agent available", response["error"])
}

func TestHandleGetLogs(t *testing.T) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/servers/test/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test"})
	w := httptest.NewRecorder()

	app.handleGetLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	
	logs, ok := response["logs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 0, len(logs))
}

// Benchmark tests
func BenchmarkHandleDashboard(b *testing.B) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(b, err)

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.handleDashboard(w, req)
	}
}

func BenchmarkSendJSON(b *testing.B) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(b, err)

	data := map[string]interface{}{
		"test": "value",
		"number": 42,
		"timestamp": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.sendJSON(w, data)
	}
}

func BenchmarkBroadcastMessage(b *testing.B) {
	config := &Config{
		Port:    8090,
		DevMode: true,
	}

	app, err := NewApp(config)
	require.NoError(b, err)

	message := map[string]interface{}{
		"type": "test",
		"message": "benchmark message",
		"timestamp": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.broadcastMessage(message)
	}
}