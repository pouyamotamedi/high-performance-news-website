package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

type PushNotificationService struct {
	repo           *repositories.PushNotificationRepository
	oneSignalAppID string
	oneSignalKey   string
	firebaseKey    string
	httpClient     *http.Client
}

type OneSignalNotification struct {
	AppID            string                 `json:"app_id"`
	IncludePlayerIDs []string               `json:"include_player_ids,omitempty"`
	IncludedSegments []string               `json:"included_segments,omitempty"`
	Headings         map[string]string      `json:"headings"`
	Contents         map[string]string      `json:"contents"`
	Data             map[string]interface{} `json:"data,omitempty"`
	URL              string                 `json:"url,omitempty"`
	SmallIcon        string                 `json:"small_icon,omitempty"`
	LargeIcon        string                 `json:"large_icon,omitempty"`
	BigPicture       string                 `json:"big_picture,omitempty"`
	SendAfter        string                 `json:"send_after,omitempty"`
}

type FirebaseNotification struct {
	To           string                 `json:"to,omitempty"`
	RegistrationIDs []string            `json:"registration_ids,omitempty"`
	Notification FirebasePayload        `json:"notification"`
	Data         map[string]interface{} `json:"data,omitempty"`
	WebPush      FirebaseWebPush        `json:"webpush,omitempty"`
}

type FirebasePayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Badge string `json:"badge,omitempty"`
	Image string `json:"image,omitempty"`
}

type FirebaseWebPush struct {
	Headers      map[string]string      `json:"headers,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Notification FirebasePayload        `json:"notification"`
	FCMOptions   FirebaseFCMOptions     `json:"fcm_options,omitempty"`
}

type FirebaseFCMOptions struct {
	Link string `json:"link,omitempty"`
}

func NewPushNotificationService(repo *repositories.PushNotificationRepository, oneSignalAppID, oneSignalKey, firebaseKey string) *PushNotificationService {
	return &PushNotificationService{
		repo:           repo,
		oneSignalAppID: oneSignalAppID,
		oneSignalKey:   oneSignalKey,
		firebaseKey:    firebaseKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Subscription management
func (s *PushNotificationService) Subscribe(subscription *models.PushSubscription) error {
	if err := subscription.Validate(); err != nil {
		return err
	}

	return s.repo.CreateSubscription(subscription)
}

func (s *PushNotificationService) Unsubscribe(endpoint string) error {
	return s.repo.DeactivateSubscription(endpoint)
}

func (s *PushNotificationService) GetSubscription(endpoint string) (*models.PushSubscription, error) {
	return s.repo.GetSubscriptionByEndpoint(endpoint)
}

// Notification creation and scheduling
func (s *PushNotificationService) CreateNotification(notification *models.PushNotification) error {
	if err := notification.Validate(); err != nil {
		return err
	}

	return s.repo.CreateNotification(notification)
}

// GetNotification retrieves a notification by ID
func (s *PushNotificationService) GetNotification(notificationID uint64) (*models.PushNotification, error) {
	return s.repo.GetNotificationByID(notificationID)
}

func (s *PushNotificationService) CreateFromTemplate(templateName string, variables map[string]string, targetType, targetValue string, scheduledAt *time.Time) error {
	template, err := s.repo.GetTemplateByName(templateName)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Replace template variables
	title := s.replaceVariables(template.Title, variables)
	body := s.replaceVariables(template.Body, variables)
	url := s.replaceVariables(template.URL, variables)

	notification := &models.PushNotification{
		Title:       title,
		Body:        body,
		Icon:        template.Icon,
		Badge:       template.Badge,
		Image:       template.Image,
		URL:         url,
		TargetType:  targetType,
		TargetValue: targetValue,
		ScheduledAt: scheduledAt,
		Status:      models.NotificationStatusPending,
	}

	return s.repo.CreateNotification(notification)
}

func (s *PushNotificationService) replaceVariables(text string, variables map[string]string) string {
	result := text
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// Notification sending
func (s *PushNotificationService) SendNotification(notificationID uint64) error {
	notification, err := s.repo.GetNotificationByID(notificationID)
	if err != nil {
		return err
	}

	if notification.Status != models.NotificationStatusPending {
		return fmt.Errorf("notification is not in pending status")
	}

	// Update status to sending
	err = s.repo.UpdateNotificationStatus(notificationID, models.NotificationStatusSending, 0)
	if err != nil {
		return err
	}

	// Get target subscriptions
	subscriptions, err := s.repo.GetTargetedSubscriptions(notification.TargetType, notification.TargetValue)
	if err != nil {
		s.repo.UpdateNotificationStatus(notificationID, models.NotificationStatusFailed, 0)
		return err
	}

	if len(subscriptions) == 0 {
		s.repo.UpdateNotificationStatus(notificationID, models.NotificationStatusSent, 0)
		return nil
	}

	// Send notifications
	totalSent := 0
	for _, subscription := range subscriptions {
		err := s.sendToSubscription(notification, subscription)
		if err != nil {
			// Log error but continue with other subscriptions
			log.Printf("Failed to send notification to subscription %d: %v", subscription.ID, err)
			s.repo.CreateDelivery(&models.PushDelivery{
				NotificationID: notificationID,
				SubscriptionID: subscription.ID,
				Status:         models.DeliveryStatusFailed,
				ErrorMessage:   err.Error(),
			})
		} else {
			totalSent++
			s.repo.CreateDelivery(&models.PushDelivery{
				NotificationID: notificationID,
				SubscriptionID: subscription.ID,
				Status:         models.DeliveryStatusSent,
			})
		}
	}

	// Update notification status
	status := models.NotificationStatusSent
	if totalSent == 0 {
		status = models.NotificationStatusFailed
	}

	return s.repo.UpdateNotificationStatus(notificationID, status, totalSent)
}

func (s *PushNotificationService) sendToSubscription(notification *models.PushNotification, subscription *models.PushSubscription) error {
	// Try OneSignal first if configured
	if s.oneSignalAppID != "" && s.oneSignalKey != "" {
		return s.sendViaOneSignal(notification, subscription)
	}

	// Fallback to Firebase if configured
	if s.firebaseKey != "" {
		return s.sendViaFirebase(notification, subscription)
	}

	return fmt.Errorf("no push notification service configured")
}

func (s *PushNotificationService) sendViaOneSignal(notification *models.PushNotification, subscription *models.PushSubscription) error {
	payload := OneSignalNotification{
		AppID: s.oneSignalAppID,
		IncludePlayerIDs: []string{subscription.Endpoint}, // OneSignal uses player IDs
		Headings: map[string]string{
			"en": notification.Title,
		},
		Contents: map[string]string{
			"en": notification.Body,
		},
		Data:      notification.Data,
		URL:       notification.URL,
		SmallIcon: notification.Icon,
		LargeIcon: notification.Badge,
		BigPicture: notification.Image,
	}

	if notification.ScheduledAt != nil && notification.ScheduledAt.After(time.Now()) {
		payload.SendAfter = notification.ScheduledAt.UTC().Format("2006-01-02 15:04:05 GMT-0000")
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://onesignal.com/api/v1/notifications", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+s.oneSignalKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OneSignal API error: %d", resp.StatusCode)
	}

	return nil
}

func (s *PushNotificationService) sendViaFirebase(notification *models.PushNotification, subscription *models.PushSubscription) error {
	payload := FirebaseNotification{
		To: subscription.Endpoint, // Firebase uses registration tokens
		Notification: FirebasePayload{
			Title: notification.Title,
			Body:  notification.Body,
			Icon:  notification.Icon,
			Badge: notification.Badge,
			Image: notification.Image,
		},
		Data: notification.Data,
		WebPush: FirebaseWebPush{
			Notification: FirebasePayload{
				Title: notification.Title,
				Body:  notification.Body,
				Icon:  notification.Icon,
				Badge: notification.Badge,
				Image: notification.Image,
			},
			FCMOptions: FirebaseFCMOptions{
				Link: notification.URL,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+s.firebaseKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Firebase API error: %d", resp.StatusCode)
	}

	return nil
}

// Batch processing for pending notifications
func (s *PushNotificationService) ProcessPendingNotifications(ctx context.Context) error {
	notifications, err := s.repo.GetPendingNotifications()
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := s.SendNotification(notification.ID)
			if err != nil {
				log.Printf("Failed to send notification %d: %v", notification.ID, err)
			}
		}
	}

	return nil
}

// Template management
func (s *PushNotificationService) CreateTemplate(template *models.PushTemplate) error {
	return s.repo.CreateTemplate(template)
}

func (s *PushNotificationService) GetTemplate(name string) (*models.PushTemplate, error) {
	return s.repo.GetTemplateByName(name)
}

func (s *PushNotificationService) GetActiveTemplates() ([]*models.PushTemplate, error) {
	return s.repo.GetActiveTemplates()
}

// Preference management
func (s *PushNotificationService) UpdatePreferences(prefs *models.NotificationPreference) error {
	return s.repo.CreateOrUpdatePreferences(prefs)
}

func (s *PushNotificationService) GetPreferences(subscriptionID uint64) (*models.NotificationPreference, error) {
	return s.repo.GetPreferencesBySubscription(subscriptionID)
}

// Analytics and tracking
func (s *PushNotificationService) TrackDelivery(deliveryID uint64) error {
	return s.repo.UpdateDeliveryStatus(deliveryID, models.DeliveryStatusDelivered)
}

func (s *PushNotificationService) TrackClick(deliveryID uint64) error {
	return s.repo.UpdateDeliveryStatus(deliveryID, models.DeliveryStatusClicked)
}

func (s *PushNotificationService) UpdateNotificationStats(notificationID uint64) error {
	delivered, clicked, err := s.repo.GetDeliveryStats(notificationID)
	if err != nil {
		return err
	}

	return s.repo.UpdateNotificationStats(notificationID, delivered, clicked)
}

// Convenience methods for common notification types
func (s *PushNotificationService) SendBreakingNews(title, body, url string) error {
	notification := &models.PushNotification{
		Title:      title,
		Body:       body,
		URL:        url,
		TargetType: models.TargetTypeAll,
		Status:     models.NotificationStatusPending,
		Data: map[string]interface{}{
			"type": "breaking_news",
		},
	}

	err := s.repo.CreateNotification(notification)
	if err != nil {
		return err
	}

	return s.SendNotification(notification.ID)
}

func (s *PushNotificationService) SendArticleNotification(article *models.Article) error {
	notification := &models.PushNotification{
		Title:      article.Title,
		Body:       article.Excerpt,
		URL:        fmt.Sprintf("/articles/%s", article.Slug),
		TargetType: models.TargetTypeCategory,
		TargetValue: fmt.Sprintf("%d", article.CategoryID),
		Status:     models.NotificationStatusPending,
		Data: map[string]interface{}{
			"type":       "new_article",
			"article_id": article.ID,
			"category":   article.CategoryID,
		},
	}

	err := s.repo.CreateNotification(notification)
	if err != nil {
		return err
	}

	return s.SendNotification(notification.ID)
}

// Cleanup old data
func (s *PushNotificationService) CleanupOldData(olderThan time.Duration) error {
	// This would typically be implemented with a database cleanup query
	// For now, we'll just log the action
	log.Printf("Cleaning up push notification data older than %v", olderThan)
	return nil
}