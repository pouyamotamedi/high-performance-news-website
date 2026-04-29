package server

import (
	"github.com/gin-gonic/gin"
)

// renderEditArticle renders the article editing page with full create article interface
func (s *Server) renderEditArticle(c *gin.Context) {
	articleID := c.Param("id")
	title := "Edit Article #" + articleID
	
	content := `<style>.editor-container { display: grid; grid-template-columns: 2fr 1fr; gap: 2rem; }.main-editor { display: flex; flex-direction: column; gap: 1rem; }.sidebar { display: flex; flex-direction: column; gap: 1rem; }.editor-toolbar { display: flex; gap: 0.5rem; padding: 0.5rem; border: 1px solid #d1d5db; border-bottom: none; border-radius: 6px 6px 0 0; background: #f9fafb; flex-wrap: wrap; }.toolbar-btn { padding: 0.5rem; border: 1px solid #d1d5db; background: white; border-radius: 4px; cursor: pointer; font-size: 0.9rem; }.toolbar-btn:hover { background: #f3f4f6; }.toolbar-btn.active { background: #3b82f6; color: white; }.content-editor { min-height: 400px; width: 100%; padding: 1rem; border: 1px solid #d1d5db; border-top: none; border-radius: 0 0 6px 6px; font-family: inherit; font-size: 1rem; resize: vertical; box-sizing: border-box; }.featured-image-preview { width: 100%; height: 200px; border: 2px dashed #d1d5db; border-radius: 8px; display: flex; align-items: center; justify-content: center; cursor: pointer; background: #f9fafb; }.featured-image-preview img { max-width: 100%; max-height: 100%; object-fit: cover; border-radius: 6px; }.tag-input { display: flex; flex-wrap: wrap; gap: 0.5rem; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; min-height: 2.5rem; }.tag-item { background: #3b82f6; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.8rem; display: flex; align-items: center; gap: 0.25rem; }.tag-remove { cursor: pointer; font-weight: bold; }.seo-preview { background: #f9fafb; padding: 1rem; border-radius: 6px; border: 1px solid #e5e7eb; }.publish-options { display: flex; flex-direction: column; gap: 0.5rem; }@media (max-width: 1024px) { .editor-container { grid-template-columns: 1fr; } }</style><div class="dashboard-card"><div class="card-title">✏️ Edit Article #` + articleID + `</div><div class="editor-container"><!-- Main Editor --><div class="main-editor"><!-- Title and Slug --><div><label for="title" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Article Title *</label><input type="text" id="title" name="title" required style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 1.1rem; font-weight: 600;"placeholder="Enter your article title..."></div><div><label for="slug" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">URL Slug</label><div style="display: flex; align-items: center; gap: 0.5rem;"><span style="color: #6b7280; font-size: 0.9rem;">https://yoursite.com/</span><input type="text" id="slug" name="slug" style="flex: 1; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-family: monospace;"placeholder="auto-generated-from-title"></div></div><!-- Rich Text Editor --><div><label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Content *</label><!-- Editor Mode Toggle --><div style="display: flex; gap: 0.5rem; margin-bottom: 0.5rem;"><button type="button" id="visualMode" class="toolbar-btn active" onclick="switchEditorMode('visual')">📝 Visual</button><button type="button" id="htmlMode" class="toolbar-btn" onclick="switchEditorMode('html')">🔧 HTML</button><button type="button" class="toolbar-btn" onclick="openMediaLibrary('editor')" style="margin-left: auto;">🖼️ Insert Media</button></div><!-- Visual Editor Toolbar --><div id="editorToolbar" class="editor-toolbar"><button type="button" class="toolbar-btn" onclick="formatText('bold')" title="Bold"><b>B</b></button><button type="button" class="toolbar-btn" onclick="formatText('italic')" title="Italic"><i>I</i></button><button type="button" class="toolbar-btn" onclick="formatText('underline')" title="Underline"><u>U</u></button><button type="button" class="toolbar-btn" onclick="formatText('strikeThrough')" title="Strikethrough"><s>S</s></button><select class="toolbar-btn" onchange="formatText('formatBlock', this.value); this.selectedIndex=0;" title="Headings"><option value="">Headings</option><option value="h1">Heading 1</option><option value="h2">Heading 2</option><option value="h3">Heading 3</option><option value="h4">Heading 4</option><option value="h5">Heading 5</option><option value="h6">Heading 6</option><option value="p">Paragraph</option></select><input type="color" class="toolbar-btn" onchange="changeTextColor(this.value)" title="Text Color" style="width: 40px; padding: 2px;"><button type="button" class="toolbar-btn" onclick="formatText('insertUnorderedList')" title="Bullet List">• List</button><button type="button" class="toolbar-btn" onclick="formatText('insertOrderedList')" title="Numbered List">1. List</button><button type="button" class="toolbar-btn" onclick="insertLink()" title="Insert Link">🔗</button><button type="button" class="toolbar-btn" onclick="formatText('insertHorizontalRule')" title="Horizontal Line">—</button><button type="button" class="toolbar-btn" onclick="formatText('justifyLeft')" title="Align Left">⬅️</button><button type="button" class="toolbar-btn" onclick="formatText('justifyCenter')" title="Align Center">↔️</button><button type="button" class="toolbar-btn" onclick="formatText('justifyRight')" title="Align Right">➡️</button></div><!-- Content Editor --><div id="contentEditor" class="content-editor" contenteditable="true" style="outline: none;" placeholder="Start writing your article..."></div><!-- HTML Editor (hidden by default) --><textarea id="htmlEditor" class="content-editor" style="display: none; font-family: monospace;" placeholder="<p>Enter your HTML content here...</p>"></textarea></div><!-- Excerpt --><div><label for="excerpt" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Excerpt</label><textarea id="excerpt" name="excerpt" rows="3" style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; resize: vertical;"placeholder="Brief summary of the article (optional - will be auto-generated if empty)"></textarea></div></div><!-- Sidebar --><div class="sidebar"><!-- Publish Options --><div class="dashboard-card" style="margin: 0;"><div class="card-title">📅 Update Options</div><div class="publish-options"><div><label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Status</label><select id="status" name="status" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;"><option value="draft">💾 Draft</option><option value="published">🌐 Published</option><option value="scheduled">⏰ Scheduled</option><option value="archived">📦 Archived</option></select></div><div style="display: flex; gap: 0.5rem; margin-top: 1rem;"><button type="button" id="updateBtn" class="action-button" onclick="updateArticle()" style="flex: 1;">💾 Update Article</button></div><div style="display: flex; gap: 0.5rem; margin-top: 0.5rem;"><button type="button" class="action-button" onclick="archiveArticle()" style="background-color: #f59e0b;">📦 Archive</button><button type="button" class="action-button" onclick="deleteArticle()" style="background-color: #ef4444;">🗑️ Delete</button></div><div style="margin-top: 0.5rem;"><button type="button" class="action-button" onclick="window.location.href='/admin/content/articles'" style="background-color: #6b7280; width: 100%;">← Back to Articles</button></div></div></div><!-- Author --><div class="dashboard-card" style="margin: 0;"><div class="card-title">👤 Author</div><input type="text" id="author_name" name="author_name" readonly style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; background-color: #f9fafb;"placeholder="Loading author..."></div><!-- Categories --><div class="dashboard-card" style="margin: 0;"><div class="card-title">📂 Categories</div><div style="margin-bottom: 0.5rem;"><select id="categorySelect" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;"><option value="">Select a category to add...</option></select></div><div id="selectedCategories" style="display: flex; flex-wrap: wrap; gap: 0.5rem; min-height: 2rem; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; background: #f9fafb;"><span style="color: #6b7280; font-size: 0.9rem;">No categories selected</span></div><div style="margin-top: 0.5rem; font-size: 0.8rem; color: #6b7280;">💡 Select multiple categories for this article</div></div><!-- Tags --><div class="dashboard-card" style="margin: 0;"><div class="card-title">🏷️ Tags</div><div id="tagContainer" class="tag-input" onclick="focusTagInput()"><input type="text" id="tagInput" placeholder="Add tags..." style="border: none; outline: none; flex: 1; min-width: 100px;"onkeydown="handleTagInput(event)" oninput="showTagSuggestions(this.value)"></div><div id="tagSuggestions" style="margin-top: 0.5rem; display: none;"><div style="font-size: 0.8rem; color: #6b7280; margin-bottom: 0.25rem;">Suggestions:</div><div id="suggestedTags" style="display: flex; flex-wrap: wrap; gap: 0.25rem;"></div></div></div><!-- Featured Image --><div class="dashboard-card" style="margin: 0;"><div class="card-title">🖼️ Featured Image</div><div id="featuredImagePreview" class="featured-image-preview" onclick="openMediaLibrary()"><div style="text-align: center; color: #6b7280;"><div style="font-size: 2rem; margin-bottom: 0.5rem;">📷</div><div>Click to select featured image</div></div></div><input type="hidden" id="featuredImageId" name="featured_image_id"><button type="button" onclick="removeFeaturedImage()" id="removeFeaturedBtn" style="width: 100%; margin-top: 0.5rem; padding: 0.5rem; background: #ef4444; color: white; border: none; border-radius: 6px; display: none;">Remove Image</button></div><!-- SEO Settings --><div class="dashboard-card" style="margin: 0;"><div class="card-title">🔍 SEO Settings</div><div style="display: flex; flex-direction: column; gap: 0.75rem;"><div><label for="metaTitle" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Meta Title</label><input type="text" id="metaTitle" name="meta_title" maxlength="60"style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem;"placeholder="SEO title (auto-generated if empty)"><div style="font-size: 0.7rem; color: #6b7280; text-align: right;"><span id="metaTitleCount">0</span>/60</div></div><div><label for="metaDescription" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Meta Description</label><textarea id="metaDescription" name="meta_description" rows="3" maxlength="160"style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem; resize: vertical;"placeholder="Brief description for search engines"></textarea><div style="font-size: 0.7rem; color: #6b7280; text-align: right;"><span id="metaDescCount">0</span>/160</div></div><div><label for="focusKeyword" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Focus Keyword</label><input type="text" id="focusKeyword" name="focus_keyword"style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem;"placeholder="Main keyword for SEO"></div><div><label style="display: flex; align-items: center; gap: 0.5rem; font-weight: 600; font-size: 0.9rem; cursor: pointer;"><input type="checkbox" id="autoLinking" name="auto_linking" checked style="width: 16px; height: 16px;">🔗 Enable Auto-linking</label><div style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;">Automatically create links to related articles and tags</div></div><!-- SEO Preview --><div class="seo-preview"><div style="font-weight: 600; margin-bottom: 0.5rem; font-size: 0.9rem;">Search Preview:</div><div id="seoPreviewTitle" style="color: #1a0dab; font-size: 1.1rem; margin-bottom: 0.25rem;">Your Article Title</div><div id="seoPreviewUrl" style="color: #006621; font-size: 0.9rem; margin-bottom: 0.25rem;">https://yoursite.com/your-article-slug</div><div id="seoPreviewDesc" style="color: #545454; font-size: 0.9rem; line-height: 1.4;">Your meta description will appear here...</div></div></div></div></div></div></div><!-- Media Library Modal --><div id="mediaLibraryModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 2000; overflow-y: auto; padding: 1rem;"><div style="position: relative; margin: 1rem auto; background: white; border-radius: 12px; width: 100%; max-width: 1000px; max-height: calc(100vh - 2rem); overflow: hidden; box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1); display: flex; flex-direction: column;"><!-- Modal Header --><div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;"><h3 style="margin: 0; font-size: 1.25rem; font-weight: 600;">📚 Media Library</h3><div style="display: flex; gap: 1rem; align-items: center; flex-wrap: wrap;"><div style="display: flex; gap: 0.5rem;"><button onclick="setMediaViewSize('small')" id="mediaViewSmall" class="toolbar-btn" style="padding: 0.5rem; font-size: 0.8rem;">Small</button><button onclick="setMediaViewSize('medium')" id="mediaViewMedium" class="toolbar-btn active" style="padding: 0.5rem; font-size: 0.8rem;">Medium</button><button onclick="setMediaViewSize('large')" id="mediaViewLarge" class="toolbar-btn" style="padding: 0.5rem; font-size: 0.8rem;">Large</button></div><button onclick="closeMediaLibrary()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer; color: #6b7280;">&times;</button></div></div><!-- Upload Area --><div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; background: #f9fafb;"><div style="display: flex; gap: 1rem; align-items: center;"><input type="file" id="mediaUploadInput" multiple accept="image/*" style="display: none;" onchange="handleMediaUpload(event)"><button onclick="document.getElementById('mediaUploadInput').click()" class="action-button">📁 Upload Images</button><div id="uploadProgress" style="display: none; flex: 1;"><div style="background: #e5e7eb; border-radius: 4px; overflow: hidden;"><div id="uploadProgressBar" style="background: #3b82f6; height: 8px; width: 0%; transition: width 0.3s;"></div></div><div id="uploadStatus" style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;"></div></div></div></div><!-- Media Grid --><div style="padding: 1rem; flex: 1; overflow-y: auto; min-height: 300px; max-height: 60vh;"><div id="mediaLibraryGrid" style="display: grid; gap: 1rem;"><div style="text-align: center; padding: 2rem; color: #6b7280;"><div style="font-size: 3rem; margin-bottom: 1rem;">📷</div><div>Loading media library...</div></div></div></div><!-- Modal Footer --><div style="padding: 1rem; border-top: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;"><div id="selectedMediaInfo" style="color: #6b7280; font-size: 0.9rem;">Select an image to continue</div><div style="display: flex; gap: 0.5rem;"><button onclick="closeMediaLibrary()" class="action-button" style="background-color: #6b7280;">Cancel</button><button onclick="insertSelectedMedia()" id="insertMediaBtn" class="action-button" disabled style="opacity: 0.5;">Insert Image</button></div></div></div></div>

        <script>
let currentEditorMode = 'visual';
let selectedTags = [];
let availableTags = [];
let categories = [];
let selectedCategories = [];
let featuredImageData = null;
let selectedMediaItem = null;
let mediaLibraryData = [];
const articleId = '` + articleID + `';

document.addEventListener("DOMContentLoaded", function() {
    initializeEditor();
    loadCategories();
    loadTags();
    setupEventListeners();
    loadArticleData();
});

// Load existing article data and populate form
async function loadArticleData() {
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/articles/' + articleId, {
            headers: { 'Authorization': 'Bearer ' + token }
        });

        if (response.ok) {
            const result = await response.json();
            const article = result.data || result;
            
            // Store article data globally for other functions to access
            window.currentArticle = article;
            
            console.log('Loaded article data:', article);
            console.log('Article tags:', article.tags);
            console.log('Article tags type:', typeof article.tags);
            console.log('Article tags length:', article.tags ? article.tags.length : 'undefined');
            console.log('Article featured_image_id:', article.featured_image_id);
            console.log('Article featured_image_url:', article.featured_image);
            
            // Populate all form fields with existing data
            document.getElementById('title').value = article.title || '';
            document.getElementById('slug').value = article.slug || '';
            document.getElementById('excerpt').value = article.excerpt || '';
            document.getElementById('status').value = article.status || 'draft';
            // Load article categories
            await loadArticleCategories();
            document.getElementById('metaTitle').value = article.meta_title || '';
            document.getElementById('metaDescription').value = article.meta_description || '';
            document.getElementById('focusKeyword').value = article.focus_keyword || '';
            document.getElementById('autoLinking').checked = article.auto_linking !== false;
            
            // Set content in both editors
            document.getElementById('contentEditor').innerHTML = article.content || '';
            document.getElementById('htmlEditor').value = article.content || '';
            
            // Load tags separately since the article API doesn't return them
            await loadArticleTags();
            
            // Set featured image
            if (article.featured_image) {
                document.getElementById('featuredImagePreview').innerHTML = 
                    '<img src="' + article.featured_image + '" alt="Featured Image" style="width: 100%; height: 100%; object-fit: cover; border-radius: 6px;">';
                document.getElementById('featuredImageId').value = String(article.featured_image_id || '');
                document.getElementById('removeFeaturedBtn').style.display = 'block';
            }
            
            // Update character counters and SEO preview
            document.getElementById('metaTitleCount').textContent = (article.meta_title || '').length;
            document.getElementById('metaDescCount').textContent = (article.meta_description || '').length;
            updateSEOPreview();
        }
    } catch (error) {
        console.error('Error loading article:', error);
        alert('Error loading article: ' + error.message);
    }
}

// Load article tags from the article data
async function loadArticleTags() {
    try {
        // Get the article data from the global variable set in loadArticleData
        const article = window.currentArticle;
        if (article && article.tags && Array.isArray(article.tags)) {
            selectedTags = article.tags.map(tag => tag.name);
            renderTags();
            console.log('Article tags loaded:', selectedTags);
        } else {
            selectedTags = [];
            renderTags();
            console.log('No tags found for this article');
        }
    } catch (error) {
        console.error('Error loading article tags:', error);
        selectedTags = [];
        renderTags();
    }
}

// Load article categories from the article data
async function loadArticleCategories() {
    try {
        const article = window.currentArticle;
        if (article) {
            // Clear existing categories
            selectedCategories = [];
            
            // If article has categories (from junction table), load them
            if (article.categories && Array.isArray(article.categories)) {
                article.categories.forEach(category => {
                    selectedCategories.push({ 
                        id: String(category.id), 
                        name: category.name 
                    });
                });
            } else if (article.category_id && article.category_name) {
                // Fallback to single category for backward compatibility
                selectedCategories.push({ 
                    id: String(article.category_id), 
                    name: article.category_name 
                });
            }
            
            updateCategoryDisplay();
            console.log('Article categories loaded:', selectedCategories);
        }
    } catch (error) {
        console.error('Error loading article categories:', error);
        selectedCategories = [];
        updateCategoryDisplay();
    }
}

// Essential functions for the enhanced edit page
function initializeEditor() {
    // Auto-generate slug from title
    document.getElementById('title').addEventListener('input', function() {
        const slug = this.value.toLowerCase().replace(/[^a-z0-9\s-]/g, '').replace(/\s+/g, '-').replace(/-+/g, '-').trim('-');
        document.getElementById('slug').value = slug;
        updateSEOPreview();
    });
    
    // Character counters
    document.getElementById('metaTitle').addEventListener('input', function() {
        document.getElementById('metaTitleCount').textContent = this.value.length;
        updateSEOPreview();
    });
    
    document.getElementById('metaDescription').addEventListener('input', function() {
        document.getElementById('metaDescCount').textContent = this.value.length;
        updateSEOPreview();
    });
}

async function loadCategories() {
    try {
        const response = await fetch('/api/v1/admin/content/categories', {
            headers: { 'Authorization': 'Bearer ' + localStorage.getItem('auth_token') }
        });
        if (response.ok) {
            const data = await response.json();
            categories = data.data.categories || [];
            const select = document.getElementById('categorySelect');
            select.innerHTML = '<option value="">Select a category to add...</option>';
            categories.forEach(category => {
                const option = document.createElement('option');
                option.value = category.id;
                option.textContent = category.name;
                select.appendChild(option);
            });
            
            // Handle category selection
            select.addEventListener('change', function() {
                if (this.value) {
                    addCategory(this.value, this.options[this.selectedIndex].text);
                    this.selectedIndex = 0; // Reset selection
                }
            });
        }
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

async function loadTags() {
    try {
        const response = await fetch('/api/v1/admin/content/tags', {
            headers: { 'Authorization': 'Bearer ' + localStorage.getItem('auth_token') }
        });
        if (response.ok) {
            const data = await response.json();
            availableTags = data.data.tags || [];
            console.log('Available tags loaded:', availableTags);
            console.log('First tag structure:', availableTags[0]);
        }
    } catch (error) {
        console.error('Failed to load tags:', error);
    }
}

function setupEventListeners() {
    // Content editor sync
    document.getElementById('contentEditor').addEventListener('input', function() {
        if (currentEditorMode === 'visual') {
            document.getElementById('htmlEditor').value = this.innerHTML;
        }
    });
}

function switchEditorMode(mode) {
    currentEditorMode = mode;
    const visualMode = document.getElementById('visualMode');
    const htmlMode = document.getElementById('htmlMode');
    const contentEditor = document.getElementById('contentEditor');
    const htmlEditor = document.getElementById('htmlEditor');
    const toolbar = document.getElementById('editorToolbar');

    if (mode === 'visual') {
        visualMode.classList.add('active');
        htmlMode.classList.remove('active');
        contentEditor.style.display = 'block';
        htmlEditor.style.display = 'none';
        toolbar.style.display = 'flex';
        contentEditor.innerHTML = htmlEditor.value;
    } else {
        htmlMode.classList.add('active');
        visualMode.classList.remove('active');
        contentEditor.style.display = 'none';
        htmlEditor.style.display = 'block';
        toolbar.style.display = 'none';
        htmlEditor.value = contentEditor.innerHTML;
    }
}

function formatText(command, value = null) {
    document.execCommand(command, false, value);
    document.getElementById('contentEditor').focus();
}

function insertLink() {
    const url = prompt('Enter the URL:', 'https://');
    if (url && url !== 'https://') {
        document.execCommand('createLink', false, url);
        document.getElementById('contentEditor').focus();
    }
}

function changeTextColor(color) {
    document.execCommand('foreColor', false, color);
    document.getElementById('contentEditor').focus();
}

// Tag management functions with suggestions
function focusTagInput() {
    document.getElementById('tagInput').focus();
}

function handleTagInput(event) {
    const input = event.target;
    const value = input.value.trim();
    if (event.key === 'Enter' || event.key === ',') {
        event.preventDefault();
        if (value && !selectedTags.includes(value)) {
            addTag(value);
            input.value = '';
            hideTagSuggestions();
        }
    } else if (event.key === 'Backspace' && input.value === '' && selectedTags.length > 0) {
        removeTag(selectedTags[selectedTags.length - 1]);
    }
}

function showTagSuggestions(query) {
    if (!query || query.length < 1) {
        hideTagSuggestions();
        return;
    }
    
    const suggestions = availableTags.filter(tag => {
        const tagName = tag.name || tag.Name || tag;
        return tagName.toLowerCase().includes(query.toLowerCase()) && !selectedTags.includes(tagName);
    }).slice(0, 5);
    
    if (suggestions.length > 0) {
        const suggestionsContainer = document.getElementById('suggestedTags');
        suggestionsContainer.innerHTML = suggestions.map(tag => {
            const tagName = tag.name || tag.Name || tag;
            return '<span onclick="addTagFromSuggestion(\'' + tagName + '\')" style="background: #e5e7eb; color: #374151; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.8rem; cursor: pointer; border: 1px solid #d1d5db;">' + tagName + '</span>';
        }).join('');
        document.getElementById('tagSuggestions').style.display = 'block';
    } else {
        hideTagSuggestions();
    }
}

function addTagFromSuggestion(tagName) {
    addTag(tagName);
    document.getElementById('tagInput').value = '';
    hideTagSuggestions();
}

function hideTagSuggestions() {
    document.getElementById('tagSuggestions').style.display = 'none';
}

function addTag(tagName) {
    if (!selectedTags.includes(tagName)) {
        selectedTags.push(tagName);
        renderTags();
    }
}

function removeTag(tagName) {
    selectedTags = selectedTags.filter(tag => tag !== tagName);
    renderTags();
}

function renderTags() {
    const container = document.getElementById('tagContainer');
    const input = document.getElementById('tagInput');
    const existingTags = container.querySelectorAll('.tag-item');
    existingTags.forEach(tag => tag.remove());
    selectedTags.forEach(tagName => {
        const tagElement = document.createElement('div');
        tagElement.className = 'tag-item';
        tagElement.innerHTML = tagName + ' <span class="tag-remove" onclick="removeTag(\'' + tagName + '\')">&times;</span>';
        container.insertBefore(tagElement, input);
    });
}

// Category management functions
function addCategory(categoryId, categoryName) {
    // Check if category is already selected
    if (selectedCategories.find(cat => cat.id === categoryId)) {
        return;
    }

    // Add to selected categories
    selectedCategories.push({ id: categoryId, name: categoryName });
    updateCategoryDisplay();
}

function removeCategory(categoryId) {
    selectedCategories = selectedCategories.filter(cat => cat.id !== categoryId);
    updateCategoryDisplay();
}

function updateCategoryDisplay() {
    const container = document.getElementById('selectedCategories');
    
    if (selectedCategories.length === 0) {
        container.innerHTML = '<span style="color: #6b7280; font-size: 0.9rem;">No categories selected</span>';
        return;
    }

    let html = '';
    selectedCategories.forEach(category => {
        html += '<span style="background: #3b82f6; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.8rem; display: flex; align-items: center; gap: 0.25rem;">' +
            category.name +
            '<span onclick="removeCategory(\'' + category.id + '\')" style="cursor: pointer; font-weight: bold;">×</span>' +
            '</span>';
    });
    
    container.innerHTML = html;
}

function getSelectedCategoryIds() {
    return selectedCategories.map(cat => parseInt(cat.id));
}

function removeFeaturedImage() {
    document.getElementById('featuredImageId').value = '';
    document.getElementById('featuredImagePreview').innerHTML = 
        '<div style="text-align: center; color: #6b7280;"><div style="font-size: 2rem; margin-bottom: 0.5rem;">📷</div><div>Click to select featured image</div></div>';
    document.getElementById('removeFeaturedBtn').style.display = 'none';
}

function updateSEOPreview() {
    const title = document.getElementById('title').value || 'Your Article Title';
    const slug = document.getElementById('slug').value || 'your-article-slug';
    const metaTitle = document.getElementById('metaTitle').value || title;
    const metaDesc = document.getElementById('metaDescription').value || 'Your meta description will appear here...';
    document.getElementById('seoPreviewTitle').textContent = metaTitle;
    document.getElementById('seoPreviewUrl').textContent = 'https://yoursite.com/' + slug;
    document.getElementById('seoPreviewDesc').textContent = metaDesc;
}

// FIXED UPDATE ARTICLE FUNCTION
async function updateArticle() {
    try {
        const updateBtn = document.getElementById('updateBtn');
        updateBtn.disabled = true;
        updateBtn.textContent = '💾 Updating...';

        // Get current content based on editor mode
        let content;
        if (currentEditorMode === 'visual') {
            content = document.getElementById('contentEditor').innerHTML;
        } else {
            content = document.getElementById('htmlEditor').value;
        }

        // Prepare update data with proper structure for the API
        const updateData = {};
        
        const title = document.getElementById('title').value.trim();
        const slug = document.getElementById('slug').value.trim();
        const excerpt = document.getElementById('excerpt').value.trim();
        const categoryIds = getSelectedCategoryIds();
        const status = document.getElementById('status').value;
        const metaTitle = document.getElementById('metaTitle').value.trim();
        const metaDescription = document.getElementById('metaDescription').value.trim();
        const focusKeyword = document.getElementById('focusKeyword').value.trim();
        const autoLinking = document.getElementById('autoLinking').checked;
        const featuredImageId = document.getElementById('featuredImageId').value;

        // Only include non-empty fields
        if (title) updateData.title = title;
        if (slug) updateData.slug = slug;
        if (content) updateData.content = content;
        if (excerpt) updateData.excerpt = excerpt;
        if (categoryIds.length > 0) updateData.category_ids = categoryIds;
        if (status) updateData.status = status;
        if (selectedTags.length > 0) updateData.tags = selectedTags;
        
        // Include featured_image_id (handle like in create page)
        updateData.featured_image_id = featuredImageId || null;
        
        // Add SEO data as nested object (based on backend handler structure)
        updateData.seo_data = {
            meta_title: metaTitle || "",
            meta_description: metaDescription || "",
            focus_keyword: focusKeyword || ""
        };
        
        updateData.auto_linking = autoLinking;

        console.log('Update data:', updateData);
        console.log('Featured image ID being sent:', featuredImageId);
        console.log('Featured image element value:', document.getElementById('featuredImageId').value);
        console.log('Featured image ID type:', typeof featuredImageId);
        console.log('Featured image ID as string:', String(featuredImageId));

        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/articles/' + articleId, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + token
            },
            body: JSON.stringify(updateData)
        });

        if (response.ok) {
            // Try to update tags separately (workaround for backend limitation)
            if (selectedTags.length > 0) {
                await updateArticleTagsSeparately();
            }
            
            // Reload article data to refresh the UI with updated information
            await loadArticleData();
            
            alert('✅ Article updated successfully!');
        } else {
            const errorData = await response.json();
            console.log('Update failed with status:', response.status);
            console.log('Error response:', errorData);
            throw new Error(errorData.message || 'Failed to update article');
        }
    } catch (error) {
        console.error('Error updating article:', error);
        alert('❌ Error updating article: ' + error.message);
    } finally {
        const updateBtn = document.getElementById('updateBtn');
        updateBtn.disabled = false;
        updateBtn.textContent = '💾 Update Article';
    }
}

async function archiveArticle() {
    if (confirm('Are you sure you want to archive this article?')) {
        try {
            const token = localStorage.getItem('auth_token');
            const response = await fetch('/api/v1/articles/' + articleId, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + token
                },
                body: JSON.stringify({ status: 'archived' })
            });

            if (response.ok) {
                alert('✅ Article archived successfully!');
                window.location.href = '/admin/content/articles';
            } else {
                throw new Error('Failed to archive article');
            }
        } catch (error) {
            alert('❌ Error archiving article: ' + error.message);
        }
    }
}

async function deleteArticle() {
    if (confirm('Are you sure you want to delete this article? This action cannot be undone.')) {
        try {
            const token = localStorage.getItem('auth_token');
            const response = await fetch('/api/v1/articles/' + articleId, {
                method: 'DELETE',
                headers: { 'Authorization': 'Bearer ' + token }
            });

            if (response.ok) {
                alert('✅ Article deleted successfully!');
                window.location.href = '/admin/content/articles';
            } else {
                throw new Error('Failed to delete article');
            }
        } catch (error) {
            alert('❌ Error deleting article: ' + error.message);
        }
    }
}

// Media Library Functions
async function openMediaLibrary(purpose = 'featured') {
    mediaLibraryPurpose = purpose;
    document.getElementById('mediaLibraryModal').style.display = 'block';
    
    // Update modal title and button text based on purpose
    const modalTitle = document.querySelector('#mediaLibraryModal h3');
    const insertBtn = document.getElementById('insertMediaBtn');
    
    if (purpose === 'editor') {
        modalTitle.textContent = 'Insert Image into Article';
        insertBtn.textContent = 'Insert Image';
    } else {
        modalTitle.textContent = 'Select Featured Image';
        insertBtn.textContent = 'Set Featured Image';
    }
    
    await loadMediaLibrary();
}

function closeMediaLibrary() {
    document.getElementById('mediaLibraryModal').style.display = 'none';
    selectedMediaItem = null;
    updateMediaSelection();
}

async function loadMediaLibrary() {
    try {
        const token = localStorage.getItem('auth_token');
        const response = await fetch('/api/v1/admin/content/media', {
            headers: { 'Authorization': 'Bearer ' + token }
        });

        if (response.ok) {
            const data = await response.json();
            mediaLibraryData = data.data.media || [];
            console.log('Media library loaded:', mediaLibraryData);
            console.log('First media item structure:', mediaLibraryData[0]);
            renderMediaGrid();
        } else {
            throw new Error('Failed to load media library');
        }
    } catch (error) {
        console.error('Error loading media library:', error);
        document.getElementById('mediaLibraryGrid').innerHTML = 
            '<div style="text-align: center; padding: 2rem; color: #ef4444;"><div style="font-size: 3rem; margin-bottom: 1rem;">⚠️</div><div>Error loading media library</div></div>';
    }
}

function renderMediaGrid() {
    const grid = document.getElementById('mediaLibraryGrid');
    
    if (mediaLibraryData.length === 0) {
        grid.innerHTML = '<div style="text-align: center; padding: 2rem; color: #6b7280;"><div style="font-size: 3rem; margin-bottom: 1rem;">📷</div><div>No media files found. Upload some images to get started.</div></div>';
        return;
    }

    const viewSize = getMediaViewSize();
    let gridCols = 'repeat(auto-fill, minmax(150px, 1fr))';
    if (viewSize === 'small') gridCols = 'repeat(auto-fill, minmax(100px, 1fr))';
    if (viewSize === 'large') gridCols = 'repeat(auto-fill, minmax(200px, 1fr))';
    
    grid.style.gridTemplateColumns = gridCols;
    
    grid.innerHTML = mediaLibraryData.map(media => {
        const itemId = String(media.id);
        return '<div class="media-item" onclick="selectMediaItem(\'' + itemId + '\')" style="cursor: pointer; border: 2px solid transparent; border-radius: 8px; overflow: hidden; transition: all 0.2s;" data-media-id="' + itemId + '">' +
            '<img src="' + media.original_url + '" alt="' + media.filename + '" style="width: 100%; height: 120px; object-fit: cover;">' +
            '<div style="padding: 0.5rem; background: white; border-top: 1px solid #e5e7eb;">' +
                '<div style="font-size: 0.8rem; font-weight: 600; margin-bottom: 0.25rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + media.filename + '</div>' +
                '<div style="font-size: 0.7rem; color: #6b7280;">' + formatFileSize(media.file_size) + '</div>' +
            '</div>' +
        '</div>';
    }).join('');
}

function selectMediaItem(mediaId) {
    console.log('Selecting media item with ID:', mediaId);
    // Media IDs are strings, so convert for comparison
    selectedMediaItem = mediaLibraryData.find(m => String(m.id) === String(mediaId));
    console.log('Selected media item:', selectedMediaItem);
    updateMediaSelection();
}

function updateMediaSelection() {
    // Update visual selection
    document.querySelectorAll('.media-item').forEach(item => {
        item.style.borderColor = 'transparent';
    });
    
    if (selectedMediaItem) {
        const selectedElement = document.querySelector('[data-media-id="' + String(selectedMediaItem.id) + '"]');
        if (selectedElement) {
            selectedElement.style.borderColor = '#3b82f6';
        }
        
        document.getElementById('selectedMediaInfo').textContent = 
            'Selected: ' + selectedMediaItem.filename + ' (' + formatFileSize(selectedMediaItem.file_size) + ')';
        document.getElementById('insertMediaBtn').disabled = false;
        document.getElementById('insertMediaBtn').style.opacity = '1';
    } else {
        document.getElementById('selectedMediaInfo').textContent = 'Select an image to continue';
        document.getElementById('insertMediaBtn').disabled = true;
        document.getElementById('insertMediaBtn').style.opacity = '0.5';
    }
}

// Global variable to track media library purpose
let mediaLibraryPurpose = 'featured'; // 'featured' or 'editor'

function insertSelectedMedia() {
    if (!selectedMediaItem) return;
    
    if (mediaLibraryPurpose === 'featured') {
        // Set as featured image
        document.getElementById('featuredImagePreview').innerHTML = 
            '<img src="' + selectedMediaItem.original_url + '" alt="Featured Image" style="width: 100%; height: 100%; object-fit: cover; border-radius: 6px;">';
        document.getElementById('featuredImageId').value = String(selectedMediaItem.id);
        document.getElementById('removeFeaturedBtn').style.display = 'block';
    } else if (mediaLibraryPurpose === 'editor') {
        // Insert into text editor
        const visualEditor = document.getElementById('contentEditor');
        const htmlEditor = document.getElementById('htmlEditor');
        const imageHtml = '<img src="' + selectedMediaItem.original_url + '" alt="' + selectedMediaItem.filename + '" style="max-width: 100%; height: auto; margin: 1rem 0;">';
        
        // Check which editor is currently active
        if (visualEditor && visualEditor.style.display !== 'none') {
            // Visual editor is active - insert HTML directly
            if (window.getSelection) {
                const selection = window.getSelection();
                if (selection.rangeCount > 0) {
                    const range = selection.getRangeAt(0);
                    range.deleteContents();
                    const imgElement = document.createElement('div');
                    imgElement.innerHTML = imageHtml;
                    range.insertNode(imgElement.firstChild);
                } else {
                    visualEditor.innerHTML += imageHtml;
                }
            } else {
                visualEditor.innerHTML += imageHtml;
            }
            // Trigger change event
            visualEditor.dispatchEvent(new Event('input'));
        } else if (htmlEditor && htmlEditor.style.display !== 'none') {
            // HTML editor is active - insert at cursor position
            if (htmlEditor.selectionStart || htmlEditor.selectionStart === 0) {
                const startPos = htmlEditor.selectionStart;
                const endPos = htmlEditor.selectionEnd;
                htmlEditor.value = htmlEditor.value.substring(0, startPos) + imageHtml + htmlEditor.value.substring(endPos, htmlEditor.value.length);
                htmlEditor.selectionStart = htmlEditor.selectionEnd = startPos + imageHtml.length;
            } else {
                htmlEditor.value += imageHtml;
            }
            // Trigger change event
            htmlEditor.dispatchEvent(new Event('input'));
        }
    }
    
    closeMediaLibrary();
}

function setMediaViewSize(size) {
    localStorage.setItem('mediaViewSize', size);
    document.querySelectorAll('#mediaViewSmall, #mediaViewMedium, #mediaViewLarge').forEach(btn => {
        btn.classList.remove('active');
    });
    document.getElementById('mediaView' + size.charAt(0).toUpperCase() + size.slice(1)).classList.add('active');
    renderMediaGrid();
}

function getMediaViewSize() {
    return localStorage.getItem('mediaViewSize') || 'medium';
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

async function handleMediaUpload(event) {
    const files = Array.from(event.target.files);
    if (files.length === 0) return;

    const uploadProgress = document.getElementById('uploadProgress');
    const uploadProgressBar = document.getElementById('uploadProgressBar');
    const uploadStatus = document.getElementById('uploadStatus');
    
    uploadProgress.style.display = 'flex';
    
    try {
        for (let i = 0; i < files.length; i++) {
            const file = files[i];
            const formData = new FormData();
            formData.append('image', file); // Changed from 'file' to 'image' to match API
            
            uploadStatus.textContent = 'Uploading ' + (i + 1) + ' of ' + files.length + ': ' + file.name;
            uploadProgressBar.style.width = ((i / files.length) * 100) + '%';
            
            const token = localStorage.getItem('auth_token');
            const response = await fetch('/api/v1/images/upload', {
                method: 'POST',
                headers: { 'Authorization': 'Bearer ' + token },
                body: formData
            });
            
            if (!response.ok) {
                throw new Error('Failed to upload ' + file.name);
            }
        }
        
        uploadProgressBar.style.width = '100%';
        uploadStatus.textContent = 'Upload complete!';
        
        // Reload media library
        await loadMediaLibrary();
        
        setTimeout(() => {
            uploadProgress.style.display = 'none';
            uploadProgressBar.style.width = '0%';
        }, 2000);
        
    } catch (error) {
        console.error('Upload error:', error);
        uploadStatus.textContent = 'Upload failed: ' + error.message;
        uploadStatus.style.color = '#ef4444';
    }
}

// Update article tags separately (workaround for backend limitation)
async function updateArticleTagsSeparately() {
    try {
        console.log('Attempting to update tags separately:', selectedTags);
        // Note: This is a workaround since the main article API doesn't handle tags
        // In a proper implementation, the article update API would handle tags
        // For now, we'll just log this limitation
        console.log('Tag update not implemented in backend - tags need to be handled separately');
    } catch (error) {
        console.error('Error updating tags separately:', error);
    }
}
        </script>`

	s.renderAdminPage(c, title, "content", content)
}