package repositories

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"high-performance-news-website/internal/models"
)

func setupUserRepositoryTest(t *testing.T) (*UserRepository, *sql.DB, func()) {
	// For unit tests, we'll skip database setup
	t.Skip("Database setup not available for unit tests")
	return nil, nil, func() {}
}

func createTestUser(t *testing.T, repo *UserRepository, username, email string, role models.UserRole) *models.User {
	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: "hashed_password",
		Role:         role,
		FirstName:    "Test",
		LastName:     "User",
		Bio:          "Test bio",
		Avatar:       "https://example.com/avatar.jpg",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createdUser, err := repo.Create(user)
	require.NoError(t, err)
	return createdUser
}

func TestUserRepository_Create(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		Role:         models.RoleReporter,
		FirstName:    "Test",
		LastName:     "User",
		Bio:          "Test bio",
		Avatar:       "https://example.com/avatar.jpg",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createdUser, err := repo.Create(user)

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.NotZero(t, createdUser.ID)
	assert.Equal(t, user.Username, createdUser.Username)
	assert.Equal(t, user.Email, createdUser.Email)
	assert.Equal(t, user.Role, createdUser.Role)
	assert.NotZero(t, createdUser.CreatedAt)
	assert.NotZero(t, createdUser.UpdatedAt)
}

func TestUserRepository_GetByID(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	tests := []struct {
		name        string
		userID      uint64
		expectError bool
	}{
		{
			name:        "existing user",
			userID:      createdUser.ID,
			expectError: false,
		},
		{
			name:        "non-existent user",
			userID:      99999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
				assert.Equal(t, createdUser.Username, user.Username)
				assert.Equal(t, createdUser.Email, user.Email)
			}
		})
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	tests := []struct {
		name        string
		username    string
		expectError bool
	}{
		{
			name:        "existing username",
			username:    "testuser",
			expectError: false,
		},
		{
			name:        "case insensitive username",
			username:    "TESTUSER",
			expectError: false,
		},
		{
			name:        "non-existent username",
			username:    "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByUsername(tt.username)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, createdUser.ID, user.ID)
				assert.Equal(t, createdUser.Username, user.Username)
			}
		})
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{
			name:        "existing email",
			email:       "test@example.com",
			expectError: false,
		},
		{
			name:        "case insensitive email",
			email:       "TEST@EXAMPLE.COM",
			expectError: false,
		},
		{
			name:        "non-existent email",
			email:       "nonexistent@example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByEmail(tt.email)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, createdUser.ID, user.ID)
				assert.Equal(t, createdUser.Email, user.Email)
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	// Update user data
	createdUser.FirstName = "Updated"
	createdUser.LastName = "Name"
	createdUser.Bio = "Updated bio"
	createdUser.Role = models.RoleEditor
	createdUser.UpdatedAt = time.Now()

	updatedUser, err := repo.Update(createdUser)

	assert.NoError(t, err)
	assert.NotNil(t, updatedUser)
	assert.Equal(t, createdUser.ID, updatedUser.ID)
	assert.Equal(t, "Updated", updatedUser.FirstName)
	assert.Equal(t, "Name", updatedUser.LastName)
	assert.Equal(t, "Updated bio", updatedUser.Bio)
	assert.Equal(t, models.RoleEditor, updatedUser.Role)
}

func TestUserRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	tests := []struct {
		name        string
		userID      uint64
		expectError bool
	}{
		{
			name:        "delete existing user",
			userID:      createdUser.ID,
			expectError: false,
		},
		{
			name:        "delete non-existent user",
			userID:      99999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
			} else {
				assert.NoError(t, err)
				
				// Verify user is deleted
				_, err := repo.GetByID(tt.userID)
				assert.Error(t, err)
				assert.Equal(t, sql.ErrNoRows, err)
			}
		})
	}
}

func TestUserRepository_List(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create multiple test users
	_ = []*models.User{
		createTestUser(t, repo, "user1", "user1@example.com", models.RoleAdmin),
		createTestUser(t, repo, "user2", "user2@example.com", models.RoleEditor),
		createTestUser(t, repo, "user3", "user3@example.com", models.RoleReporter),
	}

	tests := []struct {
		name         string
		limit        int
		offset       int
		expectedMin  int
		expectedMax  int
	}{
		{
			name:        "get all users",
			limit:       10,
			offset:      0,
			expectedMin: 3,
			expectedMax: 10,
		},
		{
			name:        "get first 2 users",
			limit:       2,
			offset:      0,
			expectedMin: 2,
			expectedMax: 2,
		},
		{
			name:        "get users with offset",
			limit:       10,
			offset:      1,
			expectedMin: 2,
			expectedMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userList, total, err := repo.List(tt.limit, tt.offset)

			assert.NoError(t, err)
			assert.NotNil(t, userList)
			assert.GreaterOrEqual(t, len(userList), tt.expectedMin)
			assert.LessOrEqual(t, len(userList), tt.expectedMax)
			assert.GreaterOrEqual(t, total, 3) // At least 3 users created
		})
	}
}

func TestUserRepository_ListByRole(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create users with different roles
	createTestUser(t, repo, "admin", "admin@example.com", models.RoleAdmin)
	createTestUser(t, repo, "editor1", "editor1@example.com", models.RoleEditor)
	createTestUser(t, repo, "editor2", "editor2@example.com", models.RoleEditor)
	createTestUser(t, repo, "reporter", "reporter@example.com", models.RoleReporter)

	tests := []struct {
		name          string
		role          models.UserRole
		expectedCount int
	}{
		{
			name:          "get admins",
			role:          models.RoleAdmin,
			expectedCount: 1,
		},
		{
			name:          "get editors",
			role:          models.RoleEditor,
			expectedCount: 2,
		},
		{
			name:          "get reporters",
			role:          models.RoleReporter,
			expectedCount: 1,
		},
		{
			name:          "get contributors (none)",
			role:          models.RoleContributor,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, total, err := repo.ListByRole(tt.role, 10, 0)

			assert.NoError(t, err)
			assert.NotNil(t, users)
			assert.Equal(t, tt.expectedCount, len(users))
			assert.Equal(t, tt.expectedCount, total)

			// Verify all users have the correct role
			for _, user := range users {
				assert.Equal(t, tt.role, user.Role)
			}
		})
	}
}

func TestUserRepository_BulkCreate(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	users := []*models.User{
		{
			Username:     "bulk1",
			Email:        "bulk1@example.com",
			PasswordHash: "hashed_password",
			Role:         models.RoleReporter,
			FirstName:    "Bulk",
			LastName:     "User1",
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Username:     "bulk2",
			Email:        "bulk2@example.com",
			PasswordHash: "hashed_password",
			Role:         models.RoleContributor,
			FirstName:    "Bulk",
			LastName:     "User2",
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	createdUsers, err := repo.BulkCreate(users)

	assert.NoError(t, err)
	assert.NotNil(t, createdUsers)
	assert.Equal(t, 2, len(createdUsers))

	for i, user := range createdUsers {
		assert.NotZero(t, user.ID)
		assert.Equal(t, users[i].Username, user.Username)
		assert.Equal(t, users[i].Email, user.Email)
		assert.Equal(t, users[i].Role, user.Role)
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create test user
	createdUser := createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)
	originalUpdatedAt := createdUser.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	err := repo.UpdateLastLogin(createdUser.ID)
	assert.NoError(t, err)

	// Verify the updated_at timestamp was changed
	updatedUser, err := repo.GetByID(createdUser.ID)
	assert.NoError(t, err)
	assert.True(t, updatedUser.UpdatedAt.After(originalUpdatedAt))
}

func TestUserRepository_GetActiveUserCount(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create active users
	createTestUser(t, repo, "active1", "active1@example.com", models.RoleReporter)
	createTestUser(t, repo, "active2", "active2@example.com", models.RoleEditor)

	// Create inactive user
	inactiveUser := &models.User{
		Username:     "inactive",
		Email:        "inactive@example.com",
		PasswordHash: "hashed_password",
		Role:         models.RoleReporter,
		IsActive:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err := repo.Create(inactiveUser)
	require.NoError(t, err)

	count, err := repo.GetActiveUserCount()
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // Only active users
}

func TestUserRepository_GetUsersByRole(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create active users with different roles
	createTestUser(t, repo, "admin", "admin@example.com", models.RoleAdmin)
	createTestUser(t, repo, "editor", "editor@example.com", models.RoleEditor)

	// Create inactive user with same role
	inactiveUser := &models.User{
		Username:     "inactive_editor",
		Email:        "inactive@example.com",
		PasswordHash: "hashed_password",
		Role:         models.RoleEditor,
		IsActive:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err := repo.Create(inactiveUser)
	require.NoError(t, err)

	// Get active editors only
	editors, err := repo.GetUsersByRole(models.RoleEditor)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(editors)) // Only active editor
	assert.Equal(t, models.RoleEditor, editors[0].Role)
	assert.True(t, editors[0].IsActive)
}

func TestUserRepository_UniqueConstraints(t *testing.T) {
	repo, _, cleanup := setupUserRepositoryTest(t)
	defer cleanup()

	// Create first user
	_ = createTestUser(t, repo, "testuser", "test@example.com", models.RoleReporter)

	// Try to create user with same username
	user2 := &models.User{
		Username:     "testuser", // Same username
		Email:        "different@example.com",
		PasswordHash: "hashed_password",
		Role:         models.RoleReporter,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := repo.Create(user2)
	assert.Error(t, err) // Should fail due to unique constraint

	// Try to create user with same email
	user3 := &models.User{
		Username:     "differentuser",
		Email:        "test@example.com", // Same email
		PasswordHash: "hashed_password",
		Role:         models.RoleReporter,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = repo.Create(user3)
	assert.Error(t, err) // Should fail due to unique constraint
}