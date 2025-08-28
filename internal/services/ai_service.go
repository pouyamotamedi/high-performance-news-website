package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// AIService interface for content quality analysis
type AIService interface {
	AnalyzeContent(title, content string) (*models.AIFeedback, error)
	GenerateMetaDescription(title, content string) (string, error)
	GenerateTitle(content string) (string, error)
	CheckGrammar(text string) ([]models.AIIssue, error)
	CheckReadability(text string) (float64, error)
	CheckAppropriateness(text string) (float64, []models.AIFlaggedContent, error)
}

// OpenAIService implements AIService using OpenAI API
type OpenAIService struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIService creates a new OpenAI service
func NewOpenAIService(apiKey, model string) *OpenAIService {
	if model == "" {
		model = "gpt-4o-mini" // Default to cost-effective model
	}
	
	return &OpenAIService{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AnalyzeContent performs comprehensive content analysis
func (ai *OpenAIService) AnalyzeContent(title, content string) (*models.AIFeedback, error) {
	startTime := time.Now()
	
	prompt := fmt.Sprintf(`
Analyze the following news article for quality, grammar, readability, and appropriateness. 
Provide a comprehensive assessment in JSON format.

Title: %s

Content: %s

Please provide analysis in this exact JSON format:
{
  "quality_score": 0.85,
  "grammar_score": 0.90,
  "readability_score": 0.80,
  "appropriateness_score": 0.95,
  "confidence": 0.88,
  "issues": [
    {
      "type": "grammar",
      "severity": "medium",
      "description": "Subject-verb disagreement in paragraph 2",
      "location": "paragraph 2",
      "suggestion": "Change 'are' to 'is'"
    }
  ],
  "suggestions": [
    {
      "type": "title",
      "priority": "medium",
      "description": "Title could be more engaging",
      "original": "Current title",
      "suggested": "Improved title"
    }
  ],
  "flagged_content": [
    {
      "type": "inappropriate",
      "content": "flagged text snippet",
      "reason": "potentially offensive language",
      "confidence": 0.75
    }
  ]
}

Scoring criteria:
- quality_score: Overall content quality (0.0-1.0)
- grammar_score: Grammar and spelling accuracy (0.0-1.0)
- readability_score: How easy the content is to read (0.0-1.0)
- appropriateness_score: Content appropriateness for news (0.0-1.0)
- confidence: AI confidence in the analysis (0.0-1.0)

Issue types: "grammar", "spelling", "readability", "inappropriate"
Issue severity: "low", "medium", "high"
Suggestion types: "title", "meta_description", "content", "seo"
Suggestion priority: "low", "medium", "high"
Flagged content types: "inappropriate", "spam", "low_quality"
`, title, content)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse the JSON response
	var analysisResult struct {
		QualityScore         float64                    `json:"quality_score"`
		GrammarScore         float64                    `json:"grammar_score"`
		ReadabilityScore     float64                    `json:"readability_score"`
		AppropriatenessScore float64                    `json:"appropriateness_score"`
		Confidence           float64                    `json:"confidence"`
		Issues               []models.AIIssue           `json:"issues"`
		Suggestions          []models.AISuggestion      `json:"suggestions"`
		FlaggedContent       []models.AIFlaggedContent  `json:"flagged_content"`
	}

	if err := json.Unmarshal([]byte(response), &analysisResult); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	feedback := &models.AIFeedback{
		Provider:             "openai",
		QualityScore:         analysisResult.QualityScore,
		GrammarScore:         &analysisResult.GrammarScore,
		ReadabilityScore:     &analysisResult.ReadabilityScore,
		AppropriatenessScore: &analysisResult.AppropriatenessScore,
		Issues:               analysisResult.Issues,
		Suggestions:          analysisResult.Suggestions,
		FlaggedContent:       analysisResult.FlaggedContent,
		ProcessingTimeMs:     processingTime,
		Confidence:           analysisResult.Confidence,
	}

	return feedback, nil
}

// GenerateMetaDescription generates SEO meta description
func (ai *OpenAIService) GenerateMetaDescription(title, content string) (string, error) {
	prompt := fmt.Sprintf(`
Generate a compelling SEO meta description (max 160 characters) for this news article:

Title: %s
Content: %s

Requirements:
- Maximum 160 characters
- Include main keywords
- Compelling and click-worthy
- Accurate summary of content
- News-appropriate tone

Return only the meta description text, no additional formatting.
`, title, content)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate meta description: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// GenerateTitle generates an improved title
func (ai *OpenAIService) GenerateTitle(content string) (string, error) {
	prompt := fmt.Sprintf(`
Generate an engaging, SEO-optimized news title for this article content:

Content: %s

Requirements:
- Maximum 60 characters for SEO
- Engaging and click-worthy
- Accurate representation of content
- News headline style
- Include main keywords

Return only the title text, no additional formatting.
`, content)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// CheckGrammar checks grammar and spelling
func (ai *OpenAIService) CheckGrammar(text string) ([]models.AIIssue, error) {
	prompt := fmt.Sprintf(`
Check the following text for grammar and spelling errors. Return results in JSON format:

Text: %s

Return format:
{
  "issues": [
    {
      "type": "grammar",
      "severity": "medium",
      "description": "Subject-verb disagreement",
      "location": "sentence 3",
      "suggestion": "Change 'are' to 'is'"
    }
  ]
}

Issue types: "grammar", "spelling"
Severity levels: "low", "medium", "high"
`, text)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return nil, fmt.Errorf("grammar check failed: %w", err)
	}

	var result struct {
		Issues []models.AIIssue `json:"issues"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse grammar check response: %w", err)
	}

	return result.Issues, nil
}

// CheckReadability calculates readability score
func (ai *OpenAIService) CheckReadability(text string) (float64, error) {
	prompt := fmt.Sprintf(`
Analyze the readability of this text and provide a score from 0.0 to 1.0 (1.0 being most readable).
Consider sentence length, word complexity, paragraph structure, and overall clarity.

Text: %s

Return only a decimal number between 0.0 and 1.0, no additional text.
`, text)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return 0, fmt.Errorf("readability check failed: %w", err)
	}

	var score float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &score); err != nil {
		return 0, fmt.Errorf("failed to parse readability score: %w", err)
	}

	return score, nil
}

// CheckAppropriateness checks content appropriateness
func (ai *OpenAIService) CheckAppropriateness(text string) (float64, []models.AIFlaggedContent, error) {
	prompt := fmt.Sprintf(`
Analyze this text for appropriateness in a news context. Check for inappropriate content, spam, or low quality.
Return results in JSON format:

Text: %s

Return format:
{
  "appropriateness_score": 0.95,
  "flagged_content": [
    {
      "type": "inappropriate",
      "content": "flagged text snippet",
      "reason": "potentially offensive language",
      "confidence": 0.75
    }
  ]
}

Flagged content types: "inappropriate", "spam", "low_quality"
Score: 0.0 (completely inappropriate) to 1.0 (completely appropriate)
`, text)

	response, err := ai.callOpenAI(prompt)
	if err != nil {
		return 0, nil, fmt.Errorf("appropriateness check failed: %w", err)
	}

	var result struct {
		AppropriatenessScore float64                   `json:"appropriateness_score"`
		FlaggedContent       []models.AIFlaggedContent `json:"flagged_content"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return 0, nil, fmt.Errorf("failed to parse appropriateness response: %w", err)
	}

	return result.AppropriatenessScore, result.FlaggedContent, nil
}

// callOpenAI makes a request to OpenAI API
func (ai *OpenAIService) callOpenAI(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": ai.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert content analyst for news articles. Provide accurate, detailed analysis in the requested format.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":   2000,
		"temperature":  0.3,
		"top_p":        1,
		"frequency_penalty": 0,
		"presence_penalty":  0,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", ai.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)

	resp, err := ai.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

// AnthropicService implements AIService using Anthropic Claude API
type AnthropicService struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewAnthropicService creates a new Anthropic service
func NewAnthropicService(apiKey, model string) *AnthropicService {
	if model == "" {
		model = "claude-3-haiku-20240307" // Default to cost-effective model
	}
	
	return &AnthropicService{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AnalyzeContent performs comprehensive content analysis using Anthropic
func (ai *AnthropicService) AnalyzeContent(title, content string) (*models.AIFeedback, error) {
	startTime := time.Now()
	
	prompt := fmt.Sprintf(`
Analyze the following news article for quality, grammar, readability, and appropriateness. 
Provide a comprehensive assessment in JSON format.

Title: %s

Content: %s

Please provide analysis in this exact JSON format:
{
  "quality_score": 0.85,
  "grammar_score": 0.90,
  "readability_score": 0.80,
  "appropriateness_score": 0.95,
  "confidence": 0.88,
  "issues": [
    {
      "type": "grammar",
      "severity": "medium",
      "description": "Subject-verb disagreement in paragraph 2",
      "location": "paragraph 2",
      "suggestion": "Change 'are' to 'is'"
    }
  ],
  "suggestions": [
    {
      "type": "title",
      "priority": "medium",
      "description": "Title could be more engaging",
      "original": "Current title",
      "suggested": "Improved title"
    }
  ],
  "flagged_content": [
    {
      "type": "inappropriate",
      "content": "flagged text snippet",
      "reason": "potentially offensive language",
      "confidence": 0.75
    }
  ]
}
`, title, content)

	response, err := ai.callAnthropic(prompt)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API call failed: %w", err)
	}

	// Parse the JSON response (similar to OpenAI implementation)
	var analysisResult struct {
		QualityScore         float64                    `json:"quality_score"`
		GrammarScore         float64                    `json:"grammar_score"`
		ReadabilityScore     float64                    `json:"readability_score"`
		AppropriatenessScore float64                    `json:"appropriateness_score"`
		Confidence           float64                    `json:"confidence"`
		Issues               []models.AIIssue           `json:"issues"`
		Suggestions          []models.AISuggestion      `json:"suggestions"`
		FlaggedContent       []models.AIFlaggedContent  `json:"flagged_content"`
	}

	if err := json.Unmarshal([]byte(response), &analysisResult); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	feedback := &models.AIFeedback{
		Provider:             "anthropic",
		QualityScore:         analysisResult.QualityScore,
		GrammarScore:         &analysisResult.GrammarScore,
		ReadabilityScore:     &analysisResult.ReadabilityScore,
		AppropriatenessScore: &analysisResult.AppropriatenessScore,
		Issues:               analysisResult.Issues,
		Suggestions:          analysisResult.Suggestions,
		FlaggedContent:       analysisResult.FlaggedContent,
		ProcessingTimeMs:     processingTime,
		Confidence:           analysisResult.Confidence,
	}

	return feedback, nil
}

// GenerateMetaDescription generates SEO meta description using Anthropic
func (ai *AnthropicService) GenerateMetaDescription(title, content string) (string, error) {
	prompt := fmt.Sprintf(`
Generate a compelling SEO meta description (max 160 characters) for this news article:

Title: %s
Content: %s

Requirements:
- Maximum 160 characters
- Include main keywords
- Compelling and click-worthy
- Accurate summary of content
- News-appropriate tone

Return only the meta description text, no additional formatting.
`, title, content)

	response, err := ai.callAnthropic(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate meta description: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// GenerateTitle generates an improved title using Anthropic
func (ai *AnthropicService) GenerateTitle(content string) (string, error) {
	prompt := fmt.Sprintf(`
Generate an engaging, SEO-optimized news title for this article content:

Content: %s

Requirements:
- Maximum 60 characters for SEO
- Engaging and click-worthy
- Accurate representation of content
- News headline style
- Include main keywords

Return only the title text, no additional formatting.
`, content)

	response, err := ai.callAnthropic(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// CheckGrammar checks grammar and spelling using Anthropic
func (ai *AnthropicService) CheckGrammar(text string) ([]models.AIIssue, error) {
	// Implementation similar to OpenAI but using Anthropic API
	return nil, fmt.Errorf("not implemented")
}

// CheckReadability calculates readability score using Anthropic
func (ai *AnthropicService) CheckReadability(text string) (float64, error) {
	// Implementation similar to OpenAI but using Anthropic API
	return 0, fmt.Errorf("not implemented")
}

// CheckAppropriateness checks content appropriateness using Anthropic
func (ai *AnthropicService) CheckAppropriateness(text string) (float64, []models.AIFlaggedContent, error) {
	// Implementation similar to OpenAI but using Anthropic API
	return 0, nil, fmt.Errorf("not implemented")
}

// callAnthropic makes a request to Anthropic API
func (ai *AnthropicService) callAnthropic(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":      ai.model,
		"max_tokens": 2000,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", ai.baseURL+"/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ai.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ai.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no response content returned")
	}

	return response.Content[0].Text, nil
}