package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"high-performance-news-website/internal/models"
)

// CloudflareCDNService implements CDN operations using Cloudflare API
type CloudflareCDNService struct {
	config       *models.CDNConfig
	httpClient   *http.Client
	failoverMode bool
	mutex        sync.RWMutex
	stats        *models.CDNStats
	lastHealthCheck time.Time
}

// NewCloudflareCDNService creates a new Cloudflare CDN service instance
func NewCloudflareCDNService(config *models.CDNConfig) *CloudflareCDNService {
	return &CloudflareCDNService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		failoverMode: false,
		stats: &models.CDNStats{
			LastUpdated: time.Now(),
		},
	}
}

// GetConfig returns the current CDN configuration
func (c *CloudflareCDNService) GetConfig() (*models.CDNConfig, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if c.config == nil {
		return nil, fmt.Errorf("CDN configuration not found")
	}
	
	// Return a copy to prevent external modifications
	configCopy := *c.config
	return &configCopy, nil
}

// UpdateConfig updates the CDN configuration
func (c *CloudflareCDNService) UpdateConfig(config *models.CDNConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	// Validate required fields
	if config.APIKey == "" || config.ZoneID == "" {
		return fmt.Errorf("API key and zone ID are required")
	}
	
	c.config = config
	c.config.UpdatedAt = time.Now()
	
	return nil
}

// TestConnection tests the connection to Cloudflare API
func (c *CloudflareCDNService) TestConnection() error {
	if c.config == nil || !c.config.Enabled {
		return fmt.Errorf("CDN is not configured or disabled")
	}
	
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s", c.config.ZoneID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Cloudflare API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Cloudflare API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// PurgeCache purges cache based on the provided request
func (c *CloudflareCDNService) PurgeCache(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	if c.config == nil || !c.config.Enabled {
		return nil, fmt.Errorf("CDN is not configured or disabled")
	}
	
	if c.failoverMode {
		return &models.CDNPurgeResponse{
			Success:   false,
			Message:   "CDN is in failover mode",
			Timestamp: time.Now(),
		}, nil
	}
	
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", c.config.ZoneID)
	
	// Prepare request body
	var requestBody map[string]interface{}
	
	if request.PurgeAll {
		requestBody = map[string]interface{}{
			"purge_everything": true,
		}
	} else {
		requestBody = make(map[string]interface{})
		if len(request.URLs) > 0 {
			requestBody["files"] = request.URLs
		}
		if len(request.Tags) > 0 {
			requestBody["tags"] = request.Tags
		}
		if len(request.Hosts) > 0 {
			requestBody["hosts"] = request.Hosts
		}
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Enable failover on connection error
		c.enableFailoverMode()
		return nil, fmt.Errorf("failed to purge cache: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var cloudflareResp struct {
		Success bool   `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(body, &cloudflareResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	response := &models.CDNPurgeResponse{
		Success:   cloudflareResp.Success,
		RequestID: cloudflareResp.Result.ID,
		Timestamp: time.Now(),
	}
	
	if !cloudflareResp.Success && len(cloudflareResp.Errors) > 0 {
		response.Message = cloudflareResp.Errors[0].Message
	} else {
		response.Message = "Cache purged successfully"
	}
	
	return response, nil
}

// PurgeURL purges cache for a single URL
func (c *CloudflareCDNService) PurgeURL(url string) error {
	request := &models.CDNPurgeRequest{
		URLs: []string{url},
	}
	
	response, err := c.PurgeCache(request)
	if err != nil {
		return err
	}
	
	if !response.Success {
		return fmt.Errorf("failed to purge URL: %s", response.Message)
	}
	
	return nil
}

// PurgeURLs purges cache for multiple URLs
func (c *CloudflareCDNService) PurgeURLs(urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	
	// Cloudflare allows up to 30 URLs per request
	const batchSize = 30
	
	for i := 0; i < len(urls); i += batchSize {
		end := i + batchSize
		if end > len(urls) {
			end = len(urls)
		}
		
		request := &models.CDNPurgeRequest{
			URLs: urls[i:end],
		}
		
		response, err := c.PurgeCache(request)
		if err != nil {
			return fmt.Errorf("failed to purge URLs batch %d-%d: %w", i, end-1, err)
		}
		
		if !response.Success {
			return fmt.Errorf("failed to purge URLs batch %d-%d: %s", i, end-1, response.Message)
		}
		
		// Add small delay between batches to avoid rate limiting
		if end < len(urls) {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return nil
}

// PurgeAll purges all cache
func (c *CloudflareCDNService) PurgeAll() error {
	request := &models.CDNPurgeRequest{
		PurgeAll: true,
	}
	
	response, err := c.PurgeCache(request)
	if err != nil {
		return err
	}
	
	if !response.Success {
		return fmt.Errorf("failed to purge all cache: %s", response.Message)
	}
	
	return nil
}

// GetStats retrieves CDN performance statistics
func (c *CloudflareCDNService) GetStats() (*models.CDNStats, error) {
	if c.config == nil || !c.config.Enabled {
		return nil, fmt.Errorf("CDN is not configured or disabled")
	}
	
	// Get analytics from Cloudflare
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/analytics/dashboard", c.config.ZoneID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add query parameters for time range (last 24 hours)
	q := req.URL.Query()
	q.Add("since", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	q.Add("until", time.Now().Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.stats, nil // Return cached stats on error
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return c.stats, nil // Return cached stats on error
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.stats, nil
	}
	
	var analyticsResp struct {
		Success bool `json:"success"`
		Result  struct {
			Totals struct {
				Requests struct {
					All       int64 `json:"all"`
					Cached    int64 `json:"cached"`
					Uncached  int64 `json:"uncached"`
				} `json:"requests"`
				Bandwidth struct {
					All       int64 `json:"all"`
					Cached    int64 `json:"cached"`
					Uncached  int64 `json:"uncached"`
				} `json:"bandwidth"`
			} `json:"totals"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(body, &analyticsResp); err != nil {
		return c.stats, nil
	}
	
	if analyticsResp.Success {
		c.mutex.Lock()
		c.stats.RequestsServed = analyticsResp.Result.Totals.Requests.All
		c.stats.OriginRequests = analyticsResp.Result.Totals.Requests.Uncached
		
		if analyticsResp.Result.Totals.Requests.All > 0 {
			c.stats.CacheHitRatio = float64(analyticsResp.Result.Totals.Requests.Cached) / float64(analyticsResp.Result.Totals.Requests.All) * 100
		}
		
		c.stats.BandwidthSaved = analyticsResp.Result.Totals.Bandwidth.Cached
		c.stats.LastUpdated = time.Now()
		c.mutex.Unlock()
	}
	
	return c.stats, nil
}

// GetHealthStatus checks CDN health status
func (c *CloudflareCDNService) GetHealthStatus() (*models.CDNHealthCheck, error) {
	healthCheck := &models.CDNHealthCheck{
		Provider:  "cloudflare",
		LastCheck: time.Now(),
	}
	
	if c.config == nil || !c.config.Enabled {
		healthCheck.Status = "down"
		healthCheck.Message = "CDN is not configured or disabled"
		return healthCheck, nil
	}
	
	if c.failoverMode {
		healthCheck.Status = "degraded"
		healthCheck.Message = "CDN is in failover mode"
		return healthCheck, nil
	}
	
	start := time.Now()
	err := c.TestConnection()
	responseTime := time.Since(start).Milliseconds()
	
	healthCheck.ResponseTime = responseTime
	
	if err != nil {
		healthCheck.Status = "down"
		healthCheck.Message = err.Error()
		healthCheck.ErrorCount++
	} else {
		healthCheck.Status = "healthy"
		healthCheck.Message = "CDN is operating normally"
		healthCheck.ErrorCount = 0
	}
	
	c.lastHealthCheck = time.Now()
	return healthCheck, nil
}

// EnableFailover enables failover mode
func (c *CloudflareCDNService) EnableFailover() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.failoverMode = true
	return nil
}

// DisableFailover disables failover mode
func (c *CloudflareCDNService) DisableFailover() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.failoverMode = false
	return nil
}

// IsFailoverActive returns whether failover mode is active
func (c *CloudflareCDNService) IsFailoverActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return c.failoverMode
}

// enableFailoverMode enables failover mode internally
func (c *CloudflareCDNService) enableFailoverMode() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.failoverMode = true
}

// PurgeArticleCache purges cache for article-related URLs
func (c *CloudflareCDNService) PurgeArticleCache(articleSlug string) error {
	if c.config == nil || !c.config.Enabled {
		return nil // Silently skip if CDN is not configured
	}
	
	urls := []string{
		fmt.Sprintf("https://%s/articles/%s", c.config.Domain, articleSlug),
		fmt.Sprintf("https://%s/", c.config.Domain), // Homepage
		fmt.Sprintf("https://%s/sitemap.xml", c.config.Domain),
		fmt.Sprintf("https://%s/rss.xml", c.config.Domain),
	}
	
	return c.PurgeURLs(urls)
}

// PurgeCategoryCache purges cache for category-related URLs
func (c *CloudflareCDNService) PurgeCategoryCache(categorySlug string) error {
	if c.config == nil || !c.config.Enabled {
		return nil
	}
	
	urls := []string{
		fmt.Sprintf("https://%s/categories/%s", c.config.Domain, categorySlug),
		fmt.Sprintf("https://%s/", c.config.Domain), // Homepage
		fmt.Sprintf("https://%s/rss/category/%s.xml", c.config.Domain, categorySlug),
	}
	
	return c.PurgeURLs(urls)
}

// PurgeTagCache purges cache for tag-related URLs
func (c *CloudflareCDNService) PurgeTagCache(tagSlug string) error {
	if c.config == nil || !c.config.Enabled {
		return nil
	}
	
	urls := []string{
		fmt.Sprintf("https://%s/tags/%s", c.config.Domain, tagSlug),
		fmt.Sprintf("https://%s/rss/tag/%s.xml", c.config.Domain, tagSlug),
	}
	
	return c.PurgeURLs(urls)
}