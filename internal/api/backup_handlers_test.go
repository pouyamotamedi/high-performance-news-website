package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/models"
)

// MockBackupService is a mock implementation of BackupServiceInterface
type MockBackupService struct {
	mock.Mock
}

func (m *MockBackupService) CreateBackup(request *models.BackupRequest) (*models.Backup, error) {
	args := m.Called(request)
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) CreateFullBackup() (*models.Backup, error) {
	args := m.Called()
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) CreateIncrementalBackup() (*models.Backup, error) {
	args := m.Called()
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) GetBackup(id uint64) (*models.Backup, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) ListBackups(limit, offset int) ([]*models.Backup, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]*models.Backup), args.Error(1)
}

func (m *MockBackupService) DeleteBackup(id uint64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockBackupService) RestoreBackup(request *models.RestoreRequest) (*models.RestoreOperation, error) {
	args := m.Called(request)
	return args.Get(0).(*models.RestoreOperation), args.Error(1)
}

func (m *MockBackupService) GetRestoreOperation(id uint64) (*models.RestoreOperation, error) {
	args := m.Called(id)
	return args.Get(0).(*models.RestoreOperation), args.Error(1)
}

func (m *MockBackupService) ListRestoreOperations(limit, offset int) ([]*models.RestoreOperation, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]*models.RestoreOperation), args.Error(1)
}

func (m *MockBackupService) RestoreToPointInTime(targetTime time.Time, targetDB string) (*models.RestoreOperation, error) {
	args := m.Called(targetTime, targetDB)
	return args.Get(0).(*models.RestoreOperation), args.Error(1)
}

func (m *MockBackupService) GetAvailableRecoveryPoints() ([]time.Time, error) {
	args := m.Called()
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *MockBackupService) ValidateBackup(backupID uint64) (*models.BackupValidation, error) {
	args := m.Called(backupID)
	return args.Get(0).(*models.BackupValidation), args.Error(1)
}

func (m *MockBackupService) RunDisasterRecoveryTest(testName string, backupID uint64) (*models.DisasterRecoveryTest, error) {
	args := m.Called(testName, backupID)
	return args.Get(0).(*models.DisasterRecoveryTest), args.Error(1)
}

func (m *MockBackupService) GetDRTestResults(testID uint64) (*models.DisasterRecoveryTest, error) {
	args := m.Called(testID)
	return args.Get(0).(*models.DisasterRecoveryTest), args.Error(1)
}

func (m *MockBackupService) ListDRTests(limit, offset int) ([]*models.DisasterRecoveryTest, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]*models.DisasterRecoveryTest), args.Error(1)
}

func (m *MockBackupService) ReplicateBackup(backupID uint64, targetName string) (*models.BackupReplication, error) {
	args := m.Called(backupID, targetName)
	return args.Get(0).(*models.BackupReplication), args.Error(1)
}

func (m *MockBackupService) GetReplicationStatus(backupID uint64) ([]*models.BackupReplication, error) {
	args := m.Called(backupID)
	return args.Get(0).([]*models.BackupReplication), args.Error(1)
}

func (m *MockBackupService) GetBackupMetrics() (*models.BackupMetrics, error) {
	args := m.Called()
	return args.Get(0).(*models.BackupMetrics), args.Error(1)
}

func (m *MockBackupService) GetBackupHealth() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockBackupService) CleanupOldBackups() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBackupService) ArchiveOldBackups(olderThan time.Time) error {
	args := m.Called(olderThan)
	return args.Error(0)
}

func (m *MockBackupService) StartBackupScheduler() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBackupService) StopBackupScheduler() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBackupService) GetSchedulerStatus() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Test setup helper
func setupBackupHandlerTest() (*gin.Engine, *MockBackupService, *BackupHandlers) {
	gin.SetMode(gin.TestMode)
	
	mockService := &MockBackupService{}
	handlers := NewBackupHandlers(mockService)
	
	router := gin.New()
	api := router.Group("/api/v1")
	handlers.RegisterRoutes(api)
	
	return router, mockService, handlers
}

func TestBackupHandlers_CreateBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	t.Run("successful backup creation", func(t *testing.T) {
		expectedBackup := &models.Backup{
			ID:     1,
			Type:   models.BackupTypeFull,
			Status: models.BackupStatusPending,
		}
		
		mockService.On("CreateBackup", mock.AnythingOfType("*models.BackupRequest")).Return(expectedBackup, nil).Once()
		
		requestBody := models.BackupRequest{
			Type:        models.BackupTypeFull,
			Description: "Test backup",
			Compress:    true,
			Encrypt:     true,
		}
		
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/backup/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "backup")
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid request format", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/backup/create", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestBackupHandlers_CreateFullBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedBackup := &models.Backup{
		ID:     1,
		Type:   models.BackupTypeFull,
		Status: models.BackupStatusPending,
	}
	
	mockService.On("CreateFullBackup").Return(expectedBackup, nil).Once()
	
	req := httptest.NewRequest("POST", "/api/v1/backup/full", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "backup")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_CreateIncrementalBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedBackup := &models.Backup{
		ID:     1,
		Type:   models.BackupTypeIncremental,
		Status: models.BackupStatusPending,
	}
	
	mockService.On("CreateIncrementalBackup").Return(expectedBackup, nil).Once()
	
	req := httptest.NewRequest("POST", "/api/v1/backup/incremental", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "backup")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_GetBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	t.Run("successful get backup", func(t *testing.T) {
		expectedBackup := &models.Backup{
			ID:     1,
			Type:   models.BackupTypeFull,
			Status: models.BackupStatusCompleted,
		}
		
		mockService.On("GetBackup", uint64(1)).Return(expectedBackup, nil).Once()
		
		req := httptest.NewRequest("GET", "/api/v1/backup/1", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "backup")
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid backup ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/backup/invalid", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestBackupHandlers_ListBackups(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedBackups := []*models.Backup{
		{ID: 1, Type: models.BackupTypeFull, Status: models.BackupStatusCompleted},
		{ID: 2, Type: models.BackupTypeIncremental, Status: models.BackupStatusCompleted},
	}
	
	mockService.On("ListBackups", 50, 0).Return(expectedBackups, nil).Once()
	
	req := httptest.NewRequest("GET", "/api/v1/backup/list", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "backups")
	assert.Equal(t, float64(50), response["limit"])
	assert.Equal(t, float64(0), response["offset"])
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_DeleteBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	mockService.On("DeleteBackup", uint64(1)).Return(nil).Once()
	
	req := httptest.NewRequest("DELETE", "/api/v1/backup/1", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_RestoreBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedRestore := &models.RestoreOperation{
		ID:             1,
		BackupID:       1,
		RestoreType:    "full",
		TargetDatabase: "test_db",
		Status:         models.BackupStatusPending,
	}
	
	mockService.On("RestoreBackup", mock.AnythingOfType("*models.RestoreRequest")).Return(expectedRestore, nil).Once()
	
	requestBody := models.RestoreRequest{
		BackupID:         1,
		RestoreType:      "full",
		TargetDatabase:   "test_db",
		OverwriteExisting: true,
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/backup/restore", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "restore_operation")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_RestoreToPointInTime(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	targetTime := time.Now().Add(-1 * time.Hour)
	expectedRestore := &models.RestoreOperation{
		ID:             1,
		BackupID:       1,
		RestoreType:    "point_in_time",
		TargetDatabase: "test_db",
		Status:         models.BackupStatusPending,
	}
	
	mockService.On("RestoreToPointInTime", mock.AnythingOfType("time.Time"), "test_db").Return(expectedRestore, nil).Once()
	
	requestBody := map[string]string{
		"target_time":     targetTime.Format(time.RFC3339),
		"target_database": "test_db",
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/backup/restore/point-in-time", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "restore_operation")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_ValidateBackup(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedValidation := &models.BackupValidation{
		ID:             1,
		BackupID:       1,
		ValidationType: "manual_validation",
		Status:         models.BackupStatusPending,
	}
	
	mockService.On("ValidateBackup", uint64(1)).Return(expectedValidation, nil).Once()
	
	req := httptest.NewRequest("POST", "/api/v1/backup/1/validate", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "validation")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_RunDisasterRecoveryTest(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedTest := &models.DisasterRecoveryTest{
		ID:       1,
		TestName: "test_dr",
		BackupID: 1,
		Status:   models.BackupStatusPending,
	}
	
	mockService.On("RunDisasterRecoveryTest", "test_dr", uint64(1)).Return(expectedTest, nil).Once()
	
	requestBody := map[string]interface{}{
		"test_name": "test_dr",
		"backup_id": 1,
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/backup/dr-test", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "dr_test")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_GetBackupMetrics(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedMetrics := &models.BackupMetrics{
		TotalBackups:      10,
		SuccessfulBackups: 8,
		FailedBackups:     2,
		TotalBackupSize:   1024000,
	}
	
	mockService.On("GetBackupMetrics").Return(expectedMetrics, nil).Once()
	
	req := httptest.NewRequest("GET", "/api/v1/backup/metrics", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "metrics")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_GetBackupHealth(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedHealth := map[string]interface{}{
		"healthy":                true,
		"backup_directory":       "accessible",
		"recent_backup_success":  true,
		"scheduler_running":      true,
	}
	
	mockService.On("GetBackupHealth").Return(expectedHealth, nil).Once()
	
	req := httptest.NewRequest("GET", "/api/v1/backup/health", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "health")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_StartBackupScheduler(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	mockService.On("StartBackupScheduler").Return(nil).Once()
	
	req := httptest.NewRequest("POST", "/api/v1/backup/scheduler/start", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_StopBackupScheduler(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	mockService.On("StopBackupScheduler").Return(nil).Once()
	
	req := httptest.NewRequest("POST", "/api/v1/backup/scheduler/stop", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	
	mockService.AssertExpectations(t)
}

func TestBackupHandlers_GetSchedulerStatus(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedStatus := map[string]interface{}{
		"running":                      true,
		"full_backup_interval":         "24h0m0s",
		"incremental_backup_interval":  "1h0m0s",
	}
	
	mockService.On("GetSchedulerStatus").Return(expectedStatus, nil).Once()
	
	req := httptest.NewRequest("GET", "/api/v1/backup/scheduler/status", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "scheduler_status")
	
	mockService.AssertExpectations(t)
}

// Integration tests
func TestBackupHandlers_Integration_BackupWorkflow(t *testing.T) {
	router, mockService, _ := setupBackupHandlerTest()
	
	// Test complete backup workflow: create -> validate -> restore
	t.Run("complete backup workflow", func(t *testing.T) {
		// Step 1: Create backup
		expectedBackup := &models.Backup{
			ID:     1,
			Type:   models.BackupTypeFull,
			Status: models.BackupStatusPending,
		}
		mockService.On("CreateFullBackup").Return(expectedBackup, nil).Once()
		
		req := httptest.NewRequest("POST", "/api/v1/backup/full", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		
		// Step 2: Validate backup
		expectedValidation := &models.BackupValidation{
			ID:       1,
			BackupID: 1,
			Status:   models.BackupStatusPending,
		}
		mockService.On("ValidateBackup", uint64(1)).Return(expectedValidation, nil).Once()
		
		req = httptest.NewRequest("POST", "/api/v1/backup/1/validate", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		
		// Step 3: Restore backup
		expectedRestore := &models.RestoreOperation{
			ID:       1,
			BackupID: 1,
			Status:   models.BackupStatusPending,
		}
		mockService.On("RestoreBackup", mock.AnythingOfType("*models.RestoreRequest")).Return(expectedRestore, nil).Once()
		
		restoreRequest := models.RestoreRequest{
			BackupID:       1,
			RestoreType:    "full",
			TargetDatabase: "test_restore_db",
		}
		jsonBody, _ := json.Marshal(restoreRequest)
		req = httptest.NewRequest("POST", "/api/v1/backup/restore", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		
		mockService.AssertExpectations(t)
	})
}

// Benchmark tests
func BenchmarkBackupHandlers_CreateBackup(b *testing.B) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedBackup := &models.Backup{
		ID:     1,
		Type:   models.BackupTypeFull,
		Status: models.BackupStatusPending,
	}
	mockService.On("CreateBackup", mock.AnythingOfType("*models.BackupRequest")).Return(expectedBackup, nil)
	
	requestBody := models.BackupRequest{
		Type:        models.BackupTypeFull,
		Description: "Benchmark backup",
	}
	jsonBody, _ := json.Marshal(requestBody)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/backup/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}

func BenchmarkBackupHandlers_ListBackups(b *testing.B) {
	router, mockService, _ := setupBackupHandlerTest()
	
	expectedBackups := []*models.Backup{
		{ID: 1, Type: models.BackupTypeFull, Status: models.BackupStatusCompleted},
	}
	mockService.On("ListBackups", 50, 0).Return(expectedBackups, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/backup/list", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}