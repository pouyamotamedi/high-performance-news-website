/**
 * Service Worker for Push Notifications
 * Handles push events, notification display, and click tracking
 */

const CACHE_NAME = 'push-notifications-v1';
const API_BASE = '/api/v1/push';

// Install event
self.addEventListener('install', (event) => {
    console.log('Service Worker installing');
    self.skipWaiting();
});

// Activate event
self.addEventListener('activate', (event) => {
    console.log('Service Worker activating');
    event.waitUntil(self.clients.claim());
});

// Push event - handle incoming push notifications
self.addEventListener('push', (event) => {
    console.log('Push event received:', event);

    let notificationData = {};
    
    if (event.data) {
        try {
            notificationData = event.data.json();
        } catch (error) {
            console.error('Failed to parse push data:', error);
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
        icon: notificationData.icon || '/static/images/icon-192x192.png',
        badge: notificationData.badge || '/static/images/badge-72x72.png',
        image: notificationData.image,
        data: {
            url: notificationData.url || '/',
            delivery_id: notificationData.delivery_id,
            notification_id: notificationData.notification_id,
            ...notificationData.data
        },
        actions: notificationData.actions || [
            {
                action: 'open',
                title: 'Open',
                icon: '/static/images/open-icon.png'
            },
            {
                action: 'close',
                title: 'Close',
                icon: '/static/images/close-icon.png'
            }
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
                // Track delivery
                if (notificationData.delivery_id) {
                    return trackDelivery(notificationData.delivery_id);
                }
            })
            .catch((error) => {
                console.error('Failed to show notification:', error);
            })
    );
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
    console.log('Notification clicked:', event);

    const notification = event.notification;
    const action = event.action;
    const data = notification.data || {};

    // Track click
    if (data.delivery_id) {
        trackClick(data.delivery_id);
    }

    // Close notification
    notification.close();

    // Handle different actions
    if (action === 'close') {
        return;
    }

    // Default action or 'open' action
    const urlToOpen = data.url || '/';

    event.waitUntil(
        self.clients.matchAll({ type: 'window', includeUncontrolled: true })
            .then((clientList) => {
                // Check if there's already a window/tab open with the target URL
                for (let i = 0; i < clientList.length; i++) {
                    const client = clientList[i];
                    if (client.url === urlToOpen && 'focus' in client) {
                        return client.focus();
                    }
                }

                // If no existing window/tab, open a new one
                if (self.clients.openWindow) {
                    return self.clients.openWindow(urlToOpen);
                }
            })
            .catch((error) => {
                console.error('Failed to handle notification click:', error);
            })
    );
});

// Notification close event
self.addEventListener('notificationclose', (event) => {
    console.log('Notification closed:', event);
    
    const notification = event.notification;
    const data = notification.data || {};

    // You could track notification dismissals here if needed
    console.log('Notification dismissed:', data);
});

// Background sync for failed requests
self.addEventListener('sync', (event) => {
    console.log('Background sync:', event);
    
    if (event.tag === 'push-tracking') {
        event.waitUntil(retryFailedRequests());
    }
});

// Message event for communication with main thread
self.addEventListener('message', (event) => {
    console.log('Service Worker received message:', event.data);
    
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});

/**
 * Track notification delivery
 */
async function trackDelivery(deliveryId) {
    try {
        const response = await fetch(`${API_BASE}/track/delivery/${deliveryId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        console.log('Delivery tracked successfully:', deliveryId);
    } catch (error) {
        console.error('Failed to track delivery:', error);
        
        // Store failed request for retry
        await storeFailedRequest('delivery', deliveryId);
        
        // Register for background sync
        if ('serviceWorker' in navigator && 'sync' in window.ServiceWorkerRegistration.prototype) {
            await self.registration.sync.register('push-tracking');
        }
    }
}

/**
 * Track notification click
 */
async function trackClick(deliveryId) {
    try {
        const response = await fetch(`${API_BASE}/track/click/${deliveryId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        console.log('Click tracked successfully:', deliveryId);
    } catch (error) {
        console.error('Failed to track click:', error);
        
        // Store failed request for retry
        await storeFailedRequest('click', deliveryId);
        
        // Register for background sync
        if ('serviceWorker' in navigator && 'sync' in window.ServiceWorkerRegistration.prototype) {
            await self.registration.sync.register('push-tracking');
        }
    }
}

/**
 * Store failed request for retry
 */
async function storeFailedRequest(type, deliveryId) {
    try {
        const db = await openDB();
        const transaction = db.transaction(['failed_requests'], 'readwrite');
        const store = transaction.objectStore('failed_requests');
        
        await store.add({
            type: type,
            deliveryId: deliveryId,
            timestamp: Date.now()
        });
        
        console.log('Failed request stored for retry:', type, deliveryId);
    } catch (error) {
        console.error('Failed to store failed request:', error);
    }
}

/**
 * Retry failed requests
 */
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
                
                // Remove successful request
                await store.delete(request.id);
            } catch (error) {
                console.error('Failed to retry request:', error);
                
                // Remove old failed requests (older than 24 hours)
                if (Date.now() - request.timestamp > 24 * 60 * 60 * 1000) {
                    await store.delete(request.id);
                }
            }
        }
    } catch (error) {
        console.error('Failed to retry failed requests:', error);
    }
}

/**
 * Open IndexedDB for storing failed requests
 */
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

/**
 * Show custom notification (for testing)
 */
function showCustomNotification(title, options = {}) {
    const defaultOptions = {
        body: 'Test notification',
        icon: '/static/images/icon-192x192.png',
        badge: '/static/images/badge-72x72.png',
        tag: 'test',
        requireInteraction: false,
        actions: [
            {
                action: 'open',
                title: 'Open',
            },
            {
                action: 'close',
                title: 'Close',
            }
        ]
    };

    const finalOptions = { ...defaultOptions, ...options };
    
    return self.registration.showNotification(title, finalOptions);
}

// Expose functions for testing
self.showCustomNotification = showCustomNotification;
self.trackDelivery = trackDelivery;
self.trackClick = trackClick;