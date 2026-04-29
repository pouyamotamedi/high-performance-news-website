-- Drop backup system tables and functions

-- Drop triggers
DROP TRIGGER IF EXISTS update_disaster_recovery_tests_updated_at ON disaster_recovery_tests;
DROP TRIGGER IF EXISTS update_restore_operations_updated_at ON restore_operations;
DROP TRIGGER IF EXISTS update_backup_validations_updated_at ON backup_validations;
DROP TRIGGER IF EXISTS update_backup_replications_updated_at ON backup_replications;
DROP TRIGGER IF EXISTS update_backups_updated_at ON backups;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS cleanup_old_backups(INTEGER);
DROP FUNCTION IF EXISTS get_backup_health();

-- Drop view
DROP VIEW IF EXISTS backup_metrics_view;

-- Drop tables (in reverse order due to foreign key constraints)
DROP TABLE IF EXISTS disaster_recovery_tests;
DROP TABLE IF EXISTS restore_operations;
DROP TABLE IF EXISTS backup_validations;
DROP TABLE IF EXISTS backup_replications;
DROP TABLE IF EXISTS backups;