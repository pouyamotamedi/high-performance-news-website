package api

import (
	"sync"
	"time"
)

// RateLimiter provides simple in-memory rate limiting
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientInfo
}

type clientInfo struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientInfo),
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// Allow checks if a request should be allowed based on rate limiting
func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	client, exists := rl.clients[key]
	
	if !exists || now.After(client.resetTime) {
		// New client or window has expired
		rl.clients[key] = &clientInfo{
			count:     1,
			resetTime: now.Add(window),
		}
		return true
	}
	
	if client.count >= limit {
		return false
	}
	
	client.count++
	return true
}

// cleanup removes expired entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, client := range rl.clients {
			if now.After(client.resetTime) {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// GetRemainingRequests returns the number of remaining requests for a key
func (rl *RateLimiter) GetRemainingRequests(key string, limit int) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	client, exists := rl.clients[key]
	if !exists {
		return limit
	}
	
	remaining := limit - client.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetResetTime returns when the rate limit will reset for a key
func (rl *RateLimiter) GetResetTime(key string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	client, exists := rl.clients[key]
	if !exists {
		return time.Now()
	}
	
	return client.resetTime
}