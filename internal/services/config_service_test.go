package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConfigService_GetSystemConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockCacheService, sqlmock.Sqlmock)
		expectedError bool
		expectedName  string
	}{
		{
			name: "successful config retrieval from cache",
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				config := &SystemConfiguration{
					SiteName:        "Cached Site",
					SiteDescription: "From cache",
					UpdatedAt:       time.Now(),
				}
				configData, _ := json.Marshal(config)
				cache.On("Get", "system:config").Return(configData, nil)
			},
			expectedError: false,
			expectedName:  "Cached Site",
		},
		{
			name: "successful config retrieval from database",
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				// Cache miss
				cache.On("Get", "system:config").Return([]byte{}, assert.AnError)
				
				// Database query
				configData := map[string]interface{}{
					"site_name":        "DB Site",
					"site_description": "From database",
					"cache_enabled":    true,
				}
				configJSON, _ := json.Marshal(configData)
				
				rows := sqlmock.NewRows([]string{"config_data", "updated_at", "updated_by"}).
					AddRow(string(configJSON), time.Now(), 1)
				
				dbMock.ExpectQuery(`SELECT config_data, updated_at, updated_by FROM system_config WHERE id = 1`).
					WillReturnRows(rows)
				
				// Cache set
				cache.On("Set", "system:config", mock.AnythingOfType("[]uint8"), 5*time.Minute).Return(nil)
			},
			expectedError: false,
			expectedName:  "DB Site",
		},
		{
			name: "no config in database - create default",
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				// Cache miss
				cache.On("Get", "system:config").Return([]byte{}, assert.AnError)
				
				// Database query returns no rows
				dbMock.ExpectQuery(`SELECT config_data, updated_at, updated_by FROM system_config WHERE id = 1`).
					WillReturnError(sql.ErrNoRows)
				
				// Insert default config
				dbMock.ExpectExec(`INSERT INTO system_config \(id, config_data, updated_at, updated_by\)`).
					WithArgs(1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				
				// Cache set
				cache.On("Set", "system:config", mock.AnythingOfType("[]uint8"), 5*time.Minute).Return(nil)
			},
			expectedError: false,
			expectedName:  "High Performance News Website", // Default name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db, dbMock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// Create mock cache
			mockCache := &MockCacheService{}

			// Setup mocks
			tt.setupMocks(mockCache, dbMock)

			// Create service
			service := NewConfigService(db, mockCache)

			// Test
			config, err := service.GetSystemConfiguration()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.expectedName, config.SiteName)
				
				// Verify sensitive data is sanitized
				assert.Empty(t, config.NotificationSettings.SMTPPassword)
				assert.Empty(t, config.NotificationSettings.SlackWebhookURL)
				assert.Empty(t, config.BackupSettings.S3AccessKey)
				assert.Empty(t, config.BackupSettings.S3SecretKey)
			}

			// Verify expectations
			mockCache.AssertExpectations(t)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestConfigService_UpdateSystemConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		updates       map[string]interface{}
		setupMocks    func(*MockCacheService, sqlmock.Sqlmock)
		expectedError bool
		expectedName  string
	}{
		{
			name: "successful config update",
			updates: map[string]interface{}{
				"site_name":        "Updated Site Name",
				"maintenance_mode": true,
				"cache_enabled":    false,
			},
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				// Get current config (cache miss, then DB)
				cache.On("Get", "system:config").Return([]byte{}, assert.AnError)
				
				configData := map[string]interface{}{
					"site_name":        "Original Site",
					"site_description": "Original description",
					"cache_enabled":    true,
				}
				configJSON, _ := json.Marshal(configData)
				
				rows := sqlmock.NewRows([]string{"config_data", "updated_at", "updated_by"}).
					AddRow(string(configJSON), time.Now(), 1)
				
				dbMock.ExpectQuery(`SELECT config_data, updated_at, updated_by FROM system_config WHERE id = 1`).
					WillReturnRows(rows)
				
				cache.On("Set", "system:config", mock.AnythingOfType("[]uint8"), 5*time.Minute).Return(nil)
				
				// Update config in database
				dbMock.ExpectExec(`UPDATE system_config SET config_data = \$1, updated_at = \$2, updated_by = \$3 WHERE id = 1`).
					WithArgs(mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), uint64(0)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				
				// Clear cache
				cache.On("Delete", "system:config").Return(nil)
			},
			expectedError: false,
			expectedName:  "Updated Site Name",
		},
		{
			name: "validation error - empty site name",
			updates: map[string]interface{}{
				"site_name": "",
			},
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				// Get current config
				cache.On("Get", "system:config").Return([]byte{}, assert.AnError)
				
				configData := map[string]interface{}{
					"site_name":        "Original Site",
					"site_description": "Original description",
				}
				configJSON, _ := json.Marshal(configData)
				
				rows := sqlmock.NewRows([]string{"config_data", "updated_at", "updated_by"}).
					AddRow(string(configJSON), time.Now(), 1)
				
				dbMock.ExpectQuery(`SELECT config_data, updated_at, updated_by FROM system_config WHERE id = 1`).
					WillReturnRows(rows)
				
				cache.On("Set", "system:config", mock.AnythingOfType("[]uint8"), 5*time.Minute).Return(nil)
			},
			expectedError: true,
		},
		{
			name: "validation error - invalid articles per page",
			updates: map[string]interface{}{
				"articles_per_page": 150, // Too high
			},
			setupMocks: func(cache *MockCacheService, dbMock sqlmock.Sqlmock) {
				// Get current config
				cache.On("Get", "system:config").Return([]byte{}, assert.AnError)
				
				configData := map[string]interface{}{
					"site_name":         "Original Site",
					"articles_per_page": 20,
				}
				configJSON, _ := json.Marshal(configData)
				
				rows := sqlmock.NewRows([]string{"config_data", "updated_at", "updated_by"}).
					AddRow(string(configJSON), time.Now(), 1)
				
				dbMock.ExpectQuery(`SELECT config_data, updated_at, updated_by FROM system_config WHERE id = 1`).
					WillReturnRows(rows)
				
				cache.On("Set", "system:config", mock.AnythingOfType("[]uint8"), 5*time.Minute).Return(nil)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db, dbMock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// Create mock cache
			mockCache := &MockCacheService{}

			// Setup mocks
			tt.setupMocks(mockCache, dbMock)

			// Create service
			service := NewConfigService(db, mockCache)

			// Test
			config, err := service.UpdateSystemConfiguration(tt.updates)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.expectedName, config.SiteName)
				
				// Verify sensitive data is sanitized
				assert.Empty(t, config.NotificationSettings.SMTPPassword)
				assert.Empty(t, config.NotificationSettings.SlackWebhookURL)
				assert.Empty(t, config.BackupSettings.S3AccessKey)
				assert.Empty(t, config.BackupSettings.S3SecretKey)
			}

			// Verify expectations
			mockCache.AssertExpectations(t)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestConfigService_createDefaultConfig(t *testing.T) {
	// Create mock database
	db, dbMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Create mock cache
	mockCache := &MockCacheService{}

	// Setup mock
	dbMock.ExpectExec(`INSERT INTO system_config \(id, config_data, updated_at, updated_by\)`).
		WithArgs(1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Create service
	service := NewConfigService(db, mockCache)

	// Test
	config, err := service.createDefaultConfig()

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "High Performance News Website", config.SiteName)
	assert.Equal(t, "A fast and scalable news platform", config.SiteDescription)
	assert.Equal(t, 20, config.ArticlesPerPage)
	assert.True(t, config.CacheEnabled)
	assert.True(t, config.SearchEnabled)
	assert.True(t, config.CommentsEnabled)
	assert.False(t, config.MaintenanceMode)

	// Verify SEO settings
	assert.Equal(t, "High Performance News Website", config.SEOSettings.MetaTitle)
	assert.True(t, config.SEOSettings.SitemapEnabled)
	assert.True(t, config.SEOSettings.RobotsEnabled)

	// Verify security settings
	assert.Equal(t, 24, config.SecuritySettings.JWTExpirationHours)
	assert.Equal(t, 8, config.SecuritySettings.PasswordMinLength)
	assert.True(t, config.SecuritySettings.RequireStrongPassword)
	assert.Equal(t, 5, config.SecuritySettings.MaxLoginAttempts)

	// Verify backup settings
	assert.False(t, config.BackupSettings.AutoBackupEnabled)
	assert.Equal(t, 24, config.BackupSettings.BackupFrequencyHours)
	assert.Equal(t, 30, config.BackupSettings.RetentionDays)

	// Verify expectations
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestConfigService_validateConfig(t *testing.T) {
	service := &ConfigService{}

	tests := []struct {
		name          string
		config        *SystemConfiguration
		expectedError bool
		errorContains string
	}{
		{
			name: "valid configuration",
			config: &SystemConfiguration{
				SiteName:        "Valid Site",
				ArticlesPerPage: 20,
				MaxUploadSize:   10,
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 8,
				},
			},
			expectedError: false,
		},
		{
			name: "empty site name",
			config: &SystemConfiguration{
				SiteName:        "",
				ArticlesPerPage: 20,
				MaxUploadSize:   10,
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 8,
				},
			},
			expectedError: true,
			errorContains: "site_name cannot be empty",
		},
		{
			name: "invalid articles per page - too low",
			config: &SystemConfiguration{
				SiteName:        "Valid Site",
				ArticlesPerPage: 0,
				MaxUploadSize:   10,
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 8,
				},
			},
			expectedError: true,
			errorContains: "articles_per_page must be between 1 and 100",
		},
		{
			name: "invalid articles per page - too high",
			config: &SystemConfiguration{
				SiteName:        "Valid Site",
				ArticlesPerPage: 150,
				MaxUploadSize:   10,
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 8,
				},
			},
			expectedError: true,
			errorContains: "articles_per_page must be between 1 and 100",
		},
		{
			name: "invalid max upload size",
			config: &SystemConfiguration{
				SiteName:        "Valid Site",
				ArticlesPerPage: 20,
				MaxUploadSize:   150, // Too high
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 8,
				},
			},
			expectedError: true,
			errorContains: "max_upload_size must be between 1 and 100 MB",
		},
		{
			name: "invalid password min length",
			config: &SystemConfiguration{
				SiteName:        "Valid Site",
				ArticlesPerPage: 20,
				MaxUploadSize:   10,
				SecuritySettings: SecuritySettings{
					PasswordMinLength: 4, // Too short
				},
			},
			expectedError: true,
			errorContains: "password_min_length must be at least 6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateConfig(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigService_sanitizeConfig(t *testing.T) {
	service := &ConfigService{}

	config := &SystemConfiguration{
		NotificationSettings: NotificationSettings{
			SMTPPassword:    "secret-password",
			SlackWebhookURL: "https://hooks.slack.com/secret",
		},
		BackupSettings: BackupSettings{
			S3AccessKey: "AKIAIOSFODNN7EXAMPLE",
			S3SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
	}

	service.sanitizeConfig(config)

	// Verify sensitive data is removed
	assert.Empty(t, config.NotificationSettings.SMTPPassword)
	assert.Empty(t, config.NotificationSettings.SlackWebhookURL)
	assert.Empty(t, config.BackupSettings.S3AccessKey)
	assert.Empty(t, config.BackupSettings.S3SecretKey)
}