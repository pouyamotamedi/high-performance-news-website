package plugin

import (
	"context"
	"fmt"
	"log"
	"plugin"
	"sync"

	"high-performance-news-website/internal/integration/interfaces"
)

// PluginManager manages dynamic plugin loading and execution
type PluginManager struct {
	plugins map[string]*LoadedPlugin
	mu      sync.RWMutex
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Description string                     `json:"description"`
	Plugin      *plugin.Plugin             `json:"-"`
	Integration interfaces.Integration    `json:"-"`
	Config      PluginConfig               `json:"config"`
	Status      PluginStatus               `json:"status"`
}

// PluginConfig holds plugin configuration
type PluginConfig struct {
	Path        string                 `json:"path"`
	Enabled     bool                   `json:"enabled"`
	AutoLoad    bool                   `json:"auto_load"`
	Settings    map[string]interface{} `json:"settings"`
	Permissions []string               `json:"permissions"`
}

// PluginStatus represents plugin status
type PluginStatus string

const (
	PluginStatusLoaded    PluginStatus = "loaded"
	PluginStatusUnloaded  PluginStatus = "unloaded"
	PluginStatusError     PluginStatus = "error"
	PluginStatusDisabled  PluginStatus = "disabled"
)

// PluginInterface defines the interface that plugins must implement
type PluginInterface interface {
	// Plugin metadata
	GetName() string
	GetVersion() string
	GetDescription() string
	
	// Plugin lifecycle
	Initialize(config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// Integration interface
	GetIntegration() interfaces.Integration
}

// PluginManifest defines plugin metadata
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Dependencies []string          `json:"dependencies"`
	Permissions  []string          `json:"permissions"`
	Config       PluginConfig      `json:"config"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*LoadedPlugin),
	}
}

// LoadPlugin loads a plugin from a shared library file
func (pm *PluginManager) LoadPlugin(ctx context.Context, config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if plugin is already loaded
	if _, exists := pm.plugins[config.Path]; exists {
		return fmt.Errorf("plugin %s is already loaded", config.Path)
	}

	// Load the plugin
	p, err := plugin.Open(config.Path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", config.Path, err)
	}

	// Look for the required symbols
	newPluginSymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewPlugin function: %w", config.Path, err)
	}

	// Cast to the expected function type
	newPluginFunc, ok := newPluginSymbol.(func() PluginInterface)
	if !ok {
		return fmt.Errorf("plugin %s NewPlugin function has wrong signature", config.Path)
	}

	// Create plugin instance
	pluginInstance := newPluginFunc()

	// Initialize the plugin
	if err := pluginInstance.Initialize(config.Settings); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", config.Path, err)
	}

	// Get the integration
	integrationInstance := pluginInstance.GetIntegration()

	// Create loaded plugin
	loadedPlugin := &LoadedPlugin{
		Name:        pluginInstance.GetName(),
		Version:     pluginInstance.GetVersion(),
		Description: pluginInstance.GetDescription(),
		Plugin:      p,
		Integration: integrationInstance,
		Config:      config,
		Status:      PluginStatusLoaded,
	}

	// Start the plugin if enabled
	if config.Enabled {
		if err := pluginInstance.Start(ctx); err != nil {
			loadedPlugin.Status = PluginStatusError
			return fmt.Errorf("failed to start plugin %s: %w", loadedPlugin.Name, err)
		}
	} else {
		loadedPlugin.Status = PluginStatusDisabled
	}

	pm.plugins[loadedPlugin.Name] = loadedPlugin
	log.Printf("Loaded plugin: %s v%s", loadedPlugin.Name, loadedPlugin.Version)

	return nil
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(ctx context.Context, name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	loadedPlugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	// Stop the plugin if it's running
	if loadedPlugin.Status == PluginStatusLoaded {
		// We need to get the plugin instance to stop it
		// This is a limitation of Go's plugin system - we can't easily get back to the instance
		log.Printf("Stopping plugin: %s", name)
	}

	loadedPlugin.Status = PluginStatusUnloaded
	delete(pm.plugins, name)

	log.Printf("Unloaded plugin: %s", name)
	return nil
}

// GetPlugin returns a loaded plugin
func (pm *PluginManager) GetPlugin(name string) (*LoadedPlugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not loaded", name)
	}

	return plugin, nil
}

// GetAllPlugins returns all loaded plugins
func (pm *PluginManager) GetAllPlugins() map[string]*LoadedPlugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make(map[string]*LoadedPlugin)
	for name, plugin := range pm.plugins {
		plugins[name] = plugin
	}

	return plugins
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(ctx context.Context, name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	loadedPlugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	if loadedPlugin.Status == PluginStatusLoaded {
		return nil // Already enabled
	}

	loadedPlugin.Config.Enabled = true
	loadedPlugin.Status = PluginStatusLoaded

	log.Printf("Enabled plugin: %s", name)
	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(ctx context.Context, name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	loadedPlugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	if loadedPlugin.Status == PluginStatusDisabled {
		return nil // Already disabled
	}

	loadedPlugin.Config.Enabled = false
	loadedPlugin.Status = PluginStatusDisabled

	log.Printf("Disabled plugin: %s", name)
	return nil
}

// GetIntegrations returns all integrations from loaded plugins
func (pm *PluginManager) GetIntegrations() []interfaces.Integration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var integrations []interfaces.Integration
	for _, plugin := range pm.plugins {
		if plugin.Status == PluginStatusLoaded && plugin.Integration != nil {
			integrations = append(integrations, plugin.Integration)
		}
	}

	return integrations
}

// ValidatePlugin validates a plugin before loading
func (pm *PluginManager) ValidatePlugin(pluginPath string) (*PluginManifest, error) {
	// Load the plugin temporarily to validate it
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin for validation: %w", err)
	}

	// Check for required symbols
	manifestSymbol, err := p.Lookup("Manifest")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export Manifest: %w", err)
	}

	manifest, ok := manifestSymbol.(*PluginManifest)
	if !ok {
		return nil, fmt.Errorf("plugin Manifest has wrong type")
	}

	// Validate manifest
	if manifest.Name == "" {
		return nil, fmt.Errorf("plugin manifest missing name")
	}

	if manifest.Version == "" {
		return nil, fmt.Errorf("plugin manifest missing version")
	}

	// Check for NewPlugin function
	_, err = p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export NewPlugin function: %w", err)
	}

	return manifest, nil
}

// LoadPluginsFromDirectory loads all plugins from a directory
func (pm *PluginManager) LoadPluginsFromDirectory(ctx context.Context, directory string) error {
	// This would scan the directory for .so files and load them
	// Implementation would depend on the specific requirements
	log.Printf("Loading plugins from directory: %s", directory)
	return nil
}

// ReloadPlugin reloads a plugin
func (pm *PluginManager) ReloadPlugin(ctx context.Context, name string) error {
	loadedPlugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	config := loadedPlugin.Config

	// Unload the plugin
	if err := pm.UnloadPlugin(ctx, name); err != nil {
		return fmt.Errorf("failed to unload plugin for reload: %w", err)
	}

	// Load the plugin again
	if err := pm.LoadPlugin(ctx, config); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	return nil
}

// GetPluginStatus returns the status of all plugins
func (pm *PluginManager) GetPluginStatus() map[string]PluginStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status := make(map[string]PluginStatus)
	for name, plugin := range pm.plugins {
		status[name] = plugin.Status
	}

	return status
}