package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// AIPerformanceTracker tracks performance patterns in AI-generated code
type AIPerformanceTracker struct {
	baselines       *PerformanceBaselines
	currentMetrics  *CurrentPerformanceMetrics
	regressionDetector *RegressionDetector
	trendAnalyzer   *TrendAnalyzer
	mu              sync.RWMutex
}

// PerformanceBaselines stores baseline performance metrics
type PerformanceBaselines struct {
	DatabaseQueryTime   float64   `json:"database_query_time_ms"`
	APIResponseTime     float64   `json:"api_response_time_ms"`
	MemoryAllocation    int64     `json:"memory_allocation_bytes"`
	CPUUtilization      float64   `json:"cpu_utilization_percent"`
	GoroutineCount      int       `json:"goroutine_count"`
	GCPauseTime         float64   `json:"gc_pause_time_ms"`
	ThroughputRPS       float64   `json:"throughput_rps"`
	ErrorRate           float64   `json:"error_rate_percent"`
	CacheHitRate        float64   `json:"cache_hit_rate_percent"`
	EstablishedAt       time.Time `json:"established_at"`
	SampleSize          int       `json:"sample_size"`
	ConfidenceLevel     float64   `json:"confidence_level"`
}

// CurrentPerformanceMetrics stores current performance measurements
type CurrentPerformanceMetrics struct {
	DatabaseQueries     []QueryPerformance    `json:"database_queries"`
	APIRequests         []APIPerformance      `json:"api_requests"`
	MemorySnapshots     []MemorySnapshot      `json:"memory_snapshots"`
	CPUSnapshots        []CPUSnapshot         `json:"cpu_snapshots"`
	GCEvents            []GCEvent             `json:"gc_events"`
	ErrorEvents         []ErrorEvent          `json:"error_events"`
	CacheOperations     []CacheOperation      `json:"cache_operations"`
	LastUpdated         time.Time             `json:"last_updated"`
}

// QueryPerformance tracks individual database query performance
type QueryPerformance struct {
	Query           string        `json:"query"`
	ExecutionTime   time.Duration `json:"execution_time"`
	RowsAffected    int64         `json:"rows_affected"`
	Timestamp       time.Time     `json:"timestamp"`
	StackTrace      string        `json:"stack_trace"`
	IsAIGenerated   bool          `json:"is_ai_generated"`
	QueryType       string        `json:"query_type"`
	TableName       string        `json:"table_name"`
	IndexesUsed     []string      `json:"indexes_used"`
	PlanCost        float64       `json:"plan_cost"`
}

// APIPerformance tracks API request performance
type APIPerformance struct {
	Endpoint        string        `json:"endpoint"`
	Method          string        `json:"method"`
	ResponseTime    time.Duration `json:"response_time"`
	StatusCode      int           `json:"status_code"`
	RequestSize     int64         `json:"request_size"`
	ResponseSize    int64         `json:"response_size"`
	Timestamp       time.Time     `json:"timestamp"`
	IsAIGenerated   bool          `json:"is_ai_generated"`
	UserAgent       string        `json:"user_agent"`
	RemoteAddr      string        `json:"remote_addr"`
}

// MemorySnapshot captures memory usage at a point in time
type MemorySnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	AllocBytes      uint64    `json:"alloc_bytes"`
	TotalAllocBytes uint64    `json:"total_alloc_bytes"`
	SysBytes        uint64    `json:"sys_bytes"`
	NumGC           uint32    `json:"num_gc"`
	HeapObjects     uint64    `json:"heap_objects"`
	StackInUse      uint64    `json:"stack_in_use"`
	GoroutineCount  int       `json:"goroutine_count"`
}

// CPUSnapshot captures CPU usage at a point in time
type CPUSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	UserTime        float64   `json:"user_time_percent"`
	SystemTime      float64   `json:"system_time_percent"`
	IdleTime        float64   `json:"idle_time_percent"`
	ProcessCPU      float64   `json:"process_cpu_percent"`
}

// GCEvent captures garbage collection events
type GCEvent struct {
	Timestamp       time.Time     `json:"timestamp"`
	PauseTime       time.Duration `json:"pause_time"`
	HeapSize        uint64        `json:"heap_size"`
	GCNum           uint32        `json:"gc_num"`
	ForcedGC        bool          `json:"forced_gc"`
}

// ErrorEvent captures error occurrences
type ErrorEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	ErrorType       string    `json:"error_type"`
	ErrorMessage    string    `json:"error_message"`
	StackTrace      string    `json:"stack_trace"`
	IsAIGenerated   bool      `json:"is_ai_generated"`
	Handled         bool      `json:"handled"`
	Recovered       bool      `json:"recovered"`
	Context         string    `json:"context"`
}

// CacheOperation captures cache operation performance
type CacheOperation struct {
	Timestamp       time.Time     `json:"timestamp"`
	Operation       string        `json:"operation"` // GET, SET, DELETE
	Key             string        `json:"key"`
	Hit             bool          `json:"hit"`
	ExecutionTime   time.Duration `json:"execution_time"`
	ValueSize       int           `json:"value_size"`
	TTL             time.Duration `json:"ttl"`
	IsAIGenerated   bool          `json:"is_ai_generated"`
}

// RegressionDetector detects performance regressions
type RegressionDetector struct {
	windowSize      int
	thresholds      *RegressionThresholds
	detectionRules  []RegressionRule
}

// RegressionThresholds defines thresholds for regression detection
type RegressionThresholds struct {
	QueryTimeIncrease    float64 `json:"query_time_increase_percent"`
	ResponseTimeIncrease float64 `json:"response_time_increase_percent"`
	MemoryIncrease       float64 `json:"memory_increase_percent"`
	ThroughputDecrease   float64 `json:"throughput_decrease_percent"`
	ErrorRateIncrease    float64 `json:"error_rate_increase_percent"`
	MinSampleSize        int     `json:"min_sample_size"`
	ConfidenceLevel      float64 `json:"confidence_level"`
}

// RegressionRule defines a rule for detecting regressions
type RegressionRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Metric      string                 `json:"metric"`
	Threshold   float64                `json:"threshold"`
	Direction   string                 `json:"direction"` // "increase" or "decrease"
	Checker     func(*PerformanceBaselines, *CurrentPerformanceMetrics) *RegressionAlert
}

// RegressionAlert represents a detected performance regression
type RegressionAlert struct {
	RuleName        string      `json:"rule_name"`
	Severity        string      `json:"severity"`
	Description     string      `json:"description"`
	CurrentValue    interface{} `json:"current_value"`
	BaselineValue   interface{} `json:"baseline_value"`
	PercentChange   float64     `json:"percent_change"`
	DetectedAt      time.Time   `json:"detected_at"`
	Confidence      float64     `json:"confidence"`
	Recommendation  string      `json:"recommendation"`
	AffectedCode    []string    `json:"affected_code"`
}

// TrendAnalyzer analyzes performance trends
type TrendAnalyzer struct {
	dataPoints      map[string][]float64
	timePoints      []time.Time
	maxDataPoints   int
}

// NewAIPerformanceTracker creates a new AI performance tracker
func NewAIPerformanceTracker() *AIPerformanceTracker {
	tracker := &AIPerformanceTracker{
		baselines: &PerformanceBaselines{
			DatabaseQueryTime: 50.0,   // 50ms
			APIResponseTime:   200.0,  // 200ms
			MemoryAllocation:  100 * 1024 * 1024, // 100MB
			CPUUtilization:    30.0,   // 30%
			GoroutineCount:    100,    // 100 goroutines
			GCPauseTime:       5.0,    // 5ms
			ThroughputRPS:     500.0,  // 500 RPS
			ErrorRate:         1.0,    // 1%
			CacheHitRate:      95.0,   // 95%
			ConfidenceLevel:   0.95,
		},
		currentMetrics: &CurrentPerformanceMetrics{
			DatabaseQueries: make([]QueryPerformance, 0),
			APIRequests:     make([]APIPerformance, 0),
			MemorySnapshots: make([]MemorySnapshot, 0),
			CPUSnapshots:    make([]CPUSnapshot, 0),
			GCEvents:        make([]GCEvent, 0),
			ErrorEvents:     make([]ErrorEvent, 0),
			CacheOperations: make([]CacheOperation, 0),
		},
		regressionDetector: &RegressionDetector{
			windowSize: 100,
			thresholds: &RegressionThresholds{
				QueryTimeIncrease:    50.0, // 50% increase
				ResponseTimeIncrease: 30.0, // 30% increase
				MemoryIncrease:       40.0, // 40% increase
				ThroughputDecrease:   25.0, // 25% decrease
				ErrorRateIncrease:    100.0, // 100% increase
				MinSampleSize:        20,
				ConfidenceLevel:      0.90,
			},
		},
		trendAnalyzer: &TrendAnalyzer{
			dataPoints:    make(map[string][]float64),
			timePoints:    make([]time.Time, 0),
			maxDataPoints: 1000,
		},
	}
	
	tracker.initializeRegressionRules()
	return tracker
}

// initializeRegressionRules sets up regression detection rules
func (t *AIPerformanceTracker) initializeRegressionRules() {
	t.regressionDetector.detectionRules = []RegressionRule{
		{
			Name:        "database_query_regression",
			Description: "Database query performance regression",
			Metric:      "query_time",
			Threshold:   t.regressionDetector.thresholds.QueryTimeIncrease,
			Direction:   "increase",
			Checker:     t.checkQueryTimeRegression,
		},
		{
			Name:        "api_response_regression",
			Description: "API response time regression",
			Metric:      "response_time",
			Threshold:   t.regressionDetector.thresholds.ResponseTimeIncrease,
			Direction:   "increase",
			Checker:     t.checkResponseTimeRegression,
		},
		{
			Name:        "memory_usage_regression",
			Description: "Memory usage regression",
			Metric:      "memory_usage",
			Threshold:   t.regressionDetector.thresholds.MemoryIncrease,
			Direction:   "increase",
			Checker:     t.checkMemoryRegression,
		},
		{
			Name:        "throughput_regression",
			Description: "Throughput performance regression",
			Metric:      "throughput",
			Threshold:   t.regressionDetector.thresholds.ThroughputDecrease,
			Direction:   "decrease",
			Checker:     t.checkThroughputRegression,
		},
		{
			Name:        "error_rate_regression",
			Description: "Error rate regression",
			Metric:      "error_rate",
			Threshold:   t.regressionDetector.thresholds.ErrorRateIncrease,
			Direction:   "increase",
			Checker:     t.checkErrorRateRegression,
		},
	}
}

// TrackDatabaseQuery tracks a database query performance
func (t *AIPerformanceTracker) TrackDatabaseQuery(query string, duration time.Duration, rowsAffected int64, isAIGenerated bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	queryPerf := QueryPerformance{
		Query:         query,
		ExecutionTime: duration,
		RowsAffected:  rowsAffected,
		Timestamp:     time.Now(),
		StackTrace:    t.getStackTrace(),
		IsAIGenerated: isAIGenerated,
		QueryType:     t.determineQueryType(query),
		TableName:     t.extractTableName(query),
	}
	
	// Keep only recent queries (last 1000)
	if len(t.currentMetrics.DatabaseQueries) >= 1000 {
		t.currentMetrics.DatabaseQueries = t.currentMetrics.DatabaseQueries[1:]
	}
	t.currentMetrics.DatabaseQueries = append(t.currentMetrics.DatabaseQueries, queryPerf)
	
	// Update trend data
	t.trendAnalyzer.addDataPoint("query_time", float64(duration.Nanoseconds())/1e6)
	
	t.currentMetrics.LastUpdated = time.Now()
}

// TrackAPIRequest tracks an API request performance
func (t *AIPerformanceTracker) TrackAPIRequest(endpoint, method string, duration time.Duration, statusCode int, requestSize, responseSize int64, isAIGenerated bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	apiPerf := APIPerformance{
		Endpoint:      endpoint,
		Method:        method,
		ResponseTime:  duration,
		StatusCode:    statusCode,
		RequestSize:   requestSize,
		ResponseSize:  responseSize,
		Timestamp:     time.Now(),
		IsAIGenerated: isAIGenerated,
	}
	
	// Keep only recent requests (last 1000)
	if len(t.currentMetrics.APIRequests) >= 1000 {
		t.currentMetrics.APIRequests = t.currentMetrics.APIRequests[1:]
	}
	t.currentMetrics.APIRequests = append(t.currentMetrics.APIRequests, apiPerf)
	
	// Update trend data
	t.trendAnalyzer.addDataPoint("response_time", float64(duration.Nanoseconds())/1e6)
	
	t.currentMetrics.LastUpdated = time.Now()
}

// TrackMemoryUsage captures current memory usage
func (t *AIPerformanceTracker) TrackMemoryUsage() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	snapshot := MemorySnapshot{
		Timestamp:       time.Now(),
		AllocBytes:      memStats.Alloc,
		TotalAllocBytes: memStats.TotalAlloc,
		SysBytes:        memStats.Sys,
		NumGC:           memStats.NumGC,
		HeapObjects:     memStats.HeapObjects,
		StackInUse:      memStats.StackInuse,
		GoroutineCount:  runtime.NumGoroutine(),
	}
	
	// Keep only recent snapshots (last 100)
	if len(t.currentMetrics.MemorySnapshots) >= 100 {
		t.currentMetrics.MemorySnapshots = t.currentMetrics.MemorySnapshots[1:]
	}
	t.currentMetrics.MemorySnapshots = append(t.currentMetrics.MemorySnapshots, snapshot)
	
	// Update trend data
	t.trendAnalyzer.addDataPoint("memory_usage", float64(memStats.Alloc))
	
	t.currentMetrics.LastUpdated = time.Now()
}

// TrackError tracks an error occurrence
func (t *AIPerformanceTracker) TrackError(errorType, errorMessage, context string, handled, recovered, isAIGenerated bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	errorEvent := ErrorEvent{
		Timestamp:     time.Now(),
		ErrorType:     errorType,
		ErrorMessage:  errorMessage,
		StackTrace:    t.getStackTrace(),
		IsAIGenerated: isAIGenerated,
		Handled:       handled,
		Recovered:     recovered,
		Context:       context,
	}
	
	// Keep only recent errors (last 500)
	if len(t.currentMetrics.ErrorEvents) >= 500 {
		t.currentMetrics.ErrorEvents = t.currentMetrics.ErrorEvents[1:]
	}
	t.currentMetrics.ErrorEvents = append(t.currentMetrics.ErrorEvents, errorEvent)
	
	t.currentMetrics.LastUpdated = time.Now()
}

// TrackCacheOperation tracks cache operation performance
func (t *AIPerformanceTracker) TrackCacheOperation(operation, key string, hit bool, duration time.Duration, valueSize int, ttl time.Duration, isAIGenerated bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	cacheOp := CacheOperation{
		Timestamp:     time.Now(),
		Operation:     operation,
		Key:           key,
		Hit:           hit,
		ExecutionTime: duration,
		ValueSize:     valueSize,
		TTL:           ttl,
		IsAIGenerated: isAIGenerated,
	}
	
	// Keep only recent operations (last 500)
	if len(t.currentMetrics.CacheOperations) >= 500 {
		t.currentMetrics.CacheOperations = t.currentMetrics.CacheOperations[1:]
	}
	t.currentMetrics.CacheOperations = append(t.currentMetrics.CacheOperations, cacheOp)
	
	t.currentMetrics.LastUpdated = time.Now()
}

// DetectRegressions detects performance regressions
func (t *AIPerformanceTracker) DetectRegressions() []*RegressionAlert {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var alerts []*RegressionAlert
	
	for _, rule := range t.regressionDetector.detectionRules {
		if alert := rule.Checker(t.baselines, t.currentMetrics); alert != nil {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts
}

// checkQueryTimeRegression checks for database query time regression
func (t *AIPerformanceTracker) checkQueryTimeRegression(baselines *PerformanceBaselines, current *CurrentPerformanceMetrics) *RegressionAlert {
	if len(current.DatabaseQueries) < t.regressionDetector.thresholds.MinSampleSize {
		return nil
	}
	
	// Calculate average query time for AI-generated queries
	var totalTime time.Duration
	var aiQueryCount int
	
	for _, query := range current.DatabaseQueries {
		if query.IsAIGenerated {
			totalTime += query.ExecutionTime
			aiQueryCount++
		}
	}
	
	if aiQueryCount == 0 {
		return nil
	}
	
	avgTimeMs := float64(totalTime.Nanoseconds()) / float64(aiQueryCount) / 1e6
	baselineMs := baselines.DatabaseQueryTime
	
	percentIncrease := (avgTimeMs - baselineMs) / baselineMs * 100
	
	if percentIncrease > t.regressionDetector.thresholds.QueryTimeIncrease {
		return &RegressionAlert{
			RuleName:      "database_query_regression",
			Severity:      t.calculateSeverity(percentIncrease, t.regressionDetector.thresholds.QueryTimeIncrease),
			Description:   fmt.Sprintf("AI-generated database queries are %.1f%% slower than baseline", percentIncrease),
			CurrentValue:  avgTimeMs,
			BaselineValue: baselineMs,
			PercentChange: percentIncrease,
			DetectedAt:    time.Now(),
			Confidence:    t.calculateConfidence(aiQueryCount),
			Recommendation: "Review recent AI-generated database queries for optimization opportunities",
			AffectedCode:  t.getAffectedQueryCode(),
		}
	}
	
	return nil
}

// checkResponseTimeRegression checks for API response time regression
func (t *AIPerformanceTracker) checkResponseTimeRegression(baselines *PerformanceBaselines, current *CurrentPerformanceMetrics) *RegressionAlert {
	if len(current.APIRequests) < t.regressionDetector.thresholds.MinSampleSize {
		return nil
	}
	
	// Calculate average response time for AI-generated endpoints
	var totalTime time.Duration
	var aiRequestCount int
	
	for _, request := range current.APIRequests {
		if request.IsAIGenerated {
			totalTime += request.ResponseTime
			aiRequestCount++
		}
	}
	
	if aiRequestCount == 0 {
		return nil
	}
	
	avgTimeMs := float64(totalTime.Nanoseconds()) / float64(aiRequestCount) / 1e6
	baselineMs := baselines.APIResponseTime
	
	percentIncrease := (avgTimeMs - baselineMs) / baselineMs * 100
	
	if percentIncrease > t.regressionDetector.thresholds.ResponseTimeIncrease {
		return &RegressionAlert{
			RuleName:      "api_response_regression",
			Severity:      t.calculateSeverity(percentIncrease, t.regressionDetector.thresholds.ResponseTimeIncrease),
			Description:   fmt.Sprintf("AI-generated API endpoints are %.1f%% slower than baseline", percentIncrease),
			CurrentValue:  avgTimeMs,
			BaselineValue: baselineMs,
			PercentChange: percentIncrease,
			DetectedAt:    time.Now(),
			Confidence:    t.calculateConfidence(aiRequestCount),
			Recommendation: "Review recent AI-generated API code for performance bottlenecks",
			AffectedCode:  t.getAffectedAPICode(),
		}
	}
	
	return nil
}

// checkMemoryRegression checks for memory usage regression
func (t *AIPerformanceTracker) checkMemoryRegression(baselines *PerformanceBaselines, current *CurrentPerformanceMetrics) *RegressionAlert {
	if len(current.MemorySnapshots) < t.regressionDetector.thresholds.MinSampleSize {
		return nil
	}
	
	// Calculate average memory usage from recent snapshots
	var totalMemory uint64
	recentSnapshots := current.MemorySnapshots
	if len(recentSnapshots) > 20 {
		recentSnapshots = recentSnapshots[len(recentSnapshots)-20:]
	}
	
	for _, snapshot := range recentSnapshots {
		totalMemory += snapshot.AllocBytes
	}
	
	avgMemoryBytes := float64(totalMemory) / float64(len(recentSnapshots))
	baselineBytes := float64(baselines.MemoryAllocation)
	
	percentIncrease := (avgMemoryBytes - baselineBytes) / baselineBytes * 100
	
	if percentIncrease > t.regressionDetector.thresholds.MemoryIncrease {
		return &RegressionAlert{
			RuleName:      "memory_usage_regression",
			Severity:      t.calculateSeverity(percentIncrease, t.regressionDetector.thresholds.MemoryIncrease),
			Description:   fmt.Sprintf("Memory usage increased by %.1f%% above baseline", percentIncrease),
			CurrentValue:  avgMemoryBytes,
			BaselineValue: baselineBytes,
			PercentChange: percentIncrease,
			DetectedAt:    time.Now(),
			Confidence:    t.calculateConfidence(len(recentSnapshots)),
			Recommendation: "Check for memory leaks in recent AI-generated code",
			AffectedCode:  []string{"Recent AI-generated functions with high memory allocation"},
		}
	}
	
	return nil
}

// checkThroughputRegression checks for throughput regression
func (t *AIPerformanceTracker) checkThroughputRegression(baselines *PerformanceBaselines, current *CurrentPerformanceMetrics) *RegressionAlert {
	if len(current.APIRequests) < t.regressionDetector.thresholds.MinSampleSize {
		return nil
	}
	
	// Calculate current throughput (simplified)
	recentRequests := current.APIRequests
	if len(recentRequests) > 100 {
		recentRequests = recentRequests[len(recentRequests)-100:]
	}
	
	if len(recentRequests) < 2 {
		return nil
	}
	
	timeSpan := recentRequests[len(recentRequests)-1].Timestamp.Sub(recentRequests[0].Timestamp)
	if timeSpan.Seconds() == 0 {
		return nil
	}
	
	currentThroughput := float64(len(recentRequests)) / timeSpan.Seconds()
	baselineThroughput := baselines.ThroughputRPS
	
	percentDecrease := (baselineThroughput - currentThroughput) / baselineThroughput * 100
	
	if percentDecrease > t.regressionDetector.thresholds.ThroughputDecrease {
		return &RegressionAlert{
			RuleName:      "throughput_regression",
			Severity:      t.calculateSeverity(percentDecrease, t.regressionDetector.thresholds.ThroughputDecrease),
			Description:   fmt.Sprintf("Throughput decreased by %.1f%% below baseline", percentDecrease),
			CurrentValue:  currentThroughput,
			BaselineValue: baselineThroughput,
			PercentChange: percentDecrease,
			DetectedAt:    time.Now(),
			Confidence:    t.calculateConfidence(len(recentRequests)),
			Recommendation: "Review system capacity and AI-generated code efficiency",
			AffectedCode:  t.getAffectedAPICode(),
		}
	}
	
	return nil
}

// checkErrorRateRegression checks for error rate regression
func (t *AIPerformanceTracker) checkErrorRateRegression(baselines *PerformanceBaselines, current *CurrentPerformanceMetrics) *RegressionAlert {
	if len(current.ErrorEvents) < t.regressionDetector.thresholds.MinSampleSize {
		return nil
	}
	
	// Calculate error rate for AI-generated code
	var aiErrors, totalAIOperations int
	
	// Count AI-generated errors
	for _, errorEvent := range current.ErrorEvents {
		if errorEvent.IsAIGenerated {
			aiErrors++
		}
	}
	
	// Estimate total AI operations (simplified)
	for _, query := range current.DatabaseQueries {
		if query.IsAIGenerated {
			totalAIOperations++
		}
	}
	for _, request := range current.APIRequests {
		if request.IsAIGenerated {
			totalAIOperations++
		}
	}
	
	if totalAIOperations == 0 {
		return nil
	}
	
	currentErrorRate := float64(aiErrors) / float64(totalAIOperations) * 100
	baselineErrorRate := baselines.ErrorRate
	
	percentIncrease := (currentErrorRate - baselineErrorRate) / baselineErrorRate * 100
	
	if percentIncrease > t.regressionDetector.thresholds.ErrorRateIncrease {
		return &RegressionAlert{
			RuleName:      "error_rate_regression",
			Severity:      t.calculateSeverity(percentIncrease, t.regressionDetector.thresholds.ErrorRateIncrease),
			Description:   fmt.Sprintf("AI-generated code error rate increased by %.1f%%", percentIncrease),
			CurrentValue:  currentErrorRate,
			BaselineValue: baselineErrorRate,
			PercentChange: percentIncrease,
			DetectedAt:    time.Now(),
			Confidence:    t.calculateConfidence(totalAIOperations),
			Recommendation: "Review error handling in recent AI-generated code",
			AffectedCode:  t.getAffectedErrorCode(),
		}
	}
	
	return nil
}

// Helper methods
func (t *AIPerformanceTracker) calculateSeverity(percentChange, threshold float64) string {
	if percentChange > threshold*2 {
		return "critical"
	} else if percentChange > threshold*1.5 {
		return "high"
	} else if percentChange > threshold {
		return "medium"
	}
	return "low"
}

func (t *AIPerformanceTracker) calculateConfidence(sampleSize int) float64 {
	if sampleSize >= 100 {
		return 0.95
	} else if sampleSize >= 50 {
		return 0.90
	} else if sampleSize >= 20 {
		return 0.80
	}
	return 0.70
}

func (t *AIPerformanceTracker) getStackTrace() string {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (t *AIPerformanceTracker) determineQueryType(query string) string {
	query = strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(query, "SELECT") {
		return "SELECT"
	} else if strings.HasPrefix(query, "INSERT") {
		return "INSERT"
	} else if strings.HasPrefix(query, "UPDATE") {
		return "UPDATE"
	} else if strings.HasPrefix(query, "DELETE") {
		return "DELETE"
	}
	return "OTHER"
}

func (t *AIPerformanceTracker) extractTableName(query string) string {
	// Simplified table name extraction
	query = strings.ToUpper(strings.TrimSpace(query))
	if strings.Contains(query, "FROM ") {
		parts := strings.Split(query, "FROM ")
		if len(parts) > 1 {
			tablePart := strings.TrimSpace(parts[1])
			tableWords := strings.Fields(tablePart)
			if len(tableWords) > 0 {
				return strings.ToLower(tableWords[0])
			}
		}
	}
	return "unknown"
}

func (t *AIPerformanceTracker) getAffectedQueryCode() []string {
	// This would typically analyze stack traces to identify affected code
	return []string{"Recent AI-generated database access functions"}
}

func (t *AIPerformanceTracker) getAffectedAPICode() []string {
	// This would typically analyze stack traces to identify affected code
	return []string{"Recent AI-generated API handlers"}
}

func (t *AIPerformanceTracker) getAffectedErrorCode() []string {
	// This would typically analyze error contexts to identify affected code
	return []string{"Recent AI-generated error-prone functions"}
}

// addDataPoint adds a data point to trend analysis
func (ta *TrendAnalyzer) addDataPoint(metric string, value float64) {
	if ta.dataPoints[metric] == nil {
		ta.dataPoints[metric] = make([]float64, 0)
	}
	
	// Keep only recent data points
	if len(ta.dataPoints[metric]) >= ta.maxDataPoints {
		ta.dataPoints[metric] = ta.dataPoints[metric][1:]
	}
	
	ta.dataPoints[metric] = append(ta.dataPoints[metric], value)
	
	// Update time points
	if len(ta.timePoints) >= ta.maxDataPoints {
		ta.timePoints = ta.timePoints[1:]
	}
	ta.timePoints = append(ta.timePoints, time.Now())
}

// UpdateBaselines updates performance baselines based on recent stable performance
func (t *AIPerformanceTracker) UpdateBaselines(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Update baselines based on recent stable performance
	if len(t.currentMetrics.DatabaseQueries) >= 100 {
		var totalTime time.Duration
		var count int
		
		for _, query := range t.currentMetrics.DatabaseQueries {
			if query.IsAIGenerated {
				totalTime += query.ExecutionTime
				count++
			}
		}
		
		if count > 0 {
			t.baselines.DatabaseQueryTime = float64(totalTime.Nanoseconds()) / float64(count) / 1e6
		}
	}
	
	if len(t.currentMetrics.APIRequests) >= 100 {
		var totalTime time.Duration
		var count int
		
		for _, request := range t.currentMetrics.APIRequests {
			if request.IsAIGenerated {
				totalTime += request.ResponseTime
				count++
			}
		}
		
		if count > 0 {
			t.baselines.APIResponseTime = float64(totalTime.Nanoseconds()) / float64(count) / 1e6
		}
	}
	
	if len(t.currentMetrics.MemorySnapshots) >= 20 {
		var totalMemory uint64
		recentSnapshots := t.currentMetrics.MemorySnapshots[len(t.currentMetrics.MemorySnapshots)-20:]
		
		for _, snapshot := range recentSnapshots {
			totalMemory += snapshot.AllocBytes
		}
		
		t.baselines.MemoryAllocation = int64(totalMemory / uint64(len(recentSnapshots)))
	}
	
	t.baselines.EstablishedAt = time.Now()
	t.baselines.SampleSize = 100
	
	return nil
}

// GetCurrentMetrics returns current performance metrics
func (t *AIPerformanceTracker) GetCurrentMetrics() *CurrentPerformanceMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	metricsCopy := *t.currentMetrics
	return &metricsCopy
}

// GetBaselines returns current performance baselines
func (t *AIPerformanceTracker) GetBaselines() *PerformanceBaselines {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	baselinesCopy := *t.baselines
	return &baselinesCopy
}

// GetTrendAnalysis returns trend analysis for a specific metric
func (t *AIPerformanceTracker) GetTrendAnalysis(metric string) []float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if data, exists := t.trendAnalyzer.dataPoints[metric]; exists {
		// Return a copy
		dataCopy := make([]float64, len(data))
		copy(dataCopy, data)
		return dataCopy
	}
	
	return nil
}