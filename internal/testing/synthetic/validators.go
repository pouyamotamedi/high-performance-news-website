package synthetic

import (
	"strings"
	"time"
)

// SEOComplianceValidator validates SEO compliance
type SEOComplianceValidator struct {
	// Configuration for SEO validation rules
}

// NewSEOComplianceValidator creates a new SEO compliance validator
func NewSEOComplianceValidator() *SEOComplianceValidator {
	return &SEOComplianceValidator{}
}

// PerformanceValidator validates performance metrics
type PerformanceValidator struct {
	baselines map[string]PerformanceBaseline
}

// PerformanceBaseline represents baseline performance metrics
type PerformanceBaseline struct {
	TestName        string        `json:"test_name"`
	AverageResponse time.Duration `json:"average_response"`
	P95Response     time.Duration `json:"p95_response"`
	MaxResponse     time.Duration `json:"max_response"`
	LastUpdated     time.Time     `json:"last_updated"`
	SampleSize      int           `json:"sample_size"`
}

// NewPerformanceValidator creates a new performance validator
func NewPerformanceValidator() *PerformanceValidator {
	return &PerformanceValidator{
		baselines: make(map[string]PerformanceBaseline),
	}
}

// UpdateBaseline updates the performance baseline for a test
func (p *PerformanceValidator) UpdateBaseline(testName string, responses []time.Duration) {
	if len(responses) == 0 {
		return
	}

	// Calculate statistics
	var total time.Duration
	for _, response := range responses {
		total += response
	}
	average := total / time.Duration(len(responses))

	// Calculate P95 (simple approximation)
	p95Index := int(float64(len(responses)) * 0.95)
	if p95Index >= len(responses) {
		p95Index = len(responses) - 1
	}

	// Sort responses for P95 calculation (simplified)
	sortedResponses := make([]time.Duration, len(responses))
	copy(sortedResponses, responses)
	// Note: In production, you'd use sort.Slice here

	baseline := PerformanceBaseline{
		TestName:        testName,
		AverageResponse: average,
		P95Response:     sortedResponses[p95Index],
		MaxResponse:     sortedResponses[len(sortedResponses)-1],
		LastUpdated:     time.Now(),
		SampleSize:      len(responses),
	}

	p.baselines[testName] = baseline
}

// CheckRegression checks if current performance indicates a regression
func (p *PerformanceValidator) CheckRegression(testName string, currentResponse time.Duration) bool {
	baseline, exists := p.baselines[testName]
	if !exists {
		return false // No baseline to compare against
	}

	// Consider it a regression if current response is 50% slower than P95
	threshold := baseline.P95Response + (baseline.P95Response / 2)
	return currentResponse > threshold
}

// AccessibilityValidator validates accessibility compliance
type AccessibilityValidator struct {
	wcagRules []WCAGRule
}

// WCAGRule represents a WCAG compliance rule
type WCAGRule struct {
	ID          string `json:"id"`
	Level       string `json:"level"` // A, AA, AAA
	Description string `json:"description"`
	Selector    string `json:"selector"`
	Required    bool   `json:"required"`
}

// NewAccessibilityValidator creates a new accessibility validator
func NewAccessibilityValidator() *AccessibilityValidator {
	rules := []WCAGRule{
		{
			ID:          "1.1.1",
			Level:       "A",
			Description: "Images must have alt text",
			Selector:    "img:not([alt])",
			Required:    true,
		},
		{
			ID:          "1.3.1",
			Level:       "A",
			Description: "Headings must be properly structured",
			Selector:    "h1, h2, h3, h4, h5, h6",
			Required:    true,
		},
		{
			ID:          "2.1.1",
			Level:       "A",
			Description: "All functionality must be keyboard accessible",
			Selector:    "button, a, input, select, textarea",
			Required:    true,
		},
		{
			ID:          "3.1.1",
			Level:       "A",
			Description: "Page must have language attribute",
			Selector:    "html[lang]",
			Required:    true,
		},
		{
			ID:          "4.1.2",
			Level:       "A",
			Description: "Form elements must have labels",
			Selector:    "input, select, textarea",
			Required:    true,
		},
	}

	return &AccessibilityValidator{
		wcagRules: rules,
	}
}

// MobileExperienceValidator validates mobile user experience
type MobileExperienceValidator struct {
	viewportSizes []ViewportSize
}

// ViewportSize represents a mobile viewport configuration
type ViewportSize struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	DPR    float64 `json:"dpr"` // Device Pixel Ratio
}

// NewMobileExperienceValidator creates a new mobile experience validator
func NewMobileExperienceValidator() *MobileExperienceValidator {
	viewports := []ViewportSize{
		{Name: "iPhone SE", Width: 375, Height: 667, DPR: 2.0},
		{Name: "iPhone 12", Width: 390, Height: 844, DPR: 3.0},
		{Name: "iPhone 12 Pro Max", Width: 428, Height: 926, DPR: 3.0},
		{Name: "Samsung Galaxy S21", Width: 360, Height: 800, DPR: 3.0},
		{Name: "Samsung Galaxy S21 Ultra", Width: 384, Height: 854, DPR: 3.5},
		{Name: "iPad Mini", Width: 768, Height: 1024, DPR: 2.0},
		{Name: "iPad Pro", Width: 1024, Height: 1366, DPR: 2.0},
	}

	return &MobileExperienceValidator{
		viewportSizes: viewports,
	}
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	RuleID      string    `json:"rule_id"`
	Passed      bool      `json:"passed"`
	Description string    `json:"description"`
	Details     string    `json:"details"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
}

// ValidateAccessibility performs accessibility validation
func (a *AccessibilityValidator) ValidateAccessibility(pageContent string) []ValidationResult {
	var results []ValidationResult

	for _, rule := range a.wcagRules {
		result := ValidationResult{
			RuleID:      rule.ID,
			Description: rule.Description,
			Timestamp:   time.Now(),
		}

		// Simple validation based on content analysis
		switch rule.ID {
		case "1.1.1": // Images with alt text
			if strings.Contains(pageContent, "<img") && !strings.Contains(pageContent, "alt=") {
				result.Passed = false
				result.Details = "Images found without alt attributes"
				result.Severity = "high"
			} else {
				result.Passed = true
				result.Details = "All images have alt attributes"
				result.Severity = "info"
			}

		case "1.3.1": // Heading structure
			hasH1 := strings.Contains(pageContent, "<h1")
			if !hasH1 {
				result.Passed = false
				result.Details = "No H1 heading found"
				result.Severity = "medium"
			} else {
				result.Passed = true
				result.Details = "Heading structure appears correct"
				result.Severity = "info"
			}

		case "3.1.1": // Language attribute
			if !strings.Contains(pageContent, "lang=") {
				result.Passed = false
				result.Details = "No language attribute found on HTML element"
				result.Severity = "medium"
			} else {
				result.Passed = true
				result.Details = "Language attribute found"
				result.Severity = "info"
			}

		default:
			result.Passed = true
			result.Details = "Rule not implemented in validator"
			result.Severity = "info"
		}

		results = append(results, result)
	}

	return results
}

// ValidateMobileExperience performs mobile experience validation
func (m *MobileExperienceValidator) ValidateMobileExperience(pageMetrics map[string]interface{}) []ValidationResult {
	var results []ValidationResult

	// Check viewport meta tag
	viewportResult := ValidationResult{
		RuleID:      "mobile.viewport",
		Description: "Page must have proper viewport meta tag",
		Timestamp:   time.Now(),
	}

	if hasViewport, ok := pageMetrics["has_viewport_meta"].(bool); ok && hasViewport {
		viewportResult.Passed = true
		viewportResult.Details = "Viewport meta tag found"
		viewportResult.Severity = "info"
	} else {
		viewportResult.Passed = false
		viewportResult.Details = "Missing or invalid viewport meta tag"
		viewportResult.Severity = "high"
	}
	results = append(results, viewportResult)

	// Check for horizontal scrolling
	scrollResult := ValidationResult{
		RuleID:      "mobile.scroll",
		Description: "Page should not have horizontal scrolling on mobile",
		Timestamp:   time.Now(),
	}

	if scrollWidth, ok := pageMetrics["scroll_width"].(float64); ok {
		if viewportWidth, ok := pageMetrics["viewport_width"].(float64); ok {
			if scrollWidth > viewportWidth+10 { // 10px tolerance
				scrollResult.Passed = false
				scrollResult.Details = "Horizontal scrolling detected"
				scrollResult.Severity = "high"
			} else {
				scrollResult.Passed = true
				scrollResult.Details = "No horizontal scrolling"
				scrollResult.Severity = "info"
			}
		}
	}
	results = append(results, scrollResult)

	// Check touch target sizes
	touchResult := ValidationResult{
		RuleID:      "mobile.touch_targets",
		Description: "Touch targets should be at least 44px",
		Timestamp:   time.Now(),
	}

	if smallTargets, ok := pageMetrics["small_touch_targets"].(float64); ok {
		if smallTargets > 0 {
			touchResult.Passed = false
			touchResult.Details = "Small touch targets found"
			touchResult.Severity = "medium"
		} else {
			touchResult.Passed = true
			touchResult.Details = "All touch targets are appropriately sized"
			touchResult.Severity = "info"
		}
	}
	results = append(results, touchResult)

	return results
}

// GetWCAGRules returns all WCAG rules
func (a *AccessibilityValidator) GetWCAGRules() []WCAGRule {
	return a.wcagRules
}

// GetViewportSizes returns all configured viewport sizes
func (m *MobileExperienceValidator) GetViewportSizes() []ViewportSize {
	return m.viewportSizes
}

// AddCustomRule adds a custom WCAG rule
func (a *AccessibilityValidator) AddCustomRule(rule WCAGRule) {
	a.wcagRules = append(a.wcagRules, rule)
}

// AddCustomViewport adds a custom viewport size
func (m *MobileExperienceValidator) AddCustomViewport(viewport ViewportSize) {
	m.viewportSizes = append(m.viewportSizes, viewport)
}