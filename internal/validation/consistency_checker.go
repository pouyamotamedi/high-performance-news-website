package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/database"
)

// ConsistencyChecker performs data consistency validation using sample-based approach
type ConsistencyChecker struct {
	db        *database.DB
	scheduler *CheckScheduler
	reporter  *ConsistencyReporter
}

// ConsistencyCheckType represents different types of consistency checks
type ConsistencyCheckType string

const (
	CheckTypeSample              ConsistencyCheckType = "sample"
	CheckTypeReferentialIntegrity ConsistencyCheckType = "referential_integrity"
	CheckTypeMultilingual        ConsistencyCheckType = "multilingual"
	CheckTypeSEOMetadata         ConsistencyCheckType = "seo_metadata"
)

// CheckStatus represents the status of a consistency check
type CheckStatus string

const (
	CheckStatusPassed  CheckStatus = "passed"
	CheckStatusFailed  CheckStatus = "failed"
	CheckStatusWarning CheckStatus = "warning"
)

// ConsistencyCheck represents a consistency validation run
type ConsistencyCheck struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Type       ConsistencyCheckType `json:"type"`
	Status     CheckStatus          `json:"status"`
	Issues     []ConsistencyIssue   `json:"issues"`
	ExecutedAt time.Time            `json:"executed_at"`
	Duration   time.Duration        `json:"duration"`
	SampleSize int                  `json:"sample_size"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ConsistencyIssue represents a data consistency problem
type ConsistencyIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	ArticleID   *uint64                `json:"article_id,omitempty"`
	CategoryID  *uint64                `json:"category_id,omitempty"`
	TagID       *uint64                `json:"tag_id,omitempty"`
	UserID      *uint64                `json:"user_id,omitempty"`
	Details     map[string]interface{} `json:"details"`
	CreatedAt   time.Time              `json:"created_at"`
}

// SampleArticle represents a sampled article for consistency checking
type SampleArticle struct {
	ID                 uint64    `json:"id" db:"id"`
	Title              string    `json:"title" db:"title"`
	Slug               string    `json:"slug" db:"slug"`
	AuthorID           uint64    `json:"author_id" db:"author_id"`
	CategoryID         uint64    `json:"category_id" db:"category_id"`
	Status             string    `json:"status" db:"status"`
	PublishedAt        *time.Time `json:"published_at" db:"published_at"`
	LanguageCode       string    `json:"language_code" db:"language_code"`
	TranslationGroupID *uint64   `json:"translation_group_id" db:"translation_group_id"`
	MetaTitle          string    `json:"meta_title" db:"meta_title"`
	MetaDescription    string    `json:"meta_description" db:"meta_description"`
	CanonicalURL       string    `json:"canonical_url" db:"canonical_url"`
	SchemaType         string    `json:"schema_type" db:"schema_type"`
}

// NewConsistencyChecker creates a new consistency checker
func NewConsistencyChecker(db *database.DB) *ConsistencyChecker {
	return &ConsistencyChecker{
		db:        db,
		scheduler: NewCheckScheduler(),
		reporter:  NewConsistencyReporter(),
	}
}

// ValidateDataConsistency performs comprehensive data consistency validation
func (c *ConsistencyChecker) ValidateDataConsistency(ctx context.Context) (*ConsistencyCheck, error) {
	check := &ConsistencyCheck{
		ID:         generateCheckID(),
		Name:       "Sample-Based Data Consistency Check",
		Type:       CheckTypeSample,
		ExecutedAt: time.Now(),
		SampleSize: 1000, // Check 1000 random articles
		Metadata:   make(map[string]interface{}),
	}

	start := time.Now()
	log.Printf("Starting data consistency check with sample size: %d", check.SampleSize)

	// Sample recent articles for consistency checking
	sampleArticles, err := c.getSampleArticles(ctx, check.SampleSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get sample articles: %w", err)
	}

	check.Metadata["actual_sample_size"] = len(sampleArticles)
	log.Printf("Retrieved %d articles for consistency checking", len(sampleArticles))

	// Run consistency checks
	var allIssues []ConsistencyIssue

	// 1. Referential integrity validation
	refIssues := c.validateReferentialIntegrity(ctx, sampleArticles)
	allIssues = append(allIssues, refIssues...)

	// 2. Multilingual content consistency validation
	multilingualIssues := c.validateMultilingualConsistency(ctx, sampleArticles)
	allIssues = append(allIssues, multilingualIssues...)

	// 3. SEO metadata consistency checking
	seoIssues := c.validateSEOMetadataConsistency(ctx, sampleArticles)
	allIssues = append(allIssues, seoIssues...)

	check.Issues = allIssues
	check.Duration = time.Since(start)
	check.Status = c.determineCheckStatus(allIssues)

	// Store metadata about the check
	check.Metadata["referential_issues"] = len(refIssues)
	check.Metadata["multilingual_issues"] = len(multilingualIssues)
	check.Metadata["seo_issues"] = len(seoIssues)

	log.Printf("Consistency check completed in %v with %d issues found", check.Duration, len(allIssues))

	return check, nil
}

// getSampleArticles retrieves a random sample of articles using TABLESAMPLE
func (c *ConsistencyChecker) getSampleArticles(ctx context.Context, count int) ([]SampleArticle, error) {
	// Use TABLESAMPLE for efficient random sampling across partitions
	query := `
		WITH sampled_articles AS (
			SELECT a.id, a.title, a.slug, a.author_id, a.category_id, a.status, 
				   a.published_at, a.language_code, a.translation_group_id,
				   a.meta_title, a.meta_description, a.canonical_url, a.schema_type
			FROM articles a TABLESAMPLE SYSTEM (5) -- Sample ~5% of rows
			WHERE a.status = 'published' 
			AND a.published_at > NOW() - INTERVAL '30 days' -- Focus on recent articles
			ORDER BY RANDOM()
			LIMIT $1
		)
		SELECT * FROM sampled_articles
	`

	rows, err := c.db.QueryContext(ctx, query, count)
	if err != nil {
		return nil, fmt.Errorf("error sampling articles: %w", err)
	}
	defer rows.Close()

	var articles []SampleArticle
	for rows.Next() {
		var article SampleArticle
		var metaTitle, metaDescription, canonicalURL, schemaType sql.NullString

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.AuthorID,
			&article.CategoryID, &article.Status, &article.PublishedAt,
			&article.LanguageCode, &article.TranslationGroupID,
			&metaTitle, &metaDescription, &canonicalURL, &schemaType,
		)
		if err != nil {
			log.Printf("Error scanning article: %v", err)
			continue
		}

		// Handle nullable fields
		article.MetaTitle = metaTitle.String
		article.MetaDescription = metaDescription.String
		article.CanonicalURL = canonicalURL.String
		article.SchemaType = schemaType.String

		articles = append(articles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sample articles: %w", err)
	}

	return articles, nil
}

// validateReferentialIntegrity checks for broken references in sampled data
func (c *ConsistencyChecker) validateReferentialIntegrity(ctx context.Context, articles []SampleArticle) []ConsistencyIssue {
	var issues []ConsistencyIssue

	log.Printf("Validating referential integrity for %d articles", len(articles))

	for _, article := range articles {
		// Check if author exists
		if !c.authorExists(ctx, article.AuthorID) {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "broken_author_reference",
				Description: fmt.Sprintf("Article %d references non-existent author %d", article.ID, article.AuthorID),
				Severity:    "high",
				ArticleID:   &article.ID,
				UserID:      &article.AuthorID,
				Details: map[string]interface{}{
					"article_title": article.Title,
					"article_slug":  article.Slug,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check if category exists
		if !c.categoryExists(ctx, article.CategoryID) {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "broken_category_reference",
				Description: fmt.Sprintf("Article %d references non-existent category %d", article.ID, article.CategoryID),
				Severity:    "high",
				ArticleID:   &article.ID,
				CategoryID:  &article.CategoryID,
				Details: map[string]interface{}{
					"article_title": article.Title,
					"article_slug":  article.Slug,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check if translation group exists (if specified)
		if article.TranslationGroupID != nil && !c.translationGroupExists(ctx, *article.TranslationGroupID) {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "broken_translation_group_reference",
				Description: fmt.Sprintf("Article %d references non-existent translation group %d", article.ID, *article.TranslationGroupID),
				Severity:    "medium",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title":        article.Title,
					"translation_group_id": *article.TranslationGroupID,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for orphaned article tags
		orphanedTags := c.findOrphanedArticleTags(ctx, article.ID)
		for _, tagID := range orphanedTags {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "orphaned_article_tag",
				Description: fmt.Sprintf("Article %d has tag reference to non-existent tag %d", article.ID, tagID),
				Severity:    "medium",
				ArticleID:   &article.ID,
				TagID:       &tagID,
				Details: map[string]interface{}{
					"article_title": article.Title,
				},
				CreatedAt: time.Now(),
			})
		}
	}

	log.Printf("Found %d referential integrity issues", len(issues))
	return issues
}

// validateMultilingualConsistency checks multilingual content consistency
func (c *ConsistencyChecker) validateMultilingualConsistency(ctx context.Context, articles []SampleArticle) []ConsistencyIssue {
	var issues []ConsistencyIssue

	log.Printf("Validating multilingual consistency for %d articles", len(articles))

	// Group articles by translation group
	translationGroups := make(map[uint64][]SampleArticle)
	for _, article := range articles {
		if article.TranslationGroupID != nil {
			translationGroups[*article.TranslationGroupID] = append(translationGroups[*article.TranslationGroupID], article)
		}
	}

	for groupID, groupArticles := range translationGroups {
		if len(groupArticles) < 2 {
			continue // Skip groups with only one article
		}

		// Check for inconsistent publication status within translation group
		publishedCount := 0
		for _, article := range groupArticles {
			if article.Status == "published" {
				publishedCount++
			}
		}

		// If some but not all articles in a translation group are published, flag as inconsistent
		if publishedCount > 0 && publishedCount < len(groupArticles) {
			for _, article := range groupArticles {
				if article.Status != "published" {
					issues = append(issues, ConsistencyIssue{
						ID:          generateIssueID(),
						Type:        "translation_status_inconsistency",
						Description: fmt.Sprintf("Article %d in translation group %d has inconsistent status '%s' while other translations are published", article.ID, groupID, article.Status),
						Severity:    "medium",
						ArticleID:   &article.ID,
						Details: map[string]interface{}{
							"translation_group_id": groupID,
							"article_status":       article.Status,
							"published_count":      publishedCount,
							"total_count":          len(groupArticles),
						},
						CreatedAt: time.Now(),
					})
				}
			}
		}

		// Check for duplicate language codes within translation group
		languageCounts := make(map[string]int)
		for _, article := range groupArticles {
			languageCounts[article.LanguageCode]++
		}

		for langCode, count := range languageCounts {
			if count > 1 {
				issues = append(issues, ConsistencyIssue{
					ID:          generateIssueID(),
					Type:        "duplicate_language_in_translation_group",
					Description: fmt.Sprintf("Translation group %d has %d articles with language code '%s'", groupID, count, langCode),
					Severity:    "high",
					Details: map[string]interface{}{
						"translation_group_id": groupID,
						"language_code":        langCode,
						"duplicate_count":      count,
					},
					CreatedAt: time.Now(),
				})
			}
		}
	}

	log.Printf("Found %d multilingual consistency issues", len(issues))
	return issues
}

// validateSEOMetadataConsistency checks SEO metadata consistency
func (c *ConsistencyChecker) validateSEOMetadataConsistency(ctx context.Context, articles []SampleArticle) []ConsistencyIssue {
	var issues []ConsistencyIssue

	log.Printf("Validating SEO metadata consistency for %d articles", len(articles))

	for _, article := range articles {
		// Check for missing meta title
		if strings.TrimSpace(article.MetaTitle) == "" {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "missing_meta_title",
				Description: fmt.Sprintf("Article %d is missing meta title", article.ID),
				Severity:    "medium",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title": article.Title,
					"article_slug":  article.Slug,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for missing meta description
		if strings.TrimSpace(article.MetaDescription) == "" {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "missing_meta_description",
				Description: fmt.Sprintf("Article %d is missing meta description", article.ID),
				Severity:    "medium",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title": article.Title,
					"article_slug":  article.Slug,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for invalid schema type
		validSchemaTypes := map[string]bool{
			"NewsArticle": true,
			"Article":     true,
			"BlogPosting": true,
		}
		if article.SchemaType != "" && !validSchemaTypes[article.SchemaType] {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "invalid_schema_type",
				Description: fmt.Sprintf("Article %d has invalid schema type '%s'", article.ID, article.SchemaType),
				Severity:    "high",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title":   article.Title,
					"invalid_schema":  article.SchemaType,
					"valid_schemas":   []string{"NewsArticle", "Article", "BlogPosting"},
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for canonical URL format issues
		if article.CanonicalURL != "" && !c.isValidURL(article.CanonicalURL) {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "invalid_canonical_url",
				Description: fmt.Sprintf("Article %d has invalid canonical URL format", article.ID),
				Severity:    "medium",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title":  article.Title,
					"canonical_url":  article.CanonicalURL,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for meta title length (should be ≤ 60 characters)
		if len(article.MetaTitle) > 60 {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "meta_title_too_long",
				Description: fmt.Sprintf("Article %d has meta title longer than 60 characters (%d chars)", article.ID, len(article.MetaTitle)),
				Severity:    "low",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title":     article.Title,
					"meta_title_length": len(article.MetaTitle),
					"meta_title":        article.MetaTitle,
				},
				CreatedAt: time.Now(),
			})
		}

		// Check for meta description length (should be ≤ 160 characters)
		if len(article.MetaDescription) > 160 {
			issues = append(issues, ConsistencyIssue{
				ID:          generateIssueID(),
				Type:        "meta_description_too_long",
				Description: fmt.Sprintf("Article %d has meta description longer than 160 characters (%d chars)", article.ID, len(article.MetaDescription)),
				Severity:    "low",
				ArticleID:   &article.ID,
				Details: map[string]interface{}{
					"article_title":            article.Title,
					"meta_description_length":  len(article.MetaDescription),
					"meta_description":         article.MetaDescription,
				},
				CreatedAt: time.Now(),
			})
		}
	}

	log.Printf("Found %d SEO metadata consistency issues", len(issues))
	return issues
}

// Helper methods for validation

func (c *ConsistencyChecker) authorExists(ctx context.Context, authorID uint64) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND is_active = true)"
	err := c.db.QueryRowContext(ctx, query, authorID).Scan(&exists)
	return err == nil && exists
}

func (c *ConsistencyChecker) categoryExists(ctx context.Context, categoryID uint64) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)"
	err := c.db.QueryRowContext(ctx, query, categoryID).Scan(&exists)
	return err == nil && exists
}

func (c *ConsistencyChecker) translationGroupExists(ctx context.Context, groupID uint64) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM translation_groups WHERE id = $1)"
	err := c.db.QueryRowContext(ctx, query, groupID).Scan(&exists)
	return err == nil && exists
}

func (c *ConsistencyChecker) findOrphanedArticleTags(ctx context.Context, articleID uint64) []uint64 {
	query := `
		SELECT at.tag_id 
		FROM article_tags at 
		LEFT JOIN tags t ON at.tag_id = t.id 
		WHERE at.article_id = $1 AND t.id IS NULL
	`
	
	rows, err := c.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var orphanedTags []uint64
	for rows.Next() {
		var tagID uint64
		if err := rows.Scan(&tagID); err == nil {
			orphanedTags = append(orphanedTags, tagID)
		}
	}

	return orphanedTags
}

func (c *ConsistencyChecker) isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func (c *ConsistencyChecker) determineCheckStatus(issues []ConsistencyIssue) CheckStatus {
	if len(issues) == 0 {
		return CheckStatusPassed
	}

	// Check for high severity issues
	for _, issue := range issues {
		if issue.Severity == "high" {
			return CheckStatusFailed
		}
	}

	// If only medium or low severity issues, return warning
	return CheckStatusWarning
}

// Utility functions

func generateCheckID() string {
	return fmt.Sprintf("check_%d", time.Now().UnixNano())
}

func generateIssueID() string {
	return fmt.Sprintf("issue_%d", time.Now().UnixNano())
}