package deployment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClientInterface defines the interface for SSH operations
type SSHClientInterface interface {
	Close() error
	ExecuteCommand(command string) (*CommandResult, error)
	ExecuteScript(script string) (*CommandResult, error)
	WriteFile(remotePath, content string, mode os.FileMode) error
	ReadFile(remotePath string) (string, error)
	FileExists(remotePath string) (bool, error)
	DirectoryExists(remotePath string) (bool, error)
	UploadFile(localPath, remotePath string, mode os.FileMode) error
	DownloadFile(remotePath, localPath string) error
	GetSystemInfo() (*SystemInfo, error)
}

// SSHClient represents an SSH connection to a remote server
type SSHClient struct {
	client *ssh.Client
	config *Config
	server *ServerConfig
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(config *Config, server *ServerConfig) (*SSHClient, error) {
	var authMethods []ssh.AuthMethod

	// Add key-based authentication if key file is provided
	if server.KeyFile != "" {
		key, err := os.ReadFile(server.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password authentication if password is provided
	if server.Password != "" {
		authMethods = append(authMethods, ssh.Password(server.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            server.User,
		Auth:            authMethods,
		Timeout:         config.SSH.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
	}

	port := server.Port
	if port == 0 {
		port = 22
	}

	address := fmt.Sprintf("%s:%d", server.Host, port)

	var client *ssh.Client
	var err error

	// Retry connection with exponential backoff
	for i := 0; i < config.SSH.ConnectRetries; i++ {
		client, err = ssh.Dial("tcp", address, sshConfig)
		if err == nil {
			break
		}

		if i < config.SSH.ConnectRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server after %d retries: %w", config.SSH.ConnectRetries, err)
	}

	return &SSHClient{
		client: client,
		config: config,
		server: server,
	}, nil
}

// Close closes the SSH connection
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ExecuteCommand executes a command on the remote server
func (c *SSHClient) ExecuteCommand(command string) (*CommandResult, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Set timeout for command execution
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case err := <-done:
		result := &CommandResult{
			Command:  command,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: 0,
		}

		if err != nil {
			if exitError, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitError.ExitStatus()
			} else {
				return nil, fmt.Errorf("command execution failed: %w", err)
			}
		}

		return result, nil

	case <-time.After(c.config.SSH.CommandTimeout):
		session.Signal(ssh.SIGKILL)
		return nil, fmt.Errorf("command timed out after %v", c.config.SSH.CommandTimeout)
	}
}

// ExecuteScript executes a multi-line script on the remote server
func (c *SSHClient) ExecuteScript(script string) (*CommandResult, error) {
	// Create a temporary script file
	tempScript := fmt.Sprintf("/tmp/deploy_script_%d.sh", time.Now().Unix())
	
	// Upload the script
	if err := c.WriteFile(tempScript, script, 0755); err != nil {
		return nil, fmt.Errorf("failed to upload script: %w", err)
	}

	// Execute the script
	result, err := c.ExecuteCommand(fmt.Sprintf("bash %s", tempScript))
	
	// Clean up the temporary script
	c.ExecuteCommand(fmt.Sprintf("rm -f %s", tempScript))
	
	return result, err
}

// WriteFile writes content to a file on the remote server
func (c *SSHClient) WriteFile(remotePath, content string, mode os.FileMode) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Create the directory if it doesn't exist
	dir := filepath.Dir(remotePath)
	if dir != "." {
		if _, err := c.ExecuteCommand(fmt.Sprintf("mkdir -p %s", dir)); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Use SCP to transfer the file
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		
		fmt.Fprintf(w, "C%#o %d %s\n", mode, len(content), filepath.Base(remotePath))
		fmt.Fprint(w, content)
		fmt.Fprint(w, "\x00")
	}()

	if err := session.Run(fmt.Sprintf("scp -t %s", remotePath)); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadFile reads content from a file on the remote server
func (c *SSHClient) ReadFile(remotePath string) (string, error) {
	result, err := c.ExecuteCommand(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return "", err
	}
	
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to read file: %s", result.Stderr)
	}
	
	return result.Stdout, nil
}

// FileExists checks if a file exists on the remote server
func (c *SSHClient) FileExists(remotePath string) (bool, error) {
	result, err := c.ExecuteCommand(fmt.Sprintf("test -f %s", remotePath))
	if err != nil {
		return false, err
	}
	
	return result.ExitCode == 0, nil
}

// DirectoryExists checks if a directory exists on the remote server
func (c *SSHClient) DirectoryExists(remotePath string) (bool, error) {
	result, err := c.ExecuteCommand(fmt.Sprintf("test -d %s", remotePath))
	if err != nil {
		return false, err
	}
	
	return result.ExitCode == 0, nil
}

// UploadFile uploads a local file to the remote server
func (c *SSHClient) UploadFile(localPath, remotePath string, mode os.FileMode) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}
	
	return c.WriteFile(remotePath, string(content), mode)
}

// DownloadFile downloads a file from the remote server to local path
func (c *SSHClient) DownloadFile(remotePath, localPath string) error {
	content, err := c.ReadFile(remotePath)
	if err != nil {
		return err
	}
	
	// Create local directory if it doesn't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}
	
	return os.WriteFile(localPath, []byte(content), 0644)
}

// GetSystemInfo retrieves basic system information from the remote server
func (c *SSHClient) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}
	
	// Get OS information
	result, err := c.ExecuteCommand("uname -a")
	if err == nil && result.ExitCode == 0 {
		info.OS = strings.TrimSpace(result.Stdout)
	}
	
	// Get CPU information
	result, err = c.ExecuteCommand("nproc")
	if err == nil && result.ExitCode == 0 {
		info.CPUCores = strings.TrimSpace(result.Stdout)
	}
	
	// Get memory information
	result, err = c.ExecuteCommand("free -h | grep Mem | awk '{print $2}'")
	if err == nil && result.ExitCode == 0 {
		info.Memory = strings.TrimSpace(result.Stdout)
	}
	
	// Get disk space
	result, err = c.ExecuteCommand("df -h / | tail -1 | awk '{print $2}'")
	if err == nil && result.ExitCode == 0 {
		info.DiskSpace = strings.TrimSpace(result.Stdout)
	}
	
	return info, nil
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
}

// SystemInfo represents basic system information
type SystemInfo struct {
	OS        string
	CPUCores  string
	Memory    string
	DiskSpace string
}

// IsSuccess returns true if the command executed successfully
func (r *CommandResult) IsSuccess() bool {
	return r.ExitCode == 0
}

// GetOutput returns the appropriate output (stdout if successful, stderr if failed)
func (r *CommandResult) GetOutput() string {
	if r.IsSuccess() {
		return r.Stdout
	}
	return r.Stderr
}