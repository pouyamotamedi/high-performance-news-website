# Implementation Plan

Convert the feature design into a series of prompts for a code-generation LLM that will implement each step in a test-driven manner. Prioritize best practices, incremental progress, and early testing, ensuring no big jumps in complexity at any stage. Make sure that each prompt builds on the previous prompts, and ends with wiring things together. There should be no hanging or orphaned code that isn't integrated into a previous step. Focus ONLY on tasks that involve writing, modifying, or testing code.

## Phase 1: Core Infrastructure Setup

- [x] 1. Initialize Go project structure and core dependencies





  - Create Go module with proper directory structure (cmd, internal, pkg, web, migrations)
  - Set up dependency management with go.mod including Gin, PostgreSQL driver, DragonflyDB client
  - Configure development environment with Docker Compose for PostgreSQL, DragonflyDB, and PgBouncer
  - Create basic configuration management using environment variables and Viper
  - Configure code quality tools (golangci-lint, gofmt, go vet)
  - Set up hot-reload for development using air or similar tool
  - _Requirements: 17, 18_

- [x] 2. Implement database connection and migration system





  - Set up PgBouncer for connection pooling in transaction mode with 200 max connections
  - Create database connection pool with optimized settings (150 max connections, 40 idle)
  - Implement migration system using golang-migrate with up/down migration support
  - Create initial database schema for articles table with partitioning by published_at
  - Add BRIN indexes and composite indexes for performance optimization
  - Write unit tests for database connection, PgBouncer integration, and migration functionality
  - _Requirements: 1.5, 17, 22_

- [x] 3. Set up automated partition management system





  - Implement PartitionManager struct with CreateDailyPartitions and DropOldPartitions methods
  - Create daily partition creation logic for next 7 days with proper error handling
  - Implement old partition cleanup with configurable retention (30 days default)
  - Add cron job scheduling for daily partition maintenance
  - Write comprehensive tests for partition creation, deletion, and error scenarios
  - _Requirements: 1.5, 22_

- [x] 4. Implement caching layer with DragonflyDB





  - Create CacheService interface with Get, Set, Delete, DeletePattern, Exists methods
  - Implement DragonflyDB client with connection pooling and error handling
  - Configure cache TTL constants (homepage: 15min, articles: 24h, categories: 30min)
  - Add cache key patterns and invalidation strategies
  - Write unit tests for all cache operations and TTL behavior
  - _Requirements: 5, 5.5, 17_

## Phase 2: Core Content Management System

- [x] 5. Create core data models and validation





  - Implement Article, User, Category, Tag structs with proper JSON and database tags
  - Add SEOData struct with meta title, description, keywords, canonical URL fields
  - Create validation functions for all data models with comprehensive error handling
  - Implement slug generation and uniqueness validation
  - Write unit tests for all data models and validation logic
  - _Requirements: 1, 6, 7, 8_

- [x] 6. Implement user management and authentication system








  - Create User model with role-based access (Admin, Editor, Reporter, Contributor)
  - Implement JWT-based authentication with secure token generation and validation
  - Add password hashing using bcrypt with proper salt rounds
  - Create user CRUD operations with role-based permissions
  - Write comprehensive tests for authentication, authorization, and user management
  - _Requirements: 6, 12_

- [x] 7. Build article repository layer with prepared statements





  - Create ArticleRepository with CRUD operations using prepared statements
  - Implement GetArticleBySlug, GetArticlesByCategory, GetTrendingArticles methods
  - Add bulk operations for high-volume article processing using PostgreSQL COPY
  - Implement graceful degradation with cache → database → static file fallback
  - Write unit and integration tests for all repository methods
  - _Requirements: 1, 1.5, 22_

- [x] 8. Implement category and tag management system





  - Create Category and Tag repositories with hierarchical category support
  - Implement tag keyword bank functionality with unique keyword constraints
  - Add bulk creation operations for categories and tags
  - Create relationship management between articles, categories, and tags
  - Write tests for category hierarchy, tag relationships, and bulk operations
  - _Requirements: 7, 65_

- [x] 8.5. Create initial load testing framework


  - Set up basic load testing with 100 concurrent users using k6 or similar tool
  - Test article creation at 35 articles/minute rate (50K daily target)
  - Establish performance baselines for database operations and API responses
  - Identify early bottlenecks in database queries and connection handling
  - Create automated performance regression testing
  - _Requirements: 22_

## Phase 3: API and Content Integration

- [x] 9. Build RESTful API with comprehensive endpoints











  - Implement article CRUD API with pagination, filtering, and sorting
  - Create bulk operations API for high-volume content management (1000+ articles per request)
  - Add search API with advanced filtering and faceted search
  - Implement user management API with role-based access control
  - Write API tests for all endpoints, error handling, and performance
  - _Requirements: 1, 2, 6, 33_

- [x] 10. Implement rate limiting and security middleware






  - Create per-endpoint rate limiting with different limits for each API
  - Implement CSRF protection and 2FA for admin accounts
  - Add API key management with rotation and expiration
  - Create security headers and input validation middleware
  - Write security tests for rate limiting, CSRF, 2FA, and input validation
  - _Requirements: 12, 30_

- [x] 11. Implement comment system with moderation





  - Create Comment model with nested thread support and user relationships
  - Implement moderation queue with approval workflow and spam detection
  - Add comment API endpoints with rate limiting and authentication
  - Create comment threading and reply functionality
  - Write tests for comment creation, moderation, threading, and spam detection
  - _Requirements: 37_

- [x] 12. Build content ingestion from external sources








  - Implement API endpoints for automated content ingestion (n8n, other tools)
  - Create content validation and sanitization for external sources
  - Add source tracking and attribution management
  - Implement duplicate content detection and handling
  - Write tests for content ingestion, validation, and duplicate detection
  - _Requirements: 2_

## Phase 4: Advanced Content Features

- [x] 13. Build auto-linking system with keyword matching





  - Implement Trie data structure for efficient longest-match keyword detection
  - Create KeywordMatcher with ProcessArticleLinks method for automatic internal linking
  - Add priority-based keyword matching (longest match wins, one link per keyword per article)
  - Implement per-article auto-linking override functionality
  - Write comprehensive tests for keyword matching, link generation, and edge cases
  - _Requirements: 7, 65_

- [x] 14. Implement delayed canonicalization system





  - Create CanonicalManager with 48-hour delay scheduling functionality
  - Implement canonical queue processing with database-backed job storage
  - Add admin override for immediate canonicalization implementation
  - Create canonical URL generation for tags and categories
  - Write tests for scheduling, processing, and URL generation logic
  - _Requirements: 8, 41_

- [x] 15. Build multilingual support system





  - Add language columns to articles, categories, and tags tables
  - Implement translation group management for article relationships
  - Create language-aware URL routing and content retrieval
  - Add RTL/LTR layout support with proper CSS and template handling
  - Write tests for multilingual content management and URL generation
  - _Requirements: 4, 44_

- [x] 16. Implement content versioning and moderation





  - Create article version history tracking with author and timestamp
  - Implement content moderation queue with approval/rejection workflow
  - Add AI-powered content quality checking integration (OpenAI/Anthropic)
  - Create bulk moderation tools for high-volume content processing
  - Write tests for versioning, moderation workflow, and AI integration
  - _Requirements: 35, 36_

## Phase 5: Performance Optimization

- [x] 17. Build static HTML generation system












  - Implement StaticGenerator with template-based HTML generation
  - Create static file generation for articles, homepage, category, and tag pages
  - Add nginx configuration for static-first serving with dynamic fallback
  - Implement cache warming and static file regeneration on content updates
  - Write tests for static generation, file serving, and cache invalidation
  - _Requirements: 5.5, 42, 46, 47_

- [x] 18. Implement memory-aware job queue system





  - Create JobQueue with memory pressure monitoring (28GB threshold)
  - Implement priority-based job processing (high/medium/low priority queues)
  - Add job types for static generation, image processing, search indexing, notifications
  - Create worker pool management with graceful shutdown and error recovery
  - Write tests for job queuing, processing, memory monitoring, and error handling
  - _Requirements: 1, 5.5, 22_

- [x] 19. Build search system with MeiliSearch integration

















  - Implement SearchIndexer with batched operations (1000 articles per batch)
  - Create search API endpoints with filtering, sorting, and pagination
  - Add real-time search indexing on article publication with fallback to PostgreSQL
  - Implement search result caching and performance optimization
  - Write tests for indexing, searching, batching, and fallback mechanisms
  - _Requirements: 33, 22_

- [x] 20. Implement image processing pipeline





  - Create ImageProcessor with multi-format generation (WebP, AVIF, JPEG)
  - Add responsive image generation (thumbnail, mobile, tablet, desktop sizes)
  - Implement error recovery for critical vs non-critical image sizes
  - Create lazy loading and progressive image loading support
  - Write tests for image processing, format generation, and error handling
  - _Requirements: 21, 49_

## Phase 6: Frontend Integration

- [x] 21. Build admin panel backend services











  - Implement dashboard metrics API with real-time data
  - Create content management APIs for articles, categories, tags, users
  - Add system monitoring APIs for health checks, performance metrics
  - Implement configuration management API for site settings
  - Write tests for admin APIs, metrics collection, and monitoring
  - _Requirements: 9, 10, 34_

- [x] 22. Create frontend template system with SSR










  - Implement Go template engine with layout inheritance and partials
  - Create responsive templates for homepage, article, category, tag pages
  - Add mobile-first design with RTL/LTR support and dark mode
  - Implement Progressive Web App features with service worker
  - Write tests for template rendering, responsive design, and PWA functionality
  - _Requirements: 42, 43, 44, 45, 46, 47, 48, 52_

## Phase 7: SEO and Content Syndication

- [x] 23. Implement comprehensive SEO system







  - Create automatic schema markup generation (NewsArticle, Article, BlogPosting)
  - Implement sitemap generation with instant updates and news sitemaps
  - Add breadcrumb navigation with schema markup
  - Create meta tag optimization and Open Graph/Twitter Card generation
  - Write tests for schema generation, sitemaps, and meta tag optimization
  - _Requirements: 3, 23, 44, 65_

- [x] 24. Build RSS feed system with delayed publishing











  - Implement RSSGenerator with 2-hour delay for content inclusion
  - Create category and tag-specific RSS feeds with proper caching
  - Add RSS feed management and customization options
  - Implement feed validation and error handling
  - Write tests for RSS generation, delay logic, and feed validation
  - _Requirements: 13, 40_

- [x] 24.5. Implement Google News optimization


  - Create Google News sitemap with 1000 article limit per file
  - Add required news metadata (news_keywords, stock_tickers, article sections)
  - Implement Google News RSS feed format with proper GUID structure
  - Add news-specific structured data and Publisher Center integration
  - Write tests for Google News sitemap generation and validation
  - _Requirements: 23_

- [x] 25. Implement social media integration

















  - Create social media publisher for Facebook Instant Articles, Telegram, Twitter
  - Add automatic posting with retry logic and exponential backoff
  - Implement social media credential management and rotation
  - Create webhook endpoints for social media callbacks
  - Write tests for social media posting, retry logic, and webhook handling
  - _Requirements: 24, 31_

- [x] 26. Build email service integration





  - Implement newsletter system with SendGrid/Mailgun integration
  - Create subscriber management with double opt-in and GDPR compliance
  - Add email template system and bulk sending capabilities (100K emails/hour)
  - Implement bounce handling and unsubscribe management
  - Write tests for email sending, subscriber management, and compliance
  - _Requirements: 25_

## Phase 8: Analytics and Monitoring

- [x] 27. Implement comprehensive analytics system






  - Create article view tracking with IP-based analytics
  - Implement engagement tracking (likes, dislikes, shares, comments)
  - Add user behavior analytics and performance metrics collection
  - Create reporting system with time-range filtering and export capabilities
  - Write tests for analytics collection, reporting, and data export
  - _Requirements: 9, 37, 38_

- [x] 28. Build performance monitoring system
























  - Implement Prometheus metrics collection for all system components
  - Create health check system with publishing rate, cache hit rate monitoring
  - Add resource usage monitoring (CPU, memory, disk, database connections)
  - Implement alerting system with configurable thresholds
  - Write tests for metrics collection, health checks, and alerting
  - _Requirements: 19, 22, 64_

- [x] 29. Create advertisement management system













  - Implement ad placement system with targeting by categories and tags
  - Create ad performance tracking (impressions, clicks, CTR)
  - Add lazy loading and Core Web Vitals optimization for ads
  - Implement ad rotation and A/B testing capabilities
  - Write tests for ad serving, targeting, tracking, and performance optimization
  - _Requirements: 11, 28_

- [x] 30. Build push notification system





  - Implement push notification service with OneSignal/Firebase integration
  - Create user subscription management and targeting
  - Add notification scheduling and delivery tracking
  - Implement notification templates and personalization
  - Write tests for notification sending, targeting, and delivery tracking
  - _Requirements: 26_

## Phase 9: Advanced Features and Integrations

- [x] 31. Implement widget and template management system





  - Create widget system with pre-built components (latest articles, popular, categories)
  - Implement template override system with visual previews
  - Add drag-and-drop widget placement and configuration
  - Create theme customization with color, font, and layout options
  - Write tests for widget management, template system, and customization
  - _Requirements: 55, 56, 57_

- [x] 32. Build AI integration system






  - Implement AI content quality checking and optimization
  - Create automatic meta description and title generation
  - Add bulk content optimization and SEO enhancement
  - Implement llms.txt generation and AI-ready structured data
  - Write tests for AI integration, content optimization, and quality checking
  - _Requirements: 16, 35_

- [x] 33. Implement CDN integration (optional)








  - Create CDN integration with Cloudflare API
  - Implement cache purging and optimization
  - Add CDN failover and performance monitoring
  - Create CDN configuration management
  - Write tests for CDN integration, purging, and failover
  - _Requirements: 27_

- [x] 34. Build backup and disaster recovery system









  - Implement automated backup system with point-in-time recovery
  - Create cross-region backup replication and validation
  - Add disaster recovery testing and validation procedures
  - Implement backup encryption and compression
  - Write tests for backup creation, restoration, and disaster recovery
  - _Requirements: 20_

## Phase 10: Deployment and Operations

- [ ] 35. Create zero-touch deployment system
  - Implement deployment agent with SSH-based server management
  - Create automated server setup and dependency installation
  - Add blue-green deployment with automatic rollback capabilities
  - Implement deployment validation and health checking
  - Write tests for deployment automation, rollback, and validation
  - _Requirements: 59, 60, 61, 63_

- [ ] 36. Build desktop deployment application
  - Create cross-platform desktop app (Windows, macOS, Linux)
  - Implement server connection management and credential storage
  - Add deployment monitoring and log viewing capabilities
  - Create backup management and system monitoring interface
  - Write tests for desktop app functionality and server communication
  - _Requirements: 60_

- [ ] 37. Implement monitoring and alerting system
  - Create comprehensive system monitoring with real-time dashboards
  - Implement automated alerting for performance and system issues
  - Add log aggregation and analysis capabilities
  - Create operational runbooks and automated remediation
  - Write tests for monitoring, alerting, and automated responses
  - _Requirements: 19, 64_

- [ ] 38. Build configuration and settings management
  - Implement dynamic configuration system with hot reloading
  - Create settings management API and admin interface
  - Add feature flag system for gradual rollouts
  - Implement configuration validation and rollback
  - Write tests for configuration management, validation, and rollback
  - _Requirements: 10, 32_

## Phase 11: Testing and Optimization

- [ ] 39. Implement comprehensive testing suite
  - Create unit tests for all core functionality with >90% coverage
  - Implement integration tests for database, cache, and external services
  - Add end-to-end tests for critical user journeys
  - Create performance tests for 50K articles/day load scenarios
  - Write load tests for concurrent users and peak traffic scenarios
  - _Requirements: 22_

- [ ] 40. Perform security hardening and testing
  - Implement security scanning and vulnerability assessment
  - Create penetration testing procedures and remediation
  - Add security monitoring and incident response procedures
  - Implement compliance checking and audit logging
  - Write security tests for authentication, authorization, and data protection
  - _Requirements: 12, 30_

- [ ] 41. Optimize performance and scalability
  - Implement database query optimization and index tuning
  - Create caching optimization and cache warming strategies
  - Add performance profiling and bottleneck identification
  - Implement horizontal scaling preparation and load balancing
  - Write performance benchmarks and optimization tests
  - _Requirements: 1, 5, 22_

- [ ] 42. Final integration and deployment preparation
  - Integrate all system components with comprehensive error handling
  - Create production deployment scripts and configuration
  - Implement monitoring and alerting for production environment
  - Create operational documentation and runbooks
  - Perform final system testing and validation before production deployment
  - _Requirements: All requirements integration_