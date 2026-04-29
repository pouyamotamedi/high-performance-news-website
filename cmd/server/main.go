package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/server"
)

var (
	version = "1.3.0" // Application version
	buildTime = "unknown" // Build timestamp (can be set during build)
	gitCommit = "unknown" // Git commit hash (can be set during build)
)

func main() {
	// Define command line flags
	var showVersion = flag.Bool("version", false, "Show version information and exit")
	var showHelp = flag.Bool("help", false, "Show help information and exit")
	flag.Parse()

	// Handle help flag
	if *showHelp {
		fmt.Printf("News Server v%s - High Performance News Website\n\n", version)
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n\n", os.Args[0])
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  See .env.example for configuration options")
		os.Exit(0)
	}

	// Handle version flag
	if *showVersion {
		fmt.Printf("News Server v%s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}