package services

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
)

// ConfigService handles system configuration management with hot reloading
type ConfigService struct {
	db                *sql.DB
	cache             cache.CacheService
	config            map[string]*models.Configuration
	featureFlags      map[string]*models.FeatureFlag
	mutex             sync.RWMutex
	subscribers       []chan ConfigChangeEvent
	subscribersMutex  sync.RWMutex
	hotReloadEnabled  bool
	reloadInterval    time.Duration
	stopReload        chan bool
}

// ConfigChangeEvent represents a configuration change event
type ConfigChangeEvent struct {
	Key       string      `json:"key"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Timestamp time.Time   `json:"timestamp"`
	ChangedBy uint64      `json:"changed_by"`
}

// NewConfigService creates a new ConfigService instance with hot reloading
func NewConfigService(db *sql.DB, cache cache.CacheService) *ConfigService {
	service := &ConfigService{
		db:               db,
		cache:            cache,
		config:           make(map[string]*models.Configuration),
		featureFlags:     make(map[string]*models.FeatureFlag),
		mutex:            sync.RWMutex{},
		subscribers:      make([]chan ConfigChangeEvent, 0),
		subscribersMutex: sync.RWMutex{},
		hotReloadEnabled: true,
		reloadInterval:   30 * time.Second,
		stopReload:       make(chan bool),
	}

	// Initialize with default configurations
	service.LoadDefaults()
	
	// Start hot reload if enabled
	if service.hotReloadEnabled {
		go service.startHotReload()
	}

	return service
}

// Get retrieves a configuration value by key
func (cs *ConfigService) Get(key string) (string, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	config, exists := cs.config[key]
	if !exists {
		return "", fmt.Errorf("configuration key '%s' not found", key)
	}
	
	return config.Value, nil
}

// GetTyped retrieves a configuration value in its proper type
func (cs *ConfigService) GetTyped(key string) (interface{}, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	config, exists := cs.config[key]
	if !exists {
		return nil, fmt.Errorf("configuration key '%s' not found", key)
	}
	
	return config.GetTypedValue()
}

// GetConfig retrieves the full configuration object
func (cs *ConfigService) GetConfig(key string) (*models.Configuration, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	config, exists := cs.config[key]
	if !exists {
		return nil, fmt.Errorf("configuration key '%s' not found", key)
	}
	
	return config, nil
}

// Set updates a configuration value with validation and history tracking
func (cs *ConfigService) Set(key string, value interface{}) error {
	return cs.SetWithContext(context.Background(), key, value, 0, "")
}

// SetWithContext updates a configuration value with context and history
func (cs *ConfigService) SetWithContext(ctx context.Context, key string, value interface{}, changedBy uint64, reason string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	// Get existing configuration
	existingConfig, exists := cs.config[key]
	if !exists {
		return fmt.Errorf("configuration key '%s' not found", key)
	}
	
	// Store old value for history
	oldValue := existingConfig.Value
	
	// Validate the new value
	if err := cs.validateValue(existingConfig, value); err != nil {
		return fmt.Errorf("validation failed for key '%s': %w", key, err)
	}
	
	// Create new configuration with updated value
	newConfig := *existingConfig
	if err := newConfig.SetTypedValue(value); err != nil {
		return fmt.Errorf("failed to set typed value: %w", err)
	}
	newConfig.UpdatedAt = time.Now()
	
	// Update in database
	if err := cs.updateConfigInDB(ctx, &newConfig); err != nil {
		return fmt.Errorf("failed to update configuration in database: %w", err)
	}
	
	// Record history
	if err := cs.recordConfigHistory(ctx, key, oldValue, newConfig.Value, changedBy, reason); err != nil {
		// Log error but don't fail the update
		fmt.Printf("Warning: failed to record configuration history: %v\n", err)
	}
	
	// Update in memory
	cs.config[key] = &newConfig
	
	// Clear cache
	cs.clearConfigCache(key)
	
	// Notify subscribers
	go cs.notifySubscribers(ConfigChangeEvent{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  newConfig.Value,
		Timestamp: time.Now(),
		ChangedBy: changedBy,
	})
	
	return nil
}

// GetAll returns all configuration values
func (cs *ConfigService) GetAll() map[string]*models.Configuration {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	result := make(map[string]*models.Configuration)
	for k, v := range cs.config {
		// Create a copy to avoid race conditions
		configCopy := *v
		result[k] = &configCopy
	}
	
	return result
}

// GetAllByCategory returns all configurations in a specific category
func (cs *ConfigService) GetAllByCategory(category string) map[string]*models.Configuration {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	result := make(map[string]*models.Configuration)
	for k, v := range cs.config {
		if v.Category == category {
			configCopy := *v
			result[k] = &configCopy
		}
	}
	
	return result
}

// Delete removes a configuration key (only if not required)
func (cs *ConfigService) Delete(key string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	config, exists := cs.config[key]
	if !exists {
		return fmt.Errorf("configuration key '%s' not found", key)
	}
	
	// Check if configuration is required
	if config.Validation != nil && config.Validation.Required {
		return fmt.Errorf("cannot delete required configuration key '%s'", key)
	}
	
	// Delete from database
	if err := cs.deleteConfigFromDB(context.Background(), key); err != nil {
		return fmt.Errorf("failed to delete configuration from database: %w", err)
	}
	
	// Delete from memory
	delete(cs.config, key)
	
	// Clear cache
	cs.clearConfigCache(key)
	
	return nil
}

// LoadDefaults loads default configuration values
func (cs *ConfigService) LoadDefaults() {
	defaults := []*models.Configuration{
		{Key: "site_name", Value: "High Performance News Website", Type: models.ConfigTypeString, Category: "site", Description: "Website name"},
		{Key: "site_description", Value: "A fast and scalable news platform", Type: models.ConfigTypeString, Category: "site", Description: "Website description"},
		{Key: "site_url", Value: "https://example.com", Type: models.ConfigTypeString, Category: "site", Description: "Website URL"},
		{Key: "site_logo", Value: "", Type: models.ConfigTypeString, Category: "site", Description: "Website logo URL"},
		{Key: "site_favicon", Value: "", Type: models.ConfigTypeString, Category: "site", Description: "Website favicon URL"},
		{Key: "cache_ttl", Value: "3600", Type: models.ConfigTypeInt, Category: "performance", Description: "Cache TTL in seconds"},
		{Key: "static_generation", Value: "true", Type: models.ConfigTypeBool, Category: "performance", Description: "Enable static HTML generation"},
		{Key: "compression_enabled", Value: "true", Type: models.ConfigTypeBool, Category: "performance", Description: "Enable compression"},
		{Key: "comments_enabled", Value: "true", Type: models.ConfigTypeBool, Category: "features", Description: "Enable comments system"},
		{Key: "registration_enabled", Value: "false", Type: models.ConfigTypeBool, Category: "features", Description: "Enable user registration"},
		{Key: "search_enabled", Value: "true", Type: models.ConfigTypeBool, Category: "features", Description: "Enable search functionality"},
		{Key: "analytics_enabled", Value: "false", Type: models.ConfigTypeBool, Category: "features", Description: "Enable analytics tracking"},
		{Key: "social_sharing", Value: "true", Type: models.ConfigTypeBool, Category: "features", Description: "Enable social sharing"},
		{Key: "newsletter_enabled", Value: "false", Type: models.ConfigTypeBool, Category: "features", Description: "Enable newsletter"},
		{Key: "theme", Value: "default", Type: models.ConfigTypeString, Category: "appearance", Description: "Website theme"},
		{Key: "primary_color", Value: "#007bff", Type: models.ConfigTypeString, Category: "appearance", Description: "Primary color"},
		{Key: "secondary_color", Value: "#6c757d", Type: models.ConfigTypeString, Category: "appearance", Description: "Secondary color"},
		{Key: "articles_per_page", Value: "10", Type: models.ConfigTypeInt, Category: "content", Description: "Articles per page"},
		{Key: "excerpt_length", Value: "150", Type: models.ConfigTypeInt, Category: "content", Description: "Article excerpt length"},
		{Key: "allow_comments", Value: "true", Type: models.ConfigTypeBool, Category: "content", Description: "Allow comments on articles"},
		{Key: "moderate_comments", Value: "true", Type: models.ConfigTypeBool, Category: "content", Description: "Moderate comments before publishing"},
		{Key: "meta_title", Value: "", Type: models.ConfigTypeString, Category: "seo", Description: "Default meta title"},
		{Key: "meta_description", Value: "", Type: models.ConfigTypeString, Category: "seo", Description: "Default meta description"},
		{Key: "meta_keywords", Value: "", Type: models.ConfigTypeString, Category: "seo", Description: "Default meta keywords"},
		{Key: "google_analytics", Value: "", Type: models.ConfigTypeString, Category: "analytics", Description: "Google Analytics tracking ID"},
		{Key: "admin_email", Value: "", Type: models.ConfigTypeString, Category: "system", Description: "Administrator email"},
	}
	
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	for _, config := range defaults {
		if _, exists := cs.config[config.Key]; !exists {
			config.CreatedAt = time.Now()
			config.UpdatedAt = time.Now()
			cs.config[config.Key] = config
		}
	}
}

// Feature Flag Methods

// GetFeatureFlag retrieves a feature flag by key
func (cs *ConfigService) GetFeatureFlag(key string) (*models.FeatureFlag, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	flag, exists := cs.featureFlags[key]
	if !exists {
		return nil, fmt.Errorf("feature flag '%s' not found", key)
	}
	
	return flag, nil
}

// IsFeatureEnabled checks if a feature flag is enabled for given context
func (cs *ConfigService) IsFeatureEnabled(key string, context map[string]interface{}) bool {
	flag, err := cs.GetFeatureFlag(key)
	if err != nil {
		return false
	}
	
	return flag.IsEnabled(context)
}

// SetFeatureFlag creates or updates a feature flag
func (cs *ConfigService) SetFeatureFlag(flag *models.FeatureFlag) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	// Update in database
	if err := cs.updateFeatureFlagInDB(context.Background(), flag); err != nil {
		return fmt.Errorf("failed to update feature flag in database: %w", err)
	}
	
	// Update in memory
	cs.featureFlags[flag.Key] = flag
	
	// Clear cache
	cs.clearFeatureFlagCache(flag.Key)
	
	return nil
}

// GetAllFeatureFlags returns all feature flags
func (cs *ConfigService) GetAllFeatureFlags() map[string]*models.FeatureFlag {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	result := make(map[string]*models.FeatureFlag)
	for k, v := range cs.featureFlags {
		flagCopy := *v
		result[k] = &flagCopy
	}
	
	return result
}

// Configuration Management Methods

// CreateSnapshot creates a configuration snapshot for rollback
func (cs *ConfigService) CreateSnapshot(name, description string, createdBy uint64) (*models.ConfigurationSnapshot, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	snapshot := &models.ConfigurationSnapshot{
		Name:        name,
		Description: description,
		Config:      make(map[string]models.Configuration),
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
	
	// Copy current configuration
	for k, v := range cs.config {
		snapshot.Config[k] = *v
	}
	
	// Save to database
	if err := cs.saveSnapshotToDB(context.Background(), snapshot); err != nil {
		return nil, fmt.Errorf("failed to save snapshot to database: %w", err)
	}
	
	return snapshot, nil
}

// RestoreSnapshot restores configuration from a snapshot
func (cs *ConfigService) RestoreSnapshot(snapshotID uint64, restoredBy uint64) error {
	// Load snapshot from database
	snapshot, err := cs.loadSnapshotFromDB(context.Background(), snapshotID)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}
	
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	// Restore configuration
	for key, config := range snapshot.Config {
		// Record history for each change
		if existingConfig, exists := cs.config[key]; exists {
			cs.recordConfigHistory(context.Background(), key, existingConfig.Value, config.Value, restoredBy, fmt.Sprintf("Restored from snapshot: %s", snapshot.Name))
		}
		
		// Update configuration
		config.UpdatedAt = time.Now()
		cs.config[key] = &config
		
		// Update in database
		cs.updateConfigInDB(context.Background(), &config)
		
		// Clear cache
		cs.clearConfigCache(key)
	}
	
	return nil
}

// ValidateConfiguration validates all configurations
func (cs *ConfigService) ValidateConfiguration() []error {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	var errors []error
	
	for key, config := range cs.config {
		if config.Validation != nil {
			if err := cs.validateConfigValue(config); err != nil {
				errors = append(errors, fmt.Errorf("validation failed for '%s': %w", key, err))
			}
		}
	}
	
	return errors
}

// Hot Reload Methods

// Subscribe subscribes to configuration change events
func (cs *ConfigService) Subscribe() <-chan ConfigChangeEvent {
	cs.subscribersMutex.Lock()
	defer cs.subscribersMutex.Unlock()
	
	ch := make(chan ConfigChangeEvent, 100)
	cs.subscribers = append(cs.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscription
func (cs *ConfigService) Unsubscribe(ch <-chan ConfigChangeEvent) {
	cs.subscribersMutex.Lock()
	defer cs.subscribersMutex.Unlock()
	
	for i, subscriber := range cs.subscribers {
		if subscriber == ch {
			cs.subscribers = append(cs.subscribers[:i], cs.subscribers[i+1:]...)
			close(subscriber)
			break
		}
	}
}

// ReloadFromDatabase reloads configuration from database
func (cs *ConfigService) ReloadFromDatabase() error {
	configs, err := cs.loadConfigsFromDB(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load configurations from database: %w", err)
	}
	
	flags, err := cs.loadFeatureFlagsFromDB(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load feature flags from database: %w", err)
	}
	
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	// Update configurations
	for _, config := range configs {
		cs.config[config.Key] = config
	}
	
	// Update feature flags
	for _, flag := range flags {
		cs.featureFlags[flag.Key] = flag
	}
	
	return nil
}

// Stop stops the hot reload process
func (cs *ConfigService) Stop() {
	if cs.hotReloadEnabled {
		cs.stopReload <- true
	}
}

// Private helper methods

// startHotReload starts the hot reload process
func (cs *ConfigService) startHotReload() {
	ticker := time.NewTicker(cs.reloadInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := cs.ReloadFromDatabase(); err != nil {
				fmt.Printf("Hot reload error: %v\n", err)
			}
		case <-cs.stopReload:
			return
		}
	}
}

// notifySubscribers notifies all subscribers of configuration changes
func (cs *ConfigService) notifySubscribers(event ConfigChangeEvent) {
	cs.subscribersMutex.RLock()
	defer cs.subscribersMutex.RUnlock()
	
	for _, subscriber := range cs.subscribers {
		select {
		case subscriber <- event:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// validateValue validates a configuration value against its rules
func (cs *ConfigService) validateValue(config *models.Configuration, value interface{}) error {
	if config.Validation == nil {
		return nil
	}
	
	return cs.validateConfigValue(&models.Configuration{
		Type:       config.Type,
		Validation: config.Validation,
		Value:      fmt.Sprintf("%v", value),
	})
}

// validateConfigValue validates a configuration value
func (cs *ConfigService) validateConfigValue(config *models.Configuration) error {
	if config.Validation == nil {
		return nil
	}
	
	rules := config.Validation
	
	// Required check
	if rules.Required && config.Value == "" {
		return fmt.Errorf("value is required")
	}
	
	// Type-specific validation
	switch config.Type {
	case models.ConfigTypeString:
		return cs.validateStringValue(config.Value, rules)
	case models.ConfigTypeInt:
		return cs.validateIntValue(config.Value, rules)
	case models.ConfigTypeFloat:
		return cs.validateFloatValue(config.Value, rules)
	case models.ConfigTypeBool:
		return cs.validateBoolValue(config.Value, rules)
	}
	
	return nil
}

// validateStringValue validates string configuration values
func (cs *ConfigService) validateStringValue(value string, rules *models.ConfigValidationRules) error {
	// Length validation
	if rules.MinLength != nil && len(value) < *rules.MinLength {
		return fmt.Errorf("value must be at least %d characters", *rules.MinLength)
	}
	if rules.MaxLength != nil && len(value) > *rules.MaxLength {
		return fmt.Errorf("value must be at most %d characters", *rules.MaxLength)
	}
	
	// Pattern validation
	if rules.Pattern != "" {
		matched, err := regexp.MatchString(rules.Pattern, value)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("value does not match required pattern")
		}
	}
	
	// Options validation
	if len(rules.Options) > 0 {
		for _, option := range rules.Options {
			if value == option {
				return nil
			}
		}
		return fmt.Errorf("value must be one of: %v", rules.Options)
	}
	
	return nil
}

// validateIntValue validates integer configuration values
func (cs *ConfigService) validateIntValue(value string, rules *models.ConfigValidationRules) error {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("value must be an integer")
	}
	
	floatValue := float64(intValue)
	if rules.MinValue != nil && floatValue < *rules.MinValue {
		return fmt.Errorf("value must be at least %v", *rules.MinValue)
	}
	if rules.MaxValue != nil && floatValue > *rules.MaxValue {
		return fmt.Errorf("value must be at most %v", *rules.MaxValue)
	}
	
	return nil
}

// validateFloatValue validates float configuration values
func (cs *ConfigService) validateFloatValue(value string, rules *models.ConfigValidationRules) error {
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("value must be a number")
	}
	
	if rules.MinValue != nil && floatValue < *rules.MinValue {
		return fmt.Errorf("value must be at least %v", *rules.MinValue)
	}
	if rules.MaxValue != nil && floatValue > *rules.MaxValue {
		return fmt.Errorf("value must be at most %v", *rules.MaxValue)
	}
	
	return nil
}

// validateBoolValue validates boolean configuration values
func (cs *ConfigService) validateBoolValue(value string, rules *models.ConfigValidationRules) error {
	_, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("value must be a boolean (true/false)")
	}
	
	return nil
}

// clearConfigCache clears cache for a configuration key
func (cs *ConfigService) clearConfigCache(key string) {
	if cs.cache != nil {
		ctx := context.Background()
		cs.cache.Delete(ctx, fmt.Sprintf("config:%s", key))
	}
}

// clearFeatureFlagCache clears cache for a feature flag key
func (cs *ConfigService) clearFeatureFlagCache(key string) {
	if cs.cache != nil {
		ctx := context.Background()
		cs.cache.Delete(ctx, fmt.Sprintf("feature_flag:%s", key))
	}
}

// Database operations (placeholder implementations)
// In a real implementation, these would interact with the database

func (cs *ConfigService) updateConfigInDB(ctx context.Context, config *models.Configuration) error {
	// TODO: Implement database update
	return nil
}

func (cs *ConfigService) deleteConfigFromDB(ctx context.Context, key string) error {
	// TODO: Implement database deletion
	return nil
}

func (cs *ConfigService) recordConfigHistory(ctx context.Context, key, oldValue, newValue string, changedBy uint64, reason string) error {
	// TODO: Implement history recording
	return nil
}

func (cs *ConfigService) updateFeatureFlagInDB(ctx context.Context, flag *models.FeatureFlag) error {
	// TODO: Implement feature flag database update
	return nil
}

func (cs *ConfigService) saveSnapshotToDB(ctx context.Context, snapshot *models.ConfigurationSnapshot) error {
	// TODO: Implement snapshot saving
	return nil
}

func (cs *ConfigService) loadSnapshotFromDB(ctx context.Context, snapshotID uint64) (*models.ConfigurationSnapshot, error) {
	// TODO: Implement snapshot loading
	return nil, fmt.Errorf("not implemented")
}

func (cs *ConfigService) loadConfigsFromDB(ctx context.Context) ([]*models.Configuration, error) {
	// TODO: Implement configuration loading from database
	return nil, nil
}

func (cs *ConfigService) loadFeatureFlagsFromDB(ctx context.Context) ([]*models.FeatureFlag, error) {
	// TODO: Implement feature flag loading from database
	return nil, nil
}