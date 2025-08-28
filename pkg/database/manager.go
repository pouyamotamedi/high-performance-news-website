package database

import (
	"fmt"
	"log"
	"path/filepath"

	"high-performance-news-website/internal/config"
)

// Manager handles database connections, migrations, and partition management
type Manager struct {
	DB               *DB
	Migrator         *Migrator
	PartitionManager *PartitionManager
	config           *config.DatabaseConfig
}

// NewManager creates a new database manager with all components
func NewManager(cfg *config.DatabaseConfig, migrationsPath string) (*Manager, error) {
	var db *DB
	var err error

	// Create database connection (PgBouncer or direct)
	if cfg.UsePgBouncer {
		log.Println("Connecting to database via PgBouncer...")
		db, err = NewPgBouncerConnection(cfg)
	} else {
		log.Println("Connecting to database directly...")
		db, err = NewConnection(cfg)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	// Create migrator
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute migrations path: %w", err)
	}

	migrator, err := NewMigrator(db.DB, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	// Create partition manager
	partitionManager := NewPartitionManager(db.DB)

	return &Manager{
		DB:               db,
		Migrator:         migrator,
		PartitionManager: partitionManager,
		config:           cfg,
	}, nil
}

// Initialize sets up the database with migrations and partitions
func (m *Manager) Initialize() error {
	log.Println("Initializing database...")

	// Run migrations
	log.Println("Running database migrations...")
	if err := m.Migrator.Up(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create initial partitions
	log.Println("Creating database partitions...")
	if err := m.Migrator.CreatePartitions(); err != nil {
		return fmt.Errorf("failed to create partitions: %w", err)
	}

	// Set up partition maintenance
	log.Println("Setting up partition maintenance...")
	if err := m.PartitionManager.SchedulePartitionMaintenance(); err != nil {
		return fmt.Errorf("failed to schedule partition maintenance: %w", err)
	}

	// Start partition scheduler
	m.PartitionManager.StartPartitionScheduler()

	log.Println("Database initialization completed successfully")
	return nil
}

// Health checks the health of all database components
func (m *Manager) Health() error {
	if err := m.DB.Health(); err != nil {
		return fmt.Errorf("database connection health check failed: %w", err)
	}

	// Check if we can get migration version (indicates migrator is working)
	if _, _, err := m.Migrator.Version(); err != nil {
		log.Printf("Migration version check warning: %v", err)
		// Don't fail health check for this, as it might be expected in some cases
	}

	return nil
}

// GetStats returns database connection statistics
func (m *Manager) GetStats() DatabaseStats {
	dbStats := m.DB.GetStats()
	
	version, dirty, err := m.Migrator.Version()
	if err != nil {
		version = 0
		dirty = false
	}

	partitions, err := m.PartitionManager.GetPartitionInfo()
	if err != nil {
		partitions = []PartitionInfo{}
	}

	return DatabaseStats{
		ConnectionStats:   dbStats,
		MigrationVersion:  version,
		MigrationDirty:    dirty,
		PartitionCount:    len(partitions),
		Partitions:        partitions,
		UsePgBouncer:      m.config.UsePgBouncer,
	}
}

// Close closes all database connections and resources
func (m *Manager) Close() error {
	var errors []error

	// Close migrator
	if err := m.Migrator.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close migrator: %w", err))
	}

	// Close database connection
	if err := m.DB.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close database: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing database manager: %v", errors)
	}

	log.Println("Database manager closed successfully")
	return nil
}

// RunMaintenance manually triggers partition maintenance
func (m *Manager) RunMaintenance() error {
	log.Println("Running database maintenance...")
	
	if err := m.PartitionManager.RunMaintenance(); err != nil {
		return fmt.Errorf("failed to run partition maintenance: %w", err)
	}

	log.Println("Database maintenance completed successfully")
	return nil
}

// Migrate runs database migrations up or down
func (m *Manager) Migrate(direction string, steps int) error {
	switch direction {
	case "up":
		if steps > 0 {
			return m.Migrator.Steps(steps)
		}
		return m.Migrator.Up()
	case "down":
		if steps > 0 {
			return m.Migrator.Steps(-steps)
		}
		return m.Migrator.Down()
	default:
		return fmt.Errorf("invalid migration direction: %s (use 'up' or 'down')", direction)
	}
}

// ForceVersion forces the migration version without running migrations
func (m *Manager) ForceVersion(version int) error {
	return m.Migrator.Force(version)
}

// CreateDailyPartitions creates daily partitions for the next 7 days
func (m *Manager) CreateDailyPartitions() error {
	return m.PartitionManager.CreateDailyPartitions()
}

// DropOldPartitions removes partitions older than the specified retention period
func (m *Manager) DropOldPartitions(retentionDays int) error {
	return m.PartitionManager.DropOldPartitions(retentionDays)
}

type DatabaseStats struct {
	ConnectionStats   interface{}     `json:"connection_stats"`
	MigrationVersion  uint            `json:"migration_version"`
	MigrationDirty    bool            `json:"migration_dirty"`
	PartitionCount    int             `json:"partition_count"`
	Partitions        []PartitionInfo `json:"partitions"`
	UsePgBouncer      bool            `json:"use_pgbouncer"`
}