# Zero-Touch Deployment System

This package implements a zero-touch deployment system with SSH-based server management, automated server setup, and blue-green deployment with automatic rollback capabilities.

## Features

- **SSH-based Server Management**: Secure remote server operations via SSH
- **Automated Server Setup**: Automatic installation of dependencies and system configuration
- **Blue-Green Deployment**: Zero-downtime deployments with automatic rollback
- **Rolling Deployment**: Alternative deployment strategy for simpler scenarios
- **Health Checking**: Automatic application health verification with configurable timeouts
- **Resource Monitoring**: Real-time server resource usage monitoring
- **Deployment History**: Track and audit deployment activities
- **Atomic Operations**: Ensure deployment consistency with atomic file operations
- **Systemd Integration**: Automatic service creation and management

## Quick Start

### 1. Configuration

Create a deployment configuration file (YAML format):

```yaml
servers:
  production:
    host: "your-server.com"
    port: 22
    user: "deploy"
    key_file: "/path/to/ssh/key"

app:
  name: "news-website"
  binary: "./bin/news-website"
  port: 8080
  health_path: "/health"
  dependencies:
    - "postgresql-client"
    - "nginx"

deploy:
  strategy: "blue-green"
  health_timeout: 2m
  validation_checks:
    - "which curl"
    - "systemctl --version"
```

### 2. Basic Usage

```go
package main

import (
    "log"
    "high-performance-news-website/internal/deployment"
)

func main() {
    // Load configuration
    config, err := deployment.LoadConfig("deploy-config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Create deployment agent
    agent := deployment.NewAgent(config)

    // Setup server (first time only)
    if err := agent.SetupServer("production"); err != nil {
        log.Fatal(err)
    }

    // Deploy application
    if err := agent.Deploy("production"); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Reference

### Server Configuration

```yaml
servers:
  server-name:
    host: "server.example.com"     # Required: Server hostname or IP
    port: 22                       # Optional: SSH port (default: 22)
    user: "deploy"                 # Required: SSH username
    key_file: "/path/to/key"       # SSH private key file
    password: "password"           # Alternative to key_file
```

### Application Configuration

```yaml
app:
  name: "app-name"                 # Required: Application name
  binary: "./bin/app"              # Required: Path to application binary
  config_file: "./config.yaml"    # Optional: Application config file
  port: 8080                       # Required: Application port
  health_path: "/health"           # Optional: Health check endpoint
  environment:                     # Optional: Environment variables
    ENV: "production"
    LOG_LEVEL: "info"
  dependencies:                    # Optional: System dependencies
    - "postgresql-client"
    - "nginx"
```

### SSH Configuration

```yaml
ssh:
  timeout: 30s                     # Connection timeout
  connect_retries: 3               # Connection retry attempts
  command_timeout: 5m              # Command execution timeout
  keep_alive: 30s                  # Keep-alive interval
```

### Deployment Configuration

```yaml
deploy:
  strategy: "blue-green"           # Deployment strategy: "blue-green" or "rolling"
  health_check_url: "http://localhost:8080/health"  # Health check URL
  health_timeout: 2m               # Health check timeout
  rollback_timeout: 5m             # Rollback timeout
  backup_retention: 5              # Number of backups to retain
  pre_deploy_hooks:                # Commands to run before deployment
    - "echo 'Starting deployment'"
  post_deploy_hooks:               # Commands to run after deployment
    - "echo 'Deployment complete'"
  validation_checks:               # Prerequisites validation
    - "which curl"
    - "systemctl --version"
```

## Deployment Strategies

### Blue-Green Deployment

Blue-green deployment ensures zero-downtime deployments by:

1. Creating a new version alongside the current version
2. Performing health checks on the new version
3. Switching traffic to the new version atomically
4. Automatically rolling back if health checks fail

```yaml
deploy:
  strategy: "blue-green"
  health_timeout: 2m
```

### Rolling Deployment

Rolling deployment is simpler but may have brief downtime:

1. Stop the current application
2. Deploy the new version
3. Start the new application
4. Perform health checks

```yaml
deploy:
  strategy: "rolling"
```

## API Reference

### Agent

```go
type Agent struct {
    config *Config
}

// Create new deployment agent
func NewAgent(config *Config) *Agent

// Validate configuration
func (a *Agent) ValidateConfiguration() error

// Setup server (install dependencies, create users, etc.)
func (a *Agent) SetupServer(serverName string) error

// Deploy application to server
func (a *Agent) Deploy(serverName string) error

// Get server status and resource usage
func (a *Agent) GetServerStatus(serverName string) (*ServerStatus, error)

// Get deployment history
func (a *Agent) GetDeploymentHistory(serverName string, limit int) ([]map[string]interface{}, error)
```

### Configuration

```go
// Load configuration from YAML file
func LoadConfig(filename string) (*Config, error)

// Validate configuration
func (c *Config) Validate() error

// Get server configuration by name
func (c *Config) GetServerConfig(name string) (*ServerConfig, error)
```

## Error Handling

The deployment system includes comprehensive error handling:

- **Connection Errors**: Automatic retry with exponential backoff
- **Command Failures**: Detailed error reporting with stdout/stderr
- **Health Check Failures**: Automatic rollback to previous version
- **Validation Errors**: Pre-deployment validation to catch issues early

## Security Considerations

- **SSH Key Authentication**: Preferred over password authentication
- **File Permissions**: Proper file and directory permissions are set
- **User Isolation**: Applications run under dedicated system users
- **Command Validation**: Input validation for all remote commands

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test -v ./internal/deployment/

# Run specific test
go test -v ./internal/deployment/ -run TestBlueGreenDeployment

# Run with coverage
go test -v -cover ./internal/deployment/
```

## Examples

See `example-usage.go` for a complete CLI example and `example-config.yaml` for a sample configuration file.

## Requirements Satisfied

This implementation satisfies the following requirements from the high-performance news website specification:

- **SSH-based server management**: Secure remote operations via SSH
- **Automated server setup**: Dependency installation and system configuration
- **Blue-green deployment**: Zero-downtime deployments with rollback
- **Deployment validation**: Pre-deployment checks and health monitoring
- **Comprehensive testing**: Unit and integration tests for all functionality

## Troubleshooting

### Common Issues

1. **SSH Connection Failed**
   - Verify SSH key permissions (600)
   - Check server hostname and port
   - Ensure SSH service is running on target server

2. **Health Check Failed**
   - Verify application is listening on configured port
   - Check health endpoint returns 200 status
   - Review application logs for startup issues

3. **Permission Denied**
   - Ensure SSH user has sudo privileges
   - Check file permissions on uploaded binaries
   - Verify systemd service permissions

### Debug Mode

Enable verbose logging by setting log level:

```go
log.SetLevel(log.DebugLevel)
```

This will provide detailed information about SSH commands, file operations, and deployment steps.