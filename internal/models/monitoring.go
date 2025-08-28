package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	ID          uint64                 `json:"id" db:"id"`
	Component   string                 `json:"component" db:"component"`
	Status      HealthStatus           `json:"status" db:"status"`
	Message     string                 `json:"message" db:"message"`
	ResponseTime time.Duration         `json:"response_time" db:"response_time"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CheckedAt   time.Time              `json:"checked_at" db:"checked_at"`
}

// SystemMetrics represents system resource metrics
type SystemMetrics struct {
	ID                uint64    `json:"id" db:"id"`
	CPUUsage          float64   `json:"cpu_usage" db:"cpu_usage"`
	MemoryUsage       float64   `json:"memory_usage" db:"memory_usage"`
	MemoryTotal       uint64    `json:"memory_total" db:"memory_total"`
	MemoryUsed        uint64    `json:"memory_used" db:"memory_used"`
	DiskUsage         float64   `json:"disk_usage" db:"disk_usage"`
	DiskTotal         uint64    `json:"disk_total" db:"disk_total"`
	DiskUsed          uint64    `json:"disk_used" db:"disk_used"`
	NetworkBytesIn    uint64    `json:"network_bytes_in" db:"network_bytes_in"`
	NetworkBytesOut   uint64    `json:"network_bytes_out" db:"network_bytes_out"`
	LoadAverage1      float64   `json:"load_average_1" db:"load_average_1"`
	LoadAverage5      float64   `json:"load_average_5" db:"load_average_5"`
	LoadAverage15     float64   `json:"load_average_15" db:"load_average_15"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// DatabaseMetrics represents database performance metrics
type DatabaseMetrics struct {
	ID                    uint64    `json:"id" db:"id"`
	ActiveConnections     int       `json:"active_connections" db:"active_connections"`
	IdleConnections       int       `json:"idle_connections" db:"idle_connections"`
	MaxConnections        int       `json:"max_connections" db:"max_connections"`
	SlowQueries           int64     `json:"slow_queries" db:"slow_queries"`
	AverageQueryTime      float64   `json:"average_query_time" db:"average_query_time"`
	QueriesPerSecond      float64   `json:"queries_per_second" db:"queries_per_second"`
	CacheHitRatio         float64   `json:"cache_hit_ratio" db:"cache_hit_ratio"`
	DeadlockCount         int64     `json:"deadlock_count" db:"deadlock_count"`
	TempFilesCreated      int64     `json:"temp_files_created" db:"temp_files_created"`
	CheckpointWriteTime   float64   `json:"checkpoint_write_time" db:"checkpoint_write_time"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// CacheMetrics represents cache performance metrics
type CacheMetrics struct {
	ID              uint64    `json:"id" db:"id"`
	HitCount        int64     `json:"hit_count" db:"hit_count"`
	MissCount       int64     `json:"miss_count" db:"miss_count"`
	HitRate         float64   `json:"hit_rate" db:"hit_rate"`
	KeyCount        int64     `json:"key_count" db:"key_count"`
	MemoryUsage     uint64    `json:"memory_usage" db:"memory_usage"`
	MemoryTotal     uint64    `json:"memory_total" db:"memory_total"`
	EvictedKeys     int64     `json:"evicted_keys" db:"evicted_keys"`
	ExpiredKeys     int64     `json:"expired_keys" db:"expired_keys"`
	OperationsPerSec float64  `json:"operations_per_sec" db:"operations_per_sec"`
	AverageLatency  float64   `json:"average_latency" db:"average_latency"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// PublishingMetrics represents content publishing metrics
type PublishingMetrics struct {
	ID                    uint64    `json:"id" db:"id"`
	ArticlesPublished     int64     `json:"articles_published" db:"articles_published"`
	PublishingRate        float64   `json:"publishing_rate" db:"publishing_rate"` // articles per minute
	AveragePublishTime    float64   `json:"average_publish_time" db:"average_publish_time"`
	FailedPublications    int64     `json:"failed_publications" db:"failed_publications"`
	QueuedArticles        int64     `json:"queued_articles" db:"queued_articles"`
	ProcessingArticles    int64     `json:"processing_articles" db:"processing_articles"`
	StaticPagesGenerated  int64     `json:"static_pages_generated" db:"static_pages_generated"`
	CacheInvalidations    int64     `json:"cache_invalidations" db:"cache_invalidations"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// Alert represents a system alert
type Alert struct {
	ID          uint64                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Severity    AlertSeverity          `json:"severity" db:"severity"`
	Status      AlertStatus            `json:"status" db:"status"`
	Component   string                 `json:"component" db:"component"`
	Metric      string                 `json:"metric" db:"metric"`
	Threshold   float64                `json:"threshold" db:"threshold"`
	CurrentValue float64               `json:"current_value" db:"current_value"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	TriggeredAt time.Time              `json:"triggered_at" db:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at" db:"resolved_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// AlertRule represents an alert rule configuration
type AlertRule struct {
	ID          uint64                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Component   string                 `json:"component" db:"component"`
	Metric      string                 `json:"metric" db:"metric"`
	Operator    string                 `json:"operator" db:"operator"` // >, <, >=, <=, ==, !=
	Threshold   float64                `json:"threshold" db:"threshold"`
	Severity    AlertSeverity          `json:"severity" db:"severity"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Cooldown    time.Duration          `json:"cooldown" db:"cooldown"`
	Conditions  map[string]interface{} `json:"conditions" db:"conditions"`
	Actions     AlertActions           `json:"actions" db:"actions"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// AlertAction represents an action to take when an alert is triggered
type AlertAction struct {
	Type       string                 `json:"type"`       // email, slack, webhook
	Target     string                 `json:"target"`     // email address, webhook URL, etc.
	Template   string                 `json:"template"`   // message template
	Metadata   map[string]interface{} `json:"metadata"`   // additional action-specific data
}

// AlertActions is a custom type for a slice of AlertAction to implement database interfaces
type AlertActions []AlertAction

// Scan implements the sql.Scanner interface for AlertActions
func (a *AlertActions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AlertActions", value)
	}
}

// Value implements the driver.Valuer interface for AlertActions
func (a AlertActions) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// MonitoringDashboard represents monitoring dashboard data
type MonitoringDashboard struct {
	SystemHealth        HealthStatus      `json:"system_health"`
	SystemMetrics       SystemMetrics     `json:"system_metrics"`
	DatabaseMetrics     DatabaseMetrics   `json:"database_metrics"`
	CacheMetrics        CacheMetrics      `json:"cache_metrics"`
	PublishingMetrics   PublishingMetrics `json:"publishing_metrics"`
	ActiveAlerts        []Alert           `json:"active_alerts"`
	RecentHealthChecks  []HealthCheck     `json:"recent_health_checks"`
	PerformanceTrends   PerformanceTrends `json:"performance_trends"`
	LastUpdated         time.Time         `json:"last_updated"`
}

// PerformanceTrends represents performance trends over time
type PerformanceTrends struct {
	CPUTrend            []TrendPoint `json:"cpu_trend"`
	MemoryTrend         []TrendPoint `json:"memory_trend"`
	DatabaseTrend       []TrendPoint `json:"database_trend"`
	CacheTrend          []TrendPoint `json:"cache_trend"`
	PublishingTrend     []TrendPoint `json:"publishing_trend"`
}

// TrendPoint represents a single point in a performance trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricThresholds represents configurable metric thresholds
type MetricThresholds struct {
	CPUWarning              float64 `json:"cpu_warning"`
	CPUCritical             float64 `json:"cpu_critical"`
	MemoryWarning           float64 `json:"memory_warning"`
	MemoryCritical          float64 `json:"memory_critical"`
	DiskWarning             float64 `json:"disk_warning"`
	DiskCritical            float64 `json:"disk_critical"`
	DBConnectionsWarning    int     `json:"db_connections_warning"`
	DBConnectionsCritical   int     `json:"db_connections_critical"`
	CacheHitRateWarning     float64 `json:"cache_hit_rate_warning"`
	CacheHitRateCritical    float64 `json:"cache_hit_rate_critical"`
	PublishingRateWarning   float64 `json:"publishing_rate_warning"`
	PublishingRateCritical  float64 `json:"publishing_rate_critical"`
	ResponseTimeWarning     float64 `json:"response_time_warning"`
	ResponseTimeCritical    float64 `json:"response_time_critical"`
}

// DefaultMetricThresholds returns default metric thresholds
func DefaultMetricThresholds() MetricThresholds {
	return MetricThresholds{
		CPUWarning:              70.0,
		CPUCritical:             85.0,
		MemoryWarning:           80.0,
		MemoryCritical:          90.0,
		DiskWarning:             85.0,
		DiskCritical:            95.0,
		DBConnectionsWarning:    120,
		DBConnectionsCritical:   140,
		CacheHitRateWarning:     0.7,
		CacheHitRateCritical:    0.5,
		PublishingRateWarning:   25.0,
		PublishingRateCritical:  15.0,
		ResponseTimeWarning:     1000.0, // ms
		ResponseTimeCritical:    2000.0, // ms
	}
}

// JobQueue interface to avoid import cycles
type JobQueue interface {
	Enqueue(job *Job) error
	Dequeue() (*Job, error)
	Start() error
	Stop() error
	GetStats() JobStats
}

// Job represents a background job
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    JobPriority            `json:"priority"`
	Payload     map[string]interface{} `json:"payload"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	MaxAttempts int                    `json:"max_attempts"`
	Attempts    int                    `json:"attempts"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// JobPriority represents job priority levels
type JobPriority string

const (
	JobPriorityLow    JobPriority = "low"
	JobPriorityMedium JobPriority = "medium"
	JobPriorityHigh   JobPriority = "high"
)

// JobStats represents job queue statistics
type JobStats struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// ValidateAlert validates alert data
func ValidateAlert(alert *Alert) error {
	var errors []string

	if alert.Name == "" {
		errors = append(errors, "name is required")
	}

	if alert.Component == "" {
		errors = append(errors, "component is required")
	}

	if alert.Metric == "" {
		errors = append(errors, "metric is required")
	}

	validSeverities := map[AlertSeverity]bool{
		AlertSeverityInfo:     true,
		AlertSeverityWarning:  true,
		AlertSeverityCritical: true,
	}
	if !validSeverities[alert.Severity] {
		errors = append(errors, "invalid severity")
	}

	validStatuses := map[AlertStatus]bool{
		AlertStatusActive:     true,
		AlertStatusResolved:   true,
		AlertStatusSuppressed: true,
	}
	if !validStatuses[alert.Status] {
		errors = append(errors, "invalid status")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Alert validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// MonitoringConfig represents monitoring configuration (avoiding import cycle)
type MonitoringConfig struct {
	EnablePrometheus     bool          `json:"enable_prometheus"`
	PrometheusPort       int           `json:"prometheus_port"`
	EnableHealthChecks   bool          `json:"enable_health_checks"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	EnableResourceMonitoring bool      `json:"enable_resource_monitoring"`
	ResourceCheckInterval    time.Duration `json:"resource_check_interval"`
	CPUThreshold            float64   `json:"cpu_threshold"`
	MemoryThreshold         float64   `json:"memory_threshold"`
	DiskThreshold           float64   `json:"disk_threshold"`
	EnableAlerting          bool      `json:"enable_alerting"`
	AlertCheckInterval      time.Duration `json:"alert_check_interval"`
	AlertCooldownPeriod     time.Duration `json:"alert_cooldown_period"`
	EmailAlerting           bool      `json:"email_alerting"`
	SlackAlerting           bool      `json:"slack_alerting"`
	WebhookAlerting         bool      `json:"webhook_alerting"`
	SlackWebhookURL         string    `json:"slack_webhook_url"`
	AlertWebhookURL         string    `json:"alert_webhook_url"`
	AlertEmailRecipients    string    `json:"alert_email_recipients"`
}

// ValidateAlertRule validates alert rule data
func ValidateAlertRule(rule *AlertRule) error {
	var errors []string

	if rule.Name == "" {
		errors = append(errors, "name is required")
	}

	if rule.Component == "" {
		errors = append(errors, "component is required")
	}

	if rule.Metric == "" {
		errors = append(errors, "metric is required")
	}

	validOperators := map[string]bool{
		">":  true,
		"<":  true,
		">=": true,
		"<=": true,
		"==": true,
		"!=": true,
	}
	if !validOperators[rule.Operator] {
		errors = append(errors, "invalid operator")
	}

	validSeverities := map[AlertSeverity]bool{
		AlertSeverityInfo:     true,
		AlertSeverityWarning:  true,
		AlertSeverityCritical: true,
	}
	if !validSeverities[rule.Severity] {
		errors = append(errors, "invalid severity")
	}

	if rule.Cooldown < 0 {
		errors = append(errors, "cooldown must be non-negative")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Alert rule validation failed",
			Fields:  errors,
		}
	}

	return nil
}