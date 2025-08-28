package services

import (
	"fmt"
	"sync"
)

// ConfigService handles system configuration management
type ConfigService struct {
	config map[string]string
	mutex  sync.RWMutex
}

// NewConfigService creates a new ConfigService instance
func NewConfigService() *ConfigService {
	return &ConfigService{
		config: make(map[string]string),
		mutex:  sync.RWMutex{},
	}
}

// Get retrieves a configuration value by key
func (cs *ConfigService) Get(key string) (string, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	value, exists := cs.config[key]
	if !exists {
		return "", fmt.Errorf("configuration key '%s' not found", key)
	}
	
	return value, nil
}

// Set updates a configuration value
func (cs *ConfigService) Set(key string, value interface{}) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case int:
		strValue = fmt.Sprintf("%d", v)
	case float64:
		strValue = fmt.Sprintf("%.2f", v)
	case bool:
		strValue = fmt.Sprintf("%t", v)
	default:
		strValue = fmt.Sprintf("%v", v)
	}
	
	cs.config[key] = strValue
	return nil
}

// GetAll returns all configuration values
func (cs *ConfigService) GetAll() map[string]string {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	result := make(map[string]string)
	for k, v := range cs.config {
		result[k] = v
	}
	
	return result
}

// Delete removes a configuration key
func (cs *ConfigService) Delete(key string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	delete(cs.config, key)
	return nil
}

// LoadDefaults loads default configuration values
func (cs *ConfigService) LoadDefaults() {
	defaults := map[string]string{
		"site_name":           "High Performance News Website",
		"site_description":    "A fast and scalable news platform",
		"site_url":            "https://example.com",
		"site_logo":           "",
		"site_favicon":        "",
		"cache_ttl":           "3600",
		"static_generation":   "true",
		"compression_enabled": "true",
		"comments_enabled":    "true",
		"registration_enabled": "false",
		"search_enabled":      "true",
		"analytics_enabled":   "false",
		"social_sharing":      "true",
		"newsletter_enabled":  "false",
		"theme":               "default",
		"primary_color":       "#007bff",
		"secondary_color":     "#6c757d",
		"articles_per_page":   "10",
		"excerpt_length":      "150",
		"allow_comments":      "true",
		"moderate_comments":   "true",
		"meta_title":          "",
		"meta_description":    "",
		"meta_keywords":       "",
		"google_analytics":    "",
		"admin_email":         "",
	}
	
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	for key, value := range defaults {
		if _, exists := cs.config[key]; !exists {
			cs.config[key] = value
		}
	}
}