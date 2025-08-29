package config

import (
	"time"
)

// AnalyticsConfig holds analytics-related configuration
type AnalyticsConfig struct {
	// Tracking settings
	EnableTracking        bool          `json:"enable_tracking"`
	EnableIPTracking      bool          `json:"enable_ip_tracking"`
	EnableUserAgentTracking bool        `json:"enable_user_agent_tracking"`
	EnableBehaviorTracking bool         `json:"enable_behavior_tracking"`
	EnablePerformanceTracking bool      `json:"enable_performance_tracking"`
	
	// Data retention settings
	ViewDataRetentionDays      int `json:"view_data_retention_days"`
	BehaviorDataRetentionDays  int `json:"behavior_data_retention_days"`
	ReportRetentionDays        int `json:"report_retention_days"`
	
	// Performance settings
	BatchSize              int           `json:"batch_size"`
	FlushInterval          time.Duration `json:"flush_interval"`
	MaxConcurrentRequests  int           `json:"max_concurrent_requests"`
	
	// Export settings
	MaxExportRows          int           `json:"max_export_rows"`
	ExportTimeoutSeconds   int           `json:"export_timeout_seconds"`
	
	// Privacy settings
	AnonymizeIPs           bool          `json:"anonymize_ips"`
	RespectDoNotTrack      bool          `json:"respect_do_not_track"`
	CookieConsentRequired  bool          `json:"cookie_consent_required"`
	
	// Geolocation settings
	EnableGeolocation      bool          `json:"enable_geolocation"`
	GeolocationProvider    string        `json:"geolocation_provider"`
	GeolocationAPIKey      string        `json:"geolocation_api_key"`
	
	// Real-time analytics
	EnableRealTime         bool          `json:"enable_real_time"`
	RealTimeBufferSize     int           `json:"real_time_buffer_size"`
	
	// Alerting settings
	EnableAlerting         bool          `json:"enable_alerting"`
	AlertThresholds        AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds defines thresholds for analytics alerts
type AlertThresholds struct {
	HighTrafficThreshold    int64   `json:"high_traffic_threshold"`
	LowEngagementThreshold  float64 `json:"low_engagement_threshold"`
	HighBounceRateThreshold float64 `json:"high_bounce_rate_threshold"`
	SlowResponseThreshold   float64 `json:"slow_response_threshold"`
}

// LoadAnalyticsConfig loads analytics configuration from environment variables
func LoadAnalyticsConfig() *AnalyticsConfig {
	config := &AnalyticsConfig{
		// Default values
		EnableTracking:            getEnvBool("ANALYTICS_ENABLE_TRACKING", true),
		EnableIPTracking:          getEnvBool("ANALYTICS_ENABLE_IP_TRACKING", true),
		EnableUserAgentTracking:   getEnvBool("ANALYTICS_ENABLE_USER_AGENT_TRACKING", true),
		EnableBehaviorTracking:    getEnvBool("ANALYTICS_ENABLE_BEHAVIOR_TRACKING", true),
		EnablePerformanceTracking: getEnvBool("ANALYTICS_ENABLE_PERFORMANCE_TRACKING", true),
		
		ViewDataRetentionDays:     getEnvInt("ANALYTICS_VIEW_DATA_RETENTION_DAYS", 365),
		BehaviorDataRetentionDays: getEnvInt("ANALYTICS_BEHAVIOR_DATA_RETENTION_DAYS", 90),
		ReportRetentionDays:       getEnvInt("ANALYTICS_REPORT_RETENTION_DAYS", 30),
		
		BatchSize:             getEnvInt("ANALYTICS_BATCH_SIZE", 1000),
		FlushInterval:         time.Duration(getEnvInt("ANALYTICS_FLUSH_INTERVAL_SECONDS", 30)) * time.Second,
		MaxConcurrentRequests: getEnvInt("ANALYTICS_MAX_CONCURRENT_REQUESTS", 10),
		
		MaxExportRows:        getEnvInt("ANALYTICS_MAX_EXPORT_ROWS", 100000),
		ExportTimeoutSeconds: getEnvInt("ANALYTICS_EXPORT_TIMEOUT_SECONDS", 300),
		
		AnonymizeIPs:          getEnvBool("ANALYTICS_ANONYMIZE_IPS", false),
		RespectDoNotTrack:     getEnvBool("ANALYTICS_RESPECT_DO_NOT_TRACK", true),
		CookieConsentRequired: getEnvBool("ANALYTICS_COOKIE_CONSENT_REQUIRED", false),
		
		EnableGeolocation:   getEnvBool("ANALYTICS_ENABLE_GEOLOCATION", false),
		GeolocationProvider: getEnvString("ANALYTICS_GEOLOCATION_PROVIDER", ""),
		GeolocationAPIKey:   getEnvString("ANALYTICS_GEOLOCATION_API_KEY", ""),
		
		EnableRealTime:     getEnvBool("ANALYTICS_ENABLE_REAL_TIME", false),
		RealTimeBufferSize: getEnvInt("ANALYTICS_REAL_TIME_BUFFER_SIZE", 1000),
		
		EnableAlerting: getEnvBool("ANALYTICS_ENABLE_ALERTING", false),
		AlertThresholds: AlertThresholds{
			HighTrafficThreshold:    int64(getEnvInt("ANALYTICS_HIGH_TRAFFIC_THRESHOLD", 10000)),
			LowEngagementThreshold:  getEnvFloat("ANALYTICS_LOW_ENGAGEMENT_THRESHOLD", 0.02),
			HighBounceRateThreshold: getEnvFloat("ANALYTICS_HIGH_BOUNCE_RATE_THRESHOLD", 0.8),
			SlowResponseThreshold:   getEnvFloat("ANALYTICS_SLOW_RESPONSE_THRESHOLD", 2000.0),
		},
	}
	
	return config
}

// Validate validates the analytics configuration
func (c *AnalyticsConfig) Validate() error {
	if c.BatchSize <= 0 {
		c.BatchSize = 1000
	}
	
	if c.FlushInterval <= 0 {
		c.FlushInterval = 30 * time.Second
	}
	
	if c.MaxConcurrentRequests <= 0 {
		c.MaxConcurrentRequests = 10
	}
	
	if c.ViewDataRetentionDays <= 0 {
		c.ViewDataRetentionDays = 365
	}
	
	if c.BehaviorDataRetentionDays <= 0 {
		c.BehaviorDataRetentionDays = 90
	}
	
	if c.ReportRetentionDays <= 0 {
		c.ReportRetentionDays = 30
	}
	
	if c.MaxExportRows <= 0 {
		c.MaxExportRows = 100000
	}
	
	if c.ExportTimeoutSeconds <= 0 {
		c.ExportTimeoutSeconds = 300
	}
	
	if c.RealTimeBufferSize <= 0 {
		c.RealTimeBufferSize = 1000
	}
	
	return nil
}

// IsTrackingEnabled returns true if tracking is enabled and privacy requirements are met
func (c *AnalyticsConfig) IsTrackingEnabled(doNotTrack bool, hasConsent bool) bool {
	if !c.EnableTracking {
		return false
	}
	
	if c.RespectDoNotTrack && doNotTrack {
		return false
	}
	
	if c.CookieConsentRequired && !hasConsent {
		return false
	}
	
	return true
}

// ShouldAnonymizeIP returns true if IP addresses should be anonymized
func (c *AnalyticsConfig) ShouldAnonymizeIP() bool {
	return c.AnonymizeIPs
}

// GetRetentionDays returns the retention period for different data types
func (c *AnalyticsConfig) GetRetentionDays(dataType string) int {
	switch dataType {
	case "views":
		return c.ViewDataRetentionDays
	case "behavior":
		return c.BehaviorDataRetentionDays
	case "reports":
		return c.ReportRetentionDays
	default:
		return c.ViewDataRetentionDays
	}
}

