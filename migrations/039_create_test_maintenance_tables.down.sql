-- Drop Test Maintenance and Evolution System Tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_test_metadata_timestamp_trigger ON test_metadata;
DROP TRIGGER IF EXISTS update_test_evolution_timestamp_trigger ON test_evolution;
DROP TRIGGER IF EXISTS update_maintenance_schedules_timestamp_trigger ON maintenance_schedules;

-- Drop functions
DROP FUNCTION IF EXISTS update_test_metadata_timestamp();

-- Drop views
DROP VIEW IF EXISTS active_tests;
DROP VIEW IF EXISTS deprecated_tests;
DROP VIEW IF EXISTS high_failure_tests;
DROP VIEW IF EXISTS slow_tests;
DROP VIEW IF EXISTS low_coverage_tests;
DROP VIEW IF EXISTS test_relationship_summary;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS refactoring_opportunities;
DROP TABLE IF EXISTS test_quality_metrics;
DROP TABLE IF EXISTS maintenance_schedules;
DROP TABLE IF EXISTS test_metric_snapshots;
DROP TABLE IF EXISTS test_changes;
DROP TABLE IF EXISTS test_evolution;
DROP TABLE IF EXISTS test_lifecycle_events;
DROP TABLE IF EXISTS test_migrations;
DROP TABLE IF EXISTS test_relationships;
DROP TABLE IF EXISTS test_metadata;