// Main JavaScript functionality
class NewsApp {
  constructor() {
    this.init();
  }

  init() {
    this.setupEventListeners();
    this.setupMobileMenu();
    this.setupSearch();
    this.setupLazyLoading();
    this.setupSmoothScrolling();
    this.setupFormValidation();
    this.setupAccessibility();
  }

  setupEventListeners() {
    // Mobile menu toggle
    const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
    const mobileNavOverlay = document.querySelector('.mobile-nav-overlay');
    
    if (mobileMenuToggle && mobileNavOverlay) {
      mobileMenuToggle.addEventListener('click', () => {
        this.toggleMobileMenu();
      });

      // Close mobile menu when clicking outside
      mobileNavOverlay.addEventListener('click', (e) => {
        if (e.target === mobileNavOverlay) {
          this.closeMobileMenu();
        }
      });
    }

    // Search toggle
    const searchToggle = document.querySelector('.search-toggle');
    const searchOverlay = document.querySelector('.search-overlay');
    const searchClose = document.querySelector('.search-close');
    
    if (searchToggle && searchOverlay) {
      searchToggle.addEventListener('click', () => {
        this.toggleSearch();
      });
    }
    
    if (searchClose) {
      searchClose.addEventListener('click', () => {
        this.closeSearch();
      });
    }

    // Language switcher
    const languageToggle = document.querySelector('.language-toggle');
    const languageDropdown = document.querySelector('.language-dropdown');
    
    if (languageToggle && languageDropdown) {
      languageToggle.addEventListener('click', () => {
        this.toggleLanguageDropdown();
      });

      // Close dropdown when clicking outside
      document.addEventListener('click', (e) => {
        if (!languageToggle.contains(e.target) && !languageDropdown.contains(e.target)) {
          this.closeLanguageDropdown();
        }
      });
    }

    // User menu
    const userToggle = document.querySelector('.user-toggle');
    const userDropdown = document.querySelector('.user-dropdown');
    
    if (userToggle && userDropdown) {
      userToggle.addEventListener('click', () => {
        this.toggleUserDropdown();
      });

      // Close dropdown when clicking outside
      document.addEventListener('click', (e) => {
        if (!userToggle.contains(e.target) && !userDropdown.contains(e.target)) {
          this.closeUserDropdown();
        }
      });
    }

    // View toggle buttons
    const viewButtons = document.querySelectorAll('.view-button');
    viewButtons.forEach(button => {
      button.addEventListener('click', () => {
        this.toggleView(button.dataset.view);
      });
    });

    // Keyboard navigation
    document.addEventListener('keydown', (e) => {
      this.handleKeyboardNavigation(e);
    });

    // Window resize
    window.addEventListener('resize', () => {
      this.handleResize();
    });
  }

  setupMobileMenu() {
    const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
    
    if (mobileMenuToggle) {
      // Add ARIA attributes
      mobileMenuToggle.setAttribute('aria-expanded', 'false');
      mobileMenuToggle.setAttribute('aria-controls', 'mobile-navigation');
    }
  }

  toggleMobileMenu() {
    const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
    const mobileNavOverlay = document.querySelector('.mobile-nav-overlay');
    const body = document.body;
    
    const isOpen = mobileNavOverlay.classList.contains('active');
    
    if (isOpen) {
      this.closeMobileMenu();
    } else {
      mobileNavOverlay.classList.add('active');
      mobileMenuToggle.classList.add('active');
      mobileMenuToggle.setAttribute('aria-expanded', 'true');
      body.classList.add('mobile-menu-open');
      
      // Focus first menu item
      const firstMenuItem = mobileNavOverlay.querySelector('.mobile-nav-link');
      if (firstMenuItem) {
        firstMenuItem.focus();
      }
    }
  }

  closeMobileMenu() {
    const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
    const mobileNavOverlay = document.querySelector('.mobile-nav-overlay');
    const body = document.body;
    
    mobileNavOverlay.classList.remove('active');
    mobileMenuToggle.classList.remove('active');
    mobileMenuToggle.setAttribute('aria-expanded', 'false');
    body.classList.remove('mobile-menu-open');
  }

  setupSearch() {
    const searchForm = document.querySelector('.search-form');
    const searchInput = document.querySelector('.search-input');
    
    if (searchForm && searchInput) {
      // Auto-focus search input when overlay opens
      const searchOverlay = document.querySelector('.search-overlay');
      const observer = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
          if (mutation.type === 'attributes' && mutation.attributeName === 'class') {
            if (searchOverlay.classList.contains('active')) {
              setTimeout(() => searchInput.focus(), 100);
            }
          }
        });
      });
      
      observer.observe(searchOverlay, { attributes: true });

      // Search suggestions (if implemented)
      searchInput.addEventListener('input', (e) => {
        this.handleSearchInput(e.target.value);
      });
    }
  }

  toggleSearch() {
    const searchOverlay = document.querySelector('.search-overlay');
    const body = document.body;
    
    const isOpen = searchOverlay.classList.contains('active');
    
    if (isOpen) {
      this.closeSearch();
    } else {
      searchOverlay.classList.add('active');
      body.classList.add('search-open');
    }
  }

  closeSearch() {
    const searchOverlay = document.querySelector('.search-overlay');
    const body = document.body;
    
    searchOverlay.classList.remove('active');
    body.classList.remove('search-open');
  }

  handleSearchInput(query) {
    // Debounce search suggestions
    clearTimeout(this.searchTimeout);
    this.searchTimeout = setTimeout(() => {
      if (query.length >= 2) {
        this.fetchSearchSuggestions(query);
      }
    }, 300);
  }

  async fetchSearchSuggestions(query) {
    try {
      const response = await fetch(`/api/search/suggestions?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const suggestions = await response.json();
        this.displaySearchSuggestions(suggestions);
      }
    } catch (error) {
      console.warn('Failed to fetch search suggestions:', error);
    }
  }

  displaySearchSuggestions(suggestions) {
    // Implementation for displaying search suggestions
    // This would create a dropdown with suggestions
    console.log('Search suggestions:', suggestions);
  }

  toggleLanguageDropdown() {
    const languageDropdown = document.querySelector('.language-dropdown');
    const languageToggle = document.querySelector('.language-toggle');
    
    const isOpen = languageDropdown.classList.contains('active');
    
    if (isOpen) {
      this.closeLanguageDropdown();
    } else {
      languageDropdown.classList.add('active');
      languageToggle.setAttribute('aria-expanded', 'true');
    }
  }

  closeLanguageDropdown() {
    const languageDropdown = document.querySelector('.language-dropdown');
    const languageToggle = document.querySelector('.language-toggle');
    
    languageDropdown.classList.remove('active');
    languageToggle.setAttribute('aria-expanded', 'false');
  }

  toggleUserDropdown() {
    const userDropdown = document.querySelector('.user-dropdown');
    const userToggle = document.querySelector('.user-toggle');
    
    const isOpen = userDropdown.classList.contains('active');
    
    if (isOpen) {
      this.closeUserDropdown();
    } else {
      userDropdown.classList.add('active');
      userToggle.setAttribute('aria-expanded', 'true');
    }
  }

  closeUserDropdown() {
    const userDropdown = document.querySelector('.user-dropdown');
    const userToggle = document.querySelector('.user-toggle');
    
    userDropdown.classList.remove('active');
    userToggle.setAttribute('aria-expanded', 'false');
  }

  toggleView(viewType) {
    const articlesContainer = document.querySelector('.articles-container');
    const viewButtons = document.querySelectorAll('.view-button');
    
    if (articlesContainer) {
      articlesContainer.setAttribute('data-view', viewType);
      
      // Update button states
      viewButtons.forEach(button => {
        button.classList.toggle('active', button.dataset.view === viewType);
      });
      
      // Save preference
      try {
        localStorage.setItem('preferred-view', viewType);
      } catch (e) {
        // Ignore localStorage errors
      }
    }
  }

  setupLazyLoading() {
    // Lazy load images
    const images = document.querySelectorAll('img[loading="lazy"]');
    
    if ('IntersectionObserver' in window) {
      const imageObserver = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            const img = entry.target;
            if (img.dataset.src) {
              img.src = img.dataset.src;
              img.removeAttribute('data-src');
            }
            imageObserver.unobserve(img);
          }
        });
      });

      images.forEach(img => imageObserver.observe(img));
    }
  }

  setupSmoothScrolling() {
    // Smooth scroll for anchor links
    const anchorLinks = document.querySelectorAll('a[href^="#"]');
    
    anchorLinks.forEach(link => {
      link.addEventListener('click', (e) => {
        const href = link.getAttribute('href');
        const target = document.querySelector(href);
        
        if (target) {
          e.preventDefault();
          target.scrollIntoView({
            behavior: 'smooth',
            block: 'start'
          });
          
          // Update URL without jumping
          history.pushState(null, null, href);
        }
      });
    });
  }

  setupFormValidation() {
    const forms = document.querySelectorAll('form[data-validate]');
    
    forms.forEach(form => {
      form.addEventListener('submit', (e) => {
        if (!this.validateForm(form)) {
          e.preventDefault();
        }
      });

      // Real-time validation
      const inputs = form.querySelectorAll('input, textarea, select');
      inputs.forEach(input => {
        input.addEventListener('blur', () => {
          this.validateField(input);
        });
      });
    });
  }

  validateForm(form) {
    const inputs = form.querySelectorAll('input, textarea, select');
    let isValid = true;
    
    inputs.forEach(input => {
      if (!this.validateField(input)) {
        isValid = false;
      }
    });
    
    return isValid;
  }

  validateField(field) {
    const value = field.value.trim();
    const type = field.type;
    const required = field.hasAttribute('required');
    
    let isValid = true;
    let errorMessage = '';
    
    // Required validation
    if (required && !value) {
      isValid = false;
      errorMessage = 'This field is required';
    }
    
    // Email validation
    if (type === 'email' && value) {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailRegex.test(value)) {
        isValid = false;
        errorMessage = 'Please enter a valid email address';
      }
    }
    
    // URL validation
    if (type === 'url' && value) {
      try {
        new URL(value);
      } catch {
        isValid = false;
        errorMessage = 'Please enter a valid URL';
      }
    }
    
    this.displayFieldValidation(field, isValid, errorMessage);
    return isValid;
  }

  displayFieldValidation(field, isValid, errorMessage) {
    const fieldGroup = field.closest('.form-group');
    if (!fieldGroup) return;
    
    // Remove existing error
    const existingError = fieldGroup.querySelector('.field-error');
    if (existingError) {
      existingError.remove();
    }
    
    // Update field state
    field.classList.toggle('field-invalid', !isValid);
    field.classList.toggle('field-valid', isValid);
    
    // Add error message
    if (!isValid && errorMessage) {
      const errorElement = document.createElement('div');
      errorElement.className = 'field-error';
      errorElement.textContent = errorMessage;
      fieldGroup.appendChild(errorElement);
    }
  }

  setupAccessibility() {
    // Skip link functionality
    const skipLink = document.querySelector('.skip-link');
    if (skipLink) {
      skipLink.addEventListener('click', (e) => {
        e.preventDefault();
        const target = document.querySelector(skipLink.getAttribute('href'));
        if (target) {
          target.focus();
          target.scrollIntoView();
        }
      });
    }

    // Focus management for modals and overlays
    this.setupFocusTrapping();
    
    // Announce dynamic content changes
    this.setupLiveRegions();
  }

  setupFocusTrapping() {
    const overlays = document.querySelectorAll('.search-overlay, .mobile-nav-overlay');
    
    overlays.forEach(overlay => {
      overlay.addEventListener('keydown', (e) => {
        if (e.key === 'Tab') {
          this.trapFocus(e, overlay);
        }
        
        if (e.key === 'Escape') {
          this.closeOverlay(overlay);
        }
      });
    });
  }

  trapFocus(e, container) {
    const focusableElements = container.querySelectorAll(
      'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
    );
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    
    if (e.shiftKey) {
      if (document.activeElement === firstElement) {
        e.preventDefault();
        lastElement.focus();
      }
    } else {
      if (document.activeElement === lastElement) {
        e.preventDefault();
        firstElement.focus();
      }
    }
  }

  closeOverlay(overlay) {
    if (overlay.classList.contains('search-overlay')) {
      this.closeSearch();
    } else if (overlay.classList.contains('mobile-nav-overlay')) {
      this.closeMobileMenu();
    }
  }

  setupLiveRegions() {
    // Create live region for announcements
    if (!document.querySelector('#live-region')) {
      const liveRegion = document.createElement('div');
      liveRegion.id = 'live-region';
      liveRegion.setAttribute('aria-live', 'polite');
      liveRegion.setAttribute('aria-atomic', 'true');
      liveRegion.style.cssText = 'position: absolute; left: -10000px; width: 1px; height: 1px; overflow: hidden;';
      document.body.appendChild(liveRegion);
    }
  }

  announce(message) {
    const liveRegion = document.querySelector('#live-region');
    if (liveRegion) {
      liveRegion.textContent = message;
    }
  }

  handleKeyboardNavigation(e) {
    // Global keyboard shortcuts
    if (e.ctrlKey || e.metaKey) {
      switch (e.key) {
        case 'k':
          e.preventDefault();
          this.toggleSearch();
          break;
        case '/':
          e.preventDefault();
          this.toggleSearch();
          break;
      }
    }
    
    // Escape key handling
    if (e.key === 'Escape') {
      this.closeSearch();
      this.closeMobileMenu();
      this.closeLanguageDropdown();
      this.closeUserDropdown();
    }
  }

  handleResize() {
    // Close mobile menu on desktop
    if (window.innerWidth >= 1024) {
      this.closeMobileMenu();
    }
  }

  // Utility methods
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  throttle(func, limit) {
    let inThrottle;
    return function() {
      const args = arguments;
      const context = this;
      if (!inThrottle) {
        func.apply(context, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.newsApp = new NewsApp();
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { NewsApp };
}

// Debug function to test offline indicator (remove in production)
window.testOfflineMode = function() {
  if (window.pwaManager) {
    document.body.classList.add('offline');
    window.pwaManager.showOfflineIndicator();
    window.pwaManager.showNotification('Testing offline mode', 'warning', false);
  }
};

// Debug function to test online mode (remove in production)
window.testOnlineMode = function() {
  if (window.pwaManager) {
    document.body.classList.remove('offline');
    document.body.classList.add('online');
    window.pwaManager.hideOfflineIndicator();
    window.pwaManager.showNotification('Back online!', 'success', false);
  }
};