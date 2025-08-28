package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"your-project/internal/models"
	"your-project/internal/services"
)

// Mock service for testing
type mockPushNotificationService struct {
	subscriptions map[string]*models.PushSubscription
	notifications map[uint64]*models.PushNotification
	templates     map[string]*models.PushTemplate
	preferences   map[uint64]*models.NotificationPreference
	nextID        uint64
}

func newMockPushNotificationService() *mockPushNotificationService {
	return &mockPushNotificationService{
		subscriptions: make(map[string]*models.PushSubscription),
		notifications: make(map[uint64]*models.PushNotification),
		templates:     make(map[string]*models.PushTemplate),
		preferences:   make(map[uint64]*models.NotificationPreference),
		nextID:        1,
	}
}

func (m *mockPushNotificationService) Subscribe(subscription *models.PushSubscription) error {
	m.nextID++
	subscription.ID = m.nextID
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()
	m.subscriptions[subscription.Endpoint] = subscription
	return nil
}

func (m *mockPushNotificationService) Unsubscribe(endpoint string) error {
	if sub, exists := m.subscriptions[endpoint]; exists {
		sub.IsActive = false
		return nil
	}
	return &NotFoundError{Message: "subscription not found"}
}

func (m *mockPushNotificationService) GetSubscription(endpoint string) (*models.PushSubscription, error) {
	if sub, exists := m.subscriptions[endpoint]; exists {
		return sub, nil
	}
	return nil, &NotFoundError{Message: "subscription not found"}
}

func (m *mockPushNotificationService) CreateNotification(notification *models.PushNotification) error {
	m.nextID++
	notification.ID = m.nextID
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()
	m.notifications[notification.ID] = notification
	return nil
}func
 (m *mockPushNotificationService) SendNotification(id uint64) error {
	if notif, exists := m.notifications[id]; exists {
		notif.Status = models.NotificationStatusSent
		now := time.Now()
		notif.SentAt = &now
		return nil
	}
	return &NotFoundError{Message: "notification not found"}
}

func (m *mockPushNotificationService) GetNotification(id uint64) (*models.PushNotification, error) {
	if notif, exists := m.notifications[id]; exists {
		return notif, nil
	}
	return nil, &NotFoundError{Message: "notification not found"}
}

func (m *mockPushNotificationService) CreateTemplate(template *models.PushTemplate) error {
	m.nextID++
	template.ID = m.nextID
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	m.templates[template.Name] = template
	return nil
}

func (m *mockPushNotificationService) GetTemplate(name string) (*models.PushTemplate, error) {
	if template, exists := m.templates[name]; exists {
		return template, nil
	}
	return &NotFoundError{Message: "template not found"}
}

func (m *mockPushNotificationService) GetActiveTemplates() ([]*models.PushTemplate, error) {
	var active []*models.PushTemplate
	for _, template := range m.templates {
		if template.IsActive {
			active = append(active, template)
		}
	}
	return active, nil
}

func (m *mockPushNotificationService) UpdatePreferences(prefs *models.NotificationPreference) error {
	m.nextID++
	prefs.ID = m.nextID
	prefs.CreatedAt = time.Now()
	prefs.UpdatedAt = time.Now()
	m.preferences[prefs.SubscriptionID] = prefs
	return nil
}

func (m *mockPushNotificationService) GetPreferences(subscriptionID uint64) (*models.NotificationPreference, error) {
	if prefs, exists := m.preferences[subscriptionID]; exists {
		return prefs, nil
	}
	return nil, &NotFoundError{Message: "preferences not found"}
}

func (m *mockPushNotificationService) TrackDelivery(deliveryID uint64) error {
	return nil // Mock implementation
}

func (m *mockPushNotificationService) TrackClick(deliveryID uint64) error {
	return nil // Mock implementation
}

// Test setup
func setupTestRouter() (*gin.Engine, *mockPushNotificationService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockService := newMockPushNotificationService()
	handlers := NewPushNotificationHandlers(mockService)
	
	api := router.Group("/api/v1")
	handlers.RegisterRoutes(api)
	
	return router, mockService
}

// Mock middleware for testing
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock authentication - always pass
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock role check - always pass
		c.Next()
	}
}

// Test functions
func TestPushNotificationHandlers_Subscribe(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := map[string]interface{}{
		"endpoint": "https://fcm.googleapis.com/fcm/send/test",
		"p256dh":   "test-p256dh",
		"auth":     "test-auth",
		"user_id":  1,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Subscription created successfully" {
		t.Error("Expected success message")
	}

	if response["subscription_id"] == nil {
		t.Error("Expected subscription_id in response")
	}
}

func TestPushNotificationHandlers_Subscribe_InvalidData(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := map[string]interface{}{
		"endpoint": "", // Missing endpoint
		"p256dh":   "test-p256dh",
		"auth":     "test-auth",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPushNotificationHandlers_Unsubscribe(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create subscription first
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	mockService.Subscribe(subscription)

	reqBody := map[string]interface{}{
		"endpoint": "https://fcm.googleapis.com/fcm/send/test",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/unsubscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Unsubscribed successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_UpdatePreferences(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create subscription first
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	mockService.Subscribe(subscription)

	reqBody := map[string]interface{}{
		"subscription_id":      subscription.ID,
		"breaking_news":        true,
		"category_updates":     false,
		"tag_updates":          true,
		"author_updates":       false,
		"preferred_categories": []uint64{1, 2, 3},
		"preferred_tags":       []uint64{4, 5},
		"preferred_authors":    []uint64{6},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/preferences", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Preferences updated successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_GetPreferences(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create subscription and preferences
	subscription := &models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test",
		P256DH:   "test-p256dh",
		Auth:     "test-auth",
		IsActive: true,
	}
	mockService.Subscribe(subscription)

	prefs := &models.NotificationPreference{
		SubscriptionID:  subscription.ID,
		BreakingNews:    true,
		CategoryUpdates: false,
	}
	mockService.UpdatePreferences(prefs)

	req := httptest.NewRequest("GET", "/api/v1/push/preferences/"+fmt.Sprintf("%d", subscription.ID), nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.NotificationPreference
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.BreakingNews != true {
		t.Error("Expected breaking_news to be true")
	}
}

func TestPushNotificationHandlers_TrackDelivery(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest("POST", "/api/v1/push/track/delivery/123", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Delivery tracked successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_TrackClick(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest("POST", "/api/v1/push/track/click/123", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Click tracked successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_CreateNotification(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := map[string]interface{}{
		"title":       "Test Notification",
		"body":        "Test Body",
		"target_type": "all",
		"send_now":    false,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/admin/notifications", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Notification created successfully" {
		t.Error("Expected success message")
	}

	if response["notification_id"] == nil {
		t.Error("Expected notification_id in response")
	}
}

func TestPushNotificationHandlers_GetNotification(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create notification first
	notification := &models.PushNotification{
		Title:      "Test Notification",
		Body:       "Test Body",
		TargetType: models.TargetTypeAll,
		Status:     models.NotificationStatusPending,
	}
	mockService.CreateNotification(notification)

	req := httptest.NewRequest("GET", "/api/v1/push/admin/notifications/"+fmt.Sprintf("%d", notification.ID), nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PushNotification
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Title != "Test Notification" {
		t.Errorf("Expected title 'Test Notification', got %s", response.Title)
	}
}

func TestPushNotificationHandlers_SendNotification(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create notification first
	notification := &models.PushNotification{
		Title:      "Test Notification",
		Body:       "Test Body",
		TargetType: models.TargetTypeAll,
		Status:     models.NotificationStatusPending,
	}
	mockService.CreateNotification(notification)

	req := httptest.NewRequest("POST", "/api/v1/push/admin/notifications/"+fmt.Sprintf("%d", notification.ID)+"/send", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Notification sent successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_CreateTemplate(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := map[string]interface{}{
		"name":      "test_template",
		"title":     "Test {{title}}",
		"body":      "Body {{body}}",
		"variables": []string{"title", "body"},
		"is_active": true,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/push/admin/templates", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Template created successfully" {
		t.Error("Expected success message")
	}
}

func TestPushNotificationHandlers_GetTemplate(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create template first
	template := &models.PushTemplate{
		Name:      "test_template",
		Title:     "Test {{title}}",
		Body:      "Body {{body}}",
		Variables: []string{"title", "body"},
		IsActive:  true,
	}
	mockService.CreateTemplate(template)

	req := httptest.NewRequest("GET", "/api/v1/push/admin/templates/test_template", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.PushTemplate
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Name != "test_template" {
		t.Errorf("Expected name 'test_template', got %s", response.Name)
	}
}

func TestPushNotificationHandlers_ListTemplates(t *testing.T) {
	router, mockService := setupTestRouter()

	// Create templates
	template1 := &models.PushTemplate{
		Name:     "template1",
		Title:    "Template 1",
		Body:     "Body 1",
		IsActive: true,
	}
	template2 := &models.PushTemplate{
		Name:     "template2",
		Title:    "Template 2",
		Body:     "Body 2",
		IsActive: true,
	}
	mockService.CreateTemplate(template1)
	mockService.CreateTemplate(template2)

	req := httptest.NewRequest("GET", "/api/v1/push/admin/templates", nil)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	templates := response["templates"].([]interface{})
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}

	count := response["count"].(float64)
	if int(count) != 2 {
		t.Errorf("Expected count 2, got %d", int(count))
	}
}

// Error type for testing
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

