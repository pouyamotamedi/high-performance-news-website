// Progressive Web App functionality
class PWAManager {
  constructor() {
    this.deferredPrompt = null;
    this.isInstalled = false;
    this.init();
  }

  init() {
    this.checkInstallation();
    this.setupEventListeners();
    this.setupServiceWorker();
    this.handleOfflineStatus();
  }

  checkInstallation() {
    // Check if app is installed
    if (window.matchMedia('(display-mode: standalone)').matches || 
        window.navigator.standalone === true) {
      this.isInstalled = true;
      document.body.classList.add('pwa-installed');
    }
  }

  setupEventListeners() {
    // Listen for beforeinstallprompt event
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault();
      this.deferredPrompt = e;
      this.showInstallButton();
    });

    // Listen for app installed event
    window.addEventListener('appinstalled', () => {
      this.isInstalled = true;
      this.hideInstallButton();
      this.showInstallSuccess();
      document.body.classList.add('pwa-installed');
    });

    // Setup install button if it exists
    const installButton = document.querySelector('.pwa-install-button');
    if (installButton) {
      installButton.addEventListener('click', () => this.promptInstall());
    }

    // Handle online/offline status
    window.addEventListener('online', () => this.handleOnlineStatus());
    window.addEventListener('offline', () => this.handleOfflineStatus());
  }

  setupServiceWorker() {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/static/sw.js')
        .then((registration) => {
          console.log('ServiceWorker registered successfully');
          
          // Check for updates
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                this.showUpdateAvailable();
              }
            });
          });
        })
        .catch((error) => {
          console.log('ServiceWorker registration failed:', error);
        });

      // Listen for messages from service worker
      navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data && event.data.type === 'CACHE_UPDATED') {
          this.showCacheUpdated();
        }
      });
    }
  }

  showInstallButton() {
    const installButton = document.querySelector('.pwa-install-button');
    if (installButton) {
      installButton.style.display = 'block';
      installButton.classList.add('show');
    } else {
      this.createInstallButton();
    }
  }

  hideInstallButton() {
    const installButton = document.querySelector('.pwa-install-button');
    if (installButton) {
      installButton.style.display = 'none';
      installButton.classList.remove('show');
    }
  }

  createInstallButton() {
    const button = document.createElement('button');
    button.className = 'pwa-install-button btn btn-primary';
    button.innerHTML = `
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" stroke="currentColor" stroke-width="2"/>
        <polyline points="7,10 12,15 17,10" stroke="currentColor" stroke-width="2"/>
        <line x1="12" y1="15" x2="12" y2="3" stroke="currentColor" stroke-width="2"/>
      </svg>
      Install App
    `;
    
    button.addEventListener('click', () => this.promptInstall());
    
    // Add to header or create floating button
    const header = document.querySelector('.site-header .header-actions');
    if (header) {
      header.appendChild(button);
    } else {
      button.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 1000;
      `;
      document.body.appendChild(button);
    }
  }

  async promptInstall() {
    if (!this.deferredPrompt) return;

    this.deferredPrompt.prompt();
    const { outcome } = await this.deferredPrompt.userChoice;
    
    if (outcome === 'accepted') {
      console.log('User accepted the install prompt');
    } else {
      console.log('User dismissed the install prompt');
    }
    
    this.deferredPrompt = null;
  }

  showInstallSuccess() {
    this.showNotification('App installed successfully!', 'success');
  }

  showUpdateAvailable() {
    const notification = this.showNotification(
      'A new version is available. Refresh to update.',
      'info',
      true,
      [
        {
          text: 'Refresh',
          action: () => window.location.reload()
        },
        {
          text: 'Later',
          action: () => this.hideNotification()
        }
      ]
    );
  }

  showCacheUpdated() {
    this.showNotification('Content updated and cached for offline use.', 'success');
  }

  handleOnlineStatus() {
    document.body.classList.remove('offline');
    document.body.classList.add('online');
    this.hideOfflineIndicator();
    this.showNotification('You are back online!', 'success');
  }

  handleOfflineStatus() {
    document.body.classList.remove('online');
    document.body.classList.add('offline');
    this.showOfflineIndicator();
    this.showNotification('You are offline. Some features may be limited.', 'warning', false);
  }

  showOfflineIndicator() {
    // Remove any existing offline indicator first
    const existingIndicator = document.querySelector('.offline-indicator');
    if (existingIndicator) {
      existingIndicator.remove();
    }
    
    // Create new offline indicator
    const indicator = document.createElement('div');
    indicator.className = 'offline-indicator';
    indicator.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M1 9l2 2c4.97-4.97 13.03-4.97 18 0l2-2C16.93 2.93 7.07 2.93 1 9z" stroke="currentColor" stroke-width="2"/>
        <path d="M5 13l2 2c2.76-2.76 7.24-2.76 10 0l2-2C15.24 9.24 8.76 9.24 5 13z" stroke="currentColor" stroke-width="2"/>
        <path d="M9 17l2 2c.55-.55 1.45-.55 2 0l2-2C13.24 15.24 10.76 15.24 9 17z" stroke="currentColor" stroke-width="2"/>
        <line x1="1" y1="1" x2="23" y2="23" stroke="currentColor" stroke-width="2"/>
      </svg>
      Offline
    `;
    
    // Insert at the very beginning of body to ensure it's at the top
    document.body.insertBefore(indicator, document.body.firstChild);
    
    // Force styles to ensure proper positioning
    indicator.style.cssText = `
      position: fixed !important;
      top: 0 !important;
      left: 0 !important;
      right: 0 !important;
      background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%) !important;
      color: white !important;
      padding: 8px 16px !important;
      text-align: center !important;
      font-size: 14px !important;
      font-weight: 500 !important;
      z-index: 10000 !important;
      display: flex !important;
      align-items: center !important;
      justify-content: center !important;
      gap: 8px !important;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1) !important;
      transform: translateY(0) !important;
      transition: transform 0.3s ease-in-out !important;
    `;
    
    // Add show class after a brief delay to trigger animation
    setTimeout(() => {
      indicator.classList.add('show', 'animate-in');
    }, 10);
  }

  hideOfflineIndicator() {
    const indicator = document.querySelector('.offline-indicator');
    if (indicator) {
      indicator.classList.add('animate-out');
      indicator.classList.remove('show', 'animate-in');
      setTimeout(() => {
        if (indicator.parentNode) {
          indicator.remove();
        }
      }, 300);
    }
  }

  showNotification(message, type = 'info', persistent = false, actions = []) {
    // Remove existing notifications
    this.hideNotification();
    
    const notification = document.createElement('div');
    notification.className = `pwa-notification pwa-notification-${type}`;
    
    const icon = this.getNotificationIcon(type);
    
    notification.innerHTML = `
      <div class="notification-content">
        <div class="notification-icon">${icon}</div>
        <div class="notification-message">${message}</div>
        ${actions.length > 0 ? `
          <div class="notification-actions">
            ${actions.map(action => `
              <button class="notification-action" data-action="${action.text.toLowerCase()}">
                ${action.text}
              </button>
            `).join('')}
          </div>
        ` : ''}
        ${!persistent ? '<button class="notification-close" aria-label="Close">&times;</button>' : ''}
      </div>
    `;
    
    document.body.appendChild(notification);
    
    // Animate in using CSS classes
    requestAnimationFrame(() => {
      notification.classList.add('show');
    });
    
    // Setup event listeners
    const closeButton = notification.querySelector('.notification-close');
    if (closeButton) {
      closeButton.addEventListener('click', () => this.hideNotification());
    }
    
    actions.forEach(action => {
      const button = notification.querySelector(`[data-action="${action.text.toLowerCase()}"]`);
      if (button) {
        button.addEventListener('click', () => {
          action.action();
          this.hideNotification();
        });
      }
    });
    
    // Auto-hide after 5 seconds if not persistent
    if (!persistent) {
      setTimeout(() => this.hideNotification(), 5000);
    }
    
    return notification;
  }

  hideNotification() {
    const notification = document.querySelector('.pwa-notification');
    if (notification) {
      notification.classList.remove('show');
      setTimeout(() => {
        if (notification.parentNode) {
          notification.remove();
        }
      }, 300);
    }
  }

  getNotificationIcon(type) {
    const icons = {
      success: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M9 12l2 2 4-4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
      </svg>`,
      warning: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" stroke="currentColor" stroke-width="2"/>
        <line x1="12" y1="9" x2="12" y2="13" stroke="currentColor" stroke-width="2"/>
        <line x1="12" y1="17" x2="12.01" y2="17" stroke="currentColor" stroke-width="2"/>
      </svg>`,
      error: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
        <line x1="15" y1="9" x2="9" y2="15" stroke="currentColor" stroke-width="2"/>
        <line x1="9" y1="9" x2="15" y2="15" stroke="currentColor" stroke-width="2"/>
      </svg>`,
      info: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
        <line x1="12" y1="16" x2="12" y2="12" stroke="currentColor" stroke-width="2"/>
        <line x1="12" y1="8" x2="12.01" y2="8" stroke="currentColor" stroke-width="2"/>
      </svg>`
    };
    
    return icons[type] || icons.info;
  }

  // Public API
  isAppInstalled() {
    return this.isInstalled;
  }

  canInstall() {
    return !!this.deferredPrompt;
  }

  async install() {
    return this.promptInstall();
  }
}

// Initialize PWA manager when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.pwaManager = new PWAManager();
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { PWAManager };
}