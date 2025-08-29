# Desktop Deployment Application

A cross-platform desktop application for managing deployments of the high-performance news website. This application provides a user-friendly GUI interface for server management, deployment monitoring, and system administration.

## Features

### 🖥️ Cross-Platform Support
- **Windows** (x64)
- **Linux** (x64)
- **macOS** (Intel & Apple Silicon)

### 🚀 Deployment Management
- **Server Connection Management**: Secure SSH-based server connections with credential storage
- **Blue-Green Deployments**: Zero-downtime deployments with automatic rollback
- **Rolling Deployments**: Alternative deployment strategy for simpler scenarios
- **Deployment History**: Track and audit all deployment activities
- **Real-time Monitoring**: Live deployment progress and status updates

### 📊 System Monitoring
- **Resource Usage**: Monitor CPU, memory, disk usage, and load averages
- **Health Checks**: Automated application health verification
- **Server Status**: Real-time connection status and system information
- **Performance Metrics**: Track deployment performance and success rates

### 🔧 Configuration Management
- **Visual Configuration Editor**: Easy-to-use interface for deployment settings
- **Configuration Validation**: Automatic validation of deployment configurations
- **Multiple Server Support**: Manage multiple servers from a single interface
- **Backup Management**: Automated backup creation and restoration

### 📝 Logging and Monitoring
- **Real-time Logs**: Live log streaming with filtering and search
- **Log Levels**: Filter by info, warning, error, and success messages
- **Server-specific Logs**: View logs for individual servers
- **Export Capabilities**: Export logs and deployment history

## Installation

### Download Pre-built Binaries

1. Go to the [Releases](https://github.com/your-org/news-website/releases) page
2. Download the appropriate binary for your platform:
   - `news-deploy-windows-amd64.zip` for Windows
   - `news-deploy-linux-amd64.tar.gz` for Linux
   - `news-deploy-darwin-amd64.tar.gz` for macOS Intel
   - `news-deploy-darwin-arm64.tar.gz` for macOS Apple Silicon

3. Extract the archive and run the application

### Build from Source

#### Prerequisites
- Go 1.21 or later
- Git

#### Build Steps

```bash
# Clone the repository
git clone https://github.com/your-org/news-website.git
cd news-website/cmd/desktop-deploy

# Install dependencies
go mod download

# Build for current platform
make build

# Or build for all platforms
make build-all

# Run the application
./dist/news-deploy
```

## Quick Start

### 1. Start the Application

**Windows:**
```cmd
news-deploy.exe
```

**Linux/macOS:**
```bash
./news-deploy
```

The application will:
- Start a web server on port 8090
- Automatically open your default browser
- Display the deployment dashboard

### 2. Load Configuration

1. Click on **Configuration** in the sidebar
2. Enter the path to your deployment configuration file
3. Click **Load Configuration**

Example configuration file (`deploy-config.yaml`):

```yaml
servers:
  production:
    host: "your-server.com"
    port: 22
    user: "deploy"
    key_file: "/path/to/ssh/private/key"
  
  staging:
    host: "staging-server.com"
    port: 22
    user: "deploy"
    key_file: "/path/to/ssh/private/key"

app:
  name: "news-website"
  binary: "./bin/news-website"
  config_file: "./config.yaml"
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

### 3. Setup Servers

1. Go to **Servers** page
2. Click **Setup** next to a server to install dependencies and configure the environment
3. Monitor the setup progress in real-time

### 4. Deploy Application

1. Click **Deploy** next to a configured server
2. Monitor deployment progress on the dashboard
3. View deployment logs in the **Logs** section

## Configuration Reference

### Server Configuration

```yaml
servers:
  server-name:
    host: "server.example.com"     # Server hostname or IP
    port: 22                       # SSH port (default: 22)
    user: "deploy"                 # SSH username
    key_file: "/path/to/key"       # SSH private key file
    password: "password"           # Alternative to key_file (not recommended)
```

### Application Configuration

```yaml
app:
  name: "app-name"                 # Application name
  binary: "./bin/app"              # Path to application binary
  config_file: "./config.yaml"    # Application config file (optional)
  port: 8080                       # Application port
  health_path: "/health"           # Health check endpoint (optional)
  environment:                     # Environment variables (optional)
    ENV: "production"
    LOG_LEVEL: "info"
  dependencies:                    # System dependencies (optional)
    - "postgresql-client"
    - "nginx"
```

### Deployment Configuration

```yaml
deploy:
  strategy: "blue-green"           # "blue-green" or "rolling"
  health_timeout: 2m               # Health check timeout
  rollback_timeout: 5m             # Rollback timeout
  backup_retention: 5              # Number of backups to retain
  validation_checks:               # Pre-deployment validation
    - "which curl"
    - "systemctl --version"
```

## Command Line Options

```bash
# Start with custom port
./news-deploy --port 8091

# Start without opening browser
./news-deploy --no-browser

# Development mode (with hot reload)
./news-deploy --dev

# Show help
./news-deploy --help
```

## Development

### Prerequisites
- Go 1.21+
- Node.js (for frontend development)
- Make

### Development Setup

```bash
# Clone repository
git clone https://github.com/your-org/news-website.git
cd news-website/cmd/desktop-deploy

# Install dependencies
make deps

# Run in development mode
make dev

# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Format code
make fmt
```

### Project Structure

```
cmd/desktop-deploy/
├── main.go                 # Application entry point
├── build.go               # Cross-platform build script
├── Makefile              # Build automation
├── README.md             # This file
├── static/               # Static assets (CSS, JS, images)
│   ├── css/
│   └── js/
├── templates/            # HTML templates
│   ├── base.html
│   ├── dashboard.html
│   ├── servers.html
│   ├── config.html
│   └── logs.html
└── tests/                # Test files
    ├── integration_test.go
    └── ...

internal/desktop/         # Desktop app package
├── app.go               # Main application logic
├── app_test.go          # Unit tests
└── ...
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-windows
make build-linux
make build-macos

# Create release packages
make package
```

## API Reference

The desktop application exposes a REST API for programmatic access:

### Configuration
- `GET /api/config` - Get current configuration status
- `POST /api/config` - Update configuration
- `POST /api/config/load` - Load configuration from file

### Servers
- `GET /api/servers` - List all configured servers
- `GET /api/servers/{name}/status` - Get server status
- `POST /api/servers/{name}/setup` - Setup server
- `POST /api/servers/{name}/deploy` - Deploy to server
- `GET /api/servers/{name}/history` - Get deployment history
- `GET /api/servers/{name}/logs` - Get server logs

### WebSocket
- `WS /ws` - Real-time updates and notifications

## Security Considerations

### SSH Key Management
- Store SSH private keys securely with proper file permissions (600)
- Use SSH key authentication instead of passwords
- Rotate SSH keys regularly

### Network Security
- The desktop app runs on localhost by default
- Use HTTPS in production environments
- Implement proper firewall rules on target servers

### Credential Storage
- SSH credentials are not stored in the application
- Configuration files should be secured with appropriate permissions
- Use environment variables for sensitive configuration

## Troubleshooting

### Common Issues

#### SSH Connection Failed
- Verify SSH key permissions: `chmod 600 /path/to/ssh/key`
- Check server hostname and port
- Ensure SSH service is running on target server
- Verify SSH user has appropriate permissions

#### Health Check Failed
- Verify application is listening on configured port
- Check health endpoint returns 200 status
- Review application logs for startup issues
- Ensure firewall allows connections to application port

#### Deployment Failed
- Check disk space on target server
- Verify binary file exists and is executable
- Review deployment logs for specific error messages
- Ensure all dependencies are installed

#### Configuration Validation Failed
- Verify all required fields are present
- Check file paths exist and are accessible
- Validate YAML syntax
- Ensure SSH keys are readable

### Debug Mode

Enable verbose logging by starting with debug flag:

```bash
./news-deploy --debug
```

This provides detailed information about:
- SSH connections and commands
- File operations and transfers
- Deployment steps and timing
- WebSocket connections and messages

### Log Files

Application logs are stored in:
- **Windows**: `%APPDATA%/news-deploy/logs/`
- **Linux**: `~/.local/share/news-deploy/logs/`
- **macOS**: `~/Library/Application Support/news-deploy/logs/`

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Run tests: `make test`
5. Run linter: `make lint`
6. Commit your changes: `git commit -am 'Add feature'`
7. Push to the branch: `git push origin feature-name`
8. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [https://docs.example.com](https://docs.example.com)
- **Issues**: [GitHub Issues](https://github.com/your-org/news-website/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/news-website/discussions)
- **Email**: support@example.com

## Changelog

### v1.0.0 (2024-01-XX)
- Initial release
- Cross-platform desktop application
- Server connection management
- Blue-green and rolling deployments
- Real-time monitoring and logging
- Configuration management interface
- WebSocket-based real-time updates