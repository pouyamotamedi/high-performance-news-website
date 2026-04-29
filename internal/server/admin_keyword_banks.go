package server

import (
	"github.com/gin-gonic/gin"
)

// handleAdminKeywordBanks renders the keyword banks management page
func (s *Server) handleAdminKeywordBanks(c *gin.Context) {
	title := "Keyword Banks Management"
	content := `
	<link rel="stylesheet" href="/static/css/admin-keyword-banks.css">
	
	<div class="kb-container">
		<div class="kb-header">
			<h1>🏦 Keyword Banks Management</h1>
			<p>Create and manage custom keyword banks with dedicated URLs for auto-linking</p>
		</div>
		
		<div class="kb-actions">
			<button class="kb-btn kb-btn-primary" onclick="openCreateModal()">➕ Create New Keyword Bank</button>
			<a href="/admin/autolinking" class="kb-btn kb-btn-secondary" style="text-decoration: none;">← Back to Auto-Linking</a>
		</div>
		
		<div class="kb-card">
			<div id="loading" class="kb-loading">Loading keyword banks...</div>
			<div id="empty-state" class="kb-empty" style="display: none;">
				<h3>No Keyword Banks Yet</h3>
				<p>Create your first keyword bank to start linking keywords to custom URLs</p>
			</div>
			<table id="banks-table" class="kb-table" style="display: none;">
				<thead>
					<tr>
						<th>Name</th>
						<th>URL</th>
						<th>Keywords</th>
						<th>Status</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody id="banks-tbody"></tbody>
			</table>
		</div>
	</div>
	
	<!-- Modal -->
	<div id="modal" class="kb-modal">
		<div class="kb-modal-content">
			<div style="display: flex; justify-content: space-between; margin-bottom: 1.5rem;">
				<h2 id="modal-title">Create Keyword Bank</h2>
				<button onclick="closeModal()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">&times;</button>
			</div>
			<form id="bank-form" onsubmit="handleSubmit(event)">
				<input type="hidden" id="bank-id">
				<div class="kb-form-group">
					<label for="bank-name">Name *</label>
					<input type="text" id="bank-name" class="kb-form-control" required placeholder="e.g., External Resources">
				</div>
				<div class="kb-form-group">
					<label for="bank-url">Target URL *</label>
					<input type="url" id="bank-url" class="kb-form-control" required placeholder="https://example.com/page">
				</div>
				<div class="kb-form-group">
					<label for="bank-keywords">Keywords *</label>
					<textarea id="bank-keywords" class="kb-form-control" required placeholder="Enter one keyword per line"></textarea>
				</div>
				<div class="kb-form-group">
					<label for="bank-description">Description</label>
					<textarea id="bank-description" class="kb-form-control" placeholder="Optional description"></textarea>
				</div>
				<div class="kb-form-group">
					<label><input type="checkbox" id="bank-active" checked> Active</label>
				</div>
				<div class="kb-action-buttons">
					<button type="submit" class="kb-btn kb-btn-success">💾 Save</button>
					<button type="button" class="kb-btn kb-btn-secondary" onclick="closeModal()">Cancel</button>
				</div>
			</form>
		</div>
	</div>
	
	<script src="/static/js/admin/keyword_banks.js"></script>
	`
	s.renderAdminPage(c, title, "keyword-banks", content)
}
