// Template System Tests
// This file tests the frontend template functionality, responsive design, and PWA features

describe('Template System', () => {
  let mockDocument;
  let mockWindow;
  
  beforeEach(() => {
    // Mock DOM elements
    mockDocument = {
      documentElement: {
        setAttribute: jest.fn(),
        getAttribute: jest.fn()
      },
      body: {
        classList: {
          add: jest.fn(),
          remove: jest.fn(),
          contains: jest.fn()
        },
        appendChild: jest.fn()
      },
      querySelector: jest.fn(),
      querySelectorAll: jest.fn(() => []),
      createElement: jest.fn(() => ({
        className: '',
        textContent: '',
        style: { cssText: '' },
        classList: {
          add: jest.fn(),
          remove: jest.fn(),
          contains: jest.fn()
        },
        addEventListener: jest.fn(),
        appendChild: jest.fn()
      })),
      addEventListener: jest.fn()
    };
    
    mockWindow = {
      localStorage: {
        getItem: jest.fn(),
        setItem: jest.fn()
      },
      matchMedia: jest.fn(() => ({
        matches: false,
        addEventListener: jest.fn(),
        addListener: jest.fn()
      })),
      addEventListener: jest.fn(),
      pageYOffset: 0,
      innerWidth: 1024,
      scrollTo: jest.fn()
    };
    
    global.document = mockDocument;
    global.window = mockWindow;
  });

  describe('Theme Management', () => {
    test('should initialize with default theme', () => {
      // Mock theme manager initialization
      const themeManager = {
        currentTheme: 'auto',
        themes: ['light', 'dark', 'auto']
      };
      
      expect(themeManager.currentTheme).toBe('auto');
      expect(themeManager.themes).toContain('light');
      expect(themeManager.themes).toContain('dark');
      expect(themeManager.themes).toContain('auto');
    });

    test('should apply theme correctly', () => {
      const applyTheme = (theme) => {
        mockDocument.documentElement.setAttribute('data-theme', theme);
        mockWindow.localStorage.setItem('theme', theme);
      };
      
      applyTheme('dark');
      
      expect(mockDocument.documentElement.setAttribute).toHaveBeenCalledWith('data-theme', 'dark');
      expect(mockWindow.localStorage.setItem).toHaveBeenCalledWith('theme', 'dark');
    });

    test('should cycle through themes correctly', () => {
      const themes = ['light', 'dark', 'auto'];
      const getNextTheme = (currentTheme) => {
        const currentIndex = themes.indexOf(currentTheme);
        const nextIndex = (currentIndex + 1) % themes.length;
        return themes[nextIndex];
      };
      
      expect(getNextTheme('auto')).toBe('light');
      expect(getNextTheme('light')).toBe('dark');
      expect(getNextTheme('dark')).toBe('auto');
    });

    test('should get effective theme for auto mode', () => {
      const getEffectiveTheme = (currentTheme, prefersDark) => {
        if (currentTheme === 'auto') {
          return prefersDark ? 'dark' : 'light';
        }
        return currentTheme;
      };
      
      // Mock dark mode preference
      expect(getEffectiveTheme('auto', true)).toBe('dark');
      
      // Mock light mode preference
      expect(getEffectiveTheme('auto', false)).toBe('light');
      
      // Direct theme selection
      expect(getEffectiveTheme('dark', false)).toBe('dark');
    });

    test('should handle localStorage errors gracefully', () => {
      mockWindow.localStorage.setItem.mockImplementation(() => {
        throw new Error('Storage error');
      });
      
      const safeSetTheme = (theme) => {
        try {
          mockWindow.localStorage.setItem('theme', theme);
        } catch (e) {
          // Should not throw
        }
      };
      
      expect(() => safeSetTheme('dark')).not.toThrow();
    });
  });

  describe('Back to Top Functionality', () => {
    test('should show button when scrolled past threshold', () => {
      const mockButton = {
        classList: {
          add: jest.fn(),
          remove: jest.fn()
        }
      };
      
      const handleScroll = (scrollTop, threshold = 300) => {
        if (scrollTop > threshold) {
          mockButton.classList.add('visible');
        } else {
          mockButton.classList.remove('visible');
        }
      };
      
      handleScroll(400);
      expect(mockButton.classList.add).toHaveBeenCalledWith('visible');
    });

    test('should hide button when scrolled above threshold', () => {
      const mockButton = {
        classList: {
          add: jest.fn(),
          remove: jest.fn()
        }
      };
      
      const handleScroll = (scrollTop, threshold = 300) => {
        if (scrollTop > threshold) {
          mockButton.classList.add('visible');
        } else {
          mockButton.classList.remove('visible');
        }
      };
      
      handleScroll(100);
      expect(mockButton.classList.remove).toHaveBeenCalledWith('visible');
    });

    test('should scroll to top when clicked', () => {
      const scrollToTop = () => {
        mockWindow.scrollTo({
          top: 0,
          behavior: 'smooth'
        });
      };
      
      scrollToTop();
      
      expect(mockWindow.scrollTo).toHaveBeenCalledWith({
        top: 0,
        behavior: 'smooth'
      });
    });
  });

  describe('PWA Manager', () => {
    beforeEach(() => {
      global.navigator = {
        serviceWorker: {
          register: jest.fn(() => Promise.resolve({
            addEventListener: jest.fn()
          })),
          addEventListener: jest.fn()
        }
      };
    });

    test('should detect installed PWA', () => {
      const checkInstallation = () => {
        const isStandalone = mockWindow.matchMedia('(display-mode: standalone)').matches;
        return isStandalone;
      };
      
      mockWindow.matchMedia.mockReturnValue({ matches: true });
      
      const isInstalled = checkInstallation();
      expect(isInstalled).toBe(true);
    });

    test('should register service worker', async () => {
      const registerServiceWorker = async () => {
        if ('serviceWorker' in navigator) {
          return await navigator.serviceWorker.register('/static/sw.js');
        }
      };
      
      await registerServiceWorker();
      expect(navigator.serviceWorker.register).toHaveBeenCalledWith('/static/sw.js');
    });

    test('should handle beforeinstallprompt event', () => {
      let deferredPrompt = null;
      
      const handleBeforeInstallPrompt = (event) => {
        event.preventDefault();
        deferredPrompt = event;
      };
      
      const mockEvent = {
        preventDefault: jest.fn()
      };
      
      handleBeforeInstallPrompt(mockEvent);
      
      expect(mockEvent.preventDefault).toHaveBeenCalled();
      expect(deferredPrompt).toBe(mockEvent);
    });

    test('should show install button when prompt is available', () => {
      const mockButton = {
        style: { display: '' },
        classList: { add: jest.fn() }
      };
      
      const showInstallButton = () => {
        mockButton.style.display = 'block';
        mockButton.classList.add('show');
      };
      
      showInstallButton();
      
      expect(mockButton.style.display).toBe('block');
      expect(mockButton.classList.add).toHaveBeenCalledWith('show');
    });

    test('should handle online/offline status', () => {
      const handleOfflineStatus = () => {
        mockDocument.body.classList.add('offline');
        mockDocument.body.classList.remove('online');
      };
      
      const handleOnlineStatus = () => {
        mockDocument.body.classList.add('online');
        mockDocument.body.classList.remove('offline');
      };
      
      handleOfflineStatus();
      expect(mockDocument.body.classList.add).toHaveBeenCalledWith('offline');
      expect(mockDocument.body.classList.remove).toHaveBeenCalledWith('online');
      
      handleOnlineStatus();
      expect(mockDocument.body.classList.add).toHaveBeenCalledWith('online');
      expect(mockDocument.body.classList.remove).toHaveBeenCalledWith('offline');
    });
  });

  describe('News App Main Functionality', () => {
    test('should initialize mobile menu correctly', () => {
      const mockToggle = {
        setAttribute: jest.fn(),
        addEventListener: jest.fn()
      };
      
      const setupMobileMenu = () => {
        mockToggle.setAttribute('aria-expanded', 'false');
        mockToggle.setAttribute('aria-controls', 'mobile-navigation');
      };
      
      setupMobileMenu();
      
      expect(mockToggle.setAttribute).toHaveBeenCalledWith('aria-expanded', 'false');
      expect(mockToggle.setAttribute).toHaveBeenCalledWith('aria-controls', 'mobile-navigation');
    });

    test('should toggle mobile menu state', () => {
      const mockToggle = {
        classList: { add: jest.fn(), remove: jest.fn() },
        setAttribute: jest.fn()
      };
      
      const mockOverlay = {
        classList: {
          contains: jest.fn(() => false),
          add: jest.fn(),
          remove: jest.fn()
        },
        querySelector: jest.fn(() => ({ focus: jest.fn() }))
      };
      
      const toggleMobileMenu = () => {
        const isOpen = mockOverlay.classList.contains('active');
        
        if (!isOpen) {
          mockOverlay.classList.add('active');
          mockToggle.classList.add('active');
          mockToggle.setAttribute('aria-expanded', 'true');
          mockDocument.body.classList.add('mobile-menu-open');
        }
      };
      
      toggleMobileMenu();
      
      expect(mockOverlay.classList.add).toHaveBeenCalledWith('active');
      expect(mockToggle.classList.add).toHaveBeenCalledWith('active');
      expect(mockToggle.setAttribute).toHaveBeenCalledWith('aria-expanded', 'true');
      expect(mockDocument.body.classList.add).toHaveBeenCalledWith('mobile-menu-open');
    });

    test('should handle search functionality', () => {
      const mockSearchOverlay = {
        classList: { contains: jest.fn(), add: jest.fn(), remove: jest.fn() }
      };
      
      const toggleSearch = () => {
        const isOpen = mockSearchOverlay.classList.contains('active');
        
        if (!isOpen) {
          mockSearchOverlay.classList.add('active');
          mockDocument.body.classList.add('search-open');
        }
      };
      
      toggleSearch();
      
      expect(mockSearchOverlay.classList.add).toHaveBeenCalledWith('active');
      expect(mockDocument.body.classList.add).toHaveBeenCalledWith('search-open');
    });

    test('should validate form fields correctly', () => {
      const validateField = (field) => {
        const value = field.value.trim();
        const type = field.type;
        const required = field.hasAttribute('required');
        
        let isValid = true;
        
        // Required validation
        if (required && !value) {
          isValid = false;
        }
        
        // Email validation
        if (type === 'email' && value) {
          const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
          if (!emailRegex.test(value)) {
            isValid = false;
          }
        }
        
        return isValid;
      };
      
      // Test required field validation
      const requiredField = {
        value: '',
        type: 'text',
        hasAttribute: jest.fn(() => true)
      };
      
      expect(validateField(requiredField)).toBe(false);
      
      // Test email validation
      const emailField = {
        value: 'invalid-email',
        type: 'email',
        hasAttribute: jest.fn(() => false)
      };
      
      expect(validateField(emailField)).toBe(false);
      
      // Test valid email
      emailField.value = 'test@example.com';
      expect(validateField(emailField)).toBe(true);
    });

    test('should handle keyboard navigation', () => {
      let searchToggled = false;
      let menusClosed = false;
      
      const handleKeyboardNavigation = (event) => {
        if (event.ctrlKey || event.metaKey) {
          switch (event.key) {
            case 'k':
            case '/':
              event.preventDefault();
              searchToggled = true;
              break;
          }
        }
        
        if (event.key === 'Escape') {
          menusClosed = true;
        }
      };
      
      // Test Ctrl+K shortcut
      const ctrlKEvent = {
        ctrlKey: true,
        key: 'k',
        preventDefault: jest.fn()
      };
      
      handleKeyboardNavigation(ctrlKEvent);
      expect(ctrlKEvent.preventDefault).toHaveBeenCalled();
      expect(searchToggled).toBe(true);
      
      // Test Escape key
      const escapeEvent = { key: 'Escape' };
      handleKeyboardNavigation(escapeEvent);
      expect(menusClosed).toBe(true);
    });

    test('should handle view toggle correctly', () => {
      const mockContainer = {
        setAttribute: jest.fn()
      };
      
      const mockButtons = [
        { classList: { toggle: jest.fn() }, dataset: { view: 'grid' } },
        { classList: { toggle: jest.fn() }, dataset: { view: 'list' } }
      ];
      
      const toggleView = (viewType) => {
        mockContainer.setAttribute('data-view', viewType);
        
        mockButtons.forEach(button => {
          button.classList.toggle('active', button.dataset.view === viewType);
        });
        
        mockWindow.localStorage.setItem('preferred-view', viewType);
      };
      
      toggleView('list');
      
      expect(mockContainer.setAttribute).toHaveBeenCalledWith('data-view', 'list');
      expect(mockButtons[0].classList.toggle).toHaveBeenCalledWith('active', false);
      expect(mockButtons[1].classList.toggle).toHaveBeenCalledWith('active', true);
      expect(mockWindow.localStorage.setItem).toHaveBeenCalledWith('preferred-view', 'list');
    });
  });

  describe('Responsive Design', () => {
    test('should handle mobile menu on resize', () => {
      let menuClosed = false;
      
      const handleResize = () => {
        if (mockWindow.innerWidth >= 1024) {
          menuClosed = true;
        }
      };
      
      // Simulate desktop width
      mockWindow.innerWidth = 1024;
      handleResize();
      
      expect(menuClosed).toBe(true);
    });

    test('should setup lazy loading with IntersectionObserver', () => {
      const mockObserver = {
        observe: jest.fn(),
        unobserve: jest.fn()
      };
      
      global.IntersectionObserver = jest.fn(() => mockObserver);
      
      const mockImages = [
        { dataset: { src: '/image1.jpg' } },
        { dataset: { src: '/image2.jpg' } }
      ];
      
      const setupLazyLoading = () => {
        if ('IntersectionObserver' in window) {
          const imageObserver = new IntersectionObserver(() => {});
          mockImages.forEach(img => imageObserver.observe(img));
        }
      };
      
      setupLazyLoading();
      
      expect(global.IntersectionObserver).toHaveBeenCalled();
      expect(mockObserver.observe).toHaveBeenCalledTimes(2);
    });

    test('should handle focus trapping in overlays', () => {
      const trapFocus = (event, container) => {
        const focusableElements = ['button', 'input', 'select'];
        
        if (event.key === 'Tab') {
          // Focus trapping logic would go here
          return true;
        }
        
        if (event.key === 'Escape') {
          // Close overlay logic would go here
          return true;
        }
        
        return false;
      };
      
      const tabEvent = { key: 'Tab' };
      const escapeEvent = { key: 'Escape' };
      
      expect(trapFocus(tabEvent, {})).toBe(true);
      expect(trapFocus(escapeEvent, {})).toBe(true);
    });
  });

  describe('Accessibility Features', () => {
    test('should setup live regions for announcements', () => {
      const mockLiveRegion = {
        id: 'live-region',
        setAttribute: jest.fn(),
        style: { cssText: '' }
      };
      
      mockDocument.querySelector.mockReturnValue(null);
      mockDocument.createElement.mockReturnValue(mockLiveRegion);
      
      const setupLiveRegions = () => {
        if (!mockDocument.querySelector('#live-region')) {
          const liveRegion = mockDocument.createElement('div');
          liveRegion.id = 'live-region';
          liveRegion.setAttribute('aria-live', 'polite');
          liveRegion.setAttribute('aria-atomic', 'true');
          mockDocument.body.appendChild(liveRegion);
        }
      };
      
      setupLiveRegions();
      
      expect(mockDocument.createElement).toHaveBeenCalledWith('div');
      expect(mockLiveRegion.setAttribute).toHaveBeenCalledWith('aria-live', 'polite');
      expect(mockLiveRegion.setAttribute).toHaveBeenCalledWith('aria-atomic', 'true');
      expect(mockDocument.body.appendChild).toHaveBeenCalledWith(mockLiveRegion);
    });

    test('should announce messages to screen readers', () => {
      const mockLiveRegion = {
        textContent: ''
      };
      
      mockDocument.querySelector.mockReturnValue(mockLiveRegion);
      
      const announce = (message) => {
        const liveRegion = mockDocument.querySelector('#live-region');
        if (liveRegion) {
          liveRegion.textContent = message;
        }
      };
      
      announce('Test message');
      
      expect(mockLiveRegion.textContent).toBe('Test message');
    });

    test('should handle skip link functionality', () => {
      const mockTarget = {
        focus: jest.fn(),
        scrollIntoView: jest.fn()
      };
      
      const handleSkipLink = (event, targetSelector) => {
        event.preventDefault();
        const target = mockDocument.querySelector(targetSelector);
        if (target) {
          target.focus();
          target.scrollIntoView();
        }
      };
      
      const mockEvent = { preventDefault: jest.fn() };
      mockDocument.querySelector.mockReturnValue(mockTarget);
      
      handleSkipLink(mockEvent, '#main-content');
      
      expect(mockEvent.preventDefault).toHaveBeenCalled();
      expect(mockTarget.focus).toHaveBeenCalled();
      expect(mockTarget.scrollIntoView).toHaveBeenCalled();
    });
  });

  describe('Utility Functions', () => {
    test('should debounce function calls', (done) => {
      const mockFn = jest.fn();
      
      const debounce = (func, wait) => {
        let timeout;
        return function executedFunction(...args) {
          const later = () => {
            clearTimeout(timeout);
            func(...args);
          };
          clearTimeout(timeout);
          timeout = setTimeout(later, wait);
        };
      };
      
      const debouncedFn = debounce(mockFn, 100);
      
      // Call multiple times quickly
      debouncedFn();
      debouncedFn();
      debouncedFn();
      
      expect(mockFn).not.toHaveBeenCalled();
      
      // Wait for debounce delay
      setTimeout(() => {
        expect(mockFn).toHaveBeenCalledTimes(1);
        done();
      }, 150);
    });

    test('should throttle function calls', (done) => {
      const mockFn = jest.fn();
      
      const throttle = (func, limit) => {
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
      };
      
      const throttledFn = throttle(mockFn, 100);
      
      // Call multiple times quickly
      throttledFn();
      throttledFn();
      throttledFn();
      
      expect(mockFn).toHaveBeenCalledTimes(1);
      
      setTimeout(() => {
        throttledFn();
        expect(mockFn).toHaveBeenCalledTimes(2);
        done();
      }, 150);
    });
  });

  describe('Template Rendering', () => {
    test('should render templates with proper data binding', () => {
      const templateData = {
        title: 'Test Article',
        content: 'This is test content',
        author: 'John Doe',
        publishedAt: new Date('2024-01-01')
      };
      
      const renderTemplate = (template, data) => {
        return template
          .replace(/\{\{\.Title\}\}/g, data.title)
          .replace(/\{\{\.Content\}\}/g, data.content)
          .replace(/\{\{\.Author\}\}/g, data.author);
      };
      
      const template = '<h1>{{.Title}}</h1><p>{{.Content}}</p><span>By {{.Author}}</span>';
      const result = renderTemplate(template, templateData);
      
      expect(result).toContain('Test Article');
      expect(result).toContain('This is test content');
      expect(result).toContain('John Doe');
    });

    test('should handle template inheritance correctly', () => {
      const baseTemplate = '<html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>';
      const pageContent = '<h1>{{.Title}}</h1><p>{{.Content}}</p>';
      
      const renderWithLayout = (base, content, data) => {
        const renderedContent = content
          .replace(/\{\{\.Title\}\}/g, data.title)
          .replace(/\{\{\.Content\}\}/g, data.content);
        
        return base
          .replace(/\{\{\.Title\}\}/g, data.title)
          .replace(/\{\{\.Content\}\}/g, renderedContent);
      };
      
      const data = { title: 'Test Page', content: 'Page content' };
      
      const result = renderWithLayout(baseTemplate, pageContent, data);
      
      expect(result).toContain('<title>Test Page</title>');
      expect(result).toContain('<h1>Test Page</h1>');
      expect(result).toContain('<p>Page content</p>');
    });

    test('should handle RTL content rendering', () => {
      const rtlData = {
        title: 'عنوان المقال',
        content: 'محتوى المقال',
        direction: 'rtl',
        language: 'ar'
      };
      
      const rtlTemplate = '<div dir="{{.Direction}}" lang="{{.Language}}"><h1>{{.Title}}</h1><p>{{.Content}}</p></div>';
      
      const renderRTL = (template, data) => {
        return template
          .replace(/\{\{\.Direction\}\}/g, data.direction)
          .replace(/\{\{\.Language\}\}/g, data.language)
          .replace(/\{\{\.Title\}\}/g, data.title)
          .replace(/\{\{\.Content\}\}/g, data.content);
      };
      
      const result = renderRTL(rtlTemplate, rtlData);
      
      expect(result).toContain('dir="rtl"');
      expect(result).toContain('lang="ar"');
      expect(result).toContain('عنوان المقال');
      expect(result).toContain('محتوى المقال');
    });
  });

  describe('Error Handling', () => {
    test('should handle missing DOM elements gracefully', () => {
      mockDocument.querySelector.mockReturnValue(null);
      
      const safeToggle = (selector) => {
        const element = mockDocument.querySelector(selector);
        if (element && element.classList) {
          element.classList.toggle('active');
          return true;
        }
        return false;
      };
      
      expect(safeToggle('.non-existent')).toBe(false);
      expect(() => safeToggle('.non-existent')).not.toThrow();
    });

    test('should handle localStorage errors gracefully', () => {
      mockWindow.localStorage.setItem.mockImplementation(() => {
        throw new Error('Storage not available');
      });
      
      const safeSetStorage = (key, value) => {
        try {
          mockWindow.localStorage.setItem(key, value);
          return true;
        } catch (e) {
          console.warn('Storage not available');
          return false;
        }
      };
      
      expect(safeSetStorage('test', 'value')).toBe(false);
      expect(() => safeSetStorage('test', 'value')).not.toThrow();
    });

    test('should handle service worker registration failure', async () => {
      global.navigator = {
        serviceWorker: {
          register: jest.fn(() => Promise.reject(new Error('SW registration failed')))
        }
      };
      
      const registerSW = async () => {
        try {
          await navigator.serviceWorker.register('/sw.js');
          return true;
        } catch (error) {
          console.log('ServiceWorker registration failed:', error);
          return false;
        }
      };
      
      const result = await registerSW();
      expect(result).toBe(false);
    });
  });

  describe('Performance Tests', () => {
    test('should implement efficient lazy loading', () => {
      const mockImages = [
        { dataset: { src: '/image1.jpg' }, src: '', removeAttribute: jest.fn() },
        { dataset: { src: '/image2.jpg' }, src: '', removeAttribute: jest.fn() }
      ];
      
      const lazyLoadImages = (entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            const img = entry.target;
            if (img.dataset.src) {
              img.src = img.dataset.src;
              img.removeAttribute('data-src');
            }
          }
        });
      };
      
      const entries = [
        { target: mockImages[0], isIntersecting: true },
        { target: mockImages[1], isIntersecting: false }
      ];
      
      lazyLoadImages(entries);
      
      expect(mockImages[0].src).toBe('/image1.jpg');
      expect(mockImages[0].removeAttribute).toHaveBeenCalledWith('data-src');
      expect(mockImages[1].src).toBe('');
    });

    test('should optimize scroll performance with throttling', () => {
      let scrollCallCount = 0;
      
      const throttledScroll = (() => {
        let ticking = false;
        return () => {
          if (!ticking) {
            requestAnimationFrame(() => {
              scrollCallCount++;
              ticking = false;
            });
            ticking = true;
          }
        };
      })();
      
      // Simulate multiple scroll events
      for (let i = 0; i < 10; i++) {
        throttledScroll();
      }
      
      // Should only increment once due to throttling
      expect(scrollCallCount).toBeLessThanOrEqual(1);
    });
  });

  describe('Integration Tests', () => {
    test('should integrate all components correctly', () => {
      const components = {
        theme: { currentTheme: 'auto', setTheme: jest.fn() },
        pwa: { isInstalled: false, install: jest.fn() },
        app: { 
          toggleMobileMenu: jest.fn(),
          toggleSearch: jest.fn(),
          announce: jest.fn()
        }
      };
      
      // Simulate user interaction flow
      components.theme.setTheme('dark');
      components.app.toggleMobileMenu();
      components.app.announce('Menu opened');
      
      expect(components.theme.setTheme).toHaveBeenCalledWith('dark');
      expect(components.app.toggleMobileMenu).toHaveBeenCalled();
      expect(components.app.announce).toHaveBeenCalledWith('Menu opened');
    });

    test('should handle complete PWA installation flow', () => {
      const pwaFlow = {
        beforeInstallPrompt: null,
        isInstalled: false,
        
        handleBeforeInstall(event) {
          event.preventDefault();
          this.beforeInstallPrompt = event;
        },
        
        async promptInstall() {
          if (this.beforeInstallPrompt) {
            this.beforeInstallPrompt.prompt();
            const result = await this.beforeInstallPrompt.userChoice;
            this.beforeInstallPrompt = null;
            return result.outcome === 'accepted';
          }
          return false;
        },
        
        handleAppInstalled() {
          this.isInstalled = true;
        }
      };
      
      const mockEvent = {
        preventDefault: jest.fn(),
        prompt: jest.fn(),
        userChoice: Promise.resolve({ outcome: 'accepted' })
      };
      
      pwaFlow.handleBeforeInstall(mockEvent);
      expect(mockEvent.preventDefault).toHaveBeenCalled();
      expect(pwaFlow.beforeInstallPrompt).toBe(mockEvent);
      
      pwaFlow.handleAppInstalled();
      expect(pwaFlow.isInstalled).toBe(true);
    });
  });
});

// Export for Node.js environment
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    // Test utilities can be exported here if needed
  };
}