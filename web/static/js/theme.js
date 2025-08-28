// Theme management system
class ThemeManager {
  constructor() {
    this.themes = ['light', 'dark', 'auto'];
    this.currentTheme = this.getStoredTheme() || 'auto';
    this.init();
  }

  init() {
    this.applyTheme(this.currentTheme);
    this.setupEventListeners();
    this.setupSystemThemeListener();
  }

  getStoredTheme() {
    try {
      return localStorage.getItem('theme');
    } catch (e) {
      return null;
    }
  }

  setStoredTheme(theme) {
    try {
      localStorage.setItem('theme', theme);
    } catch (e) {
      console.warn('Could not save theme preference');
    }
  }

  applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    this.currentTheme = theme;
    this.setStoredTheme(theme);
    this.updateThemeIndicator(theme);
  }

  getNextTheme() {
    const currentIndex = this.themes.indexOf(this.currentTheme);
    const nextIndex = (currentIndex + 1) % this.themes.length;
    return this.themes[nextIndex];
  }

  toggleTheme() {
    const nextTheme = this.getNextTheme();
    this.applyTheme(nextTheme);
    this.showThemeIndicator(nextTheme);
  }

  setupEventListeners() {
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
      themeToggle.addEventListener('click', () => this.toggleTheme());
      
      // Keyboard support
      themeToggle.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          this.toggleTheme();
        }
      });
    }

    // Listen for system theme changes
    this.setupSystemThemeListener();
  }

  setupSystemThemeListener() {
    if (window.matchMedia) {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      
      const handleSystemThemeChange = (e) => {
        if (this.currentTheme === 'auto') {
          // Force re-application of auto theme
          this.applyTheme('auto');
        }
      };

      // Modern browsers
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener('change', handleSystemThemeChange);
      } else {
        // Fallback for older browsers
        mediaQuery.addListener(handleSystemThemeChange);
      }
    }
  }

  updateThemeIndicator(theme) {
    // Update any theme indicators in the UI
    const indicators = document.querySelectorAll('[data-theme-indicator]');
    indicators.forEach(indicator => {
      indicator.textContent = this.getThemeDisplayName(theme);
    });
  }

  getThemeDisplayName(theme) {
    const names = {
      light: 'Light',
      dark: 'Dark',
      auto: 'Auto'
    };
    return names[theme] || theme;
  }

  showThemeIndicator(theme) {
    // Create or show theme change indicator
    let indicator = document.querySelector('.theme-indicator');
    
    if (!indicator) {
      indicator = document.createElement('div');
      indicator.className = 'theme-indicator';
      document.body.appendChild(indicator);
    }

    const displayName = this.getThemeDisplayName(theme);
    indicator.textContent = `Theme: ${displayName}`;
    indicator.classList.add('show');

    // Hide after 2 seconds
    setTimeout(() => {
      indicator.classList.remove('show');
    }, 2000);
  }

  // Public API
  setTheme(theme) {
    if (this.themes.includes(theme)) {
      this.applyTheme(theme);
    }
  }

  getCurrentTheme() {
    return this.currentTheme;
  }

  getEffectiveTheme() {
    if (this.currentTheme === 'auto') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return this.currentTheme;
  }
}

// Back to top functionality
class BackToTop {
  constructor() {
    this.button = document.getElementById('back-to-top');
    this.threshold = 300; // Show button after scrolling 300px
    this.init();
  }

  init() {
    if (!this.button) return;

    this.setupEventListeners();
    this.handleScroll(); // Check initial state
  }

  setupEventListeners() {
    // Scroll listener with throttling
    let ticking = false;
    
    const handleScroll = () => {
      if (!ticking) {
        requestAnimationFrame(() => {
          this.handleScroll();
          ticking = false;
        });
        ticking = true;
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });

    // Click listener
    this.button.addEventListener('click', () => this.scrollToTop());

    // Keyboard support
    this.button.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        this.scrollToTop();
      }
    });
  }

  handleScroll() {
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
    
    if (scrollTop > this.threshold) {
      this.button.classList.add('visible');
    } else {
      this.button.classList.remove('visible');
    }
  }

  scrollToTop() {
    // Smooth scroll to top
    window.scrollTo({
      top: 0,
      behavior: 'smooth'
    });

    // Focus management for accessibility
    setTimeout(() => {
      const skipLink = document.querySelector('.skip-link');
      if (skipLink) {
        skipLink.focus();
      }
    }, 500);
  }
}

// Initialize theme system when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  // Initialize theme manager
  window.themeManager = new ThemeManager();
  
  // Initialize back to top
  window.backToTop = new BackToTop();
  
  // Expose theme API globally
  window.setTheme = (theme) => window.themeManager.setTheme(theme);
  window.getCurrentTheme = () => window.themeManager.getCurrentTheme();
  window.getEffectiveTheme = () => window.themeManager.getEffectiveTheme();
});

// Handle theme preference from URL or other sources
document.addEventListener('DOMContentLoaded', () => {
  const urlParams = new URLSearchParams(window.location.search);
  const themeParam = urlParams.get('theme');
  
  if (themeParam && window.themeManager) {
    window.themeManager.setTheme(themeParam);
  }
});

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { ThemeManager, BackToTop };
}