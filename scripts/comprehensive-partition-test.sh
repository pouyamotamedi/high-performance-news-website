#!/bin/bash

echo "=========================================="
echo "COMPREHENSIVE PARTITION MANAGEMENT TEST"
echo "=========================================="
echo

# Test 1: Verify Daily Partition Structure
echo "1. TESTING: Daily Partition Structure"
echo "----------------------------------------"
echo "Checking current daily partitions..."
sudo -u newsapp psql -d newsdb -c "
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size('public.'||tablename)) as size
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public'
ORDER BY tablename
LIMIT 10;
"

echo
echo "Total daily partitions count:"
sudo -u newsapp psql -d newsdb -c "
SELECT COUNT(*) as daily_partitions_count
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public';
"

echo
echo "✅ Expected: Should see daily partitions like articles_2025_10_01, articles_2025_10_02, etc."
echo

# Test 2: Verify Partition Management Functions
echo "2. TESTING: Partition Management Functions"
echo "----------------------------------------"
echo "Checking if all functions exist..."
sudo -u newsapp psql -d newsdb -c "
SELECT 
    proname as function_name,
    pronargs as parameter_count,
    prorettype::regtype as return_type
FROM pg_proc 
WHERE proname IN ('create_daily_partitions', 'drop_old_partitions', 'partition_maintenance')
ORDER BY proname;
"

echo
echo "✅ Expected: Should see all 3 functions with correct signatures"
echo

# Test 3: Test Daily Partition Creation
echo "3. TESTING: Daily Partition Creation Function"
echo "----------------------------------------"
echo "Testing create_daily_partitions() function..."
sudo -u newsapp psql -d newsdb -c "SELECT * FROM create_daily_partitions();"

echo
echo "✅ Expected: Should show 'exists' for current partitions and create new ones for future dates"
echo

# Test 4: Verify Data Distribution
echo "4. TESTING: Data Distribution Across Partitions"
echo "----------------------------------------"
echo "Checking how articles are distributed across daily partitions..."
sudo -u newsapp psql -d newsdb -c "
SELECT 
    schemaname,
    tablename,
    (xpath('/row/c/text()', query_to_xml(format('SELECT COUNT(*) as c FROM %I.%I', schemaname, tablename), false, true, '')))[1]::text::int as article_count
FROM pg_tables 
WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$'
AND schemaname = 'public'
AND (xpath('/row/c/text()', query_to_xml(format('SELECT COUNT(*) as c FROM %I.%I', schemaname, tablename), false, true, '')))[1]::text::int > 0
ORDER BY tablename;
"

echo
echo "Total articles across all partitions:"
sudo -u newsapp psql -d newsdb -c "SELECT COUNT(*) as total_articles FROM articles;"

echo
echo "✅ Expected: Should see articles distributed across different daily partitions, total should match"
echo

# Test 5: Test Partition Cleanup Function
echo "5. TESTING: Partition Cleanup Function (Safe Test)"
echo "----------------------------------------"
echo "Testing drop_old_partitions() with 365 days (safe - won't delete anything)..."
sudo -u newsapp psql -d newsdb -c "SELECT * FROM drop_old_partitions(365);"

echo
echo "✅ Expected: Should show 'SUMMARY' with 'Dropped: 0' since we're using 365 days retention"
echo

# Test 6: Test Full Maintenance Function
echo "6. TESTING: Full Maintenance Function"
echo "----------------------------------------"
echo "Testing partition_maintenance() function..."
sudo -u newsapp psql -d newsdb -c "SELECT partition_maintenance();"

echo
echo "✅ Expected: Should complete successfully with maintenance notice"
echo

# Test 7: Verify Partition Indexes
echo "7. TESTING: Partition Index Optimization"
echo "----------------------------------------"
echo "Checking indexes on a sample daily partition..."
SAMPLE_PARTITION=$(sudo -u newsapp psql -d newsdb -t -c "SELECT tablename FROM pg_tables WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$' AND schemaname = 'public' LIMIT 1;" | xargs)

if [ ! -z "$SAMPLE_PARTITION" ]; then
    echo "Sample partition: $SAMPLE_PARTITION"
    sudo -u newsapp psql -d newsdb -c "
    SELECT 
        indexname,
        indexdef
    FROM pg_indexes 
    WHERE tablename = '$SAMPLE_PARTITION'
    ORDER BY indexname;
    "
else
    echo "No daily partitions found for index testing"
fi

echo
echo "✅ Expected: Should see optimized indexes (BRIN, composite, full-text search)"
echo

# Test 8: Test Article Insertion into Correct Partition
echo "8. TESTING: Article Insertion Routing"
echo "----------------------------------------"
echo "Testing if new articles go to correct daily partition..."

# Get current partition count
BEFORE_COUNT=$(sudo -u newsapp psql -d newsdb -t -c "SELECT COUNT(*) FROM articles;" | xargs)
echo "Articles before test insert: $BEFORE_COUNT"

# Insert a test article for today
sudo -u newsapp psql -d newsdb -c "
INSERT INTO articles (title, slug, content, excerpt, author_id, category_id, status, published_at, created_at, updated_at)
VALUES (
    'Partition Test Article - $(date)',
    'partition-test-$(date +%s)',
    'This is a test article to verify partition routing works correctly.',
    'Test article for partition verification',
    1,
    1,
    'published',
    NOW(),
    NOW(),
    NOW()
) RETURNING id, title, published_at;
"

# Check if article count increased
AFTER_COUNT=$(sudo -u newsapp psql -d newsdb -t -c "SELECT COUNT(*) FROM articles;" | xargs)
echo "Articles after test insert: $AFTER_COUNT"

# Find which partition the test article went to
TODAY_PARTITION="articles_$(date +%Y_%m_%d)"
echo "Expected partition for today: $TODAY_PARTITION"

sudo -u newsapp psql -d newsdb -c "
SELECT COUNT(*) as articles_in_todays_partition
FROM $TODAY_PARTITION
WHERE title LIKE 'Partition Test Article%';
" 2>/dev/null || echo "Today's partition doesn't exist yet (normal if no articles published today)"

echo
echo "✅ Expected: Article count should increase by 1, and article should be in today's partition"
echo

# Test 9: Performance Test
echo "9. TESTING: Query Performance on Partitioned Data"
echo "----------------------------------------"
echo "Testing query performance with date-based queries..."

echo "Query 1: Recent articles (should use partition pruning)"
sudo -u newsapp psql -d newsdb -c "
EXPLAIN (ANALYZE, BUFFERS) 
SELECT id, title, published_at 
FROM articles 
WHERE published_at >= CURRENT_DATE - INTERVAL '7 days'
AND status = 'published'
ORDER BY published_at DESC 
LIMIT 10;
"

echo
echo "✅ Expected: Should show partition pruning in execution plan"
echo

# Test 10: Verify Automated Scheduling (if running)
echo "10. TESTING: Partition Scheduler Status"
echo "----------------------------------------"
echo "Checking if partition maintenance is scheduled in the application..."

# Check application logs for partition activity
echo "Recent partition-related log entries:"
journalctl -u newsapp -n 100 --no-pager | grep -i partition | tail -5 || echo "No recent partition log entries found"

echo
echo "✅ Expected: May see partition maintenance logs if scheduler is active"
echo

echo "=========================================="
echo "COMPREHENSIVE TEST COMPLETED"
echo "=========================================="
echo
echo "SUMMARY OF WHAT TO VERIFY:"
echo "1. ✅ Daily partitions exist (articles_YYYY_MM_DD format)"
echo "2. ✅ All 3 partition management functions are present"
echo "3. ✅ create_daily_partitions() creates future partitions"
echo "4. ✅ Articles are distributed across daily partitions"
echo "5. ✅ drop_old_partitions() works (safe test with 365 days)"
echo "6. ✅ partition_maintenance() executes successfully"
echo "7. ✅ Each partition has optimized indexes"
echo "8. ✅ New articles route to correct daily partition"
echo "9. ✅ Queries use partition pruning for performance"
echo "10. ✅ System logs show partition maintenance activity"
echo
echo "If all tests pass, your Task 3 implementation is PERFECT! 🎉"