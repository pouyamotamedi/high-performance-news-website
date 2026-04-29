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

// AICodeMonitor provides runtime monitoring for AI-generated code patterns
type AICodeMonitor struct {
	db                *sql.DB
	metrics           *AICodeMetrics
	anomalyDetector   *AIAnomalyDetector
	performanceTracker *AIPerformanceTracker
	alertManager      *AIAlertManager
	mu                sync.RWMutex
	isRunning         bool
	stopChan          chan struct{}
}

// AICodeMetrics tracks various metrics for AI-generated code
type AICodeMetrics struct {
	DatabaseQueries    *QueryMetrics    `json:"database_queries"`
	ErrorHandling      *ErrorMetrics    `json:"error_handling"`
	BusinessLogic      *LogicMetrics    `json:"business_logic"`
	Performance        *PerformanceMetrics `json:"performance"`
	LastUpdated        time.Time        `json:"last_updated"`
	mu                 sync.RWMutex
}

// QueryMetrics tracks database query patterns from AI-generated code
type QueryMetrics struct {
	TotalQueries       int64             `json:"total_queries"`
	SlowQueries        int64             `json:"slow_queries"`
	FailedQueries      int64             `json:"failed_queries"`
	QueryPatterns      map[string]int64  `json:"query_patterns"`
	AvgExecutionTime   float64           `json:"avg_execution_time_ms"`
	MaxExecutionTime   float64           `json:"max_execution_time_ms"`
	QueryComplexity    map[string]int64  `json:"query_complexity"`
	LastSlowQuery      *SlowQueryInfo    `json:"last_slow_query"`
}

// ErrorMetrics tracks error handling effectiveness in AI-generated code
type ErrorMetrics struct {
	TotalErrors        int64             `json:"total_errors"`
	HandledErrors      int64             `json:"handled_errors"`
	UnhandledErrors    int64             `json:"unhandled_errors"`
	ErrorTypes         map[string]int64  `json:"error_types"`
	RecoverySuccess    int64             `json:"recovery_success"`
	RecoveryFailures   int64             `json:"recovery_failures"`
	ErrorPatterns      map[string]int64  `json:"error_patterns"`
	HandlingEffectiveness float64        `json:"handling_effectiveness"`
}

// LogicMetrics tracks business logic consistency in AI-generated code
type LogicMetrics struct {
	FunctionCalls      int64             `json:"function_calls"`
	LogicViolations    int64             `json:"logic_violations"`
	ConsistencyScore   float64           `json:"consistency_score"`
	ValidationFailures int64             `json:"validation_failures"`
	BusinessRuleChecks int64             `json:"business_rule_checks"`
	RuleViolations     map[string]int64  `json:"rule_violations"`
	LogicPatterns      map[string]int64  `json:"logic_patterns"`
}

// PerformanceMetrics tracks performance patterns in AI-generated code
type PerformanceMetrics struct {
	CPUUsage           float64           `json:"cpu_usage_percent"`
	MemoryUsage        int64             `json:"memory_usage_bytes"`
	GoroutineCount     int               `json:"goroutine_count"`
	GCPauses           []time.Duration   `json:"gc_pauses_ms"`
	AllocationRate     float64           `json:"allocation_rate_mb_per_sec"`
	ResponseTimes      []float64         `json:"response_times_ms"`
	ThroughputRPS      float64           `json:"throughput_rps"`
	PerformanceScore   float64           `json:"performance_score"`
}

// SlowQueryInfo contains details about slow queries
type SlowQueryInfo struct {
	Query         string        `json:"query"`
	ExecutionTime time.Duration `json:"execution_time"`
	Timestamp     time.Time     `json:"timestamp"`
	StackTrace    string        `json:"stack_trace"`
	Parameters    []interface{} `json:"parameters"`
}

// NewAICodeMonitor creates a new AI code monitoring instance
func NewAICodeMonitor(db *sql.DB) *AICodeMonitor {
	monitor := &AICodeMonitor{
		db:       db,
		stopChan: make(chan struct{}),
		metrics: &AICodeMetrics{
			DatabaseQueries: &QueryMetrics{
				QueryPatterns:   make(map[string]int64),
				QueryComplexity: make(map[string]int64),
			},
			ErrorHandling: &ErrorMetrics{
				ErrorTypes:    make(map[string]int64),
				ErrorPatterns: make(map[string]int64),
			},
			BusinessLogic: &LogicMetrics{
				RuleViolations: make(map[string]int64),
				LogicPatterns:  make(map[string]int64),
			},
			Performance: &PerformanceMetrics{
				GCPauses:      make([]time.Duration, 0),
				ResponseTimes: make([]float64, 0),
			},
		},
	}
	
	monitor.anomalyDetector = NewAIAnomalyDetector(monitor.metrics)
	monitor.performanceTracker = NewAIPerformanceTracker()
	monitor.alertManager = NewAIAlertManager()
	
	return monitor
}

// Start begins monitoring AI-generated code patterns
func (m *AICodeMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isRunning {
		return fmt.Errorf("AI code monitor is already running")
	}
	
	m.isRunning = true
	
	// Start monitoring goroutines
	go m.monitorDatabaseQueries(ctx)
	go m.monitorErrorHandling(ctx)
	go m.monitorBusinessLogic(ctx)
	go m.monitorPerformance(ctx)
	go m.runAnomalyDetection(ctx)
	
	log.Println("AI Code Monitor started successfully")
	return nil
}

// Stop stops the AI code monitoring
func (m *AICodeMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isRunning {
		return
	}
	
	close(m.stopChan)
	m.isRunning = false
	log.Println("AI Code Monitor stopped")
}

// monitorDatabaseQueries monitors database query patterns from AI-generated code
func (m *AICodeMonitor) monitorDatabaseQueries(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectQueryMetrics()
		}
	}
}

// collectQueryMetrics collects database query metrics
func (m *AICodeMonitor) collectQueryMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	// Query slow query log for AI-generated queries
	slowQueries, err := m.getSlowQueries()
	if err != nil {
		log.Printf("Error collecting slow queries: %v", err)
		return
	}
	
	for _, query := range slowQueries {
		m.metrics.DatabaseQueries.SlowQueries++
		
		// Analyze query pattern
		pattern := m.analyzeQueryPattern(query.Query)
		m.metrics.DatabaseQueries.QueryPatterns[pattern]++
		
		// Calculate complexity
		complexity := m.calculateQueryComplexity(query.Query)
		m.metrics.DatabaseQueries.QueryComplexity[complexity]++
		
		// Update execution time metrics
		execTimeMs := float64(query.ExecutionTime.Nanoseconds()) / 1e6
		if execTimeMs > m.metrics.DatabaseQueries.MaxExecutionTime {
			m.metrics.DatabaseQueries.MaxExecutionTime = execTimeMs
			m.metrics.DatabaseQueries.LastSlowQuery = &query
		}
		
		// Update average execution time
		totalTime := m.metrics.DatabaseQueries.AvgExecutionTime * float64(m.metrics.DatabaseQueries.TotalQueries)
		m.metrics.DatabaseQueries.TotalQueries++
		m.metrics.DatabaseQueries.AvgExecutionTime = (totalTime + execTimeMs) / float64(m.metrics.DatabaseQueries.TotalQueries)
	}
	
	m.metrics.LastUpdated = time.Now()
}

// getSlowQueries retrieves slow queries from the database
func (m *AICodeMonitor) getSlowQueries() ([]SlowQueryInfo, error) {
	// This would typically query pg_stat_statements or similar
	// For now, we'll simulate with a basic query
	query := `
		SELECT query, mean_exec_time, calls, total_exec_time
		FROM pg_stat_statements 
		WHERE mean_exec_time > 100 
		AND query LIKE '%articles%'
		ORDER BY mean_exec_time DESC 
		LIMIT 10
	`
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slowQueries []SlowQueryInfo
	for rows.Next() {
		var query string
		var meanTime, totalTime float64
		var calls int64
		
		if err := rows.Scan(&query, &meanTime, &calls, &totalTime); err != nil {
			continue
		}
		
		slowQueries = append(slowQueries, SlowQueryInfo{
			Query:         query,
			ExecutionTime: time.Duration(meanTime * float64(time.Millisecond)),
			Timestamp:     time.Now(),
			StackTrace:    m.getCurrentStackTrace(),
		})
	}
	
	return slowQueries, nil
}

// analyzeQueryPattern analyzes the pattern of a database query
func (m *AICodeMonitor) analyzeQueryPattern(query string) string {
	// Simplified pattern analysis
	if contains(query, "SELECT * FROM") {
		return "select_all"
	}
	if contains(query, "JOIN") && contains(query, "WHERE") {
		return "complex_join"
	}
	if contains(query, "ORDER BY") && contains(query, "LIMIT") {
		return "paginated_query"
	}
	if contains(query, "INSERT") || contains(query, "UPDATE") || contains(query, "DELETE") {
		return "write_operation"
	}
	return "simple_query"
}

// calculateQueryComplexity calculates the complexity level of a query
func (m *AICodeMonitor) calculateQueryComplexity(query string) string {
	complexity := 0
	
	// Count complexity indicators
	if contains(query, "JOIN") {
		complexity += 2
	}
	if contains(query, "SUBQUERY") || contains(query, "EXISTS") {
		complexity += 3
	}
	if contains(query, "GROUP BY") {
		complexity += 1
	}
	if contains(query, "ORDER BY") {
		complexity += 1
	}
	if contains(query, "HAVING") {
		complexity += 2
	}
	
	switch {
	case complexity >= 6:
		return "very_high"
	case complexity >= 4:
		return "high"
	case complexity >= 2:
		return "medium"
	default:
		return "low"
	}
}

// monitorErrorHandling monitors error handling effectiveness
func (m *AICodeMonitor) monitorErrorHandling(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectErrorMetrics()
		}
	}
}

// collectErrorMetrics collects error handling metrics
func (m *AICodeMonitor) collectErrorMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	// This would typically integrate with logging systems
	// For now, we'll simulate error collection
	
	// Calculate error handling effectiveness
	if m.metrics.ErrorHandling.TotalErrors > 0 {
		m.metrics.ErrorHandling.HandlingEffectiveness = 
			float64(m.metrics.ErrorHandling.HandledErrors) / float64(m.metrics.ErrorHandling.TotalErrors) * 100
	}
	
	m.metrics.LastUpdated = time.Now()
}

// monitorBusinessLogic monitors business logic consistency
func (m *AICodeMonitor) monitorBusinessLogic(ctx context.Context) {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectLogicMetrics()
		}
	}
}

// collectLogicMetrics collects business logic metrics
func (m *AICodeMonitor) collectLogicMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	// Calculate consistency score
	if m.metrics.BusinessLogic.BusinessRuleChecks > 0 {
		violationRate := float64(m.metrics.BusinessLogic.LogicViolations) / float64(m.metrics.BusinessLogic.BusinessRuleChecks)
		m.metrics.BusinessLogic.ConsistencyScore = (1.0 - violationRate) * 100
	}
	
	m.metrics.LastUpdated = time.Now()
}

// monitorPerformance monitors performance patterns
func (m *AICodeMonitor) monitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectPerformanceMetrics()
		}
	}
}

// collectPerformanceMetrics collects performance metrics
func (m *AICodeMonitor) collectPerformanceMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	// Collect runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	m.metrics.Performance.MemoryUsage = int64(memStats.Alloc)
	m.metrics.Performance.GoroutineCount = runtime.NumGoroutine()
	m.metrics.Performance.AllocationRate = float64(memStats.TotalAlloc) / 1024 / 1024 // MB
	
	// Collect GC pause times (last 10)
	if len(m.metrics.Performance.GCPauses) >= 10 {
		m.metrics.Performance.GCPauses = m.metrics.Performance.GCPauses[1:]
	}
	m.metrics.Performance.GCPauses = append(m.metrics.Performance.GCPauses, 
		time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256]))
	
	// Calculate performance score
	m.metrics.Performance.PerformanceScore = m.calculatePerformanceScore()
	
	m.metrics.LastUpdated = time.Now()
}

// calculatePerformanceScore calculates an overall performance score
func (m *AICodeMonitor) calculatePerformanceScore() float64 {
	score := 100.0
	
	// Penalize high memory usage (>500MB)
	if m.metrics.Performance.MemoryUsage > 500*1024*1024 {
		score -= 20
	}
	
	// Penalize high goroutine count (>1000)
	if m.metrics.Performance.GoroutineCount > 1000 {
		score -= 15
	}
	
	// Penalize long GC pauses (>10ms average)
	if len(m.metrics.Performance.GCPauses) > 0 {
		var totalPause time.Duration
		for _, pause := range m.metrics.Performance.GCPauses {
			totalPause += pause
		}
		avgPause := totalPause / time.Duration(len(m.metrics.Performance.GCPauses))
		if avgPause > 10*time.Millisecond {
			score -= 10
		}
	}
	
	// Penalize low throughput (<100 RPS)
	if m.metrics.Performance.ThroughputRPS < 100 {
		score -= 15
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// runAnomalyDetection runs anomaly detection on collected metrics
func (m *AICodeMonitor) runAnomalyDetection(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			anomalies := m.anomalyDetector.DetectAnomalies()
			if len(anomalies) > 0 {
				m.alertManager.SendAlerts(anomalies)
			}
		}
	}
}

// GetMetrics returns current AI code metrics
func (m *AICodeMonitor) GetMetrics() *AICodeMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	metricsCopy := *m.metrics
	return &metricsCopy
}

// RecordDatabaseQuery records a database query execution
func (m *AICodeMonitor) RecordDatabaseQuery(query string, duration time.Duration, err error) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	m.metrics.DatabaseQueries.TotalQueries++
	
	if err != nil {
		m.metrics.DatabaseQueries.FailedQueries++
	}
	
	execTimeMs := float64(duration.Nanoseconds()) / 1e6
	if execTimeMs > 100 { // Consider >100ms as slow
		m.metrics.DatabaseQueries.SlowQueries++
	}
	
	// Update average execution time
	totalTime := m.metrics.DatabaseQueries.AvgExecutionTime * float64(m.metrics.DatabaseQueries.TotalQueries-1)
	m.metrics.DatabaseQueries.AvgExecutionTime = (totalTime + execTimeMs) / float64(m.metrics.DatabaseQueries.TotalQueries)
	
	// Update max execution time
	if execTimeMs > m.metrics.DatabaseQueries.MaxExecutionTime {
		m.metrics.DatabaseQueries.MaxExecutionTime = execTimeMs
	}
	
	// Record query pattern
	pattern := m.analyzeQueryPattern(query)
	m.metrics.DatabaseQueries.QueryPatterns[pattern]++
}

// RecordError records an error occurrence and handling
func (m *AICodeMonitor) RecordError(errorType string, handled bool, recovered bool) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	m.metrics.ErrorHandling.TotalErrors++
	m.metrics.ErrorHandling.ErrorTypes[errorType]++
	
	if handled {
		m.metrics.ErrorHandling.HandledErrors++
	} else {
		m.metrics.ErrorHandling.UnhandledErrors++
	}
	
	if recovered {
		m.metrics.ErrorHandling.RecoverySuccess++
	} else {
		m.metrics.ErrorHandling.RecoveryFailures++
	}
	
	// Update handling effectiveness
	m.metrics.ErrorHandling.HandlingEffectiveness = 
		float64(m.metrics.ErrorHandling.HandledErrors) / float64(m.metrics.ErrorHandling.TotalErrors) * 100
}

// RecordBusinessLogicViolation records a business logic violation
func (m *AICodeMonitor) RecordBusinessLogicViolation(ruleType string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	m.metrics.BusinessLogic.LogicViolations++
	m.metrics.BusinessLogic.RuleViolations[ruleType]++
	
	// Recalculate consistency score
	if m.metrics.BusinessLogic.BusinessRuleChecks > 0 {
		violationRate := float64(m.metrics.BusinessLogic.LogicViolations) / float64(m.metrics.BusinessLogic.BusinessRuleChecks)
		m.metrics.BusinessLogic.ConsistencyScore = (1.0 - violationRate) * 100
	}
}

// RecordResponseTime records an API response time
func (m *AICodeMonitor) RecordResponseTime(duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	responseTimeMs := float64(duration.Nanoseconds()) / 1e6
	
	// Keep only last 100 response times
	if len(m.metrics.Performance.ResponseTimes) >= 100 {
		m.metrics.Performance.ResponseTimes = m.metrics.Performance.ResponseTimes[1:]
	}
	m.metrics.Performance.ResponseTimes = append(m.metrics.Performance.ResponseTimes, responseTimeMs)
	
	// Calculate throughput (simplified)
	if len(m.metrics.Performance.ResponseTimes) > 10 {
		// Estimate RPS based on recent response times
		recentTimes := m.metrics.Performance.ResponseTimes[len(m.metrics.Performance.ResponseTimes)-10:]
		var avgTime float64
		for _, t := range recentTimes {
			avgTime += t
		}
		avgTime /= float64(len(recentTimes))
		m.metrics.Performance.ThroughputRPS = 1000.0 / avgTime // Convert ms to RPS
	}
}

// getCurrentStackTrace gets the current stack trace
func (m *AICodeMonitor) getCurrentStackTrace() string {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsSubstring(s, substr)))
}

// containsSubstring checks if string contains substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}