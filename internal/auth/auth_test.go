package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"high-performance-news-website/internal/models"
)

func TestAuthService_HashPassword(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")

	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{
			name:        "valid strong password",
			password:    "SecurePass123!",
			expectError: false,
		},
		{
			name:        "password too short",
			password:    "Short1!",
			expectError: true,
		},
		{
			name:        "password too long",
			password:    "ThisPasswordIsWayTooLongAndExceedsTheMaximumLengthOf128CharactersWhichShouldCauseAnErrorWhenTryingToHashItBecauseItViolatesOurPasswordPolicy",
			expectError: true,
		},
		{
			name:        "password without uppercase",
			password:    "lowercase123!",
			expectError: true,
		},
		{
			name:        "password without lowercase",
			password:    "UPPERCASE123!",
			expectError: true,
		},
		{
			name:        "password without numbers",
			password:    "NoNumbers!",
			expectError: true,
		},
		{
			name:        "password without special characters",
			password:    "NoSpecial123",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.HashPassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash) // Hash should be different from password
			}
		})
	}
}

func TestAuthService_VerifyPassword(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	password := "SecurePass123!"
	
	// Hash the password
	hash, err := authService.HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name         string
		hashedPass   string
		plainPass    string
		expectError  bool
	}{
		{
			name:        "correct password",
			hashedPass:  hash,
			plainPass:   password,
			expectError: false,
		},
		{
			name:        "incorrect password",
			hashedPass:  hash,
			plainPass:   "WrongPassword123!",
			expectError: true,
		},
		{
			name:        "empty password",
			hashedPass:  hash,
			plainPass:   "",
			expectError: true,
		},
		{
			name:        "invalid hash",
			hashedPass:  "invalid-hash",
			plainPass:   password,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.VerifyPassword(tt.hashedPass, tt.plainPass)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_GenerateAccessToken(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, expiresAt, err := authService.GenerateAccessToken(user)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(25*time.Hour))) // Should be within 25 hours
}

func TestAuthService_GenerateRefreshToken(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	userID := uint64(1)
	token, err := authService.GenerateRefreshToken(userID)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_GenerateTokenPair(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	tokenPair, err := authService.GenerateTokenPair(user)
	
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.True(t, tokenPair.ExpiresAt.After(time.Now()))
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	// Generate a valid token
	validToken, _, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: false,
		},
		{
			name:        "invalid token format",
			token:       "invalid.token.format",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "malformed token",
			token:       "not-a-jwt-token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateAccessToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, user.ID, claims.UserID)
				assert.Equal(t, user.Username, claims.Username)
				assert.Equal(t, user.Role, claims.Role)
			}
		})
	}
}

func TestAuthService_ValidateRefreshToken(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	userID := uint64(1)

	// Generate a valid refresh token
	validToken, err := authService.GenerateRefreshToken(userID)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid refresh token",
			token:       validToken,
			expectError: false,
		},
		{
			name:        "invalid token format",
			token:       "invalid.token.format",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "access token used as refresh token",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwicm9sZSI6ImFkbWluIn0.invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateRefreshToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
			}
		})
	}
}

func TestAuthService_TokenExpiration(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	// Generate token
	token, expiresAt, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	// Validate immediately (should work)
	claims, err := authService.ValidateAccessToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Check expiration time is reasonable
	expectedExpiry := time.Now().Add(TokenExpiration)
	assert.WithinDuration(t, expectedExpiry, expiresAt, time.Minute)
}

func TestAuthService_DifferentSecrets(t *testing.T) {
	authService1 := NewAuthService("secret1", "refresh-secret1")
	authService2 := NewAuthService("secret2", "refresh-secret2")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	// Generate token with first service
	token, _, err := authService1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Try to validate with second service (should fail)
	claims, err := authService2.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// Validate with correct service (should work)
	claims, err = authService1.ValidateAccessToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
}

func TestGenerateSecureSecret(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "32 byte secret",
			length: 32,
		},
		{
			name:   "64 byte secret",
			length: 64,
		},
		{
			name:   "16 byte secret",
			length: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := GenerateSecureSecret(tt.length)
			
			assert.NoError(t, err)
			assert.NotEmpty(t, secret)
			
			// Generate another secret and ensure they're different
			secret2, err := GenerateSecureSecret(tt.length)
			assert.NoError(t, err)
			assert.NotEqual(t, secret, secret2)
		})
	}
}

func TestAuthService_RoleBasedClaims(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	roles := []models.UserRole{
		models.RoleAdmin,
		models.RoleEditor,
		models.RoleReporter,
		models.RoleContributor,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			user := &models.User{
				ID:       1,
				Username: "testuser",
				Role:     role,
			}

			token, _, err := authService.GenerateAccessToken(user)
			require.NoError(t, err)

			claims, err := authService.ValidateAccessToken(token)
			require.NoError(t, err)
			
			assert.Equal(t, role, claims.Role)
			assert.Equal(t, user.ID, claims.UserID)
			assert.Equal(t, user.Username, claims.Username)
		})
	}
}

func TestAuthService_TokenReuse(t *testing.T) {
	authService := NewAuthService("test-jwt-secret", "test-refresh-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	// Generate multiple tokens for the same user
	token1, _, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)
	
	token2, _, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)

	// Both tokens should be valid but different
	assert.NotEqual(t, token1, token2)

	claims1, err := authService.ValidateAccessToken(token1)
	assert.NoError(t, err)
	assert.NotNil(t, claims1)

	claims2, err := authService.ValidateAccessToken(token2)
	assert.NoError(t, err)
	assert.NotNil(t, claims2)

	// Claims should have same user data
	assert.Equal(t, claims1.UserID, claims2.UserID)
	assert.Equal(t, claims1.Username, claims2.Username)
	assert.Equal(t, claims1.Role, claims2.Role)
}