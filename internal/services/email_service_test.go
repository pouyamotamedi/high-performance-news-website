package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
)

// MockEmailProvider is a mock implementation of EmailProvider
type MockEmailProvider struct {
	mock.Mock
}

func (m *MockEmailProvider) SendEmail(ctx context.Context, email *EmailMessage) (*SendResult, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*SendResult), args.Error(1)
}

func (m *MockEmailProvider) SendBulkEmails(ctx context.Context, emails []*EmailMessage) ([]*SendResult, error) {
	args := m.Called(ctx, emails)
	return args.Get(0).([]*SendResult), args.Error(1)
}

func TestEmailService_Subscribe(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.SubscribeRequest
		expectError bool
		expectEmail bool
	}{
		{
			name: "successful subscription",
			request: &models.SubscribeRequest{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Source:    "website",
			},
			expectError: false,
			expectEmail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockProvider := &MockEmailProvider{}
			config := &config.EmailConfig{
				Provider:  "sendgrid",
				FromEmail: "noreply@example.com",
				FromName:  "Test Site",
				BaseURL:   "http://localhost:8080",
				Enabled:   true,
			}

			// Mock successful email sending
			if tt.expectEmail {
				mockProvider.On("SendEmail", mock.Anything, mock.MatchedBy(func(email *EmailMessage) bool {
					return email.To == tt.request.Email && 
						   email.Subject != "" && 
						   email.HTMLContent != ""
				})).Return(&SendResult{
					MessageID: "test-message-id",
					Status:    "sent",
				}, nil)
			}

			service := &emailService{
				db:     nil, // Would use mock DB in real implementation
				config: config,
				client: mockProvider,
			}

			// Execute
			subscriber, err := service.Subscribe(context.Background(), tt.request)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, subscriber)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, subscriber)
				assert.Equal(t, tt.request.Email, subscriber.Email)
				assert.Equal(t, tt.request.FirstName, subscriber.FirstName)
				assert.Equal(t, tt.request.LastName, subscriber.LastName)
				assert.Equal(t, models.SubscriberStatusPending, subscriber.Status)
				assert.NotEmpty(t, subscriber.ConfirmationToken)
				assert.NotEmpty(t, subscriber.UnsubscribeToken)
			}

			// Verify mock expectations
			mockProvider.AssertExpectations(t)
		})
	}
}