package repositories

import (
	"context"
	"database/sql"
	"time"
)

// CodeInjection represents code injection settings
type CodeInjection struct {
	ID              int       `db:"id" json:"id"`
	HeaderCode      string    `db:"header_code" json:"header_code"`           // Code for <head> section
	BodyStartCode   string    `db:"body_start_code" json:"body_start_code"`   // Code right after <body>
	FooterCode      string    `db:"footer_code" json:"footer_code"`           // Code before </body>
	CustomCSS       string    `db:"custom_css" json:"custom_css"`             // Custom CSS
	CustomJS        string    `db:"custom_js" json:"custom_js"`               // Custom JavaScript
	HeaderEnabled   bool      `db:"header_enabled" json:"header_enabled"`
	BodyStartEnabled bool     `db:"body_start_enabled" json:"body_start_enabled"`
	FooterEnabled   bool      `db:"footer_enabled" json:"footer_enabled"`
	CSSEnabled      bool      `db:"css_enabled" json:"css_enabled"`
	JSEnabled       bool      `db:"js_enabled" json:"js_enabled"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// CodeInjectionRepository handles code injection storage
type CodeInjectionRepository struct {
	db *sql.DB
}

// NewCodeInjectionRepository creates a new code injection repository
func NewCodeInjectionRepository(db *sql.DB) *CodeInjectionRepository {
	return &CodeInjectionRepository{db: db}
}

// InitTable creates the code_injection table if it doesn't exist
func (r *CodeInjectionRepository) InitTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS code_injection (
			id INTEGER PRIMARY KEY DEFAULT 1,
			header_code TEXT DEFAULT '',
			body_start_code TEXT DEFAULT '',
			footer_code TEXT DEFAULT '',
			custom_css TEXT DEFAULT '',
			custom_js TEXT DEFAULT '',
			header_enabled BOOLEAN DEFAULT false,
			body_start_enabled BOOLEAN DEFAULT false,
			footer_enabled BOOLEAN DEFAULT false,
			css_enabled BOOLEAN DEFAULT false,
			js_enabled BOOLEAN DEFAULT false,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT single_row CHECK (id = 1)
		)`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	// Insert default row if not exists
	insertQuery := `
		INSERT INTO code_injection (id, header_code, body_start_code, footer_code, custom_css, custom_js)
		VALUES (1, '', '', '', '', '')
		ON CONFLICT (id) DO NOTHING`
	_, err = r.db.ExecContext(ctx, insertQuery)
	return err
}

// Get retrieves current code injection settings
func (r *CodeInjectionRepository) Get(ctx context.Context) (*CodeInjection, error) {
	query := `
		SELECT id, header_code, body_start_code, footer_code, custom_css, custom_js,
		       header_enabled, body_start_enabled, footer_enabled, css_enabled, js_enabled, updated_at
		FROM code_injection
		WHERE id = 1`

	var ci CodeInjection
	err := r.db.QueryRowContext(ctx, query).Scan(
		&ci.ID, &ci.HeaderCode, &ci.BodyStartCode, &ci.FooterCode,
		&ci.CustomCSS, &ci.CustomJS, &ci.HeaderEnabled, &ci.BodyStartEnabled,
		&ci.FooterEnabled, &ci.CSSEnabled, &ci.JSEnabled, &ci.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return empty settings if no row exists
		return &CodeInjection{
			ID:               1,
			HeaderEnabled:    false,
			BodyStartEnabled: false,
			FooterEnabled:    false,
			CSSEnabled:       false,
			JSEnabled:        false,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &ci, nil
}

// UpdateHeaderCode updates header code
func (r *CodeInjectionRepository) UpdateHeaderCode(ctx context.Context, code string, enabled bool) error {
	query := `
		UPDATE code_injection 
		SET header_code = $1, header_enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`
	_, err := r.db.ExecContext(ctx, query, code, enabled)
	return err
}

// UpdateBodyStartCode updates body start code (right after <body>)
func (r *CodeInjectionRepository) UpdateBodyStartCode(ctx context.Context, code string, enabled bool) error {
	query := `
		UPDATE code_injection 
		SET body_start_code = $1, body_start_enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`
	_, err := r.db.ExecContext(ctx, query, code, enabled)
	return err
}

// UpdateFooterCode updates footer code
func (r *CodeInjectionRepository) UpdateFooterCode(ctx context.Context, code string, enabled bool) error {
	query := `
		UPDATE code_injection 
		SET footer_code = $1, footer_enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`
	_, err := r.db.ExecContext(ctx, query, code, enabled)
	return err
}

// UpdateCustomCSS updates custom CSS
func (r *CodeInjectionRepository) UpdateCustomCSS(ctx context.Context, css string, enabled bool) error {
	query := `
		UPDATE code_injection 
		SET custom_css = $1, css_enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`
	_, err := r.db.ExecContext(ctx, query, css, enabled)
	return err
}

// UpdateCustomJS updates custom JavaScript
func (r *CodeInjectionRepository) UpdateCustomJS(ctx context.Context, js string, enabled bool) error {
	query := `
		UPDATE code_injection 
		SET custom_js = $1, js_enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`
	_, err := r.db.ExecContext(ctx, query, js, enabled)
	return err
}

// GetEnabledCode returns only enabled code sections for rendering
func (r *CodeInjectionRepository) GetEnabledCode(ctx context.Context) (headerCode, bodyStartCode, footerCode, customCSS, customJS string, err error) {
	ci, err := r.Get(ctx)
	if err != nil {
		return "", "", "", "", "", err
	}

	if ci.HeaderEnabled {
		headerCode = ci.HeaderCode
	}
	if ci.BodyStartEnabled {
		bodyStartCode = ci.BodyStartCode
	}
	if ci.FooterEnabled {
		footerCode = ci.FooterCode
	}
	if ci.CSSEnabled {
		customCSS = ci.CustomCSS
	}
	if ci.JSEnabled {
		customJS = ci.CustomJS
	}

	return headerCode, bodyStartCode, footerCode, customCSS, customJS, nil
}
