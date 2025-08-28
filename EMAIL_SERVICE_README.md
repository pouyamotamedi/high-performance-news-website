# Email Service Integration

This document describes the email service integration for the high-performance news website, providing newsletter functionality with SendGrid/Mailgun integration, subscriber management, and GDPR compliance.

## Features

### Core Features
- ✅ **Double Opt-in Subscription**: GDPR-compliant subscription process
- ✅ **Multi-Provider Support**: SendGrid and Mailgun integration
- ✅ **Bulk Email Sending**: 100K+ emails per hour capability
- ✅ **Email Templates**: Customizable HTML/text email templates
- ✅ **Campaign Management**: Create, schedule, and send email campaigns
- ✅ **Subscriber Management**: Full subscriber lifecycle management
- ✅ **Bounce Handling**: Automatic bounce and complaint processing
- ✅ **Webhook Processing**: Real-time email event processing
- ✅ **Analytics & Reporting**: Comprehensive email performance metrics
- ✅ **GDPR Compliance**: Data export and deletion capabilities

### Advanced Features
- ✅ **Rate Limiting**: Configurable sending limits per provider
- ✅ **Queue Processing**: Background job processing for bulk operations
- ✅ **Template Personalization**: Dynamic content based on subscriber data
- ✅ **Unsubscribe Management**: One-click unsubscribe with token validation
- ✅ **Email Validation**: Input validation and sanitization
- ✅ **Retry Logic**: Automatic retry for failed email sends
- ✅ **Performance Monitoring**: Email service health checks and metrics

## Architecture

### Database Schema

The email service uses the following database tables:

```sql
-- Subscribers with GDPR compliance
email_subscribers (
    id, email, status, confirmation_token, unsubscribe_token,
    first_name, last_name, preferences, source,
    confirmed_at, unsubscribed_at, created_at, updated_at
)

-- Email campaigns
email_campaigns (
    id, name, subject, template_id, content, status,
    scheduled_at, sent_at, recipient_count, sent_count,
    delivered_count, opened_count, clicked_count,
    bounced_count, unsubscribed_count, created_by,
    created_at, updated_at
)

-- Email templates
email_templates (
    id, name, subject, html_content, text_content,
    template_type, is_active, created_at, updated_at
)

-- Email send tracking (partitioned by date)
email_sends (
    id, campaign_id, subscriber_id, email, status,
    external_id, sent_at, delivered_at, opened_at,
    clicked_at, bounced_at, bounce_reason,
    unsubscribed_at, created_at
)
```

### Service Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Layer     │    │  Email Service   │    │  Email Provider │
│                 │    │                  │    │                 │
│ - REST APIs     │───▶│ - Subscription   │───▶│ - SendGrid      │
│ - Webhooks      │    │ - Campaigns      │    │ - Mailgun       │
│ - Admin Panel   │    │ - Templates      │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │                       ▼                       │
         │              ┌──────────────────┐             │
         │              │  Queue System    │             │
         │              │                  │             │
         │              │ - Bulk Sending   │             │
         │              │ - Retry Logic    │             │
         │              │ - Rate Limiting  │             │
         │              └──────────────────┘             │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Database      │    │  Webhook Handler │    │   Monitoring    │
│                 │    │                  │    │                 │
│ - PostgreSQL    │    │ - Event Processing│    │ - Metrics       │
│ - Partitioned   │    │ - Stats Updates  │    │ - Health Checks │
│ - Indexed       │    │ - GDPR Compliance│    │ - Alerting      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Installation & Setup

### 1. Database Migration

Run the email service migration:

```bash
# Apply the migration
./migrate -path migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" up

# Verify tables were created
psql -d news_website -c "\dt email_*"
```

### 2. Environment Configuration

Copy the example environment file and configure:

```bash
cp .env.email.example .env.email
```

Edit `.env.email` with your settings:

```bash
# Required: Choose email provider
EMAIL_PROVIDER=sendgrid  # or mailgun

# Required: Email settings
EMAIL_FROM=noreply@yournewssite.com
EMAIL_FROM_NAME=Your News Site
BASE_URL=https://yournewssite.com

# Required: Provider API keys
SENDGRID_API_KEY=your_sendgrid_api_key_here
# OR
MAILGUN_API_KEY=your_mailgun_api_key_here
MAILGUN_DOMAIN=mg.yournewssite.com
```

### 3. Provider Setup

#### SendGrid Setup
1. Create a SendGrid account
2. Generate an API key with full access
3. Set up webhook endpoints:
   - `https://yournewssite.com/api/v1/webhooks/email/sendgrid/events`
4. Configure webhook events: delivered, open, click, bounce, dropped, spamreport, unsubscribe

#### Mailgun Setup
1. Create a Mailgun account
2. Add and verify your domain
3. Generate an API key
4. Set up webhook endpoints:
   - `https://yournewssite.com/api/v1/webhooks/email/mailgun/events`
5. Configure webhook events: delivered, opened, clicked, bounced, dropped, complained, unsubscribed

### 4. Service Integration

Add email service to your main application:

```go
// main.go
func main() {
    // Load configuration
    emailConfig := config.LoadEmailConfig()
    
    // Create email service
    emailService, err := services.NewEmailService(db, emailConfig)
    if err != nil {
        log.Fatal("Failed to create email service:", err)
    }
    
    // Create email handlers
    emailHandlers := api.NewEmailHandlers(emailService)
    
    // Register routes
    v1 := router.Group("/api/v1")
    emailHandlers.RegisterRoutes(v1)
    
    // Start email processor
    emailProcessor := queue.NewEmailProcessor(emailService, jobQueue)
    go emailProcessor.Start(context.Background())
}
```

## API Usage

### Public APIs

#### Subscribe to Newsletter
```bash
curl -X POST http://localhost:8080/api/v1/email/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "preferences": {
      "categories": ["tech", "news"],
      "frequency": "daily"
    }
  }'
```

#### Confirm Subscription
```bash
curl -X GET http://localhost:8080/api/v1/email/confirm/CONFIRMATION_TOKEN
```

#### Unsubscribe
```bash
curl -X GET http://localhost:8080/api/v1/email/unsubscribe/UNSUBSCRIBE_TOKEN
```

### Admin APIs

#### Get Subscribers
```bash
curl -X GET http://localhost:8080/api/v1/admin/email/subscribers?status=confirmed&limit=50 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Create Campaign
```bash
curl -X POST http://localhost:8080/api/v1/admin/email/campaigns \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Weekly Newsletter",
    "subject": "This Week in News",
    "content": "<h1>Weekly Newsletter</h1><p>Latest news updates...</p>",
    "scheduled_at": "2024-01-15T10:00:00Z"
  }'
```

#### Send Campaign
```bash
curl -X POST http://localhost:8080/api/v1/admin/email/campaigns/1/send \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Get Email Statistics
```bash
curl -X GET http://localhost:8080/api/v1/admin/email/stats \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Email Templates

### Default Templates

The system includes three default templates:

1. **Confirmation Email**: Double opt-in confirmation
2. **Welcome Email**: Sent after subscription confirmation
3. **Newsletter Template**: Base template for campaigns

### Template Variables

Available variables in templates:

- `{{first_name}}` - Subscriber's first name
- `{{last_name}}` - Subscriber's last name
- `{{email}}` - Subscriber's email address
- `{{site_name}}` - Site name from configuration
- `{{confirmation_url}}` - Subscription confirmation URL
- `{{unsubscribe_url}}` - Unsubscribe URL
- `{{subject}}` - Email subject (for newsletter templates)
- `{{content}}` - Email content (for newsletter templates)

### Custom Templates

Create custom templates via API:

```bash
curl -X POST http://localhost:8080/api/v1/admin/email/templates \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Custom Newsletter",
    "subject": "{{subject}}",
    "html_content": "<html><body><h1>{{subject}}</h1>{{content}}<p><a href=\"{{unsubscribe_url}}\">Unsubscribe</a></p></body></html>",
    "text_content": "{{subject}}\n\n{{content}}\n\nUnsubscribe: {{unsubscribe_url}}",
    "template_type": "newsletter"
  }'
```

## Performance & Scaling

### Rate Limiting

The service implements rate limiting to respect provider limits:

- **SendGrid**: 100 emails/second (configurable)
- **Mailgun**: 10 emails/second for free tier, higher for paid plans
- **Batch Processing**: 1000 emails per batch (configurable)

### Queue Processing

Bulk email operations are processed asynchronously:

```go
// Queue bulk email sending
emails := []services.BulkEmail{
    {To: "user1@example.com", Subject: "Newsletter", HTMLContent: "..."},
    {To: "user2@example.com", Subject: "Newsletter", HTMLContent: "..."},
}

err := emailProcessor.QueueBulkEmails(emails)
```

### Database Optimization

- **Partitioned Tables**: `email_sends` partitioned by date
- **BRIN Indexes**: Optimized for time-series data
- **Connection Pooling**: Efficient database connections
- **Prepared Statements**: Optimized query performance

## Monitoring & Analytics

### Key Metrics

The service tracks comprehensive email metrics:

- **Subscriber Metrics**: Total, active, pending, unsubscribed
- **Campaign Metrics**: Sent, delivered, opened, clicked, bounced
- **Performance Metrics**: Send rate, delivery rate, open rate, click rate
- **System Metrics**: Queue size, processing time, error rate

### Health Checks

Monitor service health:

```bash
curl -X GET http://localhost:8080/api/v1/system/health
```

Response:
```json
{
  "status": "healthy",
  "email_service": {
    "status": "healthy",
    "provider": "sendgrid",
    "queue_size": 0,
    "last_send": "2024-01-15T10:30:00Z"
  }
}
```

## GDPR Compliance

### Data Protection

The service implements GDPR compliance features:

- **Double Opt-in**: Required for all subscriptions
- **Consent Tracking**: Records subscription source and timestamp
- **Data Minimization**: Only collects necessary subscriber data
- **Right to Access**: API for data export
- **Right to Erasure**: API for data deletion
- **Data Portability**: Structured data export

### Data Export

Export subscriber data:

```bash
curl -X GET http://localhost:8080/api/v1/admin/email/subscribers/user@example.com/export \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Data Deletion

Delete subscriber data:

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/email/subscribers/user@example.com \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Testing

### Unit Tests

Run the email service tests:

```bash
go test ./internal/services -v -run TestEmailService
```

### Integration Tests

Test with real email providers (requires API keys):

```bash
EMAIL_PROVIDER=sendgrid SENDGRID_API_KEY=your_key go test ./internal/services -v -run TestEmailService_Integration
```

### Load Testing

Test bulk email performance:

```bash
go test ./internal/services -v -run BenchmarkEmailService -bench=.
```

## Troubleshooting

### Common Issues

1. **Emails not sending**
   - Check API keys are correct
   - Verify provider account status
   - Check rate limiting settings

2. **High bounce rates**
   - Validate email addresses before sending
   - Check sender reputation
   - Review email content for spam triggers

3. **Webhook events not processing**
   - Verify webhook URLs are accessible
   - Check webhook secret configuration
   - Review webhook event logs

### Debug Mode

Enable debug logging:

```bash
EMAIL_DEBUG=true LOG_LEVEL=debug ./your-app
```

### Dry Run Mode

Test without sending actual emails:

```bash
EMAIL_DRY_RUN=true ./your-app
```

## Security Considerations

### API Security
- JWT token authentication for admin APIs
- Rate limiting on public endpoints
- Input validation and sanitization
- CSRF protection for web forms

### Email Security
- SPF/DKIM/DMARC configuration
- Secure webhook endpoints
- Token-based unsubscribe links
- Encrypted API credentials

### Data Security
- Encrypted database connections
- Secure token generation
- PII data protection
- Audit logging

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review the API documentation
3. Check provider documentation (SendGrid/Mailgun)
4. Enable debug logging for detailed error information

## License

This email service integration is part of the high-performance news website project.