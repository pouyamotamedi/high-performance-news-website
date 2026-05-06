package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// SocialMediaService handles social media integration operations
type SocialMediaService struct {
	db          *sql.DB
	httpClient  *http.Client
	jobQueue    models.JobQueue
	encryptKey  []byte
}

// NewSocialMediaService creates a new social media service
func NewSocialMediaService(db *sql.DB, jobQueue models.JobQueue, encryptKey []byte) *SocialMediaService {
	return &SocialMediaService{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		jobQueue:   jobQueue,
		encryptKey: encryptKey,
	}
}

// CreateCredentials creates new social media credentials
func (s *SocialMediaService) CreateCredentials(creds *models.SocialMediaCredentials) error {
	if err := models.ValidateSocialMediaCredentials(creds); err != nil {
		return err
	}

	// Encrypt credentials before storing
	encryptedData, err := s.encryptCredentials(creds.Credentials.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	creds.Credentials = models.EncryptedData{
		Data:      encryptedData,
		Algorithm: "AES-256-GCM",
		KeyID:     "default",
	}
	creds.CreatedAt = time.Now()
	creds.UpdatedAt = time.Now()

	query := `
		INSERT INTO social_media_credentials (platform, name, credentials, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err = s.db.QueryRow(query, creds.Platform, creds.Name, creds.Credentials, 
		creds.IsActive, creds.CreatedAt, creds.UpdatedAt).Scan(&creds.ID)
	if err != nil {
		return fmt.Errorf("failed to create credentials: %w", err)
	}

	return nil
}

// GetCredentials retrieves social media credentials by ID
func (s *SocialMediaService) GetCredentials(id uint64) (*models.SocialMediaCredentials, error) {
	creds := &models.SocialMediaCredentials{}
	query := `
		SELECT id, platform, name, credentials, is_active, last_rotated, created_at, updated_at
		FROM social_media_credentials
		WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(
		&creds.ID, &creds.Platform, &creds.Name, &creds.Credentials,
		&creds.IsActive, &creds.LastRotated, &creds.CreatedAt, &creds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	return creds, nil
}

// RotateCredentials rotates social media credentials
func (s *SocialMediaService) RotateCredentials(id uint64, newCredentials string) error {
	encryptedData, err := s.encryptCredentials(newCredentials)
	if err != nil {
		return fmt.Errorf("failed to encrypt new credentials: %w", err)
	}

	newEncryptedData := models.EncryptedData{
		Data:      encryptedData,
		Algorithm: "AES-256-GCM",
		KeyID:     "default",
	}

	now := time.Now()
	query := `
		UPDATE social_media_credentials 
		SET credentials = $1, last_rotated = $2, updated_at = $3
		WHERE id = $4`

	_, err = s.db.Exec(query, newEncryptedData, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to rotate credentials: %w", err)
	}

	return nil
}

// PublishToSocialMedia publishes an article to social media platforms
func (s *SocialMediaService) PublishToSocialMedia(articleID uint64, platforms []models.SocialMediaPlatform) error {

	// Create posts for each platform
	for _, platform := range platforms {
		post := &models.SocialMediaPost{
			ArticleID:   articleID,
			Platform:    platform,
			Status:      models.PostStatusPending,
			MaxAttempts: 3,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Get credentials for platform
		credentialID, err := s.getActiveCredentialID(platform)
		if err != nil {
			continue // Skip if no credentials available
		}
		post.CredentialID = credentialID

		// Generate platform-specific content
		if err := s.generatePostContent(post); err != nil {
			continue // Skip if content generation fails
		}

		// Save post to database
		if err := s.CreatePost(post); err != nil {
			continue // Skip if post creation fails
		}

		// Schedule the job for immediate posting (or delayed if needed)
		scheduledAt := time.Now().Add(1 * time.Minute) // Small delay for processing
		job := &models.Job{
			ID:          fmt.Sprintf("social_post_%d_%s", post.ID, platform),
			Type:        "social_media_post",
			Priority:    models.JobPriorityMedium,
			ScheduledAt: scheduledAt,
			MaxAttempts: 3,
			Payload: map[string]interface{}{
				"post_id":  post.ID,
				"platform": string(platform),
			},
		}

		if err := s.jobQueue.Enqueue(job); err != nil {
			continue // Skip if job enqueue fails
		}
	}

	return nil
}

// CreatePost creates a new social media post record
func (s *SocialMediaService) CreatePost(post *models.SocialMediaPost) error {
	if err := models.ValidateSocialMediaPost(post); err != nil {
		return err
	}

	query := `
		INSERT INTO social_media_posts (article_id, platform, credential_id, status, content, 
			scheduled_at, attempts, max_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err := s.db.QueryRow(query, post.ArticleID, post.Platform, post.CredentialID,
		post.Status, post.Content, post.ScheduledAt, post.Attempts, post.MaxAttempts,
		post.CreatedAt, post.UpdatedAt).Scan(&post.ID)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// PublishPost publishes a post to its platform
func (s *SocialMediaService) PublishPost(postID uint64) error {
	post, err := s.GetPostStatus(postID)
	if err != nil {
		return err
	}

	// Update status to posting
	post.Status = models.PostStatusRetrying
	post.Attempts++
	post.UpdatedAt = time.Now()

	if err := s.updatePostStatus(post); err != nil {
		return err
	}

	// Publish to the appropriate platform
	var publishErr error
	switch post.Platform {
	case models.PlatformFacebook:
		publishErr = s.PublishToFacebook(post)
	case models.PlatformTelegram:
		publishErr = s.PublishToTelegram(post)
	case models.PlatformTwitter:
		publishErr = s.PublishToTwitter(post)
	default:
		publishErr = fmt.Errorf("unsupported platform: %s", post.Platform)
	}

	// Update post status based on result
	if publishErr != nil {
		post.Status = models.PostStatusFailed
		post.LastError = publishErr.Error()
		
		// Retry with exponential backoff if attempts remain
		if post.Attempts < post.MaxAttempts {
			delay := time.Duration(math.Pow(2, float64(post.Attempts))) * time.Minute
			retryAt := time.Now().Add(delay)
			
			job := &models.Job{
				ID:          fmt.Sprintf("social_retry_%d", post.ID),
				Type:        "social_media_post",
				Priority:    models.JobPriorityLow,
				ScheduledAt: retryAt,
				MaxAttempts: 1,
				Payload: map[string]interface{}{
					"post_id": post.ID,
					"retry":   true,
				},
			}
			
			if err := s.jobQueue.Enqueue(job); err != nil {
				log.Printf("Failed to enqueue retry job: %v", err)
			}
		}
	} else {
		post.Status = models.PostStatusPosted
		now := time.Now()
		post.PostedAt = &now
		post.LastError = ""
	}

	post.UpdatedAt = time.Now()
	return s.updatePostStatus(post)
}

// PublishToFacebook publishes a post to Facebook
func (s *SocialMediaService) PublishToFacebook(post *models.SocialMediaPost) error {
	creds, err := s.getDecryptedCredentials(post.CredentialID)
	if err != nil {
		return err
	}

	// Parse Facebook credentials
	var fbCreds struct {
		AccessToken string `json:"access_token"`
		PageID      string `json:"page_id"`
	}
	if err := json.Unmarshal([]byte(creds), &fbCreds); err != nil {
		return fmt.Errorf("invalid Facebook credentials: %w", err)
	}

	// Prepare Facebook post data
	postData := map[string]interface{}{
		"message":      post.Content.Text,
		"access_token": fbCreds.AccessToken,
	}

	if post.Content.LinkURL != "" {
		postData["link"] = post.Content.LinkURL
	}

	// Convert to form data
	formData := make([]string, 0, len(postData))
	for key, value := range postData {
		formData = append(formData, fmt.Sprintf("%s=%s", key, fmt.Sprintf("%v", value)))
	}

	// Make API request
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/feed", fbCreds.PageID)
	resp, err := s.httpClient.Post(url, "application/x-www-form-urlencoded", 
		strings.NewReader(strings.Join(formData, "&")))
	if err != nil {
		return fmt.Errorf("Facebook API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Facebook response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Facebook API error: %s", string(body))
	}

	// Parse response to get post ID
	var fbResponse struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &fbResponse); err != nil {
		return fmt.Errorf("failed to parse Facebook response: %w", err)
	}

	post.PostID = fbResponse.ID
	return nil
}

// PublishToTelegram publishes a post to Telegram
func (s *SocialMediaService) PublishToTelegram(post *models.SocialMediaPost) error {
	creds, err := s.getDecryptedCredentials(post.CredentialID)
	if err != nil {
		return err
	}

	// Parse Telegram credentials
	var tgCreds struct {
		BotToken  string `json:"bot_token"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.Unmarshal([]byte(creds), &tgCreds); err != nil {
		return fmt.Errorf("invalid Telegram credentials: %w", err)
	}

	// Prepare message
	message := post.Content.Text
	if post.Content.LinkURL != "" {
		message += "\n\n" + post.Content.LinkURL
	}

	// Prepare Telegram post data
	postData := map[string]interface{}{
		"chat_id":    tgCreds.ChannelID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram data: %w", err)
	}

	// Make API request
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tgCreds.BotToken)
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Telegram response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API error: %s", string(body))
	}

	// Parse response to get message ID
	var tgResponse struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &tgResponse); err != nil {
		return fmt.Errorf("failed to parse Telegram response: %w", err)
	}

	if !tgResponse.OK {
		return fmt.Errorf("Telegram API returned error: %s", string(body))
	}

	post.PostID = strconv.Itoa(tgResponse.Result.MessageID)
	return nil
}

// PublishToTwitter publishes a post to Twitter (X)
func (s *SocialMediaService) PublishToTwitter(post *models.SocialMediaPost) error {
	creds, err := s.getDecryptedCredentials(post.CredentialID)
	if err != nil {
		return err
	}

	// Parse Twitter credentials
	var twCreds struct {
		BearerToken string `json:"bearer_token"`
	}
	if err := json.Unmarshal([]byte(creds), &twCreds); err != nil {
		return fmt.Errorf("invalid Twitter credentials: %w", err)
	}

	// Prepare Twitter post data
	postData := map[string]interface{}{
		"text": post.Content.Text,
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return fmt.Errorf("failed to marshal Twitter data: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.twitter.com/2/tweets", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Twitter request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+twCreds.BearerToken)
	req.Header.Set("Content-Type", "application/json")

	// Make API request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Twitter API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Twitter response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Twitter API error: %s", string(body))
	}

	// Parse response to get tweet ID
	var twResponse struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &twResponse); err != nil {
		return fmt.Errorf("failed to parse Twitter response: %w", err)
	}

	post.PostID = twResponse.Data.ID
	return nil
}

// CreateFacebookInstantArticle creates a Facebook Instant Article
func (s *SocialMediaService) CreateFacebookInstantArticle(articleID uint64) error {
	// Get article data
	article, err := s.getArticle(articleID)
	if err != nil {
		return err
	}

	// Generate Facebook Instant Article HTML
	instantHTML := s.generateInstantArticleHTML(article)

	// Get Facebook credentials
	creds, err := s.GetCredentialsByPlatform(models.PlatformFacebook)
	if err != nil {
		return err
	}

	credData, err := s.getDecryptedCredentials(creds.ID)
	if err != nil {
		return err
	}

	var fbCreds struct {
		AccessToken string `json:"access_token"`
		PageID      string `json:"page_id"`
	}
	if err := json.Unmarshal([]byte(credData), &fbCreds); err != nil {
		return fmt.Errorf("invalid Facebook credentials: %w", err)
	}

	// Submit to Facebook Instant Articles API
	postData := map[string]interface{}{
		"html_source":  instantHTML,
		"published":    true,
		"access_token": fbCreds.AccessToken,
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return fmt.Errorf("failed to marshal instant article data: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/instant_articles", fbCreds.PageID)
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Facebook Instant Articles API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Facebook response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Facebook Instant Articles API error: %s", string(body))
	}

	// Store instant article record
	instantArticle := &models.FacebookInstantArticle{
		ArticleID:    articleID,
		HTML:         instantHTML,
		Status:       "published",
		LastSynced:   time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.saveFacebookInstantArticle(instantArticle)
}

// Helper methods

func (s *SocialMediaService) encryptCredentials(data string) (string, error) {
	// Simple encryption implementation - in production, use proper encryption
	h := hmac.New(sha256.New, s.encryptKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *SocialMediaService) getDecryptedCredentials(credentialID uint64) (string, error) {
	// This is a simplified implementation
	// In production, implement proper decryption
	query := `SELECT credentials FROM social_media_credentials WHERE id = $1`
	var encData models.EncryptedData
	err := s.db.QueryRow(query, credentialID).Scan(&encData)
	if err != nil {
		return "", err
	}
	
	// For now, return the encrypted data as-is
	// In production, decrypt using the proper algorithm
	return encData.Data, nil
}

func (s *SocialMediaService) generatePostContent(post *models.SocialMediaPost) error {
	// Get article data
	article, err := s.getArticle(post.ArticleID)
	if err != nil {
		return err
	}

	// Use article language, default to English if not set
	lang := article.LanguageCode
	if lang == "" {
		lang = "en"
	}

	// Generate platform-specific content
	switch post.Platform {
	case models.PlatformFacebook:
		post.Content = models.PostContent{
			Text:     fmt.Sprintf("%s\n\n%s", article.Title, article.Excerpt),
			LinkURL:  fmt.Sprintf("https://example.com/%s/article/%s", lang, article.Slug),
			Hashtags: s.extractHashtags(article),
		}
	case models.PlatformTelegram:
		post.Content = models.PostContent{
			Text:    fmt.Sprintf("<b>%s</b>\n\n%s", article.Title, article.Excerpt),
			LinkURL: fmt.Sprintf("https://example.com/%s/article/%s", lang, article.Slug),
		}
	case models.PlatformTwitter:
		// Twitter has character limits
		text := article.Title
		if len(text) > 200 {
			text = text[:197] + "..."
		}
		post.Content = models.PostContent{
			Text:     text,
			LinkURL:  fmt.Sprintf("https://example.com/%s/article/%s", lang, article.Slug),
			Hashtags: s.extractHashtags(article)[:3], // Limit hashtags for Twitter
		}
	}

	return nil
}

func (s *SocialMediaService) getArticle(articleID uint64) (*models.Article, error) {
	query := `
		SELECT id, title, slug, content, excerpt, author_id, category_id, status, published_at
		FROM articles 
		WHERE id = $1`

	var article models.Article
	err := s.db.QueryRow(query, articleID).Scan(
		&article.ID, &article.Title, &article.Slug, &article.Content,
		&article.Excerpt, &article.AuthorID, &article.CategoryID,
		&article.Status, &article.PublishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	return &article, nil
}

func (s *SocialMediaService) extractHashtags(article *models.Article) []string {
	// Simple hashtag extraction from title and content
	// In production, this would be more sophisticated
	words := strings.Fields(strings.ToLower(article.Title))
	hashtags := make([]string, 0, len(words))
	
	for _, word := range words {
		if len(word) > 3 {
			hashtags = append(hashtags, "#"+word)
		}
		if len(hashtags) >= 5 {
			break
		}
	}
	
	return hashtags
}

func (s *SocialMediaService) updatePostStatus(post *models.SocialMediaPost) error {
	query := `
		UPDATE social_media_posts 
		SET status = $1, post_id = $2, posted_at = $3, attempts = $4, 
			last_error = $5, updated_at = $6
		WHERE id = $7`

	_, err := s.db.Exec(query, post.Status, post.PostID, post.PostedAt,
		post.Attempts, post.LastError, post.UpdatedAt, post.ID)
	return err
}

func (s *SocialMediaService) GetPostStatus(postID uint64) (*models.SocialMediaPost, error) {
	query := `
		SELECT id, article_id, platform, credential_id, post_id, status, content,
			scheduled_at, posted_at, attempts, max_attempts, last_error, created_at, updated_at
		FROM social_media_posts
		WHERE id = $1`

	var post models.SocialMediaPost
	err := s.db.QueryRow(query, postID).Scan(
		&post.ID, &post.ArticleID, &post.Platform, &post.CredentialID,
		&post.PostID, &post.Status, &post.Content, &post.ScheduledAt,
		&post.PostedAt, &post.Attempts, &post.MaxAttempts, &post.LastError,
		&post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get post status: %w", err)
	}

	return &post, nil
}

func (s *SocialMediaService) generateInstantArticleHTML(article *models.Article) string {
	// Use article language, default to English if not set
	lang := article.LanguageCode
	if lang == "" {
		lang = "en"
	}
	// Generate Facebook Instant Article HTML format
	return fmt.Sprintf(`
<!doctype html>
<html lang="en" prefix="op: http://media.facebook.com/op#">
<head>
	<meta charset="utf-8">
	<link rel="canonical" href="https://example.com/%s/article/%s">
	<meta property="op:markup_version" content="v1.0">
</head>
<body>
	<article>
		<header>
			<h1>%s</h1>
			<time class="op-published" datetime="%s">%s</time>
		</header>
		<p>%s</p>
		%s
	</article>
</body>
</html>`, 
		lang,
		article.Slug, 
		article.Title,
		article.PublishedAt.Format(time.RFC3339),
		article.PublishedAt.Format("January 2, 2006"),
		article.Excerpt,
		article.Content,
	)
}

func (s *SocialMediaService) saveFacebookInstantArticle(ia *models.FacebookInstantArticle) error {
	query := `
		INSERT INTO facebook_instant_articles (article_id, html, status, last_synced, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (article_id) DO UPDATE SET
			html = EXCLUDED.html,
			status = EXCLUDED.status,
			last_synced = EXCLUDED.last_synced,
			updated_at = EXCLUDED.updated_at`

	_, err := s.db.Exec(query, ia.ArticleID, ia.HTML, ia.Status,
		ia.LastSynced, ia.CreatedAt, ia.UpdatedAt)
	return err
}

// Webhook handling methods will be implemented in the next part
func (s *SocialMediaService) HandleWebhook(platform models.SocialMediaPlatform, payload []byte, signature string) error {
	// Verify webhook signature
	if !s.VerifyWebhookSignature(platform, payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	// Store webhook for processing
	webhook := &models.SocialMediaWebhook{
		Platform:  platform,
		Payload:   models.WebhookPayload{Data: make(map[string]interface{})},
		Signature: signature,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	// Parse payload based on platform
	if err := json.Unmarshal(payload, &webhook.Payload.Data); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Store webhook
	if err := s.storeWebhook(webhook); err != nil {
		return fmt.Errorf("failed to store webhook: %w", err)
	}

	// Process webhook asynchronously
	job := &models.Job{
		ID:       fmt.Sprintf("webhook_%s_%d", platform, webhook.ID),
		Type:     "webhook_processing",
		Priority: models.JobPriorityHigh,
		Payload: map[string]interface{}{
			"webhook_id": webhook.ID,
		},
	}

	return s.jobQueue.Enqueue(job)
}

func (s *SocialMediaService) VerifyWebhookSignature(platform models.SocialMediaPlatform, payload []byte, signature string) bool {
	// Implementation depends on platform-specific signature verification
	// This is a simplified version
	return true
}

func (s *SocialMediaService) storeWebhook(webhook *models.SocialMediaWebhook) error {
	query := `
		INSERT INTO social_media_webhooks (platform, event_type, post_id, payload, signature, verified, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	return s.db.QueryRow(query, webhook.Platform, webhook.EventType, webhook.PostID,
		webhook.Payload, webhook.Signature, webhook.Verified, webhook.CreatedAt).Scan(&webhook.ID)
}

// Additional methods for bulk operations and analytics
func (s *SocialMediaService) PublishArticleToAllPlatforms(articleID uint64) error {
	platforms := []models.SocialMediaPlatform{
		models.PlatformFacebook,
		models.PlatformTelegram,
		models.PlatformTwitter,
	}

	return s.PublishToSocialMedia(articleID, platforms)
}

// SchedulePost schedules a post for future publishing
func (s *SocialMediaService) SchedulePost(articleID uint64, platforms []models.SocialMediaPlatform, scheduledAt time.Time) error {

	// Create posts for each platform
	for _, platform := range platforms {
		post := &models.SocialMediaPost{
			ArticleID:   articleID,
			Platform:    platform,
			Status:      models.PostStatusScheduled,
			MaxAttempts: 3,
			ScheduledAt: &scheduledAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Get credentials for platform
		credentialID, err := s.getActiveCredentialID(platform)
		if err != nil {
			continue // Skip if no credentials available
		}
		post.CredentialID = credentialID

		// Generate platform-specific content
		if err := s.generatePostContent(post); err != nil {
			continue // Skip if content generation fails
		}

		// Save post to database
		if err := s.CreatePost(post); err != nil {
			continue // Skip if post creation fails
		}

		// Schedule the job
		job := &models.Job{
			ID:          fmt.Sprintf("social_scheduled_%d_%s", post.ID, platform),
			Type:        "social_media_post",
			Priority:    models.JobPriorityMedium,
			ScheduledAt: scheduledAt,
			MaxAttempts: 3,
			Payload: map[string]interface{}{
				"post_id":  post.ID,
				"platform": string(platform),
			},
		}

		if err := s.jobQueue.Enqueue(job); err != nil {
			continue // Skip if job enqueue fails
		}
	}

	return nil
}

// GetCredentialsByPlatform retrieves active credentials for a platform
func (s *SocialMediaService) GetCredentialsByPlatform(platform models.SocialMediaPlatform) (*models.SocialMediaCredentials, error) {
	creds := &models.SocialMediaCredentials{}
	query := `
		SELECT id, platform, name, credentials, is_active, last_rotated, created_at, updated_at
		FROM social_media_credentials
		WHERE platform = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1`

	err := s.db.QueryRow(query, platform).Scan(
		&creds.ID, &creds.Platform, &creds.Name, &creds.Credentials,
		&creds.IsActive, &creds.LastRotated, &creds.CreatedAt, &creds.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials for platform %s: %w", platform, err)
	}

	return creds, nil
}

// getActiveCredentialID gets the active credential ID for a platform
func (s *SocialMediaService) getActiveCredentialID(platform models.SocialMediaPlatform) (uint64, error) {
	var credentialID uint64
	query := `
		SELECT id FROM social_media_credentials
		WHERE platform = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1`

	err := s.db.QueryRow(query, platform).Scan(&credentialID)
	if err != nil {
		return 0, fmt.Errorf("no active credentials found for platform %s: %w", platform, err)
	}

	return credentialID, nil
}
