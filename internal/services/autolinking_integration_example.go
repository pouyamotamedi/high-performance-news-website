package services

import (
	"context"
	"fmt"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/pkg/database"
)

// AutoLinkingIntegrationExample demonstrates how to integrate the auto-linking service
// with the article creation workflow
func AutoLinkingIntegrationExample() {
	// This is an example of how the auto-linking service would be integrated
	// in a real application setup
	
	// 1. Initialize repositories (these would be real database connections)
	var tagRepo TagRepositoryInterface // This would be *repositories.TagRepository
	
	// 2. Create the auto-linking service
	autoLinkService := NewAutoLinkingService(tagRepo)
	
	// 3. Create article service with auto-linking
	var articleRepo *repositories.ArticleRepository // This would be initialized with real DB
	var db *database.DB // This would be the actual database connection
	articleService := NewArticleService(db, articleRepo, autoLinkService)
	
	// 4. Example usage
	ctx := context.Background()
	
	// Create an article with auto-linking enabled
	article := &models.Article{
		Title:       "The Future of Artificial Intelligence",
		Content:     "Artificial intelligence and machine learning are transforming technology. Python programming is essential for AI development.",
		AutoLinking: true, // Enable auto-linking for this article
		Status:      "published",
	}
	
	// The article service will automatically process the content for auto-linking
	// when Create() is called, assuming tags exist with keywords like:
	// - Technology tag with keywords: ["artificial intelligence", "machine learning", "AI"]
	// - Programming tag with keywords: ["Python", "JavaScript", "Go"]
	
	user := &models.User{ID: 1} // Mock user
	createdArticle, err := articleService.Create(ctx, article, user)
	if err != nil {
		fmt.Printf("Error creating article: %v\n", err)
		return
	}
	
	// The resulting article content would have automatic internal links:
	// "The Future of <a href="/tags/technology">Artificial Intelligence</a>"
	// "<a href="/tags/technology">Artificial intelligence</a> and <a href="/tags/technology">machine learning</a> are transforming technology. <a href="/tags/programming">Python</a> programming is essential for AI development."
	
	fmt.Printf("Created article with auto-linking: %s\n", createdArticle.Title)
	fmt.Printf("Processed content: %s\n", createdArticle.Content)
}

// SetupAutoLinkingService demonstrates how to set up the auto-linking service
// with proper error handling and keyword loading
func SetupAutoLinkingService(tagRepo TagRepositoryInterface) (*AutoLinkingService, error) {
	ctx := context.Background()
	
	// Create the service
	service := NewAutoLinkingService(tagRepo)
	
	// Load keywords from database
	err := service.LoadKeywords(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load keywords: %w", err)
	}
	
	// Validate for conflicts
	conflicts, err := service.ValidateKeywordConflicts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate keyword conflicts: %w", err)
	}
	
	if len(conflicts) > 0 {
		fmt.Printf("Warning: Found keyword conflicts:\n")
		for _, conflict := range conflicts {
			fmt.Printf("  - %s\n", conflict)
		}
	}
	
	// Get stats
	stats := service.GetTrieStats()
	fmt.Printf("Auto-linking service initialized with %d keywords in %d nodes\n", 
		stats["total_keywords"], stats["total_nodes"])
	
	return service, nil
}

// RefreshAutoLinkingKeywords demonstrates how to refresh keywords when tags are updated
func RefreshAutoLinkingKeywords(service *AutoLinkingService) error {
	ctx := context.Background()
	
	// This should be called whenever tags or their keywords are modified
	err := service.RefreshKeywords(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh keywords: %w", err)
	}
	
	fmt.Println("Auto-linking keywords refreshed successfully")
	return nil
}

// ProcessExistingArticles demonstrates how to process existing articles for auto-linking
func ProcessExistingArticles(articleService *ArticleService, autoLinkService *AutoLinkingService) error {
	ctx := context.Background()
	
	// This would be used to process existing articles when auto-linking is first enabled
	// or when keyword banks are significantly updated
	
	// Get articles that need processing (this is a simplified example)
	// In reality, you'd want to process in batches
	articles, _, err := articleService.List(ctx, 100, 0, ArticleFilters{Status: "published"}, "id", "asc")
	if err != nil {
		return fmt.Errorf("failed to get articles: %w", err)
	}
	
	processed := 0
	for _, article := range articles {
		if !article.AutoLinking {
			continue // Skip articles with auto-linking disabled
		}
		
		// Process the article content
		processedContent, err := autoLinkService.ProcessHTMLContent(ctx, &article)
		if err != nil {
			fmt.Printf("Warning: Failed to process article %d: %v\n", article.ID, err)
			continue
		}
		
		// Update the article if content changed
		if processedContent != article.Content {
			updateReq := &UpdateArticleRequest{
				Content: &processedContent,
			}
			
			// You'd need to get the article author for permission checking
			user := &models.User{ID: article.AuthorID}
			_, err = articleService.Update(ctx, article.ID, updateReq, user)
			if err != nil {
				fmt.Printf("Warning: Failed to update article %d: %v\n", article.ID, err)
				continue
			}
			
			processed++
		}
	}
	
	fmt.Printf("Processed %d articles for auto-linking\n", processed)
	return nil
}