# Docker-Based Environment Isolation System

## Overview

The Docker-based environment isolation system provides isolated test environments for comprehensive testing. Each environment includes dedicated database and cache containers with automatic resource management and health monitoring.

## Features

### 1. Isolated Test Environments
- **Dedicated Containers**: Each environment gets its own PostgreSQL and Redis containers
- **Resource Isolation**: Memory, CPU, and disk space limits per environment
- **Network Isolation**: Containers run in isolated Docker networks
- **Data Isolation**: Each environment has its own database and cache instance

### 2. Resource Management
- **Resource Pool**: Centralized resource allocation and tracking
- **Automatic Limits**: Prevents resource exhaustion with configurable limits
- **Utilization Monitoring**: Real-time resource utilization tracking
- **Cleanup**: Automatic resource release when environments are destroyed

### 3. Health Monitoring
- **Container Health**: Monitors Docker container health status
- **Database Connectivity**: Tests database connections and responsiveness
- **Cache Availability**: Verifies cache service availability
- **Automatic Recovery**: Detects and reports unhealthy environments

### 4. Lifecycle Management
- **Asynchronous Creation**: Non-blocking environment creation
- **Status Tracking**: Real-time status updates during creation
- **Graceful Cleanup**: Proper container shutdown and resource cleanup
- **Bulk Operations**: Support for managing multiple environments

## Architecture

```
TestEnvironmentManager
├── ResourcePool (manages resource allocation)
├── EnvironmentHealthMonitor (monitors container health)
└── IsolatedEnvironment[]
    ├── PostgreSQL Container (database)
    ├── Redis Container (cache)
    └── Resource Allocation
```

## Usage Examples

### Basic Environment Creation

```go
// Create environment manager
manager, err := NewTestEnvironmentManager()
if err != nil {
    log.Fatal(err)
}
defer manager.Shutdown()

// Create isolated environment
env, err := manager.CreateIsolatedEnvironment("integration-tests")
if err != nil {
    log.Fatal(err)
}

// Wait for environment to be ready
for {
    currentEnv, _ := manager.GetEnvironment(env.ID)
    if currentEnv.Status == EnvironmentStatusReady {
        break
    }
    time.Sleep(5 * time.Second)
}

// Use the environment
db, err := sql.Open("postgres", env.DatabaseURL)
// ... run tests ...

// Cleanup
manager.CleanupEnvironment(env.ID)
```

### Resource Monitoring

```go
// Check resource utilization
memUtil, cpuUtil, envUtil := manager.resourcePool.GetUtilization()
fmt.Printf("Memory: %.1f%%, CPU: %.1f%%, Environments: %.1f%%\n", 
    memUtil*100, cpuUtil*100, envUtil*100)

// List all environments
environments := manager.ListEnvironments()
for _, env := range environments {
    fmt.Printf("Environment %s: %s (%s)\n", 
        env.ID, env.TestSuite, env.Status)
}
```

### Health Monitoring

```go
// Get environment health status
env, err := manager.GetEnvironment(envID)
if err == nil {
    fmt.Printf("Health: %s, Last Check: %s\n", 
        env.HealthStatus, env.LastHealthCheck)
}
```

## Configuration

### Default Resource Limits
- **Memory per Environment**: 512MB
- **CPU per Environment**: 50% of one core
- **Maximum Environments**: 10
- **Total Memory Limit**: 8GB
- **Total CPU Limit**: 4 cores

### Container Images
- **Database**: `postgres:15-alpine`
- **Cache**: `redis:7-alpine`

### Health Check Settings
- **Check Interval**: 30 seconds
- **Check Timeout**: 10 seconds
- **Container Start Timeout**: 60 seconds

## Environment States

1. **Creating**: Environment is being set up
2. **Ready**: Environment is ready for use
3. **Failed**: Environment creation failed
4. **Cleaning**: Environment is being cleaned up
5. **Destroyed**: Environment has been removed

## Resource Pool Management

The resource pool ensures efficient resource utilization:

- **Allocation Tracking**: Tracks memory, CPU, and environment slots
- **Limit Enforcement**: Prevents over-allocation of resources
- **Utilization Metrics**: Provides real-time utilization percentages
- **Automatic Cleanup**: Releases resources when environments are destroyed

## Health Monitoring

The health monitor continuously checks:

- **Container Status**: Verifies containers are running
- **Database Connectivity**: Tests database connections
- **Cache Availability**: Verifies cache service responses
- **Resource Usage**: Monitors resource consumption

## Error Handling

The system handles various error conditions:

- **Docker Unavailable**: Graceful degradation when Docker is not available
- **Image Pull Failures**: Fallback strategies for missing images
- **Network Issues**: Automatic retry and fallback mechanisms
- **Resource Exhaustion**: Queue management and resource limits

## Testing

### Unit Tests
- Resource pool allocation and limits
- Utilization calculations
- Concurrent access safety
- Negative value protection

### Integration Tests
- Full environment lifecycle
- Database connectivity
- Container isolation
- Resource management
- Health monitoring

### Prerequisites for Integration Tests
- Docker daemon running
- Required container images available:
  - `postgres:15-alpine`
  - `redis:7-alpine`

## Performance Considerations

- **Parallel Creation**: Supports creating multiple environments concurrently
- **Resource Pooling**: Efficient resource allocation and reuse
- **Lazy Cleanup**: Deferred cleanup to avoid blocking operations
- **Health Check Batching**: Efficient batch health checking

## Security

- **Container Isolation**: Each environment runs in isolated containers
- **Network Segmentation**: Isolated Docker networks per environment
- **Resource Limits**: Prevents resource exhaustion attacks
- **Credential Management**: Secure handling of database credentials

## Troubleshooting

### Common Issues

1. **Docker Not Available**
   - Ensure Docker daemon is running
   - Check Docker permissions
   - Verify Docker API version compatibility

2. **Container Creation Failures**
   - Check available disk space
   - Verify container images are available
   - Check Docker network configuration

3. **Resource Exhaustion**
   - Monitor resource utilization
   - Adjust resource limits in configuration
   - Clean up unused environments

4. **Health Check Failures**
   - Check container logs
   - Verify network connectivity
   - Increase health check timeouts

### Debugging

Enable verbose logging:
```go
config := DefaultEnvironmentConfig()
config.LogLevel = "debug"
config.LogContainerOutput = true
```

Check environment status:
```go
env, err := manager.GetEnvironment(envID)
if err == nil {
    fmt.Printf("Status: %s, Error: %s\n", env.Status, env.ErrorMessage)
}
```

## Future Enhancements

- **Template Support**: Pre-configured environment templates
- **Persistent Volumes**: Support for persistent data storage
- **Multi-Host Support**: Distribute environments across multiple Docker hosts
- **Advanced Networking**: Custom network configurations and service discovery
- **Metrics Integration**: Integration with monitoring systems like Prometheus
- **Auto-scaling**: Automatic environment scaling based on demand