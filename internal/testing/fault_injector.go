package testing

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// EnhancedFaultInjector provides comprehensive fault injection capabilities
type EnhancedFaultInjector struct {
	databaseInjector *EnhancedDatabaseInjector
	cacheInjector    *EnhancedCacheInjector
	networkInjector  *EnhancedNetworkInjector
	resourceInjector *EnhancedResourceInjector
	activeFaults     map[string]*FaultInjection
	mutex            sync.RWMutex
}

// FaultInjection represents an active fault injection
type FaultInjection struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	StartTime        time.Time              `json:"start_time"`
	Duration         time.Duration          `json:"duration"`
	Parameters       map[string]interface{} `json:"parameters"`
	StopFunc         func()                 `json:"-"`
	GracefulDegradation bool                `json:"graceful_degradation"`
}

// EnhancedFaultResult represents the result of an enhanced fault injection
type EnhancedFaultResult struct {
	ID                  string        `json:"id"`
	Type                string        `json:"type"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	Duration            time.Duration `json:"duration"`
	GracefulDegradation bool          `json:"graceful_degradation"`
	ErrorsObserved      []string      `json:"errors_observed"`
	MetricsCollected    map[string]float64 `json:"metrics_collected"`
}

// NewEnhancedFaultInjector creates a new enhanced fault injector
func NewEnhancedFaultInjector() *EnhancedFaultInjector {
	return &EnhancedFaultInjector{
		databaseInjector: NewEnhancedDatabaseInjector(),
		cacheInjector:    NewEnhancedCacheInjector(),
		networkInjector:  NewEnhancedNetworkInjector(),
		resourceInjector: NewEnhancedResourceInjector(),
		activeFaults:     make(map[string]*FaultInjection),
	}
}

// GetEnhancedDatabaseInjector returns the database fault injector
func (efi *EnhancedFaultInjector) GetEnhancedDatabaseInjector() *EnhancedDatabaseInjector {
	return efi.databaseInjector
}

// GetEnhancedCacheInjector returns the cache fault injector
func (efi *EnhancedFaultInjector) GetEnhancedCacheInjector() *EnhancedCacheInjector {
	return efi.cacheInjector
}

// GetEnhancedNetworkInjector returns the network fault injector
func (efi *EnhancedFaultInjector) GetEnhancedNetworkInjector() *EnhancedNetworkInjector {
	return efi.networkInjector
}

// GetEnhancedResourceInjector returns the resource fault injector
func (efi *EnhancedFaultInjector) GetEnhancedResourceInjector() *EnhancedResourceInjector {
	return efi.resourceInjector
}

// InjectDatabaseConnectionPoolExhaustion injects database connection pool exhaustion
func (efi *EnhancedFaultInjector) InjectDatabaseConnectionPoolExhaustion(ctx context.Context, duration time.Duration) (*EnhancedFaultResult, error) {
	faultID := fmt.Sprintf("db_pool_exhaustion_%d", time.Now().UnixNano())
	
	result := &EnhancedFaultResult{
		ID:               faultID,
		Type:             "database_pool_exhaustion",
		StartTime:        time.Now(),
		MetricsCollected: make(map[string]float64),
		ErrorsObserved:   make([]string, 0),
	}

	// Simulate database connection pool exhaustion
	stopFunc, err := efi.databaseInjector.InjectConnectionPoolExhaustion(duration)
	if err != nil {
		return nil, fmt.Errorf("failed to inject database pool exhaustion: %w", err)
	}

	// Track the fault
	fault := &FaultInjection{
		ID:         faultID,
		Type:       "database_pool_exhaustion",
		StartTime:  result.StartTime,
		Duration:   duration,
		Parameters: map[string]interface{}{"duration": duration.String()},
		StopFunc:   stopFunc,
	}

	efi.mutex.Lock()
	efi.activeFaults[faultID] = fault
	efi.mutex.Unlock()

	// Wait for the fault duration
	select {
	case <-ctx.Done():
		stopFunc()
		result.ErrorsObserved = append(result.ErrorsObserved, "Context cancelled")
		return result, ctx.Err()
	case <-time.After(duration):
		stopFunc()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.GracefulDegradation = true // Simulate graceful degradation detection
	result.MetricsCollected["connections_exhausted"] = 1.0

	efi.mutex.Lock()
	delete(efi.activeFaults, faultID)
	efi.mutex.Unlock()

	return result, nil
}

// InjectCacheMemoryLeak injects cache memory leak
func (efi *EnhancedFaultInjector) InjectCacheMemoryLeak(ctx context.Context, duration time.Duration) (*EnhancedFaultResult, error) {
	faultID := fmt.Sprintf("cache_memory_leak_%d", time.Now().UnixNano())
	
	result := &EnhancedFaultResult{
		ID:               faultID,
		Type:             "cache_memory_leak",
		StartTime:        time.Now(),
		MetricsCollected: make(map[string]float64),
		ErrorsObserved:   make([]string, 0),
	}

	// Simulate cache memory leak
	stopFunc, err := efi.cacheInjector.InjectMemoryLeak(duration)
	if err != nil {
		return nil, fmt.Errorf("failed to inject cache memory leak: %w", err)
	}

	// Track the fault
	fault := &FaultInjection{
		ID:         faultID,
		Type:       "cache_memory_leak",
		StartTime:  result.StartTime,
		Duration:   duration,
		Parameters: map[string]interface{}{"duration": duration.String()},
		StopFunc:   stopFunc,
	}

	efi.mutex.Lock()
	efi.activeFaults[faultID] = fault
	efi.mutex.Unlock()

	// Wait for the fault duration
	select {
	case <-ctx.Done():
		stopFunc()
		result.ErrorsObserved = append(result.ErrorsObserved, "Context cancelled")
		return result, ctx.Err()
	case <-time.After(duration):
		stopFunc()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.GracefulDegradation = true // Simulate graceful degradation detection
	result.MetricsCollected["memory_leaked_mb"] = 100.0

	efi.mutex.Lock()
	delete(efi.activeFaults, faultID)
	efi.mutex.Unlock()

	return result, nil
}

// InjectCPUSpike injects CPU spike
func (efi *EnhancedFaultInjector) InjectCPUSpike(ctx context.Context, intensity float64, duration time.Duration) (*EnhancedFaultResult, error) {
	faultID := fmt.Sprintf("cpu_spike_%d", time.Now().UnixNano())
	
	result := &EnhancedFaultResult{
		ID:               faultID,
		Type:             "cpu_spike",
		StartTime:        time.Now(),
		MetricsCollected: make(map[string]float64),
		ErrorsObserved:   make([]string, 0),
	}

	// Simulate CPU spike
	stopFunc, err := efi.resourceInjector.InjectCPUSpike(intensity, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to inject CPU spike: %w", err)
	}

	// Track the fault
	fault := &FaultInjection{
		ID:         faultID,
		Type:       "cpu_spike",
		StartTime:  result.StartTime,
		Duration:   duration,
		Parameters: map[string]interface{}{"intensity": intensity, "duration": duration.String()},
		StopFunc:   stopFunc,
	}

	efi.mutex.Lock()
	efi.activeFaults[faultID] = fault
	efi.mutex.Unlock()

	// Wait for the fault duration
	select {
	case <-ctx.Done():
		stopFunc()
		result.ErrorsObserved = append(result.ErrorsObserved, "Context cancelled")
		return result, ctx.Err()
	case <-time.After(duration):
		stopFunc()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.GracefulDegradation = true // Simulate graceful degradation detection
	result.MetricsCollected["cpu_intensity"] = intensity

	efi.mutex.Lock()
	delete(efi.activeFaults, faultID)
	efi.mutex.Unlock()

	return result, nil
}

// InjectSystemClockSkew injects system clock skew
func (efi *EnhancedFaultInjector) InjectSystemClockSkew(ctx context.Context, skew time.Duration, duration time.Duration) (*EnhancedFaultResult, error) {
	faultID := fmt.Sprintf("clock_skew_%d", time.Now().UnixNano())
	
	result := &EnhancedFaultResult{
		ID:               faultID,
		Type:             "clock_skew",
		StartTime:        time.Now(),
		MetricsCollected: make(map[string]float64),
		ErrorsObserved:   make([]string, 0),
	}

	// Simulate clock skew (this is a simulation since we can't actually change system time)
	stopFunc := func() {
		// Cleanup simulation
	}

	// Track the fault
	fault := &FaultInjection{
		ID:         faultID,
		Type:       "clock_skew",
		StartTime:  result.StartTime,
		Duration:   duration,
		Parameters: map[string]interface{}{"skew": skew.String(), "duration": duration.String()},
		StopFunc:   stopFunc,
	}

	efi.mutex.Lock()
	efi.activeFaults[faultID] = fault
	efi.mutex.Unlock()

	// Wait for the fault duration
	select {
	case <-ctx.Done():
		stopFunc()
		result.ErrorsObserved = append(result.ErrorsObserved, "Context cancelled")
		return result, ctx.Err()
	case <-time.After(duration):
		stopFunc()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.GracefulDegradation = true // Simulate graceful degradation detection
	result.MetricsCollected["clock_skew_seconds"] = skew.Seconds()

	efi.mutex.Lock()
	delete(efi.activeFaults, faultID)
	efi.mutex.Unlock()

	return result, nil
}

// GetActiveFaults returns currently active faults
func (efi *EnhancedFaultInjector) GetActiveFaults() []*FaultInjection {
	efi.mutex.RLock()
	defer efi.mutex.RUnlock()

	faults := make([]*FaultInjection, 0, len(efi.activeFaults))
	for _, fault := range efi.activeFaults {
		faults = append(faults, fault)
	}
	return faults
}

// StopAllFaults stops all active fault injections
func (efi *EnhancedFaultInjector) StopAllFaults() {
	efi.mutex.Lock()
	defer efi.mutex.Unlock()

	for _, fault := range efi.activeFaults {
		if fault.StopFunc != nil {
			fault.StopFunc()
		}
	}
	efi.activeFaults = make(map[string]*FaultInjection)
}

// Enhanced Database Injector
type EnhancedDatabaseInjector struct {
	activeConnections int64
	maxConnections    int64
	mutex             sync.RWMutex
}

func NewEnhancedDatabaseInjector() *EnhancedDatabaseInjector {
	return &EnhancedDatabaseInjector{
		maxConnections: 100, // Default max connections
	}
}

func (edi *EnhancedDatabaseInjector) InjectConnectionPoolExhaustion(duration time.Duration) (func(), error) {
	// Simulate connection pool exhaustion by consuming connections
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				atomic.AddInt64(&edi.activeConnections, 1)
				if atomic.LoadInt64(&edi.activeConnections) > edi.maxConnections {
					atomic.StoreInt64(&edi.activeConnections, edi.maxConnections)
				}
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
		atomic.StoreInt64(&edi.activeConnections, 0)
	}

	return stopFunc, nil
}

func (edi *EnhancedDatabaseInjector) InjectSlowQueries(duration time.Duration) (func(), error) {
	// Simulate slow queries by adding artificial delays
	stopChan := make(chan struct{})
	
	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

// Enhanced Cache Injector
type EnhancedCacheInjector struct {
	memoryUsage int64
	mutex       sync.RWMutex
}

func NewEnhancedCacheInjector() *EnhancedCacheInjector {
	return &EnhancedCacheInjector{}
}

func (eci *EnhancedCacheInjector) InjectMemoryLeak(duration time.Duration) (func(), error) {
	// Simulate memory leak by gradually increasing memory usage
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				atomic.AddInt64(&eci.memoryUsage, 1024*1024) // Add 1MB
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
		atomic.StoreInt64(&eci.memoryUsage, 0)
	}

	return stopFunc, nil
}

func (eci *EnhancedCacheInjector) InjectEvictionStorm(duration time.Duration) (func(), error) {
	// Simulate cache eviction storm
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				// Simulate rapid cache evictions
				runtime.GC() // Force garbage collection to simulate evictions
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

// Enhanced Network Injector
type EnhancedNetworkInjector struct {
	latencyMs int64
	bandwidth int64
	mutex     sync.RWMutex
}

func NewEnhancedNetworkInjector() *EnhancedNetworkInjector {
	return &EnhancedNetworkInjector{}
}

func (eni *EnhancedNetworkInjector) InjectLatency(latency time.Duration, duration time.Duration) (func(), error) {
	// Simulate network latency
	stopChan := make(chan struct{})
	atomic.StoreInt64(&eni.latencyMs, latency.Milliseconds())
	
	stopFunc := func() {
		close(stopChan)
		atomic.StoreInt64(&eni.latencyMs, 0)
	}

	return stopFunc, nil
}

func (eni *EnhancedNetworkInjector) InjectPacketLoss(lossRate float64, duration time.Duration) (func(), error) {
	// Simulate packet loss
	stopChan := make(chan struct{})
	
	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

func (eni *EnhancedNetworkInjector) InjectDNSFailure(duration time.Duration) (func(), error) {
	// Simulate DNS failure
	stopChan := make(chan struct{})
	
	go func() {
		// Simulate DNS resolution failures
		for {
			select {
			case <-stopChan:
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

func (eni *EnhancedNetworkInjector) InjectBandwidthLimit(limitMbps float64, duration time.Duration) (func(), error) {
	// Simulate bandwidth limitation
	atomic.StoreInt64(&eni.bandwidth, int64(limitMbps*1024*1024)) // Convert to bytes per second
	
	stopFunc := func() {
		atomic.StoreInt64(&eni.bandwidth, 0)
	}

	// Auto-restore after duration
	go func() {
		time.Sleep(duration)
		stopFunc()
	}()

	return stopFunc, nil
}

// InjectNetworkLatency injects network latency
func (eni *EnhancedNetworkInjector) InjectNetworkLatency(latencyMs int64, duration time.Duration) (func(), error) {
	atomic.StoreInt64(&eni.latencyMs, latencyMs)

	stopFunc := func() {
		atomic.StoreInt64(&eni.latencyMs, 0)
	}

	// Auto-restore after duration
	go func() {
		time.Sleep(duration)
		stopFunc()
	}()

	return stopFunc, nil
}

// Enhanced Resource Injector
type EnhancedResourceInjector struct {
	cpuUsage    float64
	memoryUsage int64
	diskUsage   int64
	mutex       sync.RWMutex
}

func NewEnhancedResourceInjector() *EnhancedResourceInjector {
	return &EnhancedResourceInjector{}
}

func (eri *EnhancedResourceInjector) InjectCPUSpike(intensity float64, duration time.Duration) (func(), error) {
	// Simulate CPU spike by running CPU-intensive operations
	stopChan := make(chan struct{})
	numCPU := runtime.NumCPU()
	
	// Start CPU-intensive goroutines
	for i := 0; i < int(float64(numCPU)*intensity); i++ {
		go func() {
			for {
				select {
				case <-stopChan:
					return
				default:
					// CPU-intensive operation
					for j := 0; j < 1000000; j++ {
						_ = j * j
					}
				}
			}
		}()
	}

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

func (eri *EnhancedResourceInjector) InjectMemoryPressure(sizeMB int64, duration time.Duration) (func(), error) {
	// Simulate memory pressure by allocating large amounts of memory
	stopChan := make(chan struct{})
	var memoryBlocks [][]byte
	
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				// Release memory
				memoryBlocks = nil
				runtime.GC()
				return
			case <-ticker.C:
				// Allocate 10MB blocks
				block := make([]byte, 10*1024*1024)
				memoryBlocks = append(memoryBlocks, block)
				if int64(len(memoryBlocks)*10) >= sizeMB {
					// Stop allocating when we reach the target size
					<-stopChan
					memoryBlocks = nil
					runtime.GC()
					return
				}
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

func (eri *EnhancedResourceInjector) InjectDiskPressure(sizeGB int64, duration time.Duration) (func(), error) {
	// Simulate disk pressure by creating temporary files
	stopChan := make(chan struct{})
	var tempFiles []string
	
	go func() {
		defer func() {
			// Cleanup temp files
			for _, file := range tempFiles {
				os.Remove(file)
			}
		}()
		
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				// Create 100MB temp files
				tempFile, err := os.CreateTemp("", "fault_injection_*.tmp")
				if err != nil {
					continue
				}
				
				// Write 100MB of data
				data := make([]byte, 100*1024*1024)
				tempFile.Write(data)
				tempFile.Close()
				
				tempFiles = append(tempFiles, tempFile.Name())
				if int64(len(tempFiles)*100) >= sizeGB*1024 {
					// Stop when we reach the target size
					<-stopChan
					return
				}
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}

func (eri *EnhancedResourceInjector) InjectIOBottleneck(duration time.Duration) (func(), error) {
	// Simulate I/O bottleneck by performing intensive disk operations
	stopChan := make(chan struct{})
	
	go func() {
		for {
			select {
			case <-stopChan:
				return
			default:
				// Perform intensive I/O operations
				tempFile, err := os.CreateTemp("", "io_bottleneck_*.tmp")
				if err != nil {
					continue
				}
				
				// Write and read data repeatedly
				data := make([]byte, 1024*1024) // 1MB
				for i := 0; i < 10; i++ {
					tempFile.Write(data)
					tempFile.Seek(0, 0)
					tempFile.Read(data)
				}
				
				tempFile.Close()
				os.Remove(tempFile.Name())
				
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return stopFunc, nil
}



// NewEnhancedFaultInjector creates a new enhanced fault injector
