package services

import (
	"database/sql"
	"fmt"
	"strings"

	"high-performance-news-website/internal/models"
)

// MultilingualService handles multilingual content operations
type MultilingualService struct {
	db *sql.DB
}

// NewMultilingualService creates a new multilingual service
func NewMultilingualService(db *sql.DB) *MultilingualService {
	return &MultilingualService{
		db: db,
	}
}

// GetLanguages returns all supported languages
func (s *MultilingualService) GetLanguages() ([]models.Language, error) {
	query := `
		SELECT code, name, native_name, direction, is_active, sort_order, created_at
		FROM languages
		ORDER BY sort_order, name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query languages: %w", err)
	}
	defer rows.Close()

	var languages []models.Language
	for rows.Next() {
		var lang models.Language
		err := rows.Scan(
			&lang.Code,
			&lang.Name,
			&lang.NativeName,
			&lang.Direction,
			&lang.IsActive,
			&lang.SortOrder,
			&lang.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, lang)
	}

	return languages, nil
}

// GetActiveLanguages returns only active languages
func (s *MultilingualService) GetActiveLanguages() ([]models.Language, error) {
	query := `
		SELECT code, name, native_name, direction, is_active, sort_order, created_at
		FROM languages
		WHERE is_active = true
		ORDER BY sort_order, name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active languages: %w", err)
	}
	defer rows.Close()

	var languages []models.Language
	for rows.Next() {
		var lang models.Language
		err := rows.Scan(
			&lang.Code,
			&lang.Name,
			&lang.NativeName,
			&lang.Direction,
			&lang.IsActive,
			&lang.SortOrder,
			&lang.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}
		languages = append(languages, lang)
	}

	return languages, nil
}

// GetLanguageConfig returns the current language configuration
func (s *MultilingualService) GetLanguageConfig() (*models.LanguageConfig, error) {
	languages, err := s.GetActiveLanguages()
	if err != nil {
		return nil, err
	}

	config := &models.LanguageConfig{
		DefaultLanguage:  "en", // English is default
		FallbackLanguage: "en",
		ActiveLanguages:  make([]string, 0, len(languages)),
		RTLLanguages:     make([]string, 0),
	}

	for _, lang := range languages {
		config.ActiveLanguages = append(config.ActiveLanguages, lang.Code)
		if lang.Direction == "rtl" {
			config.RTLLanguages = append(config.RTLLanguages, lang.Code)
		}
	}

	return config, nil
}

// CreateTranslationGroup creates a new translation group and links content
func (s *MultilingualService) CreateTranslationGroup(groupType string, contentIDs []uint64) (uint64, error) {
	if len(contentIDs) < 2 {
		return 0, fmt.Errorf("translation group must contain at least 2 items")
	}

	// Validate group type
	validTypes := map[string]bool{
		"article":  true,
		"category": true,
		"tag":      true,
	}
	if !validTypes[groupType] {
		return 0, fmt.Errorf("invalid group type: %s", groupType)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create translation group
	var groupID uint64
	err = tx.QueryRow(
		"INSERT INTO translation_groups (group_type) VALUES ($1) RETURNING id",
		groupType,
	).Scan(&groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to create translation group: %w", err)
	}

	// Link content to the group
	var tableName string
	switch groupType {
	case "article":
		tableName = "articles"
	case "category":
		tableName = "categories"
	case "tag":
		tableName = "tags"
	}

	for _, contentID := range contentIDs {
		query := fmt.Sprintf(
			"UPDATE %s SET translation_group_id = $1 WHERE id = $2",
			tableName,
		)
		_, err = tx.Exec(query, groupID, contentID)
		if err != nil {
			return 0, fmt.Errorf("failed to link content %d to translation group: %w", contentID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return groupID, nil
}

// GetArticleTranslations returns an article with its translations
func (s *MultilingualService) GetArticleTranslations(articleID uint64) (*models.MultilingualArticle, error) {
	query := `
		SELECT 
			a.id, a.title, a.slug, a.language_code, a.translation_group_id,
			a.published_at, a.status, a.content, a.excerpt, a.author_id, a.category_id,
			a.view_count, a.like_count, a.dislike_count, a.created_at, a.updated_at,
			a.meta_title, a.meta_description, a.canonical_url, a.schema_type, a.auto_linking,
			l.name as language_name, l.native_name as language_native_name, l.direction as language_direction,
			COALESCE(
				ARRAY_AGG(
					JSON_BUILD_OBJECT(
						'id', ta.id,
						'title', ta.title,
						'slug', ta.slug,
						'language_code', ta.language_code,
						'language_name', tl.name,
						'language_native_name', tl.native_name
					) ORDER BY tl.sort_order
				) FILTER (WHERE ta.id != a.id AND ta.id IS NOT NULL),
				ARRAY[]::json[]
			) as translations
		FROM articles a
		LEFT JOIN languages l ON a.language_code = l.code
		LEFT JOIN articles ta ON a.translation_group_id = ta.translation_group_id
		LEFT JOIN languages tl ON ta.language_code = tl.code
		WHERE a.id = $1
		GROUP BY a.id, a.title, a.slug, a.language_code, a.translation_group_id,
				 a.published_at, a.status, a.content, a.excerpt, a.author_id, a.category_id,
				 a.view_count, a.like_count, a.dislike_count, a.created_at, a.updated_at,
				 a.meta_title, a.meta_description, a.canonical_url, a.schema_type, a.auto_linking,
				 l.name, l.native_name, l.direction
	`

	var article models.MultilingualArticle
	var translationsJSON []byte

	err := s.db.QueryRow(query, articleID).Scan(
		&article.ID,
		&article.Title,
		&article.Slug,
		&article.LanguageCode,
		&article.TranslationGroupID,
		&article.PublishedAt,
		&article.Status,
		&article.Content,
		&article.Excerpt,
		&article.AuthorID,
		&article.CategoryID,
		&article.ViewCount,
		&article.LikeCount,
		&article.DislikeCount,
		&article.CreatedAt,
		&article.UpdatedAt,
		&article.SEOData.MetaTitle,
		&article.SEOData.MetaDescription,
		&article.SEOData.CanonicalURL,
		&article.SEOData.SchemaType,
		&article.AutoLinking,
		&article.LanguageName,
		&article.LanguageNativeName,
		&article.LanguageDirection,
		&translationsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to query article translations: %w", err)
	}

	// Parse translations JSON
	if len(translationsJSON) > 0 {
		if err := article.Translations.Scan(translationsJSON); err != nil {
			return nil, fmt.Errorf("failed to parse translations: %w", err)
		}
	}

	return &article, nil
}

// GetArticlesByLanguage returns articles in a specific language with fallback
func (s *MultilingualService) GetArticlesByLanguage(languageCode, fallbackLanguage string, limit, offset int) ([]models.MultilingualArticle, error) {
	query := `
		WITH language_articles AS (
			SELECT a.id, a.title, a.slug, a.excerpt, a.language_code, a.published_at, 
				   a.author_id, a.category_id, a.view_count, a.like_count, a.dislike_count,
				   false as is_fallback
			FROM articles a
			WHERE a.language_code = $1 
			AND a.status = 'published'
			ORDER BY a.published_at DESC
			LIMIT $3 OFFSET $4
		),
		fallback_articles AS (
			SELECT a.id, a.title, a.slug, a.excerpt, a.language_code, a.published_at,
				   a.author_id, a.category_id, a.view_count, a.like_count, a.dislike_count,
				   true as is_fallback
			FROM articles a
			WHERE a.language_code = $2 
			AND a.status = 'published'
			AND NOT EXISTS (
				SELECT 1 FROM language_articles la WHERE la.id = a.id
			)
			ORDER BY a.published_at DESC
			LIMIT ($3 - (SELECT COUNT(*) FROM language_articles))
		)
		SELECT * FROM language_articles
		UNION ALL
		SELECT * FROM fallback_articles
		ORDER BY published_at DESC
	`

	rows, err := s.db.Query(query, languageCode, fallbackLanguage, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by language: %w", err)
	}
	defer rows.Close()

	var articles []models.MultilingualArticle
	for rows.Next() {
		var article models.MultilingualArticle
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.LanguageCode,
			&article.PublishedAt,
			&article.AuthorID,
			&article.CategoryID,
			&article.ViewCount,
			&article.LikeCount,
			&article.DislikeCount,
			&article.IsFallback,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
	}

	return articles, nil
}

// GetCategoriesByLanguage returns categories in a specific language
func (s *MultilingualService) GetCategoriesByLanguage(languageCode string) ([]models.MultilingualCategory, error) {
	query := `
		SELECT 
			c.id, c.name, c.slug, c.description, c.parent_id, c.sort_order,
			c.language_code, c.translation_group_id, c.created_at, c.updated_at,
			l.name as language_name, l.native_name as language_native_name, l.direction as language_direction,
			COALESCE(
				ARRAY_AGG(
					JSON_BUILD_OBJECT(
						'id', tc.id,
						'name', tc.name,
						'slug', tc.slug,
						'language_code', tc.language_code,
						'language_name', tl.name,
						'language_native_name', tl.native_name
					) ORDER BY tl.sort_order
				) FILTER (WHERE tc.id != c.id AND tc.id IS NOT NULL),
				ARRAY[]::json[]
			) as translations
		FROM categories c
		LEFT JOIN languages l ON c.language_code = l.code
		LEFT JOIN categories tc ON c.translation_group_id = tc.translation_group_id
		LEFT JOIN languages tl ON tc.language_code = tl.code
		WHERE c.language_code = $1
		GROUP BY c.id, c.name, c.slug, c.description, c.parent_id, c.sort_order,
				 c.language_code, c.translation_group_id, c.created_at, c.updated_at,
				 l.name, l.native_name, l.direction
		ORDER BY c.sort_order, c.name
	`

	rows, err := s.db.Query(query, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories by language: %w", err)
	}
	defer rows.Close()

	var categories []models.MultilingualCategory
	for rows.Next() {
		var category models.MultilingualCategory
		var translationsJSON []byte

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&category.ParentID,
			&category.SortOrder,
			&category.LanguageCode,
			&category.TranslationGroupID,
			&category.CreatedAt,
			&category.UpdatedAt,
			&category.LanguageName,
			&category.LanguageNativeName,
			&category.LanguageDirection,
			&translationsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Parse translations JSON
		if len(translationsJSON) > 0 {
			if err := category.Translations.Scan(translationsJSON); err != nil {
				return nil, fmt.Errorf("failed to parse translations: %w", err)
			}
		}

		categories = append(categories, category)
	}

	return categories, nil
}

// GetTagsByLanguage returns tags in a specific language
func (s *MultilingualService) GetTagsByLanguage(languageCode string) ([]models.MultilingualTag, error) {
	query := `
		SELECT 
			t.id, t.name, t.slug, t.description, t.keywords, t.color,
			t.language_code, t.translation_group_id, t.created_at, t.updated_at,
			l.name as language_name, l.native_name as language_native_name, l.direction as language_direction,
			COALESCE(
				ARRAY_AGG(
					JSON_BUILD_OBJECT(
						'id', tt.id,
						'name', tt.name,
						'slug', tt.slug,
						'language_code', tt.language_code,
						'language_name', tl.name,
						'language_native_name', tl.native_name
					) ORDER BY tl.sort_order
				) FILTER (WHERE tt.id != t.id AND tt.id IS NOT NULL),
				ARRAY[]::json[]
			) as translations
		FROM tags t
		LEFT JOIN languages l ON t.language_code = l.code
		LEFT JOIN tags tt ON t.translation_group_id = tt.translation_group_id
		LEFT JOIN languages tl ON tt.language_code = tl.code
		WHERE t.language_code = $1
		GROUP BY t.id, t.name, t.slug, t.description, t.keywords, t.color,
				 t.language_code, t.translation_group_id, t.created_at, t.updated_at,
				 l.name, l.native_name, l.direction
		ORDER BY t.name
	`

	rows, err := s.db.Query(query, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags by language: %w", err)
	}
	defer rows.Close()

	var tags []models.MultilingualTag
	for rows.Next() {
		var tag models.MultilingualTag
		var translationsJSON []byte
		var keywordsJSON []byte

		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Description,
			&keywordsJSON,
			&tag.Color,
			&tag.LanguageCode,
			&tag.TranslationGroupID,
			&tag.CreatedAt,
			&tag.UpdatedAt,
			&tag.LanguageName,
			&tag.LanguageNativeName,
			&tag.LanguageDirection,
			&translationsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		// Parse keywords JSON
		if len(keywordsJSON) > 0 {
			if err := tag.Tag.Scan(keywordsJSON); err != nil {
				return nil, fmt.Errorf("failed to parse keywords: %w", err)
			}
		}

		// Parse translations JSON
		if len(translationsJSON) > 0 {
			if err := tag.Translations.Scan(translationsJSON); err != nil {
				return nil, fmt.Errorf("failed to parse translations: %w", err)
			}
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// GenerateLanguageRouteInfo generates routing information for a given content and language
func (s *MultilingualService) GenerateLanguageRouteInfo(contentType, slug, languageCode string) (*models.LanguageRouteInfo, error) {
	config, err := s.GetLanguageConfig()
	if err != nil {
		return nil, err
	}

	isDefault := languageCode == config.DefaultLanguage
	var urlPrefix string
	if !isDefault {
		urlPrefix = "/" + languageCode
	}

	// Get language info
	var direction string
	for _, lang := range config.RTLLanguages {
		if lang == languageCode {
			direction = "rtl"
			break
		}
	}
	if direction == "" {
		direction = "ltr"
	}

	// Generate alternate URLs for all active languages
	alternateURLs := make(map[string]string)
	for _, langCode := range config.ActiveLanguages {
		if langCode == languageCode {
			continue
		}

		var prefix string
		if langCode != config.DefaultLanguage {
			prefix = "/" + langCode
		}

		alternateURLs[langCode] = fmt.Sprintf("%s/%s/%s", prefix, contentType, slug)
	}

	return &models.LanguageRouteInfo{
		LanguageCode:  languageCode,
		IsDefault:     isDefault,
		URLPrefix:     urlPrefix,
		Direction:     direction,
		AlternateURLs: alternateURLs,
	}, nil
}

// ValidateLanguageCode checks if a language code is supported and active
func (s *MultilingualService) ValidateLanguageCode(languageCode string) error {
	if languageCode == "" {
		return fmt.Errorf("language code is required")
	}

	if len(languageCode) != 2 {
		return fmt.Errorf("language code must be exactly 2 characters")
	}

	query := "SELECT COUNT(*) FROM languages WHERE code = $1 AND is_active = true"
	var count int
	err := s.db.QueryRow(query, languageCode).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate language code: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("language code '%s' is not supported or not active", languageCode)
	}

	return nil
}

// GetDefaultLanguage returns the default language code
func (s *MultilingualService) GetDefaultLanguage() string {
	return "en" // English is the default language
}

// IsRTLLanguage checks if a language is right-to-left
func (s *MultilingualService) IsRTLLanguage(languageCode string) (bool, error) {
	query := "SELECT direction FROM languages WHERE code = $1"
	var direction string
	err := s.db.QueryRow(query, languageCode).Scan(&direction)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("language code '%s' not found", languageCode)
		}
		return false, fmt.Errorf("failed to check language direction: %w", err)
	}

	return direction == "rtl", nil
}

// ExtractLanguageFromURL extracts language code from URL path
func ExtractLanguageFromURL(path string) (string, string) {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Split path into segments
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return "en", path // Default to English
	}

	// Check if first segment is a language code
	firstSegment := segments[0]
	if len(firstSegment) == 2 {
		// Assume it's a language code, return it and the remaining path
		remainingPath := strings.Join(segments[1:], "/")
		return firstSegment, remainingPath
	}

	// No language code in URL, return default
	return "en", path
}

// LanguageInfo holds information about a language for templates
type LanguageInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	Direction  string `json:"direction"`
	URL        string `json:"url"`
	IsActive   bool   `json:"is_active"`
}

// AlternateURL represents an alternate language URL for hreflang
type AlternateURL struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// GenerateAlternateURLs generates alternate URLs for all active languages
func (s *MultilingualService) GenerateAlternateURLs(baseURL, currentPath string) ([]AlternateURL, error) {
	languages, err := s.GetActiveLanguages()
	if err != nil {
		return nil, err
	}

	var alternates []AlternateURL

	// Clean the path - remove any existing language prefix
	cleanPath := currentPath
	if len(cleanPath) > 3 && cleanPath[0] == '/' {
		// Check if path starts with a language code like /en/, /de/, etc.
		possibleLang := cleanPath[1:3]
		if len(cleanPath) > 3 && cleanPath[3] == '/' {
			// Validate if it's actually a language code
			for _, lang := range languages {
				if lang.Code == possibleLang {
					cleanPath = cleanPath[3:] // Remove /xx prefix
					break
				}
			}
		} else if len(cleanPath) == 3 {
			// Path is just /xx
			for _, lang := range languages {
				if lang.Code == possibleLang {
					cleanPath = "/"
					break
				}
			}
		}
	}

	// Generate alternate URLs for each language
	for _, lang := range languages {
		url := fmt.Sprintf("%s/%s%s", baseURL, lang.Code, cleanPath)
		// Clean up double slashes
		url = strings.ReplaceAll(url, "//", "/")
		url = strings.Replace(url, ":/", "://", 1) // Fix protocol

		alternates = append(alternates, AlternateURL{
			Lang: lang.Code,
			URL:  url,
		})
	}

	// Add x-default pointing to English
	defaultURL := fmt.Sprintf("%s/en%s", baseURL, cleanPath)
	defaultURL = strings.ReplaceAll(defaultURL, "//", "/")
	defaultURL = strings.Replace(defaultURL, ":/", "://", 1)
	alternates = append(alternates, AlternateURL{
		Lang: "x-default",
		URL:  defaultURL,
	})

	return alternates, nil
}

// GenerateCanonicalURL generates the canonical URL for a page
func (s *MultilingualService) GenerateCanonicalURL(baseURL, languageCode, path string) string {
	url := fmt.Sprintf("%s/%s%s", baseURL, languageCode, path)
	// Clean up double slashes
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.Replace(url, ":/", "://", 1) // Fix protocol
	return url
}

// GetAvailableLanguagesForTemplate returns language info formatted for templates
func (s *MultilingualService) GetAvailableLanguagesForTemplate(baseURL, currentPath string) ([]LanguageInfo, error) {
	languages, err := s.GetActiveLanguages()
	if err != nil {
		return nil, err
	}

	// Clean the path - remove any existing language prefix
	cleanPath := currentPath
	if len(cleanPath) > 3 && cleanPath[0] == '/' {
		possibleLang := cleanPath[1:3]
		if len(cleanPath) > 3 && cleanPath[3] == '/' {
			for _, lang := range languages {
				if lang.Code == possibleLang {
					cleanPath = cleanPath[3:]
					break
				}
			}
		} else if len(cleanPath) == 3 {
			for _, lang := range languages {
				if lang.Code == possibleLang {
					cleanPath = "/"
					break
				}
			}
		}
	}

	var result []LanguageInfo
	for _, lang := range languages {
		url := fmt.Sprintf("/%s%s", lang.Code, cleanPath)
		// Clean up double slashes
		url = strings.ReplaceAll(url, "//", "/")

		result = append(result, LanguageInfo{
			Code:       lang.Code,
			Name:       lang.Name,
			NativeName: lang.NativeName,
			Direction:  lang.Direction,
			URL:        url,
			IsActive:   lang.IsActive,
		})
	}

	return result, nil
}

// GetLanguageByCode returns a specific language by its code
func (s *MultilingualService) GetLanguageByCode(code string) (*models.Language, error) {
	query := `
		SELECT code, name, native_name, direction, is_active, sort_order, created_at
		FROM languages
		WHERE code = $1
	`

	var lang models.Language
	err := s.db.QueryRow(query, code).Scan(
		&lang.Code,
		&lang.Name,
		&lang.NativeName,
		&lang.Direction,
		&lang.IsActive,
		&lang.SortOrder,
		&lang.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("language '%s' not found", code)
		}
		return nil, fmt.Errorf("failed to get language: %w", err)
	}

	return &lang, nil
}

// GetLanguageNativeName returns the native name for a language code
func (s *MultilingualService) GetLanguageNativeName(code string) string {
	nativeNames := map[string]string{
		"en": "English",
		"de": "Deutsch",
		"fr": "Français",
		"es": "Español",
		"ar": "العربية",
	}
	if name, ok := nativeNames[code]; ok {
		return name
	}
	return code
}

// GetLanguageName returns the English name for a language code
func (s *MultilingualService) GetLanguageName(code string) string {
	names := map[string]string{
		"en": "English",
		"de": "German",
		"fr": "French",
		"es": "Spanish",
		"ar": "Arabic",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
