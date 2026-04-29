package models

import (
	"time"
)

// CDNConfig represents CDN configuration settings
type CDNConfig struct {
	ID          uint64    `json:"id" db:"id"`
	Provider    string    `json:"provider" db:"provider"` // cloudflare, aws, etc.
	APIKey      string    `json:"api_key" db:"api_key"`
	APISecret   string    `json:"api_secret" db:"api_secret"`
	ZoneID      string    `json:"zone_id" db:"zone_id"`
	Domain      string    `json:"domain" db:"domain"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CDNPurgeRequest represents a cache purge request
type CDNPurgeRequest struct {
	URLs     []string `json:"urls,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Hosts    []string `json:"hosts,omitempty"`
	PurgeAll bool     `json:"purge_all,omitempty"`
}

// CDNPurgeResponse represents the response from a purge request
type CDNPurgeResponse struct {
	Success   bool      `json:"success"`
	RequestID string    `json:"request_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// CDNStats represents CDN performance statistics
type CDNStats struct {
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
	BandwidthSaved   int64   `json:"bandwidth_saved"`
	RequestsServed   int64   `json:"requests_served"`
	OriginRequests   int64   `json:"origin_requests"`
	ResponseTime     int64   `json:"response_time_ms"`
	LastUpdated      time.Time `json:"last_updated"`
}

// CDNHealthCheck represents CDN health status
type CDNHealthCheck struct {
	Provider    string    `json:"provider"`
	Status      string    `json:"status"` // healthy, degraded, down
	ResponseTime int64    `json:"response_time_ms"`
	LastCheck   time.Time `json:"last_check"`
	ErrorCount  int       `json:"error_count"`
	Message     string    `json:"message,omitempty"`
}