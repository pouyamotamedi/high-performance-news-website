# Configuration and Settings Management System

## Overview

This document describes the implementation of Task 38: "Build configuration and settings management" for the high-performance news website. The system provides dynamic configuration management with hot reloading, feature flags, validation, and rollback capabilities.

## Features Implemented

### 1. Dynamic Configuration System with Hot Reloading

- **Hot Reload**: Automatic configuration reloading from database every 30 seconds
- **Real-time Updates**: Configuration changes are immediately available across the application
- **Event Subscription**: Components can subscribe to configuration change events
- **Thread-Safe**: All operations are protected with read-write mutexes

### 2. Settings Management API and Admin Interface

- **RESTful API**: Complete CRUD operations for configurations
- **Category-based Organization**: Configurations grouped by categories (site, performance, features, etc.)
- **Type Safety**: Support for string, int, float, bool, JSON, and array types
- **Secret Management**: Sensitive configurations are hidden from non-admin users

### 3. Feature Flag System for Gradual Rollouts

- **Percentage Rollouts**: Control feature availability by percentage of users
- **Conditional Logic**: Feature flags with user role, IP range, and custom conditions
- **Context-aware Evaluation**: Feature flags evaluated based on user context
- **Real-time Toggle**: Enable/disable features without code deployment

### 4. Configuration Validation and Rollback

- **Validation Rules**: Min/max values, length constraints, regex patterns, and allowed options
- **Snapshot System**: Create configuration snapshots for rollback
- **History Tracking**: Complete audit trail of configuration changes
- **Rollback Capability**: Restore to any previous configuration state

### 5. Comprehensive Testing

- **Unit Tests**: Full test coverage for configuration service
- **Validation Tests**: Tests for all validation rules and edge cases
- **Rollback Tests**: Tests for snapshot creation and restoration
- **API Tests**: Tests for all configuration management endpoints

## File Structure

```
internal/
├── models/
│   └── configuration.go          # Configuration data models
├── services/
│   ├── config_service.go          # Main configuration service
│   ├── config_service_test.go     # Configuration service tests
│   ├── config_validation_test.go  # Validation tests
│   └── config_rollback_test.go    # Rollback functionality tests
├── api/
│   ├── configuration_handlers.go      # API handlers
│   └── configuration_handlers_test.go # API handler tests
└── routes.go                      # Route registration

migrations/
├── 034_create_configuration_tables.up.sql   # Database schema
└── 034_create_configuration_tables.down.sql # Schema rollback
```

## Database Schema

### Tables Created

1. **configurations**: Main configuration storage
   - Supports typed values (string, int, float, bool, JSON, array)
   - Category-based organization
   - Validation rules stored as JSONB
   - Secret flag for sensitive data

2. **feature_flags**: Feature flag management
   - Rollout configuration with percentage and conditions
   - Enable/disable toggle
   - Context-based evaluation rules

3. **configuration_history**: Audit trail
   - Tracks all configuration changes
   - Records who made changes and why
   - Supports rollback operations

4. **configuration_snapshots**: Snapshot system
   - Complete configuration state capture
   - Named snapshots with descriptions
   - Point-in-time restoration

## API Endpoints

### Configuration Management
- `GET /api/v1/admin/config` - Get all configurations
- `GET /api/v1/admin/config/:key` - Get specific configuration
- `PUT /api/v1/admin/config/:key` - Update configuration
- `DELETE /api/v1/admin/config/:key` - Delete configuration
- `POST /api/v1/admin/config/validate` - Validate all configurations
- `GET /api/v1/admin/config/categories/:category` - Get configurations by category

### Configuration Snapshots
- `GET /api/v1/admin/config/snapshots` - List all snapshots
- `POST /api/v1/admin/config/snapshots` - Create new snapshot
- `POST /api/v1/admin/config/snapshots/:id/restore` - Restore from snapshot

### Configuration History
- `GET /api/v1/admin/config/:key/history` - Get configuration change history

### Feature Flags
- `GET /api/v1/admin/feature-flags` - Get all feature flags
- `GET /api/v1/admin/feature-flags/:key` - Get specific feature flag
- `PUT /api/v1/admin/feature-flags/:key` - Update feature flag
- `POST /api/v1/admin/feature-flags` - Create feature flag
- `DELETE /api/v1/admin/feature-flags/:key` - Delete feature flag
- `POST /api/v1/admin/feature-flags/:key/check` - Check if feature is enabled

### Hot Reload
- `POST /api/v1/admin/config/reload` - Manually reload configuration from database

## Configuration Types and Validation

### Supported Types
- **string**: Text values with length and pattern validation
- **int**: Integer values with min/max validation
- **float**: Floating-point values with min/max validation
- **bool**: Boolean values (true/false, 1/0)
- **json**: Complex JSON objects
- **array**: Array of values

### Validation Rules
- **Required**: Field must have a value
- **MinLength/MaxLength**: String length constraints
- **MinValue/MaxValue**: Numeric range constraints
- **Pattern**: Regular expression validation
- **Options**: Allowed values list

## Feature Flag System

### Rollout Strategies
- **Percentage**: Enable for X% of users based on user ID hash
- **User Groups**: Enable for specific user roles or groups
- **IP Ranges**: Enable for specific IP address ranges
- **Custom Conditions**: Complex conditional logic

### Context Evaluation
Feature flags are evaluated based on context:
```json
{
  "user_id": 123,
  "user_role": "admin",
  "ip_address": "192.168.1.1",
  "custom_field": "value"
}
```

## Usage Examples

### Basic Configuration Management
```go
// Get configuration
value, err := configService.Get("site_name")

// Get typed configuration
cacheTTL, err := configService.GetTyped("cache_ttl") // returns int

// Update configuration with history
err := configService.SetWithContext(ctx, "site_name", "New Name", userID, "Rebranding")

// Get configurations by category
siteConfigs := configService.GetAllByCategory("site")
```

### Feature Flag Usage
```go
// Check if feature is enabled
context := map[string]interface{}{
    "user_id": userID,
    "user_role": userRole,
}
enabled := configService.IsFeatureEnabled("new_editor", context)

// Create feature flag with rollout
flag := &models.FeatureFlag{
    Key: "beta_feature",
    Name: "Beta Feature",
    Enabled: true,
    Rollout: &models.FeatureFlagRollout{
        Percentage: 25, // 25% rollout
    },
}
configService.SetFeatureFlag(flag)
```

### Configuration Snapshots
```go
// Create snapshot
snapshot, err := configService.CreateSnapshot("pre_deployment", "Before major update", userID)

// Restore from snapshot
err := configService.RestoreSnapshot(snapshotID, userID)
```

### Hot Reload Subscription
```go
// Subscribe to configuration changes
ch := configService.Subscribe()

go func() {
    for event := range ch {
        fmt.Printf("Configuration %s changed from %v to %v\n", 
            event.Key, event.OldValue, event.NewValue)
    }
}()

// Unsubscribe when done
configService.Unsubscribe(ch)
```

## Default Configurations

The system comes with pre-configured settings:

### Site Settings
- `site_name`: Website name
- `site_description`: Website description
- `site_url`: Website URL
- `site_logo`: Logo URL
- `site_favicon`: Favicon URL

### Performance Settings
- `cache_ttl`: Cache time-to-live in seconds
- `static_generation`: Enable static HTML generation
- `compression_enabled`: Enable compression

### Feature Settings
- `comments_enabled`: Enable comments system
- `search_enabled`: Enable search functionality
- `analytics_enabled`: Enable analytics tracking
- `social_sharing`: Enable social sharing

### Appearance Settings
- `theme`: Website theme
- `primary_color`: Primary color
- `secondary_color`: Secondary color

## Default Feature Flags

- `new_editor`: New article editor
- `advanced_search`: Advanced search with filters
- `dark_mode`: Dark mode theme option
- `ai_content_suggestions`: AI-powered content suggestions
- `real_time_notifications`: Real-time push notifications
- `beta_features`: Access to beta features

## Security Considerations

1. **Secret Management**: Sensitive configurations are marked as secret and hidden from API responses
2. **Role-based Access**: Only admin users can modify configurations
3. **Audit Trail**: All changes are logged with user ID and reason
4. **Validation**: All inputs are validated before storage
5. **Encryption**: Database connections use SSL/TLS

## Performance Optimizations

1. **Caching**: Configurations are cached in DragonflyDB for fast access
2. **Hot Reload**: Configurable reload interval (default 30 seconds)
3. **Batch Operations**: Support for bulk configuration updates
4. **Prepared Statements**: Database queries use prepared statements
5. **Connection Pooling**: Efficient database connection management

## Requirements Satisfied

This implementation satisfies the following requirements from the specification:

- **Requirement 10**: Customizable admin interface for website appearance and layout
- **Requirement 32**: Google Tag Manager integration and tag management capabilities

The configuration system provides:
- ✅ Dynamic configuration system with hot reloading
- ✅ Settings management API and admin interface
- ✅ Feature flag system for gradual rollouts
- ✅ Configuration validation and rollback
- ✅ Comprehensive tests for all functionality

## Future Enhancements

1. **Configuration Templates**: Pre-defined configuration sets for different environments
2. **A/B Testing Integration**: Built-in A/B testing with feature flags
3. **Configuration Approval Workflow**: Multi-step approval for critical changes
4. **External Configuration Sources**: Support for external configuration providers
5. **Configuration Diff Viewer**: Visual comparison of configuration changes
6. **Bulk Import/Export**: Configuration backup and migration tools

## Conclusion

The configuration and settings management system provides a robust, scalable solution for managing application settings with enterprise-grade features including hot reloading, feature flags, validation, and rollback capabilities. The system is designed to handle high-volume operations while maintaining data consistency and providing comprehensive audit trails.