package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"high-performance-news-website/internal/desktop"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

func main() {
	var (
		port     = flag.Int("port", 8090, "Port to run the desktop app on")
		noBrowser = flag.Bool("no-browser", false, "Don't open browser automatically")
		dev      = flag.Bool("dev", false, "Development mode")
	)
	flag.Parse()

	// Create desktop app
	app, err := desktop.NewApp(&desktop.Config{
		Port:          *port,
		StaticFiles:   staticFiles,
		TemplateFiles: templateFiles,
		DevMode:       *dev,
	})
	if err != nil {
		log.Fatalf("Failed to create desktop app: %v", err)
	}

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: app.Router(),
	}

	// Start server in goroutine
	go func() {
		log.Printf("Desktop deployment app starting on http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Open browser if not disabled
	if !*noBrowser {
		time.Sleep(1 * time.Second) // Wait for server to start
		openBrowser(fmt.Sprintf("http://localhost:%d", *port))
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down desktop app...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Desktop app stopped")
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
		log.Printf("Please open your browser and navigate to: %s", url)
	}
}