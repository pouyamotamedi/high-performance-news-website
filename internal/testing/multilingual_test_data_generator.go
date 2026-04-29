package testing

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// MultilingualLanguage represents a language configuration for test data generation
type MultilingualLanguage struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	NativeName   string   `json:"native_name"`
	Direction    string   `json:"direction"` // "ltr" or "rtl"
	CharacterSet string   `json:"character_set"`
	Fonts        []string `json:"fonts"`
	WordList     []string `json:"word_list"`
	Sentences    []string `json:"sentence_templates"`
}

// MultilingualTestDataGenerator generates realistic multilingual test data
type MultilingualTestDataGenerator struct {
	db                *sql.DB
	languages         map[string]*MultilingualLanguage
	dataVersions      *DataVersionManager
	anonymizer        *DataAnonymizer
	relationshipMgr   *RelationshipManager
	metadataGenerator *MetadataGenerator
}

// MultilingualTestArticle represents a test article with all necessary fields
type MultilingualTestArticle struct {
	ID                 uint64                 `json:"id"`
	Title              string                 `json:"title"`
	Slug               string                 `json:"slug"`
	Content            string                 `json:"content"`
	Excerpt            string                 `json:"excerpt"`
	AuthorID           uint64                 `json:"author_id"`
	CategoryID         uint64                 `json:"category_id"`
	Tags               []models.Tag           `json:"tags"`
	Status             string                 `json:"status"`
	PublishedAt        *time.Time             `json:"published_at"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	ViewCount          uint64                 `json:"view_count"`
	LikeCount          uint64                 `json:"like_count"`
	DislikeCount       uint64                 `json:"dislike_count"`
	LanguageCode       string                 `json:"language_code"`
	TranslationGroupID *uint64                `json:"translation_group_id"`
	SEOMetadata        models.SEOData         `json:"seo_metadata"`
	Relationships      map[string]interface{} `json:"relationships"`
}

// TestUser represents a test user with realistic data
type TestUser struct {
	ID           uint64           `json:"id"`
	Username     string           `json:"username"`
	Email        string           `json:"email"`
	PasswordHash string           `json:"password_hash"`
	Role         models.UserRole  `json:"role"`
	FirstName    string           `json:"first_name"`
	LastName     string           `json:"last_name"`
	Bio          string           `json:"bio"`
	Avatar       string           `json:"avatar"`
	IsActive     bool             `json:"is_active"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(db *sql.DB) *TestDataGenerator {
	generator := &TestDataGenerator{
		db:                db,
		languages:         make(map[string]*Language),
		dataVersions:      NewDataVersionManager(),
		anonymizer:        NewDataAnonymizer(),
		relationshipMgr:   NewRelationshipManager(),
		metadataGenerator: NewMetadataGenerator(),
	}
	
	generator.initializeLanguages()
	return generator
}

// initializeLanguages sets up language configurations with realistic word lists
func (g *TestDataGenerator) initializeLanguages() {
	// English language configuration
	g.languages["en"] = &Language{
		Code:         "en",
		Name:         "English",
		NativeName:   "English",
		Direction:    "ltr",
		CharacterSet: "latin",
		Fonts:        []string{"Arial", "Helvetica", "Times New Roman"},
		WordList: []string{
			"news", "article", "report", "story", "breaking", "update", "analysis",
			"politics", "economy", "technology", "sports", "culture", "science",
			"health", "education", "environment", "business", "international",
			"local", "government", "society", "development", "research", "study",
			"investigation", "interview", "conference", "meeting", "announcement",
			"decision", "policy", "reform", "initiative", "program", "project",
			"campaign", "election", "vote", "parliament", "minister", "president",
			"official", "spokesperson", "expert", "analyst", "journalist", "reporter",
		},
		Sentences: []string{
			"The %s announced a new %s today.",
			"According to %s, the %s will %s.",
			"Experts believe that %s could %s.",
			"The latest %s shows that %s.",
			"In a recent %s, officials discussed %s.",
			"The %s department reported %s.",
			"New research indicates that %s.",
			"The government plans to %s.",
		},
	}

	// Persian language configuration
	g.languages["fa"] = &Language{
		Code:         "fa",
		Name:         "Persian",
		NativeName:   "فارسی",
		Direction:    "rtl",
		CharacterSet: "persian",
		Fonts:        []string{"Tahoma", "Arial Unicode MS", "B Nazanin"},
		WordList: []string{
			"خبر", "گزارش", "مقاله", "تحلیل", "بررسی", "مطالعه", "تحقیق",
			"سیاست", "اقتصاد", "فناوری", "ورزش", "فرهنگ", "علم", "بهداشت",
			"آموزش", "محیط‌زیست", "تجارت", "بین‌المللی", "محلی", "دولت",
			"جامعه", "توسعه", "پژوهش", "مصاحبه", "کنفرانس", "جلسه", "اعلام",
			"تصمیم", "سیاست", "اصلاحات", "ابتکار", "برنامه", "پروژه",
			"کمپین", "انتخابات", "رای", "مجلس", "وزیر", "رئیس‌جمهور",
			"مقام", "سخنگو", "کارشناس", "تحلیلگر", "روزنامه‌نگار", "خبرنگار",
			"شهر", "استان", "کشور", "منطقه", "جهان", "ملت", "مردم", "شهروند",
		},
		Sentences: []string{
			"%s امروز %s جدیدی را اعلام کرد.",
			"بر اساس گزارش %s، %s قرار است %s.",
			"کارشناسان معتقدند که %s می‌تواند %s.",
			"آخرین %s نشان می‌دهد که %s.",
			"در %s اخیر، مقامات درباره %s بحث کردند.",
			"وزارت %s گزارش داد که %s.",
			"تحقیقات جدید نشان می‌دهد که %s.",
			"دولت قصد دارد %s.",
		},
	}

	// Arabic language configuration
	g.languages["ar"] = &Language{
		Code:         "ar",
		Name:         "Arabic",
		NativeName:   "العربية",
		Direction:    "rtl",
		CharacterSet: "arabic",
		Fonts:        []string{"Tahoma", "Arial Unicode MS", "Traditional Arabic"},
		WordList: []string{
			"خبر", "تقرير", "مقال", "تحليل", "دراسة", "بحث", "تحقيق",
			"سياسة", "اقتصاد", "تكنولوجيا", "رياضة", "ثقافة", "علم", "صحة",
			"تعليم", "بيئة", "تجارة", "دولي", "محلي", "حكومة", "مجتمع",
			"تنمية", "بحث", "مقابلة", "مؤتمر", "اجتماع", "إعلان", "قرار",
			"سياسة", "إصلاحات", "مبادرة", "برنامج", "مشروع", "حملة",
			"انتخابات", "تصويت", "برلمان", "وزير", "رئيس", "مسؤول",
			"متحدث", "خبير", "محلل", "صحفي", "مراسل", "مدينة", "محافظة",
			"دولة", "منطقة", "عالم", "أمة", "شعب", "مواطن",
		},
		Sentences: []string{
			"أعلن %s اليوم عن %s جديد.",
			"وفقاً لـ %s، سوف %s %s.",
			"يعتقد الخبراء أن %s يمكن أن %s.",
			"يظهر آخر %s أن %s.",
			"في %s الأخير، ناقش المسؤولون %s.",
			"أفادت وزارة %s بأن %s.",
			"تشير الأبحاث الجديدة إلى أن %s.",
			"تخطط الحكومة لـ %s.",
		},
	}
}

// GenerateMultilingualTestData generates realistic multilingual test data
func (g *TestDataGenerator) GenerateMultilingualTestData(count int) ([]TestArticle, error) {
	return g.GenerateMultilingualTestDataParallel(count, 4) // Use 4 workers by default
}

// GenerateMultilingualTestDataParallel generates test data using parallel workers
func (g *TestDataGenerator) GenerateMultilingualTestDataParallel(count int, workers int) ([]TestArticle, error) {
	if workers <= 0 {
		workers = 1
	}
	
	// Calculate articles per language
	languageCount := len(g.languages)
	articlesPerLang := count / languageCount
	
	// Create channels for work distribution
	jobs := make(chan GenerationJob, count)
	results := make(chan TestArticle, count)
	
	// Start workers
	for w := 0; w < workers; w++ {
		go g.generationWorker(jobs, results)
	}
	
	// Generate translation groups first
	translationGroups := g.generateTranslationGroups(articlesPerLang / 3) // Assume 1/3 have translations
	
	// Send jobs to workers
	articleID := uint64(1)
	jobCount := 0
	
	for i := 0; i < articlesPerLang; i++ {
		for langCode, lang := range g.languages {
			var translationGroupID *uint64
			if i < len(translationGroups) {
				groupID := translationGroups[i]
				translationGroupID = &groupID
			}
			
			job := GenerationJob{
				ID:                 articleID,
				Language:           lang,
				LanguageCode:       langCode,
				TranslationGroupID: translationGroupID,
				Index:              i,
			}
			
			jobs <- job
			articleID++
			jobCount++
		}
	}
	close(jobs)
	
	// Collect results
	articles := make([]TestArticle, 0, jobCount)
	for i := 0; i < jobCount; i++ {
		article := <-results
		articles = append(articles, article)
	}
	
	// Generate relationships after all articles are created
	g.generateCrossArticleRelationships(articles)
	
	return articles, nil
}

// GenerationJob represents a job for generating a single article
type GenerationJob struct {
	ID                 uint64
	Language           *Language
	LanguageCode       string
	TranslationGroupID *uint64
	Index              int
}

// generationWorker processes generation jobs
func (g *TestDataGenerator) generationWorker(jobs <-chan GenerationJob, results chan<- TestArticle) {
	for job := range jobs {
		article := TestArticle{
			ID:                 job.ID,
			LanguageCode:       job.LanguageCode,
			Title:              g.generateRealisticTitle(job.Language),
			Content:            g.generateRealisticContent(job.Language, 500+g.randomInt(2000)),
			Excerpt:            g.generateExcerpt(job.Language),
			Slug:               g.generateSlug(job.Language),
			AuthorID:           g.selectRandomAuthor(),
			CategoryID:         g.selectRandomCategory(),
			Tags:               g.generateRandomTags(job.Language, 3+g.randomInt(5)),
			Status:             g.selectRandomStatus(),
			PublishedAt:        g.generatePublishedDate(),
			CreatedAt:          time.Now().Add(-time.Duration(g.randomInt(35)) * 24 * time.Hour),
			UpdatedAt:          time.Now().Add(-time.Duration(g.randomInt(7)) * 24 * time.Hour),
			ViewCount:          uint64(g.randomInt(10000)),
			LikeCount:          uint64(g.randomInt(500)),
			DislikeCount:       uint64(g.randomInt(50)),
			TranslationGroupID: job.TranslationGroupID,
			SEOMetadata:        g.generateSEOMetadata(job.Language),
			Relationships:      make(map[string]interface{}),
		}
		
		// Add basic relationships
		article.Relationships = g.relationshipMgr.GenerateRelationships(article.ID, job.LanguageCode)
		
		results <- article
	}
}

// generateCrossArticleRelationships generates relationships between articles
func (g *TestDataGenerator) generateCrossArticleRelationships(articles []TestArticle) {
	// Create lookup maps for efficient relationship generation
	articlesByLang := make(map[string][]TestArticle)
	articlesById := make(map[uint64]*TestArticle)
	
	for i := range articles {
		lang := articles[i].LanguageCode
		articlesByLang[lang] = append(articlesByLang[lang], articles[i])
		articlesById[articles[i].ID] = &articles[i]
	}
	
	// Generate related article relationships
	for i := range articles {
		if g.randomInt(100) < 20 { // 20% chance of having related articles
			relatedCount := 1 + g.randomInt(3) // 1-3 related articles
			sameLangArticles := articlesByLang[articles[i].LanguageCode]
			
			for j := 0; j < relatedCount && j < len(sameLangArticles)-1; j++ {
				relatedIndex := g.randomInt(len(sameLangArticles))
				if sameLangArticles[relatedIndex].ID != articles[i].ID {
					if articles[i].Relationships["related"] == nil {
						articles[i].Relationships["related"] = make([]uint64, 0)
					}
					relatedList := articles[i].Relationships["related"].([]uint64)
					relatedList = append(relatedList, sameLangArticles[relatedIndex].ID)
					articles[i].Relationships["related"] = relatedList
				}
			}
		}
	}
}

// generateRealisticTitle creates a realistic title in the specified language
func (g *TestDataGenerator) generateRealisticTitle(lang *Language) string {
	words := lang.WordList
	titleLength := 3 + g.randomInt(7) // 3-9 words
	
	var titleWords []string
	for i := 0; i < titleLength; i++ {
		word := words[g.randomInt(len(words))]
		titleWords = append(titleWords, word)
	}
	
	title := strings.Join(titleWords, " ")
	
	// Capitalize first letter for LTR languages
	if lang.Direction == "ltr" && len(title) > 0 {
		title = strings.ToUpper(string(title[0])) + title[1:]
	}
	
	return title
}

// generateRealisticContent creates realistic content in the specified language
func (g *TestDataGenerator) generateRealisticContent(lang *Language, targetLength int) string {
	sentences := lang.Sentences
	words := lang.WordList
	
	var content strings.Builder
	currentLength := 0
	
	for currentLength < targetLength {
		// Select a random sentence template
		template := sentences[g.randomInt(len(sentences))]
		
		// Fill template with random words
		sentence := g.fillSentenceTemplate(template, words)
		
		if content.Len() > 0 {
			content.WriteString(" ")
		}
		content.WriteString(sentence)
		currentLength = content.Len()
		
		// Add paragraph breaks occasionally
		if g.randomInt(100) < 15 { // 15% chance
			content.WriteString("\n\n")
		}
	}
	
	return content.String()
}

// fillSentenceTemplate fills a sentence template with random words
func (g *TestDataGenerator) fillSentenceTemplate(template string, words []string) string {
	// Count placeholders
	placeholderCount := strings.Count(template, "%s")
	
	// Generate replacement words
	replacements := make([]interface{}, placeholderCount)
	for i := 0; i < placeholderCount; i++ {
		replacements[i] = words[g.randomInt(len(words))]
	}
	
	return fmt.Sprintf(template, replacements...)
}

// generateExcerpt creates an excerpt from content or generates one
func (g *TestDataGenerator) generateExcerpt(lang *Language) string {
	words := lang.WordList
	excerptLength := 10 + g.randomInt(20) // 10-29 words
	
	var excerptWords []string
	for i := 0; i < excerptLength; i++ {
		word := words[g.randomInt(len(words))]
		excerptWords = append(excerptWords, word)
	}
	
	return strings.Join(excerptWords, " ")
}

// generateSlug creates a URL-friendly slug
func (g *TestDataGenerator) generateSlug(lang *Language) string {
	words := lang.WordList
	slugLength := 2 + g.randomInt(4) // 2-5 words
	
	var slugWords []string
	for i := 0; i < slugLength; i++ {
		word := words[g.randomInt(len(words))]
		// For RTL languages, use transliterated versions
		if lang.Direction == "rtl" {
			word = g.transliterateToLatin(word)
		}
		slugWords = append(slugWords, strings.ToLower(word))
	}
	
	return strings.Join(slugWords, "-")
}

// transliterateToLatin converts RTL text to Latin characters for URLs
func (g *TestDataGenerator) transliterateToLatin(text string) string {
	// Simple transliteration mapping (in real implementation, use proper library)
	transliterations := map[string]string{
		// Persian
		"خبر":     "khabar",
		"گزارش":   "gozaresh",
		"مقاله":   "maghale",
		"تحلیل":   "tahlil",
		"سیاست":  "siasat",
		"اقتصاد":  "eghtesad",
		"فناوری":  "fanavari",
		"ورزش":   "varzesh",
		"فرهنگ":  "farhang",
		// Arabic
		"خبر":     "khabar",
		"تقرير":   "taqrir",
		"مقال":    "maqal",
		"تحليل":   "tahlil",
		"سياسة":   "siyasa",
		"اقتصاد":  "iqtisad",
	}
	
	if latin, exists := transliterations[text]; exists {
		return latin
	}
	
	// Fallback: generate random Latin word
	latinWords := []string{"news", "report", "article", "story", "update", "analysis"}
	return latinWords[g.randomInt(len(latinWords))]
}

// selectRandomAuthor returns a random author ID
func (g *TestDataGenerator) selectRandomAuthor() uint64 {
	return uint64(1 + g.randomInt(50)) // Assume 50 authors
}

// selectRandomCategory returns a random category ID
func (g *TestDataGenerator) selectRandomCategory() uint64 {
	return uint64(1 + g.randomInt(20)) // Assume 20 categories
}

// generateRandomTags creates random tags for an article
func (g *TestDataGenerator) generateRandomTags(lang *Language, count int) []models.Tag {
	var tags []models.Tag
	words := lang.WordList
	
	for i := 0; i < count; i++ {
		tag := models.Tag{
			ID:   uint64(1 + g.randomInt(100)), // Assume 100 possible tags
			Name: words[g.randomInt(len(words))],
			Slug: g.generateSlug(lang),
		}
		tags = append(tags, tag)
	}
	
	return tags
}

// selectRandomStatus returns a random article status
func (g *TestDataGenerator) selectRandomStatus() string {
	statuses := []string{"published", "draft", "archived"}
	weights := []int{70, 25, 5} // 70% published, 25% draft, 5% archived
	
	random := g.randomInt(100)
	cumulative := 0
	
	for i, weight := range weights {
		cumulative += weight
		if random < cumulative {
			return statuses[i]
		}
	}
	
	return "published" // fallback
}

// generatePublishedDate creates a realistic published date
func (g *TestDataGenerator) generatePublishedDate() *time.Time {
	// 80% chance of being published
	if g.randomInt(100) < 20 {
		return nil // Not published
	}
	
	// Published within last 30 days
	daysAgo := g.randomInt(30)
	publishedAt := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
	return &publishedAt
}

// generateSEOMetadata creates realistic SEO metadata
func (g *TestDataGenerator) generateSEOMetadata(lang *Language) models.SEOData {
	return g.metadataGenerator.GenerateSEOMetadata(lang)
}

// generateTranslationGroups creates translation group IDs
func (g *TestDataGenerator) generateTranslationGroups(count int) []uint64 {
	var groups []uint64
	for i := 0; i < count; i++ {
		groups = append(groups, uint64(i+1))
	}
	return groups
}

// randomInt generates a random integer between 0 and max-1
func (g *TestDataGenerator) randomInt(max int) int {
	if max <= 0 {
		return 0
	}
	
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Fallback to time-based seed
		return int(time.Now().UnixNano()) % max
	}
	
	return int(n.Int64())
}

// GenerateTestUsers creates realistic test users
func (g *TestDataGenerator) GenerateTestUsers(count int) ([]TestUser, error) {
	var users []TestUser
	
	roles := []models.UserRole{
		models.RoleAdmin,
		models.RoleEditor,
		models.RoleReporter,
		models.RoleContributor,
	}
	
	roleWeights := []int{5, 15, 30, 50} // Distribution percentages
	
	for i := 0; i < count; i++ {
		role := g.selectWeightedRole(roles, roleWeights)
		
		user := TestUser{
			ID:           uint64(i + 1),
			Username:     g.generateUsername(),
			Email:        g.generateEmail(),
			PasswordHash: g.generatePasswordHash(),
			Role:         role,
			FirstName:    g.generateFirstName(),
			LastName:     g.generateLastName(),
			Bio:          g.generateBio(),
			Avatar:       g.generateAvatarURL(),
			IsActive:     g.randomInt(100) < 90, // 90% active
			CreatedAt:    time.Now().Add(-time.Duration(g.randomInt(365)) * 24 * time.Hour),
			UpdatedAt:    time.Now().Add(-time.Duration(g.randomInt(30)) * 24 * time.Hour),
		}
		
		users = append(users, user)
	}
	
	return users, nil
}

// selectWeightedRole selects a role based on weights
func (g *TestDataGenerator) selectWeightedRole(roles []models.UserRole, weights []int) models.UserRole {
	random := g.randomInt(100)
	cumulative := 0
	
	for i, weight := range weights {
		cumulative += weight
		if random < cumulative {
			return roles[i]
		}
	}
	
	return models.RoleContributor // fallback
}

// generateUsername creates a realistic username
func (g *TestDataGenerator) generateUsername() string {
	prefixes := []string{"user", "reporter", "editor", "writer", "journalist"}
	prefix := prefixes[g.randomInt(len(prefixes))]
	suffix := g.randomInt(9999)
	return fmt.Sprintf("%s%d", prefix, suffix)
}

// generateEmail creates a realistic email address
func (g *TestDataGenerator) generateEmail() string {
	domains := []string{"example.com", "test.com", "demo.org", "sample.net"}
	username := g.generateUsername()
	domain := domains[g.randomInt(len(domains))]
	return fmt.Sprintf("%s@%s", username, domain)
}

// generatePasswordHash creates a test password hash
func (g *TestDataGenerator) generatePasswordHash() string {
	// In real implementation, use proper password hashing
	return "$2a$10$example.hash.for.testing.purposes.only"
}

// generateFirstName creates a realistic first name
func (g *TestDataGenerator) generateFirstName() string {
	names := []string{
		"Ahmad", "Ali", "Hassan", "Mohammad", "Reza", "Mehdi", "Hossein",
		"Fateme", "Zahra", "Maryam", "Atefeh", "Nasrin", "Sara", "Leila",
		"Ahmed", "Omar", "Khalid", "Youssef", "Amira", "Nour", "Layla",
		"John", "David", "Michael", "Sarah", "Emma", "Lisa", "Jennifer",
	}
	return names[g.randomInt(len(names))]
}

// generateLastName creates a realistic last name
func (g *TestDataGenerator) generateLastName() string {
	names := []string{
		"Ahmadi", "Hosseini", "Mohammadi", "Rezaei", "Karimi", "Moradi",
		"Al-Ahmad", "Al-Hassan", "Al-Mohammad", "Bin-Omar", "Bin-Khalid",
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
	}
	return names[g.randomInt(len(names))]
}

// generateBio creates a realistic user bio
func (g *TestDataGenerator) generateBio() string {
	bios := []string{
		"Experienced journalist with focus on political reporting.",
		"Technology writer covering latest developments in AI and software.",
		"Sports reporter covering local and international events.",
		"Cultural correspondent with expertise in arts and literature.",
		"Economic analyst writing about market trends and business news.",
		"Health and science writer covering medical breakthroughs.",
		"Environmental journalist focusing on climate change issues.",
		"Education reporter covering policy and institutional changes.",
	}
	
	if g.randomInt(100) < 30 { // 30% chance of no bio
		return ""
	}
	
	return bios[g.randomInt(len(bios))]
}

// generateAvatarURL creates a test avatar URL
func (g *TestDataGenerator) generateAvatarURL() string {
	if g.randomInt(100) < 40 { // 40% chance of no avatar
		return ""
	}
	
	avatarID := g.randomInt(100)
	return fmt.Sprintf("https://example.com/avatars/user_%d.jpg", avatarID)
}

// BulkInsertArticles efficiently inserts large numbers of articles into the database
func (g *TestDataGenerator) BulkInsertArticles(articles []TestArticle) error {
	if g.db == nil {
		return fmt.Errorf("database connection not set")
	}
	
	// Use transaction for better performance
	tx, err := g.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Prepare bulk insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO articles (
			id, title, slug, content, excerpt, author_id, category_id, 
			status, published_at, created_at, updated_at, view_count, 
			like_count, dislike_count, language_code, translation_group_id,
			meta_title, meta_description, canonical_url, schema_type
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert articles in batches
	batchSize := 1000
	for i := 0; i < len(articles); i += batchSize {
		end := i + batchSize
		if end > len(articles) {
			end = len(articles)
		}
		
		for j := i; j < end; j++ {
			article := articles[j]
			_, err = stmt.Exec(
				article.ID,
				article.Title,
				article.Slug,
				article.Content,
				article.Excerpt,
				article.AuthorID,
				article.CategoryID,
				article.Status,
				article.PublishedAt,
				article.CreatedAt,
				article.UpdatedAt,
				article.ViewCount,
				article.LikeCount,
				article.DislikeCount,
				article.LanguageCode,
				article.TranslationGroupID,
				article.SEOMetadata.MetaTitle,
				article.SEOMetadata.MetaDescription,
				article.SEOMetadata.CanonicalURL,
				article.SEOMetadata.SchemaType,
			)
			if err != nil {
				return fmt.Errorf("failed to insert article %d: %w", article.ID, err)
			}
		}
		
		// Log progress for large datasets
		if len(articles) > 10000 {
			log.Printf("Inserted batch %d-%d of %d articles", i+1, end, len(articles))
		}
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	log.Printf("Successfully inserted %d articles", len(articles))
	return nil
}

// BulkInsertUsers efficiently inserts large numbers of users into the database
func (g *TestDataGenerator) BulkInsertUsers(users []TestUser) error {
	if g.db == nil {
		return fmt.Errorf("database connection not set")
	}
	
	tx, err := g.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO users (
			id, username, email, password_hash, role, first_name, last_name,
			bio, avatar, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	for _, user := range users {
		_, err = stmt.Exec(
			user.ID,
			user.Username,
			user.Email,
			user.PasswordHash,
			string(user.Role),
			user.FirstName,
			user.LastName,
			user.Bio,
			user.Avatar,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert user %d: %w", user.ID, err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	log.Printf("Successfully inserted %d users", len(users))
	return nil
}

// GenerateLargeScaleTestData generates large-scale test data for performance testing
func (g *TestDataGenerator) GenerateLargeScaleTestData(articleCount int) (*LargeScaleTestDataset, error) {
	log.Printf("Generating large-scale test dataset with %d articles", articleCount)
	
	dataset := &LargeScaleTestDataset{
		GeneratedAt: time.Now(),
		Config: LargeScaleConfig{
			ArticleCount: articleCount,
			UserCount:    articleCount / 50, // 1 user per 50 articles
			CategoryCount: 20,
			TagCount:     100,
		},
	}
	
	// Generate users first
	log.Printf("Generating %d users...", dataset.Config.UserCount)
	users, err := g.GenerateTestUsers(dataset.Config.UserCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate users: %w", err)
	}
	dataset.Users = users
	
	// Generate categories
	log.Printf("Generating %d categories...", dataset.Config.CategoryCount)
	dataset.Categories = g.generateTestCategories(dataset.Config.CategoryCount)
	
	// Generate tags
	log.Printf("Generating %d tags...", dataset.Config.TagCount)
	dataset.Tags = g.generateTestTags(dataset.Config.TagCount)
	
	// Generate articles with progress tracking
	log.Printf("Generating %d articles...", articleCount)
	articles, err := g.GenerateMultilingualTestDataParallel(articleCount, 8) // Use 8 workers for large datasets
	if err != nil {
		return nil, fmt.Errorf("failed to generate articles: %w", err)
	}
	dataset.Articles = articles
	
	// Generate performance metrics
	dataset.Metrics = g.calculateDatasetMetrics(dataset)
	
	log.Printf("Large-scale test dataset generation completed")
	return dataset, nil
}

// LargeScaleTestDataset represents a large-scale test dataset
type LargeScaleTestDataset struct {
	Articles    []TestArticle      `json:"articles"`
	Users       []TestUser         `json:"users"`
	Categories  []TestCategory     `json:"categories"`
	Tags        []TestTag          `json:"tags"`
	GeneratedAt time.Time          `json:"generated_at"`
	Config      LargeScaleConfig   `json:"config"`
	Metrics     DatasetMetrics     `json:"metrics"`
}

// LargeScaleConfig holds configuration for large-scale data generation
type LargeScaleConfig struct {
	ArticleCount  int `json:"article_count"`
	UserCount     int `json:"user_count"`
	CategoryCount int `json:"category_count"`
	TagCount      int `json:"tag_count"`
}

// DatasetMetrics holds metrics about the generated dataset
type DatasetMetrics struct {
	TotalSize           int64                  `json:"total_size_bytes"`
	GenerationDuration  time.Duration          `json:"generation_duration"`
	ArticlesPerSecond   float64                `json:"articles_per_second"`
	LanguageDistribution map[string]int        `json:"language_distribution"`
	StatusDistribution  map[string]int         `json:"status_distribution"`
	QualityScore        float64                `json:"quality_score"`
	RelationshipStats   map[string]interface{} `json:"relationship_stats"`
}

// TestCategory represents a test category
type TestCategory struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ParentID    *uint64   `json:"parent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TestTag represents a test tag
type TestTag struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// generateTestCategories generates test categories
func (g *TestDataGenerator) generateTestCategories(count int) []TestCategory {
	categories := make([]TestCategory, count)
	
	categoryNames := map[string][]string{
		"en": {"Politics", "Economy", "Technology", "Sports", "Culture", "Science", "Health", "Education", "Environment", "Business"},
		"fa": {"سیاست", "اقتصاد", "فناوری", "ورزش", "فرهنگ", "علم", "بهداشت", "آموزش", "محیط‌زیست", "تجارت"},
		"ar": {"السياسة", "الاقتصاد", "التكنولوجيا", "الرياضة", "الثقافة", "العلوم", "الصحة", "التعليم", "البيئة", "الأعمال"},
	}
	
	baseNames := categoryNames["en"]
	
	for i := 0; i < count; i++ {
		nameIndex := i % len(baseNames)
		baseName := baseNames[nameIndex]
		
		category := TestCategory{
			ID:          uint64(i + 1),
			Name:        baseName,
			Slug:        strings.ToLower(strings.ReplaceAll(baseName, " ", "-")),
			Description: fmt.Sprintf("Test category for %s content", baseName),
			CreatedAt:   time.Now().Add(-time.Duration(g.randomInt(365)) * 24 * time.Hour),
		}
		
		// Add parent relationship for some categories
		if i > 10 && g.randomInt(100) < 30 { // 30% chance of having parent
			parentID := uint64(1 + g.randomInt(10)) // Parent from first 10 categories
			category.ParentID = &parentID
		}
		
		categories[i] = category
	}
	
	return categories
}

// generateTestTags generates test tags
func (g *TestDataGenerator) generateTestTags(count int) []TestTag {
	tags := make([]TestTag, count)
	
	tagNames := []string{
		"breaking", "analysis", "interview", "report", "investigation", "update", "exclusive",
		"opinion", "review", "preview", "recap", "trending", "viral", "controversial",
		"educational", "informative", "urgent", "developing", "confirmed", "unconfirmed",
		"local", "international", "national", "regional", "global", "community", "public",
		"private", "government", "corporate", "academic", "research", "study", "survey",
	}
	
	colors := []string{"#FF5733", "#33FF57", "#3357FF", "#FF33F1", "#F1FF33", "#33FFF1"}
	
	for i := 0; i < count; i++ {
		nameIndex := i % len(tagNames)
		baseName := tagNames[nameIndex]
		
		// Add variation to avoid exact duplicates
		name := baseName
		if i >= len(tagNames) {
			name = fmt.Sprintf("%s-%d", baseName, i/len(tagNames)+1)
		}
		
		tag := TestTag{
			ID:        uint64(i + 1),
			Name:      name,
			Slug:      strings.ToLower(strings.ReplaceAll(name, " ", "-")),
			Color:     colors[g.randomInt(len(colors))],
			CreatedAt: time.Now().Add(-time.Duration(g.randomInt(365)) * 24 * time.Hour),
		}
		
		tags[i] = tag
	}
	
	return tags
}

// calculateDatasetMetrics calculates metrics for the generated dataset
func (g *TestDataGenerator) calculateDatasetMetrics(dataset *LargeScaleTestDataset) DatasetMetrics {
	metrics := DatasetMetrics{
		LanguageDistribution: make(map[string]int),
		StatusDistribution:   make(map[string]int),
		RelationshipStats:    make(map[string]interface{}),
	}
	
	// Calculate language distribution
	for _, article := range dataset.Articles {
		metrics.LanguageDistribution[article.LanguageCode]++
		metrics.StatusDistribution[article.Status]++
	}
	
	// Calculate quality score (percentage of valid articles)
	validArticles := 0
	for _, article := range dataset.Articles {
		if g.isValidArticle(article) {
			validArticles++
		}
	}
	metrics.QualityScore = float64(validArticles) / float64(len(dataset.Articles))
	
	// Calculate relationship statistics
	translationGroups := make(map[uint64]int)
	relatedArticles := 0
	
	for _, article := range dataset.Articles {
		if article.TranslationGroupID != nil {
			translationGroups[*article.TranslationGroupID]++
		}
		if related, exists := article.Relationships["related"]; exists && related != nil {
			relatedArticles++
		}
	}
	
	metrics.RelationshipStats["translation_groups"] = len(translationGroups)
	metrics.RelationshipStats["articles_with_translations"] = len(translationGroups)
	metrics.RelationshipStats["articles_with_related"] = relatedArticles
	
	return metrics
}

// isValidArticle checks if an article is valid
func (g *TestDataGenerator) isValidArticle(article TestArticle) bool {
	return article.Title != "" &&
		article.Content != "" &&
		article.Slug != "" &&
		article.AuthorID > 0 &&
		article.CategoryID > 0 &&
		article.LanguageCode != ""
}

// SaveDatasetToFile saves the dataset to a file for later use
func (g *TestDataGenerator) SaveDatasetToFile(dataset *LargeScaleTestDataset, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(dataset); err != nil {
		return fmt.Errorf("failed to encode dataset: %w", err)
	}
	
	log.Printf("Dataset saved to %s", filename)
	return nil
}

// LoadDatasetFromFile loads a dataset from a file
func (g *TestDataGenerator) LoadDatasetFromFile(filename string) (*LargeScaleTestDataset, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	var dataset LargeScaleTestDataset
	decoder := json.NewDecoder(file)
	
	if err := decoder.Decode(&dataset); err != nil {
		return nil, fmt.Errorf("failed to decode dataset: %w", err)
	}
	
	log.Printf("Dataset loaded from %s", filename)
	return &dataset, nil
}

// GenerateRealisticRelationships generates realistic relationships between entities
func (g *TestDataGenerator) GenerateRealisticRelationships(articles []TestArticle, users []TestUser, categories []TestCategory, tags []TestTag) error {
	log.Printf("Generating realistic relationships for %d articles", len(articles))
	
	// Update relationship manager with entity counts
	g.relationshipMgr.SetEntityCount("article", uint64(len(articles)))
	g.relationshipMgr.SetEntityCount("user", uint64(len(users)))
	g.relationshipMgr.SetEntityCount("category", uint64(len(categories)))
	g.relationshipMgr.SetEntityCount("tag", uint64(len(tags)))
	
	// Generate article-tag relationships
	for i := range articles {
		tagCount := 3 + g.randomInt(5) // 3-7 tags per article
		usedTags := make(map[int]bool)
		
		articles[i].Tags = make([]models.Tag, 0, tagCount)
		
		for j := 0; j < tagCount && len(articles[i].Tags) < len(tags); j++ {
			tagIndex := g.randomInt(len(tags))
			if !usedTags[tagIndex] {
				tag := models.Tag{
					ID:   tags[tagIndex].ID,
					Name: tags[tagIndex].Name,
					Slug: tags[tagIndex].Slug,
				}
				articles[i].Tags = append(articles[i].Tags, tag)
				usedTags[tagIndex] = true
			}
		}
	}
	
	log.Printf("Relationship generation completed")
	return nil
}

// OptimizeForPerformanceTesting optimizes the dataset for performance testing
func (g *TestDataGenerator) OptimizeForPerformanceTesting(dataset *LargeScaleTestDataset) error {
	log.Printf("Optimizing dataset for performance testing")
	
	// Ensure realistic distribution of published dates (simulate real traffic patterns)
	now := time.Now()
	
	for i := range dataset.Articles {
		// 70% published in last 30 days, 20% in last 90 days, 10% older
		random := g.randomInt(100)
		var daysAgo int
		
		if random < 70 {
			daysAgo = g.randomInt(30) // Last 30 days
		} else if random < 90 {
			daysAgo = 30 + g.randomInt(60) // 30-90 days ago
		} else {
			daysAgo = 90 + g.randomInt(275) // 90-365 days ago
		}
		
		publishedAt := now.Add(-time.Duration(daysAgo) * 24 * time.Hour)
		dataset.Articles[i].PublishedAt = &publishedAt
		
		// Adjust view counts based on age (older articles have more views)
		ageFactor := float64(daysAgo) / 365.0
		baseViews := 100 + g.randomInt(1000)
		dataset.Articles[i].ViewCount = uint64(float64(baseViews) * (1.0 + ageFactor*2.0))
	}
	
	// Ensure realistic author distribution (some authors are more prolific)
	authorArticleCounts := make(map[uint64]int)
	for _, article := range dataset.Articles {
		authorArticleCounts[article.AuthorID]++
	}
	
	// Make some authors more prolific (Pareto distribution)
	prolificAuthors := make([]uint64, 0)
	for authorID, count := range authorArticleCounts {
		if count > len(dataset.Articles)/len(dataset.Users)*2 { // Authors with 2x average
			prolificAuthors = append(prolificAuthors, authorID)
		}
	}
	
	log.Printf("Performance optimization completed. %d prolific authors identified", len(prolificAuthors))
	return nil
}