-- Drop function
DROP FUNCTION IF EXISTS create_monthly_analytics_partitions();

-- Drop tables (this will also drop partitions)
DROP TABLE IF EXISTS analytics_reports;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS user_behavior;