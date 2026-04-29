package task25

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// FailureDetector automatically detects various types of failures
type FailureDetector struct {
	failures        map[string]FailureEvent
	resolvedCount   int
	mu              sync.RWMutex
	checks          []HealthCheck
	checkInterval   time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
}

// HealthCheck defines a health check function
type HealthCheck struct {
	Name        string
	Type        FailureType
	CheckFunc   func() error
	Interval    time.Duration
	Timeout     time.Duration
	Enabled     bool
	LastCheck   time.Time
	LastResult  error
}

// NewFailureDetector creates a new failure detector
func NewFailureDetector() *FailureDetector {
	ctx, cancel := context.WithCancel(context.Background())
	
	fd := &FailureDetector{
		failures:      make(map[string]FailureEvent),
		resolvedCount: 0,
		checkInterval: 5 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Register default health checks
	fd.registerDefaultHealthChecks()
	
	return fd
}

// Start begins failure detection monitoring
func (fd *FailureDetector) Start(ctx context.Context) error {
	log.Println("Starting Failure Detector...")
	
	// Start health check loop
	go fd.healthCheckLoop()
	
	log.Println("Failure Detector started")
	return nil
}

// Stop gracefully shuts down the failure detector
func (fd *FailureDetector) Stop() error {
	log.Println("Stopping Failure Detector...")
	fd.cancel()
	return nil
}

// healthCheckLoop runs periodic health checks
func (fd *FailureDetector) healthCheckLoop() {
	ticker := time.NewTicker(fd.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-fd.ctx.Done():
			return
		case <-ticker.C:
			fd.runHealthChecks()
		}
	}
}

// runHealthChecks executes all enabled health checks
func (fd *FailureDetector) runHealthChecks() {
	for i := range fd.checks {
		check := &fd.checks[i]
		
		if !check.Enabled {
			continue
		}
		
		// Skip if not time for this check yet
		if time.Since(check.LastCheck) < check.Interval {
			continue
		}
		
		go fd.executeHealthCheck(check)
	}
}

// executeHealthCheck runs a single health check
func (fd *FailureDetector) executeHealthCheck(check *HealthCheck) {
	check.LastCheck = time.Now()
	
	// Create timeout context
	ctx, cancel := context.WithTimeout(fd.ctx, check.Timeout)
	defer cancel()
	
	// Execute check with timeout
	done := make(chan error, 1)
	go func() {
		done <- check.CheckFunc()
	}()
	
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		err = fmt.Errorf("health check timeout: %s", check.Name)
	}
	
	check.LastResult = err
	
	if err != nil {
		fd.reportFailure(check, err)
	}
}

// reportFailure creates a failure event
func (fd *FailureDetector) reportFailure(check *HealthCheck, err error) {
	failureID := fmt.Sprintf("%s_%d", check.Name, time.Now().Unix())
	
	failure := FailureEvent{
		ID:         failureID,
		Type:       check.Type,
		Severity:   fd.determineSeverity(check.Type, err),
		Component:  check.Name,
		Message:    err.Error(),
		Context:    map[string]interface{}{
			"check_name": check.Name,
			"check_type": string(check.Type),
		},
		DetectedAt: time.Now(),
		Resolved:   false,
	}
	
	fd.mu.Lock()
	fd.failures[failureID] = failure
	fd.mu.Unlock()
	
	log.Printf("Failure detected: %s - %s", failureID, err.Error())
}

// determineSeverity determines failure severity based on type and error
func (fd *FailureDetector) determineSeverity(failureType FailureType, err error) FailureSeverity {
	switch failureType {
	case FailureTypeDatabase:
		return SeverityCritical
	case FailureTypeCache:
		return SeverityHigh
	case FailureTypeNetwork:
		return SeverityMedium
	case FailureTypeMemory:
		return SeverityHigh
	case FailureTypeDisk:
		return SeverityMedium
	case FailureTypeService:
		return SeverityMedium
	case FailureTypeEnvironment:
		return SeverityHigh
	case FailureTypeTest:
		return SeverityLow
	default:
		return SeverityMedium
	}
}

// GetPendingFailures returns all unresolved failures
func (fd *FailureDetector) GetPendingFailures() []FailureEvent {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	
	var pending []FailureEvent
	for _, failure := range fd.failures {
		if !failure.Resolved {
			pending = append(pending, failure)
		}
	}
	
	return pending
}

// MarkResolved marks a failure as resolved
func (fd *FailureDetector) MarkResolved(failureID string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	
	if failure, exists := fd.failures[failureID]; exists {
		now := time.Now()
		failure.Resolved = true
		failure.ResolvedAt = &now
		fd.failures[failureID] = failure
		fd.resolvedCount++
		
		log.Printf("Marked failure as resolved: %s", failureID)
	}
}

// GetResolvedCount returns the number of resolved failures
func (fd *FailureDetector) GetResolvedCount() int {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	return fd.resolvedCount
}

// registerDefaultHealthChecks sets up built-in health checks
func (fd *FailureDetector) registerDefaultHealthChecks() {
	// Database connectivity check
	fd.checks = append(fd.checks, HealthCheck{
		Name:      "database_connectivity",
		Type:      FailureTypeDatabase,
		CheckFunc: fd.checkDatabaseConnectivity,
		Interval:  10 * time.Second,
		Timeout:   5 * time.Second,
		Enabled:   true,
	})
	
	// Cache connectivity check
	fd.checks = append(fd.checks, HealthCheck{
		Name:      "cache_connectivity",
		Type:      FailureTypeCache,
		CheckFunc: fd.checkCacheConnectivity,
		Interval:  15 * time.Second,
		Timeout:   3 * time.Second,
		Enabled:   true,
	})
	
	// Memory usage check
	fd.checks = append(fd.checks, HealthCheck{
		Name:      "memory_usage",
		Type:      FailureTypeMemory,
		CheckFunc: fd.checkMemoryUsage,
		Interval:  30 * time.Second,
		Timeout:   2 * time.Second,
		Enabled:   true,
	})
	
	// Disk space check
	fd.checks = append(fd.checks, HealthCheck{
		Name:      "disk_space",
		Type:      FailureTypeDisk,
		CheckFunc: fd.checkDiskSpace,
		Interval:  60 * time.Second,
		Timeout:   5 * time.Second,
		Enabled:   true,
	})
	
	// Network connectivity check
	fd.checks = append(fd.checks, HealthCheck{
		Name:      "network_connectivity",
		Type:      FailureTypeNetwork,
		CheckFunc: fd.checkNetworkConnectivity,
		Interval:  20 * time.Second,
		Timeout:   10 * time.Second,
		Enabled:   true,
	})
}

// Health check implementations
func (fd *FailureDetector) checkDatabaseConnectivity() error {
	// This would connect to the actual database
	// For now, simulate a check
	
	// Simulate occasional failures
	if time.Now().Second()%30 == 0 {
		return fmt.Errorf("database connection failed")
	}
	
	return nil // Simulated success
}

func (fd *FailureDetector) checkCacheConnectivity() error {
	// This would connect to the actual cache service (Redis/DragonflyDB)
	// For now, simulate a check
	
	// Simulate occasional failures
	if time.Now().Second()%25 == 0 {
		return fmt.Errorf("cache service unavailable")
	}
	
	return nil // Simulated success
}

func (fd *FailureDetector) checkMemoryUsage() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Check if memory usage is too high (>80% of available)
	const maxMemoryMB = 1024 // 1GB limit for example
	currentMemoryMB := m.Alloc / 1024 / 1024
	
	if currentMemoryMB > maxMemoryMB*80/100 {
		return fmt.Errorf("high memory usage: %d MB (>80%% of %d MB)", 
			currentMemoryMB, maxMemoryMB)
	}
	
	return nil
}

func (fd *FailureDetector) checkDiskSpace() error {
	// This would check actual disk space
	// For now, simulate a check
	
	// Simulate occasional disk space issues
	if time.Now().Second()%45 == 0 {
		return fmt.Errorf("low disk space detected")
	}
	
	return nil // Simulated success
}

func (fd *FailureDetector) checkNetworkConnectivity() error {
	// Test network connectivity to a known endpoint
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get("http://httpbin.org/status/200")
	if err != nil {
		return fmt.Errorf("network connectivity failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("network check failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// AddCustomHealthCheck allows adding custom health checks
func (fd *FailureDetector) AddCustomHealthCheck(check HealthCheck) {
	fd.checks = append(fd.checks, check)
	log.Printf("Added custom health check: %s", check.Name)
}

// GetHealthCheckStatus returns the status of all health checks
func (fd *FailureDetector) GetHealthCheckStatus() []HealthCheckStatus {
	var status []HealthCheckStatus
	
	for _, check := range fd.checks {
		status = append(status, HealthCheckStatus{
			Name:       check.Name,
			Type:       string(check.Type),
			Enabled:    check.Enabled,
			LastCheck:  check.LastCheck,
			LastResult: check.LastResult,
			Healthy:    check.LastResult == nil,
		})
	}
	
	return status
}

type HealthCheckStatus struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Enabled    bool      `json:"enabled"`
	LastCheck  time.Time `json:"last_check"`
	LastResult error     `json:"last_result,omitempty"`
	Healthy    bool      `json:"healthy"`
}