package task25

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// CascadeFailurePreventor prevents and isolates cascade failures
type CascadeFailurePreventor struct {
	isolationRules    map[string]IsolationRule
	circuitBreakers   map[string]*CircuitBreaker
	bulkheads         map[string]*Bulkhead
	rateLimiters      map[string]*RateLimiter
	isolatedComponents map[string]IsolatedComponent
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// IsolationRule defines how to isolate components during failures
type IsolationRule struct {
	ComponentID     string                 `json:"component_id"`
	TriggerType     string                 `json:"trigger_type"`
	IsolationMethod IsolationMethod       `json:"isolation_method"`
	Timeout         time.Duration         `json:"timeout"`
	Dependencies    []string              `json:"dependencies"`
	Action          func(string) error    `json:"-"`
	Rollback        func(string) error    `json:"-"`
}

// Bulkhead implements the bulkhead pattern for resource isolation
type Bulkhead struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	MaxConcurrency  int                   `json:"max_concurrency"`
	CurrentLoad     int                   `json:"current_load"`
	Queue           chan BulkheadRequest  `json:"-"`
	Timeout         time.Duration         `json:"timeout"`
	Enabled         bool                  `json:"enabled"`
	mu              sync.RWMutex
}

// BulkheadRequest represents a request in the bulkhead
type BulkheadRequest struct {
	ID       string
	Action   func() error
	Response chan BulkheadResponse
}

// BulkheadResponse represents the response from a bulkhead request
type BulkheadResponse struct {
	Error error
	Duration time.Duration
}

// RateLimit implements rate limiting for cascade prevention
type RateLimit struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	Window            time.Duration `json:"window"`
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	ID            string                 `json:"id"`
	Limit         RateLimit             `json:"limit"`
	tokens        int
	lastRefill    time.Time
	mu            sync.Mutex
}

// IsolatedComponent represents a component that has been isolated
type IsolatedComponent struct {
	ComponentID   string                 `json:"component_id"`
	IsolatedAt    time.Time             `json:"isolated_at"`
	Reason        string                 `json:"reason"`
	Method        IsolationMethod       `json:"method"`
	Dependencies  []string              `json:"dependencies"`
	AutoRecover   bool                  `json:"auto_recover"`
	RecoverAfter  time.Duration         `json:"recover_after"`
}

type IsolationMethod string

const (
	IsolationMethodCircuitBreaker IsolationMethod = "circuit_breaker"
	IsolationMethodBulkhead       IsolationMethod = "bulkhead"
	IsolationMethodRateLimit      IsolationMethod = "rate_limit"
	IsolationMethodComplete       IsolationMethod = "complete"
	IsolationMethodGraceful       IsolationMethod = "graceful"
)

// NewCascadeFailurePreventor creates a new cascade failure preventor
func NewCascadeFailurePreventor() *CascadeFailurePreventor {
	ctx, cancel := context.WithCancel(context.Background())
	
	cfp := &CascadeFailurePreventor{
		isolationRules:     make(map[string]IsolationRule),
		circuitBreakers:    make(map[string]*CircuitBreaker),
		bulkheads:          make(map[string]*Bulkhead),
		rateLimiters:       make(map[string]*RateLimiter),
		isolatedComponents: make(map[string]IsolatedComponent),
		ctx:                ctx,
		cancel:             cancel,
	}
	
	// Initialize default isolation rules and components
	cfp.initializeDefaultIsolationRules()
	cfp.initializeDefaultBulkheads()
	cfp.initializeDefaultRateLimiters()
	
	return cfp
}

// Start begins cascade failure prevention monitoring
func (cfp *CascadeFailurePreventor) Start(ctx context.Context) error {
	log.Println("Starting Cascade Failure Preventor...")
	
	// Start bulkhead processors
	for _, bulkhead := range cfp.bulkheads {
		go cfp.processBulkheadRequests(bulkhead)
	}
	
	// Start auto-recovery monitoring
	go cfp.autoRecoveryLoop()
	
	log.Println("Cascade Failure Preventor started")
	return nil
}

// Stop gracefully shuts down the cascade failure preventor
func (cfp *CascadeFailurePreventor) Stop() error {
	log.Println("Stopping Cascade Failure Preventor...")
	cfp.cancel()
	return nil
}

// ShouldPreventCascade determines if a failure should trigger cascade prevention
func (cfp *CascadeFailurePreventor) ShouldPreventCascade(failure FailureEvent) bool {
	cfp.mu.RLock()
	defer cfp.mu.RUnlock()
	
	// Check if component is already isolated
	if _, isolated := cfp.isolatedComponents[failure.Component]; isolated {
		return false // Already isolated
	}
	
	// Check failure severity and type
	switch failure.Severity {
	case SeverityCritical:
		return true // Always prevent cascade for critical failures
	case SeverityHigh:
		// Check if component has isolation rules
		_, hasRules := cfp.isolationRules[failure.Component]
		return hasRules
	default:
		return false // Don't prevent cascade for medium/low severity
	}
}

// IsolateFailure isolates a failed component to prevent cascade
func (cfp *CascadeFailurePreventor) IsolateFailure(failure FailureEvent) error {
	cfp.mu.Lock()
	defer cfp.mu.Unlock()
	
	log.Printf("Isolating component to prevent cascade: %s", failure.Component)
	
	// Find appropriate isolation rule
	rule, exists := cfp.isolationRules[failure.Component]
	if !exists {
		// Use default isolation method
		rule = IsolationRule{
			ComponentID:     failure.Component,
			TriggerType:     string(failure.Type),
			IsolationMethod: IsolationMethodCircuitBreaker,
			Timeout:         5 * time.Minute,
			Action:          cfp.defaultIsolationAction,
			Rollback:        cfp.defaultRollbackAction,
		}
	}
	
	// Execute isolation
	if err := rule.Action(failure.Component); err != nil {
		return fmt.Errorf("failed to isolate component %s: %w", failure.Component, err)
	}
	
	// Track isolated component
	cfp.isolatedComponents[failure.Component] = IsolatedComponent{
		ComponentID:  failure.Component,
		IsolatedAt:   time.Now(),
		Reason:       failure.Message,
		Method:       rule.IsolationMethod,
		Dependencies: rule.Dependencies,
		AutoRecover:  true,
		RecoverAfter: rule.Timeout,
	}
	
	// Isolate dependencies if needed
	for _, dep := range rule.Dependencies {
		if err := cfp.isolateDependency(dep, failure.Component); err != nil {
			log.Printf("Warning: failed to isolate dependency %s: %v", dep, err)
		}
	}
	
	log.Printf("Component isolated successfully: %s (Method: %s)", 
		failure.Component, rule.IsolationMethod)
	
	return nil
}

// ExecuteWithBulkhead executes an action within a bulkhead
func (cfp *CascadeFailurePreventor) ExecuteWithBulkhead(bulkheadID string, action func() error) error {
	cfp.mu.RLock()
	bulkhead, exists := cfp.bulkheads[bulkheadID]
	cfp.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("bulkhead not found: %s", bulkheadID)
	}
	
	if !bulkhead.Enabled {
		return fmt.Errorf("bulkhead disabled: %s", bulkheadID)
	}
	
	// Create request
	request := BulkheadRequest{
		ID:       fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Action:   action,
		Response: make(chan BulkheadResponse, 1),
	}
	
	// Submit to bulkhead queue
	select {
	case bulkhead.Queue <- request:
		// Wait for response
		select {
		case response := <-request.Response:
			return response.Error
		case <-time.After(bulkhead.Timeout):
			return fmt.Errorf("bulkhead request timeout: %s", bulkheadID)
		}
	case <-time.After(1 * time.Second):
		return fmt.Errorf("bulkhead queue full: %s", bulkheadID)
	}
}

// CheckRateLimit checks if a request is within rate limits
func (cfp *CascadeFailurePreventor) CheckRateLimit(limiterID string) bool {
	cfp.mu.RLock()
	limiter, exists := cfp.rateLimiters[limiterID]
	cfp.mu.RUnlock()
	
	if !exists {
		return true // No rate limit configured
	}
	
	return limiter.Allow()
}

// processBulkheadRequests processes requests for a bulkhead
func (cfp *CascadeFailurePreventor) processBulkheadRequests(bulkhead *Bulkhead) {
	for {
		select {
		case <-cfp.ctx.Done():
			return
		case request := <-bulkhead.Queue:
			cfp.processBulkheadRequest(bulkhead, request)
		}
	}
}

// processBulkheadRequest processes a single bulkhead request
func (cfp *CascadeFailurePreventor) processBulkheadRequest(bulkhead *Bulkhead, request BulkheadRequest) {
	// Check if we can accept the request
	bulkhead.mu.Lock()
	if bulkhead.CurrentLoad >= bulkhead.MaxConcurrency {
		bulkhead.mu.Unlock()
		request.Response <- BulkheadResponse{
			Error: fmt.Errorf("bulkhead at capacity: %s", bulkhead.ID),
		}
		return
	}
	bulkhead.CurrentLoad++
	bulkhead.mu.Unlock()
	
	// Execute the request
	start := time.Now()
	err := request.Action()
	duration := time.Since(start)
	
	// Update load
	bulkhead.mu.Lock()
	bulkhead.CurrentLoad--
	bulkhead.mu.Unlock()
	
	// Send response
	request.Response <- BulkheadResponse{
		Error:    err,
		Duration: duration,
	}
}

// autoRecoveryLoop monitors for components that can be auto-recovered
func (cfp *CascadeFailurePreventor) autoRecoveryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-cfp.ctx.Done():
			return
		case <-ticker.C:
			cfp.checkAutoRecovery()
		}
	}
}

// checkAutoRecovery checks if any isolated components can be recovered
func (cfp *CascadeFailurePreventor) checkAutoRecovery() {
	cfp.mu.Lock()
	defer cfp.mu.Unlock()
	
	for componentID, isolated := range cfp.isolatedComponents {
		if isolated.AutoRecover && time.Since(isolated.IsolatedAt) >= isolated.RecoverAfter {
			log.Printf("Auto-recovering isolated component: %s", componentID)
			
			if err := cfp.recoverComponent(componentID); err != nil {
				log.Printf("Auto-recovery failed for component %s: %v", componentID, err)
			} else {
				delete(cfp.isolatedComponents, componentID)
				log.Printf("Component auto-recovered successfully: %s", componentID)
			}
		}
	}
}

// Allow implements token bucket rate limiting
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Refill tokens based on time elapsed
	if rl.lastRefill.IsZero() {
		rl.lastRefill = now
		rl.tokens = rl.Limit.BurstSize
	} else {
		elapsed := now.Sub(rl.lastRefill)
		tokensToAdd := int(elapsed.Seconds()) * rl.Limit.RequestsPerSecond
		rl.tokens = min(rl.tokens+tokensToAdd, rl.Limit.BurstSize)
		rl.lastRefill = now
	}
	
	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// initializeDefaultIsolationRules sets up default isolation rules
func (cfp *CascadeFailurePreventor) initializeDefaultIsolationRules() {
	// Database isolation rule
	cfp.isolationRules["database"] = IsolationRule{
		ComponentID:     "database",
		TriggerType:     "connection_failure",
		IsolationMethod: IsolationMethodCircuitBreaker,
		Timeout:         10 * time.Minute,
		Dependencies:    []string{"cache", "api"},
		Action:          cfp.isolateDatabaseComponent,
		Rollback:        cfp.recoverDatabaseComponent,
	}
	
	// Cache isolation rule
	cfp.isolationRules["cache"] = IsolationRule{
		ComponentID:     "cache",
		TriggerType:     "service_failure",
		IsolationMethod: IsolationMethodBulkhead,
		Timeout:         5 * time.Minute,
		Dependencies:    []string{},
		Action:          cfp.isolateCacheComponent,
		Rollback:        cfp.recoverCacheComponent,
	}
	
	// API isolation rule
	cfp.isolationRules["api"] = IsolationRule{
		ComponentID:     "api",
		TriggerType:     "high_error_rate",
		IsolationMethod: IsolationMethodRateLimit,
		Timeout:         3 * time.Minute,
		Dependencies:    []string{},
		Action:          cfp.isolateAPIComponent,
		Rollback:        cfp.recoverAPIComponent,
	}
}

// initializeDefaultBulkheads sets up default bulkheads
func (cfp *CascadeFailurePreventor) initializeDefaultBulkheads() {
	// Database bulkhead
	cfp.bulkheads["database"] = &Bulkhead{
		ID:             "database",
		Name:           "Database Operations Bulkhead",
		MaxConcurrency: 50,
		CurrentLoad:    0,
		Queue:          make(chan BulkheadRequest, 100),
		Timeout:        30 * time.Second,
		Enabled:        true,
	}
	
	// Cache bulkhead
	cfp.bulkheads["cache"] = &Bulkhead{
		ID:             "cache",
		Name:           "Cache Operations Bulkhead",
		MaxConcurrency: 100,
		CurrentLoad:    0,
		Queue:          make(chan BulkheadRequest, 200),
		Timeout:        10 * time.Second,
		Enabled:        true,
	}
	
	// External API bulkhead
	cfp.bulkheads["external_api"] = &Bulkhead{
		ID:             "external_api",
		Name:           "External API Calls Bulkhead",
		MaxConcurrency: 20,
		CurrentLoad:    0,
		Queue:          make(chan BulkheadRequest, 50),
		Timeout:        60 * time.Second,
		Enabled:        true,
	}
}

// initializeDefaultRateLimiters sets up default rate limiters
func (cfp *CascadeFailurePreventor) initializeDefaultRateLimiters() {
	// API rate limiter
	cfp.rateLimiters["api"] = &RateLimiter{
		ID: "api",
		Limit: RateLimit{
			RequestsPerSecond: 100,
			BurstSize:         200,
			Window:            1 * time.Second,
		},
		tokens:     200,
		lastRefill: time.Now(),
	}
	
	// Database rate limiter
	cfp.rateLimiters["database"] = &RateLimiter{
		ID: "database",
		Limit: RateLimit{
			RequestsPerSecond: 50,
			BurstSize:         100,
			Window:            1 * time.Second,
		},
		tokens:     100,
		lastRefill: time.Now(),
	}
}

// Component isolation implementations
func (cfp *CascadeFailurePreventor) isolateDatabaseComponent(componentID string) error {
	log.Printf("Isolating database component: %s", componentID)
	// This would implement actual database isolation
	return nil
}

func (cfp *CascadeFailurePreventor) isolateCacheComponent(componentID string) error {
	log.Printf("Isolating cache component: %s", componentID)
	// This would implement actual cache isolation
	return nil
}

func (cfp *CascadeFailurePreventor) isolateAPIComponent(componentID string) error {
	log.Printf("Isolating API component: %s", componentID)
	// This would implement actual API isolation
	return nil
}

// Component recovery implementations
func (cfp *CascadeFailurePreventor) recoverDatabaseComponent(componentID string) error {
	log.Printf("Recovering database component: %s", componentID)
	// This would implement actual database recovery
	return nil
}

func (cfp *CascadeFailurePreventor) recoverCacheComponent(componentID string) error {
	log.Printf("Recovering cache component: %s", componentID)
	// This would implement actual cache recovery
	return nil
}

func (cfp *CascadeFailurePreventor) recoverAPIComponent(componentID string) error {
	log.Printf("Recovering API component: %s", componentID)
	// This would implement actual API recovery
	return nil
}

// Helper methods
func (cfp *CascadeFailurePreventor) defaultIsolationAction(componentID string) error {
	log.Printf("Applying default isolation to component: %s", componentID)
	return nil
}

func (cfp *CascadeFailurePreventor) defaultRollbackAction(componentID string) error {
	log.Printf("Applying default rollback to component: %s", componentID)
	return nil
}

func (cfp *CascadeFailurePreventor) isolateDependency(dependency, reason string) error {
	log.Printf("Isolating dependency %s due to %s", dependency, reason)
	return nil
}

func (cfp *CascadeFailurePreventor) recoverComponent(componentID string) error {
	log.Printf("Recovering component: %s", componentID)
	
	// Find and execute rollback action
	if rule, exists := cfp.isolationRules[componentID]; exists && rule.Rollback != nil {
		return rule.Rollback(componentID)
	}
	
	// Use default recovery
	return cfp.defaultRollbackAction(componentID)
}

// GetIsolationStatus returns the current isolation status
func (cfp *CascadeFailurePreventor) GetIsolationStatus() IsolationStatus {
	cfp.mu.RLock()
	defer cfp.mu.RUnlock()
	
	return IsolationStatus{
		IsolatedComponents: len(cfp.isolatedComponents),
		ActiveBulkheads:    len(cfp.bulkheads),
		ActiveRateLimiters: len(cfp.rateLimiters),
	}
}

type IsolationStatus struct {
	IsolatedComponents int `json:"isolated_components"`
	ActiveBulkheads    int `json:"active_bulkheads"`
	ActiveRateLimiters int `json:"active_rate_limiters"`
}