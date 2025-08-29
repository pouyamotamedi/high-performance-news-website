package config

import (
	"time"

	"high-performance-news-website/internal/models"
)

// CDNConfig holds CDN-related configuration
type CDNConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	Provider          string        `yaml:"provider" json:"provider"`
	APIKey            string        `yaml:"api_key" json:"api_key"`
	APISecret         string        `yaml:"api_secret" json:"api_secret"`
	ZoneID            string        `yaml:"zone_id" json:"zone_id"`
	Domain            string        `yaml:"domain" json:"domain"`
	PurgeTimeout      time.Duration `yaml:"purge_timeout" json:"purge_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	FailoverEnabled   bool          `yaml:"failover_enabled" json:"failover_enabled"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
}

// LoadCDNConfig loads CDN configuration from environment variables
func LoadCDNConfig() *CDNConfig {
	config := &CDNConfig{
		Enabled:             getEnvBool("CDN_ENABLED", false),
		Provider:            getEnvString("CDN_PROVIDER", "cloudflare"),
		APIKey:              getEnvString("CDN_API_KEY", ""),
		APISecret:           getEnvString("CDN_API_SECRET", ""),
		ZoneID:              getEnvString("CDN_ZONE_ID", ""),
		Domain:              getEnvString("CDN_DOMAIN", ""),
		PurgeTimeout:        getEnvDuration("CDN_PURGE_TIMEOUT", 30*time.Second),
		HealthCheckInterval: getEnvDuration("CDN_HEALTH_CHECK_INTERVAL", 5*time.Minute),
		FailoverEnabled:     getEnvBool("CDN_FAILOVER_ENABLED", true),
		MaxRetries:          getEnvInt("CDN_MAX_RETRIES", 3),
	}

	return config
}

// ToModel converts CDNConfig to models.CDNConfig
func (c *CDNConfig) ToModel() *models.CDNConfig {
	return &models.CDNConfig{
		Provider:  c.Provider,
		APIKey:    c.APIKey,
		APISecret: c.APISecret,
		ZoneID:    c.ZoneID,
		Domain:    c.Domain,
		Enabled:   c.Enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate validates the CDN configuration
func (c *CDNConfig) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if CDN is disabled
	}

	if c.Provider == "" {
		return &ValidationError{Field: "provider", Message: "CDN provider is required"}
	}

	if c.APIKey == "" {
		return &ValidationError{Field: "api_key", Message: "CDN API key is required"}
	}

	if c.ZoneID == "" {
		return &ValidationError{Field: "zone_id", Message: "CDN zone ID is required"}
	}

	if c.Domain == "" {
		return &ValidationError{Field: "domain", Message: "CDN domain is required"}
	}

	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

