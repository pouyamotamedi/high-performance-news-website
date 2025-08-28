package main

import (
	"log"
	"os"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	log.Println("Starting High Performance News Website")
	log.Println("Attempting to connect to database and cache services...")
	log.Println("If services are not available, RSS feeds will use mock data")
	log.Println("")
	log.Println("Available endpoints:")
	log.Println("  - http://localhost:8080/ (Homepage)")
	log.Println("  - http://localhost:8080/rss (RSS Feed)")
	log.Println("  - http://localhost:8080/rss/category/tech")
	log.Println("  - http://localhost:8080/rss/tag/news")
	log.Println("  - http://localhost:8080/rss/googlenews")
	log.Println("  - http://localhost:8080/sitemap-news.xml")
	log.Println("  - http://localhost:8080/sitemap-news-index.xml")
	log.Println("  - http://localhost:8080/health")

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}