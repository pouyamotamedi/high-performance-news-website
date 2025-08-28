package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// WidgetRepository handles widget data operations
type WidgetRepository struct {
	db *sql.DB
}

// NewWidgetRepository creates a new widget repository
func NewWidgetRepository(db *sql.DB) *WidgetRepository {
	return &WidgetRepository{db: db}
}

// Create creates a new widget
func (r *WidgetRepository) Create(widget *models.Widget) (*models.Widget, error) {
	configJSON, err := json.Marshal(widget.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO widgets (name, type, title, description, config, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err = r.db.QueryRow(
		query,
		widget.Name,
		widget.Type,
		widget.Title,
		widget.Description,
		configJSON,
		widget.IsActive,
		widget.SortOrder,
		now,
		now,
	).Scan(&widget.ID, &widget.CreatedAt, &widget.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create widget: %w", err)
	}

	return widget, nil
}

// GetByID retrieves a widget by ID
func (r *WidgetRepository) GetByID(id uint64) (*models.Widget, error) {
	query := `
		SELECT id, name, type, title, description, config, is_active, sort_order, created_at, updated_at
		FROM widgets
		WHERE id = $1`

	widget := &models.Widget{}
	var configJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&widget.ID,
		&widget.Name,
		&widget.Type,
		&widget.Title,
		&widget.Description,
		&configJSON,
		&widget.IsActive,
		&widget.SortOrder,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("widget not found")
		}
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}

	if err := json.Unmarshal(configJSON, &widget.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return widget, nil
}

// GetAll retrieves all widgets
func (r *WidgetRepository) GetAll() ([]*models.Widget, error) {
	query := `
		SELECT id, name, type, title, description, config, is_active, sort_order, created_at, updated_at
		FROM widgets
		ORDER BY sort_order ASC, name ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query widgets: %w", err)
	}
	defer rows.Close()

	var widgets []*models.Widget
	for rows.Next() {
		widget := &models.Widget{}
		var configJSON []byte

		err := rows.Scan(
			&widget.ID,
			&widget.Name,
			&widget.Type,
			&widget.Title,
			&widget.Description,
			&configJSON,
			&widget.IsActive,
			&widget.SortOrder,
			&widget.CreatedAt,
			&widget.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan widget: %w", err)
		}

		if err := json.Unmarshal(configJSON, &widget.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		widgets = append(widgets, widget)
	}

	return widgets, nil
}

// GetByType retrieves widgets by type
func (r *WidgetRepository) GetByType(widgetType models.WidgetType) ([]*models.Widget, error) {
	query := `
		SELECT id, name, type, title, description, config, is_active, sort_order, created_at, updated_at
		FROM widgets
		WHERE type = $1 AND is_active = true
		ORDER BY sort_order ASC, name ASC`

	rows, err := r.db.Query(query, widgetType)
	if err != nil {
		return nil, fmt.Errorf("failed to query widgets by type: %w", err)
	}
	defer rows.Close()

	var widgets []*models.Widget
	for rows.Next() {
		widget := &models.Widget{}
		var configJSON []byte

		err := rows.Scan(
			&widget.ID,
			&widget.Name,
			&widget.Type,
			&widget.Title,
			&widget.Description,
			&configJSON,
			&widget.IsActive,
			&widget.SortOrder,
			&widget.CreatedAt,
			&widget.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan widget: %w", err)
		}

		if err := json.Unmarshal(configJSON, &widget.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		widgets = append(widgets, widget)
	}

	return widgets, nil
}

// Update updates a widget
func (r *WidgetRepository) Update(widget *models.Widget) error {
	configJSON, err := json.Marshal(widget.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE widgets
		SET name = $2, type = $3, title = $4, description = $5, config = $6, 
		    is_active = $7, sort_order = $8, updated_at = $9
		WHERE id = $1`

	result, err := r.db.Exec(
		query,
		widget.ID,
		widget.Name,
		widget.Type,
		widget.Title,
		widget.Description,
		configJSON,
		widget.IsActive,
		widget.SortOrder,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update widget: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("widget not found")
	}

	return nil
}

// Delete deletes a widget
func (r *WidgetRepository) Delete(id uint64) error {
	query := `DELETE FROM widgets WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete widget: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("widget not found")
	}

	return nil
}

// CreatePlacement creates a new widget placement
func (r *WidgetRepository) CreatePlacement(placement *models.WidgetPlacement) (*models.WidgetPlacement, error) {
	query := `
		INSERT INTO widget_placements (widget_id, page_type, zone, position, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		placement.WidgetID,
		placement.PageType,
		placement.Zone,
		placement.Position,
		placement.IsActive,
		now,
		now,
	).Scan(&placement.ID, &placement.CreatedAt, &placement.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create widget placement: %w", err)
	}

	return placement, nil
}

// GetPlacementsByPage retrieves widget placements for a specific page and zone
func (r *WidgetRepository) GetPlacementsByPage(pageType models.PageType, zone models.WidgetZone) ([]*models.WidgetPlacement, error) {
	query := `
		SELECT wp.id, wp.widget_id, wp.page_type, wp.zone, wp.position, wp.is_active, 
		       wp.created_at, wp.updated_at,
		       w.id, w.name, w.type, w.title, w.description, w.config, w.is_active, 
		       w.sort_order, w.created_at, w.updated_at
		FROM widget_placements wp
		JOIN widgets w ON wp.widget_id = w.id
		WHERE (wp.page_type = $1 OR wp.page_type = $2) AND wp.zone = $3 
		      AND wp.is_active = true AND w.is_active = true
		ORDER BY wp.position ASC, w.sort_order ASC`

	rows, err := r.db.Query(query, pageType, models.PageTypeGlobal, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to query widget placements: %w", err)
	}
	defer rows.Close()

	var placements []*models.WidgetPlacement
	for rows.Next() {
		placement := &models.WidgetPlacement{}
		widget := &models.Widget{}
		var configJSON []byte

		err := rows.Scan(
			&placement.ID,
			&placement.WidgetID,
			&placement.PageType,
			&placement.Zone,
			&placement.Position,
			&placement.IsActive,
			&placement.CreatedAt,
			&placement.UpdatedAt,
			&widget.ID,
			&widget.Name,
			&widget.Type,
			&widget.Title,
			&widget.Description,
			&configJSON,
			&widget.IsActive,
			&widget.SortOrder,
			&widget.CreatedAt,
			&widget.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan widget placement: %w", err)
		}

		if err := json.Unmarshal(configJSON, &widget.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		placement.Widget = widget
		placements = append(placements, placement)
	}

	return placements, nil
}

// UpdatePlacement updates a widget placement
func (r *WidgetRepository) UpdatePlacement(placement *models.WidgetPlacement) error {
	query := `
		UPDATE widget_placements
		SET widget_id = $2, page_type = $3, zone = $4, position = $5, 
		    is_active = $6, updated_at = $7
		WHERE id = $1`

	result, err := r.db.Exec(
		query,
		placement.ID,
		placement.WidgetID,
		placement.PageType,
		placement.Zone,
		placement.Position,
		placement.IsActive,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update widget placement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("widget placement not found")
	}

	return nil
}

// DeletePlacement deletes a widget placement
func (r *WidgetRepository) DeletePlacement(id uint64) error {
	query := `DELETE FROM widget_placements WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete widget placement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("widget placement not found")
	}

	return nil
}

// UpdatePlacementPositions updates the positions of multiple widget placements
func (r *WidgetRepository) UpdatePlacementPositions(placements []*models.WidgetPlacement) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `UPDATE widget_placements SET position = $2, updated_at = $3 WHERE id = $1`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, placement := range placements {
		_, err := stmt.Exec(placement.ID, placement.Position, now)
		if err != nil {
			return fmt.Errorf("failed to update placement position: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}