package testing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
)

// DataAnonymizer handles anonymization of production data for testing
type DataAnonymizer struct {
	rules       []AnonymizationRule
	patterns    map[string]*regexp.Regexp
	replacements map[string][]string
	mutex       sync.RWMutex
}

// AnonymizationRule defines how to anonymize specific data types
type AnonymizationRule struct {
	Name        string                 `json:"name"`
	Pattern     string                 `json:"pattern"`
	Type        AnonymizationType      `json:"type"`
	Replacement string                 `json:"replacement"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AnonymizationType defines the type of anonymization to apply
type AnonymizationType string

const (
	AnonymizeReplace    AnonymizationType = "replace"
	AnonymizeMask       AnonymizationType = "mask"
	AnonymizeHash       AnonymizationType = "hash"
	AnonymizeRandomize  AnonymizationType = "randomize"
	AnonymizeRedact     AnonymizationType = "redact"
)

// NewDataAnonymizer creates a new data anonymizer
func NewDataAnonymizer() *DataAnonymizer {
	anonymizer := &DataAnonymizer{
		rules:        make([]AnonymizationRule, 0),
		patterns:     make(map[string]*regexp.Regexp),
		replacements: make(map[string][]string),
	}

	anonymizer.initializeDefaultRules()
	return anonymizer
}

// initializeDefaultRules sets up default anonymization rules
func (da *DataAnonymizer) initializeDefaultRules() {
	defaultRules := []AnonymizationRule{
		{
			Name:        "email_anonymization",
			Pattern:     `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Type:        AnonymizeReplace,
			Replacement: "[email]",
			Enabled:     true,
		},
		{
			Name:        "phone_anonymization",
			Pattern:     `\+?[\d\s\-\(\)]{10,}`,
			Type:        AnonymizeReplace,
			Replacement: "[phone_number]",
			Enabled:     true,
		},
		{
			Name:        "name_anonymization",
			Pattern:     `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
			Type:        AnonymizeReplace,
			Replacement: "[name]",
			Enabled:     true,
		},
		{
			Name:        "address_anonymization",
			Pattern:     `\d+\s+[A-Za-z\s]+(?:Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd)`,
			Type:        AnonymizeReplace,
			Replacement: "[address]",
			Enabled:     true,
		},
		{
			Name:        "credit_card_anonymization",
			Pattern:     `\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`,
			Type:        AnonymizeMask,
			Replacement: "****-****-****-****",
			Enabled:     true,
		},
		{
			Name:        "ssn_anonymization",
			Pattern:     `\b\d{3}-\d{2}-\d{4}\b`,
			Type:        AnonymizeMask,
			Replacement: "***-**-****",
			Enabled:     true,
		},
	}

	for _, rule := range defaultRules {
		da.AddRule(rule)
	}

	// Initialize replacement data
	da.replacements["names"] = []string{
		"John Doe", "Jane Smith", "Bob Johnson", "Alice Brown", "Charlie Wilson",
		"Diana Davis", "Eve Miller", "Frank Garcia", "Grace Martinez", "Henry Lopez",
	}

	da.replacements["emails"] = []string{
		"user1@example.com", "user2@example.com", "user3@example.com",
		"test1@demo.org", "test2@demo.org", "sample@test.net",
	}

	da.replacements["phones"] = []string{
		"(555) 123-4567", "(555) 234-5678", "(555) 345-6789",
		"(555) 456-7890", "(555) 567-8901", "(555) 678-9012",
	}
}

// AddRule adds a new anonymization rule
func (da *DataAnonymizer) AddRule(rule AnonymizationRule) error {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	// Compile regex pattern
	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern for rule %s: %w", rule.Name, err)
	}

	da.rules = append(da.rules, rule)
	da.patterns[rule.Name] = pattern

	log.Printf("Added anonymization rule: %s", rule.Name)
	return nil
}

// AnonymizeText anonymizes text content based on configured rules
func (da *DataAnonymizer) AnonymizeText(text string) string {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	result := text

	for _, rule := range da.rules {
		if !rule.Enabled {
			continue
		}

		pattern := da.patterns[rule.Name]
		if pattern == nil {
			continue
		}

		switch rule.Type {
		case AnonymizeReplace:
			result = pattern.ReplaceAllString(result, rule.Replacement)
		case AnonymizeMask:
			result = da.maskMatches(result, pattern, rule.Replacement)
		case AnonymizeHash:
			result = da.hashMatches(result, pattern)
		case AnonymizeRandomize:
			result = da.randomizeMatches(result, pattern, rule.Name)
		case AnonymizeRedact:
			result = pattern.ReplaceAllString(result, "[REDACTED]")
		}
	}

	return result
}

// AnonymizeData anonymizes structured data
func (da *DataAnonymizer) AnonymizeData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = da.AnonymizeText(v)
		case map[string]interface{}:
			result[key] = da.AnonymizeData(v)
		case []interface{}:
			result[key] = da.anonymizeSlice(v)
		default:
			result[key] = value
		}
	}

	return result
}

// anonymizeSlice anonymizes slice data
func (da *DataAnonymizer) anonymizeSlice(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))

	for i, item := range slice {
		switch v := item.(type) {
		case string:
			result[i] = da.AnonymizeText(v)
		case map[string]interface{}:
			result[i] = da.AnonymizeData(v)
		case []interface{}:
			result[i] = da.anonymizeSlice(v)
		default:
			result[i] = item
		}
	}

	return result
}

// maskMatches masks matched patterns
func (da *DataAnonymizer) maskMatches(text string, pattern *regexp.Regexp, mask string) string {
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		if mask != "" {
			return mask
		}
		// Default masking with asterisks
		return strings.Repeat("*", len(match))
	})
}

// hashMatches hashes matched patterns
func (da *DataAnonymizer) hashMatches(text string, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		hash := da.generateHash(match)
		return fmt.Sprintf("[HASH:%s]", hash[:8]) // Use first 8 characters of hash
	})
}

// randomizeMatches randomizes matched patterns with realistic replacements
func (da *DataAnonymizer) randomizeMatches(text string, pattern *regexp.Regexp, ruleName string) string {
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		return da.getRandomReplacement(ruleName, match)
	})
}

// generateHash generates a hash for a given string
func (da *DataAnonymizer) generateHash(input string) string {
	// Simple hash implementation (in production, use proper cryptographic hash)
	hash := 0
	for _, char := range input {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}

// getRandomReplacement gets a random replacement for a matched pattern
func (da *DataAnonymizer) getRandomReplacement(ruleName, original string) string {
	var replacements []string

	// Determine replacement type based on rule name or pattern
	switch {
	case strings.Contains(ruleName, "email"):
		replacements = da.replacements["emails"]
	case strings.Contains(ruleName, "name"):
		replacements = da.replacements["names"]
	case strings.Contains(ruleName, "phone"):
		replacements = da.replacements["phones"]
	default:
		return "[ANONYMIZED]"
	}

	if len(replacements) == 0 {
		return "[ANONYMIZED]"
	}

	// Use hash of original to ensure consistent replacement
	hash := 0
	for _, char := range original {
		hash = hash*31 + int(char)
	}
	index := hash % len(replacements)
	if index < 0 {
		index = -index
	}

	return replacements[index]
}

// ValidateAnonymization validates that anonymization was successful
func (da *DataAnonymizer) ValidateAnonymization(original, anonymized string) ValidationResult {
	result := ValidationResult{
		IsValid:     true,
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	da.mutex.RLock()
	defer da.mutex.RUnlock()

	// Check if any sensitive patterns still exist
	for _, rule := range da.rules {
		if !rule.Enabled {
			continue
		}

		pattern := da.patterns[rule.Name]
		if pattern == nil {
			continue
		}

		matches := pattern.FindAllString(anonymized, -1)
		if len(matches) > 0 {
			result.IsValid = false
			result.Issues = append(result.Issues, 
				fmt.Sprintf("Rule '%s' still has %d matches in anonymized text", rule.Name, len(matches)))
			result.Suggestions = append(result.Suggestions, 
				fmt.Sprintf("Review and strengthen the '%s' anonymization rule", rule.Name))
		}
	}

	// Check anonymization coverage
	coverage := da.calculateAnonymizationCoverage(original, anonymized)
	if coverage < 0.5 { // Less than 50% changed
		result.Suggestions = append(result.Suggestions, 
			"Low anonymization coverage detected, consider adding more rules")
	}

	return result
}

// ValidationResult represents the result of anonymization validation
type ValidationResult struct {
	IsValid     bool     `json:"is_valid"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Coverage    float64  `json:"coverage"`
}

// calculateAnonymizationCoverage calculates how much of the text was anonymized
func (da *DataAnonymizer) calculateAnonymizationCoverage(original, anonymized string) float64 {
	if len(original) == 0 {
		return 1.0
	}

	changes := 0
	minLen := len(original)
	if len(anonymized) < minLen {
		minLen = len(anonymized)
	}

	for i := 0; i < minLen; i++ {
		if original[i] != anonymized[i] {
			changes++
		}
	}

	// Account for length differences
	changes += abs(len(original) - len(anonymized))

	return float64(changes) / float64(len(original))
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetRules returns all anonymization rules
func (da *DataAnonymizer) GetRules() []AnonymizationRule {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	rules := make([]AnonymizationRule, len(da.rules))
	copy(rules, da.rules)
	return rules
}

// EnableRule enables or disables a specific rule
func (da *DataAnonymizer) EnableRule(ruleName string, enabled bool) error {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	for i, rule := range da.rules {
		if rule.Name == ruleName {
			da.rules[i].Enabled = enabled
			log.Printf("Rule '%s' enabled: %t", ruleName, enabled)
			return nil
		}
	}

	return fmt.Errorf("rule '%s' not found", ruleName)
}

// RemoveRule removes a specific rule
func (da *DataAnonymizer) RemoveRule(ruleName string) error {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	for i, rule := range da.rules {
		if rule.Name == ruleName {
			// Remove rule from slice
			da.rules = append(da.rules[:i], da.rules[i+1:]...)
			delete(da.patterns, ruleName)
			log.Printf("Removed anonymization rule: %s", ruleName)
			return nil
		}
	}

	return fmt.Errorf("rule '%s' not found", ruleName)
}

// GenerateAnonymizationReport generates a report of anonymization activities
func (da *DataAnonymizer) GenerateAnonymizationReport(original, anonymized string) AnonymizationReport {
	report := AnonymizationReport{
		OriginalLength:   len(original),
		AnonymizedLength: len(anonymized),
		RulesApplied:     make([]string, 0),
		Coverage:         da.calculateAnonymizationCoverage(original, anonymized),
		Timestamp:        time.Now(),
	}

	da.mutex.RLock()
	defer da.mutex.RUnlock()

	// Check which rules were applied
	for _, rule := range da.rules {
		if !rule.Enabled {
			continue
		}

		pattern := da.patterns[rule.Name]
		if pattern == nil {
			continue
		}

		originalMatches := pattern.FindAllString(original, -1)
		anonymizedMatches := pattern.FindAllString(anonymized, -1)

		if len(originalMatches) > len(anonymizedMatches) {
			report.RulesApplied = append(report.RulesApplied, rule.Name)
		}
	}

	return report
}

// AnonymizationReport represents a report of anonymization activities
type AnonymizationReport struct {
	OriginalLength   int       `json:"original_length"`
	AnonymizedLength int       `json:"anonymized_length"`
	RulesApplied     []string  `json:"rules_applied"`
	Coverage         float64   `json:"coverage"`
	Timestamp        time.Time `json:"timestamp"`
}

// GenerateRandomID generates a random ID for anonymization
func (da *DataAnonymizer) GenerateRandomID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}