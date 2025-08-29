package repositories

import (
	"database/sql"
	"fmt"
	"high-performance-news-website/internal/models"
)

type BackupRepository struct {
	db *sql.DB
}

func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// Backup CRUD operations
func (r *BackupRepository) Create(backup *models.Backup) error {
	query := `
		INSERT INTO backups (type, status, file_path, file_size, checksum, compressed, encrypted, started_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		backup.Type,
		backup.Status,
		backup.FilePath,
		backup.FileSize,
		backup.Checksum,
		backup.Compressed,
		backup.Encrypted,
		backup.StartedAt,
		backup.Metadata,
	).Scan(&backup.ID, &backup.CreatedAt, &backup.UpdatedAt)
}

func (r *BackupRepository) GetByID(id uint64) (*models.Backup, error) {
	backup := &models.Backup{}
	query := `
		SELECT id, type, status, file_path, file_size, checksum, compressed, encrypted,
		       started_at, completed_at, error_msg, metadata, created_at, updated_at
		FROM backups WHERE id = $1`
	
	err := r.db.QueryRow(query, id).Scan(
		&backup.ID,
		&backup.Type,
		&backup.Status,
		&backup.FilePath,
		&backup.FileSize,
		&backup.Checksum,
		&backup.Compressed,
		&backup.Encrypted,
		&backup.StartedAt,
		&backup.CompletedAt,
		&backup.ErrorMsg,
		&backup.Metadata,
		&backup.CreatedAt,
		&backup.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return backup, nil
}

func (r *BackupRepository) Update(backup *models.Backup) error {
	query := `
		UPDATE backups 
		SET status = $2, file_size = $3, checksum = $4, completed_at = $5, 
		    error_msg = $6, metadata = $7, updated_at = NOW()
		WHERE id = $1`
	
	_, err := r.db.Exec(query,
		backup.ID,
		backup.Status,
		backup.FileSize,
		backup.Checksum,
		backup.CompletedAt,
		backup.ErrorMsg,
		backup.Metadata,
	)
	
	return err
}

func (r *BackupRepository) List(limit, offset int) ([]*models.Backup, error) {
	query := `
		SELECT id, type, status, file_path, file_size, checksum, compressed, encrypted,
		       started_at, completed_at, error_msg, metadata, created_at, updated_at
		FROM backups 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var backups []*models.Backup
	for rows.Next() {
		backup := &models.Backup{}
		err := rows.Scan(
			&backup.ID,
			&backup.Type,
			&backup.Status,
			&backup.FilePath,
			&backup.FileSize,
			&backup.Checksum,
			&backup.Compressed,
			&backup.Encrypted,
			&backup.StartedAt,
			&backup.CompletedAt,
			&backup.ErrorMsg,
			&backup.Metadata,
			&backup.CreatedAt,
			&backup.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	
	return backups, nil
}

func (r *BackupRepository) Delete(id uint64) error {
	query := `DELETE FROM backups WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BackupRepository) GetByStatus(status models.BackupStatus) ([]*models.Backup, error) {
	query := `
		SELECT id, type, status, file_path, file_size, checksum, compressed, encrypted,
		       started_at, completed_at, error_msg, metadata, created_at, updated_at
		FROM backups 
		WHERE status = $1
		ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var backups []*models.Backup
	for rows.Next() {
		backup := &models.Backup{}
		err := rows.Scan(
			&backup.ID,
			&backup.Type,
			&backup.Status,
			&backup.FilePath,
			&backup.FileSize,
			&backup.Checksum,
			&backup.Compressed,
			&backup.Encrypted,
			&backup.StartedAt,
			&backup.CompletedAt,
			&backup.ErrorMsg,
			&backup.Metadata,
			&backup.CreatedAt,
			&backup.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	
	return backups, nil
}

func (r *BackupRepository) GetOlderThan(days int) ([]*models.Backup, error) {
	query := `
		SELECT id, type, status, file_path, file_size, checksum, compressed, encrypted,
		       started_at, completed_at, error_msg, metadata, created_at, updated_at
		FROM backups 
		WHERE created_at < NOW() - INTERVAL '%d days'
		ORDER BY created_at ASC`
	
	rows, err := r.db.Query(fmt.Sprintf(query, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var backups []*models.Backup
	for rows.Next() {
		backup := &models.Backup{}
		err := rows.Scan(
			&backup.ID,
			&backup.Type,
			&backup.Status,
			&backup.FilePath,
			&backup.FileSize,
			&backup.Checksum,
			&backup.Compressed,
			&backup.Encrypted,
			&backup.StartedAt,
			&backup.CompletedAt,
			&backup.ErrorMsg,
			&backup.Metadata,
			&backup.CreatedAt,
			&backup.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	
	return backups, nil
}

// Backup Replication operations
func (r *BackupRepository) CreateReplication(replication *models.BackupReplication) error {
	query := `
		INSERT INTO backup_replications (backup_id, target_name, target_location, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		replication.BackupID,
		replication.TargetName,
		replication.TargetLocation,
		replication.Status,
		replication.StartedAt,
	).Scan(&replication.ID, &replication.CreatedAt, &replication.UpdatedAt)
}

func (r *BackupRepository) UpdateReplication(replication *models.BackupReplication) error {
	query := `
		UPDATE backup_replications 
		SET status = $2, replication_size = $3, replication_time = $4, 
		    completed_at = $5, error_msg = $6, updated_at = NOW()
		WHERE id = $1`
	
	_, err := r.db.Exec(query,
		replication.ID,
		replication.Status,
		replication.ReplicationSize,
		replication.ReplicationTime,
		replication.CompletedAt,
		replication.ErrorMsg,
	)
	
	return err
}

func (r *BackupRepository) GetReplicationsByBackupID(backupID uint64) ([]*models.BackupReplication, error) {
	query := `
		SELECT id, backup_id, target_name, target_location, status, replication_size,
		       replication_time, started_at, completed_at, error_msg, created_at, updated_at
		FROM backup_replications 
		WHERE backup_id = $1
		ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, backupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var replications []*models.BackupReplication
	for rows.Next() {
		replication := &models.BackupReplication{}
		err := rows.Scan(
			&replication.ID,
			&replication.BackupID,
			&replication.TargetName,
			&replication.TargetLocation,
			&replication.Status,
			&replication.ReplicationSize,
			&replication.ReplicationTime,
			&replication.StartedAt,
			&replication.CompletedAt,
			&replication.ErrorMsg,
			&replication.CreatedAt,
			&replication.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		replications = append(replications, replication)
	}
	
	return replications, nil
}

// Backup Validation operations
func (r *BackupRepository) CreateValidation(validation *models.BackupValidation) error {
	query := `
		INSERT INTO backup_validations (backup_id, validation_type, status, started_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		validation.BackupID,
		validation.ValidationType,
		validation.Status,
		validation.StartedAt,
	).Scan(&validation.ID, &validation.CreatedAt, &validation.UpdatedAt)
}

func (r *BackupRepository) UpdateValidation(validation *models.BackupValidation) error {
	query := `
		UPDATE backup_validations 
		SET status = $2, validation_time = $3, result = $4, 
		    completed_at = $5, error_msg = $6, updated_at = NOW()
		WHERE id = $1`
	
	_, err := r.db.Exec(query,
		validation.ID,
		validation.Status,
		validation.ValidationTime,
		validation.Result,
		validation.CompletedAt,
		validation.ErrorMsg,
	)
	
	return err
}

// Restore operations
func (r *BackupRepository) CreateRestoreOperation(restore *models.RestoreOperation) error {
	query := `
		INSERT INTO restore_operations (backup_id, restore_type, target_database, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		restore.BackupID,
		restore.RestoreType,
		restore.TargetDatabase,
		restore.Status,
		restore.StartedAt,
	).Scan(&restore.ID, &restore.CreatedAt, &restore.UpdatedAt)
}

func (r *BackupRepository) UpdateRestoreOperation(restore *models.RestoreOperation) error {
	query := `
		UPDATE restore_operations 
		SET status = $2, restore_time = $3, records_restored = $4, 
		    completed_at = $5, error_msg = $6, updated_at = NOW()
		WHERE id = $1`
	
	_, err := r.db.Exec(query,
		restore.ID,
		restore.Status,
		restore.RestoreTime,
		restore.RecordsRestored,
		restore.CompletedAt,
		restore.ErrorMsg,
	)
	
	return err
}

func (r *BackupRepository) GetRestoreOperationByID(id uint64) (*models.RestoreOperation, error) {
	restore := &models.RestoreOperation{}
	query := `
		SELECT id, backup_id, restore_type, target_database, status, restore_time,
		       records_restored, started_at, completed_at, error_msg, created_at, updated_at
		FROM restore_operations WHERE id = $1`
	
	err := r.db.QueryRow(query, id).Scan(
		&restore.ID,
		&restore.BackupID,
		&restore.RestoreType,
		&restore.TargetDatabase,
		&restore.Status,
		&restore.RestoreTime,
		&restore.RecordsRestored,
		&restore.StartedAt,
		&restore.CompletedAt,
		&restore.ErrorMsg,
		&restore.CreatedAt,
		&restore.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return restore, nil
}

func (r *BackupRepository) ListRestoreOperations(limit, offset int) ([]*models.RestoreOperation, error) {
	query := `
		SELECT id, backup_id, restore_type, target_database, status, restore_time,
		       records_restored, started_at, completed_at, error_msg, created_at, updated_at
		FROM restore_operations 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var operations []*models.RestoreOperation
	for rows.Next() {
		restore := &models.RestoreOperation{}
		err := rows.Scan(
			&restore.ID,
			&restore.BackupID,
			&restore.RestoreType,
			&restore.TargetDatabase,
			&restore.Status,
			&restore.RestoreTime,
			&restore.RecordsRestored,
			&restore.StartedAt,
			&restore.CompletedAt,
			&restore.ErrorMsg,
			&restore.CreatedAt,
			&restore.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		operations = append(operations, restore)
	}
	
	return operations, nil
}

// Disaster Recovery Test operations
func (r *BackupRepository) CreateDRTest(test *models.DisasterRecoveryTest) error {
	query := `
		INSERT INTO disaster_recovery_tests (test_name, backup_id, test_type, status, test_environment, started_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	
	return r.db.QueryRow(query,
		test.TestName,
		test.BackupID,
		test.TestType,
		test.Status,
		test.TestEnvironment,
		test.StartedAt,
	).Scan(&test.ID, &test.CreatedAt, &test.UpdatedAt)
}

func (r *BackupRepository) UpdateDRTest(test *models.DisasterRecoveryTest) error {
	query := `
		UPDATE disaster_recovery_tests 
		SET status = $2, recovery_time = $3, data_integrity = $4, test_results = $5,
		    completed_at = $6, error_msg = $7, updated_at = NOW()
		WHERE id = $1`
	
	_, err := r.db.Exec(query,
		test.ID,
		test.Status,
		test.RecoveryTime,
		test.DataIntegrity,
		test.TestResults,
		test.CompletedAt,
		test.ErrorMsg,
	)
	
	return err
}

func (r *BackupRepository) GetDRTestByID(id uint64) (*models.DisasterRecoveryTest, error) {
	test := &models.DisasterRecoveryTest{}
	query := `
		SELECT id, test_name, backup_id, test_type, status, test_environment, recovery_time,
		       data_integrity, test_results, started_at, completed_at, error_msg, created_at, updated_at
		FROM disaster_recovery_tests WHERE id = $1`
	
	err := r.db.QueryRow(query, id).Scan(
		&test.ID,
		&test.TestName,
		&test.BackupID,
		&test.TestType,
		&test.Status,
		&test.TestEnvironment,
		&test.RecoveryTime,
		&test.DataIntegrity,
		&test.TestResults,
		&test.StartedAt,
		&test.CompletedAt,
		&test.ErrorMsg,
		&test.CreatedAt,
		&test.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return test, nil
}

func (r *BackupRepository) ListDRTests(limit, offset int) ([]*models.DisasterRecoveryTest, error) {
	query := `
		SELECT id, test_name, backup_id, test_type, status, test_environment, recovery_time,
		       data_integrity, test_results, started_at, completed_at, error_msg, created_at, updated_at
		FROM disaster_recovery_tests 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tests []*models.DisasterRecoveryTest
	for rows.Next() {
		test := &models.DisasterRecoveryTest{}
		err := rows.Scan(
			&test.ID,
			&test.TestName,
			&test.BackupID,
			&test.TestType,
			&test.Status,
			&test.TestEnvironment,
			&test.RecoveryTime,
			&test.DataIntegrity,
			&test.TestResults,
			&test.StartedAt,
			&test.CompletedAt,
			&test.ErrorMsg,
			&test.CreatedAt,
			&test.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tests = append(tests, test)
	}
	
	return tests, nil
}

// Metrics operations
func (r *BackupRepository) GetMetrics() (*models.BackupMetrics, error) {
	metrics := &models.BackupMetrics{}
	
	// Get basic backup counts
	query := `
		SELECT 
			COUNT(*) as total_backups,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_backups,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_backups,
			COALESCE(SUM(file_size), 0) as total_backup_size,
			COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000), 0) as average_backup_time,
			MAX(completed_at) as last_backup_time,
			MAX(CASE WHEN status = 'completed' THEN completed_at END) as last_successful_backup
		FROM backups`
	
	err := r.db.QueryRow(query).Scan(
		&metrics.TotalBackups,
		&metrics.SuccessfulBackups,
		&metrics.FailedBackups,
		&metrics.TotalBackupSize,
		&metrics.AverageBackupTime,
		&metrics.LastBackupTime,
		&metrics.LastSuccessfulBackup,
	)
	if err != nil {
		return nil, err
	}
	
	// Get replication metrics
	replicationQuery := `
		SELECT 
			COUNT(DISTINCT target_name) as replication_targets,
			COUNT(CASE WHEN status = 'running' THEN 1 END) as active_replications,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_replications
		FROM backup_replications`
	
	err = r.db.QueryRow(replicationQuery).Scan(
		&metrics.ReplicationTargets,
		&metrics.ActiveReplications,
		&metrics.FailedReplications,
	)
	if err != nil {
		return nil, err
	}
	
	// Get DR test metrics
	drQuery := `
		SELECT 
			MAX(completed_at) as last_dr_test,
			COALESCE(bool_and(data_integrity), false) as dr_test_success
		FROM disaster_recovery_tests 
		WHERE status = 'completed' AND completed_at > NOW() - INTERVAL '30 days'`
	
	err = r.db.QueryRow(drQuery).Scan(
		&metrics.LastDRTest,
		&metrics.DRTestSuccess,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	return metrics, nil
}