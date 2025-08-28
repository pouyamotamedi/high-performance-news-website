package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// ThemeRepository handles theme data operations
type ThemeRepository struct {
	db *sql.DB
}

// NewThemeRepository creates a new theme repository
func NewThemeRepository(db *sql.DB) *ThemeRepository {
	return &ThemeRepository{db: db}
}

// Create creates a new theme
func (r *ThemeRepository) Create(theme *models.Theme) (*models.Theme, error) {
	configJSON, err := json.Marshal(theme.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO themes (name, description, is_active, is_default, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err = r.db.QueryRow(
		query,
		theme.Name,
		theme.Description,
		theme.IsActive,
		theme.IsDefault,
		configJSON,
		now,
		now,
	).Scan(&theme.ID, &theme.CreatedAt, &theme.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create theme: %w", err)
	}

	return theme, nil
}

// GetByID retrieves a theme by ID
func (r *ThemeRepository) GetByID(id uint64) (*models.Theme, error) {
	query := `
		SELECT id, name, description, is_active, is_default, config, created_at, updated_at
		FROM themes
		WHERE id = $1`

	theme := &models.Theme{}
	var configJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		&theme.IsActive,
		&theme.IsDefault,
		&configJSON,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("theme not found")
		}
		return nil, fmt.Errorf("failed to get theme: %w", err)
	}

	if err := json.Unmarshal(configJSON, &theme.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return theme, nil
}

// GetActive retrieves the active theme
func (r *ThemeRepository) GetActive() (*models.Theme, error) {
	query := `
		SELECT id, name, description, is_active, is_default, config, created_at, updated_at
		FROM themes
		WHERE is_active = true
		LIMIT 1`

	theme := &models.Theme{}
	var configJSON []byte

	err := r.db.QueryRow(query).Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		&theme.IsActive,
		&theme.IsDefault,
		&configJSON,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active theme found")
		}
		return nil, fmt.Errorf("failed to get active theme: %w", err)
	}

	if err := json.Unmarshal(configJSON, &theme.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return theme, nil
}

// GetAll retrieves all themes
func (r *ThemeRepository) GetAll() ([]*models.Theme, error) {
	query := `
		SELECT id, name, description, is_active, is_default, config, created_at, updated_at
		FROM themes
		ORDER BY is_default DESC, is_active DESC, name ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query themes: %w", err)
	}
	defer rows.Close()

	var themes []*models.Theme
	for rows.Next() {
		theme := &models.Theme{}
		var configJSON []byte

		err := rows.Scan(
			&theme.ID,
			&theme.Name,
			&theme.Description,
			&theme.IsActive,
			&theme.IsDefault,
			&configJSON,
			&theme.CreatedAt,
			&theme.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan theme: %w", err)
		}

		if err := json.Unmarshal(configJSON, &theme.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		themes = append(themes, theme)
	}

	return themes, nil
}

// Update updates a theme
func (r *ThemeRepository) Update(theme *models.Theme) error {
	configJSON, err := json.Marshal(theme.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE themes
		SET name = $2, description = $3, is_active = $4, is_default = $5, 
		    config = $6, updated_at = $7
		WHERE id = $1`

	result, err := r.db.Exec(
		query,
		theme.ID,
		theme.Name,
		theme.Description,
		theme.IsActive,
		theme.IsDefault,
		configJSON,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update theme: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("theme not found")
	}

	return nil
}

// SetActive sets a theme as active and deactivates all others
func (r *ThemeRepository) SetActive(id uint64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Deactivate all themes
	_, err = tx.Exec("UPDATE themes SET is_active = false, updated_at = $1", time.Now())
	if err != nil {
		return fmt.Errorf("failed to deactivate themes: %w", err)
	}

	// Activate the selected theme
	result, err := tx.Exec("UPDATE themes SET is_active = true, updated_at = $1 WHERE id = $2", time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to activate theme: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("theme not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete deletes a theme
func (r *ThemeRepository) Delete(id uint64) error {
	// Check if it's the active theme
	theme, err := r.GetByID(id)
	if err != nil {
		return err
	}

	if theme.IsActive {
		return fmt.Errorf("cannot delete active theme")
	}

	query := `DELETE FROM themes WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete theme: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("theme not found")
	}

	return nil
}

// CreateTemplateOverride creates a new template override
func (r *ThemeRepository) CreateTemplateOverride(override *models.TemplateOverride) (*models.TemplateOverride, error) {
	query := `
		INSERT INTO template_overrides (name, template_path, content, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		override.Name,
		override.TemplatePath,
		override.Content,
		override.IsActive,
		now,
		now,
	).Scan(&override.ID, &override.CreatedAt, &override.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create template override: %w", err)
	}

	return override, nil
}

// GetTemplateOverrideByPath retrieves a template override by path
func (r *ThemeRepository) GetTemplateOverrideByPath(path string) (*models.TemplateOverride, error) {
	query := `
		SELECT id, name, template_path, content, is_active, created_at, updated_at
		FROM template_overrides
		WHERE template_path = $1 AND is_active = true`

	override := &models.TemplateOverride{}

	err := r.db.QueryRow(query, path).Scan(
		&override.ID,
		&override.Name,
		&override.TemplatePath,
		&override.Content,
		&override.IsActive,
		&override.CreatedAt,
		&override.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template override not found")
		}
		return nil, fmt.Errorf("failed to get template override: %w", err)
	}

	return override, nil
}

// GetAllTemplateOverrides retrieves all template overrides
func (r *ThemeRepository) GetAllTemplateOverrides() ([]*models.TemplateOverride, error) {
	query := `
		SELECT id, name, template_path, content, is_active, created_at, updated_at
		FROM template_overrides
		ORDER BY template_path ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query template overrides: %w", err)
	}
	defer rows.Close()

	var overrides []*models.TemplateOverride
	for rows.Next() {
		override := &models.TemplateOverride{}

		err := rows.Scan(
			&override.ID,
			&override.Name,
			&override.TemplatePath,
			&override.Content,
			&override.IsActive,
			&override.CreatedAt,
			&override.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template override: %w", err)
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}

// UpdateTemplateOverride updates a template override
func (r *ThemeRepository) UpdateTemplateOverride(override *models.TemplateOverride) error {
	query := `
		UPDATE template_overrides
		SET name = $2, template_path = $3, content = $4, is_active = $5, updated_at = $6
		WHERE id = $1`

	result, err := r.db.Exec(
		query,
		override.ID,
		override.Name,
		override.TemplatePath,
		override.Content,
		override.IsActive,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update template override: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template override not found")
	}

	return nil
}

// DeleteTemplateOverride deletes a template override
func (r *ThemeRepository) DeleteTemplateOverride(id uint64) error {
	query := `DELETE FROM template_overrides WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template override: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template override not found")
	}

	return nil
}