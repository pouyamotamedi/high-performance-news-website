-- Rollback script for Test Reliability Tracking System Tables

-- Drop the view first
DROP VIEW IF EXISTS flaky_tests_summary;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS environment_optimization_attempts;
DROP TABLE IF EXISTS environment_global_config;
DROP TABLE IF EXISTS environment_process_config;
DROP TABLE IF EXISTS environment_storage_config;
DROP TABLE IF EXISTS environment_network_config;
DROP TABLE IF EXISTS environment_resource_limits;
DROP TABLE IF EXISTS test_environment_adjustments;
DROP TABLE IF EXISTS test_remediation_attempts;
DROP TABLE IF EXISTS test_remediation_suggestions;
DROP TABLE IF EXISTS test_failure_patterns;
DROP TABLE IF EXISTS test_quarantine;
DROP TABLE IF EXISTS test_reliability_metrics;
DROP TABLE IF EXISTS test_flakiness;
DROP TABLE IF EXISTS test_executions;