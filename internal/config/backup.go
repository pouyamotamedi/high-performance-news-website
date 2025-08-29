package config

import (
	"time"
)

// BackupConfig holds all backup-related configuration
type BackupConfig struct {
	// General backup settings
	Enabled           bool          `mapstructure:"enabled"`
	BackupDir         string        `mapstructure:"backup_dir"`
	RetentionDays     int           `mapstructure:"retention_days"`
	CompressionLevel  int           `mapstructure:"compression_level"`
	EncryptionEnabled bool          `mapstructure:"encryption_enabled"`
	EncryptionKey     string        `mapstructure:"encryption_key"`
	
	// Backup scheduling
	FullBackupInterval        time.Duration `mapstructure:"full_backup_interval"`
	IncrementalBackupInterval time.Duration `mapstructure:"incremental_backup_interval"`
	
	// Cross-region replication
	CrossRegionEnabled bool                    `mapstructure:"cross_region_enabled"`
	ReplicationTargets []ReplicationTarget    `mapstructure:"replication_targets"`
	
	// Point-in-time recovery
	WALArchiveEnabled bool   `mapstructure:"wal_archive_enabled"`
	WALArchiveDir     string `mapstructure:"wal_archive_dir"`
	
	// Disaster recovery testing
	TestingEnabled   bool          `mapstructure:"testing_enabled"`
	TestingInterval  time.Duration `mapstructure:"testing_interval"`
	TestingRetention int           `mapstructure:"testing_retention"`
	
	// Notification settings
	NotificationEnabled bool     `mapstructure:"notification_enabled"`
	NotificationEmails  []string `mapstructure:"notification_emails"`
	SlackWebhookURL     string   `mapstructure:"slack_webhook_url"`
}

// ReplicationTarget represents a cross-region backup target
type ReplicationTarget struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"` // s3, ftp, sftp, local
	Endpoint string `mapstructure:"endpoint"`
	Region   string `mapstructure:"region"`
	Bucket   string `mapstructure:"bucket"`
	
	// Authentication
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	
	// Settings
	Enabled     bool `mapstructure:"enabled"`
	Compression bool `mapstructure:"compression"`
	Encryption  bool `mapstructure:"encryption"`
}

// SetBackupDefaults sets default values for backup configuration
func SetBackupDefaults() {
	// General backup defaults
	setDefault("backup.enabled", true)
	setDefault("backup.backup_dir", "/var/backups/news-website")
	setDefault("backup.retention_days", 30)
	setDefault("backup.compression_level", 6)
	setDefault("backup.encryption_enabled", true)
	setDefault("backup.encryption_key", "")
	
	// Backup scheduling defaults
	setDefault("backup.full_backup_interval", "24h")
	setDefault("backup.incremental_backup_interval", "1h")
	
	// Cross-region replication defaults
	setDefault("backup.cross_region_enabled", false)
	setDefault("backup.replication_targets", []ReplicationTarget{})
	
	// Point-in-time recovery defaults
	setDefault("backup.wal_archive_enabled", true)
	setDefault("backup.wal_archive_dir", "/var/backups/news-website/wal")
	
	// Disaster recovery testing defaults
	setDefault("backup.testing_enabled", true)
	setDefault("backup.testing_interval", "168h") // Weekly
	setDefault("backup.testing_retention", 5)
	
	// Notification defaults
	setDefault("backup.notification_enabled", true)
	setDefault("backup.notification_emails", []string{})
	setDefault("backup.slack_webhook_url", "")
}

// Helper function to set defaults (to be used with viper)
func setDefault(key string, value interface{}) {
	// This will be called from the main config package
	// Implementation depends on viper setup
}