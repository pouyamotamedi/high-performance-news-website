package testing

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ResourceTracker monitors system resource usage during test execution
type ResourceTracker struct {
	currentUsage    ResourceUsage
	usageHistory    []ResourceUsage
	thresholds      ResourceThresholds
	alerts          []ResourceAlert
	mu              sync.RWMutex
	isRunning       bool
	stopChan        chan struct{}
	process         *process.Process
	lastNetStats    []net.IOCountersStat
	lastDiskStats   []disk.IOCountersStat
}

// ResourceThresholds defines alert thresholds for resource usage
type ResourceThresholds struct {
	CPUPercent      float64 `json:"cpu_percent"`
	MemoryMB        float64 `json:"memory_mb"`
	DiskIOKBPerSec  float64 `json:"disk_io_kb_per_sec"`
	NetworkIOKBPerSec float64 `json:"network_io_kb_per_sec"`
	DatabaseConns   int     `json:"database_connections"`
}

// ResourceAlert represents a resource usage alert
type ResourceAlert struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// ResourceMetrics contains detailed resource metrics
type ResourceMetrics struct {
	System      SystemMetrics      `json:"system"`
	Process     ProcessMetrics     `json:"process"`
	Database    DatabaseMetrics    `json:"database"`
	Cache       CacheMetrics       `json:"cache"`
	TestSpecific TestResourceMetrics `json:"test_specific"`
	Timestamp   time.Time          `json:"timestamp"`
}

// SystemMetrics contains system-wide resource metrics
type SystemMetrics struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryUsedMB   float64 `json:"memory_used_mb"`
	MemoryTotalMB  float64 `json:"memory_total_mb"`
	MemoryPercent  float64 `json:"memory_percent"`
	DiskUsedGB     float64 `json:"disk_used_gb"`
	DiskTotalGB    float64 `json:"disk_total_gb"`
	DiskPercent    float64 `json:"disk_percent"`
	LoadAverage    []float64 `json:"load_average"`
	Uptime         float64 `json:"uptime"`
}

// ProcessMetrics contains process-specific resource metrics
type ProcessMetrics struct {
	PID           int32   `json:"pid"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	NumThreads    int32   `json:"num_threads"`
	NumFDs        int32   `json:"num_fds"`
	DiskReadKB    float64 `json:"disk_read_kb"`
	DiskWriteKB   float64 `json:"disk_write_kb"`
	NetRecvKB     float64 `json:"net_recv_kb"`
	NetSentKB     float64 `json:"net_sent_kb"`
}

// DatabaseMetrics contains database-specific metrics
type DatabaseMetrics struct {
	ActiveConnections int     `json:"active_connections"`
	IdleConnections   int     `json:"idle_connections"`
	MaxConnections    int     `json:"max_connections"`
	SlowQueries       int64   `json:"slow_queries"`
	QueryRate         float64 `json:"query_rate"`
	AvgQueryTime      float64 `json:"avg_query_time_ms"`
	DeadlockCount     int64   `json:"deadlock_count"`
	CacheHitRatio     float64 `json:"cache_hit_ratio"`
}

// CacheMetrics contains cache-specific metrics
type CacheMetrics struct {
	HitRate       float64 `json:"hit_rate"`
	MissRate      float64 `json:"miss_rate"`
	TotalHits     int64   `json:"total_hits"`
	TotalMisses   int64   `json:"total_misses"`
	KeyCount      int64   `json:"key_count"`
	MemoryUsedMB  float64 `json:"memory_used_mb"`
	EvictionCount int64   `json:"eviction_count"`
	ConnectionCount int   `json:"connection_count"`
}

// TestResourceMetrics contains test-specific resource metrics
type TestResourceMetrics struct {
	ActiveTests        int     `json:"active_tests"`
	TestEnvironments   int     `json:"test_environments"`
	TempFilesCreated   int64   `json:"temp_files_created"`
	TempDiskUsageMB    float64 `json:"temp_disk_usage_mb"`
	TestDataSizeMB     float64 `json:"test_data_size_mb"`
	MockServerCount    int     `json:"mock_server_count"`
	ContainerCount     int     `json:"container_count"`
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker() *ResourceTracker {
	// Get current process
	currentProcess, err := process.NewProcess(int32(runtime.GOMAXPROCS(0)))
	if err != nil {
		log.Printf("Warning: Could not get current process for monitoring: %v", err)
	}

	return &ResourceTracker{
		usageHistory: make([]ResourceUsage, 0),
		alerts:       make([]ResourceAlert, 0),
		stopChan:     make(chan struct{}),
		process:      currentProcess,
		thresholds: ResourceThresholds{
			CPUPercent:        80.0,
			MemoryMB:         2048.0, // 2GB
			DiskIOKBPerSec:   10240.0, // 10MB/s
			NetworkIOKBPerSec: 5120.0, // 5MB/s
			DatabaseConns:    140,     // Out of 150 max
		},
	}
}

// Start begins resource tracking
func (r *ResourceTracker) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return fmt.Errorf("resource tracker is already running")
	}

	// Initialize baseline stats
	if err := r.initializeBaseline(); err != nil {
		return fmt.Errorf("failed to initialize baseline stats: %w", err)
	}

	r.isRunning = true

	// Start monitoring goroutines
	go r.trackResources(ctx)
	go r.checkThresholds(ctx)
	go r.cleanupHistory(ctx)

	log.Printf("Resource tracker started")
	return nil
}

// Stop stops resource tracking
func (r *ResourceTracker) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}

	close(r.stopChan)
	r.isRunning = false

	log.Printf("Resource tracker stopped")
}

// GetCurrentUsage returns the current resource usage
func (r *ResourceTracker) GetCurrentUsage() ResourceUsage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.currentUsage
}

// GetDetailedMetrics returns detailed resource metrics
func (r *ResourceTracker) GetDetailedMetrics() (*ResourceMetrics, error) {
	systemMetrics, err := r.getSystemMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}

	processMetrics, err := r.getProcessMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get process metrics: %w", err)
	}

	databaseMetrics := r.getDatabaseMetrics()
	cacheMetrics := r.getCacheMetrics()
	testMetrics := r.getTestResourceMetrics()

	return &ResourceMetrics{
		System:       systemMetrics,
		Process:      processMetrics,
		Database:     databaseMetrics,
		Cache:        cacheMetrics,
		TestSpecific: testMetrics,
		Timestamp:    time.Now(),
	}, nil
}

// GetUsageHistory returns resource usage history
func (r *ResourceTracker) GetUsageHistory(limit int) []ResourceUsage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.usageHistory) {
		limit = len(r.usageHistory)
	}

	// Return most recent entries
	start := len(r.usageHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]ResourceUsage, limit)
	copy(history, r.usageHistory[start:])
	return history
}

// GetActiveAlerts returns currently active resource alerts
func (r *ResourceTracker) GetActiveAlerts() []ResourceAlert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeAlerts := make([]ResourceAlert, 0)
	for _, alert := range r.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	return activeAlerts
}

// SetThresholds updates resource thresholds
func (r *ResourceTracker) SetThresholds(thresholds ResourceThresholds) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.thresholds = thresholds
	log.Printf("Resource thresholds updated: %+v", thresholds)
}

// trackResources continuously tracks resource usage
func (r *ResourceTracker) trackResources(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.updateResourceUsage()
		}
	}
}

// updateResourceUsage updates current resource usage
func (r *ResourceTracker) updateResourceUsage() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Printf("Failed to get CPU usage: %v", err)
		cpuPercent = []float64{0}
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Failed to get memory usage: %v", err)
		memInfo = &mem.VirtualMemoryStat{}
	}

	// Get disk I/O
	diskIO := r.getDiskIO()

	// Get network I/O
	networkIO := r.getNetworkIO()

	// Update current usage
	r.currentUsage = ResourceUsage{
		CPUPercent:    cpuPercent[0],
		MemoryMB:      float64(memInfo.Used) / 1024 / 1024,
		DiskIOKB:      diskIO,
		NetworkIOKB:   networkIO,
		DatabaseConns: r.getDatabaseConnections(),
		CacheHits:     r.getCacheHits(),
		CacheMisses:   r.getCacheMisses(),
		Timestamp:     time.Now(),
	}

	// Add to history
	r.usageHistory = append(r.usageHistory, r.currentUsage)

	// Limit history size
	if len(r.usageHistory) > 1000 {
		r.usageHistory = r.usageHistory[len(r.usageHistory)-1000:]
	}
}

// checkThresholds checks if resource usage exceeds thresholds
func (r *ResourceTracker) checkThresholds(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.evaluateThresholds()
		}
	}
}

// evaluateThresholds evaluates current usage against thresholds
func (r *ResourceTracker) evaluateThresholds() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	usage := r.currentUsage

	// Check CPU threshold
	if usage.CPUPercent > r.thresholds.CPUPercent {
		r.createAlert("cpu", fmt.Sprintf("CPU usage %.1f%% exceeds threshold %.1f%%", 
			usage.CPUPercent, r.thresholds.CPUPercent), 
			usage.CPUPercent, r.thresholds.CPUPercent, "warning", now)
	}

	// Check memory threshold
	if usage.MemoryMB > r.thresholds.MemoryMB {
		r.createAlert("memory", fmt.Sprintf("Memory usage %.1fMB exceeds threshold %.1fMB", 
			usage.MemoryMB, r.thresholds.MemoryMB), 
			usage.MemoryMB, r.thresholds.MemoryMB, "warning", now)
	}

	// Check disk I/O threshold
	if usage.DiskIOKB > r.thresholds.DiskIOKBPerSec {
		r.createAlert("disk_io", fmt.Sprintf("Disk I/O %.1fKB/s exceeds threshold %.1fKB/s", 
			usage.DiskIOKB, r.thresholds.DiskIOKBPerSec), 
			usage.DiskIOKB, r.thresholds.DiskIOKBPerSec, "warning", now)
	}

	// Check network I/O threshold
	if usage.NetworkIOKB > r.thresholds.NetworkIOKBPerSec {
		r.createAlert("network_io", fmt.Sprintf("Network I/O %.1fKB/s exceeds threshold %.1fKB/s", 
			usage.NetworkIOKB, r.thresholds.NetworkIOKBPerSec), 
			usage.NetworkIOKB, r.thresholds.NetworkIOKBPerSec, "warning", now)
	}

	// Check database connections threshold
	if usage.DatabaseConns > r.thresholds.DatabaseConns {
		r.createAlert("database_connections", fmt.Sprintf("Database connections %d exceeds threshold %d", 
			usage.DatabaseConns, r.thresholds.DatabaseConns), 
			float64(usage.DatabaseConns), float64(r.thresholds.DatabaseConns), "critical", now)
	}
}

// createAlert creates a new resource alert
func (r *ResourceTracker) createAlert(alertType, message string, value, threshold float64, severity string, timestamp time.Time) {
	// Check if similar alert already exists and is not resolved
	for i, alert := range r.alerts {
		if alert.Type == alertType && !alert.Resolved {
			// Update existing alert
			r.alerts[i].Value = value
			r.alerts[i].Timestamp = timestamp
			return
		}
	}

	// Create new alert
	alert := ResourceAlert{
		Type:      alertType,
		Message:   message,
		Value:     value,
		Threshold: threshold,
		Severity:  severity,
		Timestamp: timestamp,
		Resolved:  false,
	}

	r.alerts = append(r.alerts, alert)
	log.Printf("Resource alert created: %s - %s", severity, message)
}

// cleanupHistory periodically cleans up old usage history and resolved alerts
func (r *ResourceTracker) cleanupHistory(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.performCleanup()
		}
	}
}

// performCleanup removes old data
func (r *ResourceTracker) performCleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up old alerts (keep resolved alerts for 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)
	newAlerts := make([]ResourceAlert, 0)

	for _, alert := range r.alerts {
		if !alert.Resolved || (alert.ResolvedAt != nil && alert.ResolvedAt.After(cutoff)) {
			newAlerts = append(newAlerts, alert)
		}
	}

	oldAlertCount := len(r.alerts)
	r.alerts = newAlerts
	newAlertCount := len(r.alerts)

	if oldAlertCount != newAlertCount {
		log.Printf("Cleaned up resource alerts: removed %d old alerts, kept %d", 
			oldAlertCount-newAlertCount, newAlertCount)
	}
}

// Helper methods for getting specific metrics

func (r *ResourceTracker) initializeBaseline() error {
	// Initialize network and disk stats for delta calculations
	netStats, err := net.IOCounters(false)
	if err != nil {
		log.Printf("Warning: Could not get network stats: %v", err)
	} else {
		r.lastNetStats = netStats
	}

	diskStats, err := disk.IOCounters()
	if err != nil {
		log.Printf("Warning: Could not get disk stats: %v", err)
	} else {
		// Convert map to slice
		r.lastDiskStats = make([]disk.IOCountersStat, 0, len(diskStats))
		for _, stat := range diskStats {
			r.lastDiskStats = append(r.lastDiskStats, stat)
		}
	}

	return nil
}

func (r *ResourceTracker) getDiskIO() float64 {
	// This is a simplified implementation
	// In a real implementation, you would calculate the delta from the last measurement
	return 0.0 // Placeholder
}

func (r *ResourceTracker) getNetworkIO() float64 {
	// This is a simplified implementation
	// In a real implementation, you would calculate the delta from the last measurement
	return 0.0 // Placeholder
}

func (r *ResourceTracker) getDatabaseConnections() int {
	// This would integrate with your database monitoring
	// For now, return a placeholder value
	return 0
}

func (r *ResourceTracker) getCacheHits() int64 {
	// This would integrate with your cache monitoring
	// For now, return a placeholder value
	return 0
}

func (r *ResourceTracker) getCacheMisses() int64 {
	// This would integrate with your cache monitoring
	// For now, return a placeholder value
	return 0
}

func (r *ResourceTracker) getSystemMetrics() (SystemMetrics, error) {
	// Get CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return SystemMetrics{}, err
	}

	// Get memory info
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return SystemMetrics{}, err
	}

	// Get disk info
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return SystemMetrics{}, err
	}

	return SystemMetrics{
		CPUPercent:    cpuPercent[0],
		MemoryUsedMB:  float64(memInfo.Used) / 1024 / 1024,
		MemoryTotalMB: float64(memInfo.Total) / 1024 / 1024,
		MemoryPercent: memInfo.UsedPercent,
		DiskUsedGB:    float64(diskInfo.Used) / 1024 / 1024 / 1024,
		DiskTotalGB:   float64(diskInfo.Total) / 1024 / 1024 / 1024,
		DiskPercent:   diskInfo.UsedPercent,
		LoadAverage:   []float64{0, 0, 0}, // Placeholder
		Uptime:        0, // Placeholder
	}, nil
}

func (r *ResourceTracker) getProcessMetrics() (ProcessMetrics, error) {
	if r.process == nil {
		return ProcessMetrics{}, fmt.Errorf("process not available")
	}

	cpuPercent, err := r.process.CPUPercent()
	if err != nil {
		return ProcessMetrics{}, err
	}

	memInfo, err := r.process.MemoryInfo()
	if err != nil {
		return ProcessMetrics{}, err
	}

	numThreads, err := r.process.NumThreads()
	if err != nil {
		numThreads = 0
	}

	return ProcessMetrics{
		PID:           r.process.Pid,
		CPUPercent:    cpuPercent,
		MemoryMB:      float64(memInfo.RSS) / 1024 / 1024,
		MemoryPercent: 0, // Would need system memory total to calculate
		NumThreads:    numThreads,
		NumFDs:        0, // Placeholder
		DiskReadKB:    0, // Placeholder
		DiskWriteKB:   0, // Placeholder
		NetRecvKB:     0, // Placeholder
		NetSentKB:     0, // Placeholder
	}, nil
}

func (r *ResourceTracker) getDatabaseMetrics() DatabaseMetrics {
	// This would integrate with your database monitoring
	return DatabaseMetrics{
		ActiveConnections: 0,
		IdleConnections:   0,
		MaxConnections:    150,
		SlowQueries:       0,
		QueryRate:         0,
		AvgQueryTime:      0,
		DeadlockCount:     0,
		CacheHitRatio:     0,
	}
}

func (r *ResourceTracker) getCacheMetrics() CacheMetrics {
	// This would integrate with your cache monitoring
	return CacheMetrics{
		HitRate:         0,
		MissRate:        0,
		TotalHits:       0,
		TotalMisses:     0,
		KeyCount:        0,
		MemoryUsedMB:    0,
		EvictionCount:   0,
		ConnectionCount: 0,
	}
}

func (r *ResourceTracker) getTestResourceMetrics() TestResourceMetrics {
	// This would track test-specific resources
	return TestResourceMetrics{
		ActiveTests:      0,
		TestEnvironments: 0,
		TempFilesCreated: 0,
		TempDiskUsageMB:  0,
		TestDataSizeMB:   0,
		MockServerCount:  0,
		ContainerCount:   0,
	}
}