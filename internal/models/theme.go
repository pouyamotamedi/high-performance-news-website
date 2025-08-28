package models

import (
	"encoding/json"
	"time"
)

// Theme represents a website theme configuration
type Theme struct {
	ID          uint64                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	IsDefault   bool                   `json:"is_default" db:"is_default"`
	Config      map[string]interface{} `json:"config" db:"config"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ThemeConfig holds the theme configuration
type ThemeConfig struct {
	// Colors
	Colors ColorScheme `json:"colors"`
	
	// Typography
	Typography Typography `json:"typography"`
	
	// Layout
	Layout Layout `json:"layout"`
	
	// Logo and Branding
	Branding Branding `json:"branding"`
	
	// Custom CSS
	CustomCSS string `json:"custom_css"`
	
	// Custom JavaScript
	CustomJS string `json:"custom_js"`
}

// ColorScheme defines the color palette for the theme
type ColorScheme struct {
	Primary     string `json:"primary"`
	Secondary   string `json:"secondary"`
	Accent      string `json:"accent"`
	Background  string `json:"background"`
	Surface     string `json:"surface"`
	Text        string `json:"text"`
	TextMuted   string `json:"text_muted"`
	Border      string `json:"border"`
	Success     string `json:"success"`
	Warning     string `json:"warning"`
	Error       string `json:"error"`
	Info        string `json:"info"`
}

// Typography defines the font settings
type Typography struct {
	FontFamily      string  `json:"font_family"`
	HeadingFont     string  `json:"heading_font"`
	BaseFontSize    string  `json:"base_font_size"`
	LineHeight      float64 `json:"line_height"`
	HeadingWeight   string  `json:"heading_weight"`
	BodyWeight      string  `json:"body_weight"`
	LetterSpacing   string  `json:"letter_spacing"`
}

// Layout defines the layout settings
type Layout struct {
	MaxWidth        string `json:"max_width"`
	SidebarWidth    string `json:"sidebar_width"`
	HeaderHeight    string `json:"header_height"`
	FooterHeight    string `json:"footer_height"`
	BorderRadius    string `json:"border_radius"`
	Spacing         string `json:"spacing"`
	GridColumns     int    `json:"grid_columns"`
	ShowSidebar     bool   `json:"show_sidebar"`
	SidebarPosition string `json:"sidebar_position"` // left, right
	HeaderStyle     string `json:"header_style"`     // fixed, static, sticky
	FooterStyle     string `json:"footer_style"`     // fixed, static
}

// Branding defines the branding elements
type Branding struct {
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description"`
	LogoURL         string `json:"logo_url"`
	FaviconURL      string `json:"favicon_url"`
	ShowSiteName    bool   `json:"show_site_name"`
	ShowDescription bool   `json:"show_description"`
}

// TemplateOverride represents a custom template override
type TemplateOverride struct {
	ID           uint64    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	TemplatePath string    `json:"template_path" db:"template_path"`
	Content      string    `json:"content" db:"content"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// GetConfig returns the theme configuration as a typed struct
func (t *Theme) GetConfig() (*ThemeConfig, error) {
	configBytes, err := json.Marshal(t.Config)
	if err != nil {
		return nil, err
	}
	
	var config ThemeConfig
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SetConfig sets the theme configuration from a typed struct
func (t *Theme) SetConfig(config *ThemeConfig) error {
	configMap := make(map[string]interface{})
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		return err
	}
	
	t.Config = configMap
	return nil
}

// Validate validates the theme data
func (t *Theme) Validate() error {
	if t.Name == "" {
		return NewValidationError("name", "Theme name is required")
	}
	
	return nil
}

// Validate validates the template override data
func (to *TemplateOverride) Validate() error {
	if to.Name == "" {
		return NewValidationError("name", "Template override name is required")
	}
	
	if to.TemplatePath == "" {
		return NewValidationError("template_path", "Template path is required")
	}
	
	if to.Content == "" {
		return NewValidationError("content", "Template content is required")
	}
	
	return nil
}

// GetDefaultThemeConfig returns a default theme configuration
func GetDefaultThemeConfig() *ThemeConfig {
	return &ThemeConfig{
		Colors: ColorScheme{
			Primary:     "#3b82f6",
			Secondary:   "#64748b",
			Accent:      "#f59e0b",
			Background:  "#ffffff",
			Surface:     "#f8fafc",
			Text:        "#1e293b",
			TextMuted:   "#64748b",
			Border:      "#e2e8f0",
			Success:     "#10b981",
			Warning:     "#f59e0b",
			Error:       "#ef4444",
			Info:        "#3b82f6",
		},
		Typography: Typography{
			FontFamily:      "Inter, system-ui, sans-serif",
			HeadingFont:     "Inter, system-ui, sans-serif",
			BaseFontSize:    "16px",
			LineHeight:      1.6,
			HeadingWeight:   "600",
			BodyWeight:      "400",
			LetterSpacing:   "0",
		},
		Layout: Layout{
			MaxWidth:        "1200px",
			SidebarWidth:    "300px",
			HeaderHeight:    "80px",
			FooterHeight:    "auto",
			BorderRadius:    "8px",
			Spacing:         "1rem",
			GridColumns:     12,
			ShowSidebar:     true,
			SidebarPosition: "right",
			HeaderStyle:     "sticky",
			FooterStyle:     "static",
		},
		Branding: Branding{
			SiteName:        "News Website",
			SiteDescription: "Your trusted source for news",
			LogoURL:         "",
			FaviconURL:      "",
			ShowSiteName:    true,
			ShowDescription: true,
		},
		CustomCSS: "",
		CustomJS:  "",
	}
}