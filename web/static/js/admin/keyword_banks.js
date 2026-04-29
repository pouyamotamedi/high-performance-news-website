// Load keyword banks on page load
document.addEventListener('DOMContentLoaded', () => {
    loadKeywordBanks();
});

// Load all keyword banks
async function loadKeywordBanks() {
    try {
        const response = await fetch('/api/v1/admin-panel/keyword-banks', {
            credentials: 'include'
        });
        
        if (!response.ok) {
            throw new Error('Failed to load keyword banks');
        }
        
        const banks = await response.json();
        displayKeywordBanks(banks);
    } catch (error) {
        console.error('Error loading keyword banks:', error);
        showNotification('Failed to load keyword banks', 'error');
        document.getElementById('loading').style.display = 'none';
    }
}

// Display keyword banks in table
function displayKeywordBanks(banks) {
    const loading = document.getElementById('loading');
    const emptyState = document.getElementById('empty-state');
    const table = document.getElementById('banks-table');
    const tbody = document.getElementById('banks-tbody');
    
    loading.style.display = 'none';
    
    if (!banks || banks.length === 0) {
        emptyState.style.display = 'block';
        table.style.display = 'none';
        return;
    }
    
    emptyState.style.display = 'none';
    table.style.display = 'table';
    
    tbody.innerHTML = banks.map(bank => `
        <tr>
            <td><strong>${escapeHtml(bank.name)}</strong></td>
            <td>
                <a href="${escapeHtml(bank.url)}" target="_blank" style="color: #3b82f6; text-decoration: none;">
                    ${escapeHtml(bank.url)}
                </a>
            </td>
            <td>
                <div class="keywords-preview" title="${escapeHtml(bank.keywords.join(', '))}">
                    ${bank.keywords.length} keyword${bank.keywords.length !== 1 ? 's' : ''}: 
                    ${escapeHtml(bank.keywords.slice(0, 3).join(', '))}
                    ${bank.keywords.length > 3 ? '...' : ''}
                </div>
            </td>
            <td>
                <span class="kb-badge ${bank.is_active ? 'kb-badge-active' : 'kb-badge-inactive'}">
                    ${bank.is_active ? 'Active' : 'Inactive'}
                </span>
            </td>
            <td>
                <div class="kb-action-buttons">
                    <button class="kb-btn kb-btn-sm kb-btn-primary" onclick="editBank(${bank.id})">
                        ✏️ Edit
                    </button>
                    <button class="kb-btn kb-btn-sm ${bank.is_active ? 'kb-btn-secondary' : 'kb-btn-success'}" 
                            onclick="toggleBankStatus(${bank.id}, ${!bank.is_active})">
                        ${bank.is_active ? '⏸️ Disable' : '▶️ Enable'}
                    </button>
                    <button class="kb-btn kb-btn-sm kb-btn-danger" onclick="deleteBank(${bank.id}, '${escapeHtml(bank.name)}')">
                        🗑️ Delete
                    </button>
                </div>
            </td>
        </tr>
    `).join('');
}

// Open create modal
function openCreateModal() {
    document.getElementById('modal-title').textContent = 'Create Keyword Bank';
    document.getElementById('bank-form').reset();
    document.getElementById('bank-id').value = '';
    document.getElementById('bank-active').checked = true;
    document.getElementById('modal').style.display = 'block';
}

// Edit bank
async function editBank(id) {
    try {
        const response = await fetch(`/api/v1/admin-panel/keyword-banks/${id}`, {
            credentials: 'include'
        });
        
        if (!response.ok) {
            throw new Error('Failed to load keyword bank');
        }
        
        const bank = await response.json();
        
        document.getElementById('modal-title').textContent = 'Edit Keyword Bank';
        document.getElementById('bank-id').value = bank.id;
        document.getElementById('bank-name').value = bank.name;
        document.getElementById('bank-url').value = bank.url;
        document.getElementById('bank-keywords').value = bank.keywords.join('\n');
        document.getElementById('bank-description').value = bank.description || '';
        document.getElementById('bank-active').checked = bank.is_active;
        document.getElementById('modal').style.display = 'block';
    } catch (error) {
        console.error('Error loading keyword bank:', error);
        showNotification('Failed to load keyword bank', 'error');
    }
}

// Close modal
function closeModal() {
    document.getElementById('modal').style.display = 'none';
}

// Handle form submit
async function handleSubmit(event) {
    event.preventDefault();
    
    const id = document.getElementById('bank-id').value;
    const name = document.getElementById('bank-name').value.trim();
    const url = document.getElementById('bank-url').value.trim();
    const keywordsText = document.getElementById('bank-keywords').value.trim();
    const description = document.getElementById('bank-description').value.trim();
    const isActive = document.getElementById('bank-active').checked;
    
    // Parse keywords (one per line, remove empty lines)
    const keywords = keywordsText
        .split('\n')
        .map(k => k.trim())
        .filter(k => k.length > 0);
    
    if (keywords.length === 0) {
        showNotification('Please enter at least one keyword', 'error');
        return;
    }
    
    const data = {
        name,
        url,
        keywords,
        description,
        is_active: isActive
    };
    
    try {
        const method = id ? 'PUT' : 'POST';
        const endpoint = id 
            ? `/api/v1/admin-panel/keyword-banks/${id}`
            : '/api/v1/admin-panel/keyword-banks';
        
        const response = await fetch(endpoint, {
            method,
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save keyword bank');
        }
        
        showNotification(
            id ? 'Keyword bank updated successfully' : 'Keyword bank created successfully',
            'success'
        );
        
        closeModal();
        loadKeywordBanks();
    } catch (error) {
        console.error('Error saving keyword bank:', error);
        showNotification(error.message, 'error');
    }
}

// Toggle bank status
async function toggleBankStatus(id, newStatus) {
    try {
        const response = await fetch(`/api/v1/admin-panel/keyword-banks/${id}`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ is_active: newStatus })
        });
        
        if (!response.ok) {
            throw new Error('Failed to update status');
        }
        
        showNotification(
            `Keyword bank ${newStatus ? 'enabled' : 'disabled'} successfully`,
            'success'
        );
        
        loadKeywordBanks();
    } catch (error) {
        console.error('Error updating status:', error);
        showNotification('Failed to update status', 'error');
    }
}

// Delete bank
async function deleteBank(id, name) {
    if (!confirm(`Are you sure you want to delete the keyword bank "${name}"? This action cannot be undone.`)) {
        return;
    }
    
    try {
        const response = await fetch(`/api/v1/admin-panel/keyword-banks/${id}`, {
            method: 'DELETE',
            credentials: 'include'
        });
        
        if (!response.ok) {
            throw new Error('Failed to delete keyword bank');
        }
        
        showNotification('Keyword bank deleted successfully', 'success');
        loadKeywordBanks();
    } catch (error) {
        console.error('Error deleting keyword bank:', error);
        showNotification('Failed to delete keyword bank', 'error');
    }
}

// Show notification
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `kb-notification kb-notification-${type}`;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

// Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('modal');
    if (event.target === modal) {
        closeModal();
    }
};
