package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	
	err := r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.FirstName,
		user.LastName,
		user.Bio,
		user.Avatar,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uint64) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		WHERE id = $1`
	
	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.Bio,
		&user.Avatar,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		WHERE username = $1`
	
	user := &models.User{}
	err := r.db.QueryRow(query, strings.ToLower(username)).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.Bio,
		&user.Avatar,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		WHERE email = $1`
	
	user := &models.User{}
	err := r.db.QueryRow(query, strings.ToLower(email)).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.Bio,
		&user.Avatar,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

// Update updates a user in the database
func (r *UserRepository) Update(user *models.User) (*models.User, error) {
	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, role = $5, first_name = $6, last_name = $7, bio = $8, avatar = $9, is_active = $10, updated_at = $11
		WHERE id = $1
		RETURNING updated_at`
	
	err := r.db.QueryRow(
		query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.FirstName,
		user.LastName,
		user.Bio,
		user.Avatar,
		user.IsActive,
		user.UpdatedAt,
	).Scan(&user.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	return user, nil
}

// Delete deletes a user from the database
func (r *UserRepository) Delete(id uint64) error {
	query := `DELETE FROM users WHERE id = $1`
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(limit, offset int) ([]*models.User, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM users`
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}
	
	// Get users with pagination
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.FirstName,
			&user.LastName,
			&user.Bio,
			&user.Avatar,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, total, nil
}

// ListByRole retrieves a paginated list of users by role
func (r *UserRepository) ListByRole(role models.UserRole, limit, offset int) ([]*models.User, int, error) {
	// Get total count for role
	var total int
	countQuery := `SELECT COUNT(*) FROM users WHERE role = $1`
	err := r.db.QueryRow(countQuery, role).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count by role: %w", err)
	}
	
	// Get users with pagination
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		WHERE role = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	
	rows, err := r.db.Query(query, role, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users by role: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.FirstName,
			&user.LastName,
			&user.Bio,
			&user.Avatar,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, total, nil
}

// BulkCreate creates multiple users in a single transaction
func (r *UserRepository) BulkCreate(users []*models.User) ([]*models.User, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO users (username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	
	stmt, err := tx.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	var createdUsers []*models.User
	for _, user := range users {
		err := stmt.QueryRow(
			user.Username,
			user.Email,
			user.PasswordHash,
			user.Role,
			user.FirstName,
			user.LastName,
			user.Bio,
			user.Avatar,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
		
		if err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", user.Username, err)
		}
		
		createdUsers = append(createdUsers, user)
	}
	
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return createdUsers, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(userID uint64) error {
	query := `UPDATE users SET updated_at = $1 WHERE id = $2`
	
	_, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	
	return nil
}

// GetActiveUserCount returns the count of active users
func (r *UserRepository) GetActiveUserCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE is_active = true`
	
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active user count: %w", err)
	}
	
	return count, nil
}

// GetUsersByRole returns users filtered by role
func (r *UserRepository) GetUsersByRole(role models.UserRole) ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, first_name, last_name, bio, avatar, is_active, created_at, updated_at
		FROM users
		WHERE role = $1 AND is_active = true
		ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, role)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by role: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.FirstName,
			&user.LastName,
			&user.Bio,
			&user.Avatar,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, nil
}

// GetTotalCount returns the total number of users
func (r *UserRepository) GetTotalCount() (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM users WHERE is_active = true"
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total user count: %w", err)
	}
	return count, nil
}

// GetNewUsersToday returns the count of users created today
func (r *UserRepository) GetNewUsersToday() (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*) FROM users 
		WHERE DATE(created_at) = CURRENT_DATE 
		AND is_active = true`
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get new users today count: %w", err)
	}
	return count, nil
}

// GetNewUsersThisMonth returns the count of users created this month
func (r *UserRepository) GetNewUsersThisMonth() (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*) FROM users 
		WHERE DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)
		AND is_active = true`
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get new users this month count: %w", err)
	}
	return count, nil
}

// GetCountByRole returns the count of users by role
func (r *UserRepository) GetCountByRole(role models.UserRole) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM users WHERE role = $1 AND is_active = true"
	err := r.db.QueryRow(query, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count by role: %w", err)
	}
	return count, nil
}