package config

import (
	"time"
)

// MonitoringConfig holds monitoring and alerting configuration
type MonitoringConfig struct {
	// Prometheus settings
	EnablePrometheus     bool   `json:"enable_prometheus"`
	PrometheusPort       int    `json:"prometheus_port"`
	PrometheusPath       string `json:"prometheus_path"`
	
	// Health check settings
	EnableHealthChecks   bool          `json:"enable_health_checks"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	HealthCheckTimeout   time.Duration `json:"health_check_timeout"`
	HealthCheckPort      int           `json:"health_check_port"`
	HealthCheckPath      string        `json:"health_check_path"`
	
	// Resource monitoring settings
	EnableResourceMonitoring bool          `json:"enable_resource_monitoring"`
	ResourceCheckInterval    time.Duration `json:"resource_check_interval"`
	CPUThreshold            float64       `json:"cpu_threshold"`
	MemoryThreshold         float64       `json:"memory_threshold"`
	DiskThreshold           float64       `json:"disk_threshold"`
	
	// Database monitoring
	EnableDBMonitoring      bool          `json:"enable_db_monitoring"`
	DBConnectionThreshold   int           `json:"db_connection_threshold"`
	SlowQueryThreshold      time.Duration `json:"slow_query_threshold"`
	
	// Cache monitoring
	EnableCacheMonitoring   bool    `json:"enable_cache_monitoring"`
	CacheHitRateThreshold   float64 `json:"cache_hit_rate_threshold"`
	
	// Publishing rate monitoring
	EnablePublishingMonitoring bool    `json:"enable_publishing_monitoring"`
	PublishingRateThreshold    float64 `json:"publishing_rate_threshold"` // articles per minute
	
	// Alerting settings
	EnableAlerting          bool          `json:"enable_alerting"`
	AlertCheckInterval      time.Duration `json:"alert_check_interval"`
	AlertCooldownPeriod     time.Duration `json:"alert_cooldown_period"`
	
	// Alert channels
	EmailAlerting           bool   `json:"email_alerting"`
	SlackAlerting           bool   `json:"slack_alerting"`
	WebhookAlerting         bool   `json:"webhook_alerting"`
	SlackWebhookURL         string `json:"slack_webhook_url"`
	AlertWebhookURL         string `json:"alert_webhook_url"`
	AlertEmailRecipients    string `json:"alert_email_recipients"`
	
	// Retention settings
	MetricsRetentionDays    int `json:"metrics_retention_days"`
	AlertHistoryRetentionDays int `json:"alert_history_retention_days"`
}

// LoadMonitoringConfig loads monitoring configuration from environment variables
func LoadMonitoringConfig() *MonitoringConfig {
	config := &MonitoringConfig{
		// Prometheus defaults
		EnablePrometheus:     getEnvBool("MONITORING_ENABLE_PROMETHEUS", true),
		PrometheusPort:       getEnvInt("MONITORING_PROMETHEUS_PORT", 9090),
		PrometheusPath:       getEnvString("MONITORING_PROMETHEUS_PATH", "/metrics"),
		
		// Health check defaults
		EnableHealthChecks:   getEnvBool("MONITORING_ENABLE_HEALTH_CHECKS", true),
		HealthCheckInterval:  time.Duration(getEnvInt("MONITORING_HEALTH_CHECK_INTERVAL_SECONDS", 30)) * time.Second,
		HealthCheckTimeout:   time.Duration(getEnvInt("MONITORING_HEALTH_CHECK_TIMEOUT_SECONDS", 5)) * time.Second,
		HealthCheckPort:      getEnvInt("MONITORING_HEALTH_CHECK_PORT", 8081),
		HealthCheckPath:      getEnvString("MONITORING_HEALTH_CHECK_PATH", "/health"),
		
		// Resource monitoring defaults
		EnableResourceMonitoring: getEnvBool("MONITORING_ENABLE_RESOURCE_MONITORING", true),
		ResourceCheckInterval:    time.Duration(getEnvInt("MONITORING_RESOURCE_CHECK_INTERVAL_SECONDS", 60)) * time.Second,
		CPUThreshold:            getEnvFloat("MONITORING_CPU_THRESHOLD", 80.0),
		MemoryThreshold:         getEnvFloat("MONITORING_MEMORY_THRESHOLD", 85.0),
		DiskThreshold:           getEnvFloat("MONITORING_DISK_THRESHOLD", 90.0),
		
		// Database monitoring defaults
		EnableDBMonitoring:      getEnvBool("MONITORING_ENABLE_DB_MONITORING", true),
		DBConnectionThreshold:   getEnvInt("MONITORING_DB_CONNECTION_THRESHOLD", 140), // 140 out of 150 max
		SlowQueryThreshold:      time.Duration(getEnvInt("MONITORING_SLOW_QUERY_THRESHOLD_MS", 1000)) * time.Millisecond,
		
		// Cache monitoring defaults
		EnableCacheMonitoring:   getEnvBool("MONITORING_ENABLE_CACHE_MONITORING", true),
		CacheHitRateThreshold:   getEnvFloat("MONITORING_CACHE_HIT_RATE_THRESHOLD", 0.8), // 80%
		
		// Publishing rate monitoring defaults
		EnablePublishingMonitoring: getEnvBool("MONITORING_ENABLE_PUBLISHING_MONITORING", true),
		PublishingRateThreshold:    getEnvFloat("MONITORING_PUBLISHING_RATE_THRESHOLD", 35.0), // 35 articles/minute
		
		// Alerting defaults
		EnableAlerting:          getEnvBool("MONITORING_ENABLE_ALERTING", true),
		AlertCheckInterval:      time.Duration(getEnvInt("MONITORING_ALERT_CHECK_INTERVAL_SECONDS", 60)) * time.Second,
		AlertCooldownPeriod:     time.Duration(getEnvInt("MONITORING_ALERT_COOLDOWN_MINUTES", 15)) * time.Minute,
		
		// Alert channels defaults
		EmailAlerting:           getEnvBool("MONITORING_EMAIL_ALERTING", false),
		SlackAlerting:           getEnvBool("MONITORING_SLACK_ALERTING", false),
		WebhookAlerting:         getEnvBool("MONITORING_WEBHOOK_ALERTING", false),
		SlackWebhookURL:         getEnvString("MONITORING_SLACK_WEBHOOK_URL", ""),
		AlertWebhookURL:         getEnvString("MONITORING_ALERT_WEBHOOK_URL", ""),
		AlertEmailRecipients:    getEnvString("MONITORING_ALERT_EMAIL_RECIPIENTS", ""),
		
		// Retention defaults
		MetricsRetentionDays:      getEnvInt("MONITORING_METRICS_RETENTION_DAYS", 30),
		AlertHistoryRetentionDays: getEnvInt("MONITORING_ALERT_HISTORY_RETENTION_DAYS", 90),
	}
	
	return config
}

// Validate validates the monitoring configuration
func (c *MonitoringConfig) Validate() error {
	if c.PrometheusPort <= 0 || c.PrometheusPort > 65535 {
		c.PrometheusPort = 9090
	}
	
	if c.HealthCheckPort <= 0 || c.HealthCheckPort > 65535 {
		c.HealthCheckPort = 8081
	}
	
	if c.HealthCheckInterval <= 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	
	if c.HealthCheckTimeout <= 0 {
		c.HealthCheckTimeout = 5 * time.Second
	}
	
	if c.ResourceCheckInterval <= 0 {
		c.ResourceCheckInterval = 60 * time.Second
	}
	
	if c.CPUThreshold <= 0 || c.CPUThreshold > 100 {
		c.CPUThreshold = 80.0
	}
	
	if c.MemoryThreshold <= 0 || c.MemoryThreshold > 100 {
		c.MemoryThreshold = 85.0
	}
	
	if c.DiskThreshold <= 0 || c.DiskThreshold > 100 {
		c.DiskThreshold = 90.0
	}
	
	if c.DBConnectionThreshold <= 0 {
		c.DBConnectionThreshold = 140
	}
	
	if c.SlowQueryThreshold <= 0 {
		c.SlowQueryThreshold = 1000 * time.Millisecond
	}
	
	if c.CacheHitRateThreshold <= 0 || c.CacheHitRateThreshold > 1 {
		c.CacheHitRateThreshold = 0.8
	}
	
	if c.PublishingRateThreshold <= 0 {
		c.PublishingRateThreshold = 35.0
	}
	
	if c.AlertCheckInterval <= 0 {
		c.AlertCheckInterval = 60 * time.Second
	}
	
	if c.AlertCooldownPeriod <= 0 {
		c.AlertCooldownPeriod = 15 * time.Minute
	}
	
	if c.MetricsRetentionDays <= 0 {
		c.MetricsRetentionDays = 30
	}
	
	if c.AlertHistoryRetentionDays <= 0 {
		c.AlertHistoryRetentionDays = 90
	}
	
	return nil
}

// IsAlertingConfigured returns true if at least one alerting channel is configured
func (c *MonitoringConfig) IsAlertingConfigured() bool {
	if !c.EnableAlerting {
		return false
	}
	
	return (c.EmailAlerting && c.AlertEmailRecipients != "") ||
		   (c.SlackAlerting && c.SlackWebhookURL != "") ||
		   (c.WebhookAlerting && c.AlertWebhookURL != "")
}

