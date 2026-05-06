package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Development mode handlers for missing routes

func (s *Server) handleDevHomepage(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>High Performance News Website - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .status { background: #d4edda; color: #155724; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .endpoint-group { background: #f8f9fa; padding: 20px; margin: 15px 0; border-radius: 5px; }
        .endpoint { display: block; color: #007bff; text-decoration: none; padding: 8px 0; border-bottom: 1px solid #eee; }
        .endpoint:hover { background: #e9ecef; padding-left: 10px; transition: all 0.2s; }
        .note { background: #fff3cd; color: #856404; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .working { color: #28a745; font-weight: bold; }
        .mock { color: #6c757d; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 High Performance News Website</h1>
        
        <div class="status">
            <strong>Development Mode Active</strong> - Using mock services, no database/cache required
        </div>

        <h2>📡 RSS Feeds <span class="working">(Working)</span></h2>
        <div class="endpoint-group">
            <a href="/rss" class="endpoint">📰 Main RSS Feed - /rss</a>
            <a href="/rss.xml" class="endpoint">📰 Main RSS Feed (XML) - /rss.xml</a>
            <a href="/feed" class="endpoint">📰 Alternative Feed - /feed</a>
            <a href="/rss/category/tech" class="endpoint">🏷️ Technology Category - /rss/category/tech</a>
            <a href="/rss/category/sports" class="endpoint">⚽ Sports Category - /rss/category/sports</a>
            <a href="/rss/tag/news" class="endpoint">🔖 News Tag - /rss/tag/news</a>
            <a href="/rss/tag/breaking" class="endpoint">🔖 Breaking Tag - /rss/tag/breaking</a>
            <a href="/rss/googlenews" class="endpoint">📺 Google News RSS - /rss/googlenews</a>
        </div>

        <h2>🗺️ Google News Sitemaps <span class="working">(Working)</span></h2>
        <div class="endpoint-group">
            <a href="/sitemap-news.xml" class="endpoint">🗺️ News Sitemap - /sitemap-news.xml</a>
            <a href="/sitemap-news-index.xml" class="endpoint">📋 Sitemap Index - /sitemap-news-index.xml</a>
            <a href="/sitemap-news-1.xml" class="endpoint">🗺️ News Sitemap Page 1 - /sitemap-news-1.xml</a>
        </div>

        <h2>🌐 Website Pages <span class="mock">(Mock Data)</span></h2>
        <div class="endpoint-group">
            <a href="/en/category/tech" class="endpoint">💻 Technology Category - /en/category/tech</a>
            <a href="/en/category/sports" class="endpoint">⚽ Sports Category - /en/category/sports</a>
            <a href="/en/category/politics" class="endpoint">🏛️ Politics Category - /en/category/politics</a>
            <a href="/en/tag/breaking" class="endpoint">🔖 Breaking News Tag - /en/tag/breaking</a>
            <a href="/en/tag/news" class="endpoint">📰 News Tag - /en/tag/news</a>
            <a href="/en/latest" class="endpoint">🕐 Latest Articles - /en/latest</a>
            <a href="/en/trending" class="endpoint">📈 Trending Articles - /en/trending</a>
            <a href="/en/article/sample-article-1" class="endpoint">📄 Sample Article - /en/article/sample-article-1</a>
        </div>

        <h2>🔧 System Endpoints</h2>
        <div class="endpoint-group">
            <a href="/health" class="endpoint">❤️ Health Check - /health</a>
        </div>

        <div class="note">
            <strong>Note:</strong> This is development mode with mock data. RSS feeds and sitemaps contain sample content. 
            To use with real data, set up PostgreSQL and DragonflyDB services and restart without development mode.
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleDevCategory(c *gin.Context) {
	slug := c.Param("slug")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + slug + ` Category - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        .article { background: #f8f9fa; padding: 20px; margin: 15px 0; border-radius: 5px; border-left: 4px solid #007bff; }
        .article h3 { margin-top: 0; color: #007bff; }
        .meta { color: #6c757d; font-size: 0.9em; }
        .back { display: inline-block; margin-bottom: 20px; color: #007bff; text-decoration: none; }
        .back:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back">← Back to Home</a>
        <h1>📂 ` + slug + ` Category</h1>
        
        <div class="article">
            <h3>Sample ` + slug + ` Article 1</h3>
            <p class="meta">Published: January 1, 2024 | Author: Demo Author</p>
            <p>This is a sample article from the ` + slug + ` category. In development mode, this shows mock content to demonstrate the category page layout.</p>
        </div>
        
        <div class="article">
            <h3>Sample ` + slug + ` Article 2</h3>
            <p class="meta">Published: January 2, 2024 | Author: Demo Author</p>
            <p>Another sample article from the ` + slug + ` category. The RSS feed for this category is available at <a href="/rss/category/` + slug + `">/rss/category/` + slug + `</a></p>
        </div>
        
        <div class="article">
            <h3>Sample ` + slug + ` Article 3</h3>
            <p class="meta">Published: January 3, 2024 | Author: Demo Author</p>
            <p>A third sample article demonstrating the category functionality in development mode.</p>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleDevTag(c *gin.Context) {
	slug := c.Param("slug")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + slug + ` Tag - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #28a745; padding-bottom: 10px; }
        .article { background: #f8f9fa; padding: 20px; margin: 15px 0; border-radius: 5px; border-left: 4px solid #28a745; }
        .article h3 { margin-top: 0; color: #28a745; }
        .meta { color: #6c757d; font-size: 0.9em; }
        .back { display: inline-block; margin-bottom: 20px; color: #28a745; text-decoration: none; }
        .back:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back">← Back to Home</a>
        <h1>🔖 ` + slug + ` Tag</h1>
        
        <div class="article">
            <h3>Sample Article Tagged with ` + slug + `</h3>
            <p class="meta">Published: January 1, 2024 | Author: Demo Author</p>
            <p>This is a sample article tagged with "` + slug + `". In development mode, this shows mock content to demonstrate the tag page layout.</p>
        </div>
        
        <div class="article">
            <h3>Another ` + slug + ` Tagged Article</h3>
            <p class="meta">Published: January 2, 2024 | Author: Demo Author</p>
            <p>Another sample article tagged with "` + slug + `". The RSS feed for this tag is available at <a href="/rss/tag/` + slug + `">/rss/tag/` + slug + `</a></p>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleDevLatest(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageNum, _ := strconv.Atoi(page)
	
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Latest Articles - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #dc3545; padding-bottom: 10px; }
        .article { background: #f8f9fa; padding: 20px; margin: 15px 0; border-radius: 5px; border-left: 4px solid #dc3545; }
        .article h3 { margin-top: 0; color: #dc3545; }
        .meta { color: #6c757d; font-size: 0.9em; }
        .back { display: inline-block; margin-bottom: 20px; color: #dc3545; text-decoration: none; }
        .back:hover { text-decoration: underline; }
        .pagination { text-align: center; margin-top: 30px; }
        .pagination a { display: inline-block; padding: 8px 16px; margin: 0 4px; color: #dc3545; text-decoration: none; border: 1px solid #dc3545; border-radius: 4px; }
        .pagination a:hover { background: #dc3545; color: white; }
        .pagination .current { background: #dc3545; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back">← Back to Home</a>
        <h1>🕐 Latest Articles (Page ` + page + `)</h1>
        
        <div class="article">
            <h3>Breaking: Latest News Article ` + strconv.Itoa(pageNum*3-2) + `</h3>
            <p class="meta">Published: January ` + strconv.Itoa(pageNum*3-2) + `, 2024 | Category: Breaking News</p>
            <p>This is the latest breaking news article. In development mode, this shows mock content sorted by publication date.</p>
        </div>
        
        <div class="article">
            <h3>Technology Update: Latest Tech News ` + strconv.Itoa(pageNum*3-1) + `</h3>
            <p class="meta">Published: January ` + strconv.Itoa(pageNum*3-1) + `, 2024 | Category: Technology</p>
            <p>Latest technology news and updates. This demonstrates the latest articles functionality in development mode.</p>
        </div>
        
        <div class="article">
            <h3>Sports Update: Latest Sports News ` + strconv.Itoa(pageNum*3) + `</h3>
            <p class="meta">Published: January ` + strconv.Itoa(pageNum*3) + `, 2024 | Category: Sports</p>
            <p>Latest sports news and updates from around the world.</p>
        </div>
        
        <div class="pagination">
            <a href="/latest?page=1">1</a>
            <a href="/latest?page=2">2</a>
            <a href="/latest?page=3">3</a>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleDevTrending(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Trending Articles - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #ffc107; padding-bottom: 10px; }
        .article { background: #f8f9fa; padding: 20px; margin: 15px 0; border-radius: 5px; border-left: 4px solid #ffc107; }
        .article h3 { margin-top: 0; color: #ffc107; }
        .meta { color: #6c757d; font-size: 0.9em; }
        .back { display: inline-block; margin-bottom: 20px; color: #ffc107; text-decoration: none; }
        .back:hover { text-decoration: underline; }
        .trending-badge { background: #ffc107; color: #212529; padding: 2px 8px; border-radius: 12px; font-size: 0.8em; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back">← Back to Home</a>
        <h1>📈 Trending Articles</h1>
        
        <div class="article">
            <h3>🔥 Most Popular Article This Week <span class="trending-badge">#1 Trending</span></h3>
            <p class="meta">Published: January 1, 2024 | Views: 15,420 | Category: Breaking News</p>
            <p>This is the most trending article this week. In development mode, this shows mock trending content based on view counts and engagement.</p>
        </div>
        
        <div class="article">
            <h3>🚀 Viral Technology News <span class="trending-badge">#2 Trending</span></h3>
            <p class="meta">Published: January 2, 2024 | Views: 12,350 | Category: Technology</p>
            <p>A viral technology news article that's trending across social media platforms.</p>
        </div>
        
        <div class="article">
            <h3>⚡ Breaking Sports Story <span class="trending-badge">#3 Trending</span></h3>
            <p class="meta">Published: January 3, 2024 | Views: 9,870 | Category: Sports</p>
            <p>A breaking sports story that's capturing everyone's attention and trending rapidly.</p>
        </div>
        
        <div class="article">
            <h3>🌟 Popular Opinion Piece <span class="trending-badge">#4 Trending</span></h3>
            <p class="meta">Published: January 4, 2024 | Views: 8,650 | Category: Opinion</p>
            <p>A thought-provoking opinion piece that's generating lots of discussion and shares.</p>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleDevArticle(c *gin.Context) {
	slug := c.Param("slug")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + slug + ` - Development Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #6f42c1; padding-bottom: 10px; }
        .meta { color: #6c757d; font-size: 0.9em; margin-bottom: 20px; }
        .content { line-height: 1.6; }
        .back { display: inline-block; margin-bottom: 20px; color: #6f42c1; text-decoration: none; }
        .back:hover { text-decoration: underline; }
        .tags { margin-top: 20px; }
        .tag { display: inline-block; background: #6f42c1; color: white; padding: 4px 12px; border-radius: 16px; font-size: 0.8em; margin-right: 8px; text-decoration: none; }
        .tag:hover { background: #5a32a3; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back">← Back to Home</a>
        <h1>📄 ` + slug + `</h1>
        
        <div class="meta">
            Published: January 1, 2024 | Author: Demo Author | Category: Sample Category | Views: 1,234
        </div>
        
        <div class="content">
            <p>This is a sample article with the slug "` + slug + `". In development mode, this shows mock article content to demonstrate the article page layout and functionality.</p>
            
            <p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
            
            <p>Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.</p>
            
            <p>This article demonstrates the full article view in development mode. In production, this would show real article content from the database.</p>
        </div>
        
        <div class="tags">
            <a href="/en/tag/sample" class="tag">sample</a>
            <a href="/en/tag/development" class="tag">development</a>
            <a href="/en/tag/news" class="tag">news</a>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}