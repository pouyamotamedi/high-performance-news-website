-- Drop comprehensive monitoring system tables
-- This migration removes all monitoring and alerting system tables

-- Drop views first
DROP VIEW IF EXISTS system_health_summary;
DROP VIEW IF EXISTS recent_runbook_executions;
DROP VIEW IF EXISTS recent_remediation_executions;
DROP VIEW IF EXISTS active_alerts;

-- Drop triggers
DROP TRIGGER IF EXISTS update_monitoring_configuration_updated_at ON monitoring_configuration;
DROP TRIGGER IF EXISTS update_alert_history_updated_at ON alert_history;
DROP TRIGGER IF EXISTS update_operational_runbooks_updated_at ON operational_runbooks;
DROP TRIGGER IF EXISTS update_remediation_actions_updated_at ON remediation_actions;
DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS drop_old_monitoring_partitions();
DROP FUNCTION IF EXISTS create_daily_monitoring_partitions();

-- Drop main tables
DROP TABLE IF EXISTS monitoring_configuration;
DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS publishing_metrics_history;
DROP TABLE IF EXISTS cache_metrics_history;
DROP TABLE IF EXISTS database_metrics_history;
DROP TABLE IF EXISTS system_metrics_history;
DROP TABLE IF EXISTS runbook_executions;
DROP TABLE IF EXISTS operational_runbooks;
DROP TABLE IF EXISTS remediation_executions;
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS log_entries;

-- Drop any remaining partitions (they should be dropped automatically with parent tables)
-- But just in case, we'll try to drop some common ones
DROP TABLE IF EXISTS log_entries_2024_01;
DROP TABLE IF EXISTS log_entries_2024_02;
DROP TABLE IF EXISTS log_entries_2024_03;
DROP TABLE IF EXISTS system_metrics_history_2024_01;
DROP TABLE IF EXISTS system_metrics_history_2024_02;
DROP TABLE IF EXISTS database_metrics_history_2024_01;
DROP TABLE IF EXISTS database_metrics_history_2024_02;
DROP TABLE IF EXISTS cache_metrics_history_2024_01;
DROP TABLE IF EXISTS cache_metrics_history_2024_02;
DROP TABLE IF EXISTS publishing_metrics_history_2024_01;
DROP TABLE IF EXISTS publishing_metrics_history_2024_02;