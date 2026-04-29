package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

func TestOpenAIService_AnalyzeContent(t *testing.T) {
	// Create mock OpenAI server
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "{\"quality_score\": 0.85, \"grammar_score\": 0.90, \"readability_score\": 0.80, \"appropriateness_score\": 0.95, \"confidence\": 0.88, \"issues\": [{\"type\": \"grammar\", \"severity\": \"medium\", \"description\": \"Subject-verb disagreement\", \"location\": \"paragraph 2\", \"suggestion\": \"Change 'are' to 'is'\"}], \"suggestions\": [{\"type\": \"title\", \"priority\": \"medium\", \"description\": \"Title could be more engaging\", \"original\": \"Current title\", \"suggested\": \"Improved title\"}], \"flagged_content\": []}"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Bearer token in Authorization header, got %s", authHeader)
		}
		
		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	// Create OpenAI service with mock server
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	// Test analyze content
	feedback, err := aiService.AnalyzeContent("Test Article", "This is test content for analysis.")
	if err != nil {
		t.Fatalf("Failed to analyze content: %v", err)
	}
	
	if feedback.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %s", feedback.Provider)
	}
	
	if feedback.QualityScore != 0.85 {
		t.Errorf("Expected quality score 0.85, got %f", feedback.QualityScore)
	}
	
	if feedback.GrammarScore == nil || *feedback.GrammarScore != 0.90 {
		t.Errorf("Expected grammar score 0.90, got %v", feedback.GrammarScore)
	}
	
	if feedback.ReadabilityScore == nil || *feedback.ReadabilityScore != 0.80 {
		t.Errorf("Expected readability score 0.80, got %v", feedback.ReadabilityScore)
	}
	
	if feedback.AppropriatenessScore == nil || *feedback.AppropriatenessScore != 0.95 {
		t.Errorf("Expected appropriateness score 0.95, got %v", feedback.AppropriatenessScore)
	}
	
	if feedback.Confidence != 0.88 {
		t.Errorf("Expected confidence 0.88, got %f", feedback.Confidence)
	}
	
	if len(feedback.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(feedback.Issues))
	} else {
		issue := feedback.Issues[0]
		if issue.Type != "grammar" {
			t.Errorf("Expected issue type 'grammar', got %s", issue.Type)
		}
		if issue.Severity != "medium" {
			t.Errorf("Expected issue severity 'medium', got %s", issue.Severity)
		}
		if issue.Description != "Subject-verb disagreement" {
			t.Errorf("Expected issue description 'Subject-verb disagreement', got %s", issue.Description)
		}
	}
	
	if len(feedback.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(feedback.Suggestions))
	} else {
		suggestion := feedback.Suggestions[0]
		if suggestion.Type != "title" {
			t.Errorf("Expected suggestion type 'title', got %s", suggestion.Type)
		}
		if suggestion.Priority != "medium" {
			t.Errorf("Expected suggestion priority 'medium', got %s", suggestion.Priority)
		}
	}
	
	if len(feedback.FlaggedContent) != 0 {
		t.Errorf("Expected 0 flagged content, got %d", len(feedback.FlaggedContent))
	}
	
	if feedback.ProcessingTimeMs <= 0 {
		t.Errorf("Expected positive processing time, got %d", feedback.ProcessingTimeMs)
	}
}

func TestOpenAIService_GenerateMetaDescription(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "This is a compelling meta description for the test article that summarizes the main points effectively."
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	metaDescription, err := aiService.GenerateMetaDescription("Test Article", "This is test content.")
	if err != nil {
		t.Fatalf("Failed to generate meta description: %v", err)
	}
	
	expected := "This is a compelling meta description for the test article that summarizes the main points effectively."
	if metaDescription != expected {
		t.Errorf("Expected meta description '%s', got '%s'", expected, metaDescription)
	}
}

func TestOpenAIService_GenerateTitle(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "Engaging News Title That Captures Attention"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	title, err := aiService.GenerateTitle("This is test content for title generation.")
	if err != nil {
		t.Fatalf("Failed to generate title: %v", err)
	}
	
	expected := "Engaging News Title That Captures Attention"
	if title != expected {
		t.Errorf("Expected title '%s', got '%s'", expected, title)
	}
}

func TestOpenAIService_CheckGrammar(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "{\"issues\": [{\"type\": \"grammar\", \"severity\": \"high\", \"description\": \"Missing comma\", \"location\": \"sentence 1\", \"suggestion\": \"Add comma after 'However'\"}]}"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	issues, err := aiService.CheckGrammar("However the weather was nice.")
	if err != nil {
		t.Fatalf("Failed to check grammar: %v", err)
	}
	
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	} else {
		issue := issues[0]
		if issue.Type != "grammar" {
			t.Errorf("Expected issue type 'grammar', got %s", issue.Type)
		}
		if issue.Severity != "high" {
			t.Errorf("Expected issue severity 'high', got %s", issue.Severity)
		}
		if issue.Description != "Missing comma" {
			t.Errorf("Expected issue description 'Missing comma', got %s", issue.Description)
		}
		if issue.Suggestion != "Add comma after 'However'" {
			t.Errorf("Expected suggestion 'Add comma after 'However'', got %s", issue.Suggestion)
		}
	}
}

func TestOpenAIService_CheckReadability(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "0.75"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	score, err := aiService.CheckReadability("This is a test sentence for readability analysis.")
	if err != nil {
		t.Fatalf("Failed to check readability: %v", err)
	}
	
	if score != 0.75 {
		t.Errorf("Expected readability score 0.75, got %f", score)
	}
}

func TestOpenAIService_CheckAppropriateness(t *testing.T) {
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "{\"appropriateness_score\": 0.92, \"flagged_content\": [{\"type\": \"inappropriate\", \"content\": \"questionable phrase\", \"reason\": \"potentially offensive\", \"confidence\": 0.65}]}"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-api-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	score, flaggedContent, err := aiService.CheckAppropriateness("This text contains a questionable phrase.")
	if err != nil {
		t.Fatalf("Failed to check appropriateness: %v", err)
	}
	
	if score != 0.92 {
		t.Errorf("Expected appropriateness score 0.92, got %f", score)
	}
	
	if len(flaggedContent) != 1 {
		t.Errorf("Expected 1 flagged content item, got %d", len(flaggedContent))
	} else {
		flagged := flaggedContent[0]
		if flagged.Type != "inappropriate" {
			t.Errorf("Expected flagged type 'inappropriate', got %s", flagged.Type)
		}
		if flagged.Content != "questionable phrase" {
			t.Errorf("Expected flagged content 'questionable phrase', got %s", flagged.Content)
		}
		if flagged.Reason != "potentially offensive" {
			t.Errorf("Expected flagged reason 'potentially offensive', got %s", flagged.Reason)
		}
		if flagged.Confidence != 0.65 {
			t.Errorf("Expected flagged confidence 0.65, got %f", flagged.Confidence)
		}
	}
}

func TestOpenAIService_ErrorHandling(t *testing.T) {
	// Test API error response
	errorResponse := `{
		"error": {
			"message": "Invalid API key"
		}
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(errorResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("invalid-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	_, err := aiService.AnalyzeContent("Test", "Content")
	if err == nil {
		t.Error("Expected error for invalid API key")
	}
	
	if !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("Expected API error message, got: %v", err)
	}
}

func TestOpenAIService_InvalidJSON(t *testing.T) {
	// Test invalid JSON response
	mockResponse := `{
		"choices": [{
			"message": {
				"content": "invalid json content"
			}
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	_, err := aiService.AnalyzeContent("Test", "Content")
	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
	
	if !strings.Contains(err.Error(), "failed to parse AI response") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestAnthropicService_AnalyzeContent(t *testing.T) {
	// Create mock Anthropic server
	mockResponse := `{
		"content": [{
			"text": "{\"quality_score\": 0.88, \"grammar_score\": 0.92, \"readability_score\": 0.85, \"appropriateness_score\": 0.96, \"confidence\": 0.90, \"issues\": [], \"suggestions\": [], \"flagged_content\": []}"
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		
		apiKey := r.Header.Get("x-api-key")
		if apiKey != "test-api-key" {
			t.Errorf("Expected x-api-key test-api-key, got %s", apiKey)
		}
		
		version := r.Header.Get("anthropic-version")
		if version != "2023-06-01" {
			t.Errorf("Expected anthropic-version 2023-06-01, got %s", version)
		}
		
		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	// Create Anthropic service with mock server
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	// Test analyze content
	feedback, err := aiService.AnalyzeContent("Test Article", "This is test content for analysis.")
	if err != nil {
		t.Fatalf("Failed to analyze content: %v", err)
	}
	
	if feedback.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", feedback.Provider)
	}
	
	if feedback.QualityScore != 0.88 {
		t.Errorf("Expected quality score 0.88, got %f", feedback.QualityScore)
	}
	
	if feedback.GrammarScore == nil || *feedback.GrammarScore != 0.92 {
		t.Errorf("Expected grammar score 0.92, got %v", feedback.GrammarScore)
	}
	
	if feedback.Confidence != 0.90 {
		t.Errorf("Expected confidence 0.90, got %f", feedback.Confidence)
	}
}

func TestAIService_ModelDefaults(t *testing.T) {
	// Test OpenAI default model
	openAIService := NewOpenAIService("test-key", "")
	if openAIService.model != "gpt-4o-mini" {
		t.Errorf("Expected default OpenAI model 'gpt-4o-mini', got %s", openAIService.model)
	}
	
	// Test Anthropic default model
	anthropicService := NewAnthropicService("test-key", "")
	if anthropicService.model != "claude-3-haiku-20240307" {
		t.Errorf("Expected default Anthropic model 'claude-3-haiku-20240307', got %s", anthropicService.model)
	}
}

func TestAIService_RequestTimeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than 30s timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "response"}}]}`))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	_, err := aiService.AnalyzeContent("Test", "Content")
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestAnthropicService_CheckGrammar(t *testing.T) {
	mockResponse := `{
		"content": [{
			"text": "{\"issues\": [{\"type\": \"grammar\", \"severity\": \"high\", \"description\": \"Missing comma\", \"location\": \"sentence 1\", \"suggestion\": \"Add comma after 'However'\"}]}"
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	issues, err := aiService.CheckGrammar("However the weather was nice.")
	if err != nil {
		t.Fatalf("Failed to check grammar: %v", err)
	}
	
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	} else {
		issue := issues[0]
		if issue.Type != "grammar" {
			t.Errorf("Expected issue type 'grammar', got %s", issue.Type)
		}
		if issue.Severity != "high" {
			t.Errorf("Expected issue severity 'high', got %s", issue.Severity)
		}
		if issue.Description != "Missing comma" {
			t.Errorf("Expected issue description 'Missing comma', got %s", issue.Description)
		}
	}
}

func TestAnthropicService_CheckReadability(t *testing.T) {
	mockResponse := `{
		"content": [{
			"text": "0.82"
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	score, err := aiService.CheckReadability("This is a test sentence for readability analysis.")
	if err != nil {
		t.Fatalf("Failed to check readability: %v", err)
	}
	
	if score != 0.82 {
		t.Errorf("Expected readability score 0.82, got %f", score)
	}
}

func TestAnthropicService_CheckAppropriateness(t *testing.T) {
	mockResponse := `{
		"content": [{
			"text": "{\"appropriateness_score\": 0.94, \"flagged_content\": [{\"type\": \"inappropriate\", \"content\": \"questionable phrase\", \"reason\": \"potentially offensive\", \"confidence\": 0.68}]}"
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	score, flaggedContent, err := aiService.CheckAppropriateness("This text contains a questionable phrase.")
	if err != nil {
		t.Fatalf("Failed to check appropriateness: %v", err)
	}
	
	if score != 0.94 {
		t.Errorf("Expected appropriateness score 0.94, got %f", score)
	}
	
	if len(flaggedContent) != 1 {
		t.Errorf("Expected 1 flagged content item, got %d", len(flaggedContent))
	} else {
		flagged := flaggedContent[0]
		if flagged.Type != "inappropriate" {
			t.Errorf("Expected flagged type 'inappropriate', got %s", flagged.Type)
		}
		if flagged.Content != "questionable phrase" {
			t.Errorf("Expected flagged content 'questionable phrase', got %s", flagged.Content)
		}
		if flagged.Reason != "potentially offensive" {
			t.Errorf("Expected flagged reason 'potentially offensive', got %s", flagged.Reason)
		}
		if flagged.Confidence != 0.68 {
			t.Errorf("Expected flagged confidence 0.68, got %f", flagged.Confidence)
		}
	}
}

func TestAnthropicService_GenerateMetaDescription(t *testing.T) {
	mockResponse := `{
		"content": [{
			"text": "This is a compelling meta description generated by Anthropic Claude for the test article."
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	metaDescription, err := aiService.GenerateMetaDescription("Test Article", "This is test content.")
	if err != nil {
		t.Fatalf("Failed to generate meta description: %v", err)
	}
	
	expected := "This is a compelling meta description generated by Anthropic Claude for the test article."
	if metaDescription != expected {
		t.Errorf("Expected meta description '%s', got '%s'", expected, metaDescription)
	}
}

func TestAnthropicService_GenerateTitle(t *testing.T) {
	mockResponse := `{
		"content": [{
			"text": "Claude-Generated Engaging News Title"
		}]
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("test-api-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	title, err := aiService.GenerateTitle("This is test content for title generation.")
	if err != nil {
		t.Fatalf("Failed to generate title: %v", err)
	}
	
	expected := "Claude-Generated Engaging News Title"
	if title != expected {
		t.Errorf("Expected title '%s', got '%s'", expected, title)
	}
}

func TestAnthropicService_ErrorHandling(t *testing.T) {
	// Test API error response
	errorResponse := `{
		"error": {
			"message": "Invalid API key"
		}
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(errorResponse))
	}))
	defer server.Close()
	
	aiService := NewAnthropicService("invalid-key", "claude-3-haiku-20240307")
	aiService.baseURL = server.URL
	
	_, err := aiService.AnalyzeContent("Test", "Content")
	if err == nil {
		t.Error("Expected error for invalid API key")
	}
	
	if !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("Expected API error message, got: %v", err)
	}
}

func TestAIService_ComprehensiveAnalysis(t *testing.T) {
	// Test comprehensive analysis with both OpenAI and Anthropic
	testCases := []struct {
		name      string
		service   AIService
		setupMock func() *httptest.Server
	}{
		{
			name: "OpenAI Comprehensive Analysis",
			setupMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mockResponse := `{
						"choices": [{
							"message": {
								"content": "{\"quality_score\": 0.92, \"grammar_score\": 0.88, \"readability_score\": 0.85, \"appropriateness_score\": 0.96, \"confidence\": 0.91, \"issues\": [{\"type\": \"grammar\", \"severity\": \"low\", \"description\": \"Minor punctuation issue\", \"location\": \"paragraph 3\", \"suggestion\": \"Add comma\"}], \"suggestions\": [{\"type\": \"seo\", \"priority\": \"high\", \"description\": \"Optimize for search engines\", \"original\": \"current text\", \"suggested\": \"optimized text\"}], \"flagged_content\": []}"
							}
						}]
					}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(mockResponse))
				}))
			},
		},
		{
			name: "Anthropic Comprehensive Analysis",
			setupMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mockResponse := `{
						"content": [{
							"text": "{\"quality_score\": 0.89, \"grammar_score\": 0.93, \"readability_score\": 0.87, \"appropriateness_score\": 0.98, \"confidence\": 0.94, \"issues\": [], \"suggestions\": [{\"type\": \"content\", \"priority\": \"medium\", \"description\": \"Enhance content depth\", \"original\": \"shallow content\", \"suggested\": \"detailed content\"}], \"flagged_content\": []}"
						}]
					}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(mockResponse))
				}))
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupMock()
			defer server.Close()
			
			var aiService AIService
			if strings.Contains(tc.name, "OpenAI") {
				openAIService := NewOpenAIService("test-key", "gpt-4o-mini")
				openAIService.baseURL = server.URL
				aiService = openAIService
			} else {
				anthropicService := NewAnthropicService("test-key", "claude-3-haiku-20240307")
				anthropicService.baseURL = server.URL
				aiService = anthropicService
			}
			
			// Test comprehensive analysis
			feedback, err := aiService.AnalyzeContent("Test News Article", "This is comprehensive test content for AI analysis.")
			if err != nil {
				t.Fatalf("Failed to analyze content: %v", err)
			}
			
			// Verify quality scores are reasonable
			if feedback.QualityScore < 0.8 || feedback.QualityScore > 1.0 {
				t.Errorf("Expected quality score between 0.8-1.0, got %f", feedback.QualityScore)
			}
			
			if feedback.GrammarScore != nil && (*feedback.GrammarScore < 0.8 || *feedback.GrammarScore > 1.0) {
				t.Errorf("Expected grammar score between 0.8-1.0, got %f", *feedback.GrammarScore)
			}
			
			if feedback.ReadabilityScore != nil && (*feedback.ReadabilityScore < 0.8 || *feedback.ReadabilityScore > 1.0) {
				t.Errorf("Expected readability score between 0.8-1.0, got %f", *feedback.ReadabilityScore)
			}
			
			if feedback.AppropriatenessScore != nil && (*feedback.AppropriatenessScore < 0.9 || *feedback.AppropriatenessScore > 1.0) {
				t.Errorf("Expected appropriateness score between 0.9-1.0, got %f", *feedback.AppropriatenessScore)
			}
			
			if feedback.Confidence < 0.9 || feedback.Confidence > 1.0 {
				t.Errorf("Expected confidence between 0.9-1.0, got %f", feedback.Confidence)
			}
			
			// Verify provider is set correctly
			expectedProvider := "openai"
			if strings.Contains(tc.name, "Anthropic") {
				expectedProvider = "anthropic"
			}
			
			if feedback.Provider != expectedProvider {
				t.Errorf("Expected provider '%s', got '%s'", expectedProvider, feedback.Provider)
			}
			
			// Verify processing time is recorded
			if feedback.ProcessingTimeMs <= 0 {
				t.Errorf("Expected positive processing time, got %d", feedback.ProcessingTimeMs)
			}
		})
	}
}

func TestAIService_RateLimitHandling(t *testing.T) {
	// Test rate limit handling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	_, err := aiService.AnalyzeContent("Test", "Content")
	if err == nil {
		t.Error("Expected rate limit error")
	}
	
	if !strings.Contains(err.Error(), "API request failed with status 429") {
		t.Errorf("Expected rate limit error message, got: %v", err)
	}
}

func TestAIService_LargeContentHandling(t *testing.T) {
	// Test handling of large content
	largeContent := strings.Repeat("This is a very long article content. ", 1000)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"choices": [{
				"message": {
					"content": "{\"quality_score\": 0.75, \"grammar_score\": 0.80, \"readability_score\": 0.70, \"appropriateness_score\": 0.95, \"confidence\": 0.85, \"issues\": [], \"suggestions\": [], \"flagged_content\": []}"
				}
			}]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()
	
	aiService := NewOpenAIService("test-key", "gpt-4o-mini")
	aiService.baseURL = server.URL
	
	feedback, err := aiService.AnalyzeContent("Long Article", largeContent)
	if err != nil {
		t.Fatalf("Failed to analyze large content: %v", err)
	}
	
	if feedback.QualityScore != 0.75 {
		t.Errorf("Expected quality score 0.75, got %f", feedback.QualityScore)
	}
}