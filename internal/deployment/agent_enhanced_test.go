package deployment

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Enhanced Server Management
func TestGetServerStatusEnhanced(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name: "test-app",
			Port: 8080,
		},
		Servers: map[string]ServerConfig{
			"test": {
				Host:    "localhost",
				User:    "test",
				KeyFile: "/path/to/key",
			},
		},
	}

	agent := NewAgent(config)

	// Test that the method exists and has correct signature
	assert.NotNil(t, agent.GetServerStatus)
}

func TestValidateConfigurationEnhanced(t *testing.T) {
	// Create a temporary binary file
	tmpBinary, err := os.CreateTemp("", "test-binary-*")
	require.NoError(t, err)
	defer os.Remove(tmpBinary.Name())
	tmpBinary.WriteString("#!/bin/bash\necho 'test binary'")
	tmpBinary.Close()
	os.Chmod(tmpBinary.Name(), 0755)

	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: tmpBinary.Name(),
			Port:   8080,
		},
	}

	agent := NewAgent(config)
	err = agent.ValidateConfiguration()
	assert.NoError(t, err)
}

func TestValidateConfigurationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "missing binary",
			config: &Config{
				App: AppConfig{
					Name:   "test-app",
					Binary: "/nonexistent/binary",
					Port:   8080,
				},
			},
			wantErr: "binary file not found",
		},
		{
			name: "empty binary path",
			config: &Config{
				App: AppConfig{
					Name: "test-app",
					Port: 8080,
				},
			},
			wantErr: "application binary not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(tt.config)
			err := agent.ValidateConfiguration()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// Test Blue-Green Deployment with Enhanced Rollback
func TestBlueGreenDeploymentWithAutomaticRollbackEnhanced(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "test-binary",
			Port:   8080,
		},
		Deploy: DeployConfig{
			Strategy:      "blue-green",
			HealthTimeout: 10 * time.Second,
		},
	}

	// Create a test binary file
	tmpBinary, err := os.CreateTemp("", "test-binary-*")
	require.NoError(t, err)
	defer os.Remove(tmpBinary.Name())
	tmpBinary.WriteString("#!/bin/bash\necho 'test binary'")
	tmpBinary.Close()
	os.Chmod(tmpBinary.Name(), 0755)
	config.App.Binary = tmpBinary.Name()

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	// Set up successful pre-deployment checks
	mockClient.SetCommandResult("sudo systemctl is-active test-app", &CommandResult{
		Command:  "sudo systemctl is-active test-app",
		Stdout:   "active",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("df -h /opt | tail -1 | awk '{print $5}' | sed 's/%//'", &CommandResult{
		Command:  "df -h /opt | tail -1 | awk '{print $5}' | sed 's/%//'",
		Stdout:   "50",
		ExitCode: 0,
	})

	// Set up health check failure to trigger rollback
	healthCheckCmd := "curl -f -s -m 10 -w '%{http_code}:%{time_total}:%{size_download}' http://localhost:8080/health"
	mockClient.SetCommandResult(healthCheckCmd, &CommandResult{
		Command:  healthCheckCmd,
		Stdout:   "",
		Stderr:   "Connection refused",
		ExitCode: 7,
	})

	// Test that deployment fails and triggers rollback
	err = agent.blueGreenDeploy(mockClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed, rolled back successfully")

	// Verify rollback commands were executed
	commands := mockClient.GetExecutedCommands()
	assert.Contains(t, commands, "sudo systemctl stop test-app")
}

// Test Automated Server Setup
func TestAutomatedServerSetupEnhanced(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name: "test-app",
			Dependencies: []string{
				"postgresql-client",
				"redis-tools",
			},
		},
	}

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	// Test server setup components
	err := agent.updateSystem(mockClient)
	assert.NoError(t, err)

	err = agent.installDependencies(mockClient)
	assert.NoError(t, err)

	err = agent.setupAppUser(mockClient)
	assert.NoError(t, err)

	err = agent.setupDirectories(mockClient)
	assert.NoError(t, err)

	commands := mockClient.GetExecutedCommands()

	// Verify system update commands
	assert.Contains(t, commands, "sudo apt-get update")
	assert.Contains(t, commands, "sudo apt-get upgrade -y")

	// Verify dependency installation
	foundInstallCmd := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "sudo apt-get install -y") && 
		   strings.Contains(cmd, "postgresql-client") && 
		   strings.Contains(cmd, "redis-tools") {
			foundInstallCmd = true
			break
		}
	}
	assert.True(t, foundInstallCmd, "Dependency installation command should be executed")

	// Verify directory setup
	assert.Contains(t, commands, "mkdir -p /opt/test-app")
	assert.Contains(t, commands, "mkdir -p /var/log/test-app")
	assert.Contains(t, commands, "mkdir -p /var/lib/test-app")
}

// Test Resource Usage Monitoring
func TestResourceUsageMonitoringEnhanced(t *testing.T) {
	config := &Config{}
	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	// Set up resource usage mock responses
	mockClient.SetCommandResult("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | sed 's/%us,//'", &CommandResult{
		Command:  "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | sed 's/%us,//'",
		Stdout:   "25.3",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("free | grep Mem | awk '{printf \"%.1f\", ($3/$2) * 100.0}'", &CommandResult{
		Command:  "free | grep Mem | awk '{printf \"%.1f\", ($3/$2) * 100.0}'",
		Stdout:   "67.8",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("df -h /opt | tail -1 | awk '{print $5}'", &CommandResult{
		Command:  "df -h /opt | tail -1 | awk '{print $5}'",
		Stdout:   "45%",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("uptime | awk -F'load average:' '{print $2}'", &CommandResult{
		Command:  "uptime | awk -F'load average:' '{print $2}'",
		Stdout:   " 1.23, 1.45, 1.67",
		ExitCode: 0,
	})

	usage := agent.getResourceUsage(mockClient)
	assert.NotNil(t, usage)
	assert.Equal(t, "25.3", usage.CPUUsage)
	assert.Equal(t, "67.8%", usage.MemoryUsage)
	assert.Equal(t, "45%", usage.DiskUsage)
	assert.Equal(t, "1.23, 1.45, 1.67", usage.LoadAverage)
}

// Test Atomic Operations
func TestAtomicDeploymentOperationsEnhanced(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "test-binary",
		},
	}

	// Create a test binary file
	tmpBinary, err := os.CreateTemp("", "test-binary-*")
	require.NoError(t, err)
	defer os.Remove(tmpBinary.Name())
	tmpBinary.WriteString("#!/bin/bash\necho 'test binary'")
	tmpBinary.Close()
	os.Chmod(tmpBinary.Name(), 0755)
	config.App.Binary = tmpBinary.Name()

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	state := &DeploymentState{
		AppName:        "test-app",
		DeployDir:      "/opt/test-app",
		CurrentVersion: "test-app-current",
		NewVersion:     "test-app-1640995200",
		BackupVersion:  "test-app-backup",
		StartTime:      time.Now(),
	}

	// Test atomic application deployment
	err = agent.deployApplicationAtomic(mockClient, state)
	assert.NoError(t, err)

	commands := mockClient.GetExecutedCommands()
	
	// Verify temporary directory was used
	foundTempDir := false
	for _, cmd := range commands {
		if strings.Contains(cmd, ".tmp-test-app-1640995200") {
			foundTempDir = true
			break
		}
	}
	assert.True(t, foundTempDir, "Should use temporary directory for atomic deployment")

	// Verify atomic move operation
	foundMoveCmd := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "mv") && strings.Contains(cmd, ".tmp-") {
			foundMoveCmd = true
			break
		}
	}
	assert.True(t, foundMoveCmd, "Should perform atomic move operation")
}

// Test Deployment State Tracking and Checkpoints
func TestDeploymentStateTrackingEnhanced(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name: "test-app",
		},
	}

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	state := &DeploymentState{
		AppName:        "test-app",
		DeployDir:      "/opt/test-app",
		CurrentVersion: "test-app-current",
		NewVersion:     "test-app-1640995200",
		BackupVersion:  "test-app-backup",
		StartTime:      time.Now(),
	}

	// Test checkpoint creation
	err := agent.createDeploymentCheckpoint(mockClient, state)
	assert.NoError(t, err)
	assert.NotEmpty(t, state.CheckpointPath)

	// Verify checkpoint file was created
	files := mockClient.GetUploadedFiles()
	found := false
	for path := range files {
		if strings.Contains(path, "checkpoint-") && strings.HasSuffix(path, ".json") {
			found = true
			break
		}
	}
	assert.True(t, found, "Checkpoint file should be created")
}