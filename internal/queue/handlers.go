package queue

import (
	"context"
	"fmt"
	"log"
	"time"
)

// StaticGenerationHandler handles static HTML generation jobs
type StaticGenerationHandler struct{}

func NewStaticGenerationHandler() *StaticGenerationHandler {
	return &StaticGenerationHandler{}
}

func (h *StaticGenerationHandler) Handle(ctx context.Context, job *Job) error {
	log.Printf("Processing static generation job %s", job.ID)
	
	// Extract payload data
	articleID, ok := job.Payload["article_id"]
	if !ok {
		return fmt.Errorf("missing article_id in payload")
	}

	pageType, ok := job.Payload["page_type"].(string)
	if !ok {
		pageType = "article"
	}

	// Simulate static generation work
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		// Actual implementation would generate static HTML files
		log.Printf("Generated static %s page for article %v", pageType, articleID)
		return nil
	}
}

func (h *StaticGenerationHandler) GetJobType() JobType {
	return JobTypeStaticGeneration
}

// ImageProcessingHandler handles image processing jobs
type ImageProcessingHandler struct{}

func NewImageProcessingHandler() *ImageProcessingHandler {
	return &ImageProcessingHandler{}
}

func (h *ImageProcessingHandler) Handle(ctx context.Context, job *Job) error {
	log.Printf("Processing image processing job %s", job.ID)
	
	// Extract payload data
	imagePath, ok := job.Payload["image_path"].(string)
	if !ok {
		return fmt.Errorf("missing image_path in payload")
	}

	formats, ok := job.Payload["formats"].([]interface{})
	if !ok {
		formats = []interface{}{"webp", "avif", "jpeg"}
	}

	// Simulate image processing work
	for _, format := range formats {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			log.Printf("Generated %s format for image %s", format, imagePath)
		}
	}

	log.Printf("Completed image processing for %s", imagePath)
	return nil
}

func (h *ImageProcessingHandler) GetJobType() JobType {
	return JobTypeImageProcessing
}

// SearchIndexingHandler handles search index updates
type SearchIndexingHandler struct{}

func NewSearchIndexingHandler() *SearchIndexingHandler {
	return &SearchIndexingHandler{}
}

func (h *SearchIndexingHandler) Handle(ctx context.Context, job *Job) error {
	log.Printf("Processing search indexing job %s", job.ID)
	
	// Extract payload data
	documentID, ok := job.Payload["document_id"]
	if !ok {
		return fmt.Errorf("missing document_id in payload")
	}

	action, ok := job.Payload["action"].(string)
	if !ok {
		action = "index"
	}

	// Simulate search indexing work
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		switch action {
		case "index":
			log.Printf("Indexed document %v in search engine", documentID)
		case "update":
			log.Printf("Updated document %v in search engine", documentID)
		case "delete":
			log.Printf("Deleted document %v from search engine", documentID)
		default:
			return fmt.Errorf("unknown search action: %s", action)
		}
		return nil
	}
}

func (h *SearchIndexingHandler) GetJobType() JobType {
	return JobTypeSearchIndexing
}

// NotificationsHandler handles notification sending jobs
type NotificationsHandler struct{}

func NewNotificationsHandler() *NotificationsHandler {
	return &NotificationsHandler{}
}

func (h *NotificationsHandler) Handle(ctx context.Context, job *Job) error {
	log.Printf("Processing notifications job %s", job.ID)
	
	// Extract payload data
	notificationType, ok := job.Payload["type"].(string)
	if !ok {
		return fmt.Errorf("missing notification type in payload")
	}

	recipients, ok := job.Payload["recipients"].([]interface{})
	if !ok {
		return fmt.Errorf("missing recipients in payload")
	}

	message, ok := job.Payload["message"].(string)
	if !ok {
		return fmt.Errorf("missing message in payload")
	}

	// Simulate notification sending
	for i, recipient := range recipients {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			log.Printf("Sent %s notification to recipient %d: %v (message: %s)", 
				notificationType, i+1, recipient, message)
		}
	}

	log.Printf("Completed sending %d %s notifications", len(recipients), notificationType)
	return nil
}

func (h *NotificationsHandler) GetJobType() JobType {
	return JobTypeNotifications
}

// JobHandlerRegistry manages job handlers
type JobHandlerRegistry struct {
	handlers map[JobType]JobHandler
}

func NewJobHandlerRegistry() *JobHandlerRegistry {
	return &JobHandlerRegistry{
		handlers: make(map[JobType]JobHandler),
	}
}

func (r *JobHandlerRegistry) Register(handler JobHandler) {
	r.handlers[handler.GetJobType()] = handler
}

func (r *JobHandlerRegistry) Get(jobType JobType) (JobHandler, bool) {
	handler, exists := r.handlers[jobType]
	return handler, exists
}

func (r *JobHandlerRegistry) GetAll() map[JobType]JobHandler {
	result := make(map[JobType]JobHandler)
	for k, v := range r.handlers {
		result[k] = v
	}
	return result
}

// RegisterDefaultHandlers registers all default job handlers
func (r *JobHandlerRegistry) RegisterDefaultHandlers() {
	r.Register(NewStaticGenerationHandler())
	r.Register(NewImageProcessingHandler())
	r.Register(NewSearchIndexingHandler())
	r.Register(NewNotificationsHandler())
}