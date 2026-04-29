package synthetic

import (
	"encoding/json"
	"sync"
	"time"
)

// ResultStore interface for storing monitoring results
type ResultStore interface {
	Store(result MonitoringResult) error
	GetResults(testName string, since time.Time) ([]MonitoringResult, error)
	GetLatestResults(limit int) ([]MonitoringResult, error)
	GetTestSummary(testName string, duration time.Duration) (TestSummary, error)
}

// MemoryResultStore implements ResultStore using in-memory storage
type MemoryResultStore struct {
	results []MonitoringResult
	mu      sync.RWMutex
}

// TestSummary provides aggregated test statistics
type TestSummary struct {
	TestName        string        `json:"test_name"`
	TotalRuns       int           `json:"total_runs"`
	SuccessRate     float64       `json:"success_rate"`
	AverageResponse time.Duration `json:"average_response"`
	LastRun         time.Time     `json:"last_run"`
	Trends          []TrendPoint  `json:"trends"`
}

// TrendPoint represents a data point in performance trends
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Status    TestStatus `json:"status"`
}

// NewMemoryResultStore creates a new in-memory result store
func NewMemoryResultStore() *MemoryResultStore {
	return &MemoryResultStore{
		results: make([]MonitoringResult, 0),
	}
}

// Store saves a monitoring result
func (m *MemoryResultStore) Store(result MonitoringResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = append(m.results, result)

	// Keep only last 10000 results to prevent memory issues
	if len(m.results) > 10000 {
		m.results = m.results[len(m.results)-10000:]
	}

	return nil
}

// GetResults retrieves results for a specific test since a given time
func (m *MemoryResultStore) GetResults(testName string, since time.Time) ([]MonitoringResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []MonitoringResult
	for _, result := range m.results {
		if result.TestName == testName && result.Timestamp.After(since) {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// GetLatestResults retrieves the most recent results across all tests
func (m *MemoryResultStore) GetLatestResults(limit int) ([]MonitoringResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.results) == 0 {
		return []MonitoringResult{}, nil
	}

	start := len(m.results) - limit
	if start < 0 {
		start = 0
	}

	return m.results[start:], nil
}

// GetTestSummary generates a summary for a specific test over a duration
func (m *MemoryResultStore) GetTestSummary(testName string, duration time.Duration) (TestSummary, error) {
	since := time.Now().Add(-duration)
	results, err := m.GetResults(testName, since)
	if err != nil {
		return TestSummary{}, err
	}

	if len(results) == 0 {
		return TestSummary{
			TestName:  testName,
			TotalRuns: 0,
		}, nil
	}

	// Calculate statistics
	totalRuns := len(results)
	successCount := 0
	var totalDuration time.Duration
	var trends []TrendPoint

	for _, result := range results {
		if result.Status == StatusPassed {
			successCount++
		}
		totalDuration += result.Duration

		// Add to trends
		trends = append(trends, TrendPoint{
			Timestamp: result.Timestamp,
			Value:     float64(result.Duration.Milliseconds()),
			Status:    result.Status,
		})
	}

	successRate := float64(successCount) / float64(totalRuns) * 100
	averageResponse := totalDuration / time.Duration(totalRuns)

	return TestSummary{
		TestName:        testName,
		TotalRuns:       totalRuns,
		SuccessRate:     successRate,
		AverageResponse: averageResponse,
		LastRun:         results[len(results)-1].Timestamp,
		Trends:          trends,
	}, nil
}

// GetAllTestNames returns all unique test names
func (m *MemoryResultStore) GetAllTestNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	testNames := make(map[string]bool)
	for _, result := range m.results {
		testNames[result.TestName] = true
	}

	names := make([]string, 0, len(testNames))
	for name := range testNames {
		names = append(names, name)
	}

	return names
}

// ExportResults exports results as JSON
func (m *MemoryResultStore) ExportResults() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.MarshalIndent(m.results, "", "  ")
}

// ImportResults imports results from JSON
func (m *MemoryResultStore) ImportResults(data []byte) error {
	var results []MonitoringResult
	if err := json.Unmarshal(data, &results); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = append(m.results, results...)
	return nil
}

// ClearResults clears all stored results
func (m *MemoryResultStore) ClearResults() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = make([]MonitoringResult, 0)
}

// GetResultsCount returns the total number of stored results
func (m *MemoryResultStore) GetResultsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.results)
}