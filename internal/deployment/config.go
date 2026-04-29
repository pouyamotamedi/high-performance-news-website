package deployment

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the deployment configuration
type Config struct {
	Servers map[string]ServerConfig `yaml:"servers"`
	App     AppConfig               `yaml:"app"`
	SSH     SSHConfig               `yaml:"ssh"`
	Deploy  DeployConfig            `yaml:"deploy"`
}

// ServerConfig represents configuration for a specific server
type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	KeyFile  string `yaml:"key_file"`
	Password string `yaml:"password,omitempty"`
}

// AppConfig represents application-specific configuration
type AppConfig struct {
	Name        string            `yaml:"name"`
	Binary      string            `yaml:"binary"`
	ConfigFile  string            `yaml:"config_file"`
	Port        int               `yaml:"port"`
	HealthPath  string            `yaml:"health_path"`
	Environment map[string]string `yaml:"environment"`
	Dependencies []string         `yaml:"dependencies"`
}

// SSHConfig represents SSH connection configuration
type SSHConfig struct {
	Timeout         time.Duration `yaml:"timeout"`
	ConnectRetries  int           `yaml:"connect_retries"`
	CommandTimeout  time.Duration `yaml:"command_timeout"`
	KeepAlive       time.Duration `yaml:"keep_alive"`
}

// DeployConfig represents deployment-specific configuration
type DeployConfig struct {
	Strategy         string        `yaml:"strategy"` // blue-green, rolling
	HealthCheckURL   string        `yaml:"health_check_url"`
	HealthTimeout    time.Duration `yaml:"health_timeout"`
	RollbackTimeout  time.Duration `yaml:"rollback_timeout"`
	BackupRetention  int           `yaml:"backup_retention"`
	PreDeployHooks   []string      `yaml:"pre_deploy_hooks"`
	PostDeployHooks  []string      `yaml:"post_deploy_hooks"`
	ValidationChecks []string      `yaml:"validation_checks"`
}

// LoadConfig loads deployment configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.SSH.Timeout == 0 {
		config.SSH.Timeout = 30 * time.Second
	}
	if config.SSH.ConnectRetries == 0 {
		config.SSH.ConnectRetries = 3
	}
	if config.SSH.CommandTimeout == 0 {
		config.SSH.CommandTimeout = 5 * time.Minute
	}
	if config.SSH.KeepAlive == 0 {
		config.SSH.KeepAlive = 30 * time.Second
	}
	if config.Deploy.HealthTimeout == 0 {
		config.Deploy.HealthTimeout = 2 * time.Minute
	}
	if config.Deploy.RollbackTimeout == 0 {
		config.Deploy.RollbackTimeout = 5 * time.Minute
	}
	if config.Deploy.BackupRetention == 0 {
		config.Deploy.BackupRetention = 5
	}
	if config.Deploy.Strategy == "" {
		config.Deploy.Strategy = "blue-green"
	}

	return &config, nil
}

// GetServerConfig returns the configuration for a specific server
func (c *Config) GetServerConfig(name string) (*ServerConfig, error) {
	server, exists := c.Servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found in configuration", name)
	}
	return &server, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.App.Binary == "" {
		return fmt.Errorf("app.binary is required")
	}
	if c.App.Port == 0 {
		return fmt.Errorf("app.port is required")
	}

	for name, server := range c.Servers {
		if server.Host == "" {
			return fmt.Errorf("server '%s': host is required", name)
		}
		if server.User == "" {
			return fmt.Errorf("server '%s': user is required", name)
		}
		if server.KeyFile == "" && server.Password == "" {
			return fmt.Errorf("server '%s': either key_file or password is required", name)
		}
	}

	return nil
}