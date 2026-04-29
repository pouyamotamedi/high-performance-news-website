package queue

import (
	"fmt"
	"log"
	"strconv"
)

// SocialMediaService interface to avoid import cycles
type SocialMediaService interface {
	PublishPost(postID uint64) error
	CreateFacebookInstantArticle(articleID uint64) error
}

// SocialMediaProcessor handles social media job processing
type SocialMediaProcessor struct {
	socialService SocialMediaService
}

// NewSocialMediaProcessor creates a new social media processor
func NewSocialMediaProcessor(socialService SocialMediaService) *SocialMediaProcessor {
	return &SocialMediaProcessor{
		socialService: socialService,
	}
}

// ProcessJob processes social media related jobs
func (p *SocialMediaProcessor) ProcessJob(job *Job) error {
	switch job.Type {
	case "social_media_post":
		return p.processSocialMediaPost(job)
	case "webhook_processing":
		return p.processWebhook(job)
	case "facebook_instant_article":
		return p.processFacebookInstantArticle(job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// processSocialMediaPost processes a social media posting job
func (p *SocialMediaProcessor) processSocialMediaPost(job *Job) error {
	// Extract post ID from job payload
	postIDInterface, exists := job.Payload["post_id"]
	if !exists {
		return fmt.Errorf("post_id not found in job payload")
	}

	var postID uint64
	switch v := postIDInterface.(type) {
	case float64:
		postID = uint64(v)
	case int:
		postID = uint64(v)
	case int64:
		postID = uint64(v)
	case uint64:
		postID = v
	case string:
		var err error
		postID, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid post_id format: %v", v)
		}
	default:
		return fmt.Errorf("invalid post_id type: %T", v)
	}

	// Check if this is a retry
	isRetry, _ := job.Payload["retry"].(bool)
	if isRetry {
		log.Printf("Retrying social media post %d", postID)
	}

	// Publish the post
	if err := p.socialService.PublishPost(postID); err != nil {
		log.Printf("Failed to publish social media post %d: %v", postID, err)
		return err
	}

	log.Printf("Successfully published social media post %d", postID)
	return nil
}

// processWebhook processes a webhook event
func (p *SocialMediaProcessor) processWebhook(job *Job) error {
	webhookIDInterface, exists := job.Payload["webhook_id"]
	if !exists {
		return fmt.Errorf("webhook_id not found in job payload")
	}

	webhookID, ok := webhookIDInterface.(uint64)
	if !ok {
		return fmt.Errorf("invalid webhook_id type: %T", webhookIDInterface)
	}

	// Process the webhook (implementation depends on specific webhook handling logic)
	log.Printf("Processing webhook %d", webhookID)
	
	// For now, just mark as processed
	// In a real implementation, you would:
	// 1. Retrieve the webhook from database
	// 2. Process the webhook data based on platform and event type
	// 3. Update relevant records (post status, analytics, etc.)
	// 4. Mark webhook as processed
	
	return nil
}

// processFacebookInstantArticle processes Facebook Instant Article creation
func (p *SocialMediaProcessor) processFacebookInstantArticle(job *Job) error {
	articleIDInterface, exists := job.Payload["article_id"]
	if !exists {
		return fmt.Errorf("article_id not found in job payload")
	}

	articleID, ok := articleIDInterface.(uint64)
	if !ok {
		return fmt.Errorf("invalid article_id type: %T", articleIDInterface)
	}

	// Create Facebook Instant Article
	if err := p.socialService.CreateFacebookInstantArticle(articleID); err != nil {
		log.Printf("Failed to create Facebook Instant Article for article %d: %v", articleID, err)
		return err
	}

	log.Printf("Successfully created Facebook Instant Article for article %d", articleID)
	return nil
}

// RegisterSocialMediaProcessor registers the social media processor with the job queue
func RegisterSocialMediaProcessor(jobQueue JobQueue, processor *SocialMediaProcessor) {
	// This would be called during application startup to register the processor
	// The actual implementation depends on how your job queue system works
	
	// Example pseudo-code:
	// jobQueue.RegisterProcessor("social_media_post", processor.ProcessJob)
	// jobQueue.RegisterProcessor("webhook_processing", processor.ProcessJob)
	// jobQueue.RegisterProcessor("facebook_instant_article", processor.ProcessJob)
}

// SocialMediaJobScheduler provides convenience methods for scheduling social media jobs
type SocialMediaJobScheduler struct {
	jobQueue JobQueue
}

// NewSocialMediaJobScheduler creates a new social media job scheduler
func NewSocialMediaJobScheduler(jobQueue JobQueue) *SocialMediaJobScheduler {
	return &SocialMediaJobScheduler{
		jobQueue: jobQueue,
	}
}

// SchedulePostJob schedules a social media post job
func (s *SocialMediaJobScheduler) SchedulePostJob(postID uint64, platform string, priority Priority) error {
	job := &Job{
		ID:       fmt.Sprintf("social_post_%d_%s", postID, platform),
		Type:     "social_media_post",
		Priority: priority,
		Payload: map[string]interface{}{
			"post_id":  postID,
			"platform": platform,
		},
	}

	return s.jobQueue.Enqueue(job)
}

// ScheduleWebhookJob schedules a webhook processing job
func (s *SocialMediaJobScheduler) ScheduleWebhookJob(webhookID uint64) error {
	job := &Job{
		ID:       fmt.Sprintf("webhook_%d", webhookID),
		Type:     "webhook_processing",
		Priority: PriorityHigh, // Webhooks should be processed quickly
		Payload: map[string]interface{}{
			"webhook_id": webhookID,
		},
	}

	return s.jobQueue.Enqueue(job)
}

// ScheduleFacebookInstantArticleJob schedules a Facebook Instant Article creation job
func (s *SocialMediaJobScheduler) ScheduleFacebookInstantArticleJob(articleID uint64) error {
	job := &Job{
		ID:       fmt.Sprintf("fb_instant_%d", articleID),
		Type:     "facebook_instant_article",
		Priority: PriorityMedium,
		Payload: map[string]interface{}{
			"article_id": articleID,
		},
	}

	return s.jobQueue.Enqueue(job)
}

// BatchSchedulePostJobs schedules multiple social media post jobs
func (s *SocialMediaJobScheduler) BatchSchedulePostJobs(posts []struct {
	PostID   uint64
	Platform string
}) error {
	for _, post := range posts {
		if err := s.SchedulePostJob(post.PostID, post.Platform, PriorityMedium); err != nil {
			log.Printf("Failed to schedule job for post %d on %s: %v", post.PostID, post.Platform, err)
			// Continue with other posts even if one fails
		}
	}
	return nil
}

// GetSocialMediaJobStats returns statistics for social media jobs
func (s *SocialMediaJobScheduler) GetSocialMediaJobStats() (map[string]int, error) {
	// Get queue sizes for each priority level
	highSize := s.jobQueue.GetQueueSize(PriorityHigh)
	mediumSize := s.jobQueue.GetQueueSize(PriorityMedium)
	lowSize := s.jobQueue.GetQueueSize(PriorityLow)
	
	// Return basic queue statistics
	// Note: Detailed per-job-type stats would require additional tracking
	socialStats := map[string]int{
		"high_priority_queue":   highSize,
		"medium_priority_queue": mediumSize,
		"low_priority_queue":    lowSize,
		"total_queued":          highSize + mediumSize + lowSize,
	}

	return socialStats, nil
}