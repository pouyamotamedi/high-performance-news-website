// Global application JavaScript

// WebSocket connection
let ws = null;
let wsReconnectAttempts = 0;
const maxReconnectAttempts = 5;

// Initialize WebSocket connection
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = function(event) {
        console.log('WebSocket connected');
        wsReconnectAttempts = 0;
        updateConnectionStatus(true);
    };
    
    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        handleWebSocketMessage(data);
    };
    
    ws.onclose = function(event) {
        console.log('WebSocket disconnected');
        updateConnectionStatus(false);
        
        // Attempt to reconnect
        if (wsReconnectAttempts < maxReconnectAttempts) {
            wsReconnectAttempts++;
            setTimeout(() => {
                console.log(`Attempting to reconnect (${wsReconnectAttempts}/${maxReconnectAttempts})`);
                initWebSocket();
            }, 2000 * wsReconnectAttempts);
        }
    };
    
    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        updateConnectionStatus(false, 'Connection error');
    };
    
    // Make WebSocket available globally
    window.ws = ws;
}

// Handle WebSocket messages
function handleWebSocketMessage(data) {
    console.log('WebSocket message:', data);
    
    switch (data.type) {
        case 'connected':
            showNotification('Connected to deployment service', 'success');
            break;
        case 'setup_start':
        case 'deploy_start':
            showNotification(data.message, 'info');
            break;
        case 'setup_complete':
        case 'deploy_complete':
            showNotification(data.message, 'success');
            break;
        case 'setup_error':
        case 'deploy_error':
            showNotification(data.message, 'error');
            break;
        default:
            console.log('Unknown message type:', data.type);
    }
}

// Update connection status indicator
function updateConnectionStatus(connected, message = '') {
    const indicator = document.getElementById('status-indicator');
    const text = document.getElementById('status-text');
    
    if (!indicator || !text) return;
    
    if (connected) {
        indicator.className = 'w-3 h-3 bg-green-400 rounded-full';
        text.textContent = 'Connected';
        text.className = 'text-sm text-green-600';
    } else {
        indicator.className = 'w-3 h-3 bg-red-400 rounded-full';
        text.textContent = message || 'Disconnected';
        text.className = 'text-sm text-red-600';
    }
}

// Notification system
function showNotification(message, type = 'info', duration = 5000) {
    const container = document.getElementById('notifications');
    if (!container) return;
    
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    
    const iconMap = {
        success: 'fas fa-check-circle text-green-400',
        error: 'fas fa-exclamation-circle text-red-400',
        warning: 'fas fa-exclamation-triangle text-yellow-400',
        info: 'fas fa-info-circle text-blue-400'
    };
    
    notification.innerHTML = `
        <div class="p-4">
            <div class="flex items-start">
                <div class="flex-shrink-0">
                    <i class="${iconMap[type] || iconMap.info}"></i>
                </div>
                <div class="ml-3 w-0 flex-1 pt-0.5">
                    <p class="text-sm font-medium text-gray-900">${message}</p>
                </div>
                <div class="ml-4 flex-shrink-0 flex">
                    <button class="bg-white rounded-md inline-flex text-gray-400 hover:text-gray-500 focus:outline-none" onclick="removeNotification(this)">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
            </div>
        </div>
    `;
    
    container.appendChild(notification);
    
    // Auto-remove after duration
    if (duration > 0) {
        setTimeout(() => {
            removeNotification(notification.querySelector('button'));
        }, duration);
    }
}

// Remove notification
function removeNotification(button) {
    const notification = button.closest('.notification');
    if (notification) {
        notification.classList.add('removing');
        setTimeout(() => {
            notification.remove();
        }, 300);
    }
}

// Utility functions
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

function formatDuration(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
        return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
        return `${minutes}m ${secs}s`;
    } else {
        return `${secs}s`;
    }
}

function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString();
}

// API helper functions
async function apiRequest(url, options = {}) {
    try {
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// Loading state management
function showLoading(element, message = 'Loading...') {
    if (typeof element === 'string') {
        element = document.getElementById(element);
    }
    
    if (element) {
        element.innerHTML = `
            <div class="flex items-center justify-center py-4">
                <div class="spinner mr-2"></div>
                <span class="text-gray-600">${message}</span>
            </div>
        `;
    }
}

function hideLoading(element) {
    if (typeof element === 'string') {
        element = document.getElementById(element);
    }
    
    if (element) {
        element.innerHTML = '';
    }
}

// Form validation helpers
function validateRequired(value, fieldName) {
    if (!value || value.trim() === '') {
        throw new Error(`${fieldName} is required`);
    }
    return value.trim();
}

function validateEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
        throw new Error('Invalid email format');
    }
    return email;
}

function validatePort(port) {
    const portNum = parseInt(port);
    if (isNaN(portNum) || portNum < 1 || portNum > 65535) {
        throw new Error('Port must be between 1 and 65535');
    }
    return portNum;
}

// Local storage helpers
function saveToStorage(key, data) {
    try {
        localStorage.setItem(key, JSON.stringify(data));
    } catch (error) {
        console.error('Failed to save to storage:', error);
    }
}

function loadFromStorage(key, defaultValue = null) {
    try {
        const data = localStorage.getItem(key);
        return data ? JSON.parse(data) : defaultValue;
    } catch (error) {
        console.error('Failed to load from storage:', error);
        return defaultValue;
    }
}

function removeFromStorage(key) {
    try {
        localStorage.removeItem(key);
    } catch (error) {
        console.error('Failed to remove from storage:', error);
    }
}

// Theme management
function initTheme() {
    const savedTheme = loadFromStorage('theme', 'light');
    applyTheme(savedTheme);
}

function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    saveToStorage('theme', theme);
}

function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    applyTheme(newTheme);
}

// Keyboard shortcuts
function initKeyboardShortcuts() {
    document.addEventListener('keydown', function(event) {
        // Ctrl/Cmd + R: Refresh current page
        if ((event.ctrlKey || event.metaKey) && event.key === 'r') {
            event.preventDefault();
            window.location.reload();
        }
        
        // Ctrl/Cmd + K: Focus search (if available)
        if ((event.ctrlKey || event.metaKey) && event.key === 'k') {
            event.preventDefault();
            const searchInput = document.querySelector('input[type="search"], input[placeholder*="search" i]');
            if (searchInput) {
                searchInput.focus();
            }
        }
        
        // Escape: Close modals/notifications
        if (event.key === 'Escape') {
            const notifications = document.querySelectorAll('.notification');
            notifications.forEach(notification => {
                const closeButton = notification.querySelector('button');
                if (closeButton) {
                    removeNotification(closeButton);
                }
            });
        }
    });
}

// Initialize application
document.addEventListener('DOMContentLoaded', function() {
    console.log('Desktop deployment application initializing...');
    
    // Initialize WebSocket connection
    initWebSocket();
    
    // Initialize theme
    initTheme();
    
    // Initialize keyboard shortcuts
    initKeyboardShortcuts();
    
    // Show welcome message
    setTimeout(() => {
        showNotification('Desktop deployment application ready', 'success');
    }, 1000);
    
    console.log('Desktop deployment application initialized');
});

// Handle page visibility changes
document.addEventListener('visibilitychange', function() {
    if (document.hidden) {
        console.log('Page hidden');
    } else {
        console.log('Page visible');
        // Reconnect WebSocket if needed
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            initWebSocket();
        }
    }
});

// Handle window beforeunload
window.addEventListener('beforeunload', function(event) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
    }
});

// Export functions for global use
window.showNotification = showNotification;
window.removeNotification = removeNotification;
window.apiRequest = apiRequest;
window.showLoading = showLoading;
window.hideLoading = hideLoading;
window.formatBytes = formatBytes;
window.formatDuration = formatDuration;
window.formatTimestamp = formatTimestamp;