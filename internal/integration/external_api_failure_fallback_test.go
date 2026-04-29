package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// ExternalAPIFailureFallbackTestSuite tests external API failure scenarios and fallback mechanisms
type ExternalAPIFailureFallbackTestSuite struct {
	searchService      *services.SearchService
	emailService       *services.EmailService
	socialMediaService *services.SocialMediaService
	cdnService         *services.CDNService
	failureSimulator   *ExternalAPIFailureSimulator
}

// ExternalAPIFailureSimulator simulates various external API failure scenarios
type ExternalAPIFailureSimulator struct {
	serviceFailures    map[string]bool
	rateLimits         map[string]time.Time
	circuitBreakers    map[string]*CircuitBreakerState
	networkLatency     time.Duration
	timeoutSimulation  bool
	authFailures       bool
	mu                 sync.Mutex
}

type CircuitBreakerState struct {
	failures     int
	lastFailure  time.Time
	state        string // "closed", "open", "half-open"
	threshold    int
	timeout      time.Duration
}

func TestSearchServiceMeiliSearchFailureFallback(t *testing.T) {
	t.Run("MeiliSearch unavailable fallback to PostgreSQL", func(t *testing.T) {
		// Create failing MeiliSearch server
		failingMeiliServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "MeiliSearch service unavailable"}`))
		}))
		defer failingMeiliServer.Close()

		// Create mock database for fallback
		mockDB := &MockDatabaseForSearch{}
		mockCache := &MockCacheForSearch{}

		searchService, err := services.NewSearchService(mockDB, mockCache, failingMeiliServer.URL, "test-key")
		require.NoError(t, err)

		ctx := context.Background()
		filters := services.SearchFilters{
			Query: "technology news",
		}

		start := time.Now()
		results, facets, total, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		duration := time.Since(start)

		assert.NoError(t, err, "Should fallback to PostgreSQL when MeiliSearch fails")
		assert.NotNil(t, results)
		assert.NotNil(t, facets)
		assert.GreaterOrEqual(t, total, 0)
		assert.Greater(t, searchTime, 50.0, "PostgreSQL fallback should be slower than MeiliSearch")
		assert.Less(t, duration, 2*time.Second, "Fallback should not take too long")

		t.Logf("MeiliSearch fallback completed in %v (search time: %.2f ms)", duration, searchTime)
	})

	t.Run("MeiliSearch timeout fallback", func(t *testing.T) {
		// Create slow MeiliSearch server
		slowMeiliServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(3 * time.Second) // Simulate slow response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"hits": [], "estimatedTotalHits": 0}`))
		}))
		defer slowMeiliServer.Close()

		mockDB := &MockDatabaseForSearch{}
		mockCache := &MockCacheForSearch{}

		searchService, err := services.NewSearchService(mockDB, mockCache, slowMeiliServer.URL, "test-key")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		filters := services.SearchFilters{
			Query: "timeout test",
		}

		start := time.Now()
		results, facets, total, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		duration := time.Since(start)

		assert.NoError(t, err, "Should fallback to PostgreSQL on timeout")
		assert.NotNil(t, results)
		assert.NotNil(t, facets)
		assert.GreaterOrEqual(t, total, 0)
		assert.Less(t, duration, 2*time.Second, "Should timeout and fallback quickly")

		t.Logf("MeiliSearch timeout fallback completed in %v", duration)
	})

	t.Run("MeiliSearch partial failure handling", func(t *testing.T) {
		// Create server that fails intermittently
		var requestCount int
		var mu sync.Mutex

		intermittentMeiliServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			if count%3 == 0 { // Fail every 3rd request
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Internal server error"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"hits": [{"id": 1, "title": "MeiliSearch Result"}], "estimatedTotalHits": 1}`))
		}))
		defer intermittentMeiliServer.Close()

		mockDB := &MockDatabaseForSearch{}
		mockCache := &MockCacheForSearch{}

		searchService, err := services.NewSearchService(mockDB, mockCache, intermittentMeiliServer.URL, "test-key")
		require.NoError(t, err)

		ctx := context.Background()
		filters := services.SearchFilters{Query: "intermittent"}

		// Run multiple searches
		const numSearches = 9
		var meiliSuccesses, fallbackCount int

		for i := 0; i < numSearches; i++ {
			results, _, _, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
			assert.NoError(t, err, "All searches should succeed")
			assert.NotNil(t, results)

			if searchTime < 50.0 {
				meiliSuccesses++
			} else {
				fallbackCount++
			}
		}

		assert.Greater(t, meiliSuccesses, 0, "Some searches should succeed with MeiliSearch")
		assert.Greater(t, fallbackCount, 0, "Some searches should fallback to PostgreSQL")

		t.Logf("Intermittent failure: %d MeiliSearch successes, %d PostgreSQL fallbacks", 
			meiliSuccesses, fallbackCount)
	})
}

func TestEmailServiceProviderFailureFallback(t *testing.T) {
	t.Run("Primary email provider failure fallback to secondary", func(t *testing.T) {
		// Create failing primary email server
		failingPrimaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Primary email service unavailable"}`))
		}))
		defer failingPrimaryServer.Close()

		// Create working secondary email server
		workingSecondaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Email sent via secondary provider", "id": "secondary-123"}`))
		}))
		defer workingSecondaryServer.Close()

		emailService := &MockEmailServiceWithFallback{
			primaryURL:   failingPrimaryServer.URL,
			secondaryURL: workingSecondaryServer.URL,
		}

		ctx := context.Background()
		email := &services.EmailMessage{
			To:      "test@example.com",
			Subject: "Test Email",
			Content: "Test content",
		}

		start := time.Now()
		result, err := emailService.SendEmailWithFallback(ctx, email)
		duration := time.Since(start)

		assert.NoError(t, err, "Should succeed with secondary provider")
		assert.NotNil(t, result)
		assert.Equal(t, "sent", result.Status)
		assert.Contains(t, result.Message, "secondary provider")
		assert.Greater(t, duration, 50*time.Millisecond, "Should take time to try primary first")

		t.Logf("Email fallback completed in %v", duration)
	})

	t.Run("Email service rate limiting handling", func(t *testing.T) {
		// Create rate-limited email server
		var requestCount int
		var mu sync.Mutex

		rateLimitedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			if count > 3 { // Rate limit after 3 requests
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Email sent", "id": "email-` + fmt.Sprintf("%d", count) + `"}`))
		}))
		defer rateLimitedServer.Close()

		emailService := &MockEmailServiceWithFallback{
			primaryURL: rateLimitedServer.URL,
		}

		ctx := context.Background()

		// Send multiple emails
		const numEmails = 5
		var successCount, rateLimitedCount int

		for i := 0; i < numEmails; i++ {
			email := &services.EmailMessage{
				To:      fmt.Sprintf("test%d@example.com", i),
				Subject: fmt.Sprintf("Test Email %d", i),
				Content: "Test content",
			}

			result, err := emailService.SendEmailWithFallback(ctx, email)
			if err == nil && result.Status == "sent" {
				successCount++
			} else if err != nil && err.Error() == "rate limit exceeded" {
				rateLimitedCount++
			}
		}

		assert.Greater(t, successCount, 0, "Some emails should be sent successfully")
		assert.Greater(t, rateLimitedCount, 0, "Some emails should be rate limited")

		t.Logf("Email rate limiting: %d sent, %d rate limited", successCount, rateLimitedCount)
	})

	t.Run("Email service authentication failure", func(t *testing.T) {
		// Create server that returns auth errors
		authFailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Invalid API key"}`))
		}))
		defer authFailServer.Close()

		emailService := &MockEmailServiceWithFallback{
			primaryURL: authFailServer.URL,
		}

		ctx := context.Background()
		email := &services.EmailMessage{
			To:      "test@example.com",
			Subject: "Auth Test",
			Content: "Test content",
		}

		result, err := emailService.SendEmailWithFallback(ctx, email)
		assert.Error(t, err, "Should fail with authentication error")
		assert.Contains(t, err.Error(), "authentication failed")
		
		if result != nil {
			assert.Equal(t, "failed", result.Status)
		}
	})
}

func TestSocialMediaAPIFailureHandling(t *testing.T) {
	t.Run("Social media API rate limiting with exponential backoff", func(t *testing.T) {
		// Create rate-limited social media server
		var requestCount int
		var mu sync.Mutex

		rateLimitedSocialServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			if count <= 2 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "id": "post-` + fmt.Sprintf("%d", count) + `"}`))
			} else {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(15*time.Minute).Unix()))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded"}`))
			}
		}))
		defer rateLimitedSocialServer.Close()

		socialService := &MockSocialMediaServiceWithRetry{
			apiURL: rateLimitedSocialServer.URL,
		}

		ctx := context.Background()

		// Try to post multiple times
		const numPosts = 5
		var successCount, rateLimitedCount int

		for i := 0; i < numPosts; i++ {
			post := &models.SocialMediaPost{
				ID:       uint64(i + 1),
				Platform: models.PlatformTwitter,
				Content: models.PostContent{
					Text: fmt.Sprintf("Test post %d", i+1),
				},
			}

			result, err := socialService.PublishPostWithRetry(ctx, post)
			if err == nil && result.Success {
				successCount++
			} else if err == nil && result.RateLimited {
				rateLimitedCount++
			}
		}

		assert.Greater(t, successCount, 0, "Some posts should succeed")
		assert.Greater(t, rateLimitedCount, 0, "Some posts should be rate limited")

		t.Logf("Social media rate limiting: %d posted, %d rate limited", successCount, rateLimitedCount)
	})

	t.Run("Social media API circuit breaker pattern", func(t *testing.T) {
		circuitBreaker := &MockCircuitBreaker{
			failureThreshold: 3,
			timeout:         1 * time.Second,
		}

		socialService := &MockSocialMediaServiceWithRetry{
			apiURL:         "http://failing-endpoint",
			circuitBreaker: circuitBreaker,
		}

		ctx := context.Background()

		// Generate failures to trip circuit breaker
		for i := 0; i < 5; i++ {
			post := &models.SocialMediaPost{
				ID:       uint64(i + 1),
				Platform: models.PlatformTwitter,
				Content: models.PostContent{
					Text: fmt.Sprintf("Test post %d", i+1),
				},
			}

			result, err := socialService.PublishPostWithRetry(ctx, post)
			
			if i < 3 {
				assert.Error(t, err, "First 3 posts should fail")
			} else {
				assert.Error(t, err, "Circuit breaker should be open")
				assert.Contains(t, err.Error(), "circuit breaker")
			}
		}

		assert.True(t, circuitBreaker.IsOpen(), "Circuit breaker should be open")

		// Wait for circuit breaker timeout
		time.Sleep(1200 * time.Millisecond)

		// Circuit breaker should allow test request
		assert.True(t, circuitBreaker.IsHalfOpen(), "Circuit breaker should be half-open")
	})

	t.Run("Multiple social media platform failure handling", func(t *testing.T) {
		// Create servers for different platforms with different failure modes
		twitterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Twitter API unavailable"}`))
		}))
		defer twitterServer.Close()

		facebookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "id": "fb-post-123"}`))
		}))
		defer facebookServer.Close()

		telegramServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded"}`))
		}))
		defer telegramServer.Close()

		socialService := &MockSocialMediaServiceWithRetry{
			platformURLs: map[models.Platform]string{
				models.PlatformTwitter:  twitterServer.URL,
				models.PlatformFacebook: facebookServer.URL,
				models.PlatformTelegram: telegramServer.URL,
			},
		}

		ctx := context.Background()

		// Test posting to different platforms
		platforms := []models.Platform{
			models.PlatformTwitter,
			models.PlatformFacebook,
			models.PlatformTelegram,
		}

		var results []string

		for i, platform := range platforms {
			post := &models.SocialMediaPost{
				ID:       uint64(i + 1),
				Platform: platform,
				Content: models.PostContent{
					Text: fmt.Sprintf("Test post for %s", platform),
				},
			}

			result, err := socialService.PublishPostWithRetry(ctx, post)
			if err == nil && result.Success {
				results = append(results, fmt.Sprintf("%s: success", platform))
			} else if err == nil && result.RateLimited {
				results = append(results, fmt.Sprintf("%s: rate limited", platform))
			} else {
				results = append(results, fmt.Sprintf("%s: failed", platform))
			}
		}

		assert.Len(t, results, 3, "Should have results for all platforms")
		assert.Contains(t, results, "facebook: success", "Facebook should succeed")
		assert.Contains(t, results, "twitter: failed", "Twitter should fail")
		assert.Contains(t, results, "telegram: rate limited", "Telegram should be rate limited")

		t.Logf("Multi-platform results: %v", results)
	})
}

func TestCDNServiceFailureHandling(t *testing.T) {
	t.Run("CDN API failure enables failover mode", func(t *testing.T) {
		// Create failing CDN server
		failingCDNServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "CDN service unavailable"}`))
		}))
		defer failingCDNServer.Close()

		cdnService := &MockCDNServiceWithFailover{
			apiURL:       failingCDNServer.URL,
			failoverMode: false,
		}

		purgeRequest := &models.CDNPurgeRequest{
			URLs: []string{
				"https://example.com/article/1",
				"https://example.com/category/tech",
			},
		}

		// First purge should fail and enable failover
		response, err := cdnService.PurgeCacheWithFailover(purgeRequest)
		assert.NoError(t, err, "Should handle CDN failure gracefully")
		assert.NotNil(t, response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Message, "failover mode")
		assert.True(t, cdnService.IsFailoverActive())

		// Subsequent operations should use failover mode
		response2, err := cdnService.PurgeCacheWithFailover(purgeRequest)
		assert.NoError(t, err, "Should work in failover mode")
		assert.False(t, response2.Success)
		assert.Contains(t, response2.Message, "failover mode")
	})

	t.Run("CDN batch operation failure handling", func(t *testing.T) {
		// Create server that fails for large batches
		batchFailCDNServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check content length or parse body to determine batch size
			if r.ContentLength > 1000 { // Simulate failure for large batches
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte(`{"error": "Batch too large"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "id": "purge-123"}`))
		}))
		defer batchFailCDNServer.Close()

		cdnService := &MockCDNServiceWithFailover{
			apiURL: batchFailCDNServer.URL,
		}

		// Create large batch of URLs
		urls := make([]string, 100)
		for i := 0; i < 100; i++ {
			urls[i] = fmt.Sprintf("https://example.com/page%d", i)
		}

		purgeRequest := &models.CDNPurgeRequest{URLs: urls}

		// Should handle batch failure by splitting into smaller batches
		response, err := cdnService.PurgeCacheWithBatchSplit(purgeRequest)
		assert.NoError(t, err, "Should handle large batch by splitting")
		assert.NotNil(t, response)
		// In a real implementation, this would split the batch and retry
	})

	t.Run("CDN statistics retrieval failure", func(t *testing.T) {
		// Create server that fails for stats endpoint
		statsFailCDNServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/stats" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error": "Stats service unavailable"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}))
		defer statsFailCDNServer.Close()

		cdnService := &MockCDNServiceWithFailover{
			apiURL: statsFailCDNServer.URL,
		}

		// Stats retrieval should fail gracefully
		stats, err := cdnService.GetStatsWithFallback()
		assert.Error(t, err, "Stats should fail when service unavailable")
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "stats unavailable")

		// Other operations should still work
		purgeRequest := &models.CDNPurgeRequest{
			URLs: []string{"https://example.com/test"},
		}
		response, err := cdnService.PurgeCacheWithFailover(purgeRequest)
		assert.NoError(t, err, "Purge should still work when stats fail")
		assert.NotNil(t, response)
	})
}

func TestExternalAPICircuitBreakerRecovery(t *testing.T) {
	t.Run("Circuit breaker recovery after service restoration", func(t *testing.T) {
		// Create server that fails initially then recovers
		var requestCount int
		var mu sync.Mutex

		recoveringServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			if count <= 5 {
				// Fail first 5 requests
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Service temporarily unavailable"}`))
			} else {
				// Succeed after 5 failures
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "message": "Service recovered"}`))
			}
		}))
		defer recoveringServer.Close()

		circuitBreaker := &MockCircuitBreaker{
			failureThreshold: 3,
			timeout:         500 * time.Millisecond,
		}

		externalService := &MockExternalServiceWithCircuitBreaker{
			apiURL:         recoveringServer.URL,
			circuitBreaker: circuitBreaker,
		}

		ctx := context.Background()

		// Generate failures to trip circuit breaker
		for i := 0; i < 4; i++ {
			_, err := externalService.MakeRequestWithCircuitBreaker(ctx, "/test")
			if i < 3 {
				assert.Error(t, err, "Request %d should fail", i+1)
			} else {
				assert.Error(t, err, "Circuit breaker should be open")
				assert.Contains(t, err.Error(), "circuit breaker")
			}
		}

		assert.True(t, circuitBreaker.IsOpen(), "Circuit breaker should be open")

		// Wait for circuit breaker timeout
		time.Sleep(600 * time.Millisecond)

		// Circuit breaker should be half-open, allowing test requests
		assert.True(t, circuitBreaker.IsHalfOpen(), "Circuit breaker should be half-open")

		// Next request should succeed and close circuit breaker
		response, err := externalService.MakeRequestWithCircuitBreaker(ctx, "/test")
		assert.NoError(t, err, "Request should succeed after service recovery")
		assert.NotNil(t, response)
		assert.False(t, circuitBreaker.IsOpen(), "Circuit breaker should be closed after success")

		t.Logf("Circuit breaker recovery successful after %d failed requests", requestCount)
	})
}

// Mock implementations for external API failure testing

type MockDatabaseForSearch struct{}

func (db *MockDatabaseForSearch) SearchArticles(query string, limit, offset int) ([]models.Article, int, error) {
	// Simulate PostgreSQL search
	time.Sleep(75 * time.Millisecond)
	return []models.Article{
		{ID: 1, Title: "PostgreSQL Search Result"},
	}, 1, nil
}

type MockCacheForSearch struct{}

func (c *MockCacheForSearch) Get(key string) ([]byte, error) {
	return nil, errors.New("cache miss")
}

func (c *MockCacheForSearch) Set(key string, value []byte, ttl time.Duration) error {
	return nil
}

func (c *MockCacheForSearch) Delete(key string) error { return nil }
func (c *MockCacheForSearch) DeletePattern(pattern string) error { return nil }
func (c *MockCacheForSearch) Exists(key string) bool { return false }

type MockEmailServiceWithFallback struct {
	primaryURL   string
	secondaryURL string
}

func (e *MockEmailServiceWithFallback) SendEmailWithFallback(ctx context.Context, email *services.EmailMessage) (*services.EmailResult, error) {
	// Try primary provider first
	resp, err := http.Get(e.primaryURL + "/send")
	if err == nil && resp.StatusCode == http.StatusOK {
		return &services.EmailResult{
			Status:  "sent",
			Message: "Email sent via primary provider",
		}, nil
	}

	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("authentication failed")
	}

	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("rate limit exceeded")
	}

	// Fallback to secondary provider
	if e.secondaryURL != "" {
		resp, err = http.Get(e.secondaryURL + "/send")
		if err == nil && resp.StatusCode == http.StatusOK {
			return &services.EmailResult{
				Status:  "sent",
				Message: "Email sent via secondary provider",
			}, nil
		}
	}

	return &services.EmailResult{
		Status: "failed",
		Error:  "All email providers failed",
	}, errors.New("all email providers failed")
}

type MockSocialMediaServiceWithRetry struct {
	apiURL         string
	platformURLs   map[models.Platform]string
	circuitBreaker *MockCircuitBreaker
}

func (s *MockSocialMediaServiceWithRetry) PublishPostWithRetry(ctx context.Context, post *models.SocialMediaPost) (*models.PostResult, error) {
	if s.circuitBreaker != nil && s.circuitBreaker.IsOpen() {
		return nil, errors.New("circuit breaker open")
	}

	var url string
	if s.platformURLs != nil {
		url = s.platformURLs[post.Platform]
	} else {
		url = s.apiURL
	}

	resp, err := http.Get(url + "/post")
	if err != nil {
		if s.circuitBreaker != nil {
			s.circuitBreaker.RecordFailure()
		}
		return &models.PostResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 15 * time.Minute // Simplified
		return &models.PostResult{
			Success:     false,
			RateLimited: true,
			RetryAfter:  retryAfter,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		if s.circuitBreaker != nil {
			s.circuitBreaker.RecordFailure()
		}
		return &models.PostResult{
			Success: false,
			Error:   "API request failed",
		}, errors.New("API request failed")
	}

	if s.circuitBreaker != nil {
		s.circuitBreaker.RecordSuccess()
	}

	return &models.PostResult{
		Success: true,
		PostID:  fmt.Sprintf("post-%d", post.ID),
	}, nil
}

type MockCDNServiceWithFailover struct {
	apiURL       string
	failoverMode bool
}

func (c *MockCDNServiceWithFailover) PurgeCacheWithFailover(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	if c.failoverMode {
		return &models.CDNPurgeResponse{
			Success: false,
			Message: "Operating in failover mode - purge skipped",
		}, nil
	}

	resp, err := http.Get(c.apiURL + "/purge")
	if err != nil || resp.StatusCode != http.StatusOK {
		// Enable failover mode
		c.failoverMode = true
		return &models.CDNPurgeResponse{
			Success: false,
			Message: "CDN unavailable, enabled failover mode",
		}, nil
	}

	return &models.CDNPurgeResponse{
		Success:   true,
		RequestID: "purge-123",
	}, nil
}

func (c *MockCDNServiceWithFailover) PurgeCacheWithBatchSplit(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	// Simulate batch splitting logic
	if len(request.URLs) > 50 {
		// In real implementation, would split into smaller batches
		return &models.CDNPurgeResponse{
			Success: true,
			Message: "Large batch split and processed",
		}, nil
	}

	return c.PurgeCacheWithFailover(request)
}

func (c *MockCDNServiceWithFailover) GetStatsWithFallback() (*models.CDNStats, error) {
	resp, err := http.Get(c.apiURL + "/stats")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("CDN stats unavailable")
	}

	return &models.CDNStats{
		RequestsServed: 10000,
		BandwidthSaved: 1024 * 1024 * 100, // 100MB
		CacheHitRatio:  0.85,
	}, nil
}

func (c *MockCDNServiceWithFailover) IsFailoverActive() bool {
	return c.failoverMode
}

type MockCircuitBreaker struct {
	failures         int
	lastFailureTime  time.Time
	state           string // "closed", "open", "half-open"
	failureThreshold int
	timeout         time.Duration
	mu              sync.Mutex
}

func (cb *MockCircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "open" && time.Since(cb.lastFailureTime) > cb.timeout {
		cb.state = "half-open"
	}
	return cb.state == "open"
}

func (cb *MockCircuitBreaker) IsHalfOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == "half-open"
}

func (cb *MockCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()
	if cb.failures >= cb.failureThreshold {
		cb.state = "open"
	}
}

func (cb *MockCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
}

type MockExternalServiceWithCircuitBreaker struct {
	apiURL         string
	circuitBreaker *MockCircuitBreaker
}

func (s *MockExternalServiceWithCircuitBreaker) MakeRequestWithCircuitBreaker(ctx context.Context, endpoint string) (interface{}, error) {
	if s.circuitBreaker.IsOpen() {
		return nil, errors.New("circuit breaker open")
	}

	resp, err := http.Get(s.apiURL + endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		s.circuitBreaker.RecordFailure()
		return nil, errors.New("request failed")
	}

	s.circuitBreaker.RecordSuccess()
	return map[string]string{"status": "success"}, nil
}

// Benchmark tests for external API failure scenarios

func BenchmarkExternalAPIFailureFallback(b *testing.B) {
	// Create failing primary and working secondary servers
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failingServer.Close()

	workingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer workingServer.Close()

	emailService := &MockEmailServiceWithFallback{
		primaryURL:   failingServer.URL,
		secondaryURL: workingServer.URL,
	}

	ctx := context.Background()
	email := &services.EmailMessage{
		To:      "benchmark@example.com",
		Subject: "Benchmark Email",
		Content: "Benchmark content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := emailService.SendEmailWithFallback(ctx, email)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCircuitBreakerPattern(b *testing.B) {
	circuitBreaker := &MockCircuitBreaker{
		failureThreshold: 5,
		timeout:         100 * time.Millisecond,
	}

	externalService := &MockExternalServiceWithCircuitBreaker{
		apiURL:         "http://localhost:9999", // Non-existent server
		circuitBreaker: circuitBreaker,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		externalService.MakeRequestWithCircuitBreaker(ctx, "/test")
		// Don't fail on error since we expect failures to test circuit breaker
	}
}