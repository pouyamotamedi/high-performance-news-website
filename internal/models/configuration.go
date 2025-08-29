package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Configuration represents a system configuration setting
type Configuration struct {
	ID          uint64                 `json:"id" db:"id"`
	Key         string                 `json:"key" db:"key"`
	Value       string                 `json:"value" db:"value"`
	Type        ConfigurationType      `json:"type" db:"type"`
	Category    string                 `json:"category" db:"category"`
	Description string                 `json:"description" db:"description"`
	IsSecret    bool                   `json:"is_secret" db:"is_secret"`
	Validation  *ConfigValidationRules `json:"validation,omitempty" db:"validation"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ConfigurationType represents the type of configuration value
type ConfigurationType string

const (
	ConfigTypeString  ConfigurationType = "string"
	ConfigTypeInt     ConfigurationType = "int"
	ConfigTypeFloat   ConfigurationType = "float"
	ConfigTypeBool    ConfigurationType = "bool"
	ConfigTypeJSON    ConfigurationType = "json"
	ConfigTypeArray   ConfigurationType = "array"
)

// ConfigValidationRules defines validation rules for configuration values
type ConfigValidationRules struct {
	Required bool        `json:"required"`
	MinValue *float64    `json:"min_value,omitempty"`
	MaxValue *float64    `json:"max_value,omitempty"`
	MinLength *int       `json:"min_length,omitempty"`
	MaxLength *int       `json:"max_length,omitempty"`
	Pattern   string     `json:"pattern,omitempty"`
	Options   []string   `json:"options,omitempty"`
}

// FeatureFlag represents a feature flag for gradual rollouts
type FeatureFlag struct {
	ID          uint64                 `json:"id" db:"id"`
	Key         string                 `json:"key" db:"key"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Rollout     *FeatureFlagRollout    `json:"rollout,omitempty" db:"rollout"`
	Conditions  []FeatureFlagCondition `json:"conditions,omitempty" db:"conditions"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// FeatureFlagRollout defines rollout strategy for feature flags
type FeatureFlagRollout struct {
	Percentage int      `json:"percentage"` // 0-100
	UserGroups []string `json:"user_groups,omitempty"`
	IPRanges   []string `json:"ip_ranges,omitempty"`
}

// FeatureFlagCondition defines conditions for feature flag activation
type FeatureFlagCondition struct {
	Type     string      `json:"type"`     // user_role, user_id, ip_address, etc.
	Operator string      `json:"operator"` // equals, contains, in, etc.
	Value    interface{} `json:"value"`
}

// ConfigurationHistory tracks configuration changes for rollback
type ConfigurationHistory struct {
	ID           uint64    `json:"id" db:"id"`
	ConfigKey    string    `json:"config_key" db:"config_key"`
	OldValue     string    `json:"old_value" db:"old_value"`
	NewValue     string    `json:"new_value" db:"new_value"`
	ChangedBy    uint64    `json:"changed_by" db:"changed_by"`
	ChangeReason string    `json:"change_reason" db:"change_reason"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ConfigurationSnapshot represents a complete configuration snapshot
type ConfigurationSnapshot struct {
	ID          uint64                    `json:"id" db:"id"`
	Name        string                    `json:"name" db:"name"`
	Description string                    `json:"description" db:"description"`
	Config      map[string]Configuration  `json:"config"`
	CreatedBy   uint64                    `json:"created_by" db:"created_by"`
	CreatedAt   time.Time                 `json:"created_at" db:"created_at"`
}

// GetTypedValue returns the configuration value in its proper type
func (c *Configuration) GetTypedValue() (interface{}, error) {
	switch c.Type {
	case ConfigTypeString:
		return c.Value, nil
	case ConfigTypeInt:
		var val int
		if err := json.Unmarshal([]byte(c.Value), &val); err != nil {
			return nil, err
		}
		return val, nil
	case ConfigTypeFloat:
		var val float64
		if err := json.Unmarshal([]byte(c.Value), &val); err != nil {
			return nil, err
		}
		return val, nil
	case ConfigTypeBool:
		var val bool
		if err := json.Unmarshal([]byte(c.Value), &val); err != nil {
			return nil, err
		}
		return val, nil
	case ConfigTypeJSON:
		var val interface{}
		if err := json.Unmarshal([]byte(c.Value), &val); err != nil {
			return nil, err
		}
		return val, nil
	case ConfigTypeArray:
		var val []interface{}
		if err := json.Unmarshal([]byte(c.Value), &val); err != nil {
			return nil, err
		}
		return val, nil
	default:
		return c.Value, nil
	}
}

// SetTypedValue sets the configuration value from a typed value
func (c *Configuration) SetTypedValue(value interface{}) error {
	switch c.Type {
	case ConfigTypeString:
		if str, ok := value.(string); ok {
			c.Value = str
		} else {
			// Convert to string representation
			c.Value = fmt.Sprintf("%v", value)
		}
	default:
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return err
		}
		c.Value = string(jsonValue)
	}
	return nil
}

// IsEnabled checks if a feature flag is enabled for given context
func (f *FeatureFlag) IsEnabled(context map[string]interface{}) bool {
	if !f.Enabled {
		return false
	}

	// Check conditions
	for _, condition := range f.Conditions {
		if !f.evaluateCondition(condition, context) {
			return false
		}
	}

	// Check rollout percentage
	if f.Rollout != nil && f.Rollout.Percentage < 100 {
		// Simple hash-based percentage rollout
		// In production, use a more sophisticated algorithm
		if userID, ok := context["user_id"].(uint64); ok {
			return int(userID%100) < f.Rollout.Percentage
		}
	}

	return true
}

// evaluateCondition evaluates a single feature flag condition
func (f *FeatureFlag) evaluateCondition(condition FeatureFlagCondition, context map[string]interface{}) bool {
	contextValue, exists := context[condition.Type]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return contextValue == condition.Value
	case "contains":
		if str, ok := contextValue.(string); ok {
			if searchStr, ok := condition.Value.(string); ok {
				return contains(str, searchStr)
			}
		}
	case "in":
		if values, ok := condition.Value.([]interface{}); ok {
			for _, val := range values {
				if contextValue == val {
					return true
				}
			}
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(substr) <= len(s) && s[len(s)-len(substr):] == substr) ||
		(len(substr) <= len(s) && s[:len(substr)] == substr) ||
		(len(substr) < len(s) && findSubstring(s, substr)))
}

// findSubstring is a simple substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}