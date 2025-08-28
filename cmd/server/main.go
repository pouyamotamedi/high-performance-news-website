package main

import (
	"log"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/server"
)

func main() {
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