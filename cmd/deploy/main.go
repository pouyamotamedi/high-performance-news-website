package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"high-performance-news-website/internal/deployment"
)

func main() {
	var (
		configFile = flag.String("config", "deploy.yaml", "Deployment configuration file")
		action     = flag.String("action", "deploy", "Action to perform: deploy, rollback, health-check, setup, status, list-deployments, validate")
		target     = flag.String("target", "", "Target server (required)")
		version    = flag.String("version", "", "Version to deploy (for rollback)")
		dryRun     = flag.Bool("dry-run", false, "Perform a dry run without making changes")
		limit      = flag.Int("limit", 10, "Limit for list operations")
		format     = flag.String("format", "text", "Output format: text, json")
	)
	flag.Parse()

	config, err := deployment.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	agent := deployment.NewAgent(config)

	// Actions that don't require a target
	switch *action {
	case "validate":
		if err := agent.ValidateConfiguration(); err != nil {
			log.Fatalf("Configuration validation failed: %v", err)
		}
		fmt.Println("Configuration validation passed")
		return
	}

	// All other actions require a target
	if *target == "" {
		fmt.Fprintf(os.Stderr, "Error: target server is required\n")
		flag.Usage()
		os.Exit(1)
	}

	switch *action {
	case "deploy":
		if err := agent.Deploy(*target, *dryRun); err != nil {
			log.Fatalf("Deployment failed: %v", err)
		}
		fmt.Println("Deployment completed successfully")

	case "rollback":
		if *version == "" {
			log.Fatal("Version is required for rollback")
		}
		if err := agent.Rollback(*target, *version, *dryRun); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")

	case "health-check":
		healthy, err := agent.HealthCheck(*target)
		if err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
		if healthy {
			fmt.Println("Server is healthy")
		} else {
			fmt.Println("Server is unhealthy")
			os.Exit(1)
		}

	case "setup":
		if err := agent.SetupServer(*target, *dryRun); err != nil {
			log.Fatalf("Server setup failed: %v", err)
		}
		fmt.Println("Server setup completed successfully")

	case "status":
		status, err := agent.GetServerStatus(*target)
		if err != nil {
			log.Fatalf("Failed to get server status: %v", err)
		}
		printServerStatus(status, *format)

	case "list-deployments":
		deployments, err := agent.ListDeployments(*target, *limit)
		if err != nil {
			log.Fatalf("Failed to list deployments: %v", err)
		}
		printDeployments(deployments, *format)

	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", *action)
		flag.Usage()
		os.Exit(1)
	}
}

// printServerStatus prints server status in the specified format
func printServerStatus(status *deployment.ServerStatus, format string) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
	default:
		fmt.Printf("Server Status for %s\n", status.Target)
		fmt.Printf("Timestamp: %s\n", status.Timestamp.Format(time.RFC3339))
		fmt.Printf("Service Status: %s (Active: %v)\n", status.ServiceStatus, status.ServiceActive)
		fmt.Printf("Application Healthy: %v\n", status.ApplicationHealthy)
		fmt.Printf("Current Version: %s\n", status.CurrentVersion)
		
		if len(status.AvailableVersions) > 0 {
			fmt.Printf("Available Versions: %s\n", strings.Join(status.AvailableVersions, ", "))
		}
		
		if status.SystemInfo != nil {
			fmt.Printf("System Info:\n")
			fmt.Printf("  OS: %s\n", status.SystemInfo.OS)
			fmt.Printf("  CPU Cores: %s\n", status.SystemInfo.CPUCores)
			fmt.Printf("  Memory: %s\n", status.SystemInfo.Memory)
			fmt.Printf("  Disk Space: %s\n", status.SystemInfo.DiskSpace)
		}
		
		if status.ResourceUsage != nil {
			fmt.Printf("Resource Usage:\n")
			fmt.Printf("  CPU: %s\n", status.ResourceUsage.CPUUsage)
			fmt.Printf("  Memory: %s\n", status.ResourceUsage.MemoryUsage)
			fmt.Printf("  Disk: %s\n", status.ResourceUsage.DiskUsage)
			fmt.Printf("  Load Average: %s\n", status.ResourceUsage.LoadAverage)
		}
	}
}

// printDeployments prints deployment history in the specified format
func printDeployments(deployments []*deployment.DeploymentRecord, format string) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(deployments, "", "  ")
		fmt.Println(string(data))
	default:
		if len(deployments) == 0 {
			fmt.Println("No deployments found")
			return
		}
		
		fmt.Printf("%-20s %-25s %-10s %-15s %-10s\n", "Timestamp", "Version", "Duration", "Type", "Status")
		fmt.Println(strings.Repeat("-", 80))
		
		for _, dep := range deployments {
			timestamp := time.Unix(dep.Timestamp, 0).Format("2006-01-02 15:04:05")
			duration := fmt.Sprintf("%.1fs", dep.Duration)
			fmt.Printf("%-20s %-25s %-10s %-15s %-10s\n", 
				timestamp, dep.Version, duration, dep.DeploymentType, dep.Status)
		}
	}
}