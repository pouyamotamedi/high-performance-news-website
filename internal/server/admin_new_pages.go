package server

import (
	"github.com/gin-gonic/gin"
)

// ============================================================================
// DASHBOARD
// ============================================================================

// renderAdminDashboard renders the main dashboard with real-time stats
func (s *Server) renderAdminDashboard(c *gin.Context) {
	content := `
	<div class="dashboard-grid" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1.5rem; margin-bottom: 2rem;">
		<div class="dashboard-card">
			<div class="card-title">📰 Total Articles</div>
			<div class="metric" id="totalArticles">-</div>
			<p style="color: #64748b; font-size: 0.875rem;">Published content</p>
		</div>
		<div class="dashboard-card">
			<div class="card-title">👁️ Today's Views</div>
			<div class="metric" id="todayViews">-</div>
			<p style="color: #64748b; font-size: 0.875rem;">Page views today</p>
		</div>
		<div class="dashboard-card">
			<div class="card-title">👥 Active Users</div>
			<div class="metric" id="activeUsers">-</div>
			<p style="color: #64748b; font-size: 0.875rem;">Currently online</p>
		</div>
		<div class="dashboard-card">
			<div class="card-title">💬 Pending Comments</div>
			<div class="metric" id="pendingComments">-</div>
			<p style="color: #64748b; font-size: 0.875rem;">Awaiting moderation</p>
		</div>
	</div>

	<div style="display: grid; grid-template-columns: 2fr 1fr; gap: 1.5rem;">
		<div class="dashboard-card">
			<div class="card-title">📈 Traffic Overview (Last 7 Days)</div>
			<div id="trafficChart" style="height: 300px; display: flex; align-items: center; justify-content: center; color: #64748b;">
				Loading chart...
			</div>
		</div>
		<div class="dashboard-card">
			<div class="card-title">🔥 Top Articles</div>
			<div id="topArticles" style="max-height: 300px; overflow-y: auto;">
				<p style="color: #64748b;">Loading...</p>
			</div>
		</div>
	</div>

	<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1.5rem; margin-top: 1.5rem;">
		<div class="dashboard-card">
			<div class="card-title">⚡ Quick Actions</div>
			<div style="display: flex; flex-direction: column; gap: 0.5rem;">
				<a href="/admin/content/create" class="action-button">📝 Create New Article</a>
				<a href="/admin/comments" class="action-button">💬 Moderate Comments</a>
				<a href="/admin/content-ingestion" class="action-button">📥 Process Pending Content</a>
				<a href="/admin/analytics" class="action-button">📊 View Analytics</a>
			</div>
		</div>
		<div class="dashboard-card">
			<div class="card-title">🖥️ System Status</div>
			<div id="systemStatus">
				<p>Database: <span class="status-badge status-healthy">Healthy</span></p>
				<p>Cache: <span class="status-badge status-healthy">Operational</span></p>
				<p>Search: <span class="status-badge status-healthy">Available</span></p>
				<p>CDN: <span class="status-badge status-healthy">Connected</span></p>
			</div>
		</div>
		<div class="dashboard-card">
			<div class="card-title">📅 Recent Activity</div>
			<div id="recentActivity" style="max-height: 200px; overflow-y: auto;">
				<p style="color: #64748b; font-size: 0.875rem;">• Admin logged in</p>
				<p style="color: #64748b; font-size: 0.875rem;">• System started successfully</p>
				<p style="color: #64748b; font-size: 0.875rem;">• Cache cleared</p>
			</div>
		</div>
	</div>

	<script>
		async function loadDashboardData() {
			const token = localStorage.getItem('auth_token');
			const headers = { 'Authorization': 'Bearer ' + token };
			
			try {
				// Load monitoring dashboard
				const monitorRes = await fetch('/api/v1/monitoring/dashboard');
				if (monitorRes.ok) {
					const data = await monitorRes.json();
					document.getElementById('activeUsers').textContent = data.active_users || '0';
				}
				
				// Load analytics overview
				const analyticsRes = await fetch('/api/v1/analytics/overview', { headers });
				if (analyticsRes.ok) {
					const data = await analyticsRes.json();
					if (data.data) {
						document.getElementById('totalArticles').textContent = data.data.total_articles || '0';
						document.getElementById('todayViews').textContent = data.data.total_views || '0';
					}
				}
				
				// Load comment stats
				const commentsRes = await fetch('/api/v1/admin/comments/stats', { headers });
				if (commentsRes.ok) {
					const data = await commentsRes.json();
					if (data.stats) {
						document.getElementById('pendingComments').textContent = data.stats.pending || '0';
					}
				}
			} catch (error) {
				console.error('Failed to load dashboard data:', error);
			}
		}
		
		document.addEventListener('DOMContentLoaded', loadDashboardData);
	</script>`

	s.renderAdminPage(c, "Dashboard", "dashboard", content)
}

// ============================================================================
// ADVERTISEMENTS
// ============================================================================

func (s *Server) handleAdminAdsCampaigns(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📢 Advertisement Campaigns</div>
		<div style="margin-bottom: 1rem;">
			<button class="action-button" onclick="createCampaign()">➕ Create Campaign</button>
		</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/ads/campaigns</code>
		</p>
		<div id="campaignsList" style="margin-top: 1rem;">
			<p style="color: #64748b;">Loading campaigns...</p>
		</div>
	</div>
	<script>
		async function loadCampaigns() {
			// Connect to /api/v1/ads/campaigns
			document.getElementById('campaignsList').innerHTML = '<p style="color: #64748b;">No campaigns found. Create your first campaign.</p>';
		}
		loadCampaigns();
	</script>`
	s.renderAdminPage(c, "Ad Campaigns", "advertisements", content)
}

func (s *Server) handleAdminAdsSlots(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📍 Advertisement Slots</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/ads/slots</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Manage ad placement slots across your website (header, sidebar, in-content, footer).</p>
	</div>`
	s.renderAdminPage(c, "Ad Slots", "advertisements", content)
}

func (s *Server) handleAdminAdsCreatives(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🎨 Ad Creatives</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/ads/creatives</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Upload and manage advertisement images, banners, and HTML creatives.</p>
	</div>`
	s.renderAdminPage(c, "Ad Creatives", "advertisements", content)
}

func (s *Server) handleAdminAdsTargeting(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🎯 Ad Targeting</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/ads/targeting</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Configure targeting rules based on categories, tags, user behavior, and demographics.</p>
	</div>`
	s.renderAdminPage(c, "Ad Targeting", "advertisements", content)
}

func (s *Server) handleAdminAdsAnalytics(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📊 Ad Analytics</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/ads/reports/performance</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View impressions, clicks, CTR, and revenue analytics for your ad campaigns.</p>
	</div>`
	s.renderAdminPage(c, "Ad Analytics", "advertisements", content)
}

// ============================================================================
// PUSH NOTIFICATIONS
// ============================================================================

func (s *Server) handleAdminPushSend(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔔 Send Push Notification</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/push/admin/notifications</code>
		</p>
		<form style="margin-top: 1rem;">
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Title</label>
				<input type="text" style="width: 100%; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="Notification title">
			</div>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Message</label>
				<textarea style="width: 100%; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px; min-height: 100px;" placeholder="Notification message"></textarea>
			</div>
			<button type="button" class="action-button">📤 Send Notification</button>
		</form>
	</div>`
	s.renderAdminPage(c, "Send Notification", "push-notifications", content)
}

func (s *Server) handleAdminPushTemplates(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📋 Notification Templates</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/push/admin/templates</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Create reusable notification templates for breaking news, updates, and promotions.</p>
	</div>`
	s.renderAdminPage(c, "Notification Templates", "push-notifications", content)
}

func (s *Server) handleAdminPushSubscribers(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">👥 Push Subscribers</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/push/admin/subscriptions</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View and manage push notification subscribers and their preferences.</p>
	</div>`
	s.renderAdminPage(c, "Push Subscribers", "push-notifications", content)
}

func (s *Server) handleAdminPushAnalytics(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📈 Push Analytics</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/push/admin/analytics/overview</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Track delivery rates, open rates, and click-through rates for push notifications.</p>
	</div>`
	s.renderAdminPage(c, "Push Analytics", "push-notifications", content)
}

// ============================================================================
// SOCIAL MEDIA
// ============================================================================

func (s *Server) handleAdminSocialAccounts(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔗 Connected Social Accounts</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/admin/social-media/credentials</code>
		</p>
		<div style="margin-top: 1rem; display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
			<div style="padding: 1rem; border: 1px solid #e2e8f0; border-radius: 8px; text-align: center;">
				<span style="font-size: 2rem;">📘</span>
				<p style="font-weight: 500; margin: 0.5rem 0;">Facebook</p>
				<span class="status-badge status-draft">Not Connected</span>
			</div>
			<div style="padding: 1rem; border: 1px solid #e2e8f0; border-radius: 8px; text-align: center;">
				<span style="font-size: 2rem;">🐦</span>
				<p style="font-weight: 500; margin: 0.5rem 0;">Twitter/X</p>
				<span class="status-badge status-draft">Not Connected</span>
			</div>
			<div style="padding: 1rem; border: 1px solid #e2e8f0; border-radius: 8px; text-align: center;">
				<span style="font-size: 2rem;">✈️</span>
				<p style="font-weight: 500; margin: 0.5rem 0;">Telegram</p>
				<span class="status-badge status-draft">Not Connected</span>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "Social Accounts", "social-media", content)
}

func (s *Server) handleAdminSocialAutoPublish(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🤖 Auto Publish Settings</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/admin/articles/:id/publish-all-social</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Configure automatic publishing of articles to connected social media platforms.</p>
	</div>`
	s.renderAdminPage(c, "Auto Publish", "social-media", content)
}

func (s *Server) handleAdminSocialScheduled(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📅 Scheduled Posts</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/admin/articles/:id/schedule-social</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View and manage scheduled social media posts.</p>
	</div>`
	s.renderAdminPage(c, "Scheduled Posts", "social-media", content)
}

func (s *Server) handleAdminSocialAnalytics(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📊 Social Media Analytics</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Social media analytics will track engagement, reach, and performance of your social posts across all connected platforms.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "Social Analytics", "social-media", content)
}

// ============================================================================
// NEWSLETTER
// ============================================================================

func (s *Server) handleAdminNewsletter(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">✉️ Newsletter Management</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Newsletter management will allow you to create email campaigns, manage subscribers, design templates, and track email analytics. Integration with popular email services like SendGrid, Mailchimp, or Amazon SES will be supported.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "Newsletter", "newsletter", content)
}


// ============================================================================
// ANALYTICS
// ============================================================================

func (s *Server) handleAdminAnalyticsTraffic(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🚦 Traffic Analytics</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/analytics/overview</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View detailed traffic analytics including page views, unique visitors, bounce rate, and traffic sources.</p>
	</div>`
	s.renderAdminPage(c, "Traffic Analytics", "analytics", content)
}

func (s *Server) handleAdminAnalyticsContent(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📄 Content Performance</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/analytics/articles</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Analyze article performance, engagement metrics, and content trends.</p>
	</div>`
	s.renderAdminPage(c, "Content Performance", "analytics", content)
}

func (s *Server) handleAdminAnalyticsAudience(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">👥 Audience Insights</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/analytics/users</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Understand your audience demographics, behavior patterns, and preferences.</p>
	</div>`
	s.renderAdminPage(c, "Audience Insights", "analytics", content)
}

func (s *Server) handleAdminAnalyticsRealtime(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">⚡ Real-time Analytics</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/monitoring/dashboard</code>
		</p>
		<div id="realtimeStats" style="margin-top: 1rem;">
			<p>Active Users: <span class="metric" id="activeNow">-</span></p>
			<p>Page Views (last hour): <span class="metric" id="viewsHour">-</span></p>
		</div>
	</div>
	<script>
		async function loadRealtime() {
			try {
				const res = await fetch('/api/v1/monitoring/dashboard');
				if (res.ok) {
					const data = await res.json();
					document.getElementById('activeNow').textContent = data.active_users || '0';
				}
			} catch (e) { console.error(e); }
		}
		loadRealtime();
		setInterval(loadRealtime, 30000);
	</script>`
	s.renderAdminPage(c, "Real-time Analytics", "analytics", content)
}

// ============================================================================
// SEO
// ============================================================================

func (s *Server) handleAdminSEOSettings(c *gin.Context) {
	// Serve the SEO settings template file
	c.File("web/templates/admin/seo_settings.html")
}

func (s *Server) handleAdminSEOOverview(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔍 SEO Overview</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoints: <code>/api/v1/seo/*</code>
		</p>
		<div style="margin-top: 1rem; display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
			<div style="padding: 1rem; background: #f8fafc; border-radius: 8px;">
				<p style="font-weight: 500;">Sitemap Status</p>
				<span class="status-badge status-healthy">Active</span>
			</div>
			<div style="padding: 1rem; background: #f8fafc; border-radius: 8px;">
				<p style="font-weight: 500;">Google News</p>
				<span class="status-badge status-healthy">Configured</span>
			</div>
			<div style="padding: 1rem; background: #f8fafc; border-radius: 8px;">
				<p style="font-weight: 500;">Schema Markup</p>
				<span class="status-badge status-healthy">Enabled</span>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "SEO Overview", "seo", content)
}

func (s *Server) handleAdminSEOSitemap(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🗺️ Sitemap Management</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			Sitemap URL: <code>/sitemap.xml</code>
		</p>
		<div style="margin-top: 1rem;">
			<p><strong>Available Sitemaps:</strong></p>
			<ul style="margin-left: 1.5rem; color: #64748b;">
				<li><a href="/sitemap.xml" target="_blank">/sitemap.xml</a> - Main sitemap index</li>
				<li><a href="/sitemap-main.xml" target="_blank">/sitemap-main.xml</a> - Static pages</li>
				<li><a href="/sitemap-articles-1.xml" target="_blank">/sitemap-articles-1.xml</a> - Articles</li>
				<li><a href="/sitemap-news-1.xml" target="_blank">/sitemap-news-1.xml</a> - Google News</li>
				<li><a href="/sitemap-categories.xml" target="_blank">/sitemap-categories.xml</a> - Categories</li>
				<li><a href="/sitemap-tags.xml" target="_blank">/sitemap-tags.xml</a> - Tags</li>
			</ul>
		</div>
	</div>`
	s.renderAdminPage(c, "Sitemap", "seo", content)
}

func (s *Server) handleAdminSEOGoogleNews(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📰 Google News Settings</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/google-news/*</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Configure Google News sitemap settings, publication name, and article eligibility rules.</p>
	</div>`
	s.renderAdminPage(c, "Google News", "seo", content)
}

func (s *Server) handleAdminSEOSchema(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📋 Schema Markup</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoints: <code>/api/v1/seo/schema/*</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Manage structured data (JSON-LD) for articles, organization, and breadcrumbs.</p>
	</div>`
	s.renderAdminPage(c, "Schema Markup", "seo", content)
}

func (s *Server) handleAdminSEORedirects(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">↪️ URL Redirects</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>URL redirect management will allow you to create 301/302 redirects, manage broken links, and set up redirect rules for SEO optimization.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "URL Redirects", "seo", content)
}

// ============================================================================
// APPEARANCE
// ============================================================================

func (s *Server) handleAdminMenus(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📋 Menu Management</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Menu management will allow you to create and organize navigation menus, add custom links, and arrange menu items with drag-and-drop functionality.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "Menus", "menus", content)
}

// ============================================================================
// DISTRIBUTION
// ============================================================================

func (s *Server) handleAdminRSS(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📡 RSS Feeds</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			RSS Endpoints available at <code>/rss/*</code>
		</p>
		<div style="margin-top: 1rem;">
			<p><strong>Available RSS Feeds:</strong></p>
			<ul style="margin-left: 1.5rem; color: #64748b;">
				<li><a href="/rss" target="_blank">/rss</a> - Main feed (all articles)</li>
				<li><a href="/rss/category/technology" target="_blank">/rss/category/:slug</a> - Category feeds</li>
				<li><a href="/rss/tag/news" target="_blank">/rss/tag/:slug</a> - Tag feeds</li>
			</ul>
		</div>
	</div>`
	s.renderAdminPage(c, "RSS Feeds", "rss", content)
}

func (s *Server) handleAdminCDNConfig(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">⚙️ CDN Configuration</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/admin/cdn/config</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Configure CDN settings, origin servers, and caching rules.</p>
	</div>`
	s.renderAdminPage(c, "CDN Configuration", "cdn", content)
}

func (s *Server) handleAdminCDNPurge(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🗑️ Purge CDN Cache</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoints: <code>/api/v1/admin/cdn/purge/*</code>
		</p>
		<div style="margin-top: 1rem;">
			<button class="action-button" style="background: #dc2626;">🗑️ Purge All Cache</button>
			<p style="margin-top: 1rem; color: #64748b;">Or purge specific URLs:</p>
			<input type="text" style="width: 100%; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px; margin-top: 0.5rem;" placeholder="Enter URL to purge">
		</div>
	</div>`
	s.renderAdminPage(c, "Purge CDN Cache", "cdn", content)
}

func (s *Server) handleAdminCDNStats(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📊 CDN Statistics</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/admin/cdn/stats</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View CDN performance metrics, cache hit rates, and bandwidth usage.</p>
	</div>`
	s.renderAdminPage(c, "CDN Statistics", "cdn", content)
}

// ============================================================================
// BACKUP & RESTORE
// ============================================================================

func (s *Server) handleAdminBackupCreate(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">💾 Create Backup</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoints: <code>/api/v1/admin/backup/full</code>, <code>/api/v1/admin/backup/incremental</code>
		</p>
		<div style="margin-top: 1rem; display: flex; gap: 1rem;">
			<button class="action-button">💾 Full Backup</button>
			<button class="action-button" style="background: #059669;">📦 Incremental Backup</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Create Backup", "backup", content)
}

func (s *Server) handleAdminBackupList(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📋 Backup History</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/admin/backup/list</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">View all previous backups with timestamps, sizes, and status.</p>
	</div>`
	s.renderAdminPage(c, "Backup History", "backup", content)
}

func (s *Server) handleAdminBackupRestore(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔄 Restore from Backup</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/admin/backup/restore</code>
		</p>
		<p style="margin-top: 1rem; color: #dc2626; font-weight: 500;">⚠️ Warning: Restoring a backup will overwrite current data.</p>
	</div>`
	s.renderAdminPage(c, "Restore Backup", "backup", content)
}

func (s *Server) handleAdminBackupSchedule(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">⏰ Backup Schedule</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoints: <code>/api/v1/admin/backup/scheduler/*</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Configure automatic backup schedules (daily, weekly, monthly).</p>
	</div>`
	s.renderAdminPage(c, "Backup Schedule", "backup", content)
}

// ============================================================================
// SYSTEM - renderAdminSystem is defined in admin_system.go
// ============================================================================

func (s *Server) handleAdminLogs(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📜 System Logs</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>System logs viewer will display application logs, error logs, access logs, and audit trails with filtering and search capabilities.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "System Logs", "logs", content)
}

func (s *Server) handleAdminSettingsEmail(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">✉️ Email Settings</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Email settings will allow you to configure SMTP servers, email templates, and notification preferences.</em>
		</p>
	</div>`
	s.renderAdminPage(c, "Email Settings", "settings", content)
}

func (s *Server) handleAdminSettingsAPI(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔑 API Keys</div>
		<p style="background: #dbeafe; padding: 1rem; border-radius: 8px; color: #1e40af;">
			<strong>ℹ️ This feature should be connected to related API</strong><br>
			API Endpoint: <code>/api/v1/auth/api-key/*</code>
		</p>
		<p style="margin-top: 1rem; color: #64748b;">Manage API keys for external integrations and third-party access.</p>
	</div>`
	s.renderAdminPage(c, "API Keys", "settings", content)
}


// ============================================================================
// THEMES & WIDGETS (Already have backend APIs)
// ============================================================================

func (s *Server) handleAdminThemes(c *gin.Context) {
	content := `
	<style>
		.theme-tabs { display: flex; gap: 0; border-bottom: 2px solid #e2e8f0; margin-bottom: 1.5rem; }
		.theme-tab { padding: 0.75rem 1.5rem; background: none; border: none; cursor: pointer; font-weight: 500; color: #64748b; border-bottom: 2px solid transparent; margin-bottom: -2px; transition: all 0.2s; }
		.theme-tab:hover { color: #3b82f6; }
		.theme-tab.active { color: #3b82f6; border-bottom-color: #3b82f6; }
		.tab-content { display: none; }
		.tab-content.active { display: block; }
		.settings-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1.5rem; }
		.settings-section { background: #f8fafc; border-radius: 12px; padding: 1.5rem; }
		.settings-section h3 { margin: 0 0 1rem 0; font-size: 1rem; color: #1e293b; display: flex; align-items: center; gap: 0.5rem; }
		.form-group { margin-bottom: 1rem; }
		.form-label { display: block; margin-bottom: 0.5rem; font-weight: 500; color: #374151; font-size: 0.875rem; }
		.form-input, .form-select { width: 100%; padding: 0.625rem 0.75rem; border: 1px solid #e2e8f0; border-radius: 8px; font-size: 0.9375rem; background: #fff; box-sizing: border-box; }
		.form-input:focus, .form-select:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.1); }
		.color-input-group { display: flex; gap: 0.5rem; align-items: center; }
		.color-input-group input[type="color"] { width: 40px; height: 36px; padding: 2px; border: 1px solid #e2e8f0; border-radius: 6px; cursor: pointer; }
		.color-input-group input[type="text"] { flex: 1; }
		.checkbox-label { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; }
		.checkbox-label input { width: 18px; height: 18px; }
		.save-bar { position: sticky; bottom: 0; background: #fff; padding: 1rem; border-top: 1px solid #e2e8f0; margin: 1.5rem -1.5rem -1.5rem; display: flex; justify-content: space-between; align-items: center; }
		.btn-save { background: #3b82f6; color: #fff; padding: 0.75rem 2rem; border: none; border-radius: 8px; font-weight: 600; cursor: pointer; }
		.btn-save:hover { background: #2563eb; }
		.btn-save:disabled { background: #94a3b8; cursor: not-allowed; }
		.status-msg { padding: 0.5rem 1rem; border-radius: 6px; font-size: 0.875rem; }
		.status-msg.success { background: #dcfce7; color: #166534; }
		.status-msg.error { background: #fee2e2; color: #dc2626; }
		.logo-upload-group { display: flex; gap: 0.5rem; align-items: flex-start; }
		.logo-upload-group .form-input { flex: 1; }
		.btn-upload { background: #f1f5f9; color: #475569; padding: 0.625rem 1rem; border: 1px solid #e2e8f0; border-radius: 8px; cursor: pointer; white-space: nowrap; font-size: 0.875rem; }
		.btn-upload:hover { background: #e2e8f0; }
		.logo-preview { width: 120px; height: 60px; border: 2px dashed #e2e8f0; border-radius: 8px; display: flex; align-items: center; justify-content: center; margin-top: 0.5rem; overflow: hidden; background: #fff; }
		.logo-preview img { max-width: 100%; max-height: 100%; object-fit: contain; }
		.logo-preview.empty { color: #94a3b8; font-size: 0.75rem; }
		.upload-hint { font-size: 0.75rem; color: #64748b; margin-top: 0.25rem; }
	</style>
	
	<div class="dashboard-card">
		<div class="card-title">🎨 Theme Settings</div>
		
		<div class="theme-tabs">
			<button class="theme-tab active" data-tab="header">Header</button>
			<button class="theme-tab" data-tab="colors">Colors</button>
			<button class="theme-tab" data-tab="typography">Typography</button>
			<button class="theme-tab" data-tab="layout">Layout</button>
		</div>
		
		<!-- Header Settings Tab -->
		<div class="tab-content active" id="tab-header">
			<div class="settings-grid">
				<div class="settings-section">
					<h3>🏷️ Site Branding</h3>
					<div class="form-group">
						<label class="form-label">Site Name</label>
						<input type="text" class="form-input" id="siteName" placeholder="My News Website">
					</div>
					<div class="form-group">
						<label class="form-label">Site Description</label>
						<input type="text" class="form-input" id="siteDescription" placeholder="Your trusted source for news">
					</div>
					<div class="form-group">
						<label class="checkbox-label">
							<input type="checkbox" id="showSiteName"> Show site name in header
						</label>
					</div>
				</div>
				
				<div class="settings-section">
					<h3>🖼️ Logo</h3>
					<div class="form-group">
						<label class="form-label">Logo</label>
						<div class="logo-upload-group">
							<input type="text" class="form-input" id="logoUrl" placeholder="/static/images/logo.svg">
							<input type="file" id="logoFile" accept="image/*" style="display:none">
							<button type="button" class="btn-upload" onclick="document.getElementById('logoFile').click()">📁 Upload</button>
						</div>
						<div class="upload-hint">Enter URL or upload an image file (PNG, SVG, JPG)</div>
						<div class="logo-preview empty" id="logoPreview">No logo</div>
					</div>
					<div class="form-group">
						<label class="form-label">Favicon</label>
						<div class="logo-upload-group">
							<input type="text" class="form-input" id="faviconUrl" placeholder="/static/images/favicon.ico">
							<input type="file" id="faviconFile" accept="image/*,.ico" style="display:none">
							<button type="button" class="btn-upload" onclick="document.getElementById('faviconFile').click()">📁 Upload</button>
						</div>
						<div class="upload-hint">Enter URL or upload an icon file (ICO, PNG)</div>
					</div>
				</div>
				
				<div class="settings-section">
					<h3>📐 Header Layout</h3>
					<div class="form-group">
						<label class="form-label">Header Style</label>
						<select class="form-select" id="headerStyle">
							<option value="sticky">Sticky (stays on top when scrolling)</option>
							<option value="fixed">Fixed (always visible)</option>
							<option value="static">Static (scrolls with page)</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label">Header Height</label>
						<input type="text" class="form-input" id="headerHeight" placeholder="80px">
					</div>
				</div>
			</div>
		</div>
		
		<!-- Colors Tab -->
		<div class="tab-content" id="tab-colors">
			<div class="settings-grid">
				<div class="settings-section">
					<h3>🎨 Primary Colors</h3>
					<div class="form-group">
						<label class="form-label">Primary Color</label>
						<div class="color-input-group">
							<input type="color" id="colorPrimaryPicker" value="#3b82f6">
							<input type="text" class="form-input" id="colorPrimary" value="#3b82f6">
						</div>
					</div>
					<div class="form-group">
						<label class="form-label">Secondary Color</label>
						<div class="color-input-group">
							<input type="color" id="colorSecondaryPicker" value="#64748b">
							<input type="text" class="form-input" id="colorSecondary" value="#64748b">
						</div>
					</div>
					<div class="form-group">
						<label class="form-label">Accent Color</label>
						<div class="color-input-group">
							<input type="color" id="colorAccentPicker" value="#f59e0b">
							<input type="text" class="form-input" id="colorAccent" value="#f59e0b">
						</div>
					</div>
				</div>
				
				<div class="settings-section">
					<h3>📝 Text & Background</h3>
					<div class="form-group">
						<label class="form-label">Background Color</label>
						<div class="color-input-group">
							<input type="color" id="colorBackgroundPicker" value="#ffffff">
							<input type="text" class="form-input" id="colorBackground" value="#ffffff">
						</div>
					</div>
					<div class="form-group">
						<label class="form-label">Text Color</label>
						<div class="color-input-group">
							<input type="color" id="colorTextPicker" value="#1e293b">
							<input type="text" class="form-input" id="colorText" value="#1e293b">
						</div>
					</div>
					<div class="form-group">
						<label class="form-label">Border Color</label>
						<div class="color-input-group">
							<input type="color" id="colorBorderPicker" value="#e2e8f0">
							<input type="text" class="form-input" id="colorBorder" value="#e2e8f0">
						</div>
					</div>
				</div>
			</div>
		</div>
		
		<!-- Typography Tab -->
		<div class="tab-content" id="tab-typography">
			<div class="settings-grid">
				<div class="settings-section">
					<h3>🔤 Fonts</h3>
					<div class="form-group">
						<label class="form-label">Body Font Family</label>
						<select class="form-select" id="fontFamily">
							<option value="Inter, system-ui, sans-serif">Inter</option>
							<option value="Roboto, system-ui, sans-serif">Roboto</option>
							<option value="Open Sans, system-ui, sans-serif">Open Sans</option>
							<option value="Lato, system-ui, sans-serif">Lato</option>
							<option value="system-ui, sans-serif">System Default</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label">Heading Font Family</label>
						<select class="form-select" id="headingFont">
							<option value="Inter, system-ui, sans-serif">Inter</option>
							<option value="Roboto, system-ui, sans-serif">Roboto</option>
							<option value="Open Sans, system-ui, sans-serif">Open Sans</option>
							<option value="Playfair Display, serif">Playfair Display</option>
							<option value="system-ui, sans-serif">System Default</option>
						</select>
					</div>
				</div>
				
				<div class="settings-section">
					<h3>📏 Sizing</h3>
					<div class="form-group">
						<label class="form-label">Base Font Size</label>
						<select class="form-select" id="baseFontSize">
							<option value="14px">14px (Small)</option>
							<option value="16px">16px (Default)</option>
							<option value="18px">18px (Large)</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label">Line Height</label>
						<select class="form-select" id="lineHeight">
							<option value="1.4">1.4 (Compact)</option>
							<option value="1.6">1.6 (Default)</option>
							<option value="1.8">1.8 (Relaxed)</option>
						</select>
					</div>
				</div>
			</div>
		</div>
		
		<!-- Layout Tab -->
		<div class="tab-content" id="tab-layout">
			<div class="settings-grid">
				<div class="settings-section">
					<h3>📐 Container</h3>
					<div class="form-group">
						<label class="form-label">Max Width</label>
						<select class="form-select" id="maxWidth">
							<option value="1024px">1024px (Narrow)</option>
							<option value="1200px">1200px (Default)</option>
							<option value="1400px">1400px (Wide)</option>
							<option value="100%">Full Width</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label">Border Radius</label>
						<select class="form-select" id="borderRadius">
							<option value="0">None (Square)</option>
							<option value="4px">4px (Subtle)</option>
							<option value="8px">8px (Default)</option>
							<option value="12px">12px (Rounded)</option>
							<option value="16px">16px (Very Rounded)</option>
						</select>
					</div>
				</div>
				
				<div class="settings-section">
					<h3>📊 Sidebar</h3>
					<div class="form-group">
						<label class="checkbox-label">
							<input type="checkbox" id="showSidebar"> Show Sidebar
						</label>
					</div>
					<div class="form-group">
						<label class="form-label">Sidebar Position</label>
						<select class="form-select" id="sidebarPosition">
							<option value="right">Right</option>
							<option value="left">Left</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label">Sidebar Width</label>
						<input type="text" class="form-input" id="sidebarWidth" placeholder="300px">
					</div>
				</div>
			</div>
		</div>
		
		<div class="save-bar">
			<div id="statusMsg"></div>
			<button class="btn-save" id="saveBtn" onclick="saveTheme()">💾 Save Changes</button>
		</div>
	</div>
	
	<script>
		const token = localStorage.getItem('auth_token');
		let currentTheme = null;
		
		// Tab switching
		document.querySelectorAll('.theme-tab').forEach(tab => {
			tab.addEventListener('click', () => {
				document.querySelectorAll('.theme-tab').forEach(t => t.classList.remove('active'));
				document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
				tab.classList.add('active');
				document.getElementById('tab-' + tab.dataset.tab).classList.add('active');
			});
		});
		
		// Color picker sync
		['Primary', 'Secondary', 'Accent', 'Background', 'Text', 'Border'].forEach(name => {
			const picker = document.getElementById('color' + name + 'Picker');
			const input = document.getElementById('color' + name);
			if (picker && input) {
				picker.addEventListener('input', () => input.value = picker.value);
				input.addEventListener('input', () => picker.value = input.value);
			}
		});
		
		// Logo preview
		document.getElementById('logoUrl').addEventListener('input', function() {
			const preview = document.getElementById('logoPreview');
			if (this.value) {
				preview.innerHTML = '<img src="' + this.value + '" alt="Logo preview" onerror="this.parentElement.innerHTML=\'Invalid URL\';this.parentElement.classList.add(\'empty\')">';
				preview.classList.remove('empty');
			} else {
				preview.innerHTML = 'No logo';
				preview.classList.add('empty');
			}
		});
		
		// File upload handlers
		document.getElementById('logoFile').addEventListener('change', async function(e) {
			if (e.target.files.length > 0) {
				await uploadFile(e.target.files[0], 'logoUrl');
			}
		});
		
		document.getElementById('faviconFile').addEventListener('change', async function(e) {
			if (e.target.files.length > 0) {
				await uploadFile(e.target.files[0], 'faviconUrl');
			}
		});
		
		async function uploadFile(file, targetInputId) {
			const formData = new FormData();
			formData.append('image', file);  // API expects 'image' field
			
			try {
				showStatus('Uploading...', 'success');
				const res = await fetch('/api/v1/images/upload', {  // Correct endpoint
					method: 'POST',
					headers: { 'Authorization': 'Bearer ' + token },
					body: formData
				});
				
				if (res.ok) {
					const data = await res.json();
					// API returns image object with variants
					let url = '';
					if (data.image && data.image.original_url) {
						url = data.image.original_url;
					} else if (data.variants && data.variants.length > 0) {
						// Use the largest variant URL
						const largeVariant = data.variants.find(v => v.size === 'large') || data.variants[0];
						url = largeVariant.url;
					} else if (data.url || data.file_url || data.path) {
						url = data.url || data.file_url || data.path;
					}
					
					if (url) {
						document.getElementById(targetInputId).value = url;
						document.getElementById(targetInputId).dispatchEvent(new Event('input'));
						showStatus('File uploaded successfully', 'success');
					} else {
						showStatus('Upload succeeded but no URL returned', 'error');
					}
				} else {
					const err = await res.json();
					showStatus('Upload failed: ' + (err.error || 'Unknown error'), 'error');
				}
			} catch (e) {
				console.error(e);
				showStatus('Upload error: ' + e.message, 'error');
			}
		}
		
		async function loadTheme() {
			try {
				const res = await fetch('/api/v1/admin/themes/active', {
					headers: { 'Authorization': 'Bearer ' + token }
				});
				if (res.ok) {
					const data = await res.json();
					// API returns theme directly, not wrapped in {theme: ...}
					currentTheme = data.id ? data : (data.theme || data);
					if (currentTheme && currentTheme.id) {
						populateForm(currentTheme);
						showStatus('Theme loaded', 'success');
					} else {
						showStatus('No active theme found', 'error');
					}
				} else {
					const err = await res.json();
					showStatus('Failed to load theme: ' + (err.error || 'Unknown error'), 'error');
				}
			} catch (e) {
				console.error(e);
				showStatus('Error loading theme: ' + e.message, 'error');
			}
		}
		
		function populateForm(theme) {
			const config = theme.config || {};
			const branding = config.branding || {};
			const layout = config.layout || {};
			const colors = config.colors || {};
			const typography = config.typography || {};
			
			// Header/Branding
			document.getElementById('siteName').value = branding.site_name || '';
			document.getElementById('siteDescription').value = branding.site_description || '';
			document.getElementById('showSiteName').checked = branding.show_site_name !== false;
			document.getElementById('logoUrl').value = branding.logo_url || '';
			document.getElementById('faviconUrl').value = branding.favicon_url || '';
			document.getElementById('headerStyle').value = layout.header_style || 'sticky';
			document.getElementById('headerHeight').value = layout.header_height || '80px';
			
			// Trigger logo preview
			document.getElementById('logoUrl').dispatchEvent(new Event('input'));
			
			// Colors
			const colorFields = ['Primary', 'Secondary', 'Accent', 'Background', 'Text', 'Border'];
			const colorKeys = ['primary', 'secondary', 'accent', 'background', 'text', 'border'];
			colorFields.forEach((field, i) => {
				const val = colors[colorKeys[i]] || '';
				const input = document.getElementById('color' + field);
				const picker = document.getElementById('color' + field + 'Picker');
				if (input) input.value = val;
				if (picker && val) picker.value = val;
			});
			
			// Typography
			document.getElementById('fontFamily').value = typography.font_family || 'Inter, system-ui, sans-serif';
			document.getElementById('headingFont').value = typography.heading_font || 'Inter, system-ui, sans-serif';
			document.getElementById('baseFontSize').value = typography.base_font_size || '16px';
			document.getElementById('lineHeight').value = String(typography.line_height || 1.6);
			
			// Layout
			document.getElementById('maxWidth').value = layout.max_width || '1200px';
			document.getElementById('borderRadius').value = layout.border_radius || '8px';
			document.getElementById('showSidebar').checked = layout.show_sidebar !== false;
			document.getElementById('sidebarPosition').value = layout.sidebar_position || 'right';
			document.getElementById('sidebarWidth').value = layout.sidebar_width || '300px';
		}
		
		async function saveTheme() {
			if (!currentTheme) {
				showStatus('No theme loaded', 'error');
				return;
			}
			
			const btn = document.getElementById('saveBtn');
			btn.disabled = true;
			btn.textContent = 'Saving...';
			
			// Build config object
			const config = {
				branding: {
					site_name: document.getElementById('siteName').value,
					site_description: document.getElementById('siteDescription').value,
					show_site_name: document.getElementById('showSiteName').checked,
					show_description: true,
					logo_url: document.getElementById('logoUrl').value,
					favicon_url: document.getElementById('faviconUrl').value
				},
				layout: {
					header_style: document.getElementById('headerStyle').value,
					header_height: document.getElementById('headerHeight').value,
					footer_style: currentTheme.config?.layout?.footer_style || 'static',
					footer_height: currentTheme.config?.layout?.footer_height || 'auto',
					max_width: document.getElementById('maxWidth').value,
					border_radius: document.getElementById('borderRadius').value,
					spacing: currentTheme.config?.layout?.spacing || '1rem',
					grid_columns: currentTheme.config?.layout?.grid_columns || 12,
					show_sidebar: document.getElementById('showSidebar').checked,
					sidebar_position: document.getElementById('sidebarPosition').value,
					sidebar_width: document.getElementById('sidebarWidth').value
				},
				colors: {
					primary: document.getElementById('colorPrimary').value,
					secondary: document.getElementById('colorSecondary').value,
					accent: document.getElementById('colorAccent').value,
					background: document.getElementById('colorBackground').value,
					text: document.getElementById('colorText').value,
					text_muted: currentTheme.config?.colors?.text_muted || '#64748b',
					border: document.getElementById('colorBorder').value,
					surface: currentTheme.config?.colors?.surface || '#f8fafc',
					success: currentTheme.config?.colors?.success || '#10b981',
					warning: currentTheme.config?.colors?.warning || '#f59e0b',
					error: currentTheme.config?.colors?.error || '#ef4444',
					info: currentTheme.config?.colors?.info || '#3b82f6'
				},
				typography: {
					font_family: document.getElementById('fontFamily').value,
					heading_font: document.getElementById('headingFont').value,
					base_font_size: document.getElementById('baseFontSize').value,
					line_height: parseFloat(document.getElementById('lineHeight').value),
					heading_weight: currentTheme.config?.typography?.heading_weight || '600',
					body_weight: currentTheme.config?.typography?.body_weight || '400',
					letter_spacing: currentTheme.config?.typography?.letter_spacing || '0'
				},
				custom_css: currentTheme.config?.custom_css || '',
				custom_js: currentTheme.config?.custom_js || ''
			};
			
			try {
				const res = await fetch('/api/v1/admin/themes/' + currentTheme.id, {
					method: 'PUT',
					headers: {
						'Authorization': 'Bearer ' + token,
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({
						name: currentTheme.name,
						description: currentTheme.description,
						config: config
					})
				});
				
				if (res.ok) {
					showStatus('✅ Theme saved successfully!', 'success');
					// Reload to get updated data
					await loadTheme();
				} else {
					const err = await res.json();
					showStatus('Failed to save: ' + (err.error || 'Unknown error'), 'error');
				}
			} catch (e) {
				console.error(e);
				showStatus('Error saving theme: ' + e.message, 'error');
			} finally {
				btn.disabled = false;
				btn.textContent = '💾 Save Changes';
			}
		}
		
		function showStatus(msg, type) {
			const el = document.getElementById('statusMsg');
			el.textContent = msg;
			el.className = 'status-msg ' + type;
			if (type === 'success') {
				setTimeout(() => { el.textContent = ''; el.className = ''; }, 3000);
			}
		}
		
		// Load theme on page load
		loadTheme();
	</script>`
	s.renderAdminPage(c, "Theme Settings", "themes", content)
}

func (s *Server) handleAdminWidgets(c *gin.Context) {
	content := `
	<style>
		.widget-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1rem; margin-top: 1rem; }
		.widget-card { background: #fff; border: 1px solid #e2e8f0; border-radius: 12px; padding: 1.5rem; transition: all 0.2s; }
		.widget-card:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.1); transform: translateY(-2px); }
		.widget-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1rem; }
		.widget-name { font-weight: 600; font-size: 1.1rem; color: #1e293b; }
		.widget-type { display: inline-block; padding: 0.25rem 0.75rem; background: #e0e7ff; color: #3730a3; border-radius: 20px; font-size: 0.75rem; font-weight: 500; }
		.widget-meta { display: flex; gap: 1rem; margin-top: 0.75rem; font-size: 0.875rem; color: #64748b; }
		.widget-actions { display: flex; gap: 0.5rem; margin-top: 1rem; padding-top: 1rem; border-top: 1px solid #e2e8f0; }
		.widget-btn { padding: 0.5rem 1rem; border: none; border-radius: 6px; cursor: pointer; font-size: 0.875rem; transition: all 0.2s; }
		.widget-btn-edit { background: #3b82f6; color: white; }
		.widget-btn-edit:hover { background: #2563eb; }
		.widget-btn-delete { background: #fee2e2; color: #dc2626; }
		.widget-btn-delete:hover { background: #fecaca; }
		.widget-btn-toggle { background: #f0fdf4; color: #16a34a; }
		.widget-btn-toggle.inactive { background: #fef3c7; color: #d97706; }
		.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: none; align-items: center; justify-content: center; z-index: 1000; }
		.modal-overlay.active { display: flex; }
		.modal-content { background: white; border-radius: 16px; padding: 2rem; max-width: 600px; width: 90%; max-height: 90vh; overflow-y: auto; }
		.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
		.modal-title { font-size: 1.25rem; font-weight: 600; }
		.modal-close { background: none; border: none; font-size: 1.5rem; cursor: pointer; color: #64748b; }
		.form-group { margin-bottom: 1rem; }
		.form-label { display: block; margin-bottom: 0.5rem; font-weight: 500; color: #374151; }
		.form-input, .form-select, .form-textarea { width: 100%; padding: 0.75rem; border: 1px solid #e2e8f0; border-radius: 8px; font-size: 1rem; }
		.form-input:focus, .form-select:focus, .form-textarea:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.1); }
		.form-textarea { min-height: 100px; resize: vertical; }
		.form-actions { display: flex; gap: 1rem; justify-content: flex-end; margin-top: 1.5rem; padding-top: 1rem; border-top: 1px solid #e2e8f0; }
		.btn-primary { background: #3b82f6; color: white; padding: 0.75rem 1.5rem; border: none; border-radius: 8px; cursor: pointer; font-weight: 500; }
		.btn-primary:hover { background: #2563eb; }
		.btn-secondary { background: #f1f5f9; color: #475569; padding: 0.75rem 1.5rem; border: none; border-radius: 8px; cursor: pointer; font-weight: 500; }
		.btn-secondary:hover { background: #e2e8f0; }
		.zone-tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap; }
		.zone-tab { padding: 0.5rem 1rem; border: 1px solid #e2e8f0; border-radius: 20px; cursor: pointer; font-size: 0.875rem; transition: all 0.2s; }
		.zone-tab.active { background: #3b82f6; color: white; border-color: #3b82f6; }
		.zone-tab:hover:not(.active) { background: #f1f5f9; }
		.empty-state { text-align: center; padding: 3rem; color: #64748b; }
		.empty-state-icon { font-size: 3rem; margin-bottom: 1rem; }
	</style>
	<div class="dashboard-card">
		<div class="card-title">📦 Widget Management</div>
		<p style="margin-bottom: 1rem; color: #64748b;">Manage widgets for different page zones (sidebar, footer, homepage sections)</p>
		
		<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
			<div class="zone-tabs" id="zoneTabs">
				<button class="zone-tab active" data-zone="all">All Widgets</button>
				<button class="zone-tab" data-zone="sidebar">Sidebar</button>
				<button class="zone-tab" data-zone="footer">Footer</button>
				<button class="zone-tab" data-zone="homepage">Homepage</button>
				<button class="zone-tab" data-zone="article">Article Page</button>
			</div>
			<button class="action-button" onclick="openCreateModal()">➕ Create Widget</button>
		</div>
		
		<div id="widgetsList" class="widget-grid">
			<p style="color: #64748b; grid-column: 1/-1; text-align: center; padding: 2rem;">Loading widgets...</p>
		</div>
	</div>
	
	<!-- Create/Edit Widget Modal -->
	<div class="modal-overlay" id="widgetModal">
		<div class="modal-content">
			<div class="modal-header">
				<h3 class="modal-title" id="modalTitle">Create Widget</h3>
				<button class="modal-close" onclick="closeModal()">&times;</button>
			</div>
			<form id="widgetForm" onsubmit="saveWidget(event)">
				<input type="hidden" id="widgetId">
				<div class="form-group">
					<label class="form-label">Widget Name *</label>
					<input type="text" class="form-input" id="widgetName" required placeholder="e.g., Popular Articles">
				</div>
				<div class="form-group">
					<label class="form-label">Widget Type *</label>
					<select class="form-select" id="widgetType" required onchange="updateConfigFields()">
						<option value="">Select type...</option>
						<option value="latest_articles">Latest Articles</option>
						<option value="popular_articles">Popular Articles</option>
						<option value="category_articles">Category Articles</option>
						<option value="tag_cloud">Tag Cloud</option>
						<option value="newsletter">Newsletter Signup</option>
						<option value="social_links">Social Links</option>
						<option value="custom_html">Custom HTML</option>
						<option value="advertisement">Advertisement</option>
					</select>
				</div>
				<div class="form-group">
					<label class="form-label">Zone *</label>
					<select class="form-select" id="widgetZone" required>
						<option value="">Select zone...</option>
						<option value="sidebar">Sidebar</option>
						<option value="footer">Footer</option>
						<option value="homepage_hero">Homepage Hero</option>
						<option value="homepage_main">Homepage Main</option>
						<option value="article_sidebar">Article Sidebar</option>
						<option value="article_bottom">Article Bottom</option>
					</select>
				</div>
				<div class="form-group">
					<label class="form-label">Position (Order)</label>
					<input type="number" class="form-input" id="widgetPosition" value="0" min="0">
				</div>
				<div class="form-group" id="configFields">
					<!-- Dynamic config fields based on widget type -->
				</div>
				<div class="form-group">
					<label style="display: flex; align-items: center; gap: 0.5rem;">
						<input type="checkbox" id="widgetActive" checked> Active
					</label>
				</div>
				<div class="form-actions">
					<button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
					<button type="submit" class="btn-primary">Save Widget</button>
				</div>
			</form>
		</div>
	</div>
	
	<script>
		let allWidgets = [];
		let currentZone = 'all';
		const token = localStorage.getItem('auth_token');
		
		// Zone tab click handlers
		document.querySelectorAll('.zone-tab').forEach(tab => {
			tab.addEventListener('click', () => {
				document.querySelectorAll('.zone-tab').forEach(t => t.classList.remove('active'));
				tab.classList.add('active');
				currentZone = tab.dataset.zone;
				renderWidgets();
			});
		});
		
		async function loadWidgets() {
			try {
				const res = await fetch('/api/v1/admin/widgets', {
					headers: { 'Authorization': 'Bearer ' + token }
				});
				if (res.ok) {
					const data = await res.json();
					allWidgets = data.widgets || [];
					renderWidgets();
				} else {
					throw new Error('Failed to load');
				}
			} catch (e) {
				console.error(e);
				document.getElementById('widgetsList').innerHTML = '<p style="color: #dc2626; grid-column: 1/-1; text-align: center;">Failed to load widgets.</p>';
			}
		}
		
		function renderWidgets() {
			const container = document.getElementById('widgetsList');
			let widgets = allWidgets;
			
			if (currentZone !== 'all') {
				widgets = allWidgets.filter(w => w.zone && w.zone.includes(currentZone));
			}
			
			if (widgets.length === 0) {
				container.innerHTML = '<div class="empty-state" style="grid-column: 1/-1;"><div class="empty-state-icon">📦</div><p>No widgets found in this zone.</p><button class="action-button" onclick="openCreateModal()">Create your first widget</button></div>';
				return;
			}
			
			let html = '';
			widgets.forEach(widget => {
				html += '<div class="widget-card">' +
					'<div class="widget-header">' +
						'<div class="widget-name">' + escapeHtml(widget.name) + '</div>' +
						'<span class="widget-type">' + formatType(widget.type) + '</span>' +
					'</div>' +
					'<div class="widget-meta">' +
						'<span>📍 ' + formatZone(widget.zone) + '</span>' +
						'<span>📊 Position: ' + (widget.position || 0) + '</span>' +
					'</div>' +
					'<div class="widget-actions">' +
						'<button class="widget-btn widget-btn-toggle ' + (widget.is_active ? '' : 'inactive') + '" onclick="toggleWidget(' + widget.id + ', ' + !widget.is_active + ')">' + (widget.is_active ? '✓ Active' : '○ Inactive') + '</button>' +
						'<button class="widget-btn widget-btn-edit" onclick="editWidget(' + widget.id + ')">✏️ Edit</button>' +
						'<button class="widget-btn widget-btn-delete" onclick="deleteWidget(' + widget.id + ')">🗑️</button>' +
					'</div>' +
				'</div>';
			});
			container.innerHTML = html;
		}
		
		function formatType(type) {
			const types = {
				'latest_articles': 'Latest Articles',
				'popular_articles': 'Popular Articles',
				'category_articles': 'Category Articles',
				'tag_cloud': 'Tag Cloud',
				'newsletter': 'Newsletter',
				'social_links': 'Social Links',
				'custom_html': 'Custom HTML',
				'advertisement': 'Advertisement'
			};
			return types[type] || type;
		}
		
		function formatZone(zone) {
			const zones = {
				'sidebar': 'Sidebar',
				'footer': 'Footer',
				'homepage_hero': 'Homepage Hero',
				'homepage_main': 'Homepage Main',
				'article_sidebar': 'Article Sidebar',
				'article_bottom': 'Article Bottom'
			};
			return zones[zone] || zone;
		}
		
		function escapeHtml(text) {
			const div = document.createElement('div');
			div.textContent = text;
			return div.innerHTML;
		}
		
		function openCreateModal() {
			document.getElementById('modalTitle').textContent = 'Create Widget';
			document.getElementById('widgetForm').reset();
			document.getElementById('widgetId').value = '';
			document.getElementById('widgetActive').checked = true;
			document.getElementById('configFields').innerHTML = '';
			document.getElementById('widgetModal').classList.add('active');
		}
		
		function closeModal() {
			document.getElementById('widgetModal').classList.remove('active');
		}
		
		async function editWidget(id) {
			const widget = allWidgets.find(w => w.id === id);
			if (!widget) return;
			
			document.getElementById('modalTitle').textContent = 'Edit Widget';
			document.getElementById('widgetId').value = widget.id;
			document.getElementById('widgetName').value = widget.name;
			document.getElementById('widgetType').value = widget.type;
			document.getElementById('widgetZone').value = widget.zone;
			document.getElementById('widgetPosition').value = widget.position || 0;
			document.getElementById('widgetActive').checked = widget.is_active;
			updateConfigFields();
			document.getElementById('widgetModal').classList.add('active');
		}
		
		async function saveWidget(e) {
			e.preventDefault();
			const id = document.getElementById('widgetId').value;
			const data = {
				name: document.getElementById('widgetName').value,
				type: document.getElementById('widgetType').value,
				zone: document.getElementById('widgetZone').value,
				position: parseInt(document.getElementById('widgetPosition').value) || 0,
				is_active: document.getElementById('widgetActive').checked,
				config: {}
			};
			
			try {
				const url = id ? '/api/v1/admin/widgets/' + id : '/api/v1/admin/widgets';
				const method = id ? 'PUT' : 'POST';
				const res = await fetch(url, {
					method: method,
					headers: {
						'Authorization': 'Bearer ' + token,
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(data)
				});
				
				if (res.ok) {
					closeModal();
					loadWidgets();
					alert(id ? 'Widget updated successfully!' : 'Widget created successfully!');
				} else {
					const err = await res.json();
					alert('Error: ' + (err.error || 'Failed to save widget'));
				}
			} catch (e) {
				console.error(e);
				alert('Failed to save widget');
			}
		}
		
		async function toggleWidget(id, active) {
			try {
				const widget = allWidgets.find(w => w.id === id);
				if (!widget) return;
				
				const res = await fetch('/api/v1/admin/widgets/' + id, {
					method: 'PUT',
					headers: {
						'Authorization': 'Bearer ' + token,
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({ ...widget, is_active: active })
				});
				
				if (res.ok) {
					loadWidgets();
				}
			} catch (e) {
				console.error(e);
			}
		}
		
		async function deleteWidget(id) {
			if (!confirm('Are you sure you want to delete this widget?')) return;
			
			try {
				const res = await fetch('/api/v1/admin/widgets/' + id, {
					method: 'DELETE',
					headers: { 'Authorization': 'Bearer ' + token }
				});
				
				if (res.ok) {
					loadWidgets();
					alert('Widget deleted successfully!');
				}
			} catch (e) {
				console.error(e);
				alert('Failed to delete widget');
			}
		}
		
		function updateConfigFields() {
			const type = document.getElementById('widgetType').value;
			const container = document.getElementById('configFields');
			let html = '';
			
			switch(type) {
				case 'latest_articles':
				case 'popular_articles':
					html = '<label class="form-label">Number of Articles</label><input type="number" class="form-input" id="configLimit" value="5" min="1" max="20">';
					break;
				case 'category_articles':
					html = '<label class="form-label">Category ID</label><input type="number" class="form-input" id="configCategoryId" placeholder="Enter category ID">';
					break;
				case 'custom_html':
					html = '<label class="form-label">HTML Content</label><textarea class="form-textarea" id="configHtml" placeholder="Enter custom HTML..."></textarea>';
					break;
				case 'advertisement':
					html = '<label class="form-label">Ad Code</label><textarea class="form-textarea" id="configAdCode" placeholder="Paste your ad code here..."></textarea>';
					break;
			}
			
			container.innerHTML = html;
		}
		
		loadWidgets();
	</script>`
	s.renderAdminPage(c, "Widget Management", "widgets", content)
}

// ============================================================================
// SETTINGS HANDLERS - Already defined in server.go
// handleAdminSettingsGeneral, handleAdminSettingsPerformance, 
// handleAdminSettingsSecurity, handleAdminSettingsBackup
// ============================================================================


// ============================================================================
// INTEGRATIONS - THIRD PARTY
// ============================================================================

func (s *Server) handleAdminGoogleAnalytics(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📊 Google Analytics Integration</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Google Analytics integration will allow you to connect your GA4 property to track website traffic, user behavior, conversions, and generate detailed reports directly in the admin panel.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Configuration</h4>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Google Analytics Measurement ID (GA4)</label>
				<input type="text" id="ga-measurement-id" style="width: 100%; max-width: 400px; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="G-XXXXXXXXXX">
				<p style="color: #64748b; font-size: 0.875rem; margin-top: 0.25rem;">Enter your GA4 Measurement ID (starts with G-)</p>
			</div>
			<div style="margin-bottom: 1rem;">
				<label style="display: flex; align-items: center; gap: 0.5rem;">
					<input type="checkbox" id="ga-enabled"> Enable Google Analytics tracking
				</label>
			</div>
			<button class="action-button" disabled>💾 Save Configuration</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Google Analytics", "integrations", content)
}

func (s *Server) handleAdminTagManager(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🏷️ Google Tag Manager Integration</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Google Tag Manager integration will allow you to manage all your marketing and analytics tags from one place without modifying code. Perfect for tracking pixels, conversion tracking, and remarketing tags.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Configuration</h4>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">GTM Container ID</label>
				<input type="text" id="gtm-container-id" style="width: 100%; max-width: 400px; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="GTM-XXXXXXX">
				<p style="color: #64748b; font-size: 0.875rem; margin-top: 0.25rem;">Enter your GTM Container ID (starts with GTM-)</p>
			</div>
			<div style="margin-bottom: 1rem;">
				<label style="display: flex; align-items: center; gap: 0.5rem;">
					<input type="checkbox" id="gtm-enabled"> Enable Google Tag Manager
				</label>
			</div>
			<button class="action-button" disabled>💾 Save Configuration</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Google Tag Manager", "integrations", content)
}

func (s *Server) handleAdminSearchConsole(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔍 Google Search Console Integration</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Google Search Console integration will help you monitor your site's presence in Google Search results, submit sitemaps, and view search performance data directly in the admin panel.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Verification</h4>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Site Verification Code</label>
				<input type="text" id="gsc-verification" style="width: 100%; max-width: 500px; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="google-site-verification=XXXXXXXX">
				<p style="color: #64748b; font-size: 0.875rem; margin-top: 0.25rem;">Enter the meta tag verification code from Search Console</p>
			</div>
			<button class="action-button" disabled>💾 Save Verification</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Google Search Console", "integrations", content)
}

func (s *Server) handleAdminAdSense(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">💰 Google AdSense Integration</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Google AdSense integration will allow you to monetize your website with display ads. Configure ad units, auto ads, and track your earnings directly from the admin panel.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Configuration</h4>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">AdSense Publisher ID</label>
				<input type="text" id="adsense-pub-id" style="width: 100%; max-width: 400px; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="ca-pub-XXXXXXXXXXXXXXXX">
				<p style="color: #64748b; font-size: 0.875rem; margin-top: 0.25rem;">Enter your AdSense Publisher ID (starts with ca-pub-)</p>
			</div>
			<div style="margin-bottom: 1rem;">
				<label style="display: flex; align-items: center; gap: 0.5rem;">
					<input type="checkbox" id="adsense-auto-ads"> Enable Auto Ads
				</label>
			</div>
			<button class="action-button" disabled>💾 Save Configuration</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Google AdSense", "integrations", content)
}

func (s *Server) handleAdminFacebookPixel(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📘 Facebook Pixel Integration</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Facebook Pixel integration will help you track conversions from Facebook ads, optimize ads based on collected data, build targeted audiences, and remarket to people who have taken action on your website.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Configuration</h4>
			<div style="margin-bottom: 1rem;">
				<label style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Facebook Pixel ID</label>
				<input type="text" id="fb-pixel-id" style="width: 100%; max-width: 400px; padding: 0.5rem; border: 1px solid #e2e8f0; border-radius: 6px;" placeholder="XXXXXXXXXXXXXXXX">
				<p style="color: #64748b; font-size: 0.875rem; margin-top: 0.25rem;">Enter your Facebook Pixel ID (16-digit number)</p>
			</div>
			<div style="margin-bottom: 1rem;">
				<label style="display: flex; align-items: center; gap: 0.5rem;">
					<input type="checkbox" id="fb-pixel-enabled"> Enable Facebook Pixel
				</label>
			</div>
			<button class="action-button" disabled>💾 Save Configuration</button>
		</div>
	</div>`
	s.renderAdminPage(c, "Facebook Pixel", "integrations", content)
}

// ============================================================================
// CODE INJECTION
// ============================================================================

func (s *Server) handleAdminHeaderScripts(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📝 Header Scripts</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Header scripts injection allows you to add custom code to the &lt;head&gt; section of every page. Perfect for meta tags, verification codes, analytics scripts, and custom fonts.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Custom Header Code</h4>
			<p style="color: #64748b; font-size: 0.875rem; margin-bottom: 0.5rem;">This code will be injected into the &lt;head&gt; section of all pages:</p>
			<textarea id="header-scripts" style="width: 100%; min-height: 200px; padding: 1rem; border: 1px solid #e2e8f0; border-radius: 6px; font-family: monospace; font-size: 0.875rem;" placeholder="<!-- Add your header scripts here -->
<meta name='verification' content='xxx'>
<script>...</script>"></textarea>
			<div style="margin-top: 1rem;">
				<button class="action-button" disabled>💾 Save Header Scripts</button>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "Header Scripts", "code-injection", content)
}

func (s *Server) handleAdminFooterScripts(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">📝 Footer Scripts</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Footer scripts injection allows you to add custom code before the closing &lt;/body&gt; tag. Ideal for chat widgets, analytics scripts, and third-party integrations that should load after page content.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Custom Footer Code</h4>
			<p style="color: #64748b; font-size: 0.875rem; margin-bottom: 0.5rem;">This code will be injected before the &lt;/body&gt; tag on all pages:</p>
			<textarea id="footer-scripts" style="width: 100%; min-height: 200px; padding: 1rem; border: 1px solid #e2e8f0; border-radius: 6px; font-family: monospace; font-size: 0.875rem;" placeholder="<!-- Add your footer scripts here -->
<script src='https://example.com/widget.js'></script>
<script>...</script>"></textarea>
			<div style="margin-top: 1rem;">
				<button class="action-button" disabled>💾 Save Footer Scripts</button>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "Footer Scripts", "code-injection", content)
}

func (s *Server) handleAdminCustomCSS(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">🎨 Custom CSS</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Custom CSS allows you to add your own styles to customize the appearance of your website without modifying theme files. Changes are applied site-wide.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Custom Stylesheet</h4>
			<p style="color: #64748b; font-size: 0.875rem; margin-bottom: 0.5rem;">Add custom CSS rules to override or extend the default styles:</p>
			<textarea id="custom-css" style="width: 100%; min-height: 300px; padding: 1rem; border: 1px solid #e2e8f0; border-radius: 6px; font-family: monospace; font-size: 0.875rem;" placeholder="/* Custom CSS */
.article-title {
    font-size: 2rem;
    color: #1a1a1a;
}

.sidebar {
    background: #f5f5f5;
}"></textarea>
			<div style="margin-top: 1rem;">
				<button class="action-button" disabled>💾 Save Custom CSS</button>
				<button class="action-button" style="background: #64748b; margin-left: 0.5rem;" disabled>👁️ Preview</button>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "Custom CSS", "code-injection", content)
}

func (s *Server) handleAdminCustomJS(c *gin.Context) {
	content := `
	<div class="dashboard-card">
		<div class="card-title">⚡ Custom JavaScript</div>
		<p style="background: #fef3c7; padding: 1rem; border-radius: 8px; color: #92400e;">
			<strong>🚧 This feature coming soon</strong><br>
			<em>Custom JavaScript allows you to add your own scripts to enhance website functionality. Use this for custom interactions, third-party integrations, or tracking code.</em>
		</p>
		<div style="margin-top: 1.5rem;">
			<h4 style="margin-bottom: 1rem;">Custom JavaScript</h4>
			<p style="color: #dc2626; font-size: 0.875rem; margin-bottom: 0.5rem;">⚠️ Warning: Invalid JavaScript can break your website. Test thoroughly before saving.</p>
			<textarea id="custom-js" style="width: 100%; min-height: 300px; padding: 1rem; border: 1px solid #e2e8f0; border-radius: 6px; font-family: monospace; font-size: 0.875rem;" placeholder="// Custom JavaScript
document.addEventListener('DOMContentLoaded', function() {
    // Your code here
    console.log('Custom JS loaded');
});"></textarea>
			<div style="margin-top: 1rem;">
				<button class="action-button" disabled>💾 Save Custom JavaScript</button>
			</div>
		</div>
	</div>`
	s.renderAdminPage(c, "Custom JavaScript", "code-injection", content)
}
