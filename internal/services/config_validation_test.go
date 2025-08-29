package services

import (
	"testing"

	"high-performance-news-website/internal/models"
)

func TestConfigService_ValidateStringValue(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		value       string
		rules       *models.ConfigValidationRules
		expectError bool
	}{
		{
			name:  "Valid string within length limits",
			value: "Hello World",
			rules: &models.ConfigValidationRules{
				MinLength: func() *int { v := 5; return &v }(),
				MaxLength: func() *int { v := 20; return &v }(),
			},
			expectError: false,
		},
		{
			name:  "String too short",
			value: "Hi",
			rules: &models.ConfigValidationRules{
				MinLength: func() *int { v := 5; return &v }(),
			},
			expectError: true,
		},
		{
			name:  "String too long",
			value: "This is a very long string that exceeds the maximum length",
			rules: &models.ConfigValidationRules{
				MaxLength: func() *int { v := 20; return &v }(),
			},
			expectError: true,
		},
		{
			name:  "Valid string matching pattern",
			value: "test@example.com",
			rules: &models.ConfigValidationRules{
				Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			},
			expectError: false,
		},
		{
			name:  "String not matching pattern",
			value: "invalid-email",
			rules: &models.ConfigValidationRules{
				Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			},
			expectError: true,
		},
		{
			name:  "Valid option",
			value: "option1",
			rules: &models.ConfigValidationRules{
				Options: []string{"option1", "option2", "option3"},
			},
			expectError: false,
		},
		{
			name:  "Invalid option",
			value: "invalid_option",
			rules: &models.ConfigValidationRules{
				Options: []string{"option1", "option2", "option3"},
			},
			expectError: true,
		},
		{
			name:  "Required field with value",
			value: "some value",
			rules: &models.ConfigValidationRules{
				Required: true,
			},
			expectError: false,
		},
		{
			name:  "Required field without value",
			value: "",
			rules: &models.ConfigValidationRules{
				Required: true,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateStringValue(tt.value, tt.rules)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigService_ValidateIntValue(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		value       string
		rules       *models.ConfigValidationRules
		expectError bool
	}{
		{
			name:  "Valid integer within range",
			value: "50",
			rules: &models.ConfigValidationRules{
				MinValue: func() *float64 { v := 10.0; return &v }(),
				MaxValue: func() *float64 { v := 100.0; return &v }(),
			},
			expectError: false,
		},
		{
			name:  "Integer below minimum",
			value: "5",
			rules: &models.ConfigValidationRules{
				MinValue: func() *float64 { v := 10.0; return &v }(),
			},
			expectError: true,
		},
		{
			name:  "Integer above maximum",
			value: "150",
			rules: &models.ConfigValidationRules{
				MaxValue: func() *float64 { v := 100.0; return &v }(),
			},
			expectError: true,
		},
		{
			name:        "Invalid integer format",
			value:       "not_a_number",
			rules:       &models.ConfigValidationRules{},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateIntValue(tt.value, tt.rules)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigService_ValidateFloatValue(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		value       string
		rules       *models.ConfigValidationRules
		expectError bool
	}{
		{
			name:  "Valid float within range",
			value: "50.5",
			rules: &models.ConfigValidationRules{
				MinValue: func() *float64 { v := 10.0; return &v }(),
				MaxValue: func() *float64 { v := 100.0; return &v }(),
			},
			expectError: false,
		},
		{
			name:  "Float below minimum",
			value: "5.5",
			rules: &models.ConfigValidationRules{
				MinValue: func() *float64 { v := 10.0; return &v }(),
			},
			expectError: true,
		},
		{
			name:  "Float above maximum",
			value: "150.5",
			rules: &models.ConfigValidationRules{
				MaxValue: func() *float64 { v := 100.0; return &v }(),
			},
			expectError: true,
		},
		{
			name:        "Invalid float format",
			value:       "not_a_number",
			rules:       &models.ConfigValidationRules{},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateFloatValue(tt.value, tt.rules)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigService_ValidateBoolValue(t *testing.T) {
	service := setupConfigServiceTest()
	
	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{
			name:        "Valid boolean true",
			value:       "true",
			expectError: false,
		},
		{
			name:        "Valid boolean false",
			value:       "false",
			expectError: false,
		},
		{
			name:        "Valid boolean 1",
			value:       "1",
			expectError: false,
		},
		{
			name:        "Valid boolean 0",
			value:       "0",
			expectError: false,
		},
		{
			name:        "Invalid boolean",
			value:       "maybe",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateBoolValue(tt.value, &models.ConfigValidationRules{})
			
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigService_ComplexValidation(t *testing.T) {
	service := setupConfigServiceTest()
	
	// Test a configuration with multiple validation rules
	config := &models.Configuration{
		Key:         "complex_config",
		Value:       "test@example.com",
		Type:        models.ConfigTypeString,
		Category:    "test",
		Description: "Complex validation test",
		Validation: &models.ConfigValidationRules{
			Required:  true,
			MinLength: func() *int { v := 5; return &v }(),
			MaxLength: func() *int { v := 50; return &v }(),
			Pattern:   `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
	}
	
	// Valid value should pass
	err := service.validateConfigValue(config)
	if err != nil {
		t.Errorf("Expected valid configuration to pass validation, got error: %v", err)
	}
	
	// Test with invalid email pattern
	config.Value = "invalid-email"
	err = service.validateConfigValue(config)
	if err == nil {
		t.Error("Expected validation error for invalid email pattern")
	}
	
	// Test with empty required field
	config.Value = ""
	err = service.validateConfigValue(config)
	if err == nil {
		t.Error("Expected validation error for empty required field")
	}
	
	// Test with value too short
	config.Value = "a@b"
	err = service.validateConfigValue(config)
	if err == nil {
		t.Error("Expected validation error for value too short")
	}
}