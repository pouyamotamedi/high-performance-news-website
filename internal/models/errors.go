package models

import (
	"fmt"
	"strings"
)

// ValidationError represents validation errors with detailed field information
type ValidationError struct {
	Message string   `json:"message"`
	Fields  []string `json:"fields"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return e.Message
	}
	
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(e.Fields, ", "))
}

// AddField adds a field error to the validation error
func (e *ValidationError) AddField(field string) {
	e.Fields = append(e.Fields, field)
}

// HasFields returns true if there are field errors
func (e *ValidationError) HasFields() bool {
	return len(e.Fields) > 0
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string `json:"resource"`
	ID       string `json:"id"`
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.Resource, e.ID)
}

// DuplicateError represents a duplicate resource error
type DuplicateError struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

// Error implements the error interface
func (e *DuplicateError) Error() string {
	return fmt.Sprintf("%s with %s '%s' already exists", e.Resource, e.Field, e.Value)
}

// UnauthorizedError represents an authorization error
type UnauthorizedError struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

// Error implements the error interface
func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized to %s %s", e.Action, e.Resource)
}

// NewValidationError creates a new validation error
func NewValidationError(message string, fields ...string) *ValidationError {
	return &ValidationError{
		Message: message,
		Fields:  fields,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// NewDuplicateError creates a new duplicate error
func NewDuplicateError(resource, field, value string) *DuplicateError {
	return &DuplicateError{
		Resource: resource,
		Field:    field,
		Value:    value,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(action, resource string) *UnauthorizedError {
	return &UnauthorizedError{
		Action:   action,
		Resource: resource,
	}
}