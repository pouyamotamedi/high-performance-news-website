package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/database"
)

// MonitoringPersistenceService handles persistence of monitoring data
type MonitoringPersistenceService struct {
	db               *sql.DB
	partitionCache   map[string]time.Time // tracks which partitions we've verified exist
	partitionCacheMu sync.RWMutex
}

// NewMonitoringPersistenceService creates a new MonitoringPersistenceService
func NewMonitoringPersistenceService(db *sql.DB, dbWrapper *database.DB) *MonitoringPersistenceService {
	return &MonitoringPersistenceService{
		db:             db,
		partitionCache: make(map[string]time.Time),
	}
}

// ensureMetricsPartition ensures a partition exists for the given table and date.
// It uses a cache to avoid checking on every write (checks once per day per table).
func (mps *MonitoringPersistenceService) ensureMetricsPartition(ctx context.Context, tableName string, t time.Time) error {
	// Create cache key based on table and date
	dateStr := t.Format("2006_01_02")
	cacheKey := tableName + "_" + dateStr
	
	// Check cache first (read lock)
	mps.partitionCacheMu.RLock()
	if _, exists := mps.partitionCache[cacheKey]; exists {
		mps.partitionCacheMu.RUnlock()
		return nil // Already verified this partition exists
	}
	mps.partitionCacheMu.RUnlock()
	
	// Need to check/create partition (write lock)
	mps.partitionCacheMu.Lock()
	defer mps.partitionCacheMu.Unlock()
	
	// Double-check after acquiring write lock
	if _, exists := mps.partitionCache[cacheKey]; exists {
		return nil
	}
	
	partitionName := tableName + "_" + dateStr
	startDate := t.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)
	
	// Check if partition exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pg_class WHERE relname = $1)`
	if err := mps.db.QueryRowContext(ctx, checkQuery, partitionName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check partition existence: %v", err)
	}
	
	if !exists {
		// Create the partition
		createQuery := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')`,
			partitionName,
			tableName,
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"),
		)
		if _, err := mps.db.ExecContext(ctx, createQuery); err != nil {
			// Ignore "already exists" errors (race condition with another process)
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create partition %s: %v", partitionName, err)
			}
		}
		log.Printf("Created metrics partition: %s", partitionName)
	}
	
	// Cache that this partition exists
	mps.partitionCache[cacheKey] = time.Now()
	
	// Also pre-create partitions for the next 7 days to avoid issues
	go mps.preCreateFuturePartitions(tableName, t)
	
	return nil
}

// preCreateFuturePartitions creates partitions for the next 7 days in the background
func (mps *MonitoringPersistenceService) preCreateFuturePartitions(tableName string, baseTime time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	for i := 1; i <= 7; i++ {
		futureDate := baseTime.Add(time.Duration(i) * 24 * time.Hour)
		dateStr := futureDate.Format("2006_01_02")
		partitionName := tableName + "_" + dateStr
		startDate := futureDate.Truncate(24 * time.Hour)
		endDate := startDate.Add(24 * time.Hour)
		
		// Check if already in cache
		cacheKey := tableName + "_" + dateStr
		mps.partitionCacheMu.RLock()
		_, exists := mps.partitionCache[cacheKey]
		mps.partitionCacheMu.RUnlock()
		if exists {
			continue
		}
		
		// Create partition
		createQuery := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')`,
			partitionName,
			tableName,
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"),
		)
		if _, err := mps.db.ExecContext(ctx, createQuery); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("Warning: failed to pre-create partition %s: %v", partitionName, err)
			}
		}
		
		// Update cache
		mps.partitionCacheMu.Lock()
		mps.partitionCache[cacheKey] = time.Now()
		mps.partitionCacheMu.Unlock()
	}
}

// SaveHealthCheck saves a health check result to the database
func (mps *MonitoringPersistenceService) SaveHealthCheck(ctx context.Context, healthCheck *models.HealthCheck) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	metadataJSON, err := json.Marshal(healthCheck.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO health_checks (component, status, message, response_time_ms, metadata, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = mps.db.ExecContext(ctx, query,
		healthCheck.Component,
		string(healthCheck.Status),
		healthCheck.Message,
		healthCheck.ResponseTime.Milliseconds(),
		metadataJSON,
		healthCheck.CheckedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save health check: %v", err)
	}

	return nil
}

// SaveSystemMetrics saves system metrics to the database
func (mps *MonitoringPersistenceService) SaveSystemMetrics(ctx context.Context, metrics *models.SystemMetrics) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	now := time.Now()
	
	// Ensure partition exists before inserting
	if err := mps.ensureMetricsPartition(ctx, "system_metrics", now); err != nil {
		log.Printf("Warning: failed to ensure system_metrics partition: %v", err)
	}

	query := `
		INSERT INTO system_metrics (
			cpu_usage, memory_usage, memory_total, memory_used,
			disk_usage, disk_total, disk_used, network_bytes_in, network_bytes_out,
			load_average_1, load_average_5, load_average_15, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := mps.db.ExecContext(ctx, query,
		metrics.CPUUsage,
		metrics.MemoryUsage,
		metrics.MemoryTotal,
		metrics.MemoryUsed,
		metrics.DiskUsage,
		metrics.DiskTotal,
		metrics.DiskUsed,
		metrics.NetworkBytesIn,
		metrics.NetworkBytesOut,
		metrics.LoadAverage1,
		metrics.LoadAverage5,
		metrics.LoadAverage15,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to save system metrics: %v", err)
	}

	return nil
}

// SaveDatabaseMetrics saves database metrics to the database
func (mps *MonitoringPersistenceService) SaveDatabaseMetrics(ctx context.Context, metrics *models.DatabaseMetrics) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	now := time.Now()
	
	// Ensure partition exists before inserting
	if err := mps.ensureMetricsPartition(ctx, "database_metrics", now); err != nil {
		log.Printf("Warning: failed to ensure database_metrics partition: %v", err)
	}

	query := `
		INSERT INTO database_metrics (
			active_connections, idle_connections, max_connections, slow_queries,
			average_query_time, queries_per_second, cache_hit_ratio, deadlock_count,
			temp_files_created, checkpoint_write_time, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := mps.db.ExecContext(ctx, query,
		metrics.ActiveConnections,
		metrics.IdleConnections,
		metrics.MaxConnections,
		metrics.SlowQueries,
		metrics.AverageQueryTime,
		metrics.QueriesPerSecond,
		metrics.CacheHitRatio,
		metrics.DeadlockCount,
		metrics.TempFilesCreated,
		metrics.CheckpointWriteTime,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to save database metrics: %v", err)
	}

	return nil
}

// SaveCacheMetrics saves cache metrics to the database
func (mps *MonitoringPersistenceService) SaveCacheMetrics(ctx context.Context, metrics *models.CacheMetrics) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	now := time.Now()
	
	// Ensure partition exists before inserting
	if err := mps.ensureMetricsPartition(ctx, "cache_metrics", now); err != nil {
		log.Printf("Warning: failed to ensure cache_metrics partition: %v", err)
	}

	query := `
		INSERT INTO cache_metrics (
			hit_count, miss_count, hit_rate, key_count, memory_usage, memory_total,
			evicted_keys, expired_keys, operations_per_sec, average_latency, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := mps.db.ExecContext(ctx, query,
		metrics.HitCount,
		metrics.MissCount,
		metrics.HitRate,
		metrics.KeyCount,
		metrics.MemoryUsage,
		metrics.MemoryTotal,
		metrics.EvictedKeys,
		metrics.ExpiredKeys,
		metrics.OperationsPerSec,
		metrics.AverageLatency,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to save cache metrics: %v", err)
	}

	return nil
}

// SavePublishingMetrics saves publishing metrics to the database
func (mps *MonitoringPersistenceService) SavePublishingMetrics(ctx context.Context, metrics *models.PublishingMetrics) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	now := time.Now()
	
	// Ensure partition exists before inserting
	if err := mps.ensureMetricsPartition(ctx, "publishing_metrics", now); err != nil {
		log.Printf("Warning: failed to ensure publishing_metrics partition: %v", err)
	}

	query := `
		INSERT INTO publishing_metrics (
			articles_published, publishing_rate, average_publish_time, failed_publications,
			queued_articles, processing_articles, static_pages_generated, cache_invalidations, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := mps.db.ExecContext(ctx, query,
		metrics.ArticlesPublished,
		metrics.PublishingRate,
		metrics.AveragePublishTime,
		metrics.FailedPublications,
		metrics.QueuedArticles,
		metrics.ProcessingArticles,
		metrics.StaticPagesGenerated,
		metrics.CacheInvalidations,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to save publishing metrics: %v", err)
	}

	return nil
}

// SaveAlert saves an alert to the database
func (mps *MonitoringPersistenceService) SaveAlert(ctx context.Context, alert *models.Alert) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	metadataJSON, err := json.Marshal(alert.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO alerts (
			name, description, severity, status, component, metric,
			threshold, current_value, metadata, triggered_at, resolved_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	err = mps.db.QueryRowContext(ctx, query,
		alert.Name,
		alert.Description,
		string(alert.Severity),
		string(alert.Status),
		alert.Component,
		alert.Metric,
		alert.Threshold,
		alert.CurrentValue,
		metadataJSON,
		alert.TriggeredAt,
		alert.ResolvedAt,
		alert.CreatedAt,
		alert.UpdatedAt,
	).Scan(&alert.ID)

	if err != nil {
		return fmt.Errorf("failed to save alert: %v", err)
	}

	return nil
}

// UpdateAlert updates an existing alert in the database
func (mps *MonitoringPersistenceService) UpdateAlert(ctx context.Context, alert *models.Alert) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	metadataJSON, err := json.Marshal(alert.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		UPDATE alerts SET
			status = $2, current_value = $3, metadata = $4, resolved_at = $5, updated_at = $6
		WHERE id = $1`

	_, err = mps.db.ExecContext(ctx, query,
		alert.ID,
		string(alert.Status),
		alert.CurrentValue,
		metadataJSON,
		alert.ResolvedAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update alert: %v", err)
	}

	return nil
}

// GetActiveAlerts retrieves active alerts from the database
func (mps *MonitoringPersistenceService) GetActiveAlerts(ctx context.Context) ([]models.Alert, error) {
	if mps.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT id, name, description, severity, status, component, metric,
			   threshold, current_value, metadata, triggered_at, resolved_at, created_at, updated_at
		FROM alerts
		WHERE status = 'active'
		ORDER BY triggered_at DESC`

	rows, err := mps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alerts: %v", err)
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		var metadataJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.Name,
			&alert.Description,
			&alert.Severity,
			&alert.Status,
			&alert.Component,
			&alert.Metric,
			&alert.Threshold,
			&alert.CurrentValue,
			&metadataJSON,
			&alert.TriggeredAt,
			&alert.ResolvedAt,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning alert: %v", err)
			continue
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &alert.Metadata); err != nil {
				alert.Metadata = make(map[string]interface{})
			}
		} else {
			alert.Metadata = make(map[string]interface{})
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAlertHistory retrieves alert history from the database
func (mps *MonitoringPersistenceService) GetAlertHistory(ctx context.Context, limit int) ([]models.Alert, error) {
	if mps.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, name, description, severity, status, component, metric,
			   threshold, current_value, metadata, triggered_at, resolved_at, created_at, updated_at
		FROM alerts
		ORDER BY triggered_at DESC
		LIMIT $1`

	rows, err := mps.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %v", err)
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		var metadataJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.Name,
			&alert.Description,
			&alert.Severity,
			&alert.Status,
			&alert.Component,
			&alert.Metric,
			&alert.Threshold,
			&alert.CurrentValue,
			&metadataJSON,
			&alert.TriggeredAt,
			&alert.ResolvedAt,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning alert: %v", err)
			continue
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &alert.Metadata); err != nil {
				alert.Metadata = make(map[string]interface{})
			}
		} else {
			alert.Metadata = make(map[string]interface{})
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// SaveAlertRule saves an alert rule to the database
func (mps *MonitoringPersistenceService) SaveAlertRule(ctx context.Context, rule *models.AlertRule) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		conditionsJSON = []byte("{}")
	}

	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		actionsJSON = []byte("[]")
	}

	query := `
		INSERT INTO alert_rules (
			name, description, component, metric, operator, threshold, severity,
			enabled, cooldown_minutes, conditions, actions, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	err = mps.db.QueryRowContext(ctx, query,
		rule.Name,
		rule.Description,
		rule.Component,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		string(rule.Severity),
		rule.Enabled,
		int(rule.Cooldown.Minutes()),
		conditionsJSON,
		actionsJSON,
		rule.CreatedAt,
		rule.UpdatedAt,
	).Scan(&rule.ID)

	if err != nil {
		return fmt.Errorf("failed to save alert rule: %v", err)
	}

	return nil
}

// GetAlertRules retrieves alert rules from the database
func (mps *MonitoringPersistenceService) GetAlertRules(ctx context.Context) ([]models.AlertRule, error) {
	if mps.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT id, name, description, component, metric, operator, threshold, severity,
			   enabled, cooldown_minutes, conditions, actions, created_at, updated_at
		FROM alert_rules
		ORDER BY name`

	rows, err := mps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rules: %v", err)
	}
	defer rows.Close()

	var rules []models.AlertRule
	for rows.Next() {
		var rule models.AlertRule
		var conditionsJSON, actionsJSON []byte
		var cooldownMinutes int

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.Component,
			&rule.Metric,
			&rule.Operator,
			&rule.Threshold,
			&rule.Severity,
			&rule.Enabled,
			&cooldownMinutes,
			&conditionsJSON,
			&actionsJSON,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning alert rule: %v", err)
			continue
		}

		rule.Cooldown = time.Duration(cooldownMinutes) * time.Minute

		// Parse conditions
		if len(conditionsJSON) > 0 {
			if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
				rule.Conditions = make(map[string]interface{})
			}
		} else {
			rule.Conditions = make(map[string]interface{})
		}

		// Parse actions
		if len(actionsJSON) > 0 {
			if err := json.Unmarshal(actionsJSON, &rule.Actions); err != nil {
				rule.Actions = models.AlertActions{}
			}
		} else {
			rule.Actions = models.AlertActions{}
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// GetRecentHealthChecks retrieves recent health checks from the database
func (mps *MonitoringPersistenceService) GetRecentHealthChecks(ctx context.Context, component string, limit int) ([]models.HealthCheck, error) {
	if mps.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	if limit <= 0 {
		limit = 10
	}

	var query string
	var args []interface{}

	if component != "" {
		query = `
			SELECT id, component, status, message, response_time_ms, metadata, checked_at, created_at
			FROM health_checks
			WHERE component = $1
			ORDER BY checked_at DESC
			LIMIT $2`
		args = []interface{}{component, limit}
	} else {
		query = `
			SELECT id, component, status, message, response_time_ms, metadata, checked_at, created_at
			FROM health_checks
			ORDER BY checked_at DESC
			LIMIT $1`
		args = []interface{}{limit}
	}

	rows, err := mps.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent health checks: %v", err)
	}
	defer rows.Close()

	var checks []models.HealthCheck
	for rows.Next() {
		var check models.HealthCheck
		var metadataJSON []byte
		var responseTimeMs int64

		err := rows.Scan(
			&check.ID,
			&check.Component,
			&check.Status,
			&check.Message,
			&responseTimeMs,
			&metadataJSON,
			&check.CheckedAt,
		)
		if err != nil {
			log.Printf("Error scanning health check: %v", err)
			continue
		}

		check.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &check.Metadata); err != nil {
				check.Metadata = make(map[string]interface{})
			}
		} else {
			check.Metadata = make(map[string]interface{})
		}

		checks = append(checks, check)
	}

	return checks, nil
}

// CleanupOldData removes old monitoring data based on retention settings
func (mps *MonitoringPersistenceService) CleanupOldData(ctx context.Context) error {
	if mps.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Clean up old health checks (keep 7 days)
	_, err := mps.db.ExecContext(ctx, "DELETE FROM health_checks WHERE checked_at < NOW() - INTERVAL '7 days'")
	if err != nil {
		log.Printf("Error cleaning up old health checks: %v", err)
	}

	// Clean up old alerts (keep 90 days)
	_, err = mps.db.ExecContext(ctx, "DELETE FROM alerts WHERE created_at < NOW() - INTERVAL '90 days'")
	if err != nil {
		log.Printf("Error cleaning up old alerts: %v", err)
	}

	// Clean up expired user sessions
	_, err = mps.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE expires_at < NOW()")
	if err != nil {
		log.Printf("Error cleaning up expired user sessions: %v", err)
	}

	// Call partition management functions
	_, err = mps.db.ExecContext(ctx, "SELECT create_daily_monitoring_partitions()")
	if err != nil {
		log.Printf("Error creating daily monitoring partitions: %v", err)
	}

	_, err = mps.db.ExecContext(ctx, "SELECT drop_old_monitoring_partitions()")
	if err != nil {
		log.Printf("Error dropping old monitoring partitions: %v", err)
	}

	return nil
}