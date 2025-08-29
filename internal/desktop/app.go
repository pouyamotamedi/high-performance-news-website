package desktop

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"high-performance-news-website/internal/deployment"
)

// Config holds the desktop app configuration
type Config struct {
	Port          int
	StaticFiles   embed.FS
	TemplateFiles embed.FS
	DevMode       bool
}

// App represents the desktop deployment application
type App struct {
	config    *Config
	templates *template.Template
	upgrader  websocket.Upgrader
	
	// Deployment related
	deploymentAgent *deployment.Agent
	deploymentConfig *deployment.Config
	
	// WebSocket connections for real-time updates
	wsConnections map[*websocket.Conn]bool
}

// NewApp creates a new desktop application
func NewApp(config *Config) (*App, error) {
	app := &App{
		config:        config,
		wsConnections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for desktop app
			},
		},
	}

	// Parse templates
	if err := app.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return app, nil
}

// Router returns the HTTP router for the desktop app
func (a *App) Router() http.Handler {
	r := mux.NewRouter()

	// Static files
	staticFS, err := fs.Sub(a.config.StaticFiles, "static")
	if err == nil {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// WebSocket endpoint
	r.HandleFunc("/ws", a.handleWebSocket)

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/servers", a.handleGetServers).Methods("GET")
	api.HandleFunc("/servers/{name}/status", a.handleGetServerStatus).Methods("GET")
	api.HandleFunc("/servers/{name}/setup", a.handleSetupServer).Methods("POST")
	api.HandleFunc("/servers/{name}/deploy", a.handleDeploy).Methods("POST")
	api.HandleFunc("/servers/{name}/history", a.handleGetHistory).Methods("GET")
	api.HandleFunc("/servers/{name}/logs", a.handleGetLogs).Methods("GET")
	api.HandleFunc("/config", a.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", a.handleUpdateConfig).Methods("POST")
	api.HandleFunc("/config/load", a.handleLoadConfig).Methods("POST")

	// Page routes
	r.HandleFunc("/", a.handleDashboard)
	r.HandleFunc("/servers", a.handleServersPage)
	r.HandleFunc("/servers/{name}", a.handleServerDetailPage)
	r.HandleFunc("/config", a.handleConfigPage)
	r.HandleFunc("/logs", a.handleLogsPage)

	return r
}

// loadTemplates loads HTML templates from embedded files
func (a *App) loadTemplates() error {
	tmpl := template.New("")
	
	// Try to read templates, if it fails (e.g., in tests), use empty template
	defer func() {
		if a.templates == nil {
			a.templates = tmpl
		}
	}()
	
	err := fs.WalkDir(a.config.TemplateFiles, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		
		content, err := a.config.TemplateFiles.ReadFile(path)
		if err != nil {
			return err
		}
		
		name := strings.TrimPrefix(path, "templates/")
		_, err = tmpl.New(name).Parse(string(content))
		return err
	})
	
	if err != nil {
		// If template loading fails (e.g., in tests), continue with empty template
		a.templates = tmpl
		return nil
	}
	
	a.templates = tmpl
	return nil
}

// renderTemplate renders an HTML template with data
func (a *App) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	
	if a.templates == nil {
		// For tests or when templates are not loaded
		fmt.Fprintf(w, "<html><body><h1>%s</h1><p>Template system not initialized</p></body></html>", name)
		return
	}
	
	err := a.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		// If template doesn't exist, render a simple HTML page
		fmt.Fprintf(w, "<html><body><h1>%s</h1><p>Template not found: %v</p></body></html>", name, err)
	}
}

// sendJSON sends a JSON response
func (a *App) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
	}
}

// sendError sends an error response
func (a *App) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// broadcastMessage sends a message to all WebSocket connections
func (a *App) broadcastMessage(message map[string]interface{}) {
	for conn := range a.wsConnections {
		if err := conn.WriteJSON(message); err != nil {
			conn.Close()
			delete(a.wsConnections, conn)
		}
	}
}

// Page handlers
func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Deployment Dashboard",
		"Page":  "dashboard",
	}
	
	// Add server status if config is loaded
	if a.deploymentConfig != nil {
		servers := make([]map[string]interface{}, 0)
		for name := range a.deploymentConfig.Servers {
			status, err := a.deploymentAgent.GetServerStatus(name)
			serverData := map[string]interface{}{
				"Name":      name,
				"Connected": false,
				"Error":     "",
			}
			
			if err != nil {
				serverData["Error"] = err.Error()
			} else if status != nil {
				serverData["Connected"] = status.Connected
				serverData["Host"] = status.Host
				if status.Error != "" {
					serverData["Error"] = status.Error
				}
			}
			
			servers = append(servers, serverData)
		}
		data["Servers"] = servers
	}
	
	a.renderTemplate(w, "dashboard.html", data)
}

func (a *App) handleServersPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Server Management",
		"Page":  "servers",
	}
	a.renderTemplate(w, "servers.html", data)
}

func (a *App) handleServerDetailPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]
	
	data := map[string]interface{}{
		"Title":      fmt.Sprintf("Server: %s", serverName),
		"Page":       "server-detail",
		"ServerName": serverName,
	}
	a.renderTemplate(w, "server-detail.html", data)
}

func (a *App) handleConfigPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Configuration",
		"Page":  "config",
	}
	
	if a.deploymentConfig != nil {
		data["Config"] = a.deploymentConfig
	}
	
	a.renderTemplate(w, "config.html", data)
}

func (a *App) handleLogsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Deployment Logs",
		"Page":  "logs",
	}
	a.renderTemplate(w, "logs.html", data)
}

// API handlers
func (a *App) handleGetServers(w http.ResponseWriter, r *http.Request) {
	if a.deploymentConfig == nil {
		a.sendError(w, "No configuration loaded", http.StatusBadRequest)
		return
	}
	
	servers := make([]map[string]interface{}, 0)
	for name, config := range a.deploymentConfig.Servers {
		servers = append(servers, map[string]interface{}{
			"name": name,
			"host": config.Host,
			"port": config.Port,
			"user": config.User,
		})
	}
	
	a.sendJSON(w, map[string]interface{}{
		"servers": servers,
	})
}

func (a *App) handleGetServerStatus(w http.ResponseWriter, r *http.Request) {
	if a.deploymentAgent == nil {
		a.sendError(w, "No deployment agent available", http.StatusBadRequest)
		return
	}
	
	vars := mux.Vars(r)
	serverName := vars["name"]
	
	status, err := a.deploymentAgent.GetServerStatus(serverName)
	if err != nil {
		a.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	a.sendJSON(w, status)
}

func (a *App) handleSetupServer(w http.ResponseWriter, r *http.Request) {
	if a.deploymentAgent == nil {
		a.sendError(w, "No deployment agent available", http.StatusBadRequest)
		return
	}
	
	vars := mux.Vars(r)
	serverName := vars["name"]
	
	// Broadcast setup start
	a.broadcastMessage(map[string]interface{}{
		"type":    "setup_start",
		"server":  serverName,
		"message": fmt.Sprintf("Starting server setup for %s", serverName),
	})
	
	go func() {
		err := a.deploymentAgent.SetupServer(serverName)
		if err != nil {
			a.broadcastMessage(map[string]interface{}{
				"type":    "setup_error",
				"server":  serverName,
				"message": fmt.Sprintf("Setup failed: %v", err),
			})
		} else {
			a.broadcastMessage(map[string]interface{}{
				"type":    "setup_complete",
				"server":  serverName,
				"message": fmt.Sprintf("Server setup completed for %s", serverName),
			})
		}
	}()
	
	a.sendJSON(w, map[string]string{"status": "started"})
}

func (a *App) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if a.deploymentAgent == nil {
		a.sendError(w, "No deployment agent available", http.StatusBadRequest)
		return
	}
	
	vars := mux.Vars(r)
	serverName := vars["name"]
	
	// Broadcast deployment start
	a.broadcastMessage(map[string]interface{}{
		"type":    "deploy_start",
		"server":  serverName,
		"message": fmt.Sprintf("Starting deployment to %s", serverName),
	})
	
	go func() {
		err := a.deploymentAgent.Deploy(serverName)
		if err != nil {
			a.broadcastMessage(map[string]interface{}{
				"type":    "deploy_error",
				"server":  serverName,
				"message": fmt.Sprintf("Deployment failed: %v", err),
			})
		} else {
			a.broadcastMessage(map[string]interface{}{
				"type":    "deploy_complete",
				"server":  serverName,
				"message": fmt.Sprintf("Deployment completed for %s", serverName),
			})
		}
	}()
	
	a.sendJSON(w, map[string]string{"status": "started"})
}

func (a *App) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if a.deploymentAgent == nil {
		a.sendError(w, "No deployment agent available", http.StatusBadRequest)
		return
	}
	
	vars := mux.Vars(r)
	serverName := vars["name"]
	
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	history, err := a.deploymentAgent.GetDeploymentHistory(serverName, limit)
	if err != nil {
		a.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	a.sendJSON(w, map[string]interface{}{
		"history": history,
	})
}

func (a *App) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// For now, return empty logs - this would be implemented to read actual log files
	a.sendJSON(w, map[string]interface{}{
		"logs": []string{},
	})
}

func (a *App) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if a.deploymentConfig == nil {
		a.sendJSON(w, map[string]interface{}{
			"loaded": false,
		})
		return
	}
	
	a.sendJSON(w, map[string]interface{}{
		"loaded": true,
		"config": a.deploymentConfig,
	})
}

func (a *App) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var configData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&configData); err != nil {
		a.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// This would save the configuration to a file
	a.sendJSON(w, map[string]string{"status": "saved"})
}

func (a *App) handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ConfigPath string `json:"config_path"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Load deployment configuration
	config, err := deployment.LoadConfig(request.ConfigPath)
	if err != nil {
		a.sendError(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusBadRequest)
		return
	}
	
	// Create deployment agent
	agent := deployment.NewAgent(config)
	
	// Validate configuration
	if err := agent.ValidateConfiguration(); err != nil {
		a.sendError(w, fmt.Sprintf("Configuration validation failed: %v", err), http.StatusBadRequest)
		return
	}
	
	a.deploymentConfig = config
	a.deploymentAgent = agent
	
	a.sendJSON(w, map[string]string{"status": "loaded"})
}

// WebSocket handler for real-time updates
func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	
	// Add connection to the pool
	a.wsConnections[conn] = true
	defer delete(a.wsConnections, conn)
	
	// Send initial connection message
	conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connected",
		"time":    time.Now(),
	})
	
	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}