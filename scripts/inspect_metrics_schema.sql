-- Comprehensive metrics table schema inspection
-- Run with: sudo -u newsapp psql -d newsdb -f /home/newsapp/news-website/scripts/inspect_metrics_schema.sql

\echo '=== SYSTEM_METRICS TABLE STRUCTURE ==='
\d system_metrics

\echo ''
\echo '=== DATABASE_METRICS TABLE STRUCTURE ==='
\d database_metrics

\echo ''
\echo '=== CACHE_METRICS TABLE STRUCTURE ==='
\d cache_metrics

\echo ''
\echo '=== PUBLISHING_METRICS TABLE STRUCTURE ==='
\d publishing_metrics

\echo ''
\echo '=== CHECK FOR _HISTORY VARIANTS ==='
SELECT tablename FROM pg_tables WHERE tablename LIKE '%_metrics_history' ORDER BY tablename;

\echo ''
\echo '=== SYSTEM_METRICS_HISTORY STRUCTURE (if exists) ==='
\d system_metrics_history

\echo ''
\echo '=== DATABASE_METRICS_HISTORY STRUCTURE (if exists) ==='
\d database_metrics_history

\echo ''
\echo '=== CACHE_METRICS_HISTORY STRUCTURE (if exists) ==='
\d cache_metrics_history

\echo ''
\echo '=== PUBLISHING_METRICS_HISTORY STRUCTURE (if exists) ==='
\d publishing_metrics_history

\echo ''
\echo '=== ALL PARTITIONS FOR METRICS TABLES ==='
SELECT 
    parent.relname as parent_table,
    child.relname as partition_name,
    pg_get_expr(child.relpartbound, child.oid) as partition_range
FROM pg_inherits
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
JOIN pg_class child ON pg_inherits.inhrelid = child.oid
WHERE parent.relname IN ('system_metrics', 'database_metrics', 'cache_metrics', 'publishing_metrics',
                         'system_metrics_history', 'database_metrics_history', 'cache_metrics_history', 'publishing_metrics_history')
ORDER BY parent.relname, child.relname;

\echo ''
\echo '=== CHECK PARTITION FUNCTIONS ==='
SELECT proname, prosrc 
FROM pg_proc 
WHERE proname LIKE '%partition%' OR proname LIKE '%metrics%'
ORDER BY proname;

\echo ''
\echo '=== CHECK TRIGGERS ON METRICS TABLES ==='
SELECT 
    tgname as trigger_name,
    tgrelid::regclass as table_name,
    tgtype,
    tgenabled
FROM pg_trigger
WHERE tgrelid::regclass::text LIKE '%metrics%'
ORDER BY table_name;
