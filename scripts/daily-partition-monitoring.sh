#!/bin/bash

echo "=========================================="
echo "DAILY PARTITION MONITORING SCRIPT"
echo "Run this daily to monitor partition health"
echo "=========================================="
echo

# Monitor 1: Check if new partitions are being created
echo "1. PARTITION CREATION MONITORING"
echo "----------------------------------------"
echo "Partitions created in the last 7 days:"
sudo -u newsapp psql -d newsdb -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size('public.'||tablename)) as size
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public'
AND tablename >= 'articles_' || to_char(CURRENT_DATE - INTERVAL '7 days', 'YYYY_MM_DD')
ORDER BY tablename;
"

echo

# Monitor 2: Check partition sizes and growth
echo "2. PARTITION SIZE MONITORING"
echo "----------------------------------------"
echo "Largest partitions (top 10):"
sudo -u newsapp psql -d newsdb -c "
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size('public.'||tablename)) as size,
    pg_total_relation_size('public.'||tablename) as size_bytes
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public'
ORDER BY pg_total_relation_size('public.'||tablename) DESC
LIMIT 10;
"

echo

# Monitor 3: Check for old partitions that should be cleaned up
echo "3. OLD PARTITION MONITORING"
echo "----------------------------------------"
echo "Partitions older than 30 days (candidates for cleanup):"
sudo -u newsapp psql -d newsdb -c "
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size('public.'||tablename)) as size,
    CURRENT_DATE - to_date(regexp_replace(tablename, '^articles_', ''), 'YYYY_MM_DD') as days_old
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public'
AND CURRENT_DATE - to_date(regexp_replace(tablename, '^articles_', ''), 'YYYY_MM_DD') > 30
ORDER BY days_old DESC;
"

echo

# Monitor 4: Check partition maintenance function status
echo "4. MAINTENANCE FUNCTION STATUS"
echo "----------------------------------------"
echo "Testing partition maintenance functions..."
sudo -u newsapp psql -d newsdb -c "
SELECT 
    proname as function_name,
    'OK' as status
FROM pg_proc 
WHERE proname IN ('create_daily_partitions', 'drop_old_partitions', 'partition_maintenance');
"

echo

# Monitor 5: Check recent article distribution
echo "5. RECENT ARTICLE DISTRIBUTION"
echo "----------------------------------------"
echo "Articles published in the last 7 days by partition:"
sudo -u newsapp psql -d newsdb -c "
SELECT 
    DATE(published_at) as publish_date,
    COUNT(*) as article_count
FROM articles 
WHERE published_at >= CURRENT_DATE - INTERVAL '7 days'
AND status = 'published'
GROUP BY DATE(published_at)
ORDER BY publish_date DESC;
"

echo

# Monitor 6: Performance check
echo "6. PARTITION PERFORMANCE CHECK"
echo "----------------------------------------"
echo "Query performance test (recent articles):"
sudo -u newsapp psql -d newsdb -c "
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) 
SELECT COUNT(*) 
FROM articles 
WHERE published_at >= CURRENT_DATE - INTERVAL '1 day'
AND status = 'published';
" | head -10

echo

echo "=========================================="
echo "MONITORING COMPLETE"
echo "=========================================="
echo
echo "WHAT TO WATCH FOR:"
echo "• New partitions should be created automatically"
echo "• Partition sizes should grow reasonably"
echo "• Old partitions (>30 days) should be cleaned up"
echo "• All maintenance functions should be present"
echo "• Articles should distribute across recent partitions"
echo "• Queries should show partition pruning in execution plans"
echo
echo "Run this script daily to monitor partition health!"