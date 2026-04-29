package models

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateStruct validates any struct using reflection and validation tags
func ValidateStruct(s interface{}) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", s)
	}
	
	t := v.Type()
	var errors []string
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		// Get validation tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}
		
		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := validateField(field, fieldType.Name, rule); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	
	if len(errors) > 0 {
		return &ValidationError{
			Message: "Struct validation failed",
			Fields:  errors,
		}
	}
	
	return nil
}

// validateField validates a single field against a rule
func validateField(field reflect.Value, fieldName, rule string) error {
	switch {
	case rule == "required":
		if isZeroValue(field) {
			return fmt.Errorf("%s is required", fieldName)
		}
	case strings.HasPrefix(rule, "max="):
		maxStr := strings.TrimPrefix(rule, "max=")
		if field.Kind() == reflect.String {
			if len(field.String()) > parseInt(maxStr) {
				return fmt.Errorf("%s exceeds maximum length of %s", fieldName, maxStr)
			}
		}
	case strings.HasPrefix(rule, "min="):
		minStr := strings.TrimPrefix(rule, "min=")
		if field.Kind() == reflect.String {
			if len(field.String()) < parseInt(minStr) {
				return fmt.Errorf("%s is below minimum length of %s", fieldName, minStr)
			}
		}
	case rule == "email":
		if field.Kind() == reflect.String {
			if !IsValidEmail(field.String()) {
				return fmt.Errorf("%s must be a valid email", fieldName)
			}
		}
	case rule == "url":
		if field.Kind() == reflect.String && field.String() != "" {
			if !IsValidURL(field.String()) {
				return fmt.Errorf("%s must be a valid URL", fieldName)
			}
		}
	case rule == "slug":
		if field.Kind() == reflect.String && field.String() != "" {
			if !IsValidSlug(field.String()) {
				return fmt.Errorf("%s must be a valid slug", fieldName)
			}
		}
	case rule == "hexcolor":
		if field.Kind() == reflect.String && field.String() != "" {
			if !IsValidHexColor(field.String()) {
				return fmt.Errorf("%s must be a valid hex color", fieldName)
			}
		}
	case strings.HasPrefix(rule, "oneof="):
		values := strings.Split(strings.TrimPrefix(rule, "oneof="), " ")
		if field.Kind() == reflect.String {
			valid := false
			for _, v := range values {
				if field.String() == v {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("%s must be one of: %s", fieldName, strings.Join(values, ", "))
			}
		}
	}
	
	return nil
}

// isZeroValue checks if a reflect.Value is zero
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// parseInt converts string to int, returns 0 if invalid
func parseInt(s string) int {
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			return 0
		}
	}
	return result
}