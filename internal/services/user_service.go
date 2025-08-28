package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// UserService handles user-related operations
type UserService struct {
	repo        *repositories.UserRepository
	authService *auth.AuthService
}

// NewUserService creates a new user service
func NewUserService(db *sql.DB, authService *auth.AuthService) *UserService {
	return &UserService{
		repo:        repositories.NewUserRepository(db),
		authService: authService,
	}
}



// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username  string           `json:"username" validate:"required,min=3,max=50"`
	Email     string           `json:"email" validate:"required,email,max=255"`
	Password  string           `json:"password" validate:"required,min=8,max=128"`
	Role      models.UserRole  `json:"role" validate:"required"`
	FirstName string           `json:"first_name" validate:"max=100"`
	LastName  string           `json:"last_name" validate:"max=100"`
	Bio       string           `json:"bio" validate:"max=1000"`
	Avatar    string           `json:"avatar" validate:"omitempty,url,max=500"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Username  *string          `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email     *string          `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Role      *models.UserRole `json:"role,omitempty"`
	FirstName *string          `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string          `json:"last_name,omitempty" validate:"omitempty,max=100"`
	Bio       *string          `json:"bio,omitempty" validate:"omitempty,max=1000"`
	Avatar    *string          `json:"avatar,omitempty" validate:"omitempty,url,max=500"`
	IsActive  *bool            `json:"is_active,omitempty"`
}

// ChangePasswordRequest represents a request to change user password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User   *models.User      `json:"user"`
	Tokens *auth.TokenPair   `json:"tokens"`
}

// Create creates a new user with role-based permission checking
func (s *UserService) Create(req *CreateUserRequest, currentUser *models.User) (*models.User, error) {
	// Check permissions - only admins and editors can create users
	if currentUser != nil && !currentUser.HasPermission("manage_users") {
		// Editors can only create reporters and contributors
		if currentUser.Role == models.RoleEditor {
			if req.Role != models.RoleReporter && req.Role != models.RoleContributor {
				return nil, auth.ErrInsufficientPermissions
			}
		} else {
			return nil, auth.ErrInsufficientPermissions
		}
	}
	
	// Create user model
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Role:      req.Role,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Bio:       req.Bio,
		Avatar:    req.Avatar,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Validate user data
	if err := models.ValidateUser(user); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}
	
	// Check if username already exists
	existingUser, err := s.repo.GetByUsername(user.Username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, &models.ValidationError{
			Message: "Username already exists",
			Fields:  []string{"username already taken"},
		}
	}
	
	// Check if email already exists
	existingUser, err = s.repo.GetByEmail(user.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, &models.ValidationError{
			Message: "Email already exists",
			Fields:  []string{"email already taken"},
		}
	}
	
	// Hash password
	hashedPassword, err := s.authService.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = hashedPassword
	
	// Prepare for database insertion
	user.PrepareForDB()
	
	// Create user in database
	createdUser, err := s.repo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return createdUser, nil
}

// GetByID retrieves a user by ID with permission checking
func (s *UserService) GetByID(id uint64, currentUser *models.User) (*models.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check permissions - users can view themselves, admins can view all, editors can view reporters/contributors
	if currentUser != nil && !s.canViewUser(currentUser, user) {
		return nil, auth.ErrInsufficientPermissions
	}
	
	return user, nil
}

// GetByUsername retrieves a user by username (used for authentication)
func (s *UserService) GetByUsername(username string) (*models.User, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	
	return user, nil
}

// Update updates a user with role-based permission checking
func (s *UserService) Update(id uint64, req *UpdateUserRequest, currentUser *models.User) (*models.User, error) {
	// Get existing user
	user, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check permissions
	if currentUser != nil && !currentUser.CanManageUser(user) {
		return nil, auth.ErrInsufficientPermissions
	}
	
	// Update fields if provided
	if req.Username != nil {
		// Check username uniqueness
		existingUser, err := s.repo.GetByUsername(*req.Username)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
		}
		if existingUser != nil && existingUser.ID != user.ID {
			return nil, &models.ValidationError{
				Message: "Username already exists",
				Fields:  []string{"username already taken"},
			}
		}
		user.Username = *req.Username
	}
	
	if req.Email != nil {
		// Check email uniqueness
		existingUser, err := s.repo.GetByEmail(*req.Email)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
		}
		if existingUser != nil && existingUser.ID != user.ID {
			return nil, &models.ValidationError{
				Message: "Email already exists",
				Fields:  []string{"email already taken"},
			}
		}
		user.Email = *req.Email
	}
	
	if req.Role != nil {
		// Only admins can change roles, editors can only change to reporter/contributor
		if currentUser != nil {
			if currentUser.Role != models.RoleAdmin {
				if currentUser.Role == models.RoleEditor {
					if *req.Role != models.RoleReporter && *req.Role != models.RoleContributor {
						return nil, auth.ErrInsufficientPermissions
					}
				} else {
					return nil, auth.ErrInsufficientPermissions
				}
			}
		}
		user.Role = *req.Role
	}
	
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	
	if req.Bio != nil {
		user.Bio = *req.Bio
	}
	
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	
	if req.IsActive != nil {
		// Only admins can change active status
		if currentUser != nil && currentUser.Role != models.RoleAdmin {
			return nil, auth.ErrInsufficientPermissions
		}
		user.IsActive = *req.IsActive
	}
	
	user.UpdatedAt = time.Now()
	
	// Validate updated user
	if err := models.ValidateUser(user); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}
	
	// Prepare for database update
	user.PrepareForDB()
	
	// Update user in database
	updatedUser, err := s.repo.Update(user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	return updatedUser, nil
}

// Delete deletes a user with permission checking
func (s *UserService) Delete(id uint64, currentUser *models.User) error {
	// Get existing user
	user, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check permissions - only admins can delete users
	if currentUser != nil && currentUser.Role != models.RoleAdmin {
		return auth.ErrInsufficientPermissions
	}
	
	// Prevent self-deletion
	if currentUser != nil && currentUser.ID == user.ID {
		return &models.ValidationError{
			Message: "Cannot delete your own account",
			Fields:  []string{"self-deletion not allowed"},
		}
	}
	
	// Delete user from database
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	return nil
}

// List retrieves a list of users with pagination and permission checking
func (s *UserService) List(limit, offset int, currentUser *models.User) ([]*models.User, int, error) {
	// Check permissions - only admins and editors can list users
	if currentUser != nil && !currentUser.HasPermission("manage_users") {
		return nil, 0, auth.ErrInsufficientPermissions
	}
	
	var users []*models.User
	var total int
	var err error
	
	// Editors can only see reporters and contributors
	if currentUser != nil && currentUser.Role == models.RoleEditor {
		// Get reporters
		reporters, reporterTotal, err := s.repo.ListByRole(models.RoleReporter, limit/2, offset/2)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list reporters: %w", err)
		}
		
		// Get contributors
		contributors, contributorTotal, err := s.repo.ListByRole(models.RoleContributor, limit/2, offset/2)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list contributors: %w", err)
		}
		
		users = append(reporters, contributors...)
		total = reporterTotal + contributorTotal
	} else {
		// Admins can see all users
		users, total, err = s.repo.List(limit, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list users: %w", err)
		}
	}
	
	return users, total, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(userID uint64, req *ChangePasswordRequest, currentUser *models.User) error {
	// Get user
	user, err := s.repo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check permissions - users can change their own password, admins can change any password
	if currentUser != nil && currentUser.ID != user.ID && currentUser.Role != models.RoleAdmin {
		return auth.ErrInsufficientPermissions
	}
	
	// Verify current password (except for admin changing other user's password)
	if currentUser != nil && currentUser.ID == user.ID {
		if err := s.authService.VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
			return auth.ErrInvalidCredentials
		}
	}
	
	// Hash new password
	hashedPassword, err := s.authService.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}
	
	// Update password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()
	
	_, err = s.repo.Update(user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	return nil
}

// GetByEmail retrieves a user by email
func (s *UserService) GetByEmail(email string) (*models.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	
	return user, nil
}

// Login authenticates a user and returns tokens
func (s *UserService) Login(req *LoginRequest) (*LoginResponse, error) {
	// Try to get user by email first (since login form uses email)
	user, err := s.GetByEmail(strings.ToLower(strings.TrimSpace(req.Username)))
	if err != nil {
		// If not found by email, try by username
		user, err = s.GetByUsername(strings.ToLower(strings.TrimSpace(req.Username)))
		if err != nil {
			return nil, auth.ErrInvalidCredentials
		}
	}
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}
	
	// Check if user is active
	if !user.IsActive {
		return nil, auth.ErrUserInactive
	}
	
	// Verify password
	if err := s.authService.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, auth.ErrInvalidCredentials
	}
	
	// Generate tokens
	tokens, err := s.authService.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	
	return &LoginResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *UserService) RefreshToken(refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	claims, err := s.authService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	
	// Get user
	user, err := s.repo.GetByID(claims.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check if user is still active
	if !user.IsActive {
		return nil, auth.ErrUserInactive
	}
	
	// Generate new token pair
	tokens, err := s.authService.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	
	return tokens, nil
}

// canViewUser checks if currentUser can view targetUser
func (s *UserService) canViewUser(currentUser, targetUser *models.User) bool {
	// Users can view themselves
	if currentUser.ID == targetUser.ID {
		return true
	}
	
	// Admins can view everyone
	if currentUser.Role == models.RoleAdmin {
		return true
	}
	
	// Editors can view reporters and contributors
	if currentUser.Role == models.RoleEditor {
		return targetUser.Role == models.RoleReporter || targetUser.Role == models.RoleContributor
	}
	
	return false
}

// GetTotalCount returns the total number of users
func (s *UserService) GetTotalCount() (int64, error) {
	return s.repo.GetTotalCount()
}

// GetNewUsersToday returns the count of users created today
func (s *UserService) GetNewUsersToday() (int64, error) {
	return s.repo.GetNewUsersToday()
}

// GetNewUsersThisMonth returns the count of users created this month
func (s *UserService) GetNewUsersThisMonth() (int64, error) {
	return s.repo.GetNewUsersThisMonth()
}

// GetCountByRole returns the count of users by role
func (s *UserService) GetCountByRole(role models.UserRole) (int64, error) {
	return s.repo.GetCountByRole(role)
}

// HealthCheck checks if the user service is healthy
func (s *UserService) HealthCheck() error {
	// Simple health check - try to count users
	_, err := s.GetTotalCount()
	return err
}