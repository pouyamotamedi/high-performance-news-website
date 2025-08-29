package deployment

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "/path/to/binary",
			Port:   8080,
		},
	}

	agent := NewAgent(config)
	assert.NotNil(t, agent)
	assert.Equal(t, config, agent.config)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
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
			},
			wantErr: false,
		},
		{
			name: "missing app name",
			config: &Config{
				App: AppConfig{
					Binary: "/path/to/binary",
					Port:   8080,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock SSH client for testing deployment logic
type MockSSHClient struct {
	commands []string
	results  map[string]*CommandResult
	files    map[string]string
}

func NewMockSSHClient() *MockSSHClient {
	return &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}
}

func (m *MockSSHClient) ExecuteCommand(command string) (*CommandResult, error) {
	m.commands = append(m.commands, command)
	
	if result, exists := m.results[command]; exists {
		return result, nil
	}
	
	// Default success result
	return &CommandResult{
		Command:  command,
		Stdout:   "success",
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

func (m *MockSSHClient) ExecuteScript(script string) (*CommandResult, error) {
	return m.ExecuteCommand(script)
}

func (m *MockSSHClient) WriteFile(remotePath, content string, mode os.FileMode) error {
	m.files[remotePath] = content
	return nil
}

func (m *MockSSHClient) ReadFile(remotePath string) (string, error) {
	if content, exists := m.files[remotePath]; exists {
		return content, nil
	}
	return "", os.ErrNotExist
}

func (m *MockSSHClient) FileExists(remotePath string) (bool, error) {
	_, exists := m.files[remotePath]
	return exists, nil
}

func (m *MockSSHClient) DirectoryExists(remotePath string) (bool, error) {
	return true, nil // Assume directories exist for testing
}

func (m *MockSSHClient) UploadFile(localPath, remotePath string, mode os.FileMode) error {
	// Simulate file upload
	m.files[remotePath] = "uploaded content"
	return nil
}

func (m *MockSSHClient) DownloadFile(remotePath, localPath string) error {
	return nil
}

func (m *MockSSHClient) GetSystemInfo() (*SystemInfo, error) {
	return &SystemInfo{
		OS:        "Linux test 5.4.0",
		CPUCores:  "4",
		Memory:    "8G",
		DiskSpace: "100G",
	}, nil
}

func (m *MockSSHClient) Close() error {
	return nil
}

func (m *MockSSHClient) SetCommandResult(command string, result *CommandResult) {
	m.results[command] = result
}

func (m *MockSSHClient) GetExecutedCommands() []string {
	return m.commands
}

func (m *MockSSHClient) GetUploadedFiles() map[string]string {
	return m.files
}

func TestDeploymentValidation(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "test-binary",
			Port:   8080,
		},
		Deploy: DeployConfig{
			ValidationChecks: []string{
				"which curl",
				"systemctl --version",
			},
		},
	}

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	// Test successful validation
	err := agent.validateDeployment(mockClient)
	assert.NoError(t, err)

	commands := mockClient.GetExecutedCommands()
	assert.Contains(t, commands, "which curl")
	assert.Contains(t, commands, "systemctl --version")
}

func TestSystemdServiceCreation(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "test-binary",
			Port:   8080,
			Environment: map[string]string{
				"ENV":       "production",
				"LOG_LEVEL": "info",
			},
		},
	}

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	err := agent.createSystemdService(mockClient, "/opt/test-app")
	assert.NoError(t, err)

	// Check if service file was created
	serviceContent, exists := mockClient.files["/etc/systemd/system/test-app.service"]
	assert.True(t, exists)
	assert.Contains(t, serviceContent, "Description=test-app")
	assert.Contains(t, serviceContent, "ExecStart=/opt/test-app/test-binary")
	assert.Contains(t, serviceContent, "Environment=PORT=8080")
	assert.Contains(t, serviceContent, "Environment=ENV=production")
	assert.Contains(t, serviceContent, "Environment=LOG_LEVEL=info")

	// Check if systemd commands were executed
	commands := mockClient.GetExecutedCommands()
	assert.Contains(t, commands, "sudo systemctl daemon-reload")
	assert.Contains(t, commands, "sudo systemctl enable test-app")
}

func TestAutomatedServerSetup(t *testing.T) {
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

func TestBlueGreenDeploymentWithRollback(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:   "test-app",
			Binary: "test-binary",
			Port:   8080,
		},
		Deploy: DeployConfig{
			Strategy:      "blue-green",
			HealthTimeout: 30 * time.Second,
		},
	}

	agent := NewAgent(config)
	mockClient := NewMockSSHClient()

	// Create a test binary file
	tmpBinary, err := os.CreateTemp("", "test-binary-*")
	require.NoError(t, err)
	defer os.Remove(tmpBinary.Name())
	tmpBinary.WriteString("#!/bin/bash\necho 'test binary'")
	tmpBinary.Close()
	os.Chmod(tmpBinary.Name(), 0755)
	config.App.Binary = tmpBinary.Name()

	// Set up mock responses for successful deployment
	mockClient.SetCommandResult("sudo systemctl is-active test-app", &CommandResult{
		Command:  "sudo systemctl is-active test-app",
		Stdout:   "active",
		ExitCode: 0,
	})

	// Mock disk space check
	mockClient.SetCommandResult("df -h /opt | tail -1 | awk '{print $5}' | sed 's/%//'", &CommandResult{
		Command:  "df -h /opt | tail -1 | awk '{print $5}' | sed 's/%//'",
		Stdout:   "50",
		ExitCode: 0,
	})

	// Test successful blue-green deployment
	err = agent.blueGreenDeploy(mockClient)
	assert.NoError(t, err)

	commands := mockClient.GetExecutedCommands()
	
	// Check that a temporary directory was created (atomic deployment)
	foundTempDir := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "mkdir -p /opt/test-app/.tmp-") {
			foundTempDir = true
			break
		}
	}
	assert.True(t, foundTempDir, "Should create temporary directory for atomic deployment")
	
	assert.Contains(t, commands, "sudo systemctl daemon-reload")
	assert.Contains(t, commands, "sudo systemctl enable test-app")
}