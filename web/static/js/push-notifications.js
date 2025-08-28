/**
 * Push Notification Manager
 * Handles push notification subscription, preferences, and tracking
 */
class PushNotificationManager {
    constructor(options = {}) {
        this.apiBase = options.apiBase || '/api/v1/push';
        this.vapidPublicKey = options.vapidPublicKey || '';
        this.oneSignalAppId = options.oneSignalAppId || '';
        this.firebaseConfig = options.firebaseConfig || null;
        this.subscription = null;
        this.subscriptionId = null;
        this.isSupported = this.checkSupport();
        
        this.init();
    }

    /**
     * Check if push notifications are supported
     */
    checkSupport() {
        return 'serviceWorker' in navigator && 'PushManager' in window;
    }

    /**
     * Initialize push notification manager
     */
    async init() {
        if (!this.isSupported) {
            console.warn('Push notifications are not supported in this browser');
            return;
        }

        try {
            // Register service worker
            const registration = await navigator.serviceWorker.register('/sw.js');
            console.log('Service Worker registered:', registration);

            // Check for existing subscription
            this.subscription = await registration.pushManager.getSubscription();
            if (this.subscription) {
                this.subscriptionId = await this.getSubscriptionId();
                this.updateUI(true);
            }

            // Initialize OneSignal if configured
            if (this.oneSignalAppId) {
                this.initOneSignal();
            }

            // Initialize Firebase if configured
            if (this.firebaseConfig) {
                this.initFirebase();
            }

        } catch (error) {
            console.error('Failed to initialize push notifications:', error);
        }
    }

    /**
     * Initialize OneSignal
     */
    initOneSignal() {
        if (typeof OneSignal !== 'undefined') {
            OneSignal.init({
                appId: this.oneSignalAppId,
                notifyButton: {
                    enable: true,
                },
                allowLocalhostAsSecureOrigin: true,
            });

            OneSignal.on('subscriptionChange', (isSubscribed) => {
                console.log('OneSignal subscription changed:', isSubscribed);
                this.updateUI(isSubscribed);
            });
        }
    }

    /**
     * Initialize Firebase
     */
    async initFirebase() {
        if (typeof firebase !== 'undefined') {
            firebase.initializeApp(this.firebaseConfig);
            const messaging = firebase.messaging();

            // Request permission and get token
            try {
                const token = await messaging.getToken({ vapidKey: this.vapidPublicKey });
                if (token) {
                    console.log('Firebase token:', token);
                    await this.subscribeWithToken(token);
                }
            } catch (error) {
                console.error('Failed to get Firebase token:', error);
            }

            // Handle foreground messages
            messaging.onMessage((payload) => {
                console.log('Message received:', payload);
                this.showNotification(payload.notification);
            });
        }
    }

    /**
     * Request notification permission
     */
    async requestPermission() {
        if (!this.isSupported) {
            throw new Error('Push notifications are not supported');
        }

        const permission = await Notification.requestPermission();
        if (permission !== 'granted') {
            throw new Error('Notification permission denied');
        }

        return permission;
    }

    /**
     * Subscribe to push notifications
     */
    async subscribe() {
        try {
            await this.requestPermission();

            const registration = await navigator.serviceWorker.ready;
            
            // Convert VAPID key
            const applicationServerKey = this.urlBase64ToUint8Array(this.vapidPublicKey);
            
            // Subscribe to push manager
            this.subscription = await registration.pushManager.subscribe({
                userVisibleOnly: true,
                applicationServerKey: applicationServerKey
            });

            // Send subscription to server
            const response = await fetch(`${this.apiBase}/subscribe`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    endpoint: this.subscription.endpoint,
                    p256dh: btoa(String.fromCharCode(...new Uint8Array(this.subscription.getKey('p256dh')))),
                    auth: btoa(String.fromCharCode(...new Uint8Array(this.subscription.getKey('auth')))),
                    user_id: this.getCurrentUserId()
                })
            });

            if (!response.ok) {
                throw new Error('Failed to subscribe on server');
            }

            const data = await response.json();
            this.subscriptionId = data.subscription_id;
            
            this.updateUI(true);
            this.showMessage('Successfully subscribed to push notifications!', 'success');
            
            return this.subscription;

        } catch (error) {
            console.error('Failed to subscribe:', error);
            this.showMessage('Failed to subscribe to push notifications', 'error');
            throw error;
        }
    }

    /**
     * Subscribe with token (for Firebase)
     */
    async subscribeWithToken(token) {
        try {
            const response = await fetch(`${this.apiBase}/subscribe`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    endpoint: token,
                    p256dh: 'firebase',
                    auth: 'firebase',
                    user_id: this.getCurrentUserId()
                })
            });

            if (!response.ok) {
                throw new Error('Failed to subscribe with token');
            }

            const data = await response.json();
            this.subscriptionId = data.subscription_id;
            this.updateUI(true);

        } catch (error) {
            console.error('Failed to subscribe with token:', error);
        }
    }

    /**
     * Unsubscribe from push notifications
     */
    async unsubscribe() {
        try {
            if (this.subscription) {
                // Unsubscribe from push manager
                await this.subscription.unsubscribe();
                
                // Notify server
                await fetch(`${this.apiBase}/unsubscribe`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        endpoint: this.subscription.endpoint
                    })
                });

                this.subscription = null;
                this.subscriptionId = null;
                this.updateUI(false);
                this.showMessage('Successfully unsubscribed from push notifications', 'success');
            }

        } catch (error) {
            console.error('Failed to unsubscribe:', error);
            this.showMessage('Failed to unsubscribe from push notifications', 'error');
        }
    }

    /**
     * Update notification preferences
     */
    async updatePreferences(preferences) {
        if (!this.subscriptionId) {
            throw new Error('No active subscription');
        }

        try {
            const response = await fetch(`${this.apiBase}/preferences`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    subscription_id: this.subscriptionId,
                    user_id: this.getCurrentUserId(),
                    ...preferences
                })
            });

            if (!response.ok) {
                throw new Error('Failed to update preferences');
            }

            this.showMessage('Notification preferences updated successfully', 'success');

        } catch (error) {
            console.error('Failed to update preferences:', error);
            this.showMessage('Failed to update notification preferences', 'error');
            throw error;
        }
    }

    /**
     * Get notification preferences
     */
    async getPreferences() {
        if (!this.subscriptionId) {
            return null;
        }

        try {
            const response = await fetch(`${this.apiBase}/preferences/${this.subscriptionId}`);
            if (!response.ok) {
                throw new Error('Failed to get preferences');
            }

            return await response.json();

        } catch (error) {
            console.error('Failed to get preferences:', error);
            return null;
        }
    }

    /**
     * Track notification delivery
     */
    async trackDelivery(deliveryId) {
        try {
            await fetch(`${this.apiBase}/track/delivery/${deliveryId}`, {
                method: 'POST'
            });
        } catch (error) {
            console.error('Failed to track delivery:', error);
        }
    }

    /**
     * Track notification click
     */
    async trackClick(deliveryId) {
        try {
            await fetch(`${this.apiBase}/track/click/${deliveryId}`, {
                method: 'POST'
            });
        } catch (error) {
            console.error('Failed to track click:', error);
        }
    }

    /**
     * Show notification (for testing)
     */
    showNotification(notification) {
        if (Notification.permission === 'granted') {
            const options = {
                body: notification.body,
                icon: notification.icon || '/static/images/icon-192x192.png',
                badge: notification.badge || '/static/images/badge-72x72.png',
                image: notification.image,
                data: notification.data || {},
                actions: notification.actions || [],
                requireInteraction: notification.requireInteraction || false,
                silent: notification.silent || false,
                tag: notification.tag || 'default',
                timestamp: Date.now()
            };

            const notif = new Notification(notification.title, options);
            
            notif.onclick = (event) => {
                event.preventDefault();
                if (notification.data && notification.data.url) {
                    window.open(notification.data.url, '_blank');
                }
                notif.close();
            };

            // Auto close after 10 seconds
            setTimeout(() => {
                notif.close();
            }, 10000);
        }
    }

    /**
     * Utility functions
     */
    urlBase64ToUint8Array(base64String) {
        const padding = '='.repeat((4 - base64String.length % 4) % 4);
        const base64 = (base64String + padding)
            .replace(/-/g, '+')
            .replace(/_/g, '/');

        const rawData = window.atob(base64);
        const outputArray = new Uint8Array(rawData.length);

        for (let i = 0; i < rawData.length; ++i) {
            outputArray[i] = rawData.charCodeAt(i);
        }
        return outputArray;
    }

    getCurrentUserId() {
        // This should be implemented based on your authentication system
        const userMeta = document.querySelector('meta[name="user-id"]');
        return userMeta ? parseInt(userMeta.content) : null;
    }

    async getSubscriptionId() {
        // This would typically be stored in localStorage or retrieved from server
        return localStorage.getItem('push_subscription_id');
    }

    updateUI(isSubscribed) {
        // Update UI elements based on subscription status
        const subscribeBtn = document.getElementById('push-subscribe-btn');
        const unsubscribeBtn = document.getElementById('push-unsubscribe-btn');
        const preferencesSection = document.getElementById('push-preferences');

        if (subscribeBtn) {
            subscribeBtn.style.display = isSubscribed ? 'none' : 'block';
        }
        if (unsubscribeBtn) {
            unsubscribeBtn.style.display = isSubscribed ? 'block' : 'none';
        }
        if (preferencesSection) {
            preferencesSection.style.display = isSubscribed ? 'block' : 'none';
        }

        // Store subscription status
        localStorage.setItem('push_subscribed', isSubscribed.toString());
        if (this.subscriptionId) {
            localStorage.setItem('push_subscription_id', this.subscriptionId.toString());
        }
    }

    showMessage(message, type = 'info') {
        // Show user-friendly messages
        const messageDiv = document.getElementById('push-messages');
        if (messageDiv) {
            messageDiv.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
            setTimeout(() => {
                messageDiv.innerHTML = '';
            }, 5000);
        } else {
            console.log(`${type.toUpperCase()}: ${message}`);
        }
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Get configuration from meta tags
    const vapidKey = document.querySelector('meta[name="vapid-public-key"]')?.content || '';
    const oneSignalAppId = document.querySelector('meta[name="onesignal-app-id"]')?.content || '';
    const firebaseConfigMeta = document.querySelector('meta[name="firebase-config"]')?.content;
    
    let firebaseConfig = null;
    if (firebaseConfigMeta) {
        try {
            firebaseConfig = JSON.parse(firebaseConfigMeta);
        } catch (e) {
            console.error('Invalid Firebase config:', e);
        }
    }

    // Initialize push notification manager
    window.pushManager = new PushNotificationManager({
        vapidPublicKey: vapidKey,
        oneSignalAppId: oneSignalAppId,
        firebaseConfig: firebaseConfig
    });

    // Bind event listeners
    const subscribeBtn = document.getElementById('push-subscribe-btn');
    const unsubscribeBtn = document.getElementById('push-unsubscribe-btn');
    const preferencesForm = document.getElementById('push-preferences-form');

    if (subscribeBtn) {
        subscribeBtn.addEventListener('click', async () => {
            try {
                await window.pushManager.subscribe();
            } catch (error) {
                console.error('Subscription failed:', error);
            }
        });
    }

    if (unsubscribeBtn) {
        unsubscribeBtn.addEventListener('click', async () => {
            try {
                await window.pushManager.unsubscribe();
            } catch (error) {
                console.error('Unsubscription failed:', error);
            }
        });
    }

    if (preferencesForm) {
        preferencesForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(preferencesForm);
            const preferences = {
                breaking_news: formData.get('breaking_news') === 'on',
                category_updates: formData.get('category_updates') === 'on',
                tag_updates: formData.get('tag_updates') === 'on',
                author_updates: formData.get('author_updates') === 'on',
                preferred_categories: formData.getAll('preferred_categories').map(id => parseInt(id)),
                preferred_tags: formData.getAll('preferred_tags').map(id => parseInt(id)),
                preferred_authors: formData.getAll('preferred_authors').map(id => parseInt(id))
            };

            try {
                await window.pushManager.updatePreferences(preferences);
            } catch (error) {
                console.error('Failed to update preferences:', error);
            }
        });
    }
});

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = PushNotificationManager;
}