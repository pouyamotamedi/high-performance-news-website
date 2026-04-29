package services

import (
	"database/sql"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// VersioningService handles article version management
type VersioningService struct {
	db *sql.DB
}

// NewVersioningService creates a new versioning service
func NewVersioningService(db *sql.DB) *VersioningService {
	return &VersioningService{
		db: db,
	}
}

// CreateVersion creates a new version of an article
func (vs *VersioningService) CreateVersion(article *models.Article, changeSummary string, createdBy uint64) (*models.ArticleVersion, error) {
	// Get next version number
	var nextVersion int
	err := vs.db.QueryRow(`
		SELECT COALESCE(MAX(version_number), 0) + 1 
		FROM article_versions 
		WHERE article_id = $1
	`, article.ID).Scan(&nextVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	// Create version record
	version := &models.ArticleVersion{
		ArticleID:          article.ID,
		VersionNumber:      nextVersion,
		Title:              article.Title,
		Slug:               article.Slug,
		Content:            article.Content,
		Excerpt:            article.Excerpt,
		AuthorID:           article.AuthorID,
		CategoryID:         article.CategoryID,
		Status:             article.Status,
		PublishedAt:        article.PublishedAt,
		MetaTitle:          article.SEOData.MetaTitle,
		MetaDescription:    article.SEOData.MetaDescription,
		CanonicalURL:       article.SEOData.CanonicalURL,
		SchemaType:         article.SEOData.SchemaType,
		LanguageCode:       article.LanguageCode,
		TranslationGroupID: article.TranslationGroupID,
		AutoLinking:        article.AutoLinking,
		ChangeSummary:      changeSummary,
		CreatedBy:          createdBy,
		CreatedAt:          time.Now(),
	}

	// Insert version
	err = vs.db.QueryRow(`
		INSERT INTO article_versions (
			article_id, version_number, title, slug, content, excerpt,
			author_id, category_id, status, published_at,
			meta_title, meta_description, canonical_url, schema_type,
			language_code, translation_group_id, auto_linking,
			change_summary, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) RETURNING id
	`,
		version.ArticleID, version.VersionNumber, version.Title, version.Slug,
		version.Content, version.Excerpt, version.AuthorID, version.CategoryID,
		version.Status, version.PublishedAt, version.MetaTitle, version.MetaDescription,
		version.CanonicalURL, version.SchemaType, version.LanguageCode,
		version.TranslationGroupID, version.AutoLinking, version.ChangeSummary,
		version.CreatedBy, version.CreatedAt,
	).Scan(&version.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create article version: %w", err)
	}

	return version, nil
}

// GetVersionHistory returns all versions of an article
func (vs *VersioningService) GetVersionHistory(articleID uint64) ([]models.ArticleVersion, error) {
	rows, err := vs.db.Query(`
		SELECT id, article_id, version_number, title, slug, content, excerpt,
			   author_id, category_id, status, published_at,
			   meta_title, meta_description, canonical_url, schema_type,
			   language_code, translation_group_id, auto_linking,
			   change_summary, created_by, created_at
		FROM article_versions
		WHERE article_id = $1
		ORDER BY version_number DESC
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}
	defer rows.Close()

	var versions []models.ArticleVersion
	for rows.Next() {
		var version models.ArticleVersion
		err := rows.Scan(
			&version.ID, &version.ArticleID, &version.VersionNumber,
			&version.Title, &version.Slug, &version.Content, &version.Excerpt,
			&version.AuthorID, &version.CategoryID, &version.Status, &version.PublishedAt,
			&version.MetaTitle, &version.MetaDescription, &version.CanonicalURL,
			&version.SchemaType, &version.LanguageCode, &version.TranslationGroupID,
			&version.AutoLinking, &version.ChangeSummary, &version.CreatedBy, &version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, version)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating versions: %w", err)
	}

	return versions, nil
}

// GetVersion returns a specific version of an article
func (vs *VersioningService) GetVersion(articleID uint64, versionNumber int) (*models.ArticleVersion, error) {
	var version models.ArticleVersion
	err := vs.db.QueryRow(`
		SELECT id, article_id, version_number, title, slug, content, excerpt,
			   author_id, category_id, status, published_at,
			   meta_title, meta_description, canonical_url, schema_type,
			   language_code, translation_group_id, auto_linking,
			   change_summary, created_by, created_at
		FROM article_versions
		WHERE article_id = $1 AND version_number = $2
	`, articleID, versionNumber).Scan(
		&version.ID, &version.ArticleID, &version.VersionNumber,
		&version.Title, &version.Slug, &version.Content, &version.Excerpt,
		&version.AuthorID, &version.CategoryID, &version.Status, &version.PublishedAt,
		&version.MetaTitle, &version.MetaDescription, &version.CanonicalURL,
		&version.SchemaType, &version.LanguageCode, &version.TranslationGroupID,
		&version.AutoLinking, &version.ChangeSummary, &version.CreatedBy, &version.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("version %d not found for article %d", versionNumber, articleID)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &version, nil
}

// GetLatestVersion returns the latest version of an article
func (vs *VersioningService) GetLatestVersion(articleID uint64) (*models.ArticleVersion, error) {
	var version models.ArticleVersion
	err := vs.db.QueryRow(`
		SELECT id, article_id, version_number, title, slug, content, excerpt,
			   author_id, category_id, status, published_at,
			   meta_title, meta_description, canonical_url, schema_type,
			   language_code, translation_group_id, auto_linking,
			   change_summary, created_by, created_at
		FROM article_versions
		WHERE article_id = $1
		ORDER BY version_number DESC
		LIMIT 1
	`, articleID).Scan(
		&version.ID, &version.ArticleID, &version.VersionNumber,
		&version.Title, &version.Slug, &version.Content, &version.Excerpt,
		&version.AuthorID, &version.CategoryID, &version.Status, &version.PublishedAt,
		&version.MetaTitle, &version.MetaDescription, &version.CanonicalURL,
		&version.SchemaType, &version.LanguageCode, &version.TranslationGroupID,
		&version.AutoLinking, &version.ChangeSummary, &version.CreatedBy, &version.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no versions found for article %d", articleID)
		}
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	return &version, nil
}

// CompareVersions returns the differences between two versions
func (vs *VersioningService) CompareVersions(articleID uint64, version1, version2 int) (*VersionComparison, error) {
	v1, err := vs.GetVersion(articleID, version1)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", version1, err)
	}

	v2, err := vs.GetVersion(articleID, version2)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", version2, err)
	}

	comparison := &VersionComparison{
		Version1: v1,
		Version2: v2,
		Changes:  make(map[string]VersionChange),
	}

	// Compare fields
	if v1.Title != v2.Title {
		comparison.Changes["title"] = VersionChange{
			Field:    "title",
			OldValue: v1.Title,
			NewValue: v2.Title,
		}
	}

	if v1.Content != v2.Content {
		comparison.Changes["content"] = VersionChange{
			Field:    "content",
			OldValue: v1.Content,
			NewValue: v2.Content,
		}
	}

	if v1.Excerpt != v2.Excerpt {
		comparison.Changes["excerpt"] = VersionChange{
			Field:    "excerpt",
			OldValue: v1.Excerpt,
			NewValue: v2.Excerpt,
		}
	}

	if v1.Status != v2.Status {
		comparison.Changes["status"] = VersionChange{
			Field:    "status",
			OldValue: v1.Status,
			NewValue: v2.Status,
		}
	}

	if v1.MetaTitle != v2.MetaTitle {
		comparison.Changes["meta_title"] = VersionChange{
			Field:    "meta_title",
			OldValue: v1.MetaTitle,
			NewValue: v2.MetaTitle,
		}
	}

	if v1.MetaDescription != v2.MetaDescription {
		comparison.Changes["meta_description"] = VersionChange{
			Field:    "meta_description",
			OldValue: v1.MetaDescription,
			NewValue: v2.MetaDescription,
		}
	}

	return comparison, nil
}

// RestoreVersion restores an article to a specific version
func (vs *VersioningService) RestoreVersion(articleID uint64, versionNumber int, restoredBy uint64) error {
	// Get the version to restore
	version, err := vs.GetVersion(articleID, versionNumber)
	if err != nil {
		return fmt.Errorf("failed to get version to restore: %w", err)
	}

	// Update the main article with version data
	_, err = vs.db.Exec(`
		UPDATE articles SET
			title = $2, slug = $3, content = $4, excerpt = $5,
			author_id = $6, category_id = $7, status = $8,
			meta_title = $9, meta_description = $10, canonical_url = $11,
			schema_type = $12, language_code = $13, translation_group_id = $14,
			auto_linking = $15, updated_at = NOW()
		WHERE id = $1
	`,
		articleID, version.Title, version.Slug, version.Content, version.Excerpt,
		version.AuthorID, version.CategoryID, version.Status,
		version.MetaTitle, version.MetaDescription, version.CanonicalURL,
		version.SchemaType, version.LanguageCode, version.TranslationGroupID,
		version.AutoLinking,
	)

	if err != nil {
		return fmt.Errorf("failed to restore article to version %d: %w", versionNumber, err)
	}

	// Create a new version record for the restoration
	changeSummary := fmt.Sprintf("Restored to version %d", versionNumber)
	_, err = vs.CreateVersion(&models.Article{
		ID:                 articleID,
		Title:              version.Title,
		Slug:               version.Slug,
		Content:            version.Content,
		Excerpt:            version.Excerpt,
		AuthorID:           version.AuthorID,
		CategoryID:         version.CategoryID,
		Status:             version.Status,
		LanguageCode:       version.LanguageCode,
		TranslationGroupID: version.TranslationGroupID,
		AutoLinking:        version.AutoLinking,
		SEOData: models.SEOData{
			MetaTitle:       version.MetaTitle,
			MetaDescription: version.MetaDescription,
			CanonicalURL:    version.CanonicalURL,
			SchemaType:      version.SchemaType,
		},
	}, changeSummary, restoredBy)

	if err != nil {
		return fmt.Errorf("failed to create restoration version: %w", err)
	}

	return nil
}

// DeleteOldVersions deletes versions older than the specified number of days
func (vs *VersioningService) DeleteOldVersions(daysToKeep int) (int, error) {
	result, err := vs.db.Exec(`
		DELETE FROM article_versions 
		WHERE created_at < NOW() - INTERVAL '%d days'
		AND version_number NOT IN (
			SELECT MAX(version_number) 
			FROM article_versions 
			GROUP BY article_id
		)
	`, daysToKeep)

	if err != nil {
		return 0, fmt.Errorf("failed to delete old versions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetVersionStats returns statistics about article versions
func (vs *VersioningService) GetVersionStats() (*VersionStats, error) {
	var stats VersionStats

	// Get total versions
	err := vs.db.QueryRow(`
		SELECT COUNT(*) FROM article_versions
	`).Scan(&stats.TotalVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total versions: %w", err)
	}

	// Get articles with versions
	err = vs.db.QueryRow(`
		SELECT COUNT(DISTINCT article_id) FROM article_versions
	`).Scan(&stats.ArticlesWithVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles with versions: %w", err)
	}

	// Get average versions per article
	err = vs.db.QueryRow(`
		SELECT AVG(version_count)::DECIMAL(10,2)
		FROM (
			SELECT COUNT(*) as version_count 
			FROM article_versions 
			GROUP BY article_id
		) as version_counts
	`).Scan(&stats.AverageVersionsPerArticle)
	if err != nil {
		return nil, fmt.Errorf("failed to get average versions per article: %w", err)
	}

	// Get most versioned article
	err = vs.db.QueryRow(`
		SELECT article_id, COUNT(*) as version_count
		FROM article_versions
		GROUP BY article_id
		ORDER BY version_count DESC
		LIMIT 1
	`).Scan(&stats.MostVersionedArticleID, &stats.MaxVersionsForArticle)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get most versioned article: %w", err)
	}

	return &stats, nil
}

// VersionComparison represents a comparison between two versions
type VersionComparison struct {
	Version1 *models.ArticleVersion    `json:"version1"`
	Version2 *models.ArticleVersion    `json:"version2"`
	Changes  map[string]VersionChange  `json:"changes"`
}

// VersionChange represents a change between versions
type VersionChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// VersionStats represents statistics about article versions
type VersionStats struct {
	TotalVersions             int     `json:"total_versions"`
	ArticlesWithVersions      int     `json:"articles_with_versions"`
	AverageVersionsPerArticle float64 `json:"average_versions_per_article"`
	MostVersionedArticleID    uint64  `json:"most_versioned_article_id"`
	MaxVersionsForArticle     int     `json:"max_versions_for_article"`
}