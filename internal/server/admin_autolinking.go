package server

import (
	"github.com/gin-gonic/gin"
)

// handleAdminAutoLinking renders the auto-linking management page
func (s *Server) handleAdminAutoLinking(c *gin.Context) {
	title := "Auto-Linking Management"
	content := `
	<div class="dashboard-card">
		<div class="card-title">🔗 Auto-Linking System</div>
		
		<!-- System Status -->
		<div class="status-section" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; background: #f9fafb;">
			<h3 style="margin-bottom: 1rem;">📊 System Status</h3>
			<div id="systemStatus" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem;">
				<div class="stat-card">
					<div class="stat-label">Total Keywords</div>
					<div class="stat-value" id="totalKeywords">-</div>
				</div>
				<div class="stat-card">
					<div class="stat-label">Active Tags</div>
					<div class="stat-value" id="activeTags">-</div>
				</div>
				<div class="stat-card">
					<div class="stat-label">Active Keyword Banks</div>
					<div class="stat-value" id="activeBanks">-</div>
				</div>
				<div class="stat-card">
					<div class="stat-label">System Status</div>
					<div class="stat-value" id="systemStatusValue">-</div>
				</div>
			</div>
		</div>

		<!-- Global Settings -->
		<div class="settings-section" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px;">
			<h3 style="margin-bottom: 1rem;">⚙️ Global Settings</h3>
			<div style="display: grid; gap: 1rem;">
				<div>
					<label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer;">
						<input type="checkbox" id="globalAutoLinking" onchange="updateGlobalSettings()">
						<span style="font-weight: 600;">Enable Auto-Linking Globally</span>
					</label>
					<p style="color: #6b7280; font-size: 0.875rem; margin-top: 0.25rem; margin-left: 1.5rem;">
						When enabled, new articles will have auto-linking enabled by default
					</p>
				</div>
				<div>
					<label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer;">
						<input type="checkbox" id="contentIngestionAutoLinking" onchange="updateGlobalSettings()">
						<span style="font-weight: 600;">Enable for Content Ingestion API</span>
					</label>
					<p style="color: #6b7280; font-size: 0.875rem; margin-top: 0.25rem; margin-left: 1.5rem;">
						Apply auto-linking to articles created through the Content Ingestion API
					</p>
				</div>
			</div>
		</div>

		<!-- Reprocess Articles -->
		<div class="reprocess-section" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; background: #fef3c7;">
			<h3 style="margin-bottom: 0.5rem;">🔄 Reprocess Existing Articles</h3>
			<p style="color: #92400e; margin-bottom: 1rem;">
				Apply auto-linking to all existing articles. This will update article content with keyword links.
			</p>
			<button onclick="reprocessAllArticles()" class="action-button" style="background-color: #f59e0b;">
				⚡ Reprocess All Articles
			</button>
		</div>

		<!-- Keyword Management -->
		<div class="keywords-section" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px;">
			<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
				<h3>🏷️ Keyword Bank</h3>
				<div style="display: flex; gap: 0.5rem;">
					<a href="/admin/keyword-banks" class="action-button" style="background-color: #3b82f6; text-decoration: none;">
						🏦 Manage Keyword Banks
					</a>
					<button onclick="refreshKeywords()" class="action-button" style="background-color: #10b981;">
						🔄 Refresh Keywords
					</button>
					<a href="/admin/tags" class="action-button" style="background-color: #6366f1;">
						➕ Manage Tags
					</a>
				</div>
			</div>
			<p style="color: #6b7280; margin-bottom: 1rem;">
				Keywords are managed through tags. Each tag can have multiple keywords that will be automatically linked in articles.
			</p>
			
			<!-- Search and Filter -->
			<div style="margin-bottom: 1rem;">
				<input type="text" id="keywordSearch" placeholder="Search keywords..." 
					   onkeyup="filterKeywords()"
					   style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
			</div>
			
			<!-- Keywords List -->
			<div id="keywordsList" style="max-height: 400px; overflow-y: auto;">
				<div style="text-align: center; padding: 2rem; color: #9ca3af;">
					Loading keywords...
				</div>
			</div>
		</div>

		<!-- Conflict Detection -->
		<div class="conflicts-section" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px;">
			<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
				<h3>⚠️ Keyword Conflicts</h3>
				<button onclick="checkConflicts()" class="action-button" style="background-color: #f59e0b;">
					🔍 Check Conflicts
				</button>
			</div>
			<p style="color: #6b7280; margin-bottom: 1rem;">
				Conflicts occur when the same keyword is assigned to multiple tags. The system uses longest-match priority.
			</p>
			<div id="conflictsList">
				<div style="text-align: center; padding: 1rem; color: #9ca3af;">
					Click "Check Conflicts" to scan for keyword conflicts
				</div>
			</div>
		</div>

		<!-- Test Auto-Linking -->
		<div class="test-section" style="padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; background: #f9fafb;">
			<h3 style="margin-bottom: 1rem;">🧪 Test Auto-Linking</h3>
			<p style="color: #6b7280; margin-bottom: 1rem;">
				Test how auto-linking will work on sample text
			</p>
			<textarea id="testText" placeholder="Enter text to test auto-linking..." 
					  style="width: 100%; min-height: 150px; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; margin-bottom: 1rem;"></textarea>
			<button onclick="testAutoLinking()" class="action-button" style="background-color: #8b5cf6;">
				🔗 Test Auto-Linking
			</button>
			<div id="testResult" style="margin-top: 1rem; padding: 1rem; border: 1px solid #d1d5db; border-radius: 6px; background: white; display: none;">
				<h4 style="margin-bottom: 0.5rem;">Result:</h4>
				<div id="testResultContent"></div>
			</div>
		</div>
	</div>

	<style>
		.stat-card {
			padding: 1rem;
			background: white;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			text-align: center;
		}
		.stat-label {
			font-size: 0.875rem;
			color: #6b7280;
			margin-bottom: 0.5rem;
		}
		.stat-value {
			font-size: 1.5rem;
			font-weight: 700;
			color: #1f2937;
		}
		.keyword-item {
			padding: 0.75rem;
			border-bottom: 1px solid #e5e7eb;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.keyword-item:hover {
			background: #f9fafb;
		}
		.keyword-text {
			font-weight: 600;
			color: #1f2937;
		}
		.keyword-tag {
			display: inline-block;
			padding: 0.25rem 0.75rem;
			background: #e0e7ff;
			color: #3730a3;
			border-radius: 9999px;
			font-size: 0.875rem;
		}
		.conflict-item {
			padding: 0.75rem;
			background: #fef3c7;
			border: 1px solid #fbbf24;
			border-radius: 6px;
			margin-bottom: 0.5rem;
		}
	</style>

	<script src="/static/js/admin/autolinking.js?v=20251119-01"></script>
	`
	s.renderAdminPage(c, title, "autolinking", content)
}
