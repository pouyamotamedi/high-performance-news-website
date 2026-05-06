package testing

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"high-performance-news-website/internal/models"
)

// MetadataGenerator generates realistic SEO and metadata for test content
type MetadataGenerator struct {
	seoTemplates    map[string][]SEOTemplate
	schemaTypes     []string
	canonicalRules  []CanonicalRule
	metaPatterns    map[string][]string
	mutex           sync.RWMutex
}

// SEOTemplate defines templates for generating SEO metadata
type SEOTemplate struct {
	Type        string   `json:"type"`
	TitleFormat string   `json:"title_format"`
	DescFormat  string   `json:"desc_format"`
	Keywords    []string `json:"keywords"`
	Language    string   `json:"language"`
}

// CanonicalRule defines rules for generating canonical URLs
type CanonicalRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Priority    int    `json:"priority"`
}

// initializeSEOTemplates sets up SEO title and description templates
func (mg *MetadataGenerator) initializeSEOTemplates() {
	// English SEO templates
	mg.seoTemplates["meta_title_en"] = []string{
		"%s - Latest News & Updates",
		"Breaking: %s | News Website",
		"%s - Analysis & Reports",
		"Latest %s News - Stay Informed",
		"%s Updates - Breaking News",
		"In-Depth: %s Coverage",
		"%s - News, Analysis & Opinion",
		"Today's %s Headlines",
	}
	
	mg.seoTemplates["meta_description_en"] = []string{
		"Stay updated with the latest %s news, analysis, and expert opinions. Read comprehensive coverage and breaking updates.",
		"Get the latest %s news and in-depth analysis. Stay informed with our comprehensive coverage and expert insights.",
		"Breaking %s news and updates. Read detailed reports, analysis, and expert commentary on the latest developments.",
		"Comprehensive %s coverage with breaking news, analysis, and expert opinions. Stay informed with our latest reports.",
		"Latest %s developments, news, and analysis. Get expert insights and comprehensive coverage of current events.",
	}
	
	// Persian SEO templates
	mg.seoTemplates["meta_title_fa"] = []string{
		"%s - آخرین اخبار و گزارش‌ها",
		"فوری: %s | سایت خبری",
		"%s - تحلیل و بررسی",
		"آخرین اخبار %s - مطلع باشید",
		"به‌روزرسانی %s - اخبار فوری",
		"گزارش کامل: پوشش %s",
		"%s - اخبار، تحلیل و نظرات",
		"تیترهای امروز %s",
	}
	
	mg.seoTemplates["meta_description_fa"] = []string{
		"با آخرین اخبار %s، تحلیل‌ها و نظرات کارشناسان به‌روز باشید. پوشش جامع و به‌روزرسانی‌های فوری را بخوانید.",
		"آخرین اخبار %s و تحلیل‌های عمیق را دریافت کنید. با پوشش جامع و بینش‌های کارشناسی ما مطلع باشید.",
		"اخبار فوری %s و به‌روزرسانی‌ها. گزارش‌های تفصیلی، تحلیل و نظرات کارشناسان درباره آخرین تحولات را بخوانید.",
		"پوشش جامع %s با اخبار فوری، تحلیل و نظرات کارشناسان. با آخرین گزارش‌های ما مطلع باشید.",
		"آخرین تحولات %s، اخبار و تحلیل‌ها. بینش‌های کارشناسی و پوشش جامع رویدادهای جاری را دریافت کنید.",
	}
	
	// Arabic SEO templates
	mg.seoTemplates["meta_title_ar"] = []string{
		"%s - آخر الأخبار والتقارير",
		"عاجل: %s | موقع إخباري",
		"%s - تحليل ومراجعة",
		"آخر أخبار %s - ابق مطلعاً",
		"تحديثات %s - أخبار عاجلة",
		"تقرير شامل: تغطية %s",
		"%s - أخبار وتحليل وآراء",
		"عناوين اليوم %s",
	}
	
	mg.seoTemplates["meta_description_ar"] = []string{
		"ابق مطلعاً على آخر أخبار %s والتحليلات وآراء الخبراء. اقرأ التغطية الشاملة والتحديثات العاجلة.",
		"احصل على آخر أخبار %s والتحليلات المتعمقة. ابق مطلعاً مع تغطيتنا الشاملة ورؤى الخبراء.",
		"أخبار %s العاجلة والتحديثات. اقرأ التقارير التفصيلية والتحليلات وتعليقات الخبراء حول آخر التطورات.",
		"تغطية شاملة لـ %s مع الأخبار العاجلة والتحليلات وآراء الخبراء. ابق مطلعاً مع تقاريرنا الأخيرة.",
		"آخر تطورات %s والأخبار والتحليلات. احصل على رؤى الخبراء والتغطية الشاملة للأحداث الجارية.",
	}
}

// initializeKeywordLists sets up keyword lists for different languages and topics
func (mg *MetadataGenerator) initializeKeywordLists() {
	// English keywords by topic
	mg.keywordLists["politics_en"] = []string{
		"politics", "government", "election", "policy", "parliament", "minister",
		"democracy", "legislation", "campaign", "vote", "political party", "reform",
		"governance", "public policy", "political analysis", "election results",
	}
	
	mg.keywordLists["technology_en"] = []string{
		"technology", "innovation", "artificial intelligence", "software", "hardware",
		"digital transformation", "cybersecurity", "data science", "machine learning",
		"blockchain", "cloud computing", "mobile technology", "internet", "startup",
	}
	
	mg.keywordLists["economy_en"] = []string{
		"economy", "finance", "business", "market", "investment", "banking",
		"economic growth", "inflation", "unemployment", "trade", "stock market",
		"cryptocurrency", "economic policy", "fiscal policy", "monetary policy",
	}
	
	// Persian keywords by topic
	mg.keywordLists["politics_fa"] = []string{
		"سیاست", "دولت", "انتخابات", "سیاست‌گذاری", "مجلس", "وزیر",
		"دموکراسی", "قانون‌گذاری", "کمپین", "رای", "حزب سیاسی", "اصلاحات",
		"حکمرانی", "سیاست عمومی", "تحلیل سیاسی", "نتایج انتخابات",
	}
	
	mg.keywordLists["technology_fa"] = []string{
		"فناوری", "نوآوری", "هوش مصنوعی", "نرم‌افزار", "سخت‌افزار",
		"تحول دیجیتال", "امنیت سایبری", "علم داده", "یادگیری ماشین",
		"بلاک‌چین", "رایانش ابری", "فناوری موبایل", "اینترنت", "استارتاپ",
	}
	
	mg.keywordLists["economy_fa"] = []string{
		"اقتصاد", "مالی", "تجارت", "بازار", "سرمایه‌گذاری", "بانکداری",
		"رشد اقتصادی", "تورم", "بیکاری", "تجارت", "بورس اوراق بهادار",
		"ارز دیجیتال", "سیاست اقتصادی", "سیاست مالی", "سیاست پولی",
	}
	
	// Arabic keywords by topic
	mg.keywordLists["politics_ar"] = []string{
		"سياسة", "حكومة", "انتخابات", "سياسة عامة", "برلمان", "وزير",
		"ديمقراطية", "تشريع", "حملة", "تصويت", "حزب سياسي", "إصلاحات",
		"حكم", "السياسة العامة", "التحليل السياسي", "نتائج الانتخابات",
	}
	
	mg.keywordLists["technology_ar"] = []string{
		"تكنولوجيا", "ابتكار", "الذكاء الاصطناعي", "برمجيات", "أجهزة",
		"التحول الرقمي", "الأمن السيبراني", "علم البيانات", "التعلم الآلي",
		"بلوك تشين", "الحوسبة السحابية", "تكنولوجيا المحمول", "إنترنت", "شركة ناشئة",
	}
	
	mg.keywordLists["economy_ar"] = []string{
		"اقتصاد", "مالية", "أعمال", "سوق", "استثمار", "مصرفية",
		"النمو الاقتصادي", "التضخم", "البطالة", "التجارة", "سوق الأسهم",
		"العملة المشفرة", "السياسة الاقتصادية", "السياسة المالية", "السياسة النقدية",
	}
}

// initializeSchemaTemplates sets up structured data templates
func (mg *MetadataGenerator) initializeSchemaTemplates() {
	// NewsArticle schema template
	mg.schemaTemplates["NewsArticle"] = map[string]interface{}{
		"@context":         "https://schema.org",
		"@type":           "NewsArticle",
		"headline":        "",
		"description":     "",
		"image":          "",
		"datePublished":   "",
		"dateModified":    "",
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  "",
		},
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  "News Website",
			"logo": map[string]interface{}{
				"@type": "ImageObject",
				"url":   "https://example.com/logo.png",
			},
		},
		"mainEntityOfPage": map[string]interface{}{
			"@type": "WebPage",
			"@id":   "",
		},
	}
	
	// Article schema template
	mg.schemaTemplates["Article"] = map[string]interface{}{
		"@context":       "https://schema.org",
		"@type":         "Article",
		"headline":      "",
		"description":   "",
		"image":        "",
		"datePublished": "",
		"dateModified":  "",
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  "",
		},
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  "News Website",
		},
	}
	
	// BlogPosting schema template
	mg.schemaTemplates["BlogPosting"] = map[string]interface{}{
		"@context":       "https://schema.org",
		"@type":         "BlogPosting",
		"headline":      "",
		"description":   "",
		"image":        "",
		"datePublished": "",
		"dateModified":  "",
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  "",
		},
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  "News Website",
		},
		"mainEntityOfPage": map[string]interface{}{
			"@type": "WebPage",
			"@id":   "",
		},
	}
}

// GenerateSEOMetadata generates realistic SEO metadata for a given language
func (mg *MetadataGenerator) GenerateSEOMetadata(lang *Language) models.SEOData {
	langCode := lang.Code
	
	// Generate meta title
	metaTitle := mg.generateMetaTitle(langCode)
	
	// Generate meta description
	metaDescription := mg.generateMetaDescription(langCode)
	
	// Generate keywords
	keywords := mg.generateKeywords(langCode)
	
	// Generate canonical URL
	canonicalURL := mg.generateCanonicalURL()
	
	// Select schema type
	schemaType := mg.selectSchemaType()
	
	return models.SEOData{
		MetaTitle:       metaTitle,
		MetaDescription: metaDescription,
		Keywords:        keywords,
		CanonicalURL:    canonicalURL,
		SchemaType:      schemaType,
	}
}

// generateMetaTitle creates a realistic meta title
func (mg *MetadataGenerator) generateMetaTitle(langCode string) string {
	templateKey := "meta_title_" + langCode
	templates, exists := mg.seoTemplates[templateKey]
	if !exists {
		templates = mg.seoTemplates["meta_title_en"] // Fallback to English
	}
	
	template := templates[mg.randomInt(len(templates))]
	topic := mg.generateTopic(langCode)
	
	title := fmt.Sprintf(template, topic)
	
	// Ensure title is within SEO limits (50-60 characters)
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	
	return title
}

// generateMetaDescription creates a realistic meta description
func (mg *MetadataGenerator) generateMetaDescription(langCode string) string {
	templateKey := "meta_description_" + langCode
	templates, exists := mg.seoTemplates[templateKey]
	if !exists {
		templates = mg.seoTemplates["meta_description_en"] // Fallback to English
	}
	
	template := templates[mg.randomInt(len(templates))]
	topic := mg.generateTopic(langCode)
	
	description := fmt.Sprintf(template, topic)
	
	// Ensure description is within SEO limits (150-160 characters)
	if len(description) > 160 {
		description = description[:157] + "..."
	}
	
	return description
}

// generateKeywords creates realistic keywords for the content
func (mg *MetadataGenerator) generateKeywords(langCode string) []string {
	topics := []string{"politics", "technology", "economy", "sports", "culture"}
	topic := topics[mg.randomInt(len(topics))]
	
	keywordKey := topic + "_" + langCode
	keywords, exists := mg.keywordLists[keywordKey]
	if !exists {
		keywords = mg.keywordLists[topic+"_en"] // Fallback to English
	}
	
	// Select 3-7 random keywords
	keywordCount := 3 + mg.randomInt(5)
	var selectedKeywords []string
	usedIndices := make(map[int]bool)
	
	for len(selectedKeywords) < keywordCount && len(selectedKeywords) < len(keywords) {
		index := mg.randomInt(len(keywords))
		if !usedIndices[index] {
			selectedKeywords = append(selectedKeywords, keywords[index])
			usedIndices[index] = true
		}
	}
	
	return selectedKeywords
}

// generateCanonicalURL creates a canonical URL
func (mg *MetadataGenerator) generateCanonicalURL() string {
	// Generate a realistic canonical URL
	domains := []string{
		"https://example.com",
		"https://news-website.com",
		"https://dailynews.org",
	}
	
	domain := domains[mg.randomInt(len(domains))]
	slug := mg.generateURLSlug()
	
	return fmt.Sprintf("%s/en/article/%s", domain, slug)
}

// generateURLSlug creates a URL-friendly slug
func (mg *MetadataGenerator) generateURLSlug() string {
	words := []string{
		"breaking", "news", "update", "report", "analysis", "latest",
		"government", "economy", "technology", "sports", "culture",
		"politics", "business", "international", "local", "health",
	}
	
	slugLength := 2 + mg.randomInt(4) // 2-5 words
	var slugWords []string
	
	for i := 0; i < slugLength; i++ {
		word := words[mg.randomInt(len(words))]
		slugWords = append(slugWords, word)
	}
	
	return strings.Join(slugWords, "-")
}

// selectSchemaType selects an appropriate schema type
func (mg *MetadataGenerator) selectSchemaType() string {
	schemaTypes := []string{"NewsArticle", "Article", "BlogPosting"}
	weights := []int{60, 30, 10} // 60% NewsArticle, 30% Article, 10% BlogPosting
	
	random := mg.randomInt(100)
	cumulative := 0
	
	for i, weight := range weights {
		cumulative += weight
		if random < cumulative {
			return schemaTypes[i]
		}
	}
	
	return "NewsArticle" // fallback
}

// generateTopic creates a topic for SEO content
func (mg *MetadataGenerator) generateTopic(langCode string) string {
	topics := map[string][]string{
		"en": {
			"Politics", "Technology", "Economy", "Sports", "Culture",
			"Health", "Education", "Environment", "Business", "Science",
		},
		"fa": {
			"سیاست", "فناوری", "اقتصاد", "ورزش", "فرهنگ",
			"بهداشت", "آموزش", "محیط‌زیست", "تجارت", "علم",
		},
		"ar": {
			"السياسة", "التكنولوجيا", "الاقتصاد", "الرياضة", "الثقافة",
			"الصحة", "التعليم", "البيئة", "الأعمال", "العلوم",
		},
	}
	
	langTopics, exists := topics[langCode]
	if !exists {
		langTopics = topics["en"] // Fallback to English
	}
	
	return langTopics[mg.randomInt(len(langTopics))]
}

// GenerateStructuredData generates structured data for an article
func (mg *MetadataGenerator) GenerateStructuredData(schemaType, title, description, authorName string) map[string]interface{} {
	template, exists := mg.schemaTemplates[schemaType]
	if !exists {
		template = mg.schemaTemplates["NewsArticle"] // Fallback
	}
	
	// Deep copy template
	schema := mg.deepCopyMap(template)
	
	// Fill in the data
	schema["headline"] = title
	schema["description"] = description
	schema["datePublished"] = time.Now().Format(time.RFC3339)
	schema["dateModified"] = time.Now().Format(time.RFC3339)
	
	// Set author
	if author, ok := schema["author"].(map[string]interface{}); ok {
		author["name"] = authorName
	}
	
	// Generate image URL
	schema["image"] = mg.generateImageURL()
	
	// Set main entity page
	if mainEntity, ok := schema["mainEntityOfPage"].(map[string]interface{}); ok {
		mainEntity["@id"] = mg.generateCanonicalURL()
	}
	
	return schema
}

// generateImageURL creates a realistic image URL
func (mg *MetadataGenerator) generateImageURL() string {
	imageTypes := []string{"news", "politics", "technology", "sports", "business"}
	imageType := imageTypes[mg.randomInt(len(imageTypes))]
	imageID := mg.randomInt(1000)
	
	return fmt.Sprintf("https://example.com/images/%s/%d.jpg", imageType, imageID)
}

// deepCopyMap creates a deep copy of a map
func (mg *MetadataGenerator) deepCopyMap(original map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	
	for key, value := range original {
		switch v := value.(type) {
		case map[string]interface{}:
			copy[key] = mg.deepCopyMap(v)
		case []interface{}:
			copySlice := make([]interface{}, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					copySlice[i] = mg.deepCopyMap(itemMap)
				} else {
					copySlice[i] = item
				}
			}
			copy[key] = copySlice
		default:
			copy[key] = value
		}
	}
	
	return copy
}

// GenerateOpenGraphMetadata generates Open Graph metadata
func (mg *MetadataGenerator) GenerateOpenGraphMetadata(title, description, imageURL, url string) map[string]string {
	return map[string]string{
		"og:title":       title,
		"og:description": description,
		"og:image":       imageURL,
		"og:url":         url,
		"og:type":        "article",
		"og:site_name":   "News Website",
	}
}

// GenerateTwitterCardMetadata generates Twitter Card metadata
func (mg *MetadataGenerator) GenerateTwitterCardMetadata(title, description, imageURL string) map[string]string {
	return map[string]string{
		"twitter:card":        "summary_large_image",
		"twitter:title":       title,
		"twitter:description": description,
		"twitter:image":       imageURL,
		"twitter:site":        "@newswebsite",
	}
}

// ValidateSEOMetadata validates SEO metadata for compliance
func (mg *MetadataGenerator) ValidateSEOMetadata(seo models.SEOData) []string {
	var issues []string
	
	// Check meta title length
	if len(seo.MetaTitle) == 0 {
		issues = append(issues, "Meta title is empty")
	} else if len(seo.MetaTitle) > 60 {
		issues = append(issues, "Meta title exceeds 60 characters")
	} else if len(seo.MetaTitle) < 30 {
		issues = append(issues, "Meta title is too short (less than 30 characters)")
	}
	
	// Check meta description length
	if len(seo.MetaDescription) == 0 {
		issues = append(issues, "Meta description is empty")
	} else if len(seo.MetaDescription) > 160 {
		issues = append(issues, "Meta description exceeds 160 characters")
	} else if len(seo.MetaDescription) < 120 {
		issues = append(issues, "Meta description is too short (less than 120 characters)")
	}
	
	// Check keywords
	if len(seo.Keywords) == 0 {
		issues = append(issues, "No keywords specified")
	} else if len(seo.Keywords) > 10 {
		issues = append(issues, "Too many keywords (more than 10)")
	}
	
	// Check canonical URL
	if seo.CanonicalURL == "" {
		issues = append(issues, "Canonical URL is empty")
	}
	
	// Check schema type
	validSchemaTypes := map[string]bool{
		"NewsArticle": true,
		"Article":     true,
		"BlogPosting": true,
	}
	if !validSchemaTypes[seo.SchemaType] {
		issues = append(issues, "Invalid schema type")
	}
	
	return issues
}

// randomInt generates a random integer between 0 and max-1
func (mg *MetadataGenerator) randomInt(max int) int {
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