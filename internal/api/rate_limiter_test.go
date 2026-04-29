package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter()
	
	tests := []struct {
		name     string
		key      string
		limit    int
		window   time.Duration
		requests int
		expected []bool
	}{
		{
			name:     "Within limit",
			key:      "user1",
			limit:    5,
			window:   time.Minute,
			requests: 3,
			expected: []bool{true, true, true},
		},
		{
			name:     "Exceeds limit",
			key:      "user2",
			limit:    2,
			window:   time.Minute,
			requests: 4,
			expected: []bool{true, true, false, false},
		},
		{
			name:     "Different keys",
			key:      "user3",
			limit:    1,
			window:   time.Minute,
			requests: 1,
			expected: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.requests; i++ {
				result := rl.Allow(tt.key, tt.limit, tt.window)
				assert.Equal(t, tt.expected[i], result, "Request %d should be %v", i+1, tt.expected[i])
			}
		})
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter()
	key := "test_user"
	limit := 2
	window := 100 * time.Millisecond

	// Use up the limit
	assert.True(t, rl.Allow(key, limit, window))
	assert.True(t, rl.Allow(key, limit, window))
	assert.False(t, rl.Allow(key, limit, window))

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	assert.True(t, rl.Allow(key, limit, window))
}

func TestRateLimiter_GetRemainingRequests(t *testing.T) {
	rl := NewRateLimiter()
	key := "test_user"
	limit := 5

	// Initially should have full limit
	remaining := rl.GetRemainingRequests(key, limit)
	assert.Equal(t, limit, remaining)

	// After one request
	rl.Allow(key, limit, time.Minute)
	remaining = rl.GetRemainingRequests(key, limit)
	assert.Equal(t, limit-1, remaining)

	// After hitting limit
	for i := 0; i < limit; i++ {
		rl.Allow(key, limit, time.Minute)
	}
	remaining = rl.GetRemainingRequests(key, limit)
	assert.Equal(t, 0, remaining)
}

func TestRateLimiter_GetResetTime(t *testing.T) {
	rl := NewRateLimiter()
	key := "test_user"
	limit := 1
	window := time.Hour

	before := time.Now()
	rl.Allow(key, limit, window)
	resetTime := rl.GetResetTime(key)
	after := time.Now().Add(window)

	assert.True(t, resetTime.After(before))
	assert.True(t, resetTime.Before(after))
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter()
	key := "concurrent_user"
	limit := 10
	window := time.Minute

	// Test concurrent access
	results := make(chan bool, 20)
	
	for i := 0; i < 20; i++ {
		go func() {
			results <- rl.Allow(key, limit, window)
		}()
	}

	allowed := 0
	denied := 0
	
	for i := 0; i < 20; i++ {
		if <-results {
			allowed++
		} else {
			denied++
		}
	}

	assert.Equal(t, limit, allowed)
	assert.Equal(t, 10, denied)
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	rl := NewRateLimiter()
	limit := 2
	window := time.Minute

	// Each key should have its own limit
	assert.True(t, rl.Allow("user1", limit, window))
	assert.True(t, rl.Allow("user2", limit, window))
	assert.True(t, rl.Allow("user1", limit, window))
	assert.True(t, rl.Allow("user2", limit, window))
	
	// Both should be at limit now
	assert.False(t, rl.Allow("user1", limit, window))
	assert.False(t, rl.Allow("user2", limit, window))
}