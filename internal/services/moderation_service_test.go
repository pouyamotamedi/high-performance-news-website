package services

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	_ "github.com/lib/pq"
)

// MockAIService for testing
type MockAIService struct {
	shouldFail bool
	feedback   *models.AIFeedback
}

func (m *MockAIService) AnalyzeContent(title, content string) (*models.AIFeedback, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock AI service error")
	}
	
	if m.feedback != nil {
		return m.feedback, nil
	}
	
	// Default mock response
	return &models.AIFeedback{
		Provider:             "mock",
		QualityScore:         0.85,
		GrammarScore:         &[]float64{0.90}[0],
		ReadabilityScore:     &[]float64{0.80}[0],
		AppropriatenessScore: &[]float64{0.95}[0],
		Issues: []models.AIIssue{
			{
				Type:        "grammar",
				Severity:    "low",
				Description: "Minor grammar issue",
				Suggestion:  "Fix grammar",
			},
		},
		Suggestions: []models.AISuggestion{
			{
				Type:        "title",
				Priority:    "medium",
				Description: "Title could be improved",
				Suggested:   "Better title",
			},
		},
		FlaggedContent:   []models.AIFlaggedContent{},
		ProcessingTimeMs: 100,
		Confidence:       0.88,
	}, nil
}

func (m *MockAIService) GenerateMetaDescription(title, content string) (string, error) {
	return "Generated meta description", nil
}

func (m *MockAIService) GenerateTitle(content string) (string, error) {
	return "Generated title", nil
}

func (m *MockAIService) CheckGrammar(text string) ([]models.AIIssue, error) {
	return []models.AIIssue{}, nil
}

func (m *MockAIService) CheckReadability(text string) (float64, error) {
	return 0.8, nil
}

func (m *MockAIService) CheckAppropriateness(text string) (float64, []models.AIFlaggedContent, error) {
	return 0.9, []models.AIFlaggedContent{}, nil
}

func setupModerationTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "postgres://test:test@localhost/test_db?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	
	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS moderation_queue (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT NOT NULL,
			article_version_id BIGINT,
			content_type VARCHAR(20) NOT NULL DEFAULT 'article',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 1,
			submitted_by BIGINT NOT NULL,
			assigned_to BIGINT,
			ai_quality_score DECIMAL(3,2),
			ai_feedback JSONB,
			moderator_notes TEXT,
			rejection_reason TEXT,
			auto_approved BOOLEAN DEFAULT false,
			submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			reviewed_at TIMESTAMP WITH TIME ZONE,
			reviewed_by BIGINT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create moderation_queue table: %v", err)
	}
	
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS content_quality_checks (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT NOT NULL,
			article_version_id BIGINT,
			ai_provider VARCHAR(20) NOT NULL,
			quality_score DECIMAL(3,2) NOT NULL,
			grammar_score DECIMAL(3,2),
			readability_score DECIMAL(3,2),
			appropriateness_score DECIMAL(3,2),
			issues_found JSONB,
			suggestions JSONB,
			flagged_content JSONB,
			processing_time_ms INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create content_quality_checks table: %v", err)
	}
	
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS moderation_actions (
			id BIGSERIAL PRIMARY KEY,
			moderation_queue_id BIGINT NOT NULL,
			action VARCHAR(20) NOT NULL,
			performed_by BIGINT NOT NULL,
			notes TEXT,
			previous_status VARCHAR(20),
			new_status VARCHAR(20),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create moderation_actions table: %v", err)
	}
	
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			moderation_status VARCHAR(20) DEFAULT 'approved',
			moderation_notes TEXT,
			last_moderated_at TIMESTAMP WITH TIME ZONE,
			last_moderated_by BIGINT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create articles table: %v", err)
	}
	
	return db
}

func cleanupModerationTestDB(t *testing.T, db *sql.DB) {
	tables := []string{"moderation_actions", "content_quality_checks", "moderation_queue", "articles"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			t.Errorf("Failed to cleanup table %s: %v", table, err)
		}
	}
	db.Close()
}

func TestModerationService_SubmitForModeration(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test article
	_, err := db.Exec(`
		INSERT INTO articles (id, title, content) 
		VALUES (1, 'Test Article', 'Test content for moderation')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Test submitting for moderation
	moderationItem, err := ms.SubmitForModeration(1, "article", 2, 1)
	if err != nil {
		t.Fatalf("Failed to submit for moderation: %v", err)
	}
	
	if moderationItem.ID == 0 {
		t.Error("Moderation item ID should not be 0")
	}
	
	if moderationItem.ArticleID != 1 {
		t.Errorf("Expected article ID 1, got %d", moderationItem.ArticleID)
	}
	
	if moderationItem.Status != models.ModerationStatusPending {
		t.Errorf("Expected status pending, got %s", moderationItem.Status)
	}
	
	if moderationItem.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", moderationItem.Priority)
	}
	
	// Test submitting same article again (should fail)
	_, err = ms.SubmitForModeration(1, "article", 1, 1)
	if err == nil {
		t.Error("Expected error when submitting same article twice")
	}
}

func TestModerationService_RunAIQualityCheck(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test article
	_, err := db.Exec(`
		INSERT INTO articles (id, title, content) 
		VALUES (1, 'Test Article', 'Test content for AI analysis')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	// Insert moderation item
	var moderationID uint64
	err = db.QueryRow(`
		INSERT INTO moderation_queue (article_id, content_type, status, priority, submitted_by)
		VALUES (1, 'article', 'pending', 1, 1)
		RETURNING id
	`).Scan(&moderationID)
	if err != nil {
		t.Fatalf("Failed to insert moderation item: %v", err)
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Test AI quality check
	err = ms.RunAIQualityCheck(moderationID)
	if err != nil {
		t.Fatalf("Failed to run AI quality check: %v", err)
	}
	
	// Verify quality check was stored
	var qualityScore float64
	var provider string
	err = db.QueryRow(`
		SELECT quality_score, ai_provider 
		FROM content_quality_checks 
		WHERE article_id = 1
	`).Scan(&qualityScore, &provider)
	if err != nil {
		t.Fatalf("Failed to get quality check: %v", err)
	}
	
	if qualityScore != 0.85 {
		t.Errorf("Expected quality score 0.85, got %f", qualityScore)
	}
	
	if provider != "mock" {
		t.Errorf("Expected provider 'mock', got %s", provider)
	}
	
	// Verify moderation queue was updated
	var aiScore sql.NullFloat64
	err = db.QueryRow(`
		SELECT ai_quality_score FROM moderation_queue WHERE id = $1
	`, moderationID).Scan(&aiScore)
	if err != nil {
		t.Fatalf("Failed to get updated moderation item: %v", err)
	}
	
	if !aiScore.Valid || aiScore.Float64 != 0.85 {
		t.Errorf("Expected AI score 0.85, got %v", aiScore)
	}
}

func TestModerationService_AutoApproval(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test article
	_, err := db.Exec(`
		INSERT INTO articles (id, title, content) 
		VALUES (1, 'High Quality Article', 'This is high quality content')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	// Create mock AI service with high quality score
	mockAI := &MockAIService{
		feedback: &models.AIFeedback{
			Provider:             "mock",
			QualityScore:         0.95, // High score for auto-approval
			GrammarScore:         &[]float64{0.98}[0],
			ReadabilityScore:     &[]float64{0.92}[0],
			AppropriatenessScore: &[]float64{0.99}[0],
			Issues:               []models.AIIssue{}, // No issues
			Suggestions:          []models.AISuggestion{},
			FlaggedContent:       []models.AIFlaggedContent{}, // No flagged content
			ProcessingTimeMs:     100,
			Confidence:           0.95,
		},
	}
	
	ms := NewModerationService(db, mockAI)
	
	// Submit for moderation
	moderationItem, err := ms.SubmitForModeration(1, "article", 1, 1)
	if err != nil {
		t.Fatalf("Failed to submit for moderation: %v", err)
	}
	
	// Wait a moment for async AI processing
	time.Sleep(100 * time.Millisecond)
	
	// Check if item was auto-approved
	var status string
	var autoApproved bool
	err = db.QueryRow(`
		SELECT status, auto_approved FROM moderation_queue WHERE id = $1
	`, moderationItem.ID).Scan(&status, &autoApproved)
	if err != nil {
		t.Fatalf("Failed to get moderation status: %v", err)
	}
	
	if status != "approved" {
		t.Errorf("Expected status 'approved', got %s", status)
	}
	
	if !autoApproved {
		t.Error("Expected auto_approved to be true")
	}
}

func TestModerationService_ApproveContent(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test data
	_, err := db.Exec(`
		INSERT INTO articles (id, title, content) 
		VALUES (1, 'Test Article', 'Test content')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	var moderationID uint64
	err = db.QueryRow(`
		INSERT INTO moderation_queue (article_id, content_type, status, priority, submitted_by)
		VALUES (1, 'article', 'pending', 1, 1)
		RETURNING id
	`).Scan(&moderationID)
	if err != nil {
		t.Fatalf("Failed to insert moderation item: %v", err)
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Test approving content
	err = ms.ApproveContent(moderationID, 2, "Content looks good", false)
	if err != nil {
		t.Fatalf("Failed to approve content: %v", err)
	}
	
	// Verify moderation queue was updated
	var status string
	var notes string
	var reviewedBy sql.NullInt64
	err = db.QueryRow(`
		SELECT status, moderator_notes, reviewed_by 
		FROM moderation_queue WHERE id = $1
	`, moderationID).Scan(&status, &notes, &reviewedBy)
	if err != nil {
		t.Fatalf("Failed to get updated moderation item: %v", err)
	}
	
	if status != "approved" {
		t.Errorf("Expected status 'approved', got %s", status)
	}
	
	if notes != "Content looks good" {
		t.Errorf("Expected notes 'Content looks good', got %s", notes)
	}
	
	if !reviewedBy.Valid || reviewedBy.Int64 != 2 {
		t.Errorf("Expected reviewed_by 2, got %v", reviewedBy)
	}
	
	// Verify article moderation status was updated
	var articleStatus string
	err = db.QueryRow(`
		SELECT moderation_status FROM articles WHERE id = 1
	`).Scan(&articleStatus)
	if err != nil {
		t.Fatalf("Failed to get article moderation status: %v", err)
	}
	
	if articleStatus != "approved" {
		t.Errorf("Expected article status 'approved', got %s", articleStatus)
	}
	
	// Verify action was logged
	var actionCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM moderation_actions 
		WHERE moderation_queue_id = $1 AND action = 'approve'
	`, moderationID).Scan(&actionCount)
	if err != nil {
		t.Fatalf("Failed to get action count: %v", err)
	}
	
	if actionCount != 1 {
		t.Errorf("Expected 1 approve action, got %d", actionCount)
	}
}

func TestModerationService_RejectContent(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test data
	_, err := db.Exec(`
		INSERT INTO articles (id, title, content) 
		VALUES (1, 'Test Article', 'Test content')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test article: %v", err)
	}
	
	var moderationID uint64
	err = db.QueryRow(`
		INSERT INTO moderation_queue (article_id, content_type, status, priority, submitted_by)
		VALUES (1, 'article', 'pending', 1, 1)
		RETURNING id
	`).Scan(&moderationID)
	if err != nil {
		t.Fatalf("Failed to insert moderation item: %v", err)
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Test rejecting content
	err = ms.RejectContent(moderationID, 2, "Inappropriate content", "Content violates guidelines")
	if err != nil {
		t.Fatalf("Failed to reject content: %v", err)
	}
	
	// Verify moderation queue was updated
	var status string
	var reason string
	var notes string
	err = db.QueryRow(`
		SELECT status, rejection_reason, moderator_notes 
		FROM moderation_queue WHERE id = $1
	`, moderationID).Scan(&status, &reason, &notes)
	if err != nil {
		t.Fatalf("Failed to get updated moderation item: %v", err)
	}
	
	if status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", status)
	}
	
	if reason != "Inappropriate content" {
		t.Errorf("Expected reason 'Inappropriate content', got %s", reason)
	}
	
	if notes != "Content violates guidelines" {
		t.Errorf("Expected notes 'Content violates guidelines', got %s", notes)
	}
	
	// Verify article moderation status was updated
	var articleStatus string
	err = db.QueryRow(`
		SELECT moderation_status FROM articles WHERE id = 1
	`).Scan(&articleStatus)
	if err != nil {
		t.Fatalf("Failed to get article moderation status: %v", err)
	}
	
	if articleStatus != "rejected" {
		t.Errorf("Expected article status 'rejected', got %s", articleStatus)
	}
}

func TestModerationService_GetModerationQueue(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test articles
	for i := 1; i <= 5; i++ {
		_, err := db.Exec(`
			INSERT INTO articles (id, title, content) 
			VALUES ($1, $2, $3)
		`, i, fmt.Sprintf("Article %d", i), fmt.Sprintf("Content %d", i))
		if err != nil {
			t.Fatalf("Failed to insert test article %d: %v", i, err)
		}
	}
	
	// Insert moderation items with different statuses and priorities
	moderationData := []struct {
		articleID int
		status    string
		priority  int
	}{
		{1, "pending", 3},
		{2, "pending", 1},
		{3, "approved", 2},
		{4, "rejected", 1},
		{5, "flagged", 4},
	}
	
	for _, data := range moderationData {
		_, err := db.Exec(`
			INSERT INTO moderation_queue (article_id, content_type, status, priority, submitted_by)
			VALUES ($1, 'article', $2, $3, 1)
		`, data.articleID, data.status, data.priority)
		if err != nil {
			t.Fatalf("Failed to insert moderation item: %v", err)
		}
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Test getting all items
	items, total, err := ms.GetModerationQueue(ModerationFilters{}, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get moderation queue: %v", err)
	}
	
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	
	if len(items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(items))
	}
	
	// Items should be ordered by priority DESC, submitted_at ASC
	if items[0].Priority != 4 {
		t.Errorf("Expected first item priority 4, got %d", items[0].Priority)
	}
	
	// Test filtering by status
	pendingItems, pendingTotal, err := ms.GetModerationQueue(
		ModerationFilters{Status: []string{"pending"}}, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get pending items: %v", err)
	}
	
	if pendingTotal != 2 {
		t.Errorf("Expected 2 pending items, got %d", pendingTotal)
	}
	
	if len(pendingItems) != 2 {
		t.Errorf("Expected 2 pending items in result, got %d", len(pendingItems))
	}
	
	// Test filtering by priority
	highPriorityItems, highPriorityTotal, err := ms.GetModerationQueue(
		ModerationFilters{Priority: []int{3, 4}}, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get high priority items: %v", err)
	}
	
	if highPriorityTotal != 2 {
		t.Errorf("Expected 2 high priority items, got %d", highPriorityTotal)
	}
	
	// Test pagination
	page1, _, err := ms.GetModerationQueue(ModerationFilters{}, 2, 0)
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}
	
	page2, _, err := ms.GetModerationQueue(ModerationFilters{}, 2, 2)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}
	
	if len(page1) != 2 {
		t.Errorf("Expected 2 items in page 1, got %d", len(page1))
	}
	
	if len(page2) != 2 {
		t.Errorf("Expected 2 items in page 2, got %d", len(page2))
	}
	
	// Pages should not overlap
	if page1[0].ID == page2[0].ID {
		t.Error("Pages should not overlap")
	}
}

func TestModerationService_GetModerationStats(t *testing.T) {
	db := setupModerationTestDB(t)
	defer cleanupModerationTestDB(t, db)
	
	// Insert test data with specific timestamps
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	
	// Insert moderation items
	testData := []struct {
		status       string
		autoApproved bool
		reviewedAt   *time.Time
	}{
		{"pending", false, nil},
		{"pending", false, nil},
		{"approved", false, &now},
		{"approved", true, &now},
		{"rejected", false, &now},
		{"approved", false, &yesterday}, // Should not count in last 24h
	}
	
	for i, data := range testData {
		_, err := db.Exec(`
			INSERT INTO moderation_queue (
				article_id, content_type, status, priority, submitted_by,
				auto_approved, reviewed_at, submitted_at
			) VALUES ($1, 'article', $2, 1, 1, $3, $4, $5)
		`, i+1, data.status, data.autoApproved, data.reviewedAt, now)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
	
	mockAI := &MockAIService{}
	ms := NewModerationService(db, mockAI)
	
	// Get stats
	stats, err := ms.GetModerationStats()
	if err != nil {
		t.Fatalf("Failed to get moderation stats: %v", err)
	}
	
	if stats.PendingCount != 2 {
		t.Errorf("Expected 2 pending items, got %d", stats.PendingCount)
	}
	
	if stats.ApprovedLast24h != 2 {
		t.Errorf("Expected 2 approved in last 24h, got %d", stats.ApprovedLast24h)
	}
	
	if stats.RejectedLast24h != 1 {
		t.Errorf("Expected 1 rejected in last 24h, got %d", stats.RejectedLast24h)
	}
	
	if stats.AutoApprovedLast24h != 1 {
		t.Errorf("Expected 1 auto-approved in last 24h, got %d", stats.AutoApprovedLast24h)
	}
}