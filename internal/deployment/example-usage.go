package deployment

import (
	"fmt"
	"log"
	"os"
)

// ExampleUsage demonstrates how to use the deployment agent
func ExampleUsage() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run example-usage.go <config-file> <server-name> [command]")
		fmt.Println("Commands: setup, deploy, status, history")
		os.Exit(1)
	}

	configFile := os.Args[1]
	serverName := os.Args[2]
	command := "deploy"
	if len(os.Args) > 3 {
		command = os.Args[3]
	}

	// Load configuration
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create deployment agent
	agent := NewAgent(config)

	// Validate configuration
	if err := agent.ValidateConfiguration(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	switch command {
	case "setup":
		fmt.Printf("Setting up server: %s\n", serverName)
		if err := agent.SetupServer(serverName); err != nil {
			log.Fatalf("Server setup failed: %v", err)
		}
		fmt.Println("Server setup completed successfully!")

	case "deploy":
		fmt.Printf("Deploying to server: %s\n", serverName)
		if err := agent.Deploy(serverName); err != nil {
			log.Fatalf("Deployment failed: %v", err)
		}
		fmt.Println("Deployment completed successfully!")

	case "status":
		fmt.Printf("Getting status for server: %s\n", serverName)
		status, err := agent.GetServerStatus(serverName)
		if err != nil {
			log.Fatalf("Failed to get server status: %v", err)
		}
		
		fmt.Printf("Server: %s\n", status.Name)
		fmt.Printf("Host: %s\n", status.Host)
		fmt.Printf("Connected: %v\n", status.Connected)
		if status.SystemInfo != nil {
			fmt.Printf("OS: %s\n", status.SystemInfo.OS)
			fmt.Printf("CPU Cores: %s\n", status.SystemInfo.CPUCores)
			fmt.Printf("Memory: %s\n", status.SystemInfo.Memory)
			fmt.Printf("Disk Space: %s\n", status.SystemInfo.DiskSpace)
		}
		if status.ResourceUsage != nil {
			fmt.Printf("CPU Usage: %s\n", status.ResourceUsage.CPUUsage)
			fmt.Printf("Memory Usage: %s\n", status.ResourceUsage.MemoryUsage)
			fmt.Printf("Disk Usage: %s\n", status.ResourceUsage.DiskUsage)
			fmt.Printf("Load Average: %s\n", status.ResourceUsage.LoadAverage)
		}
		if status.Error != "" {
			fmt.Printf("Error: %s\n", status.Error)
		}

	case "history":
		fmt.Printf("Getting deployment history for server: %s\n", serverName)
		history, err := agent.GetDeploymentHistory(serverName, 10)
		if err != nil {
			log.Fatalf("Failed to get deployment history: %v", err)
		}
		
		if len(history) == 0 {
			fmt.Println("No deployment history found")
		} else {
			fmt.Printf("Found %d deployment records:\n", len(history))
			for i, record := range history {
				fmt.Printf("%d. App: %v, Version: %v, Time: %v\n", 
					i+1, record["app_name"], record["new_version"], record["timestamp"])
			}
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: setup, deploy, status, history")
		os.Exit(1)
	}
}