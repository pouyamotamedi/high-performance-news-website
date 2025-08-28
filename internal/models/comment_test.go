package models

import (
	"testing"
)

func TestValidateComment(t *testing.T) {
	tests := []struct {
		name    string
		comment *Comment
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid comment",
			comment: &Comment{
				ArticleID:   1,
				Content:     "This is a valid comment",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: false,
		},
		{
			name: "missing article ID",
			comment: &Comment{
				Content:     "This is a comment",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "article_id is required",
		},
		{
			name: "empty content",
			comment: &Comment{
				ArticleID:   1,
				Content:     "",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "content is required",
		},
		{
			name: "content too long",
			comment: &Comment{
				ArticleID:   1,
				Content:     string(make([]byte, 2001)), // 2001 characters
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "content must be less than 2000 characters",
		},
		{
			name: "empty author name",
			comment: &Comment{
				ArticleID:   1,
				Content:     "This is a comment",
				AuthorName:  "",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "author_name is required",
		},
		{
			name: "author name too long",
			comment: &Comment{
				ArticleID:   1,
				Content:     "This is a comment",
				AuthorName:  string(make([]byte, 101)), // 101 characters
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "author_name must be less than 100 characters",
		},
		{
			name: "invalid email",
			comment: &Comment{
				ArticleID:   1,
				Content:     "This is a comment",
				AuthorName:  "John Doe",
				AuthorEmail: "invalid-email",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "author_email must be a valid email address",
		},
		{
			name: "invalid status",
			comment: &Comment{
				ArticleID:   1,
				Content:     "This is a comment",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      "invalid",
			},
			wantErr: true,
			errMsg:  "status must be one of: pending, approved, rejected, spam",
		},
		{
			name: "parent ID same as comment ID",
			comment: &Comment{
				ID:          1,
				ArticleID:   1,
				ParentID:    func() *uint64 { id := uint64(1); return &id }(),
				Content:     "This is a comment",
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				Status:      CommentStatusPending,
			},
			wantErr: true,
			errMsg:  "parent_id cannot be the same as comment id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComment(tt.comment)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateComment() expected error but got none")
					return
				}
				
				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("ValidateComment() expected ValidationError but got %T", err)
					return
				}
				
				found := false
				for _, field := range validationErr.Fields {
					if field == tt.errMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateComment() expected error message '%s' but got %v", tt.errMsg, validationErr.Fields)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateComment() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsValidCommentStatus(t *testing.T) {
	tests := []struct {
		status CommentStatus
		want   bool
	}{
		{CommentStatusPending, true},
		{CommentStatusApproved, true},
		{CommentStatusRejected, true},
		{CommentStatusSpam, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := IsValidCommentStatus(tt.status); got != tt.want {
				t.Errorf("IsValidCommentStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentPrepareForDB(t *testing.T) {
	comment := &Comment{
		Content:     "  This is a comment with extra spaces  ",
		AuthorName:  "  John Doe  ",
		AuthorEmail: "  JOHN@EXAMPLE.COM  ",
		Status:      "",
	}

	comment.PrepareForDB()

	if comment.Content != "This is a comment with extra spaces" {
		t.Errorf("PrepareForDB() content = %q, want %q", comment.Content, "This is a comment with extra spaces")
	}

	if comment.AuthorName != "John Doe" {
		t.Errorf("PrepareForDB() author_name = %q, want %q", comment.AuthorName, "John Doe")
	}

	if comment.AuthorEmail != "john@example.com" {
		t.Errorf("PrepareForDB() author_email = %q, want %q", comment.AuthorEmail, "john@example.com")
	}

	if comment.Status != CommentStatusPending {
		t.Errorf("PrepareForDB() status = %q, want %q", comment.Status, CommentStatusPending)
	}
}

func TestCommentMethods(t *testing.T) {
	t.Run("IsApproved", func(t *testing.T) {
		comment := &Comment{Status: CommentStatusApproved}
		if !comment.IsApproved() {
			t.Error("IsApproved() should return true for approved comment")
		}

		comment.Status = CommentStatusPending
		if comment.IsApproved() {
			t.Error("IsApproved() should return false for pending comment")
		}
	})

	t.Run("IsSpam", func(t *testing.T) {
		comment := &Comment{Status: CommentStatusSpam}
		if !comment.IsSpam() {
			t.Error("IsSpam() should return true for spam comment")
		}

		comment.Status = CommentStatusApproved
		if comment.IsSpam() {
			t.Error("IsSpam() should return false for approved comment")
		}
	})

	t.Run("CanBeRepliedTo", func(t *testing.T) {
		comment := &Comment{Status: CommentStatusApproved}
		if !comment.CanBeRepliedTo() {
			t.Error("CanBeRepliedTo() should return true for approved comment")
		}

		comment.Status = CommentStatusPending
		if comment.CanBeRepliedTo() {
			t.Error("CanBeRepliedTo() should return false for pending comment")
		}
	})
}

func TestSanitizeCommentContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove HTML tags",
			input:    "This is <script>alert('xss')</script> a comment",
			expected: "This is &lt;script&gt;alert('xss')&lt;/script&gt; a comment",
		},
		{
			name:     "normalize whitespace",
			input:    "This is\n\n\na comment\n\n\nwith extra\n\n\nlines",
			expected: "This is\na comment\nwith extra\nlines",
		},
		{
			name:     "remove empty lines",
			input:    "Line 1\n\n\nLine 2\n\n\n\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "trim spaces",
			input:    "  This is a comment  \n  with spaces  ",
			expected: "This is a comment\nwith spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCommentContent(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeCommentContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectSpam(t *testing.T) {
	tests := []struct {
		name     string
		comment  *Comment
		wantSpam bool
		minScore float64
	}{
		{
			name: "normal comment",
			comment: &Comment{
				Content: "This is a normal comment about the article.",
			},
			wantSpam: false,
			minScore: 0.0,
		},
		{
			name: "comment with excessive links",
			comment: &Comment{
				Content: "Check out https://example.com and https://spam.com and https://more-spam.com",
			},
			wantSpam: false, // 3 links, threshold is >2, so score is 0.3
			minScore: 0.3,
		},
		{
			name: "comment with spam keywords",
			comment: &Comment{
				Content: "You are the winner of our lottery! Click here to claim your prize!",
			},
			wantSpam: false, // Contains "winner" and "click here", score should be 0.4
			minScore: 0.4,
		},
		{
			name: "comment with excessive caps",
			comment: &Comment{
				Content: "THIS IS A VERY LOUD COMMENT WITH LOTS OF CAPS!!!",
			},
			wantSpam: false, // High caps ratio, score should be 0.2
			minScore: 0.2,
		},
		{
			name: "very short comment",
			comment: &Comment{
				Content: "ok",
			},
			wantSpam: false, // Very short, score should be 0.1
			minScore: 0.1,
		},
		{
			name: "comment with repetitive characters",
			comment: &Comment{
				Content: "This is sooooo good!!!!! I love it!!!!",
			},
			wantSpam: false, // Repetitive chars, score should be 0.2
			minScore: 0.2,
		},
		{
			name: "high spam score comment",
			comment: &Comment{
				Content: "CONGRATULATIONS! You won the lottery! Visit https://spam.com and https://more-spam.com and https://even-more-spam.com NOW!!!!",
			},
			wantSpam: true, // Multiple factors: caps, spam keywords, excessive links
			minScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection := DetectSpam(tt.comment)
			
			if detection.IsSpam != tt.wantSpam {
				t.Errorf("DetectSpam() IsSpam = %v, want %v", detection.IsSpam, tt.wantSpam)
			}
			
			if detection.Score < tt.minScore {
				t.Errorf("DetectSpam() Score = %f, want at least %f", detection.Score, tt.minScore)
			}
			
			if detection.Confidence != detection.Score {
				t.Errorf("DetectSpam() Confidence = %f, want %f (should equal Score)", detection.Confidence, detection.Score)
			}
		})
	}
}

func TestHasRepetitiveChars(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"normal text", false},
		{"sooooo good", true},
		{"!!!!", true},
		{"abc", false},
		{"aaa", false}, // Only 3 chars, need 4+ for repetitive
		{"aaaa", true},
		{"", false},
		{"ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasRepetitiveChars(tt.input); got != tt.want {
				t.Errorf("hasRepetitiveChars() = %v, want %v", got, tt.want)
			}
		})
	}
}