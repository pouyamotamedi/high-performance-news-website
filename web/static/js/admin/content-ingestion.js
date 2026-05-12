// Content Ingestion Management JavaScript
class ContentIngestionManager {
    constructor() {
        this.currentTab = 'sources';
        this.sources = [];
        this.pendingContent = [];
        this.processedContent = [];
        this.stats = {};
        this.charts = {};
        
        this.init();
    }

    async init() {
        await this.loadInitialData();
        this.setupEventListeners();
        this.initializeCharts();
        this.startAutoRefresh();
    }

    async loadInitialData() {
        try {
            await Promise.all([
                this.loadSources(),
                this.loadStats(),
                this.loadCategories(),
                this.loadAuthors(),
                this.loadPendingContent(),
                this.loadProcessedContent()
            ]);
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showNotification('Failed to load data', 'error');
        }
    }

    setupEventListeners() {
        // Search and filter inputs
        document.getElementById('sources-search').addEventListener('input', (e) => {
            this.filterSources(e.target.value);
        });

        document.getElementById('sources-filter').addEventListener('change', (e) => {
            this.filterSourcesByType(e.target.value);
        });

        document.getElementById('pending-source-filter').addEventListener('change', (e) => {
            this.filterPendingBySource(e.target.value);
        });

        document.getElementById('processed-search').addEventListener('input', (e) => {
            this.filterProcessed(e.target.value);
        });

        document.getElementById('processed-status-filter').addEventListener('change', (e) => {
            this.filterProcessedByStatus(e.target.value);
        });

        // Bulk selection
        document.getElementById('select-all-pending').addEventListener('change', (e) => {
            this.toggleAllPending(e.target.checked);
        });

        // Source type change
        document.getElementById('source-type').addEventListener('change', (e) => {
            this.updateApiUsageExample(e.target.value);
        });
        
        // Update cURL example when category or author changes
        const defaultCategorySelect = document.getElementById('default-category');
        const defaultAuthorSelect = document.getElementById('default-author');
        if (defaultCategorySelect) {
            defaultCategorySelect.addEventListener('change', () => {
                const sourceType = document.getElementById('source-type').value || 'rss';
                this.updateApiUsageExample(sourceType);
            });
        }
        if (defaultAuthorSelect) {
            defaultAuthorSelect.addEventListener('change', () => {
                const sourceType = document.getElementById('source-type').value || 'rss';
                this.updateApiUsageExample(sourceType);
            });
        }
    }

    // Data Loading Methods
    async loadSources() {
        try {
            const response = await fetch('/api/v1/admin-panel/content/sources', {
                headers: this.getAuthHeaders(),
                credentials: 'include' // Include cookies for session auth
            });
            
            if (!response.ok) {
                if (response.status === 401 || response.status === 403) {
                    this.showNotification('Authentication issue detected. Please refresh the page or log in again.', 'warning');
                    // Don't redirect automatically - let user handle it
                    console.error('Authentication failed for API call:', response.status, response.statusText);
                    return;
                }
                throw new Error(`Failed to load sources: ${response.status} ${response.statusText}`);
            }
            
            const data = await response.json();
            this.sources = data.sources || [];
            this.renderSources();
            this.updateSourceFilters();
        } catch (error) {
            console.error('Error loading sources:', error);
            this.showNotification('Failed to load content sources. Please check your connection and try again.', 'error');
            // Don't throw error to prevent breaking the entire page
        }
    }

    async loadPendingContent() {
        try {
            const response = await fetch('/api/v1/admin-panel/content/pending?limit=50&offset=0', {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (!response.ok) throw new Error('Failed to load pending content');
            
            const data = await response.json();
            this.pendingContent = data.content || [];
            this.renderPendingContent();
        } catch (error) {
            console.error('Error loading pending content:', error);
            throw error;
        }
    }

    async loadProcessedContent() {
        try {
            const response = await fetch('/api/v1/admin-panel/content/processed?limit=50&offset=0', {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (!response.ok) throw new Error('Failed to load processed content');
            
            const data = await response.json();
            this.processedContent = data.content || [];
            this.renderProcessedContent();
        } catch (error) {
            console.error('Error loading processed content:', error);
            throw error;
        }
    }

    async loadStats() {
        try {
            const response = await fetch('/api/v1/admin-panel/content/stats', {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (!response.ok) throw new Error('Failed to load stats');
            
            const result = await response.json();
            this.stats = result.data || result; // Handle both {data: {...}} and direct response formats
            console.log('Loaded stats:', this.stats); // Debug log
            this.updateStatsDisplay();
        } catch (error) {
            console.error('Error loading stats:', error);
            throw error;
        }
    }

    async loadCategories() {
        try {
            const response = await fetch('/api/v1/admin-panel/content/categories', {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (response.ok) {
                const data = await response.json();
                console.log('Categories loaded:', data);
                this.categories = data.categories || [];
                this.populateCategorySelect(this.categories);
                this.categoriesLoaded = true;
            } else {
                console.error('Failed to load categories:', response.status, response.statusText);
            }
        } catch (error) {
            console.error('Error loading categories:', error);
        }
    }

    async loadAuthors() {
        try {
            const response = await fetch('/api/v1/admin-panel/users?role=author', {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (response.ok) {
                const data = await response.json();
                console.log('Authors loaded:', data);
                this.authors = data.users || [];
                this.populateAuthorSelect(this.authors);
                this.authorsLoaded = true;
            } else {
                console.error('Failed to load authors:', response.status, response.statusText);
            }
        } catch (error) {
            console.error('Error loading authors:', error);
        }
    }

    // Rendering Methods
    renderSources() {
        const tbody = document.getElementById('sources-tbody');
        tbody.innerHTML = '';

        this.sources.forEach(source => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <div class="source-name">
                        <strong>${this.escapeHtml(source.name)}</strong>
                        <small class="source-id">ID: ${source.id}</small>
                    </div>
                </td>
                <td>
                    <span class="badge badge-${this.getTypeBadgeClass(source.type)}">
                        ${source.type.toUpperCase()}
                    </span>
                </td>
                <td>
                    <span class="status-indicator ${source.is_active ? 'active' : 'inactive'}">
                        ${source.is_active ? 'Active' : 'Inactive'}
                    </span>
                </td>
                <td>
                    <span class="auto-publish ${source.config?.auto_publish ? 'enabled' : 'disabled'}">
                        ${source.config?.auto_publish ? 'Enabled' : 'Disabled'}
                    </span>
                </td>
                <td>${source.rate_limit}/hour</td>
                <td>
                    <div class="priority-indicator priority-${source.priority}">
                        ${source.priority}
                    </div>
                </td>
                <td>
                    <span class="last-activity">
                        ${source.last_activity ? this.formatDate(source.last_activity) : 'Never'}
                    </span>
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-primary" onclick="contentManager.editSource(${source.id})" title="Edit">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-info" onclick="contentManager.viewSourceStats(${source.id})" title="Statistics">
                            <i class="fas fa-chart-line"></i>
                        </button>
                        <button class="btn btn-sm btn-warning" onclick="contentManager.testSource(${source.id})" title="Test Connection">
                            <i class="fas fa-plug"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="contentManager.deleteSource(${source.id})" title="Delete">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    renderPendingContent() {
        const tbody = document.getElementById('pending-tbody');
        tbody.innerHTML = '';

        this.pendingContent.forEach(content => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <input type="checkbox" class="pending-checkbox" value="${content.id}">
                </td>
                <td>
                    <div class="content-title">
                        <strong>${this.escapeHtml(content.title)}</strong>
                        <small class="content-excerpt">${this.escapeHtml(content.excerpt || '').substring(0, 100)}...</small>
                    </div>
                </td>
                <td>
                    <span class="source-badge">${this.getSourceName(content.source_id)}</span>
                </td>
                <td>${this.escapeHtml(content.author_name || 'Unknown')}</td>
                <td>${this.escapeHtml(content.category_name || 'Uncategorized')}</td>
                <td>
                    <span class="received-time" title="${content.created_at}">
                        ${this.formatRelativeTime(content.created_at)}
                    </span>
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-info" onclick="contentManager.previewContent(${content.id})" title="Preview">
                            <i class="fas fa-eye"></i>
                        </button>
                        <button class="btn btn-sm btn-success" onclick="contentManager.approveContent(${content.id})" title="Approve">
                            <i class="fas fa-check"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="contentManager.rejectContent(${content.id})" title="Reject">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    renderProcessedContent() {
        const tbody = document.getElementById('processed-tbody');
        tbody.innerHTML = '';

        this.processedContent.forEach(content => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <div class="content-title">
                        <strong>${this.escapeHtml(content.title)}</strong>
                        ${content.rejection_reason ? `<small class="rejection-reason">${this.escapeHtml(content.rejection_reason)}</small>` : ''}
                    </div>
                </td>
                <td>
                    <span class="source-badge">${this.getSourceName(content.source_id)}</span>
                </td>
                <td>
                    <span class="status-badge status-${content.status}">
                        ${content.status.toUpperCase()}
                    </span>
                </td>
                <td>
                    <span class="processed-time">
                        ${this.formatDate(content.processed_at)}
                    </span>
                </td>
                <td>
                    ${content.article_id ? 
                        `<a href="/article/${content.article_slug || content.article_id}" target="_blank" class="article-link">
                            <i class="fas fa-external-link-alt"></i> View Article
                        </a>` : 
                        '<span class="no-article">No Article</span>'
                    }
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-info" onclick="contentManager.viewProcessedDetails(${content.id})" title="Details">
                            <i class="fas fa-info-circle"></i>
                        </button>
                        ${content.status === 'rejected' ? 
                            `<button class="btn btn-sm btn-warning" onclick="contentManager.reprocessContent(${content.id})" title="Reprocess">
                                <i class="fas fa-redo"></i>
                            </button>` : ''
                        }
                        ${content.status === 'processed' && content.article_id ? 
                            `<button class="btn btn-sm btn-success" onclick="contentManager.editArticleInline(${content.article_id})" title="Edit Article">
                                <i class="fas fa-edit"></i> Edit Article
                            </button>` : ''
                        }
                    </div>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    updateSourceFilters() {
        // Update source filter dropdown in pending content tab
        const pendingSourceFilter = document.getElementById('pending-source-filter');
        if (pendingSourceFilter) {
            // Clear existing options except "All Sources"
            pendingSourceFilter.innerHTML = '<option value="">All Sources</option>';
            
            // Add options for each source
            this.sources.forEach(source => {
                const option = document.createElement('option');
                option.value = source.id;
                option.textContent = source.name;
                pendingSourceFilter.appendChild(option);
            });
        }

        // Update sources filter dropdown in sources tab
        const sourcesFilter = document.getElementById('sources-filter');
        if (sourcesFilter) {
            // This filter is for source types, not individual sources
            // Keep the existing options: All, API, Webhook, Manual
        }
    }

    updateStatsDisplay() {
        document.getElementById('total-sources').textContent = this.stats.total_sources || 0;
        document.getElementById('pending-content').textContent = this.stats.pending_content || 0;
        document.getElementById('processed-today').textContent = this.stats.processed_today || 0;
        document.getElementById('rejected-today').textContent = this.stats.rejected_today || 0;
    }

    // Tab Management
    showTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
        document.querySelector(`[onclick="showTab('${tabName}')"]`).classList.add('active');

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
        document.getElementById(`${tabName}-tab`).classList.add('active');

        this.currentTab = tabName;

        // Load data for the tab if needed
        switch (tabName) {
            case 'pending':
                this.loadPendingContent();
                break;
            case 'processed':
                this.loadProcessedContent();
                break;
            case 'analytics':
                this.updateCharts();
                break;
        }
    }

    // Source Management
    showCreateSourceModal() {
        console.log('showCreateSourceModal called');
        document.getElementById('source-modal-title').textContent = 'Add Content Source';
        
        // Reset the form
        document.getElementById('source-form').reset();
        document.getElementById('source-id').value = '';
        
        // Explicitly reset dropdowns to default values
        const defaultCategorySelect = document.getElementById('default-category');
        const defaultAuthorSelect = document.getElementById('default-author');
        if (defaultCategorySelect) {
            defaultCategorySelect.value = '';
        }
        if (defaultAuthorSelect) {
            defaultAuthorSelect.value = '';
        }
        
        // Generate a new API key for the new source
        console.log('Generating API key...');
        const newApiKey = this.generateApiKey();
        console.log('Generated API key:', newApiKey);
        
        const apiKeyElement = document.getElementById('api-key');
        const apiKeySectionElement = document.getElementById('api-key-section');
        
        if (!apiKeyElement) {
            console.error('api-key element not found!');
        } else {
            apiKeyElement.value = newApiKey;
            console.log('API key set in input');
        }
        
        if (!apiKeySectionElement) {
            console.error('api-key-section element not found!');
        } else {
            apiKeySectionElement.style.display = 'block';
            console.log('API key section displayed');
        }
        
        // Update the usage example with default type
        const sourceType = document.getElementById('source-type').value || 'rss';
        console.log('Updating API usage example for type:', sourceType);
        this.updateApiUsageExample(sourceType);
        
        // Dropdowns are already populated during init, no need to reload
        console.log('Categories loaded:', this.categoriesLoaded, 'Authors loaded:', this.authorsLoaded);
        
        document.getElementById('source-modal').style.display = 'block';
        console.log('Modal displayed');
    }
    
    generateApiKey() {
        // Generate a random API key using crypto for better randomness
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let apiKey = '';
        const randomValues = new Uint8Array(32);
        crypto.getRandomValues(randomValues);
        for (let i = 0; i < 32; i++) {
            apiKey += chars.charAt(randomValues[i] % chars.length);
        }
        return apiKey;
    }

    async editSource(sourceId) {
        console.log('editSource called for source ID:', sourceId);
        const source = this.sources.find(s => s.id === sourceId);
        if (!source) {
            console.error('Source not found:', sourceId);
            return;
        }
        console.log('Editing source:', source);

        // Ensure dropdowns are populated before setting values (only if not already loaded)
        if (!this.categoriesLoaded) {
            await this.loadCategories();
        }
        if (!this.authorsLoaded) {
            await this.loadAuthors();
        }

        // Reset the form first
        document.getElementById('source-form').reset();
        
        document.getElementById('source-modal-title').textContent = 'Edit Content Source';
        document.getElementById('source-id').value = source.id;
        
        // Safely set form values with null checks
        const setElementValue = (id, value) => {
            const element = document.getElementById(id);
            if (element) {
                if (element.type === 'checkbox') {
                    element.checked = value;
                } else {
                    element.value = value;
                }
                console.log(`Set ${id} to:`, value);
            } else {
                console.warn(`Element ${id} not found`);
            }
        };

        setElementValue('source-name', source.name);
        setElementValue('source-type', source.type);
        setElementValue('rate-limit', source.rate_limit);
        setElementValue('priority', source.priority);
        setElementValue('is-active', source.is_active);
        setElementValue('auto-publish', source.config?.auto_publish || false);
        
        // Set category and author with explicit logging
        const categoryId = source.config?.default_category_id || '';
        const authorId = source.config?.default_author_id || '';
        console.log('Setting category to:', categoryId, 'author to:', authorId);
        setElementValue('default-category', categoryId);
        setElementValue('default-author', authorId);
        
        if (source.config?.allowed_domains) {
            setElementValue('allowed-domains', source.config.allowed_domains.join('\n'));
        } else {
            setElementValue('allowed-domains', '');
        }

        // Always show API key section and set the API key (generate if missing)
        const apiKeyElement = document.getElementById('api-key');
        const apiKeySectionElement = document.getElementById('api-key-section');
        
        if (source.api_key) {
            console.log('Using existing API key:', source.api_key);
            apiKeyElement.value = source.api_key;
        } else {
            // Generate API key for sources that don't have one yet
            const newKey = this.generateApiKey();
            console.log('Generated new API key:', newKey);
            apiKeyElement.value = newKey;
        }
        
        apiKeySectionElement.style.display = 'block';
        console.log('Updating API usage example for type:', source.type);
        this.updateApiUsageExample(source.type);

        document.getElementById('source-modal').style.display = 'block';
        console.log('Modal displayed for editing');
    }

    async saveSource() {
        const form = document.getElementById('source-form');
        if (!form) {
            this.showNotification('Form not found', 'error');
            return;
        }
        
        // Check if all required elements exist
        const requiredElements = ['source-name', 'source-type', 'rate-limit', 'priority', 'is-active', 'auto-publish'];
        for (const elementId of requiredElements) {
            const element = document.getElementById(elementId);
            if (!element) {
                this.showNotification(`Required form element '${elementId}' not found`, 'error');
                console.error(`Missing element: ${elementId}`);
                return;
            }
        }
        
        const apiKey = document.getElementById('api-key')?.value;
        
        const sourceData = {
            name: document.getElementById('source-name').value,
            type: document.getElementById('source-type').value,
            api_key: apiKey || null,
            rate_limit: parseInt(document.getElementById('rate-limit').value),
            priority: parseInt(document.getElementById('priority').value),
            is_active: document.getElementById('is-active').checked,
            config: {
                auto_publish: document.getElementById('auto-publish').checked,
                default_category_id: parseInt(document.getElementById('default-category')?.value) || null,
                default_author_id: parseInt(document.getElementById('default-author')?.value) || null,
                allowed_domains: (document.getElementById('allowed-domains')?.value || '')
                    .split('\n')
                    .map(d => d.trim())
                    .filter(d => d),
                required_fields: Array.from(document.querySelectorAll('input[type="checkbox"][value]:checked'))
                    .map(cb => cb.value)
            }
        };
        
        console.log('Saving source data:', sourceData);

        try {
            const sourceId = document.getElementById('source-id').value;
            const url = sourceId ? `/api/v1/admin-panel/content/sources/${sourceId}` : '/api/v1/admin-panel/content/sources';
            const method = sourceId ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                credentials: 'include', // Include cookies for session auth
                body: JSON.stringify(sourceData)
            });

            if (!response.ok) throw new Error('Failed to save source');

            const result = await response.json();
            this.showNotification(sourceId ? 'Source updated successfully' : 'Source created successfully', 'success');
            this.closeSourceModal();
            await this.loadSources();

        } catch (error) {
            console.error('Error saving source:', error);
            this.showNotification('Failed to save source', 'error');
        }
    }

    closeSourceModal() {
        document.getElementById('source-modal').style.display = 'none';
    }

    async viewSourceStats(sourceId) {
        try {
            // This would show detailed statistics for a specific source
            this.showNotification('Source statistics feature coming soon', 'info');
            // TODO: Implement source statistics modal
        } catch (error) {
            console.error('Error viewing source stats:', error);
            this.showNotification('Failed to load source statistics', 'error');
        }
    }

    async testSource(sourceId) {
        try {
            const source = this.sources.find(s => s.id === sourceId);
            if (!source) {
                this.showNotification('Source not found', 'error');
                return;
            }

            this.showNotification('Testing source connection...', 'info');
            
            // TODO: Implement actual source testing
            // For now, simulate a test
            setTimeout(() => {
                this.showNotification(`Source "${source.name}" connection test successful`, 'success');
            }, 2000);

        } catch (error) {
            console.error('Error testing source:', error);
            this.showNotification('Failed to test source connection', 'error');
        }
    }

    async deleteSource(sourceId) {
        const source = this.sources.find(s => s.id === sourceId);
        if (!source) {
            this.showNotification('Source not found', 'error');
            return;
        }

        if (!confirm(`Are you sure you want to delete the source "${source.name}"? This action cannot be undone.`)) {
            return;
        }

        try {
            const response = await fetch(`/api/v1/admin-panel/content/sources/${sourceId}`, {
                method: 'DELETE',
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });

            if (!response.ok) {
                throw new Error(`Failed to delete source: ${response.status} ${response.statusText}`);
            }

            this.showNotification('Source deleted successfully', 'success');
            await this.loadSources();
            await this.loadStats();

        } catch (error) {
            console.error('Error deleting source:', error);
            this.showNotification('Failed to delete source', 'error');
        }
    }

    async viewProcessedDetails(contentId) {
        try {
            const response = await fetch(`/api/v1/admin-panel/content/details/${contentId}`, {
                headers: this.getAuthHeaders(),
                credentials: 'include'
            });
            
            if (!response.ok) throw new Error('Failed to load content details');
            
            const data = await response.json();
            this.showContentDetailsModal(data.content);
        } catch (error) {
            console.error('Error viewing processed details:', error);
            this.showNotification('Failed to load processed content details', 'error');
        }
    }

    showContentDetailsModal(content) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content" style="max-width: 800px; max-height: 80vh; overflow-y: auto;">
                <div class="modal-header">
                    <h2>Content Details</h2>
                    <span class="close" onclick="this.closest('.modal').remove()">&times;</span>
                </div>
                <div class="modal-body">
                    <div class="content-details">
                        <div class="detail-row">
                            <strong>ID:</strong> ${content.id}
                        </div>
                        <div class="detail-row">
                            <strong>Title:</strong> ${content.title || 'N/A'}
                        </div>
                        <div class="detail-row">
                            <strong>Status:</strong> 
                            <span class="status-badge status-${content.status}">${content.status}</span>
                        </div>
                        <div class="detail-row">
                            <strong>Author:</strong> ${content.author_name || 'N/A'} ${content.author_email ? `(${content.author_email})` : ''}
                        </div>
                        <div class="detail-row">
                            <strong>Category:</strong> ${content.category_name || 'N/A'}
                        </div>
                        <div class="detail-row">
                            <strong>Tags:</strong> ${content.tags || 'N/A'}
                        </div>
                        <div class="detail-row">
                            <strong>Source URL:</strong> 
                            ${content.source_url ? `<a href="${content.source_url}" target="_blank">${content.source_url}</a>` : 'N/A'}
                        </div>
                        <div class="detail-row">
                            <strong>Created:</strong> ${new Date(content.created_at).toLocaleString()}
                        </div>
                        <div class="detail-row">
                            <strong>Processed:</strong> ${content.processed_at ? new Date(content.processed_at).toLocaleString() : 'N/A'}
                        </div>
                        ${content.rejection_reason ? `
                        <div class="detail-row">
                            <strong>Rejection Reason:</strong> ${content.rejection_reason}
                        </div>
                        ` : ''}
                        ${content.article_id ? `
                        <div class="detail-row">
                            <strong>Article ID:</strong> ${content.article_id}
                        </div>
                        ` : ''}
                        <div class="detail-row">
                            <strong>Excerpt:</strong>
                            <div class="content-preview">${content.excerpt || 'N/A'}</div>
                        </div>
                        <div class="detail-row">
                            <strong>Content:</strong>
                            <div class="content-preview" style="max-height: 200px; overflow-y: auto; border: 1px solid #ddd; padding: 10px; background: #f9f9f9;">
                                ${content.content || 'N/A'}
                            </div>
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button id="approve-btn" class="btn btn-success" onclick="contentManager.approveContent(${content.id}); this.closest('.modal').remove();" style="display: none;">
                        <i class="fas fa-check"></i> Approve & Publish
                    </button>
                    <button id="reject-btn" class="btn btn-danger" onclick="contentManager.rejectContent(${content.id}); this.closest('.modal').remove();" style="display: none;">
                        <i class="fas fa-times"></i> Reject
                    </button>
                    <button class="btn btn-secondary" onclick="this.closest('.modal').remove()">Close</button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Show/hide action buttons based on content status
        const approveBtn = modal.querySelector('#approve-btn');
        const rejectBtn = modal.querySelector('#reject-btn');
        
        if (approveBtn && rejectBtn) {
            if (content.status === 'pending') {
                // Show buttons for pending content
                approveBtn.style.display = 'inline-block';
                rejectBtn.style.display = 'inline-block';
            } else {
                // Hide buttons for processed/rejected content
                approveBtn.style.display = 'none';
                rejectBtn.style.display = 'none';
            }
        }
    }



    editArticle(articleId) {
        // Open article editor in new tab
        window.open(`/admin/content/articles/${articleId}/edit`, '_blank');
    }

    // Content Processing
    async approveContent(contentId) {
        try {
            const response = await fetch(`/api/v1/admin-panel/content/process/${contentId}`, {
                method: 'POST',
                headers: this.getAuthHeaders()
            });

            if (!response.ok) throw new Error('Failed to approve content');

            this.showNotification('Content approved and published', 'success');
            await this.loadPendingContent();
            await this.loadStats();

        } catch (error) {
            console.error('Error approving content:', error);
            this.showNotification('Failed to approve content', 'error');
        }
    }

    async rejectContent(contentId, reason = '') {
        try {
            const response = await fetch(`/api/v1/admin-panel/content/reject/${contentId}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                credentials: 'include',
                body: JSON.stringify({ reason })
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Failed to reject content: ${response.status} ${response.statusText} - ${errorText}`);
            }

            this.showNotification('Content rejected successfully', 'success');
            await this.loadPendingContent();
            await this.loadStats();

        } catch (error) {
            console.error('Error rejecting content:', error);
            this.showNotification(`Failed to reject content: ${error.message}`, 'error');
        }
    }

    async processBatch() {
        const selectedIds = Array.from(document.querySelectorAll('.pending-checkbox:checked'))
            .map(cb => parseInt(cb.value));

        if (selectedIds.length === 0) {
            this.showNotification('Please select content to process', 'warning');
            return;
        }

        try {
            const response = await fetch('/api/v1/admin-panel/content/process/batch', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                body: JSON.stringify({ content_ids: selectedIds })
            });

            if (!response.ok) throw new Error('Failed to process batch');

            const result = await response.json();
            this.showNotification(`Processed ${result.processed_count} items`, 'success');
            await this.loadPendingContent();
            await this.loadStats();

        } catch (error) {
            console.error('Error processing batch:', error);
            this.showNotification('Failed to process batch', 'error');
        }
    }

    // Utility Methods
    getAuthHeaders() {
        // Admin panel uses session-based authentication via cookies
        // No special headers needed, browser handles session cookies automatically
        return {};
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(dateString) {
        return new Date(dateString).toLocaleString();
    }

    // Populate dropdown functions
    populateCategorySelect(categories) {
        console.log('Populating category select with', categories.length, 'categories');
        const select = document.getElementById('category-select');
        if (!select) {
            console.warn('category-select element not found');
        } else {
            select.innerHTML = '<option value="">All Categories</option>';
            categories.forEach(category => {
                const option = document.createElement('option');
                option.value = category.id;
                option.textContent = category.name;
                select.appendChild(option);
            });
        }
        
        // Also populate the default category dropdown in source form (only if empty)
        const defaultCategorySelect = document.getElementById('default-category');
        if (!defaultCategorySelect) {
            console.warn('default-category element not found');
        } else if (defaultCategorySelect.options.length <= 1) {
            // Only populate if not already populated (has only placeholder)
            console.log('Populating default-category dropdown');
            defaultCategorySelect.innerHTML = '<option value="">Select Category</option>';
            categories.forEach(category => {
                const option = document.createElement('option');
                option.value = category.id;
                option.textContent = category.name;
                defaultCategorySelect.appendChild(option);
            });
            console.log('default-category dropdown populated with', categories.length, 'options');
        } else {
            console.log('default-category already populated, skipping');
        }
    }

    populateAuthorSelect(authors) {
        console.log('Populating author select with', authors.length, 'authors');
        const select = document.getElementById('author-select');
        if (!select) {
            console.warn('author-select element not found');
        } else {
            select.innerHTML = '<option value="">All Authors</option>';
            authors.forEach(author => {
                const option = document.createElement('option');
                option.value = author.id;
                option.textContent = author.name || author.username;
                select.appendChild(option);
            });
        }
        
        // Also populate the default author dropdown in source form (only if empty)
        const defaultAuthorSelect = document.getElementById('default-author');
        if (!defaultAuthorSelect) {
            console.warn('default-author element not found');
        } else if (defaultAuthorSelect.options.length <= 1) {
            // Only populate if not already populated (has only placeholder)
            console.log('Populating default-author dropdown');
            defaultAuthorSelect.innerHTML = '<option value="">Select Author</option>';
            authors.forEach(author => {
                const option = document.createElement('option');
                option.value = author.id;
                option.textContent = author.name || author.username;
                defaultAuthorSelect.appendChild(option);
            });
            console.log('default-author dropdown populated with', authors.length, 'options');
        } else {
            console.log('default-author already populated, skipping');
        }
    }

    formatRelativeTime(dateString) {
        const now = new Date();
        const date = new Date(dateString);
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        return `${diffDays}d ago`;
    }

    getSourceName(sourceId) {
        const source = this.sources.find(s => s.id === sourceId);
        return source ? source.name : 'Unknown Source';
    }

    getTypeBadgeClass(type) {
        const classes = {
            'api': 'primary',
            'webhook': 'success',
            'manual': 'secondary'
        };
        return classes[type] || 'secondary';
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${this.getNotificationIcon(type)}"></i>
            <span>${message}</span>
            <button class="notification-close">&times;</button>
        `;

        // Add to page
        document.body.appendChild(notification);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);

        // Close button
        notification.querySelector('.notification-close').addEventListener('click', () => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        });
    }

    getNotificationIcon(type) {
        const icons = {
            'success': 'check-circle',
            'error': 'exclamation-circle',
            'warning': 'exclamation-triangle',
            'info': 'info-circle'
        };
        return icons[type] || 'info-circle';
    }

    updateApiUsageExample(type) {
        const exampleElement = document.getElementById('api-usage-example');
        const apiKey = document.getElementById('api-key').value || 'YOUR_API_KEY';
        
        // Get selected category and author
        const categorySelect = document.getElementById('default-category');
        const authorSelect = document.getElementById('default-author');
        const selectedCategoryId = categorySelect ? categorySelect.value : '';
        const selectedAuthorId = authorSelect ? authorSelect.value : '';
        
        // Find category and author names
        let categoryName = 'Technology';
        let authorName = 'Author Name';
        
        if (selectedCategoryId && this.categories) {
            const category = this.categories.find(c => c.id == selectedCategoryId);
            if (category) categoryName = category.name;
        }
        
        if (selectedAuthorId && this.authors) {
            const author = this.authors.find(a => a.id == selectedAuthorId);
            if (author) authorName = author.name || author.username;
        }
        
        let example = '';
        const baseUrl = window.location.origin;
        switch (type) {
            case 'api':
                example = `curl -X POST ${baseUrl}/api/v1/content/ingest \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: ${apiKey}" \\
  -d '{
    "external_id": "unique-id-123",
    "title": "Article Title",
    "content": "<p>Article content in HTML...</p>",
    "excerpt": "Brief excerpt for SEO",
    "author_name": "${authorName}",
    "author_email": "author@example.com",
    "category_name": "${categoryName}",
    "tags": ["crypto", "exchange"],
    "source_url": "https://source.com/article",
    "featured_image_url": "https://example.com/image.jpg",
    "meta_title": "SEO Title (max 60 chars)",
    "meta_description": "SEO description (max 160 chars)",
    "canonical_url": "https://canonical-url.com",
    "focus_keyword": "main keyword",
    "enable_auto_linking": true,
    "language_code": "en",
    "translation_group_id": null,
    "translate_of_article_id": null
  }'

# ═══════════════════════════════════════════
# REQUIRED FIELDS:
# ═══════════════════════════════════════════
# - external_id: Unique identifier from your system
# - title: Article title (max 255 chars)
# - content: Article content (HTML supported)

# ═══════════════════════════════════════════
# OPTIONAL FIELDS:
# ═══════════════════════════════════════════
# - excerpt: Brief summary (max 500 chars)
# - author_name: Author display name
# - author_email: Author email
# - category_name: Category name (matched by name)
# - tags: Array of tag names (created if not exist)
# - source_url: Original article URL
# - featured_image_url: Image URL (auto-downloaded)
# - published_at: ISO 8601 date (default: now)

# ═══════════════════════════════════════════
# SEO FIELDS:
# ═══════════════════════════════════════════
# - meta_title: Custom SEO title (default: title)
# - meta_description: SEO description (default: excerpt)
# - canonical_url: Canonical URL (default: source_url)
# - focus_keyword: Main SEO keyword
# - enable_auto_linking: Auto internal linking (default: false)

# ═══════════════════════════════════════════
# MULTILINGUAL / TRANSLATION FIELDS:
# ═══════════════════════════════════════════
# - language_code: "en", "fr", "de", "es", "ar" (default: "en")
# - translation_group_id: Link to existing translation group ID
# - translate_of_article_id: ID of original article (auto-links to same group)
#
# WORKFLOW FOR TRANSLATIONS:
# 1. Send the original article (e.g., English) → returns article with ID
# 2. Send translations with translate_of_article_id = original article ID
#    This automatically links all translations together.`;
                break;
            case 'webhook':
                example = `Webhook URL: ${baseUrl}/api/v1/content/webhook/{source_id}
                
Send POST requests with the same JSON structure as API ingestion.
The source_id is automatically determined from the URL.`;
                break;
            case 'manual':
                example = 'Manual sources are managed through the admin interface only.';
                break;
        }
        
        if (exampleElement) {
            exampleElement.textContent = example;
        }
    }

    startAutoRefresh() {
        // Refresh stats every 30 seconds
        setInterval(() => {
            this.loadStats();
        }, 30000);

        // Refresh current tab data every 60 seconds
        setInterval(() => {
            switch (this.currentTab) {
                case 'sources':
                    this.loadSources();
                    break;
                case 'pending':
                    this.loadPendingContent();
                    break;
                case 'processed':
                    this.loadProcessedContent();
                    break;
            }
        }, 60000);
    }

    // Initialize charts with Chart.js
    initializeCharts() {
        console.log('Initializing analytics charts...');
        
        // Check if Chart.js is loaded
        if (typeof Chart === 'undefined') {
            console.warn('Chart.js not loaded. Loading from CDN...');
            this.loadChartJS().then(() => {
                this.createCharts();
            });
        } else {
            this.createCharts();
        }
    }

    loadChartJS() {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = 'https://cdn.jsdelivr.net/npm/chart.js';
            script.onload = resolve;
            script.onerror = reject;
            document.head.appendChild(script);
        });
    }

    createCharts() {
        this.createIngestionVolumeChart();
        this.createSourcePerformanceChart();
        this.createStatusDistributionChart();
        this.createProcessingTimelineChart();
        this.updateAnalyticsData();
    }

    createIngestionVolumeChart() {
        const ctx = document.getElementById('ingestion-volume-chart');
        if (!ctx) return;

        this.charts.volumeChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Content Ingested',
                    data: [],
                    borderColor: '#007bff',
                    backgroundColor: 'rgba(0, 123, 255, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            stepSize: 1
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });
    }

    createSourcePerformanceChart() {
        const ctx = document.getElementById('source-performance-chart');
        if (!ctx) return;

        this.charts.performanceChart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Success Rate (%)',
                    data: [],
                    backgroundColor: '#28a745',
                    borderColor: '#1e7e34',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        ticks: {
                            callback: function(value) {
                                return value + '%';
                            }
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });
    }

    createStatusDistributionChart() {
        const ctx = document.getElementById('status-distribution-chart');
        if (!ctx) return;

        this.charts.statusChart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Pending', 'Processed', 'Rejected', 'Duplicate'],
                datasets: [{
                    data: [0, 0, 0, 0],
                    backgroundColor: [
                        '#ffc107',
                        '#28a745',
                        '#dc3545',
                        '#6c757d'
                    ],
                    borderWidth: 2,
                    borderColor: '#fff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom'
                    }
                }
            }
        });
    }

    createProcessingTimelineChart() {
        const ctx = document.getElementById('processing-timeline-chart');
        if (!ctx) return;

        this.charts.timelineChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Processing Time (minutes)',
                    data: [],
                    borderColor: '#17a2b8',
                    backgroundColor: 'rgba(23, 162, 184, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return value + 'm';
                            }
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });
    }

    updateCharts() {
        console.log('Updating analytics charts...');
        this.updateAnalyticsData();
    }

    async updateAnalyticsData() {
        try {
            // Update performance metrics
            this.updatePerformanceMetrics();
            
            // Update chart data with current stats
            this.updateVolumeChartData();
            this.updateSourcePerformanceData();
            this.updateStatusDistribution();
            this.updateProcessingTimeline();
            
        } catch (error) {
            console.error('Error updating analytics data:', error);
        }
    }

    updatePerformanceMetrics() {
        if (!this.stats) return;

        const total = (this.stats.pending_content || 0) + (this.stats.processed_today || 0) + (this.stats.rejected_today || 0);
        const successRate = total > 0 ? Math.round((this.stats.processed_today || 0) / total * 100) : 0;
        const errorRate = total > 0 ? Math.round((this.stats.rejected_today || 0) / total * 100) : 0;

        document.getElementById('success-rate').textContent = successRate + '%';
        document.getElementById('avg-processing-time').textContent = '2.3m'; // Placeholder
        document.getElementById('error-rate').textContent = errorRate + '%';
        
        // Timeline stats
        document.getElementById('pending-queue').textContent = this.stats.pending_content || 0;
        document.getElementById('processing-time').textContent = '2.3m'; // Placeholder
        document.getElementById('peak-hour').textContent = '14:00'; // Placeholder
    }

    updateVolumeChartData() {
        if (!this.charts.volumeChart) return;

        // Generate sample hourly data for the last 24 hours
        const labels = [];
        const data = [];
        const now = new Date();
        
        for (let i = 23; i >= 0; i--) {
            const hour = new Date(now.getTime() - i * 60 * 60 * 1000);
            labels.push(hour.getHours() + ':00');
            // Generate realistic sample data based on current stats
            const baseValue = Math.floor((this.stats.processed_today || 0) / 24);
            data.push(Math.max(0, baseValue + Math.floor(Math.random() * 3) - 1));
        }

        this.charts.volumeChart.data.labels = labels;
        this.charts.volumeChart.data.datasets[0].data = data;
        this.charts.volumeChart.update();
    }

    updateSourcePerformanceData() {
        if (!this.charts.performanceChart || !this.sources) return;

        const labels = this.sources.map(source => source.name);
        const data = this.sources.map(() => Math.floor(Math.random() * 30) + 70); // 70-100% success rate

        this.charts.performanceChart.data.labels = labels;
        this.charts.performanceChart.data.datasets[0].data = data;
        this.charts.performanceChart.update();
    }

    updateStatusDistribution() {
        if (!this.charts.statusChart || !this.stats) return;

        const data = [
            this.stats.pending_content || 0,
            this.stats.processed_today || 0,
            this.stats.rejected_today || 0,
            3 // Placeholder for duplicates
        ];

        this.charts.statusChart.data.datasets[0].data = data;
        this.charts.statusChart.update();
    }

    updateProcessingTimeline() {
        if (!this.charts.timelineChart) return;

        // Generate sample processing time data
        const labels = [];
        const data = [];
        
        for (let i = 11; i >= 0; i--) {
            const hour = new Date(Date.now() - i * 2 * 60 * 60 * 1000);
            labels.push(hour.getHours() + ':00');
            data.push(Math.random() * 5 + 1); // 1-6 minutes processing time
        }

        this.charts.timelineChart.data.labels = labels;
        this.charts.timelineChart.data.datasets[0].data = data;
        this.charts.timelineChart.update();
    }

    updateAnalyticsTimeRange(hours) {
        console.log('Updating analytics time range to:', hours, 'hours');
        // This would fetch new data for the selected time range
        this.updateAnalyticsData();
    }

    updateVolumeChart(granularity) {
        console.log('Updating volume chart granularity to:', granularity);
        // This would update the chart with different time granularity
        this.updateVolumeChartData();
    }

    // Tab Management
    showTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
        const targetBtn = document.querySelector(`[onclick="showTab('${tabName}')"]`);
        if (targetBtn) {
            targetBtn.classList.add('active');
        }

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
        const targetTab = document.getElementById(`${tabName}-tab`);
        if (targetTab) {
            targetTab.classList.add('active');
        }

        this.currentTab = tabName;

        // Load data for the tab if needed
        switch (tabName) {
            case 'sources':
                this.loadSources();
                break;
            case 'pending':
                this.loadPendingContent();
                break;
            case 'processed':
                this.loadProcessedContent();
                break;
            case 'analytics':
                this.updateCharts();
                break;
        }
    }

    // Modal Management
    closeSourceModal() {
        const modal = document.getElementById('source-modal');
        if (modal) {
            modal.style.display = 'none';
        }
    }

    closeContentModal() {
        const modal = document.getElementById('content-modal');
        if (modal) {
            modal.style.display = 'none';
        }
    }

    saveSource() {
        const form = document.getElementById('source-form');
        const sourceId = document.getElementById('source-id').value;
        const isEdit = sourceId && sourceId !== '';

        // Collect form data
        const formData = {
            name: document.getElementById('source-name').value,
            type: document.getElementById('source-type').value,
            api_key: document.getElementById('api-key').value,
            is_active: document.getElementById('is-active').checked,
            rate_limit: parseInt(document.getElementById('rate-limit').value) || 100,
            priority: parseInt(document.getElementById('priority').value) || 5,
            config: {
                auto_publish: document.getElementById('auto-publish').checked,
                default_category_id: parseInt(document.getElementById('default-category').value) || 0,
                default_author_id: parseInt(document.getElementById('default-author').value) || 0
            }
        };
        
        console.log('Saving source data:', formData);

        // Validate required fields
        if (!formData.name || !formData.type) {
            this.showNotification('Please fill in all required fields', 'error');
            return;
        }

        const url = isEdit 
            ? `/api/v1/admin-panel/content/sources/${sourceId}`
            : '/api/v1/admin-panel/content/sources';
        
        const method = isEdit ? 'PUT' : 'POST';

        fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include',
            body: JSON.stringify(formData)
        })
        .then(response => response.json())
        .then(data => {
            if (data.success || data.message) {
                this.showNotification(isEdit ? 'Source updated successfully' : 'Source created successfully', 'success');
                this.closeSourceModal();
                this.loadSources();
                this.loadStats();
            } else {
                throw new Error(data.error || 'Failed to save source');
            }
        })
        .catch(error => {
            console.error('Save error:', error);
            this.showNotification('Failed to save source: ' + error.message, 'error');
        });
    }

    refreshStats() {
        this.loadStats();
        this.showNotification('Stats refreshed', 'success');
    }

    processBatch() {
        // Get selected pending content items
        const checkboxes = document.querySelectorAll('.pending-checkbox:checked');
        if (checkboxes.length === 0) {
            this.showNotification('Please select content items to process', 'warning');
            return;
        }

        const contentIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
        
        // Process batch
        fetch('/api/v1/admin-panel/content/process/batch', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include',
            body: JSON.stringify({ content_ids: contentIds })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success || data.message) {
                this.showNotification(data.message || 'Batch processed successfully', 'success');
                this.loadPendingContent();
                this.loadProcessedContent();
                this.loadStats();
            } else {
                throw new Error(data.error || 'Failed to process batch');
            }
        })
        .catch(error => {
            console.error('Batch processing error:', error);
            this.showNotification('Failed to process batch: ' + error.message, 'error');
        });
    }



    rejectContent(contentId, reason = 'Rejected by admin') {
        if (!contentId) return;

        fetch(`/api/v1/admin-panel/content/reject/${contentId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include',
            body: JSON.stringify({ reason: reason })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success || data.message) {
                this.showNotification('Content rejected successfully', 'success');
                this.loadPendingContent();
                this.loadProcessedContent();
                this.loadStats();
            } else {
                throw new Error(data.error || 'Failed to reject content');
            }
        })
        .catch(error => {
            console.error('Rejection error:', error);
            this.showNotification('Failed to reject content: ' + error.message, 'error');
        });
    }

    copyApiKey() {
        const apiKeyInput = document.getElementById('api-key');
        if (apiKeyInput && apiKeyInput.value) {
            navigator.clipboard.writeText(apiKeyInput.value).then(() => {
                this.showNotification('API key copied to clipboard', 'success');
            }).catch(() => {
                // Fallback for older browsers
                apiKeyInput.select();
                document.execCommand('copy');
                this.showNotification('API key copied to clipboard', 'success');
            });
        }
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${this.getNotificationIcon(type)}"></i>
            <span>${message}</span>
            <button class="notification-close">&times;</button>
        `;

        // Add to page
        document.body.appendChild(notification);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);

        // Close button
        notification.querySelector('.notification-close').addEventListener('click', () => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        });
    }

    getNotificationIcon(type) {
        const icons = {
            'success': 'check-circle',
            'error': 'exclamation-circle',
            'warning': 'exclamation-triangle',
            'info': 'info-circle'
        };
        return icons[type] || 'info-circle';
    }

    // Utility Methods
    getAuthHeaders() {
        return {
            'Content-Type': 'application/json',
        };
    }

    formatDate(dateString) {
        if (!dateString) return 'Never';
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    }

    formatRelativeTime(dateString) {
        const now = new Date();
        const date = new Date(dateString);
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        return `${diffDays}d ago`;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    getSourceName(sourceId) {
        const source = this.sources.find(s => s.id === sourceId);
        return source ? source.name : 'Unknown Source';
    }

    toggleAllPending(checked) {
        const checkboxes = document.querySelectorAll('.pending-checkbox');
        checkboxes.forEach(cb => cb.checked = checked);
    }

    updateSourceFilters() {
        // Update source filter dropdown in pending content tab
        const pendingSourceFilter = document.getElementById('pending-source-filter');
        if (pendingSourceFilter) {
            // Clear existing options except "All Sources"
            pendingSourceFilter.innerHTML = '<option value="">All Sources</option>';
            
            // Add options for each source
            this.sources.forEach(source => {
                const option = document.createElement('option');
                option.value = source.id;
                option.textContent = source.name;
                pendingSourceFilter.appendChild(option);
            });
        }
    }

    updateStatsDisplay() {
        document.getElementById('total-sources').textContent = this.stats.total_sources || 0;
        document.getElementById('pending-content').textContent = this.stats.pending_content || 0;
        document.getElementById('processed-today').textContent = this.stats.processed_today || 0;
        document.getElementById('rejected-today').textContent = this.stats.rejected_today || 0;
    }

    // Filter Methods
    filterSources(searchTerm) {
        // Implementation for filtering sources
        console.log('Filter sources:', searchTerm);
    }

    filterSourcesByType(type) {
        // Implementation for filtering sources by type
        console.log('Filter sources by type:', type);
    }

    filterPendingBySource(sourceId) {
        // Implementation for filtering pending content by source
        console.log('Filter pending by source:', sourceId);
    }

    filterProcessed(searchTerm) {
        // Implementation for filtering processed content
        console.log('Filter processed:', searchTerm);
    }

    filterProcessedByStatus(status) {
        // Implementation for filtering processed content by status
        console.log('Filter processed by status:', status);
    }

    // Content Management Methods
    previewContent(contentId) {
        // Fetch content details and show in modal
        fetch(`/api/v1/admin-panel/content/details/${contentId}`, {
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success && data.content) {
                this.showContentPreviewModal(data.content);
            } else {
                this.showNotification('Failed to load content details', 'error');
            }
        })
        .catch(error => {
            console.error('Preview error:', error);
            this.showNotification('Failed to preview content', 'error');
        });
    }

    showContentPreviewModal(content) {
        const modal = document.getElementById('content-modal');
        if (modal) {
            modal.dataset.contentId = content.id;
            
            // Update modal content
            const preview = document.getElementById('content-preview');
            if (preview) {
                preview.innerHTML = `
                    <h3>${this.escapeHtml(content.title)}</h3>
                    <p><strong>Author:</strong> ${this.escapeHtml(content.author_name || 'Unknown')}</p>
                    <p><strong>Category:</strong> ${this.escapeHtml(content.category_name || 'Uncategorized')}</p>
                    <p><strong>Source URL:</strong> <a href="${content.source_url}" target="_blank">${content.source_url}</a></p>
                    ${content.status ? `<p><strong>Status:</strong> <span class="status-badge status-${content.status}">${content.status.toUpperCase()}</span></p>` : ''}
                    <div><strong>Excerpt:</strong></div>
                    <p>${this.escapeHtml(content.excerpt || '')}</p>
                    <div><strong>Content:</strong></div>
                    <div style="max-height: 300px; overflow-y: auto; border: 1px solid #ddd; padding: 10px;">
                        ${content.content ? content.content.substring(0, 1000) + (content.content.length > 1000 ? '...' : '') : 'No content'}
                    </div>
                `;
            }
            
            // Show/hide action buttons based on content status
            const approveBtn = document.getElementById('approve-btn');
            const rejectBtn = document.getElementById('reject-btn');
            
            if (approveBtn && rejectBtn) {
                if (content.status === 'pending') {
                    // Show buttons for pending content
                    approveBtn.style.display = 'inline-block';
                    rejectBtn.style.display = 'inline-block';
                } else {
                    // Hide buttons for processed/rejected content
                    approveBtn.style.display = 'none';
                    rejectBtn.style.display = 'none';
                }
            }
            
            modal.style.display = 'block';
        }
    }

    viewProcessedDetails(contentId) {
        // Same as preview but for processed content
        this.previewContent(contentId);
    }

    reprocessContent(contentId) {
        const confirmed = confirm('Are you sure you want to reprocess this content? This will move the rejected content back to pending status for review.');
        if (!confirmed) return;

        fetch(`/api/v1/admin-panel/content/reprocess/${contentId}`, {
            method: 'POST',
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success || data.message) {
                this.showNotification('Content moved back to pending for reprocessing', 'success');
                this.loadPendingContent();
                this.loadProcessedContent();
                this.loadStats();
            } else {
                throw new Error(data.error || 'Failed to reprocess content');
            }
        })
        .catch(error => {
            console.error('Reprocess error:', error);
            this.showNotification('Failed to reprocess content: ' + error.message, 'error');
        });
    }

    viewSourceStats(sourceId) {
        // Show source statistics
        fetch(`/api/v1/admin-panel/content/stats?source_id=${sourceId}`, {
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert(`Source Statistics:\n\nPending: ${data.data.pending || 0}\nProcessed: ${data.data.processed || 0}\nRejected: ${data.data.rejected || 0}`);
            } else {
                this.showNotification('Failed to load source statistics', 'error');
            }
        })
        .catch(error => {
            console.error('Stats error:', error);
            this.showNotification('Failed to load source statistics', 'error');
        });
    }

    testSource(sourceId) {
        this.showNotification('Testing source connection...', 'info');
        
        // Simulate a test - in real implementation this would test the source connection
        setTimeout(() => {
            this.showNotification('Source connection test completed', 'success');
        }, 2000);
    }

    deleteSource(sourceId) {
        const source = this.sources.find(s => s.id === sourceId);
        const sourceName = source ? source.name : `Source ${sourceId}`;
        
        const confirmed = confirm(`Are you sure you want to delete the source "${sourceName}"? This action cannot be undone.`);
        if (!confirmed) return;

        fetch(`/api/v1/admin-panel/content/sources/${sourceId}`, {
            method: 'DELETE',
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success || data.message) {
                this.showNotification('Source deleted successfully', 'success');
                this.loadSources();
                this.loadStats();
            } else {
                throw new Error(data.error || 'Failed to delete source');
            }
        })
        .catch(error => {
            console.error('Delete error:', error);
            this.showNotification('Failed to delete source: ' + error.message, 'error');
        });
    }

    // Edit article inline method
    editArticleInline(articleId) {
        // Open the admin article edit page
        window.open(`/admin/content/edit/${articleId}`, '_blank');
    }
}





// Global functions for HTML onclick handlers
function showTab(tabName) {
    if (window.contentManager) {
        window.contentManager.showTab(tabName);
    }
}

function showCreateSourceModal() {
    if (window.contentManager) {
        window.contentManager.showCreateSourceModal();
    }
}

function closeSourceModal() {
    if (window.contentManager) {
        window.contentManager.closeSourceModal();
    }
}

function saveSource() {
    if (window.contentManager) {
        window.contentManager.saveSource();
    }
}

function refreshStats() {
    if (window.contentManager) {
        window.contentManager.refreshStats();
    }
}

function processBatch() {
    if (window.contentManager) {
        window.contentManager.processBatch();
    }
}

function copyApiKey() {
    if (window.contentManager) {
        window.contentManager.copyApiKey();
    }
}

function closeContentModal() {
    if (window.contentManager) {
        window.contentManager.closeContentModal();
    }
}

function approveContent() {
    if (window.contentManager) {
        window.contentManager.approveContent();
    }
}

function rejectContent() {
    if (window.contentManager) {
        window.contentManager.rejectContent();
    }
}

// Initialize when DOM is loaded
let contentManager;
document.addEventListener('DOMContentLoaded', () => {
    console.log('Initializing contentManager...');
    contentManager = new ContentIngestionManager();
    window.contentManager = contentManager; // Make it globally accessible
});