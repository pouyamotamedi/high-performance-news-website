package testing

import (
	"sync"
	"time"
)

// SecurityTestRunner orchestrates comprehensive security testing
type SecurityTestRunner struct {
	projectRoot     string
	scanner         *SecurityScanner
	alerting        *SecurityAlerting
	reportDir       string
	config          SecurityTestConfig
	mutex           sync.RWMutex
	activeScans     map[string]*SecurityScan
}

// SecurityTestConfig holds configuration for security testing
type SecurityTestConfig struct {
	EnabledTests []string
	Timeout      time.Duration
}

// SecurityScan represents an active security scan
type SecurityScan struct {
	ID        string
	Type      string
	Status    string
	StartTime time.Time
}