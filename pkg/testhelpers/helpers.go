package testhelpers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// TestConfig holds configuration for test infrastructure
type TestConfig struct {
	DatabaseURL string
	CacheURL    string
	TestDataDir string
}

// GetTestConfig returns test configuration from environment or defaults
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: getEnvOrDefault("TEST_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/news_website_test?sslmode=disable"),
		CacheURL:    getEnvOrDefault("TEST_CACHE_URL", "redis://localhost:6379/1"),
		TestDataDir: getEnvOrDefault("TEST_DATA_DIR", "./testdata"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MockCacheService provides a unified mock cache service for all tests
type MockCacheService struct {
	mock.Mock
	data map[string][]byte
	ttls map[string]time.Time
	mu   sync.RWMutex
}

// NewMockCacheService creates a new mock cache service
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Time),
	}
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Get(0).([]byte), args.Error(1)
	}
	
	// Otherwise use internal data store
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if ttl, exists := m.ttls[key]; exists && time.Now().After(ttl) {
		delete(m.data, key)
		delete(m.ttls, key)
		return nil, fmt.Errorf("key not found")
	}
	
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	
	return nil, fmt.Errorf("key not found")
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Otherwise use internal data store
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data[key] = value
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	}
	
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Otherwise use internal data store
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.data, key)
	delete(m.ttls, key)
	
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Error(0)
	}
	
	// Simple pattern matching for testing (just prefix matching)
	m.mu.Lock()
	defer m.mu.Unlock()
	
	prefix := pattern
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix = pattern[:len(pattern)-1]
	}
	
	for key := range m.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(m.data, key)
			delete(m.ttls, key)
		}
	}
	
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	
	// If mock expectations are set, use them
	if len(args) > 0 {
		return args.Bool(0), args.Error(1)
	}
	
	// Otherwise use internal data store
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if ttl, exists := m.ttls[key]; exists && time.Now().After(ttl) {
		delete(m.data, key)
		delete(m.ttls, key)
		return false, nil
	}
	
	_, exists := m.data[key]
	return exists, nil
}

func (m *MockCacheService) Close() error {
	args := m.Called()
	if len(args) > 0 {
		return args.Error(0)
	}
	return nil
}

func (m *MockCacheService) Health(ctx context.Context) error {
	args := m.Called(ctx)
	if len(args) > 0 {
		return args.Error(0)
	}
	return nil
}

// Clear clears all data from the mock cache
func (m *MockCacheService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data = make(map[string][]byte)
	m.ttls = make(map[string]time.Time)
}