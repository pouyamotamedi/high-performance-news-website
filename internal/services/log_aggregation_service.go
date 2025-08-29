package services

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
)

// LogLevel represents log severity levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        uint64                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
	Source    string                 `json:"source"`
	TraceID   string                 `json:"trace_id,omitempty"`
	UserID    *uint64                `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// LogFile represents a monitored log file
type LogFile struct {
	Path      string
	Component string
	Parser    LogParser
	Watcher   *LogWatcher
}

// LogParser interface for parsing different log formats
type LogParser interface {
	Parse(line string) (*LogEntry, error)
}

// LogWatcher monitors a log file for changes
type LogWatcher struct {
	file     *os.File
	scanner  *bufio.Scanner
	stopChan chan struct{}
	mutex    sync.Mutex
}

// LogAggregationService handles log collection and analysis
type LogAggregationService struct {
	db              *sql.DB
	config          *config.MonitoringConfig
	alertingService *AlertingService
	
	// Log files being monitored
	logFiles map[string]*LogFile
	mutex    sync.RWMutex
	
	// Log processing
	logChannel  chan *LogEntry
	stopChannel chan struct{}
	isRunning   bool
	
	// Pattern matching for anomaly detection
	errorPatterns []*regexp.Regexp
	
	// Statistics
	stats LogStatistics
	statsMutex sync.RWMutex
}

// LogStatistics tracks log processing statistics
type LogStatistics struct {
	TotalEntries    int64            `json:"total_entries"`
	EntriesByLevel  map[string]int64 `json:"entries_by_level"`
	EntriesByComponent map[string]int64 `json:"entries_by_component"`
	ErrorsLastHour  int64            `json:"errors_last_hour"`
	LastProcessed   time.Time        `json:"last_processed"`
}

// NewLogAggregationService creates a new LogAggregationService
func NewLogAggregationService(
	db *sql.DB,
	config *config.MonitoringConfig,
	alertingService *AlertingService,
) *LogAggregationService {
	service := &LogAggregationService{
		db:              db,
		config:          config,
		alertingService: alertingService,
		logFiles:        make(map[string]*LogFile),
		logChannel:      make(chan *LogEntry, 10000), // Buffered channel
		stopChannel:     make(chan struct{}),
		stats: LogStatistics{
			EntriesByLevel:     make(map[string]int64),
			EntriesByComponent: make(map[string]int64),
		},
	}
	
	// Initialize error patterns for anomaly detection
	service.initializeErrorPatterns()
	
	return service
}

// StartLogAggregation starts the log aggregation service
func (las *LogAggregationService) StartLogAggregation(ctx context.Context) {
	las.mutex.Lock()
	defer las.mutex.Unlock()
	
	if las.isRunning {
		return
	}
	
	log.Println("Starting log aggregation service...")
	las.isRunning = true
	
	// Start log processing goroutine
	go las.processLogs(ctx)
	
	// Start log file watchers
	go las.startLogWatchers(ctx)
	
	log.Println("Log aggregation service started")
}

// StopLogAggregation stops the log aggregation service
func (las *LogAggregationService) StopLogAggregation() {
	las.mutex.Lock()
	defer las.mutex.Unlock()
	
	if !las.isRunning {
		return
	}
	
	log.Println("Stopping log aggregation service...")
	close(las.stopChannel)
	
	// Stop all log watchers
	for _, logFile := range las.logFiles {
		if logFile.Watcher != nil {
			logFile.Watcher.Stop()
		}
	}
	
	las.isRunning = false
	log.Println("Log aggregation service stopped")
}

// AddLogFile adds a log file to be monitored
func (las *LogAggregationService) AddLogFile(filePath, component string) error {
	las.mutex.Lock()
	defer las.mutex.Unlock()
	
	if _, exists := las.logFiles[filePath]; exists {
		return fmt.Errorf("log file already being monitored: %s", filePath)
	}
	
	// Create appropriate parser based on component
	parser := las.createParser(component)
	
	logFile := &LogFile{
		Path:      filePath,
		Component: component,
		Parser:    parser,
	}
	
	las.logFiles[filePath] = logFile
	
	// Start watching if service is running
	if las.isRunning {
		go las.watchLogFile(logFile)
	}
	
	log.Printf("Added log file for monitoring: %s (%s)", filePath, component)
	return nil
}

// RemoveLogFile removes a log file from monitoring
func (las *LogAggregationService) RemoveLogFile(filePath string) error {
	las.mutex.Lock()
	defer las.mutex.Unlock()
	
	logFile, exists := las.logFiles[filePath]
	if !exists {
		return fmt.Errorf("log file not being monitored: %s", filePath)
	}
	
	// Stop watcher
	if logFile.Watcher != nil {
		logFile.Watcher.Stop()
	}
	
	delete(las.logFiles, filePath)
	log.Printf("Removed log file from monitoring: %s", filePath)
	return nil
}

// processLogs processes incoming log entries
func (las *LogAggregationService) processLogs(ctx context.Context) {
	log.Println("Starting log processing goroutine...")
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Log processing stopped")
			return
		case <-las.stopChannel:
			log.Println("Log processing stopped")
			return
		case logEntry := <-las.logChannel:
			if logEntry != nil {
				las.processLogEntry(logEntry)
			}
		}
	}
}

// processLogEntry processes a single log entry
func (las *LogAggregationService) processLogEntry(entry *LogEntry) {
	// Update statistics
	las.updateStatistics(entry)
	
	// Save to database
	if err := las.saveLogEntry(entry); err != nil {
		log.Printf("Error saving log entry: %v", err)
	}
	
	// Check for anomalies and trigger alerts
	las.checkForAnomalies(entry)
	
	// Pattern matching for specific issues
	las.analyzeLogEntry(entry)
}

// updateStatistics updates log processing statistics
func (las *LogAggregationService) updateStatistics(entry *LogEntry) {
	las.statsMutex.Lock()
	defer las.statsMutex.Unlock()
	
	las.stats.TotalEntries++
	las.stats.EntriesByLevel[string(entry.Level)]++
	las.stats.EntriesByComponent[entry.Component]++
	las.stats.LastProcessed = time.Now()
	
	// Count errors in the last hour
	if entry.Level == LogLevelError || entry.Level == LogLevelFatal {
		if time.Since(entry.Timestamp) <= time.Hour {
			las.stats.ErrorsLastHour++
		}
	}
}

// saveLogEntry saves a log entry to the database
func (las *LogAggregationService) saveLogEntry(entry *LogEntry) error {
	if las.db == nil {
		return nil
	}
	
	contextJSON, _ := json.Marshal(entry.Context)
	
	query := `
		INSERT INTO log_entries 
		(timestamp, level, component, message, context, source, trace_id, user_id, request_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := las.db.Exec(
		query, entry.Timestamp, entry.Level, entry.Component, entry.Message,
		contextJSON, entry.Source, entry.TraceID, entry.UserID, entry.RequestID, entry.CreatedAt,
	)
	
	return err
}

// checkForAnomalies checks for log anomalies and triggers alerts
func (las *LogAggregationService) checkForAnomalies(entry *LogEntry) {
	// Check for high error rates
	if entry.Level == LogLevelError || entry.Level == LogLevelFatal {
		las.statsMutex.RLock()
		errorCount := las.stats.ErrorsLastHour
		las.statsMutex.RUnlock()
		
		// Alert if error rate is too high
		if errorCount > 100 { // Configurable threshold
			alert := &models.Alert{
				Name:         "high_error_rate",
				Description:  fmt.Sprintf("High error rate detected: %d errors in the last hour", errorCount),
				Severity:     models.AlertSeverityWarning,
				Status:       models.AlertStatusActive,
				Component:    "logging",
				Metric:       "error_rate",
				Threshold:    100,
				CurrentValue: float64(errorCount),
				Metadata:     map[string]interface{}{"component": entry.Component},
				TriggeredAt:  time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			
			if las.alertingService != nil {
				las.alertingService.SendAlert(alert)
			}
		}
	}
}

// analyzeLogEntry analyzes log entries for specific patterns
func (las *LogAggregationService) analyzeLogEntry(entry *LogEntry) {
	message := strings.ToLower(entry.Message)
	
	// Check for critical patterns
	criticalPatterns := map[string]string{
		"out of memory":     "system_out_of_memory",
		"disk full":         "disk_full",
		"connection refused": "connection_refused",
		"database error":    "database_error",
		"timeout":           "timeout_error",
		"panic":             "application_panic",
	}
	
	for pattern, alertName := range criticalPatterns {
		if strings.Contains(message, pattern) {
			alert := &models.Alert{
				Name:         alertName,
				Description:  fmt.Sprintf("Critical pattern detected in logs: %s", pattern),
				Severity:     models.AlertSeverityCritical,
				Status:       models.AlertStatusActive,
				Component:    entry.Component,
				Metric:       "log_pattern",
				Threshold:    1,
				CurrentValue: 1,
				Metadata: map[string]interface{}{
					"pattern":    pattern,
					"log_entry":  entry.Message,
					"source":     entry.Source,
					"timestamp":  entry.Timestamp,
				},
				TriggeredAt: time.Now(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			
			if las.alertingService != nil {
				las.alertingService.SendAlert(alert)
			}
			break
		}
	}
}

// startLogWatchers starts watching all configured log files
func (las *LogAggregationService) startLogWatchers(ctx context.Context) {
	las.mutex.RLock()
	defer las.mutex.RUnlock()
	
	for _, logFile := range las.logFiles {
		go las.watchLogFile(logFile)
	}
}

// watchLogFile watches a single log file for changes
func (las *LogAggregationService) watchLogFile(logFile *LogFile) {
	log.Printf("Starting to watch log file: %s", logFile.Path)
	
	// Check if file exists
	if _, err := os.Stat(logFile.Path); os.IsNotExist(err) {
		log.Printf("Log file does not exist: %s", logFile.Path)
		return
	}
	
	file, err := os.Open(logFile.Path)
	if err != nil {
		log.Printf("Error opening log file %s: %v", logFile.Path, err)
		return
	}
	defer file.Close()
	
	// Seek to end of file to only read new entries
	file.Seek(0, 2)
	
	scanner := bufio.NewScanner(file)
	
	watcher := &LogWatcher{
		file:     file,
		scanner:  scanner,
		stopChan: make(chan struct{}),
	}
	
	logFile.Watcher = watcher
	
	// Watch for new lines
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-las.stopChannel:
			return
		case <-watcher.stopChan:
			return
		case <-ticker.C:
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					entry, err := logFile.Parser.Parse(line)
					if err != nil {
						log.Printf("Error parsing log line from %s: %v", logFile.Path, err)
						continue
					}
					
					entry.Source = logFile.Path
					entry.Component = logFile.Component
					entry.CreatedAt = time.Now()
					
					// Send to processing channel
					select {
					case las.logChannel <- entry:
					default:
						log.Printf("Log channel full, dropping log entry")
					}
				}
			}
		}
	}
}

// Stop stops a log watcher
func (lw *LogWatcher) Stop() {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()
	
	close(lw.stopChan)
}

// createParser creates an appropriate log parser for the component
func (las *LogAggregationService) createParser(component string) LogParser {
	switch component {
	case "nginx":
		return &NginxLogParser{}
	case "postgresql":
		return &PostgreSQLLogParser{}
	case "application":
		return &ApplicationLogParser{}
	default:
		return &GenericLogParser{}
	}
}

// initializeErrorPatterns initializes regex patterns for error detection
func (las *LogAggregationService) initializeErrorPatterns() {
	patterns := []string{
		`(?i)error`,
		`(?i)exception`,
		`(?i)panic`,
		`(?i)fatal`,
		`(?i)critical`,
		`(?i)timeout`,
		`(?i)connection.*refused`,
		`(?i)out of memory`,
		`(?i)disk.*full`,
	}
	
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			las.errorPatterns = append(las.errorPatterns, regex)
		}
	}
}

// GetRecentLogs returns recent log entries
func (las *LogAggregationService) GetRecentLogs(component string, level LogLevel, limit int, since time.Time) ([]LogEntry, error) {
	if las.db == nil {
		return []LogEntry{}, nil
	}
	
	query := `
		SELECT id, timestamp, level, component, message, context, source, 
		       trace_id, user_id, request_id, created_at
		FROM log_entries
		WHERE created_at >= $1
	`
	args := []interface{}{since}
	argCount := 1
	
	if component != "" {
		argCount++
		query += fmt.Sprintf(" AND component = $%d", argCount)
		args = append(args, component)
	}
	
	if level != "" {
		argCount++
		query += fmt.Sprintf(" AND level = $%d", argCount)
		args = append(args, level)
	}
	
	query += " ORDER BY created_at DESC"
	
	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}
	
	rows, err := las.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []LogEntry
	for rows.Next() {
		var entry LogEntry
		var contextJSON []byte
		
		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Level, &entry.Component,
			&entry.Message, &contextJSON, &entry.Source, &entry.TraceID,
			&entry.UserID, &entry.RequestID, &entry.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		// Parse context JSON
		if len(contextJSON) > 0 {
			json.Unmarshal(contextJSON, &entry.Context)
		} else {
			entry.Context = make(map[string]interface{})
		}
		
		logs = append(logs, entry)
	}
	
	return logs, nil
}

// GetLogStatistics returns log processing statistics
func (las *LogAggregationService) GetLogStatistics(since time.Time) (map[string]interface{}, error) {
	las.statsMutex.RLock()
	defer las.statsMutex.RUnlock()
	
	stats := map[string]interface{}{
		"total_entries":       las.stats.TotalEntries,
		"entries_by_level":    las.stats.EntriesByLevel,
		"entries_by_component": las.stats.EntriesByComponent,
		"errors_last_hour":    las.stats.ErrorsLastHour,
		"last_processed":      las.stats.LastProcessed,
	}
	
	// Get database statistics if available
	if las.db != nil {
		dbStats, err := las.getDatabaseLogStatistics(since)
		if err == nil {
			for k, v := range dbStats {
				stats[k] = v
			}
		}
	}
	
	return stats, nil
}

// getDatabaseLogStatistics gets log statistics from database
func (las *LogAggregationService) getDatabaseLogStatistics(since time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count total entries since timestamp
	var totalCount int64
	err := las.db.QueryRow("SELECT COUNT(*) FROM log_entries WHERE created_at >= $1", since).Scan(&totalCount)
	if err != nil {
		return stats, err
	}
	stats["db_total_since"] = totalCount
	
	// Count by level
	levelQuery := `
		SELECT level, COUNT(*) 
		FROM log_entries 
		WHERE created_at >= $1 
		GROUP BY level
	`
	rows, err := las.db.Query(levelQuery, since)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	
	levelCounts := make(map[string]int64)
	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err == nil {
			levelCounts[level] = count
		}
	}
	stats["db_entries_by_level"] = levelCounts
	
	return stats, nil
}

// CleanupOldLogs removes old log entries from database
func (las *LogAggregationService) CleanupOldLogs(retentionDays int) error {
	if las.db == nil {
		return nil
	}
	
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	query := "DELETE FROM log_entries WHERE created_at < $1"
	result, err := las.db.Exec(query, cutoffDate)
	if err != nil {
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old log entries (older than %d days)", rowsAffected, retentionDays)
	
	return nil
}

// Log Parser Implementations

// GenericLogParser parses generic log format
type GenericLogParser struct{}

func (p *GenericLogParser) Parse(line string) (*LogEntry, error) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   line,
		Context:   make(map[string]interface{}),
	}
	
	// Try to extract log level from message
	lowerLine := strings.ToLower(line)
	if strings.Contains(lowerLine, "error") {
		entry.Level = LogLevelError
	} else if strings.Contains(lowerLine, "warn") {
		entry.Level = LogLevelWarning
	} else if strings.Contains(lowerLine, "debug") {
		entry.Level = LogLevelDebug
	} else if strings.Contains(lowerLine, "fatal") {
		entry.Level = LogLevelFatal
	}
	
	return entry, nil
}

// NginxLogParser parses Nginx access and error logs
type NginxLogParser struct{}

func (p *NginxLogParser) Parse(line string) (*LogEntry, error) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   line,
		Context:   make(map[string]interface{}),
	}
	
	// Parse Nginx error log format
	if strings.Contains(line, "[error]") {
		entry.Level = LogLevelError
	} else if strings.Contains(line, "[warn]") {
		entry.Level = LogLevelWarning
	}
	
	return entry, nil
}

// PostgreSQLLogParser parses PostgreSQL logs
type PostgreSQLLogParser struct{}

func (p *PostgreSQLLogParser) Parse(line string) (*LogEntry, error) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   line,
		Context:   make(map[string]interface{}),
	}
	
	// Parse PostgreSQL log levels
	if strings.Contains(line, "ERROR:") {
		entry.Level = LogLevelError
	} else if strings.Contains(line, "WARNING:") {
		entry.Level = LogLevelWarning
	} else if strings.Contains(line, "FATAL:") {
		entry.Level = LogLevelFatal
	}
	
	return entry, nil
}

// ApplicationLogParser parses application logs (JSON format)
type ApplicationLogParser struct{}

func (p *ApplicationLogParser) Parse(line string) (*LogEntry, error) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   line,
		Context:   make(map[string]interface{}),
	}
	
	// Try to parse as JSON
	var jsonLog map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
		// Extract fields from JSON log
		if timestamp, ok := jsonLog["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				entry.Timestamp = t
			}
		}
		
		if level, ok := jsonLog["level"].(string); ok {
			entry.Level = LogLevel(strings.ToLower(level))
		}
		
		if message, ok := jsonLog["message"].(string); ok {
			entry.Message = message
		}
		
		if traceID, ok := jsonLog["trace_id"].(string); ok {
			entry.TraceID = traceID
		}
		
		if requestID, ok := jsonLog["request_id"].(string); ok {
			entry.RequestID = requestID
		}
		
		// Store additional context
		for k, v := range jsonLog {
			if k != "timestamp" && k != "level" && k != "message" && k != "trace_id" && k != "request_id" {
				entry.Context[k] = v
			}
		}
	}
	
	return entry, nil
}