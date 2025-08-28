package models

import (
	"testing"
)

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid category",
			category: Category{
				Name:        "Technology",
				Description: "Technology related articles",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			category: Category{
				Description: "Technology related articles",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "name too long",
			category: Category{
				Name: string(make([]byte, 101)), // 101 characters
			},
			wantErr: true,
			errMsg:  "name must be less than 100 characters",
		},
		{
			name: "description too long",
			category: Category{
				Name:        "Technology",
				Description: string(make([]byte, 501)), // 501 characters
			},
			wantErr: true,
			errMsg:  "description must be less than 500 characters",
		},
		{
			name: "self-referencing parent",
			category: Category{
				ID:       1,
				Name:     "Technology",
				ParentID: func() *uint64 { id := uint64(1); return &id }(),
			},
			wantErr: true,
			errMsg:  "category cannot be its own parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCategory(&tt.category)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCategory() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestCategoryPrepareForDB(t *testing.T) {
	category := &Category{
		Name:        "  Technology  ",
		Description: "  Tech articles  ",
	}

	category.PrepareForDB()

	if category.Name != "Technology" {
		t.Errorf("PrepareForDB() name = %v, want %v", category.Name, "Technology")
	}
	if category.Description != "Tech articles" {
		t.Errorf("PrepareForDB() description = %v, want %v", category.Description, "Tech articles")
	}
	if category.Slug == "" {
		t.Errorf("PrepareForDB() should generate slug")
	}
}

func TestCategoryIsRoot(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     bool
	}{
		{
			name: "root category",
			category: Category{
				Name: "Technology",
			},
			want: true,
		},
		{
			name: "child category",
			category: Category{
				Name:     "Programming",
				ParentID: func() *uint64 { id := uint64(1); return &id }(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.IsRoot()
			if got != tt.want {
				t.Errorf("Category.IsRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategoryGetPath(t *testing.T) {
	// Create a hierarchy: Technology -> Programming -> Go
	tech := &Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	programming := &Category{
		ID:       2,
		Name:     "Programming",
		Slug:     "programming",
		ParentID: &tech.ID,
		Parent:   tech,
	}

	golang := &Category{
		ID:       3,
		Name:     "Go",
		Slug:     "go",
		ParentID: &programming.ID,
		Parent:   programming,
	}

	tests := []struct {
		name     string
		category *Category
		want     string
	}{
		{
			name:     "root category path",
			category: tech,
			want:     "Technology",
		},
		{
			name:     "child category path",
			category: programming,
			want:     "Technology/Programming",
		},
		{
			name:     "grandchild category path",
			category: golang,
			want:     "Technology/Programming/Go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.GetPath()
			if got != tt.want {
				t.Errorf("Category.GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategoryGetSlugPath(t *testing.T) {
	// Create a hierarchy: Technology -> Programming -> Go
	tech := &Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	programming := &Category{
		ID:       2,
		Name:     "Programming",
		Slug:     "programming",
		ParentID: &tech.ID,
		Parent:   tech,
	}

	golang := &Category{
		ID:       3,
		Name:     "Go",
		Slug:     "go",
		ParentID: &programming.ID,
		Parent:   programming,
	}

	tests := []struct {
		name     string
		category *Category
		want     string
	}{
		{
			name:     "root category slug path",
			category: tech,
			want:     "technology",
		},
		{
			name:     "child category slug path",
			category: programming,
			want:     "technology/programming",
		},
		{
			name:     "grandchild category slug path",
			category: golang,
			want:     "technology/programming/go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.GetSlugPath()
			if got != tt.want {
				t.Errorf("Category.GetSlugPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategoryHasChildren(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     bool
	}{
		{
			name: "category with children",
			category: Category{
				Name: "Technology",
				Children: []Category{
					{Name: "Programming"},
				},
			},
			want: true,
		},
		{
			name: "category without children",
			category: Category{
				Name: "Technology",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.HasChildren()
			if got != tt.want {
				t.Errorf("Category.HasChildren() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategoryGetDepth(t *testing.T) {
	// Create a hierarchy: Technology -> Programming -> Go
	tech := &Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}

	programming := &Category{
		ID:       2,
		Name:     "Programming",
		Slug:     "programming",
		ParentID: &tech.ID,
		Parent:   tech,
	}

	golang := &Category{
		ID:       3,
		Name:     "Go",
		Slug:     "go",
		ParentID: &programming.ID,
		Parent:   programming,
	}

	tests := []struct {
		name     string
		category *Category
		want     int
	}{
		{
			name:     "root category depth",
			category: tech,
			want:     0,
		},
		{
			name:     "child category depth",
			category: programming,
			want:     1,
		},
		{
			name:     "grandchild category depth",
			category: golang,
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.GetDepth()
			if got != tt.want {
				t.Errorf("Category.GetDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHierarchy(t *testing.T) {
	tests := []struct {
		name       string
		categories []Category
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid hierarchy",
			categories: []Category{
				{ID: 1, Name: "Technology"},
				{ID: 2, Name: "Programming", ParentID: func() *uint64 { id := uint64(1); return &id }()},
			},
			wantErr: false,
		},
		{
			name: "invalid parent reference",
			categories: []Category{
				{ID: 1, Name: "Technology"},
				{ID: 2, Name: "Programming", ParentID: func() *uint64 { id := uint64(99); return &id }()},
			},
			wantErr: true,
			errMsg:  "parent category with ID 99 does not exist",
		},
		{
			name: "circular reference",
			categories: []Category{
				{ID: 1, Name: "Technology", ParentID: func() *uint64 { id := uint64(2); return &id }()},
				{ID: 2, Name: "Programming", ParentID: func() *uint64 { id := uint64(1); return &id }()},
			},
			wantErr: true,
			errMsg:  "circular reference detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHierarchy(tt.categories)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHierarchy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateHierarchy() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}