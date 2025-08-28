package services

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// Mock repository for testing
type mockPushNotificationRepository struct {
	subscriptions   map[string]*models.PushSubscription
	notifications   map[uint64]*models.PushNotification
	deliveries      map[uint64]*models.PushDelivery
	templates       map[string]*models.PushTemplate
	preferences     map[uint64]*models.NotificationPreference
	nextID          uint64
}

func newMockPushNotificationRepository() *mockPushNotificationRepository {
	return &mockPushNotificationRepository{
		subscriptions: make(map[string]*models.PushSubscription),
		notifications: make(map[uint64]*models.PushNotification),
		deliveries:    make(map[uint64]*models.PushDelivery),
		templates:     make(map[string]*models.PushTemplate),
		preferences:   make(map[uint64]*models.NotificationPreference),
		nextID:        1,
	}
}

func (m *mockPushNotificationRepository) CreateSubscription(subscription *models.PushSubscription) error {
	m.nextID++
	subscription.ID = m.nextID
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()
	m.subscriptions[subscription.Endpoint] = subscription
	return nil
}

func (m *mockPushNotificationRepository) GetSubscriptionByEndpoint(endpoint string) (*models.PushSubscription, error) {
	if sub, exists := m.subscriptions[endpoint]; exists {
		return sub, nil
	}
	return nil, &NotFoundError{Message: "subscription not found"}
}

func (m *mockPushNotificationRepository) GetActiveSubscriptions() ([]*models.PushSubscription, error) {
	var active []*models.PushSubscription
	for _, sub := range m.subscriptions {
		if sub.IsActive {
			active = append(active, sub)
		}
	}
	return active, nil
}

func (m *mockPushNotificationRepository) GetTargetedSubscriptions(targetType, targetValue string) ([]*models.PushSubscription, error) {
	// Simplified targeting for tests
	return m.GetActiveSubscriptions()
}

func (m *mockPushNotificationRepository) DeactivateSubscription(endpoint string) error {
	if sub, exists := m.subscriptions[endpoint]; exists {
		sub.IsActive = false
		sub.UpdatedAt = time.Now()
		return nil
	}
	return &NotFoundError{Message: "subscription not found"}
}

func (m *mockPushNotificationRepository) CreateNotification(notification *models.PushNotification) error {
	m.nextID++
	notification.ID = m.nextID
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockPushNotificationRepository) GetNotificationByID(id uint64) (*models.PushNotification, error) {
	if notif, exists := m.notifications[id]; exists {
		return notif, nil
	}
	return nil, &NotFoundError{Message: "notification not found"}
}

func (m *mockPushNotificationRepository) GetPendingNotifications() ([]*models.PushNotification, error) {
	var pending []*models.PushNotification
	for _, notif := range m.notifications {
		if notif.Status == models.NotificationStatusPending {
			pending = append(pending, notif)
		}
	}
	return pending, nil
}

func (m *mockPushNotificationRepository) UpdateNotificationStatus(id uint64, status string, totalSent int) error {
	if notif, exists := m.notifications[id]; exists {
		notif.Status = status
		notif.TotalSent = totalSent
		if status == models.NotificationStatusSent {
			now := time.Now()
			notif.SentAt = &now
		}
		notif.UpdatedAt = time.Now()
		return nil
	}
	return &NotFoundError{Message: "notification not found"}
}

func (m *mockPushNotificationRepository) UpdateNotificationStats(id uint64, totalDelivered, totalClicked int) error {
	if notif, exists := m.notifications[id]; exists {
		notif.TotalDelivered = totalDelivered
		notif.TotalClicked = totalClicked
		notif.UpdatedAt = time.Now()
		return nil
	}
	return &NotFoundError{Message: "notification not found"}
}

func (m *mockPushNotificationRepository) CreateDelivery(delivery *models.PushDelivery) error {
	m.nextID++
	delivery.ID = m.nextID
	delivery.CreatedAt = time.Now()
	delivery.UpdatedAt = time.Now()
	m.deliveries[delivery.ID] = delivery
	return nil
}

func (m *mockPushNotificationRepository) UpdateDeliveryStatus(id uint64, status string) error {
	if delivery, exists := m.deliveries[id]; exists {
		delivery.Status = status
		now := time.Now()
		switch status {
		case models.DeliveryStatusDelivered:
			delivery.DeliveredAt = &now
		case models.DeliveryStatusClicked:
			delivery.ClickedAt = &now
		}
		delivery.UpdatedAt = now
		return nil
	}
	return &NotFoundError{Message: "delivery not found"}
}

func (m *mockPushNotificationRepository) GetDeliveryStats(notificationID uint64) (delivered, clicked int, err error) {
	for _, delivery := range m.deliveries {
		if delivery.NotificationID == notificationID {
			if delivery.Status == models.DeliveryStatusDelivered || delivery.Status == models.DeliveryStatusClicked {
				delivered++
			}
			if delivery.Status == models.DeliveryStatusClicked {
				clicked++
			}
		}
	}
	return delivered, clicked, nil
}

func (m *mockPushNotificationRepository) CreateTemplate(template *models.PushTemplate) error {
	m.nextID++
	template.ID = m.nextID
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	m.templates[template.Name] = template
	return nil
}

func (m *mockPushNotificationRepository) GetTemplateByName(name string) (*models.PushTemplate, error) {
	if template, exists := m.templates[name]; exists && template.IsActive {
		return template, nil
	}
	return nil, &NotFoundError{Message: "template not found"}
}

func (m *mockPushNotificationRepository) GetActiveTemplates() ([]*models.PushTemplate, error) {
	var active []*models.PushTemplate
	for _, template := range m.templates {
		if template.IsActive {
			active = append(active, template)
		}
	}
	return active, nil
}

func (m *mockPushNotificationRepository) CreateOrUpdatePreferences(prefs *models.NotificationPreference) error {
	m.nextID++
	prefs.ID = m.nextID
	prefs.CreatedAt = time.Now()
	prefs.UpdatedAt = time.Now()
	m.preferences[prefs.SubscriptionID] = prefs
	return nil
}

func (m *mockPushNotificationRepository) GetPreferencesBySubscription(subscriptionID uint64) (*models.NotificationPreference, error) {
	if prefs, exists := m.preferences[subscriptionID]; exists {
		return prefs, nil
	}
	return nil, &NotFoundError{Message: "preferences not found"}
}

// Error types for testing
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// Test functions
func TestPushNotificationService_Subscribe(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}

	err := service.Subscribe(subscription)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if subscription.ID == 0 {
		t.Error("Subscription ID should be set")
	}

	// Verify subscription was stored
	stored, err := service.GetSubscription(subscription.Endpoint)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if stored.Endpoint != subscription.Endpoint {
		t.Errorf("Expected endpoint %s, got %s", subscription.Endpoint, stored.Endpoint)
	}
}

func TestPushNotificationService_Unsubscribe(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Create subscription first
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	service.Subscribe(subscription)

	// Unsubscribe
	err := service.Unsubscribe(subscription.Endpoint)
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	// Verify subscription is deactivated
	stored, err := service.GetSubscription(subscription.Endpoint)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if stored.IsActive {
		t.Error("Subscription should be deactivated")
	}
}

func TestPushNotificationService_CreateNotification(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	notification := &models.PushNotification{
		Title:      "Test Notification",
		Body:       "Test Body",
		TargetType: models.TargetTypeAll,
	}

	err := service.CreateNotification(notification)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	if notification.ID == 0 {
		t.Error("Notification ID should be set")
	}

	if notification.Status != models.NotificationStatusPending {
		t.Errorf("Expected status %s, got %s", models.NotificationStatusPending, notification.Status)
	}
}

func TestPushNotificationService_CreateFromTemplate(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Create template first
	template := &models.PushTemplate{
		Name:      "article_published",
		Title:     "New Article: {{title}}",
		Body:      "{{author}} published: {{title}}",
		Variables: []string{"title", "author"},
		IsActive:  true,
	}
	service.CreateTemplate(template)

	// Create notification from template
	variables := map[string]string{
		"title":  "Breaking News",
		"author": "John Doe",
	}

	err := service.CreateFromTemplate("article_published", variables, models.TargetTypeAll, "", nil)
	if err != nil {
		t.Fatalf("CreateFromTemplate failed: %v", err)
	}

	// Verify notification was created with replaced variables
	notifications := repo.notifications
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	var notification *models.PushNotification
	for _, n := range notifications {
		notification = n
		break
	}

	expectedTitle := "New Article: Breaking News"
	if notification.Title != expectedTitle {
		t.Errorf("Expected title %s, got %s", expectedTitle, notification.Title)
	}

	expectedBody := "John Doe published: Breaking News"
	if notification.Body != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, notification.Body)
	}
}

func TestPushNotificationService_SendNotification(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "", "", "") // No external service configured

	// Create subscription
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	service.Subscribe(subscription)

	// Create notification
	notification := &models.PushNotification{
		Title:      "Test Notification",
		Body:       "Test Body",
		TargetType: models.TargetTypeAll,
	}
	service.CreateNotification(notification)

	// Send notification (will fail due to no external service, but should update status)
	err := service.SendNotification(notification.ID)
	
	// We expect this to fail since no external service is configured
	// but the notification status should be updated
	stored, _ := repo.GetNotificationByID(notification.ID)
	if stored.Status == models.NotificationStatusPending {
		t.Error("Notification status should have been updated from pending")
	}
}

func TestPushNotificationService_ProcessPendingNotifications(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "", "", "")

	// Create multiple pending notifications
	for i := 0; i < 3; i++ {
		notification := &models.PushNotification{
			Title:      "Test Notification",
			Body:       "Test Body",
			TargetType: models.TargetTypeAll,
		}
		service.CreateNotification(notification)
	}

	// Process pending notifications
	ctx := context.Background()
	err := service.ProcessPendingNotifications(ctx)
	if err != nil {
		t.Fatalf("ProcessPendingNotifications failed: %v", err)
	}

	// Verify all notifications were processed
	pending, _ := repo.GetPendingNotifications()
	if len(pending) > 0 {
		t.Errorf("Expected 0 pending notifications, got %d", len(pending))
	}
}

func TestPushNotificationService_TemplateManagement(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Create template
	template := &models.PushTemplate{
		Name:      "test_template",
		Title:     "Test {{title}}",
		Body:      "Body {{body}}",
		Variables: []string{"title", "body"},
		IsActive:  true,
	}

	err := service.CreateTemplate(template)
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}

	// Get template
	retrieved, err := service.GetTemplate("test_template")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	if retrieved.Name != template.Name {
		t.Errorf("Expected name %s, got %s", template.Name, retrieved.Name)
	}

	// Get active templates
	templates, err := service.GetActiveTemplates()
	if err != nil {
		t.Fatalf("GetActiveTemplates failed: %v", err)
	}

	if len(templates) != 1 {
		t.Errorf("Expected 1 active template, got %d", len(templates))
	}
}

func TestPushNotificationService_PreferenceManagement(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Create subscription first
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	service.Subscribe(subscription)

	// Update preferences
	prefs := &models.NotificationPreference{
		SubscriptionID:      subscription.ID,
		BreakingNews:        true,
		CategoryUpdates:     false,
		TagUpdates:          true,
		AuthorUpdates:       false,
		PreferredCategories: []uint64{1, 2, 3},
	}

	err := service.UpdatePreferences(prefs)
	if err != nil {
		t.Fatalf("UpdatePreferences failed: %v", err)
	}

	// Get preferences
	retrieved, err := service.GetPreferences(subscription.ID)
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}

	if retrieved.BreakingNews != prefs.BreakingNews {
		t.Error("BreakingNews preference not saved correctly")
	}

	if len(retrieved.PreferredCategories) != 3 {
		t.Errorf("Expected 3 preferred categories, got %d", len(retrieved.PreferredCategories))
	}
}

func TestPushNotificationService_DeliveryTracking(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Create delivery
	delivery := &models.PushDelivery{
		NotificationID: 1,
		SubscriptionID: 1,
		Status:         models.DeliveryStatusSent,
	}
	repo.CreateDelivery(delivery)

	// Track delivery
	err := service.TrackDelivery(delivery.ID)
	if err != nil {
		t.Fatalf("TrackDelivery failed: %v", err)
	}

	// Verify status updated
	updated := repo.deliveries[delivery.ID]
	if updated.Status != models.DeliveryStatusDelivered {
		t.Errorf("Expected status %s, got %s", models.DeliveryStatusDelivered, updated.Status)
	}

	// Track click
	err = service.TrackClick(delivery.ID)
	if err != nil {
		t.Fatalf("TrackClick failed: %v", err)
	}

	// Verify status updated
	updated = repo.deliveries[delivery.ID]
	if updated.Status != models.DeliveryStatusClicked {
		t.Errorf("Expected status %s, got %s", models.DeliveryStatusClicked, updated.Status)
	}
}

func TestPushNotificationService_ConvenienceMethods(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "", "", "")

	// Test SendBreakingNews
	err := service.SendBreakingNews("Breaking News", "Important update", "/news/breaking")
	// This will fail due to no external service, but notification should be created
	
	notifications := repo.notifications
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	var notification *models.PushNotification
	for _, n := range notifications {
		notification = n
		break
	}

	if notification.Title != "Breaking News" {
		t.Errorf("Expected title 'Breaking News', got %s", notification.Title)
	}

	if notification.Data["type"] != "breaking_news" {
		t.Error("Expected data type to be 'breaking_news'")
	}
}

func TestPushNotificationService_ReplaceVariables(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	text := "Hello {{name}}, you have {{count}} new messages in {{category}}"
	variables := map[string]string{
		"name":     "John",
		"count":    "5",
		"category": "News",
	}

	result := service.replaceVariables(text, variables)
	expected := "Hello John, you have 5 new messages in News"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPushNotificationService_ValidationErrors(t *testing.T) {
	repo := newMockPushNotificationRepository()
	service := NewPushNotificationService(repo, "test-app-id", "test-key", "test-firebase-key")

	// Test invalid subscription
	invalidSub := &models.PushSubscription{
		Endpoint: "", // Missing endpoint
		P256DH:   "test",
		Auth:     "test",
	}

	err := service.Subscribe(invalidSub)
	if err == nil {
		t.Error("Expected validation error for invalid subscription")
	}

	// Test invalid notification
	invalidNotif := &models.PushNotification{
		Title: "", // Missing title
		Body:  "Test",
	}

	err = service.CreateNotification(invalidNotif)
	if err == nil {
		t.Error("Expected validation error for invalid notification")
	}
}