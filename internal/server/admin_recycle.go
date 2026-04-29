package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) renderRecycleBin(c *gin.Context) {
	title := "Recycle Bin"
	content := `
        <div class="dashboard-card">
            <div class="card-title">🗑️ Recycle Bin - Deleted Articles</div>
            <p style="color: #6b7280; margin-bottom: 1rem;">Articles in the recycle bin can be restored or permanently deleted.</p>
            
            <div id="trashContainer">
                <div style="text-align: center; padding: 2rem; color: #6b7280;">Loading deleted articles...</div>
            </div>
        </div>

        <script>
            document.addEventListener('DOMContentLoaded', function() {
                loadDeletedArticles();
            });

            async function loadDeletedArticles() {
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/articles?status=deleted&limit=1000', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });

                    if (response.ok) {
                        const data = await response.json();
                        const articles = data.articles || [];
                        renderDeletedArticles(articles);
                    } else {
                        document.getElementById('trashContainer').innerHTML = 
                            '<div style="text-align: center; padding: 2rem; color: #ef4444;">Failed to load deleted articles</div>';
                    }
                } catch (error) {
                    document.getElementById('trashContainer').innerHTML = 
                        '<div style="text-align: center; padding: 2rem; color: #ef4444;">Error: ' + error.message + '</div>';
                }
            }

            function renderDeletedArticles(articles) {
                const container = document.getElementById('trashContainer');
                
                if (articles.length === 0) {
                    container.innerHTML = '<div style="text-align: center; padding: 2rem; color: #6b7280;">🎉 Recycle bin is empty!</div>';
                    return;
                }

                let html = '<table style="width: 100%; border-collapse: collapse;">';
                html += '<thead><tr style="background-color: #f9fafb;">';
                html += '<th style="padding: 0.75rem; text-align: left; border-bottom: 1px solid #e5e7eb;">Title</th>';
                html += '<th style="padding: 0.75rem; text-align: left; border-bottom: 1px solid #e5e7eb;">Deleted Date</th>';
                html += '<th style="padding: 0.75rem; text-align: left; border-bottom: 1px solid #e5e7eb;">Actions</th>';
                html += '</tr></thead><tbody>';

                articles.forEach(article => {
                    const deletedDate = new Date(article.updated_at).toLocaleDateString();
                    
                    html += '<tr style="border-bottom: 1px solid #e5e7eb;">';
                    html += '<td style="padding: 0.75rem;"><strong>' + escapeHtml(article.title) + '</strong><br><small style="color: #6b7280;">' + escapeHtml(article.slug) + '</small></td>';
                    html += '<td style="padding: 0.75rem;">' + deletedDate + '</td>';
                    html += '<td style="padding: 0.75rem;">';
                    html += '<button onclick="restoreArticle(' + article.id + ')" style="padding: 0.25rem 0.5rem; margin-right: 0.5rem; background-color: #10b981; color: white; border: none; border-radius: 4px; cursor: pointer;" title="Restore Article">↩️</button>';
                    html += '<button onclick="permanentlyDelete(' + article.id + ')" style="padding: 0.25rem 0.5rem; background-color: #ef4444; color: white; border: none; border-radius: 4px; cursor: pointer;" title="Delete Forever">🗑️</button>';
                    html += '</td>';
                    html += '</tr>';
                });

                html += '</tbody></table>';
                html += '<div style="margin-top: 1rem; text-align: center;">';
                html += '<a href="/admin/content/articles" style="padding: 0.5rem 1rem; background-color: #6b7280; color: white; text-decoration: none; border-radius: 6px;">← Back to Articles</a>';
                html += '</div>';
                
                container.innerHTML = html;
            }

            async function restoreArticle(id) {
                if (!confirm('Are you sure you want to restore this article?')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/articles/' + id, {
                        method: 'PATCH',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify({ status: 'draft' })
                    });

                    if (response.ok) {
                        alert('Article restored successfully!');
                        loadDeletedArticles();
                    } else {
                        alert('Failed to restore article');
                    }
                } catch (error) {
                    alert('Error restoring article: ' + error.message);
                }
            }

            async function permanentlyDelete(id) {
                if (!confirm('Are you sure you want to PERMANENTLY delete this article? This action cannot be undone!')) {
                    return;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/articles/' + id + '?permanent=true', {
                        method: 'DELETE',
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });

                    if (response.ok) {
                        alert('Article permanently deleted');
                        loadDeletedArticles();
                    } else {
                        const errorData = await response.json();
                        alert('Failed to permanently delete article: ' + (errorData.message || 'Unknown error'));
                    }
                } catch (error) {
                    alert('Error permanently deleting article: ' + error.message);
                }
            }

            function escapeHtml(text) {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            }
        </script>
    `
	s.renderAdminPage(c, title, "content", content)
}

