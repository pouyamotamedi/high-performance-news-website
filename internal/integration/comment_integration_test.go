package integration

import (
	"testing"

	"high-performance-news-website/internal/models"
)

// TestCommentSystemIntegration tests the comment system components working together
func TestCommentSystemIntegration(t *testing.T) {
	t.Run("comment validation and spam detection", func(t *testing.T) {
		// Test 1: Valid comment
		comment := &models.Comment{
			ArticleID:   1,
			Content:     "This is a great article! I really enjoyed reading it.",
			AuthorName:  "John Doe",
			AuthorEmail: "john@example.com",
			Status:      models.CommentStatusPending,
		}

		// Validate the comment
		err := models.ValidateComment(comment)
		if err != nil {
			t.Errorf("Valid comment failed validation: %v", err)
		}

		// Test spam detection
		spamDetection := models.DetectSpam(comment)
		if spamDetection.IsSpam {
			t.Error("Normal comment was flagged as spam")
		}

		if spamDetection.Score >= 0.5 {
			t.Errorf("Normal comment has high spam score: %f", spamDetection.Score)
		}

		// Test 2: Spam comment
		spamComment := &models.Comment{
			ArticleID:   1,
			Content:     "CONGRATULATIONS! You won the lottery! Visit https://spam.com and https://more-spam.com and https://even-more-spam.com NOW!!!!",
			AuthorName:  "Spammer",
			AuthorEmail: "spam@example.com",
			Status:      models.CommentStatusPending,
		}

		// Validate the spam comment (should still be valid structurally)
		err = models.ValidateComment(spamComment)
		if err != nil {
			t.Errorf("Spam comment failed structural validation: %v", err)
		}

		// Test spam detection
		spamDetection = models.DetectSpam(spamComment)
		if !spamDetection.IsSpam {
			t.Error("Obvious spam comment was not flagged as spam")
		}

		if spamDetection.Score < 0.5 {
			t.Errorf("Spam comment has low spam score: %f", spamDetection.Score)
		}

		// Test 3: Comment with threading
		replyComment := &models.Comment{
			ArticleID:   1,
			ParentID:    &comment.ID,
			Content:     "I agree with your comment!",
			AuthorName:  "Jane Doe",
			AuthorEmail: "jane@example.com",
			Status:      models.CommentStatusPending,
		}

		// Set comment ID for parent validation
		comment.ID = 1
		replyComment.ParentID = &comment.ID

		err = models.ValidateComment(replyComment)
		if err != nil {
			t.Errorf("Reply comment failed validation: %v", err)
		}

		// Test 4: Invalid comment (self-referencing parent)
		invalidComment := &models.Comment{
			ID:          2,
			ArticleID:   1,
			ParentID:    func() *uint64 { id := uint64(2); return &id }(),
			Content:     "Invalid comment",
			AuthorName:  "Test User",
			AuthorEmail: "test@example.com",
			Status:      models.CommentStatusPending,
		}

		err = models.ValidateComment(invalidComment)
		if err == nil {
			t.Error("Self-referencing comment should fail validation")
		}
	})

	t.Run("comment status workflow", func(t *testing.T) {
		comment := &models.Comment{
			ArticleID:   1,
			Content:     "Test comment",
			AuthorName:  "Test User",
			AuthorEmail: "test@example.com",
			Status:      models.CommentStatusPending,
		}

		// Test initial state
		if comment.IsApproved() {
			t.Error("Pending comment should not be approved")
		}

		if comment.CanBeRepliedTo() {
			t.Error("Pending comment should not allow replies")
		}

		// Test approved state
		comment.Status = models.CommentStatusApproved
		if !comment.IsApproved() {
			t.Error("Approved comment should be approved")
		}

		if !comment.CanBeRepliedTo() {
			t.Error("Approved comment should allow replies")
		}

		// Test spam state
		comment.Status = models.CommentStatusSpam
		if !comment.IsSpam() {
			t.Error("Spam comment should be marked as spam")
		}

		if comment.CanBeRepliedTo() {
			t.Error("Spam comment should not allow replies")
		}
	})

	t.Run("comment content sanitization", func(t *testing.T) {
		comment := &models.Comment{
			ArticleID:   1,
			Content:     "  This is a <script>alert('xss')</script> comment with   extra   spaces  ",
			AuthorName:  "  Test User  ",
			AuthorEmail: "  TEST@EXAMPLE.COM  ",
			Status:      models.CommentStatusPending,
		}

		// Test sanitization
		comment.SanitizeContent()
		comment.PrepareForDB()

		expectedContent := "This is a &lt;script&gt;alert('xss')&lt;/script&gt; comment with   extra   spaces"
		if comment.Content != expectedContent {
			t.Errorf("Content not sanitized correctly. Expected: %q, Got: %q", expectedContent, comment.Content)
		}

		if comment.AuthorName != "Test User" {
			t.Errorf("Author name not trimmed correctly. Expected: %q, Got: %q", "Test User", comment.AuthorName)
		}

		if comment.AuthorEmail != "test@example.com" {
			t.Errorf("Author email not normalized correctly. Expected: %q, Got: %q", "test@example.com", comment.AuthorEmail)
		}

		if comment.Status != models.CommentStatusPending {
			t.Errorf("Status not set correctly. Expected: %q, Got: %q", models.CommentStatusPending, comment.Status)
		}
	})
}