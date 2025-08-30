-- Drop consistency validation tables and related objects

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_old_consistency_data(INTEGER);
DROP FUNCTION IF EXISTS get_consistency_health_score();
DROP FUNCTION IF EXISTS update_consistency_updated_at();

-- Drop views
DROP VIEW IF EXISTS consistency_check_performance;
DROP VIEW IF EXISTS consistency_dashboard;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_consistency_schedules_updated_at ON consistency_schedules;
DROP TRIGGER IF EXISTS trigger_manual_review_queue_updated_at ON manual_review_queue;
DROP TRIGGER IF EXISTS trigger_consistency_issues_updated_at ON consistency_issues;

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS consistency_alerts;
DROP TABLE IF EXISTS consistency_trends;
DROP TABLE IF EXISTS consistency_check_results;
DROP TABLE IF EXISTS consistency_schedules;
DROP TABLE IF EXISTS manual_review_queue;
DROP TABLE IF EXISTS remediation_suggestions;
DROP TABLE IF EXISTS consistency_issues;