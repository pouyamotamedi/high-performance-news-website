// Multilingual JavaScript functionality

(function() {
    'use strict';
    
    // Initialize multilingual features when DOM is loaded
    document.addEventListener('DOMContentLoaded', function() {
        initLanguageSwitcher();
        initRTLSupport();
        initAccessibility();
        initLocalStorage();
    });
    
    // Language switcher functionality
    function initLanguageSwitcher() {
        const languageToggle = document.querySelector('.language-toggle');
        const languageDropdown = document.querySelector('.language-dropdown');
        
        if (!languageToggle || !languageDropdown) return;
        
        // Toggle dropdown on click
        languageToggle.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            
            const isExpanded = languageToggle.getAttribute('aria-expanded') === 'true';
            languageToggle.setAttribute('aria-expanded', !isExpanded);
            languageDropdown.style.display = isExpanded ? 'none' : 'block';
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', function(e) {
            if (!languageToggle.contains(e.target) && !languageDropdown.contains(e.target)) {
                languageToggle.setAttribute('aria-expanded', 'false');
                languageDropdown.style.display = 'none';
            }
        });
        
        // Handle keyboard navigation
        languageToggle.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                languageToggle.click();
            } else if (e.key === 'Escape') {
                languageToggle.setAttribute('aria-expanded', 'false');
                languageDropdown.style.display = 'none';
                languageToggle.focus();
            }
        });
        
        // Handle language option selection
        const languageOptions = languageDropdown.querySelectorAll('.language-option');
        languageOptions.forEach(function(option, index) {
            option.addEventListener('keydown', function(e) {
                if (e.key === 'Enter') {
                    window.location.href = option.href;
                } else if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    const nextOption = languageOptions[index + 1] || languageOptions[0];
                    nextOption.focus();
                } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    const prevOption = languageOptions[index - 1] || languageOptions[languageOptions.length - 1];
                    prevOption.focus();
                } else if (e.key === 'Escape') {
                    languageToggle.setAttribute('aria-expanded', 'false');
                    languageDropdown.style.display = 'none';
                    languageToggle.focus();
                }
            });
        });
    }
    
    // RTL/LTR support enhancements
    function initRTLSupport() {
        const htmlElement = document.documentElement;
        const direction = htmlElement.getAttribute('dir');
        const languageCode = htmlElement.getAttribute('lang');
        
        // Add CSS classes for enhanced styling
        document.body.classList.add('dir-' + direction);
        document.body.classList.add('lang-' + languageCode);
        
        // Handle form inputs for RTL languages
        if (direction === 'rtl') {
            const inputs = document.querySelectorAll('input[type="text"], input[type="email"], input[type="search"], textarea');
            inputs.forEach(function(input) {
                if (!input.hasAttribute('dir')) {
                    input.setAttribute('dir', 'rtl');
                }
            });
        }
        
        // Handle number formatting for different locales
        formatNumbers(languageCode);
    }
    
    // Format numbers according to language locale
    function formatNumbers(languageCode) {
        const numberElements = document.querySelectorAll('[data-number]');
        
        numberElements.forEach(function(element) {
            const number = parseFloat(element.dataset.number);
            if (!isNaN(number)) {
                let formattedNumber;
                
                try {
                    // Use Intl.NumberFormat for proper localization
                    const formatter = new Intl.NumberFormat(getLocaleFromLanguageCode(languageCode));
                    formattedNumber = formatter.format(number);
                } catch (e) {
                    // Fallback to basic formatting
                    formattedNumber = number.toLocaleString();
                }
                
                element.textContent = formattedNumber;
            }
        });
    }
    
    // Get proper locale from language code
    function getLocaleFromLanguageCode(languageCode) {
        const localeMap = {
            'fa': 'fa-IR',
            'ar': 'ar-SA',
            'en': 'en-US'
        };
        
        return localeMap[languageCode] || languageCode;
    }
    
    // Initialize accessibility features
    function initAccessibility() {
        // Add skip link functionality
        const skipLink = document.querySelector('.skip-link');
        if (skipLink) {
            skipLink.addEventListener('click', function(e) {
                e.preventDefault();
                const target = document.querySelector('#main-content');
                if (target) {
                    target.focus();
                    target.scrollIntoView();
                }
            });
        }
        
        // Enhance keyboard navigation for articles
        const articleLinks = document.querySelectorAll('.article-card-title a');
        articleLinks.forEach(function(link) {
            link.addEventListener('keydown', function(e) {
                if (e.key === 'Enter') {
                    window.location.href = link.href;
                }
            });
        });
        
        // Add ARIA labels for better screen reader support
        enhanceARIALabels();
    }
    
    // Enhance ARIA labels based on language
    function enhanceARIALabels() {
        const languageCode = document.documentElement.getAttribute('lang');
        
        const ariaLabels = {
            'fa': {
                'read-more': 'ادامه مطلب',
                'close': 'بستن',
                'menu': 'منو',
                'search': 'جستجو',
                'loading': 'در حال بارگذاری...'
            },
            'ar': {
                'read-more': 'اقرأ المزيد',
                'close': 'إغلاق',
                'menu': 'القائمة',
                'search': 'بحث',
                'loading': 'جاري التحميل...'
            },
            'en': {
                'read-more': 'Read more',
                'close': 'Close',
                'menu': 'Menu',
                'search': 'Search',
                'loading': 'Loading...'
            }
        };
        
        const labels = ariaLabels[languageCode] || ariaLabels['en'];
        
        // Apply labels to elements
        document.querySelectorAll('[data-aria-label]').forEach(function(element) {
            const labelKey = element.dataset.ariaLabel;
            if (labels[labelKey]) {
                element.setAttribute('aria-label', labels[labelKey]);
            }
        });
    }
    
    // Initialize local storage for language preferences
    function initLocalStorage() {
        const languageCode = document.documentElement.getAttribute('lang');
        
        // Save current language preference
        try {
            localStorage.setItem('preferred-language', languageCode);
        } catch (e) {
            // Local storage not available, ignore
        }
        
        // Handle language switching with preference saving
        const languageOptions = document.querySelectorAll('.language-option');
        languageOptions.forEach(function(option) {
            option.addEventListener('click', function() {
                const selectedLang = option.getAttribute('hreflang');
                try {
                    localStorage.setItem('preferred-language', selectedLang);
                } catch (e) {
                    // Local storage not available, ignore
                }
            });
        });
    }
    
    // Utility function to get current language
    function getCurrentLanguage() {
        return document.documentElement.getAttribute('lang') || 'fa';
    }
    
    // Utility function to check if current language is RTL
    function isRTL() {
        return document.documentElement.getAttribute('dir') === 'rtl';
    }
    
    // Format dates according to language locale
    function formatDate(date, languageCode) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        
        try {
            const locale = getLocaleFromLanguageCode(languageCode);
            return date.toLocaleDateString(locale, {
                year: 'numeric',
                month: 'long',
                day: 'numeric'
            });
        } catch (e) {
            return date.toLocaleDateString();
        }
    }
    
    // Format relative time (e.g., "2 hours ago")
    function formatRelativeTime(date, languageCode) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        
        const now = new Date();
        const diffInSeconds = Math.floor((now - date) / 1000);
        
        const timeUnits = {
            'fa': {
                'second': 'ثانیه',
                'minute': 'دقیقه',
                'hour': 'ساعت',
                'day': 'روز',
                'week': 'هفته',
                'month': 'ماه',
                'year': 'سال',
                'ago': 'پیش'
            },
            'ar': {
                'second': 'ثانية',
                'minute': 'دقيقة',
                'hour': 'ساعة',
                'day': 'يوم',
                'week': 'أسبوع',
                'month': 'شهر',
                'year': 'سنة',
                'ago': 'منذ'
            },
            'en': {
                'second': 'second',
                'minute': 'minute',
                'hour': 'hour',
                'day': 'day',
                'week': 'week',
                'month': 'month',
                'year': 'year',
                'ago': 'ago'
            }
        };
        
        const units = timeUnits[languageCode] || timeUnits['en'];
        
        if (diffInSeconds < 60) {
            return `${diffInSeconds} ${units.second}${diffInSeconds !== 1 ? 's' : ''} ${units.ago}`;
        } else if (diffInSeconds < 3600) {
            const minutes = Math.floor(diffInSeconds / 60);
            return `${minutes} ${units.minute}${minutes !== 1 ? 's' : ''} ${units.ago}`;
        } else if (diffInSeconds < 86400) {
            const hours = Math.floor(diffInSeconds / 3600);
            return `${hours} ${units.hour}${hours !== 1 ? 's' : ''} ${units.ago}`;
        } else if (diffInSeconds < 604800) {
            const days = Math.floor(diffInSeconds / 86400);
            return `${days} ${units.day}${days !== 1 ? 's' : ''} ${units.ago}`;
        } else {
            return formatDate(date, languageCode);
        }
    }
    
    // Apply relative time formatting to elements
    document.addEventListener('DOMContentLoaded', function() {
        const timeElements = document.querySelectorAll('[data-time]');
        const languageCode = getCurrentLanguage();
        
        timeElements.forEach(function(element) {
            const timestamp = element.dataset.time;
            const date = new Date(timestamp);
            
            if (!isNaN(date.getTime())) {
                element.textContent = formatRelativeTime(date, languageCode);
                element.setAttribute('title', formatDate(date, languageCode));
            }
        });
    });
    
    // Export utility functions for use in other scripts
    window.MultilingualUtils = {
        getCurrentLanguage: getCurrentLanguage,
        isRTL: isRTL,
        formatDate: formatDate,
        formatRelativeTime: formatRelativeTime,
        getLocaleFromLanguageCode: getLocaleFromLanguageCode
    };
    
})();