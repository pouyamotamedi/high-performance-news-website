package deployment

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Agent represents the deployment agent
type Agent struct {
	config *Config
}

// DeploymentState tracks the state of an ongoing deployment
type DeploymentState struct {
	AppName        string
	DeployDir      string
	CurrentVersion string
	NewVersion     string
	BackupVersion  string
	StartTime      time.Time
	CheckpointPath string
}

// ServerStatus represents the status of a server
type ServerStatus struct {
	Name         string
	Host         string
	Connected    bool
	SystemInfo   *SystemInfo
	ResourceUsage *ResourceUsage
	LastChecked  time.Time
	Error        string
}

// ResourceUsage represents server resource usage
type ResourceUsage struct {
	CPUUsage     string
	MemoryUsage  string
	DiskUsage    string
	LoadAverage  string
}

// NewAgent creates a new deployment agent
func NewAgent(config *Config) *Agent {
	return &Agent{
		config: config,
	}
}

// GetServerStatus returns the status of a specific server
func (a *Agent) GetServerStatus(serverName string) (*ServerStatus, error) {
	serverConfig, err := a.config.GetServerConfig(serverName)
	if err != nil {
		return nil, err
	}

	status := &ServerStatus{
		Name:        serverName,
		Host:        serverConfig.Host,
		Connected:   false,
		LastChecked: time.Now(),
	}

	// Try to connect and get system info
	client, err := NewSSHClient(a.config, serverConfig)
	if err != nil {
		status.Error = err.Error()
		return status, nil
	}
	defer client.Close()

	status.Connected = true

	// Get system information
	systemInfo, err := client.GetSystemInfo()
	if err != nil {
		status.Error = fmt.Sprintf("Failed to get system info: %v", err)
	} else {
		status.SystemInfo = systemInfo
	}

	// Get resource usage
	resourceUsage := a.getResourceUsage(client)
	status.ResourceUsage = resourceUsage

	return status, nil
}

// ValidateConfiguration validates the deployment configuration
func (a *Agent) ValidateConfiguration() error {
	// Validate basic config
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Check if binary file exists
	if a.config.App.Binary == "" {
		return fmt.Errorf("app.binary is required")
	}

	if _, err := os.Stat(a.config.App.Binary); os.IsNotExist(err) {
		return fmt.Errorf("binary file not found: %s", a.config.App.Binary)
	}

	// Validate binary is executable
	info, err := os.Stat(a.config.App.Binary)
	if err != nil {
		return fmt.Errorf("failed to check binary file: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary file is not executable: %s", a.config.App.Binary)
	}

	return nil
}

// Deploy performs a deployment to the specified server
func (a *Agent) Deploy(serverName string) error {
	serverConfig, err := a.config.GetServerConfig(serverName)
	if err != nil {
		return err
	}

	client, err := NewSSHClient(a.config, serverConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer client.Close()

	log.Printf("Starting deployment to server: %s", serverName)

	// Validate deployment prerequisites
	if err := a.validateDeployment(client); err != nil {
		return fmt.Errorf("deployment validation failed: %w", err)
	}

	// Perform deployment based on strategy
	switch a.config.Deploy.Strategy {
	case "blue-green":
		return a.blueGreenDeploy(client)
	case "rolling":
		return a.rollingDeploy(client)
	default:
		return fmt.Errorf("unsupported deployment strategy: %s", a.config.Deploy.Strategy)
	}
}

// SetupServer performs initial server setup
func (a *Agent) SetupServer(serverName string) error {
	serverConfig, err := a.config.GetServerConfig(serverName)
	if err != nil {
		return err
	}

	client, err := NewSSHClient(a.config, serverConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer client.Close()

	log.Printf("Setting up server: %s", serverName)

	// Update system
	if err := a.updateSystem(client); err != nil {
		return fmt.Errorf("failed to update system: %w", err)
	}

	// Install dependencies
	if err := a.installDependencies(client); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Setup application user
	if err := a.setupAppUser(client); err != nil {
		return fmt.Errorf("failed to setup app user: %w", err)
	}

	// Setup directories
	if err := a.setupDirectories(client); err != nil {
		return fmt.Errorf("failed to setup directories: %w", err)
	}

	log.Printf("Server setup completed: %s", serverName)
	return nil
}

// validateDeployment validates deployment prerequisites
func (a *Agent) validateDeployment(client SSHClientInterface) error {
	log.Println("Validating deployment prerequisites...")

	for _, check := range a.config.Deploy.ValidationChecks {
		result, err := client.ExecuteCommand(check)
		if err != nil {
			return fmt.Errorf("validation check failed: %s - %w", check, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("validation check failed: %s - %s", check, result.Stderr)
		}
	}

	return nil
}

// blueGreenDeploy performs blue-green deployment
func (a *Agent) blueGreenDeploy(client SSHClientInterface) error {
	log.Println("Starting blue-green deployment...")

	deployDir := fmt.Sprintf("/opt/%s", a.config.App.Name)
	timestamp := time.Now().Unix()
	
	state := &DeploymentState{
		AppName:        a.config.App.Name,
		DeployDir:      deployDir,
		CurrentVersion: fmt.Sprintf("%s-current", a.config.App.Name),
		NewVersion:     fmt.Sprintf("%s-%d", a.config.App.Name, timestamp),
		BackupVersion:  fmt.Sprintf("%s-backup", a.config.App.Name),
		StartTime:      time.Now(),
	}

	// Create deployment checkpoint
	if err := a.createDeploymentCheckpoint(client, state); err != nil {
		return fmt.Errorf("failed to create deployment checkpoint: %w", err)
	}

	// Check disk space
	if err := a.checkDiskSpace(client); err != nil {
		return fmt.Errorf("insufficient disk space: %w", err)
	}

	// Deploy application atomically
	if err := a.deployApplicationAtomic(client, state); err != nil {
		return fmt.Errorf("failed to deploy application: %w", err)
	}

	// Create systemd service
	if err := a.createSystemdService(client, deployDir); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Perform health check
	if err := a.performHealthCheck(client); err != nil {
		log.Printf("Health check failed, rolling back...")
		if rollbackErr := a.rollback(client, state); rollbackErr != nil {
			return fmt.Errorf("health check failed and rollback failed: %w, rollback error: %w", err, rollbackErr)
		}
		return fmt.Errorf("health check failed, rolled back successfully: %w", err)
	}

	log.Println("Blue-green deployment completed successfully")
	return nil
}

// rollingDeploy performs rolling deployment
func (a *Agent) rollingDeploy(client SSHClientInterface) error {
	log.Println("Starting rolling deployment...")
	
	deployDir := fmt.Sprintf("/opt/%s", a.config.App.Name)
	
	// Stop service
	if _, err := client.ExecuteCommand(fmt.Sprintf("sudo systemctl stop %s", a.config.App.Name)); err != nil {
		log.Printf("Warning: failed to stop service: %v", err)
	}

	// Deploy application
	if err := a.deployApplication(client, deployDir); err != nil {
		return fmt.Errorf("failed to deploy application: %w", err)
	}

	// Create systemd service
	if err := a.createSystemdService(client, deployDir); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Start service
	if _, err := client.ExecuteCommand(fmt.Sprintf("sudo systemctl start %s", a.config.App.Name)); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Perform health check
	if err := a.performHealthCheck(client); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	log.Println("Rolling deployment completed successfully")
	return nil
}

// updateSystem updates the system packages
func (a *Agent) updateSystem(client SSHClientInterface) error {
	log.Println("Updating system packages...")

	commands := []string{
		"sudo apt-get update",
		"sudo apt-get upgrade -y",
	}

	for _, cmd := range commands {
		result, err := client.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmd, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("command failed '%s': %s", cmd, result.Stderr)
		}
	}

	return nil
}

// installDependencies installs required dependencies
func (a *Agent) installDependencies(client SSHClientInterface) error {
	if len(a.config.App.Dependencies) == 0 {
		return nil
	}

	log.Println("Installing dependencies...")

	deps := strings.Join(a.config.App.Dependencies, " ")
	cmd := fmt.Sprintf("sudo apt-get install -y %s", deps)

	result, err := client.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("dependency installation failed: %s", result.Stderr)
	}

	return nil
}

// setupAppUser creates application user if it doesn't exist
func (a *Agent) setupAppUser(client SSHClientInterface) error {
	log.Println("Setting up application user...")

	username := a.config.App.Name
	
	// Check if user exists
	result, err := client.ExecuteCommand(fmt.Sprintf("id %s", username))
	if err == nil && result.ExitCode == 0 {
		log.Printf("User %s already exists", username)
		return nil
	}

	// Create user
	cmd := fmt.Sprintf("sudo useradd -r -s /bin/false -d /opt/%s %s", username, username)
	result, err = client.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("user creation failed: %s", result.Stderr)
	}

	return nil
}

// setupDirectories creates necessary directories
func (a *Agent) setupDirectories(client SSHClientInterface) error {
	log.Println("Setting up directories...")

	dirs := []string{
		fmt.Sprintf("/opt/%s", a.config.App.Name),
		fmt.Sprintf("/var/log/%s", a.config.App.Name),
		fmt.Sprintf("/var/lib/%s", a.config.App.Name),
	}

	for _, dir := range dirs {
		cmd := fmt.Sprintf("mkdir -p %s", dir)
		result, err := client.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("directory creation failed %s: %s", dir, result.Stderr)
		}

		// Set ownership
		chownCmd := fmt.Sprintf("sudo chown %s:%s %s", a.config.App.Name, a.config.App.Name, dir)
		if _, err := client.ExecuteCommand(chownCmd); err != nil {
			log.Printf("Warning: failed to set ownership for %s: %v", dir, err)
		}
	}

	return nil
}

// getResourceUsage retrieves server resource usage
func (a *Agent) getResourceUsage(client SSHClientInterface) *ResourceUsage {
	usage := &ResourceUsage{}

	// Get CPU usage
	if result, err := client.ExecuteCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | sed 's/%us,//'"); err == nil && result.ExitCode == 0 {
		usage.CPUUsage = strings.TrimSpace(result.Stdout)
	}

	// Get memory usage
	if result, err := client.ExecuteCommand("free | grep Mem | awk '{printf \"%.1f\", ($3/$2) * 100.0}'"); err == nil && result.ExitCode == 0 {
		usage.MemoryUsage = strings.TrimSpace(result.Stdout) + "%"
	}

	// Get disk usage
	if result, err := client.ExecuteCommand("df -h /opt | tail -1 | awk '{print $5}'"); err == nil && result.ExitCode == 0 {
		usage.DiskUsage = strings.TrimSpace(result.Stdout)
	}

	// Get load average
	if result, err := client.ExecuteCommand("uptime | awk -F'load average:' '{print $2}'"); err == nil && result.ExitCode == 0 {
		usage.LoadAverage = strings.TrimSpace(result.Stdout)
	}

	return usage
}

// deployApplicationAtomic deploys the application atomically
func (a *Agent) deployApplicationAtomic(client SSHClientInterface, state *DeploymentState) error {
	log.Println("Deploying application atomically...")

	tempDir := fmt.Sprintf("%s/.tmp-%s", state.DeployDir, state.NewVersion)
	finalDir := fmt.Sprintf("%s/%s", state.DeployDir, state.NewVersion)

	// Create temporary directory
	if _, err := client.ExecuteCommand(fmt.Sprintf("mkdir -p %s", tempDir)); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Upload binary to temporary location
	remoteBinary := fmt.Sprintf("%s/%s", tempDir, filepath.Base(a.config.App.Binary))
	if err := client.UploadFile(a.config.App.Binary, remoteBinary, 0755); err != nil {
		return fmt.Errorf("failed to upload binary: %w", err)
	}

	// Upload config file if specified
	if a.config.App.ConfigFile != "" {
		remoteConfig := fmt.Sprintf("%s/config.yaml", tempDir)
		if err := client.UploadFile(a.config.App.ConfigFile, remoteConfig, 0644); err != nil {
			return fmt.Errorf("failed to upload config: %w", err)
		}
	}

	// Atomic move to final location
	if _, err := client.ExecuteCommand(fmt.Sprintf("mv %s %s", tempDir, finalDir)); err != nil {
		return fmt.Errorf("failed to move to final location: %w", err)
	}

	// Update current symlink
	currentLink := fmt.Sprintf("%s/current", state.DeployDir)
	if _, err := client.ExecuteCommand(fmt.Sprintf("ln -sfn %s %s", finalDir, currentLink)); err != nil {
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	return nil
}

// deployApplication deploys the application (non-atomic)
func (a *Agent) deployApplication(client SSHClientInterface, deployDir string) error {
	log.Println("Deploying application...")

	// Create deployment directory
	if _, err := client.ExecuteCommand(fmt.Sprintf("mkdir -p %s", deployDir)); err != nil {
		return fmt.Errorf("failed to create deploy directory: %w", err)
	}

	// Upload binary
	remoteBinary := fmt.Sprintf("%s/%s", deployDir, filepath.Base(a.config.App.Binary))
	if err := client.UploadFile(a.config.App.Binary, remoteBinary, 0755); err != nil {
		return fmt.Errorf("failed to upload binary: %w", err)
	}

	// Upload config file if specified
	if a.config.App.ConfigFile != "" {
		remoteConfig := fmt.Sprintf("%s/config.yaml", deployDir)
		if err := client.UploadFile(a.config.App.ConfigFile, remoteConfig, 0644); err != nil {
			return fmt.Errorf("failed to upload config: %w", err)
		}
	}

	return nil
}

// createDeploymentCheckpoint creates a deployment checkpoint
func (a *Agent) createDeploymentCheckpoint(client SSHClientInterface, state *DeploymentState) error {
	log.Println("Creating deployment checkpoint...")

	checkpoint := map[string]interface{}{
		"app_name":        state.AppName,
		"deploy_dir":      state.DeployDir,
		"current_version": state.CurrentVersion,
		"new_version":     state.NewVersion,
		"backup_version":  state.BackupVersion,
		"start_time":      state.StartTime,
		"timestamp":       time.Now(),
	}

	checkpointData, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint data: %w", err)
	}

	checkpointPath := fmt.Sprintf("/tmp/checkpoint-%s-%d.json", state.AppName, time.Now().Unix())
	if err := client.WriteFile(checkpointPath, string(checkpointData), 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	state.CheckpointPath = checkpointPath
	return nil
}

// createSystemdService creates a systemd service file
func (a *Agent) createSystemdService(client SSHClientInterface, deployDir string) error {
	log.Println("Creating systemd service...")

	serviceName := a.config.App.Name
	binaryPath := fmt.Sprintf("%s/%s", deployDir, filepath.Base(a.config.App.Binary))
	
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=%s
Restart=always
RestartSec=5
Environment=PORT=%d
`, serviceName, serviceName, serviceName, deployDir, binaryPath, a.config.App.Port)

	// Add custom environment variables
	for key, value := range a.config.App.Environment {
		serviceContent += fmt.Sprintf("Environment=%s=%s\n", key, value)
	}

	serviceContent += `
[Install]
WantedBy=multi-user.target
`

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := client.WriteFile(servicePath, serviceContent, 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd and enable service
	commands := []string{
		"sudo systemctl daemon-reload",
		fmt.Sprintf("sudo systemctl enable %s", serviceName),
	}

	for _, cmd := range commands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to execute systemd command '%s': %w", cmd, err)
		}
	}

	return nil
}

// checkDiskSpace checks if there's enough disk space
func (a *Agent) checkDiskSpace(client SSHClientInterface) error {
	result, err := client.ExecuteCommand("df -h /opt | tail -1 | awk '{print $5}' | sed 's/%//'")
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("disk space check failed: %s", result.Stderr)
	}

	usage, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return fmt.Errorf("failed to parse disk usage: %w", err)
	}

	if usage > 90 {
		return fmt.Errorf("insufficient disk space: %d%% used", usage)
	}

	return nil
}

// performHealthCheck performs application health check
func (a *Agent) performHealthCheck(client SSHClientInterface) error {
	log.Println("Performing health check...")

	healthURL := fmt.Sprintf("http://localhost:%d", a.config.App.Port)
	if a.config.App.HealthPath != "" {
		healthURL += a.config.App.HealthPath
	} else {
		healthURL += "/health"
	}

	// Start the service first
	serviceName := a.config.App.Name
	if _, err := client.ExecuteCommand(fmt.Sprintf("sudo systemctl start %s", serviceName)); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait a moment for service to start
	time.Sleep(5 * time.Second)

	// Perform health check with timeout
	timeout := a.config.Deploy.HealthTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	healthCmd := fmt.Sprintf("curl -f -s -m 10 -w '%%{http_code}:%%{time_total}:%%{size_download}' %s", healthURL)
	
	start := time.Now()
	for time.Since(start) < timeout {
		result, err := client.ExecuteCommand(healthCmd)
		if err == nil && result.ExitCode == 0 {
			log.Printf("Health check passed: %s", result.Stdout)
			return nil
		}
		
		log.Printf("Health check attempt failed, retrying in 10 seconds...")
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("health check failed after %v", timeout)
}

// rollback performs deployment rollback
func (a *Agent) rollback(client SSHClientInterface, state *DeploymentState) error {
	log.Println("Performing rollback...")

	serviceName := a.config.App.Name
	
	// Stop the service
	if _, err := client.ExecuteCommand(fmt.Sprintf("sudo systemctl stop %s", serviceName)); err != nil {
		log.Printf("Warning: failed to stop service during rollback: %v", err)
	}

	// Restore previous version if it exists
	backupPath := fmt.Sprintf("%s/%s", state.DeployDir, state.BackupVersion)
	currentPath := fmt.Sprintf("%s/current", state.DeployDir)
	
	exists, err := client.DirectoryExists(backupPath)
	if err != nil {
		return fmt.Errorf("failed to check backup directory: %w", err)
	}

	if exists {
		// Restore from backup
		if _, err := client.ExecuteCommand(fmt.Sprintf("ln -sfn %s %s", backupPath, currentPath)); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
		
		// Start the service
		if _, err := client.ExecuteCommand(fmt.Sprintf("sudo systemctl start %s", serviceName)); err != nil {
			return fmt.Errorf("failed to start service after rollback: %w", err)
		}
	}

	return nil
}

// GetDeploymentHistory returns deployment history
func (a *Agent) GetDeploymentHistory(serverName string, limit int) ([]map[string]interface{}, error) {
	serverConfig, err := a.config.GetServerConfig(serverName)
	if err != nil {
		return nil, err
	}

	client, err := NewSSHClient(a.config, serverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer client.Close()

	// List checkpoint files
	result, err := client.ExecuteCommand("ls -la /tmp/checkpoint-*.json 2>/dev/null | tail -n " + strconv.Itoa(limit))
	if err != nil || result.ExitCode != 0 {
		return []map[string]interface{}{}, nil // Return empty history if no checkpoints found
	}

	var history []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		
		filename := fields[8]
		
		// Read checkpoint file
		content, err := client.ReadFile(filename)
		if err != nil {
			continue
		}
		
		var checkpoint map[string]interface{}
		if err := json.Unmarshal([]byte(content), &checkpoint); err != nil {
			continue
		}
		
		history = append(history, checkpoint)
	}

	return history, nil
}