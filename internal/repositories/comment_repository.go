package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"high-performance-news-website/internal/models"
)

// CommentRepository handles database operations for comments
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new comment repository
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create inserts a new comment into the database
func (r *CommentRepository) Create(comment *models.Comment) (*models.Comment, error) {
	comment.PrepareForDB()
	
	if err := models.ValidateComment(comment); err != nil {
		return nil, err
	}
	
	query := `
		INSERT INTO comments (article_id, user_id, parent_id, content, author_name, 
			author_email, author_ip, user_agent, status, spam_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	
	err := r.db.QueryRow(query,
		comment.ArticleID,
		comment.UserID,
		comment.ParentID,
		comment.Content,
		comment.AuthorName,
		comment.AuthorEmail,
		comment.AuthorIP,
		comment.UserAgent,
		comment.Status,
		comment.SpamScore,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	
	return comment, nil
}

// GetByID retrieves a comment by its ID
func (r *CommentRepository) GetByID(id uint64) (*models.Comment, error) {
	query := `
		SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
			c.author_email, c.author_ip, c.user_agent, c.status, c.spam_score,
			c.moderated_by, c.moderated_at, c.created_at, c.updated_at,
			u.id, u.username, u.first_name, u.last_name, u.avatar,
			m.id, m.username, m.first_name, m.last_name
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN users m ON c.moderated_by = m.id
		WHERE c.id = $1`
	
	comment := &models.Comment{}
	var user models.User
	var moderator models.User
	var userID, moderatorID sql.NullInt64
	var userUsername, userFirstName, userLastName, userAvatar sql.NullString
	var modUsername, modFirstName, modLastName sql.NullString
	
	err := r.db.QueryRow(query, id).Scan(
		&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
		&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
		&comment.AuthorIP, &comment.UserAgent, &comment.Status, &comment.SpamScore,
		&comment.ModeratedBy, &comment.ModeratedAt, &comment.CreatedAt, &comment.UpdatedAt,
		&userID, &userUsername, &userFirstName, &userLastName, &userAvatar,
		&moderatorID, &modUsername, &modFirstName, &modLastName,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewNotFoundError("comment", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}
	
	// Populate user information if available
	if userID.Valid {
		user.ID = uint64(userID.Int64)
		user.Username = userUsername.String
		user.FirstName = userFirstName.String
		user.LastName = userLastName.String
		user.Avatar = userAvatar.String
		comment.User = &user
	}
	
	// Populate moderator information if available
	if moderatorID.Valid {
		moderator.ID = uint64(moderatorID.Int64)
		moderator.Username = modUsername.String
		moderator.FirstName = modFirstName.String
		moderator.LastName = modLastName.String
		comment.Moderator = &moderator
	}
	
	return comment, nil
}

// GetByArticleID retrieves comments for a specific article with threading
func (r *CommentRepository) GetByArticleID(articleID uint64, status models.CommentStatus) ([]models.Comment, error) {
	query := `
		SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
			c.author_email, c.status, c.moderated_by, c.moderated_at, c.created_at, c.updated_at,
			u.id, u.username, u.first_name, u.last_name, u.avatar
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.article_id = $1 AND c.status = $2
		ORDER BY c.created_at ASC`
	
	rows, err := r.db.Query(query, articleID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()
	
	var comments []models.Comment
	
	// Collect all comments as a flat list - let frontend organize them
	for rows.Next() {
		comment := models.Comment{}
		var user models.User
		var userID sql.NullInt64
		var userUsername, userFirstName, userLastName, userAvatar sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
			&comment.Status, &comment.ModeratedBy, &comment.ModeratedAt,
			&comment.CreatedAt, &comment.UpdatedAt,
			&userID, &userUsername, &userFirstName, &userLastName, &userAvatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		
		// Populate user information if available
		if userID.Valid {
			user.ID = uint64(userID.Int64)
			user.Username = userUsername.String
			user.FirstName = userFirstName.String
			user.LastName = userLastName.String
			user.Avatar = userAvatar.String
			comment.User = &user
		}
		
		comments = append(comments, comment)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}
	
	return comments, nil
}

// GetPendingComments retrieves comments pending moderation
func (r *CommentRepository) GetPendingComments(limit, offset int) ([]models.Comment, error) {
	query := `
		SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
			c.author_email, c.status, c.spam_score, c.created_at, c.updated_at,
			u.id, u.username, u.first_name, u.last_name, u.avatar,
			a.title, a.slug
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN articles a ON c.article_id = a.id
		WHERE c.status = $1
		ORDER BY c.created_at ASC
		LIMIT $2 OFFSET $3`
	
	rows, err := r.db.Query(query, models.CommentStatusPending, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending comments: %w", err)
	}
	defer rows.Close()
	
	var comments []models.Comment
	
	for rows.Next() {
		comment := models.Comment{}
		var user models.User
		var userID sql.NullInt64
		var userUsername, userFirstName, userLastName, userAvatar sql.NullString
		var articleTitle, articleSlug sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
			&comment.Status, &comment.SpamScore, &comment.CreatedAt, &comment.UpdatedAt,
			&userID, &userUsername, &userFirstName, &userLastName, &userAvatar,
			&articleTitle, &articleSlug,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending comment: %w", err)
		}
		
		// Populate user information if available
		if userID.Valid {
			user.ID = uint64(userID.Int64)
			user.Username = userUsername.String
			user.FirstName = userFirstName.String
			user.LastName = userLastName.String
			user.Avatar = userAvatar.String
			comment.User = &user
		}
		
		// Store article info for URL generation
		if articleSlug.Valid {
			comment.ArticleSlug = articleSlug.String
		}
		if articleTitle.Valid {
			comment.ArticleTitle = articleTitle.String
		}
		
		comments = append(comments, comment)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending comments: %w", err)
	}
	
	return comments, nil
}

// UpdateStatus updates the moderation status of a comment
func (r *CommentRepository) UpdateStatus(commentID uint64, status models.CommentStatus, moderatorID uint64, reason string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Update comment status
	query := `
		UPDATE comments 
		SET status = $1, moderated_by = $2, moderated_at = NOW(), updated_at = NOW()
		WHERE id = $3`
	
	result, err := tx.Exec(query, status, moderatorID, commentID)
	if err != nil {
		return fmt.Errorf("failed to update comment status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return models.NewNotFoundError("comment", fmt.Sprintf("%d", commentID))
	}
	
	// Log moderation action
	logQuery := `
		INSERT INTO comment_moderation_log (comment_id, action, moderator_id, reason, created_at)
		VALUES ($1, $2, $3, $4, NOW())`
	
	_, err = tx.Exec(logQuery, commentID, status, moderatorID, reason)
	if err != nil {
		return fmt.Errorf("failed to log moderation action: %w", err)
	}
	
	return tx.Commit()
}

// BulkUpdateStatus updates the status of multiple comments
func (r *CommentRepository) BulkUpdateStatus(commentIDs []uint64, status models.CommentStatus, moderatorID uint64, reason string) error {
	if len(commentIDs) == 0 {
		return nil
	}
	
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Update comments status
	query := `
		UPDATE comments 
		SET status = $1, moderated_by = $2, moderated_at = NOW(), updated_at = NOW()
		WHERE id = ANY($3)`
	
	_, err = tx.Exec(query, status, moderatorID, pq.Array(commentIDs))
	if err != nil {
		return fmt.Errorf("failed to bulk update comment status: %w", err)
	}
	
	// Log moderation actions
	logQuery := `
		INSERT INTO comment_moderation_log (comment_id, action, moderator_id, reason, created_at)
		SELECT unnest($1::bigint[]), $2, $3, $4, NOW()`
	
	_, err = tx.Exec(logQuery, pq.Array(commentIDs), status, moderatorID, reason)
	if err != nil {
		return fmt.Errorf("failed to log bulk moderation actions: %w", err)
	}
	
	return tx.Commit()
}

// Delete removes a comment and its replies
func (r *CommentRepository) Delete(commentID uint64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// First, delete all replies recursively
	deleteRepliesQuery := `
		WITH RECURSIVE comment_tree AS (
			SELECT id FROM comments WHERE id = $1
			UNION ALL
			SELECT c.id FROM comments c
			INNER JOIN comment_tree ct ON c.parent_id = ct.id
		)
		DELETE FROM comments WHERE id IN (SELECT id FROM comment_tree)`
	
	_, err = tx.Exec(deleteRepliesQuery, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment tree: %w", err)
	}
	
	return tx.Commit()
}

// GetCommentCount returns the total number of approved comments for an article
func (r *CommentRepository) GetCommentCount(articleID uint64) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE article_id = $1 AND status = $2`
	
	var count int
	err := r.db.QueryRow(query, articleID, models.CommentStatusApproved).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get comment count: %w", err)
	}
	
	return count, nil
}

// GetModerationStats returns statistics about comment moderation
func (r *CommentRepository) GetModerationStats() (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM comments
		GROUP BY status`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation stats: %w", err)
	}
	defer rows.Close()
	
	stats := make(map[string]int)
	total := 0
	
	for rows.Next() {
		var status string
		var count int
		
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan moderation stats: %w", err)
		}
		
		stats[status] = count
		total += count
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating moderation stats: %w", err)
	}
	
	// Add total count
	stats["total"] = total
	
	return stats, nil
}

// GetRecentComments returns recently created comments for monitoring with pagination
func (r *CommentRepository) GetRecentComments(limit, offset int) ([]models.Comment, int, error) {
	// First get the total count
	countQuery := `SELECT COUNT(*) FROM comments`
	var total int
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total comment count: %w", err)
	}
	
	// Then get the paginated results
	query := `
		SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
			c.author_email, c.status, c.spam_score, c.created_at,
			a.title, a.slug
		FROM comments c
		LEFT JOIN articles a ON c.article_id = a.id
		ORDER BY c.created_at DESC
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get recent comments: %w", err)
	}
	defer rows.Close()
	
	var comments []models.Comment
	
	for rows.Next() {
		comment := models.Comment{}
		var articleTitle, articleSlug sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
			&comment.Status, &comment.SpamScore, &comment.CreatedAt,
			&articleTitle, &articleSlug,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan recent comment: %w", err)
		}
		
		// Store article info for URL generation
		if articleSlug.Valid {
			comment.ArticleSlug = articleSlug.String
		}
		if articleTitle.Valid {
			comment.ArticleTitle = articleTitle.String
		}
		
		comments = append(comments, comment)
	}
	
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recent comments: %w", err)
	}
	
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating recent comments: %w", err)
	}
	
	return comments, total, nil
}

// SpamSettings represents spam detection configuration
type SpamSettings struct {
	Keywords  []string `json:"keywords"`
	Threshold float64  `json:"threshold"`
}

// GetSpamSettings retrieves current spam detection settings
func (r *CommentRepository) GetSpamSettings() (*SpamSettings, error) {
	query := `SELECT keywords, threshold FROM spam_settings WHERE id = 1`
	
	var keywordsJSON string
	var threshold float64
	
	err := r.db.QueryRow(query).Scan(&keywordsJSON, &threshold)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("spam settings not found")
		}
		return nil, fmt.Errorf("failed to get spam settings: %w", err)
	}
	
	// Parse JSON keywords
	var keywords []string
	if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
		return nil, fmt.Errorf("failed to parse keywords: %w", err)
	}
	
	return &SpamSettings{
		Keywords:  keywords,
		Threshold: threshold,
	}, nil
}

// UpdateSpamSettings updates spam detection settings
func (r *CommentRepository) UpdateSpamSettings(keywords []string, threshold float64) error {
	// Convert keywords to JSON
	keywordsJSON, err := json.Marshal(keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}
	
	// Upsert the settings
	query := `
		INSERT INTO spam_settings (id, keywords, threshold, updated_at) 
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) 
		DO UPDATE SET keywords = $1, threshold = $2, updated_at = NOW()`
	
	_, err = r.db.Exec(query, string(keywordsJSON), threshold)
	if err != nil {
		return fmt.Errorf("failed to update spam settings: %w", err)
	}
	
	return nil
}

// RecalculateAllSpamScores recalculates spam scores for all comments using current settings
func (r *CommentRepository) RecalculateAllSpamScores() (int, error) {
	// Get current spam settings
	spamSettings, err := r.GetSpamSettings()
	if err != nil {
		// Use default settings if none exist
		spamSettings = &SpamSettings{
			Keywords:  []string{"viagra", "casino", "lottery", "winner", "congratulations", "click here", "free money"},
			Threshold: 0.5,
		}
	}

	// Get all comments
	query := `SELECT id, content, author_name, author_email FROM comments`
	rows, err := r.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	updateCount := 0
	for rows.Next() {
		var id uint64
		var content, authorName, authorEmail string
		
		err := rows.Scan(&id, &content, &authorName, &authorEmail)
		if err != nil {
			continue // Skip problematic rows
		}

		// Create a temporary comment for spam detection
		tempComment := &models.Comment{
			ID:          id,
			Content:     content,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
		}

		// Calculate new spam score
		detection := models.DetectSpamWithSettings(tempComment, spamSettings.Keywords, spamSettings.Threshold)
		
		// Update the comment's spam score
		updateQuery := `UPDATE comments SET spam_score = $1 WHERE id = $2`
		_, err = r.db.Exec(updateQuery, detection.Score, id)
		if err == nil {
			updateCount++
		}
	}

	return updateCount, nil
}

// SearchComments searches comments by content or author
func (r *CommentRepository) SearchComments(query string, status models.CommentStatus, limit, offset int) ([]models.Comment, error) {
	var searchQuery string
	var args []interface{}
	
	if query == "" {
		// Status-only filtering
		searchQuery = `
			SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
				c.author_email, c.status, c.spam_score, c.created_at,
				a.title, a.slug
			FROM comments c
			LEFT JOIN articles a ON c.article_id = a.id
			WHERE ($1 = '' OR c.status = $1)
			ORDER BY c.created_at DESC
			LIMIT $2 OFFSET $3`
		
		statusFilter := ""
		if status != "" {
			statusFilter = string(status)
		}
		args = []interface{}{statusFilter, limit, offset}
	} else {
		// Text search with optional status filtering
		searchQuery = `
			SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
				c.author_email, c.status, c.spam_score, c.created_at,
				a.title, a.slug
			FROM comments c
			LEFT JOIN articles a ON c.article_id = a.id
			WHERE (c.content ILIKE $1 OR c.author_name ILIKE $1 OR c.author_email ILIKE $1)
			AND ($2 = '' OR c.status = $2)
			ORDER BY c.created_at DESC
			LIMIT $3 OFFSET $4`
		
		searchTerm := "%" + query + "%"
		statusFilter := ""
		if status != "" {
			statusFilter = string(status)
		}
		args = []interface{}{searchTerm, statusFilter, limit, offset}
	}
	
	rows, err := r.db.Query(searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search comments: %w", err)
	}
	defer rows.Close()
	
	var comments []models.Comment
	
	for rows.Next() {
		comment := models.Comment{}
		var articleTitle, articleSlug sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
			&comment.Status, &comment.SpamScore, &comment.CreatedAt,
			&articleTitle, &articleSlug,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		
		// Store article info for URL generation
		if articleSlug.Valid {
			comment.ArticleSlug = articleSlug.String
		}
		if articleTitle.Valid {
			comment.ArticleTitle = articleTitle.String
		}
		
		comments = append(comments, comment)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}
	
	return comments, nil
}

// UpdateContent updates the content of a comment
func (r *CommentRepository) UpdateContent(commentID uint64, content string) error {
	query := `UPDATE comments SET content = $1, updated_at = NOW() WHERE id = $2`
	
	result, err := r.db.Exec(query, content, commentID)
	if err != nil {
		return fmt.Errorf("failed to update comment content: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return models.NewNotFoundError("comment", fmt.Sprintf("%d", commentID))
	}
	
	return nil
}

// GetReplies retrieves all replies to a specific comment
func (r *CommentRepository) GetReplies(parentID uint64) ([]models.Comment, error) {
	query := `
		SELECT c.id, c.article_id, c.user_id, c.parent_id, c.content, c.author_name,
			c.author_email, c.author_ip, c.user_agent, c.status, c.spam_score,
			c.moderated_by, c.moderated_at, c.created_at, c.updated_at,
			u.id, u.username, u.first_name, u.last_name, u.avatar
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.parent_id = $1
		ORDER BY c.created_at ASC`
	
	rows, err := r.db.Query(query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}
	defer rows.Close()
	
	var replies []models.Comment
	
	for rows.Next() {
		comment := models.Comment{}
		var user models.User
		var userID sql.NullInt64
		var userUsername, userFirstName, userLastName, userAvatar sql.NullString
		
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.AuthorName, &comment.AuthorEmail,
			&comment.AuthorIP, &comment.UserAgent, &comment.Status, &comment.SpamScore,
			&comment.ModeratedBy, &comment.ModeratedAt, &comment.CreatedAt, &comment.UpdatedAt,
			&userID, &userUsername, &userFirstName, &userLastName, &userAvatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reply: %w", err)
		}
		
		// Populate user information if available
		if userID.Valid {
			user.ID = uint64(userID.Int64)
			user.Username = userUsername.String
			user.FirstName = userFirstName.String
			user.LastName = userLastName.String
			user.Avatar = userAvatar.String
			comment.User = &user
		}
		
		replies = append(replies, comment)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating replies: %w", err)
	}
	
	return replies, nil
}