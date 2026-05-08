package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// renderAdminPage renders a generic admin page with the new sidebar layout
func (s *Server) renderAdminPage(c *gin.Context, title, page, content string) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + ` - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin-sidebar.css">
    <style>
        /* Additional page-specific styles */
        .dashboard-card { background: var(--card-bg); padding: 1.5rem; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin-bottom: 1.5rem; }
        .card-title { font-size: 1.1rem; font-weight: 600; margin-bottom: 1rem; color: #1e293b; }
        .action-button { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.625rem 1rem; background-color: #3b82f6; color: white; text-decoration: none; border-radius: 8px; font-size: 0.875rem; font-weight: 500; border: none; cursor: pointer; transition: background-color 0.2s; }
        .action-button:hover { background-color: #2563eb; color: white; }
        .status-badge { display: inline-flex; align-items: center; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 500; }
        .status-published, .status-healthy { background-color: #dcfce7; color: #16a34a; }
        .status-draft { background-color: #fef3c7; color: #d97706; }
        .status-archived { background-color: #f1f5f9; color: #475569; }
        .status-deleted { background-color: #fee2e2; color: #dc2626; }
        .metric { font-size: 2rem; font-weight: bold; color: #3b82f6; }
        .articles-table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
        .articles-table th, .articles-table td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid #e2e8f0; }
        .articles-table th { background-color: #f8fafc; font-weight: 600; color: #64748b; font-size: 0.75rem; text-transform: uppercase; }
        .articles-table tr:hover { background-color: #f8fafc; }
        .editor-container { display: grid; grid-template-columns: 2fr 1fr; gap: 2rem; }
        .main-editor { display: flex; flex-direction: column; gap: 1rem; }
        .sidebar-panel { display: flex; flex-direction: column; gap: 1rem; }
        .content-editor { min-height: 400px; width: 100%; padding: 1rem; border: 1px solid #e2e8f0; border-radius: 8px; font-family: inherit; font-size: 1rem; resize: vertical; }
        .analytics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 1.5rem; margin-bottom: 2rem; }
        .chart-container { background: white; padding: 1.5rem; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .chart-container h3 { margin: 0 0 1rem 0; font-size: 1.1rem; color: #1e293b; }
        .chart-wrapper { position: relative; height: 300px; width: 100%; }
        @media (max-width: 1024px) { 
            .editor-container { grid-template-columns: 1fr; }
            .analytics-grid { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <div class="admin-layout">
        <!-- Sidebar -->
        <aside class="admin-sidebar" id="sidebar">
            <div class="sidebar-header">
                <a href="/admin/dashboard" class="sidebar-logo">
                    <div class="sidebar-logo-icon">📰</div>
                    <span class="logo-text">News Admin</span>
                </a>
            </div>
            <nav class="sidebar-nav">
                <!-- Dashboard -->
                <div class="nav-section">
                    <div class="nav-item">
                        <a href="/admin/dashboard" class="nav-link" data-page="dashboard">
                            <span class="nav-icon">📊</span>
                            <span class="nav-text">Dashboard</span>
                        </a>
                    </div>
                </div>

                <!-- Content Management -->
                <div class="nav-section">
                    <div class="nav-section-title">Content</div>
                    <div class="nav-item" data-section="content">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">📝</span>
                            <span class="nav-text">Content</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/content/create" class="nav-link">Create Article</a>
                            <a href="/admin/content/articles" class="nav-link">Manage Articles</a>
                            <a href="/admin/content/categories" class="nav-link">Categories</a>
                            <a href="/admin/content/tags" class="nav-link">Tags</a>
                            <a href="/admin/content/media" class="nav-link">Media Library</a>
                            <a href="/admin/content/trash" class="nav-link">Trash</a>
                        </div>
                    </div>
                </div>

                <!-- Autolinking -->
                <div class="nav-section">
                    <div class="nav-item" data-section="autolinking">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">🔗</span>
                            <span class="nav-text">Autolinking</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/autolinking" class="nav-link">Overview</a>
                            <a href="/admin/keyword-banks" class="nav-link">Keyword Banks</a>
                        </div>
                    </div>
                </div>

                <!-- Comments -->
                <div class="nav-section">
                    <div class="nav-item" data-section="comments">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">💬</span>
                            <span class="nav-text">Comments</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/comments" class="nav-link">Moderate</a>
                            <a href="/admin/comments/analytics" class="nav-link">Analytics</a>
                            <a href="/admin/comments/settings" class="nav-link">Settings</a>
                        </div>
                    </div>
                </div>

                <!-- Management Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Management</div>
                    <div class="nav-item" data-section="users">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">👥</span>
                            <span class="nav-text">Users</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/users/list" class="nav-link">All Users</a>
                            <a href="/admin/users/create" class="nav-link">Create User</a>
                            <a href="/admin/users/roles" class="nav-link">Roles</a>
                        </div>
                    </div>
                </div>

                <!-- Content Ingestion -->
                <div class="nav-section">
                    <div class="nav-item">
                        <a href="/admin/content-ingestion" class="nav-link" data-page="content-ingestion">
                            <span class="nav-icon">📥</span>
                            <span class="nav-text">Content Ingestion</span>
                        </a>
                    </div>
                </div>

                <!-- Marketing & Engagement Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Marketing</div>
                    <div class="nav-item" data-section="advertisements">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">📢</span>
                            <span class="nav-text">Advertisements</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/ads/campaigns" class="nav-link">Campaigns</a>
                            <a href="/admin/ads/slots" class="nav-link">Ad Slots</a>
                            <a href="/admin/ads/creatives" class="nav-link">Creatives</a>
                            <a href="/admin/ads/targeting" class="nav-link">Targeting</a>
                            <a href="/admin/ads/analytics" class="nav-link">Ad Analytics</a>
                        </div>
                    </div>
                    <div class="nav-item" data-section="push-notifications">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">🔔</span>
                            <span class="nav-text">Push Notifications</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/push/send" class="nav-link">Send Notification</a>
                            <a href="/admin/push/templates" class="nav-link">Templates</a>
                            <a href="/admin/push/subscribers" class="nav-link">Subscribers</a>
                            <a href="/admin/push/analytics" class="nav-link">Analytics</a>
                        </div>
                    </div>
                    <div class="nav-item" data-section="social-media">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">📱</span>
                            <span class="nav-text">Social Media</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/social/accounts" class="nav-link">Connected Accounts</a>
                            <a href="/admin/social/auto-publish" class="nav-link">Auto Publish</a>
                            <a href="/admin/social/scheduled" class="nav-link">Scheduled Posts</a>
                            <a href="/admin/social/analytics" class="nav-link">Social Analytics</a>
                        </div>
                    </div>
                    <div class="nav-item">
                        <a href="/admin/newsletter" class="nav-link" data-page="newsletter">
                            <span class="nav-icon">✉️</span>
                            <span class="nav-text">Newsletter</span>
                        </a>
                    </div>
                </div>

                <!-- Analytics & SEO Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Analytics & SEO</div>
                    <div class="nav-item" data-section="analytics">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">📈</span>
                            <span class="nav-text">Analytics</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/analytics" class="nav-link">Overview</a>
                            <a href="/admin/analytics/traffic" class="nav-link">Traffic</a>
                            <a href="/admin/analytics/content" class="nav-link">Content Performance</a>
                            <a href="/admin/analytics/audience" class="nav-link">Audience</a>
                            <a href="/admin/analytics/realtime" class="nav-link">Real-time</a>
                        </div>
                    </div>
                    <div class="nav-item" data-section="seo">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">🔍</span>
                            <span class="nav-text">SEO</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/seo" class="nav-link">SEO Settings</a>
                            <a href="/admin/seo/overview" class="nav-link">Overview</a>
                            <a href="/admin/seo/sitemap" class="nav-link">Sitemap</a>
                            <a href="/admin/seo/google-news" class="nav-link">Google News</a>
                            <a href="/admin/seo/schema" class="nav-link">Schema Markup</a>
                            <a href="/admin/seo/redirects" class="nav-link">Redirects</a>
                        </div>
                    </div>
                </div>

                <!-- Appearance Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Appearance</div>
                    <div class="nav-item">
                        <a href="/admin/themes" class="nav-link" data-page="themes">
                            <span class="nav-icon">🎨</span>
                            <span class="nav-text">Themes</span>
                        </a>
                    </div>
                    <div class="nav-item">
                        <a href="/admin/widgets" class="nav-link" data-page="widgets">
                            <span class="nav-icon">📦</span>
                            <span class="nav-text">Widgets</span>
                        </a>
                    </div>
                    <div class="nav-item">
                        <a href="/admin/menus" class="nav-link" data-page="menus">
                            <span class="nav-icon">📋</span>
                            <span class="nav-text">Menus</span>
                        </a>
                    </div>
                </div>

                <!-- Distribution Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Distribution</div>
                    <div class="nav-item">
                        <a href="/admin/rss" class="nav-link" data-page="rss">
                            <span class="nav-icon">📡</span>
                            <span class="nav-text">RSS Feeds</span>
                        </a>
                    </div>
                    <div class="nav-item" data-section="cdn">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">🌐</span>
                            <span class="nav-text">CDN</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/cdn/config" class="nav-link">Configuration</a>
                            <a href="/admin/cdn/purge" class="nav-link">Purge Cache</a>
                            <a href="/admin/cdn/stats" class="nav-link">Statistics</a>
                        </div>
                    </div>
                </div>

                <!-- Integrations Section -->
                <div class="nav-section">
                    <div class="nav-section-title">Integrations</div>
                    <div class="nav-item" data-section="integrations">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">🔌</span>
                            <span class="nav-text">Third-Party</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/integrations/google-analytics" class="nav-link">Google Analytics</a>
                            <a href="/admin/integrations/tag-manager" class="nav-link">Google Tag Manager</a>
                            <a href="/admin/integrations/search-console" class="nav-link">Search Console</a>
                            <a href="/admin/integrations/adsense" class="nav-link">Google AdSense</a>
                            <a href="/admin/integrations/facebook-pixel" class="nav-link">Facebook Pixel</a>
                        </div>
                    </div>
                    <div class="nav-item" data-section="code-injection">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">💉</span>
                            <span class="nav-text">Code Injection</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/code/header" class="nav-link">Header Scripts</a>
                            <a href="/admin/code/footer" class="nav-link">Footer Scripts</a>
                            <a href="/admin/code/custom-css" class="nav-link">Custom CSS</a>
                            <a href="/admin/code/custom-js" class="nav-link">Custom JavaScript</a>
                        </div>
                    </div>
                </div>

                <!-- System Section -->
                <div class="nav-section">
                    <div class="nav-section-title">System</div>
                    <div class="nav-item" data-section="settings">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">⚙️</span>
                            <span class="nav-text">Settings</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/settings/general" class="nav-link">General</a>
                            <a href="/admin/settings/performance" class="nav-link">Performance</a>
                            <a href="/admin/settings/security" class="nav-link">Security</a>
                            <a href="/admin/settings/email" class="nav-link">Email</a>
                            <a href="/admin/settings/api" class="nav-link">API Keys</a>
                        </div>
                    </div>
                    <div class="nav-item" data-section="backup">
                        <div class="nav-link" onclick="toggleSubmenu(this)">
                            <span class="nav-icon">💾</span>
                            <span class="nav-text">Backup & Restore</span>
                            <span class="nav-arrow">›</span>
                        </div>
                        <div class="nav-submenu">
                            <a href="/admin/backup/create" class="nav-link">Create Backup</a>
                            <a href="/admin/backup/list" class="nav-link">Backup History</a>
                            <a href="/admin/backup/restore" class="nav-link">Restore</a>
                            <a href="/admin/backup/schedule" class="nav-link">Schedule</a>
                        </div>
                    </div>
                    <div class="nav-item">
                        <a href="/admin/system" class="nav-link" data-page="system">
                            <span class="nav-icon">🖥️</span>
                            <span class="nav-text">System Monitor</span>
                        </a>
                    </div>
                    <div class="nav-item">
                        <a href="/admin/logs" class="nav-link" data-page="logs">
                            <span class="nav-icon">📜</span>
                            <span class="nav-text">Logs</span>
                        </a>
                    </div>
                </div>
            </nav>
        </aside>

        <!-- Mobile Overlay -->
        <div class="sidebar-overlay" id="sidebarOverlay" onclick="closeMobileSidebar()"></div>

        <!-- Main Content -->
        <main class="admin-main">
            <header class="admin-header">
                <div class="header-left">
                    <button class="mobile-menu-btn" onclick="toggleMobileSidebar()" title="Toggle Menu">☰</button>
                    <h1 class="page-title">` + title + `</h1>
                </div>
                <div class="header-right">
                    <button class="header-btn" onclick="window.open('/', '_blank')" title="View Site">🌐</button>
                    <div class="user-dropdown">
                        <div class="user-menu" onclick="toggleUserDropdown()">
                            <div class="user-avatar" id="userAvatar">A</div>
                            <div class="user-info">
                                <div class="user-name" id="userName">Admin</div>
                                <div class="user-role" id="userRole">Administrator</div>
                            </div>
                        </div>
                        <div class="dropdown-menu" id="userDropdown">
                            <a href="/admin/profile" class="dropdown-item">👤 Profile</a>
                            <div class="dropdown-divider"></div>
                            <button class="dropdown-item" onclick="logout()" style="color: #dc2626;">🚪 Logout</button>
                        </div>
                    </div>
                </div>
            </header>
            <div class="admin-content">
                ` + content + `
            </div>
        </main>
    </div>

    <script>
        // Immediate auth check
        (function() {
            const token = localStorage.getItem('auth_token');
            const role = localStorage.getItem('user_role');
            if (!token || (role !== 'admin' && role !== 'editor')) {
                window.location.href = '/admin/login';
                return;
            }
            document.body.classList.add('authenticated');
        })();

        function toggleMobileSidebar() {
            const sidebar = document.getElementById('sidebar');
            const overlay = document.getElementById('sidebarOverlay');
            const isOpen = sidebar.classList.contains('mobile-open');
            
            if (isOpen) {
                closeMobileSidebar();
            } else {
                sidebar.classList.add('mobile-open');
                overlay.classList.add('active');
                document.body.classList.add('sidebar-open');
            }
        }

        function closeMobileSidebar() {
            const sidebar = document.getElementById('sidebar');
            const overlay = document.getElementById('sidebarOverlay');
            sidebar.classList.remove('mobile-open');
            overlay.classList.remove('active');
            document.body.classList.remove('sidebar-open');
        }

        // Close mobile sidebar when clicking outside
        document.addEventListener('click', function(e) {
            const sidebar = document.getElementById('sidebar');
            const mobileBtn = document.querySelector('.mobile-menu-btn');
            if (window.innerWidth <= 1024 && 
                sidebar.classList.contains('mobile-open') && 
                !sidebar.contains(e.target) && 
                !mobileBtn.contains(e.target)) {
                closeMobileSidebar();
            }
        });

        function toggleSubmenu(element) {
            const navItem = element.closest('.nav-item');
            navItem.classList.toggle('open');
        }

        function toggleUserDropdown() {
            document.getElementById('userDropdown').classList.toggle('show');
        }

        document.addEventListener('click', function(e) {
            if (!e.target.closest('.user-dropdown')) {
                document.getElementById('userDropdown').classList.remove('show');
            }
        });

        // Initialize sidebar state
        if (window.innerWidth <= 1024) {
            // On mobile, ensure sidebar starts hidden
            const sidebar = document.getElementById('sidebar');
            const overlay = document.getElementById('sidebarOverlay');
            sidebar.classList.remove('mobile-open');
            overlay.classList.remove('active');
            document.body.classList.remove('sidebar-open');
        }

        // Handle window resize
        window.addEventListener('resize', function() {
            if (window.innerWidth > 1024) {
                // Desktop: close mobile menu if open
                closeMobileSidebar();
            }
        });

        // Highlight active menu and expand parent
        function highlightActiveMenu() {
            const currentPath = window.location.pathname;
            const navLinks = document.querySelectorAll('.nav-link[href]');
            
            navLinks.forEach(link => {
                const href = link.getAttribute('href');
                if (currentPath === href || (href !== '/admin/dashboard' && currentPath.startsWith(href))) {
                    link.classList.add('active');
                    // Expand parent submenu if exists
                    const parentItem = link.closest('.nav-item');
                    if (parentItem && parentItem.querySelector('.nav-submenu')) {
                        parentItem.classList.add('open');
                    }
                }
            });
        }

        function logout() {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user_role');
            localStorage.removeItem('username');
            window.location.href = '/admin/login';
        }

        document.addEventListener('DOMContentLoaded', function() {
            highlightActiveMenu();
            // Update user info
            const username = localStorage.getItem('username') || 'Admin';
            const role = localStorage.getItem('user_role') || 'admin';
            document.getElementById('userName').textContent = username;
            document.getElementById('userAvatar').textContent = username.charAt(0).toUpperCase();
            document.getElementById('userRole').textContent = role === 'admin' ? 'Administrator' : 'Editor';
        });
    </script>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// handleAdminComments renders the comment management page
func (s *Server) handleAdminComments(c *gin.Context) {
	content := s.getCommentManagementContent()
	s.renderAdminPage(c, "Comment Management", "comments", content)
}

// handleAdminCommentsAnalytics renders comment analytics
func (s *Server) handleAdminCommentsAnalytics(c *gin.Context) {
	s.renderAdminPage(c, "Comment Analytics", "comments", `
		<div class="dashboard-card">
			<div class="card-title">Comment Analytics</div>
			<p>Comment analytics and statistics will be displayed here.</p>
		</div>
	`)
}

// handleAdminCommentsSettings renders comment settings
func (s *Server) handleAdminCommentsSettings(c *gin.Context) {
	s.renderAdminPage(c, "Comment Settings", "comments", `
		<div class="dashboard-card">
			<div class="card-title">Comment Settings</div>
			<p>Comment moderation and configuration settings will be displayed here.</p>
		</div>
	`)
}

// handleAdminContentIngestion renders the content ingestion management page
func (s *Server) handleAdminContentIngestion(c *gin.Context) {
	content := s.getContentIngestionContent()
	s.renderAdminPage(c, "Content Ingestion Management", "content-ingestion", content)
}

// getContentIngestionContent returns the content for content ingestion page
func (s *Server) getContentIngestionContent() string {
	return `
    <link rel="stylesheet" href="/static/css/admin/content-ingestion.css?v=20251112-10">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css" integrity="sha512-9usAa10IRO0HhonpyAIVpjrylPvoDwiPUiKdWk5t3PyolY1cOd4DSE0Ga+ri4AuTroPR5aQvXU9xC6qOPnzFeg==" crossorigin="anonymous" referrerpolicy="no-referrer">
    
    <!-- Content Ingestion Management Interface -->
        <div class="content-ingestion-management">
            <!-- Header Section -->
            <div class="page-header">
                <div class="header-content">
                    <h1><i class="fas fa-download"></i> Content Ingestion Management</h1>
                    <p class="subtitle">Manage external content sources, monitor ingestion, and process pending content</p>
                </div>
                <div class="header-actions">
                    <button class="btn btn-primary" onclick="showCreateSourceModal()">
                        <i class="fas fa-plus"></i> Add Content Source
                    </button>
                    <button class="btn btn-secondary" onclick="refreshStats()">
                        <i class="fas fa-sync"></i> Refresh Stats
                    </button>
                </div>
            </div>

            <!-- Statistics Dashboard -->
            <div class="stats-dashboard">
                <div class="stat-card">
                    <div class="stat-icon">
                        <i class="fas fa-rss text-blue"></i>
                    </div>
                    <div class="stat-content">
                        <h3 id="total-sources">0</h3>
                        <p>Active Sources</p>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">
                        <i class="fas fa-clock text-orange"></i>
                    </div>
                    <div class="stat-content">
                        <h3 id="pending-content">0</h3>
                        <p>Pending Content</p>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">
                        <i class="fas fa-check text-green"></i>
                    </div>
                    <div class="stat-content">
                        <h3 id="processed-today">0</h3>
                        <p>Processed Today</p>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">
                        <i class="fas fa-times text-red"></i>
                    </div>
                    <div class="stat-content">
                        <h3 id="rejected-today">0</h3>
                        <p>Rejected Today</p>
                    </div>
                </div>
            </div>

            <!-- Navigation Tabs -->
            <div class="tab-navigation">
                <button class="tab-btn active" onclick="showTab('sources')">
                    <i class="fas fa-database"></i> Content Sources
                </button>
                <button class="tab-btn" onclick="showTab('pending')">
                    <i class="fas fa-hourglass-half"></i> Pending Content
                </button>
                <button class="tab-btn" onclick="showTab('processed')">
                    <i class="fas fa-check-circle"></i> Processed Content
                </button>
                <button class="tab-btn" onclick="showTab('analytics')">
                    <i class="fas fa-chart-bar"></i> Analytics
                </button>
            </div>

            <!-- Content Sources Tab -->
            <div id="sources-tab" class="tab-content active">
                <div class="section-header">
                    <h2>Content Sources</h2>
                    <div class="section-actions">
                        <input type="text" id="sources-search" placeholder="Search sources..." class="search-input">
                        <select id="sources-filter" class="filter-select">
                            <option value="">All Types</option>
                            <option value="api">API</option>
                            <option value="webhook">Webhook</option>
                            <option value="manual">Manual</option>
                        </select>
                    </div>
                </div>

                <div class="table-container">
                    <table class="data-table" id="sources-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Type</th>
                                <th>Status</th>
                                <th>Auto Publish</th>
                                <th>Rate Limit</th>
                                <th>Priority</th>
                                <th>Last Activity</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="sources-tbody">
                            <tr>
                                <td colspan="8" style="text-align: center; padding: 40px; color: #6c757d;">
                                    <i class="fas fa-spinner fa-spin"></i> Loading content sources...
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Pending Content Tab -->
            <div id="pending-tab" class="tab-content">
                <div class="section-header">
                    <h2>Pending Content</h2>
                    <div class="section-actions">
                        <button class="btn btn-success" onclick="processBatch()">
                            <i class="fas fa-play"></i> Process Batch
                        </button>
                        <select id="pending-source-filter" class="filter-select">
                            <option value="">All Sources</option>
                        </select>
                    </div>
                </div>

                <div class="table-container">
                    <table class="data-table" id="pending-table">
                        <thead>
                            <tr>
                                <th><input type="checkbox" id="select-all-pending"></th>
                                <th>Title</th>
                                <th>Source</th>
                                <th>Author</th>
                                <th>Category</th>
                                <th>Received</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="pending-tbody">
                            <tr>
                                <td colspan="7" style="text-align: center; padding: 40px; color: #6c757d;">
                                    <i class="fas fa-spinner fa-spin"></i> Loading pending content...
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Processed Content Tab -->
            <div id="processed-tab" class="tab-content">
                <div class="section-header">
                    <h2>Processed Content</h2>
                    <div class="section-actions">
                        <input type="text" id="processed-search" placeholder="Search processed content..." class="search-input">
                        <select id="processed-status-filter" class="filter-select">
                            <option value="">All Status</option>
                            <option value="processed">Processed</option>
                            <option value="rejected">Rejected</option>
                            <option value="duplicate">Duplicate</option>
                        </select>
                    </div>
                </div>

                <div class="table-container">
                    <table class="data-table" id="processed-table">
                        <thead>
                            <tr>
                                <th>Title</th>
                                <th>Source</th>
                                <th>Status</th>
                                <th>Processed</th>
                                <th>Article</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="processed-tbody">
                            <tr>
                                <td colspan="6" style="text-align: center; padding: 40px; color: #6c757d;">
                                    <i class="fas fa-spinner fa-spin"></i> Loading processed content...
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Analytics Tab -->
            <div id="analytics-tab" class="tab-content">
                <div class="analytics-header">
                    <h2>Content Ingestion Analytics</h2>
                    <div class="time-range-selector">
                        <select id="analytics-timerange" onchange="contentManager.updateAnalyticsTimeRange(this.value)">
                            <option value="24">Last 24 Hours</option>
                            <option value="168">Last 7 Days</option>
                            <option value="720">Last 30 Days</option>
                            <option value="2160">Last 90 Days</option>
                        </select>
                    </div>
                </div>

                <div class="analytics-grid">
                    <!-- Ingestion Volume Chart -->
                    <div class="chart-container">
                        <h3>Ingestion Volume</h3>
                        <div class="chart-controls">
                            <select id="volume-granularity" onchange="contentManager.updateVolumeChart(this.value)">
                                <option value="hour">Hourly</option>
                                <option value="day">Daily</option>
                                <option value="week">Weekly</option>
                            </select>
                        </div>
                        <div class="chart-wrapper">
                            <canvas id="ingestion-volume-chart"></canvas>
                        </div>
                    </div>

                    <!-- Source Performance Chart -->
                    <div class="chart-container">
                        <h3>Source Performance</h3>
                        <div class="performance-metrics">
                            <div class="metric-card">
                                <h4>Success Rate</h4>
                                <div class="metric-value" id="success-rate">-</div>
                            </div>
                            <div class="metric-card">
                                <h4>Avg Processing Time</h4>
                                <div class="metric-value" id="avg-processing-time">-</div>
                            </div>
                            <div class="metric-card">
                                <h4>Error Rate</h4>
                                <div class="metric-value" id="error-rate">-</div>
                            </div>
                        </div>
                        <div class="chart-wrapper">
                            <canvas id="source-performance-chart"></canvas>
                        </div>
                    </div>

                    <!-- Content Status Flow -->
                    <div class="chart-container">
                        <h3>Content Status Distribution</h3>
                        <div class="chart-wrapper">
                            <canvas id="status-distribution-chart"></canvas>
                        </div>
                    </div>

                    <!-- Processing Timeline -->
                    <div class="chart-container">
                        <h3>Processing Timeline</h3>
                        <div class="timeline-stats">
                            <div class="timeline-item">
                                <span class="timeline-label">Pending Queue</span>
                                <span class="timeline-value" id="pending-queue">-</span>
                            </div>
                            <div class="timeline-item">
                                <span class="timeline-label">Avg Processing Time</span>
                                <span class="timeline-value" id="processing-time">-</span>
                            </div>
                            <div class="timeline-item">
                                <span class="timeline-label">Peak Hour</span>
                                <span class="timeline-value" id="peak-hour">-</span>
                            </div>
                        </div>
                        <div class="chart-wrapper">
                            <canvas id="processing-timeline-chart"></canvas>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Create/Edit Source Modal -->
        <div id="source-modal" class="modal" style="display: none;">
            <div class="modal-content large">
                <div class="modal-header">
                    <h2 id="source-modal-title">Add Content Source</h2>
                    <button class="modal-close" onclick="closeSourceModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="source-form">
                        <input type="hidden" id="source-id">
                        
                        <!-- Basic Information -->
                        <div class="form-section">
                            <h3>Basic Information</h3>
                            <div class="form-row">
                                <div class="form-group">
                                    <label for="source-name">Source Name *</label>
                                    <input type="text" id="source-name" required maxlength="100">
                                </div>
                                <div class="form-group">
                                    <label for="source-type">Type *</label>
                                    <select id="source-type" required>
                                        <option value="">Select Type</option>
                                        <option value="api">API</option>
                                        <option value="webhook">Webhook</option>
                                        <option value="manual">Manual</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <!-- Configuration -->
                        <div class="form-section">
                            <h3>Configuration</h3>
                            <div class="form-row">
                                <div class="form-group">
                                    <label for="rate-limit">Rate Limit (per hour)</label>
                                    <input type="number" id="rate-limit" min="1" max="10000" value="100">
                                </div>
                                <div class="form-group">
                                    <label for="priority">Priority (1-10)</label>
                                    <input type="number" id="priority" min="1" max="10" value="5">
                                </div>
                            </div>
                            <div class="form-row">
                                <div class="form-group checkbox-group">
                                    <label>
                                        <input type="checkbox" id="is-active" checked>
                                        Active
                                    </label>
                                </div>
                                <div class="form-group checkbox-group">
                                    <label>
                                        <input type="checkbox" id="auto-publish">
                                        Auto Publish (trusted source)
                                    </label>
                                </div>
                            </div>
                            
                            <!-- Default Settings -->
                            <div class="form-row">
                                <div class="form-group">
                                    <label for="default-category">Default Category</label>
                                    <select id="default-category">
                                        <option value="">Select Category</option>
                                    </select>
                                </div>
                                <div class="form-group">
                                    <label for="default-author">Default Author</label>
                                    <select id="default-author">
                                        <option value="">Select Author</option>
                                    </select>
                                </div>
                            </div>
                            
                            <!-- Advanced Configuration -->
                            <div class="form-group">
                                <label for="allowed-domains">Allowed Domains (one per line)</label>
                                <textarea id="allowed-domains" rows="3" placeholder="example.com&#10;news.example.com"></textarea>
                            </div>
                        </div>

                        <!-- API Key Display -->
                        <div class="form-section" id="api-key-section" style="display: none;">
                            <h3>API Access</h3>
                            <div class="api-key-display">
                                <label>API Key</label>
                                <div class="api-key-container">
                                    <input type="text" id="api-key" readonly>
                                    <button type="button" class="btn btn-secondary" onclick="copyApiKey()">
                                        <i class="fas fa-copy"></i> Copy
                                    </button>
                                </div>
                            </div>
                            <div class="api-usage-info">
                                <h4>Usage Instructions</h4>
                                <div class="code-block">
                                    <pre id="api-usage-example">API usage example will appear here</pre>
                                </div>
                            </div>
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" onclick="closeSourceModal()">Cancel</button>
                    <button type="button" class="btn btn-primary" onclick="saveSource()">Save Source</button>
                </div>
            </div>
        </div>

        <!-- Content Preview Modal -->
        <div id="content-modal" class="modal" style="display: none;">
            <div class="modal-content large">
                <div class="modal-header">
                    <h2 id="content-modal-title">Content Preview</h2>
                    <button class="modal-close" onclick="closeContentModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div id="content-preview">
                        <!-- Dynamic content preview -->
                    </div>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" onclick="closeContentModal()">Close</button>
                    <button type="button" class="btn btn-danger" onclick="rejectContent()" id="reject-btn">
                        <i class="fas fa-times"></i> Reject
                    </button>
                    <button type="button" class="btn btn-success" onclick="approveContent()" id="approve-btn">
                        <i class="fas fa-check"></i> Approve & Publish
                    </button>
                </div>
            </div>
        </div>

    <script src="/static/js/admin/content-ingestion.js?v=20251117-02"></script>
    <script>
        // Ensure contentManager is available globally
        window.addEventListener('DOMContentLoaded', function() {
            if (typeof contentManager === 'undefined') {
                console.log('Initializing contentManager...');
                // Wait a bit for the script to load
                setTimeout(function() {
                    if (typeof ContentIngestionManager !== 'undefined') {
                        window.contentManager = new ContentIngestionManager();
                    }
                }, 100);
            }
        });
    </script>`
}

// getCommentManagementContent returns the content for comment management page
func (s *Server) getCommentManagementContent() string {
	return `
    <style>
        .filters {
            display: flex;
            gap: 1rem;
            margin-bottom: 2rem;
            flex-wrap: wrap;
            align-items: center;
        }
        
        .filter-group {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        .filter-select, .filter-input {
            padding: 0.5rem;
            border: 1px solid #d1d5db;
            border-radius: 4px;
            font-size: 0.875rem;
        }
        
        .btn {
            padding: 0.5rem 1rem;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.875rem;
            transition: background-color 0.2s;
        }
        
        .btn-primary {
            background-color: #3b82f6;
            color: white;
        }
        
        .btn-primary:hover {
            background-color: #2563eb;
        }
        
        .btn-success {
            background-color: #10b981;
            color: white;
        }
        
        .btn-success:hover {
            background-color: #059669;
        }
        
        .btn-danger {
            background-color: #ef4444;
            color: white;
        }
        
        .btn-danger:hover {
            background-color: #dc2626;
        }
        
        .btn-warning {
            background-color: #f59e0b;
            color: white;
        }
        
        .btn-warning:hover {
            background-color: #d97706;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        
        .stat-card {
            background: white;
            padding: 1.5rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        
        .stat-number {
            font-size: 2rem;
            font-weight: bold;
            color: #1f2937;
        }
        
        .stat-label {
            color: #6b7280;
            font-size: 0.875rem;
            margin-top: 0.5rem;
        }
        
        .comments-table {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .table-header {
            background-color: #f9fafb;
            padding: 1rem;
            border-bottom: 1px solid #e5e7eb;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .bulk-actions {
            display: flex;
            gap: 0.5rem;
            align-items: center;
        }
        
        .comment-item {
            padding: 1rem;
            border-bottom: 1px solid #e5e7eb;
            display: grid;
            grid-template-columns: auto 1fr auto auto;
            gap: 1rem;
            align-items: start;
        }
        
        .comment-item:last-child {
            border-bottom: none;
        }
        
        .comment-checkbox {
            margin-top: 0.25rem;
        }
        
        .comment-content {
            flex: 1;
        }
        
        .comment-meta {
            display: flex;
            gap: 1rem;
            margin-bottom: 0.5rem;
            font-size: 0.875rem;
            color: #6b7280;
        }
        
        .comment-text {
            color: #1f2937;
            line-height: 1.5;
            margin-bottom: 0.5rem;
        }
        
        .comment-article {
            font-size: 0.875rem;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .comment-article:hover {
            text-decoration: underline;
        }
        
        .status-badge {
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 500;
            text-transform: uppercase;
        }
        
        .status-pending {
            background-color: #fef3c7;
            color: #92400e;
        }
        
        .status-approved {
            background-color: #d1fae5;
            color: #065f46;
        }
        
        .status-rejected {
            background-color: #fee2e2;
            color: #991b1b;
        }
        
        .status-spam {
            background-color: #fde2e8;
            color: #be185d;
        }
        
        .comment-actions {
            display: flex;
            gap: 0.5rem;
        }
        
        .action-btn {
            padding: 0.25rem 0.5rem;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.75rem;
            transition: background-color 0.2s;
        }
        
        .spam-score {
            font-size: 0.75rem;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            background-color: #f3f4f6;
            color: #374151;
        }
        
        .spam-high {
            background-color: #fee2e2;
            color: #991b1b;
        }
        
        .spam-medium {
            background-color: #fef3c7;
            color: #92400e;
        }
        
        .spam-low {
            background-color: #d1fae5;
            color: #065f46;
        }
        
        .pagination {
            display: flex;
            justify-content: center;
            gap: 0.5rem;
            margin-top: 2rem;
        }
        
        .pagination button {
            padding: 0.5rem 1rem;
            border: 1px solid #d1d5db;
            background: white;
            cursor: pointer;
            border-radius: 4px;
        }
        
        .pagination button:hover {
            background-color: #f9fafb;
        }
        
        .pagination button.active {
            background-color: #3b82f6;
            color: white;
            border-color: #3b82f6;
        }
        
        .loading {
            text-align: center;
            padding: 2rem;
            color: #6b7280;
        }
        
        .empty-state {
            text-align: center;
            padding: 3rem;
            color: #6b7280;
        }
        
        .empty-state h3 {
            margin-bottom: 0.5rem;
            color: #374151;
        }
    </style>
    
    <div class="comment-management">
        <!-- Statistics -->
        <div class="stats-grid" id="statsGrid">
            <div class="stat-card">
                <div class="stat-number" id="totalComments">-</div>
                <div class="stat-label">Total Comments</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="pendingComments">-</div>
                <div class="stat-label">Pending Review</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="approvedComments">-</div>
                <div class="stat-label">Approved</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="spamComments">-</div>
                <div class="stat-label">Spam Detected</div>
            </div>
        </div>
        
        <!-- Filters -->
        <div class="filters">
            <div class="filter-group">
                <label for="statusFilter">Status:</label>
                <select id="statusFilter" class="filter-select">
                    <option value="">All Statuses</option>
                    <option value="pending">Pending</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                    <option value="spam">Spam</option>
                </select>
            </div>
            
            <div class="filter-group">
                <label for="searchFilter">Search:</label>
                <input type="text" id="searchFilter" class="filter-input" placeholder="Search comments...">
            </div>
            
            <button onclick="loadComments()" class="btn btn-primary">Apply Filters</button>
            <button onclick="resetFilters()" class="btn">Reset</button>
        </div>
        
        <!-- Comments Table -->
        <div class="comments-table">
            <div class="table-header">
                <div>
                    <input type="checkbox" id="selectAll" onchange="toggleSelectAll()">
                    <label for="selectAll">Select All</label>
                </div>
                <div class="bulk-actions">
                    <button onclick="bulkModerate('approve')" class="btn btn-success">Approve Selected</button>
                    <button onclick="bulkModerate('reject')" class="btn btn-danger">Reject Selected</button>
                    <button onclick="bulkModerate('spam')" class="btn btn-warning">Mark as Spam</button>
                </div>
            </div>
            
            <div id="commentsContainer">
                <div class="loading">Loading comments...</div>
            </div>
        </div>
        
        <!-- Pagination -->
        <div class="pagination" id="pagination"></div>
    </div>

    <script>
        let currentPage = 1;
        let totalPages = 1;
        const pageSize = 20;
        
        // Load comments on page load
        document.addEventListener('DOMContentLoaded', function() {
            // Check authentication
            const token = localStorage.getItem('auth_token');
            const role = localStorage.getItem('user_role');
            
            if (!token || (role !== 'admin' && role !== 'editor')) {
                window.location.href = '/admin/login';
                return;
            }
            
            loadStats();
            loadComments();
        });
        
        async function loadStats() {
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/stats', {
                    credentials: 'include',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    }
                });
                const data = await response.json();
                
                if (data.stats) {
                    document.getElementById('totalComments').textContent = data.stats.total || 0;
                    document.getElementById('pendingComments').textContent = data.stats.pending || 0;
                    document.getElementById('approvedComments').textContent = data.stats.approved || 0;
                    document.getElementById('spamComments').textContent = data.stats.spam || 0;
                }
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        }
        
        async function loadComments() {
            const container = document.getElementById('commentsContainer');
            container.innerHTML = '<div class="loading">Loading comments...</div>';
            
            try {
                const status = document.getElementById('statusFilter').value;
                const search = document.getElementById('searchFilter').value;
                const offset = (currentPage - 1) * pageSize;
                
                let url = '/api/v1/admin/comments/pending?limit=' + pageSize + '&offset=' + offset;
                
                if (status) {
                    url = '/api/v1/admin/comments/search?q=' + encodeURIComponent(search || '') + '&status=' + status + '&limit=' + pageSize + '&offset=' + offset;
                } else if (search) {
                    url = '/api/v1/admin/comments/search?q=' + encodeURIComponent(search) + '&limit=' + pageSize + '&offset=' + offset;
                }
                
                const token = localStorage.getItem('auth_token');
                const response = await fetch(url, {
                    credentials: 'include',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    }
                });
                const data = await response.json();
                
                if (data.comments && data.comments.length > 0) {
                    displayComments(data.comments);
                } else {
                    container.innerHTML = '<div class="empty-state"><h3>No comments found</h3><p>No comments match your current filters.</p></div>';
                }
                
                updatePagination(data.total || 0);
                
            } catch (error) {
                console.error('Error loading comments:', error);
                container.innerHTML = '<div class="empty-state"><h3>Error loading comments</h3><p>Please try refreshing the page.</p></div>';
            }
        }
        
        function displayComments(comments) {
            const container = document.getElementById('commentsContainer');
            
            const html = comments.map(comment => {
                return '<div class="comment-item">' +
                    '<input type="checkbox" class="comment-checkbox" value="' + comment.id + '">' +
                    '<div class="comment-content">' +
                        '<div class="comment-meta">' +
                            '<strong>' + escapeHtml(comment.author_name) + '</strong>' +
                            '<span>' + escapeHtml(comment.author_email) + '</span>' +
                            '<span>' + formatDate(comment.created_at) + '</span>' +
                            '<span class="spam-score ' + getSpamScoreClass(comment.spam_score) + '">Spam: ' + (comment.spam_score * 100).toFixed(1) + '%</span>' +
                        '</div>' +
                        '<div class="comment-text">' + escapeHtml(comment.content) + '</div>' +
                        '<a href="/en/article/' + (comment.article_slug || comment.article_id) + '" class="comment-article" target="_blank">View Article →</a>' +
                    '</div>' +
                    '<div class="status-badge status-' + comment.status + '">' + comment.status + '</div>' +
                    '<div class="comment-actions">' +
                        (comment.status === 'pending' ? 
                            '<button onclick="moderateComment(' + comment.id + ', \'approve\')" class="action-btn btn-success">Approve</button>' +
                            '<button onclick="moderateComment(' + comment.id + ', \'reject\')" class="action-btn btn-danger">Reject</button>' +
                            '<button onclick="moderateComment(' + comment.id + ', \'spam\')" class="action-btn btn-warning">Spam</button>' :
                            '<button onclick="moderateComment(' + comment.id + ', \'pending\')" class="action-btn btn-primary">Review</button>') +
                        '<button onclick="editComment(' + comment.id + ')" class="action-btn" style="background-color: #6b7280;">Edit</button>' +
                        '<button onclick="deleteComment(' + comment.id + ')" class="action-btn btn-danger">Delete</button>' +
                        '<button onclick="viewReplies(' + comment.id + ')" class="action-btn" style="background-color: #8b5cf6;">Replies</button>' +
                    '</div>' +
                '</div>';
            }).join('');
            
            container.innerHTML = html;
        }
        
        async function moderateComment(commentId, action) {
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/' + commentId + '/moderate', {
                    method: 'PUT',
                    credentials: 'include',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({ action: action })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    showNotification('Comment ' + action + 'ed successfully', 'success');
                    loadComments();
                    loadStats();
                } else {
                    showNotification(data.message || 'Failed to ' + action + ' comment', 'error');
                }
            } catch (error) {
                console.error('Error moderating comment:', error);
                showNotification('Failed to ' + action + ' comment', 'error');
            }
        }
        
        async function bulkModerate(action) {
            const checkboxes = document.querySelectorAll('.comment-checkbox:checked');
            const commentIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
            
            if (commentIds.length === 0) {
                showNotification('Please select comments to moderate', 'warning');
                return;
            }
            
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/bulk-moderate', {
                    method: 'PUT',
                    credentials: 'include',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({ 
                        comment_ids: commentIds, 
                        action: action 
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    showNotification(commentIds.length + ' comments ' + action + 'ed successfully', 'success');
                    loadComments();
                    loadStats();
                    document.getElementById('selectAll').checked = false;
                } else {
                    showNotification(data.message || 'Failed to ' + action + ' comments', 'error');
                }
            } catch (error) {
                console.error('Error bulk moderating comments:', error);
                showNotification('Failed to ' + action + ' comments', 'error');
            }
        }
        
        function toggleSelectAll() {
            const selectAll = document.getElementById('selectAll');
            const checkboxes = document.querySelectorAll('.comment-checkbox');
            
            checkboxes.forEach(checkbox => {
                checkbox.checked = selectAll.checked;
            });
        }
        
        function resetFilters() {
            document.getElementById('statusFilter').value = '';
            document.getElementById('searchFilter').value = '';
            currentPage = 1;
            loadComments();
        }
        
        function updatePagination(total) {
            totalPages = Math.ceil(total / pageSize);
            const pagination = document.getElementById('pagination');
            
            if (totalPages <= 1) {
                pagination.innerHTML = '';
                return;
            }
            
            let html = '';
            
            if (currentPage > 1) {
                html += '<button onclick="changePage(' + (currentPage - 1) + ')">← Previous</button>';
            }
            
            for (let i = Math.max(1, currentPage - 2); i <= Math.min(totalPages, currentPage + 2); i++) {
                html += '<button onclick="changePage(' + i + ')" class="' + (i === currentPage ? 'active' : '') + '">' + i + '</button>';
            }
            
            if (currentPage < totalPages) {
                html += '<button onclick="changePage(' + (currentPage + 1) + ')">Next →</button>';
            }
            
            pagination.innerHTML = html;
        }
        
        function changePage(page) {
            currentPage = page;
            loadComments();
        }
        
        function getSpamScoreClass(score) {
            if (score > 0.7) return 'spam-high';
            if (score > 0.3) return 'spam-medium';
            return 'spam-low';
        }
        
        function formatDate(dateString) {
            const date = new Date(dateString);
            return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
        }
        
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        function showNotification(message, type) {
            type = type || 'info';
            const notification = document.createElement('div');
            notification.style.cssText = 'position: fixed; top: 20px; right: 20px; padding: 1rem 1.5rem; border-radius: 8px; color: white; font-weight: 500; z-index: 1000; animation: slideIn 0.3s ease-out; background-color: ' + 
                (type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : type === 'warning' ? '#f59e0b' : '#3b82f6');
            notification.textContent = message;
            
            document.body.appendChild(notification);
            
            setTimeout(function() {
                notification.style.animation = 'slideOut 0.3s ease-out';
                setTimeout(function() {
                    if (notification.parentNode) {
                        document.body.removeChild(notification);
                    }
                }, 300);
            }, 3000);
        }
        
        async function editComment(commentId) {
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/comments/' + commentId, {
                    headers: {
                        'Authorization': 'Bearer ' + token
                    }
                });
                const data = await response.json();
                
                if (response.ok) {
                    const newContent = prompt('Edit comment content:', data.comment.content);
                    if (newContent && newContent !== data.comment.content) {
                        await updateComment(commentId, newContent);
                    }
                } else {
                    showNotification('Failed to load comment for editing', 'error');
                }
            } catch (error) {
                console.error('Error editing comment:', error);
                showNotification('Failed to edit comment', 'error');
            }
        }
        
        async function updateComment(commentId, content) {
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/' + commentId + '/edit', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({ content: content })
                });
                
                if (response.ok) {
                    showNotification('Comment updated successfully', 'success');
                    loadComments();
                } else {
                    showNotification('Failed to update comment', 'error');
                }
            } catch (error) {
                console.error('Error updating comment:', error);
                showNotification('Failed to update comment', 'error');
            }
        }
        
        async function deleteComment(commentId) {
            if (!confirm('Are you sure you want to delete this comment? This action cannot be undone.')) {
                return;
            }
            
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/' + commentId, {
                    method: 'DELETE',
                    headers: {
                        'Authorization': 'Bearer ' + token
                    }
                });
                
                if (response.ok) {
                    showNotification('Comment deleted successfully', 'success');
                    loadComments();
                    loadStats();
                } else {
                    showNotification('Failed to delete comment', 'error');
                }
            } catch (error) {
                console.error('Error deleting comment:', error);
                showNotification('Failed to delete comment', 'error');
            }
        }
        
        async function viewReplies(commentId) {
            try {
                const token = localStorage.getItem('auth_token');
                const response = await fetch('/api/v1/admin/comments/' + commentId + '/replies', {
                    headers: {
                        'Authorization': 'Bearer ' + token
                    }
                });
                const data = await response.json();
                
                if (response.ok && data.replies && data.replies.length > 0) {
                    showRepliesModal(data.replies);
                } else {
                    showNotification('No replies found for this comment', 'info');
                }
            } catch (error) {
                console.error('Error loading replies:', error);
                showNotification('Failed to load replies', 'error');
            }
        }
        
        function showRepliesModal(replies) {
            const modal = document.createElement('div');
            modal.style.cssText = 'position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: flex; justify-content: center; align-items: center; z-index: 1000;';
            
            const content = document.createElement('div');
            content.style.cssText = 'background: white; padding: 2rem; border-radius: 8px; max-width: 800px; max-height: 80vh; overflow-y: auto; width: 90%;';
            
            content.innerHTML = '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;"><h3>Comment Replies</h3><button onclick="this.closest(\'div[style*=fixed]\').remove()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">&times;</button></div><div>' +
                replies.map(function(reply) {
                    return '<div style="border: 1px solid #e5e7eb; padding: 1rem; margin-bottom: 1rem; border-radius: 4px;"><div style="font-weight: bold; margin-bottom: 0.5rem;">' + escapeHtml(reply.author_name) + '</div><div style="color: #6b7280; font-size: 0.875rem; margin-bottom: 0.5rem;">' + formatDate(reply.created_at) + '</div><div>' + escapeHtml(reply.content) + '</div><div style="margin-top: 0.5rem;"><span class="status-badge status-' + reply.status + '">' + reply.status + '</span></div></div>';
                }).join('') +
            '</div>';
            
            modal.appendChild(content);
            document.body.appendChild(modal);
        }

        // Add CSS for animations
        const style = document.createElement('style');
        style.textContent = '@keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } } @keyframes slideOut { from { transform: translateX(0); opacity: 1; } to { transform: translateX(100%); opacity: 0; } }';
        document.head.appendChild(style);
    </script>`
}
