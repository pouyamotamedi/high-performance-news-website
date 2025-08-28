// Theme Management JavaScript
class ThemeManager {
    constructor() {
        this.themes = [];
        this.activeTheme = null;
        this.currentConfig = null;
        this.init();
    }

    async init() {
        await this.loadThemes();
        await this.loadActiveTheme();
        this.setupEventListeners();
        this.setupTabs();
        this.updatePreview();
    }

    async loadThemes() {
        try {
            const response = await fetch('/api/v1/admin/themes');
            const data = await response.json();
            this.themes = data.themes || [];
            this.renderThemes();
        } catch (error) {
            console.error('Failed to load themes:', error);
            this.showError('Failed to load themes');
        }
    }

    async loadActiveTheme() {
        try {
            const response = await fetch('/api/v1/admin/themes/active');
            this.activeTheme = await response.json();
            this.currentConfig = this.activeTheme.config;
            this.populateConfigForm();
        } catch (error) {
            console.error('Failed to load active theme:', error);
            this.showError('Failed to load active theme');
        }
    }

    renderThemes() {
        const themeList = document.getElementById('theme-list');
        themeList.innerHTML = '';

        this.themes.forEach(theme => {
            const themeElement = this.createThemeElement(theme);
            themeList.appendChild(themeElement);
        });
    }

    createThemeElement(theme) {
        const div = document.createElement('div');
        div.className = `theme-item ${theme.is_active ? 'active' : ''}`;
        div.dataset.themeId = theme.id;

        const statusBadges = [];
        if (theme.is_active) statusBadges.push('<span class="status-badge status-active">Active</span>');
        if (theme.is_default) statusBadges.push('<span class="status-badge status-default">Default</span>');

        div.innerHTML = `
            <h3 class="theme-name">${this.escapeHtml(theme.name)}</h3>
            <p class="theme-description">${this.escapeHtml(theme.description || '')}</p>
            <div class="theme-status">
                ${statusBadges.join('')}
            </div>
            <div class="theme-actions">
                ${!theme.is_active ? `<button class="btn btn-primary" onclick="themeManager.activateTheme(${theme.id})">Activate</button>` : ''}
                <button class="btn btn-secondary" onclick="themeManager.editTheme(${theme.id})">Edit</button>
                ${!theme.is_active && !theme.is_default ? `<button class="btn btn-danger" onclick="themeManager.deleteTheme(${theme.id})">Delete</button>` : ''}
            </div>
        `;

        return div;
    }

    setupEventListeners() {
        // Create theme button
        document.getElementById('create-theme-btn').addEventListener('click', () => {
            this.createNewTheme();
        });

        // Save theme button
        document.getElementById('save-theme-btn').addEventListener('click', () => {
            this.saveThemeChanges();
        });

        // Preview theme button
        document.getElementById('preview-theme-btn').addEventListener('click', () => {
            this.updatePreview();
        });

        // Color picker changes
        this.setupColorPickers();

        // Form input changes
        this.setupFormInputs();
    }

    setupColorPickers() {
        const colorInputs = document.querySelectorAll('input[type="color"]');
        colorInputs.forEach(input => {
            input.addEventListener('change', (e) => {
                this.updateConfigValue(e.target.id.replace('-', '_'), e.target.value);
                this.updatePreview();
            });
        });
    }

    setupFormInputs() {
        const inputs = document.querySelectorAll('#typography-tab input, #layout-tab input, #branding-tab input, #custom-tab textarea');
        inputs.forEach(input => {
            input.addEventListener('change', (e) => {
                let value = e.target.value;
                if (e.target.type === 'checkbox') {
                    value = e.target.checked;
                } else if (e.target.type === 'number') {
                    value = parseFloat(e.target.value);
                }
                this.updateConfigValue(e.target.id.replace('-', '_'), value);
                this.updatePreview();
            });
        });
    }

    setupTabs() {
        const tabButtons = document.querySelectorAll('.tab-btn');
        const tabPanes = document.querySelectorAll('.tab-pane');

        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                // Remove active class from all tabs
                tabButtons.forEach(btn => btn.classList.remove('active'));
                tabPanes.forEach(pane => pane.classList.remove('active'));

                // Add active class to clicked tab
                button.classList.add('active');
                const tabId = button.dataset.tab + '-tab';
                document.getElementById(tabId).classList.add('active');
            });
        });
    }

    populateConfigForm() {
        if (!this.currentConfig) return;

        // Colors
        if (this.currentConfig.colors) {
            Object.entries(this.currentConfig.colors).forEach(([key, value]) => {
                const input = document.getElementById(key.replace('_', '-') + '-color');
                if (input) input.value = value;
            });
        }

        // Typography
        if (this.currentConfig.typography) {
            Object.entries(this.currentConfig.typography).forEach(([key, value]) => {
                const input = document.getElementById(key.replace('_', '-'));
                if (input) {
                    if (input.type === 'number') {
                        input.value = parseFloat(value);
                    } else {
                        input.value = value;
                    }
                }
            });
        }

        // Layout
        if (this.currentConfig.layout) {
            Object.entries(this.currentConfig.layout).forEach(([key, value]) => {
                const input = document.getElementById(key.replace('_', '-'));
                if (input) {
                    if (input.type === 'checkbox') {
                        input.checked = value;
                    } else if (input.type === 'number') {
                        input.value = parseFloat(value);
                    } else {
                        input.value = value;
                    }
                }
            });
        }

        // Branding
        if (this.currentConfig.branding) {
            Object.entries(this.currentConfig.branding).forEach(([key, value]) => {
                const input = document.getElementById(key.replace('_', '-'));
                if (input) {
                    if (input.type === 'checkbox') {
                        input.checked = value;
                    } else {
                        input.value = value;
                    }
                }
            });
        }

        // Custom CSS/JS
        if (this.currentConfig.custom_css) {
            document.getElementById('custom-css').value = this.currentConfig.custom_css;
        }
        if (this.currentConfig.custom_js) {
            document.getElementById('custom-js').value = this.currentConfig.custom_js;
        }
    }

    updateConfigValue(path, value) {
        if (!this.currentConfig) this.currentConfig = {};

        const pathParts = path.split('_');
        if (pathParts.length === 2) {
            // Nested config like colors_primary
            const [section, key] = pathParts;
            if (!this.currentConfig[section]) this.currentConfig[section] = {};
            this.currentConfig[section][key] = value;
        } else {
            // Direct config like custom_css
            this.currentConfig[path] = value;
        }
    }

    async updatePreview() {
        if (!this.currentConfig) return;

        try {
            // Generate CSS from current config
            const tempTheme = {
                ...this.activeTheme,
                config: this.currentConfig
            };

            const response = await fetch(`/api/v1/admin/themes/${this.activeTheme.id}/css`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(tempTheme)
            });

            if (response.ok) {
                const css = await response.text();
                this.applyPreviewCSS(css);
                this.generatePreviewHTML();
            }
        } catch (error) {
            console.error('Failed to update preview:', error);
        }
    }

    applyPreviewCSS(css) {
        // Remove existing preview styles
        const existingStyle = document.getElementById('preview-styles');
        if (existingStyle) {
            existingStyle.remove();
        }

        // Add new preview styles
        const style = document.createElement('style');
        style.id = 'preview-styles';
        style.textContent = css;
        document.head.appendChild(style);
    }

    generatePreviewHTML() {
        const preview = document.getElementById('theme-preview');
        const branding = this.currentConfig.branding || {};
        
        preview.innerHTML = `
            <div class="preview-header">
                <div class="preview-nav">
                    <div class="preview-logo">${this.escapeHtml(branding.site_name || 'News Website')}</div>
                    <div class="preview-menu">
                        <a href="#">Home</a>
                        <a href="#">News</a>
                        <a href="#">Sports</a>
                        <a href="#">Technology</a>
                    </div>
                </div>
            </div>
            <div class="preview-content">
                <div class="preview-main">
                    <article class="preview-article">
                        <h3>Breaking News: Sample Article Title</h3>
                        <p>This is a sample article excerpt that demonstrates how your content will look with the current theme configuration. The typography, colors, and spacing are all applied according to your settings.</p>
                    </article>
                    <article class="preview-article">
                        <h3>Technology Update: Another Sample Article</h3>
                        <p>Another sample article to show how multiple articles will appear on your website. This helps you visualize the overall layout and design.</p>
                    </article>
                </div>
                <div class="preview-sidebar">
                    <div class="preview-widget">
                        <h4>Latest Articles</h4>
                        <ul>
                            <li><a href="#">Sample Article 1</a></li>
                            <li><a href="#">Sample Article 2</a></li>
                            <li><a href="#">Sample Article 3</a></li>
                        </ul>
                    </div>
                    <div class="preview-widget">
                        <h4>Categories</h4>
                        <ul>
                            <li><a href="#">Technology</a></li>
                            <li><a href="#">Sports</a></li>
                            <li><a href="#">Politics</a></li>
                        </ul>
                    </div>
                </div>
            </div>
        `;
    }

    async activateTheme(themeId) {
        try {
            const response = await fetch(`/api/v1/admin/themes/${themeId}/activate`, {
                method: 'POST'
            });

            if (response.ok) {
                await this.loadThemes();
                await this.loadActiveTheme();
                this.showSuccess('Theme activated successfully');
            } else {
                const error = await response.json();
                this.showError(error.error || 'Failed to activate theme');
            }
        } catch (error) {
            console.error('Failed to activate theme:', error);
            this.showError('Failed to activate theme');
        }
    }

    async editTheme(themeId) {
        const theme = this.themes.find(t => t.id == themeId);
        if (theme) {
            this.activeTheme = theme;
            this.currentConfig = theme.config;
            this.populateConfigForm();
            this.updatePreview();
        }
    }

    async deleteTheme(themeId) {
        if (!confirm('Are you sure you want to delete this theme?')) return;

        try {
            const response = await fetch(`/api/v1/admin/themes/${themeId}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                await this.loadThemes();
                this.showSuccess('Theme deleted successfully');
            } else {
                const error = await response.json();
                this.showError(error.error || 'Failed to delete theme');
            }
        } catch (error) {
            console.error('Failed to delete theme:', error);
            this.showError('Failed to delete theme');
        }
    }

    async createNewTheme() {
        const name = prompt('Enter theme name:');
        if (!name) return;

        const description = prompt('Enter theme description (optional):') || '';

        try {
            const response = await fetch('/api/v1/admin/themes', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: name,
                    description: description,
                    is_active: false,
                    is_default: false,
                    config: this.getDefaultConfig()
                })
            });

            if (response.ok) {
                await this.loadThemes();
                this.showSuccess('Theme created successfully');
            } else {
                const error = await response.json();
                this.showError(error.error || 'Failed to create theme');
            }
        } catch (error) {
            console.error('Failed to create theme:', error);
            this.showError('Failed to create theme');
        }
    }

    async saveThemeChanges() {
        if (!this.activeTheme) return;

        try {
            const updatedTheme = {
                ...this.activeTheme,
                config: this.currentConfig
            };

            const response = await fetch(`/api/v1/admin/themes/${this.activeTheme.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(updatedTheme)
            });

            if (response.ok) {
                await this.loadThemes();
                this.showSuccess('Theme saved successfully');
            } else {
                const error = await response.json();
                this.showError(error.error || 'Failed to save theme');
            }
        } catch (error) {
            console.error('Failed to save theme:', error);
            this.showError('Failed to save theme');
        }
    }

    getDefaultConfig() {
        return {
            colors: {
                primary: '#3b82f6',
                secondary: '#64748b',
                accent: '#f59e0b',
                background: '#ffffff',
                surface: '#f8fafc',
                text: '#1e293b',
                text_muted: '#64748b',
                border: '#e2e8f0',
                success: '#10b981',
                warning: '#f59e0b',
                error: '#ef4444',
                info: '#3b82f6'
            },
            typography: {
                font_family: 'Inter, system-ui, sans-serif',
                heading_font: 'Inter, system-ui, sans-serif',
                base_font_size: '16px',
                line_height: 1.6,
                heading_weight: '600',
                body_weight: '400',
                letter_spacing: '0'
            },
            layout: {
                max_width: '1200px',
                sidebar_width: '300px',
                header_height: '80px',
                footer_height: 'auto',
                border_radius: '8px',
                spacing: '1rem',
                grid_columns: 12,
                show_sidebar: true,
                sidebar_position: 'right',
                header_style: 'sticky',
                footer_style: 'static'
            },
            branding: {
                site_name: 'News Website',
                site_description: 'Your trusted source for news',
                logo_url: '',
                favicon_url: '',
                show_site_name: true,
                show_description: true
            },
            custom_css: '',
            custom_js: ''
        };
    }

    // Utility methods
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    showSuccess(message) {
        // You can implement a toast notification system here
        alert(message);
    }

    showError(message) {
        // You can implement a toast notification system here
        alert('Error: ' + message);
    }
}

// Initialize theme manager when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.themeManager = new ThemeManager();
});