package deployment

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigDefaults(t *testing.T) {
	configContent := `
servers:
  test:
    host: "localhost"
    user: "test"
    key_file: "/path/to/key"

app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test SSH defaults
	assert.Equal(t, 30*time.Second, config.SSH.Timeout)
	assert.Equal(t, 3, config.SSH.ConnectRetries)
	assert.Equal(t, 5*time.Minute, config.SSH.CommandTimeout)
	assert.Equal(t, 30*time.Second, config.SSH.KeepAlive)

	// Test Deploy defaults
	assert.Equal(t, 2*time.Minute, config.Deploy.HealthTimeout)
	assert.Equal(t, 5*time.Minute, config.Deploy.RollbackTimeout)
	assert.Equal(t, 5, config.Deploy.BackupRetention)
	assert.Equal(t, "blue-green", config.Deploy.Strategy)
}

func TestLoadConfigCustomValues(t *testing.T) {
	configContent := `
servers:
  production:
    host: "prod.example.com"
    port: 2222
    user: "deploy"
    key_file: "/home/user/.ssh/id_rsa"
  staging:
    host: "staging.example.com"
    user: "deploy"
    password: "secret"

app:
  name: "news-website"
  binary: "./news-server"
  config_file: "./config.yaml"
  port: 8080
  health_path: "/health"
  environment:
    ENV: "production"
    LOG_LEVEL: "info"
    DATABASE_URL: "postgres://user:pass@localhost/db"
  dependencies:
    - "postgresql-client"
    - "redis-tools"

ssh:
  timeout: 45s
  connect_retries: 5
  command_timeout: 10m
  keep_alive: 60s

deploy:
  strategy: "rolling"
  health_check_url: "http://localhost:8080/api/health"
  health_timeout: 3m
  rollback_timeout: 10m
  backup_retention: 10
  pre_deploy_hooks:
    - "echo 'Starting deployment'"
    - "sudo systemctl status nginx"
  post_deploy_hooks:
    - "echo 'Deployment completed'"
    - "sudo systemctl reload nginx"
  validation_checks:
    - "which curl"
    - "systemctl --version"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test servers
	assert.Len(t, config.Servers, 2)
	
	prodServer, err := config.GetServerConfig("production")
	require.NoError(t, err)
	assert.Equal(t, "prod.example.com", prodServer.Host)
	assert.Equal(t, 2222, prodServer.Port)
	assert.Equal(t, "deploy", prodServer.User)
	assert.Equal(t, "/home/user/.ssh/id_rsa", prodServer.KeyFile)
	assert.Equal(t, "", prodServer.Password)

	stagingServer, err := config.GetServerConfig("staging")
	require.NoError(t, err)
	assert.Equal(t, "staging.example.com", stagingServer.Host)
	assert.Equal(t, 0, stagingServer.Port) // Default port
	assert.Equal(t, "deploy", stagingServer.User)
	assert.Equal(t, "", stagingServer.KeyFile)
	assert.Equal(t, "secret", stagingServer.Password)

	// Test app config
	assert.Equal(t, "news-website", config.App.Name)
	assert.Equal(t, "./news-server", config.App.Binary)
	assert.Equal(t, "./config.yaml", config.App.ConfigFile)
	assert.Equal(t, 8080, config.App.Port)
	assert.Equal(t, "/health", config.App.HealthPath)
	
	// Test environment variables
	assert.Len(t, config.App.Environment, 3)
	assert.Equal(t, "production", config.App.Environment["ENV"])
	assert.Equal(t, "info", config.App.Environment["LOG_LEVEL"])
	assert.Equal(t, "postgres://user:pass@localhost/db", config.App.Environment["DATABASE_URL"])
	
	// Test dependencies
	assert.Len(t, config.App.Dependencies, 2)
	assert.Contains(t, config.App.Dependencies, "postgresql-client")
	assert.Contains(t, config.App.Dependencies, "redis-tools")

	// Test SSH config
	assert.Equal(t, 45*time.Second, config.SSH.Timeout)
	assert.Equal(t, 5, config.SSH.ConnectRetries)
	assert.Equal(t, 10*time.Minute, config.SSH.CommandTimeout)
	assert.Equal(t, 60*time.Second, config.SSH.KeepAlive)

	// Test deploy config
	assert.Equal(t, "rolling", config.Deploy.Strategy)
	assert.Equal(t, "http://localhost:8080/api/health", config.Deploy.HealthCheckURL)
	assert.Equal(t, 3*time.Minute, config.Deploy.HealthTimeout)
	assert.Equal(t, 10*time.Minute, config.Deploy.RollbackTimeout)
	assert.Equal(t, 10, config.Deploy.BackupRetention)
	
	// Test hooks
	assert.Len(t, config.Deploy.PreDeployHooks, 2)
	assert.Equal(t, "echo 'Starting deployment'", config.Deploy.PreDeployHooks[0])
	assert.Equal(t, "sudo systemctl status nginx", config.Deploy.PreDeployHooks[1])
	
	assert.Len(t, config.Deploy.PostDeployHooks, 2)
	assert.Equal(t, "echo 'Deployment completed'", config.Deploy.PostDeployHooks[0])
	assert.Equal(t, "sudo systemctl reload nginx", config.Deploy.PostDeployHooks[1])
	
	// Test validation checks
	assert.Len(t, config.Deploy.ValidationChecks, 2)
	assert.Equal(t, "which curl", config.Deploy.ValidationChecks[0])
	assert.Equal(t, "systemctl --version", config.Deploy.ValidationChecks[1])
}

func TestConfigValidationSuccess(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "/path/to/binary",
			Port:   8080,
		},
		Servers: map[string]ServerConfig{
			"test": {
				Host:    "localhost",
				User:    "test",
				KeyFile: "/path/to/key",
			},
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "missing app name",
			config: &Config{
				App: AppConfig{
					Binary: "/path/to/binary",
					Port:   8080,
				},
			},
			wantErr: "app.name is required",
		},
		{
			name: "missing app binary",
			config: &Config{
				App: AppConfig{
					Name: "test-app",
					Port: 8080,
				},
			},
			wantErr: "app.binary is required",
		},
		{
			name: "missing app port",
			config: &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/path/to/binary",
				},
			},
			wantErr: "app.port is required",
		},
		{
			name: "missing server host",
			config: &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/path/to/binary",
					Port:   8080,
				},
				Servers: map[string]ServerConfig{
					"test": {
						User:    "test",
						KeyFile: "/path/to/key",
					},
				},
			},
			wantErr: "server 'test': host is required",
		},
		{
			name: "missing server user",
			config: &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/path/to/binary",
					Port:   8080,
				},
				Servers: map[string]ServerConfig{
					"test": {
						Host:    "localhost",
						KeyFile: "/path/to/key",
					},
				},
			},
			wantErr: "server 'test': user is required",
		},
		{
			name: "missing server auth",
			config: &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/path/to/binary",
					Port:   8080,
				},
				Servers: map[string]ServerConfig{
					"test": {
						Host: "localhost",
						User: "test",
					},
				},
			},
			wantErr: "server 'test': either key_file or password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestGetServerConfigSuccess(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"production": {
				Host:    "prod.example.com",
				Port:    22,
				User:    "deploy",
				KeyFile: "/path/to/key",
			},
			"staging": {
				Host:     "staging.example.com",
				User:     "deploy",
				Password: "secret",
			},
		},
	}

	// Test existing server
	server, err := config.GetServerConfig("production")
	assert.NoError(t, err)
	assert.Equal(t, "prod.example.com", server.Host)
	assert.Equal(t, 22, server.Port)
	assert.Equal(t, "deploy", server.User)
	assert.Equal(t, "/path/to/key", server.KeyFile)

	// Test another existing server
	server, err = config.GetServerConfig("staging")
	assert.NoError(t, err)
	assert.Equal(t, "staging.example.com", server.Host)
	assert.Equal(t, "deploy", server.User)
	assert.Equal(t, "secret", server.Password)
}

func TestGetServerConfigNotFound(t *testing.T) {
	config := &Config{
		Servers: map[string]ServerConfig{
			"production": {
				Host:    "prod.example.com",
				User:    "deploy",
				KeyFile: "/path/to/key",
			},
		},
	}

	_, err := config.GetServerConfig("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server 'nonexistent' not found")
}

func TestLoadConfigEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Should have defaults applied
	assert.Equal(t, 30*time.Second, config.SSH.Timeout)
	assert.Equal(t, 3, config.SSH.ConnectRetries)
	assert.Equal(t, "blue-green", config.Deploy.Strategy)
}

func TestLoadConfigPartialFile(t *testing.T) {
	configContent := `
app:
  name: "partial-app"
  binary: "/path/to/binary"
  port: 9000

ssh:
  timeout: 60s
`

	tmpFile, err := os.CreateTemp("", "partial-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test loaded values
	assert.Equal(t, "partial-app", config.App.Name)
	assert.Equal(t, "/path/to/binary", config.App.Binary)
	assert.Equal(t, 9000, config.App.Port)
	assert.Equal(t, 60*time.Second, config.SSH.Timeout)

	// Test defaults for missing values
	assert.Equal(t, 3, config.SSH.ConnectRetries)
	assert.Equal(t, 5*time.Minute, config.SSH.CommandTimeout)
	assert.Equal(t, "blue-green", config.Deploy.Strategy)
	assert.Equal(t, 5, config.Deploy.BackupRetention)
}

func TestLoadConfigInvalidDuration(t *testing.T) {
	configContent := `
ssh:
  timeout: "invalid-duration"
`

	tmpFile, err := os.CreateTemp("", "invalid-duration-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestConfigWithComplexEnvironment(t *testing.T) {
	configContent := `
app:
  name: "complex-app"
  binary: "/path/to/binary"
  port: 8080
  environment:
    DATABASE_URL: "postgres://user:pass@localhost:5432/db?sslmode=disable"
    REDIS_URL: "redis://localhost:6379/0"
    LOG_LEVEL: "debug"
    FEATURE_FLAGS: "flag1,flag2,flag3"
    API_KEYS: "key1:value1,key2:value2"
`

	tmpFile, err := os.CreateTemp("", "complex-env-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Len(t, config.App.Environment, 5)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db?sslmode=disable", config.App.Environment["DATABASE_URL"])
	assert.Equal(t, "redis://localhost:6379/0", config.App.Environment["REDIS_URL"])
	assert.Equal(t, "debug", config.App.Environment["LOG_LEVEL"])
	assert.Equal(t, "flag1,flag2,flag3", config.App.Environment["FEATURE_FLAGS"])
	assert.Equal(t, "key1:value1,key2:value2", config.App.Environment["API_KEYS"])
}

func TestConfigWithMultipleServers(t *testing.T) {
	configContent := `
servers:
  web1:
    host: "web1.example.com"
    user: "deploy"
    key_file: "/path/to/web1.key"
  web2:
    host: "web2.example.com"
    port: 2222
    user: "deploy"
    key_file: "/path/to/web2.key"
  db1:
    host: "db1.example.com"
    user: "dbadmin"
    password: "dbpass"
  lb1:
    host: "lb1.example.com"
    user: "lbadmin"
    key_file: "/path/to/lb.key"

app:
  name: "multi-server-app"
  binary: "/path/to/binary"
  port: 8080
`

	tmpFile, err := os.CreateTemp("", "multi-server-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Len(t, config.Servers, 4)

	// Test web1
	web1, err := config.GetServerConfig("web1")
	require.NoError(t, err)
	assert.Equal(t, "web1.example.com", web1.Host)
	assert.Equal(t, 0, web1.Port) // Default
	assert.Equal(t, "deploy", web1.User)
	assert.Equal(t, "/path/to/web1.key", web1.KeyFile)

	// Test web2
	web2, err := config.GetServerConfig("web2")
	require.NoError(t, err)
	assert.Equal(t, "web2.example.com", web2.Host)
	assert.Equal(t, 2222, web2.Port)
	assert.Equal(t, "deploy", web2.User)
	assert.Equal(t, "/path/to/web2.key", web2.KeyFile)

	// Test db1
	db1, err := config.GetServerConfig("db1")
	require.NoError(t, err)
	assert.Equal(t, "db1.example.com", db1.Host)
	assert.Equal(t, "dbadmin", db1.User)
	assert.Equal(t, "dbpass", db1.Password)

	// Test lb1
	lb1, err := config.GetServerConfig("lb1")
	require.NoError(t, err)
	assert.Equal(t, "lb1.example.com", lb1.Host)
	assert.Equal(t, "lbadmin", lb1.User)
	assert.Equal(t, "/path/to/lb.key", lb1.KeyFile)
}

// Test Advanced Configuration Features
func TestConfigurationWithHooks(t *testing.T) {
	configContent := `
servers:
  production:
    host: "prod.example.com"
    user: "deploy"
    key_file: "/path/to/key"

app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080

deploy:
  strategy: "blue-green"
  pre_deploy_hooks:
    - "echo 'Starting deployment'"
    - "sudo systemctl status nginx"
    - "df -h /opt"
  post_deploy_hooks:
    - "echo 'Deployment completed'"
    - "sudo systemctl reload nginx"
    - "curl -f http://localhost:8080/health"
  validation_checks:
    - "which curl"
    - "systemctl --version"
    - "sudo -n true"
`

	tmpFile, err := os.CreateTemp("", "hooks-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test pre-deploy hooks
	assert.Len(t, config.Deploy.PreDeployHooks, 3)
	assert.Equal(t, "echo 'Starting deployment'", config.Deploy.PreDeployHooks[0])
	assert.Equal(t, "sudo systemctl status nginx", config.Deploy.PreDeployHooks[1])
	assert.Equal(t, "df -h /opt", config.Deploy.PreDeployHooks[2])

	// Test post-deploy hooks
	assert.Len(t, config.Deploy.PostDeployHooks, 3)
	assert.Equal(t, "echo 'Deployment completed'", config.Deploy.PostDeployHooks[0])
	assert.Equal(t, "sudo systemctl reload nginx", config.Deploy.PostDeployHooks[1])
	assert.Equal(t, "curl -f http://localhost:8080/health", config.Deploy.PostDeployHooks[2])

	// Test validation checks
	assert.Len(t, config.Deploy.ValidationChecks, 3)
	assert.Equal(t, "which curl", config.Deploy.ValidationChecks[0])
	assert.Equal(t, "systemctl --version", config.Deploy.ValidationChecks[1])
	assert.Equal(t, "sudo -n true", config.Deploy.ValidationChecks[2])
}

func TestConfigurationWithComplexDependencies(t *testing.T) {
	configContent := `
app:
  name: "news-website"
  binary: "./news-server"
  port: 8080
  dependencies:
    - "postgresql-client"
    - "redis-tools"
    - "nginx"
    - "certbot"
    - "htop"
    - "curl"
    - "wget"
    - "git"
    - "build-essential"
    - "python3-pip"
`

	tmpFile, err := os.CreateTemp("", "deps-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Len(t, config.App.Dependencies, 10)
	assert.Contains(t, config.App.Dependencies, "postgresql-client")
	assert.Contains(t, config.App.Dependencies, "redis-tools")
	assert.Contains(t, config.App.Dependencies, "nginx")
	assert.Contains(t, config.App.Dependencies, "certbot")
	assert.Contains(t, config.App.Dependencies, "build-essential")
	assert.Contains(t, config.App.Dependencies, "python3-pip")
}

func TestConfigurationTimeoutSettings(t *testing.T) {
	configContent := `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080

ssh:
  timeout: 45s
  connect_retries: 5
  command_timeout: 15m
  keep_alive: 60s

deploy:
  health_timeout: 5m
  rollback_timeout: 10m
  backup_retention: 10
`

	tmpFile, err := os.CreateTemp("", "timeout-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test SSH timeouts
	assert.Equal(t, 45*time.Second, config.SSH.Timeout)
	assert.Equal(t, 5, config.SSH.ConnectRetries)
	assert.Equal(t, 15*time.Minute, config.SSH.CommandTimeout)
	assert.Equal(t, 60*time.Second, config.SSH.KeepAlive)

	// Test deployment timeouts
	assert.Equal(t, 5*time.Minute, config.Deploy.HealthTimeout)
	assert.Equal(t, 10*time.Minute, config.Deploy.RollbackTimeout)
	assert.Equal(t, 10, config.Deploy.BackupRetention)
}

func TestConfigurationValidationWithMissingFields(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing app section",
			configYAML: `
servers:
  test:
    host: "localhost"
    user: "test"
    key_file: "/path/to/key"
`,
			expectError: true,
			errorMsg:    "app.name is required",
		},
		{
			name: "missing servers section",
			configYAML: `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080
`,
			expectError: false, // Servers can be empty for validation-only operations
		},
		{
			name: "invalid port",
			configYAML: `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 0
`,
			expectError: true,
			errorMsg:    "app.port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "validation-config-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.configYAML)
			require.NoError(t, err)
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			require.NoError(t, err)

			err = config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigurationWithMultipleDeploymentStrategies(t *testing.T) {
	strategies := []string{"blue-green", "rolling"}

	for _, strategy := range strategies {
		t.Run(fmt.Sprintf("strategy_%s", strategy), func(t *testing.T) {
			configContent := fmt.Sprintf(`
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080

deploy:
  strategy: "%s"
  health_timeout: 3m
`, strategy)

			tmpFile, err := os.CreateTemp("", fmt.Sprintf("strategy-%s-*.yaml", strategy))
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(configContent)
			require.NoError(t, err)
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			require.NoError(t, err)

			assert.Equal(t, strategy, config.Deploy.Strategy)
			assert.Equal(t, 3*time.Minute, config.Deploy.HealthTimeout)
		})
	}
}

func TestConfigurationEnvironmentVariableHandling(t *testing.T) {
	configContent := `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080
  environment:
    DATABASE_URL: "postgres://user:pass@localhost:5432/db?sslmode=disable&connect_timeout=10"
    REDIS_URL: "redis://localhost:6379/0"
    LOG_LEVEL: "debug"
    FEATURE_FLAGS: "feature1,feature2,feature3"
    API_KEYS: "key1:value1,key2:value2,key3:value3"
    JSON_CONFIG: '{"timeout": 30, "retries": 3, "endpoints": ["api1", "api2"]}'
    MULTILINE_CONFIG: |
      line1
      line2
      line3
`

	tmpFile, err := os.CreateTemp("", "env-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Len(t, config.App.Environment, 7)
	
	// Test complex database URL
	assert.Equal(t, "postgres://user:pass@localhost:5432/db?sslmode=disable&connect_timeout=10", 
		config.App.Environment["DATABASE_URL"])
	
	// Test Redis URL
	assert.Equal(t, "redis://localhost:6379/0", config.App.Environment["REDIS_URL"])
	
	// Test comma-separated values
	assert.Equal(t, "feature1,feature2,feature3", config.App.Environment["FEATURE_FLAGS"])
	assert.Equal(t, "key1:value1,key2:value2,key3:value3", config.App.Environment["API_KEYS"])
	
	// Test JSON configuration
	assert.Contains(t, config.App.Environment["JSON_CONFIG"], `"timeout": 30`)
	assert.Contains(t, config.App.Environment["JSON_CONFIG"], `"retries": 3`)
	
	// Test multiline configuration
	assert.Contains(t, config.App.Environment["MULTILINE_CONFIG"], "line1")
	assert.Contains(t, config.App.Environment["MULTILINE_CONFIG"], "line2")
	assert.Contains(t, config.App.Environment["MULTILINE_CONFIG"], "line3")
}

func TestConfigurationServerAuthentication(t *testing.T) {
	tests := []struct {
		name         string
		serverConfig ServerConfig
		expectValid  bool
	}{
		{
			name: "key file authentication",
			serverConfig: ServerConfig{
				Host:    "localhost",
				User:    "deploy",
				KeyFile: "/path/to/key",
			},
			expectValid: true,
		},
		{
			name: "password authentication",
			serverConfig: ServerConfig{
				Host:     "localhost",
				User:     "deploy",
				Password: "secret",
			},
			expectValid: true,
		},
		{
			name: "both key and password",
			serverConfig: ServerConfig{
				Host:     "localhost",
				User:     "deploy",
				KeyFile:  "/path/to/key",
				Password: "secret",
			},
			expectValid: true, // Both are allowed, key takes precedence
		},
		{
			name: "no authentication",
			serverConfig: ServerConfig{
				Host: "localhost",
				User: "deploy",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/path/to/binary",
					Port:   8080,
				},
				Servers: map[string]ServerConfig{
					"test": tt.serverConfig,
				},
			}

			err := config.Validate()
			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "either key_file or password is required")
			}
		})
	}
}

func TestConfigurationDefaultPortHandling(t *testing.T) {
	configContent := `
servers:
  with_port:
    host: "server1.example.com"
    port: 2222
    user: "deploy"
    key_file: "/path/to/key"
  without_port:
    host: "server2.example.com"
    user: "deploy"
    key_file: "/path/to/key"

app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080
`

	tmpFile, err := os.CreateTemp("", "port-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Test explicit port
	serverWithPort, err := config.GetServerConfig("with_port")
	require.NoError(t, err)
	assert.Equal(t, 2222, serverWithPort.Port)

	// Test default port (should be 0, handled by SSH client)
	serverWithoutPort, err := config.GetServerConfig("without_port")
	require.NoError(t, err)
	assert.Equal(t, 0, serverWithoutPort.Port)
}

func TestConfigurationHealthCheckSettings(t *testing.T) {
	configContent := `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080
  health_path: "/api/v1/health"

deploy:
  health_check_url: "http://localhost:8080/api/v1/health"
  health_timeout: 2m30s
`

	tmpFile, err := os.CreateTemp("", "health-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/health", config.App.HealthPath)
	assert.Equal(t, "http://localhost:8080/api/v1/health", config.Deploy.HealthCheckURL)
	assert.Equal(t, 2*time.Minute+30*time.Second, config.Deploy.HealthTimeout)
}

func TestConfigurationBackupRetentionSettings(t *testing.T) {
	tests := []struct {
		name              string
		backupRetention   int
		expectedRetention int
	}{
		{"default retention", 0, 5},
		{"custom retention", 10, 10},
		{"minimal retention", 1, 1},
		{"high retention", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := fmt.Sprintf(`
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080

deploy:
  backup_retention: %d
`, tt.backupRetention)

			// Handle default case
			if tt.backupRetention == 0 {
				configContent = `
app:
  name: "test-app"
  binary: "/path/to/binary"
  port: 8080

deploy:
  strategy: "blue-green"
`
			}

			tmpFile, err := os.CreateTemp("", "backup-config-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(configContent)
			require.NoError(t, err)
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			require.NoError(t, err)

			assert.Equal(t, tt.expectedRetention, config.Deploy.BackupRetention)
		})
	}
}