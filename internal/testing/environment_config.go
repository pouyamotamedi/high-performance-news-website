package testing

import (
	"fmt"
	"time"
)

// EnvironmentConfig holds configuration for test environment management
type EnvironmentConfig struct {
	// Docker configuration
	DockerHost     string `json:"docker_host" yaml:"docker_host"`
	DockerVersion  string `json:"docker_version" yaml:"docker_version"`
	NetworkName    string `json:"network_name" yaml:"network_name"`
	
	// Resource limits
	MaxMemoryPerEnv    int64 `json:"max_memory_per_env" yaml:"max_memory_per_env"`       // bytes
	MaxCPUPerEnv       int64 `json:"max_cpu_per_env" yaml:"max_cpu_per_env"`             // CPU quota
	MaxDiskPerEnv      int64 `json:"max_disk_per_env" yaml:"max_disk_per_env"`           // bytes
	MaxEnvironments    int   `json:"max_environments" yaml:"max_environments"`
	TotalMemoryLimit   int64 `json:"total_memory_limit" yaml:"total_memory_limit"`       // bytes
	TotalCPULimit      int64 `json:"total_cpu_limit" yaml:"total_cpu_limit"`             // CPU quota
	
	// Container images
	PostgreSQLImage    string `json:"postgresql_image" yaml:"postgresql_image"`
	RedisImage         string `json:"redis_image" yaml:"redis_image"`
	
	// Health monitoring
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout" yaml:"health_check_timeout"`
	ContainerStartTimeout time.Duration `json:"container_start_timeout" yaml:"container_start_timeout"`
	
	// Cleanup settings
	AutoCleanupAfter   time.Duration `json:"auto_cleanup_after" yaml:"auto_cleanup_after"`
	CleanupOnShutdown  bool          `json:"cleanup_on_shutdown" yaml:"cleanup_on_shutdown"`
	
	// Performance settings
	ParallelCreation   int  `json:"parallel_creation" yaml:"parallel_creation"`
	EnableResourcePool bool `json:"enable_resource_pool" yaml:"enable_resource_pool"`
	
	// Logging
	LogLevel           string `json:"log_level" yaml:"log_level"`
	LogContainerOutput bool   `json:"log_container_output" yaml:"log_container_output"`
}

// DefaultEnvironmentConfig returns the default configuration
func DefaultEnvironmentConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		DockerHost:    "",  // Use default Docker host
		DockerVersion: "",  // Use API version negotiation
		NetworkName:   "test-network",
		
		MaxMemoryPerEnv:    512 * 1024 * 1024, // 512MB per environment
		MaxCPUPerEnv:       50000,              // 50% CPU per environment
		MaxDiskPerEnv:      1024 * 1024 * 1024, // 1GB disk per environment
		MaxEnvironments:    10,
		TotalMemoryLimit:   8 * 1024 * 1024 * 1024, // 8GB total
		TotalCPULimit:      400000,                   // 4 CPU cores total
		
		PostgreSQLImage:    "postgres:15-alpine",
		RedisImage:         "redis:7-alpine",
		
		HealthCheckInterval:   30 * time.Second,
		HealthCheckTimeout:    10 * time.Second,
		ContainerStartTimeout: 60 * time.Second,
		
		AutoCleanupAfter:  2 * time.Hour, // Auto cleanup after 2 hours
		CleanupOnShutdown: true,
		
		ParallelCreation:   3, // Create up to 3 environments in parallel
		EnableResourcePool: true,
		
		LogLevel:           "info",
		LogContainerOutput: false,
	}
}

// Validate checks if the configuration is valid
func (c *EnvironmentConfig) Validate() error {
	if c.MaxMemoryPerEnv <= 0 {
		return fmt.Errorf("max_memory_per_env must be positive")
	}
	
	if c.MaxCPUPerEnv <= 0 {
		return fmt.Errorf("max_cpu_per_env must be positive")
	}
	
	if c.MaxEnvironments <= 0 {
		return fmt.Errorf("max_environments must be positive")
	}
	
	if c.TotalMemoryLimit < c.MaxMemoryPerEnv {
		return fmt.Errorf("total_memory_limit must be at least max_memory_per_env")
	}
	
	if c.TotalCPULimit < c.MaxCPUPerEnv {
		return fmt.Errorf("total_cpu_limit must be at least max_cpu_per_env")
	}
	
	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be positive")
	}
	
	if c.HealthCheckTimeout <= 0 {
		return fmt.Errorf("health_check_timeout must be positive")
	}
	
	if c.ContainerStartTimeout <= 0 {
		return fmt.Errorf("container_start_timeout must be positive")
	}
	
	if c.ParallelCreation <= 0 {
		return fmt.Errorf("parallel_creation must be positive")
	}
	
	return nil
}

// GetResourceLimits returns the resource limits for a single environment
func (c *EnvironmentConfig) GetResourceLimits() ResourceAllocation {
	return ResourceAllocation{
		Memory:    c.MaxMemoryPerEnv,
		CPUQuota:  c.MaxCPUPerEnv,
		DiskSpace: c.MaxDiskPerEnv,
	}
}

// CanCreateEnvironment checks if a new environment can be created with current limits
func (c *EnvironmentConfig) CanCreateEnvironment(currentEnvs int, allocatedMemory, allocatedCPU int64) bool {
	if currentEnvs >= c.MaxEnvironments {
		return false
	}
	
	if allocatedMemory + c.MaxMemoryPerEnv > c.TotalMemoryLimit {
		return false
	}
	
	if allocatedCPU + c.MaxCPUPerEnv > c.TotalCPULimit {
		return false
	}
	
	return true
}

// GetDatabaseConfig returns database-specific configuration
func (c *EnvironmentConfig) GetDatabaseConfig(envID string) map[string]string {
	return map[string]string{
		"POSTGRES_DB":              "test_" + envID,
		"POSTGRES_USER":            "testuser",
		"POSTGRES_PASSWORD":        "testpass",
		"POSTGRES_INITDB_ARGS":     "--auth-host=trust",
		"POSTGRES_MAX_CONNECTIONS": "100",
		"POSTGRES_SHARED_BUFFERS":  "128MB",
	}
}

// GetCacheConfig returns cache-specific configuration
func (c *EnvironmentConfig) GetCacheConfig(envID string) map[string]string {
	return map[string]string{
		"REDIS_MAXMEMORY":        "256mb",
		"REDIS_MAXMEMORY_POLICY": "allkeys-lru",
		"REDIS_SAVE":             "", // Disable persistence for tests
		"REDIS_APPENDONLY":       "no",
	}
}

// EnvironmentTemplate defines a template for creating environments
type EnvironmentTemplate struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Services    []ServiceTemplate `json:"services" yaml:"services"`
	Networks    []string          `json:"networks" yaml:"networks"`
	Volumes     []VolumeTemplate  `json:"volumes" yaml:"volumes"`
}

// ServiceTemplate defines a service within an environment template
type ServiceTemplate struct {
	Name         string            `json:"name" yaml:"name"`
	Image        string            `json:"image" yaml:"image"`
	Environment  map[string]string `json:"environment" yaml:"environment"`
	Ports        []PortMapping     `json:"ports" yaml:"ports"`
	HealthCheck  *HealthCheck      `json:"health_check" yaml:"health_check"`
	Resources    ResourceAllocation `json:"resources" yaml:"resources"`
	Dependencies []string          `json:"dependencies" yaml:"dependencies"`
}

// PortMapping defines port mapping for a service
type PortMapping struct {
	ContainerPort int    `json:"container_port" yaml:"container_port"`
	HostPort      int    `json:"host_port" yaml:"host_port"`
	Protocol      string `json:"protocol" yaml:"protocol"`
}

// HealthCheck defines health check configuration
type HealthCheck struct {
	Test     []string      `json:"test" yaml:"test"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
	Retries  int           `json:"retries" yaml:"retries"`
}

// VolumeTemplate defines a volume template
type VolumeTemplate struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"` // "bind", "volume", "tmpfs"
	Source     string `json:"source" yaml:"source"`
	Target     string `json:"target" yaml:"target"`
	ReadOnly   bool   `json:"read_only" yaml:"read_only"`
}

// GetDefaultTemplates returns default environment templates
func GetDefaultTemplates() []EnvironmentTemplate {
	return []EnvironmentTemplate{
		{
			Name:        "basic-web-app",
			Description: "Basic web application with database and cache",
			Services: []ServiceTemplate{
				{
					Name:  "database",
					Image: "postgres:15-alpine",
					Environment: map[string]string{
						"POSTGRES_DB":       "testdb",
						"POSTGRES_USER":     "testuser",
						"POSTGRES_PASSWORD": "testpass",
					},
					Ports: []PortMapping{
						{ContainerPort: 5432, Protocol: "tcp"},
					},
					HealthCheck: &HealthCheck{
						Test:     []string{"CMD-SHELL", "pg_isready -U testuser -d testdb"},
						Interval: 10 * time.Second,
						Timeout:  5 * time.Second,
						Retries:  5,
					},
					Resources: ResourceAllocation{
						Memory:   256 * 1024 * 1024, // 256MB
						CPUQuota: 25000,              // 25% CPU
					},
				},
				{
					Name:  "cache",
					Image: "redis:7-alpine",
					Ports: []PortMapping{
						{ContainerPort: 6379, Protocol: "tcp"},
					},
					HealthCheck: &HealthCheck{
						Test:     []string{"CMD", "redis-cli", "ping"},
						Interval: 10 * time.Second,
						Timeout:  3 * time.Second,
						Retries:  3,
					},
					Resources: ResourceAllocation{
						Memory:   256 * 1024 * 1024, // 256MB
						CPUQuota: 25000,              // 25% CPU
					},
				},
			},
		},
		{
			Name:        "microservices",
			Description: "Microservices environment with multiple databases",
			Services: []ServiceTemplate{
				{
					Name:  "postgres",
					Image: "postgres:15-alpine",
					Environment: map[string]string{
						"POSTGRES_DB":       "maindb",
						"POSTGRES_USER":     "testuser",
						"POSTGRES_PASSWORD": "testpass",
					},
					Resources: ResourceAllocation{
						Memory:   512 * 1024 * 1024, // 512MB
						CPUQuota: 50000,              // 50% CPU
					},
				},
				{
					Name:  "redis",
					Image: "redis:7-alpine",
					Resources: ResourceAllocation{
						Memory:   256 * 1024 * 1024, // 256MB
						CPUQuota: 25000,              // 25% CPU
					},
				},
			},
		},
	}
}