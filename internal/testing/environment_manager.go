package testing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// EnvironmentStatus represents the current status of a test environment
type EnvironmentStatus string

const (
	EnvironmentStatusCreating EnvironmentStatus = "creating"
	EnvironmentStatusReady    EnvironmentStatus = "ready"
	EnvironmentStatusFailed   EnvironmentStatus = "failed"
	EnvironmentStatusCleaning EnvironmentStatus = "cleaning"
	EnvironmentStatusDestroyed EnvironmentStatus = "destroyed"
)

// ResourceAllocation represents the resources allocated to an environment
type ResourceAllocation struct {
	Memory    int64  `json:"memory"`     // Memory in bytes
	CPUQuota  int64  `json:"cpu_quota"`  // CPU quota
	DiskSpace int64  `json:"disk_space"` // Disk space in bytes
	NetworkID string `json:"network_id"` // Docker network ID
}

// IsolatedEnvironment represents a single isolated test environment
type IsolatedEnvironment struct {
	ID              string             `json:"id"`
	ContainerID     string             `json:"container_id"`
	CacheContainerID string            `json:"cache_container_id"`
	DatabaseURL     string             `json:"database_url"`
	CacheURL        string             `json:"cache_url"`
	Status          EnvironmentStatus  `json:"status"`
	Resources       ResourceAllocation `json:"resources"`
	TestSuite       string             `json:"test_suite"`
	CreatedAt       time.Time          `json:"created_at"`
	LastHealthCheck time.Time          `json:"last_health_check"`
	HealthStatus    string             `json:"health_status"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	mutex           sync.RWMutex
}

// ResourcePool manages available resources for test environments
type ResourcePool struct {
	MaxMemory       int64 `json:"max_memory"`
	MaxCPU          int64 `json:"max_cpu"`
	MaxEnvironments int   `json:"max_environments"`
	
	allocatedMemory int64
	allocatedCPU    int64
	activeEnvs      int
	mutex           sync.RWMutex
}

// EnvironmentHealthMonitor monitors the health of test environments
type EnvironmentHealthMonitor struct {
	environments map[string]*IsolatedEnvironment
	docker       *client.Client
	checkInterval time.Duration
	mutex        sync.RWMutex
	stopChan     chan struct{}
}

// TestEnvironmentManager manages Docker-based isolated test environments
type TestEnvironmentManager struct {
	docker          *client.Client
	environments    map[string]*IsolatedEnvironment
	resourcePool    *ResourcePool
	healthMonitor   *EnvironmentHealthMonitor
	mutex           sync.RWMutex
	networkID       string
}

// NewTestEnvironmentManager creates a new test environment manager
func NewTestEnvironmentManager() (*TestEnvironmentManager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Create dedicated test network
	networkID, err := createTestNetwork(dockerClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create test network: %w", err)
	}

	resourcePool := &ResourcePool{
		MaxMemory:       8 * 1024 * 1024 * 1024, // 8GB
		MaxCPU:          400000,                   // 4 CPU cores (400% quota)
		MaxEnvironments: 10,
	}

	healthMonitor := &EnvironmentHealthMonitor{
		environments:  make(map[string]*IsolatedEnvironment),
		docker:        dockerClient,
		checkInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}

	manager := &TestEnvironmentManager{
		docker:        dockerClient,
		environments:  make(map[string]*IsolatedEnvironment),
		resourcePool:  resourcePool,
		healthMonitor: healthMonitor,
		networkID:     networkID,
	}

	// Start health monitoring
	go healthMonitor.Start()

	return manager, nil
}

// CreateIsolatedEnvironment creates a new isolated test environment with dedicated resources
func (t *TestEnvironmentManager) CreateIsolatedEnvironment(testSuite string) (*IsolatedEnvironment, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Check resource availability
	if !t.resourcePool.CanAllocate(512*1024*1024, 50000) { // 512MB, 50% CPU
		return nil, fmt.Errorf("insufficient resources available")
	}

	envID := generateUniqueID()
	env := &IsolatedEnvironment{
		ID:        envID,
		TestSuite: testSuite,
		Status:    EnvironmentStatusCreating,
		CreatedAt: time.Now(),
		Resources: ResourceAllocation{
			Memory:   512 * 1024 * 1024, // 512MB
			CPUQuota: 50000,              // 50% CPU
		},
	}

	t.environments[envID] = env

	// Create environment asynchronously
	go func() {
		if err := t.createEnvironmentContainers(env); err != nil {
			env.mutex.Lock()
			env.Status = EnvironmentStatusFailed
			env.ErrorMessage = err.Error()
			env.mutex.Unlock()
			log.Printf("Failed to create environment %s: %v", envID, err)
			return
		}

		env.mutex.Lock()
		env.Status = EnvironmentStatusReady
		env.mutex.Unlock()

		// Add to health monitoring
		t.healthMonitor.AddEnvironment(env)

		log.Printf("Environment %s created successfully for test suite %s", envID, testSuite)
	}()

	return env, nil
}

// createEnvironmentContainers creates the database and cache containers for an environment
func (t *TestEnvironmentManager) createEnvironmentContainers(env *IsolatedEnvironment) error {
	ctx := context.Background()

	// Create PostgreSQL container
	dbPort, err := t.getAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to get available port for database: %w", err)
	}

	dbContainerConfig := &container.Config{
		Image: "postgres:15-alpine",
		Env: []string{
			"POSTGRES_DB=test_" + env.ID,
			"POSTGRES_USER=testuser",
			"POSTGRES_PASSWORD=testpass",
			"POSTGRES_INITDB_ARGS=--auth-host=trust",
		},
		ExposedPorts: nat.PortSet{
			"5432/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "pg_isready -U testuser -d test_" + env.ID},
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  5,
		},
	}

	dbHostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: strconv.Itoa(dbPort),
				},
			},
		},
		Resources: container.Resources{
			Memory:   env.Resources.Memory / 2, // Half memory for DB
			CPUQuota: env.Resources.CPUQuota / 2,
		},
		AutoRemove: true,
	}

	dbContainer, err := t.docker.ContainerCreate(ctx, dbContainerConfig, dbHostConfig, nil, nil, "test-db-"+env.ID)
	if err != nil {
		return fmt.Errorf("failed to create database container: %w", err)
	}

	if err := t.docker.ContainerStart(ctx, dbContainer.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start database container: %w", err)
	}

	env.ContainerID = dbContainer.ID
	env.DatabaseURL = fmt.Sprintf("postgres://testuser:testpass@localhost:%d/test_%s?sslmode=disable", dbPort, env.ID)

	// Create Redis/DragonflyDB container for caching
	cachePort, err := t.getAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to get available port for cache: %w", err)
	}

	cacheContainerConfig := &container.Config{
		Image: "redis:7-alpine",
		ExposedPorts: nat.PortSet{
			"6379/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "redis-cli", "ping"},
			Interval: 10 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  3,
		},
	}

	cacheHostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"6379/tcp": []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: strconv.Itoa(cachePort),
				},
			},
		},
		Resources: container.Resources{
			Memory:   env.Resources.Memory / 2, // Half memory for cache
			CPUQuota: env.Resources.CPUQuota / 2,
		},
		AutoRemove: true,
	}

	cacheContainer, err := t.docker.ContainerCreate(ctx, cacheContainerConfig, cacheHostConfig, nil, nil, "test-cache-"+env.ID)
	if err != nil {
		return fmt.Errorf("failed to create cache container: %w", err)
	}

	if err := t.docker.ContainerStart(ctx, cacheContainer.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start cache container: %w", err)
	}

	env.CacheContainerID = cacheContainer.ID
	env.CacheURL = fmt.Sprintf("redis://localhost:%d", cachePort)

	// Wait for containers to be healthy
	if err := t.waitForContainerHealth(dbContainer.ID, 60*time.Second); err != nil {
		return fmt.Errorf("database container failed to become healthy: %w", err)
	}

	if err := t.waitForContainerHealth(cacheContainer.ID, 30*time.Second); err != nil {
		return fmt.Errorf("cache container failed to become healthy: %w", err)
	}

	// Allocate resources
	t.resourcePool.AllocateResources(env.Resources)

	return nil
}

// GetEnvironment retrieves an environment by ID
func (t *TestEnvironmentManager) GetEnvironment(envID string) (*IsolatedEnvironment, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	env, exists := t.environments[envID]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", envID)
	}

	return env, nil
}

// ListEnvironments returns all environments
func (t *TestEnvironmentManager) ListEnvironments() []*IsolatedEnvironment {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	environments := make([]*IsolatedEnvironment, 0, len(t.environments))
	for _, env := range t.environments {
		environments = append(environments, env)
	}

	return environments
}

// CleanupEnvironment stops and removes an environment, releasing its resources
func (t *TestEnvironmentManager) CleanupEnvironment(envID string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	env, exists := t.environments[envID]
	if !exists {
		return fmt.Errorf("environment %s not found", envID)
	}

	env.mutex.Lock()
	env.Status = EnvironmentStatusCleaning
	env.mutex.Unlock()

	ctx := context.Background()

	// Remove from health monitoring
	t.healthMonitor.RemoveEnvironment(envID)

	// Stop and remove database container
	if env.ContainerID != "" {
		if err := t.docker.ContainerStop(ctx, env.ContainerID, container.StopOptions{}); err != nil {
			log.Printf("Warning: failed to stop database container %s: %v", env.ContainerID, err)
		}

		if err := t.docker.ContainerRemove(ctx, env.ContainerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			log.Printf("Warning: failed to remove database container %s: %v", env.ContainerID, err)
		}
	}

	// Stop and remove cache container
	if env.CacheContainerID != "" {
		if err := t.docker.ContainerStop(ctx, env.CacheContainerID, container.StopOptions{}); err != nil {
			log.Printf("Warning: failed to stop cache container %s: %v", env.CacheContainerID, err)
		}

		if err := t.docker.ContainerRemove(ctx, env.CacheContainerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			log.Printf("Warning: failed to remove cache container %s: %v", env.CacheContainerID, err)
		}
	}

	// Release resources
	t.resourcePool.ReleaseResources(env.Resources)

	// Update status and remove from tracking
	env.mutex.Lock()
	env.Status = EnvironmentStatusDestroyed
	env.mutex.Unlock()

	delete(t.environments, envID)

	log.Printf("Environment %s cleaned up successfully", envID)
	return nil
}

// CleanupAllEnvironments cleans up all environments
func (t *TestEnvironmentManager) CleanupAllEnvironments() error {
	t.mutex.RLock()
	envIDs := make([]string, 0, len(t.environments))
	for id := range t.environments {
		envIDs = append(envIDs, id)
	}
	t.mutex.RUnlock()

	var lastError error
	for _, envID := range envIDs {
		if err := t.CleanupEnvironment(envID); err != nil {
			log.Printf("Failed to cleanup environment %s: %v", envID, err)
			lastError = err
		}
	}

	return lastError
}

// Shutdown stops the environment manager and cleans up all resources
func (t *TestEnvironmentManager) Shutdown() error {
	// Stop health monitoring
	t.healthMonitor.Stop()

	// Cleanup all environments
	if err := t.CleanupAllEnvironments(); err != nil {
		log.Printf("Error during environment cleanup: %v", err)
	}

	// Remove test network
	if t.networkID != "" {
		ctx := context.Background()
		if err := t.docker.NetworkRemove(ctx, t.networkID); err != nil {
			log.Printf("Warning: failed to remove test network %s: %v", t.networkID, err)
		}
	}

	// Close Docker client
	return t.docker.Close()
}

// Helper methods

func (t *TestEnvironmentManager) getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func (t *TestEnvironmentManager) waitForContainerHealth(containerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container %s to become healthy", containerID)
		case <-ticker.C:
			inspect, err := t.docker.ContainerInspect(context.Background(), containerID)
			if err != nil {
				continue
			}

			if inspect.State.Health != nil && inspect.State.Health.Status == "healthy" {
				return nil
			}

			if inspect.State.Health != nil && inspect.State.Health.Status == "unhealthy" {
				return fmt.Errorf("container %s became unhealthy", containerID)
			}
		}
	}
}

func generateUniqueID() string {
	return uuid.New().String()[:8]
}

func createTestNetwork(dockerClient *client.Client) (string, error) {
	ctx := context.Background()
	
	networkName := "test-network-" + generateUniqueID()
	
	network, err := dockerClient.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
		// Remove problematic options that might cause numerical range issues
	})
	if err != nil {
		// If network creation fails, try to use the default bridge network
		log.Printf("Warning: failed to create custom network, using default bridge: %v", err)
		return "bridge", nil
	}

	return network.ID, nil
}

// ResourcePool methods

func (r *ResourcePool) CanAllocate(memory, cpu int64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.allocatedMemory+memory <= r.MaxMemory &&
		r.allocatedCPU+cpu <= r.MaxCPU &&
		r.activeEnvs < r.MaxEnvironments
}

func (r *ResourcePool) AllocateResources(allocation ResourceAllocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.allocatedMemory += allocation.Memory
	r.allocatedCPU += allocation.CPUQuota
	r.activeEnvs++
}

func (r *ResourcePool) ReleaseResources(allocation ResourceAllocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.allocatedMemory -= allocation.Memory
	r.allocatedCPU -= allocation.CPUQuota
	r.activeEnvs--

	// Ensure we don't go negative
	if r.allocatedMemory < 0 {
		r.allocatedMemory = 0
	}
	if r.allocatedCPU < 0 {
		r.allocatedCPU = 0
	}
	if r.activeEnvs < 0 {
		r.activeEnvs = 0
	}
}

func (r *ResourcePool) GetUtilization() (float64, float64, float64) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	memoryUtil := float64(r.allocatedMemory) / float64(r.MaxMemory)
	cpuUtil := float64(r.allocatedCPU) / float64(r.MaxCPU)
	envUtil := float64(r.activeEnvs) / float64(r.MaxEnvironments)

	return memoryUtil, cpuUtil, envUtil
}

// EnvironmentHealthMonitor methods

func (h *EnvironmentHealthMonitor) Start() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.performHealthChecks()
		}
	}
}

func (h *EnvironmentHealthMonitor) Stop() {
	close(h.stopChan)
}

func (h *EnvironmentHealthMonitor) AddEnvironment(env *IsolatedEnvironment) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.environments[env.ID] = env
}

func (h *EnvironmentHealthMonitor) RemoveEnvironment(envID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.environments, envID)
}

func (h *EnvironmentHealthMonitor) performHealthChecks() {
	h.mutex.RLock()
	environments := make([]*IsolatedEnvironment, 0, len(h.environments))
	for _, env := range h.environments {
		environments = append(environments, env)
	}
	h.mutex.RUnlock()

	for _, env := range environments {
		h.checkEnvironmentHealth(env)
	}
}

func (h *EnvironmentHealthMonitor) checkEnvironmentHealth(env *IsolatedEnvironment) {
	env.mutex.Lock()
	defer env.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthy := true
	var healthIssues []string

	// Check database container health
	if env.ContainerID != "" {
		inspect, err := h.docker.ContainerInspect(ctx, env.ContainerID)
		if err != nil {
			healthy = false
			healthIssues = append(healthIssues, fmt.Sprintf("DB container inspect failed: %v", err))
		} else if !inspect.State.Running {
			healthy = false
			healthIssues = append(healthIssues, "DB container not running")
		} else if inspect.State.Health != nil && inspect.State.Health.Status != "healthy" {
			healthy = false
			healthIssues = append(healthIssues, fmt.Sprintf("DB container unhealthy: %s", inspect.State.Health.Status))
		}
	}

	// Check cache container health
	if env.CacheContainerID != "" {
		inspect, err := h.docker.ContainerInspect(ctx, env.CacheContainerID)
		if err != nil {
			healthy = false
			healthIssues = append(healthIssues, fmt.Sprintf("Cache container inspect failed: %v", err))
		} else if !inspect.State.Running {
			healthy = false
			healthIssues = append(healthIssues, "Cache container not running")
		} else if inspect.State.Health != nil && inspect.State.Health.Status != "healthy" {
			healthy = false
			healthIssues = append(healthIssues, fmt.Sprintf("Cache container unhealthy: %s", inspect.State.Health.Status))
		}
	}

	// Test database connectivity
	if env.DatabaseURL != "" {
		db, err := sql.Open("postgres", env.DatabaseURL)
		if err == nil {
			if err := db.Ping(); err != nil {
				healthy = false
				healthIssues = append(healthIssues, fmt.Sprintf("DB ping failed: %v", err))
			}
			db.Close()
		} else {
			healthy = false
			healthIssues = append(healthIssues, fmt.Sprintf("DB connection failed: %v", err))
		}
	}

	env.LastHealthCheck = time.Now()
	if healthy {
		env.HealthStatus = "healthy"
	} else {
		env.HealthStatus = fmt.Sprintf("unhealthy: %v", healthIssues)
		log.Printf("Environment %s health check failed: %v", env.ID, healthIssues)
	}
}