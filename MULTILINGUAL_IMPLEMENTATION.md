# Multilingual System Implementation

## Overview

This document describes the complete multilingual system implementation for the news website. The system supports 5 languages with English as the default:

- **en** (English) - Default language, LTR
- **de** (German) - LTR
- **fr** (French) - LTR
- **es** (Spanish) - LTR
- **ar** (Arabic) - RTL

**Note:** Persian (fa) has been removed as per requirements.

## URL Structure

The system uses a subfolder-based URL strategy:

```
domain.com/          → 301 redirect to /en/
domain.com/en/       → English homepage
domain.com/en/article/slug
domain.com/en/category/slug
domain.com/en/tag/slug
domain.com/de/       → German homepage
domain.com/de/article/slug
...
domain.com/ar/       → Arabic homepage (RTL)
```

## Files Modified/Created

### 1. Database Migration
- `migrations/053_update_languages_multilingual.up.sql` - Adds new languages, removes Persian, updates defaults
- `migrations/053_update_languages_multilingual.down.sql` - Rollback migration

### 2. Configuration
- `configs/config.yaml` - Added multilingual configuration section
- `internal/config/config.go` - Added `MultilingualConfig` struct with helper methods

### 3. Services
- `internal/services/multilingual_service.go` - Updated default language to English, added helper methods:
  - `GenerateAlternateURLs()` - Generates hreflang URLs
  - `GenerateCanonicalURL()` - Generates canonical URLs
  - `GetAvailableLanguagesForTemplate()` - Returns language info for templates
  - `GetLanguageByCode()` - Gets language details
  - `GetLanguageNativeName()` / `GetLanguageName()` - Language name helpers

- `internal/services/content_ingestion_service.go` - Changed default language from 'fa' to 'en'

- `internal/services/article_service.go` - Updated `ArticleFilters` struct:
  - Added `LanguageCode` field for filtering articles by language
  - Added `TagID` field (pointer) for filtering articles by tag
  - Fixed `CategoryID` to use pointer type for proper nil checking
  - Updated `List` method to support LanguageCode and TagID filters

### 4. Middleware
- `internal/api/multilingual_middleware.go` - Updated defaults to English

### 5. Server/Routing
- `internal/server/server.go` - Added:
  - Root redirect (`/` → `/en/`)
  - `setupMultilingualRoutes()` - Sets up language-prefixed routes for all 5 languages
  - Multilingual handlers for all page types:
    - `handleMultilingualHomepage` - Language-specific homepage
    - `handleMultilingualArticle` - Article pages with language prefix
    - `handleMultilingualCategory` - Category pages with language prefix
    - `handleMultilingualCategories` - Categories listing
    - `handleMultilingualTag` - Tag pages with language prefix
    - `handleMultilingualTags` - Tags listing
    - `handleMultilingualLatest` - Latest articles
    - `handleMultilingualTrending` - Trending articles
    - `handleMultilingualSearch` - Search page
    - `handleMultilingualAbout` - About page
    - `handleMultilingualContact` - Contact page
  - Sitemap handlers with hreflang support:
    - `handleSitemapIndex` - Main sitemap index
    - `handleLanguageSitemap` - Language-specific sitemaps
  - Helper functions:
    - `getLanguageDirection()` - Returns "ltr" or "rtl"
    - `getLanguageNativeName()` - Returns native language name
    - `getLanguageName()` - Returns English language name
    - `generateAlternateURLs()` - Generates hreflang URLs
    - `generateCanonicalURL()` - Generates canonical URL
    - `getAvailableLanguages()` - Returns language info for switcher
    - `addMultilingualData()` - Adds all multilingual data to template context

### 6. Templates
- `web/templates/layouts/multilingual_base.html` - Updated translations for all 5 languages

## SEO Features

### Canonical URLs
Each page has a canonical URL pointing to its language-specific version:
```html
<link rel="canonical" href="https://domain.com/en/article/slug">
```

### Hreflang Tags
All pages include hreflang tags for all language versions:
```html
<link rel="alternate" hreflang="en" href="https://domain.com/en/article/slug">
<link rel="alternate" hreflang="de" href="https://domain.com/de/article/slug">
<link rel="alternate" hreflang="fr" href="https://domain.com/fr/article/slug">
<link rel="alternate" hreflang="es" href="https://domain.com/es/article/slug">
<link rel="alternate" hreflang="ar" href="https://domain.com/ar/article/slug">
<link rel="alternate" hreflang="x-default" href="https://domain.com/en/article/slug">
```

### Multilingual Sitemaps
- `/sitemap.xml` - Sitemap index pointing to language-specific sitemaps
- `/sitemap-en.xml` - English sitemap with hreflang
- `/sitemap-de.xml` - German sitemap with hreflang
- `/sitemap-fr.xml` - French sitemap with hreflang
- `/sitemap-es.xml` - Spanish sitemap with hreflang
- `/sitemap-ar.xml` - Arabic sitemap with hreflang

Each URL in the sitemaps includes hreflang links to all language versions.

## RTL Support

Arabic (ar) is configured as RTL:
- `dir="rtl"` attribute on HTML element
- RTL-specific CSS loaded (`/static/css/rtl.css`)
- Language direction passed to templates via `LanguageDirection`

## Template Data

All multilingual handlers pass the following data to templates:

```go
data["LanguageCode"] = "en"           // Current language code
data["LanguageDirection"] = "ltr"     // "ltr" or "rtl"
data["LanguageName"] = "English"      // English name
data["LanguageNativeName"] = "English" // Native name
data["AlternateURLs"] = [...]         // Array of alternate language URLs
data["AvailableLanguages"] = [...]    // Array of all languages for switcher
data["CanonicalURL"] = "..."          // Canonical URL for this page
data["IsRTL"] = false                 // Boolean for RTL check
data["BaseURL"] = "https://..."       // Base URL from config
```

## Running the Migration

To apply the migration:

```bash
# Using migrate tool
migrate -path migrations -database "postgres://user:pass@host:port/dbname?sslmode=disable" up

# Or using the built-in migrate command
go run cmd/migrate/main.go up
```

## Configuration

Add to `configs/config.yaml`:

```yaml
multilingual:
  enabled: true
  default_language: "en"
  fallback_language: "en"
  active_languages:
    - "en"
    - "de"
    - "fr"
    - "es"
    - "ar"
  rtl_languages:
    - "ar"
  url_strategy: "subfolder"
  redirect_root: true
```

## Testing

1. **Root Redirect**: Visit `/` and verify redirect to `/en/`
2. **Language Pages**: Visit `/en/`, `/de/`, `/fr/`, `/es/`, `/ar/`
3. **Article Pages**: Visit `/{lang}/article/{slug}`
4. **Category Pages**: Visit `/{lang}/category/{slug}`
5. **Tag Pages**: Visit `/{lang}/tag/{slug}`
6. **Sitemaps**: Check `/sitemap.xml` and `/sitemap-{lang}.xml`
7. **RTL**: Verify Arabic pages have RTL layout
8. **Language Switcher**: Test switching between languages

## Future Enhancements

1. **Content Translation**: Implement translation groups for articles
2. **Admin Panel**: Add language selector in article editor
3. **Translation API**: Add endpoints for managing translations
4. **Language Detection**: Auto-detect user's preferred language
5. **Translation Memory**: Store and reuse translations
