package models

import (
	"testing"
)

func TestValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     Tag
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tag",
			tag: Tag{
				Name:        "Programming",
				Description: "Programming related articles",
				Keywords:    []string{"programming", "code", "development"},
				Color:       "#FF0000",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tag: Tag{
				Description: "Programming related articles",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "name too long",
			tag: Tag{
				Name: string(make([]byte, 101)), // 101 characters
			},
			wantErr: true,
			errMsg:  "name must be less than 100 characters",
		},
		{
			name: "description too long",
			tag: Tag{
				Name:        "Programming",
				Description: string(make([]byte, 501)), // 501 characters
			},
			wantErr: true,
			errMsg:  "description must be less than 500 characters",
		},
		{
			name: "invalid color format",
			tag: Tag{
				Name:  "Programming",
				Color: "red",
			},
			wantErr: true,
			errMsg:  "color must be a valid hex color",
		},
		{
			name: "invalid keywords",
			tag: Tag{
				Name:     "Programming",
				Keywords: []string{"programming", "programming"}, // duplicate
			},
			wantErr: true,
			errMsg:  "duplicate keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTag(&tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateTag() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateKeywords(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid keywords",
			keywords: []string{"programming", "code", "development"},
			wantErr:  false,
		},
		{
			name:     "duplicate keywords",
			keywords: []string{"programming", "Programming"},
			wantErr:  true,
			errMsg:   "duplicate keyword",
		},
		{
			name:     "keyword too long",
			keywords: []string{string(make([]byte, 101))}, // 101 characters
			wantErr:  true,
			errMsg:   "must be less than 100 characters",
		},
		{
			name:     "invalid keyword characters",
			keywords: []string{"programming@code"},
			wantErr:  true,
			errMsg:   "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeywords(tt.keywords)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeywords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateKeywords() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		{
			name:  "valid hex color uppercase",
			color: "#FF0000",
			want:  true,
		},
		{
			name:  "valid hex color lowercase",
			color: "#ff0000",
			want:  true,
		},
		{
			name:  "valid hex color mixed case",
			color: "#Ff0000",
			want:  true,
		},
		{
			name:  "invalid color without hash",
			color: "FF0000",
			want:  false,
		},
		{
			name:  "invalid color too short",
			color: "#FF00",
			want:  false,
		},
		{
			name:  "invalid color too long",
			color: "#FF000000",
			want:  false,
		},
		{
			name:  "invalid color with invalid characters",
			color: "#GG0000",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("IsValidHexColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		want    bool
	}{
		{
			name:    "valid keyword",
			keyword: "programming",
			want:    true,
		},
		{
			name:    "valid keyword with spaces",
			keyword: "web development",
			want:    true,
		},
		{
			name:    "valid keyword with hyphens",
			keyword: "machine-learning",
			want:    true,
		},
		{
			name:    "valid keyword with apostrophe",
			keyword: "don't",
			want:    true,
		},
		{
			name:    "invalid keyword with special characters",
			keyword: "programming@code",
			want:    false,
		},
		{
			name:    "empty keyword",
			keyword: "",
			want:    false,
		},
		{
			name:    "keyword with only spaces",
			keyword: "   ",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidKeyword(tt.keyword)
			if got != tt.want {
				t.Errorf("IsValidKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagPrepareForDB(t *testing.T) {
	tag := &Tag{
		Name:        "  Programming  ",
		Description: "  Programming articles  ",
		Keywords:    []string{"  programming  ", "Programming", "code", ""},
		Color:       "",
	}

	tag.PrepareForDB()

	if tag.Name != "Programming" {
		t.Errorf("PrepareForDB() name = %v, want %v", tag.Name, "Programming")
	}
	if tag.Description != "Programming articles" {
		t.Errorf("PrepareForDB() description = %v, want %v", tag.Description, "Programming articles")
	}
	if tag.Color != "#000000" {
		t.Errorf("PrepareForDB() color = %v, want %v", tag.Color, "#000000")
	}
	if tag.Slug == "" {
		t.Errorf("PrepareForDB() should generate slug")
	}
	
	// Check keywords normalization
	expectedKeywords := []string{"programming", "code"}
	if len(tag.Keywords) != len(expectedKeywords) {
		t.Errorf("PrepareForDB() keywords length = %v, want %v", len(tag.Keywords), len(expectedKeywords))
	}
}

func TestNormalizeKeywords(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		want     []string
	}{
		{
			name:     "normal keywords",
			keywords: []string{"programming", "code", "development"},
			want:     []string{"programming", "code", "development"},
		},
		{
			name:     "keywords with duplicates",
			keywords: []string{"programming", "Programming", "code"},
			want:     []string{"programming", "code"},
		},
		{
			name:     "keywords with empty strings",
			keywords: []string{"programming", "", "code", "   "},
			want:     []string{"programming", "code"},
		},
		{
			name:     "keywords with whitespace",
			keywords: []string{"  programming  ", "code", "  development  "},
			want:     []string{"programming", "code", "development"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKeywords(tt.keywords)
			if len(got) != len(tt.want) {
				t.Errorf("NormalizeKeywords() = %v, want %v", got, tt.want)
				return
			}
			for i, keyword := range got {
				if keyword != tt.want[i] {
					t.Errorf("NormalizeKeywords() = %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

func TestTagAddKeyword(t *testing.T) {
	tag := &Tag{
		Keywords: []string{"programming", "code"},
	}

	// Add new keyword
	tag.AddKeyword("development")
	if len(tag.Keywords) != 3 {
		t.Errorf("AddKeyword() keywords length = %v, want %v", len(tag.Keywords), 3)
	}

	// Try to add duplicate keyword (case-insensitive)
	tag.AddKeyword("Programming")
	if len(tag.Keywords) != 3 {
		t.Errorf("AddKeyword() should not add duplicate keyword")
	}

	// Try to add empty keyword
	tag.AddKeyword("")
	if len(tag.Keywords) != 3 {
		t.Errorf("AddKeyword() should not add empty keyword")
	}
}

func TestTagRemoveKeyword(t *testing.T) {
	tag := &Tag{
		Keywords: []string{"programming", "code", "development"},
	}

	// Remove existing keyword
	tag.RemoveKeyword("code")
	if len(tag.Keywords) != 2 {
		t.Errorf("RemoveKeyword() keywords length = %v, want %v", len(tag.Keywords), 2)
	}

	// Try to remove non-existent keyword
	tag.RemoveKeyword("nonexistent")
	if len(tag.Keywords) != 2 {
		t.Errorf("RemoveKeyword() should not change length for non-existent keyword")
	}

	// Remove keyword case-insensitive
	tag.RemoveKeyword("Programming")
	if len(tag.Keywords) != 1 {
		t.Errorf("RemoveKeyword() should be case-insensitive")
	}
}

func TestTagHasKeyword(t *testing.T) {
	tag := &Tag{
		Keywords: []string{"programming", "code", "development"},
	}

	tests := []struct {
		name    string
		keyword string
		want    bool
	}{
		{
			name:    "existing keyword",
			keyword: "programming",
			want:    true,
		},
		{
			name:    "existing keyword different case",
			keyword: "Programming",
			want:    true,
		},
		{
			name:    "non-existent keyword",
			keyword: "testing",
			want:    false,
		},
		{
			name:    "empty keyword",
			keyword: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tag.HasKeyword(tt.keyword)
			if got != tt.want {
				t.Errorf("Tag.HasKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagGetLongestKeyword(t *testing.T) {
	tests := []struct {
		name string
		tag  Tag
		want string
	}{
		{
			name: "keywords with different lengths",
			tag: Tag{
				Keywords: []string{"go", "programming", "web development"},
			},
			want: "web development",
		},
		{
			name: "keywords with same length",
			tag: Tag{
				Keywords: []string{"code", "test"},
			},
			want: "code", // First one found
		},
		{
			name: "no keywords",
			tag: Tag{
				Keywords: []string{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tag.GetLongestKeyword()
			if got != tt.want {
				t.Errorf("Tag.GetLongestKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}