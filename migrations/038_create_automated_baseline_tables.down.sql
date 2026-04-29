-- Rollback Automated Baseline Management Tables
-- Migration: 038_create_automated_baseline_tables.down.sql

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_automated_baseline_results_updated_at ON automated_baseline_results;
DROP TRIGGER IF EXISTS trigger_baseline_schedules_updated_at ON baseline_schedules;
DROP TRIGGER IF EXISTS trigger_baseline_validation_rules_updated_at ON baseline_validation_rules;
DROP TRIGGER IF EXISTS trigger_baseline_update_triggers_updated_at ON baseline_update_triggers;
DROP TRIGGER IF EXISTS trigger_baseline_recommendations_updated_at ON baseline_recommendations;
DROP FUNCTION IF EXISTS update_automated_baseline_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_automated_baseline_results_test_env;
DROP INDEX IF EXISTS idx_automated_baseline_results_established_at;
DROP INDEX IF EXISTS idx_automated_baseline_results_quality_score;
DROP INDEX IF EXISTS idx_baseline_schedules_next_update;
DROP INDEX IF EXISTS idx_baseline_schedules_test_env;
DROP INDEX IF EXISTS idx_baseline_validation_rules_enabled;
DROP INDEX IF EXISTS idx_baseline_update_triggers_enabled;
DROP INDEX IF EXISTS idx_baseline_quality_metrics_baseline_id;
DROP INDEX IF EXISTS idx_capacity_forecasts_result_id;
DROP INDEX IF EXISTS idx_capacity_forecasts_resource_utilization;
DROP INDEX IF EXISTS idx_performance_trend_forecasts_result_id;
DROP INDEX IF EXISTS idx_performance_trend_forecasts_metric;
DROP INDEX IF EXISTS idx_baseline_recommendations_result_id;
DROP INDEX IF EXISTS idx_baseline_recommendations_priority_status;
DROP INDEX IF EXISTS idx_automated_baseline_results_data_gin;
DROP INDEX IF EXISTS idx_baseline_schedules_metrics_gin;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS baseline_recommendations;
DROP TABLE IF EXISTS performance_trend_forecasts;
DROP TABLE IF EXISTS capacity_forecasts;
DROP TABLE IF EXISTS baseline_quality_metrics;
DROP TABLE IF EXISTS baseline_update_triggers;
DROP TABLE IF EXISTS baseline_validation_rules;
DROP TABLE IF EXISTS baseline_schedules;
DROP TABLE IF EXISTS automated_baseline_results;