package models

import (
	"time"
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeWAL         BackupType = "wal"
)

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusRunning    BackupStatus = "running"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusValidated  BackupStatus = "validated"
	BackupStatusReplicated BackupStatus = "replicated"
)

// Backup represents a backup record
type Backup struct {
	ID          uint64       `json:"id" db:"id"`
	Type        BackupType   `json:"type" db:"type"`
	Status      BackupStatus `json:"status" db:"status"`
	FilePath    string       `json:"file_path" db:"file_path"`
	FileSize    int64        `json:"file_size" db:"file_size"`
	Checksum    string       `json:"checksum" db:"checksum"`
	Compressed  bool         `json:"compressed" db:"compressed"`
	Encrypted   bool         `json:"encrypted" db:"encrypted"`
	StartedAt   time.Time    `json:"started_at" db:"started_at"`
	CompletedAt *time.Time   `json:"completed_at" db:"completed_at"`
	ErrorMsg    string       `json:"error_msg" db:"error_msg"`
	Metadata    string       `json:"metadata" db:"metadata"` // JSON metadata
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// BackupReplication represents a backup replication record
type BackupReplication struct {
	ID               uint64       `json:"id" db:"id"`
	BackupID         uint64       `json:"backup_id" db:"backup_id"`
	TargetName       string       `json:"target_name" db:"target_name"`
	TargetLocation   string       `json:"target_location" db:"target_location"`
	Status           BackupStatus `json:"status" db:"status"`
	ReplicationSize  int64        `json:"replication_size" db:"replication_size"`
	ReplicationTime  int64        `json:"replication_time" db:"replication_time"` // milliseconds
	StartedAt        time.Time    `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time   `json:"completed_at" db:"completed_at"`
	ErrorMsg         string       `json:"error_msg" db:"error_msg"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at" db:"updated_at"`
}

// BackupValidation represents a backup validation record
type BackupValidation struct {
	ID            uint64    `json:"id" db:"id"`
	BackupID      uint64    `json:"backup_id" db:"backup_id"`
	ValidationType string   `json:"validation_type" db:"validation_type"` // checksum, restore_test, integrity
	Status        BackupStatus `json:"status" db:"status"`
	ValidationTime int64    `json:"validation_time" db:"validation_time"` // milliseconds
	Result        string    `json:"result" db:"result"` // JSON result data
	StartedAt     time.Time `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time `json:"completed_at" db:"completed_at"`
	ErrorMsg      string    `json:"error_msg" db:"error_msg"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// DisasterRecoveryTest represents a disaster recovery test record
type DisasterRecoveryTest struct {
	ID              uint64    `json:"id" db:"id"`
	TestName        string    `json:"test_name" db:"test_name"`
	BackupID        uint64    `json:"backup_id" db:"backup_id"`
	TestType        string    `json:"test_type" db:"test_type"` // full_restore, partial_restore, point_in_time
	Status          BackupStatus `json:"status" db:"status"`
	TestEnvironment string    `json:"test_environment" db:"test_environment"`
	RecoveryTime    int64     `json:"recovery_time" db:"recovery_time"` // milliseconds
	DataIntegrity   bool      `json:"data_integrity" db:"data_integrity"`
	TestResults     string    `json:"test_results" db:"test_results"` // JSON test results
	StartedAt       time.Time `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	ErrorMsg        string    `json:"error_msg" db:"error_msg"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// BackupMetrics represents backup system metrics
type BackupMetrics struct {
	TotalBackups        int64   `json:"total_backups"`
	SuccessfulBackups   int64   `json:"successful_backups"`
	FailedBackups       int64   `json:"failed_backups"`
	TotalBackupSize     int64   `json:"total_backup_size"`
	AverageBackupTime   int64   `json:"average_backup_time"`
	LastBackupTime      *time.Time `json:"last_backup_time"`
	LastSuccessfulBackup *time.Time `json:"last_successful_backup"`
	ReplicationTargets  int     `json:"replication_targets"`
	ActiveReplications  int     `json:"active_replications"`
	FailedReplications  int     `json:"failed_replications"`
	LastDRTest          *time.Time `json:"last_dr_test"`
	DRTestSuccess       bool    `json:"dr_test_success"`
}

// BackupRequest represents a backup creation request
type BackupRequest struct {
	Type        BackupType `json:"type"`
	Description string     `json:"description"`
	Compress    bool       `json:"compress"`
	Encrypt     bool       `json:"encrypt"`
	Replicate   bool       `json:"replicate"`
	Validate    bool       `json:"validate"`
}

// RestoreRequest represents a restore operation request
type RestoreRequest struct {
	BackupID         uint64     `json:"backup_id"`
	TargetTimestamp  *time.Time `json:"target_timestamp,omitempty"` // For point-in-time recovery
	RestoreType      string     `json:"restore_type"` // full, partial, point_in_time
	TargetDatabase   string     `json:"target_database,omitempty"`
	OverwriteExisting bool      `json:"overwrite_existing"`
	ValidateAfter    bool       `json:"validate_after"`
}

// RestoreOperation represents a restore operation
type RestoreOperation struct {
	ID              uint64     `json:"id" db:"id"`
	BackupID        uint64     `json:"backup_id" db:"backup_id"`
	RestoreType     string     `json:"restore_type" db:"restore_type"`
	TargetDatabase  string     `json:"target_database" db:"target_database"`
	Status          BackupStatus `json:"status" db:"status"`
	RestoreTime     int64      `json:"restore_time" db:"restore_time"` // milliseconds
	RecordsRestored int64      `json:"records_restored" db:"records_restored"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	ErrorMsg        string     `json:"error_msg" db:"error_msg"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}