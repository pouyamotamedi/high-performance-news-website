package testing

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AITestScenarioCreator creates comprehensive test scenarios using AI
type AITestScenarioCreator struct {
	llmClient       LLMClient
	codeAnalyzer    *CodeAnalyzer
	requirementParser *RequirementParser
	scenarioValidator *ScenarioValidator
	config          *ScenarioConfig
}

// ScenarioConfig configuration for scenario creation
type ScenarioConfig struct {
	MaxScenariosPerRequirement int           `json:"max_scenarios_per_requirement"`
	ScenarioComplexity         string        `json:"scenario_complexity"`
	IncludeNegativeTests       bool          `json:"include_negative_tests"`
	IncludePerformanceTests    bool          `json:"include_performance_tests"`
	IncludeSecurityTests       bool          `json:"include_security_tests"`
	TestTimeout                time.Duration `json:"test_timeout"`
}

// RequirementParser parses requirements to extract testable scenarios
type RequirementParser struct {
	patterns []RequirementPattern
}

// RequirementPattern defines patterns for extracting test scenarios from requirements
type RequirementPattern struct {
	Type        string `json:"type"`
	Pattern     string `json:"pattern"`
	Priority    string `json:"priority"`
	TestType    string `json:"test_type"`
}

// ScenarioValidator validates generated scenarios for quality and completeness
type ScenarioValidator struct {
	qualityRules []QualityRule
}

// QualityRule defines rules for scenario quality validation
type QualityRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Validator   func(TestScenario) bool `json:"-"`
	Weight      float64 `json:"weight"`
}

// AITestSuite represents a complete test suite with multiple scenarios
type AITestSuite struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Scenarios   []TestScenario `json:"scenarios"`
	Coverage    CoverageReport `json:"coverage"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CoverageReport represents test coverage analysis
type CoverageReport struct {
	RequirementsCovered   int     `json:"requirements_covered"`
	TotalRequirements     int     `json:"total_requirements"`
	CoveragePercentage    float64 `json:"coverage_percentage"`
	EdgeCasesCovered      int     `json:"edge_cases_covered"`
	SecurityTestsCovered  int     `json:"security_tests_covered"`
	PerformanceTestsCovered int   `json:"performance_tests_covered"`
}

// ScenarioCreationRequest represents a request to create test scenarios
type ScenarioCreationRequest struct {
	Requirements    []string          `json:"requirements"`
	CodeContext     string            `json:"code_context"`
	TestTypes       []TestScenarioType `json:"test_types"`
	Priority        Priority          `json:"priority"`
	MaxScenarios    int               `json:"max_scenarios"`
	IncludeEdgeCases bool             `json:"include_edge_cases"`
}

// NewAITestScenarioCreator creates a new AI test scenario creator
func NewAITestScenarioCreator(llmClient LLMClient, config *ScenarioConfig) *AITestScenarioCreator {
	return &AITestScenarioCreator{
		llmClient:         llmClient,
		codeAnalyzer:      &CodeAnalyzer{},
		requirementParser: NewRequirementParser(),
		scenarioValidator: NewScenarioValidator(),
		config:           config,
	}
}

// CreateTestSuite creates a comprehensive test suite from requirements
func (c *AITestScenarioCreator) CreateTestSuite(ctx context.Context, request ScenarioCreationRequest) (*AITestSuite, error) {
	suite := &AITestSuite{
		ID:          generateTestSuiteID(),
		Name:        "AI-Generated Test Suite",
		Description: "Comprehensive test suite generated from requirements using AI",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	var allScenarios []TestScenario
	
	// Process each requirement
	for _, requirement := range request.Requirements {
		scenarios, err := c.createScenariosFromRequirement(ctx, requirement, request)
		if err != nil {
			return nil, fmt.Errorf("failed to create scenarios for requirement: %w", err)
		}
		
		allScenarios = append(allScenarios, scenarios...)
	}
	
	// Validate and filter scenarios
	validScenarios := c.scenarioValidator.ValidateScenarios(allScenarios)
	
	// Prioritize and limit scenarios
	finalScenarios := c.prioritizeAndLimitScenarios(validScenarios, request.MaxScenarios)
	
	suite.Scenarios = finalScenarios
	suite.Coverage = c.calculateCoverage(finalScenarios, request.Requirements)
	
	return suite, nil
}

// createScenariosFromRequirement creates test scenarios from a single requirement
func (c *AITestScenarioCreator) createScenariosFromRequirement(ctx context.Context, requirement string, request ScenarioCreationRequest) ([]TestScenario, error) {
	// Parse requirement to identify testable elements
	testableElements := c.requirementParser.ParseRequirement(requirement)
	
	var scenarios []TestScenario
	
	// Generate positive test scenarios
	positiveScenarios, err := c.generatePositiveScenarios(ctx, requirement, testableElements, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate positive scenarios: %w", err)
	}
	scenarios = append(scenarios, positiveScenarios...)
	
	// Generate negative test scenarios
	if c.config.IncludeNegativeTests {
		negativeScenarios, err := c.generateNegativeScenarios(ctx, requirement, testableElements, request)
		if err != nil {
			return nil, fmt.Errorf("failed to generate negative scenarios: %w", err)
		}
		scenarios = append(scenarios, negativeScenarios...)
	}
	
	// Generate edge case scenarios
	if request.IncludeEdgeCases {
		edgeCaseScenarios, err := c.generateEdgeCaseScenarios(ctx, requirement, testableElements, request)
		if err != nil {
			return nil, fmt.Errorf("failed to generate edge case scenarios: %w", err)
		}
		scenarios = append(scenarios, edgeCaseScenarios...)
	}
	
	// Generate performance scenarios
	if c.config.IncludePerformanceTests {
		performanceScenarios, err := c.generatePerformanceScenarios(ctx, requirement, testableElements, request)
		if err != nil {
			return nil, fmt.Errorf("failed to generate performance scenarios: %w", err)
		}
		scenarios = append(scenarios, performanceScenarios...)
	}
	
	// Generate security scenarios
	if c.config.IncludeSecurityTests {
		securityScenarios, err := c.generateSecurityScenarios(ctx, requirement, testableElements, request)
		if err != nil {
			return nil, fmt.Errorf("failed to generate security scenarios: %w", err)
		}
		scenarios = append(scenarios, securityScenarios...)
	}
	
	return scenarios, nil
}

// generatePositiveScenarios generates positive test scenarios
func (c *AITestScenarioCreator) generatePositiveScenarios(ctx context.Context, requirement string, elements []TestableElement, request ScenarioCreationRequest) ([]TestScenario, error) {
	prompt := c.buildPositiveScenarioPrompt(requirement, elements, request)
	
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate positive scenarios: %w", err)
	}
	
	scenarios, err := c.parseScenarioResponse(response, EdgeCaseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to parse positive scenarios: %w", err)
	}
	
	return scenarios, nil
}

// generateNegativeScenarios generates negative test scenarios
func (c *AITestScenarioCreator) generateNegativeScenarios(ctx context.Context, requirement string, elements []TestableElement, request ScenarioCreationRequest) ([]TestScenario, error) {
	prompt := c.buildNegativeScenarioPrompt(requirement, elements, request)
	
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate negative scenarios: %w", err)
	}
	
	scenarios, err := c.parseScenarioResponse(response, EdgeCaseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to parse negative scenarios: %w", err)
	}
	
	return scenarios, nil
}

// generateEdgeCaseScenarios generates edge case scenarios
func (c *AITestScenarioCreator) generateEdgeCaseScenarios(ctx context.Context, requirement string, elements []TestableElement, request ScenarioCreationRequest) ([]TestScenario, error) {
	prompt := c.buildEdgeCaseScenarioPrompt(requirement, elements, request)
	
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate edge case scenarios: %w", err)
	}
	
	scenarios, err := c.parseScenarioResponse(response, EdgeCaseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to parse edge case scenarios: %w", err)
	}
	
	return scenarios, nil
}

// generatePerformanceScenarios generates performance test scenarios
func (c *AITestScenarioCreator) generatePerformanceScenarios(ctx context.Context, requirement string, elements []TestableElement, request ScenarioCreationRequest) ([]TestScenario, error) {
	prompt := c.buildPerformanceScenarioPrompt(requirement, elements, request)
	
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate performance scenarios: %w", err)
	}
	
	scenarios, err := c.parseScenarioResponse(response, PerformanceScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to parse performance scenarios: %w", err)
	}
	
	return scenarios, nil
}

// generateSecurityScenarios generates security test scenarios
func (c *AITestScenarioCreator) generateSecurityScenarios(ctx context.Context, requirement string, elements []TestableElement, request ScenarioCreationRequest) ([]TestScenario, error) {
	prompt := c.buildSecurityScenarioPrompt(requirement, elements, request)
	
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate security scenarios: %w", err)
	}
	
	scenarios, err := c.parseScenarioResponse(response, SecurityScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to parse security scenarios: %w", err)
	}
	
	return scenarios, nil
}

// buildPositiveScenarioPrompt builds prompt for positive scenario generation
func (c *AITestScenarioCreator) buildPositiveScenarioPrompt(requirement string, elements []TestableElement, request ScenarioCreationRequest) string {
	return fmt.Sprintf(`
Generate positive test scenarios for this requirement:

Requirement: %s

Testable Elements: %v

Code Context: %s

Create test scenarios that verify the requirement works correctly under normal conditions:

1. Happy path scenarios with valid inputs
2. Typical user workflows
3. Standard data processing scenarios
4. Normal load conditions
5. Expected user interactions

For each scenario, provide:
- Clear test name and description
- Step-by-step test procedure
- Expected results
- Test data requirements
- Success criteria

Focus on scenarios that demonstrate the requirement is properly implemented.

Format as JSON array of test scenarios.
`, requirement, elements, request.CodeContext)
}

// buildNegativeScenarioPrompt builds prompt for negative scenario generation
func (c *AITestScenarioCreator) buildNegativeScenarioPrompt(requirement string, elements []TestableElement, request ScenarioCreationRequest) string {
	return fmt.Sprintf(`
Generate negative test scenarios for this requirement:

Requirement: %s

Testable Elements: %v

Create test scenarios that verify proper error handling and validation:

1. Invalid input scenarios
2. Missing required data
3. Malformed requests
4. Unauthorized access attempts
5. Resource unavailability
6. System constraint violations
7. Data corruption scenarios
8. Network failure conditions

For each scenario, provide:
- Clear test name and description
- Invalid input or condition
- Expected error response
- Error handling verification
- Recovery procedures

Focus on scenarios that test system robustness and error handling.

Format as JSON array of test scenarios.
`, requirement, elements)
}

// buildEdgeCaseScenarioPrompt builds prompt for edge case scenario generation
func (c *AITestScenarioCreator) buildEdgeCaseScenarioPrompt(requirement string, elements []TestableElement, request ScenarioCreationRequest) string {
	return fmt.Sprintf(`
Generate edge case test scenarios for this requirement:

Requirement: %s

Testable Elements: %v

Create test scenarios for boundary conditions and unusual situations:

1. Boundary value testing (min/max values)
2. Empty or null data scenarios
3. Extremely large datasets
4. Concurrent access scenarios
5. Resource exhaustion conditions
6. Timing-dependent scenarios
7. Multilingual edge cases (Persian/Arabic RTL text)
8. Special character handling
9. Unicode edge cases
10. Memory pressure scenarios

For each scenario, provide:
- Specific boundary condition being tested
- Exact input values or conditions
- Expected system behavior
- Performance implications
- Risk assessment

Focus on scenarios that could cause unexpected system behavior.

Format as JSON array of test scenarios.
`, requirement, elements)
}

// buildPerformanceScenarioPrompt builds prompt for performance scenario generation
func (c *AITestScenarioCreator) buildPerformanceScenarioPrompt(requirement string, elements []TestableElement, request ScenarioCreationRequest) string {
	return fmt.Sprintf(`
Generate performance test scenarios for this requirement:

Requirement: %s

Testable Elements: %v

Create performance test scenarios for a news website handling 50,000 articles/day:

1. Load testing scenarios (normal traffic patterns)
2. Stress testing scenarios (peak traffic)
3. Spike testing scenarios (sudden traffic increases)
4. Volume testing scenarios (large data sets)
5. Endurance testing scenarios (sustained load)
6. Scalability testing scenarios
7. Database performance scenarios
8. Cache performance scenarios
9. Static file generation performance
10. Multilingual content processing performance

For each scenario, provide:
- Performance test objective
- Load pattern and user simulation
- Performance thresholds and SLAs
- Resource monitoring requirements
- Success/failure criteria
- Bottleneck identification methods

Focus on realistic performance scenarios for a high-traffic news website.

Format as JSON array of performance test scenarios.
`, requirement, elements)
}

// buildSecurityScenarioPrompt builds prompt for security scenario generation
func (c *AITestScenarioCreator) buildSecurityScenarioPrompt(requirement string, elements []TestableElement, request ScenarioCreationRequest) string {
	return fmt.Sprintf(`
Generate security test scenarios for this requirement:

Requirement: %s

Testable Elements: %v

Create security test scenarios covering OWASP Top 10 and news website specific threats:

1. Authentication bypass attempts
2. Authorization escalation scenarios
3. SQL injection attack scenarios
4. XSS attack scenarios
5. CSRF attack scenarios
6. File upload security scenarios
7. Input validation bypass attempts
8. Session management attacks
9. API security testing
10. Content injection attacks
11. SEO spam injection attempts
12. Comment system abuse scenarios

For each scenario, provide:
- Security threat being tested
- Attack vector and payload
- Expected security response
- Detection mechanisms
- Impact assessment
- Mitigation verification

Focus on realistic security threats for a news publishing platform.

Format as JSON array of security test scenarios.
`, requirement, elements)
}

// parseScenarioResponse parses LLM response into test scenarios
func (c *AITestScenarioCreator) parseScenarioResponse(response string, scenarioType TestScenarioType) ([]TestScenario, error) {
	// For demonstration, return sample scenarios based on scenario type
	// In real implementation, would parse JSON response
	
	var scenarios []TestScenario
	
	switch scenarioType {
	case EdgeCaseScenario:
		scenarios = []TestScenario{
			{
				ID:          generateScenarioID("edge-case"),
				Name:        "Empty Input Edge Case",
				Description: "Test behavior with empty input values",
				Type:        EdgeCaseScenario,
				Priority:    PriorityHigh,
				EdgeCases: []EdgeCase{
					{
						Name:        "Empty String Input",
						Input:       "",
						Description: "Test with empty string input",
						Risk:        "medium",
					},
				},
				TestData: map[string]interface{}{
					"input":          "",
					"expected_error": "validation_error",
				},
				Expected: ExpectedResult{
					Status:   "error",
					Response: nil,
					Errors:   []string{"Input cannot be empty"},
				},
				Confidence:  0.9,
				GeneratedAt: time.Now(),
			},
		}
	case SecurityScenario:
		scenarios = []TestScenario{
			{
				ID:          generateScenarioID("security"),
				Name:        "SQL Injection Security Test",
				Description: "Test API endpoint against SQL injection attacks",
				Type:        SecurityScenario,
				Priority:    PriorityCritical,
				TestData: map[string]interface{}{
					"malicious_payload": "'; DROP TABLE articles; --",
					"endpoint":          "/api/articles",
				},
				Expected: ExpectedResult{
					Status:   "error",
					Response: "Invalid input detected",
				},
				Confidence:  0.95,
				GeneratedAt: time.Now(),
			},
		}
	case PerformanceScenario:
		scenarios = []TestScenario{
			{
				ID:          generateScenarioID("performance"),
				Name:        "High Load Performance Test",
				Description: "Test system under high concurrent load",
				Type:        PerformanceScenario,
				Priority:    PriorityHigh,
				TestData: map[string]interface{}{
					"concurrent_users": 1000,
					"duration":         "5m",
					"ramp_up":          "1m",
				},
				Expected: ExpectedResult{
					Status: "success",
					Performance: PerformanceExpectation{
						MaxDuration: 1 * time.Second,
						MaxMemory:   1073741824, // 1GB
						MaxCPU:      80.0,
					},
				},
				Confidence:  0.85,
				GeneratedAt: time.Now(),
			},
		}
	default:
		scenarios = []TestScenario{
			{
				ID:          generateScenarioID("default"),
				Name:        "Sample Test Scenario",
				Description: "This is a sample scenario generated by AI",
				Type:        scenarioType,
				Priority:    PriorityMedium,
				Confidence:  0.8,
				GeneratedAt: time.Now(),
			},
		}
	}
	
	return scenarios, nil
}



// TestableElement represents an element that can be tested
type TestableElement struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
	Constraints []string    `json:"constraints"`
}

// NewRequirementParser creates a new requirement parser
func NewRequirementParser() *RequirementParser {
	return &RequirementParser{
		patterns: []RequirementPattern{
			{
				Type:     "functional",
				Pattern:  "WHEN.*THEN.*SHALL",
				Priority: "high",
				TestType: "functional",
			},
			{
				Type:     "security",
				Pattern:  "authentication|authorization|security",
				Priority: "critical",
				TestType: "security",
			},
			{
				Type:     "performance",
				Pattern:  "performance|response time|throughput",
				Priority: "high",
				TestType: "performance",
			},
		},
	}
}

// ParseRequirement parses a requirement to extract testable elements
func (p *RequirementParser) ParseRequirement(requirement string) []TestableElement {
	var elements []TestableElement
	
	// Simple parsing logic - in real implementation would be more sophisticated
	if strings.Contains(strings.ToLower(requirement), "api") {
		elements = append(elements, TestableElement{
			Type:        "api",
			Name:        "API Endpoint",
			Description: "API functionality described in requirement",
		})
	}
	
	if strings.Contains(strings.ToLower(requirement), "database") {
		elements = append(elements, TestableElement{
			Type:        "database",
			Name:        "Database Operation",
			Description: "Database functionality described in requirement",
		})
	}
	
	if strings.Contains(strings.ToLower(requirement), "user") {
		elements = append(elements, TestableElement{
			Type:        "user_interface",
			Name:        "User Interface",
			Description: "User interface functionality described in requirement",
		})
	}
	
	return elements
}

// NewScenarioValidator creates a new scenario validator
func NewScenarioValidator() *ScenarioValidator {
	return &ScenarioValidator{
		qualityRules: []QualityRule{
			{
				Name:        "Has Description",
				Description: "Scenario must have a clear description",
				Weight:      0.2,
				Validator: func(s TestScenario) bool {
					return s.Description != ""
				},
			},
			{
				Name:        "Has Test Data",
				Description: "Scenario must have test data defined",
				Weight:      0.3,
				Validator: func(s TestScenario) bool {
					return len(s.TestData) > 0
				},
			},
			{
				Name:        "Has Expected Result",
				Description: "Scenario must have expected results defined",
				Weight:      0.3,
				Validator: func(s TestScenario) bool {
					return s.Expected.Status != ""
				},
			},
			{
				Name:        "Sufficient Confidence",
				Description: "Scenario must have sufficient confidence score",
				Weight:      0.2,
				Validator: func(s TestScenario) bool {
					return s.Confidence >= 0.6
				},
			},
		},
	}
}

// ValidateScenarios validates generated scenarios for quality
func (v *ScenarioValidator) ValidateScenarios(scenarios []TestScenario) []TestScenario {
	var validScenarios []TestScenario
	
	for _, scenario := range scenarios {
		if v.isValidScenario(scenario) {
			validScenarios = append(validScenarios, scenario)
		}
	}
	
	return validScenarios
}

// isValidScenario checks if a scenario meets quality criteria
func (v *ScenarioValidator) isValidScenario(scenario TestScenario) bool {
	totalScore := 0.0
	totalWeight := 0.0
	
	for _, rule := range v.qualityRules {
		if rule.Validator(scenario) {
			totalScore += rule.Weight
		}
		totalWeight += rule.Weight
	}
	
	qualityScore := totalScore / totalWeight
	return qualityScore >= 0.7 // 70% quality threshold
}

// prioritizeAndLimitScenarios prioritizes scenarios and limits to max count
func (c *AITestScenarioCreator) prioritizeAndLimitScenarios(scenarios []TestScenario, maxCount int) []TestScenario {
	// Sort by priority and confidence
	// For now, just return first maxCount scenarios
	if len(scenarios) <= maxCount {
		return scenarios
	}
	
	return scenarios[:maxCount]
}

// calculateCoverage calculates test coverage for the generated scenarios
func (c *AITestScenarioCreator) calculateCoverage(scenarios []TestScenario, requirements []string) CoverageReport {
	report := CoverageReport{
		TotalRequirements: len(requirements),
	}
	
	// Count different types of scenarios
	for _, scenario := range scenarios {
		switch scenario.Type {
		case SecurityScenario:
			report.SecurityTestsCovered++
		case PerformanceScenario:
			report.PerformanceTestsCovered++
		case EdgeCaseScenario:
			report.EdgeCasesCovered++
		}
	}
	
	// Assume each scenario covers at least one requirement
	report.RequirementsCovered = len(scenarios)
	if report.RequirementsCovered > report.TotalRequirements {
		report.RequirementsCovered = report.TotalRequirements
	}
	
	if report.TotalRequirements > 0 {
		report.CoveragePercentage = float64(report.RequirementsCovered) / float64(report.TotalRequirements) * 100
	}
	
	return report
}

// generateTestSuiteID generates a unique ID for test suite
func generateTestSuiteID() string {
	return fmt.Sprintf("ai-suite-%d", time.Now().Unix())
}