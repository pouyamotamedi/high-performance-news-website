package services

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// LogEntry represents a structured log entry
type LogEntry struct {
	ID        uint64                 `json:"id" db:"id"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
	Level     LogLevel               `json:"level" db:"level"`
	Component string                 `json:"component" db:"component"`
	Message   string                 `json:"message" db:"message"`
	Context   map[string]interface{} `json:"context" db:"context"`
	Source    string                 `json:"source" db:"source"`
	TraceID   string                 `json:"trace_id" db:"trace_id"`
	UserID    *uint64                `json:"user_id" db:"user_id"`
	RequestID string                 `json:"request_id" db:"request_id"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// LogPattern represents a log pattern for parsing
type LogPattern struct {
	Name    string         `json:"name"`
	Pattern *regexp.Regexp `json:"pattern"`
	Fields  []string       `json:"fields"`
}

// LogAggregationService handles log collection, parsing, and analysis
type LogAggregationService struct {
	db           *sql.DB
	config       *config.MonitoringConfig
	logPatterns  []LogPattern
	logBuffer    []LogEntry
	bufferMutex  sync.RWMutex
	alertService *AlertingService
	
	// Log file watchers
	watchers map[string]*LogWatcher
	watcherMutex sync.RWMutex
}

// LogWatcher watches a log file for changes
type LogWatcher struct {
	FilePath    string
	LastOffset  int64
	LastModTime time.Time
	Component   string
}

// NewLogAggregationService creates a new LogAggregationService
func NewLogAggregationService(db *sql.DB, config *config.MonitoringConfig, alertService *AlertingService) *LogAggregationService {
	las := &LogAggregationService{
		db:           db,
		config:       config,
		logBuffer:    make([]LogEntry, 0),
		alertService: alertService,
		watchers:     make(map[string]*LogWatcher),
	}
	
	// Initialize log patterns
	las.initLogPatterns()
	
	return las
}

// initLogPatterns initializes common log patterns for parsing
func (las *LogAggregationService) initLogPatterns() {
	las.logPatterns = []LogPattern{
		{
			Name:    "nginx_access",
			Pattern: regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "([^"]*)" "([^"]*)"$`),
			Fields:  []string{"ip", "timestamp", "method", "path", "protocol", "status", "size", "referer", "user_agent"},
		},
		{
			Name:    "nginx_error",
			Pattern: regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\] (\d+)#(\d+): (.+)$`),
			Fields:  []string{"timestamp", "level", "pid", "tid", "message"},
		},
		{
			Name:    "go_structured",
			Pattern: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z) (\w+) (.+)$`),
			Fields:  []string{"timestamp", "level", "message"},
		},
		{
			Name:    "postgresql",
			Pattern: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d+) (\w+) \[(\d+)\] (.+)$`),
			Fields:  []string{"timestamp", "level", "pid", "message"},
		},
	}
}

// StartLogAggregation starts the log aggregation system
func (las *LogAggregationService) StartLogAggregation(ctx context.Context) {
	log.Println("Starting log aggregation system...")
	
	// Start log file watchers
	go las.startLogWatchers(ctx)
	
	// Start log buffer flusher
	go las.startBufferFlusher(ctx)
	
	// Start log analysis
	go las.startLogAnalysis(ctx)
	
	log.Println("Log aggregation system started")
}

// AddLogFile adds a log file to be monitored
func (las *LogAggregationService) AddLogFile(filePath, component string) error {
	las.watcherMutex.Lock()
	defer las.watcherMutex.Unlock()
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", filePath)
	}
	
	// Create watcher
	watcher := &LogWatcher{
		FilePath:    filePath,
		LastOffset:  0,
		LastModTime: time.Time{},
		Component:   component,
	}
	
	las.watchers[filePath] = watcher
	log.Printf("Added log file watcher: %s (component: %s)", filePath, component)
	
	return nil
}

// startLogWatchers starts monitoring log files
func (las *LogAggregationService) startLogWatchers(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			las.checkLogFiles()
		}
	}
}

// checkLogFiles checks all monitored log files for changes
func (las *LogAggregationService) checkLogFiles() {
	las.watcherMutex.RLock()
	watchers := make(map[string]*LogWatcher)
	for k, v := range las.watchers {
		watchers[k] = v
	}
	las.watcherMutex.RUnlock()
	
	for filePath, watcher := range watchers {
		las.checkLogFile(watcher)
	}
}

// checkLogFile checks a single log file for changes
func (las *LogAggregationService) checkLogFile(watcher *LogWatcher) {
	fileInfo, err := os.Stat(watcher.FilePath)
	if err != nil {
		log.Printf("Error checking log file %s: %v", watcher.FilePath, err)
		return
	}
	
	// Check if file has been modified
	if fileInfo.ModTime().After(watcher.LastModTime) {
		las.readLogFile(watcher, fileInfo)
		watcher.LastModTime = fileInfo.ModTime()
	}
}

// readLogFile reads new content from a log file
func (las *LogAggregationService) readLogFile(watcher *LogWatcher, fileInfo os.FileInfo) {
	file, err := os.Open(watcher.FilePath)
	if err != nil {
		log.Printf("Error opening log file %s: %v", watcher.FilePath, err)
		return
	}
	defer file.Close()
	
	// Seek to last read position
	if watcher.LastOffset > fileInfo.Size() {
		// File was truncated, start from beginning
		watcher.LastOffset = 0
	}
	
	_, err = file.Seek(watcher.LastOffset, 0)
	if err != nil {
		log.Printf("Error seeking log file %s: %v", watcher.FilePath, err)
		return
	}
	
	scanner := bufio.NewScanner(file)
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		logEntry := las.parseLogLine(line, watcher.Component, watcher.FilePath)
		if logEntry != nil {
			las.addLogEntry(*logEntry)
			lineCount++
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading log file %s: %v", watcher.FilePath, err)
		return
	}
	
	// Update offset
	currentPos, err := file.Seek(0, 1) // Get current position
	if err == nil {
		watcher.LastOffset = currentPos
	}
	
	if lineCount > 0 {
		log.Printf("Read %d new log entries from %s", lineCount, watcher.FilePath)
	}
}

// parseLogLine parses a log line using configured patterns
func (las *LogAggregationService) parseLogLine(line, component, source string) *LogEntry {
	// Try to parse with known patterns
	for _, pattern := range las.logPatterns {
		if matches := pattern.Pattern.FindStringSubmatch(line); matches != nil {
			return las.createLogEntryFromPattern(matches, pattern, component, source)
		}
	}
	
	// If no pattern matches, create a generic log entry
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     las.inferLogLevel(line),
		Component: component,
		Message:   line,
		Context:   make(map[string]interface{}),
		Source:    source,
		CreatedAt: time.Now(),
	}
}

// createLogEntryFromPattern creates a log entry from pattern matches
func (las *LogAggregationService) createLogEntryFromPattern(matches []string, pattern LogPattern, component, source string) *LogEntry {
	entry := &LogEntry{
		Component: component,
		Source:    source,
		Context:   make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
	
	// Map matches to fields
	for i, field := range pattern.Fields {
		if i+1 < len(matches) {
			value := matches[i+1]
			
			switch field {
			case "timestamp":
				if timestamp, err := las.parseTimestamp(value); err == nil {
					entry.Timestamp = timestamp
				} else {
					entry.Timestamp = time.Now()
				}
			case "level":
				entry.Level = las.normalizeLogLevel(value)
			case "message":
				entry.Message = value
			default:
				entry.Context[field] = value
			}
		}
	}
	
	// Set defaults if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.Level == "" {
		entry.Level = las.inferLogLevel(entry.Message)
	}
	if entry.Message == "" {
		entry.Message = matches[0] // Use full match as message
	}
	
	return entry
}

// parseTimestamp parses various timestamp formats
func (las *LogAggregationService) parseTimestamp(timestampStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"02/Jan/2006:15:04:05 -0700",
		"2006/01/02 15:04:05",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// normalizeLogLevel normalizes log level strings
func (las *LogAggregationService) normalizeLogLevel(level string) LogLevel {
	level = strings.ToLower(strings.TrimSpace(level))
	
	switch level {
	case "debug", "dbg":
		return LogLevelDebug
	case "info", "information":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarning
	case "error", "err":
		return LogLevelError
	case "fatal", "critical", "crit":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// inferLogLevel infers log level from message content
func (las *LogAggregationService) inferLogLevel(message string) LogLevel {
	message = strings.ToLower(message)
	
	if strings.Contains(message, "error") || strings.Contains(message, "failed") || strings.Contains(message, "exception") {
		return LogLevelError
	}
	if strings.Contains(message, "warn") || strings.Contains(message, "warning") {
		return LogLevelWarning
	}
	if strings.Contains(message, "debug") {
		return LogLevelDebug
	}
	if strings.Contains(message, "fatal") || strings.Contains(message, "critical") {
		return LogLevelFatal
	}
	
	return LogLevelInfo
}

// addLogEntry adds a log entry to the buffer
func (las *LogAggregationService) addLogEntry(entry LogEntry) {
	las.bufferMutex.Lock()
	defer las.bufferMutex.Unlock()
	
	las.logBuffer = append(las.logBuffer, entry)
	
	// Check for immediate alerts
	if entry.Level == LogLevelError || entry.Level == LogLevelFatal {
		go las.checkLogAlert(entry)
	}
}

// startBufferFlusher periodically flushes log buffer to database
func (las *LogAggregationService) startBufferFlusher(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Flush every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			// Final flush before shutdown
			las.flushLogBuffer()
			return
		case <-ticker.C:
			las.flushLogBuffer()
		}
	}
}

// flushLogBuffer flushes the log buffer to database
func (las *LogAggregationService) flushLogBuffer() {
	las.bufferMutex.Lock()
	if len(las.logBuffer) == 0 {
		las.bufferMutex.Unlock()
		return
	}
	
	entries := make([]LogEntry, len(las.logBuffer))
	copy(entries, las.logBuffer)
	las.logBuffer = las.logBuffer[:0] // Clear buffer
	las.bufferMutex.Unlock()
	
	if las.db == nil {
		log.Printf("Database not available, discarding %d log entries", len(entries))
		return
	}
	
	// Batch insert log entries
	if err := las.batchInsertLogEntries(entries); err != nil {
		log.Printf("Error inserting log entries: %v", err)
		
		// Re-add entries to buffer for retry
		las.bufferMutex.Lock()
		las.logBuffer = append(entries, las.logBuffer...)
		las.bufferMutex.Unlock()
	} else {
		log.Printf("Flushed %d log entries to database", len(entries))
	}
}

// batchInsertLogEntries inserts log entries in batch
func (las *LogAggregationService) batchInsertLogEntries(entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	tx, err := las.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO log_entries (timestamp, level, component, message, context, source, trace_id, user_id, request_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()
	
	for _, entry := range entries {
		contextJSON, _ := json.Marshal(entry.Context)
		
		_, err = stmt.ExecContext(ctx,
			entry.Timestamp,
			string(entry.Level),
			entry.Component,
			entry.Message,
			contextJSON,
			entry.Source,
			entry.TraceID,
			entry.UserID,
			entry.RequestID,
			entry.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert log entry: %v", err)
		}
	}
	
	return tx.Commit()
}

// startLogAnalysis starts log analysis for patterns and anomalies
func (las *LogAggregationService) startLogAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Analyze every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			las.analyzeRecentLogs()
		}
	}
}

// analyzeRecentLogs analyzes recent logs for patterns and anomalies
func (las *LogAggregationService) analyzeRecentLogs() {
	if las.db == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Analyze error rate in last 5 minutes
	las.analyzeErrorRate(ctx)
	
	// Analyze log volume
	las.analyzeLogVolume(ctx)
	
	// Look for specific error patterns
	las.analyzeErrorPatterns(ctx)
}

// analyzeErrorRate analyzes error rate and triggers alerts if necessary
func (las *LogAggregationService) analyzeErrorRate(ctx context.Context) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE level = 'error' OR level = 'fatal') as error_count,
			COUNT(*) as total_count
		FROM log_entries 
		WHERE timestamp >= NOW() - INTERVAL '5 minutes'
	`
	
	var errorCount, totalCount int64
	err := las.db.QueryRowContext(ctx, query).Scan(&errorCount, &totalCount)
	if err != nil {
		log.Printf("Error analyzing error rate: %v", err)
		return
	}
	
	if totalCount == 0 {
		return
	}
	
	errorRate := float64(errorCount) / float64(totalCount) * 100
	
	// Trigger alert if error rate is high
	if errorRate > 10.0 { // 10% error rate threshold
		alert := &models.Alert{
			Name:         "high_error_rate",
			Description:  fmt.Sprintf("High error rate detected: %.2f%% (%d errors out of %d logs)", errorRate, errorCount, totalCount),
			Severity:     models.AlertSeverityWarning,
			Status:       models.AlertStatusActive,
			Component:    "logs",
			Metric:       "error_rate",
			Threshold:    10.0,
			CurrentValue: errorRate,
			Metadata:     map[string]interface{}{"error_count": errorCount, "total_count": totalCount},
			TriggeredAt:  time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		if las.alertService != nil {
			las.alertService.SendAlert(alert)
		}
	}
}

// analyzeLogVolume analyzes log volume for anomalies
func (las *LogAggregationService) analyzeLogVolume(ctx context.Context) {
	query := `
		SELECT COUNT(*) 
		FROM log_entries 
		WHERE timestamp >= NOW() - INTERVAL '5 minutes'
	`
	
	var logCount int64
	err := las.db.QueryRowContext(ctx, query).Scan(&logCount)
	if err != nil {
		log.Printf("Error analyzing log volume: %v", err)
		return
	}
	
	// Calculate logs per minute
	logsPerMinute := float64(logCount) / 5.0
	
	// Trigger alert if log volume is unusually high
	if logsPerMinute > 1000 { // 1000 logs per minute threshold
		alert := &models.Alert{
			Name:         "high_log_volume",
			Description:  fmt.Sprintf("High log volume detected: %.2f logs per minute", logsPerMinute),
			Severity:     models.AlertSeverityWarning,
			Status:       models.AlertStatusActive,
			Component:    "logs",
			Metric:       "logs_per_minute",
			Threshold:    1000.0,
			CurrentValue: logsPerMinute,
			Metadata:     map[string]interface{}{"log_count": logCount},
			TriggeredAt:  time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		if las.alertService != nil {
			las.alertService.SendAlert(alert)
		}
	}
}

// analyzeErrorPatterns looks for specific error patterns
func (las *LogAggregationService) analyzeErrorPatterns(ctx context.Context) {
	// Look for database connection errors
	las.checkErrorPattern(ctx, "database_connection_errors", 
		"message ILIKE '%connection%' AND message ILIKE '%database%' AND (level = 'error' OR level = 'fatal')",
		5, "Database connection errors detected")
	
	// Look for out of memory errors
	las.checkErrorPattern(ctx, "out_of_memory_errors",
		"message ILIKE '%out of memory%' OR message ILIKE '%oom%'",
		3, "Out of memory errors detected")
	
	// Look for authentication failures
	las.checkErrorPattern(ctx, "auth_failures",
		"message ILIKE '%authentication%' AND message ILIKE '%failed%'",
		10, "Multiple authentication failures detected")
}

// checkErrorPattern checks for a specific error pattern
func (las *LogAggregationService) checkErrorPattern(ctx context.Context, alertName, whereClause string, threshold int, description string) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM log_entries 
		WHERE timestamp >= NOW() - INTERVAL '5 minutes' AND %s
	`, whereClause)
	
	var count int64
	err := las.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Error checking error pattern %s: %v", alertName, err)
		return
	}
	
	if count >= int64(threshold) {
		alert := &models.Alert{
			Name:         alertName,
			Description:  fmt.Sprintf("%s: %d occurrences in last 5 minutes", description, count),
			Severity:     models.AlertSeverityWarning,
			Status:       models.AlertStatusActive,
			Component:    "logs",
			Metric:       "error_pattern_count",
			Threshold:    float64(threshold),
			CurrentValue: float64(count),
			Metadata:     map[string]interface{}{"pattern": whereClause, "count": count},
			TriggeredAt:  time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		if las.alertService != nil {
			las.alertService.SendAlert(alert)
		}
	}
}

// checkLogAlert checks if a log entry should trigger an immediate alert
func (las *LogAggregationService) checkLogAlert(entry LogEntry) {
	if las.alertService == nil {
		return
	}
	
	// Check for critical errors
	if entry.Level == LogLevelFatal {
		alert := &models.Alert{
			Name:         "fatal_error_logged",
			Description:  fmt.Sprintf("Fatal error in %s: %s", entry.Component, entry.Message),
			Severity:     models.AlertSeverityCritical,
			Status:       models.AlertStatusActive,
			Component:    entry.Component,
			Metric:       "fatal_error",
			Threshold:    0,
			CurrentValue: 1,
			Metadata:     map[string]interface{}{"log_entry": entry},
			TriggeredAt:  time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		las.alertService.SendAlert(alert)
	}
}

// GetRecentLogs retrieves recent log entries
func (las *LogAggregationService) GetRecentLogs(component string, level LogLevel, limit int, since time.Time) ([]LogEntry, error) {
	if las.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var query string
	var args []interface{}
	argIndex := 1
	
	query = "SELECT id, timestamp, level, component, message, context, source, trace_id, user_id, request_id, created_at FROM log_entries WHERE 1=1"
	
	if !since.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, since)
		argIndex++
	}
	
	if component != "" {
		query += fmt.Sprintf(" AND component = $%d", argIndex)
		args = append(args, component)
		argIndex++
	}
	
	if level != "" {
		query += fmt.Sprintf(" AND level = $%d", argIndex)
		args = append(args, string(level))
		argIndex++
	}
	
	query += " ORDER BY timestamp DESC"
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}
	
	rows, err := las.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query log entries: %v", err)
	}
	defer rows.Close()
	
	var entries []LogEntry
	for rows.Next() {
		var entry LogEntry
		var contextJSON []byte
		
		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Level,
			&entry.Component,
			&entry.Message,
			&contextJSON,
			&entry.Source,
			&entry.TraceID,
			&entry.UserID,
			&entry.RequestID,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning log entry: %v", err)
			continue
		}
		
		// Parse context JSON
		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &entry.Context); err != nil {
				entry.Context = make(map[string]interface{})
			}
		} else {
			entry.Context = make(map[string]interface{})
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// GetLogStatistics returns log statistics
func (las *LogAggregationService) GetLogStatistics(since time.Time) (map[string]interface{}, error) {
	if las.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	stats := make(map[string]interface{})
	
	// Get total log count
	var totalCount int64
	err := las.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM log_entries WHERE timestamp >= $1", since).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total log count: %v", err)
	}
	stats["total_logs"] = totalCount
	
	// Get log count by level
	levelQuery := `
		SELECT level, COUNT(*) 
		FROM log_entries 
		WHERE timestamp >= $1 
		GROUP BY level
	`
	rows, err := las.db.QueryContext(ctx, levelQuery, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get log count by level: %v", err)
	}
	defer rows.Close()
	
	levelCounts := make(map[string]int64)
	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			continue
		}
		levelCounts[level] = count
	}
	stats["by_level"] = levelCounts
	
	// Get log count by component
	componentQuery := `
		SELECT component, COUNT(*) 
		FROM log_entries 
		WHERE timestamp >= $1 
		GROUP BY component 
		ORDER BY COUNT(*) DESC 
		LIMIT 10
	`
	rows, err = las.db.QueryContext(ctx, componentQuery, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get log count by component: %v", err)
	}
	defer rows.Close()
	
	componentCounts := make(map[string]int64)
	for rows.Next() {
		var component string
		var count int64
		if err := rows.Scan(&component, &count); err != nil {
			continue
		}
		componentCounts[component] = count
	}
	stats["by_component"] = componentCounts
	
	return stats, nil
}

// CleanupOldLogs removes old log entries based on retention policy
func (las *LogAggregationService) CleanupOldLogs(retentionDays int) error {
	if las.db == nil {
		return fmt.Errorf("database not available")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	result, err := las.db.ExecContext(ctx, "DELETE FROM log_entries WHERE timestamp < $1", cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old logs: %v", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old log entries (older than %d days)", rowsAffected, retentionDays)
	
	return nil
}