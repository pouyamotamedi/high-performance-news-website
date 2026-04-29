#!/bin/bash
# Reindex all articles into MeiliSearch
# Usage: ./reindex-search.sh

set -e

# Load environment variables
if [ -f .env.local ]; then
    export $(grep -v '^#' .env.local | xargs)
fi

MEILI_URL="${SEARCH_MEILISEARCH_URL:-http://localhost:7700}"
MEILI_KEY="${SEARCH_MEILISEARCH_API_KEY}"
INDEX_NAME="${SEARCH_INDEX_NAME:-articles}"
DB_HOST="${NEWS_DATABASE_HOST:-localhost}"
DB_PORT="${NEWS_DATABASE_PORT:-5432}"
DB_USER="${NEWS_DATABASE_USER:-newsapp}"
DB_NAME="${NEWS_DATABASE_DBNAME:-newsdb}"

echo "=== MeiliSearch Reindex Script ==="
echo "MeiliSearch URL: $MEILI_URL"
echo "Index Name: $INDEX_NAME"
echo ""

# Check MeiliSearch health
echo "Checking MeiliSearch health..."
HEALTH=$(curl -s "$MEILI_URL/health" | grep -o '"status":"available"' || echo "")
if [ -z "$HEALTH" ]; then
    echo "ERROR: MeiliSearch is not available at $MEILI_URL"
    exit 1
fi
echo "MeiliSearch is healthy"

# Create or update index settings
echo "Configuring index settings..."
curl -s -X PATCH "$MEILI_URL/indexes/$INDEX_NAME/settings" \
    -H "Authorization: Bearer $MEILI_KEY" \
    -H "Content-Type: application/json" \
    --data '{
        "searchableAttributes": ["title", "content", "excerpt", "meta_title", "meta_description", "keywords", "tags"],
        "filterableAttributes": ["author_id", "category_id", "status", "published_at", "language_code", "tags"],
        "sortableAttributes": ["published_at", "created_at", "view_count", "like_count"],
        "rankingRules": ["words", "typo", "proximity", "attribute", "sort", "exactness"],
        "typoTolerance": {
            "enabled": true,
            "minWordSizeForTypos": {
                "oneTypo": 4,
                "twoTypos": 8
            }
        }
    }' > /dev/null

echo "Index settings configured"

# Export articles from PostgreSQL and index them
echo "Exporting articles from database..."

# Create temporary JSON file
TEMP_FILE=$(mktemp)

sudo -u $DB_USER psql -h $DB_HOST -p $DB_PORT -d $DB_NAME -t -A << 'SQL' > "$TEMP_FILE"
SELECT json_agg(row_to_json(t))
FROM (
    SELECT 
        a.id::text as id,
        a.title,
        a.slug,
        SUBSTRING(a.content, 1, 10000) as content,
        a.excerpt,
        COALESCE(
            (SELECT '/static/uploads/images/' || m.filename FROM media m WHERE m.id = a.featured_image_id),
            ''
        ) as featured_image,
        a.author_id,
        a.category_id,
        a.status,
        EXTRACT(EPOCH FROM a.published_at)::bigint as published_at,
        EXTRACT(EPOCH FROM a.created_at)::bigint as created_at,
        a.view_count,
        a.like_count,
        a.language_code,
        COALESCE(a.meta_title, '') as meta_title,
        COALESCE(a.meta_description, '') as meta_description,
        COALESCE(
            (SELECT array_agg(t.name) FROM article_tags at JOIN tags t ON at.tag_id = t.id WHERE at.article_id = a.id),
            ARRAY[]::text[]
        ) as tags
    FROM articles a
    WHERE a.status = 'published'
    ORDER BY a.published_at DESC
) t;
SQL

# Check if we got data
if [ ! -s "$TEMP_FILE" ] || [ "$(cat $TEMP_FILE)" = "null" ]; then
    echo "No articles found to index"
    rm -f "$TEMP_FILE"
    exit 0
fi

ARTICLE_COUNT=$(cat "$TEMP_FILE" | jq 'length')
echo "Found $ARTICLE_COUNT articles to index"

# Index documents
echo "Indexing documents..."
curl -s -X POST "$MEILI_URL/indexes/$INDEX_NAME/documents" \
    -H "Authorization: Bearer $MEILI_KEY" \
    -H "Content-Type: application/json" \
    --data @"$TEMP_FILE" > /dev/null

# Clean up
rm -f "$TEMP_FILE"

# Wait for indexing to complete
echo "Waiting for indexing to complete..."
sleep 5

# Get index stats
STATS=$(curl -s "$MEILI_URL/indexes/$INDEX_NAME/stats" -H "Authorization: Bearer $MEILI_KEY")
INDEXED=$(echo "$STATS" | jq '.numberOfDocuments')

echo ""
echo "=== Reindex Complete ==="
echo "Documents indexed: $INDEXED"
echo ""
