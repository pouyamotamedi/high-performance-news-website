package testing

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

// RetryManager provides intelligent retry mechanisms with exponential backoff
type RetryManager struct {
	defaultMaxRetries int
	defaultBaseDelay  time.Duration
	defaultMaxDelay   time.Duration
	jitterEnabled     bool
}

// RetryConfig defines retry behavior for specific operations
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	BaseDelay     time.Duration `json:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	Multiplier    float64       `json:"multiplier"`
	JitterEnabled bool          `json:"jitter_enabled"`
	RetryableErrors []string    `json:"retryable_errors"`
}

// RetryResult contains the result of a retry operation
type RetryResult struct {
	Success       bool          `json:"success"`
	Attempts      int           `json:"attempts"`
	TotalDuration time.Duration `json:"total_duration"`
	LastError     error         `json:"last_error,omitempty"`
	RetryDelays   []time.Duration `json:"retry_delays"`
}

// RetryableOperation defines an operation that can be retried
type RetryableOperation func(context.Context, FailureEvent) error

// NewRetryManager creates a new retry manager with default settings
func NewRetryManager() *RetryManager {
	return &RetryManager{
		defaultMaxRetries: 3,
		defaultBaseDelay:  1 * time.Second,
		defaultMaxDelay:   30 * time.Second,
		jitterEnabled:     true,
	}
}

// ExecuteWithRetry executes an operation with retry logic
func (rm *RetryManager) ExecuteWithRetry(
	operation RetryableOperation,
	failure FailureEvent,
	maxRetries int,
	timeout time.Duration,
) error {
	config := RetryConfig{
		MaxRetries:    maxRetries,
		BaseDelay:     rm.defaultBaseDelay,
		MaxDelay:      rm.defaultMaxDelay,
		Multiplier:    2.0,
		JitterEnabled: rm.jitterEnabled,
	}
	
	result := rm.ExecuteWithConfig(operation, failure, config, timeout)
	return result.LastError
}

// ExecuteWithConfig executes an operation with custom retry configuration
func (rm *RetryManager) ExecuteWithConfig(
	operation RetryableOperation,
	failure FailureEvent,
	config RetryConfig,
	timeout time.Duration,
) RetryResult {
	start := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	result := RetryResult{
		Success:     false,
		Attempts:    0,
		RetryDelays: make([]time.Duration, 0),
	}
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result.Attempts++
		
		log.Printf("Retry attempt %d/%d for failure %s", 
			attempt+1, config.MaxRetries+1, failure.ID)
		
		// Execute the operation
		err := operation(ctx, failure)
		
		if err == nil {
			result.Success = true
			result.TotalDuration = time.Since(start)
			log.Printf("Operation succeeded on attempt %d for failure %s", 
				attempt+1, failure.ID)
			return result
		}
		
		result.LastError = err
		
		// Check if we should retry this error
		if !rm.shouldRetry(err, config) {
			log.Printf("Error not retryable for failure %s: %v", failure.ID, err)
			break
		}
		
		// Don't delay after the last attempt
		if attempt < config.MaxRetries {
			delay := rm.calculateDelay(attempt, config)
			result.RetryDelays = append(result.RetryDelays, delay)
			
			log.Printf("Retrying in %v for failure %s (attempt %d)", 
				delay, failure.ID, attempt+1)
			
			// Wait with context cancellation support
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				result.LastError = fmt.Errorf("retry timeout: %w", ctx.Err())
				result.TotalDuration = time.Since(start)
				return result
			}
		}
	}
	
	result.TotalDuration = time.Since(start)
	log.Printf("All retry attempts failed for failure %s: %v", 
		failure.ID, result.LastError)
	
	return result
}

// calculateDelay calculates the delay for the next retry attempt
func (rm *RetryManager) calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt))
	
	// Apply maximum delay limit
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter to prevent thundering herd
	if config.JitterEnabled {
		jitter := rand.Float64() * 0.1 * delay // ±10% jitter
		delay += jitter - (0.05 * delay)
	}
	
	return time.Duration(delay)
}

// shouldRetry determines if an error should be retried
func (rm *RetryManager) shouldRetry(err error, config RetryConfig) bool {
	if err == nil {
		return false
	}
	
	errorMsg := err.Error()
	
	// Check against retryable error patterns
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"connection reset",
		"no such host",
		"context deadline exceeded",
	}
	
	// Add custom retryable errors from config
	retryablePatterns = append(retryablePatterns, config.RetryableErrors...)
	
	for _, pattern := range retryablePatterns {
		if contains(errorMsg, pattern) {
			return true
		}
	}
	
	// Don't retry certain types of errors
	nonRetryablePatterns := []string{
		"authentication failed",
		"permission denied",
		"not found",
		"invalid argument",
		"bad request",
	}
	
	for _, pattern := range nonRetryablePatterns {
		if contains(errorMsg, pattern) {
			return false
		}
	}
	
	// Default to retrying unknown errors
	return true
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

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryWithCircuitBreaker implements circuit breaker pattern with retry
func (rm *RetryManager) RetryWithCircuitBreaker(
	operation RetryableOperation,
	failure FailureEvent,
	config RetryConfig,
	circuitBreaker *CircuitBreaker,
) RetryResult {
	// Check circuit breaker state
	if !circuitBreaker.CanExecute() {
		return RetryResult{
			Success:   false,
			Attempts:  0,
			LastError: fmt.Errorf("circuit breaker is open"),
		}
	}
	
	result := rm.ExecuteWithConfig(operation, failure, config, 30*time.Second)
	
	// Update circuit breaker based on result
	if result.Success {
		circuitBreaker.RecordSuccess()
	} else {
		circuitBreaker.RecordFailure()
	}
	
	return result
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state          CircuitBreakerState
}

type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitBreakerClosed,
	}
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.failureCount = 0
	cb.state = CircuitBreakerClosed
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	if cb.failureCount >= cb.maxFailures {
		cb.state = CircuitBreakerOpen
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

// AdaptiveRetryManager provides adaptive retry behavior based on historical data
type AdaptiveRetryManager struct {
	*RetryManager
	successRates map[string]float64
	avgDelays    map[string]time.Duration
}

// NewAdaptiveRetryManager creates a new adaptive retry manager
func NewAdaptiveRetryManager() *AdaptiveRetryManager {
	return &AdaptiveRetryManager{
		RetryManager: NewRetryManager(),
		successRates: make(map[string]float64),
		avgDelays:    make(map[string]time.Duration),
	}
}

// ExecuteWithAdaptiveConfig executes with configuration adapted based on historical data
func (arm *AdaptiveRetryManager) ExecuteWithAdaptiveConfig(
	operation RetryableOperation,
	failure FailureEvent,
	timeout time.Duration,
) RetryResult {
	// Get adaptive configuration based on failure type
	config := arm.getAdaptiveConfig(string(failure.Type))
	
	result := arm.ExecuteWithConfig(operation, failure, config, timeout)
	
	// Update historical data
	arm.updateHistoricalData(string(failure.Type), result)
	
	return result
}

// getAdaptiveConfig returns configuration adapted for the failure type
func (arm *AdaptiveRetryManager) getAdaptiveConfig(failureType string) RetryConfig {
	config := RetryConfig{
		MaxRetries:    arm.defaultMaxRetries,
		BaseDelay:     arm.defaultBaseDelay,
		MaxDelay:      arm.defaultMaxDelay,
		Multiplier:    2.0,
		JitterEnabled: arm.jitterEnabled,
	}
	
	// Adjust based on historical success rate
	if successRate, exists := arm.successRates[failureType]; exists {
		if successRate < 0.5 {
			// Low success rate - increase retries and delays
			config.MaxRetries = int(float64(config.MaxRetries) * 1.5)
			config.BaseDelay = time.Duration(float64(config.BaseDelay) * 1.5)
		} else if successRate > 0.8 {
			// High success rate - reduce retries and delays
			config.MaxRetries = int(float64(config.MaxRetries) * 0.8)
			config.BaseDelay = time.Duration(float64(config.BaseDelay) * 0.8)
		}
	}
	
	return config
}

// updateHistoricalData updates success rates and timing data
func (arm *AdaptiveRetryManager) updateHistoricalData(failureType string, result RetryResult) {
	// Update success rate (simple moving average)
	currentRate := arm.successRates[failureType]
	if result.Success {
		arm.successRates[failureType] = currentRate*0.9 + 0.1
	} else {
		arm.successRates[failureType] = currentRate * 0.9
	}
	
	// Update average delay
	if len(result.RetryDelays) > 0 {
		totalDelay := time.Duration(0)
		for _, delay := range result.RetryDelays {
			totalDelay += delay
		}
		avgDelay := totalDelay / time.Duration(len(result.RetryDelays))
		
		currentAvg := arm.avgDelays[failureType]
		arm.avgDelays[failureType] = time.Duration(
			float64(currentAvg)*0.9 + float64(avgDelay)*0.1,
		)
	}
}