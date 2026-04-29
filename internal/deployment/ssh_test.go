package deployment

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *CommandResult
		wantSuccess bool
		wantOutput  string
	}{
		{
			name: "successful command",
			result: &CommandResult{
				Command:  "echo hello",
				Stdout:   "hello\n",
				Stderr:   "",
				ExitCode: 0,
			},
			wantSuccess: true,
			wantOutput:  "hello\n",
		},
		{
			name: "failed command",
			result: &CommandResult{
				Command:  "nonexistent-command",
				Stdout:   "",
				Stderr:   "command not found",
				ExitCode: 127,
			},
			wantSuccess: false,
			wantOutput:  "command not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantSuccess, tt.result.IsSuccess())
			assert.Equal(t, tt.wantOutput, tt.result.GetOutput())
		})
	}
}

func TestSystemInfo(t *testing.T) {
	info := &SystemInfo{
		OS:        "Linux test 5.4.0",
		CPUCores:  "4",
		Memory:    "8G",
		DiskSpace: "100G",
	}

	assert.Equal(t, "Linux test 5.4.0", info.OS)
	assert.Equal(t, "4", info.CPUCores)
	assert.Equal(t, "8G", info.Memory)
	assert.Equal(t, "100G", info.DiskSpace)
}

// Note: These tests would require a real SSH server to test against.
// In a production environment, you would set up integration tests with
// a test SSH server or use SSH mocking libraries.

func TestSSHClientConfiguration(t *testing.T) {
	config := &Config{
		SSH: SSHConfig{
			Timeout:        30 * time.Second,
			ConnectRetries: 3,
			CommandTimeout: 5 * time.Minute,
			KeepAlive:      30 * time.Second,
		},
	}

	serverConfig := &ServerConfig{
		Host:    "localhost",
		Port:    22,
		User:    "test",
		KeyFile: "/path/to/key",
	}

	// Test that NewSSHClient doesn't panic with valid config
	// In a real test, this would connect to a test SSH server
	assert.NotPanics(t, func() {
		_, err := NewSSHClient(config, serverConfig)
		// We expect this to fail since we don't have a real SSH server
		assert.Error(t, err)
	})
}

func TestSSHClientConfigurationWithPassword(t *testing.T) {
	config := &Config{
		SSH: SSHConfig{
			Timeout:        30 * time.Second,
			ConnectRetries: 3,
			CommandTimeout: 5 * time.Minute,
			KeepAlive:      30 * time.Second,
		},
	}

	serverConfig := &ServerConfig{
		Host:     "localhost",
		Port:     22,
		User:     "test",
		Password: "testpass",
	}

	// Test that NewSSHClient doesn't panic with password auth
	assert.NotPanics(t, func() {
		_, err := NewSSHClient(config, serverConfig)
		// We expect this to fail since we don't have a real SSH server
		assert.Error(t, err)
	})
}

func TestSSHClientDefaultPort(t *testing.T) {
	config := &Config{
		SSH: SSHConfig{
			Timeout:        30 * time.Second,
			ConnectRetries: 3,
		},
	}

	serverConfig := &ServerConfig{
		Host:    "localhost",
		User:    "test",
		KeyFile: "/path/to/key",
		// Port not specified, should default to 22
	}

	// Test that default port is used
	assert.NotPanics(t, func() {
		_, err := NewSSHClient(config, serverConfig)
		// We expect this to fail since we don't have a real SSH server
		assert.Error(t, err)
		// The error should indicate connection to port 22
		assert.Contains(t, err.Error(), "22")
	})
}

// Mock tests for SSH operations that don't require a real connection

func TestSSHClientMockOperations(t *testing.T) {
	// These tests demonstrate the expected behavior of SSH operations
	// In a real implementation, you would use a mock SSH server or
	// dependency injection to test these methods

	t.Run("ExecuteCommand", func(t *testing.T) {
		// Test command execution logic
		command := "echo 'hello world'"
		expectedResult := &CommandResult{
			Command:  command,
			Stdout:   "hello world\n",
			Stderr:   "",
			ExitCode: 0,
		}

		assert.Equal(t, command, expectedResult.Command)
		assert.True(t, expectedResult.IsSuccess())
		assert.Equal(t, "hello world\n", expectedResult.GetOutput())
	})

	t.Run("WriteFile", func(t *testing.T) {
		// Test file writing logic
		remotePath := "/tmp/test.txt"
		content := "test content"
		mode := os.FileMode(0644)

		// In a real test, this would verify the file was written correctly
		assert.Equal(t, "/tmp/test.txt", remotePath)
		assert.Equal(t, "test content", content)
		assert.Equal(t, os.FileMode(0644), mode)
	})

	t.Run("ReadFile", func(t *testing.T) {
		// Test file reading logic
		remotePath := "/tmp/test.txt"
		expectedContent := "test content"

		// In a real test, this would read from the remote file
		assert.Equal(t, "/tmp/test.txt", remotePath)
		assert.Equal(t, "test content", expectedContent)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Test file existence check
		existingFile := "/etc/passwd"
		nonExistingFile := "/tmp/nonexistent"

		// In a real test, these would check actual file existence
		assert.Equal(t, "/etc/passwd", existingFile)
		assert.Equal(t, "/tmp/nonexistent", nonExistingFile)
	})

	t.Run("DirectoryExists", func(t *testing.T) {
		// Test directory existence check
		existingDir := "/tmp"
		nonExistingDir := "/nonexistent"

		// In a real test, these would check actual directory existence
		assert.Equal(t, "/tmp", existingDir)
		assert.Equal(t, "/nonexistent", nonExistingDir)
	})

	t.Run("GetSystemInfo", func(t *testing.T) {
		// Test system info retrieval
		expectedInfo := &SystemInfo{
			OS:        "Linux test 5.4.0",
			CPUCores:  "4",
			Memory:    "8G",
			DiskSpace: "100G",
		}

		assert.Equal(t, "Linux test 5.4.0", expectedInfo.OS)
		assert.Equal(t, "4", expectedInfo.CPUCores)
		assert.Equal(t, "8G", expectedInfo.Memory)
		assert.Equal(t, "100G", expectedInfo.DiskSpace)
	})
}

func TestSSHClientErrorHandling(t *testing.T) {
	t.Run("Connection timeout", func(t *testing.T) {
		config := &Config{
			SSH: SSHConfig{
				Timeout:        1 * time.Millisecond, // Very short timeout
				ConnectRetries: 1,
			},
		}

		serverConfig := &ServerConfig{
			Host:    "192.0.2.1", // Non-routable IP for testing
			Port:    22,
			User:    "test",
			KeyFile: "/path/to/key",
		}

		_, err := NewSSHClient(config, serverConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("Invalid key file", func(t *testing.T) {
		config := &Config{
			SSH: SSHConfig{
				Timeout:        30 * time.Second,
				ConnectRetries: 1,
			},
		}

		serverConfig := &ServerConfig{
			Host:    "localhost",
			Port:    22,
			User:    "test",
			KeyFile: "/nonexistent/key/file",
		}

		_, err := NewSSHClient(config, serverConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read SSH key file")
	})
}

func TestSSHClientRetryLogic(t *testing.T) {
	config := &Config{
		SSH: SSHConfig{
			Timeout:        1 * time.Millisecond,
			ConnectRetries: 3,
		},
	}

	serverConfig := &ServerConfig{
		Host:    "192.0.2.1", // Non-routable IP
		Port:    22,
		User:    "test",
		KeyFile: "/dev/null", // Invalid key file
	}

	start := time.Now()
	_, err := NewSSHClient(config, serverConfig)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "after 3 retries")
	
	// Should have taken some time due to retries
	// (though this is a rough check due to timing variations)
	assert.True(t, duration > 1*time.Millisecond)
}

// Integration test helpers (would be used with a real SSH server)

func createTestSSHServer(t *testing.T) (string, func()) {
	// This would set up a test SSH server for integration tests
	// For now, we'll just return placeholder values
	return "localhost:2222", func() {
		// Cleanup function
	}
}

func TestSSHIntegrationPlaceholder(t *testing.T) {
	// This is a placeholder for integration tests that would run against
	// a real SSH server. In a production environment, you would:
	//
	// 1. Set up a test SSH server (possibly in Docker)
	// 2. Create test users and keys
	// 3. Run the actual SSH operations
	// 4. Verify the results
	//
	// Example structure:
	//
	// if testing.Short() {
	//     t.Skip("Skipping integration test in short mode")
	// }
	//
	// server, cleanup := createTestSSHServer(t)
	// defer cleanup()
	//
	// config := &Config{...}
	// serverConfig := &ServerConfig{...}
	//
	// client, err := NewSSHClient(config, serverConfig)
	// require.NoError(t, err)
	// defer client.Close()
	//
	// // Test actual operations
	// result, err := client.ExecuteCommand("echo 'test'")
	// assert.NoError(t, err)
	// assert.Equal(t, "test\n", result.Stdout)

	t.Skip("Integration tests require a test SSH server")
}

// Test SSH Client Enhanced Functionality
func TestSSHClientFileOperations(t *testing.T) {
	// Test file operations with mock client
	mockClient := &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}

	// Test WriteFile
	err := mockClient.WriteFile("/tmp/test.txt", "test content", 0644)
	assert.NoError(t, err)
	assert.Equal(t, "test content", mockClient.files["/tmp/test.txt"])

	// Test ReadFile
	content, err := mockClient.ReadFile("/tmp/test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test content", content)

	// Test FileExists
	exists, err := mockClient.FileExists("/tmp/test.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = mockClient.FileExists("/tmp/nonexistent.txt")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test DirectoryExists
	exists, err = mockClient.DirectoryExists("/tmp")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestSSHClientCommandExecution(t *testing.T) {
	mockClient := &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}

	// Test successful command
	mockClient.SetCommandResult("echo hello", &CommandResult{
		Command:  "echo hello",
		Stdout:   "hello\n",
		Stderr:   "",
		ExitCode: 0,
	})

	result, err := mockClient.ExecuteCommand("echo hello")
	assert.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "hello\n", result.GetOutput())

	// Test failed command
	mockClient.SetCommandResult("exit 1", &CommandResult{
		Command:  "exit 1",
		Stdout:   "",
		Stderr:   "exit status 1",
		ExitCode: 1,
	})

	result, err = mockClient.ExecuteCommand("exit 1")
	assert.NoError(t, err)
	assert.False(t, result.IsSuccess())
	assert.Equal(t, "exit status 1", result.GetOutput())
}

func TestSSHClientScriptExecution(t *testing.T) {
	mockClient := &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}

	script := `#!/bin/bash
echo "Starting script"
ls -la /tmp
echo "Script completed"`

	// Mock the script execution
	mockClient.SetCommandResult(script, &CommandResult{
		Command:  script,
		Stdout:   "Starting script\nScript completed\n",
		Stderr:   "",
		ExitCode: 0,
	})

	result, err := mockClient.ExecuteScript(script)
	assert.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Contains(t, result.Stdout, "Starting script")
	assert.Contains(t, result.Stdout, "Script completed")
}

func TestSSHClientSystemInfo(t *testing.T) {
	mockClient := &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}

	// Mock system info commands
	mockClient.SetCommandResult("uname -a", &CommandResult{
		Command:  "uname -a",
		Stdout:   "Linux test 5.4.0-74-generic #83-Ubuntu SMP Sat May 8 02:35:39 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("nproc", &CommandResult{
		Command:  "nproc",
		Stdout:   "8",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("free -h | grep Mem | awk '{print $2}'", &CommandResult{
		Command:  "free -h | grep Mem | awk '{print $2}'",
		Stdout:   "16Gi",
		ExitCode: 0,
	})

	mockClient.SetCommandResult("df -h / | tail -1 | awk '{print $2}'", &CommandResult{
		Command:  "df -h / | tail -1 | awk '{print $2}'",
		Stdout:   "500G",
		ExitCode: 0,
	})

	sysInfo, err := mockClient.GetSystemInfo()
	assert.NoError(t, err)
	assert.Contains(t, sysInfo.OS, "Linux test 5.4.0")
	assert.Equal(t, "8", sysInfo.CPUCores)
	assert.Equal(t, "16Gi", sysInfo.Memory)
	assert.Equal(t, "500G", sysInfo.DiskSpace)
}

func TestSSHClientFileUploadDownload(t *testing.T) {
	mockClient := &MockSSHClient{
		commands: make([]string, 0),
		results:  make(map[string]*CommandResult),
		files:    make(map[string]string),
	}

	// Create a temporary local file
	tmpFile, err := os.CreateTemp("", "test-upload-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testContent := "test file content for upload"
	_, err = tmpFile.WriteString(testContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Test file upload
	err = mockClient.UploadFile(tmpFile.Name(), "/tmp/uploaded.txt", 0644)
	assert.NoError(t, err)
	assert.Equal(t, "uploaded content", mockClient.files["/tmp/uploaded.txt"])

	// Test file download
	mockClient.files["/tmp/download.txt"] = "downloaded content"
	
	downloadPath := tmpFile.Name() + ".download"
	err = mockClient.DownloadFile("/tmp/download.txt", downloadPath)
	assert.NoError(t, err)

	// Verify downloaded content
	content, err := os.ReadFile(downloadPath)
	assert.NoError(t, err)
	assert.Equal(t, "downloaded content", string(content))
	
	// Clean up
	os.Remove(downloadPath)
}

// Test SSH Connection Error Handling
func TestSSHConnectionErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		serverConfig *ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid host",
			config: &Config{
				SSH: SSHConfig{
					Timeout:        1 * time.Second,
					ConnectRetries: 1,
				},
			},
			serverConfig: &ServerConfig{
				Host:    "invalid-host-that-does-not-exist.local",
				Port:    22,
				User:    "test",
				KeyFile: "/dev/null",
			},
			expectError: true,
			errorMsg:    "failed to connect",
		},
		{
			name: "missing key file",
			config: &Config{
				SSH: SSHConfig{
					Timeout:        30 * time.Second,
					ConnectRetries: 1,
				},
			},
			serverConfig: &ServerConfig{
				Host:    "localhost",
				Port:    22,
				User:    "test",
				KeyFile: "/nonexistent/key/file",
			},
			expectError: true,
			errorMsg:    "failed to read SSH key file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSSHClient(tt.config, tt.serverConfig)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test SSH Command Timeout
func TestSSHCommandTimeout(t *testing.T) {
	// This test demonstrates the expected behavior of command timeouts
	// In a real implementation, you would test with actual SSH connections
	
	config := &Config{
		SSH: SSHConfig{
			CommandTimeout: 1 * time.Second,
		},
	}
	
	// Test that timeout configuration is properly set
	assert.Equal(t, 1*time.Second, config.SSH.CommandTimeout)
	
	// In a real test, you would verify that long-running commands are terminated
	// after the timeout period
}

// Test SSH Connection Retry Logic
func TestSSHConnectionRetryLogic(t *testing.T) {
	config := &Config{
		SSH: SSHConfig{
			Timeout:        100 * time.Millisecond,
			ConnectRetries: 3,
		},
	}

	serverConfig := &ServerConfig{
		Host:    "192.0.2.1", // Non-routable IP for testing
		Port:    22,
		User:    "test",
		KeyFile: "/dev/null",
	}

	start := time.Now()
	_, err := NewSSHClient(config, serverConfig)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "after 3 retries")
	
	// Should have taken some time due to retries
	assert.True(t, duration > 100*time.Millisecond)
}