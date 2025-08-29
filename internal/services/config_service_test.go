package services

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// MockCacheService for testing
type MockCacheService struct {
	data map[string][]byte
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
	}
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, nil
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	// Simple pattern matching for testing
	for key := range m.data {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(m.data, key)
		}
	}
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *MockCacheService) Close() error {
	return nil
}

func setupConfigServiceTest() *ConfigService {
	cache := NewMockCacheService()
	service := &ConfigService{
		db:               nil, // Using nil for testing, would use test DB in real tests
		cache:            cache,
		config:           make(map[string]*models.Configuration),
		featureFlags:     make(map[string]*models.FeatureFlag),
		hotReloadEnabled: false, // Disable for testing
	}
	
	service.LoadDefaults()
	return service
}

func TestConfigService_Get(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		key         string
		expectError bool
		expected    string
	}{
		{
			name:        "Get existing configuration",
			key:         "site_name",
			expectError: false,
			expected:    "High Performance News Website",
		},
		{
			name:        "Get non-existent configuration",
			key:         "non_existent_key",
			expectError: true,
			expected:    "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := service.Get(tt.key)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if value != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, value)
			}
		})
	}
}

func TestConfigService_GetTyped(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		key         string
		expectError bool
		expected    interface{}
	}{
		{
			name:        "Get string configuration",
			key:         "site_name",
			expectError: false,
			expected:    "High Performance News Website",
		},
		{
			name:        "Get boolean configuration",
			key:         "static_generation",
			expectError: false,
			expected:    true,
		},
		{
			name:        "Get integer configuration",
			key:         "cache_ttl",
			expectError: false,
			expected:    3600,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := service.GetTyped(tt.key)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, value)
			}
		})
	}
}

func TestConfigService_Set(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		key         string
		value       interface{}
		expectError bool
	}{
		{
			name:        "Set existing string configuration",
			key:         "site_name",
			value:       "New Site Name",
			expectError: false,
		},
		{
			name:        "Set existing boolean configuration",
			key:         "static_generation",
			value:       false,
			expectError: false,
		},
		{
			name:        "Set non-existent configuration",
			key:         "non_existent_key",
			value:       "value",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Set(tt.key, tt.value)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Verify the value was set correctly
			if !tt.expectError {
				config, err := service.GetConfig(tt.key)
				if err != nil {
					t.Errorf("Failed to get updated configuration: %v", err)
				}
				
				typedValue, err := config.GetTypedValue()
				if err != nil {
					t.Errorf("Failed to get typed value: %v", err)
				}
				
				if typedValue != tt.value {
					t.Errorf("Expected %v, got %v", tt.value, typedValue)
				}
			}
		})
	}
}

func TestConfigService_GetAllByCategory(t *testing.T) {
	service := setupConfigServiceTest()
	
	siteConfigs := service.GetAllByCategory("site")
	
	if len(siteConfigs) == 0 {
		t.Error("Expected site configurations but got none")
	}
	
	// Check that all returned configs are in the site category
	for _, config := range siteConfigs {
		if config.Category != "site" {
			t.Errorf("Expected category 'site', got '%s'", config.Category)
		}
	}
}

func TestConfigService_FeatureFlags(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Create a test feature flag
	flag := &models.FeatureFlag{
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
	
	// Set the feature flag
	err := service.SetFeatureFlag(flag)
	if err != nil {
		t.Errorf("Failed to set feature flag: %v", err)
	}
	
	// Get the feature flag
	retrievedFlag, err := service.GetFeatureFlag("test_feature")
	if err != nil {
		t.Errorf("Failed to get feature flag: %v", err)
	}
	
	if retrievedFlag.Key != flag.Key {
		t.Errorf("Expected key %s, got %s", flag.Key, retrievedFlag.Key)
	}
	
	// Test feature flag evaluation
	context := map[string]interface{}{
		"user_id": uint64(1), // Should be enabled for user 1 (1 % 100 < 50)
	}
	
	enabled := service.IsFeatureEnabled("test_feature", context)
	if !enabled {
		t.Error("Expected feature to be enabled for user 1")
	}
	
	context["user_id"] = uint64(99) // Should be disabled for user 99 (99 % 100 >= 50)
	enabled = service.IsFeatureEnabled("test_feature", context)
	if enabled {
		t.Error("Expected feature to be disabled for user 99")
	}
}

func TestConfigService_Validation(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Add a configuration with validation rules
	config := &models.Configuration{
		Key:         "test_config",
		Value:       "10",
		Type:        models.ConfigTypeInt,
		Category:    "test",
		Description: "Test configuration with validation",
		Validation: &models.ConfigValidationRules{
			Required: true,
			MinValue: func() *float64 { v := 5.0; return &v }(),
			MaxValue: func() *float64 { v := 100.0; return &v }(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	service.config[config.Key] = config
	
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{
			name:        "Valid value within range",
			value:       50,
			expectError: false,
		},
		{
			name:        "Value below minimum",
			value:       3,
			expectError: true,
		},
		{
			name:        "Value above maximum",
			value:       150,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Set("test_config", tt.value)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigService_Subscribe(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Subscribe to configuration changes
	ch := service.Subscribe()
	
	// Update a configuration
	go func() {
		time.Sleep(10 * time.Millisecond)
		service.Set("site_name", "Updated Site Name")
	}()
	
	// Wait for the change event
	select {
	case event := <-ch:
		if event.Key != "site_name" {
			t.Errorf("Expected key 'site_name', got '%s'", event.Key)
		}
		if event.NewValue != "Updated Site Name" {
			t.Errorf("Expected new value 'Updated Site Name', got '%s'", event.NewValue)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for configuration change event")
	}
	
	// Unsubscribe
	service.Unsubscribe(ch)
}

func TestConfigService_CreateSnapshot(t *testing.T) {
	service := setupConfigServiceTest()
	
	snapshot, err := service.CreateSnapshot("test_snapshot", "Test snapshot", 1)
	if err != nil {
		t.Errorf("Failed to create snapshot: %v", err)
	}
	
	if snapshot.Name != "test_snapshot" {
		t.Errorf("Expected name 'test_snapshot', got '%s'", snapshot.Name)
	}
	
	if len(snapshot.Config) == 0 {
		t.Error("Expected snapshot to contain configurations")
	}
}

func TestConfigService_ValidateConfiguration(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Add a configuration with validation that should fail
	config := &models.Configuration{
		Key:         "invalid_config",
		Value:       "", // Empty value but required
		Type:        models.ConfigTypeString,
		Category:    "test",
		Description: "Invalid test configuration",
		Validation: &models.ConfigValidationRules{
			Required: true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	service.config[config.Key] = config
	
	errors := service.ValidateConfiguration()
	if len(errors) == 0 {
		t.Error("Expected validation errors but got none")
	}
	
	// Check that the error is about the required field
	found := false
	for _, err := range errors {
		if err.Error() == "validation failed for 'invalid_config': value is required" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected validation error for required field")
	}
}