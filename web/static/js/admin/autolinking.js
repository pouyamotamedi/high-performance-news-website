// Auto-Linking Management JavaScript

let allKeywords = [];
let allTags = [];

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    loadSystemStatus();
    loadGlobalSettings();
    loadKeywords();
});

// Load system status
async function loadSystemStatus() {
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/stats', {
            credentials: 'include'
        });
        
        if (response.ok) {
            const stats = await response.json();
            document.getElementById('totalKeywords').textContent = stats.total_keywords || 0;
            document.getElementById('activeTags').textContent = stats.active_tags || 0;
            document.getElementById('activeBanks').textContent = stats.active_banks || 0;
            document.getElementById('systemStatusValue').textContent = stats.status || 'Active';
            document.getElementById('systemStatusValue').style.color = stats.status === 'Active' ? '#10b981' : '#ef4444';
        }
    } catch (error) {
        console.error('Failed to load system status:', error);
    }
}

// Load global settings
async function loadGlobalSettings() {
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/settings', {
            credentials: 'include'
        });
        
        if (response.ok) {
            const settings = await response.json();
            document.getElementById('globalAutoLinking').checked = settings.global_enabled || false;
            document.getElementById('contentIngestionAutoLinking').checked = settings.content_ingestion_enabled || false;
        }
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

// Update global settings
async function updateGlobalSettings() {
    const settings = {
        global_enabled: document.getElementById('globalAutoLinking').checked,
        content_ingestion_enabled: document.getElementById('contentIngestionAutoLinking').checked
    };
    
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/settings', {
            method: 'PUT',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(settings)
        });
        
        if (response.ok) {
            showNotification('Settings updated successfully', 'success');
        } else {
            showNotification('Failed to update settings', 'error');
        }
    } catch (error) {
        console.error('Failed to update settings:', error);
        showNotification('Failed to update settings', 'error');
    }
}

// Load keywords from tags
async function loadKeywords() {
    try {
        // Load tags
        const tagsResponse = await fetch('/api/v1/admin-panel/tags?limit=1000', {
            credentials: 'include'
        });
        
        // Load keyword banks
        const banksResponse = await fetch('/api/v1/admin-panel/autolinking/keyword-banks', {
            credentials: 'include'
        });
        
        if (tagsResponse.ok) {
            const data = await tagsResponse.json();
            allTags = data.tags || [];
            
            // Extract all keywords from tags
            allKeywords = [];
            allTags.forEach(tag => {
                if (tag.keywords && Array.isArray(tag.keywords)) {
                    tag.keywords.forEach(keyword => {
                        allKeywords.push({
                            keyword: keyword,
                            tag: tag,
                            type: 'tag'
                        });
                    });
                }
            });
            
            // Add keywords from keyword banks
            if (banksResponse.ok) {
                const banks = await banksResponse.json();
                banks.forEach(bank => {
                    if (bank.is_active && bank.keywords && Array.isArray(bank.keywords)) {
                        bank.keywords.forEach(keyword => {
                            allKeywords.push({
                                keyword: keyword,
                                tag: {
                                    name: bank.name,
                                    color: '#8b5cf6' // Purple for keyword banks
                                },
                                type: 'bank',
                                url: bank.url
                            });
                        });
                    }
                });
            }
            
            displayKeywords(allKeywords);
        }
    } catch (error) {
        console.error('Failed to load keywords:', error);
        document.getElementById('keywordsList').innerHTML = 
            '<div style="text-align: center; padding: 2rem; color: #ef4444;">Failed to load keywords</div>';
    }
}

// Display keywords
function displayKeywords(keywords) {
    const container = document.getElementById('keywordsList');
    
    if (keywords.length === 0) {
        container.innerHTML = '<div style="text-align: center; padding: 2rem; color: #9ca3af;">No keywords found. Add keywords to tags or keyword banks to enable auto-linking.</div>';
        return;
    }
    
    let html = '';
    keywords.forEach(item => {
        const tagColor = item.tag.color || '#3b82f6';
        const typeIcon = item.type === 'bank' ? '🏦' : '🏷️';
        const typeLabel = item.type === 'bank' ? 'Bank' : 'Tag';
        html += `
            <div class="keyword-item">
                <div>
                    <span class="keyword-text">${escapeHtml(item.keyword)}</span>
                    <span style="font-size: 0.75rem; color: #6b7280; margin-left: 0.5rem;">${typeIcon} ${typeLabel}</span>
                </div>
                <div>
                    <span class="keyword-tag" style="background-color: ${tagColor}20; color: ${tagColor};">
                        ${escapeHtml(item.tag.name)}
                    </span>
                </div>
            </div>
        `;
    });
    
    container.innerHTML = html;
}

// Filter keywords
function filterKeywords() {
    const searchTerm = document.getElementById('keywordSearch').value.toLowerCase();
    
    if (!searchTerm) {
        displayKeywords(allKeywords);
        return;
    }
    
    const filtered = allKeywords.filter(item => 
        item.keyword.toLowerCase().includes(searchTerm) ||
        item.tag.name.toLowerCase().includes(searchTerm)
    );
    
    displayKeywords(filtered);
}

// Refresh keywords
async function refreshKeywords() {
    showNotification('Refreshing keywords...', 'info');
    
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/refresh', {
            method: 'POST',
            credentials: 'include'
        });
        
        if (response.ok) {
            await loadKeywords();
            await loadSystemStatus();
            showNotification('Keywords refreshed successfully', 'success');
        } else {
            showNotification('Failed to refresh keywords', 'error');
        }
    } catch (error) {
        console.error('Failed to refresh keywords:', error);
        showNotification('Failed to refresh keywords', 'error');
    }
}

// Check for conflicts
async function checkConflicts() {
    const container = document.getElementById('conflictsList');
    container.innerHTML = '<div style="text-align: center; padding: 1rem; color: #9ca3af;">Checking for conflicts...</div>';
    
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/conflicts', {
            credentials: 'include'
        });
        
        if (response.ok) {
            const conflicts = await response.json();
            
            if (conflicts.length === 0) {
                container.innerHTML = '<div style="text-align: center; padding: 1rem; color: #10b981;">✓ No conflicts found</div>';
                return;
            }
            
            let html = '';
            conflicts.forEach(conflictMessage => {
                html += `
                    <div class="conflict-item" style="padding: 0.75rem; margin-bottom: 0.5rem; background: #fef3c7; border-left: 3px solid #f59e0b; border-radius: 4px;">
                        ⚠️ ${escapeHtml(conflictMessage)}
                    </div>
                `;
            });
            
            container.innerHTML = html;
        }
    } catch (error) {
        console.error('Failed to check conflicts:', error);
        container.innerHTML = '<div style="text-align: center; padding: 1rem; color: #ef4444;">Failed to check conflicts</div>';
    }
}

// Test auto-linking
async function testAutoLinking() {
    const text = document.getElementById('testText').value;
    
    if (!text.trim()) {
        showNotification('Please enter some text to test', 'error');
        return;
    }
    
    const resultDiv = document.getElementById('testResult');
    const resultContent = document.getElementById('testResultContent');
    
    resultDiv.style.display = 'block';
    resultContent.innerHTML = '<div style="text-align: center; padding: 1rem; color: #9ca3af;">Processing...</div>';
    
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/test', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ text: text })
        });
        
        if (response.ok) {
            const result = await response.json();
            resultContent.innerHTML = result.processed_text || text;
            
            if (result.matches && result.matches.length > 0) {
                resultContent.innerHTML += `
                    <div style="margin-top: 1rem; padding: 0.75rem; background: #f3f4f6; border-radius: 6px;">
                        <strong>Matches found:</strong> ${result.matches.length}
                        <ul style="margin-top: 0.5rem; padding-left: 1.5rem;">
                            ${result.matches.map(m => `<li>${escapeHtml(m.keyword)} → ${escapeHtml(m.tag)}</li>`).join('')}
                        </ul>
                    </div>
                `;
            }
        } else {
            resultContent.innerHTML = '<div style="color: #ef4444;">Failed to process text</div>';
        }
    } catch (error) {
        console.error('Failed to test auto-linking:', error);
        resultContent.innerHTML = '<div style="color: #ef4444;">Failed to process text</div>';
    }
}

// Show notification
function showNotification(message, type = 'info') {
    const colors = {
        success: '#10b981',
        error: '#ef4444',
        info: '#3b82f6',
        warning: '#f59e0b'
    };
    
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 1rem 1.5rem;
        background: ${colors[type]};
        color: white;
        border-radius: 6px;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
    `;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);


// Reprocess all articles
async function reprocessAllArticles() {
    if (!confirm('This will reprocess all articles with auto-linking enabled. This may take a few minutes. Continue?')) {
        return;
    }
    
    // Show loading notification
    showNotification('Processing articles... Please wait.', 'info');
    
    // Disable the button
    const button = event.target;
    button.disabled = true;
    button.textContent = '⏳ Processing...';
    
    try {
        const response = await fetch('/api/v1/admin-panel/autolinking/reprocess', {
            method: 'POST',
            credentials: 'include'
        });
        
        if (response.ok) {
            const result = await response.json();
            showNotification(`Reprocessing completed! Processed: ${result.processed}, Updated: ${result.updated}`, 'success');
            setTimeout(() => {
                loadSystemStatus();
            }, 1000);
        } else {
            showNotification('Failed to reprocess articles', 'error');
        }
    } catch (error) {
        console.error('Failed to reprocess articles:', error);
        showNotification('Failed to reprocess articles', 'error');
    } finally {
        // Re-enable the button
        button.disabled = false;
        button.textContent = '⚡ Reprocess All Articles';
    }
}
