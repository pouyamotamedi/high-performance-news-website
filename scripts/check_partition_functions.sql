-- Check partition functions and their definitions
\echo '=== PARTITION FUNCTIONS ==='
SELECT proname, prosrc 
FROM pg_proc 
WHERE proname LIKE '%partition%'
ORDER BY proname;

\echo ''
\echo '=== MONITORING PARTITION FUNCTIONS ==='
SELECT proname, prosrc 
FROM pg_proc 
WHERE proname LIKE '%monitoring%' OR proname LIKE '%daily%'
ORDER BY proname;

\echo ''
\echo '=== LATEST PARTITIONS FOR EACH METRICS TABLE ==='
SELECT 
    parent.relname as parent_table,
    MAX(child.relname) as latest_partition
FROM pg_inherits
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
JOIN pg_class child ON pg_inherits.inhrelid = child.oid
WHERE parent.relname IN ('system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics')
GROUP BY parent.relname
ORDER BY parent.relname;

\echo ''
\echo '=== CACHE_METRICS STRUCTURE ==='
\d cache_metrics

\echo ''
\echo '=== PUBLISHING_METRICS STRUCTURE ==='
\d publishing_metrics
