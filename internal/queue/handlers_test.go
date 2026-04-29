package queue

import (
	"context"
	"testing"
)

func TestStaticGenerationHandler(t *testing.T) {
	handler := NewStaticGenerationHandler()

	t.Run("GetJobType", func(t *testing.T) {
		if handler.GetJobType() != JobTypeStaticGeneration {
			t.Errorf("Expected job type %s, got %s", JobTypeStaticGeneration, handler.GetJobType())
		}
	})

	t.Run("HandleValidJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "static-gen-test",
			Type: JobTypeStaticGeneration,
			Payload: map[string]interface{}{
				"article_id": 123,
				"page_type":  "article",
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleMissingArticleID", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:      "static-gen-test-missing",
			Type:    JobTypeStaticGeneration,
			Payload: map[string]interface{}{},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing article_id")
		}
	})

	t.Run("HandleContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		job := &Job{
			ID:   "static-gen-cancelled",
			Type: JobTypeStaticGeneration,
			Payload: map[string]interface{}{
				"article_id": 123,
			},
		}

		err := handler.Handle(ctx, job)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}

func TestImageProcessingHandler(t *testing.T) {
	handler := NewImageProcessingHandler()

	t.Run("GetJobType", func(t *testing.T) {
		if handler.GetJobType() != JobTypeImageProcessing {
			t.Errorf("Expected job type %s, got %s", JobTypeImageProcessing, handler.GetJobType())
		}
	})

	t.Run("HandleValidJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "image-proc-test",
			Type: JobTypeImageProcessing,
			Payload: map[string]interface{}{
				"image_path": "/path/to/image.jpg",
				"formats":    []interface{}{"webp", "avif"},
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleMissingImagePath", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:      "image-proc-missing",
			Type:    JobTypeImageProcessing,
			Payload: map[string]interface{}{},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing image_path")
		}
	})
}

func TestSearchIndexingHandler(t *testing.T) {
	handler := NewSearchIndexingHandler()

	t.Run("GetJobType", func(t *testing.T) {
		if handler.GetJobType() != JobTypeSearchIndexing {
			t.Errorf("Expected job type %s, got %s", JobTypeSearchIndexing, handler.GetJobType())
		}
	})

	t.Run("HandleValidIndexJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "search-index-test",
			Type: JobTypeSearchIndexing,
			Payload: map[string]interface{}{
				"document_id": 456,
				"action":      "index",
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleValidUpdateJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "search-update-test",
			Type: JobTypeSearchIndexing,
			Payload: map[string]interface{}{
				"document_id": 456,
				"action":      "update",
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleValidDeleteJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "search-delete-test",
			Type: JobTypeSearchIndexing,
			Payload: map[string]interface{}{
				"document_id": 456,
				"action":      "delete",
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleInvalidAction", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "search-invalid-test",
			Type: JobTypeSearchIndexing,
			Payload: map[string]interface{}{
				"document_id": 456,
				"action":      "invalid_action",
			},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for invalid action")
		}
	})

	t.Run("HandleMissingDocumentID", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:      "search-missing-test",
			Type:    JobTypeSearchIndexing,
			Payload: map[string]interface{}{},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing document_id")
		}
	})
}

func TestNotificationsHandler(t *testing.T) {
	handler := NewNotificationsHandler()

	t.Run("GetJobType", func(t *testing.T) {
		if handler.GetJobType() != JobTypeNotifications {
			t.Errorf("Expected job type %s, got %s", JobTypeNotifications, handler.GetJobType())
		}
	})

	t.Run("HandleValidJob", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "notification-test",
			Type: JobTypeNotifications,
			Payload: map[string]interface{}{
				"type":       "email",
				"recipients": []interface{}{"user1@example.com", "user2@example.com"},
				"message":    "Test notification",
			},
		}

		err := handler.Handle(ctx, job)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("HandleMissingType", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "notification-missing-type",
			Type: JobTypeNotifications,
			Payload: map[string]interface{}{
				"recipients": []interface{}{"user1@example.com"},
				"message":    "Test notification",
			},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing notification type")
		}
	})

	t.Run("HandleMissingRecipients", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "notification-missing-recipients",
			Type: JobTypeNotifications,
			Payload: map[string]interface{}{
				"type":    "email",
				"message": "Test notification",
			},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing recipients")
		}
	})

	t.Run("HandleMissingMessage", func(t *testing.T) {
		ctx := context.Background()
		job := &Job{
			ID:   "notification-missing-message",
			Type: JobTypeNotifications,
			Payload: map[string]interface{}{
				"type":       "email",
				"recipients": []interface{}{"user1@example.com"},
			},
		}

		err := handler.Handle(ctx, job)
		if err == nil {
			t.Error("Expected error for missing message")
		}
	})
}

func TestJobHandlerRegistry(t *testing.T) {
	registry := NewJobHandlerRegistry()

	t.Run("RegisterAndGet", func(t *testing.T) {
		handler := NewStaticGenerationHandler()
		registry.Register(handler)

		retrievedHandler, exists := registry.Get(JobTypeStaticGeneration)
		if !exists {
			t.Error("Expected handler to exist")
		}
		if retrievedHandler != handler {
			t.Error("Expected same handler instance")
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, exists := registry.Get(JobType("non_existent"))
		if exists {
			t.Error("Expected handler not to exist")
		}
	})

	t.Run("RegisterDefaultHandlers", func(t *testing.T) {
		registry.RegisterDefaultHandlers()

		expectedTypes := []JobType{
			JobTypeStaticGeneration,
			JobTypeImageProcessing,
			JobTypeSearchIndexing,
			JobTypeNotifications,
		}

		for _, jobType := range expectedTypes {
			_, exists := registry.Get(jobType)
			if !exists {
				t.Errorf("Expected default handler for %s to be registered", jobType)
			}
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		registry.RegisterDefaultHandlers()
		handlers := registry.GetAll()

		if len(handlers) != 4 {
			t.Errorf("Expected 4 handlers, got %d", len(handlers))
		}
	})
}