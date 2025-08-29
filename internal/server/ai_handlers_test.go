package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Mock AI service for testing
type mockAIServiceForHandlers struct {
	shouldFail bool
}

func (m *mockAIServiceForHandlers) AnalyzeContent(title, content string) (*models.AIFeedback, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("AI service error")
	}
	
	return &models.AIFeedback{
		Provider:     "mock",
		QualityScore: 0.85,
		Issues: []models.AIIssue{
			{
				Type:        "grammar",
				Severity:    "low",
				Description: "Minor issue",
				Location:    "paragraph 1",
				Suggestion:  "Fix this",
			},
		},
		Suggestions: []models.AISuggestion{
			{
				Type:        "title",
				Priority:    "medium",
				Description: "Improve title",
				Original:    title,
				Suggested:   "Better " + title,
			},
		},
		FlaggedContent:   []models.AIFlaggedContent{},
		ProcessingTimeMs: 100,
		Confidence:       0.90,
	}, nil
}

func (m *mockAIServiceForHandlers) GenerateMetaDescription(title, content string) (string, error) {
	if m.shouldFail {
		return "", fmt.Errorf("AI service error")
	}
	return "Generated meta description for " + title, nil
}

func (m *mockAIServiceForHandlers) GenerateTitle(content string) (string, error) {
	if m.shouldFail {
		return "", fmt.Errorf("AI service error")
	}
	return "AI Generated Title", nil
}

func (m *mockAIServiceForHandlers) CheckGrammar(text string) ([]models.AIIssue, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("AI service error")
	}
	return []models.AIIssue{
		{
			Type:        "grammar",
			Severity:    "medium",
			Description: "Grammar issue found",
			Location:    "sentence 1",
			Suggestion:  "Fix grammar",
		},
	}, nil
}

func (m *mockAIServiceForHandlers) CheckReadability(text string) (float64, error) {
	if m.shouldFail {
		return 0, fmt.Errorf("AI service error")
	}
	return 0.78, nil
}

func (m *mockAIServiceForHandlers) CheckAppropriateness(text string) (float64, []models.AIFlaggedContent, error) {
	if m.shouldFail {
		return 0, nil, fmt.Errorf("AI service error")
	}
	return 0.95, []models.AIFlaggedContent{}, nil
}

// Mock bulk AI service for testing
type mockBulkAIServiceForHandlers struct {
	shouldFail bool
}

func (m *mockBulkAIServiceForHandlers) OptimizeArticlesBulk(ctx context.Context, req *services.BulkOptimizationRequest) (*services.BulkOptimizationResult, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("bulk service error")
	}
	
	return &services.BulkOptimizationResult{
		TotalProcessed:   2,
		TotalOptimized:   2,
		TotalErrors:      0,
		ProcessingTimeMs: 1000,
		Results: []services.ArticleOptimizationResult{
			{
				ArticleID:        1,
				OriginalTitle:    "Original Title",
				OptimizedTitle:   stringPtr("Optimized Title"),
				OriginalMeta:     "Original Meta",
				OptimizedMeta:    stringPtr("Optimized Meta"),
				QualityScore:     float64Ptr(0.85),
				SchemaGenerated:  true,
				ProcessingTimeMs: 500,
			},
		},
		Errors: []services.BulkOptimizationError{},
	}, nil
}

func (m *mockBulkAIServiceForHandlers) GenerateQualityReport(ctx context.Context, articleIDs []uint64) (*services.QualityReport, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("quality report error")
	}
	
	return &services.QualityReport{
		TotalArticles:    len(articleIDs),
		ProcessedAt:      time.Now(),
		ProcessingTimeMs: 2000,
		Articles: []services.ArticleQualityResult{
			{
				ArticleID:    articleIDs[0],
				Title:        "Test Article",
				QualityScore: 0.88,
				Issues:       1,
				Suggestions:  2,
				Flagged:      false,
			},
		},
		QualityStats: &services.QualityStats{
			AverageScore:  0.88,
			MinScore:      0.88,
			MaxScore:      0.88,
			HighQuality:   1,
			MediumQuality: 0,
			LowQuality:    0,
		},
		Errors: []string{},
	}, nil
}

// Mock LLMs.txt service for testing
type mockLLMsTxtServiceForHandlers struct {
	shouldFail bool
}

func (m *mockLLMsTxtServiceForHandlers) GenerateLLMsTxt() (string, error) {
	if m.shouldFail {
		return "", fmt.Errorf("llms.txt generation error")
	}
	return "# llms.txt - AI-Ready News Website Data\n\nGenerated: 2024-01-01 12:00:00 UTC\n\n## Site Information\nName: Test Site", nil
}

func (m *mockLLMsTxtServiceForHandlers) GenerateLLMsTxtContent() (*services.LLMsTxtContent, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("llms.txt content generation error")
	}
	
	return &services.LLMsTxtContent{
		GeneratedAt: time.Now(),
		SiteInfo: services.SiteInfo{
			Name: "Test Site",
			URL:  "https://test.com",
			Type: "news_website",
		},
		Content: services.ContentSummary{
			TotalArticles: 10,
		},
		Categories:   []services.CategoryInfo{},
		Tags:         []services.TagInfo{},
		Authors:      []services.AuthorInfo{},
		RecentNews:   []services.ArticleSummary{},
		PopularNews:  []services.ArticleSummary{},
		APIEndpoints: []services.APIEndpoint{},
	}, nil
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func TestAIHandlers_AnalyzeContentHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	// Test successful analysis
	reqBody := `{"title": "Test Article", "content": "Test content for analysis"}`
	req := httptest.NewRequest("POST", "/api/v1/ai/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.AnalyzeContentHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response models.AIFeedback
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got %s", response.Provider)
	}
	
	if response.QualityScore != 0.85 {
		t.Errorf("Expected quality score 0.85, got %f", response.QualityScore)
	}
}

func TestAIHandlers_AnalyzeContentHandler_InvalidRequest(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	// Test missing title
	reqBody := `{"content": "Test content"}`
	req := httptest.NewRequest("POST", "/api/v1/ai/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.AnalyzeContentHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAIHandlers_GenerateMetaDescriptionHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"title": "Test Article", "content": "Test content"}`
	req := httptest.NewRequest("POST", "/api/v1/ai/generate-meta", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.GenerateMetaDescriptionHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response struct {
		MetaDescription string `json:"meta_description"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	expected := "Generated meta description for Test Article"
	if response.MetaDescription != expected {
		t.Errorf("Expected meta description '%s', got '%s'", expected, response.MetaDescription)
	}
}

func TestAIHandlers_GenerateTitleHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"content": "Test content for title generation"}`
	req := httptest.NewRequest("POST", "/api/v1/ai/generate-title", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.GenerateTitleHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Title != "AI Generated Title" {
		t.Errorf("Expected title 'AI Generated Title', got '%s'", response.Title)
	}
}

func TestAIHandlers_BulkOptimizeHandler(t *testing.T) {
	mockBulk := &mockBulkAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(nil, mockBulk, nil)
	
	reqBody := `{
		"article_ids": [1, 2],
		"optimize_title": true,
		"optimize_meta": true,
		"check_quality": true
	}`
	req := httptest.NewRequest("POST", "/api/v1/ai/bulk-optimize", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.BulkOptimizeHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response services.BulkOptimizationResult
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.TotalProcessed != 2 {
		t.Errorf("Expected 2 processed articles, got %d", response.TotalProcessed)
	}
	
	if response.TotalOptimized != 2 {
		t.Errorf("Expected 2 optimized articles, got %d", response.TotalOptimized)
	}
}

func TestAIHandlers_QualityReportHandler(t *testing.T) {
	mockBulk := &mockBulkAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(nil, mockBulk, nil)
	
	reqBody := `{"article_ids": [1, 2, 3]}`
	req := httptest.NewRequest("POST", "/api/v1/ai/quality-report", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.QualityReportHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response services.QualityReport
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.TotalArticles != 3 {
		t.Errorf("Expected 3 total articles, got %d", response.TotalArticles)
	}
	
	if len(response.Articles) != 1 {
		t.Errorf("Expected 1 article result, got %d", len(response.Articles))
	}
}

func TestAIHandlers_LLMsTxtHandler(t *testing.T) {
	mockLLMs := &mockLLMsTxtServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(nil, nil, mockLLMs)
	
	req := httptest.NewRequest("GET", "/llms.txt", nil)
	w := httptest.NewRecorder()
	
	handlers.LLMsTxtHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected content type 'text/plain; charset=utf-8', got '%s'", contentType)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "# llms.txt - AI-Ready News Website Data") {
		t.Error("Expected llms.txt header not found in response")
	}
}

func TestAIHandlers_LLMsTxtJSONHandler(t *testing.T) {
	mockLLMs := &mockLLMsTxtServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(nil, nil, mockLLMs)
	
	req := httptest.NewRequest("GET", "/api/v1/llms.json", nil)
	w := httptest.NewRecorder()
	
	handlers.LLMsTxtJSONHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", contentType)
	}
	
	var response services.LLMsTxtContent
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}
	
	if response.SiteInfo.Name != "Test Site" {
		t.Errorf("Expected site name 'Test Site', got '%s'", response.SiteInfo.Name)
	}
}

func TestAIHandlers_CheckGrammarHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"text": "This is test text for grammar checking."}`
	req := httptest.NewRequest("POST", "/api/v1/ai/check-grammar", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.CheckGrammarHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response struct {
		Issues []interface{} `json:"issues"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(response.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(response.Issues))
	}
}

func TestAIHandlers_CheckReadabilityHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"text": "This is test text for readability checking."}`
	req := httptest.NewRequest("POST", "/api/v1/ai/check-readability", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.CheckReadabilityHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response struct {
		ReadabilityScore float64 `json:"readability_score"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.ReadabilityScore != 0.78 {
		t.Errorf("Expected readability score 0.78, got %f", response.ReadabilityScore)
	}
}

func TestAIHandlers_CheckAppropriatenessHandler(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"text": "This is test text for appropriateness checking."}`
	req := httptest.NewRequest("POST", "/api/v1/ai/check-appropriateness", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.CheckAppropriatenessHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response struct {
		AppropriatenessScore float64       `json:"appropriateness_score"`
		FlaggedContent       []interface{} `json:"flagged_content"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.AppropriatenessScore != 0.95 {
		t.Errorf("Expected appropriateness score 0.95, got %f", response.AppropriatenessScore)
	}
	
	if len(response.FlaggedContent) != 0 {
		t.Errorf("Expected 0 flagged content items, got %d", len(response.FlaggedContent))
	}
}

func TestAIHandlers_ErrorHandling(t *testing.T) {
	// Test with failing AI service
	mockAI := &mockAIServiceForHandlers{shouldFail: true}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	reqBody := `{"title": "Test", "content": "Test content"}`
	req := httptest.NewRequest("POST", "/api/v1/ai/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.AnalyzeContentHandler(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "Analysis failed") {
		t.Error("Expected error message not found in response")
	}
}

func TestAIHandlers_MethodNotAllowed(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	// Test GET request on POST endpoint
	req := httptest.NewRequest("GET", "/api/v1/ai/analyze", nil)
	w := httptest.NewRecorder()
	
	handlers.AnalyzeContentHandler(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestAIHandlers_InvalidJSON(t *testing.T) {
	mockAI := &mockAIServiceForHandlers{shouldFail: false}
	handlers := NewAIHandlers(mockAI, nil, nil)
	
	// Test invalid JSON
	req := httptest.NewRequest("POST", "/api/v1/ai/analyze", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handlers.AnalyzeContentHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}