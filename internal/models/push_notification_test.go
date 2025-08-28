package models

import (
	"testing"
	"time"
)

func TestPushNotificationValidation(t *testing.T) {
	tests := []struct {
		name        string
		notification *PushNotification
		expectError bool
	}{
		{
			name: "valid notification",
			notification: &PushNotification{
				Title:      "Test Title",
				Body:       "Test Body",
				TargetType: TargetTypeAll,
			},
			expectError: false,
		},
		{
			name: "missing title",
			notification: &PushNotification{
				Body:       "Test Body",
				TargetType: TargetTypeAll,
			},
			expectError: true,
		},
		{
			name: "missing body",
			notification: &PushNotification{
				Title:      "Test Title",
				TargetType: TargetTypeAll,
			},
			expectError: true,
		},
		{
			name: "defaults applied",
			notification: &PushNotification{
				Title: "Test Title",
				Body:  "Test Body",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.notification.Validate()
			
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
			
			// Check defaults are applied
			if !tt.expectError {
				if tt.notification.TargetType == "" {
					t.Error("TargetType should default to 'all'")
				}
				if tt.notification.Status == "" {
					t.Error("Status should default to 'pending'")
				}
			}
		})
	}
}

func TestPushSubscriptionValidation(t *testing.T) {
	tests := []struct {
		name         string
		subscription *PushSubscription
		expectError  bool
	}{
		{
			name: "valid subscription",
			subscription: &PushSubscription{
				Endpoint: "https://fcm.googleapis.com/fcm/send/test",
				P256DH:   "test-p256dh-key",
				Auth:     "test-auth-key",
			},
			expectError: false,
		},
		{
			name: "missing endpoint",
			subscription: &PushSubscription{
				P256DH: "test-p256dh-key",
				Auth:   "test-auth-key",
			},
			expectError: true,
		},
		{
			name: "missing p256dh",
			subscription: &PushSubscription{
				Endpoint: "https://fcm.googleapis.com/fcm/send/test",
				Auth:     "test-auth-key",
			},
			expectError: true,
		},
		{
			name: "missing auth",
			subscription: &PushSubscription{
				Endpoint: "https://fcm.googleapis.com/fcm/send/test",
				P256DH:   "test-p256dh-key",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.subscription.Validate()
			
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestNotificationConstants(t *testing.T) {
	// Test that constants are defined correctly
	expectedStatuses := []string{
		NotificationStatusPending,
		NotificationStatusSending,
		NotificationStatusSent,
		NotificationStatusFailed,
	}

	for _, status := range expectedStatuses {
		if status == "" {
			t.Errorf("Notification status constant is empty")
		}
	}

	expectedDeliveryStatuses := []string{
		DeliveryStatusSent,
		DeliveryStatusDelivered,
		DeliveryStatusFailed,
		DeliveryStatusClicked,
	}

	for _, status := range expectedDeliveryStatuses {
		if status == "" {
			t.Errorf("Delivery status constant is empty")
		}
	}

	expectedTargetTypes := []string{
		TargetTypeAll,
		TargetTypeCategory,
		TargetTypeTag,
		TargetTypeUser,
	}

	for _, targetType := range expectedTargetTypes {
		if targetType == "" {
			t.Errorf("Target type constant is empty")
		}
	}
}

func TestPushNotificationDefaults(t *testing.T) {
	notification := &PushNotification{
		Title: "Test",
		Body:  "Test Body",
	}

	err := notification.Validate()
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if notification.TargetType != TargetTypeAll {
		t.Errorf("Expected TargetType to default to %s, got %s", TargetTypeAll, notification.TargetType)
	}

	if notification.Status != NotificationStatusPending {
		t.Errorf("Expected Status to default to %s, got %s", NotificationStatusPending, notification.Status)
	}
}

func TestPushNotificationData(t *testing.T) {
	data := map[string]interface{}{
		"article_id": 123,
		"category":   "news",
		"urgent":     true,
	}

	notification := &PushNotification{
		Title:      "Test",
		Body:       "Test Body",
		Data:       data,
		TargetType: TargetTypeAll,
	}

	err := notification.Validate()
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Check that data is preserved
	if notification.Data["article_id"] != 123 {
		t.Error("Data not preserved correctly")
	}
	if notification.Data["category"] != "news" {
		t.Error("Data not preserved correctly")
	}
	if notification.Data["urgent"] != true {
		t.Error("Data not preserved correctly")
	}
}

func TestPushTemplateVariables(t *testing.T) {
	template := &PushTemplate{
		Name:      "article_published",
		Title:     "New Article: {{title}}",
		Body:      "{{author}} published a new article in {{category}}",
		Variables: []string{"title", "author", "category"},
		IsActive:  true,
	}

	// Test that variables are stored correctly
	if len(template.Variables) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(template.Variables))
	}

	expectedVars := []string{"title", "author", "category"}
	for i, expected := range expectedVars {
		if template.Variables[i] != expected {
			t.Errorf("Expected variable %s, got %s", expected, template.Variables[i])
		}
	}
}

func TestNotificationPreferences(t *testing.T) {
	prefs := &NotificationPreference{
		SubscriptionID:      1,
		BreakingNews:        true,
		CategoryUpdates:     true,
		TagUpdates:          false,
		AuthorUpdates:       false,
		PreferredCategories: []uint64{1, 2, 3},
		PreferredTags:       []uint64{4, 5},
		PreferredAuthors:    []uint64{6},
	}

	// Test that preferences are set correctly
	if !prefs.BreakingNews {
		t.Error("BreakingNews should be true")
	}
	if !prefs.CategoryUpdates {
		t.Error("CategoryUpdates should be true")
	}
	if prefs.TagUpdates {
		t.Error("TagUpdates should be false")
	}
	if prefs.AuthorUpdates {
		t.Error("AuthorUpdates should be false")
	}

	if len(prefs.PreferredCategories) != 3 {
		t.Errorf("Expected 3 preferred categories, got %d", len(prefs.PreferredCategories))
	}
	if len(prefs.PreferredTags) != 2 {
		t.Errorf("Expected 2 preferred tags, got %d", len(prefs.PreferredTags))
	}
	if len(prefs.PreferredAuthors) != 1 {
		t.Errorf("Expected 1 preferred author, got %d", len(prefs.PreferredAuthors))
	}
}

func TestPushDeliveryTracking(t *testing.T) {
	now := time.Now()
	
	delivery := &PushDelivery{
		NotificationID: 1,
		SubscriptionID: 1,
		Status:         DeliveryStatusSent,
		DeliveredAt:    &now,
	}

	if delivery.Status != DeliveryStatusSent {
		t.Errorf("Expected status %s, got %s", DeliveryStatusSent, delivery.Status)
	}

	if delivery.DeliveredAt == nil {
		t.Error("DeliveredAt should not be nil")
	}

	if delivery.DeliveredAt.Unix() != now.Unix() {
		t.Error("DeliveredAt timestamp mismatch")
	}
}

// Helper function for tests (assuming it exists in models package)
func NewValidationError(message string) error {
	// This would be implemented in your actual models package
	return &ValidationError{Message: message}
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}