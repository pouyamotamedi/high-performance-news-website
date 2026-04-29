package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"high-performance-news-website/internal/models"
)

// BulkAIService handles bulk AI operations for content optimization
type BulkAIService struct {
	aiService     AIService
	articleRepo   ArticleRepositoryInterface
	maxConcurrent int
	batchSize     int
}

// NewBulkAIService creates a new bulk AI service
func NewBulkAIService(aiService AIService, articleRepo ArticleRepositoryInterface) *BulkAIService {
	return &BulkAIService{
		aiService:     aiService,
		articleRepo:   articleRepo,
		maxConcurrent: 5,  // Limit concurrent AI requests to avoid rate limits
		batchSize:     100, // Process articles in batches
	}
}

// BulkOptimizationRequest represents a bulk optimization request
type BulkOptimizationRequest struct {
	ArticleIDs       []uint64 `json:"article_ids,omitempty"`
	CategoryID       *uint64  `json:"category_id,omitempty"`
	TagID            *uint64  `json:"tag_id,omitempty"`
	DateFrom         *time.Time `json:"date_from,omitempty"`
	DateTo           *time.Time `json:"date_to,omitempty"`
	OptimizeTitle    bool     `json:"optimize_title"`
	OptimizeMeta     bool     `json:"optimize_meta"`
	CheckQuality     bool     `json:"check_quality"`
	GenerateSchema   bool     `json:"generate_schema"`
	MaxArticles      int      `json:"max_articles"`
}

// BulkOptimizationResult represents the result of bulk optimization
type BulkOptimizationResult struct {
	TotalProcessed   int                        `json:"total_processed"`
	TotalOptimized   int                        `json:"total_optimized"`
	TotalErrors      int                        `json:"total_errors"`
	ProcessingTimeMs int64                      `json:"processing_time_ms"`
	Results          []ArticleOptimizationResult `json:"results"`
	Errors           []BulkOptimizationError    `json:"errors"`
}

// ArticleOptimizationResult represents optimization result for a single article
type ArticleOptimizationResult struct {
	ArticleID        uint64                 `json:"article_id"`
	OriginalTitle    string                 `json:"original_title"`
	OptimizedTitle   *string                `json:"optimized_title,omitempty"`
	OriginalMeta     string                 `json:"original_meta"`
	OptimizedMeta    *string                `json:"optimized_meta,omitempty"`
	QualityScore     *float64               `json:"quality_score,omitempty"`
	AIFeedback       *models.AIFeedback     `json:"ai_feedback,omitempty"`
	SchemaGenerated  bool                   `json:"schema_generated"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
}

// BulkOptimizationError represents an error during bulk optimization
type BulkOptimizationError struct {
	ArticleID uint64 `json:"article_id"`
	Error     string `json:"error"`
}

// OptimizeArticlesBulk performs bulk optimization on articles
func (b *BulkAIService) OptimizeArticlesBulk(ctx context.Context, req *BulkOptimizationRequest) (*BulkOptimizationResult, error) {
	startTime := time.Now()
	
	// Get articles to process
	articles, err := b.getArticlesForOptimization(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}
	
	if len(articles) == 0 {
		return &BulkOptimizationResult{
			TotalProcessed:   0,
			TotalOptimized:   0,
			TotalErrors:      0,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			Results:          []ArticleOptimizationResult{},
			Errors:           []BulkOptimizationError{},
		}, nil
	}
	
	// Limit the number of articles if specified
	if req.MaxArticles > 0 && len(articles) > req.MaxArticles {
		articles = articles[:req.MaxArticles]
	}
	
	// Process articles in batches with concurrency control
	results := make([]ArticleOptimizationResult, 0, len(articles))
	errors := make([]BulkOptimizationError, 0)
	
	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, b.maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Process articles in batches
	for i := 0; i < len(articles); i += b.batchSize {
		end := i + b.batchSize
		if end > len(articles) {
			end = len(articles)
		}
		
		batch := articles[i:end]
		
		for _, article := range batch {
			wg.Add(1)
			go func(art *models.Article) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				// Process single article
				result, err := b.optimizeSingleArticle(ctx, art, req)
				
				mu.Lock()
				if err != nil {
					errors = append(errors, BulkOptimizationError{
						ArticleID: art.ID,
						Error:     err.Error(),
					})
				} else {
					results = append(results, *result)
				}
				mu.Unlock()
			}(article)
		}
		
		// Wait for batch to complete before starting next batch
		wg.Wait()
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	
	totalOptimized := 0
	for _, result := range results {
		if result.OptimizedTitle != nil || result.OptimizedMeta != nil || result.SchemaGenerated {
			totalOptimized++
		}
	}
	
	return &BulkOptimizationResult{
		TotalProcessed:   len(articles),
		TotalOptimized:   totalOptimized,
		TotalErrors:      len(errors),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		Results:          results,
		Errors:           errors,
	}, nil
}

// optimizeSingleArticle optimizes a single article
func (b *BulkAIService) optimizeSingleArticle(ctx context.Context, article *models.Article, req *BulkOptimizationRequest) (*ArticleOptimizationResult, error) {
	startTime := time.Now()
	
	result := &ArticleOptimizationResult{
		ArticleID:        article.ID,
		OriginalTitle:    article.Title,
		OriginalMeta:     article.SEOData.MetaDescription,
		ProcessingTimeMs: 0,
	}
	
	// Check quality if requested
	if req.CheckQuality {
		feedback, err := b.aiService.AnalyzeContent(article.Title, article.Content)
		if err != nil {
			log.Printf("Failed to analyze content for article %d: %v", article.ID, err)
		} else {
			result.QualityScore = &feedback.QualityScore
			result.AIFeedback = feedback
		}
	}
	
	// Optimize title if requested
	if req.OptimizeTitle {
		optimizedTitle, err := b.aiService.GenerateTitle(article.Content)
		if err != nil {
			log.Printf("Failed to generate title for article %d: %v", article.ID, err)
		} else {
			result.OptimizedTitle = &optimizedTitle
			
			// Update article in database
			article.Title = optimizedTitle
			if err := b.articleRepo.Update(article); err != nil {
				log.Printf("Failed to update article title %d: %v", article.ID, err)
			}
		}
	}
	
	// Optimize meta description if requested
	if req.OptimizeMeta {
		optimizedMeta, err := b.aiService.GenerateMetaDescription(article.Title, article.Content)
		if err != nil {
			log.Printf("Failed to generate meta description for article %d: %v", article.ID, err)
		} else {
			result.OptimizedMeta = &optimizedMeta
			
			// Update article in database
			article.SEOData.MetaDescription = optimizedMeta
			if err := b.articleRepo.Update(article); err != nil {
				log.Printf("Failed to update article meta %d: %v", article.ID, err)
			}
		}
	}
	
	// Generate schema if requested
	if req.GenerateSchema {
		// This would integrate with the SEO service to generate structured data
		result.SchemaGenerated = true
	}
	
	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// getArticlesForOptimization retrieves articles based on the request criteria
func (b *BulkAIService) getArticlesForOptimization(req *BulkOptimizationRequest) ([]*models.Article, error) {
	// If specific article IDs are provided
	if len(req.ArticleIDs) > 0 {
		articles := make([]*models.Article, 0, len(req.ArticleIDs))
		for _, id := range req.ArticleIDs {
			article, err := b.articleRepo.GetByID(id)
			if err != nil {
				log.Printf("Failed to get article %d: %v", id, err)
				continue
			}
			articles = append(articles, article)
		}
		return articles, nil
	}
	
	// Build filter criteria
	filter := &models.ArticleFilter{
		Status: "published",
	}
	
	if req.CategoryID != nil {
		filter.CategoryID = req.CategoryID
	}
	
	if req.TagID != nil {
		filter.TagID = req.TagID
	}
	
	if req.DateFrom != nil {
		filter.PublishedAfter = req.DateFrom
	}
	
	if req.DateTo != nil {
		filter.PublishedBefore = req.DateTo
	}
	
	// Get articles with pagination
	limit := 1000 // Default limit
	if req.MaxArticles > 0 && req.MaxArticles < limit {
		limit = req.MaxArticles
	}
	
	return b.articleRepo.GetByFilter(filter, 0, limit)
}

// GenerateQualityReport generates a quality report for articles
func (b *BulkAIService) GenerateQualityReport(ctx context.Context, articleIDs []uint64) (*QualityReport, error) {
	startTime := time.Now()
	
	report := &QualityReport{
		TotalArticles:    len(articleIDs),
		ProcessedAt:      time.Now(),
		Articles:         make([]ArticleQualityResult, 0, len(articleIDs)),
		QualityStats:     &QualityStats{},
	}
	
	// Process articles with concurrency control
	semaphore := make(chan struct{}, b.maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	qualityScores := make([]float64, 0, len(articleIDs))
	
	for _, articleID := range articleIDs {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Get article
			article, err := b.articleRepo.GetByID(id)
			if err != nil {
				mu.Lock()
				report.Errors = append(report.Errors, fmt.Sprintf("Failed to get article %d: %v", id, err))
				mu.Unlock()
				return
			}
			
			// Analyze content
			feedback, err := b.aiService.AnalyzeContent(article.Title, article.Content)
			if err != nil {
				mu.Lock()
				report.Errors = append(report.Errors, fmt.Sprintf("Failed to analyze article %d: %v", id, err))
				mu.Unlock()
				return
			}
			
			result := ArticleQualityResult{
				ArticleID:    id,
				Title:        article.Title,
				QualityScore: feedback.QualityScore,
				Issues:       len(feedback.Issues),
				Suggestions:  len(feedback.Suggestions),
				Flagged:      len(feedback.FlaggedContent) > 0,
				AIFeedback:   feedback,
			}
			
			mu.Lock()
			report.Articles = append(report.Articles, result)
			qualityScores = append(qualityScores, feedback.QualityScore)
			mu.Unlock()
		}(articleID)
	}
	
	wg.Wait()
	
	// Calculate statistics
	if len(qualityScores) > 0 {
		var sum float64
		min := qualityScores[0]
		max := qualityScores[0]
		
		for _, score := range qualityScores {
			sum += score
			if score < min {
				min = score
			}
			if score > max {
				max = score
			}
		}
		
		report.QualityStats.AverageScore = sum / float64(len(qualityScores))
		report.QualityStats.MinScore = min
		report.QualityStats.MaxScore = max
		
		// Count quality categories
		for _, score := range qualityScores {
			if score >= 0.8 {
				report.QualityStats.HighQuality++
			} else if score >= 0.6 {
				report.QualityStats.MediumQuality++
			} else {
				report.QualityStats.LowQuality++
			}
		}
	}
	
	report.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	return report, nil
}

// QualityReport represents a quality analysis report
type QualityReport struct {
	TotalArticles    int                     `json:"total_articles"`
	ProcessedAt      time.Time               `json:"processed_at"`
	ProcessingTimeMs int64                   `json:"processing_time_ms"`
	Articles         []ArticleQualityResult  `json:"articles"`
	QualityStats     *QualityStats           `json:"quality_stats"`
	Errors           []string                `json:"errors,omitempty"`
}

// ArticleQualityResult represents quality analysis for a single article
type ArticleQualityResult struct {
	ArticleID    uint64             `json:"article_id"`
	Title        string             `json:"title"`
	QualityScore float64            `json:"quality_score"`
	Issues       int                `json:"issues_count"`
	Suggestions  int                `json:"suggestions_count"`
	Flagged      bool               `json:"flagged"`
	AIFeedback   *models.AIFeedback `json:"ai_feedback,omitempty"`
}

// QualityStats represents quality statistics
type QualityStats struct {
	AverageScore  float64 `json:"average_score"`
	MinScore      float64 `json:"min_score"`
	MaxScore      float64 `json:"max_score"`
	HighQuality   int     `json:"high_quality"`   // >= 0.8
	MediumQuality int     `json:"medium_quality"` // 0.6-0.8
	LowQuality    int     `json:"low_quality"`    // < 0.6
}