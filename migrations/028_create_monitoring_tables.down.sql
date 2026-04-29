-- Drop monitoring tables and related objects

-- Drop views
DROP VIEW IF EXISTS latest_publishing_metrics;
DROP VIEW IF EXISTS latest_cache_metrics;
DROP VIEW IF EXISTS latest_database_metrics;
DROP VIEW IF EXISTS latest_system_metrics;

-- Drop functions
DROP FUNCTION IF EXISTS drop_old_monitoring_partitions();
DROP FUNCTION IF EXISTS create_daily_monitoring_partitions();

-- Drop tables (partitioned tables will drop their partitions automatically)
DROP TABLE IF EXISTS monitoring_config;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS publishing_metrics;
DROP TABLE IF EXISTS cache_metrics;
DROP TABLE IF EXISTS database_metrics;
DROP TABLE IF EXISTS system_metrics;
DROP TABLE IF EXISTS health_checks;