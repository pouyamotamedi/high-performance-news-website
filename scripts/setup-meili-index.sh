#!/bin/bash
# Setup MeiliSearch index settings and index articles
MEILI_KEY="newsapp_meili_master_key_2024"
MEILI_URL="http://localhost:7700"
INDEX="articles"

echo "=== Setting up MeiliSearch index ==="

# Configure index settings
curl -s -X PATCH "$MEILI_URL/indexes/$INDEX/settings" \
  -H "Authorization: Bearer $MEILI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "searchableAttributes": ["title", "content", "excerpt"],
    "filterableAttributes": ["author_id", "category_id", "status", "published_at", "language_code"],
    "sortableAttributes": ["published_at", "created_at", "view_count", "like_count"],
    "rankingRules": ["words", "typo", "proximity", "attribute", "sort", "exactness"]
  }'

echo ""
echo "Index settings configured"

# Export and index articles from PostgreSQL
echo "Exporting articles from database..."

sudo -u newsapp psql -d newsdb -t -A -c "
SELECT json_agg(row_to_json(t))
FROM (
    SELECT 
        a.id::text as id,
        a.title,
        SUBSTRING(a.content, 1, 10000) as content,
        a.excerpt,
        a.author_id,
        a.category_id,
        a.status,
        EXTRACT(EPOCH FROM a.published_at)::bigint as published_at,
        EXTRACT(EPOCH FROM a.created_at)::bigint as created_at,
        a.view_count,
        a.like_count,
        a.language_code
    FROM articles a
    WHERE a.status = 'published'
    ORDER BY a.published_at DESC
) t;" > /tmp/articles.json

# Check if we got data
if [ -s /tmp/articles.json ] && [ "$(cat /tmp/articles.json)" != "null" ]; then
    ARTICLE_COUNT=$(cat /tmp/articles.json | jq 'length')
    echo "Found $ARTICLE_COUNT articles to index"
    
    # Index documents with primary key specified
    curl -s -X POST "$MEILI_URL/indexes/$INDEX/documents?primaryKey=id" \
      -H "Authorization: Bearer $MEILI_KEY" \
      -H "Content-Type: application/json" \
      -d @/tmp/articles.json
    
    echo ""
    echo "Articles indexed"
    
    # Wait and check stats
    sleep 3
    echo "Index stats:"
    curl -s "$MEILI_URL/indexes/$INDEX/stats" -H "Authorization: Bearer $MEILI_KEY" | jq .
else
    echo "No articles found to index"
fi

rm -f /tmp/articles.json
echo "=== Done ==="
