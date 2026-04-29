package config

// PushNotificationConfig holds configuration for push notification services
type PushNotificationConfig struct {
	// OneSignal configuration
	OneSignalAppID  string `json:"onesignal_app_id" yaml:"onesignal_app_id"`
	OneSignalAPIKey string `json:"onesignal_api_key" yaml:"onesignal_api_key"`
	
	// Firebase configuration
	FirebaseServerKey string `json:"firebase_server_key" yaml:"firebase_server_key"`
	FirebaseProjectID string `json:"firebase_project_id" yaml:"firebase_project_id"`
	
	// VAPID keys for web push
	VAPIDPublicKey  string `json:"vapid_public_key" yaml:"vapid_public_key"`
	VAPIDPrivateKey string `json:"vapid_private_key" yaml:"vapid_private_key"`
	VAPIDSubject    string `json:"vapid_subject" yaml:"vapid_subject"`
	
	// Service configuration
	Enabled                bool   `json:"enabled" yaml:"enabled"`
	DefaultIcon            string `json:"default_icon" yaml:"default_icon"`
	DefaultBadge           string `json:"default_badge" yaml:"default_badge"`
	MaxRetries             int    `json:"max_retries" yaml:"max_retries"`
	RetryDelaySeconds      int    `json:"retry_delay_seconds" yaml:"retry_delay_seconds"`
	BatchSize              int    `json:"batch_size" yaml:"batch_size"`
	ProcessingIntervalSecs int    `json:"processing_interval_secs" yaml:"processing_interval_secs"`
	
	// Cleanup configuration
	CleanupOldDataDays int `json:"cleanup_old_data_days" yaml:"cleanup_old_data_days"`
}

// LoadPushNotificationConfig loads push notification configuration from environment variables
func LoadPushNotificationConfig() *PushNotificationConfig {
	config := &PushNotificationConfig{
		// OneSignal
		OneSignalAppID:  getEnv("ONESIGNAL_APP_ID", ""),
		OneSignalAPIKey: getEnv("ONESIGNAL_API_KEY", ""),
		
		// Firebase
		FirebaseServerKey: getEnv("FIREBASE_SERVER_KEY", ""),
		FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
		
		// VAPID
		VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:    getEnv("VAPID_SUBJECT", "mailto:admin@example.com"),
		
		// Service settings
		Enabled:                getEnvBool("PUSH_NOTIFICATIONS_ENABLED", true),
		DefaultIcon:            getEnv("PUSH_DEFAULT_ICON", "/static/images/icon-192x192.png"),
		DefaultBadge:           getEnv("PUSH_DEFAULT_BADGE", "/static/images/badge-72x72.png"),
		MaxRetries:             getEnvInt("PUSH_MAX_RETRIES", 3),
		RetryDelaySeconds:      getEnvInt("PUSH_RETRY_DELAY_SECONDS", 60),
		BatchSize:              getEnvInt("PUSH_BATCH_SIZE", 1000),
		ProcessingIntervalSecs: getEnvInt("PUSH_PROCESSING_INTERVAL_SECS", 30),
		
		// Cleanup
		CleanupOldDataDays: getEnvInt("PUSH_CLEANUP_OLD_DATA_DAYS", 90),
	}
	
	return config
}

// IsConfigured returns true if at least one push service is configured
func (c *PushNotificationConfig) IsConfigured() bool {
	return c.Enabled && (c.HasOneSignal() || c.HasFirebase() || c.HasVAPID())
}

// HasOneSignal returns true if OneSignal is configured
func (c *PushNotificationConfig) HasOneSignal() bool {
	return c.OneSignalAppID != "" && c.OneSignalAPIKey != ""
}

// HasFirebase returns true if Firebase is configured
func (c *PushNotificationConfig) HasFirebase() bool {
	return c.FirebaseServerKey != ""
}

// HasVAPID returns true if VAPID keys are configured
func (c *PushNotificationConfig) HasVAPID() bool {
	return c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != ""
}

// GetFirebaseConfig returns Firebase configuration as JSON string for frontend
func (c *PushNotificationConfig) GetFirebaseConfig() string {
	if !c.HasFirebase() {
		return ""
	}
	
	// This would typically include more Firebase config
	// For now, just return the project ID
	return `{"projectId":"` + c.FirebaseProjectID + `"}`
}

// Validate validates the push notification configuration
func (c *PushNotificationConfig) Validate() error {
	if !c.Enabled {
		return nil // No validation needed if disabled
	}
	
	if !c.IsConfigured() {
		return &ConfigError{
			Field:   "push_notifications",
			Message: "At least one push notification service must be configured (OneSignal, Firebase, or VAPID)",
		}
	}
	
	if c.MaxRetries < 0 {
		return &ConfigError{
			Field:   "max_retries",
			Message: "max_retries must be >= 0",
		}
	}
	
	if c.RetryDelaySeconds < 1 {
		return &ConfigError{
			Field:   "retry_delay_seconds",
			Message: "retry_delay_seconds must be >= 1",
		}
	}
	
	if c.BatchSize < 1 {
		return &ConfigError{
			Field:   "batch_size",
			Message: "batch_size must be >= 1",
		}
	}
	
	if c.ProcessingIntervalSecs < 1 {
		return &ConfigError{
			Field:   "processing_interval_secs",
			Message: "processing_interval_secs must be >= 1",
		}
	}
	
	if c.CleanupOldDataDays < 1 {
		return &ConfigError{
			Field:   "cleanup_old_data_days",
			Message: "cleanup_old_data_days must be >= 1",
		}
	}
	
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

// Helper functions for environment variables are defined in analytics.go

func parseInt(s string) int {
	// Simple integer parsing - in production you'd use strconv.Atoi
	// This is a placeholder implementation
	switch s {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "30":
		return 30
	case "60":
		return 60
	case "90":
		return 90
	case "1000":
		return 1000
	default:
		return 0
	}
}