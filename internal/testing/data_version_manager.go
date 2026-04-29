package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// DataVersionManager manages test data versioning and migration
type DataVersionManager struct {
	db       *sql.DB
	versions map[string]*DataVersion
	mutex    sync.RWMutex
}

// DataVersion represents a version of test data
type DataVersion struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
	Schema      map[string]interface{} `json:"schema"`
	Migrations  []Migration            `json:"migrations"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Migration represents a data migration
type Migration struct {
	ID          string    `json:"id"`
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Script      string    `json:"script"`
	CreatedAt   time.Time `json:"created_at"`
	Applied     bool      `json:"applied"`
}

// NewDataVersionManager creates a new data version manager
func NewDataVersionManager() *DataVersionManager {
	return &DataVersionManager{
		versions: make(map[string]*DataVersion),
	}
}

// CreateVersion creates a new data version
func (dvm *DataVersionManager) CreateVersion(name, description string, schema map[string]interface{}) (*DataVersion, error) {
	dvm.mutex.Lock()
	defer dvm.mutex.Unlock()

	version := &DataVersion{
		ID:          fmt.Sprintf("v_%d", time.Now().Unix()),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Schema:      schema,
		Migrations:  make([]Migration, 0),
		Tags:        make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}

	dvm.versions[version.ID] = version
	log.Printf("Created data version: %s (%s)", version.ID, version.Name)

	return version, nil
}

// GetVersion retrieves a data version by ID
func (dvm *DataVersionManager) GetVersion(versionID string) (*DataVersion, error) {
	dvm.mutex.RLock()
	defer dvm.mutex.RUnlock()

	version, exists := dvm.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	return version, nil
}

// AddMigration adds a migration between versions
func (dvm *DataVersionManager) AddMigration(fromVersion, toVersion, script string) error {
	dvm.mutex.Lock()
	defer dvm.mutex.Unlock()

	migration := Migration{
		ID:          fmt.Sprintf("migration_%d", time.Now().Unix()),
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Script:      script,
		CreatedAt:   time.Now(),
		Applied:     false,
	}

	// Add migration to target version
	if version, exists := dvm.versions[toVersion]; exists {
		version.Migrations = append(version.Migrations, migration)
		log.Printf("Added migration from %s to %s", fromVersion, toVersion)
	}

	return nil
}

// ApplyMigration applies a migration
func (dvm *DataVersionManager) ApplyMigration(migrationID string) error {
	dvm.mutex.Lock()
	defer dvm.mutex.Unlock()

	// Find and apply migration
	for _, version := range dvm.versions {
		for i, migration := range version.Migrations {
			if migration.ID == migrationID && !migration.Applied {
				// In a real implementation, execute the migration script
				log.Printf("Applying migration: %s", migrationID)
				version.Migrations[i].Applied = true
				return nil
			}
		}
	}

	return fmt.Errorf("migration %s not found or already applied", migrationID)
}

// ListVersions returns all available versions
func (dvm *DataVersionManager) ListVersions() []*DataVersion {
	dvm.mutex.RLock()
	defer dvm.mutex.RUnlock()

	versions := make([]*DataVersion, 0, len(dvm.versions))
	for _, version := range dvm.versions {
		versions = append(versions, version)
	}

	return versions
}

// TagVersion adds tags to a version
func (dvm *DataVersionManager) TagVersion(versionID string, tags []string) error {
	dvm.mutex.Lock()
	defer dvm.mutex.Unlock()

	version, exists := dvm.versions[versionID]
	if !exists {
		return fmt.Errorf("version %s not found", versionID)
	}

	version.Tags = append(version.Tags, tags...)
	return nil
}

// GetVersionsByTag returns versions with specific tags
func (dvm *DataVersionManager) GetVersionsByTag(tag string) []*DataVersion {
	dvm.mutex.RLock()
	defer dvm.mutex.RUnlock()

	var matchingVersions []*DataVersion
	for _, version := range dvm.versions {
		for _, versionTag := range version.Tags {
			if versionTag == tag {
				matchingVersions = append(matchingVersions, version)
				break
			}
		}
	}

	return matchingVersions
}

// ExportVersion exports a version to JSON
func (dvm *DataVersionManager) ExportVersion(versionID string) ([]byte, error) {
	version, err := dvm.GetVersion(versionID)
	if err != nil {
		return nil, err
	}

	return json.Marshal(version)
}

// ImportVersion imports a version from JSON
func (dvm *DataVersionManager) ImportVersion(data []byte) (*DataVersion, error) {
	var version DataVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	dvm.mutex.Lock()
	defer dvm.mutex.Unlock()

	dvm.versions[version.ID] = &version
	log.Printf("Imported data version: %s (%s)", version.ID, version.Name)

	return &version, nil
}