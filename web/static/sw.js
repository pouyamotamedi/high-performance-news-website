/**
 * Service Worker for News Website PWA
 * Handles offline caching, push notifications, and background sync
 * Version: 2.0.0
 */

const CACHE_VERSION = 'v2.0.0';
const STATIC_CACHE = `static-${CACHE_VERSION}`;
const DYNAMIC_CACHE = `dynamic-${CACHE_VERSION}`;
const ARTICLE_CACHE = `articles-${CACHE_VERSION}`;
const IMAGE_CACHE = `images-${CACHE_VERSION}`;

// Cache size limits (in bytes)
const MAX_CACHE_SIZE = 50 * 1024 * 1024; // 50MB total
const MAX_ARTICLE_CACHE = 20 * 1024 * 1024; // 20MB for articles
const MAX_IMAGE_CACHE = 20 * 1024 * 1024; // 20MB for images

// Static assets to cache on install
const STATIC_ASSETS = [
    '/',
    '/offline',
    '/static/css/main.css',
    '/static/css/themes.css',
    '/static/css/dark-mode-fixes.css',
    '/static/css/homepage.css',
    '/static/css/article.css',
    '/static/css/professional-news.css',
    '/static/js/main.js',
    '/static/js/theme.js',
    '/static/js/pwa.js',
    '/static/manifest.json',
    '/static/favicon.ico',
    '/static/icons/favicon-32x32.png',
    '/static/icons/favicon-16x16.png',
    '/static/icons/apple-touch-icon.png',
    '/static/icons/phosphor/newspaper.svg',
    '/static/icons/phosphor/sun.svg',
    '/static/icons/phosphor/moon.svg',
    '/static/icons/phosphor/circle-half.svg'
];

// API endpoints that should use network-first
const API_PATTERNS = [
    /^\/api\//,
    /^\/admin\//,
    /^\/newsletter\//
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
    console.log('[SW] Installing service worker...');
    event.waitUntil(
        caches.open(STATIC_CACHE)
            .then((cache) => {
                console.log('[SW] Caching static assets');
                return cache.addAll(STATIC_ASSETS.map(url => {
                    return new Request(url, { cache: 'reload' });
                })).catch(err => {
                    console.warn('[SW] Some static assets failed to cache:', err);
                    // Continue even if some assets fail
                    return Promise.resolve();
                });
            })
            .then(() => self.skipWaiting())
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
    console.log('[SW] Activating service worker...');
    event.waitUntil(
        caches.keys()
            .then((cacheNames) => {
                return Promise.all(
                    cacheNames
                        .filter((name) => {
                            return name.startsWith('static-') ||
                                   name.startsWith('dynamic-') ||
                                   name.startsWith('articles-') ||
                                   name.startsWith('images-');
                        })
                        .filter((name) => {
                            return name !== STATIC_CACHE &&
                                   name !== DYNAMIC_CACHE &&
                                   name !== ARTICLE_CACHE &&
                                   name !== IMAGE_CACHE;
                        })
                        .map((name) => {
                            console.log('[SW] Deleting old cache:', name);
                            return caches.delete(name);
                        })
                );
            })
            .then(() => self.clients.claim())
    );
});


// Fetch event - implement caching strategies
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // Skip non-GET requests
    if (request.method !== 'GET') {
        return;
    }

    // Skip chrome-extension and other non-http(s) requests
    if (!url.protocol.startsWith('http')) {
        return;
    }

    // Skip API and admin requests - always network
    if (API_PATTERNS.some(pattern => pattern.test(url.pathname))) {
        event.respondWith(networkOnly(request));
        return;
    }

    // Handle different resource types
    if (isStaticAsset(url.pathname)) {
        // Cache-first for static assets
        event.respondWith(cacheFirst(request, STATIC_CACHE));
    } else if (isImage(url.pathname)) {
        // Cache-first for images with fallback
        event.respondWith(cacheFirstWithFallback(request, IMAGE_CACHE));
    } else if (isArticle(url.pathname)) {
        // Network-first for articles (fresh content)
        event.respondWith(networkFirst(request, ARTICLE_CACHE));
    } else if (isPage(url.pathname)) {
        // Network-first for pages
        event.respondWith(networkFirst(request, DYNAMIC_CACHE));
    } else {
        // Default: network-first
        event.respondWith(networkFirst(request, DYNAMIC_CACHE));
    }
});

// Helper functions to identify resource types
function isStaticAsset(pathname) {
    return pathname.startsWith('/static/css/') ||
           pathname.startsWith('/static/js/') ||
           pathname.startsWith('/static/icons/') ||
           pathname === '/static/manifest.json' ||
           pathname === '/static/favicon.ico';
}

function isImage(pathname) {
    return pathname.startsWith('/static/images/') ||
           pathname.startsWith('/uploads/') ||
           pathname.startsWith('/media/') ||
           /\.(jpg|jpeg|png|gif|webp|avif|svg)$/i.test(pathname);
}

function isArticle(pathname) {
    return pathname.startsWith('/article/') ||
           pathname.startsWith('/news/');
}

function isPage(pathname) {
    return pathname === '/' ||
           pathname.startsWith('/category/') ||
           pathname.startsWith('/tag/') ||
           pathname.startsWith('/search') ||
           pathname.startsWith('/latest');
}

// Cache-first strategy (for static assets)
async function cacheFirst(request, cacheName) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
        return cachedResponse;
    }

    try {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, networkResponse.clone());
        }
        return networkResponse;
    } catch (error) {
        console.error('[SW] Cache-first fetch failed:', error);
        return new Response('Offline', { status: 503 });
    }
}

// Cache-first with fallback for images
async function cacheFirstWithFallback(request, cacheName) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
        return cachedResponse;
    }

    try {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, networkResponse.clone());
            // Trim cache if needed
            trimCache(cacheName, MAX_IMAGE_CACHE);
        }
        return networkResponse;
    } catch (error) {
        // Return placeholder image for failed image requests
        return caches.match('/static/icons/phosphor/newspaper.svg');
    }
}

// Network-first strategy (for articles and pages)
async function networkFirst(request, cacheName) {
    try {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, networkResponse.clone());
            // Trim cache if needed
            if (cacheName === ARTICLE_CACHE) {
                trimCache(cacheName, MAX_ARTICLE_CACHE);
            }
        }
        return networkResponse;
    } catch (error) {
        console.log('[SW] Network failed, trying cache:', request.url);
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        // Return offline page for navigation requests
        if (request.mode === 'navigate') {
            return caches.match('/offline');
        }
        return new Response('Offline', { status: 503 });
    }
}

// Network-only strategy (for API requests)
async function networkOnly(request) {
    try {
        return await fetch(request);
    } catch (error) {
        return new Response(JSON.stringify({ error: 'Offline' }), {
            status: 503,
            headers: { 'Content-Type': 'application/json' }
        });
    }
}

// Trim cache to stay within size limits
async function trimCache(cacheName, maxSize) {
    const cache = await caches.open(cacheName);
    const keys = await cache.keys();
    
    let totalSize = 0;
    const entries = [];
    
    for (const request of keys) {
        const response = await cache.match(request);
        if (response) {
            const blob = await response.clone().blob();
            entries.push({ request, size: blob.size });
            totalSize += blob.size;
        }
    }
    
    // Remove oldest entries if over limit
    if (totalSize > maxSize) {
        const toRemove = entries.slice(0, Math.ceil(entries.length * 0.3));
        for (const entry of toRemove) {
            await cache.delete(entry.request);
        }
        console.log(`[SW] Trimmed ${toRemove.length} entries from ${cacheName}`);
    }
}


// Push notification handling
self.addEventListener('push', (event) => {
    console.log('[SW] Push event received:', event);

    let notificationData = {};
    
    if (event.data) {
        try {
            notificationData = event.data.json();
        } catch (error) {
            console.error('[SW] Failed to parse push data:', error);
            notificationData = {
                title: 'New Notification',
                body: event.data.text() || 'You have a new notification',
            };
        }
    } else {
        notificationData = {
            title: 'New Notification',
            body: 'You have a new notification',
        };
    }

    const options = {
        body: notificationData.body || 'New notification',
        icon: notificationData.icon || '/static/icons/icon-192x192.png',
        badge: notificationData.badge || '/static/icons/badge-72x72.png',
        image: notificationData.image,
        data: {
            url: notificationData.url || '/',
            delivery_id: notificationData.delivery_id,
            notification_id: notificationData.notification_id,
            ...notificationData.data
        },
        actions: notificationData.actions || [
            { action: 'open', title: 'Open' },
            { action: 'close', title: 'Close' }
        ],
        requireInteraction: notificationData.requireInteraction || false,
        silent: notificationData.silent || false,
        tag: notificationData.tag || 'default',
        timestamp: Date.now(),
        vibrate: notificationData.vibrate || [200, 100, 200],
        renotify: true
    };

    event.waitUntil(
        self.registration.showNotification(notificationData.title || 'New Notification', options)
            .then(() => {
                if (notificationData.delivery_id) {
                    return trackDelivery(notificationData.delivery_id);
                }
            })
            .catch((error) => {
                console.error('[SW] Failed to show notification:', error);
            })
    );
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
    const notification = event.notification;
    const action = event.action;
    const data = notification.data || {};

    if (data.delivery_id) {
        trackClick(data.delivery_id);
    }

    notification.close();

    if (action === 'close') {
        return;
    }

    const urlToOpen = data.url || '/';

    event.waitUntil(
        self.clients.matchAll({ type: 'window', includeUncontrolled: true })
            .then((clientList) => {
                for (const client of clientList) {
                    if (client.url === urlToOpen && 'focus' in client) {
                        return client.focus();
                    }
                }
                if (self.clients.openWindow) {
                    return self.clients.openWindow(urlToOpen);
                }
            })
    );
});

// Background sync for failed requests
self.addEventListener('sync', (event) => {
    console.log('[SW] Background sync:', event.tag);
    
    if (event.tag === 'push-tracking') {
        event.waitUntil(retryFailedRequests());
    }
});

// Message event for communication with main thread
self.addEventListener('message', (event) => {
    console.log('[SW] Received message:', event.data);
    
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }
    
    if (event.data && event.data.type === 'CACHE_ARTICLE') {
        event.waitUntil(cacheArticle(event.data.url));
    }
    
    if (event.data && event.data.type === 'GET_CACHE_STATUS') {
        event.waitUntil(getCacheStatus().then(status => {
            event.ports[0].postMessage(status);
        }));
    }
});

// Cache a specific article for offline reading
async function cacheArticle(url) {
    try {
        const cache = await caches.open(ARTICLE_CACHE);
        const response = await fetch(url);
        if (response.ok) {
            await cache.put(url, response);
            console.log('[SW] Article cached:', url);
        }
    } catch (error) {
        console.error('[SW] Failed to cache article:', error);
    }
}

// Get cache status for UI
async function getCacheStatus() {
    const cacheNames = [STATIC_CACHE, DYNAMIC_CACHE, ARTICLE_CACHE, IMAGE_CACHE];
    let totalSize = 0;
    let itemCount = 0;
    
    for (const name of cacheNames) {
        try {
            const cache = await caches.open(name);
            const keys = await cache.keys();
            itemCount += keys.length;
            
            for (const request of keys) {
                const response = await cache.match(request);
                if (response) {
                    const blob = await response.clone().blob();
                    totalSize += blob.size;
                }
            }
        } catch (e) {
            // Ignore errors
        }
    }
    
    return {
        totalSize,
        itemCount,
        maxSize: MAX_CACHE_SIZE,
        percentage: Math.round((totalSize / MAX_CACHE_SIZE) * 100)
    };
}

// Track notification delivery
async function trackDelivery(deliveryId) {
    try {
        const response = await fetch(`/api/v1/push/track/delivery/${deliveryId}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        console.log('[SW] Delivery tracked:', deliveryId);
    } catch (error) {
        console.error('[SW] Failed to track delivery:', error);
        await storeFailedRequest('delivery', deliveryId);
    }
}

// Track notification click
async function trackClick(deliveryId) {
    try {
        const response = await fetch(`/api/v1/push/track/click/${deliveryId}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        console.log('[SW] Click tracked:', deliveryId);
    } catch (error) {
        console.error('[SW] Failed to track click:', error);
        await storeFailedRequest('click', deliveryId);
    }
}

// Store failed request for retry
async function storeFailedRequest(type, deliveryId) {
    try {
        const db = await openDB();
        const transaction = db.transaction(['failed_requests'], 'readwrite');
        const store = transaction.objectStore('failed_requests');
        await store.add({ type, deliveryId, timestamp: Date.now() });
    } catch (error) {
        console.error('[SW] Failed to store failed request:', error);
    }
}

// Retry failed requests
async function retryFailedRequests() {
    try {
        const db = await openDB();
        const transaction = db.transaction(['failed_requests'], 'readwrite');
        const store = transaction.objectStore('failed_requests');
        const requests = await store.getAll();
        
        for (const request of requests) {
            try {
                if (request.type === 'delivery') {
                    await trackDelivery(request.deliveryId);
                } else if (request.type === 'click') {
                    await trackClick(request.deliveryId);
                }
                await store.delete(request.id);
            } catch (error) {
                if (Date.now() - request.timestamp > 24 * 60 * 60 * 1000) {
                    await store.delete(request.id);
                }
            }
        }
    } catch (error) {
        console.error('[SW] Failed to retry requests:', error);
    }
}

// Open IndexedDB
function openDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open('PushNotificationDB', 1);
        request.onerror = () => reject(request.error);
        request.onsuccess = () => resolve(request.result);
        request.onupgradeneeded = (event) => {
            const db = event.target.result;
            if (!db.objectStoreNames.contains('failed_requests')) {
                const store = db.createObjectStore('failed_requests', { 
                    keyPath: 'id', 
                    autoIncrement: true 
                });
                store.createIndex('timestamp', 'timestamp', { unique: false });
                store.createIndex('type', 'type', { unique: false });
            }
        };
    });
}

console.log('[SW] Service Worker loaded - version', CACHE_VERSION);
