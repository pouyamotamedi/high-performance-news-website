# Synthetic Monitoring System

This package implements a comprehensive synthetic monitoring system for the news website, providing continuous validation of user journeys, performance, SEO compliance, accessibility, and mobile experience.

## Features

### User Journey Monitoring
- **Homepage Load Testing**: Validates homepage loading and core metrics
- **Article View Testing**: Tests article viewing functionality and SEO elements
- **Category Browse Testing**: Validates category page functionality
- **Search Testing**: Comprehensive search functionality validation
- **Mobile Navigation**: Tests mobile-specific navigation patterns

### Article Publishing Workflow Monitoring
- **API Testing**: Validates article creation API endpoints
- **Validation Testing**: Tests article validation processes
- **SEO Generation**: Monitors SEO metadata generation
- **Static Generation**: Validates static file generation

### Search Functionality Testing
- **Basic Search**: Tests core search functionality
- **Advanced Search**: Validates filtering and advanced search features
- **Search Suggestions**: Tests autocomplete and suggestion features
- **Multilingual Search**: Validates search in different languages

### Admin Panel Workflow Monitoring
- **Admin Authentication**: Tests admin login and dashboard access
- **Article Management**: Validates article management interface
- **User Management**: Tests user administration features
- **System Monitoring**: Validates monitoring dashboard functionality
- **Content Moderation**: Tests content moderation tools

### Continuous Validation Monitoring
- **SEO Compliance**: Continuous validation of schema markup, meta tags, canonical URLs
- **Performance Regression Detection**: Monitors for performance degradation
- **Accessibility Compliance**: WCAG 2.1 AA compliance validation
- **Mobile Experience**: Responsive design and mobile usability testing

## Architecture

```
SyntheticMonitoringService
├── SyntheticMonitor (Core monitoring engine)
│   ├── TestScheduler (Manages scheduled tests)
│   ├── ResultStore (Stores test results)
│   └── AlertManager (Handles alerting)
├── ContinuousValidator (Continuous validation)
│   ├── SEOComplianceValidator
│   ├── PerformanceValidator
│   ├── AccessibilityValidator
│   └── MobileExperienceValidator
└── HTTP Server (API and Web UI)
```

## Usage

### Running the Synthetic Monitor

```bash
# Basic usage
go run cmd/synthetic-monitor/main.go -url http://localhost:8080

# With custom configuration
go run cmd/synthetic-monitor/main.go \
  -url http://localhost:8080 \
  -port 9090 \
  -webhook https://hooks.slack.com/your-webhook \
  -api=true \
  -ui=true
```

### Configuration Options

- `-url`: Base URL to monitor (default: http://localhost:8080)
- `-port`: Port for monitoring dashboard (default: 9090)
- `-api`: Enable API endpoints (default: true)
- `-ui`: Enable web UI (default: true)
- `-webhook`: Webhook URL for alerts

### API Endpoints

- `GET /api/results?limit=50` - Get latest test results
- `GET /api/summary?test=homepage_load&duration=24h` - Get test summary
- `GET /api/jobs` - Get scheduled job status
- `GET /api/alerts?limit=50` - Get alert history
- `GET /api/health` - Health check
- `POST /api/test?test=critical_user_journeys` - Trigger manual test

### Web UI

Access the web dashboard at `http://localhost:9090` to view:
- Real-time test results
- Performance trends
- Alert history
- Job status
- Manual test triggers

## Test Types

### Critical User Journeys (Every 5 minutes)
- Homepage load validation
- Article view functionality
- Category browsing
- Basic search functionality
- Mobile navigation

### Article Publishing Workflow (Every 10 minutes)
- Article creation API testing
- Content validation processes
- SEO metadata generation
- Static file generation

### Search Functionality (Every 15 minutes)
- Basic search validation
- Advanced search features
- Search suggestions/autocomplete
- Search filters
- Multilingual search support

### Admin Panel Workflow (Every 30 minutes)
- Admin authentication
- Article management interface
- User management features
- System monitoring dashboard
- Content moderation tools

### Continuous Validation
- **SEO Compliance** (Every 30 minutes): Schema markup, meta tags, canonical URLs, sitemaps
- **Performance Regression** (Every 15 minutes): Page load times, API response times, database performance
- **Accessibility Compliance** (Every 2 hours): WCAG 2.1 AA compliance, keyboard navigation, screen reader support
- **Mobile Experience** (Every hour): Responsive design, touch interactions, mobile performance

## Alerting

The system supports multiple alerting mechanisms:

### Alert Levels
- **Critical**: System failures, security issues
- **High**: Feature failures, performance regressions
- **Medium**: Accessibility issues, mobile problems
- **Low**: Minor compliance issues

### Alert Channels
- **Webhook**: Send alerts to Slack, Teams, or custom endpoints
- **Email**: SMTP-based email notifications
- **Console**: Log-based alerts for development

### Rate Limiting
Alerts are rate-limited to prevent spam:
- Same alert type limited to once per 15 minutes
- Alert history maintained for analysis
- Configurable alert thresholds

## Performance Baselines

The system automatically establishes and maintains performance baselines:

- **Automatic Baseline Updates**: Based on recent performance data
- **Regression Detection**: Alerts when performance degrades beyond thresholds
- **Trend Analysis**: Tracks performance trends over time
- **Capacity Planning**: Provides insights for resource planning

## Accessibility Validation

WCAG 2.1 AA compliance validation includes:

- **Images**: Alt text validation
- **Headings**: Proper heading structure
- **Forms**: Label associations
- **Keyboard Navigation**: Tab order and focus management
- **Color Contrast**: Sufficient contrast ratios
- **ARIA**: Proper ARIA usage

## Mobile Experience Validation

Mobile testing covers:

- **Responsive Design**: Layout adaptation across viewport sizes
- **Touch Targets**: Minimum 44px touch target sizes
- **Performance**: Mobile-specific performance metrics
- **Navigation**: Mobile navigation patterns
- **Viewport**: Proper viewport meta tag configuration

## Extending the System

### Adding Custom Tests

```go
// Add custom test to monitor
func (s *SyntheticMonitor) customTest(ctx context.Context) MonitoringResult {
    start := time.Now()
    result := MonitoringResult{
        TestName:    "custom_test",
        TestType:    "custom",
        Timestamp:   start,
        UserJourney: "Custom Test",
        Metrics:     make(map[string]float64),
    }
    
    // Implement your test logic here
    
    result.Duration = time.Since(start)
    return result
}

// Schedule the custom test
s.testScheduler.Schedule("custom_test", 10*time.Minute, s.customTest)
```

### Adding Custom Validators

```go
type CustomValidator struct {
    // Your validator configuration
}

func (c *CustomValidator) Validate(page *rod.Page) []ValidationResult {
    // Implement your validation logic
    return []ValidationResult{}
}
```

### Custom Alert Handlers

```go
type CustomAlertManager struct {
    // Your alert configuration
}

func (c *CustomAlertManager) SendAlert(level AlertLevel, message string, result MonitoringResult) error {
    // Implement your custom alerting logic
    return nil
}
```

## Monitoring Best Practices

1. **Test Isolation**: Each test runs in isolation to prevent interference
2. **Resource Management**: Automatic cleanup of browser resources
3. **Error Handling**: Comprehensive error handling and recovery
4. **Performance**: Efficient test execution with minimal resource usage
5. **Reliability**: Retry mechanisms and fallback procedures
6. **Observability**: Detailed logging and metrics collection

## Troubleshooting

### Common Issues

1. **Browser Launch Failures**
   - Ensure Chrome/Chromium is installed
   - Check system permissions for headless browser execution

2. **Network Timeouts**
   - Verify target URL accessibility
   - Check network connectivity and firewall settings

3. **High Memory Usage**
   - Monitor browser resource cleanup
   - Adjust test frequency if needed

4. **False Positives**
   - Review test thresholds and baselines
   - Implement proper wait conditions for dynamic content

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export LOG_LEVEL=debug
go run cmd/synthetic-monitor/main.go -url http://localhost:8080
```

## Dependencies

- **go-rod/rod**: Browser automation
- **Standard Library**: HTTP server, JSON handling, context management

## License

This synthetic monitoring system is part of the comprehensive testing framework for the news website project.