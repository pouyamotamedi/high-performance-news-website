-- Remove partition management functions
DROP FUNCTION IF EXISTS partition_maintenance();
DROP FUNCTION IF EXISTS drop_old_partitions(integer);
DROP FUNCTION IF EXISTS create_daily_partitions();