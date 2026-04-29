# Models Package

This package contains the core data models and validation logic for the high-performance news website.

## Overview

The models package provides:
- Core data structures (Article, User, Category, Tag)
- Comprehensive validation functions
- Error handling types
- Database preparation utilities
- SEO data management

## Models

### Article
Represents a news article with comprehensive metadata including:
- Basic content (title, content, excerpt)
- Author and category relationships
- Tag associations
- Publication status and timestamps
- View/engagement counters
- SEO metadata

```go
article := &Article{
    Title:      "Sample Article",
    Content:    "Article content...",
    AuthorID:   1,
    CategoryID: 1,
    Status:     "published",
}

if err := ValidateArticle(article); err != nil {
    // Handle validation error
}
```

### User
Represents system users with role-based access control:
- Basic profile information
- Role-based permissions (Admin, Editor, Reporter, Contributor)
- Authentication data
- Profile customization

```go
user := &User{
    Username: "johndoe",
    Email:    "john@example.com",
    Role:     RoleReporter,
}

if err := ValidateUser(user); err != nil {
    // Handle validation error
}
```

### Category
Represents content categories with hierarchical support:
- Hierarchical structure (parent/child relationships)
- SEO-friendly slugs
- Sorting and organization

```go
category := &Category{
    Name:        "Technology",
    Description: "Tech articles",
    ParentID:    nil, // Root category
}

if err := ValidateCategory(category); err != nil {
    // Handle validation error
}
```

### Tag
Represents content tags with keyword banks for auto-linking:
- Keyword management for auto-linking
- Color coding
- SEO optimization

```go
tag := &Tag{
    Name:     "Programming",
    Keywords: []string{"programming", "code", "development"},
    Color:    "#FF0000",
}

if err := ValidateTag(tag); err != nil {
    // Handle validation error
}
```

## Validation

All models include comprehensive validation:

### Article Validation
- Title: Required, max 255 characters
- Content: Required
- Excerpt: Max 500 characters
- Author/Category: Required IDs
- Status: Must be "draft", "published", or "archived"
- SEO data: Validates meta tags, URLs, schema types

### User Validation
- Username: Required, 3-50 characters, alphanumeric + underscore
- Email: Required, valid email format
- Role: Must be valid role type
- Password: Strong password requirements (when set)

### Category Validation
- Name: Required, max 100 characters
- Description: Max 500 characters
- Hierarchy: Prevents circular references

### Tag Validation
- Name: Required, max 100 characters
- Keywords: Unique, valid characters only
- Color: Valid hex color format

## Error Handling

The package provides custom error types:

```go
// Validation errors with field details
type ValidationError struct {
    Message string
    Fields  []string
}

// Resource not found
type NotFoundError struct {
    Resource string
    ID       string
}

// Duplicate resource
type DuplicateError struct {
    Resource string
    Field    string
    Value    string
}

// Authorization errors
type UnauthorizedError struct {
    Action   string
    Resource string
}
```

## Database Preparation

All models include `PrepareForDB()` methods that:
- Sanitize and normalize data
- Generate slugs automatically
- Set default values
- Prepare for database insertion

```go
article.PrepareForDB()
// - Sanitizes title
// - Generates slug if empty
// - Sets default status and schema type
```

## SEO Features

### Slug Generation
Automatic URL-friendly slug generation:
```go
slug := GenerateSlug("Hello World!") // Returns: "hello-world"
```

### SEO Data
Comprehensive SEO metadata:
```go
seoData := SEOData{
    MetaTitle:       "Article Title",
    MetaDescription: "Article description",
    Keywords:        []string{"keyword1", "keyword2"},
    CanonicalURL:    "https://example.com/article",
    SchemaType:      "NewsArticle",
}
```

## Usage Examples

### Creating and Validating an Article
```go
article := &Article{
    Title:      "Introduction to Go",
    Content:    "Go is a programming language...",
    AuthorID:   1,
    CategoryID: 1,
    Status:     "draft",
    SEOData: SEOData{
        MetaTitle:       "Go Programming Guide",
        MetaDescription: "Learn Go programming",
        SchemaType:      "Article",
    },
}

// Validate the article
if err := ValidateArticle(article); err != nil {
    if validationErr, ok := err.(*ValidationError); ok {
        for _, field := range validationErr.Fields {
            fmt.Printf("Validation error: %s\n", field)
        }
    }
    return err
}

// Prepare for database insertion
article.PrepareForDB()
```

### User Permission Checking
```go
user := &User{Role: RoleEditor}

if user.HasPermission("publish") {
    // User can publish articles
}

if user.CanManageUser(otherUser) {
    // User can manage the other user
}
```

### Category Hierarchy
```go
// Create hierarchy: Technology -> Programming -> Go
tech := &Category{Name: "Technology"}
programming := &Category{Name: "Programming", ParentID: &tech.ID}
golang := &Category{Name: "Go", ParentID: &programming.ID}

// Get full path
path := golang.GetPath() // Returns: "Technology/Programming/Go"
```

### Tag Keyword Management
```go
tag := &Tag{Name: "Programming"}

// Add keywords for auto-linking
tag.AddKeyword("programming")
tag.AddKeyword("code")
tag.AddKeyword("development")

// Check for keywords
if tag.HasKeyword("programming") {
    // Tag has this keyword
}

// Get longest keyword for priority matching
longest := tag.GetLongestKeyword()
```

## Testing

The package includes comprehensive tests:
- Unit tests for all validation functions
- Integration tests for model interactions
- Benchmark tests for performance
- Error handling tests

Run tests:
```bash
go test ./internal/models -v
```

## Performance Considerations

- Validation functions are optimized for high-volume operations
- Slug generation is efficient for batch processing
- Database preparation methods minimize processing overhead
- Error types provide detailed information without performance impact

## Requirements Satisfied

This implementation satisfies the following requirements from the specification:

- **Requirement 1**: Article model supports high-volume content management
- **Requirement 6**: User model with comprehensive role-based access control
- **Requirement 7**: Tag model with keyword bank for auto-linking
- **Requirement 8**: SEO data structure with canonicalization support

The models are designed for ultra-high performance scenarios handling 50,000+ daily articles while maintaining data integrity and comprehensive validation.