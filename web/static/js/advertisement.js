/**
 * Advertisement Management System - Frontend
 * Handles lazy loading, Core Web Vitals optimization, and ad tracking
 * Requirements: 11, 28
 */

class AdvertisementManager {
    constructor() {
        this.ads = new Map();
        this.observer = null;
        this.performanceMetrics = {
            loadTimes: [],
            layoutShifts: [],
            errors: []
        };
        this.viewabilityThreshold = 0.5; // 50% visibility for viewability
        this.viewabilityDuration = 1000; // 1 second minimum view time
        this.init();
    }

    /**
     * Initialize the advertisement manager
     */
    init() {
        this.setupIntersectionObserver();
        this.setupPerformanceMonitoring();
        this.loadAds();
        
        // Handle page visibility changes for viewability tracking
        document.addEventListener('visibilitychange', () => {
            this.handleVisibilityChange();
        });

        // Handle beforeunload for final tracking
        window.addEventListener('beforeunload', () => {
            this.sendPendingMetrics();
        });
    }

    /**
     * Set up intersection observer for lazy loading and viewability tracking
     */
    setupIntersectionObserver() {
        const lazyLoadOptions = {
            root: null,
            rootMargin: '50px', // Load ads 50px before they come into view
            threshold: 0.1
        };

        this.observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    this.loadAd(entry.target);
                    this.observer.unobserve(entry.target);
                }
            });
        }, lazyLoadOptions);

        // Separate observer for viewability tracking
        const viewabilityOptions = {
            root: null,
            rootMargin: '0px',
            threshold: this.viewabilityThreshold
        };

        this.viewabilityObserver = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                this.handleViewabilityChange(entry);
            });
        }, viewabilityOptions);
    }

    /**
     * Set up performance monitoring for Core Web Vitals
     */
    setupPerformanceMonitoring() {
        // Monitor Cumulative Layout Shift (CLS)
        if ('LayoutShift' in window) {
            new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    if (!entry.hadRecentInput) {
                        this.performanceMetrics.layoutShifts.push({
                            value: entry.value,
                            timestamp: entry.startTime,
                            sources: entry.sources
                        });
                    }
                }
            }).observe({ entryTypes: ['layout-shift'] });
        }

        // Monitor Largest Contentful Paint (LCP)
        if ('LargestContentfulPaint' in window) {
            new PerformanceObserver((list) => {
                const entries = list.getEntries();
                const lastEntry = entries[entries.length - 1];
                this.performanceMetrics.lcp = lastEntry.startTime;
            }).observe({ entryTypes: ['largest-contentful-paint'] });
        }

        // Monitor First Input Delay (FID)
        if ('FirstInputDelay' in window) {
            new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    this.performanceMetrics.fid = entry.processingStart - entry.startTime;
                }
            }).observe({ entryTypes: ['first-input'] });
        }
    }

    /**
     * Load advertisements for the current page
     */
    async loadAds() {
        const adSlots = document.querySelectorAll('[data-ad-slot]');
        
        if (adSlots.length === 0) {
            return;
        }

        // Prepare ad request
        const request = {
            page_type: this.getPageType(),
            position: '',
            category_id: this.getCategoryId(),
            tag_ids: this.getTagIds(),
            device_type: this.getDeviceType(),
            max_ads: adSlots.length,
            user_agent: navigator.userAgent,
            page_url: window.location.href,
            referer: document.referrer
        };

        // Group slots by position for batch requests
        const slotsByPosition = new Map();
        adSlots.forEach(slot => {
            const position = slot.dataset.adPosition || 'content';
            if (!slotsByPosition.has(position)) {
                slotsByPosition.set(position, []);
            }
            slotsByPosition.get(position).push(slot);
        });

        // Load ads for each position
        for (const [position, slots] of slotsByPosition) {
            request.position = position;
            await this.loadAdsForPosition(request, slots);
        }
    }

    /**
     * Load ads for a specific position
     */
    async loadAdsForPosition(request, slots) {
        try {
            const response = await fetch('/api/v1/ads/serve', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(request)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.renderAds(data.ads, slots);
            
        } catch (error) {
            console.error('Failed to load ads:', error);
            this.performanceMetrics.errors.push({
                type: 'load_error',
                message: error.message,
                timestamp: Date.now()
            });
        }
    }

    /**
     * Render advertisements in their slots
     */
    renderAds(ads, slots) {
        ads.forEach((ad, index) => {
            if (index < slots.length) {
                const slot = slots[index];
                this.renderAd(ad, slot);
            }
        });
    }

    /**
     * Render a single advertisement
     */
    renderAd(ad, slot) {
        const startTime = performance.now();
        
        // Store ad data
        this.ads.set(slot, {
            ...ad,
            slot: slot,
            loadStartTime: startTime,
            viewStartTime: null,
            impressionSent: false,
            clickTracked: false
        });

        // Set up the slot with proper dimensions to prevent layout shift
        if (ad.width && ad.height) {
            slot.style.width = `${ad.width}px`;
            slot.style.height = `${ad.height}px`;
            slot.style.display = 'block';
        }

        // Create ad container
        const adContainer = document.createElement('div');
        adContainer.className = 'ad-container';
        adContainer.style.position = 'relative';
        adContainer.style.overflow = 'hidden';

        // Render based on ad type
        switch (ad.type) {
            case 'image':
                this.renderImageAd(ad, adContainer);
                break;
            case 'html':
                this.renderHtmlAd(ad, adContainer);
                break;
            case 'script':
                this.renderScriptAd(ad, adContainer);
                break;
            case 'video':
                this.renderVideoAd(ad, adContainer);
                break;
            default:
                console.warn('Unknown ad type:', ad.type);
                return;
        }

        // Add to slot
        slot.appendChild(adContainer);

        // Set up lazy loading or immediate loading
        if (ad.lazy_load) {
            this.observer.observe(slot);
        } else {
            this.loadAd(slot);
        }

        // Track load time
        const loadTime = performance.now() - startTime;
        this.performanceMetrics.loadTimes.push({
            adId: ad.id,
            loadTime: loadTime,
            timestamp: Date.now()
        });
    }

    /**
     * Render image advertisement
     */
    renderImageAd(ad, container) {
        const img = document.createElement('img');
        img.src = ad.content;
        img.alt = ad.alt_text || 'Advertisement';
        img.style.width = '100%';
        img.style.height = '100%';
        img.style.objectFit = 'cover';
        img.loading = ad.lazy_load ? 'lazy' : 'eager';

        // Add click tracking
        if (ad.click_url) {
            const link = document.createElement('a');
            link.href = ad.click_track_url || ad.click_url;
            link.target = '_blank';
            link.rel = 'noopener nofollow';
            link.appendChild(img);
            container.appendChild(link);

            link.addEventListener('click', (e) => {
                this.trackClick(ad);
            });
        } else {
            container.appendChild(img);
        }

        // Handle image load errors
        img.addEventListener('error', () => {
            this.handleAdError(ad, 'Image failed to load');
        });
    }

    /**
     * Render HTML advertisement
     */
    renderHtmlAd(ad, container) {
        try {
            container.innerHTML = ad.content;
            
            // Add click tracking to all links
            const links = container.querySelectorAll('a');
            links.forEach(link => {
                link.addEventListener('click', () => {
                    this.trackClick(ad);
                });
            });
        } catch (error) {
            this.handleAdError(ad, 'HTML rendering failed');
        }
    }

    /**
     * Render script advertisement
     */
    renderScriptAd(ad, container) {
        try {
            // Create script element
            const script = document.createElement('script');
            script.textContent = ad.content;
            script.async = true;
            
            // Add error handling
            script.onerror = () => {
                this.handleAdError(ad, 'Script execution failed');
            };

            container.appendChild(script);
        } catch (error) {
            this.handleAdError(ad, 'Script creation failed');
        }
    }

    /**
     * Render video advertisement
     */
    renderVideoAd(ad, container) {
        const video = document.createElement('video');
        video.src = ad.content;
        video.controls = true;
        video.style.width = '100%';
        video.style.height = '100%';
        video.preload = ad.lazy_load ? 'metadata' : 'auto';

        // Add click tracking
        video.addEventListener('click', () => {
            this.trackClick(ad);
        });

        // Handle video errors
        video.addEventListener('error', () => {
            this.handleAdError(ad, 'Video failed to load');
        });

        container.appendChild(video);
    }

    /**
     * Load an advertisement (called by intersection observer)
     */
    loadAd(slot) {
        const adData = this.ads.get(slot);
        if (!adData || adData.impressionSent) {
            return;
        }

        // Send impression tracking
        this.trackImpression(adData);
        
        // Start viewability tracking
        this.viewabilityObserver.observe(slot);
    }

    /**
     * Handle viewability changes
     */
    handleViewabilityChange(entry) {
        const slot = entry.target;
        const adData = this.ads.get(slot);
        
        if (!adData) return;

        if (entry.isIntersecting) {
            // Ad became viewable
            if (!adData.viewStartTime) {
                adData.viewStartTime = Date.now();
            }
        } else {
            // Ad became not viewable
            if (adData.viewStartTime) {
                const viewDuration = Date.now() - adData.viewStartTime;
                if (viewDuration >= this.viewabilityDuration) {
                    this.trackViewability(adData, viewDuration);
                }
                adData.viewStartTime = null;
            }
        }
    }

    /**
     * Track advertisement impression
     */
    async trackImpression(adData) {
        if (adData.impressionSent) return;

        try {
            const response = await fetch(adData.tracking_url, {
                method: 'POST',
                headers: {
                    'X-Page-URL': window.location.href
                }
            });

            if (response.ok) {
                adData.impressionSent = true;
            }
        } catch (error) {
            console.error('Failed to track impression:', error);
        }
    }

    /**
     * Track advertisement click
     */
    async trackClick(ad) {
        if (ad.clickTracked) return;

        try {
            // Use sendBeacon for reliable tracking
            const data = new FormData();
            data.append('page_url', window.location.href);
            data.append('timestamp', Date.now().toString());

            navigator.sendBeacon(ad.click_track_url, data);
            ad.clickTracked = true;
        } catch (error) {
            console.error('Failed to track click:', error);
        }
    }

    /**
     * Track viewability
     */
    trackViewability(adData, duration) {
        // Send viewability data
        const data = {
            ad_id: adData.id,
            duration: duration,
            timestamp: Date.now()
        };

        // Use sendBeacon for reliable tracking
        navigator.sendBeacon('/api/v1/ads/viewability', JSON.stringify(data));
    }

    /**
     * Handle advertisement errors
     */
    handleAdError(ad, errorMessage) {
        this.performanceMetrics.errors.push({
            adId: ad.id,
            type: 'render_error',
            message: errorMessage,
            timestamp: Date.now()
        });

        console.error(`Ad error (${ad.id}):`, errorMessage);
    }

    /**
     * Handle page visibility changes
     */
    handleVisibilityChange() {
        if (document.hidden) {
            // Page became hidden - pause viewability tracking
            this.ads.forEach(adData => {
                if (adData.viewStartTime) {
                    const viewDuration = Date.now() - adData.viewStartTime;
                    if (viewDuration >= this.viewabilityDuration) {
                        this.trackViewability(adData, viewDuration);
                    }
                    adData.viewStartTime = null;
                }
            });
        } else {
            // Page became visible - resume viewability tracking
            this.ads.forEach(adData => {
                const slot = adData.slot;
                const rect = slot.getBoundingClientRect();
                const isVisible = rect.top < window.innerHeight && rect.bottom > 0;
                
                if (isVisible && !adData.viewStartTime) {
                    adData.viewStartTime = Date.now();
                }
            });
        }
    }

    /**
     * Send pending metrics before page unload
     */
    sendPendingMetrics() {
        // Send final viewability data
        this.ads.forEach(adData => {
            if (adData.viewStartTime) {
                const viewDuration = Date.now() - adData.viewStartTime;
                if (viewDuration >= this.viewabilityDuration) {
                    this.trackViewability(adData, viewDuration);
                }
            }
        });

        // Send performance metrics
        if (this.performanceMetrics.errors.length > 0 || 
            this.performanceMetrics.layoutShifts.length > 0) {
            
            const data = {
                page_url: window.location.href,
                metrics: this.performanceMetrics,
                timestamp: Date.now()
            };

            navigator.sendBeacon('/api/v1/ads/metrics', JSON.stringify(data));
        }
    }

    /**
     * Get current page type
     */
    getPageType() {
        const path = window.location.pathname;
        
        if (path === '/' || path === '/index.html') {
            return 'homepage';
        } else if (path.includes('/article/') || path.includes('/articles/')) {
            return 'article';
        } else if (path.includes('/category/')) {
            return 'category';
        } else if (path.includes('/tag/')) {
            return 'tag';
        } else if (path.includes('/search')) {
            return 'search';
        }
        
        return 'homepage';
    }

    /**
     * Get category ID from page
     */
    getCategoryId() {
        const meta = document.querySelector('meta[name="category-id"]');
        return meta ? parseInt(meta.content) : null;
    }

    /**
     * Get tag IDs from page
     */
    getTagIds() {
        const meta = document.querySelector('meta[name="tag-ids"]');
        if (!meta) return [];
        
        return meta.content.split(',').map(id => parseInt(id.trim())).filter(id => !isNaN(id));
    }

    /**
     * Get device type
     */
    getDeviceType() {
        const width = window.innerWidth;
        
        if (width <= 768) {
            return 'mobile';
        } else if (width <= 1024) {
            return 'tablet';
        } else {
            return 'desktop';
        }
    }

    /**
     * A/B testing functionality
     */
    getTestVariant(testName) {
        // Simple hash-based A/B testing
        const userId = this.getUserId();
        const hash = this.simpleHash(userId + testName);
        return hash % 2 === 0 ? 'A' : 'B';
    }

    /**
     * Get or generate user ID for A/B testing
     */
    getUserId() {
        let userId = localStorage.getItem('ad_user_id');
        if (!userId) {
            userId = 'user_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem('ad_user_id', userId);
        }
        return userId;
    }

    /**
     * Simple hash function for A/B testing
     */
    simpleHash(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash; // Convert to 32-bit integer
        }
        return Math.abs(hash);
    }

    /**
     * Refresh ads (for ad rotation)
     */
    async refreshAds(slotSelector = null) {
        const slots = slotSelector ? 
            document.querySelectorAll(slotSelector) : 
            document.querySelectorAll('[data-ad-slot]');

        // Clear existing ads
        slots.forEach(slot => {
            if (this.ads.has(slot)) {
                this.viewabilityObserver.unobserve(slot);
                this.ads.delete(slot);
            }
            slot.innerHTML = '';
        });

        // Reload ads
        await this.loadAds();
    }

    /**
     * Get performance metrics
     */
    getPerformanceMetrics() {
        return {
            ...this.performanceMetrics,
            totalAds: this.ads.size,
            loadedAds: Array.from(this.ads.values()).filter(ad => ad.impressionSent).length
        };
    }
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.adManager = new AdvertisementManager();
    });
} else {
    window.adManager = new AdvertisementManager();
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AdvertisementManager;
}
