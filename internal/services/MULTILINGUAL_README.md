# Multilingual Support System

This document describes the implementation of the multilingual support system for the high-performance news website.

## Overview

The multilingual system provides comprehensive support for multiple languages with the following features:

- **Default Language**: Persian (fa) - RTL
- **Optional Languages**: English (en) - LTR, Arabic (ar) - RTL
- **Translation Groups**: Link related content across languages
- **Language-aware URL routing**: Automatic language detection and routing
- **RTL/LTR Layout Support**: Proper CSS and template handling
- **Fallback Mechanism**: Content fallback to default language when translations are missing

## Architecture

### Database Schema

#### Languages Table
```sql
CREATE TABLE languages (
    code VARCHAR(5) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    native_name VARCHAR(50) NOT NULL,
    direction VARCHAR(3) NOT NULL CHECK (direction IN ('ltr', 'rtl')),
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Translation Groups
```sql
CREATE TABLE translation_groups (
    id BIGSERIAL PRIMARY KEY,
    group_type VARCHAR(20) NOT NULL CHECK (group_type IN ('article', 'category', 'tag')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Content Tables Extensions
All content tables (articles, categories, tags) have been extended with:
- `language_code VARCHAR(5)` - References languages.code
- `translation_group_id BIGINT` - References translation_groups.id

### Models

#### Core Models
- `Language`: Represents a supported language
- `TranslationGroup`: Groups related translations
- `Translation`: Individual translation information
- `Translations`: Slice of translations with JSON scanning support

#### Multilingual Content Models
- `MultilingualArticle`: Article with translation information
- `MultilingualCategory`: Category with translation information
- `MultilingualTag`: Tag with translation information

#### Configuration
- `LanguageConfig`: System language configuration
- `LanguageRouteInfo`: URL routing information for languages

### Services

#### MultilingualService
Main service providing:
- Language management
- Translation group creation and management
- Language-aware content retrieval
- URL routing information generation
- Language validation

Key methods:
```go
func (s *MultilingualService) GetLanguages() ([]models.Language, error)
func (s *MultilingualService) CreateTranslationGroup(groupType string, contentIDs []uint64) (uint64, error)
func (s *MultilingualService) GetArticlesByLanguage(languageCode, fallbackLanguage string, limit, offset int) ([]models.MultilingualArticle, error)
func (s *MultilingualService) GenerateLanguageRouteInfo(contentType, slug, languageCode string) (*models.LanguageRouteInfo, error)
```

## URL Routing

### URL Structure
- Default language (Persian): `/articles/article-slug`
- Other languages: `/en/articles/article-slug`, `/ar/articles/article-slug`

### Language Detection
1. Extract language code from URL path
2. Validate against active languages
3. Fall back to default language if invalid
4. Store in request context for use throughout the request

### Middleware
- `MultilingualMiddleware`: Extracts and validates language from URL
- `LanguageRedirectMiddleware`: Redirects based on Accept-Language header
- `SetLanguageHeaders`: Sets appropriate HTTP headers

## Frontend Support

### CSS Framework
- `multilingual.css`: Base styles for both RTL and LTR
- Direction-specific styling using `[dir="rtl"]` and `[dir="ltr"]` selectors
- Language-specific font loading
- Responsive design with multilingual considerations

### JavaScript
- `multilingual.js`: Client-side multilingual functionality
- Language switcher with keyboard navigation
- RTL/LTR enhancements
- Number and date formatting per locale
- Accessibility improvements

### Templates
- `multilingual_base.html`: Base template with language support
- Automatic language detection and alternate URL generation
- Proper meta tags and structured data
- Language-specific content and navigation

## API Endpoints

### Language Management
- `GET /api/v1/languages` - Get all languages
- `GET /api/v1/languages/active` - Get active languages
- `GET /api/v1/languages/config` - Get language configuration
- `GET /api/v1/languages/:code/validate` - Validate language code

### Translation Management
- `POST /api/v1/translations/groups` - Create translation group

### Content Retrieval
- `GET /api/v1/articles/:id/translations` - Get article with translations
- `GET /api/v1/articles/language/:lang` - Get articles by language
- `GET /api/v1/categories/language/:lang` - Get categories by language
- `GET /api/v1/tags/language/:lang` - Get tags by language

### Routing Information
- `GET /api/v1/routes/:type/:slug/language/:lang` - Get language route info

## Usage Examples

### Creating Translation Groups

```go
// Create translation group for articles
contentIDs := []uint64{1, 2, 3} // English, Persian, Arabic article IDs
groupID, err := multilingualService.CreateTranslationGroup("article", contentIDs)
```

### Retrieving Content by Language

```go
// Get Persian articles with English fallback
articles, err := multilingualService.GetArticlesByLanguage("fa", "en", 20, 0)

// Get article with all translations
article, err := multilingualService.GetArticleTranslations(articleID)
```

### URL Generation

```go
// Generate route information for different languages
routeInfo, err := multilingualService.GenerateLanguageRouteInfo("articles", "my-article", "en")
// Returns: URLPrefix="/en", AlternateURLs={"fa": "/articles/my-article", "ar": "/ar/articles/my-article"}
```

### Frontend Usage

```html
<!-- Language switcher -->
<div class="language-switcher">
    <button class="language-toggle">{{.CurrentLanguage}}</button>
    <div class="language-dropdown">
        {{range .AvailableLanguages}}
        <a href="{{.URL}}" hreflang="{{.Code}}">{{.NativeName}}</a>
        {{end}}
    </div>
</div>

<!-- RTL/LTR content -->
<div dir="{{.LanguageDirection}}" lang="{{.LanguageCode}}">
    <h1>{{.Title}}</h1>
    <p>{{.Content}}</p>
</div>
```

## Performance Considerations

### Database Optimization
- Language-aware indexes on all content tables
- Efficient translation group queries using JOINs
- Partitioned tables maintain language-specific indexes

### Caching Strategy
- Language-specific cache keys
- Separate cache entries for each language version
- Translation group information cached separately

### Query Optimization
- Prepared statements for common language queries
- Efficient fallback queries using CTEs
- Minimal data transfer with selective field loading

## Testing

### Unit Tests
- Model validation and serialization
- Service method functionality
- URL extraction and routing logic

### Integration Tests
- Complete multilingual workflow
- Translation group management
- Performance with large datasets
- Language validation and fallback

### API Tests
- All multilingual endpoints
- Error handling and validation
- Middleware functionality

## Migration Guide

### Database Migration
1. Run migration `005_multilingual_support.up.sql`
2. Verify language data is inserted correctly
3. Update existing content with default language codes
4. Create translation groups for existing multilingual content

### Code Integration
1. Update models to include language fields
2. Modify repositories to handle language-aware queries
3. Add multilingual middleware to routing
4. Update templates with language support
5. Include multilingual CSS and JavaScript

### Content Migration
```sql
-- Update existing content to default language
UPDATE articles SET language_code = 'fa' WHERE language_code IS NULL;
UPDATE categories SET language_code = 'fa' WHERE language_code IS NULL;
UPDATE tags SET language_code = 'fa' WHERE language_code IS NULL;
```

## Configuration

### Environment Variables
```bash
# Default language (Persian)
DEFAULT_LANGUAGE=fa

# Active languages (comma-separated)
ACTIVE_LANGUAGES=fa,en,ar

# Enable language detection from Accept-Language header
ENABLE_LANGUAGE_DETECTION=true
```

### Language Activation
To activate additional languages:

```sql
-- Enable English
UPDATE languages SET is_active = true WHERE code = 'en';

-- Enable Arabic
UPDATE languages SET is_active = true WHERE code = 'ar';
```

## Best Practices

### Content Creation
1. Always specify language code when creating content
2. Create translation groups for related content
3. Use consistent slug patterns across languages
4. Maintain SEO metadata in each language

### URL Design
1. Use language prefixes for non-default languages
2. Keep URLs clean and SEO-friendly
3. Implement proper canonical URLs
4. Use hreflang tags for search engines

### Performance
1. Cache language-specific content separately
2. Use database indexes effectively
3. Minimize translation queries
4. Implement efficient fallback mechanisms

### Accessibility
1. Set proper lang attributes on HTML elements
2. Use appropriate fonts for each language
3. Implement keyboard navigation for language switcher
4. Provide clear language indicators

## Troubleshooting

### Common Issues

1. **Language not detected from URL**
   - Check middleware order
   - Verify language code format (2 characters)
   - Ensure language is active in database

2. **Translation groups not working**
   - Verify all content has same translation_group_id
   - Check that content exists in database
   - Ensure proper foreign key relationships

3. **RTL layout issues**
   - Check CSS direction attributes
   - Verify template dir attribute
   - Ensure proper font loading

4. **Performance issues**
   - Check database indexes
   - Verify cache configuration
   - Monitor query performance

### Debug Commands

```sql
-- Check language configuration
SELECT * FROM languages ORDER BY sort_order;

-- Check translation groups
SELECT tg.id, tg.group_type, COUNT(*) as content_count
FROM translation_groups tg
JOIN articles a ON a.translation_group_id = tg.id
GROUP BY tg.id, tg.group_type;

-- Check content distribution by language
SELECT language_code, COUNT(*) as count
FROM articles
WHERE status = 'published'
GROUP BY language_code;
```

## Future Enhancements

1. **Dynamic Language Management**: Admin interface for adding/removing languages
2. **Translation Workflow**: Editorial workflow for managing translations
3. **Auto-translation Integration**: Integration with translation services
4. **Language Analytics**: Detailed analytics per language
5. **SEO Enhancements**: Advanced multilingual SEO features
6. **Content Synchronization**: Tools for keeping translations in sync