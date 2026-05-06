package testing

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// AITestDataGenerator generates realistic test data using AI
type AITestDataGenerator struct {
	llmClient    LLMClient
	dataTemplates map[string]DataTemplate
	languages    []Language
	config       *AIDataConfig
}

// AIDataConfig configuration for AI data generation
type AIDataConfig struct {
	DefaultLanguage string            `json:"default_language"`
	MaxArticles     int               `json:"max_articles"`
	ContentLength   ContentLengthRange `json:"content_length"`
	Relationships   RelationshipConfig `json:"relationships"`
}

// ContentLengthRange defines content length parameters
type ContentLengthRange struct {
	MinWords int `json:"min_words"`
	MaxWords int `json:"max_words"`
}

// RelationshipConfig defines relationship generation parameters
type RelationshipConfig struct {
	TranslationRate    float64 `json:"translation_rate"`
	CategoryRate       float64 `json:"category_rate"`
	TagsPerArticle     int     `json:"tags_per_article"`
	AuthorsCount       int     `json:"authors_count"`
	CategoriesCount    int     `json:"categories_count"`
}

// DataTemplate represents a template for generating specific data types
type DataTemplate struct {
	Type        string            `json:"type"`
	Language    string            `json:"language"`
	Patterns    []string          `json:"patterns"`
	Vocabulary  []string          `json:"vocabulary"`
	Constraints map[string]string `json:"constraints"`
}

// Language represents language-specific configuration
type Language struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Direction    string   `json:"direction"`
	CharacterSet string   `json:"character_set"`
	Fonts        []string `json:"fonts"`
	Vocabulary   []string `json:"vocabulary"`
}

// GeneratedTestData represents a complete set of test data
type GeneratedTestData struct {
	Articles    []TestArticle    `json:"articles"`
	Authors     []TestAuthor     `json:"authors"`
	Categories  []TestCategory   `json:"categories"`
	Tags        []TestTag        `json:"tags"`
	Users       []TestUser       `json:"users"`
	Comments    []TestComment    `json:"comments"`
	GeneratedAt time.Time        `json:"generated_at"`
	Metadata    GenerationMetadata `json:"metadata"`
}

// TestArticle represents a test article with realistic content
type TestArticle struct {
	ID              int64             `json:"id"`
	Title           string            `json:"title"`
	Slug            string            `json:"slug"`
	Content         string            `json:"content"`
	Summary         string            `json:"summary"`
	Language        string            `json:"language"`
	AuthorID        int64             `json:"author_id"`
	CategoryID      int64             `json:"category_id"`
	Tags            []string          `json:"tags"`
	Status          string            `json:"status"`
	PublishedAt     *time.Time        `json:"published_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	SEOMetadata     SEOMetadata       `json:"seo_metadata"`
	TranslationOf   *int64            `json:"translation_of"`
	ViewCount       int               `json:"view_count"`
	ShareCount      int               `json:"share_count"`
	CommentCount    int               `json:"comment_count"`
}

// TestAuthor represents a test author
type TestAuthor struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Bio         string    `json:"bio"`
	Avatar      string    `json:"avatar"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"created_at"`
	ArticleCount int      `json:"article_count"`
}

// TestCategory represents a test category
type TestCategory struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	ParentID    *int64    `json:"parent_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TestTag represents a test tag
type TestTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Language  string    `json:"language"`
	UsageCount int      `json:"usage_count"`
	CreatedAt time.Time `json:"created_at"`
}

// TestUser represents a test user
type TestUser struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

// TestComment represents a test comment
type TestComment struct {
	ID        int64     `json:"id"`
	ArticleID int64     `json:"article_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	Language  string    `json:"language"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ParentID  *int64    `json:"parent_id"`
}

// SEOMetadata represents SEO metadata for articles
type SEOMetadata struct {
	MetaTitle       string            `json:"meta_title"`
	MetaDescription string            `json:"meta_description"`
	Keywords        []string          `json:"keywords"`
	CanonicalURL    string            `json:"canonical_url"`
	SchemaMarkup    map[string]interface{} `json:"schema_markup"`
	OpenGraphTags   map[string]string `json:"open_graph_tags"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	TotalArticles     int           `json:"total_articles"`
	LanguageDistribution map[string]int `json:"language_distribution"`
	GenerationTime    time.Duration `json:"generation_time"`
	AIModel           string        `json:"ai_model"`
	Seed              int64         `json:"seed"`
}

// NewAITestDataGenerator creates a new AI test data generator
func NewAITestDataGenerator(llmClient LLMClient, config *AIDataConfig) *AITestDataGenerator {
	return &AITestDataGenerator{
		llmClient: llmClient,
		config:    config,
		languages: []Language{
			{
				Code:         "en",
				Name:         "English",
				Direction:    "ltr",
				CharacterSet: "latin",
				Vocabulary:   []string{"news", "article", "breaking", "update", "report", "analysis", "opinion", "editorial"},
			},
			{
				Code:         "de",
				Name:         "German",
				Direction:    "ltr",
				CharacterSet: "latin",
				Vocabulary:   []string{"Nachrichten", "Artikel", "Bericht", "Analyse", "Meinung", "Eilmeldung", "Aktuell", "Information"},
			},
			{
				Code:         "fr",
				Name:         "French",
				Direction:    "ltr",
				CharacterSet: "latin",
				Vocabulary:   []string{"actualités", "article", "rapport", "analyse", "opinion", "urgent", "information", "nouvelles"},
			},
			{
				Code:         "es",
				Name:         "Spanish",
				Direction:    "ltr",
				CharacterSet: "latin",
				Vocabulary:   []string{"noticias", "artículo", "informe", "análisis", "opinión", "urgente", "información", "actualidad"},
			},
			{
				Code:         "ar",
				Name:         "Arabic",
				Direction:    "rtl",
				CharacterSet: "arabic",
				Vocabulary:   []string{"خبر", "مقال", "تقرير", "تحليل", "رأي", "عاجل", "إعلام", "معلومات"},
			},
		},
		dataTemplates: make(map[string]DataTemplate),
	}
}

// GenerateRealisticTestData generates comprehensive test data using AI
func (g *AITestDataGenerator) GenerateRealisticTestData(ctx context.Context, count int) (*GeneratedTestData, error) {
	start := time.Now()
	
	data := &GeneratedTestData{
		GeneratedAt: time.Now(),
		Metadata: GenerationMetadata{
			Seed:     time.Now().UnixNano(),
			AIModel:  "gpt-4",
		},
	}
	
	// Set random seed for reproducible results
	rand.Seed(data.Metadata.Seed)
	
	// Generate authors first
	authors, err := g.generateAuthors(ctx, g.config.Relationships.AuthorsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authors: %w", err)
	}
	data.Authors = authors
	
	// Generate categories
	categories, err := g.generateCategories(ctx, g.config.Relationships.CategoriesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate categories: %w", err)
	}
	data.Categories = categories
	
	// Generate tags
	tags, err := g.generateTags(ctx, count/5) // Roughly 1 tag per 5 articles
	if err != nil {
		return nil, fmt.Errorf("failed to generate tags: %w", err)
	}
	data.Tags = tags
	
	// Generate articles with realistic content
	articles, err := g.generateArticles(ctx, count, authors, categories, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to generate articles: %w", err)
	}
	data.Articles = articles
	
	// Generate users
	users, err := g.generateUsers(ctx, count/10) // 1 user per 10 articles
	if err != nil {
		return nil, fmt.Errorf("failed to generate users: %w", err)
	}
	data.Users = users
	
	// Generate comments
	comments, err := g.generateComments(ctx, articles, users)
	if err != nil {
		return nil, fmt.Errorf("failed to generate comments: %w", err)
	}
	data.Comments = comments
	
	// Update metadata
	data.Metadata.TotalArticles = len(articles)
	data.Metadata.GenerationTime = time.Since(start)
	data.Metadata.LanguageDistribution = g.calculateLanguageDistribution(articles)
	
	return data, nil
}

// generateArticles generates realistic articles using AI
func (g *AITestDataGenerator) generateArticles(ctx context.Context, count int, authors []TestAuthor, categories []TestCategory, tags []TestTag) ([]TestArticle, error) {
	var articles []TestArticle
	
	// Generate articles in batches to avoid overwhelming the LLM
	batchSize := 10
	for i := 0; i < count; i += batchSize {
		remaining := count - i
		if remaining > batchSize {
			remaining = batchSize
		}
		
		batch, err := g.generateArticleBatch(ctx, remaining, authors, categories, tags, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate article batch %d: %w", i/batchSize, err)
		}
		
		articles = append(articles, batch...)
	}
	
	// Create translation relationships
	articles = g.createTranslationRelationships(articles)
	
	return articles, nil
}

// generateArticleBatch generates a batch of articles
func (g *AITestDataGenerator) generateArticleBatch(ctx context.Context, count int, authors []TestAuthor, categories []TestCategory, tags []TestTag, startID int) ([]TestArticle, error) {
	prompt := g.buildArticleGenerationPrompt(count, authors, categories, tags)
	
	response, err := g.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate articles: %w", err)
	}
	
	articles, err := g.parseArticleResponse(response, startID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse article response: %w", err)
	}
	
	// Enhance articles with realistic metadata
	for i := range articles {
		articles[i] = g.enhanceArticleWithMetadata(articles[i], authors, categories, tags)
	}
	
	return articles, nil
}

// buildArticleGenerationPrompt creates prompt for article generation
func (g *AITestDataGenerator) buildArticleGenerationPrompt(count int, authors []TestAuthor, categories []TestCategory, tags []TestTag) string {
	return fmt.Sprintf(`
Generate %d realistic news articles for a multilingual news website. Create diverse, engaging content across different languages and topics.

Available Languages: English (en), Persian (fa), Arabic (ar)
Available Categories: %v
Available Authors: %v
Available Tags: %v

For each article, generate:
1. Compelling, realistic title (appropriate for the language)
2. Full article content (300-800 words)
3. Brief summary (50-100 words)
4. SEO-optimized meta title and description
5. Relevant keywords (3-5 per article)
6. Appropriate language and direction (RTL for Persian/Arabic)

Content should cover diverse topics:
- Breaking news and current events
- Technology and innovation
- Sports and entertainment
- Politics and economics
- Culture and society
- Health and science

Ensure content is:
- Factually plausible (but fictional)
- Culturally appropriate for each language
- SEO-optimized with natural keyword usage
- Engaging and well-structured
- Properly formatted for web publication

Format as JSON array with fields: title, content, summary, language, meta_title, meta_description, keywords, category_hint, author_hint, tag_hints.
`, count, g.extractCategoryNames(categories), g.extractAuthorNames(authors), g.extractTagNames(tags))
}

// parseArticleResponse parses AI response into article objects
func (g *AITestDataGenerator) parseArticleResponse(response string, startID int) ([]TestArticle, error) {
	// This would parse the JSON response from the LLM
	// For now, return a simplified implementation
	var articles []TestArticle
	
	// In a real implementation, this would parse the JSON response
	// For demonstration, create a few sample articles
	sampleTitles := []string{
		"Breaking: New Technology Revolutionizes News Industry",
		"خبر فوری: تکنولوژی جدید صنعت خبر را متحول می‌کند",
		"عاجل: تقنية جديدة تحدث ثورة في صناعة الأخبار",
	}
	
	for i, title := range sampleTitles {
		if i >= len(sampleTitles) {
			break
		}
		
		article := TestArticle{
			ID:      int64(startID + i + 1),
			Title:   title,
			Content: g.generateSampleContent(title),
			Summary: g.generateSampleSummary(title),
			Status:  "published",
			CreatedAt: time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
			UpdatedAt: time.Now().Add(-time.Duration(rand.Intn(7)) * 24 * time.Hour),
		}
		
		// Set language based on title (supported: en, de, fr, es, ar)
		if strings.Contains(title, "عاجل") || strings.Contains(title, "تقنية") {
			article.Language = "ar"
		} else {
			article.Language = "en"
		}
		
		articles = append(articles, article)
	}
	
	return articles, nil
}

// enhanceArticleWithMetadata adds realistic metadata to articles
func (g *AITestDataGenerator) enhanceArticleWithMetadata(article TestArticle, authors []TestAuthor, categories []TestCategory, tags []TestTag) TestArticle {
	// Assign random author
	if len(authors) > 0 {
		article.AuthorID = authors[rand.Intn(len(authors))].ID
	}
	
	// Assign random category
	if len(categories) > 0 {
		article.CategoryID = categories[rand.Intn(len(categories))].ID
	}
	
	// Assign random tags
	tagCount := rand.Intn(g.config.Relationships.TagsPerArticle) + 1
	for i := 0; i < tagCount && i < len(tags); i++ {
		article.Tags = append(article.Tags, tags[rand.Intn(len(tags))].Name)
	}
	
	// Generate slug
	article.Slug = g.generateSlug(article.Title, article.Language)
	
	// Generate SEO metadata
	article.SEOMetadata = g.generateSEOMetadata(article)
	
	// Generate realistic engagement metrics
	article.ViewCount = rand.Intn(10000) + 100
	article.ShareCount = rand.Intn(article.ViewCount/10) + 1
	article.CommentCount = rand.Intn(article.ViewCount/50) + 1
	
	// Set published date
	publishedAt := article.CreatedAt.Add(time.Duration(rand.Intn(24)) * time.Hour)
	article.PublishedAt = &publishedAt
	
	return article
}

// generateSampleContent generates sample content for demonstration
func (g *AITestDataGenerator) generateSampleContent(title string) string {
	// In a real implementation, this would use the LLM to generate full content
	return fmt.Sprintf("This is sample content for the article titled '%s'. In a real implementation, this would be generated by the AI with realistic, engaging content appropriate for the language and topic.", title)
}

// generateSampleSummary generates sample summary for demonstration
func (g *AITestDataGenerator) generateSampleSummary(title string) string {
	return fmt.Sprintf("Summary of %s", title)
}

// generateSlug generates URL-friendly slug
func (g *AITestDataGenerator) generateSlug(title, language string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, ":", "")
	slug = strings.ReplaceAll(slug, "،", "")
	return slug
}

// generateSEOMetadata generates SEO metadata for articles
func (g *AITestDataGenerator) generateSEOMetadata(article TestArticle) SEOMetadata {
	// Use article language, default to English if not set
	lang := article.Language
	if lang == "" {
		lang = "en"
	}
	return SEOMetadata{
		MetaTitle:       article.Title,
		MetaDescription: article.Summary,
		Keywords:        []string{"news", "article", "breaking"},
		CanonicalURL:    fmt.Sprintf("/%s/article/%s", lang, article.Slug),
		SchemaMarkup: map[string]interface{}{
			"@type": "NewsArticle",
			"headline": article.Title,
			"description": article.Summary,
		},
		OpenGraphTags: map[string]string{
			"og:title":       article.Title,
			"og:description": article.Summary,
			"og:type":        "article",
		},
	}
}

// Helper functions for extracting names
func (g *AITestDataGenerator) extractCategoryNames(categories []TestCategory) []string {
	names := make([]string, len(categories))
	for i, cat := range categories {
		names[i] = cat.Name
	}
	return names
}

func (g *AITestDataGenerator) extractAuthorNames(authors []TestAuthor) []string {
	names := make([]string, len(authors))
	for i, author := range authors {
		names[i] = author.Name
	}
	return names
}

func (g *AITestDataGenerator) extractTagNames(tags []TestTag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

// generateAuthors generates test authors
func (g *AITestDataGenerator) generateAuthors(ctx context.Context, count int) ([]TestAuthor, error) {
	// Simplified implementation - in real version would use LLM
	var authors []TestAuthor
	
	sampleNames := []string{
		"John Smith", "احمد محمدی", "محمد العلي",
		"Sarah Johnson", "فاطمه احمدی", "فاطمة الزهراء",
		"Michael Brown", "علی رضایی", "علي الأحمد",
	}
	
	for i := 0; i < count && i < len(sampleNames); i++ {
		author := TestAuthor{
			ID:        int64(i + 1),
			Name:      sampleNames[i],
			Email:     fmt.Sprintf("author%d@example.com", i+1),
			Bio:       fmt.Sprintf("Experienced journalist and writer - %s", sampleNames[i]),
			CreatedAt: time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour),
		}
		
		// Set language based on name (supported: en, de, fr, es, ar)
		if strings.Contains(author.Name, "محمد") || strings.Contains(author.Name, "فاطمة") || strings.Contains(author.Name, "علي") {
			author.Language = "ar"
		} else {
			author.Language = "en"
		}
		
		authors = append(authors, author)
	}
	
	return authors, nil
}

// generateCategories generates test categories
func (g *AITestDataGenerator) generateCategories(ctx context.Context, count int) ([]TestCategory, error) {
	categories := []TestCategory{
		{ID: 1, Name: "Technology", Slug: "technology", Language: "en"},
		{ID: 2, Name: "Politics", Slug: "politics", Language: "en"},
		{ID: 3, Name: "Sports", Slug: "sports", Language: "en"},
		{ID: 4, Name: "Technologie", Slug: "technologie", Language: "de"},
		{ID: 5, Name: "Technologie", Slug: "technologie-fr", Language: "fr"},
		{ID: 6, Name: "تكنولوجيا", Slug: "technology-ar", Language: "ar"},
	}
	
	for i := range categories {
		categories[i].CreatedAt = time.Now().Add(-time.Duration(rand.Intn(100)) * 24 * time.Hour)
		categories[i].Description = fmt.Sprintf("Category for %s news and articles", categories[i].Name)
	}
	
	if count < len(categories) {
		return categories[:count], nil
	}
	
	return categories, nil
}

// generateTags generates test tags
func (g *AITestDataGenerator) generateTags(ctx context.Context, count int) ([]TestTag, error) {
	sampleTags := []string{
		"breaking", "urgent", "analysis", "opinion", "exclusive",
		"eilmeldung", "analyse", "meinung", "exklusiv", "bericht",
		"عاجل", "تحليل", "رأي", "خاص", "تقرير",
	}
	
	var tags []TestTag
	for i := 0; i < count && i < len(sampleTags); i++ {
		tag := TestTag{
			ID:         int64(i + 1),
			Name:       sampleTags[i],
			Slug:       strings.ToLower(strings.ReplaceAll(sampleTags[i], " ", "-")),
			UsageCount: rand.Intn(1000) + 10,
			CreatedAt:  time.Now().Add(-time.Duration(rand.Intn(200)) * 24 * time.Hour),
		}
		
		// Set language (supported: en, de, fr, es, ar)
		if strings.Contains(tag.Name, "eilmeldung") || strings.Contains(tag.Name, "analyse") || strings.Contains(tag.Name, "meinung") || strings.Contains(tag.Name, "exklusiv") || strings.Contains(tag.Name, "bericht") {
			tag.Language = "de"
		} else if strings.Contains(tag.Name, "عاجل") || strings.Contains(tag.Name, "تحليل") {
			tag.Language = "ar"
		} else {
			tag.Language = "en"
		}
		
		tags = append(tags, tag)
	}
	
	return tags, nil
}

// generateUsers generates test users
func (g *AITestDataGenerator) generateUsers(ctx context.Context, count int) ([]TestUser, error) {
	var users []TestUser
	
	roles := []string{"admin", "editor", "author", "subscriber"}
	
	for i := 0; i < count; i++ {
		user := TestUser{
			ID:        int64(i + 1),
			Username:  fmt.Sprintf("user%d", i+1),
			Email:     fmt.Sprintf("user%d@example.com", i+1),
			Role:      roles[rand.Intn(len(roles))],
			Language:  g.languages[rand.Intn(len(g.languages))].Code,
			CreatedAt: time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour),
			IsActive:  rand.Float32() > 0.1, // 90% active users
		}
		users = append(users, user)
	}
	
	return users, nil
}

// generateComments generates test comments
func (g *AITestDataGenerator) generateComments(ctx context.Context, articles []TestArticle, users []TestUser) ([]TestComment, error) {
	var comments []TestComment
	commentID := int64(1)
	
	for _, article := range articles {
		// Generate 0-5 comments per article
		commentCount := rand.Intn(6)
		
		for i := 0; i < commentCount; i++ {
			if len(users) == 0 {
				break
			}
			
			comment := TestComment{
				ID:        commentID,
				ArticleID: article.ID,
				UserID:    users[rand.Intn(len(users))].ID,
				Content:   g.generateCommentContent(article.Language),
				Language:  article.Language,
				Status:    "approved",
				CreatedAt: article.CreatedAt.Add(time.Duration(rand.Intn(72)) * time.Hour),
			}
			
			comments = append(comments, comment)
			commentID++
		}
	}
	
	return comments, nil
}

// generateCommentContent generates sample comment content
func (g *AITestDataGenerator) generateCommentContent(language string) string {
	switch language {
	case "de":
		return "Toller Artikel! Sehr informativ und gut geschrieben."
	case "fr":
		return "Excellent article ! Très informatif et bien écrit."
	case "es":
		return "¡Excelente artículo! Muy informativo y bien escrito."
	case "ar":
		return "مقال رائع ومفيد جداً. شكراً للكاتب."
	default:
		return "Great article! Very informative and well-written."
	}
}

// createTranslationRelationships creates translation relationships between articles
func (g *AITestDataGenerator) createTranslationRelationships(articles []TestArticle) []TestArticle {
	// Group articles by similar content/topic for translation relationships
	for i := 0; i < len(articles)-2; i += 3 {
		// Make every 3rd article a translation of the first
		if i+1 < len(articles) {
			articles[i+1].TranslationOf = &articles[i].ID
		}
		if i+2 < len(articles) {
			articles[i+2].TranslationOf = &articles[i].ID
		}
	}
	
	return articles
}

// calculateLanguageDistribution calculates language distribution in generated data
func (g *AITestDataGenerator) calculateLanguageDistribution(articles []TestArticle) map[string]int {
	distribution := make(map[string]int)
	
	for _, article := range articles {
		distribution[article.Language]++
	}
	
	return distribution
}