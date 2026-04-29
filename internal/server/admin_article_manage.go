package server

import (
	"github.com/gin-gonic/gin"
)

// renderManageArticles renders the articles management page
func (s *Server) renderManageArticles(c *gin.Context) {
	title := "Manage Articles"
	content := `
        <style>
            .articles-table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
            .articles-table th, .articles-table td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #e5e7eb; }
            .articles-table th { background-color: #f9fafb; font-weight: 600; }
            .articles-table tr:hover { background-color: #f9fafb; }
            .status-badge { padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 500; }
            .status-draft { background-color: #fef3c7; color: #92400e; }
            .status-published { background-color: #d1fae5; color: #065f46; }
            .status-scheduled { background-color: #dbeafe; color: #1e40af; }
            .status-archived { background-color: #f3f4f6; color: #374151; }
            .status-deleted { background-color: #fee2e2; color: #991b1b; }
            .action-btn { padding: 0.25rem 0.5rem; margin: 0 0.125rem; border: none; border-radius: 4px; cursor: pointer; font-size: 0.75rem; }
            .btn-edit { background-color: #3b82f6; color: white; }
            .btn-delete { background-color: #ef4444; color: white; }
            .btn-view { background-color: #10b981; color: white; }
            .filters { display: flex; gap: 1rem; margin-bottom: 1rem; align-items: center; flex-wrap: wrap; }
            .filter-group { display: flex; flex-direction: column; gap: 0.25rem; }
            .filter-group label { font-size: 0.875rem; font-weight: 500; }
            .filter-input { padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 4px; }
            .bulk-actions { display: flex; gap: 0.5rem; margin-bottom: 1rem; align-items: center; }
            .pagination { display: flex; justify-content: center; align-items: center; gap: 0.5rem; margin-top: 1rem; }
            .pagination button { padding: 0.5rem 0.75rem; border: 1px solid #d1d5db; background: white; cursor: pointer; border-radius: 4px; }
            .pagination button:hover { background-color: #f3f4f6; }
            .pagination button.active { background-color: #3b82f6; color: white; border-color: #3b82f6; }
            .loading { text-align: center; padding: 2rem; color: #6b7280; }
            .no-articles { text-align: center; padding: 2rem; color: #6b7280; }
        </style>

        <div class="dashboard-card">
            <div class="card-title">📄 Articles Management</div>
            
            <!-- Filters -->
            <div class="filters">
                <div class="filter-group">
                    <label>Search</label>
                    <input type="text" id="searchInput" class="filter-input" placeholder="Search articles..." onkeyup="filterArticles()">
                </div>
                <div class="filter-group">
                    <label>Status</label>
                    <select id="statusFilter" class="filter-input" onchange="filterArticles()">
                        <option value="">All Statuses</option>
                        <option value="draft">Draft</option>
                        <option value="published">Published</option>
                        <option value="scheduled">Scheduled</option>
                        <option value="archived">Archived</option>
                        <option value="deleted">Deleted (Trash)</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>Category</label>
                    <select id="categoryFilter" class="filter-input" onchange="filterArticles()">
                        <option value="">All Categories</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>Author</label>
                    <select id="authorFilter" class="filter-input" onchange="filterArticles()">
                        <option value="">All Authors</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>&nbsp;</label>
                    <button onclick="resetFilters()" class="filter-input" style="background-color: #6b7280; color: white; border: none; cursor: pointer;">Reset</button>
                </div>
            </div>

            <!-- Bulk Actions -->
            <div class="bulk-actions">
                <input type="checkbox" id="selectAll" onchange="toggleSelectAll()">
                <label for="selectAll">Select All</label>
                <button onclick="bulkAction('publish')" class="action-button" style="background-color: #10b981;">📤 Publish Selected</button>
                <button onclick="bulkAction('draft')" class="action-button" style="background-color: #6b7280;">📝 Draft Selected</button>
                <button onclick="bulkAction('archive')" class="action-button" style="background-color: #f59e0b;">📦 Archive Selected</button>
                <button onclick="bulkAction('delete')" class="action-button" style="background-color: #ef4444;">🗑️ Move to Trash</button>
                <a href="/admin/content/trash" class="action-button" style="background-color: #8b5cf6;">🗑️ Recycle Bin</a>
            </div>

            <!-- Articles Table -->
            <div id="articlesContainer">
                <div class="loading">Loading articles...</div>
            </div>

            <!-- Pagination -->
            <div id="paginationContainer" class="pagination" style="display: none;"></div>
        </div>

        <script>
            let articles = [];
            let filteredArticles = [];
            let categories = [];
            let authors = [];
            let currentPage = 1;
            let articlesPerPage = 20;

            document.addEventListener('DOMContentLoaded', function() {
                loadArticles();
                loadCategories();
                loadAuthors();
            });

            async function loadArticles() {
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/articles?limit=1000', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });

                    if (response.ok) {
                        const data = await response.json();
                        articles = data.articles || [];
                        filteredArticles = [...articles];
                        renderArticles();
                        renderPagination();
                    } else {
                        document.getElementById('articlesContainer').innerHTML = 
                            '<div class="no-articles">Failed to load articles</div>';
                    }
                } catch (error) {
                    document.getElementById('articlesContainer').innerHTML = 
                        '<div class="no-articles">Error loading articles: ' + error.message + '</div>';
                }
            }

            async function loadCategories() {
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin/content/categories', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });

                    if (response.ok) {
                        const data = await response.json();
                        categories = data.data.categories || [];
                        
                        const categoryFilter = document.getElementById('categoryFilter');
                        categories.forEach(category => {
                            const option = document.createElement('option');
                            option.value = category.id;
                            option.textContent = category.name;
                            categoryFilter.appendChild(option);
                        });
                    }
                } catch (error) {
                    console.log('Failed to load categories:', error);
                }
            }

            async function loadAuthors() {
                // Extract unique authors from articles
                const uniqueAuthors = [...new Set(articles.map(article => article.author_id))];
                const authorFilter = document.getElementById('authorFilter');
                
                uniqueAuthors.forEach(authorId => {
                    const option = document.createElement('option');
                    option.value = authorId;
                    option.textContent = 'Author ' + authorId; // In real app, you'd fetch author names
                    authorFilter.appendChild(option);
                });
            }

            function renderArticles() {
                const container = document.getElementById('articlesContainer');
                
                if (filteredArticles.length === 0) {
                    container.innerHTML = '<div class="no-articles">No articles found</div>';
                    return;
                }

                const startIndex = (currentPage - 1) * articlesPerPage;
                const endIndex = startIndex + articlesPerPage;
                const pageArticles = filteredArticles.slice(startIndex, endIndex);

                let html = '<table class="articles-table">';
                html += '<thead><tr>';
                html += '<th><input type="checkbox" id="selectAllPage" onchange="toggleSelectAllPage()"></th>';
                html += '<th>Title</th>';
                html += '<th>Status</th>';
                html += '<th>Category</th>';
                html += '<th>Author</th>';
                html += '<th>Created</th>';
                html += '<th>Views</th>';
                html += '<th>Actions</th>';
                html += '</tr></thead><tbody>';

                pageArticles.forEach(article => {
                    const statusClass = 'status-' + article.status;
                    const createdDate = new Date(article.created_at).toLocaleDateString();
                    // Display all categories for the article
                    let categoryNames = [];
                    if (article.categories && article.categories.length > 0) {
                        categoryNames = article.categories.map(cat => cat.name);
                    } else {
                        // Fallback to primary category for backward compatibility
                        const primaryCategory = categories.find(c => c.id === article.category_id);
                        if (primaryCategory) {
                            categoryNames = [primaryCategory.name];
                        }
                    }
                    const categoryDisplay = categoryNames.length > 0 ? categoryNames.join(', ') : 'Unknown';
                    
                    html += '<tr>';
                    html += '<td><input type="checkbox" class="article-checkbox" value="' + article.id + '"></td>';
                    html += '<td><strong>' + escapeHtml(article.title) + '</strong><br><small>' + escapeHtml(article.slug) + '</small></td>';
                    html += '<td><span class="status-badge ' + statusClass + '">' + article.status.toUpperCase() + '</span></td>';
                    html += '<td>' + escapeHtml(categoryDisplay) + '</td>';
                    html += '<td>Author ' + article.author_id + '</td>';
                    html += '<td>' + createdDate + '</td>';
                    html += '<td>' + (article.view_count || 0) + '</td>';
                    html += '<td>';
                    html += '<button onclick="viewArticle(\'' + article.slug + '\')" class="action-btn btn-view" title="View Article">👁️</button>';
                    html += '<button onclick="editArticle(' + article.id + ')" class="action-btn btn-edit" title="Edit Article">✏️</button>';
                    
                    // Show different buttons based on article status
                    if (article.status === 'archived') {
                        html += '<button onclick="unarchiveArticle(' + article.id + ')" class="action-btn" style="background-color: #10b981; color: white;" title="Un-archive Article">📤</button>';
                    } else {
                        html += '<button onclick="archiveArticle(' + article.id + ')" class="action-btn" style="background-color: #f59e0b; color: white;" title="Archive Article">📦</button>';
                    }
                    
                    html += '<button onclick="' + (article.status === 'deleted' ? 'restoreArticle' : 'deleteArticle') + '(' + article.id + ')" class="action-btn ' + (article.status === 'deleted' ? '' : 'btn-delete') + '" style="' + (article.status === 'deleted' ? 'background-color: #10b981; color: white;' : '') + '" title="' + (article.status === 'deleted' ? 'Restore Article' : 'Move to Trash') + '">' + (article.status === 'deleted' ? '↩️' : '🗑️') + '</button>';
                    html += '</td>';
                    html += '</tr>';
                });

                html += '</tbody></table>';
                container.innerHTML = html;
            }

            function renderPagination() {
                const container = document.getElementById('paginationContainer');
                const totalPages = Math.ceil(filteredArticles.length / articlesPerPage);
                
                if (totalPages <= 1) {
                    container.style.display = 'none';
                    return;
                }

                container.style.display = 'flex';
                let html = '';

                // Previous button
                html += '<button onclick="changePage(' + (currentPage - 1) + ')" ' + 
                        (currentPage === 1 ? 'disabled' : '') + '>← Previous</button>';

                // Page numbers
                for (let i = 1; i <= totalPages; i++) {
                    if (i === currentPage) {
                        html += '<button class="active">' + i + '</button>';
                    } else {
                        html += '<button onclick="changePage(' + i + ')">' + i + '</button>';
                    }
                }

                // Next button
                html += '<button onclick="changePage(' + (currentPage + 1) + ')" ' + 
                        (currentPage === totalPages ? 'disabled' : '') + '>Next →</button>';

                container.innerHTML = html;
            }

            function changePage(page) {
                const totalPages = Math.ceil(filteredArticles.length / articlesPerPage);
                if (page < 1 || page > totalPages) return;
                
                currentPage = page;
                renderArticles();
                renderPagination();
            }

            function filterArticles() {
                const searchTerm = document.getElementById('searchInput').value.toLowerCase();
                const statusFilter = document.getElementById('statusFilter').value;
                const categoryFilter = document.getElementById('categoryFilter').value;
                const authorFilter = document.getElementById('authorFilter').value;

                filteredArticles = articles.filter(article => {
                    const matchesSearch = !searchTerm || 
                        article.title.toLowerCase().includes(searchTerm) ||
                        article.slug.toLowerCase().includes(searchTerm);
                    
                    const matchesStatus = !statusFilter || article.status === statusFilter;
                    const matchesCategory = !categoryFilter || article.category_id.toString() === categoryFilter;
                    const matchesAuthor = !authorFilter || article.author_id.toString() === authorFilter;

                    return matchesSearch && matchesStatus && matchesCategory && matchesAuthor;
                });

                currentPage = 1;
                renderArticles();
                renderPagination();
            }

            function resetFilters() {
                document.getElementById('searchInput').value = '';
                document.getElementById('statusFilter').value = '';
                document.getElementById('categoryFilter').value = '';
                document.getElementById('authorFilter').value = '';
                filterArticles();
            }

            function toggleSelectAll() {
                const selectAll = document.getElementById('selectAll');
                const checkboxes = document.querySelectorAll('.article-checkbox');
                checkboxes.forEach(checkbox => {
                    checkbox.checked = selectAll.checked;
                });
            }

            function toggleSelectAllPage() {
                const selectAllPage = document.getElementById('selectAllPage');
                const checkboxes = document.querySelectorAll('.article-checkbox');
                checkboxes.forEach(checkbox => {
                    checkbox.checked = selectAllPage.checked;
                });
            }

            function getSelectedArticles() {
                const checkboxes = document.querySelectorAll('.article-checkbox:checked');
                return Array.from(checkboxes).map(cb => parseInt(cb.value));
            }

            async function bulkAction(action) {
                const selectedIds = getSelectedArticles();
                if (selectedIds.length === 0) {
                    alert('Please select articles first');
                    return;
                }

                const actionNames = {
                    'publish': 'publish',
                    'draft': 'set to draft',
                    'archive': 'archive (hide from public but keep accessible)',
                    'delete': 'move to trash (can be restored later)'
                };

                if (!confirm('Are you sure you want to ' + actionNames[action] + ' ' + selectedIds.length + ' articles?')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    
                    for (const id of selectedIds) {
                        let endpoint = '/api/v1/articles/' + id;
                        let method = 'PATCH';
                        let body = {};

                        if (action === 'delete') {
                            // Move to trash (soft delete) - change status to 'deleted'
                            body = { status: 'deleted' };
                        } else {
                            body = { status: action === 'publish' ? 'published' : action };
                        }

                        const response = await fetch(endpoint, {
                            method: 'PATCH', // Always use PATCH for status changes
                            headers: {
                                'Content-Type': 'application/json',
                                'Authorization': 'Bearer ' + token
                            },
                            body: JSON.stringify(body)
                        });

                        if (!response.ok) {
                            console.error('Failed to ' + action + ' article ' + id);
                        }
                    }

                    alert('Bulk action completed successfully');
                    loadArticles(); // Reload articles
                } catch (error) {
                    alert('Error performing bulk action: ' + error.message);
                }
            }

            function viewArticle(slug) {
                window.open('/article/' + slug, '_blank');
            }

            function editArticle(id) {
                window.open('/admin/content/edit/' + id, '_blank');
            }

            async function archiveArticle(id) {
                if (!confirm('Are you sure you want to archive this article? It will be hidden from public but remain accessible in admin.')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    console.log('Archiving article:', id);
                    
                    const response = await fetch('/api/v1/articles/' + id, {
                        method: 'PATCH',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify({ status: 'archived' })
                    });

                    console.log('Archive response status:', response.status);

                    if (response.ok) {
                        alert('Article archived successfully');
                        loadArticles();
                    } else {
                        const error = await response.json();
                        console.log('Archive error:', error);
                        alert('Failed to archive article: ' + (error.message || error.error || 'Unknown error'));
                    }
                } catch (error) {
                    console.log('Archive network error:', error);
                    alert('Error archiving article: ' + error.message);
                }
            }

            async function unarchiveArticle(id) {
                if (!confirm('Are you sure you want to un-archive this article? It will be moved back to draft status.')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    console.log('Un-archiving article:', id);
                    
                    const response = await fetch('/api/v1/articles/' + id, {
                        method: 'PATCH',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify({ status: 'draft' })
                    });

                    console.log('Un-archive response status:', response.status);

                    if (response.ok) {
                        alert('Article un-archived successfully');
                        loadArticles();
                    } else {
                        const error = await response.json();
                        console.log('Un-archive error:', error);
                        alert('Failed to un-archive article: ' + (error.message || error.error || 'Unknown error'));
                    }
                } catch (error) {
                    console.log('Un-archive network error:', error);
                    alert('Error un-archiving article: ' + error.message);
                }
            }

            async function deleteArticle(id) {
                if (!confirm('Are you sure you want to move this article to trash? It can be restored later from the Recycle Bin.')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    console.log('Moving article to trash:', id);
                    
                    const response = await fetch('/api/v1/articles/' + id, {
                        method: 'PATCH',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify({ status: 'deleted' })
                    });

                    console.log('Delete response status:', response.status);

                    if (response.ok) {
                        alert('Article moved to trash successfully');
                        loadArticles();
                    } else {
                        const error = await response.json();
                        console.log('Delete error:', error);
                        alert('Failed to move article to trash: ' + (error.message || error.error || 'Unknown error'));
                    }
                } catch (error) {
                    console.log('Delete network error:', error);
                    alert('Error moving article to trash: ' + error.message);
                }
            }

            async function restoreArticle(id) {
                if (!confirm('Are you sure you want to restore this article? It will be moved back to draft status.')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    console.log('Restoring article:', id);
                    
                    const response = await fetch('/api/v1/articles/' + id, {
                        method: 'PATCH',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify({ status: 'draft' })
                    });

                    console.log('Restore response status:', response.status);

                    if (response.ok) {
                        alert('Article restored successfully');
                        loadArticles();
                    } else {
                        const error = await response.json();
                        console.log('Restore error:', error);
                        alert('Failed to restore article: ' + (error.message || error.error || 'Unknown error'));
                    }
                } catch (error) {
                    console.log('Restore network error:', error);
                    alert('Error restoring article: ' + error.message);
                }
            }

            function escapeHtml(text) {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            }
        </script>
    `
	s.renderAdminPage(c, title, "", content)
}

// renderRecycleBin renders the deleted articles (recycle bin) page
