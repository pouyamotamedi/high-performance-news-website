package server

import (
	"github.com/gin-gonic/gin"
)

// renderAdminSettings renders the settings admin page with real configuration data
func (s *Server) renderAdminSettings(c *gin.Context) {
	title := "Settings & Configuration"

	settingsContent := `
		<div class="dashboard-card" style="grid-column: 1 / -1;">
			<div class="card-title">⚙️ Site Configuration</div>
			<div id="site-config-form">
				<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem;">
					<div class="form-group">
						<label for="site-name">Site Name</label>
						<input type="text" id="site-name" class="form-input" placeholder="Loading...">
					</div>
					<div class="form-group">
						<label for="site-description">Site Description</label>
						<input type="text" id="site-description" class="form-input" placeholder="Loading...">
					</div>
					<div class="form-group">
						<label for="site-url">Site URL</label>
						<input type="url" id="site-url" class="form-input" placeholder="Loading...">
					</div>
					<div class="form-group">
						<label for="admin-email">Admin Email</label>
						<input type="email" id="admin-email" class="form-input" placeholder="Loading...">
					</div>
				</div>
				<button onclick="saveSiteConfig()" class="action-button" style="margin-top: 1rem;">💾 Save Site Settings</button>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">🎨 Appearance</div>
			<div>
				<a href="/admin/themes" class="action-button">🎨 Theme Management</a>
				<a href="/admin/widgets" class="action-button">📦 Widget Management</a>
			</div>
			<div style="margin-top: 1rem;">
				<p>Current Theme: <span id="current-theme" class="metric">Loading...</span></p>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">⚡ Performance</div>
			<div id="performance-settings">
				<div class="form-group">
					<label>
						<input type="checkbox" id="cache-enabled" checked> Enable Caching
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" id="static-generation" checked> Enable Static Generation
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" id="compression-enabled" checked> Enable Compression
					</label>
				</div>
				<div class="form-group">
					<label for="cache-ttl">Cache TTL (seconds)</label>
					<input type="number" id="cache-ttl" class="form-input" value="3600" min="60" max="86400">
				</div>
				<button onclick="savePerformanceSettings()" class="action-button">💾 Save Performance Settings</button>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">🔒 Security</div>
			<div id="security-settings">
				<div class="form-group">
					<label>
						<input type="checkbox" id="2fa-required"> Require 2FA for Admins
					</label>
				</div>
				<div class="form-group">
					<label>
						<input type="checkbox" id="rate-limiting" checked> Enable Rate Limiting
					</label>
				</div>
				<div class="form-group">
					<label for="session-timeout">Session Timeout (minutes)</label>
					<input type="number" id="session-timeout" class="form-input" value="60" min="5" max="1440">
				</div>
				<button onclick="saveSecuritySettings()" class="action-button">💾 Save Security Settings</button>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">🚀 Feature Flags</div>
			<div id="feature-flags">
				<p>Loading feature flags...</p>
			</div>
		</div>

		<div class="dashboard-card">
			<div class="card-title">📊 System Status</div>
			<div id="system-status">
				<p>Loading system status...</p>
			</div>
		</div>

		<style>
			.form-group {
				margin-bottom: 1rem;
			}
			.form-group label {
				display: block;
				margin-bottom: 0.5rem;
				font-weight: 500;
				color: #374151;
			}
			.form-input {
				width: 100%;
				padding: 0.5rem;
				border: 1px solid #d1d5db;
				border-radius: 4px;
				font-size: 1rem;
			}
			.form-input:focus {
				outline: none;
				border-color: #3b82f6;
				box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
			}
			.feature-flag-item {
				display: flex;
				justify-content: space-between;
				align-items: center;
				padding: 0.75rem;
				border-bottom: 1px solid #e5e7eb;
			}
			.feature-flag-item:last-child {
				border-bottom: none;
			}
			.toggle-switch {
				position: relative;
				width: 50px;
				height: 26px;
			}
			.toggle-switch input {
				opacity: 0;
				width: 0;
				height: 0;
			}
			.toggle-slider {
				position: absolute;
				cursor: pointer;
				top: 0;
				left: 0;
				right: 0;
				bottom: 0;
				background-color: #ccc;
				transition: 0.4s;
				border-radius: 26px;
			}
			.toggle-slider:before {
				position: absolute;
				content: "";
				height: 20px;
				width: 20px;
				left: 3px;
				bottom: 3px;
				background-color: white;
				transition: 0.4s;
				border-radius: 50%;
			}
			input:checked + .toggle-slider {
				background-color: #3b82f6;
			}
			input:checked + .toggle-slider:before {
				transform: translateX(24px);
			}
		</style>

		<script>
			const token = localStorage.getItem('auth_token');
			const headers = {
				'Authorization': 'Bearer ' + token,
				'Content-Type': 'application/json'
			};

			// Load all settings on page load
			document.addEventListener('DOMContentLoaded', function() {
				loadSiteConfig();
				loadFeatureFlags();
				loadSystemStatus();
				loadCurrentTheme();
			});

			async function loadSiteConfig() {
				try {
					const response = await fetch('/api/v1/admin/config', { headers });
					if (response.ok) {
						const data = await response.json();
						if (data.data) {
							document.getElementById('site-name').value = data.data['site_name']?.value || '';
							document.getElementById('site-description').value = data.data['site_description']?.value || '';
							document.getElementById('site-url').value = data.data['site_url']?.value || '';
							document.getElementById('admin-email').value = data.data['admin_email']?.value || '';
							document.getElementById('cache-ttl').value = data.data['cache_ttl']?.value || '3600';
						}
					}
				} catch (error) {
					console.error('Failed to load site config:', error);
				}
			}

			async function saveSiteConfig() {
				try {
					const configs = {
						'site_name': document.getElementById('site-name').value,
						'site_description': document.getElementById('site-description').value,
						'site_url': document.getElementById('site-url').value,
						'admin_email': document.getElementById('admin-email').value
					};

					for (const [key, value] of Object.entries(configs)) {
						await fetch('/api/v1/admin/config/' + key, {
							method: 'PUT',
							headers,
							body: JSON.stringify({ value })
						});
					}
					alert('Site settings saved successfully!');
				} catch (error) {
					console.error('Failed to save site config:', error);
					alert('Failed to save settings. Please try again.');
				}
			}

			async function savePerformanceSettings() {
				try {
					const configs = {
						'cache_enabled': document.getElementById('cache-enabled').checked,
						'static_generation': document.getElementById('static-generation').checked,
						'compression_enabled': document.getElementById('compression-enabled').checked,
						'cache_ttl': document.getElementById('cache-ttl').value
					};

					for (const [key, value] of Object.entries(configs)) {
						await fetch('/api/v1/admin/config/' + key, {
							method: 'PUT',
							headers,
							body: JSON.stringify({ value })
						});
					}
					alert('Performance settings saved successfully!');
				} catch (error) {
					console.error('Failed to save performance settings:', error);
					alert('Failed to save settings. Please try again.');
				}
			}

			async function saveSecuritySettings() {
				try {
					const configs = {
						'2fa_required': document.getElementById('2fa-required').checked,
						'rate_limiting': document.getElementById('rate-limiting').checked,
						'session_timeout': document.getElementById('session-timeout').value
					};

					for (const [key, value] of Object.entries(configs)) {
						await fetch('/api/v1/admin/config/' + key, {
							method: 'PUT',
							headers,
							body: JSON.stringify({ value })
						});
					}
					alert('Security settings saved successfully!');
				} catch (error) {
					console.error('Failed to save security settings:', error);
					alert('Failed to save settings. Please try again.');
				}
			}

			async function loadFeatureFlags() {
				try {
					const response = await fetch('/api/v1/admin/feature-flags', { headers });
					if (response.ok) {
						const data = await response.json();
						const container = document.getElementById('feature-flags');
						
						if (data.data && Object.keys(data.data).length > 0) {
							let html = '';
							for (const [key, flag] of Object.entries(data.data)) {
								html += '<div class="feature-flag-item">' +
									'<div>' +
									'<strong>' + (flag.name || key) + '</strong>' +
									'<p style="font-size: 0.875rem; color: #6b7280; margin: 0;">' + (flag.description || '') + '</p>' +
									'</div>' +
									'<label class="toggle-switch">' +
									'<input type="checkbox" ' + (flag.enabled ? 'checked' : '') + ' onchange="toggleFeatureFlag(\'' + key + '\', this.checked)">' +
									'<span class="toggle-slider"></span>' +
									'</label>' +
									'</div>';
							}
							container.innerHTML = html;
						} else {
							container.innerHTML = '<p style="color: #6b7280;">No feature flags configured</p>';
						}
					}
				} catch (error) {
					console.error('Failed to load feature flags:', error);
					document.getElementById('feature-flags').innerHTML = '<p style="color: #dc3545;">Failed to load feature flags</p>';
				}
			}

			async function toggleFeatureFlag(key, enabled) {
				try {
					await fetch('/api/v1/admin/feature-flags/' + key, {
						method: 'PUT',
						headers,
						body: JSON.stringify({ enabled })
					});
				} catch (error) {
					console.error('Failed to toggle feature flag:', error);
					alert('Failed to update feature flag. Please try again.');
				}
			}

			async function loadSystemStatus() {
				try {
					const response = await fetch('/api/v1/monitoring/dashboard');
					if (response.ok) {
						const data = await response.json();
						const container = document.getElementById('system-status');
						
						const healthClass = data.system_health === 'healthy' ? 'status-healthy' : 'status-degraded';
						container.innerHTML = 
							'<p>System Health: <span class="status-badge ' + healthClass + '">' + data.system_health + '</span></p>' +
							'<p>CPU Usage: <span class="metric">' + (data.system_metrics?.cpu_usage?.toFixed(1) || 0) + '%</span></p>' +
							'<p>Memory Usage: <span class="metric">' + (data.system_metrics?.memory_usage?.toFixed(1) || 0) + '%</span></p>' +
							'<p>Disk Usage: <span class="metric">' + (data.system_metrics?.disk_usage?.toFixed(1) || 0) + '%</span></p>' +
							'<p>Cache Hit Rate: <span class="metric">' + ((data.cache_metrics?.hit_rate || 0) * 100).toFixed(1) + '%</span></p>' +
							'<p>DB Connections: <span class="metric">' + (data.database_metrics?.active_connections || 0) + '/' + (data.database_metrics?.max_connections || 0) + '</span></p>';
					}
				} catch (error) {
					console.error('Failed to load system status:', error);
					document.getElementById('system-status').innerHTML = '<p style="color: #dc3545;">Failed to load system status</p>';
				}
			}

			async function loadCurrentTheme() {
				try {
					const response = await fetch('/api/v1/admin/themes/active', { headers });
					if (response.ok) {
						const data = await response.json();
						document.getElementById('current-theme').textContent = data.data?.name || 'Default';
					}
				} catch (error) {
					document.getElementById('current-theme').textContent = 'Default';
				}
			}
		</script>`

	s.renderAdminPage(c, title, "settings", settingsContent)
}

// Settings Handlers - these are now redirects to the main settings page
func (s *Server) renderGeneralSettings(c *gin.Context) {
	c.Redirect(302, "/admin/settings")
}

func (s *Server) renderPerformanceSettings(c *gin.Context) {
	c.Redirect(302, "/admin/settings")
}

func (s *Server) renderSecuritySettings(c *gin.Context) {
	c.Redirect(302, "/admin/settings")
}

func (s *Server) renderBackupSettings(c *gin.Context) {
	c.Redirect(302, "/admin/settings")
}
