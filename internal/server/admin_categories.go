package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) renderManageCategories(c *gin.Context) {
	title := "Manage Categories"
	content := `
        <div class="dashboard-card">
            <div class="card-title">📂 Category Management</div>
            <div style="margin-bottom: 2rem;">
                <h3>Add New Category</h3>
                <form id="categoryForm" style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-bottom: 2rem;">
                    <div>
                        <label for="categoryName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Category Name *</label>
                        <input type="text" id="categoryName" name="name" required
                            style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                    </div>
                    <div>
                        <label for="categorySlug" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">URL Slug *</label>
                        <input type="text" id="categorySlug" name="slug" required
                            style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                    </div>
                    <div style="grid-column: 1 / -1;">
                        <label for="categoryDescription" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Description</label>
                        <textarea id="categoryDescription" name="description" rows="3"
                                style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                                placeholder="Brief description of this category"></textarea>
                    </div>
                    <div>
                        <label for="parentCategory" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Parent Category</label>
                        <select id="parentCategory" name="parent_id"
                                style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                            <option value="">None (Top Level)</option>
                        </select>
                    </div>
                    <div>
                        <label for="sortOrder" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Sort Order</label>
                        <input type="number" id="sortOrder" name="sort_order" value="0" min="0"
                            style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                    </div>
                    <div>
                        <label for="languageCode" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Language</label>
                        <select id="languageCode" name="language_code"
                                style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                            <option value="en">English</option>
                            <option value="fa">Persian/Farsi</option>
                            <option value="ar">Arabic</option>
                        </select>
                    </div>
                    <div style="grid-column: 1 / -1;">
                        <label style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Category Image</label>
                        <div style="display: flex; gap: 1rem; align-items: end;">
                            <div style="flex: 1;">
                                <label for="categoryImageUrl" style="display: block; margin-bottom: 0.5rem; font-size: 0.875rem; color: #6b7280;">Selected Image</label>
                                <input type="url" id="categoryImageUrl" name="image_url" readonly
                                    style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; background-color: #f9fafb;"
                                    placeholder="No image selected">
                            </div>
                            <div style="flex: 1;">
                                <label for="categoryImageAlt" style="display: block; margin-bottom: 0.5rem; font-size: 0.875rem; color: #6b7280;">Alt Text</label>
                                <input type="text" id="categoryImageAlt" name="image_alt_text"
                                    style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                                    placeholder="Description for accessibility">
                            </div>
                            <div style="display: flex; gap: 0.5rem;">
                                <button type="button" id="mediaLibraryBtn" class="action-button" style="padding: 0.75rem; background-color: #3b82f6;">
                                    🖼️ Media Library
                                </button>
                                <button type="button" id="clearImageBtn" class="action-button" style="padding: 0.75rem; background-color: #ef4444;">
                                    ❌ Clear
                                </button>
                            </div>
                        </div>
                        <div id="imagePreview" style="display: none; margin-top: 1rem; padding: 1rem; border: 1px solid #d1d5db; border-radius: 6px; background: #f9fafb;">
                            <div style="display: flex; align-items: center; gap: 1rem;">
                                <img id="previewImage" src="" alt="" style="width: 80px; height: 80px; object-fit: cover; border-radius: 6px; border: 1px solid #e5e7eb;">
                                <div>
                                    <div id="previewFilename" style="font-weight: 600; margin-bottom: 0.25rem;"></div>
                                    <div id="previewDimensions" style="font-size: 0.875rem; color: #6b7280;"></div>
                                </div>
                            </div>
                        </div>
                        <div style="margin-top: 0.5rem; font-size: 0.875rem; color: #6b7280;">
                            💡 Tip: Select an image from your media library or leave empty for automatic emoji based on category name
                        </div>
                    </div>
                    <div style="display: flex; align-items: end;">
                        <button type="submit" class="action-button" style="width: 100%;">➕ Add Category</button>
                    </div>
                </form>

            </div>
            
            <!-- CSV Import Section -->
            <div style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; background: #f9fafb;">
                <h3>📥 Import Categories from CSV</h3>
                <p style="color: #6b7280; margin-bottom: 1rem;">Upload a CSV file with columns: name, slug, description, parent_id, sort_order, image_url, image_alt_text</p>
                <div style="display: flex; gap: 1rem; align-items: end;">
                    <div style="flex: 1;">
                        <input type="file" id="csvFile" accept=".csv" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                    </div>
                    <button type="button" onclick="importCSV()" class="action-button">📥 Import CSV</button>
                    <a href="/static/samples/categories-sample.csv" class="action-button" style="background-color: #6b7280;">📄 Sample CSV</a>
                </div>
            </div>
            
            <h3>Existing Categories</h3>
            
            <!-- Bulk Actions -->
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                <div style="display: flex; gap: 1rem; align-items: center;">
                    <label style="display: flex; align-items: center; gap: 0.5rem;">
                        <input type="checkbox" id="selectAllCategories" onchange="toggleSelectAll()">
                        <span>Select All</span>
                    </label>
                    <button type="button" id="bulkDeleteBtn" onclick="bulkDeleteCategories()" 
                            class="action-button" style="background-color: #ef4444; display: none;">
                        🗑️ Delete Selected
                    </button>
                </div>
                <div style="color: #6b7280; font-size: 0.9rem;">
                    <span id="selectedCount">0</span> selected
                </div>
            </div>
            
            <div id="categoriesList">
                <div style="text-align: center; padding: 2rem; color: #6b7280;">Loading categories...</div>
            </div>
            
            <div style="margin-top: 2rem;">
                <a href="/admin/content" class="action-button" style="background-color: #6b7280;">← Back to Content</a>
            </div>
        </div>

        <!-- Media Picker Modal -->
        <div id="mediaPickerModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.8); z-index: 1000; justify-content: center; align-items: center;">
            <div style="background: white; border-radius: 8px; max-width: 90%; max-height: 90%; width: 800px; overflow: hidden;">
                <div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3>Select Category Image</h3>
                    <button onclick="closeMediaPicker()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">×</button>
                </div>
                <div style="padding: 1rem; max-height: 500px; overflow-y: auto;">
                    <div id="mediaPickerGallery">
                        <div style="text-align: center; padding: 2rem; color: #6b7280;">Loading media...</div>
                    </div>
                </div>
                <div style="padding: 1rem; border-top: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <div style="font-size: 0.875rem; color: #6b7280;">
                        Click an image to select it for your category
                    </div>
                    <div style="display: flex; gap: 0.5rem;">
                        <button onclick="closeMediaPicker()" class="action-button" style="background-color: #6b7280;">Cancel</button>
                        <a href="/admin/content/media" target="_blank" class="action-button" style="background-color: #3b82f6; text-decoration: none;">📁 Manage Media</a>
                    </div>
                </div>
            </div>
        </div>
        
        <script>
            // Load categories when page loads
            document.addEventListener('DOMContentLoaded', function() {
                loadCategories();
                setupMediaPicker();
            });

            // Auto-generate slug from name and replace spaces with hyphens
            document.getElementById('categoryName').addEventListener('input', function() {
                const name = this.value;
                const slug = name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
                document.getElementById('categorySlug').value = slug;
            });

            // Also handle manual slug input
            document.getElementById('categorySlug').addEventListener('input', function() {
                this.value = this.value.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
            });

            function setupMediaPicker() {
                // Media library button
                document.getElementById('mediaLibraryBtn').addEventListener('click', function() {
                    openMediaPicker();
                });

                // Clear image button
                document.getElementById('clearImageBtn').addEventListener('click', function() {
                    clearSelectedImage();
                });
            }

            function openMediaPicker() {
                document.getElementById('mediaPickerModal').style.display = 'flex';
                loadMediaForPicker();
            }

            function closeMediaPicker() {
                document.getElementById('mediaPickerModal').style.display = 'none';
            }

            function loadMediaForPicker() {
                fetch('/api/v1/admin/content/media', {
                    headers: {
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('mediaPickerGallery').innerHTML = 
                            '<div style="color: #ef4444; padding: 1rem;">Error loading media: ' + data.message + '</div>';
                        return;
                    }
                    
                    const media = data.data.media || [];
                    displayMediaForPicker(media);
                })
                .catch(error => {
                    document.getElementById('mediaPickerGallery').innerHTML = 
                        '<div style="color: #ef4444; padding: 1rem;">Network error loading media</div>';
                });
            }

            function displayMediaForPicker(media) {
                const gallery = document.getElementById('mediaPickerGallery');
                
                if (media.length === 0) {
                    gallery.innerHTML = '<div style="text-align: center; padding: 2rem; color: #6b7280;">No media files found. <a href="/admin/content/media" target="_blank" style="color: #3b82f6;">Upload some images</a> to get started!</div>';
                    return;
                }
                
                let html = '<div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 1rem;">';
                
                media.forEach(item => {
                    // Find thumbnail variant if available
                    let thumbnailUrl = item.original_url;
                    if (item.variants && item.variants.length > 0) {
                        const thumbnail = item.variants.find(v => v.size === 'thumbnail' && v.format === 'webp') ||
                                          item.variants.find(v => v.size === 'small' && v.format === 'webp') ||
                                          item.variants[0];
                        if (thumbnail) {
                            thumbnailUrl = thumbnail.url;
                        }
                    }
                    
                    html += '<div class="media-picker-item" data-url="' + item.original_url + '" data-filename="' + item.filename + '" data-width="' + item.width + '" data-height="' + item.height + '" style="border: 2px solid transparent; border-radius: 8px; overflow: hidden; cursor: pointer; transition: all 0.2s ease;">' +
                        '<div style="aspect-ratio: 1; background-color: #f3f4f6; display: flex; align-items: center; justify-content: center;">' +
                        '<img src="' + thumbnailUrl + '" alt="' + (item.alt_text || item.filename) + '" style="max-width: 100%; max-height: 100%; object-fit: cover;">' +
                        '</div>' +
                        '<div style="padding: 0.5rem; background: white;">' +
                        '<p style="font-size: 0.75rem; font-weight: 600; margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + item.filename + '">' + item.filename + '</p>' +
                        '<p style="font-size: 0.7rem; color: #6b7280; margin: 0.25rem 0 0 0;">' + item.width + 'x' + item.height + '</p>' +
                        '</div>' +
                        '</div>';
                });
                
                html += '</div>';
                gallery.innerHTML = html;
                
                // Add click handlers for media selection
                document.querySelectorAll('.media-picker-item').forEach(item => {
                    item.addEventListener('click', function() {
                        selectMediaItem(this);
                    });
                    
                    item.addEventListener('mouseenter', function() {
                        this.style.borderColor = '#3b82f6';
                        this.style.transform = 'scale(1.02)';
                    });
                    
                    item.addEventListener('mouseleave', function() {
                        if (!this.classList.contains('selected')) {
                            this.style.borderColor = 'transparent';
                        }
                        this.style.transform = 'scale(1)';
                    });
                });
            }

            function selectMediaItem(element) {
                // Remove previous selection
                document.querySelectorAll('.media-picker-item').forEach(item => {
                    item.classList.remove('selected');
                    item.style.borderColor = 'transparent';
                });
                
                // Mark as selected
                element.classList.add('selected');
                element.style.borderColor = '#10b981';
                
                // Get data
                const url = element.dataset.url;
                const filename = element.dataset.filename;
                const width = element.dataset.width;
                const height = element.dataset.height;
                
                // Update form fields
                document.getElementById('categoryImageUrl').value = url;
                if (!document.getElementById('categoryImageAlt').value) {
                    document.getElementById('categoryImageAlt').value = filename.replace(/\.[^/.]+$/, '') + ' category image';
                }
                
                // Update preview
                updateImagePreview(url, filename, width, height);
                
                // Close modal after short delay
                setTimeout(() => {
                    closeMediaPicker();
                }, 500);
            }

            function updateImagePreview(url, filename, width, height) {
                const preview = document.getElementById('imagePreview');
                const previewImage = document.getElementById('previewImage');
                const previewFilename = document.getElementById('previewFilename');
                const previewDimensions = document.getElementById('previewDimensions');
                
                previewImage.src = url;
                previewImage.alt = filename;
                previewFilename.textContent = filename;
                previewDimensions.textContent = width + ' × ' + height + ' pixels';
                
                preview.style.display = 'block';
            }

            function clearSelectedImage() {
                document.getElementById('categoryImageUrl').value = '';
                document.getElementById('categoryImageAlt').value = '';
                document.getElementById('imagePreview').style.display = 'none';
            }
        
            //Submit a new category
            document.getElementById('categoryForm').addEventListener('submit', function(e) {
                e.preventDefault();
                
                const categoryName = document.getElementById('categoryName').value.trim();
                const categorySlug = document.getElementById('categorySlug').value.trim();
                const categoryDescription = document.getElementById('categoryDescription').value.trim();
                const parentId = document.getElementById('parentCategory').value;
                const sortOrder = parseInt(document.getElementById('sortOrder').value) || 0;
                const languageCode = document.getElementById('languageCode').value;
                
                if (!categoryName || !categorySlug) {
                    alert('Please fill in both category name and slug');
                    return;
                }
                
                const imageUrl = document.getElementById('categoryImageUrl').value.trim();
                const imageAltText = document.getElementById('categoryImageAlt').value.trim();
                
                console.log('Submitting category:', {
                    name: categoryName,
                    slug: categorySlug,
                    description: categoryDescription,
                    parent_id: parentId ? parseInt(parentId) : null,
                    sort_order: sortOrder,
                    language_code: languageCode,
                    image_url: imageUrl || null,
                    image_alt_text: imageAltText || null
                });
                
                // Make API call to create category
                fetch('/api/v1/admin/content/categories', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    },
                    body: JSON.stringify({
                        name: categoryName,
                        slug: categorySlug,
                        description: categoryDescription,
                        parent_id: parentId ? parseInt(parentId) : null,
                        sort_order: sortOrder,
                        language_code: languageCode,
                        image_url: imageUrl || null,
                        image_alt_text: imageAltText || null
                    })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('Error: ' + data.message);
                    } else {
                        alert('Category created successfully!');
                        document.getElementById('categoryForm').reset();
                        // Reset to default values
                        document.getElementById('sortOrder').value = '0';
                        document.getElementById('languageCode').value = 'en';
                        // Clear image preview
                        clearSelectedImage();
                        loadCategories(); // Refresh the list
                    }
                })
                .catch(error => {
                    alert('Network error: ' + error.message);
                });
            });


            function loadCategories() {
                fetch('/api/v1/admin/content/categories', {
                    headers: {
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('categoriesList').innerHTML = 
                            '<div style="color: #ef4444; padding: 1rem;">Error loading categories: ' + data.message + '</div>';
                        return;
                    }
                    
                    const categories = data.data.categories || [];
                    
                    // Populate parent category dropdown
                    const parentSelect = document.getElementById('parentCategory');
                    parentSelect.innerHTML = '<option value="">None (Top Level)</option>';
                    categories.forEach(category => {
                        parentSelect.innerHTML += '<option value="' + category.id + '">' + category.name + '</option>';
                    });
                    
                    // Rest of the existing code for displaying categories...
                    let html = '';
                    categories.forEach(category => {
                        // Find parent name if parent_id exists
                        let parentInfo = '';
                        if (category.parent_id) {
                            const parent = categories.find(cat => cat.id === category.parent_id);
                            parentInfo = parent ? ' (Child of: ' + parent.name + ')' : ' (Sub-category)';
                        }
                        
                        // Fix language flag detection
                        const langFlag = category.language_code === 'fa' ? '🇮🇷' : 
                                        category.language_code === 'ar' ? '🇸🇦' : '🇺🇸';
                        
                        // Create thumbnail HTML
                        let thumbnailHtml = '';
                        if (category.image_url) {
                            thumbnailHtml = '<img src="' + category.image_url + '" alt="' + (category.image_alt_text || category.name) + '" style="width: 40px; height: 40px; object-fit: cover; border-radius: 6px; margin-right: 1rem; border: 1px solid #e5e7eb;">';
                        } else {
                            // Show emoji placeholder based on category name
                            const emoji = category.name.toLowerCase().includes('tech') ? '💻' :
                                         category.name.toLowerCase().includes('sport') ? '⚽' :
                                         category.name.toLowerCase().includes('health') ? '🏥' :
                                         category.name.toLowerCase().includes('news') ? '📰' : '📂';
                            thumbnailHtml = '<div style="width: 40px; height: 40px; display: flex; align-items: center; justify-content: center; background: #f3f4f6; border-radius: 6px; margin-right: 1rem; font-size: 20px;">' + emoji + '</div>';
                        }
                        
                        html += '<div style="display: flex; justify-content: space-between; align-items: center; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; margin-bottom: 0.5rem;">' +
                            '<div style="display: flex; align-items: center; gap: 1rem;">' +
                            '<input type="checkbox" class="category-checkbox" value="' + category.id + '" onchange="updateBulkActions()">' +
                            thumbnailHtml +
                            '<div>' +
                            '<div><strong>' + category.name + '</strong> ' + langFlag + ' <span style="color: #6b7280;">[ID: ' + category.id + '] (' + category.slug + ')' + parentInfo + '</span></div>' +
                            '<div><small>' + (category.description || 'No description') + '</small></div>' +
                            '<div><span style="font-size: 0.75rem; color: #9ca3af;">Sort: ' + (category.sort_order || 0) + ' | Lang: ' + (category.language_code || 'en') + '</span></div>' +
                            '</div>' +
                            '</div>' +
                            '<div><button class="action-button" style="padding: 0.5rem 1rem; font-size: 0.8rem;" onclick="editCategory(' + category.id + ')">✏️ Edit</button>' +
                            '<button class="action-button" style="padding: 0.5rem 1rem; font-size: 0.8rem; background-color: #ef4444;" onclick="deleteCategory(' + category.id + ')">🗑️ Delete</button></div>' +
                            '</div>';
                    });

                    
                    if (html === '') {
                        html = '<div style="text-align: center; padding: 2rem; color: #6b7280;">No categories found</div>';
                    }
                    
                    document.getElementById('categoriesList').innerHTML = html;
                })
                .catch(error => {
                    document.getElementById('categoriesList').innerHTML = 
                        '<div style="color: #ef4444; padding: 1rem;">Network error loading categories</div>';
                });
            }

            function deleteCategory(id) {
                if (confirm('Are you sure you want to delete this category?')) {
                    fetch('/api/v1/admin/content/categories/' + id, {
                        method: 'DELETE',
                        headers: {
                            'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                        }
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.error) {
                            alert('Error deleting category: ' + data.message);
                        } else {
                            alert('Category deleted successfully!');
                            loadCategories();
                        }
                    })
                    .catch(error => {
                        alert('Network error: ' + error.message);
                    });
                }
            }

            function editCategory(id) {
                // Find the category data
                fetch('/api/v1/admin/content/categories', {
                    headers: {
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('Error loading category data: ' + data.message);
                        return;
                    }
                    
                    const categories = data.data.categories || [];
                    const category = categories.find(cat => cat.id === id);
                    
                    if (!category) {
                        alert('Category not found');
                        return;
                    }
                    
                    // Populate form with existing data
                    document.getElementById('categoryName').value = category.name;
                    document.getElementById('categorySlug').value = category.slug;
                    document.getElementById('categoryDescription').value = category.description || '';
                    document.getElementById('parentCategory').value = category.parent_id || '';
                    document.getElementById('sortOrder').value = category.sort_order || 0;
                    document.getElementById('languageCode').value = category.language_code || 'en';
                    document.getElementById('categoryImageUrl').value = category.image_url || '';
                    document.getElementById('categoryImageAlt').value = category.image_alt_text || '';
                    
                    // Update image preview if image exists
                    if (category.image_url) {
                        // Extract filename from URL
                        const filename = category.image_url.split('/').pop();
                        updateImagePreview(category.image_url, filename, 'Unknown', 'Unknown');
                    } else {
                        document.getElementById('imagePreview').style.display = 'none';
                    }
                    
                    // Change form to edit mode
                    const submitBtn = document.querySelector('#categoryForm button[type="submit"]');
                    submitBtn.innerHTML = '✏️ Update Category';
                    submitBtn.onclick = function(e) {
                        e.preventDefault();
                        updateCategory(id);
                    };
                    
                    // Scroll to form
                    document.getElementById('categoryForm').scrollIntoView({ behavior: 'smooth' });
                })
                .catch(error => {
                    alert('Network error: ' + error.message);
                });
            }

            function updateCategory(id) {
                const categoryName = document.getElementById('categoryName').value.trim();
                const categorySlug = document.getElementById('categorySlug').value.trim();
                const categoryDescription = document.getElementById('categoryDescription').value.trim();
                const parentId = document.getElementById('parentCategory').value;
                const sortOrder = parseInt(document.getElementById('sortOrder').value) || 0;
                const languageCode = document.getElementById('languageCode').value;
                const imageUrl = document.getElementById('categoryImageUrl').value.trim();
                const imageAltText = document.getElementById('categoryImageAlt').value.trim();
                
                if (!categoryName || !categorySlug) {
                    alert('Please fill in both category name and slug');
                    return;
                }
                
                fetch('/api/v1/admin/content/categories/' + id, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    },
                    body: JSON.stringify({
                        name: categoryName,
                        slug: categorySlug,
                        description: categoryDescription,
                        parent_id: parentId ? parseInt(parentId) : null,
                        sort_order: sortOrder,
                        language_code: languageCode,
                        image_url: imageUrl || null,
                        image_alt_text: imageAltText || null
                    })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('Error: ' + data.message);
                    } else {
                        alert('Category updated successfully!');
                        // Reset form to create mode
                        document.getElementById('categoryForm').reset();
                        document.getElementById('sortOrder').value = '0';
                        document.getElementById('languageCode').value = 'en';
                        // Clear image preview
                        clearSelectedImage();
                        const submitBtn = document.querySelector('#categoryForm button[type="submit"]');
                        submitBtn.innerHTML = '➕ Add Category';
                        submitBtn.onclick = null;
                        loadCategories(); // Refresh the list
                    }
                })
                .catch(error => {
                    alert('Network error: ' + error.message);
                });
            }

            // Bulk operations functions
            function toggleSelectAll() {
                const selectAll = document.getElementById('selectAllCategories');
                const checkboxes = document.querySelectorAll('.category-checkbox');
                checkboxes.forEach(cb => cb.checked = selectAll.checked);
                updateBulkActions();
            }

            function updateBulkActions() {
                const checkboxes = document.querySelectorAll('.category-checkbox:checked');
                const count = checkboxes.length;
                document.getElementById('selectedCount').textContent = count;
                document.getElementById('bulkDeleteBtn').style.display = count > 0 ? 'block' : 'none';
            }

            function bulkDeleteCategories() {
                const checkboxes = document.querySelectorAll('.category-checkbox:checked');
                const categoryIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
                
                if (categoryIds.length === 0) {
                    alert('Please select categories to delete');
                    return;
                }

                if (!confirm('Are you sure you want to delete ' + categoryIds.length + ' selected categories?')) {
                    return;
                }

                fetch('/api/v1/admin/content/categories/bulk', {
                    method: 'DELETE',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    },
                    body: JSON.stringify({ category_ids: categoryIds })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('Error: ' + data.message);
                        return;
                    }
                    
                    let message = 'Bulk delete completed:\n';
                    message += 'Deleted: ' + data.data.deleted_count + '\n';
                    message += 'Requested: ' + data.data.requested_count + '\n';
                    message += 'Errors: ' + data.data.error_count;
                    
                    if (data.data.errors && data.data.errors.length > 0) {
                        message += '\n\nErrors:\n' + data.data.errors.slice(0, 3).join('\n');
                        if (data.data.errors.length > 3) {
                            message += '\n... and ' + (data.data.errors.length - 3) + ' more errors';
                        }
                    }
                    
                    alert(message);
                    loadCategories(); // Reload the list
                    document.getElementById('selectAllCategories').checked = false;
                })
                .catch(error => {
                    alert('Network error: ' + error.message);
                });
            }

            function importCSV() {
                const fileInput = document.getElementById('csvFile');
                const file = fileInput.files[0];
                
                if (!file) {
                    alert('Please select a CSV file');
                    return;
                }

                if (!file.name.toLowerCase().endsWith('.csv')) {
                    alert('Please select a CSV file');
                    return;
                }

                const formData = new FormData();
                formData.append('csv_file', file);

                fetch('/api/v1/admin/content/categories/import', {
                    method: 'POST',
                    headers: {
                        'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
                    },
                    body: formData
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('Import error: ' + data.message);
                        return;
                    }
                    
                    let message = 'Import completed:\n';
                    message += 'Imported: ' + data.data.imported_count + '\n';
                    message += 'Skipped: ' + data.data.skipped_count + '\n';
                    message += 'Errors: ' + data.data.error_count;
                    
                    if (data.data.errors && data.data.errors.length > 0) {
                        message += '\n\nErrors:\n' + data.data.errors.slice(0, 5).join('\n');
                        if (data.data.errors.length > 5) {
                            message += '\n... and ' + (data.data.errors.length - 5) + ' more errors';
                        }
                    }
                    
                    alert(message);
                    loadCategories(); // Reload the list
                    fileInput.value = ''; // Clear file input
                })
                .catch(error => {
                    alert('Network error: ' + error.message);
                });
            }
                
        </script>
        <style>
            .media-picker-item.selected {
                border-color: #10b981 !important;
                box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
            }
            
            .media-picker-item:hover {
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            }
            
            #mediaPickerModal {
                backdrop-filter: blur(4px);
            }
        </style>`
	s.renderAdminPage(c, title, "content", content)
}

