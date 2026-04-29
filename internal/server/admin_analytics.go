package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// renderAdminAnalytics renders the analytics admin page with real data from the database
func (s *Server) renderAdminAnalytics(c *gin.Context) {
	title := "Analytics Dashboard"

	// Default time range: last 7 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// Check for custom date range from query params
	if start := c.Query("start"); start != "" {
		if parsed, err := time.Parse("2006-01-02", start); err == nil {
			startDate = parsed
		}
	}
	if end := c.Query("end"); end != "" {
		if parsed, err := time.Parse("2006-01-02", end); err == nil {
			endDate = parsed
		}
	}

	// Build analytics content
	var analyticsContent string

	// Check if analytics service is available
	if s.analyticsService != nil {
		ctx := context.Background()

		// Get dashboard metrics from real database
		metrics, err := s.analyticsService.GetDashboardMetrics(ctx, startDate, endDate)
		if err != nil {
			analyticsContent = s.buildErrorContent("Failed to load analytics data: " + err.Error())
		} else {
			// Get top articles
			topArticles, _ := s.analyticsService.GetTopArticles(ctx, startDate, endDate, 10)

			// Get traffic sources
			trafficSources, _ := s.analyticsService.GetTrafficSources(ctx, startDate, endDate)

			analyticsContent = s.buildRealAnalyticsContent(metrics, topArticles, trafficSources, startDate, endDate)
		}
	} else {
		analyticsContent = s.buildFallbackContent()
	}

	s.renderAdminPage(c, title, "analytics", analyticsContent)
}

// buildRealAnalyticsContent builds HTML content with real analytics data
func (s *Server) buildRealAnalyticsContent(metrics interface{}, topArticles interface{}, trafficSources interface{}, startDate, endDate time.Time) string {
	// Type assert metrics
	m, ok := metrics.(interface {
		GetTotalViews() int64
		GetUniqueVisitors() int64
		GetTotalEngagements() int64
		GetBounceRate() float64
		GetAvgTimeOnSite() float64
	})

	var totalViews, uniqueVisitors, totalEngagements int64
	var bounceRate, avgTimeOnSite float64

	if ok {
		totalViews = m.GetTotalViews()
		uniqueVisitors = m.GetUniqueVisitors()
		totalEngagements = m.GetTotalEngagements()
		bounceRate = m.GetBounceRate()
		avgTimeOnSite = m.GetAvgTimeOnSite()
	}

	// Build date range selector
	dateRangeHTML := fmt.Sprintf(`
		<div class="dashboard-card" style="grid-column: 1 / -1;">
			<div class="card-title">📅 Date Range</div>
			<form method="GET" action="/admin/analytics" style="display: flex; gap: 1rem; align-items: center; flex-wrap: wrap;">
				<label>From: <input type="date" name="start" value="%s" style="padding: 0.5rem; border: 1px solid #ddd; border-radius: 4px;"></label>
				<label>To: <input type="date" name="end" value="%s" style="padding: 0.5rem; border: 1px solid #ddd; border-radius: 4px;"></label>
				<button type="submit" style="padding: 0.5rem 1rem; background: #3b82f6; color: white; border: none; border-radius: 4px; cursor: pointer;">Apply</button>
				<span style="color: #666; font-size: 0.9rem;">Showing data from %s to %s</span>
			</form>
		</div>`,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		startDate.Format("Jan 2, 2006"),
		endDate.Format("Jan 2, 2006"),
	)

	// Build overview cards
	overviewHTML := fmt.Sprintf(`
		<div class="dashboard-card">
			<div class="card-title">📊 Page Views</div>
			<div>
				<p>Total Views: <span class="metric">%d</span></p>
				<p>Unique Visitors: <span class="metric">%d</span></p>
				<p>Avg. Time on Site: <span class="metric">%.1f sec</span></p>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">👥 User Engagement</div>
			<div>
				<p>Total Engagements: <span class="metric">%d</span></p>
				<p>Bounce Rate: <span class="metric">%.1f%%</span></p>
				<p>Engagement Rate: <span class="metric">%.2f%%</span></p>
			</div>
		</div>`,
		totalViews,
		uniqueVisitors,
		avgTimeOnSite,
		totalEngagements,
		bounceRate*100,
		func() float64 {
			if totalViews > 0 {
				return float64(totalEngagements) / float64(totalViews) * 100
			}
			return 0
		}(),
	)

	// Build top articles section
	topArticlesHTML := `
		<div class="dashboard-card">
			<div class="card-title">📈 Top Articles</div>
			<div style="max-height: 300px; overflow-y: auto;">`

	if articles, ok := topArticles.([]interface{ GetTitle() string; GetViewCount() int64 }); ok && len(articles) > 0 {
		for i, article := range articles {
			if i >= 10 {
				break
			}
			topArticlesHTML += fmt.Sprintf(`<p>%d. %s - <span class="metric">%d views</span></p>`,
				i+1, article.GetTitle(), article.GetViewCount())
		}
	} else {
		topArticlesHTML += `<p style="color: #666;">No article data available for this period</p>`
	}
	topArticlesHTML += `</div></div>`

	// Build traffic sources section
	trafficHTML := `
		<div class="dashboard-card">
			<div class="card-title">🌍 Traffic Sources</div>
			<div style="max-height: 300px; overflow-y: auto;">`

	if sources, ok := trafficSources.([]interface{ GetSource() string; GetSessions() int64 }); ok && len(sources) > 0 {
		for _, source := range sources {
			trafficHTML += fmt.Sprintf(`<p>%s: <span class="metric">%d sessions</span></p>`,
				source.GetSource(), source.GetSessions())
		}
	} else {
		trafficHTML += `<p style="color: #666;">No traffic source data available</p>`
	}
	trafficHTML += `</div></div>`

	// Build export section
	exportHTML := `
		<div class="dashboard-card">
			<div class="card-title">📥 Export Reports</div>
			<div style="display: flex; flex-direction: column; gap: 0.5rem;">
				<a href="/api/v1/analytics/export?format=csv" class="action-button" style="text-align: center;">Export as CSV</a>
				<a href="/api/v1/analytics/export?format=json" class="action-button" style="text-align: center;">Export as JSON</a>
			</div>
		</div>`

	// Build real-time metrics section (uses JavaScript to fetch from API)
	realTimeHTML := `
		<div class="dashboard-card" id="realtime-metrics">
			<div class="card-title">⚡ Real-Time Metrics</div>
			<div id="realtime-content">
				<p>Loading real-time data...</p>
			</div>
		</div>
		<script>
			async function loadRealtimeMetrics() {
				try {
					const token = localStorage.getItem('auth_token');
					const response = await fetch('/api/v1/analytics/overview', {
						headers: {
							'Authorization': 'Bearer ' + token
						}
					});
					if (response.ok) {
						const data = await response.json();
						const content = document.getElementById('realtime-content');
						if (data.data) {
							content.innerHTML = '<p>Active Users: <span class="metric">' + (data.data.active_users || 0) + '</span></p>' +
								'<p>Views Today: <span class="metric">' + (data.data.views_today || 0) + '</span></p>' +
								'<p>Last Updated: ' + new Date().toLocaleTimeString() + '</p>';
						}
					}
				} catch (error) {
					console.error('Failed to load real-time metrics:', error);
				}
			}
			loadRealtimeMetrics();
			setInterval(loadRealtimeMetrics, 30000); // Refresh every 30 seconds
		</script>`

	return dateRangeHTML + overviewHTML + topArticlesHTML + trafficHTML + exportHTML + realTimeHTML
}

// buildErrorContent builds HTML content for error state
func (s *Server) buildErrorContent(errorMsg string) string {
	return fmt.Sprintf(`
		<div class="dashboard-card" style="grid-column: 1 / -1;">
			<div class="card-title">⚠️ Error Loading Analytics</div>
			<div>
				<p style="color: #dc3545;">%s</p>
				<p>Please check the database connection and try again.</p>
				<button onclick="location.reload()" style="padding: 0.5rem 1rem; background: #3b82f6; color: white; border: none; border-radius: 4px; cursor: pointer; margin-top: 1rem;">Retry</button>
			</div>
		</div>`, errorMsg)
}

// buildFallbackContent builds HTML content when analytics service is not available
func (s *Server) buildFallbackContent() string {
	return `
		<div class="dashboard-card" style="grid-column: 1 / -1;">
			<div class="card-title">📊 Analytics Dashboard</div>
			<div>
				<p>Analytics service is initializing...</p>
				<p>Real analytics data will be displayed once the service is fully loaded.</p>
				<p style="margin-top: 1rem;">In the meantime, you can:</p>
				<ul style="margin-left: 1.5rem; margin-top: 0.5rem;">
					<li>Check the <a href="/admin/system">System Monitor</a> for service status</li>
					<li>View the <a href="/api/v1/monitoring/dashboard" target="_blank">Monitoring Dashboard API</a></li>
				</ul>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">📈 Quick Stats (API)</div>
			<div id="api-stats">
				<p>Loading from API...</p>
			</div>
		</div>

		<script>
			async function loadAPIStats() {
				try {
					const response = await fetch('/api/v1/monitoring/dashboard');
					if (response.ok) {
						const data = await response.json();
						const content = document.getElementById('api-stats');
						content.innerHTML = 
							'<p>System Health: <span class="metric" style="color: ' + (data.system_health === 'healthy' ? '#28a745' : '#dc3545') + ';">' + data.system_health + '</span></p>' +
							'<p>CPU Usage: <span class="metric">' + (data.system_metrics?.cpu_usage?.toFixed(1) || 0) + '%</span></p>' +
							'<p>Memory Usage: <span class="metric">' + (data.system_metrics?.memory_usage?.toFixed(1) || 0) + '%</span></p>' +
							'<p>Cache Hit Rate: <span class="metric">' + ((data.cache_metrics?.hit_rate || 0) * 100).toFixed(1) + '%</span></p>';
					}
				} catch (error) {
					console.error('Failed to load API stats:', error);
				}
			}
			loadAPIStats();
		</script>`
}

// renderAdminAnalyticsAPI returns analytics data as JSON for API calls
func (s *Server) renderAdminAnalyticsAPI(c *gin.Context) {
	if s.analyticsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Analytics service not available",
		})
		return
	}

	ctx := context.Background()
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// Parse date range from query params
	if start := c.Query("start"); start != "" {
		if parsed, err := time.Parse("2006-01-02", start); err == nil {
			startDate = parsed
		}
	}
	if end := c.Query("end"); end != "" {
		if parsed, err := time.Parse("2006-01-02", end); err == nil {
			endDate = parsed
		}
	}

	metrics, err := s.analyticsService.GetDashboardMetrics(ctx, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get analytics metrics",
			"details": err.Error(),
		})
		return
	}

	topArticles, _ := s.analyticsService.GetTopArticles(ctx, startDate, endDate, 10)
	trafficSources, _ := s.analyticsService.GetTrafficSources(ctx, startDate, endDate)

	c.JSON(http.StatusOK, gin.H{
		"metrics":         metrics,
		"top_articles":    topArticles,
		"traffic_sources": trafficSources,
		"date_range": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	})
}
