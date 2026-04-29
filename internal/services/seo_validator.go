package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// SEOValidator provides comprehensive SEO validation capabilities
type SEOValidator struct {
	articleRepo  *repositories.ArticleRepository
	categoryRepo *repositories.CategoryRepository
	tagRepo      *repositories.TagRepository
	httpClient   *http.Client
	baseURL      string
	siteName     string
}

// NewSEOValidator creates a new SEO validator instance
func NewSEOValidator(
	articleRepo *repositories.ArticleRepository,
	categoryRepo *repositories.CategoryRepository,
	tagRepo *repositories.TagRepository,
	baseURL, siteName string,
) *SEOValidator {
	return &SEOValidator{
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects for canonical chain detection
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		baseURL:  baseURL,
		siteName: siteName,
	}
}

// CanonicalValidationResult represents the result of canonical URL validation
type CanonicalValidationResult struct {
	URL                string                    `json:"url"`
	IsValid            bool                      `json:"is_valid"`
	CanonicalURL       string                    `json:"canonical_url"`
	CanonicalChain     []string                  `json:"canonical_chain"`
	ChainLength        int                       `json:"chain_length"`
	HasCircularRef     bool                      `json:"has_circular_ref"`
	CircularRefPoint   string                    `json:"circular_ref_point,omitempty"`
	Issues             []CanonicalIssue          `json:"issues"`
	HTTPStatus         int                       `json:"http_status"`
	ResponseTime       time.Duration             `json:"response_time"`
	ValidationTime     time.Time                 `json:"validation_time"`
	Recommendations    []string                  `json:"recommendations"`
}

// CanonicalIssue represents a specific canonical URL issue
type CanonicalIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// SchemaValidationResult represents the result of schema markup validation
type SchemaValidationResult struct {
	URL            string         `json:"url"`
	IsValid        bool           `json:"is_valid"`
	SchemaTypes    []string       `json:"schema_types"`
	Issues         []SchemaIssue  `json:"issues"`
	ValidationTime time.Time      `json:"validation_time"`
	Recommendations []string      `json:"recommendations"`
}

// SchemaIssue represents a specific schema markup issue
type SchemaIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	SchemaType  string `json:"schema_type,omitempty"`
	Property    string `json:"property,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ValidateSchemaMarkup validates schema markup for an article
func (v *SEOValidator) ValidateSchemaMarkup(articleID uint64) (*SchemaValidationResult, error) {
	if v.articleRepo == nil {
		return nil, fmt.Errorf("article repository not configured")
	}
	
	ctx := context.Background()
	article, err := v.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	
	result := &SchemaValidationResult{
		URL:            fmt.Sprintf("%s/article/%s", v.baseURL, article.Slug),
		ValidationTime: time.Now(),
		Issues:         []SchemaIssue{},
		Recommendations: []string{},
		SchemaTypes:    []string{},
	}
	
	// Validate schema type
	v.validateSchemaType(result, article)
	
	// Validate required properties
	v.validateRequiredSchemaProperties(result, article)
	
	// Validate schema consistency
	v.validateSchemaConsistency(result, article)
	
	// Validate against Google guidelines
	v.validateGoogleStructuredDataGuidelines(result, article)
	
	// Generate recommendations
	v.generateSchemaRecommendations(result)
	
	// Overall validation status
	result.IsValid = len(result.Issues) == 0 || v.hasOnlyMinorSchemaIssues(result.Issues)
	
	return result, nil
}

// validateSchemaType validates the schema type is appropriate for the content
func (v *SEOValidator) validateSchemaType(result *SchemaValidationResult, article *models.Article) {
	schemaType := article.SEOData.SchemaType
	if schemaType == "" {
		schemaType = "NewsArticle" // Default
	}
	
	result.SchemaTypes = append(result.SchemaTypes, schemaType)
	
	// Validate schema type is appropriate
	validSchemaTypes := []string{"NewsArticle", "Article", "BlogPosting"}
	isValid := false
	for _, validType := range validSchemaTypes {
		if schemaType == validType {
			isValid = true
			break
		}
	}
	
	if !isValid {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "invalid_schema_type",
			Severity:    "high",
			Description: fmt.Sprintf("Schema type '%s' is not recommended for news content", schemaType),
			SchemaType:  schemaType,
			Suggestion:  "Use 'NewsArticle' for news content, 'Article' for general articles, or 'BlogPosting' for blog posts",
		})
	}
	
	// Validate schema type consistency with content
	if schemaType == "NewsArticle" {
		// News articles should be recent
		if article.PublishedAt != nil && time.Since(*article.PublishedAt) > 7*24*time.Hour {
			result.Issues = append(result.Issues, SchemaIssue{
				Type:        "stale_news_article",
				Severity:    "medium",
				Description: "NewsArticle schema used for content older than 7 days",
				SchemaType:  schemaType,
				Suggestion:  "Consider using 'Article' schema for older content",
			})
		}
	}
}

// validateRequiredSchemaProperties validates required schema properties
func (v *SEOValidator) validateRequiredSchemaProperties(result *SchemaValidationResult, article *models.Article) {
	// Required properties for all schema types
	if article.Title == "" {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_headline",
			Severity:    "critical",
			Description: "Article is missing headline (title)",
			Property:    "headline",
			Suggestion:  "Add a descriptive title to the article",
		})
	}
	
	if article.Excerpt == "" && len(article.Content) == 0 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_description",
			Severity:    "critical",
			Description: "Article is missing description (excerpt or content)",
			Property:    "description",
			Suggestion:  "Add an excerpt or ensure content is not empty",
		})
	}
	
	if article.PublishedAt == nil {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_date_published",
			Severity:    "critical",
			Description: "Article is missing publication date",
			Property:    "datePublished",
			Suggestion:  "Set publication date for the article",
		})
	}
	
	if article.AuthorID == 0 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_author",
			Severity:    "high",
			Description: "Article is missing author information",
			Property:    "author",
			Suggestion:  "Assign an author to the article",
		})
	}
	
	// Validate image for better rich snippets
	if len(article.Content) > 0 && !strings.Contains(article.Content, "<img") {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_image",
			Severity:    "medium",
			Description: "Article lacks images for rich snippets",
			Property:    "image",
			Suggestion:  "Add relevant images to improve search appearance",
		})
	}
}

// validateSchemaConsistency validates consistency across schema properties
func (v *SEOValidator) validateSchemaConsistency(result *SchemaValidationResult, article *models.Article) {
	// Validate title length for optimal display
	if len(article.Title) > 60 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "long_headline",
			Severity:    "medium",
			Description: fmt.Sprintf("Headline is %d characters, recommended maximum is 60", len(article.Title)),
			Property:    "headline",
			Suggestion:  "Shorten headline for better search result display",
		})
	}
	
	// Validate description length
	descLength := len(article.Excerpt)
	if article.Excerpt == "" {
		descLength = len(article.Content)
		if descLength > 160 {
			descLength = 160 // Truncated description length
		}
	}
	
	if descLength > 160 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "long_description",
			Severity:    "medium",
			Description: fmt.Sprintf("Description is %d characters, recommended maximum is 160", descLength),
			Property:    "description",
			Suggestion:  "Shorten description for better search result display",
		})
	}
	
	if descLength < 50 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "short_description",
			Severity:    "low",
			Description: fmt.Sprintf("Description is %d characters, recommended minimum is 50", descLength),
			Property:    "description",
			Suggestion:  "Expand description for better search context",
		})
	}
	
	// Validate content length
	if len(article.Content) < 300 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "thin_content",
			Severity:    "medium",
			Description: "Article content is very short, may be considered thin content",
			Property:    "articleBody",
			Suggestion:  "Expand content to provide more value to readers",
		})
	}
}

// validateGoogleStructuredDataGuidelines validates against Google's guidelines
func (v *SEOValidator) validateGoogleStructuredDataGuidelines(result *SchemaValidationResult, article *models.Article) {
	// Google requires specific properties for NewsArticle
	if article.SEOData.SchemaType == "NewsArticle" || article.SEOData.SchemaType == "" {
		// Validate dateModified is present and after datePublished
		if article.PublishedAt != nil && article.UpdatedAt.Before(*article.PublishedAt) {
			result.Issues = append(result.Issues, SchemaIssue{
				Type:        "invalid_date_modified",
				Severity:    "high",
				Description: "dateModified is before datePublished",
				Property:    "dateModified",
				Suggestion:  "Ensure dateModified is equal to or after datePublished",
			})
		}
		
		// Validate publisher information
		if v.siteName == "" {
			result.Issues = append(result.Issues, SchemaIssue{
				Type:        "missing_publisher",
				Severity:    "critical",
				Description: "Publisher information is missing",
				Property:    "publisher",
				Suggestion:  "Configure site name for publisher information",
			})
		}
		
		// Validate mainEntityOfPage
		if article.SEOData.CanonicalURL == "" {
			result.Issues = append(result.Issues, SchemaIssue{
				Type:        "missing_main_entity",
				Severity:    "medium",
				Description: "mainEntityOfPage should reference the article URL",
				Property:    "mainEntityOfPage",
				Suggestion:  "Set canonical URL or ensure article URL is properly configured",
			})
		}
	}
	
	// Validate keywords are relevant and not excessive
	totalKeywords := len(article.SEOData.Keywords)
	for _, tag := range article.Tags {
		totalKeywords += len(tag.Keywords)
	}
	
	if totalKeywords > 10 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "excessive_keywords",
			Severity:    "low",
			Description: fmt.Sprintf("Article has %d keywords, recommended maximum is 10", totalKeywords),
			Property:    "keywords",
			Suggestion:  "Focus on the most relevant keywords for better SEO",
		})
	}
	
	if totalKeywords == 0 {
		result.Issues = append(result.Issues, SchemaIssue{
			Type:        "missing_keywords",
			Severity:    "medium",
			Description: "Article has no keywords defined",
			Property:    "keywords",
			Suggestion:  "Add relevant keywords to improve discoverability",
		})
	}
}

// generateSchemaRecommendations generates optimization recommendations
func (v *SEOValidator) generateSchemaRecommendations(result *SchemaValidationResult) {
	if len(result.Issues) == 0 {
		result.Recommendations = append(result.Recommendations, "Schema markup is well-optimized")
	}
	
	// Count issues by severity
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}
	
	if criticalCount > 0 {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Fix %d critical schema issues immediately", criticalCount))
	}
	
	if highCount > 0 {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Address %d high-priority schema issues", highCount))
	}
	
	if mediumCount > 0 {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Consider fixing %d medium-priority schema issues for better optimization", mediumCount))
	}
	
	// General recommendations
	result.Recommendations = append(result.Recommendations, "Test schema markup with Google's Rich Results Test")
	result.Recommendations = append(result.Recommendations, "Monitor schema markup performance in Google Search Console")
	
	if len(result.SchemaTypes) > 0 && result.SchemaTypes[0] == "NewsArticle" {
		result.Recommendations = append(result.Recommendations, "Submit to Google News for enhanced visibility")
	}
}

// hasOnlyMinorSchemaIssues checks if result has only minor issues
func (v *SEOValidator) hasOnlyMinorSchemaIssues(issues []SchemaIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			return false
		}
	}
	return true
}

// ValidateSchemaMarkupForArticles validates schema markup for multiple articles
func (v *SEOValidator) ValidateSchemaMarkupForArticles(articleIDs []uint64) (map[uint64]*SchemaValidationResult, error) {
	results := make(map[uint64]*SchemaValidationResult)
	
	for _, articleID := range articleIDs {
		result, err := v.ValidateSchemaMarkup(articleID)
		if err != nil {
			continue
		}
		results[articleID] = result
	}
	
	return results, nil
}

// GetSchemaValidationSummary provides a summary of schema validation results
func (v *SEOValidator) GetSchemaValidationSummary(results map[uint64]*SchemaValidationResult) map[string]interface{} {
	summary := map[string]interface{}{
		"total_validated":     len(results),
		"valid_count":         0,
		"invalid_count":       0,
		"schema_type_breakdown": make(map[string]int),
		"issue_breakdown":     make(map[string]int),
		"severity_breakdown":  make(map[string]int),
	}
	
	for _, result := range results {
		if result.IsValid {
			summary["valid_count"] = summary["valid_count"].(int) + 1
		} else {
			summary["invalid_count"] = summary["invalid_count"].(int) + 1
		}
		
		// Count schema types
		schemaBreakdown := summary["schema_type_breakdown"].(map[string]int)
		for _, schemaType := range result.SchemaTypes {
			schemaBreakdown[schemaType]++
		}
		
		// Count issues by type and severity
		issueBreakdown := summary["issue_breakdown"].(map[string]int)
		severityBreakdown := summary["severity_breakdown"].(map[string]int)
		for _, issue := range result.Issues {
			issueBreakdown[issue.Type]++
			severityBreakdown[issue.Severity]++
		}
	}
	
	return summary
}

// ValidateCanonicalChain validates canonical URL chains for cycle detection and optimization
func (v *SEOValidator) ValidateCanonicalChain(startURL string) (*CanonicalValidationResult, error) {
	result := &CanonicalValidationResult{
		URL:            startURL,
		ValidationTime: time.Now(),
		Issues:         []CanonicalIssue{},
		Recommendations: []string{},
	}

	start := time.Now()
	
	// Build canonical chain
	chain, httpStatus, err := v.buildCanonicalChain(startURL)
	result.ResponseTime = time.Since(start)
	result.HTTPStatus = httpStatus
	
	if err != nil {
		result.IsValid = false
		result.Issues = append(result.Issues, CanonicalIssue{
			Type:        "chain_build_error",
			Severity:    "critical",
			Description: fmt.Sprintf("Failed to build canonical chain: %v", err),
			URL:         startURL,
			Suggestion:  "Check if the URL is accessible and returns valid HTML",
		})
		return result, nil
	}

	result.CanonicalChain = chain
	result.ChainLength = len(chain)
	
	if len(chain) > 0 {
		result.CanonicalURL = chain[len(chain)-1]
	}

	// Validate chain
	v.validateChainLength(result)
	v.detectCircularReferences(result)
	v.validateChainConsistency(result)
	v.generateCanonicalRecommendations(result)

	// Overall validation status
	result.IsValid = len(result.Issues) == 0 || v.hasOnlyMinorIssues(result.Issues)

	return result, nil
}

// buildCanonicalChain follows canonical URLs to build the complete chain
func (v *SEOValidator) buildCanonicalChain(startURL string) ([]string, int, error) {
	var chain []string
	visited := make(map[string]bool)
	currentURL := startURL
	var lastStatus int

	for len(chain) < 10 { // Prevent infinite loops
		if visited[currentURL] {
			// Circular reference detected
			return chain, lastStatus, nil
		}
		
		visited[currentURL] = true
		chain = append(chain, currentURL)

		// Get canonical URL from the page
		canonicalURL, status, err := v.extractCanonicalURL(currentURL)
		lastStatus = status
		
		if err != nil {
			return chain, status, err
		}

		// If no canonical URL found or it's the same as current, we're done
		if canonicalURL == "" || canonicalURL == currentURL {
			break
		}

		// Normalize URLs for comparison
		normalizedCanonical, err := v.normalizeURL(canonicalURL)
		if err != nil {
			return chain, status, fmt.Errorf("invalid canonical URL: %v", err)
		}

		normalizedCurrent, err := v.normalizeURL(currentURL)
		if err != nil {
			return chain, status, fmt.Errorf("invalid current URL: %v", err)
		}

		if normalizedCanonical == normalizedCurrent {
			break
		}

		currentURL = normalizedCanonical
	}

	return chain, lastStatus, nil
}

// extractCanonicalURL extracts canonical URL from HTML page
func (v *SEOValidator) extractCanonicalURL(pageURL string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SEO-Validator/1.0")
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// For now, we'll simulate canonical URL extraction
	// In a real implementation, you would parse the HTML and extract the canonical link
	// This is a simplified version for testing purposes
	
	// Check if this is an internal URL and try to get canonical from database
	if v.isInternalURL(pageURL) {
		return v.getCanonicalFromDatabase(pageURL)
	}

	// For external URLs, we would parse HTML here
	return "", resp.StatusCode, nil
}

// isInternalURL checks if URL belongs to our site
func (v *SEOValidator) isInternalURL(checkURL string) bool {
	parsedURL, err := url.Parse(checkURL)
	if err != nil {
		return false
	}
	
	baseURL, err := url.Parse(v.baseURL)
	if err != nil {
		return false
	}
	
	return parsedURL.Host == baseURL.Host
}

// getCanonicalFromDatabase retrieves canonical URL from database for internal URLs
func (v *SEOValidator) getCanonicalFromDatabase(pageURL string) (string, int, error) {
	// Extract slug from URL
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return "", 200, err
	}
	
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", 200, nil
	}
	
	switch pathParts[0] {
	case "article":
		if len(pathParts) >= 2 {
			ctx := context.Background()
			article, err := v.articleRepo.GetBySlug(ctx, pathParts[1])
			if err != nil {
				return "", 404, err
			}
			if article.SEOData.CanonicalURL != "" {
				return article.SEOData.CanonicalURL, 200, nil
			}
		}
	case "category":
		if len(pathParts) >= 2 {
			_, err := v.categoryRepo.GetBySlug(pathParts[1], "")
			if err != nil {
				return "", 404, err
			}
			// Categories don't have canonical URLs in the current model
			// This is a placeholder for future implementation
		}
	case "tag":
		if len(pathParts) >= 2 {
			_, err := v.tagRepo.GetBySlug(pathParts[1], "")
			if err != nil {
				return "", 404, err
			}
			// Tags don't have canonical URLs in the current model
			// This is a placeholder for future implementation
		}
	}
	
	return "", 200, nil
}

// normalizeURL normalizes URLs for consistent comparison
func (v *SEOValidator) normalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	
	// Remove fragment
	parsedURL.Fragment = ""
	
	// Normalize path
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}
	
	// Convert to lowercase for comparison
	parsedURL.Host = strings.ToLower(parsedURL.Host)
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	
	return parsedURL.String(), nil
}

// validateChainLength checks if canonical chain length is optimal
func (v *SEOValidator) validateChainLength(result *CanonicalValidationResult) {
	if result.ChainLength > 3 {
		result.Issues = append(result.Issues, CanonicalIssue{
			Type:        "long_canonical_chain",
			Severity:    "medium",
			Description: fmt.Sprintf("Canonical chain is too long (%d hops). Recommended maximum is 3.", result.ChainLength),
			Suggestion:  "Reduce canonical chain length by pointing directly to the final canonical URL",
		})
		result.Recommendations = append(result.Recommendations, "Optimize canonical chain to reduce length to 3 or fewer hops")
	}
	
	if result.ChainLength > 5 {
		result.Issues = append(result.Issues, CanonicalIssue{
			Type:        "excessive_canonical_chain",
			Severity:    "high",
			Description: fmt.Sprintf("Canonical chain is excessively long (%d hops). This may impact SEO performance.", result.ChainLength),
			Suggestion:  "Immediately reduce canonical chain length to prevent SEO issues",
		})
	}
}

// detectCircularReferences detects circular references in canonical chains
func (v *SEOValidator) detectCircularReferences(result *CanonicalValidationResult) {
	seen := make(map[string]int)
	
	for i, url := range result.CanonicalChain {
		if prevIndex, exists := seen[url]; exists {
			result.HasCircularRef = true
			result.CircularRefPoint = url
			result.Issues = append(result.Issues, CanonicalIssue{
				Type:        "circular_canonical_reference",
				Severity:    "critical",
				Description: fmt.Sprintf("Circular canonical reference detected at URL: %s (appears at positions %d and %d)", url, prevIndex, i),
				URL:         url,
				Suggestion:  "Remove circular reference by setting a proper canonical URL",
			})
			result.Recommendations = append(result.Recommendations, "Fix circular canonical reference immediately to prevent indexing issues")
			return
		}
		seen[url] = i
	}
}

// validateChainConsistency validates consistency of canonical chain
func (v *SEOValidator) validateChainConsistency(result *CanonicalValidationResult) {
	if len(result.CanonicalChain) < 2 {
		return
	}
	
	// Check for mixed protocols (HTTP/HTTPS)
	protocols := make(map[string]bool)
	for _, urlStr := range result.CanonicalChain {
		if parsedURL, err := url.Parse(urlStr); err == nil {
			protocols[parsedURL.Scheme] = true
		}
	}
	
	if len(protocols) > 1 {
		result.Issues = append(result.Issues, CanonicalIssue{
			Type:        "mixed_protocol_canonical_chain",
			Severity:    "medium",
			Description: "Canonical chain contains mixed protocols (HTTP and HTTPS)",
			Suggestion:  "Ensure all URLs in canonical chain use the same protocol (preferably HTTPS)",
		})
		result.Recommendations = append(result.Recommendations, "Standardize all canonical URLs to use HTTPS")
	}
	
	// Check for mixed domains
	domains := make(map[string]bool)
	for _, urlStr := range result.CanonicalChain {
		if parsedURL, err := url.Parse(urlStr); err == nil {
			domains[parsedURL.Host] = true
		}
	}
	
	if len(domains) > 1 {
		result.Issues = append(result.Issues, CanonicalIssue{
			Type:        "cross_domain_canonical_chain",
			Severity:    "high",
			Description: "Canonical chain spans multiple domains",
			Suggestion:  "Ensure canonical URLs point to the same domain to maintain link equity",
		})
		result.Recommendations = append(result.Recommendations, "Keep all canonical URLs within the same domain")
	}
}

// generateCanonicalRecommendations generates optimization recommendations
func (v *SEOValidator) generateCanonicalRecommendations(result *CanonicalValidationResult) {
	if result.ChainLength == 1 {
		result.Recommendations = append(result.Recommendations, "Canonical URL setup is optimal (self-referencing)")
	}
	
	if result.ChainLength == 2 {
		result.Recommendations = append(result.Recommendations, "Canonical chain length is good (1 hop)")
	}
	
	if result.HTTPStatus != 200 {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Ensure canonical URL returns HTTP 200 (currently returns %d)", result.HTTPStatus))
	}
	
	if result.ResponseTime > 2*time.Second {
		result.Recommendations = append(result.Recommendations, "Improve page load time for better canonical URL processing")
	}
}

// hasOnlyMinorIssues checks if result has only minor issues
func (v *SEOValidator) hasOnlyMinorIssues(issues []CanonicalIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			return false
		}
	}
	return true
}

// ValidateCanonicalURLsForArticles validates canonical URLs for multiple articles
func (v *SEOValidator) ValidateCanonicalURLsForArticles(articleIDs []uint64) (map[uint64]*CanonicalValidationResult, error) {
	results := make(map[uint64]*CanonicalValidationResult)
	
	for _, articleID := range articleIDs {
		ctx := context.Background()
		article, err := v.articleRepo.GetByID(ctx, articleID)
		if err != nil {
			continue
		}
		
		articleURL := fmt.Sprintf("%s/article/%s", v.baseURL, article.Slug)
		result, err := v.ValidateCanonicalChain(articleURL)
		if err != nil {
			continue
		}
		
		results[articleID] = result
	}
	
	return results, nil
}

// GetCanonicalValidationSummary provides a summary of canonical validation results
func (v *SEOValidator) GetCanonicalValidationSummary(results map[uint64]*CanonicalValidationResult) map[string]interface{} {
	summary := map[string]interface{}{
		"total_validated":    len(results),
		"valid_count":        0,
		"invalid_count":      0,
		"circular_ref_count": 0,
		"long_chain_count":   0,
		"avg_chain_length":   0.0,
		"avg_response_time":  0.0,
		"issue_breakdown":    make(map[string]int),
	}
	
	var totalChainLength int
	var totalResponseTime time.Duration
	
	for _, result := range results {
		if result.IsValid {
			summary["valid_count"] = summary["valid_count"].(int) + 1
		} else {
			summary["invalid_count"] = summary["invalid_count"].(int) + 1
		}
		
		if result.HasCircularRef {
			summary["circular_ref_count"] = summary["circular_ref_count"].(int) + 1
		}
		
		if result.ChainLength > 3 {
			summary["long_chain_count"] = summary["long_chain_count"].(int) + 1
		}
		
		totalChainLength += result.ChainLength
		totalResponseTime += result.ResponseTime
		
		// Count issues by type
		issueBreakdown := summary["issue_breakdown"].(map[string]int)
		for _, issue := range result.Issues {
			issueBreakdown[issue.Type]++
		}
	}
	
	if len(results) > 0 {
		summary["avg_chain_length"] = float64(totalChainLength) / float64(len(results))
		summary["avg_response_time"] = totalResponseTime.Seconds() / float64(len(results))
	}
	
	return summary
}

// GoogleNewsValidationResult represents the result of Google News compliance validation
type GoogleNewsValidationResult struct {
	URL                string                    `json:"url"`
	IsCompliant        bool                      `json:"is_compliant"`
	Issues             []GoogleNewsIssue         `json:"issues"`
	ValidationTime     time.Time                 `json:"validation_time"`
	Recommendations    []string                  `json:"recommendations"`
	RSSFeedValid       bool                      `json:"rss_feed_valid"`
	SitemapValid       bool                      `json:"sitemap_valid"`
	MetadataValid      bool                      `json:"metadata_valid"`
	PublicationInfo    SEOGoogleNewsPublication    `json:"publication_info"`
}

// GoogleNewsIssue represents a specific Google News compliance issue
type GoogleNewsIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Component   string `json:"component"` // "rss", "sitemap", "metadata"
	Suggestion  string `json:"suggestion,omitempty"`
}

// SEOGoogleNewsPublication represents publication information for Google News validation
type SEOGoogleNewsPublication struct {
	Name         string    `json:"name"`
	Language     string    `json:"language"`
	LastUpdated  time.Time `json:"last_updated"`
	ArticleCount int       `json:"article_count"`
}

// ValidateGoogleNewsCompliance validates Google News compliance for the site
func (v *SEOValidator) ValidateGoogleNewsCompliance(languageCode string) (*GoogleNewsValidationResult, error) {
	result := &GoogleNewsValidationResult{
		URL:            v.baseURL,
		ValidationTime: time.Now(),
		Issues:         []GoogleNewsIssue{},
		Recommendations: []string{},
		PublicationInfo: SEOGoogleNewsPublication{
			Name:        v.siteName,
			Language:    languageCode,
			LastUpdated: time.Now(),
		},
	}

	// Validate RSS feed
	v.validateGoogleNewsRSSFeed(result, languageCode)
	
	// Validate sitemap
	v.validateGoogleNewsSitemap(result, languageCode)
	
	// Validate metadata requirements
	v.validateGoogleNewsMetadata(result, languageCode)
	
	// Generate recommendations
	v.generateGoogleNewsRecommendations(result)
	
	// Overall compliance status
	result.IsCompliant = len(result.Issues) == 0 || v.hasOnlyMinorGoogleNewsIssues(result.Issues)
	
	return result, nil
}

// validateGoogleNewsRSSFeed validates Google News RSS feed compliance
func (v *SEOValidator) validateGoogleNewsRSSFeed(result *GoogleNewsValidationResult, languageCode string) {
	// Check if RSS feed exists and is accessible
	rssURL := fmt.Sprintf("%s/rss/googlenews-%s.xml", v.baseURL, languageCode)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", rssURL, nil)
	if err != nil {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "rss_feed_unreachable",
			Severity:    "critical",
			Description: "Google News RSS feed is not accessible",
			Component:   "rss",
			Suggestion:  "Ensure Google News RSS feed is properly configured and accessible",
		})
		return
	}
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "rss_feed_error",
			Severity:    "critical",
			Description: fmt.Sprintf("Failed to fetch Google News RSS feed: %v", err),
			Component:   "rss",
			Suggestion:  "Check network connectivity and RSS feed URL",
		})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "rss_feed_http_error",
			Severity:    "critical",
			Description: fmt.Sprintf("Google News RSS feed returned HTTP %d", resp.StatusCode),
			Component:   "rss",
			Suggestion:  "Fix RSS feed endpoint to return HTTP 200",
		})
		return
	}
	
	// Validate content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "xml") {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "rss_feed_content_type",
			Severity:    "medium",
			Description: fmt.Sprintf("RSS feed content type is '%s', should be 'application/xml' or 'text/xml'", contentType),
			Component:   "rss",
			Suggestion:  "Set proper Content-Type header for RSS feed",
		})
	}
	
	// TODO: Parse and validate RSS content structure
	// For now, we'll mark as valid if accessible
	result.RSSFeedValid = true
	
	// Check feed freshness (should be updated within last 24 hours)
	lastModified := resp.Header.Get("Last-Modified")
	if lastModified != "" {
		if lastMod, err := time.Parse(time.RFC1123, lastModified); err == nil {
			if time.Since(lastMod) > 24*time.Hour {
				result.Issues = append(result.Issues, GoogleNewsIssue{
					Type:        "rss_feed_stale",
					Severity:    "medium",
					Description: "RSS feed hasn't been updated in over 24 hours",
					Component:   "rss",
					Suggestion:  "Ensure RSS feed is updated regularly with fresh content",
				})
			}
		}
	}
}

// validateGoogleNewsSitemap validates Google News sitemap compliance
func (v *SEOValidator) validateGoogleNewsSitemap(result *GoogleNewsValidationResult, languageCode string) {
	// Check if sitemap exists and is accessible
	sitemapURL := fmt.Sprintf("%s/sitemap/googlenews-%s-0.xml", v.baseURL, languageCode)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "sitemap_unreachable",
			Severity:    "critical",
			Description: "Google News sitemap is not accessible",
			Component:   "sitemap",
			Suggestion:  "Ensure Google News sitemap is properly configured and accessible",
		})
		return
	}
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "sitemap_error",
			Severity:    "critical",
			Description: fmt.Sprintf("Failed to fetch Google News sitemap: %v", err),
			Component:   "sitemap",
			Suggestion:  "Check network connectivity and sitemap URL",
		})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "sitemap_http_error",
			Severity:    "critical",
			Description: fmt.Sprintf("Google News sitemap returned HTTP %d", resp.StatusCode),
			Component:   "sitemap",
			Suggestion:  "Fix sitemap endpoint to return HTTP 200",
		})
		return
	}
	
	// Validate content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "xml") {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "sitemap_content_type",
			Severity:    "medium",
			Description: fmt.Sprintf("Sitemap content type is '%s', should be 'application/xml' or 'text/xml'", contentType),
			Component:   "sitemap",
			Suggestion:  "Set proper Content-Type header for sitemap",
		})
	}
	
	// TODO: Parse and validate sitemap XML structure
	// For now, we'll mark as valid if accessible
	result.SitemapValid = true
	
	// Check sitemap index
	indexURL := fmt.Sprintf("%s/sitemap/googlenews-index-%s.xml", v.baseURL, languageCode)
	indexReq, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err == nil {
		indexResp, err := v.httpClient.Do(indexReq)
		if err != nil || indexResp.StatusCode != http.StatusOK {
			result.Issues = append(result.Issues, GoogleNewsIssue{
				Type:        "sitemap_index_missing",
				Severity:    "medium",
				Description: "Google News sitemap index is not accessible",
				Component:   "sitemap",
				Suggestion:  "Create sitemap index for better organization",
			})
		}
		if indexResp != nil {
			indexResp.Body.Close()
		}
	}
}

// validateGoogleNewsMetadata validates Google News metadata requirements
func (v *SEOValidator) validateGoogleNewsMetadata(result *GoogleNewsValidationResult, languageCode string) {
	result.MetadataValid = true
	
	// Check publication name
	if v.siteName == "" {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "missing_publication_name",
			Severity:    "critical",
			Description: "Publication name is not configured",
			Component:   "metadata",
			Suggestion:  "Configure publication name for Google News",
		})
		result.MetadataValid = false
	}
	
	// Check language code format
	if !v.isValidLanguageCode(languageCode) {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "invalid_language_code",
			Severity:    "high",
			Description: fmt.Sprintf("Language code '%s' is not valid for Google News", languageCode),
			Component:   "metadata",
			Suggestion:  "Use valid ISO 639-1 language codes (e.g., 'en', 'fa', 'ar')",
		})
		result.MetadataValid = false
	}
	
	// Check if articles have required metadata
	if v.articleRepo != nil {
		// Get recent articles to check metadata
		cutoffTime := time.Now().Add(-48 * time.Hour)
		articles, err := v.articleRepo.GetPublishedArticlesAfterTimeWithOffset(cutoffTime, languageCode, 10, 0)
		if err == nil {
			result.PublicationInfo.ArticleCount = len(articles)
			
			for _, article := range articles {
				// Check required fields
				if article.Title == "" {
					result.Issues = append(result.Issues, GoogleNewsIssue{
						Type:        "missing_article_title",
						Severity:    "critical",
						Description: fmt.Sprintf("Article ID %d is missing title", article.ID),
						Component:   "metadata",
						Suggestion:  "Ensure all articles have titles",
					})
				}
				
				if article.PublishedAt == nil {
					result.Issues = append(result.Issues, GoogleNewsIssue{
						Type:        "missing_publication_date",
						Severity:    "critical",
						Description: fmt.Sprintf("Article ID %d is missing publication date", article.ID),
						Component:   "metadata",
						Suggestion:  "Ensure all articles have publication dates",
					})
				}
				
				// Check if article is too old for Google News (older than 2 days)
				if article.PublishedAt != nil && time.Since(*article.PublishedAt) > 48*time.Hour {
					result.Issues = append(result.Issues, GoogleNewsIssue{
						Type:        "article_too_old",
						Severity:    "low",
						Description: fmt.Sprintf("Article ID %d is older than 48 hours", article.ID),
						Component:   "metadata",
						Suggestion:  "Google News typically includes articles from the last 48 hours",
					})
				}
			}
		}
	}
}

// generateGoogleNewsRecommendations generates optimization recommendations
func (v *SEOValidator) generateGoogleNewsRecommendations(result *GoogleNewsValidationResult) {
	if result.RSSFeedValid && result.SitemapValid && result.MetadataValid {
		result.Recommendations = append(result.Recommendations, "Google News setup is compliant")
	}
	
	if !result.RSSFeedValid {
		result.Recommendations = append(result.Recommendations, "Fix RSS feed issues to improve Google News compliance")
	}
	
	if !result.SitemapValid {
		result.Recommendations = append(result.Recommendations, "Fix sitemap issues to improve Google News discoverability")
	}
	
	if !result.MetadataValid {
		result.Recommendations = append(result.Recommendations, "Fix metadata issues to meet Google News requirements")
	}
	
	// General recommendations
	result.Recommendations = append(result.Recommendations, "Submit sitemap to Google Search Console")
	result.Recommendations = append(result.Recommendations, "Monitor Google News performance in Search Console")
	result.Recommendations = append(result.Recommendations, "Ensure articles are published regularly for better visibility")
	
	if result.PublicationInfo.ArticleCount < 5 {
		result.Recommendations = append(result.Recommendations, "Publish more articles to improve Google News presence")
	}
}

// hasOnlyMinorGoogleNewsIssues checks if result has only minor issues
func (v *SEOValidator) hasOnlyMinorGoogleNewsIssues(issues []GoogleNewsIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			return false
		}
	}
	return true
}

// isValidLanguageCode validates language code format
func (v *SEOValidator) isValidLanguageCode(code string) bool {
	// Basic validation for common language codes
	validCodes := map[string]bool{
		"en": true, "fa": true, "ar": true, "es": true, "fr": true,
		"de": true, "it": true, "pt": true, "ru": true, "zh": true,
		"ja": true, "ko": true, "hi": true, "tr": true, "nl": true,
	}
	
	return validCodes[code] || len(code) == 2
}

// ValidateGoogleNewsArticle validates a specific article for Google News compliance
func (v *SEOValidator) ValidateGoogleNewsArticle(articleID uint64) (*GoogleNewsValidationResult, error) {
	if v.articleRepo == nil {
		return nil, fmt.Errorf("article repository not configured")
	}
	
	ctx := context.Background()
	article, err := v.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	
	result := &GoogleNewsValidationResult{
		URL:            fmt.Sprintf("%s/article/%s", v.baseURL, article.Slug),
		ValidationTime: time.Now(),
		Issues:         []GoogleNewsIssue{},
		Recommendations: []string{},
		PublicationInfo: SEOGoogleNewsPublication{
			Name:         v.siteName,
			Language:     article.LanguageCode,
			LastUpdated:  article.UpdatedAt,
			ArticleCount: 1,
		},
	}
	
	// Validate article metadata
	if article.Title == "" {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "missing_title",
			Severity:    "critical",
			Description: "Article is missing title",
			Component:   "metadata",
			Suggestion:  "Add a descriptive title to the article",
		})
	}
	
	if article.PublishedAt == nil {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "missing_publication_date",
			Severity:    "critical",
			Description: "Article is missing publication date",
			Component:   "metadata",
			Suggestion:  "Set publication date for the article",
		})
	}
	
	if article.AuthorID == 0 {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "missing_author",
			Severity:    "medium",
			Description: "Article is missing author information",
			Component:   "metadata",
			Suggestion:  "Assign an author to the article",
		})
	}
	
	if len(article.Content) < 100 {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "insufficient_content",
			Severity:    "medium",
			Description: "Article content is too short for news article",
			Component:   "metadata",
			Suggestion:  "Ensure articles have substantial content (at least 100 characters)",
		})
	}
	
	// Check if article is recent enough for Google News
	if article.PublishedAt != nil && time.Since(*article.PublishedAt) > 48*time.Hour {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "article_too_old",
			Severity:    "low",
			Description: "Article is older than 48 hours",
			Component:   "metadata",
			Suggestion:  "Google News typically includes articles from the last 48 hours",
		})
	}
	
	// Validate schema markup
	if article.SEOData.SchemaType == "" {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "missing_schema_type",
			Severity:    "medium",
			Description: "Article is missing schema markup type",
			Component:   "metadata",
			Suggestion:  "Set schema type to 'NewsArticle' for news content",
		})
	} else if article.SEOData.SchemaType != "NewsArticle" {
		result.Issues = append(result.Issues, GoogleNewsIssue{
			Type:        "incorrect_schema_type",
			Severity:    "low",
			Description: fmt.Sprintf("Article schema type is '%s', recommended 'NewsArticle' for news", article.SEOData.SchemaType),
			Component:   "metadata",
			Suggestion:  "Use 'NewsArticle' schema type for news content",
		})
	}
	
	result.IsCompliant = len(result.Issues) == 0 || v.hasOnlyMinorGoogleNewsIssues(result.Issues)
	result.MetadataValid = result.IsCompliant
	
	return result, nil
}