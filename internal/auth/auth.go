package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"high-performance-news-website/internal/models"
)

const (
	// BcryptCost defines the cost for bcrypt hashing (12 is recommended for production)
	BcryptCost = 12
	
	// TokenExpiration defines how long JWT tokens are valid
	TokenExpiration = 24 * time.Hour
	
	// RefreshTokenExpiration defines how long refresh tokens are valid
	RefreshTokenExpiration = 7 * 24 * time.Hour
)

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret     []byte
	refreshSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret, refreshSecret string) *AuthService {
	return &AuthService{
		jwtSecret:     []byte(jwtSecret),
		refreshSecret: []byte(refreshSecret),
	}
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   uint64           `json:"user_id"`
	Username string           `json:"username"`
	Role     models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

// RefreshClaims represents refresh token claims
type RefreshClaims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

// HashPassword hashes a password using bcrypt with proper salt rounds
func (a *AuthService) HashPassword(password string) (string, error) {
	// Validate password strength first
	if err := models.IsValidPassword(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}
	
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	
	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against its hash
func (a *AuthService) VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// GenerateTokenPair generates both access and refresh tokens for a user
func (a *AuthService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := a.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	// Generate refresh token
	refreshToken, err := a.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// GenerateAccessToken generates a JWT access token for a user
func (a *AuthService) GenerateAccessToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(TokenExpiration)
	
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "news-website",
			Subject:   fmt.Sprintf("user:%d", user.ID),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, expiresAt, nil
}

// GenerateRefreshToken generates a refresh token for a user
func (a *AuthService) GenerateRefreshToken(userID uint64) (string, error) {
	expiresAt := time.Now().Add(RefreshTokenExpiration)
	
	claims := &RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "news-website",
			Subject:   fmt.Sprintf("refresh:%d", userID),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.refreshSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	
	return tokenString, nil
}

// ValidateAccessToken validates and parses an access token
func (a *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	
	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}
	
	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token
func (a *AuthService) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.refreshSecret, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}
	
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}
	
	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("refresh token expired")
	}
	
	return claims, nil
}

// GenerateSecureSecret generates a cryptographically secure secret
func GenerateSecureSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure secret: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// AuthError represents authentication-related errors
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common authentication errors
var (
	ErrInvalidCredentials = &AuthError{
		Code:    "INVALID_CREDENTIALS",
		Message: "Invalid username or password",
	}
	ErrUserNotFound = &AuthError{
		Code:    "USER_NOT_FOUND",
		Message: "User not found",
	}
	ErrUserInactive = &AuthError{
		Code:    "USER_INACTIVE",
		Message: "User account is inactive",
	}
	ErrTokenExpired = &AuthError{
		Code:    "TOKEN_EXPIRED",
		Message: "Token has expired",
	}
	ErrInvalidToken = &AuthError{
		Code:    "INVALID_TOKEN",
		Message: "Invalid token",
	}
	ErrInsufficientPermissions = &AuthError{
		Code:    "INSUFFICIENT_PERMISSIONS",
		Message: "Insufficient permissions for this operation",
	}
)