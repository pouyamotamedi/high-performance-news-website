package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// =============================================================================
// ENTERPRISE CONFIGURATION
// =============================================================================

// SearchEngineMode defines the search engine operation mode
type SearchEngineMode string

const (
	SearchModePostgres SearchEngineMode = "postgres"
	SearchModeMeili    SearchEngineMode = "meili"
	SearchModeHybrid   SearchEngineMode = "hybrid" // Try Meili first, fallback to Postgres
)

// EnterpriseSearchConfig extends SearchConfig with enterprise features
type EnterpriseSearchConfig struct {
	SearchConfig

	// Engine mode
	EngineMode SearchEngineMode

	// Timeouts
	PostgreSQLTimeout  time.Duration
	MeiliSearchTimeout time.Duration
	CacheTimeout       time.Duration

	// Concurrency
	MaxConcurrentSearches int
	RequestQueueSize      int

	// Rate limiting
	RateLimitPerSecond int
	RateLimitBurst     int

	// Slow query threshold
	SlowQueryThreshold time.Duration

	// Circuit breaker with exponential backoff
	CircuitBreakerBackoffBase time.Duration
	CircuitBreakerBackoffMax  time.Duration

	// Index reconciliation
	ReconciliationInterval time.Duration

	// Dead letter queue
	DeadLetterRetryInterval time.Duration
	DeadLetterMaxRetries    int
}

// DefaultEnterpriseConfig returns production-safe enterprise defaults
func DefaultEnterpriseConfig() EnterpriseSearchConfig {
	return EnterpriseSearchConfig{
		SearchConfig: SearchConfig{
			CacheTTL:                5 * time.Minute,
			MeiliSearchTimeout:      10 * time.Second,
			MaxQueryLength:          500,
			MaxLimit:                50,
			MaxOffset:               10000,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   30 * time.Second,
		},
		EngineMode:                SearchModeHybrid,
		PostgreSQLTimeout:         15 * time.Second,
		MeiliSearchTimeout:        10 * time.Second,  // Enterprise-level timeout
		CacheTimeout:              2 * time.Second,
		MaxConcurrentSearches:     500,
		RequestQueueSize:          1000,
		RateLimitPerSecond:        100,
		RateLimitBurst:            200,
		SlowQueryThreshold:        100 * time.Millisecond,
		CircuitBreakerBackoffBase: 1 * time.Second,
		CircuitBreakerBackoffMax:  60 * time.Second,
		ReconciliationInterval:    1 * time.Hour,
		DeadLetterRetryInterval:   5 * time.Minute,
		DeadLetterMaxRetries:      5,
	}
}

// LoadEnterpriseConfigFromEnv loads configuration from environment variables
func LoadEnterpriseConfigFromEnv() EnterpriseSearchConfig {
	config := DefaultEnterpriseConfig()

	// Engine mode
	if mode := os.Getenv("SEARCH_ENGINE"); mode != "" {
		switch mode {
		case "postgres":
			config.EngineMode = SearchModePostgres
		case "meili":
			config.EngineMode = SearchModeMeili
		case "hybrid":
			config.EngineMode = SearchModeHybrid
		}
	}

	return config
}

// =============================================================================
// PROMETHEUS METRICS
// =============================================================================

var (
	// Search latency histogram
	searchLatencyHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "search_latency_seconds",
			Help:    "Search request latency in seconds",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.15, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"source", "status"},
	)

	// Search requests counter
	searchRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_requests_total",
			Help: "Total number of search requests",
		},
		[]string{"source", "status"},
	)

	// Cache hit ratio gauge
	cacheHitRatioGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "search_cache_hit_ratio",
			Help: "Cache hit ratio (0-1)",
		},
	)

	// Fallback rate gauge
	fallbackRateGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "search_fallback_rate",
			Help: "PostgreSQL fallback rate (0-1)",
		},
	)

	// Indexing duration histogram
	indexingDurationHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "search_indexing_duration_seconds",
			Help:    "Article indexing duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
	)

	// Circuit breaker state gauge
	circuitBreakerStateGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "search_circuit_breaker_open",
			Help: "Circuit breaker state (0=closed, 1=open)",
		},
	)

	// Concurrent searches gauge
	concurrentSearchesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "search_concurrent_requests",
			Help: "Number of concurrent search requests",
		},
	)

	// Dead letter queue size gauge
	deadLetterQueueSizeGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "search_dead_letter_queue_size",
			Help: "Number of items in dead letter queue",
		},
	)
)

// =============================================================================
// ENHANCED METRICS TRACKING
// =============================================================================

// EnhancedSearchMetrics provides detailed metrics with thread-safe operations
type EnhancedSearchMetrics struct {
	TotalSearches       int64
	MeiliSearchHits     int64
	PostgreSQLFallbacks int64
	CacheHits           int64
	CacheMisses         int64
	Errors              int64
	SlowQueries         int64
	RateLimited         int64

	// Latency tracking (in nanoseconds)
	TotalLatencyNs int64
	P95LatencyNs   int64
	P99LatencyNs   int64

	// Latency samples for percentile calculation
	latencySamples []int64
	latencyMu      sync.RWMutex
	maxSamples     int
}

// NewEnhancedSearchMetrics creates a new metrics instance
func NewEnhancedSearchMetrics() *EnhancedSearchMetrics {
	return &EnhancedSearchMetrics{
		latencySamples: make([]int64, 0, 10000),
		maxSamples:     10000,
	}
}

// RecordLatency records a latency sample
func (m *EnhancedSearchMetrics) RecordLatency(latencyNs int64) {
	m.latencyMu.Lock()
	defer m.latencyMu.Unlock()

	if len(m.latencySamples) >= m.maxSamples {
		// Remove oldest samples (keep last 80%)
		copy(m.latencySamples, m.latencySamples[m.maxSamples/5:])
		m.latencySamples = m.latencySamples[:m.maxSamples*4/5]
	}
	m.latencySamples = append(m.latencySamples, latencyNs)

	// Update total latency
	atomic.AddInt64(&m.TotalLatencyNs, latencyNs)
}

// GetPercentiles calculates P95 and P99 latencies
func (m *EnhancedSearchMetrics) GetPercentiles() (p95, p99 int64) {
	m.latencyMu.RLock()
	defer m.latencyMu.RUnlock()

	if len(m.latencySamples) == 0 {
		return 0, 0
	}

	// Sort samples for percentile calculation
	sorted := make([]int64, len(m.latencySamples))
	copy(sorted, m.latencySamples)

	// Simple bubble sort for small datasets (use quicksort for production)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return sorted[p95Index], sorted[p99Index]
}

// GetCacheHitRatio returns the cache hit ratio
func (m *EnhancedSearchMetrics) GetCacheHitRatio() float64 {
	hits := atomic.LoadInt64(&m.CacheHits)
	misses := atomic.LoadInt64(&m.CacheMisses)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// GetFallbackRate returns the PostgreSQL fallback rate
func (m *EnhancedSearchMetrics) GetFallbackRate() float64 {
	fallbacks := atomic.LoadInt64(&m.PostgreSQLFallbacks)
	meili := atomic.LoadInt64(&m.MeiliSearchHits)
	total := fallbacks + meili
	if total == 0 {
		return 0
	}
	return float64(fallbacks) / float64(total)
}

// ToMap returns metrics as a map
func (m *EnhancedSearchMetrics) ToMap() map[string]interface{} {
	p95, p99 := m.GetPercentiles()
	return map[string]interface{}{
		"total_searches":        atomic.LoadInt64(&m.TotalSearches),
		"meilisearch_hits":      atomic.LoadInt64(&m.MeiliSearchHits),
		"postgresql_fallbacks":  atomic.LoadInt64(&m.PostgreSQLFallbacks),
		"cache_hits":            atomic.LoadInt64(&m.CacheHits),
		"cache_misses":          atomic.LoadInt64(&m.CacheMisses),
		"errors":                atomic.LoadInt64(&m.Errors),
		"slow_queries":          atomic.LoadInt64(&m.SlowQueries),
		"rate_limited":          atomic.LoadInt64(&m.RateLimited),
		"cache_hit_ratio":       m.GetCacheHitRatio(),
		"fallback_rate":         m.GetFallbackRate(),
		"p95_latency_ms":        float64(p95) / 1e6,
		"p99_latency_ms":        float64(p99) / 1e6,
		"avg_latency_ms":        m.GetAverageLatencyMs(),
	}
}

// GetAverageLatencyMs returns average latency in milliseconds
func (m *EnhancedSearchMetrics) GetAverageLatencyMs() float64 {
	total := atomic.LoadInt64(&m.TotalSearches)
	if total == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&m.TotalLatencyNs)) / float64(total) / 1e6
}

// =============================================================================
// ENHANCED CIRCUIT BREAKER WITH EXPONENTIAL BACKOFF
// =============================================================================

// EnhancedCircuitBreaker implements circuit breaker with exponential backoff
type EnhancedCircuitBreaker struct {
	failures        int32
	consecutiveFail int32
	lastFailure     time.Time
	threshold       int32
	baseTimeout     time.Duration
	maxTimeout      time.Duration
	mu              sync.RWMutex
	isOpen          bool
	halfOpenAllowed int32 // Number of requests allowed in half-open state
}

// NewEnhancedCircuitBreaker creates a new enhanced circuit breaker
func NewEnhancedCircuitBreaker(threshold int, baseTimeout, maxTimeout time.Duration) *EnhancedCircuitBreaker {
	return &EnhancedCircuitBreaker{
		threshold:       int32(threshold),
		baseTimeout:     baseTimeout,
		maxTimeout:      maxTimeout,
		halfOpenAllowed: 1,
	}
}

// IsOpen returns whether the circuit breaker is open
func (cb *EnhancedCircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if !cb.isOpen {
		return false
	}

	// Calculate exponential backoff timeout
	backoffMultiplier := int32(1) << cb.consecutiveFail
	timeout := time.Duration(int64(cb.baseTimeout) * int64(backoffMultiplier))
	if timeout > cb.maxTimeout {
		timeout = cb.maxTimeout
	}

	// Check if timeout has passed (half-open state)
	if time.Since(cb.lastFailure) > timeout {
		return false
	}
	return true
}

// GetState returns the current state as a string
func (cb *EnhancedCircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if !cb.isOpen {
		return "closed"
	}

	// Calculate timeout
	backoffMultiplier := int32(1) << cb.consecutiveFail
	timeout := time.Duration(int64(cb.baseTimeout) * int64(backoffMultiplier))
	if timeout > cb.maxTimeout {
		timeout = cb.maxTimeout
	}

	if time.Since(cb.lastFailure) > timeout {
		return "half-open"
	}
	return "open"
}

// RecordSuccess records a successful operation
func (cb *EnhancedCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.consecutiveFail = 0
	cb.isOpen = false
	circuitBreakerStateGauge.Set(0)
}

// RecordFailure records a failed operation
func (cb *EnhancedCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		if !cb.isOpen {
			cb.consecutiveFail++
		}
		cb.isOpen = true
		circuitBreakerStateGauge.Set(1)
		log.Printf("[CIRCUIT_BREAKER] OPENED after %d failures (backoff level: %d)", cb.failures, cb.consecutiveFail)
	}
}

// GetBackoffDuration returns the current backoff duration
func (cb *EnhancedCircuitBreaker) GetBackoffDuration() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	backoffMultiplier := int32(1) << cb.consecutiveFail
	timeout := time.Duration(int64(cb.baseTimeout) * int64(backoffMultiplier))
	if timeout > cb.maxTimeout {
		timeout = cb.maxTimeout
	}
	return timeout
}

// =============================================================================
// DEAD LETTER QUEUE FOR FAILED INDEXING
// =============================================================================

// FailedIndexItem represents a failed indexing operation
type FailedIndexItem struct {
	ArticleID   uint64    `json:"article_id"`
	Operation   string    `json:"operation"` // "index" or "delete"
	FailedAt    time.Time `json:"failed_at"`
	RetryCount  int       `json:"retry_count"`
	LastError   string    `json:"last_error"`
	ArticleData []byte    `json:"article_data,omitempty"` // Serialized article for retry
}

// DeadLetterQueue manages failed indexing operations
type DeadLetterQueue struct {
	items      []FailedIndexItem
	mu         sync.RWMutex
	maxRetries int
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(maxRetries int) *DeadLetterQueue {
	return &DeadLetterQueue{
		items:      make([]FailedIndexItem, 0),
		maxRetries: maxRetries,
	}
}

// Add adds a failed item to the queue
func (dlq *DeadLetterQueue) Add(item FailedIndexItem) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Check if item already exists
	for i, existing := range dlq.items {
		if existing.ArticleID == item.ArticleID && existing.Operation == item.Operation {
			dlq.items[i].RetryCount++
			dlq.items[i].LastError = item.LastError
			dlq.items[i].FailedAt = item.FailedAt
			deadLetterQueueSizeGauge.Set(float64(len(dlq.items)))
			return
		}
	}

	dlq.items = append(dlq.items, item)
	deadLetterQueueSizeGauge.Set(float64(len(dlq.items)))
	log.Printf("[DLQ] Added article %d to dead letter queue (operation: %s)", item.ArticleID, item.Operation)
}

// GetRetryable returns items that can be retried
func (dlq *DeadLetterQueue) GetRetryable() []FailedIndexItem {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	var retryable []FailedIndexItem
	for _, item := range dlq.items {
		if item.RetryCount < dlq.maxRetries {
			retryable = append(retryable, item)
		}
	}
	return retryable
}

// Remove removes an item from the queue
func (dlq *DeadLetterQueue) Remove(articleID uint64, operation string) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	for i, item := range dlq.items {
		if item.ArticleID == articleID && item.Operation == operation {
			dlq.items = append(dlq.items[:i], dlq.items[i+1:]...)
			deadLetterQueueSizeGauge.Set(float64(len(dlq.items)))
			return
		}
	}
}

// Size returns the queue size
func (dlq *DeadLetterQueue) Size() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	return len(dlq.items)
}

// GetAll returns all items in the queue
func (dlq *DeadLetterQueue) GetAll() []FailedIndexItem {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	result := make([]FailedIndexItem, len(dlq.items))
	copy(result, dlq.items)
	return result
}

// =============================================================================
// CONCURRENCY LIMITER
// =============================================================================

// ConcurrencyLimiter limits concurrent operations
type ConcurrencyLimiter struct {
	semaphore chan struct{}
	current   int64
	max       int
}

// NewConcurrencyLimiter creates a new concurrency limiter
func NewConcurrencyLimiter(max int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		semaphore: make(chan struct{}, max),
		max:       max,
	}
}

// Acquire acquires a slot, blocking if necessary
func (cl *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	select {
	case cl.semaphore <- struct{}{}:
		atomic.AddInt64(&cl.current, 1)
		concurrentSearchesGauge.Set(float64(atomic.LoadInt64(&cl.current)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire tries to acquire a slot without blocking
func (cl *ConcurrencyLimiter) TryAcquire() bool {
	select {
	case cl.semaphore <- struct{}{}:
		atomic.AddInt64(&cl.current, 1)
		concurrentSearchesGauge.Set(float64(atomic.LoadInt64(&cl.current)))
		return true
	default:
		return false
	}
}

// Release releases a slot
func (cl *ConcurrencyLimiter) Release() {
	<-cl.semaphore
	atomic.AddInt64(&cl.current, -1)
	concurrentSearchesGauge.Set(float64(atomic.LoadInt64(&cl.current)))
}

// Current returns the current number of active operations
func (cl *ConcurrencyLimiter) Current() int64 {
	return atomic.LoadInt64(&cl.current)
}

// =============================================================================
// RATE LIMITER (Token Bucket)
// =============================================================================

// TokenBucketRateLimiter implements token bucket rate limiting
type TokenBucketRateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketRateLimiter creates a new rate limiter
func NewTokenBucketRateLimiter(ratePerSecond, burst int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: float64(ratePerSecond),
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *TokenBucketRateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	// Check if we have tokens
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// AvailableTokens returns the number of available tokens
func (rl *TokenBucketRateLimiter) AvailableTokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// =============================================================================
// STRUCTURED LOGGING
// =============================================================================

// SearchLogEntry represents a structured log entry
type SearchLogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Component     string                 `json:"component"`
	Operation     string                 `json:"operation"`
	Query         string                 `json:"query,omitempty"`
	Source        string                 `json:"source,omitempty"`
	LatencyMs     float64                `json:"latency_ms,omitempty"`
	ResultCount   int                    `json:"result_count,omitempty"`
	CacheHit      bool                   `json:"cache_hit,omitempty"`
	Error         string                 `json:"error,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// StructuredLogger provides JSON structured logging
type StructuredLogger struct {
	component string
	enabled   bool
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(component string) *StructuredLogger {
	return &StructuredLogger{
		component: component,
		enabled:   true,
	}
}

// Log logs a structured entry
func (sl *StructuredLogger) Log(entry SearchLogEntry) {
	if !sl.enabled {
		return
	}

	entry.Timestamp = time.Now()
	entry.Component = sl.component

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("[%s] Failed to marshal log entry: %v", sl.component, err)
		return
	}

	log.Println(string(data))
}

// Info logs an info level entry
func (sl *StructuredLogger) Info(operation string, metadata map[string]interface{}) {
	sl.Log(SearchLogEntry{
		Level:     "INFO",
		Operation: operation,
		Metadata:  metadata,
	})
}

// Error logs an error level entry
func (sl *StructuredLogger) Error(operation string, err error, metadata map[string]interface{}) {
	sl.Log(SearchLogEntry{
		Level:     "ERROR",
		Operation: operation,
		Error:     err.Error(),
		Metadata:  metadata,
	})
}

// Warn logs a warning level entry
func (sl *StructuredLogger) Warn(operation string, metadata map[string]interface{}) {
	sl.Log(SearchLogEntry{
		Level:     "WARN",
		Operation: operation,
		Metadata:  metadata,
	})
}

// SlowQuery logs a slow query warning
func (sl *StructuredLogger) SlowQuery(query string, latencyMs float64, source string, threshold float64) {
	sl.Log(SearchLogEntry{
		Level:     "WARN",
		Operation: "slow_query",
		Query:     query,
		Source:    source,
		LatencyMs: latencyMs,
		Metadata: map[string]interface{}{
			"threshold_ms": threshold,
			"exceeded_by":  latencyMs - threshold,
		},
	})
}

// SearchComplete logs a completed search
func (sl *StructuredLogger) SearchComplete(query string, source string, latencyMs float64, resultCount int, cacheHit bool) {
	sl.Log(SearchLogEntry{
		Level:       "INFO",
		Operation:   "search_complete",
		Query:       query,
		Source:      source,
		LatencyMs:   latencyMs,
		ResultCount: resultCount,
		CacheHit:    cacheHit,
	})
}

// =============================================================================
// ENHANCED HEALTH CHECK
// =============================================================================

// EnhancedHealthStatus represents detailed health status
type EnhancedHealthStatus struct {
	Status          string                        `json:"status"` // healthy, degraded, unhealthy
	Timestamp       time.Time                     `json:"timestamp"`
	Components      map[string]SearchComponentHealth `json:"components"`
	Metrics         map[string]interface{}        `json:"metrics"`
	SystemResources SystemResources               `json:"system_resources"`
}

// SearchComponentHealth represents health of a single search component
type SearchComponentHealth struct {
	Status      string        `json:"status"` // healthy, degraded, unhealthy
	Latency     time.Duration `json:"latency_ms"`
	Message     string        `json:"message,omitempty"`
	LastChecked time.Time     `json:"last_checked"`
}

// SystemResources represents system resource usage
type SystemResources struct {
	MemoryUsedMB    uint64  `json:"memory_used_mb"`
	MemoryTotalMB   uint64  `json:"memory_total_mb"`
	MemoryPercent   float64 `json:"memory_percent"`
	GoroutineCount  int     `json:"goroutine_count"`
	CPUCount        int     `json:"cpu_count"`
}

// GetSystemResources returns current system resource usage
func GetSystemResources() SystemResources {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemResources{
		MemoryUsedMB:   m.Alloc / 1024 / 1024,
		MemoryTotalMB:  m.Sys / 1024 / 1024,
		MemoryPercent:  float64(m.Alloc) / float64(m.Sys) * 100,
		GoroutineCount: runtime.NumGoroutine(),
		CPUCount:       runtime.NumCPU(),
	}
}

// =============================================================================
// INDEX RECONCILIATION
// =============================================================================

// IndexReconciliationResult represents the result of index reconciliation
type IndexReconciliationResult struct {
	Timestamp        time.Time `json:"timestamp"`
	DBCount          int64     `json:"db_count"`
	IndexCount       int64     `json:"index_count"`
	Difference       int64     `json:"difference"`
	MissingInIndex   []uint64  `json:"missing_in_index,omitempty"`
	ExtraInIndex     []string  `json:"extra_in_index,omitempty"`
	ReconcileNeeded  bool      `json:"reconcile_needed"`
	ReconcileStarted bool      `json:"reconcile_started"`
}

// =============================================================================
// AUDIT LOG
// =============================================================================

// AuditLogEntry represents an audit log entry for admin operations
type AuditLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Operation   string                 `json:"operation"`
	UserID      string                 `json:"user_id"`
	UserIP      string                 `json:"user_ip"`
	Details     map[string]interface{} `json:"details"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"error,omitempty"`
}

// AuditLogger logs admin operations
type AuditLogger struct {
	entries []AuditLogEntry
	mu      sync.RWMutex
	maxSize int
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(maxSize int) *AuditLogger {
	return &AuditLogger{
		entries: make([]AuditLogEntry, 0),
		maxSize: maxSize,
	}
}

// Log logs an audit entry
func (al *AuditLogger) Log(entry AuditLogEntry) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry.Timestamp = time.Now()

	if len(al.entries) >= al.maxSize {
		// Remove oldest entries
		al.entries = al.entries[al.maxSize/10:]
	}

	al.entries = append(al.entries, entry)

	// Also log to structured logger
	data, _ := json.Marshal(entry)
	log.Printf("[AUDIT] %s", string(data))
}

// GetRecent returns recent audit entries
func (al *AuditLogger) GetRecent(count int) []AuditLogEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if count > len(al.entries) {
		count = len(al.entries)
	}

	result := make([]AuditLogEntry, count)
	copy(result, al.entries[len(al.entries)-count:])
	return result
}
