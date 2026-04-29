package repositories

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"high-performance-news-website/internal/models"
)

func TestCommentRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		comment := &models.Comment{
			ArticleID:   1,
			Content:     "Test comment",
			AuthorName:  "John Doe",
			AuthorEmail: "john@example.com",
			AuthorIP:    "192.168.1.1",
			UserAgent:   "Mozilla/5.0",
			Status:      models.CommentStatusPending,
			SpamScore:   0.1,
		}

		now := time.Now()
		mock.ExpectQuery(`INSERT INTO comments`).
			WithArgs(
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
			).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, now, now))

		result, err := repo.Create(comment)
		if err != nil {
			t.Errorf("Create() error = %v", err)
			return
		}

		if result.ID != 1 {
			t.Errorf("Create() ID = %d, want 1", result.ID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		comment := &models.Comment{
			// Missing required fields
			Content: "",
		}

		_, err := repo.Create(comment)
		if err == nil {
			t.Error("Create() expected validation error but got none")
		}

		if _, ok := err.(*models.ValidationError); !ok {
			t.Errorf("Create() expected ValidationError but got %T", err)
		}
	})
}

func TestCommentRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful retrieval", func(t *testing.T) {
		commentID := uint64(1)
		now := time.Now()

		mock.ExpectQuery(`SELECT c\.id, c\.article_id`).
			WithArgs(commentID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "article_id", "user_id", "parent_id", "content", "author_name",
				"author_email", "author_ip", "user_agent", "status", "spam_score",
				"moderated_by", "moderated_at", "created_at", "updated_at",
				"user_id", "username", "first_name", "last_name", "avatar",
				"moderator_id", "mod_username", "mod_first_name", "mod_last_name",
			}).AddRow(
				1, 1, nil, nil, "Test comment", "John Doe",
				"john@example.com", "192.168.1.1", "Mozilla/5.0", "pending", 0.1,
				nil, nil, now, now,
				nil, nil, nil, nil, nil,
				nil, nil, nil, nil,
			))

		comment, err := repo.GetByID(commentID)
		if err != nil {
			t.Errorf("GetByID() error = %v", err)
			return
		}

		if comment.ID != commentID {
			t.Errorf("GetByID() ID = %d, want %d", comment.ID, commentID)
		}

		if comment.Content != "Test comment" {
			t.Errorf("GetByID() Content = %q, want %q", comment.Content, "Test comment")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("comment not found", func(t *testing.T) {
		commentID := uint64(999)

		mock.ExpectQuery(`SELECT c\.id, c\.article_id`).
			WithArgs(commentID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetByID(commentID)
		if err == nil {
			t.Error("GetByID() expected error but got none")
			return
		}

		if _, ok := err.(*models.NotFoundError); !ok {
			t.Errorf("GetByID() expected NotFoundError but got %T", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCommentRepository_GetByArticleID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful retrieval with threading", func(t *testing.T) {
		articleID := uint64(1)
		now := time.Now()

		// Mock data: parent comment and reply
		mock.ExpectQuery(`SELECT c\.id, c\.article_id`).
			WithArgs(articleID, models.CommentStatusApproved).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "article_id", "user_id", "parent_id", "content", "author_name",
				"author_email", "status", "moderated_by", "moderated_at", "created_at", "updated_at",
				"user_id", "username", "first_name", "last_name", "avatar",
			}).
				AddRow(1, 1, nil, nil, "Parent comment", "John Doe", "john@example.com", "approved", nil, nil, now, now, nil, nil, nil, nil, nil).
				AddRow(2, 1, nil, 1, "Reply comment", "Jane Doe", "jane@example.com", "approved", nil, nil, now, now, nil, nil, nil, nil, nil))

		comments, err := repo.GetByArticleID(articleID, models.CommentStatusApproved)
		if err != nil {
			t.Errorf("GetByArticleID() error = %v", err)
			return
		}

		if len(comments) != 1 {
			t.Errorf("GetByArticleID() returned %d top-level comments, want 1", len(comments))
			return
		}

		// Check parent comment
		parentComment := comments[0]
		if parentComment.ID != 1 {
			t.Errorf("Parent comment ID = %d, want 1", parentComment.ID)
		}

		if len(parentComment.Replies) != 1 {
			t.Errorf("Parent comment has %d replies, want 1", len(parentComment.Replies))
			return
		}

		// Check reply
		reply := parentComment.Replies[0]
		if reply.ID != 2 {
			t.Errorf("Reply ID = %d, want 2", reply.ID)
		}

		if reply.ParentID == nil || *reply.ParentID != 1 {
			t.Errorf("Reply ParentID = %v, want 1", reply.ParentID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCommentRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful status update", func(t *testing.T) {
		commentID := uint64(1)
		moderatorID := uint64(2)
		status := models.CommentStatusApproved
		reason := "Looks good"

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE comments`).
			WithArgs(status, moderatorID, commentID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO comment_moderation_log`).
			WithArgs(commentID, status, moderatorID, reason).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.UpdateStatus(commentID, status, moderatorID, reason)
		if err != nil {
			t.Errorf("UpdateStatus() error = %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("comment not found", func(t *testing.T) {
		commentID := uint64(999)
		moderatorID := uint64(2)
		status := models.CommentStatusApproved
		reason := "Looks good"

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE comments`).
			WithArgs(status, moderatorID, commentID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected
		mock.ExpectRollback()

		err := repo.UpdateStatus(commentID, status, moderatorID, reason)
		if err == nil {
			t.Error("UpdateStatus() expected error but got none")
			return
		}

		if _, ok := err.(*models.NotFoundError); !ok {
			t.Errorf("UpdateStatus() expected NotFoundError but got %T", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCommentRepository_BulkUpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful bulk update", func(t *testing.T) {
		commentIDs := []uint64{1, 2, 3}
		moderatorID := uint64(2)
		status := models.CommentStatusApproved
		reason := "Bulk approval"

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE comments`).
			WithArgs(status, moderatorID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectExec(`INSERT INTO comment_moderation_log`).
			WithArgs(sqlmock.AnyArg(), status, moderatorID, reason).
			WillReturnResult(sqlmock.NewResult(1, 3))
		mock.ExpectCommit()

		err := repo.BulkUpdateStatus(commentIDs, status, moderatorID, reason)
		if err != nil {
			t.Errorf("BulkUpdateStatus() error = %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("empty comment IDs", func(t *testing.T) {
		commentIDs := []uint64{}
		moderatorID := uint64(2)
		status := models.CommentStatusApproved
		reason := "Bulk approval"

		err := repo.BulkUpdateStatus(commentIDs, status, moderatorID, reason)
		if err != nil {
			t.Errorf("BulkUpdateStatus() with empty IDs should not error, got: %v", err)
		}
	})
}

func TestCommentRepository_GetCommentCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful count", func(t *testing.T) {
		articleID := uint64(1)
		expectedCount := 5

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM comments`).
			WithArgs(articleID, models.CommentStatusApproved).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedCount))

		count, err := repo.GetCommentCount(articleID)
		if err != nil {
			t.Errorf("GetCommentCount() error = %v", err)
			return
		}

		if count != expectedCount {
			t.Errorf("GetCommentCount() = %d, want %d", count, expectedCount)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCommentRepository_GetModerationStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful stats retrieval", func(t *testing.T) {
		mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
				AddRow("pending", 10).
				AddRow("approved", 50).
				AddRow("rejected", 5).
				AddRow("spam", 2))

		stats, err := repo.GetModerationStats()
		if err != nil {
			t.Errorf("GetModerationStats() error = %v", err)
			return
		}

		expectedStats := map[string]int{
			"pending":  10,
			"approved": 50,
			"rejected": 5,
			"spam":     2,
		}

		for status, expectedCount := range expectedStats {
			if count, exists := stats[status]; !exists || count != expectedCount {
				t.Errorf("GetModerationStats() status %s = %d, want %d", status, count, expectedCount)
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	t.Run("successful deletion", func(t *testing.T) {
		commentID := uint64(1)

		mock.ExpectBegin()
		mock.ExpectExec(`WITH RECURSIVE comment_tree`).
			WithArgs(commentID).
			WillReturnResult(sqlmock.NewResult(0, 3)) // Deleted comment and 2 replies
		mock.ExpectCommit()

		err := repo.Delete(commentID)
		if err != nil {
			t.Errorf("Delete() error = %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}