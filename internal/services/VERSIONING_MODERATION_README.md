# Content Versioning and Moderation System

This document describes the implementation of the content versioning and moderation system for the high-performance news website.

## Overview

The system provides comprehensive content versioning with full history tracking and an AI-powered moderation workflow that supports both individual and bulk operations for high-volume content processing.

## Features

### Content Versioning
- **Automatic Version Creation**: Every article update creates a new version with change tracking
- **Complete History**: Full version history with author, timestamp, and change summaries
- **Version Comparison**: Compare any two versions to see differences
- **Version Restoration**: Restore articles to any previous version
- **Performance Optimized**: Efficient queries for high-volume operations

### Content Moderation
- **AI-Powered Analysis**: Automatic quality checking using OpenAI/Anthropic APIs
- **Approval Workflow**: Pending → Approved/Rejected/Flagged workflow
- **Auto-Approval**: High-quality content can be automatically approved
- **Priority System**: 4-level priority system (1=low, 4=urgent)
- **Assignment System**: Assign moderators to specific content
- **Bulk Operations**: Process thousands of items with bulk approve/reject/flag operations

### AI Integration
- **Quality Scoring**: Overall content quality (0.0-1.0)
- **Grammar Analysis**: Grammar and spelling checking
- **Readability Assessment**: Content readability scoring
- **Appropriateness Check**: Content appropriateness for news context
- **Issue Detection**: Automatic detection of grammar, spelling, and content issues
- **Improvement Suggestions**: AI-generated suggestions for content improvement
- **Content Flagging**: Automatic flagging of inappropriate or low-quality content

## Database Schema

### Core Tables

#### article_versions
Stores complete version history for articles:
```sql
CREATE TABLE article_versions (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    version_number INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    -- ... all article fields
    change_summary TEXT,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(article_id, version_number)
);
```

#### moderation_queue
Manages content moderation workflow:
```sql
CREATE TABLE moderation_queue (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority INTEGER DEFAULT 1,
    ai_quality_score DECIMAL(3,2),
    ai_feedback JSONB,
    auto_approved BOOLEAN DEFAULT false,
    -- ... moderation fields
);
```

#### content_quality_checks
Stores AI analysis results:
```sql
CREATE TABLE content_quality_checks (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    ai_provider VARCHAR(20) NOT NULL,
    quality_score DECIMAL(3,2) NOT NULL,
    grammar_score DECIMAL(3,2),
    readability_score DECIMAL(3,2),
    appropriateness_score DECIMAL(3,2),
    issues_found JSONB,
    suggestions JSONB,
    flagged_content JSONB,
    processing_time_ms INTEGER
);
```

## API Endpoints

### Versioning API

#### Create Version
```http
POST /api/v1/articles/{id}/versions
Content-Type: application/json

{
    "change_summary": "Updated article content and title"
}
```

#### Get Version History
```http
GET /api/v1/articles/{id}/versions
```

#### Compare Versions
```http
GET /api/v1/articles/{id}/versions/{version1}/compare/{version2}
```

#### Restore Version
```http
POST /api/v1/articles/{id}/versions/restore
Content-Type: application/json

{
    "version_number": 3
}
```

### Moderation API

#### Submit for Moderation
```http
POST /api/v1/moderation/submit
Content-Type: application/json

{
    "article_id": 123,
    "content_type": "article",
    "priority": 2
}
```

#### Get Moderation Queue
```http
GET /api/v1/moderation/queue?status=pending&priority=3,4&limit=20
```

#### Approve Content
```http
POST /api/v1/moderation/{id}/approve
Content-Type: application/json

{
    "notes": "Content approved after review"
}
```

#### Bulk Operations
```http
POST /api/v1/moderation/bulk-jobs
Content-Type: application/json

{
    "job_name": "Bulk Approve High Quality",
    "job_type": "bulk_approve",
    "criteria": {
        "status": ["pending"],
        "ai_quality_min": 0.8
    }
}
```

## Service Architecture

### VersioningService
Handles all version-related operations:
- `CreateVersion()` - Creates new version
- `GetVersionHistory()` - Retrieves version history
- `CompareVersions()` - Compares two versions
- `RestoreVersion()` - Restores to specific version
- `DeleteOldVersions()` - Cleanup old versions

### ModerationService
Manages moderation workflow:
- `SubmitForModeration()` - Submits content for review
- `RunAIQualityCheck()` - Performs AI analysis
- `ApproveContent()` - Approves content
- `RejectContent()` - Rejects content
- `GetModerationQueue()` - Retrieves moderation items

### BulkModerationService
Handles bulk operations:
- `CreateBulkJob()` - Creates bulk operation job
- `ExecuteBulkJob()` - Executes bulk operation
- `GetBulkJobs()` - Lists bulk jobs

### AIService Interface
Provides AI analysis capabilities:
- `AnalyzeContent()` - Comprehensive content analysis
- `GenerateMetaDescription()` - SEO meta description generation
- `CheckGrammar()` - Grammar and spelling check
- `CheckReadability()` - Readability assessment

## Configuration

### AI Service Configuration
```go
// OpenAI Configuration
aiService := services.NewOpenAIService("your-api-key", "gpt-4o-mini")

// Anthropic Configuration
aiService := services.NewAnthropicService("your-api-key", "claude-3-haiku-20240307")
```

### Auto-Approval Thresholds
```go
const (
    AutoApproveThreshold = 0.85  // Quality score threshold
    HighPriorityThreshold = 3    // Priority level for urgent items
)
```

## Performance Considerations

### Database Optimization
- **Partitioning**: Version tables can be partitioned by date for better performance
- **Indexing**: Optimized indexes for version queries and moderation filtering
- **Prepared Statements**: All queries use prepared statements for performance

### Bulk Operations
- **Batch Processing**: Process items in batches of 100-1000
- **Progress Tracking**: Real-time progress updates for bulk jobs
- **Error Handling**: Graceful handling of individual item failures
- **Memory Management**: Efficient memory usage for large datasets

### AI Integration
- **Async Processing**: AI analysis runs asynchronously to avoid blocking
- **Rate Limiting**: Respects AI service rate limits
- **Caching**: AI results are cached to avoid duplicate analysis
- **Fallback**: Graceful degradation when AI services are unavailable

## Usage Examples

### Creating and Managing Versions
```go
// Create a new version
version, err := versioningService.CreateVersion(article, "Updated content", userID)

// Get version history
versions, err := versioningService.GetVersionHistory(articleID)

// Compare versions
comparison, err := versioningService.CompareVersions(articleID, 1, 2)

// Restore to previous version
err = versioningService.RestoreVersion(articleID, 3, userID)
```

### Moderation Workflow
```go
// Submit for moderation
moderationItem, err := moderationService.SubmitForModeration(articleID, "article", 2, userID)

// AI analysis runs automatically in background

// Approve content
err = moderationService.ApproveContent(moderationID, moderatorID, "Looks good", false)

// Bulk approve high-quality content
criteria := &models.BulkModerationCriteria{
    Status: []string{"pending"},
    AIQualityMin: &[]float64{0.8}[0],
}
job, err := bulkModerationService.CreateBulkJob("Bulk Approve", "bulk_approve", criteria, userID)
err = bulkModerationService.ExecuteBulkJob(job.ID)
```

## Testing

The system includes comprehensive tests:
- **Unit Tests**: Individual service method testing
- **Integration Tests**: End-to-end workflow testing
- **Performance Tests**: High-volume operation testing
- **AI Service Tests**: Mock AI service testing

### Running Tests
```bash
# Run all tests
go test ./internal/services/...

# Run specific test suites
go test ./internal/services/ -run TestVersioning
go test ./internal/services/ -run TestModeration
go test ./internal/services/ -run TestAI

# Run integration tests (requires test database)
go test ./internal/services/ -run TestIntegration
```

## Monitoring and Metrics

### Key Metrics
- **Version Creation Rate**: Versions created per minute
- **Moderation Queue Size**: Pending items count
- **AI Processing Time**: Average AI analysis time
- **Auto-Approval Rate**: Percentage of auto-approved content
- **Bulk Job Performance**: Items processed per minute

### Health Checks
- Database connectivity
- AI service availability
- Queue processing status
- Bulk job execution status

## Security Considerations

### Access Control
- **Role-Based Permissions**: Admin, Editor, Reporter roles
- **API Authentication**: JWT-based authentication required
- **Action Logging**: All moderation actions are logged for audit

### Data Protection
- **Input Validation**: All inputs are validated and sanitized
- **SQL Injection Prevention**: Prepared statements used throughout
- **Rate Limiting**: API endpoints have rate limiting
- **Sensitive Data**: AI feedback may contain sensitive content flags

## Deployment

### Database Migration
```bash
# Apply versioning and moderation schema
migrate -path migrations -database "postgres://..." up
```

### Environment Variables
```bash
OPENAI_API_KEY=your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key
AI_PROVIDER=openai  # or anthropic
AUTO_APPROVE_THRESHOLD=0.85
```

### Service Dependencies
- PostgreSQL 15+ (with JSONB support)
- OpenAI API or Anthropic API access
- Redis (optional, for caching AI results)

## Future Enhancements

### Planned Features
- **Visual Diff**: HTML-based visual comparison of versions
- **Collaborative Editing**: Real-time collaborative version editing
- **Advanced AI**: Custom AI models for domain-specific analysis
- **Workflow Automation**: Custom moderation workflows
- **Analytics Dashboard**: Comprehensive moderation analytics

### Scalability Improvements
- **Distributed Processing**: Multi-server bulk job processing
- **AI Service Load Balancing**: Multiple AI service providers
- **Version Archiving**: Archive old versions to cold storage
- **Real-time Updates**: WebSocket-based real-time queue updates

This system provides a robust foundation for content versioning and moderation that can handle the high-volume requirements of a news website processing 50,000+ articles daily.