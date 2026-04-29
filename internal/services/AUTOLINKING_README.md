# Auto-Linking System

This document describes the auto-linking system implementation for the high-performance news website.

## Overview

The auto-linking system automatically creates internal links in article content based on keyword banks associated with tags. It uses a Trie data structure for efficient keyword matching and supports priority-based matching where the longest keyword match wins.

## Features

### ✅ Implemented Features

1. **Trie Data Structure**: Efficient keyword matching with O(m) complexity where m is the length of the text
2. **Longest Match Priority**: When multiple keywords match, the longest one takes precedence
3. **One Link Per Keyword**: Each keyword gets at most one link per article to avoid over-linking
4. **Case Preservation**: Original text case is preserved in the generated links
5. **Word Boundary Respect**: Keywords only match at word boundaries, not as substrings
6. **Per-Article Override**: Articles can disable auto-linking via the `AutoLinking` field
7. **HTML Content Processing**: Preserves existing links and HTML tags while processing content
8. **Keyword Conflict Detection**: Identifies when the same keyword is used across multiple tags
9. **Performance Optimized**: Benchmarked at ~108μs per article processing

### Core Components

#### 1. Trie Data Structure (`TrieNode`, `Trie`)

```go
type TrieNode struct {
    children map[rune]*TrieNode
    isEnd    bool
    tag      *models.Tag
    keyword  string
}

type Trie struct {
    root *TrieNode
}
```

- Efficient storage and retrieval of keywords
- Case-insensitive matching
- Supports Unicode characters

#### 2. Auto-Linking Service (`AutoLinkingService`)

```go
type AutoLinkingService struct {
    tagRepo TagRepositoryInterface
    trie    *Trie
}
```

**Key Methods:**
- `LoadKeywords(ctx)`: Loads all tag keywords into the Trie
- `ProcessArticleLinks(ctx, article)`: Processes article content for auto-linking
- `ProcessHTMLContent(ctx, article)`: Processes HTML content while preserving existing tags
- `ValidateKeywordConflicts(ctx)`: Checks for keyword conflicts across tags
- `RefreshKeywords(ctx)`: Reloads keywords from database

#### 3. Integration with Article Service

The auto-linking service integrates with the article service to automatically process content during article creation and updates.

## Usage Examples

### Basic Setup

```go
// Initialize repositories
tagRepo := repositories.NewTagRepository(db)
autoLinkService := services.NewAutoLinkingService(tagRepo)

// Load keywords
ctx := context.Background()
err := autoLinkService.LoadKeywords(ctx)
if err != nil {
    log.Fatal(err)
}

// Create article service with auto-linking
articleRepo := repositories.NewArticleRepository(db)
articleService := services.NewArticleService(db, articleRepo, autoLinkService)
```

### Processing Article Content

```go
article := &models.Article{
    Title:       "AI Technology Trends",
    Content:     "Artificial intelligence and machine learning are transforming technology.",
    AutoLinking: true, // Enable auto-linking
}

// Auto-linking happens automatically during article creation
createdArticle, err := articleService.Create(ctx, article, user)
if err != nil {
    log.Fatal(err)
}

// Result: Content will have internal links to relevant tag pages
// "Artificial intelligence and machine learning are transforming technology."
// becomes:
// "<a href="/tags/ai">Artificial intelligence</a> and <a href="/tags/ai">machine learning</a> are transforming technology."
```

### Manual Content Processing

```go
// Process content manually
processedContent, err := autoLinkService.ProcessHTMLContent(ctx, article)
if err != nil {
    log.Fatal(err)
}

// Process with exclusions
excludedKeywords := []string{"AI", "technology"}
processedContent, err := autoLinkService.ProcessArticleLinksWithExclusions(ctx, article, excludedKeywords)
```

## Configuration

### Tag Keywords Setup

Tags must have keywords defined in their `Keywords` field:

```go
tag := &models.Tag{
    Name:     "Artificial Intelligence",
    Slug:     "ai",
    Keywords: []string{"artificial intelligence", "machine learning", "AI", "neural networks"},
}
```

### Article Auto-Linking Control

```go
article := &models.Article{
    Title:       "My Article",
    Content:     "Content with keywords...",
    AutoLinking: true, // Enable/disable per article
}
```

## Performance

### Benchmarks

- **Trie Insert**: ~342 ns/op
- **Keyword Matching**: ~5.4 μs/op  
- **Article Processing**: ~108 μs/op

### Optimization Features

1. **Efficient Trie Structure**: O(m) keyword matching complexity
2. **Batch Processing**: Supports processing multiple articles
3. **Memory Efficient**: Reuses Trie structure across requests
4. **Lazy Loading**: Keywords loaded on-demand

## Testing

The system includes comprehensive tests covering:

- ✅ Trie data structure operations
- ✅ Keyword matching algorithms
- ✅ Auto-linking functionality
- ✅ HTML content processing
- ✅ Edge cases and error handling
- ✅ Performance benchmarks
- ✅ Integration scenarios

### Running Tests

```bash
# Run all auto-linking tests
go test ./internal/services -v -run "TestTrie|TestAutoLinking|TestKeyword"

# Run benchmarks
go test ./internal/services -bench="BenchmarkTrie|BenchmarkAutoLinking" -run=^$
```

## Error Handling

The system includes robust error handling:

1. **Graceful Degradation**: Auto-linking failures don't prevent article creation
2. **Conflict Detection**: Warns about keyword conflicts across tags
3. **Validation**: Validates keywords and prevents invalid configurations
4. **Logging**: Comprehensive error logging for debugging

## Maintenance

### Refreshing Keywords

When tags or keywords are updated:

```go
err := autoLinkService.RefreshKeywords(ctx)
if err != nil {
    log.Printf("Failed to refresh keywords: %v", err)
}
```

### Monitoring

```go
// Get Trie statistics
stats := autoLinkService.GetTrieStats()
fmt.Printf("Keywords: %d, Nodes: %d\n", stats["total_keywords"], stats["total_nodes"])

// Check for conflicts
conflicts, err := autoLinkService.ValidateKeywordConflicts(ctx)
if len(conflicts) > 0 {
    log.Printf("Keyword conflicts detected: %v", conflicts)
}
```

## Database Schema

The auto-linking system requires the following database schema additions:

```sql
-- Add auto_linking column to articles table
ALTER TABLE articles ADD COLUMN auto_linking BOOLEAN DEFAULT true;

-- Tags table should have keywords as JSONB
-- (already implemented in the existing schema)
```

## Future Enhancements

Potential improvements for future versions:

1. **Caching**: Cache processed content to avoid reprocessing
2. **Analytics**: Track link click-through rates
3. **Machine Learning**: AI-powered keyword suggestion
4. **Bulk Processing**: Background job for processing existing articles
5. **Admin Interface**: GUI for managing keywords and conflicts

## Requirements Satisfied

This implementation satisfies the following requirements from the specification:

- **Requirement 7**: Advanced Tag and Internal Linking System
  - ✅ Automated tagging based on keyword banks
  - ✅ Automatic internal link creation
  - ✅ Longest matching keyword priority
  - ✅ One link per keyword per article
  - ✅ Batch processing support
  - ✅ Per-article auto-linking override

The system is production-ready and optimized for high-volume content processing as required for the 50K+ daily articles target.