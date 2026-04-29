# Migration 049: Content Ingestion Enhancements

## Overview
This migration adds support for enhanced content ingestion features including featured images, focus keywords, and automatic partition management for article_tags.

## Changes

### New Tables
1. **images** - Stores article images including featured images
   - `id` - Primary key
   - `original_url` - Original URL of the image
   - `filename` - Stored filename
   - `alt_text` - Alternative text for accessibility
   - `caption` - Image caption
   - `width`, `height` - Image dimensions
   - `file_size` - File size in bytes
   - `mime_type` - MIME type (image/jpeg, image/png, etc.)
   - `article_id` - Associated article (optional)
   - `created_at`, `updated_at` - Timestamps

2. **image_variants** - Stores responsive image variants (for future use)
   - `id` - Primary key
   - `image_id` - Reference to parent image
   - `variant_name` - Variant name (thumbnail, medium, large, etc.)
   - `filename` - Variant filename
   - `width`, `height` - Variant dimensions
   - `file_size` - Variant file size
   - `created_at` - Timestamp

### New Columns in articles table
1. **featured_image_id** (BIGINT) - Reference to the featured image
2. **focus_keyword** (VARCHAR(100)) - Primary SEO keyword for the article
3. **last_moderated_by** (BIGINT) - User who last moderated the article

### New Functions
1. **create_article_tags_daily_partitions()** - Automatically creates daily partitions for article_tags table
2. **partition_maintenance()** - Updated to include article_tags partition creation

### Indexes
- `idx_images_article_id` - For querying images by article
- `idx_images_created_at` - For time-based queries
- `idx_articles_featured_image` - For articles with featured images
- `idx_articles_focus_keyword` - For SEO keyword queries
- `idx_image_variants_image_id` - For querying image variants

### Foreign Keys
- `articles.featured_image_id` → `images.id`
- `image_variants.image_id` → `images.id` (CASCADE DELETE)

## API Support
The Content Ingestion API now supports these additional fields:
- `featured_image_url` - URL of the featured image (will be downloaded and stored)
- `meta_title` - Custom SEO title
- `meta_description` - Custom SEO description
- `canonical_url` - Canonical URL for SEO
- `focus_keyword` - Primary SEO keyword
- `enable_auto_linking` - Enable automatic internal linking
- `language_code` - Language code (default: "fa")

## Running the Migration

### On existing installations:
```bash
# The migration is idempotent and safe to run on existing databases
sudo -u newsapp psql -d newsdb -f /home/newsapp/news-website/migrations/049_add_content_ingestion_enhancements.up.sql
```

### On new installations:
The migration will be automatically applied during the initial setup.

## Rollback
To rollback this migration:
```bash
sudo -u newsapp psql -d newsdb -f /home/newsapp/news-website/migrations/049_add_content_ingestion_enhancements.down.sql
```

## Notes
- The migration uses `IF NOT EXISTS` checks to be idempotent
- Existing data is not affected
- The migration adds indexes for optimal query performance
- Partition management is now automated for article_tags table
- The images table supports future enhancements like image variants and responsive images

## Related Files
- `internal/models/content_source.go` - Updated ContentIngestionRequest model
- `internal/services/content_ingestion_service.go` - Updated to handle new fields
- `internal/models/article.go` - Article model with new fields
- `web/static/js/admin/content-ingestion.js` - Updated API usage examples
