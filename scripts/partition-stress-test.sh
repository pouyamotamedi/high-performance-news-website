#!/bin/bash

echo "=========================================="
echo "PARTITION SYSTEM STRESS TEST"
echo "Use this to test partition system under load"
echo "=========================================="
echo

echo "WARNING: This test will insert test data into your database."
read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Test cancelled."
    exit 1
fi

# Stress Test 1: Bulk Insert Test
echo "1. BULK INSERT STRESS TEST"
echo "----------------------------------------"
echo "Inserting 100 test articles across different dates..."

sudo -u newsapp psql -d newsdb -c "
DO \$\$
DECLARE
    i integer;
    test_date date;
    test_title text;
    test_slug text;
BEGIN
    FOR i IN 1..100 LOOP
        -- Distribute articles across last 10 days
        test_date := CURRENT_DATE - (i % 10);
        test_title := 'Stress Test Article ' || i || ' - ' || test_date;
        test_slug := 'stress-test-' || i || '-' || to_char(test_date, 'YYYY-MM-DD');
        
        INSERT INTO articles (
            title, slug, content, excerpt, author_id, category_id, 
            status, published_at, created_at, updated_at
        ) VALUES (
            test_title,
            test_slug,
            'This is stress test content for article ' || i || '. Generated on ' || NOW(),
            'Stress test excerpt for article ' || i,
            1,
            1,
            'published',
            test_date + (random() * interval '24 hours'),
            NOW(),
            NOW()
        );
        
        -- Show progress every 20 inserts
        IF i % 20 = 0 THEN
            RAISE NOTICE 'Inserted % articles...', i;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Bulk insert completed: 100 articles inserted';
END;
\$\$;
"

echo

# Stress Test 2: Concurrent Query Test
echo "2. CONCURRENT QUERY STRESS TEST"
echo "----------------------------------------"
echo "Running multiple concurrent queries..."

# Function to run a query in background
run_query() {
    local query_id=$1
    echo "Starting query $query_id..."
    sudo -u newsapp psql -d newsdb -c "
    SELECT 
        'Query $query_id' as query_name,
        COUNT(*) as result_count,
        MIN(published_at) as earliest_date,
        MAX(published_at) as latest_date
    FROM articles 
    WHERE published_at >= CURRENT_DATE - INTERVAL '$(($query_id + 1)) days'
    AND status = 'published';
    " > /tmp/query_$query_id.log 2>&1 &
}

# Start 5 concurrent queries
for i in {1..5}; do
    run_query $i
done

# Wait for all queries to complete
wait

# Show results
echo "Concurrent query results:"
for i in {1..5}; do
    echo "Query $i result:"
    cat /tmp/query_$i.log
    rm -f /tmp/query_$i.log
done

echo

# Stress Test 3: Partition Function Performance Test
echo "3. PARTITION FUNCTION PERFORMANCE TEST"
echo "----------------------------------------"
echo "Testing partition management functions under load..."

echo "Creating daily partitions (should handle existing partitions gracefully):"
time sudo -u newsapp psql -d newsdb -c "SELECT * FROM create_daily_partitions();"

echo
echo "Testing partition cleanup (safe test with 365 days):"
time sudo -u newsapp psql -d newsdb -c "SELECT * FROM drop_old_partitions(365);"

echo
echo "Testing full maintenance function:"
time sudo -u newsapp psql -d newsdb -c "SELECT partition_maintenance();"

echo

# Stress Test 4: Large Query Performance Test
echo "4. LARGE QUERY PERFORMANCE TEST"
echo "----------------------------------------"
echo "Testing performance of large queries across partitions..."

echo "Query 1: Count all articles (full table scan):"
time sudo -u newsapp psql -d newsdb -c "SELECT COUNT(*) FROM articles;"

echo
echo "Query 2: Recent articles with partition pruning:"
time sudo -u newsapp psql -d newsdb -c "
SELECT COUNT(*) 
FROM articles 
WHERE published_at >= CURRENT_DATE - INTERVAL '7 days'
AND status = 'published';
"

echo
echo "Query 3: Complex query with joins:"
time sudo -u newsapp psql -d newsdb -c "
SELECT 
    a.title,
    a.published_at,
    COUNT(at.tag_id) as tag_count
FROM articles a
LEFT JOIN article_tags at ON a.id = at.article_id
WHERE a.published_at >= CURRENT_DATE - INTERVAL '3 days'
AND a.status = 'published'
GROUP BY a.id, a.title, a.published_at
ORDER BY a.published_at DESC
LIMIT 20;
"

echo

# Stress Test 5: Cleanup Test Data
echo "5. CLEANUP TEST DATA"
echo "----------------------------------------"
echo "Cleaning up stress test data..."

DELETED_COUNT=$(sudo -u newsapp psql -d newsdb -t -c "
DELETE FROM articles 
WHERE title LIKE 'Stress Test Article%' 
   OR title LIKE 'Partition Test Article%';
SELECT ROW_COUNT();
" | xargs)

echo "Deleted $DELETED_COUNT test articles"

echo

echo "=========================================="
echo "STRESS TEST COMPLETED"
echo "=========================================="
echo
echo "PERFORMANCE INDICATORS TO REVIEW:"
echo "• Bulk inserts should complete quickly"
echo "• Concurrent queries should not block each other"
echo "• Partition functions should execute in reasonable time"
echo "• Recent queries should be faster than full table scans"
echo "• Complex queries should use partition pruning"
echo
echo "If all tests performed well, your partition system is robust! 🚀"