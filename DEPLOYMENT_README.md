# Zero-Touch Deployment System

This document describes the zero-touch deployment system for the high-performance news website. The system provides automated deployment with blue-green deployment strategy, automatic rollback capabilities, and comprehensive health checking.

## Features

- **Zero-Touch Deployment**: Fully automated deployment process with minimal human intervention
- **Blue-Green Deployment**: Zero-downtime deployments with automatic rollback on failure
- **SSH-Based Management**: Secure remote server management using SSH
- **Automated Server Setup**: One-command server initialization and dependency installation
- **Health Validation**: Multi-layer health checks with automatic rollback on failure
- **Deployment Validation**: Pre-deployment checks to ensure successful deployment
- **Backup and Recovery**: Automatic backup creation with integrity verification
- **Atomic Operations**: All deployment operations are atomic to prevent partial deployments

## Architecture

The deployment system consists of several components:

1. **Deployment Agent** (`internal/deployment/agent.go`): Core deployment logic
2. **SSH Client** (`internal/deployment/ssh.go`): Secure remote command execution
3. **Configuration Management** (`internal/deployment/config.go`): YAML-based configuration
4. **Command Line Interface** (`cmd/deploy/main.go`): CLI for deployment operations
5. **Shell Scripts** (`scripts/deploy.sh`): Convenient wrapper scripts

## Quick Start

### 1. Configuration

Copy the example configuration and customize it:

```bash
cp deploy.yaml.example deploy.yaml
```

Edit `deploy.yaml` to configure your servers and application settings:

```yaml
servers:
  production:
    host: "your-server.com"
    user: "deploy"
    key_file: "~/.ssh/id_rsa"

app:
  name: "news-website"
  binary: "./news-server"
  port: 8080
  health_path: "/health"
```

### 2. Build the Application

Build your application binary:

```bash
go build -o news-server ./cmd/server
```

### 3. Deploy

Deploy to your server:

```bash
./scripts/deploy.sh --target production
```

## Deployment Strategies

### Blue-Green Deployment (Recommended)

Blue-green deployment provides zero-downtime deployments by maintaining two identical production environments:

1. **Current Version (Blue)**: Currently serving traffic
2. **New Version (Green)**: New deployment being prepared

The deployment process:

1. **Validation**: Pre-deployment checks and validation
2. **Backup**: Create verified backup of current version
3. **Deploy**: Deploy new version to separate directory
4. **Switch**: Atomic symlink switch to new version
5. **Health Check**: Comprehensive health validation
6. **Rollback**: Automatic rollback on any failure

### Rolling Deployment

Rolling deployment updates the application in-place:

1. Stop current service
2. Deploy new version
3. Start service
4. Health check

## Commands

### Deploy

Deploy the application to a target server:

```bash
# Deploy to production
./scripts/deploy.sh --target production

# Deploy with dry run (validation only)
./scripts/deploy.sh --target production --dry-run

# Deploy using custom config
./scripts/deploy.sh --config custom-deploy.yaml --target production
```

### Rollback

Rollback to a previous version:

```bash
# List available versions (manual check on server)
ssh deploy@your-server.com "ls -la /opt/news-website/"

# Rollback to specific version
./scripts/deploy.sh --action rollback --target production --version news-website-1640995200
```

### Health Check

Check the health of a deployed application:

```bash
./scripts/deploy.sh --action health-check --target production
```

### Server Setup

Initialize a new server with dependencies:

```bash
./scripts/deploy.sh --action setup --target production
```

## Configuration Reference

### Server Configuration

```yaml
servers:
  server_name:
    host: "server.example.com"    # Required: Server hostname or IP
    port: 22                      # Optional: SSH port (default: 22)
    user: "deploy"                # Required: SSH username
    key_file: "~/.ssh/id_rsa"     # SSH private key file
    password: "password"          # Alternative to key_file
```

### Application Configuration

```yaml
app:
  name: "news-website"            # Required: Application name
  binary: "./news-server"         # Required: Path to application binary
  config_file: "./config.yaml"    # Optional: Configuration file to upload
  port: 8080                      # Required: Application port
  health_path: "/health"          # Optional: Health check endpoint
  environment:                    # Optional: Environment variables
    ENV: "production"
    LOG_LEVEL: "info"
  dependencies:                   # Optional: System packages to install
    - "postgresql-client"
    - "redis-tools"
```

### SSH Configuration

```yaml
ssh:
  timeout: 30s                    # Connection timeout
  connect_retries: 3              # Number of connection retries
  command_timeout: 10m            # Command execution timeout
  keep_alive: 30s                 # Keep-alive interval
```

### Deployment Configuration

```yaml
deploy:
  strategy: "blue-green"          # Deployment strategy
  health_check_url: "http://localhost:8080/health"  # Custom health check URL
  health_timeout: 3m              # Health check timeout
  rollback_timeout: 5m            # Rollback timeout
  backup_retention: 5             # Number of backups to keep
  pre_deploy_hooks:               # Commands to run before deployment
    - "echo 'Starting deployment'"
  post_deploy_hooks:              # Commands to run after deployment
    - "sudo systemctl reload nginx"
  validation_checks:              # Pre-deployment validation commands
    - "which curl"
    - "systemctl --version"
```

## Health Checks

The deployment system performs comprehensive health checks:

### 1. Service Status Check
Verifies the systemd service is active and running.

### 2. Process Check
Confirms the application process is running.

### 3. Port Binding Check
Ensures the application is listening on the configured port.

### 4. HTTP Health Check
Performs HTTP requests to the health endpoint with validation:
- HTTP status code (must be 200)
- Response time (must be under 5 seconds)
- Response body (must not be empty)
- JSON validation (if response is JSON)

### 5. Memory Usage Check
Monitors system memory usage (warns if over 95%).

### 6. Multiple Validation Points
Requires 3 consecutive successful health checks for stability.

## Automatic Rollback

The system automatically rolls back on any failure:

### Rollback Triggers
- Pre-deployment validation failure
- Application deployment failure
- Service start failure
- Health check failure
- Version switch failure

### Rollback Process
1. Stop failed service
2. Switch symlink back to backup version
3. Start service with backup version
4. Verify rollback health
5. Clean up failed deployment
6. Log rollback completion

## Security Features

### SSH Security
- Key-based authentication (recommended)
- Connection timeout and retry limits
- Secure command execution

### File Permissions
- Proper ownership (www-data:www-data)
- Secure file permissions (755 for binaries, 644 for configs)
- Isolated deployment directories

### Validation
- Pre-deployment validation checks
- Binary existence and permissions verification
- Server requirements validation
- Configuration validation

## Monitoring and Logging

### Deployment Logging
All deployment operations are logged with:
- Timestamps
- Command execution details
- Error messages and stack traces
- Performance metrics

### Deployment Records
Successful deployments create JSON records:
```json
{
  "timestamp": 1640995200,
  "version": "news-website-1640995200",
  "duration": 45.2,
  "app_name": "news-website",
  "deployment_type": "blue-green",
  "status": "success"
}
```

### Health Monitoring
- Real-time health status
- Performance metrics collection
- Resource usage monitoring

## Troubleshooting

### Common Issues

#### 1. SSH Connection Failed
```
Error: failed to connect to server after 3 retries
```
**Solutions:**
- Verify server hostname/IP
- Check SSH key permissions (600)
- Ensure SSH service is running on target server
- Verify firewall allows SSH connections

#### 2. Binary Not Found
```
Error: binary file not found: ./news-server
```
**Solutions:**
- Build the application: `go build -o news-server ./cmd/server`
- Verify binary path in deploy.yaml
- Check file permissions (must be executable)

#### 3. Health Check Failed
```
Error: health check failed, rolled back successfully
```
**Solutions:**
- Check application logs: `journalctl -u news-website -f`
- Verify health endpoint is responding
- Check port binding: `netstat -ln | grep :8080`
- Review application configuration

#### 4. Permission Denied
```
Error: failed to create directory: Permission denied
```
**Solutions:**
- Ensure SSH user has sudo access
- Verify sudo configuration allows passwordless execution
- Check directory permissions

### Debug Mode

Enable verbose logging:

```bash
./scripts/deploy.sh --target production --verbose
```

### Manual Recovery

If automatic rollback fails, manual recovery steps:

1. **SSH to server:**
   ```bash
   ssh deploy@your-server.com
   ```

2. **Check available versions:**
   ```bash
   ls -la /opt/news-website/
   ```

3. **Manual rollback:**
   ```bash
   sudo systemctl stop news-website
   cd /opt/news-website
   ln -sfn news-website-backup news-website-current
   sudo systemctl start news-website
   ```

4. **Verify health:**
   ```bash
   curl http://localhost:8080/health
   ```

## Performance Considerations

### Deployment Speed
- Atomic operations minimize downtime
- Hard-link backups for fast copying
- Parallel health checks where possible

### Resource Usage
- Minimal memory footprint during deployment
- Efficient file operations
- Cleanup of temporary files

### Network Optimization
- SSH connection reuse
- Compressed file transfers
- Minimal network round-trips

## Best Practices

### 1. Testing
- Always test deployments on staging first
- Use dry-run mode for validation
- Maintain separate configurations for each environment

### 2. Backup Strategy
- Regular database backups (separate from deployment backups)
- Test backup restoration procedures
- Monitor backup retention policies

### 3. Monitoring
- Set up alerts for deployment failures
- Monitor application performance after deployments
- Track deployment frequency and success rates

### 4. Security
- Use SSH keys instead of passwords
- Regularly rotate SSH keys
- Limit SSH user permissions
- Keep deployment logs secure

### 5. Documentation
- Document custom hooks and validation checks
- Maintain deployment runbooks
- Keep configuration files in version control

## Integration

### CI/CD Integration

The deployment system can be integrated with CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Deploy to Production
  run: |
    ./scripts/deploy.sh --target production
  env:
    SSH_PRIVATE_KEY: ${{ secrets.SSH_PRIVATE_KEY }}
```

### Monitoring Integration

Integrate with monitoring systems:

```bash
# Post-deploy hook example
post_deploy_hooks:
  - "curl -X POST https://monitoring.example.com/api/deployments -d '{\"app\":\"news-website\",\"version\":\"$VERSION\"}'"
```

## Support

For issues and questions:

1. Check the troubleshooting section above
2. Review deployment logs
3. Test with dry-run mode
4. Verify configuration syntax

## License

This deployment system is part of the high-performance news website project.