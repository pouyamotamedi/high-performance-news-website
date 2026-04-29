package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/internal/services"
)

// Auto-linking handlers with proper implementations

func (r *Router) handleAutoLinkingStats(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get auto-linking service
	autoLinkingService := r.getAutoLinkingService()
	if autoLinkingService == nil {
		c.JSON(http.StatusOK, gin.H{
			"total_keywords": 0,
			"active_tags":    0,
			"status":         "Service not initialized",
		})
		return
	}
	
	// Load keywords first to ensure Trie is populated
	if err := autoLinkingService.LoadKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load keywords",
			"message": err.Error(),
		})
		return
	}
	
	// Get stats from the service
	stats := autoLinkingService.GetTrieStats()
	
	// Get settings
	settingsRepo := r.getAutoLinkingSettingsRepo()
	settings, err := settingsRepo.GetSettings(ctx)
	
	status := "Disabled"
	if err == nil && settings.GlobalEnabled {
		status = "Active"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"total_keywords": stats["total_keywords"],
		"active_tags":    stats["total_tags"],
		"active_banks":   stats["total_banks"],
		"status":         status,
	})
}

func (r *Router) handleAutoLinkingSettings(c *gin.Context) {
	ctx := c.Request.Context()
	
	settingsRepo := r.getAutoLinkingSettingsRepo()
	settings, err := settingsRepo.GetSettings(ctx)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load settings",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"global_enabled":            settings.GlobalEnabled,
		"content_ingestion_enabled": settings.ContentIngestionEnabled,
	})
}

func (r *Router) handleAutoLinkingUpdateSettings(c *gin.Context) {
	ctx := c.Request.Context()
	
	var req struct {
		GlobalEnabled           bool `json:"global_enabled"`
		ContentIngestionEnabled bool `json:"content_ingestion_enabled"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}
	
	settingsRepo := r.getAutoLinkingSettingsRepo()
	settings := &repositories.AutoLinkingSettings{
		GlobalEnabled:           req.GlobalEnabled,
		ContentIngestionEnabled: req.ContentIngestionEnabled,
	}
	
	if err := settingsRepo.UpdateSettings(ctx, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update settings",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"data": gin.H{
			"global_enabled":            req.GlobalEnabled,
			"content_ingestion_enabled": req.ContentIngestionEnabled,
		},
	})
}

func (r *Router) handleAutoLinkingRefresh(c *gin.Context) {
	ctx := c.Request.Context()
	
	autoLinkingService := r.getAutoLinkingService()
	if autoLinkingService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Service not initialized",
			"message": "Auto-linking service is not available",
		})
		return
	}
	
	if err := autoLinkingService.RefreshKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh keywords",
			"message": err.Error(),
		})
		return
	}
	
	stats := autoLinkingService.GetTrieStats()
	
	c.JSON(http.StatusOK, gin.H{
		"message":        "Keywords refreshed successfully",
		"total_keywords": stats["total_keywords"],
		"active_tags":    stats["active_tags"],
	})
}

func (r *Router) handleAutoLinkingConflicts(c *gin.Context) {
	ctx := c.Request.Context()
	
	autoLinkingService := r.getAutoLinkingService()
	if autoLinkingService == nil {
		c.JSON(http.StatusOK, []string{})
		return
	}
	
	// Load keywords first
	if err := autoLinkingService.LoadKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load keywords",
			"message": err.Error(),
		})
		return
	}
	
	conflicts, err := autoLinkingService.ValidateKeywordConflicts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check conflicts",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, conflicts)
}

func (r *Router) handleAutoLinkingTest(c *gin.Context) {
	ctx := c.Request.Context()
	
	var req struct {
		Text string `json:"text"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}
	
	autoLinkingService := r.getAutoLinkingService()
	if autoLinkingService == nil {
		c.JSON(http.StatusOK, gin.H{
			"processed_text": req.Text,
			"matches":        []gin.H{},
		})
		return
	}
	
	// Load keywords first
	if err := autoLinkingService.LoadKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load keywords",
			"message": err.Error(),
		})
		return
	}
	
	// Create a test article
	testArticle := &models.Article{
		Content:     req.Text,
		AutoLinking: true,
	}
	
	// Process the content
	processedText, err := autoLinkingService.ProcessHTMLContent(ctx, testArticle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process text",
			"message": err.Error(),
		})
		return
	}
	
	// Get matches for display
	matches, _ := autoLinkingService.GetKeywordMatches(ctx, req.Text)
	
	matchesResponse := make([]gin.H, 0, len(matches))
	for _, match := range matches {
		matchesResponse = append(matchesResponse, gin.H{
			"keyword": match.Keyword,
			"tag":     match.Tag.Name,
			"slug":    match.Tag.Slug,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"processed_text": processedText,
		"matches":        matchesResponse,
	})
}

func (r *Router) handleGetTagsWithKeywords(c *gin.Context) {
	// Get all tags using the tag service
	if r.handler == nil || r.handler.tagService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	tags, err := r.handler.tagService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load tags",
			"message": err.Error(),
		})
		return
	}
	
	// Convert to pointer array for response
	tagPtrs := make([]*models.Tag, len(tags))
	for i := range tags {
		tagPtrs[i] = &tags[i]
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tags": tagPtrs,
	})
}

func (r *Router) handleReprocessAllArticles(c *gin.Context) {
	ctx := c.Request.Context()
	
	autoLinkingService := r.getAutoLinkingService()
	if autoLinkingService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Auto-linking service not available",
		})
		return
	}
	
	// Get article service
	if r.handler == nil || r.handler.tagService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	db := r.handler.tagService.GetDB()
	
	// STEP 1: Clean all existing auto-links first to prevent duplicates
	cleanQuery := `
		UPDATE articles 
		SET content = REGEXP_REPLACE(
			content, 
			'<a href="[^"]*"[[:space:]]+title="[^"]*">([^<]+)</a>', 
			E'\\1', 
			'g'
		)
		WHERE auto_linking = true AND content LIKE '%<a href=%'
	`
	
	_, err := db.ExecContext(ctx, cleanQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clean existing links",
			"message": err.Error(),
		})
		return
	}
	
	// STEP 2: Load keywords (including keyword banks)
	if err := autoLinkingService.LoadKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load keywords",
			"message": err.Error(),
		})
		return
	}
	
	// Get total count first
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM articles WHERE auto_linking = true").Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count articles",
			"message": err.Error(),
		})
		return
	}
	
	// Process articles synchronously with progress updates
	rows, err := db.Query("SELECT id, content, auto_linking FROM articles WHERE auto_linking = true ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to query articles",
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()
	
	processed := 0
	updated := 0
	
	for rows.Next() {
		var id uint64
		var content string
		var autoLinking bool
		
		if err := rows.Scan(&id, &content, &autoLinking); err != nil {
			continue
		}
		
		// Create article object
		article := &models.Article{
			ID:          id,
			Content:     content,
			AutoLinking: autoLinking,
		}
		
		// Process with auto-linking
		processedContent, err := autoLinkingService.ProcessHTMLContent(context.Background(), article)
		if err != nil {
			processed++
			continue
		}
		
		// Update only if content changed
		if processedContent != content {
			_, err = db.Exec("UPDATE articles SET content = $1, updated_at = NOW() WHERE id = $2", processedContent, id)
			if err == nil {
				updated++
			}
		}
		processed++
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":        "Reprocessing completed",
		"total":          totalCount,
		"processed":      processed,
		"updated":        updated,
		"status":         "completed",
	})
}

// Helper methods to get services

func (r *Router) getAutoLinkingService() *services.AutoLinkingService {
	// Get tag service from handler and create a wrapper repository
	if r.handler != nil && r.handler.tagService != nil {
		// Create a wrapper that implements TagRepositoryInterface
		tagRepoWrapper := &tagServiceWrapper{tagService: r.handler.tagService}
		
		// Check if keyword bank repository is available
		if r.handler.keywordBankRepo != nil {
			// Create service with keyword bank support
			return services.NewAutoLinkingServiceWithKeywordBanks(tagRepoWrapper, r.handler.keywordBankRepo)
		}
		
		return services.NewAutoLinkingService(tagRepoWrapper)
	}
	return nil
}

func (r *Router) getAutoLinkingSettingsRepo() *repositories.AutoLinkingSettingsRepository {
	// Get database connection from tag service
	if r.handler != nil && r.handler.tagService != nil {
		return repositories.NewAutoLinkingSettingsRepository(r.handler.tagService.GetDB())
	}
	return nil
}

// tagServiceWrapper wraps TagService to implement TagRepositoryInterface
type tagServiceWrapper struct {
	tagService *services.TagService
}

func (w *tagServiceWrapper) GetAll() ([]*models.Tag, error) {
	tags, err := w.tagService.GetAll()
	if err != nil {
		return nil, err
	}
	// Convert []models.Tag to []*models.Tag
	result := make([]*models.Tag, len(tags))
	for i := range tags {
		result[i] = &tags[i]
	}
	return result, nil
}

func (w *tagServiceWrapper) GetByID(id uint64) (*models.Tag, error) {
	return w.tagService.GetByID(id)
}

func (w *tagServiceWrapper) GetByArticleID(articleID uint64) ([]*models.Tag, error) {
	// Not implemented - return empty
	return []*models.Tag{}, nil
}

func (w *tagServiceWrapper) GetTotalCount() (int, error) {
	tags, err := w.tagService.GetAll()
	return len(tags), err
}

func (w *tagServiceWrapper) GetAllWithKeywords(ctx context.Context) ([]models.Tag, error) {
	// Use GetAll since TagService doesn't have GetAllWithKeywords
	tags, err := w.tagService.GetAll()
	return tags, err
}

// handleCleanAllLinks removes all auto-generated links from articles
func (r *Router) handleCleanAllLinks(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get database connection
	if r.handler == nil || r.handler.tagService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	db := r.handler.tagService.GetDB()
	
	// Remove all auto-generated links from articles
	query := `
		UPDATE articles 
		SET content = REGEXP_REPLACE(
			content, 
			'<a href="[^"]*"\\s+title="[^"]*">([^<]+)</a>', 
			E'\\1', 
			'g'
		),
		updated_at = NOW()
		WHERE content LIKE '%<a href=%'
	`
	
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clean links",
			"message": err.Error(),
		})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	c.JSON(http.StatusOK, gin.H{
		"message":          "All auto-links cleaned successfully",
		"articles_cleaned": rowsAffected,
		"status":           "completed",
	})
}


// handleGetKeywordBanks returns all keyword banks for display
func (r *Router) handleGetKeywordBanks(c *gin.Context) {
	ctx := c.Request.Context()
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	
	banks, err := r.handler.keywordBankRepo.GetAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get keyword banks",
		})
		return
	}
	
	c.JSON(http.StatusOK, banks)
}
