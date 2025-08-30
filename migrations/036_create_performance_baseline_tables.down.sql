-- Rollback Performance Baseline and Regression Detection Tables
-- Migration: 036_create_performance_baseline_tables.down.sql

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_performance_baselines_updated_at ON performance_baselines;
DROP FUNCTION IF EXISTS update_performance_baselines_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_performance_baselines_test_env_active;
DROP INDEX IF EXISTS idx_performance_baselines_created_at;
DROP INDEX IF EXISTS idx_performance_regression_results_test_name;
DROP INDEX IF EXISTS idx_performance_regression_results_compared_at;
DROP INDEX IF EXISTS idx_performance_regression_results_status;
DROP INDEX IF EXISTS idx_performance_metrics_history_test_metric;
DROP INDEX IF EXISTS idx_performance_metrics_history_measured_at;
DROP INDEX IF EXISTS idx_performance_metrics_history_env_version;
DROP INDEX IF EXISTS idx_performance_alerts_test_severity;
DROP INDEX IF EXISTS idx_performance_alerts_resolved;
DROP INDEX IF EXISTS idx_performance_alerts_created_at;
DROP INDEX IF EXISTS idx_performance_baselines_metrics_gin;
DROP INDEX IF EXISTS idx_performance_regression_results_data_gin;
DROP INDEX IF EXISTS idx_performance_alerts_details_gin;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS performance_alerts;
DROP TABLE IF EXISTS performance_metrics_history;
DROP TABLE IF EXISTS performance_regression_results;
DROP TABLE IF EXISTS performance_baselines;