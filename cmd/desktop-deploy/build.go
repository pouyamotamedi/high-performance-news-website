package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Build configuration for cross-platform desktop application
type BuildConfig struct {
	AppName     string
	Version     string
	OutputDir   string
	Platforms   []Platform
	EmbedAssets bool
}

type Platform struct {
	OS   string
	Arch string
}

var defaultPlatforms = []Platform{
	{"windows", "amd64"},
	{"linux", "amd64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"}, // Apple Silicon
}

func buildMain() {
	if len(os.Args) > 1 && os.Args[1] == "build" {
		buildApp()
		return
	}

	// Default: run the application
	runApp()
}

func buildApp() {
	config := BuildConfig{
		AppName:     "news-deploy",
		Version:     "1.0.0",
		OutputDir:   "dist",
		Platforms:   defaultPlatforms,
		EmbedAssets: true,
	}

	log.Printf("Building %s v%s for %d platforms", config.AppName, config.Version, len(config.Platforms))

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Build for each platform
	for _, platform := range config.Platforms {
		if err := buildForPlatform(config, platform); err != nil {
			log.Printf("Failed to build for %s/%s: %v", platform.OS, platform.Arch, err)
			continue
		}
		log.Printf("Successfully built for %s/%s", platform.OS, platform.Arch)
	}

	log.Println("Build completed!")
}

func buildForPlatform(config BuildConfig, platform Platform) error {
	// Set environment variables for cross-compilation
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", platform.OS))
	env = append(env, fmt.Sprintf("GOARCH=%s", platform.Arch))
	env = append(env, "CGO_ENABLED=0") // Disable CGO for easier cross-compilation

	// Determine output filename
	outputName := config.AppName
	if platform.OS == "windows" {
		outputName += ".exe"
	}

	outputPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s-%s-%s", config.AppName, platform.OS, platform.Arch), outputName)

	// Create platform-specific directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create platform directory: %w", err)
	}

	// Build command
	args := []string{"build"}
	
	// Add build flags
	args = append(args, "-ldflags", fmt.Sprintf("-s -w -X main.version=%s", config.Version))
	
	// Add tags for embedded assets
	if config.EmbedAssets {
		args = append(args, "-tags", "embed")
	}
	
	// Output path
	args = append(args, "-o", outputPath)
	
	// Source path
	args = append(args, ".")

	// Execute build command
	cmd := exec.Command("go", args...)
	cmd.Env = env
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w\nOutput: %s", err, string(output))
	}

	// Create additional files for the platform
	if err := createPlatformFiles(config, platform, filepath.Dir(outputPath)); err != nil {
		log.Printf("Warning: Failed to create platform files: %v", err)
	}

	return nil
}

func createPlatformFiles(config BuildConfig, platform Platform, outputDir string) error {
	// Create README
	readmeContent := fmt.Sprintf("# %s v%s\n\nCross-platform desktop deployment application for high-performance news website.\n\n## Usage\n\n1. Run the application:\n   - Windows: Double-click %s.exe\n   - Linux/macOS: ./news-deploy\n\n2. The application will start a web server on port 8090\n3. Your default browser will open automatically\n4. Load a deployment configuration file to get started\n\n## Features\n\n- Server connection management\n- Automated deployment with blue-green strategy\n- Real-time deployment monitoring\n- Backup management\n- System monitoring and resource usage\n- Deployment history tracking\n- Log viewing and filtering\n\n## Support\n\nFor issues and documentation, visit: https://github.com/your-org/news-website\n", config.AppName, config.Version, config.AppName)

	readmePath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	// Create example configuration
	exampleConfig := "# Example deployment configuration\nservers:\n  production:\n    host: \"your-server.com\"\n    port: 22\n    user: \"deploy\"\n    key_file: \"/path/to/ssh/private/key\"\n\napp:\n  name: \"news-website\"\n  binary: \"./bin/news-website\"\n  port: 8080\n  health_path: \"/health\"\n\ndeploy:\n  strategy: \"blue-green\"\n  health_timeout: 2m\n"

	configPath := filepath.Join(outputDir, "example-config.yaml")
	if err := os.WriteFile(configPath, []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to create example config: %w", err)
	}

	// Create platform-specific startup scripts
	switch platform.OS {
	case "windows":
		batchContent := fmt.Sprintf("@echo off\necho Starting %s...\n%s.exe --port 8090\npause\n", config.AppName, config.AppName)
		
		batchPath := filepath.Join(outputDir, "start.bat")
		if err := os.WriteFile(batchPath, []byte(batchContent), 0644); err != nil {
			return fmt.Errorf("failed to create batch file: %w", err)
		}

	case "linux", "darwin":
		shellContent := fmt.Sprintf("#!/bin/bash\necho \"Starting %s...\"\n./%s --port 8090\n", config.AppName, config.AppName)
		
		shellPath := filepath.Join(outputDir, "start.sh")
		if err := os.WriteFile(shellPath, []byte(shellContent), 0755); err != nil {
			return fmt.Errorf("failed to create shell script: %w", err)
		}
	}

	return nil
}

func runApp() {
	// This would normally run the main application
	// For now, just show a message
	fmt.Printf("News Website Desktop Deployment Tool\n")
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("\nTo build for all platforms, run: go run build.go build\n")
}