package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// UserRole represents different user access levels
type UserRole string

const (
	RoleAdmin       UserRole = "admin"
	RoleEditor      UserRole = "editor"
	RoleReporter    UserRole = "reporter"
	RoleContributor UserRole = "contributor"
)

// User represents a system user with role-based access
type User struct {
	ID           uint64    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username" validate:"required,min=3,max=50,alphanum"`
	Email        string    `json:"email" db:"email" validate:"required,email,max=255"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         UserRole  `json:"role" db:"role" validate:"required"`
	FirstName    string    `json:"first_name" db:"first_name" validate:"max=100"`
	LastName     string    `json:"last_name" db:"last_name" validate:"max=100"`
	Bio          string    `json:"bio" db:"bio" validate:"max=1000"`
	Avatar       string    `json:"avatar" db:"avatar" validate:"omitempty,url,max=500"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ValidateUser validates a User struct with comprehensive error checking
func ValidateUser(user *User) error {
	var errors []string

	// Username validation
	if strings.TrimSpace(user.Username) == "" {
		errors = append(errors, "username is required")
	}
	if len(user.Username) < 3 {
		errors = append(errors, "username must be at least 3 characters")
	}
	if len(user.Username) > 50 {
		errors = append(errors, "username must be less than 50 characters")
	}
	if !IsValidUsername(user.Username) {
		errors = append(errors, "username can only contain letters, numbers, and underscores")
	}

	// Email validation
	if strings.TrimSpace(user.Email) == "" {
		errors = append(errors, "email is required")
	}
	if len(user.Email) > 255 {
		errors = append(errors, "email must be less than 255 characters")
	}
	if !IsValidEmail(user.Email) {
		errors = append(errors, "email format is invalid")
	}

	// Role validation
	if !IsValidRole(user.Role) {
		errors = append(errors, "role must be one of: admin, editor, reporter, contributor")
	}

	// First name validation
	if len(user.FirstName) > 100 {
		errors = append(errors, "first_name must be less than 100 characters")
	}

	// Last name validation
	if len(user.LastName) > 100 {
		errors = append(errors, "last_name must be less than 100 characters")
	}

	// Bio validation
	if len(user.Bio) > 1000 {
		errors = append(errors, "bio must be less than 1000 characters")
	}

	// Avatar URL validation
	if user.Avatar != "" {
		if len(user.Avatar) > 500 {
			errors = append(errors, "avatar URL must be less than 500 characters")
		}
		if !IsValidURL(user.Avatar) {
			errors = append(errors, "avatar must be a valid URL")
		}
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "User validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// IsValidUsername checks if username contains only valid characters
func IsValidUsername(username string) bool {
	// Username should contain only letters, numbers, and underscores
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidRole checks if the role is valid
func IsValidRole(role UserRole) bool {
	validRoles := map[UserRole]bool{
		RoleAdmin:       true,
		RoleEditor:      true,
		RoleReporter:    true,
		RoleContributor: true,
	}
	return validRoles[role]
}

// GetRolePermissions returns the permissions for a given role
func GetRolePermissions(role UserRole) []string {
	switch role {
	case RoleAdmin:
		return []string{"create", "read", "update", "delete", "manage_users", "manage_system"}
	case RoleEditor:
		return []string{"create", "read", "update", "delete", "publish", "moderate"}
	case RoleReporter:
		return []string{"create", "read", "update"}
	case RoleContributor:
		return []string{"create", "read"}
	default:
		return []string{"read"}
	}
}

// HasPermission checks if a user role has a specific permission
func (u *User) HasPermission(permission string) bool {
	permissions := GetRolePermissions(u.Role)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// CanManageUser checks if a user can manage another user based on roles
func (u *User) CanManageUser(targetUser *User) bool {
	// Admin can manage everyone
	if u.Role == RoleAdmin {
		return true
	}
	
	// Editor can manage reporters and contributors
	if u.Role == RoleEditor {
		return targetUser.Role == RoleReporter || targetUser.Role == RoleContributor
	}
	
	// Others can only manage themselves
	return u.ID == targetUser.ID
}

// PrepareForDB prepares the user for database insertion
func (u *User) PrepareForDB() {
	u.Username = strings.TrimSpace(strings.ToLower(u.Username))
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
	u.FirstName = strings.TrimSpace(u.FirstName)
	u.LastName = strings.TrimSpace(u.LastName)
	u.Bio = strings.TrimSpace(u.Bio)
	
	if u.Role == "" {
		u.Role = RoleContributor // Default role
	}
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if fullName == "" {
		return u.Username
	}
	return fullName
}

// IsValidPassword validates password strength
func IsValidPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	if len(password) > 128 {
		return errors.New("password must be less than 128 characters")
	}
	
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	
	return nil
}