package server

import (
	"github.com/gin-gonic/gin"
)

// renderAdminContent renders the content management admin page
func (s *Server) renderAdminContent(c *gin.Context) {
	title := "Content Management"
	
	contentContent := `
        <div class="dashboard-card">
            <div class="card-title">📝 Content Statistics</div>
            <div>
                <p>Total Articles: <span class="metric">0</span></p>
                <p>Published: <span class="metric">0</span></p>
                <p>Drafts: <span class="metric">0</span></p>
                <p>Pending Review: <span class="metric">0</span></p>
            </div>
        </div>

        <div class="dashboard-card">
            <div class="card-title">Recent Articles</div>
            <div>
                <p>• "No articles yet"</p>
                <p>• "Create your first article"</p>
                <p>• "Economic Analysis Report" - Draft</p>
                <p>• "Weather Alert System" - Pending</p>
            </div>
        </div>

        <div class="dashboard-card">
            <div class="card-title">Content Actions</div>
            <div>
                <a href="/admin/content/create" class="action-button">➕ Create Article</a>
                <a href="/admin/content/articles" class="action-button">📄 Manage Articles</a>
                <a href="/admin/content/categories" class="action-button">📂 Manage Categories</a>
                <a href="/admin/content/tags" class="action-button">🏷️ Manage Tags</a>
                <a href="/admin/content/media" class="action-button">🖼️ Media Library</a>
            </div>
        </div>

        <div class="dashboard-card">
            <div class="card-title">Content Performance</div>
            <div>
                <p>Most Viewed: <span class="metric">0 views</span></p>
                <p>Most Shared: <span class="metric">0 shares</span></p>
                <p>Most Comments: <span class="metric">0 comments</span></p>
                <p>Avg. Read Time: <span class="metric">3.2 min</span></p>
            </div>
        </div>`

	s.renderAdminPage(c, title, "content", contentContent)
}