// Comment Management JavaScript
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
    
    // Set default filter to pending on first load
    document.getElementById('statusFilter').value = 'pending';
    
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
                'Authorization': `Bearer ${token}`
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
        
        let url;
        
        // Determine which endpoint to use based on filters
        if (search || status) {
            // Use search endpoint when there's a search query or status filter
            url = `/api/v1/admin/comments/search?q=${encodeURIComponent(search || '')}&status=${status}&limit=${pageSize}&offset=${offset}`;
        } else {
            // Show all recent comments when no filters are applied (All Status)
            url = `/api/v1/admin/comments/recent?limit=${pageSize}&offset=${offset}`;
        }
        
        const token = localStorage.getItem('auth_token');
        const response = await fetch(url, {
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });
        const data = await response.json();
        
        if (data.comments && data.comments.length > 0) {
            displayComments(data.comments);
        } else {
            container.innerHTML = `
                <div class="empty-state">
                    <h3>No comments found</h3>
                    <p>No comments match your current filters.</p>
                </div>
            `;
        }
        
        updatePagination(data.total || 0);
        
    } catch (error) {
        console.error('Error loading comments:', error);
        container.innerHTML = `
            <div class="empty-state">
                <h3>Error loading comments</h3>
                <p>Please try refreshing the page.</p>
            </div>
        `;
    }
}

function displayComments(comments) {
    const container = document.getElementById('commentsContainer');
    
    // Debug: Log spam scores being displayed
    comments.forEach(comment => {
        console.log(`Comment ${comment.id}: spam_score=${comment.spam_score}, content="${comment.content.substring(0, 50)}..."`);
    });
    
    const html = comments.map(comment => `
        <div class="comment-item">
            <input type="checkbox" class="comment-checkbox" value="${comment.id}">
            <div class="comment-content">
                <div class="comment-meta">
                    <strong>${escapeHtml(comment.author_name)}</strong>
                    <span>${escapeHtml(comment.author_email)}</span>
                    <span>${formatDate(comment.created_at)}</span>
                    <span class="spam-score ${getSpamScoreClass(comment.spam_score || 0)}">
                        Spam: ${((comment.spam_score || 0) * 100).toFixed(1)}%
                    </span>
                </div>
                <div class="comment-text">${escapeHtml(comment.content)}</div>
                <a href="/article/${comment.article_slug || comment.article_id}" class="comment-article" target="_blank">
                    ${comment.article_title || 'View Article'} →
                </a>
            </div>
            <div class="comment-status-column">
                <div class="status-badge status-${comment.status}">${comment.status}</div>
            </div>
            <div class="comment-actions">
                ${comment.status === 'pending' ? `
                    <button onclick="moderateComment(${comment.id}, 'approve')" class="action-btn btn-success">Approve</button>
                    <button onclick="moderateComment(${comment.id}, 'reject')" class="action-btn btn-danger">Reject</button>
                    <button onclick="moderateComment(${comment.id}, 'spam')" class="action-btn btn-warning">Spam</button>
                ` : `
                    <button onclick="moderateComment(${comment.id}, 'pending')" class="action-btn btn-primary">Review</button>
                `}
            </div>
            <div class="comment-extra-actions">
                <button onclick="editComment(${comment.id})" class="action-btn" style="background-color: #6b7280;">Edit</button>
                <button onclick="deleteComment(${comment.id})" class="action-btn btn-danger">Delete</button>
                <button onclick="viewReplies(${comment.id})" class="action-btn" style="background-color: #8b5cf6;">Replies</button>
            </div>
        </div>
    `).join('');
    
    container.innerHTML = html;
}

async function moderateComment(commentId, action) {
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch(`/api/v1/admin/comments/${commentId}/moderate`, {
            method: 'PUT',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ action: action })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showNotification(`Comment ${action}ed successfully`, 'success');
            loadComments();
            loadStats();
        } else {
            showNotification(data.message || `Failed to ${action} comment`, 'error');
        }
    } catch (error) {
        console.error('Error moderating comment:', error);
        showNotification(`Failed to ${action} comment`, 'error');
    }
}

async function editComment(commentId) {
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch(`/api/v1/comments/${commentId}`, {
            headers: {
                'Authorization': `Bearer ${token}`
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
        const response = await fetch(`/api/v1/admin/comments/${commentId}/edit`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
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
        const response = await fetch(`/api/v1/admin/comments/${commentId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`
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
        const response = await fetch(`/api/v1/admin/comments/${commentId}/replies`, {
            headers: {
                'Authorization': `Bearer ${token}`
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
    modal.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0,0,0,0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    `;
    
    const content = document.createElement('div');
    content.style.cssText = `
        background: white;
        padding: 2rem;
        border-radius: 8px;
        max-width: 800px;
        max-height: 80vh;
        overflow-y: auto;
        width: 90%;
    `;
    
    content.innerHTML = `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
            <h3>Comment Replies</h3>
            <button onclick="this.closest('.modal').remove()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">&times;</button>
        </div>
        <div>
            ${replies.map(reply => `
                <div style="border: 1px solid #e5e7eb; padding: 1rem; margin-bottom: 1rem; border-radius: 4px;">
                    <div style="font-weight: bold; margin-bottom: 0.5rem;">${escapeHtml(reply.author_name)}</div>
                    <div style="color: #6b7280; font-size: 0.875rem; margin-bottom: 0.5rem;">${formatDate(reply.created_at)}</div>
                    <div>${escapeHtml(reply.content)}</div>
                    <div style="margin-top: 0.5rem;">
                        <span class="status-badge status-${reply.status}">${reply.status}</span>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
    
    modal.appendChild(content);
    modal.className = 'modal';
    document.body.appendChild(modal);
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
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ 
                comment_ids: commentIds, 
                action: action 
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showNotification(`${commentIds.length} comments ${action}ed successfully`, 'success');
            loadComments();
            loadStats();
            document.getElementById('selectAll').checked = false;
        } else {
            showNotification(data.message || `Failed to ${action} comments`, 'error');
        }
    } catch (error) {
        console.error('Error bulk moderating comments:', error);
        showNotification(`Failed to ${action} comments`, 'error');
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
    document.getElementById('statusFilter').value = 'pending';
    document.getElementById('searchFilter').value = '';
    currentPage = 1;
    loadComments();
}

function forceRefresh() {
    // Force refresh by clearing any potential cache and reloading everything
    currentPage = 1;
    loadStats();
    loadComments();
    showNotification('Data refreshed!', 'success');
}

function updatePagination(total) {
    totalPages = Math.ceil(total / pageSize);
    const pagination = document.getElementById('pagination');
    
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }
    
    let html = '';
    
    // Previous button
    if (currentPage > 1) {
        html += `<button onclick="changePage(${currentPage - 1})">← Previous</button>`;
    }
    
    // Page numbers
    for (let i = Math.max(1, currentPage - 2); i <= Math.min(totalPages, currentPage + 2); i++) {
        html += `<button onclick="changePage(${i})" class="${i === currentPage ? 'active' : ''}">${i}</button>`;
    }
    
    // Next button
    if (currentPage < totalPages) {
        html += `<button onclick="changePage(${currentPage + 1})">Next →</button>`;
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

function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 1rem 1.5rem;
        border-radius: 8px;
        color: white;
        font-weight: 500;
        z-index: 1000;
        animation: slideIn 0.3s ease-out;
        background-color: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : type === 'warning' ? '#f59e0b' : '#3b82f6'};
    `;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    // Remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => {
            if (notification.parentNode) {
                document.body.removeChild(notification);
            }
        }, 300);
    }, 3000);
}

// Spam settings functions
function showSpamSettings() {
    const modal = document.getElementById('spamSettingsModal');
    if (modal) {
        modal.style.display = 'flex';
        loadSpamSettings();
    }
}

function hideSpamSettings() {
    document.getElementById('spamSettingsModal').style.display = 'none';
}

async function loadSpamSettings() {
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/admin/comments/spam-settings', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (response.ok) {
            const data = await response.json();
            document.getElementById('spamKeywords').value = data.keywords.join('\n');
            document.getElementById('spamThreshold').value = data.threshold;
        } else {
            // Use defaults if settings don't exist yet
            document.getElementById('spamKeywords').value = `viagra
casino
lottery
winner
congratulations
click here
free money
buy now
limited time
act now`;
            document.getElementById('spamThreshold').value = '0.5';
        }
    } catch (error) {
        console.error('Error loading spam settings:', error);
        // Use defaults on error
        document.getElementById('spamKeywords').value = `viagra
casino
lottery
winner
congratulations
click here
free money
buy now
limited time
act now`;
        document.getElementById('spamThreshold').value = '0.5';
    }
}

async function saveSpamSettings() {
    const keywords = document.getElementById('spamKeywords').value;
    const threshold = parseFloat(document.getElementById('spamThreshold').value);
    
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/admin/comments/spam-settings', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({
                keywords: keywords.split('\n').filter(k => k.trim()),
                threshold: threshold
            })
        });
        
        if (response.ok) {
            showNotification('Spam settings saved successfully!', 'success');
            hideSpamSettings();
        } else {
            showNotification('Failed to save spam settings', 'error');
        }
    } catch (error) {
        console.error('Error saving spam settings:', error);
        showNotification('Failed to save spam settings', 'error');
    }
}

async function recalculateSpamScores() {
    if (!confirm('This will recalculate spam scores for all comments using current settings. This may take a while. Continue?')) {
        return;
    }
    
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/admin/comments/recalculate-spam', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showNotification(`Spam scores recalculated for ${data.updated_count} comments!`, 'success');
            loadComments(); // Refresh the comment list
            loadStats(); // Refresh stats
        } else {
            showNotification('Failed to recalculate spam scores', 'error');
        }
    } catch (error) {
        console.error('Error recalculating spam scores:', error);
        showNotification('Failed to recalculate spam scores', 'error');
    }
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('spamSettingsModal');
    if (event.target === modal) {
        hideSpamSettings();
    }
}

// Add CSS for animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);