# Memory-Aware Job Queue System

A high-performance, memory-aware job queue system designed for the high-performance news website. This system can handle massive workloads while monitoring memory usage and implementing priority-based job processing.

## Features

- **Memory Pressure Monitoring**: Monitors system memory usage with configurable 28GB threshold
- **Priority-Based Processing**: Three priority levels (High, Medium, Low) with intelligent scheduling
- **Worker Pool Management**: Configurable worker pool with graceful shutdown and error recovery
- **Job Types**: Built-in handlers for static generation, image processing, search indexing, and notifications
- **Retry Logic**: Automatic retry with exponential backoff for failed jobs
- **Comprehensive Testing**: Full test coverage with integration tests
- **Performance Optimized**: Designed to handle 50K+ articles per day

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Job Queue     │    │   Worker Pool    │    │  Memory Monitor │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ High Prio   │ │    │ │   Worker 1   │ │    │ │ 28GB Limit  │ │
│ │ (1000 jobs) │ │    │ │   Worker 2   │ │    │ │ Monitoring  │ │
│ └─────────────┘ │    │ │   Worker N   │ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ └──────────────┘ │    └─────────────────┘
│ │ Medium Prio │ │    └──────────────────┘              │
│ │ (5000 jobs) │ │                                      │
│ └─────────────┘ │    ┌──────────────────┐              │
│ ┌─────────────┐ │    │   Job Handlers   │              │
│ │ Low Prio    │ │    │                  │              │
│ │ (10000 jobs)│ │    │ • Static Gen     │              │
│ └─────────────┘ │    │ • Image Proc     │              │
└─────────────────┘    │ • Search Index   │              │
                       │ • Notifications  │              │
                       └──────────────────┘              │
                                                         │
                       ┌──────────────────┐              │
                       │ Memory Pressure  │◄─────────────┘
                       │ Control Logic    │
                       │                  │
                       │ • Block low prio │
                       │ • Allow high prio│
                       └──────────────────┘
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "high-performance-news-website/internal/queue"
)

func main() {
    // Create queue system with default configuration
    system, err := queue.NewQueueSystem(nil)
    if err != nil {
        panic(err)
    }

    // Start the system
    ctx := context.Background()
    if err := system.Start(ctx); err != nil {
        panic(err)
    }
    defer system.Stop()

    // Enqueue jobs
    system.EnqueueStaticGeneration(123, "article", queue.PriorityHigh)
    system.EnqueueImageProcessing("/path/to/image.jpg", []string{"webp", "avif"}, queue.PriorityMedium)
    system.EnqueueSearchIndexing(123, "index", queue.PriorityMedium)
    system.EnqueueNotification("email", []interface{}{"user@example.com"}, "Hello!", queue.PriorityLow)
}
```

### Custom Configuration

```go
config := &queue.QueueConfig{
    WorkerCount:        8,                            // 8 workers
    MemoryThreshold:    16 * 1024 * 1024 * 1024,     // 16GB threshold
    HighPrioritySize:   500,                          // High priority queue size
    MediumPrioritySize: 2000,                         // Medium priority queue size
    LowPrioritySize:    5000,                         // Low priority queue size
}

system, err := queue.NewQueueSystem(config)
```

## Job Types

### 1. Static Generation Jobs
Generate static HTML files for articles, homepage, category pages, etc.

```go
err := system.EnqueueStaticGeneration(articleID, "article", queue.PriorityHigh)
```

**Payload:**
- `article_id`: ID of the article to generate
- `page_type`: Type of page ("article", "homepage", "category", etc.)

### 2. Image Processing Jobs
Process images into multiple formats (WebP, AVIF, JPEG) with different sizes.

```go
formats := []string{"webp", "avif", "jpeg"}
err := system.EnqueueImageProcessing("/path/to/image.jpg", formats, queue.PriorityMedium)
```

**Payload:**
- `image_path`: Path to the source image
- `formats`: Array of target formats to generate

### 3. Search Indexing Jobs
Update search index with new, updated, or deleted content.

```go
err := system.EnqueueSearchIndexing(documentID, "index", queue.PriorityMedium)
```

**Payload:**
- `document_id`: ID of the document to index
- `action`: Action to perform ("index", "update", "delete")

### 4. Notification Jobs
Send notifications via email, push notifications, SMS, etc.

```go
recipients := []interface{}{"user1@example.com", "user2@example.com"}
err := system.EnqueueNotification("email", recipients, "Message", queue.PriorityLow)
```

**Payload:**
- `type`: Notification type ("email", "push", "sms")
- `recipients`: Array of recipient addresses
- `message`: Message content

## Priority Levels

### High Priority (PriorityHigh = 2)
- **Use for**: Critical operations that must complete quickly
- **Examples**: Static generation for breaking news, critical system updates
- **Queue Size**: 1,000 jobs (default)
- **Memory Pressure**: Always allowed, even under memory pressure

### Medium Priority (PriorityMedium = 1)
- **Use for**: Important but not critical operations
- **Examples**: Image processing, search indexing, routine updates
- **Queue Size**: 5,000 jobs (default)
- **Memory Pressure**: Blocked when memory pressure detected

### Low Priority (PriorityLow = 0)
- **Use for**: Background tasks that can be delayed
- **Examples**: Notifications, cleanup tasks, analytics
- **Queue Size**: 10,000 jobs (default)
- **Memory Pressure**: Blocked when memory pressure detected

## Memory Pressure Handling

The system monitors memory usage and implements intelligent job scheduling:

1. **Normal Operation**: All priority levels are processed
2. **Memory Pressure Detected**: Only high priority jobs are accepted and processed
3. **Threshold**: Default 28GB, configurable per deployment
4. **Monitoring Frequency**: Every 30 seconds
5. **Recovery**: Automatic when memory usage drops below threshold

## Error Handling and Retry Logic

### Automatic Retries
- **Default Max Attempts**: 3 (configurable per job)
- **Retry Strategy**: Exponential backoff (attempt² seconds)
- **Failure Handling**: Jobs are permanently failed after max attempts

### Error Recovery
- **Worker Panics**: Automatically recovered with logging
- **Context Cancellation**: Graceful handling of timeouts and shutdowns
- **Job Timeouts**: 30-minute timeout per job (configurable)

## Performance Characteristics

### Throughput
- **Target**: 50,000+ articles per day
- **Peak Rate**: 1,000 articles per minute
- **Concurrent Jobs**: Limited by worker count and memory

### Latency
- **Job Scheduling**: < 1ms
- **Queue Operations**: < 10ms
- **Memory Check**: < 5ms

### Resource Usage
- **Memory**: Monitored and controlled
- **CPU**: Scales with worker count
- **I/O**: Optimized for high throughput

## Monitoring and Statistics

### System Health Check
```go
healthy := system.IsHealthy()
```

### Detailed Statistics
```go
stats, err := system.GetStats()
// Returns comprehensive system statistics including:
// - Queue sizes by priority
// - Worker statistics
// - Memory usage and pressure status
// - Handler registration count
```

### Key Metrics
- Queue sizes by priority level
- Active worker count and job processing
- Memory usage vs threshold
- Job success/failure rates
- Processing latency

## Testing

The system includes comprehensive tests:

```bash
# Run all tests
go test ./internal/queue -v

# Run specific test categories
go test ./internal/queue -run TestMemoryMonitor -v
go test ./internal/queue -run TestJobHandlerRegistry -v
go test ./internal/queue -run TestQueueSystem -v
```

### Test Coverage
- **Unit Tests**: All components individually tested
- **Integration Tests**: End-to-end job processing
- **Performance Tests**: Load testing and benchmarks
- **Error Scenarios**: Failure modes and recovery

## Demo Application

Run the demo to see the system in action:

```bash
go run cmd/queue-demo/main.go
```

The demo shows:
- Job enqueueing and processing
- Priority-based scheduling
- Memory pressure handling
- System statistics and health monitoring

## Integration with News Website

### Article Publishing Workflow
1. **Article Created** → Enqueue static generation (High Priority)
2. **Images Uploaded** → Enqueue image processing (Medium Priority)  
3. **Content Published** → Enqueue search indexing (Medium Priority)
4. **Notifications** → Enqueue email/push notifications (Low Priority)

### Performance Optimization
- Static HTML generation for maximum speed
- Image processing for multiple formats and sizes
- Search index updates for discoverability
- Background notifications without blocking

### Scalability
- Horizontal scaling through worker count adjustment
- Memory-aware processing prevents system overload
- Priority-based scheduling ensures critical tasks complete first

## Configuration Options

```go
type QueueConfig struct {
    WorkerCount        int    // Number of worker goroutines
    MemoryThreshold    uint64 // Memory threshold in bytes
    HighPrioritySize   int    // High priority queue capacity
    MediumPrioritySize int    // Medium priority queue capacity  
    LowPrioritySize    int    // Low priority queue capacity
}
```

### Recommended Settings

#### Development
```go
config := &QueueConfig{
    WorkerCount:        4,
    MemoryThreshold:    8 * 1024 * 1024 * 1024,  // 8GB
    HighPrioritySize:   100,
    MediumPrioritySize: 500,
    LowPrioritySize:    1000,
}
```

#### Production
```go
config := &QueueConfig{
    WorkerCount:        16,
    MemoryThreshold:    28 * 1024 * 1024 * 1024, // 28GB
    HighPrioritySize:   1000,
    MediumPrioritySize: 5000,
    LowPrioritySize:    10000,
}
```

## Best Practices

### Job Design
- Keep jobs idempotent (safe to retry)
- Use appropriate priority levels
- Include sufficient context in job payload
- Handle errors gracefully

### Performance
- Monitor queue sizes and adjust worker count
- Use batch operations when possible
- Implement circuit breakers for external services
- Monitor memory usage trends

### Reliability
- Implement proper error handling
- Use timeouts for external operations
- Log job failures for debugging
- Monitor system health continuously

## Troubleshooting

### Common Issues

#### High Memory Usage
- Check for memory leaks in job handlers
- Reduce worker count if necessary
- Implement job payload size limits
- Monitor garbage collection

#### Slow Job Processing
- Increase worker count
- Optimize job handler performance
- Check for blocking operations
- Monitor external service latency

#### Queue Backlog
- Increase queue sizes if memory allows
- Add more workers
- Optimize job processing logic
- Implement job prioritization

### Debugging

Enable detailed logging:
```go
// Logs are automatically written to stdout
// Monitor job processing, errors, and system events
```

Check system statistics:
```go
stats, _ := system.GetStats()
fmt.Printf("Stats: %+v\n", stats)
```

## Future Enhancements

- **Persistent Queue**: Database-backed job storage for durability
- **Distributed Processing**: Multi-server job distribution
- **Job Scheduling**: Cron-like scheduled job execution
- **Metrics Export**: Prometheus metrics integration
- **Web Dashboard**: Real-time monitoring interface
- **Job Dependencies**: Support for job chains and dependencies