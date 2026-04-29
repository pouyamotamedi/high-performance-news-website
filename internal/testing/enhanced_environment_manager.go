package testing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// EnhancedEnvironmentManager extends the basic environment manager with advanced features
type EnhancedEnvironmentManager struct {
	*TestEnvironmentManager
	failureRecovery   *FailureRecoveryManager
	performanceMonitor *EnvironmentPerformanceMonitor
	resourceOptimizer  *ResourceOptimizer
	alertManager      *EnvironmentAlertManager
	mutex             sync.RWMutex
}

// FailureRecoveryManager handles automatic failure detection and recovery
type FailureRecoveryManager struct {
	manager           *TestEnvironmentManager
	recoveryStrategies map[string]RecoveryStrategy
	maxRetries        int
	retryDelay        time.Duration
	mutex             sync.RWMutex
}

// RecoveryStrategy defines how to recover from specific failure types
type RecoveryStrategy struct {
	Name            string        `json:"name"`
	FailurePattern  string        `json:"failure_pattern"`
	RecoveryAction  string        `json:"recovery_action"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	Enabled         bool          `json:"enabled"`
}

// EnvironmentPerformanceMonitor tracks performance metrics for environments
type EnvironmentPerformanceMonitor struct {
	environments    map[string]*EnvironmentPerformanceMetrics
	docker          *client.Client
	monitorInterval time.Duration
	mutex           sync.RWMutex
	stopChan        chan struct{}
}

// EnvironmentPerformanceMetrics tracks performance data for an environment
type EnvironmentPerformanceMetrics struct {
	EnvironmentID     string                 `json:"environment_id"`
	CPUUsage          float64                `json:"cpu_usage"`
	MemoryUsage       int64                  `json:"memory_usage"`
	MemoryLimit       int64                  `json:"memory_limit"`
	NetworkIO         NetworkIOMetrics       `json:"network_io"`
	DiskIO            DiskIOMetrics          `json:"disk_io"`
	DatabaseMetrics   DatabaseMetrics        `json:"database_metrics"`
	CacheMetrics      CacheMetrics           `json:"cache_metrics"`
	LastUpdated       time.Time              `json:"last_updated"`
	PerformanceScore  float64                `json:"performance_score"`
	Alerts            []PerformanceAlert     `json:"alerts"`
}

// NetworkIOMetrics tracks network I/O statistics
type NetworkIOMetrics struct {
	BytesReceived int64 `json:"bytes_received"`
	BytesSent     int64 `json:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received"`
	PacketsSent   int64 `json:"packets_sent"`
}

// DiskIOMetrics tracks disk I/O statistics
type DiskIOMetrics struct {
	BytesRead    int64 `json:"bytes_read"`
	BytesWritten int64 `json:"bytes_written"`
	ReadOps      int64 `json:"read_ops"`
	WriteOps     int64 `json:"write_ops"`
}

// DatabaseMetrics tracks database-specific performance
type DatabaseMetrics struct {
	ActiveConnections int     `json:"active_connections"`
	QueryLatency      float64 `json:"query_latency_ms"`
	TransactionRate   float64 `json:"transaction_rate"`
	CacheHitRatio     float64 `json:"cache_hit_ratio"`
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	HitRate       float64 `json:"hit_rate"`
	MissRate      float64 `json:"miss_rate"`
	EvictionRate  float64 `json:"eviction_rate"`
	MemoryUsage   int64   `json:"memory_usage"`
	KeyCount      int64   `json:"key_count"`
}

// PerformanceAlert represents a performance-related alert
type PerformanceAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Threshold   float64   `json:"threshold"`
	CurrentValue float64  `json:"current_value"`
	Timestamp   time.Time `json:"timestamp"`
}

// EnvironmentAlertManager manages alerts for environment issues
type EnvironmentAlertManager struct {
	alertRules    []AlertRule           `json:"alert_rules"`
	notifications chan EnvironmentAlert `json:"-"`
	handlers      []AlertHandler        `json:"-"`
	mutex         sync.RWMutex
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Severity    string                 `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Cooldown    time.Duration          `json:"cooldown"`
	LastTriggered time.Time            `json:"last_triggered"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// EnvironmentAlert represents an environment alert
type EnvironmentAlert struct {
	ID            string                 `json:"id"`
	EnvironmentID string                 `json:"environment_id"`
	RuleName      string                 `json:"rule_name"`
	Severity      string                 `json:"severity"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata"`
	Resolved      bool                   `json:"resolved"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
}

// AlertHandler defines how to handle alerts
type AlertHandler interface {
	HandleAlert(alert EnvironmentAlert) error
}

// NewEnhancedEnvironmentManager creates a new enhanced environment manager
func NewEnhancedEnvironmentManager() (*EnhancedEnvironmentManager, error) {
	baseManager, err := NewTestEnvironmentManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create base manager: %w", err)
	}

	failureRecovery := &FailureRecoveryManager{
		manager:            baseManager,
		recoveryStrategies: getDefaultRecoveryStrategies(),
		maxRetries:         3,
		retryDelay:         30 * time.Second,
	}

	performanceMonitor := &EnvironmentPerformanceMonitor{
		environments:    make(map[string]*EnvironmentPerformanceMetrics),
		docker:          baseManager.docker,
		monitorInterval: 15 * time.Second,
		stopChan:        make(chan struct{}),
	}

	alertManager := &EnvironmentAlertManager{
		alertRules:    getDefaultAlertRules(),
		notifications: make(chan EnvironmentAlert, 100),
		handlers:      []AlertHandler{},
	}

	resourceOptimizer := NewResourceOptimizer(baseManager)

	enhanced := &EnhancedEnvironmentManager{
		TestEnvironmentManager: baseManager,
		failureRecovery:        failureRecovery,
		performanceMonitor:     performanceMonitor,
		resourceOptimizer:      resourceOptimizer,
		alertManager:           alertManager,
	}

	// Start background processes
	go performanceMonitor.Start()
	go alertManager.Start()

	return enhanced, nil
}

// CreateIsolatedEnvironmentWithRecovery creates an environment with automatic failure recovery
func (e *EnhancedEnvironmentManager) CreateIsolatedEnvironmentWithRecovery(testSuite string) (*IsolatedEnvironment, error) {
	var env *IsolatedEnvironment
	var err error

	for attempt := 0; attempt <= e.failureRecovery.maxRetries; attempt++ {
		env, err = e.TestEnvironmentManager.CreateIsolatedEnvironment(testSuite)
		if err == nil {
			// Start monitoring this environment
			e.performanceMonitor.StartMonitoring(env.ID)
			return env, nil
		}

		if attempt < e.failureRecovery.maxRetries {
			log.Printf("Environment creation attempt %d failed: %v. Retrying in %v...", 
				attempt+1, err, e.failureRecovery.retryDelay)
			
			// Apply recovery strategy if available
			if strategy, exists := e.failureRecovery.recoveryStrategies[err.Error()]; exists && strategy.Enabled {
				e.applyRecoveryStrategy(strategy, err)
			}
			
			time.Sleep(e.failureRecovery.retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to create environment after %d attempts: %w", 
		e.failureRecovery.maxRetries+1, err)
}

// applyRecoveryStrategy applies a specific recovery strategy
func (e *EnhancedEnvironmentManager) applyRecoveryStrategy(strategy RecoveryStrategy, originalError error) {
	log.Printf("Applying recovery strategy '%s' for error: %v", strategy.Name, originalError)

	switch strategy.RecoveryAction {
	case "cleanup_failed_containers":
		e.cleanupFailedContainers()
	case "free_resources":
		e.freeUnusedResources()
	case "restart_docker_service":
		log.Printf("Docker service restart required - manual intervention needed")
	case "scale_down_resources":
		e.resourceOptimizer.scaleDown()
	default:
		log.Printf("Unknown recovery action: %s", strategy.RecoveryAction)
	}
}

// cleanupFailedContainers removes containers in failed state
func (e *EnhancedEnvironmentManager) cleanupFailedContainers() {
	ctx := context.Background()
	containers, err := e.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list containers for cleanup: %v", err)
		return
	}

	for _, container := range containers {
		if container.State == "exited" || container.State == "dead" {
			// Check if it's one of our test containers
			for _, name := range container.Names {
				if len(name) > 0 && (name[1:5] == "test" || name[1:9] == "test-db-" || name[1:11] == "test-cache-") {
					log.Printf("Cleaning up failed container: %s", container.ID[:12])
					e.docker.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})
					break
				}
			}
		}
	}
}

// freeUnusedResources releases resources from idle environments
func (e *EnhancedEnvironmentManager) freeUnusedResources() {
	environments := e.ListEnvironments()
	
	for _, env := range environments {
		// Check if environment has been idle for more than 15 minutes
		if time.Since(env.LastHealthCheck) > 15*time.Minute {
			log.Printf("Freeing resources from idle environment: %s", env.ID)
			e.CleanupEnvironment(env.ID)
		}
	}
}

// GetEnvironmentPerformanceMetrics returns performance metrics for an environment
func (e *EnhancedEnvironmentManager) GetEnvironmentPerformanceMetrics(envID string) (*EnvironmentPerformanceMetrics, error) {
	e.performanceMonitor.mutex.RLock()
	defer e.performanceMonitor.mutex.RUnlock()

	metrics, exists := e.performanceMonitor.environments[envID]
	if !exists {
		return nil, fmt.Errorf("performance metrics not found for environment %s", envID)
	}

	return metrics, nil
}

// GetAllPerformanceMetrics returns performance metrics for all environments
func (e *EnhancedEnvironmentManager) GetAllPerformanceMetrics() map[string]*EnvironmentPerformanceMetrics {
	e.performanceMonitor.mutex.RLock()
	defer e.performanceMonitor.mutex.RUnlock()

	result := make(map[string]*EnvironmentPerformanceMetrics)
	for id, metrics := range e.performanceMonitor.environments {
		result[id] = metrics
	}

	return result
}

// Shutdown stops the enhanced environment manager
func (e *EnhancedEnvironmentManager) Shutdown() error {
	// Stop performance monitoring
	e.performanceMonitor.Stop()
	
	// Stop alert manager
	e.alertManager.Stop()
	
	// Stop resource optimizer
	e.resourceOptimizer.Stop()
	
	// Shutdown base manager
	return e.TestEnvironmentManager.Shutdown()
}

// EnvironmentPerformanceMonitor methods

// Start begins performance monitoring
func (e *EnvironmentPerformanceMonitor) Start() {
	ticker := time.NewTicker(e.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.collectMetrics()
		}
	}
}

// Stop stops performance monitoring
func (e *EnvironmentPerformanceMonitor) Stop() {
	close(e.stopChan)
}

// StartMonitoring begins monitoring a specific environment
func (e *EnvironmentPerformanceMonitor) StartMonitoring(envID string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.environments[envID] = &EnvironmentPerformanceMetrics{
		EnvironmentID: envID,
		LastUpdated:   time.Now(),
		Alerts:        []PerformanceAlert{},
	}
}

// StopMonitoring stops monitoring a specific environment
func (e *EnvironmentPerformanceMonitor) StopMonitoring(envID string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	delete(e.environments, envID)
}

// collectMetrics collects performance metrics for all monitored environments
func (e *EnvironmentPerformanceMonitor) collectMetrics() {
	e.mutex.RLock()
	envIDs := make([]string, 0, len(e.environments))
	for id := range e.environments {
		envIDs = append(envIDs, id)
	}
	e.mutex.RUnlock()

	for _, envID := range envIDs {
		e.collectEnvironmentMetrics(envID)
	}
}

// collectEnvironmentMetrics collects metrics for a specific environment
func (e *EnvironmentPerformanceMonitor) collectEnvironmentMetrics(envID string) {
	e.mutex.Lock()
	metrics, exists := e.environments[envID]
	if !exists {
		e.mutex.Unlock()
		return
	}
	e.mutex.Unlock()

	ctx := context.Background()

	// Find containers for this environment
	containers, err := e.docker.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		log.Printf("Failed to list containers for metrics collection: %v", err)
		return
	}

	var dbContainerID, cacheContainerID string
	for _, container := range containers {
		for _, name := range container.Names {
			if len(name) > 0 && name[1:] == "test-db-"+envID {
				dbContainerID = container.ID
			} else if len(name) > 0 && name[1:] == "test-cache-"+envID {
				cacheContainerID = container.ID
			}
		}
	}

	// Collect Docker stats
	if dbContainerID != "" {
		e.collectContainerStats(metrics, dbContainerID, "database")
	}
	if cacheContainerID != "" {
		e.collectContainerStats(metrics, cacheContainerID, "cache")
	}

	// Calculate performance score
	metrics.PerformanceScore = e.calculatePerformanceScore(metrics)
	metrics.LastUpdated = time.Now()

	// Check for performance alerts
	e.checkPerformanceAlerts(metrics)

	e.mutex.Lock()
	e.environments[envID] = metrics
	e.mutex.Unlock()
}

// collectContainerStats collects Docker container statistics
func (e *EnvironmentPerformanceMonitor) collectContainerStats(metrics *EnvironmentPerformanceMetrics, containerID, containerType string) {
	ctx := context.Background()
	
	stats, err := e.docker.ContainerStats(ctx, containerID, false)
	if err != nil {
		log.Printf("Failed to get container stats for %s: %v", containerID, err)
		return
	}
	defer stats.Body.Close()

	var containerStats types.StatsJSON
	if err := containerStats.Read(stats.Body); err != nil {
		log.Printf("Failed to read container stats: %v", err)
		return
	}

	// Update metrics based on container type
	if containerType == "database" {
		metrics.CPUUsage = calculateCPUPercent(&containerStats)
		metrics.MemoryUsage = int64(containerStats.MemoryStats.Usage)
		metrics.MemoryLimit = int64(containerStats.MemoryStats.Limit)
		
		// Update network I/O
		if len(containerStats.Networks) > 0 {
			for _, network := range containerStats.Networks {
				metrics.NetworkIO.BytesReceived += int64(network.RxBytes)
				metrics.NetworkIO.BytesSent += int64(network.TxBytes)
				metrics.NetworkIO.PacketsReceived += int64(network.RxPackets)
				metrics.NetworkIO.PacketsSent += int64(network.TxPackets)
			}
		}

		// Update disk I/O
		for _, blkio := range containerStats.BlkioStats.IoServiceBytesRecursive {
			if blkio.Op == "Read" {
				metrics.DiskIO.BytesRead += int64(blkio.Value)
			} else if blkio.Op == "Write" {
				metrics.DiskIO.BytesWritten += int64(blkio.Value)
			}
		}
	}
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	
	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

// calculatePerformanceScore calculates an overall performance score
func (e *EnvironmentPerformanceMonitor) calculatePerformanceScore(metrics *EnvironmentPerformanceMetrics) float64 {
	score := 100.0

	// Deduct points for high CPU usage
	if metrics.CPUUsage > 80 {
		score -= (metrics.CPUUsage - 80) * 2
	}

	// Deduct points for high memory usage
	if metrics.MemoryLimit > 0 {
		memoryPercent := float64(metrics.MemoryUsage) / float64(metrics.MemoryLimit) * 100
		if memoryPercent > 80 {
			score -= (memoryPercent - 80) * 2
		}
	}

	// Deduct points for low cache hit rate
	if metrics.CacheMetrics.HitRate < 0.8 {
		score -= (0.8 - metrics.CacheMetrics.HitRate) * 50
	}

	// Deduct points for high query latency
	if metrics.DatabaseMetrics.QueryLatency > 100 {
		score -= (metrics.DatabaseMetrics.QueryLatency - 100) / 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

// checkPerformanceAlerts checks for performance-related alerts
func (e *EnvironmentPerformanceMonitor) checkPerformanceAlerts(metrics *EnvironmentPerformanceMetrics) {
	// Check CPU usage alert
	if metrics.CPUUsage > 90 {
		alert := PerformanceAlert{
			Type:         "high_cpu_usage",
			Severity:     "critical",
			Message:      fmt.Sprintf("CPU usage is %.2f%%, exceeding 90%% threshold", metrics.CPUUsage),
			Threshold:    90,
			CurrentValue: metrics.CPUUsage,
			Timestamp:    time.Now(),
		}
		metrics.Alerts = append(metrics.Alerts, alert)
	}

	// Check memory usage alert
	if metrics.MemoryLimit > 0 {
		memoryPercent := float64(metrics.MemoryUsage) / float64(metrics.MemoryLimit) * 100
		if memoryPercent > 85 {
			alert := PerformanceAlert{
				Type:         "high_memory_usage",
				Severity:     "warning",
				Message:      fmt.Sprintf("Memory usage is %.2f%%, exceeding 85%% threshold", memoryPercent),
				Threshold:    85,
				CurrentValue: memoryPercent,
				Timestamp:    time.Now(),
			}
			metrics.Alerts = append(metrics.Alerts, alert)
		}
	}

	// Check cache hit rate alert
	if metrics.CacheMetrics.HitRate < 0.7 {
		alert := PerformanceAlert{
			Type:         "low_cache_hit_rate",
			Severity:     "warning",
			Message:      fmt.Sprintf("Cache hit rate is %.2f%%, below 70%% threshold", metrics.CacheMetrics.HitRate*100),
			Threshold:    0.7,
			CurrentValue: metrics.CacheMetrics.HitRate,
			Timestamp:    time.Now(),
		}
		metrics.Alerts = append(metrics.Alerts, alert)
	}

	// Check query latency alert
	if metrics.DatabaseMetrics.QueryLatency > 200 {
		alert := PerformanceAlert{
			Type:         "high_query_latency",
			Severity:     "critical",
			Message:      fmt.Sprintf("Database query latency is %.2fms, exceeding 200ms threshold", metrics.DatabaseMetrics.QueryLatency),
			Threshold:    200,
			CurrentValue: metrics.DatabaseMetrics.QueryLatency,
			Timestamp:    time.Now(),
		}
		metrics.Alerts = append(metrics.Alerts, alert)
	}
}

// EnvironmentAlertManager methods

// Start begins alert processing
func (a *EnvironmentAlertManager) Start() {
	go func() {
		for alert := range a.notifications {
			a.processAlert(alert)
		}
	}()
}

// Stop stops alert processing
func (a *EnvironmentAlertManager) Stop() {
	close(a.notifications)
}

// AddAlertHandler adds a new alert handler
func (a *EnvironmentAlertManager) AddAlertHandler(handler AlertHandler) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.handlers = append(a.handlers, handler)
}

// TriggerAlert triggers a new alert
func (a *EnvironmentAlertManager) TriggerAlert(alert EnvironmentAlert) {
	select {
	case a.notifications <- alert:
	default:
		log.Printf("Alert queue full, dropping alert: %s", alert.Message)
	}
}

// processAlert processes a single alert
func (a *EnvironmentAlertManager) processAlert(alert EnvironmentAlert) {
	// Check if alert should be triggered based on cooldown
	if !a.shouldTriggerAlert(alert) {
		return
	}

	// Update last triggered time
	a.updateLastTriggered(alert.RuleName)

	// Send to all handlers
	for _, handler := range a.handlers {
		if err := handler.HandleAlert(alert); err != nil {
			log.Printf("Alert handler failed: %v", err)
		}
	}

	log.Printf("Alert triggered: %s - %s", alert.Severity, alert.Message)
}

// shouldTriggerAlert checks if an alert should be triggered based on cooldown
func (a *EnvironmentAlertManager) shouldTriggerAlert(alert EnvironmentAlert) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for _, rule := range a.alertRules {
		if rule.Name == alert.RuleName {
			if !rule.Enabled {
				return false
			}
			
			if time.Since(rule.LastTriggered) < rule.Cooldown {
				return false
			}
			
			return true
		}
	}

	return true
}

// updateLastTriggered updates the last triggered time for a rule
func (a *EnvironmentAlertManager) updateLastTriggered(ruleName string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	for i, rule := range a.alertRules {
		if rule.Name == ruleName {
			a.alertRules[i].LastTriggered = time.Now()
			break
		}
	}
}

// LogAlertHandler implements AlertHandler for logging alerts
type LogAlertHandler struct{}

// HandleAlert handles an alert by logging it
func (l *LogAlertHandler) HandleAlert(alert EnvironmentAlert) error {
	log.Printf("[ALERT] %s: %s (Environment: %s)", alert.Severity, alert.Message, alert.EnvironmentID)
	return nil
}

// EmailAlertHandler implements AlertHandler for email notifications
type EmailAlertHandler struct {
	SMTPServer string
	From       string
	To         []string
}

// HandleAlert handles an alert by sending an email
func (e *EmailAlertHandler) HandleAlert(alert EnvironmentAlert) error {
	// In a real implementation, this would send an email
	log.Printf("Would send email alert: %s to %v", alert.Message, e.To)
	return nil
}

// SlackAlertHandler implements AlertHandler for Slack notifications
type SlackAlertHandler struct {
	WebhookURL string
	Channel    string
}

// HandleAlert handles an alert by sending to Slack
func (s *SlackAlertHandler) HandleAlert(alert EnvironmentAlert) error {
	// In a real implementation, this would send to Slack
	log.Printf("Would send Slack alert to %s: %s", s.Channel, alert.Message)
	return nil
}