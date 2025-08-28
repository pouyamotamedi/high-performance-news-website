package services

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
)

func setupUserServiceTest(t *testing.T) (*UserService, *sql.DB, func()) {
	// Setup test database - using nil for now since SetupTestDB is not implemented
	var db *sql.DB
	
	// Create auth service with test secrets
	authService := auth.NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	// Create user service
	userService := NewUserService(db, authService)
	
	cleanup := func() {
		// Cleanup function - currently no-op since we're using nil db
	}
	
	return userService, db, cleanup
}

func createTestUser(t *testing.T, service *UserService, role models.UserRole) *models.User {
	req := &CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "TestPassword123!",
		Role:      role,
		FirstName: "Test",
		LastName:  "User",
		Bio:       "Test bio",
	}
	
	user, err := service.Create(req, nil) // nil for admin creation
	require.NoError(t, err)
	return user
}

func TestUserService_Create(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		request     *CreateUserRequest
		currentUser *models.User
		expectError bool
		errorType   error
	}{
		{
			name: "successful user creation",
			request: &CreateUserRequest{
				Username:  "newuser",
				Email:     "newuser@example.com",
				Password:  "SecurePass123!",
				Role:      models.RoleReporter,
				FirstName: "New",
				LastName:  "User",
			},
			currentUser: nil, // Admin creation
			expectError: false,
		},
		{
			name: "duplicate username",
			request: &CreateUserRequest{
				Username:  "testuser", // Already exists from createTestUser
				Email:     "different@example.com",
				Password:  "SecurePass123!",
				Role:      models.RoleReporter,
			},
			currentUser: nil,
			expectError: true,
		},
		{
			name: "invalid password",
			request: &CreateUserRequest{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "weak", // Too weak
				Role:     models.RoleReporter,
			},
			currentUser: nil,
			expectError: true,
		},
		{
			name: "editor creating admin (should fail)",
			request: &CreateUserRequest{
				Username: "adminuser",
				Email:    "admin@example.com",
				Password: "SecurePass123!",
				Role:     models.RoleAdmin,
			},
			currentUser: &models.User{Role: models.RoleEditor},
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
		{
			name: "reporter creating user (should fail)",
			request: &CreateUserRequest{
				Username: "newreporter",
				Email:    "reporter@example.com",
				Password: "SecurePass123!",
				Role:     models.RoleReporter,
			},
			currentUser: &models.User{Role: models.RoleReporter},
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
	}

	// Create initial test user for duplicate tests
	createTestUser(t, service, models.RoleAdmin)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Create(tt.request, tt.currentUser)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Username, user.Username)
				assert.Equal(t, tt.request.Email, user.Email)
				assert.Equal(t, tt.request.Role, user.Role)
				assert.NotEmpty(t, user.PasswordHash)
				assert.True(t, user.IsActive)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test user
	user := createTestUser(t, service, models.RoleAdmin)

	tests := []struct {
		name        string
		request     *LoginRequest
		expectError bool
		errorType   error
	}{
		{
			name: "successful login",
			request: &LoginRequest{
				Username: "testuser",
				Password: "TestPassword123!",
			},
			expectError: false,
		},
		{
			name: "wrong password",
			request: &LoginRequest{
				Username: "testuser",
				Password: "WrongPassword123!",
			},
			expectError: true,
			errorType:   auth.ErrInvalidCredentials,
		},
		{
			name: "non-existent user",
			request: &LoginRequest{
				Username: "nonexistent",
				Password: "TestPassword123!",
			},
			expectError: true,
			errorType:   auth.ErrInvalidCredentials,
		},
		{
			name: "case insensitive username",
			request: &LoginRequest{
				Username: "TESTUSER",
				Password: "TestPassword123!",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.Login(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotNil(t, response.User)
				assert.NotNil(t, response.Tokens)
				assert.NotEmpty(t, response.Tokens.AccessToken)
				assert.NotEmpty(t, response.Tokens.RefreshToken)
				assert.Equal(t, user.ID, response.User.ID)
			}
		})
	}
}

func TestUserService_Update(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test users
	admin := createTestUser(t, service, models.RoleAdmin)
	editor := createTestUser(t, service, models.RoleEditor)
	reporter := createTestUser(t, service, models.RoleReporter)

	tests := []struct {
		name        string
		userID      uint64
		request     *UpdateUserRequest
		currentUser *models.User
		expectError bool
		errorType   error
	}{
		{
			name:   "admin updating any user",
			userID: reporter.ID,
			request: &UpdateUserRequest{
				FirstName: stringPtr("Updated"),
				LastName:  stringPtr("Name"),
			},
			currentUser: admin,
			expectError: false,
		},
		{
			name:   "user updating themselves",
			userID: reporter.ID,
			request: &UpdateUserRequest{
				Bio: stringPtr("Updated bio"),
			},
			currentUser: reporter,
			expectError: false,
		},
		{
			name:   "editor updating reporter",
			userID: reporter.ID,
			request: &UpdateUserRequest{
				FirstName: stringPtr("Editor Updated"),
			},
			currentUser: editor,
			expectError: false,
		},
		{
			name:   "reporter updating admin (should fail)",
			userID: admin.ID,
			request: &UpdateUserRequest{
				FirstName: stringPtr("Hacker"),
			},
			currentUser: reporter,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
		{
			name:   "editor changing role to admin (should fail)",
			userID: reporter.ID,
			request: &UpdateUserRequest{
				Role: rolePtr(models.RoleAdmin),
			},
			currentUser: editor,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
		{
			name:   "non-admin changing active status (should fail)",
			userID: reporter.ID,
			request: &UpdateUserRequest{
				IsActive: boolPtr(false),
			},
			currentUser: editor,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Update(tt.userID, tt.request, tt.currentUser)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				
				// Verify updates
				if tt.request.FirstName != nil {
					assert.Equal(t, *tt.request.FirstName, user.FirstName)
				}
				if tt.request.LastName != nil {
					assert.Equal(t, *tt.request.LastName, user.LastName)
				}
				if tt.request.Bio != nil {
					assert.Equal(t, *tt.request.Bio, user.Bio)
				}
			}
		})
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test users
	admin := createTestUser(t, service, models.RoleAdmin)
	user := createTestUser(t, service, models.RoleReporter)

	tests := []struct {
		name        string
		userID      uint64
		request     *ChangePasswordRequest
		currentUser *models.User
		expectError bool
		errorType   error
	}{
		{
			name:   "user changing own password",
			userID: user.ID,
			request: &ChangePasswordRequest{
				CurrentPassword: "TestPassword123!",
				NewPassword:     "NewSecurePass456!",
			},
			currentUser: user,
			expectError: false,
		},
		{
			name:   "admin changing user password",
			userID: user.ID,
			request: &ChangePasswordRequest{
				CurrentPassword: "", // Not required for admin
				NewPassword:     "AdminSetPass789!",
			},
			currentUser: admin,
			expectError: false,
		},
		{
			name:   "wrong current password",
			userID: user.ID,
			request: &ChangePasswordRequest{
				CurrentPassword: "WrongPassword123!",
				NewPassword:     "NewSecurePass456!",
			},
			currentUser: user,
			expectError: true,
			errorType:   auth.ErrInvalidCredentials,
		},
		{
			name:   "weak new password",
			userID: user.ID,
			request: &ChangePasswordRequest{
				CurrentPassword: "TestPassword123!",
				NewPassword:     "weak",
			},
			currentUser: user,
			expectError: true,
		},
		{
			name:   "unauthorized user changing password",
			userID: admin.ID,
			request: &ChangePasswordRequest{
				CurrentPassword: "TestPassword123!",
				NewPassword:     "HackerPass123!",
			},
			currentUser: user,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ChangePassword(tt.userID, tt.request, tt.currentUser)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_List(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test users
	admin := createTestUser(t, service, models.RoleAdmin)
	editor := createTestUser(t, service, models.RoleEditor)
	reporter := createTestUser(t, service, models.RoleReporter)

	tests := []struct {
		name        string
		limit       int
		offset      int
		currentUser *models.User
		expectError bool
		errorType   error
		minCount    int
	}{
		{
			name:        "admin listing all users",
			limit:       10,
			offset:      0,
			currentUser: admin,
			expectError: false,
			minCount:    3, // At least the 3 created users
		},
		{
			name:        "editor listing users",
			limit:       10,
			offset:      0,
			currentUser: editor,
			expectError: false,
			minCount:    1, // Should see at least the reporter
		},
		{
			name:        "reporter trying to list users (should fail)",
			limit:       10,
			offset:      0,
			currentUser: reporter,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, total, err := service.List(tt.limit, tt.offset, tt.currentUser)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, users)
				assert.Equal(t, 0, total)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, users)
				assert.GreaterOrEqual(t, len(users), tt.minCount)
				assert.GreaterOrEqual(t, total, tt.minCount)
			}
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test users
	admin := createTestUser(t, service, models.RoleAdmin)
	reporter := createTestUser(t, service, models.RoleReporter)

	tests := []struct {
		name        string
		userID      uint64
		currentUser *models.User
		expectError bool
		errorType   error
	}{
		{
			name:        "admin deleting user",
			userID:      reporter.ID,
			currentUser: admin,
			expectError: false,
		},
		{
			name:        "admin trying to delete self (should fail)",
			userID:      admin.ID,
			currentUser: admin,
			expectError: true,
		},
		{
			name:        "non-admin trying to delete user (should fail)",
			userID:      reporter.ID,
			currentUser: reporter,
			expectError: true,
			errorType:   auth.ErrInsufficientPermissions,
		},
		{
			name:        "deleting non-existent user",
			userID:      99999,
			currentUser: admin,
			expectError: true,
			errorType:   auth.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Delete(tt.userID, tt.currentUser)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_RefreshToken(t *testing.T) {
	service, _, cleanup := setupUserServiceTest(t)
	defer cleanup()

	// Create test user and login
	user := createTestUser(t, service, models.RoleAdmin)
	require.NotNil(t, user) // Use the user variable
	loginResp, err := service.Login(&LoginRequest{
		Username: "testuser",
		Password: "TestPassword123!",
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		refreshToken string
		expectError  bool
		errorType    error
	}{
		{
			name:         "valid refresh token",
			refreshToken: loginResp.Tokens.RefreshToken,
			expectError:  false,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid.token.here",
			expectError:  true,
			errorType:    auth.ErrInvalidToken,
		},
		{
			name:         "empty refresh token",
			refreshToken: "",
			expectError:  true,
			errorType:    auth.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := service.RefreshToken(tt.refreshToken)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, tokens)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokens)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
			}
		})
	}
}

// Helper functions for pointer values in tests
func stringPtr(s string) *string {
	return &s
}

func rolePtr(r models.UserRole) *models.UserRole {
	return &r
}

func boolPtr(b bool) *bool {
	return &b
}