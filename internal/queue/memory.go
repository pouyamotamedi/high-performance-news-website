package queue

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MemoryMonitorImpl implements memory monitoring functionality
type MemoryMonitorImpl struct {
	threshold uint64 // Memory threshold in bytes (28GB = 28 * 1024 * 1024 * 1024)
	mu        sync.RWMutex
}

// NewMemoryMonitor creates a new memory monitor with 28GB threshold
func NewMemoryMonitor() *MemoryMonitorImpl {
	return &MemoryMonitorImpl{
		threshold: 28 * 1024 * 1024 * 1024, // 28GB in bytes
	}
}

// GetMemoryUsage returns current memory usage in bytes
func (m *MemoryMonitorImpl) GetMemoryUsage() (uint64, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Return heap memory in use
	return memStats.HeapInuse, nil
}

// IsMemoryPressure checks if memory usage exceeds threshold
func (m *MemoryMonitorImpl) IsMemoryPressure() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, err := m.GetMemoryUsage()
	if err != nil {
		return true // Assume pressure if we can't get memory stats
	}
	
	return usage > m.threshold
}

// GetMemoryThreshold returns the memory threshold
func (m *MemoryMonitorImpl) GetMemoryThreshold() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.threshold
}

// SetMemoryThreshold updates the memory threshold
func (m *MemoryMonitorImpl) SetMemoryThreshold(threshold uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threshold = threshold
}

// GetMemoryStats returns detailed memory statistics
func (m *MemoryMonitorImpl) GetMemoryStats() (*MemoryStats, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &MemoryStats{
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapInuse:    memStats.HeapInuse,
		HeapReleased: memStats.HeapReleased,
		StackInuse:   memStats.StackInuse,
		StackSys:     memStats.StackSys,
		MSpanInuse:   memStats.MSpanInuse,
		MSpanSys:     memStats.MSpanSys,
		MCacheInuse:  memStats.MCacheInuse,
		MCacheSys:    memStats.MCacheSys,
		GCSys:        memStats.GCSys,
		OtherSys:     memStats.OtherSys,
		NextGC:       memStats.NextGC,
		LastGC:       time.Unix(0, int64(memStats.LastGC)),
		NumGC:        memStats.NumGC,
		Threshold:    m.threshold,
		IsPressure:   memStats.HeapInuse > m.threshold,
	}, nil
}

// MemoryStats contains detailed memory statistics
type MemoryStats struct {
	HeapAlloc    uint64    `json:"heap_alloc"`
	HeapSys      uint64    `json:"heap_sys"`
	HeapInuse    uint64    `json:"heap_inuse"`
	HeapReleased uint64    `json:"heap_released"`
	StackInuse   uint64    `json:"stack_inuse"`
	StackSys     uint64    `json:"stack_sys"`
	MSpanInuse   uint64    `json:"mspan_inuse"`
	MSpanSys     uint64    `json:"mspan_sys"`
	MCacheInuse  uint64    `json:"mcache_inuse"`
	MCacheSys    uint64    `json:"mcache_sys"`
	GCSys        uint64    `json:"gc_sys"`
	OtherSys     uint64    `json:"other_sys"`
	NextGC       uint64    `json:"next_gc"`
	LastGC       time.Time `json:"last_gc"`
	NumGC        uint32    `json:"num_gc"`
	Threshold    uint64    `json:"threshold"`
	IsPressure   bool      `json:"is_pressure"`
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}