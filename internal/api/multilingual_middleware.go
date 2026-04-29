package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MultilingualMiddleware handles language-aware routing
func MultilingualMiddleware(multilingualService *services.MultilingualService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract language from URL
		languageCode, remainingPath := services.ExtractLanguageFromURL(c.Request.URL.Path)
		
		// Validate language code
		if err := multilingualService.ValidateLanguageCode(languageCode); err != nil {
			// If invalid language code, use default and don't modify path
			languageCode = multilingualService.GetDefaultLanguage()
		} else {
			// If valid language code was extracted, update the request path
			if remainingPath != c.Request.URL.Path {
				c.Request.URL.Path = "/" + remainingPath
			}
		}
		
		// Store language information in context
		c.Set("language_code", languageCode)
		c.Set("original_path", c.Request.URL.Path)
		
		// Get language configuration
		config, err := multilingualService.GetLanguageConfig()
		if err == nil {
			c.Set("language_config", config)
			c.Set("is_rtl", config.IsRTL(languageCode))
			c.Set("is_default_language", languageCode == config.DefaultLanguage)
		}
		
		c.Next()
	}
}

// LanguageRedirectMiddleware redirects to the appropriate language version
func LanguageRedirectMiddleware(multilingualService *services.MultilingualService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		
		// Skip for static assets
		if strings.HasPrefix(c.Request.URL.Path, "/static/") ||
		   strings.HasPrefix(c.Request.URL.Path, "/assets/") ||
		   strings.HasPrefix(c.Request.URL.Path, "/favicon.ico") {
			c.Next()
			return
		}
		
		// Get preferred language from Accept-Language header
		acceptLanguage := c.GetHeader("Accept-Language")
		preferredLang := parseAcceptLanguage(acceptLanguage)
		
		// Get active languages
		config, err := multilingualService.GetLanguageConfig()
		if err != nil {
			c.Next()
			return
		}
		
		// Check if preferred language is active
		var targetLang string
		for _, activeLang := range config.ActiveLanguages {
			if activeLang == preferredLang {
				targetLang = preferredLang
				break
			}
		}
		
		// If no preferred language found, use default
		if targetLang == "" {
			targetLang = config.DefaultLanguage
		}
		
		// Extract current language from URL
		currentLang, _ := services.ExtractLanguageFromURL(c.Request.URL.Path)
		
		// If current language matches target, continue
		if currentLang == targetLang {
			c.Next()
			return
		}
		
		// If target is default language and no language in URL, continue
		if targetLang == config.DefaultLanguage && currentLang == config.DefaultLanguage {
			c.Next()
			return
		}
		
		// Redirect to appropriate language version
		var redirectURL string
		if targetLang == config.DefaultLanguage {
			// Remove language prefix for default language
			_, remainingPath := services.ExtractLanguageFromURL(c.Request.URL.Path)
			redirectURL = "/" + remainingPath
		} else {
			// Add language prefix
			_, remainingPath := services.ExtractLanguageFromURL(c.Request.URL.Path)
			redirectURL = "/" + targetLang + "/" + remainingPath
		}
		
		// Clean up double slashes
		redirectURL = strings.ReplaceAll(redirectURL, "//", "/")
		
		c.Redirect(http.StatusFound, redirectURL)
		c.Abort()
	}
}

// parseAcceptLanguage parses the Accept-Language header and returns the preferred language
func parseAcceptLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "fa" // Default to Persian
	}
	
	// Simple parsing - take the first language code
	languages := strings.Split(acceptLanguage, ",")
	if len(languages) == 0 {
		return "fa"
	}
	
	// Extract language code (before any semicolon or dash)
	firstLang := strings.TrimSpace(languages[0])
	if idx := strings.Index(firstLang, ";"); idx != -1 {
		firstLang = firstLang[:idx]
	}
	if idx := strings.Index(firstLang, "-"); idx != -1 {
		firstLang = firstLang[:idx]
	}
	
	// Ensure it's 2 characters
	if len(firstLang) >= 2 {
		return strings.ToLower(firstLang[:2])
	}
	
	return "fa"
}

// GetLanguageFromContext extracts language code from Gin context
func GetLanguageFromContext(c *gin.Context) string {
	if lang, exists := c.Get("language_code"); exists {
		if langStr, ok := lang.(string); ok {
			return langStr
		}
	}
	return "fa" // Default to Persian
}

// IsRTLFromContext checks if current language is RTL from Gin context
func IsRTLFromContext(c *gin.Context) bool {
	if isRTL, exists := c.Get("is_rtl"); exists {
		if rtl, ok := isRTL.(bool); ok {
			return rtl
		}
	}
	return true // Default to RTL (Persian)
}

// IsDefaultLanguageFromContext checks if current language is default from Gin context
func IsDefaultLanguageFromContext(c *gin.Context) bool {
	if isDefault, exists := c.Get("is_default_language"); exists {
		if def, ok := isDefault.(bool); ok {
			return def
		}
	}
	return true // Default to true (Persian is default)
}

// GetLanguageConfigFromContext extracts language config from Gin context
func GetLanguageConfigFromContext(c *gin.Context) *models.LanguageConfig {
	if config, exists := c.Get("language_config"); exists {
		if cfg, ok := config.(*models.LanguageConfig); ok {
			return cfg
		}
	}
	// Return default config
	return &models.LanguageConfig{
		DefaultLanguage:  "fa",
		FallbackLanguage: "fa",
		ActiveLanguages:  []string{"fa"},
		RTLLanguages:     []string{"fa"},
	}
}

// GenerateLanguageAlternates generates alternate language URLs for the current page
func GenerateLanguageAlternates(c *gin.Context, multilingualService *services.MultilingualService, contentType, slug string) map[string]string {
	currentLang := GetLanguageFromContext(c)
	
	routeInfo, err := multilingualService.GenerateLanguageRouteInfo(contentType, slug, currentLang)
	if err != nil {
		return make(map[string]string)
	}
	
	return routeInfo.AlternateURLs
}

// SetLanguageHeaders sets appropriate headers for multilingual content
func SetLanguageHeaders(c *gin.Context, multilingualService *services.MultilingualService) {
	languageCode := GetLanguageFromContext(c)
	
	// Set Content-Language header
	c.Header("Content-Language", languageCode)
	
	// Set Vary header to indicate language-dependent content
	c.Header("Vary", "Accept-Language")
	
	// Get language info
	isRTL, err := multilingualService.IsRTLLanguage(languageCode)
	if err == nil && isRTL {
		c.Header("X-Language-Direction", "rtl")
	} else {
		c.Header("X-Language-Direction", "ltr")
	}
}