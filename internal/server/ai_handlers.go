package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/services"
)

// AIHandlers contains handlers for AI-related endpoints
type AIHandlers struct {
	aiService     services.AIService
	bulkAIService *services.BulkAIService
	llmsTxtService *services.LLMsTxtService
}

// NewAIHandlers creates new AI handlers
func NewAIHandlers(
	aiService services.AIService,
	bulkAIService *services.BulkAIService,
	llmsTxtService *services.LLMsTxtService,
) *AIHandlers {
	return &AIHandlers{
		aiService:      aiService,
		bulkAIService:  bulkAIService,
		llmsTxtService: llmsTxtService,
	}
}

// AnalyzeContentHandler handles content analysis requests
func (h *AIHandlers) AnalyzeContentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Title == "" || req.Content == "" {
		http.Error(w, "Title and content are required", http.StatusBadRequest)
		return
	}
	
	feedback, err := h.aiService.AnalyzeContent(req.Title, req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feedback)
}

// GenerateMetaDescriptionHandler handles meta description generation
func (h *AIHandlers) GenerateMetaDescriptionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Title == "" || req.Content == "" {
		http.Error(w, "Title and content are required", http.StatusBadRequest)
		return
	}
	
	metaDescription, err := h.aiService.GenerateMetaDescription(req.Title, req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Meta description generation failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		MetaDescription string `json:"meta_description"`
	}{
		MetaDescription: metaDescription,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GenerateTitleHandler handles title generation
func (h *AIHandlers) GenerateTitleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Content string `json:"content"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}
	
	title, err := h.aiService.GenerateTitle(req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Title generation failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		Title string `json:"title"`
	}{
		Title: title,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BulkOptimizeHandler handles bulk content optimization
func (h *AIHandlers) BulkOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req services.BulkOptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Set reasonable defaults
	if req.MaxArticles == 0 {
		req.MaxArticles = 100 // Default limit
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	
	result, err := h.bulkAIService.OptimizeArticlesBulk(ctx, &req)
	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timeout", http.StatusRequestTimeout)
			return
		}
		http.Error(w, fmt.Sprintf("Bulk optimization failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// QualityReportHandler handles quality report generation
func (h *AIHandlers) QualityReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ArticleIDs []uint64 `json:"article_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if len(req.ArticleIDs) == 0 {
		http.Error(w, "Article IDs are required", http.StatusBadRequest)
		return
	}
	
	// Limit the number of articles for performance
	if len(req.ArticleIDs) > 500 {
		http.Error(w, "Too many articles (max 500)", http.StatusBadRequest)
		return
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	
	report, err := h.bulkAIService.GenerateQualityReport(ctx, req.ArticleIDs)
	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timeout", http.StatusRequestTimeout)
			return
		}
		http.Error(w, fmt.Sprintf("Quality report generation failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// LLMsTxtHandler serves the llms.txt file
func (h *AIHandlers) LLMsTxtHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	llmsTxt, err := h.llmsTxtService.GenerateLLMsTxt()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate llms.txt: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Write([]byte(llmsTxt))
}

// LLMsTxtJSONHandler serves the llms.txt content as JSON
func (h *AIHandlers) LLMsTxtJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	content, err := h.llmsTxtService.GenerateLLMsTxtContent()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate llms.txt content: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	json.NewEncoder(w).Encode(content)
}

// CheckGrammarHandler handles grammar checking requests
func (h *AIHandlers) CheckGrammarHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}
	
	issues, err := h.aiService.CheckGrammar(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("Grammar check failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		Issues []interface{} `json:"issues"`
	}{
		Issues: make([]interface{}, len(issues)),
	}
	
	for i, issue := range issues {
		response.Issues[i] = issue
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckReadabilityHandler handles readability checking requests
func (h *AIHandlers) CheckReadabilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}
	
	score, err := h.aiService.CheckReadability(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("Readability check failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		ReadabilityScore float64 `json:"readability_score"`
	}{
		ReadabilityScore: score,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckAppropriatenessHandler handles appropriateness checking requests
func (h *AIHandlers) CheckAppropriatenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}
	
	score, flaggedContent, err := h.aiService.CheckAppropriateness(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("Appropriateness check failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		AppropriatenessScore float64       `json:"appropriateness_score"`
		FlaggedContent       []interface{} `json:"flagged_content"`
	}{
		AppropriatenessScore: score,
		FlaggedContent:       make([]interface{}, len(flaggedContent)),
	}
	
	for i, flagged := range flaggedContent {
		response.FlaggedContent[i] = flagged
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BulkOptimizeStatusHandler handles checking the status of bulk optimization
func (h *AIHandlers) BulkOptimizeStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract job ID from URL path or query parameter
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		// Try to extract from path
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) > 0 {
			jobID = pathParts[len(pathParts)-1]
		}
	}
	
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}
	
	// For now, return a simple status response
	// In a real implementation, you would track job status in a database or cache
	response := struct {
		JobID     string `json:"job_id"`
		Status    string `json:"status"`
		Message   string `json:"message"`
		Progress  int    `json:"progress"`
		CreatedAt string `json:"created_at"`
	}{
		JobID:     jobID,
		Status:    "completed", // This would be dynamic in a real implementation
		Message:   "Bulk optimization completed successfully",
		Progress:  100,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterAIRoutes registers all AI-related routes
func (s *Server) RegisterAIRoutes() {
	if s.aiHandlers == nil {
		return // AI handlers not initialized
	}
	
	// Content analysis endpoints
	http.HandleFunc("/api/v1/ai/analyze", s.aiHandlers.AnalyzeContentHandler)
	http.HandleFunc("/api/v1/ai/generate-meta", s.aiHandlers.GenerateMetaDescriptionHandler)
	http.HandleFunc("/api/v1/ai/generate-title", s.aiHandlers.GenerateTitleHandler)
	
	// Individual AI checks
	http.HandleFunc("/api/v1/ai/check-grammar", s.aiHandlers.CheckGrammarHandler)
	http.HandleFunc("/api/v1/ai/check-readability", s.aiHandlers.CheckReadabilityHandler)
	http.HandleFunc("/api/v1/ai/check-appropriateness", s.aiHandlers.CheckAppropriatenessHandler)
	
	// Bulk operations
	http.HandleFunc("/api/v1/ai/bulk-optimize", s.aiHandlers.BulkOptimizeHandler)
	http.HandleFunc("/api/v1/ai/quality-report", s.aiHandlers.QualityReportHandler)
	http.HandleFunc("/api/v1/ai/bulk-status/", s.aiHandlers.BulkOptimizeStatusHandler)
	
	// LLMs.txt endpoints
	http.HandleFunc("/llms.txt", s.aiHandlers.LLMsTxtHandler)
	http.HandleFunc("/api/v1/llms.json", s.aiHandlers.LLMsTxtJSONHandler)
}

// Middleware for AI endpoints (rate limiting, authentication, etc.)
func (h *AIHandlers) AIMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Add rate limiting here if needed
		// For now, just call the next handler
		next(w, r)
	}
}