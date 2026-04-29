package failover

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// FailoverHandler manages integration failures and fallback mechanisms
type FailoverHandler struct {
	integrations map[string]*IntegrationHealth
	fallbacks    map[string][]string // integration -> fallback integrations
	circuitBreakers map[string]*CircuitBreaker
	retryPolicies map[string]*RetryPolicy
	mu           sync.RWMutex
}

// IntegrationHealth tracks the health of an integration
type IntegrationHealth struct {
	Name              string        `json:"name"`
	Status            HealthStatus  `json:"status"`
	LastSuccess       time.Time     `json:"last_success"`
	LastFailure       time.Time     `json:"last_failure"`
	ConsecutiveFailures int         `json:"consecutive_failures"`
	TotalFailures     int64         `json:"total_failures"`
	TotalRequests     int64         `json:"total_requests"`
	SuccessRate       float64       `json:"success_rate"`
	ResponseTime      time.Duration `json:"response_time"`
	LastHealthCheck   time.Time     `json:"last_health_check"`
}

// HealthStatus represents the health status of an integration
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	Name                string        `json:"name"`
	State               BreakerState  `json:"state"`
	FailureThreshold    int           `json:"failure_threshold"`
	RecoveryTimeout     time.Duration `json:"recovery_timeout"`
	LastFailureTime     time.Time     `json:"last_failure_time"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	LastStateChange     time.Time     `json:"last_state_change"`
}

// BreakerState represents the state of a circuit breaker
type BreakerState string

const (
	BreakerStateClosed   BreakerState = "closed"   // Normal operation
	BreakerStateOpen     BreakerState = "open"     // Failing fast
	BreakerStateHalfOpen BreakerState = "half_open" // Testing recovery
)

// RetryPolicy defines retry behavior for failed integrations
type RetryPolicy struct {
	MaxRetries        int           `json:"max_retries"`
	InitialDelay      time.Duration `json:"initial_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	RetryableErrors   []string      `json:"retryable_errors"`
}

// FailoverResult represents the result of a failover operation
type FailoverResult struct {
	Success           bool          `json:"success"`
	PrimaryFailed     bool          `json:"primary_failed"`
	FallbackUsed      string        `json:"fallback_used"`
	AttemptedFallbacks []string     `json:"attempted_fallbacks"`
	TotalAttempts     int           `json:"total_attempts"`
	Duration          time.Duration `json:"duration"`
	Error             string        `json:"error,omitempty"`
}

// NewFailoverHandler creates a new failover handler
func NewFailoverHandler() *FailoverHandler {
	return &FailoverHandler{
		integrations:    make(map[string]*IntegrationHealth),
		fallbacks:       make(map[string][]string),
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryPolicies:   make(map[string]*RetryPolicy),
	}
}

// RegisterIntegration registers an integration for failover handling
func (fh *FailoverHandler) RegisterIntegration(name string, fallbacks []string) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.integrations[name] = &IntegrationHealth{
		Name:            name,
		Status:          HealthStatusUnknown,
		LastHealthCheck: time.Now(),
	}

	if len(fallbacks) > 0 {
		fh.fallbacks[name] = fallbacks
	}

	// Create circuit breaker with default settings
	fh.circuitBreakers[name] = &CircuitBreaker{
		Name:             name,
		State:            BreakerStateClosed,
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		LastStateChange:  time.Now(),
	}

	// Create default retry policy
	fh.retryPolicies[name] = &RetryPolicy{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors:   []string{"timeout", "connection_error", "temporary_failure"},
	}

	log.Printf("Registered integration for failover: %s with fallbacks: %v", name, fallbacks)
}

// ExecuteWithFailover executes an operation with failover support
func (fh *FailoverHandler) ExecuteWithFailover(ctx context.Context, integrationName string, operation func(string) error) *FailoverResult {
	start := time.Now()
	result := &FailoverResult{
		AttemptedFallbacks: []string{},
	}

	// Try primary integration first
	if fh.canExecute(integrationName) {
		err := fh.executeWithRetry(ctx, integrationName, operation)
		fh.recordResult(integrationName, err)
		
		if err == nil {
			result.Success = true
			result.Duration = time.Since(start)
			return result
		}
		
		result.PrimaryFailed = true
		result.TotalAttempts++
		log.Printf("Primary integration %s failed: %v", integrationName, err)
	} else {
		result.PrimaryFailed = true
		log.Printf("Primary integration %s is unavailable (circuit breaker open)", integrationName)
	}

	// Try fallback integrations
	fh.mu.RLock()
	fallbacks := fh.fallbacks[integrationName]
	fh.mu.RUnlock()

	for _, fallback := range fallbacks {
		if !fh.canExecute(fallback) {
			log.Printf("Fallback integration %s is unavailable", fallback)
			continue
		}

		result.AttemptedFallbacks = append(result.AttemptedFallbacks, fallback)
		result.TotalAttempts++

		err := fh.executeWithRetry(ctx, fallback, operation)
		fh.recordResult(fallback, err)

		if err == nil {
			result.Success = true
			result.FallbackUsed = fallback
			result.Duration = time.Since(start)
			log.Printf("Successfully failed over to %s", fallback)
			return result
		}

		log.Printf("Fallback integration %s failed: %v", fallback, err)
	}

	// All integrations failed
	result.Duration = time.Since(start)
	result.Error = "all integrations failed"
	log.Printf("All integrations failed for %s", integrationName)

	return result
}

// canExecute checks if an integration can execute (circuit breaker check)
func (fh *FailoverHandler) canExecute(integrationName string) bool {
	fh.mu.RLock()
	breaker, exists := fh.circuitBreakers[integrationName]
	fh.mu.RUnlock()

	if !exists {
		return true // No circuit breaker, allow execution
	}

	switch breaker.State {
	case BreakerStateClosed:
		return true
	case BreakerStateOpen:
		// Check if recovery timeout has passed
		if time.Since(breaker.LastFailureTime) > breaker.RecoveryTimeout {
			fh.mu.Lock()
			breaker.State = BreakerStateHalfOpen
			breaker.LastStateChange = time.Now()
			fh.mu.Unlock()
			log.Printf("Circuit breaker for %s moved to half-open state", integrationName)
			return true
		}
		return false
	case BreakerStateHalfOpen:
		return true
	default:
		return false
	}
}

// executeWithRetry executes an operation with retry logic
func (fh *FailoverHandler) executeWithRetry(ctx context.Context, integrationName string, operation func(string) error) error {
	fh.mu.RLock()
	retryPolicy := fh.retryPolicies[integrationName]
	fh.mu.RUnlock()

	if retryPolicy == nil {
		return operation(integrationName)
	}

	var lastErr error
	delay := retryPolicy.InitialDelay

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * retryPolicy.BackoffMultiplier)
			if delay > retryPolicy.MaxDelay {
				delay = retryPolicy.MaxDelay
			}
		}

		err := operation(integrationName)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !fh.isRetryableError(err, retryPolicy) {
			break
		}

		log.Printf("Attempt %d failed for %s: %v, retrying in %v", attempt+1, integrationName, err, delay)
	}

	return lastErr
}

// isRetryableError checks if an error is retryable
func (fh *FailoverHandler) isRetryableError(err error, policy *RetryPolicy) bool {
	errorStr := err.Error()
	for _, retryableError := range policy.RetryableErrors {
		if contains(errorStr, retryableError) {
			return true
		}
	}
	return false
}

// recordResult records the result of an integration operation
func (fh *FailoverHandler) recordResult(integrationName string, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	health := fh.integrations[integrationName]
	breaker := fh.circuitBreakers[integrationName]

	if health == nil || breaker == nil {
		return
	}

	health.TotalRequests++

	if err == nil {
		// Success
		health.LastSuccess = time.Now()
		health.ConsecutiveFailures = 0
		
		// Update circuit breaker
		if breaker.State == BreakerStateHalfOpen {
			breaker.State = BreakerStateClosed
			breaker.LastStateChange = time.Now()
			log.Printf("Circuit breaker for %s closed (recovered)", integrationName)
		}
		breaker.ConsecutiveFailures = 0
		
		// Update health status
		health.Status = HealthStatusHealthy
	} else {
		// Failure
		health.LastFailure = time.Now()
		health.ConsecutiveFailures++
		health.TotalFailures++
		
		// Update circuit breaker
		breaker.ConsecutiveFailures++
		breaker.LastFailureTime = time.Now()
		
		if breaker.State == BreakerStateClosed && breaker.ConsecutiveFailures >= breaker.FailureThreshold {
			breaker.State = BreakerStateOpen
			breaker.LastStateChange = time.Now()
			log.Printf("Circuit breaker for %s opened due to %d consecutive failures", integrationName, breaker.ConsecutiveFailures)
		} else if breaker.State == BreakerStateHalfOpen {
			breaker.State = BreakerStateOpen
			breaker.LastStateChange = time.Now()
			log.Printf("Circuit breaker for %s reopened after failed recovery attempt", integrationName)
		}
		
		// Update health status
		if health.ConsecutiveFailures >= 3 {
			health.Status = HealthStatusUnhealthy
		} else {
			health.Status = HealthStatusDegraded
		}
	}

	// Update success rate
	if health.TotalRequests > 0 {
		successCount := health.TotalRequests - health.TotalFailures
		health.SuccessRate = float64(successCount) / float64(health.TotalRequests) * 100
	}

	health.LastHealthCheck = time.Now()
}

// GetIntegrationHealth returns the health status of an integration
func (fh *FailoverHandler) GetIntegrationHealth(integrationName string) *IntegrationHealth {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	if health, exists := fh.integrations[integrationName]; exists {
		// Return a copy to avoid race conditions
		healthCopy := *health
		return &healthCopy
	}

	return nil
}

// GetAllIntegrationHealth returns the health status of all integrations
func (fh *FailoverHandler) GetAllIntegrationHealth() map[string]*IntegrationHealth {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	result := make(map[string]*IntegrationHealth)
	for name, health := range fh.integrations {
		healthCopy := *health
		result[name] = &healthCopy
	}

	return result
}

// GetCircuitBreakerStatus returns the status of all circuit breakers
func (fh *FailoverHandler) GetCircuitBreakerStatus() map[string]*CircuitBreaker {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for name, breaker := range fh.circuitBreakers {
		breakerCopy := *breaker
		result[name] = &breakerCopy
	}

	return result
}

// ResetCircuitBreaker manually resets a circuit breaker
func (fh *FailoverHandler) ResetCircuitBreaker(integrationName string) error {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	breaker, exists := fh.circuitBreakers[integrationName]
	if !exists {
		return fmt.Errorf("circuit breaker for %s not found", integrationName)
	}

	breaker.State = BreakerStateClosed
	breaker.ConsecutiveFailures = 0
	breaker.LastStateChange = time.Now()

	log.Printf("Circuit breaker for %s manually reset", integrationName)
	return nil
}

// UpdateRetryPolicy updates the retry policy for an integration
func (fh *FailoverHandler) UpdateRetryPolicy(integrationName string, policy *RetryPolicy) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.retryPolicies[integrationName] = policy
	log.Printf("Updated retry policy for %s", integrationName)
}

// Helper function
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}