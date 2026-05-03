package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) renderCreateArticle(c *gin.Context) {
	title := "Create Article"
	content := `
        <style>
            .editor-container { display: grid; grid-template-columns: 2fr 1fr; gap: 2rem; }
            .main-editor { display: flex; flex-direction: column; gap: 1rem; }
            .sidebar { display: flex; flex-direction: column; gap: 1rem; }
            .editor-toolbar { display: flex; gap: 0.5rem; padding: 0.5rem; border: 1px solid #d1d5db; border-bottom: none; border-radius: 6px 6px 0 0; background: #f9fafb; flex-wrap: wrap; }
            .toolbar-btn { padding: 0.5rem; border: 1px solid #d1d5db; background: white; border-radius: 4px; cursor: pointer; font-size: 0.9rem; }
            .toolbar-btn:hover { background: #f3f4f6; }
            .toolbar-btn.active { background: #3b82f6; color: white; }
            .content-editor { min-height: 400px; width: 100%; padding: 1rem; border: 1px solid #d1d5db; border-top: none; border-radius: 0 0 6px 6px; font-family: inherit; font-size: 1rem; resize: vertical; box-sizing: border-box; }
            .featured-image-preview { width: 100%; height: 200px; border: 2px dashed #d1d5db; border-radius: 8px; display: flex; align-items: center; justify-content: center; cursor: pointer; background: #f9fafb; }
            .featured-image-preview img { max-width: 100%; max-height: 100%; object-fit: cover; border-radius: 6px; }
            .tag-input { display: flex; flex-wrap: wrap; gap: 0.5rem; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; min-height: 2.5rem; }
            .tag-item { background: #3b82f6; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.8rem; display: flex; align-items: center; gap: 0.25rem; }
            .tag-remove { cursor: pointer; font-weight: bold; }
            .seo-preview { background: #f9fafb; padding: 1rem; border-radius: 6px; border: 1px solid #e5e7eb; }
            .publish-options { display: flex; flex-direction: column; gap: 0.5rem; }
            @media (max-width: 1024px) { .editor-container { grid-template-columns: 1fr; } }
        </style>

        <div class="dashboard-card">
            <div class="card-title">✍️ Create New Article</div>
            
            <div class="editor-container">
                <!-- Main Editor -->
                <div class="main-editor">
                    <!-- Title and Slug -->
                    <div>
                        <label for="title" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Article Title *</label>
                        <input type="text" id="title" name="title" required 
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 1.1rem; font-weight: 600;"
                               placeholder="Enter your article title...">
                    </div>
                    
                    <div>
                        <label for="slug" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">URL Slug</label>
                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                            <span style="color: #6b7280; font-size: 0.9rem;">https://yoursite.com/</span>
                            <input type="text" id="slug" name="slug" 
                                   style="flex: 1; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-family: monospace;"
                                   placeholder="auto-generated-from-title">
                        </div>
                    </div>

                    <!-- Rich Text Editor -->
                    <div>
                        <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Content *</label>
                        
                        <!-- Editor Mode Toggle -->
                        <div style="display: flex; gap: 0.5rem; margin-bottom: 0.5rem;">
                            <button type="button" id="visualMode" class="toolbar-btn active" onclick="switchEditorMode('visual')">📝 Visual</button>
                            <button type="button" id="htmlMode" class="toolbar-btn" onclick="switchEditorMode('html')">🔧 HTML</button>
                            <button type="button" class="toolbar-btn" onclick="insertMedia()" style="margin-left: auto;">🖼️ Insert Media</button>
                        </div>
                        
                        <!-- Visual Editor Toolbar -->
                        <div id="editorToolbar" class="editor-toolbar">
                            <button type="button" class="toolbar-btn" onclick="formatText('bold')" title="Bold"><b>B</b></button>
                            <button type="button" class="toolbar-btn" onclick="formatText('italic')" title="Italic"><i>I</i></button>
                            <button type="button" class="toolbar-btn" onclick="formatText('underline')" title="Underline"><u>U</u></button>
                            <button type="button" class="toolbar-btn" onclick="formatText('strikeThrough')" title="Strikethrough"><s>S</s></button>
                            <select class="toolbar-btn" onchange="formatText('formatBlock', this.value); this.selectedIndex=0;" title="Headings">
                                <option value="">Headings</option>
                                <option value="h1">Heading 1</option>
                                <option value="h2">Heading 2</option>
                                <option value="h3">Heading 3</option>
                                <option value="h4">Heading 4</option>
                                <option value="h5">Heading 5</option>
                                <option value="h6">Heading 6</option>
                                <option value="p">Paragraph</option>
                            </select>
                            <input type="color" class="toolbar-btn" onchange="changeTextColor(this.value)" title="Text Color" style="width: 40px; padding: 2px;">
                            <button type="button" class="toolbar-btn" onclick="formatText('insertUnorderedList')" title="Bullet List">• List</button>
                            <button type="button" class="toolbar-btn" onclick="formatText('insertOrderedList')" title="Numbered List">1. List</button>
                            <button type="button" class="toolbar-btn" onclick="insertLink()" title="Insert Link">🔗</button>
                            <button type="button" class="toolbar-btn" onclick="formatText('insertHorizontalRule')" title="Horizontal Line">—</button>
                            <button type="button" class="toolbar-btn" onclick="formatText('justifyLeft')" title="Align Left">⬅️</button>
                            <button type="button" class="toolbar-btn" onclick="formatText('justifyCenter')" title="Align Center">↔️</button>
                            <button type="button" class="toolbar-btn" onclick="formatText('justifyRight')" title="Align Right">➡️</button>
                        </div>
                        
                        <!-- Content Editor -->
                        <div id="contentEditor" class="content-editor" contenteditable="true" 
                             style="outline: none;" 
                             placeholder="Start writing your article..."></div>
                        
                        <!-- HTML Editor (hidden by default) -->
                        <textarea id="htmlEditor" class="content-editor" style="display: none; font-family: monospace;" 
                                  placeholder="<p>Enter your HTML content here...</p>"></textarea>
                    </div>

                    <!-- Excerpt -->
                    <div>
                        <label for="excerpt" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Excerpt</label>
                        <textarea id="excerpt" name="excerpt" rows="3" 
                                  style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; resize: vertical;"
                                  placeholder="Brief summary of the article (optional - will be auto-generated if empty)"></textarea>
                    </div>
                </div>

                <!-- Sidebar -->
                <div class="sidebar">
                    <!-- Publish Options -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">📅 Publish Options</div>
                        <div class="publish-options">
                            <div>
                                <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Status</label>
                                <select id="status" name="status" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                                    <option value="draft">💾 Draft</option>
                                    <option value="published">🌐 Published</option>
                                    <option value="scheduled">⏰ Scheduled</option>
                                </select>
                            </div>
                            
                            <div id="scheduleOptions" style="display: none;">
                                <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Publish Date & Time</label>
                                <input type="datetime-local" id="publishDate" name="publish_date" 
                                       style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                            </div>
                            
                            <div style="display: flex; gap: 0.5rem; margin-top: 1rem;">
                                <button type="button" id="publishBtn" class="action-button" onclick="publishArticle()" style="flex: 1;">📝 Publish</button>
                                <button type="button" class="action-button" onclick="saveDraft()" style="background-color: #6b7280;">💾 Save</button>
                            </div>
                            <div style="display: flex; gap: 0.5rem; margin-top: 0.5rem;">
                                <button type="button" class="action-button" onclick="scheduleArticle()" style="background-color: #f59e0b; flex: 1;">⏰ Schedule</button>
                            </div>
                        </div>
                    </div>

                    <!-- Language Selection -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">🌐 Language</div>
                        <div>
                            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Article Language</label>
                            <select id="languageCode" name="language_code" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                                <option value="en">🇬🇧 English</option>
                                <option value="de">🇩🇪 Deutsch</option>
                                <option value="fr">🇫🇷 Français</option>
                                <option value="es">🇪🇸 Español</option>
                                <option value="ar">🇸🇦 العربية</option>
                            </select>
                        </div>
                        <div style="margin-top: 0.75rem;">
                            <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Translation Group</label>
                            <select id="translationGroupId" name="translation_group_id" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                                <option value="">New article (no translation group)</option>
                            </select>
                            <div style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;">
                                💡 Select an existing article to create a translation
                            </div>
                        </div>
                    </div>

                    <!-- Author -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">👤 Author</div>
                        <input type="text" id="author_name" name="author_name" 
                               style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Author name">
                    </div>

                    <!-- Categories -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">📂 Categories</div>
                        <div style="margin-bottom: 0.5rem;">
                            <select id="categorySelect" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                                <option value="">Select a category to add...</option>
                            </select>
                        </div>
                        <div id="selectedCategories" style="display: flex; flex-wrap: wrap; gap: 0.5rem; min-height: 2rem; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; background: #f9fafb;">
                            <span style="color: #6b7280; font-size: 0.9rem;">No categories selected</span>
                        </div>
                        <div style="margin-top: 0.5rem; font-size: 0.8rem; color: #6b7280;">
                            💡 Select multiple categories for this article
                        </div>
                    </div>

                    <!-- Tags -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">🏷️ Tags</div>
                        <div id="tagContainer" class="tag-input" onclick="focusTagInput()">
                            <input type="text" id="tagInput" placeholder="Add tags..." 
                                   style="border: none; outline: none; flex: 1; min-width: 100px;"
                                   onkeydown="handleTagInput(event)">
                        </div>
                        <div id="tagSuggestions" style="margin-top: 0.5rem; display: none;">
                            <div style="font-size: 0.8rem; color: #6b7280; margin-bottom: 0.25rem;">Suggestions:</div>
                            <div id="suggestedTags" style="display: flex; flex-wrap: wrap; gap: 0.25rem;"></div>
                        </div>
                    </div>

                    <!-- Featured Image -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">🖼️ Featured Image</div>
                        <div id="featuredImagePreview" class="featured-image-preview" onclick="selectFeaturedImage()">
                            <div style="text-align: center; color: #6b7280;">
                                <div style="font-size: 2rem; margin-bottom: 0.5rem;">📷</div>
                                <div>Click to select featured image</div>
                            </div>
                        </div>
                        <input type="hidden" id="featuredImageId" name="featured_image_id">
                        <button type="button" onclick="removeFeaturedImage()" id="removeFeaturedBtn" 
                                style="width: 100%; margin-top: 0.5rem; padding: 0.5rem; background: #ef4444; color: white; border: none; border-radius: 6px; display: none;">
                            Remove Image
                        </button>
                    </div>

                    <!-- SEO Settings -->
                    <div class="dashboard-card" style="margin: 0;">
                        <div class="card-title">🔍 SEO Settings</div>
                        <div style="display: flex; flex-direction: column; gap: 0.75rem;">
                            <div>
                                <label for="metaTitle" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Meta Title</label>
                                <input type="text" id="metaTitle" name="meta_title" maxlength="60"
                                       style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem;"
                                       placeholder="SEO title (auto-generated if empty)">
                                <div style="font-size: 0.7rem; color: #6b7280; text-align: right;"><span id="metaTitleCount">0</span>/60</div>
                            </div>
                            
                            <div>
                                <label for="metaDescription" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Meta Description</label>
                                <textarea id="metaDescription" name="meta_description" rows="3" maxlength="160"
                                          style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem; resize: vertical;"
                                          placeholder="Brief description for search engines"></textarea>
                                <div style="font-size: 0.7rem; color: #6b7280; text-align: right;"><span id="metaDescCount">0</span>/160</div>
                            </div>
                            
                            <div>
                                <label for="focusKeyword" style="display: block; margin-bottom: 0.25rem; font-weight: 600; font-size: 0.9rem;">Focus Keyword</label>
                                <input type="text" id="focusKeyword" name="focus_keyword"
                                       style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.9rem;"
                                       placeholder="Main keyword for SEO">
                            </div>
                            
                            <div>
                                <label style="display: flex; align-items: center; gap: 0.5rem; font-weight: 600; font-size: 0.9rem; cursor: pointer;">
                                    <input type="checkbox" id="autoLinking" name="auto_linking" checked 
                                           style="width: 16px; height: 16px;">
                                    🔗 Enable Auto-linking
                                </label>
                                <div style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;">
                                    Automatically create links to related articles and tags
                                </div>
                            </div>
                            
                            <!-- SEO Preview -->
                            <div class="seo-preview">
                                <div style="font-weight: 600; margin-bottom: 0.5rem; font-size: 0.9rem;">Search Preview:</div>
                                <div id="seoPreviewTitle" style="color: #1a0dab; font-size: 1.1rem; margin-bottom: 0.25rem;">Your Article Title</div>
                                <div id="seoPreviewUrl" style="color: #006621; font-size: 0.9rem; margin-bottom: 0.25rem;">https://yoursite.com/your-article-slug</div>
                                <div id="seoPreviewDesc" style="color: #545454; font-size: 0.9rem; line-height: 1.4;">Your meta description will appear here...</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Media Library Modal -->
        <div id="mediaLibraryModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 2000; overflow-y: auto; padding: 1rem;">
            <div style="position: relative; margin: 1rem auto; background: white; border-radius: 12px; width: 100%; max-width: 1000px; max-height: calc(100vh - 2rem); overflow: hidden; box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1); display: flex; flex-direction: column;">
                <!-- Modal Header -->
                <div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
                    <h3 style="margin: 0; font-size: 1.25rem; font-weight: 600;">📚 Media Library</h3>
                    <div style="display: flex; gap: 1rem; align-items: center; flex-wrap: wrap;">
                        <div style="display: flex; gap: 0.5rem;">
                            <button onclick="setMediaViewSize('small')" id="mediaViewSmall" class="toolbar-btn" style="padding: 0.5rem; font-size: 0.8rem;">Small</button>
                            <button onclick="setMediaViewSize('medium')" id="mediaViewMedium" class="toolbar-btn active" style="padding: 0.5rem; font-size: 0.8rem;">Medium</button>
                            <button onclick="setMediaViewSize('large')" id="mediaViewLarge" class="toolbar-btn" style="padding: 0.5rem; font-size: 0.8rem;">Large</button>
                        </div>
                        <button onclick="closeMediaLibrary()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer; color: #6b7280;">&times;</button>
                    </div>
                </div>
                
                <!-- Upload Area -->
                <div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; background: #f9fafb;">
                    <div style="display: flex; gap: 1rem; align-items: center;">
                        <input type="file" id="mediaUploadInput" multiple accept="image/*" style="display: none;" onchange="handleMediaUpload(event)">
                        <button onclick="document.getElementById('mediaUploadInput').click()" class="action-button">📁 Upload Images</button>
                        <div id="uploadProgress" style="display: none; flex: 1;">
                            <div style="background: #e5e7eb; border-radius: 4px; overflow: hidden;">
                                <div id="uploadProgressBar" style="background: #3b82f6; height: 8px; width: 0%; transition: width 0.3s;"></div>
                            </div>
                            <div id="uploadStatus" style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;"></div>
                        </div>
                    </div>
                </div>
                
                <!-- Media Grid -->
                <div style="padding: 1rem; flex: 1; overflow-y: auto; min-height: 300px; max-height: 60vh;">
                    <div id="mediaLibraryGrid" style="display: grid; gap: 1rem;">
                        <div style="text-align: center; padding: 2rem; color: #6b7280;">
                            <div style="font-size: 3rem; margin-bottom: 1rem;">📷</div>
                            <div>Loading media library...</div>
                        </div>
                    </div>
                </div>
                
                <!-- Modal Footer -->
                <div style="padding: 1rem; border-top: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <div id="selectedMediaInfo" style="color: #6b7280; font-size: 0.9rem;">
                        Select an image to continue
                    </div>
                    <div style="display: flex; gap: 0.5rem;">
                        <button onclick="closeMediaLibrary()" class="action-button" style="background-color: #6b7280;">Cancel</button>
                        <button onclick="insertSelectedMedia()" id="insertMediaBtn" class="action-button" disabled style="opacity: 0.5;">Insert Image</button>
                    </div>
                </div>
            </div>
        </div>
        
        <script>
            let currentEditorMode = 'visual';
            let selectedTags = [];
            let availableTags = [];
            let categories = [];
            let featuredImageData = null;

            // Utility function for formatting file sizes
            function formatFileSize(bytes) {
                if (bytes === 0) return '0 Bytes';
                const k = 1024;
                const sizes = ['Bytes', 'KB', 'MB', 'GB'];
                const i = Math.floor(Math.log(bytes) / Math.log(k));
                return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
            }

            document.addEventListener("DOMContentLoaded", function() {
                initializeEditor();
                loadCategories();
                loadTags();
                setupEventListeners();
                updateSEOPreview();
            });

            function initializeEditor() {
                // Initialize author field with current user
                initializeAuthor();
                
                // Auto-generate slug from title
                document.getElementById('title').addEventListener('input', function() {
                    const title = this.value;
                    const slug = generateSlug(title);
                    document.getElementById('slug').value = slug;
                    updateSEOPreview();
                });

                // Status change handler
                document.getElementById('status').addEventListener('change', function() {
                    const scheduleOptions = document.getElementById('scheduleOptions');
                    const publishBtn = document.getElementById('publishBtn');
                    
                    if (this.value === 'scheduled') {
                        scheduleOptions.style.display = 'block';
                        publishBtn.textContent = '⏰ Schedule';
                    } else if (this.value === 'published') {
                        scheduleOptions.style.display = 'none';
                        publishBtn.textContent = '📝 Publish';
                    } else {
                        scheduleOptions.style.display = 'none';
                        publishBtn.textContent = '💾 Save Draft';
                    }
                });

                // SEO field handlers
                ['metaTitle', 'metaDescription', 'focusKeyword'].forEach(id => {
                    const element = document.getElementById(id);
                    if (element) {
                        element.addEventListener('input', updateSEOPreview);
                    }
                });

                // Character counters
                document.getElementById('metaTitle').addEventListener('input', function() {
                    document.getElementById('metaTitleCount').textContent = this.value.length;
                });
                
                document.getElementById('metaDescription').addEventListener('input', function() {
                    document.getElementById('metaDescCount').textContent = this.value.length;
                });
            }

            function setupEventListeners() {
                // Content editor sync
                document.getElementById('contentEditor').addEventListener('input', function() {
                    if (currentEditorMode === 'visual') {
                        document.getElementById('htmlEditor').value = this.innerHTML;
                    }
                });

                document.getElementById('htmlEditor').addEventListener('input', function() {
                    if (currentEditorMode === 'html') {
                        document.getElementById('contentEditor').innerHTML = this.value;
                    }
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
                    }
                } catch (error) {
                    console.error('Failed to load tags:', error);
                }
            }

            function generateSlug(title) {
                return title.toLowerCase()
                    .replace(/[^a-z0-9\s-]/g, '')
                    .replace(/\s+/g, '-')
                    .replace(/-+/g, '-')
                    .trim('-');
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
                    
                    // Sync content from HTML to visual
                    contentEditor.innerHTML = htmlEditor.value;
                } else {
                    htmlMode.classList.add('active');
                    visualMode.classList.remove('active');
                    contentEditor.style.display = 'none';
                    htmlEditor.style.display = 'block';
                    toolbar.style.display = 'none';
                    
                    // Sync content from visual to HTML
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
                const selection = window.getSelection();
                if (selection.rangeCount > 0 && !selection.isCollapsed) {
                    // Text is selected, apply color to selection
                    document.execCommand('foreColor', false, color);
                } else {
                    // No text selected, set color for future typing
                    document.execCommand('foreColor', false, color);
                }
                document.getElementById('contentEditor').focus();
            }

            async function initializeAuthor() {
                try {
                    const token = localStorage.getItem('auth_token');
                    if (!token) {
                        document.getElementById('author_name').value = 'Current User';
                        return;
                    }
                    
                    // Try different auth endpoints
                    const endpoints = ['/api/v1/auth/me', '/api/v1/user/profile', '/api/v1/auth/profile'];
                    
                    for (const endpoint of endpoints) {
                        try {
                            const response = await fetch(endpoint, {
                                headers: {
                                    'Authorization': 'Bearer ' + token
                                }
                            });
                            
                            if (response.ok) {
                                const result = await response.json();
                                const user = result.data || result;
                                const authorName = (user.first_name || '') + ' ' + (user.last_name || '');
                                document.getElementById('author_name').value = authorName.trim() || user.username || 'Current User';
                                return;
                            }
                        } catch (e) {
                            continue;
                        }
                    }
                    
                    // Fallback if no endpoint works
                    document.getElementById('author_name').value = 'Current User';
                } catch (error) {
                    console.log('Could not load author info:', error);
                    document.getElementById('author_name').value = 'Current User';
                }
            }

            function insertMedia() {
                openMediaLibrary('insert');
            }

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
                        updateTagSuggestions('');
                    }
                } else if (event.key === 'Backspace' && input.value === '' && selectedTags.length > 0) {
                    removeTag(selectedTags[selectedTags.length - 1]);
                } else {
                    // Show suggestions
                    setTimeout(() => updateTagSuggestions(input.value), 100);
                }
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
                
                // Clear existing tags
                const existingTags = container.querySelectorAll('.tag-item');
                existingTags.forEach(tag => tag.remove());
                
                // Add selected tags
                selectedTags.forEach(tagName => {
                    const tagElement = document.createElement('div');
                    tagElement.className = 'tag-item';
                    tagElement.innerHTML = tagName + ' <span class="tag-remove" onclick="removeTag(\'' + tagName + '\')">&times;</span>';
                    container.insertBefore(tagElement, input);
                });
            }

            function updateTagSuggestions(query) {
                const suggestions = document.getElementById('tagSuggestions');
                const suggestedTags = document.getElementById('suggestedTags');
                
                if (!query) {
                    suggestions.style.display = 'none';
                    return;
                }
                
                const matches = availableTags
                    .filter(tag => tag.name.toLowerCase().includes(query.toLowerCase()) && !selectedTags.includes(tag.name))
                    .slice(0, 5);
                
                if (matches.length > 0) {
                    suggestedTags.innerHTML = matches.map(tag => 
                        '<button type="button" onclick="addTag(\'' + tag.name + '\'); document.getElementById(\'tagInput\').value = \'\'; updateTagSuggestions(\'\');" style="padding: 0.25rem 0.5rem; background: #e5e7eb; border: none; border-radius: 4px; cursor: pointer; font-size: 0.8rem;">' + tag.name + '</button>'
                    ).join('');
                    suggestions.style.display = 'block';
                } else {
                    suggestions.style.display = 'none';
                }
            }

            function selectFeaturedImage() {
                openMediaLibrary('featured');
            }

            function removeFeaturedImage() {
                featuredImageData = null;
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

            async function publishArticle() {
                const status = document.getElementById('status').value;
                const title = document.getElementById('title').value.trim();
                const content = currentEditorMode === 'visual' 
                    ? document.getElementById('contentEditor').innerHTML 
                    : document.getElementById('htmlEditor').value;
                const categoryIds = getSelectedCategoryIds();

                // Debug logging
                console.log('Validation check:', {
                    status: status,
                    title: title,
                    content: content ? 'has content' : 'no content',
                    categoryIds: categoryIds,
                    selectedCategories: selectedCategories
                });

                // Different validation for different statuses
                if (status === 'published') {
                    if (!title || !content || categoryIds.length === 0) {
                        let errorMsg = 'Missing required fields:\n';
                        if (!title) errorMsg += '- Title is required\n';
                        if (!content) errorMsg += '- Content is required\n';
                        if (categoryIds.length === 0) errorMsg += '- At least one category must be selected\n';
                        alert(errorMsg);
                        return;
                    }
                } else if (status === 'scheduled') {
                    if (!title || !content || categoryIds.length === 0) {
                        let errorMsg = 'Missing required fields:\n';
                        if (!title) errorMsg += '- Title is required\n';
                        if (!content) errorMsg += '- Content is required\n';
                        if (categoryIds.length === 0) errorMsg += '- At least one category must be selected\n';
                        alert(errorMsg);
                        return;
                    }
                } else if (status === 'draft') {
                    if (!title) {
                        alert('Please enter a title for the draft');
                        return;
                    }
                    // For drafts, set default values if empty
                    if (!content) {
                        content = '<p>Draft content...</p>';
                    }
                    if (categoryIds.length === 0) {
                        // For drafts, we can allow no categories, but let's add a default if available
                        const categorySelect = document.getElementById('categorySelect');
                        if (categorySelect.options.length > 1) {
                            // Add the first available category as default for drafts
                            const firstCategoryId = categorySelect.options[1].value;
                            const firstCategoryName = categorySelect.options[1].text;
                            addCategory(firstCategoryId, firstCategoryName);
                            categoryIds = getSelectedCategoryIds();
                        }
                        // For drafts, we allow empty categories, so don't return error
                    }
                }

                const featuredImageId = document.getElementById('featuredImageId').value;
                const autoLinkingCheckbox = document.getElementById('autoLinking');
                const autoLinkingValue = autoLinkingCheckbox ? autoLinkingCheckbox.checked : true;
                console.log('Auto-linking checkbox element:', autoLinkingCheckbox);
                console.log('Auto-linking checkbox checked:', autoLinkingValue);

                const formData = {
                    title: title,
                    slug: document.getElementById('slug').value || generateSlug(title),
                    content: content,
                    excerpt: document.getElementById('excerpt').value,
                    category_ids: categoryIds,
                    status: status,
                    featured_image_id: featuredImageId || null,
                    auto_linking: autoLinkingValue,
                    tags: selectedTags,
                    language_code: document.getElementById('languageCode').value || 'en',
                    translation_group_id: document.getElementById('translationGroupId').value ? parseInt(document.getElementById('translationGroupId').value) : null,
                    seo_data: {
                        meta_title: document.getElementById('metaTitle').value,
                        meta_description: document.getElementById('metaDescription').value,
                        focus_keyword: document.getElementById('focusKeyword').value
                    }
                };

                if (status === 'scheduled') {
                    const publishDate = document.getElementById('publishDate').value;
                    if (!publishDate) {
                        alert('Please select a publish date for scheduled articles');
                        return;
                    }
                    formData.scheduled_at = publishDate;
                }

                try {
                    const token = localStorage.getItem('auth_token');
                    if (!token) {
                        alert('Please log in to continue');
                        return;
                    }

                    console.log('Sending article data:', formData);

                    const response = await fetch('/api/v1/articles', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify(formData)
                    });

                    console.log('Response status:', response.status);

                    if (response.ok) {
                        const result = await response.json();
                        alert('Article ' + (status === 'published' ? 'published' : status === 'scheduled' ? 'scheduled' : 'saved') + ' successfully!');
                        window.location.href = '/admin/content';
                    } else {
                        const error = await response.json();
                        console.log('Error response:', error);
                        alert('Error: ' + (error.message || error.error || 'Failed to save article'));
                    }
                } catch (error) {
                    console.log('Network error:', error);
                    alert('Network error: ' + error.message);
                }
            }

            function saveDraft() {
                // For drafts, we don't need all fields to be filled
                const title = document.getElementById('title').value.trim();
                if (!title) {
                    alert('Please enter a title for the draft');
                    return;
                }
                document.getElementById('status').value = 'draft';
                publishArticle();
            }

            function scheduleArticle() {
                const publishDate = document.getElementById('publishDate').value;
                if (!publishDate) {
                    alert('Please select a publish date and time');
                    document.getElementById('scheduleOptions').style.display = 'block';
                    return;
                }
                document.getElementById('status').value = 'scheduled';
                publishArticle();
            }

            // Media Library Modal Functions
            let mediaLibraryMode = 'insert'; // 'insert' or 'featured'
            let selectedMediaItem = null;
            let mediaLibraryViewSize = 'medium';

            function openMediaLibrary(mode) {
                mediaLibraryMode = mode;
                selectedMediaItem = null;
                document.getElementById('mediaLibraryModal').style.display = 'block';
                document.getElementById('insertMediaBtn').disabled = true;
                document.getElementById('insertMediaBtn').style.opacity = '0.5';
                
                // Update modal title and button text based on mode
                const modalTitle = document.querySelector('#mediaLibraryModal h3');
                const insertBtn = document.getElementById('insertMediaBtn');
                
                if (mode === 'featured') {
                    modalTitle.textContent = '🖼️ Select Featured Image';
                    insertBtn.textContent = 'Set Featured Image';
                } else {
                    modalTitle.textContent = '📚 Insert Media';
                    insertBtn.textContent = 'Insert Image';
                }
                
                loadMediaLibrary();
            }

            function closeMediaLibrary() {
                document.getElementById('mediaLibraryModal').style.display = 'none';
                selectedMediaItem = null;
            }

            async function loadMediaLibrary() {
                try {
                    const response = await fetch('/api/v1/admin/content/media', {
                        headers: { 'Authorization': 'Bearer ' + localStorage.getItem('auth_token') }
                    });
                    
                    if (response.ok) {
                        const data = await response.json();
                        const media = data.data.media || [];
                        displayMediaLibrary(media);
                    } else {
                        document.getElementById('mediaLibraryGrid').innerHTML = 
                            '<div style="text-align: center; padding: 2rem; color: #ef4444;">Failed to load media library</div>';
                    }
                } catch (error) {
                    document.getElementById('mediaLibraryGrid').innerHTML = 
                        '<div style="text-align: center; padding: 2rem; color: #ef4444;">Network error loading media</div>';
                }
            }

            function displayMediaLibrary(media) {
                const grid = document.getElementById('mediaLibraryGrid');
                
                // Store media data globally for selection
                window.currentMedia = media;
                
                if (media.length === 0) {
                    grid.innerHTML = '<div style="text-align: center; padding: 2rem; color: #6b7280;">No images found. Upload some images to get started!</div>';
                    return;
                }
                
                const sizes = {
                    small: { width: '120px', height: '90px', cols: 'repeat(auto-fill, minmax(120px, 1fr))' },
                    medium: { width: '160px', height: '120px', cols: 'repeat(auto-fill, minmax(160px, 1fr))' },
                    large: { width: '200px', height: '150px', cols: 'repeat(auto-fill, minmax(200px, 1fr))' }
                };
                
                grid.style.gridTemplateColumns = sizes[mediaLibraryViewSize].cols;
                
                let html = '';
                media.forEach(item => {
                    const itemId = String(item.id);
                    
                    // Find best thumbnail
                    let thumbnailUrl = item.original_url;
                    if (item.variants && item.variants.length > 0) {
                        const thumbnail = item.variants.find(v => v.size === 'thumbnail' && v.format === 'webp') ||
                                          item.variants.find(v => v.size === 'small' && v.format === 'webp') ||
                                          item.variants[0];
                        if (thumbnail) thumbnailUrl = thumbnail.url;
                    }
                    
                    html += '<div class="media-library-item" data-media-id="' + itemId + '" style="border: 2px solid transparent; border-radius: 8px; overflow: hidden; cursor: pointer; transition: all 0.2s;" onclick="selectMediaItem(\'' + itemId + '\')">' +
                        '<div style="height: ' + sizes[mediaLibraryViewSize].height + '; background: #f3f4f6; display: flex; align-items: center; justify-content: center; position: relative;">' +
                        '<img src="' + thumbnailUrl + '" alt="' + (item.alt_text || item.filename) + '" style="max-width: 100%; max-height: 100%; object-fit: cover;" onerror="handleImageError(this, \'' + item.original_url + '\')">' +
                        '<div style="position: absolute; top: 0.25rem; right: 0.25rem; background: rgba(0,0,0,0.7); color: white; padding: 0.125rem 0.25rem; border-radius: 3px; font-size: 0.6rem;">' + item.mime_type.split('/')[1].toUpperCase() + '</div>' +
                        '</div>' +
                        '<div style="padding: 0.5rem; background: white;">' +
                        '<p style="margin: 0; font-size: 0.8rem; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + item.filename + '">' + item.filename + '</p>' +
                        '<p style="margin: 0; font-size: 0.7rem; color: #6b7280;">' + formatFileSize(item.file_size) + ' • ' + item.width + 'x' + item.height + '</p>' +
                        '</div>' +
                        '</div>';
                });
                
                grid.innerHTML = html;
            }

            function selectMediaItem(id) {
                // Remove previous selection
                document.querySelectorAll('.media-library-item').forEach(item => {
                    item.style.borderColor = 'transparent';
                });
                
                // Select current item
                const selectedElement = document.querySelector('.media-library-item[data-media-id="' + id + '"]');
                if (selectedElement) {
                    selectedElement.style.borderColor = '#3b82f6';
                    selectedElement.style.backgroundColor = '#eff6ff';
                }
                
                // Find media data
                selectedMediaItem = window.currentMedia ? window.currentMedia.find(item => String(item.id) === String(id)) : null;
                
                // Update UI
                const insertBtn = document.getElementById('insertMediaBtn');
                const infoDiv = document.getElementById('selectedMediaInfo');
                
                if (selectedMediaItem) {
                    insertBtn.disabled = false;
                    insertBtn.style.opacity = '1';
                    infoDiv.textContent = selectedMediaItem.filename + ' (' + formatFileSize(selectedMediaItem.file_size) + ')';
                } else {
                    insertBtn.disabled = true;
                    insertBtn.style.opacity = '0.5';
                    infoDiv.textContent = 'Select an image to continue';
                }
            }

            function insertSelectedMedia() {
                if (!selectedMediaItem) return;
                
                if (mediaLibraryMode === 'featured') {
                    // Set as featured image
                    setFeaturedImage(selectedMediaItem);
                } else {
                    // Insert into content
                    insertImageIntoContent(selectedMediaItem);
                }
                
                closeMediaLibrary();
            }

            function setFeaturedImage(mediaItem) {
                featuredImageData = mediaItem;
                document.getElementById('featuredImageId').value = mediaItem.id;
                
                // Find best thumbnail for preview
                let previewUrl = mediaItem.original_url;
                if (mediaItem.variants && mediaItem.variants.length > 0) {
                    const preview = mediaItem.variants.find(v => v.size === 'medium' && v.format === 'webp') ||
                                    mediaItem.variants.find(v => v.size === 'small' && v.format === 'webp') ||
                                    mediaItem.variants[0];
                    if (preview) previewUrl = preview.url;
                }
                
                document.getElementById('featuredImagePreview').innerHTML = 
                    '<img src="' + previewUrl + '" alt="' + (mediaItem.alt_text || mediaItem.filename) + '" style="width: 100%; height: 100%; object-fit: cover; border-radius: 6px;">';
                
                document.getElementById('removeFeaturedBtn').style.display = 'block';
            }

            function insertImageIntoContent(mediaItem) {
                // Generate responsive image HTML
                let imageHtml = '';
                
                if (mediaItem.variants && mediaItem.variants.length > 0) {
                    // Create responsive image with multiple formats and sizes
                    const webpVariants = mediaItem.variants.filter(v => v.format === 'webp').sort((a, b) => a.width - b.width);
                    const jpegVariants = mediaItem.variants.filter(v => v.format === 'jpeg').sort((a, b) => a.width - b.width);
                    
                    if (webpVariants.length > 0 || jpegVariants.length > 0) {
                        imageHtml = '<picture>';
                        
                        // WebP sources
                        if (webpVariants.length > 0) {
                            const srcset = webpVariants.map(v => v.url + ' ' + v.width + 'w').join(', ');
                            imageHtml += '<source type="image/webp" srcset="' + srcset + '">';
                        }
                        
                        // JPEG fallback
                        if (jpegVariants.length > 0) {
                            const srcset = jpegVariants.map(v => v.url + ' ' + v.width + 'w').join(', ');
                            imageHtml += '<source type="image/jpeg" srcset="' + srcset + '">';
                        }
                        
                        // Fallback img tag
                        const fallbackSrc = jpegVariants.length > 0 ? jpegVariants[jpegVariants.length - 1].url : mediaItem.original_url;
                        imageHtml += '<img src="' + fallbackSrc + '" alt="' + (mediaItem.alt_text || mediaItem.filename) + '" style="max-width: 100%; height: auto; border-radius: 6px; margin: 1rem 0;">';
                        imageHtml += '</picture>';
                    } else {
                        imageHtml = '<img src="' + mediaItem.original_url + '" alt="' + (mediaItem.alt_text || mediaItem.filename) + '" style="max-width: 100%; height: auto; border-radius: 6px; margin: 1rem 0;">';
                    }
                } else {
                    imageHtml = '<img src="' + mediaItem.original_url + '" alt="' + (mediaItem.alt_text || mediaItem.filename) + '" style="max-width: 100%; height: auto; border-radius: 6px; margin: 1rem 0;">';
                }
                
                // Insert into editor
                if (currentEditorMode === 'visual') {
                    const editor = document.getElementById('contentEditor');
                    const selection = window.getSelection();
                    if (selection.rangeCount > 0) {
                        const range = selection.getRangeAt(0);
                        range.deleteContents();
                        const div = document.createElement('div');
                        div.innerHTML = imageHtml;
                        range.insertNode(div.firstChild);
                    } else {
                        editor.innerHTML += imageHtml;
                    }
                    // Sync to HTML editor
                    document.getElementById('htmlEditor').value = editor.innerHTML;
                } else {
                    // Insert into HTML editor
                    const htmlEditor = document.getElementById('htmlEditor');
                    const cursorPos = htmlEditor.selectionStart;
                    const textBefore = htmlEditor.value.substring(0, cursorPos);
                    const textAfter = htmlEditor.value.substring(cursorPos);
                    htmlEditor.value = textBefore + imageHtml + textAfter;
                    // Sync to visual editor
                    document.getElementById('contentEditor').innerHTML = htmlEditor.value;
                }
            }

            async function handleMediaUpload(event) {
                const files = Array.from(event.target.files);
                if (files.length === 0) return;
                
                const progress = document.getElementById('uploadProgress');
                const progressBar = document.getElementById('uploadProgressBar');
                const status = document.getElementById('uploadStatus');
                
                progress.style.display = 'flex';
                
                let completed = 0;
                const total = files.length;
                
                for (const file of files) {
                    try {
                        const formData = new FormData();
                        formData.append('image', file);
                        
                        const response = await fetch('/api/v1/images/upload', {
                            method: 'POST',
                            headers: { 'Authorization': 'Bearer ' + localStorage.getItem('auth_token') },
                            body: formData
                        });
                        
                        if (response.ok) {
                            completed++;
                        }
                    } catch (error) {
                        console.error('Upload failed:', error);
                    }
                    
                    // Update progress
                    const percent = (completed / total) * 100;
                    progressBar.style.width = percent + '%';
                    status.textContent = completed + ' of ' + total + ' files uploaded';
                }
                
                // Hide progress and reload media library
                setTimeout(() => {
                    progress.style.display = 'none';
                    loadMediaLibrary();
                    event.target.value = ''; // Reset file input
                }, 1000);
            }

            function setMediaViewSize(size) {
                mediaLibraryViewSize = size;
                
                // Update button states
                ['small', 'medium', 'large'].forEach(s => {
                    const btn = document.getElementById('mediaView' + s.charAt(0).toUpperCase() + s.slice(1));
                    if (s === size) {
                        btn.classList.add('active');
                    } else {
                        btn.classList.remove('active');
                    }
                });
                
                // Reload with new size
                loadMediaLibrary();
            }

            // Category management functions
            let selectedCategories = [];

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
        </script>`
	s.renderAdminPage(c, title, "content", content)
}

