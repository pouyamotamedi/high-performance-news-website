-- Migration: Drop comprehensive integration testing tables
-- Version: 040
-- Description: Removes tables for comprehensive testing integration system

-- Drop views first
DROP VIEW IF EXISTS performance_metrics_trends;
DROP VIEW IF EXISTS active_alerts_summary;
DROP VIEW IF EXISTS component_health_dashboard;
DROP VIEW IF EXISTS recent_test_executions;

-- Drop functions
DROP FUNCTION IF EXISTS aggregate_integration_metrics(VARCHAR, VARCHAR, INTERVAL);
DROP FUNCTION IF EXISTS calculate_overall_health_score();
DROP FUNCTION IF EXISTS cleanup_expired_baselines();
DROP FUNCTION IF EXISTS cleanup_expired_reports();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS system_optimization_results;
DROP TABLE IF EXISTS system_validation_results;
DROP TABLE IF EXISTS system_alerts;
DROP TABLE IF EXISTS predictive_insights;
DROP TABLE IF EXISTS test_optimization_suggestions;
DROP TABLE IF EXISTS flaky_test_quarantine_log;
DROP TABLE IF EXISTS test_environment_usage;
DROP TABLE IF EXISTS data_consistency_results;
DROP TABLE IF EXISTS security_scan_results;
DROP TABLE IF EXISTS performance_baseline_snapshots;
DROP TABLE IF EXISTS ai_test_generation_history;
DROP TABLE IF EXISTS test_execution_queue;
DROP TABLE IF EXISTS component_health;
DROP TABLE IF EXISTS integration_metrics;
DROP TABLE IF EXISTS test_reports;
DROP TABLE IF EXISTS configuration_templates;
DROP TABLE IF EXISTS configuration_history;
DROP TABLE IF EXISTS enhanced_test_results;