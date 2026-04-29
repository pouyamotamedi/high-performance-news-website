package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// User API request/response types

// CreateUserAPIRequest represents a request to create a user via API
type CreateUserAPIRequest struct {
	Username  string          `json:"username" validate:"required,min=3,max=50"`
	Email     string          `json:"email" validate:"required,email,max=255"`
	Password  string          `json:"password" validate:"required,min=8,max=128"`
	Role      models.UserRole `json:"role" validate:"required"`
	FirstName string          `json:"first_name" validate:"max=100"`
	LastName  string          `json:"last_name" validate:"max=100"`
	Bio       string          `json:"bio" validate:"max=1000"`
	Avatar    string          `json:"avatar" validate:"omitempty,url,max=500"`
}

// UpdateUserAPIRequest represents a request to update a user via API
type UpdateUserAPIRequest struct {
	Username  *string          `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email     *string          `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Role      *models.UserRole `json:"role,omitempty"`
	FirstName *string          `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string          `json:"last_name,omitempty" validate:"omitempty,max=100"`
	Bio       *string          `json:"bio,omitempty" validate:"omitempty,max=1000"`
	Avatar    *string          `json:"avatar,omitempty" validate:"omitempty,url,max=500"`
	IsActive  *bool            `json:"is_active,omitempty"`
}

// ChangePasswordAPIRequest represents a password change request via API
type ChangePasswordAPIRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// LoginAPIRequest represents a login request via API
type LoginAPIRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UserListResponse represents the response for user listing
type UserListResponse struct {
	Users      []models.User `json:"users"`
	Pagination Pagination    `json:"pagination"`
}

// Authentication Handlers

// Login authenticates a user and returns tokens
func (h *APIHandler) Login(c *gin.Context) {
	var req LoginAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Convert to service request
	loginReq := &services.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	}

	// Authenticate through service
	response, err := h.userService.Login(loginReq)
	if err != nil {
		handleError(c, err)
		return
	}

	// Set secure HttpOnly cookie for admin panel compatibility
	// This is the professional approach for session management
	if response.User.Role == "admin" || response.User.Role == "editor" {
		c.SetCookie(
			"auth_token",                    // name
			response.Tokens.AccessToken,     // value
			86400,                          // maxAge (24 hours)
			"/",                            // path
			"",                             // domain (empty = current domain)
			true,                           // secure (HTTPS only)
			true,                           // httpOnly (XSS protection)
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Login successful",
	})
}

// RefreshToken refreshes an access token
func (h *APIHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Refresh token through service
	tokens, err := h.userService.RefreshToken(req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    tokens,
		Message: "Token refreshed successfully",
	})
}

// User CRUD Handlers

// CreateUser creates a new user
func (h *APIHandler) CreateUser(c *gin.Context) {
	var req CreateUserAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to service request
	createReq := &services.CreateUserRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		Role:      req.Role,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Bio:       req.Bio,
		Avatar:    req.Avatar,
	}

	// Create user through service
	user, err := h.userService.Create(createReq, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    user,
		Message: "User created successfully",
	})
}

// GetUser retrieves a user by ID
func (h *APIHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid user ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Get user through service
	user, err := h.userService.GetByID(id, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: user,
	})
}

// UpdateUser updates an existing user
func (h *APIHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid user ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	var req UpdateUserAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to service request
	updateReq := &services.UpdateUserRequest{
		Username:  req.Username,
		Email:     req.Email,
		Role:      req.Role,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Bio:       req.Bio,
		Avatar:    req.Avatar,
		IsActive:  req.IsActive,
	}

	// Update user through service
	user, err := h.userService.Update(id, updateReq, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    user,
		Message: "User updated successfully",
	})
}

// DeleteUser deletes a user
func (h *APIHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid user ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Delete user through service
	err = h.userService.Delete(id, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "User deleted successfully",
	})
}

// ListUsers retrieves users with pagination and role-based filtering
func (h *APIHandler) ListUsers(c *gin.Context) {
	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// List users through service
	users, total, err := h.userService.List(limit, offset, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	// Calculate pagination
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	response := UserListResponse{
		Users: make([]models.User, len(users)),
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// Convert pointers to values for response
	for i, user := range users {
		response.Users[i] = *user
	}

	c.JSON(http.StatusOK, response)
}

// ChangePassword changes a user's password
func (h *APIHandler) ChangePassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid user ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	var req ChangePasswordAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to service request
	changeReq := &services.ChangePasswordRequest{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	// Change password through service
	err = h.userService.ChangePassword(id, changeReq, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Password changed successfully",
	})
}

// GetCurrentUser returns the current authenticated user's information
func (h *APIHandler) GetCurrentUser(c *gin.Context) {
	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: currentUser,
	})
}

// ExportUsers exports users in various formats
func (h *APIHandler) ExportUsers(c *gin.Context) {
	// Get export parameters
	format := c.DefaultQuery("format", "csv")
	includeInactive := c.DefaultQuery("include_inactive", "true") == "true"
	includePasswords := c.DefaultQuery("include_passwords", "false") == "true"
	includeTimestamps := c.DefaultQuery("include_timestamps", "true") == "true"

	// Get current user for authorization
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Only admins can include password hashes
	if includePasswords && currentUser.Role != models.RoleAdmin {
		includePasswords = false
	}

	// Get all users (no pagination for export)
	users, _, err := h.userService.List(1000, 0, currentUser) // Max 1000 users for export
	if err != nil {
		handleError(c, err)
		return
	}

	// Filter inactive users if requested
	if !includeInactive {
		activeUsers := make([]*models.User, 0)
		for _, user := range users {
			if user.IsActive {
				activeUsers = append(activeUsers, user)
			}
		}
		users = activeUsers
	}

	switch format {
	case "csv":
		h.exportUsersCSV(c, users, includePasswords, includeTimestamps)
	case "json":
		h.exportUsersJSON(c, users, includePasswords, includeTimestamps)
	case "xlsx":
		// For now, return CSV format for Excel (can be improved later)
		h.exportUsersCSV(c, users, includePasswords, includeTimestamps)
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Unsupported export format. Supported formats: csv, json, xlsx",
		})
	}
}

// exportUsersCSV exports users in CSV format
func (h *APIHandler) exportUsersCSV(c *gin.Context, users []*models.User, includePasswords, includeTimestamps bool) {
	// Build CSV header
	header := "ID,Username,Email,First Name,Last Name,Role,Status"
	if includePasswords {
		header += ",Password Hash"
	}
	if includeTimestamps {
		header += ",Created At,Updated At"
	}
	header += "\n"

	// Build CSV content
	content := header
	for _, user := range users {
		status := "Active"
		if !user.IsActive {
			status = "Inactive"
		}

		row := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s",
			user.ID, user.Username, user.Email, user.FirstName, user.LastName, user.Role, status)

		if includePasswords {
			row += "," + user.PasswordHash
		}
		if includeTimestamps {
			row += "," + user.CreatedAt.Format("2006-01-02 15:04:05") + "," + user.UpdatedAt.Format("2006-01-02 15:04:05")
		}
		row += "\n"
		content += row
	}

	// Set headers for file download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=users_export.csv")
	c.String(http.StatusOK, content)
}

// exportUsersJSON exports users in JSON format
func (h *APIHandler) exportUsersJSON(c *gin.Context, users []*models.User, includePasswords, includeTimestamps bool) {
	// Create export data
	exportData := make([]map[string]interface{}, len(users))
	
	for i, user := range users {
		userData := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"role":       user.Role,
			"is_active":  user.IsActive,
		}

		if includePasswords {
			userData["password_hash"] = user.PasswordHash
		}
		if includeTimestamps {
			userData["created_at"] = user.CreatedAt
			userData["updated_at"] = user.UpdatedAt
		}

		exportData[i] = userData
	}

	// Set headers for file download
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=users_export.json")
	c.JSON(http.StatusOK, map[string]interface{}{
		"users":       exportData,
		"total_count": len(users),
		"exported_at": time.Now(),
	})
}
