package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

// AutoLinkingSettingsRepository handles auto-linking settings storage
type AutoLinkingSettingsRepository struct {
	db *sql.DB
}

// AutoLinkingSettings represents auto-linking configuration
type AutoLinkingSettings struct {
	ID                      int  `db:"id"`
	GlobalEnabled           bool `db:"global_enabled"`
	ContentIngestionEnabled bool `db:"content_ingestion_enabled"`
}

// NewAutoLinkingSettingsRepository creates a new settings repository
func NewAutoLinkingSettingsRepository(db *sql.DB) *AutoLinkingSettingsRepository {
	return &AutoLinkingSettingsRepository{db: db}
}

// GetSettings retrieves current auto-linking settings
func (r *AutoLinkingSettingsRepository) GetSettings(ctx context.Context) (*AutoLinkingSettings, error) {
	query := `
		SELECT id, global_enabled, content_ingestion_enabled
		FROM autolinking_settings
		ORDER BY id DESC
		LIMIT 1
	`
	
	var settings AutoLinkingSettings
	err := r.db.QueryRowContext(ctx, query).Scan(
		&settings.ID,
		&settings.GlobalEnabled,
		&settings.ContentIngestionEnabled,
	)
	
	if err == sql.ErrNoRows {
		// Return default settings if none exist
		return &AutoLinkingSettings{
			GlobalEnabled:           true,
			ContentIngestionEnabled: false,
		}, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	
	return &settings, nil
}

// UpdateSettings updates auto-linking settings
func (r *AutoLinkingSettingsRepository) UpdateSettings(ctx context.Context, settings *AutoLinkingSettings) error {
	// Check if settings exist
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM autolinking_settings)").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check settings existence: %w", err)
	}
	
	if exists {
		// Update existing settings
		query := `
			UPDATE autolinking_settings
			SET global_enabled = $1,
			    content_ingestion_enabled = $2,
			    updated_at = NOW()
			WHERE id = (SELECT id FROM autolinking_settings ORDER BY id DESC LIMIT 1)
		`
		
		_, err = r.db.ExecContext(ctx, query,
			settings.GlobalEnabled,
			settings.ContentIngestionEnabled,
		)
	} else {
		// Insert new settings
		query := `
			INSERT INTO autolinking_settings (global_enabled, content_ingestion_enabled, updated_at)
			VALUES ($1, $2, NOW())
		`
		
		_, err = r.db.ExecContext(ctx, query,
			settings.GlobalEnabled,
			settings.ContentIngestionEnabled,
		)
	}
	
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}
	
	return nil
}

// IsGlobalEnabled checks if auto-linking is globally enabled
func (r *AutoLinkingSettingsRepository) IsGlobalEnabled(ctx context.Context) (bool, error) {
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings.GlobalEnabled, nil
}

// IsContentIngestionEnabled checks if auto-linking is enabled for content ingestion
func (r *AutoLinkingSettingsRepository) IsContentIngestionEnabled(ctx context.Context) (bool, error) {
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings.ContentIngestionEnabled, nil
}
