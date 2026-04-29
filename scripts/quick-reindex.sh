#!/bin/bash
# Quick reindex script for MeiliSearch

echo "Exporting articles from database..."
sudo -u newsapp psql -d newsdb -t -A -c "SELECT json_agg(row_to_json(t)) FROM (SELECT a.id::text as id, a.title, a.slug, SUBSTRING(a.content, 1, 10000) as content, a.excerpt, a.author_id, a.category_id, a.status, EXTRACT(EPOCH FROM a.published_at)::bigint as published_at, EXTRACT(EPOCH FROM a.created_at)::bigint as created_at, a.view_count, a.like_count, a.language_code FROM articles a WHERE a.status = 'published' ORDER BY a.published_at DESC) t;" > /tmp/articles.json

echo "Checking exported data..."
cat /tmp/articles.json | head -c 500

echo ""
echo "Indexing to MeiliSearch..."
curl -s -X POST 'http://localhost:7700/indexes/articles/documents' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer newsapp_meili_master_key_2024' \
  --data @/tmp/articles.json

echo ""
echo "Done!"
