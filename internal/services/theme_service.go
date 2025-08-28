package services

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// ThemeService handles theme business logic
type ThemeService struct {
	themeRepo    *repositories.ThemeRepository
	cacheService CacheService
	templateDir  string
}

// NewThemeService creates a new theme service
func NewThemeService(
	themeRepo *repositories.ThemeRepository,
	cacheService CacheService,
	templateDir string,
) *ThemeService {
	return &ThemeService{
		themeRepo:    themeRepo,
		cacheService: cacheService,
		templateDir:  templateDir,
	}
}

// CreateTheme creates a new theme
func (s *ThemeService) CreateTheme(theme *models.Theme) (*models.Theme, error) {
	if err := theme.Validate(); err != nil {
		return nil, err
	}

	// If no config is provided, use default
	if theme.Config == nil {
		defaultConfig := models.GetDefaultThemeConfig()
		if err := theme.SetConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to set default config: %w", err)
		}
	}

	createdTheme, err := s.themeRepo.Create(theme)
	if err != nil {
		return nil, fmt.Errorf("failed to create theme: %w", err)
	}

	// Clear theme cache
	s.clearThemeCache()

	return createdTheme, nil
}

// GetTheme retrieves a theme by ID
func (s *ThemeService) GetTheme(id uint64) (*models.Theme, error) {
	return s.themeRepo.GetByID(id)
}

// GetActiveTheme retrieves the active theme
func (s *ThemeService) GetActiveTheme() (*models.Theme, error) {
	cacheKey := "active_theme"
	
	// Try to get from cache first
	if cached, err := s.getCachedTheme(cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	theme, err := s.themeRepo.GetActive()
	if err != nil {
		// If no active theme found, create and activate default theme
		if strings.Contains(err.Error(), "no active theme found") {
			return s.createDefaultTheme()
		}
		return nil, fmt.Errorf("failed to get active theme: %w", err)
	}

	// Cache the result
	s.cacheTheme(cacheKey, theme, 1*time.Hour)

	return theme, nil
}

// GetAllThemes retrieves all themes
func (s *ThemeService) GetAllThemes() ([]*models.Theme, error) {
	return s.themeRepo.GetAll()
}

// UpdateTheme updates a theme
func (s *ThemeService) UpdateTheme(theme *models.Theme) error {
	if err := theme.Validate(); err != nil {
		return err
	}

	if err := s.themeRepo.Update(theme); err != nil {
		return fmt.Errorf("failed to update theme: %w", err)
	}

	// Clear theme cache
	s.clearThemeCache()

	return nil
}

// SetActiveTheme sets a theme as active
func (s *ThemeService) SetActiveTheme(id uint64) error {
	if err := s.themeRepo.SetActive(id); err != nil {
		return fmt.Errorf("failed to set active theme: %w", err)
	}

	// Clear theme cache
	s.clearThemeCache()

	return nil
}

// DeleteTheme deletes a theme
func (s *ThemeService) DeleteTheme(id uint64) error {
	if err := s.themeRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete theme: %w", err)
	}

	// Clear theme cache
	s.clearThemeCache()

	return nil
}

// GenerateCSS generates CSS from theme configuration
func (s *ThemeService) GenerateCSS(theme *models.Theme) (string, error) {
	config, err := theme.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get theme config: %w", err)
	}

	css := `:root {
		/* Colors */
		--color-primary: ` + config.Colors.Primary + `;
		--color-secondary: ` + config.Colors.Secondary + `;
		--color-accent: ` + config.Colors.Accent + `;
		--color-background: ` + config.Colors.Background + `;
		--color-surface: ` + config.Colors.Surface + `;
		--color-text: ` + config.Colors.Text + `;
		--color-text-muted: ` + config.Colors.TextMuted + `;
		--color-border: ` + config.Colors.Border + `;
		--color-success: ` + config.Colors.Success + `;
		--color-warning: ` + config.Colors.Warning + `;
		--color-error: ` + config.Colors.Error + `;
		--color-info: ` + config.Colors.Info + `;

		/* Typography */
		--font-family: ` + config.Typography.FontFamily + `;
		--font-heading: ` + config.Typography.HeadingFont + `;
		--font-size-base: ` + config.Typography.BaseFontSize + `;
		--line-height: ` + fmt.Sprintf("%.2f", config.Typography.LineHeight) + `;
		--font-weight-heading: ` + config.Typography.HeadingWeight + `;
		--font-weight-body: ` + config.Typography.BodyWeight + `;
		--letter-spacing: ` + config.Typography.LetterSpacing + `;

		/* Layout */
		--max-width: ` + config.Layout.MaxWidth + `;
		--sidebar-width: ` + config.Layout.SidebarWidth + `;
		--header-height: ` + config.Layout.HeaderHeight + `;
		--footer-height: ` + config.Layout.FooterHeight + `;
		--border-radius: ` + config.Layout.BorderRadius + `;
		--spacing: ` + config.Layout.Spacing + `;
		--grid-columns: ` + fmt.Sprintf("%d", config.Layout.GridColumns) + `;
	}

	/* Base styles */
	body {
		font-family: var(--font-family);
		font-size: var(--font-size-base);
		line-height: var(--line-height);
		font-weight: var(--font-weight-body);
		color: var(--color-text);
		background-color: var(--color-background);
		letter-spacing: var(--letter-spacing);
	}

	h1, h2, h3, h4, h5, h6 {
		font-family: var(--font-heading);
		font-weight: var(--font-weight-heading);
		color: var(--color-text);
	}

	/* Layout styles */
	.container {
		max-width: var(--max-width);
		margin: 0 auto;
		padding: 0 var(--spacing);
	}

	.header {
		height: var(--header-height);
		background-color: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
	}

	.sidebar {
		width: var(--sidebar-width);
		background-color: var(--color-surface);
	}

	/* Component styles */
	.btn {
		background-color: var(--color-primary);
		color: white;
		border: none;
		border-radius: var(--border-radius);
		padding: calc(var(--spacing) * 0.5) var(--spacing);
		font-family: var(--font-family);
		cursor: pointer;
		transition: background-color 0.2s ease;
	}

	.btn:hover {
		background-color: var(--color-secondary);
	}

	.card {
		background-color: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--border-radius);
		padding: var(--spacing);
	}

	/* Widget styles */
	.widget {
		margin-bottom: calc(var(--spacing) * 2);
	}

	.widget-title {
		color: var(--color-primary);
		margin-bottom: var(--spacing);
		font-size: 1.2em;
	}

	.widget-content {
		color: var(--color-text);
	}

	/* Responsive design */
	@media (max-width: 768px) {
		.sidebar {
			width: 100%;
		}
		
		.container {
			padding: 0 calc(var(--spacing) * 0.5);
		}
	}`

	// Add custom CSS if provided
	if config.CustomCSS != "" {
		css += "\n\n/* Custom CSS */\n" + config.CustomCSS
	}

	return css, nil
}

// CreateTemplateOverride creates a new template override
func (s *ThemeService) CreateTemplateOverride(override *models.TemplateOverride) (*models.TemplateOverride, error) {
	if err := override.Validate(); err != nil {
		return nil, err
	}

	createdOverride, err := s.themeRepo.CreateTemplateOverride(override)
	if err != nil {
		return nil, fmt.Errorf("failed to create template override: %w", err)
	}

	// Clear template cache
	s.clearTemplateCache()

	return createdOverride, nil
}

// GetTemplateOverride retrieves a template override by path
func (s *ThemeService) GetTemplateOverride(path string) (*models.TemplateOverride, error) {
	return s.themeRepo.GetTemplateOverrideByPath(path)
}

// GetAllTemplateOverrides retrieves all template overrides
func (s *ThemeService) GetAllTemplateOverrides() ([]*models.TemplateOverride, error) {
	return s.themeRepo.GetAllTemplateOverrides()
}

// UpdateTemplateOverride updates a template override
func (s *ThemeService) UpdateTemplateOverride(override *models.TemplateOverride) error {
	if err := override.Validate(); err != nil {
		return err
	}

	if err := s.themeRepo.UpdateTemplateOverride(override); err != nil {
		return fmt.Errorf("failed to update template override: %w", err)
	}

	// Clear template cache
	s.clearTemplateCache()

	return nil
}

// DeleteTemplateOverride deletes a template override
func (s *ThemeService) DeleteTemplateOverride(id uint64) error {
	if err := s.themeRepo.DeleteTemplateOverride(id); err != nil {
		return fmt.Errorf("failed to delete template override: %w", err)
	}

	// Clear template cache
	s.clearTemplateCache()

	return nil
}

// GetTemplateContent gets template content, checking for overrides first
func (s *ThemeService) GetTemplateContent(templatePath string) (string, error) {
	// First check for template override
	override, err := s.themeRepo.GetTemplateOverrideByPath(templatePath)
	if err == nil && override != nil {
		return override.Content, nil
	}

	// Fall back to file system template
	fullPath := filepath.Join(s.templateDir, templatePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	return string(content), nil
}

// PreviewTemplate generates a preview of a template with sample data
func (s *ThemeService) PreviewTemplate(templateContent string, templatePath string) (template.HTML, error) {
	// Create a temporary template
	tmpl, err := template.New("preview").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate sample data based on template type
	data := s.generateSampleData(templatePath)

	// Execute template with sample data
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return template.HTML(buf.String()), nil
}

// generateSampleData generates sample data for template previews
func (s *ThemeService) generateSampleData(templatePath string) interface{} {
	switch {
	case strings.Contains(templatePath, "article"):
		return map[string]interface{}{
			"Title":       "Sample Article Title",
			"Content":     "This is sample article content for preview purposes.",
			"Author":      "John Doe",
			"PublishedAt": time.Now(),
			"Category":    "Technology",
			"Tags":        []string{"sample", "preview", "test"},
		}
	case strings.Contains(templatePath, "homepage"):
		return map[string]interface{}{
			"LatestArticles": []map[string]interface{}{
				{
					"Title":       "Latest Article 1",
					"Excerpt":     "This is a sample excerpt...",
					"PublishedAt": time.Now(),
				},
				{
					"Title":       "Latest Article 2",
					"Excerpt":     "Another sample excerpt...",
					"PublishedAt": time.Now().Add(-1 * time.Hour),
				},
			},
			"FeaturedArticle": map[string]interface{}{
				"Title":   "Featured Article",
				"Excerpt": "This is the featured article excerpt...",
			},
		}
	case strings.Contains(templatePath, "category"):
		return map[string]interface{}{
			"Category": map[string]interface{}{
				"Name":        "Sample Category",
				"Description": "This is a sample category description.",
			},
			"Articles": []map[string]interface{}{
				{
					"Title":       "Category Article 1",
					"Excerpt":     "Sample excerpt for category article...",
					"PublishedAt": time.Now(),
				},
			},
		}
	default:
		return map[string]interface{}{
			"Title":   "Sample Page",
			"Content": "This is sample content for preview.",
		}
	}
}

// createDefaultTheme creates and activates a default theme
func (s *ThemeService) createDefaultTheme() (*models.Theme, error) {
	defaultTheme := &models.Theme{
		Name:        "Default Theme",
		Description: "Default theme for the news website",
		IsActive:    true,
		IsDefault:   true,
	}

	defaultConfig := models.GetDefaultThemeConfig()
	if err := defaultTheme.SetConfig(defaultConfig); err != nil {
		return nil, fmt.Errorf("failed to set default config: %w", err)
	}

	createdTheme, err := s.themeRepo.Create(defaultTheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create default theme: %w", err)
	}

	return createdTheme, nil
}

// Helper methods for caching
func (s *ThemeService) clearThemeCache() {
	if s.cacheService != nil {
		s.cacheService.DeletePattern("theme:*")
		s.cacheService.Delete("active_theme")
	}
}

func (s *ThemeService) clearTemplateCache() {
	if s.cacheService != nil {
		s.cacheService.DeletePattern("template:*")
	}
}

func (s *ThemeService) getCachedTheme(cacheKey string) (*models.Theme, error) {
	// Implementation would depend on your cache service
	// This is a placeholder
	return nil, fmt.Errorf("not implemented")
}

func (s *ThemeService) cacheTheme(cacheKey string, theme *models.Theme, ttl time.Duration) {
	// Implementation would depend on your cache service
	// This is a placeholder
}