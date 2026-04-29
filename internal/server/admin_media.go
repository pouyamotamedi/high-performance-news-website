package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) renderMediaLibrary(c *gin.Context) {
	title := "Media Library"
	content := `<div class="dashboard-card">
		<div class="card-title">🖼️ Media Library</div>
		
		<!-- Upload Section -->
		<div style="margin-bottom: 2rem;">
			<h3>Upload New Media</h3>
			<form id="mediaForm" style="margin-bottom: 2rem;">
				<div id="dropZone" style="border: 2px dashed #d1d5db; border-radius: 8px; padding: 2rem; text-align: center; margin-bottom: 1rem; transition: all 0.3s ease;">
					<input type="file" id="mediaFile" multiple accept="image/*" style="display: none;">
					<div id="dropContent">
						<div style="font-size: 3rem; margin-bottom: 1rem;">📁</div>
						<button type="button" onclick="document.getElementById('mediaFile').click()" class="action-button">Choose Images</button>
						<p style="margin-top: 1rem; color: #6b7280;">Drag and drop images here or click to browse</p>
						<p style="font-size: 0.8rem; color: #9ca3af;">Supports: JPEG, PNG, WebP • Auto-generates: AVIF, WebP, multiple sizes</p>
					</div>
				</div>
				<div id="uploadProgress" style="display: none; margin-bottom: 1rem;">
					<div style="background-color: #f3f4f6; border-radius: 8px; padding: 1rem;">
						<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
							<span id="uploadStatus">Uploading...</span>
							<span id="uploadPercent">0%</span>
						</div>
						<div style="width: 100%; background-color: #e5e7eb; border-radius: 4px; height: 8px;">
							<div id="progressBar" style="background-color: #3b82f6; height: 100%; border-radius: 4px; width: 0%; transition: width 0.3s ease;"></div>
						</div>
					</div>
				</div>
				<button type="submit" id="uploadBtn" class="action-button">⬆️ Upload Files</button>
			</form>
		</div>
		
		<!-- Media Gallery -->
		<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
			<h3>Media Gallery</h3>
			<div style="display: flex; gap: 1rem; align-items: center;">
				<select id="filterType" style="padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 4px;">
					<option value="all">All Media</option>
					<option value="image/jpeg">JPEG Images</option>
					<option value="image/png">PNG Images</option>
					<option value="image/webp">WebP Images</option>
				</select>
				<button onclick="loadMedia()" class="action-button" style="padding: 0.5rem 1rem; font-size: 0.8rem;">🔄 Refresh</button>
			</div>
		</div>
		
		<div id="mediaGallery">
			<div style="text-align: center; padding: 2rem; color: #6b7280;">Loading media...</div>
		</div>
		
		<div style="margin-top: 2rem;">
			<a href="/admin/content" class="action-button" style="background-color: #6b7280;">← Back to Content</a>
		</div>
	</div>

	<!-- Media Preview Modal -->
	<div id="mediaModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.8); z-index: 1000; justify-content: center; align-items: center;">
		<div style="background: white; border-radius: 8px; max-width: 90%; max-height: 90%; overflow: auto;">
			<div style="padding: 1rem; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
				<h3 id="modalTitle">Media Details</h3>
				<button onclick="closeModal()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">×</button>
			</div>
			<div id="modalContent" style="padding: 1rem;">
				<!-- Modal content will be populated by JavaScript -->
			</div>
		</div>
	</div>

	<script>
		let dragCounter = 0;

		// Utility function for formatting file sizes
		function formatFileSize(bytes) {
			if (bytes === 0) return '0 Bytes';
			const k = 1024;
			const sizes = ['Bytes', 'KB', 'MB', 'GB'];
			const i = Math.floor(Math.log(bytes) / Math.log(k));
			return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
		}

		// Initialize when page loads
		document.addEventListener('DOMContentLoaded', function() {
			loadMedia();
			setupDragAndDrop();
			setupFileInput();
		});

		// Setup drag and drop functionality
		function setupDragAndDrop() {
			const dropZone = document.getElementById('dropZone');
			
			dropZone.addEventListener('dragenter', function(e) {
				e.preventDefault();
				dragCounter++;
				dropZone.style.borderColor = '#3b82f6';
				dropZone.style.backgroundColor = '#eff6ff';
			});
			
			dropZone.addEventListener('dragleave', function(e) {
				e.preventDefault();
				dragCounter--;
				if (dragCounter === 0) {
					dropZone.style.borderColor = '#d1d5db';
					dropZone.style.backgroundColor = 'transparent';
				}
			});
			
			dropZone.addEventListener('dragover', function(e) {
				e.preventDefault();
			});
			
			dropZone.addEventListener('drop', function(e) {
				e.preventDefault();
				dragCounter = 0;
				dropZone.style.borderColor = '#d1d5db';
				dropZone.style.backgroundColor = 'transparent';
				
				const files = e.dataTransfer.files;
				if (files.length > 0) {
					handleFiles(files);
				}
			});
		}

		// Setup file input
		function setupFileInput() {
			document.getElementById('mediaFile').addEventListener('change', function(e) {
				if (e.target.files.length > 0) {
					handleFiles(e.target.files);
				}
			});
		}

		// Handle file selection
		function handleFiles(files) {
			const fileArray = Array.from(files);
			const imageFiles = fileArray.filter(file => file.type.startsWith('image/'));
			
			if (imageFiles.length === 0) {
				alert('Please select image files only.');
				return;
			}
			
			// Show selected files
			const dropContent = document.getElementById('dropContent');
			dropContent.innerHTML = '<p><strong>' + imageFiles.length + ' image(s) selected</strong></p>' +
				'<p style="font-size: 0.8rem; color: #6b7280;">Ready to upload and process</p>';
			
			// Store files for upload
			document.getElementById('mediaForm').files = imageFiles;
		}

		// Form submission
		document.getElementById('mediaForm').addEventListener('submit', function(e) {
			e.preventDefault();
			
			const files = this.files;
			if (!files || files.length === 0) {
				alert('Please select files to upload.');
				return;
			}
			
			uploadFiles(files);
		});

		// Upload files using existing image upload API
		async function uploadFiles(files) {
			const uploadProgress = document.getElementById('uploadProgress');
			const uploadStatus = document.getElementById('uploadStatus');
			const uploadPercent = document.getElementById('uploadPercent');
			const progressBar = document.getElementById('progressBar');
			const uploadBtn = document.getElementById('uploadBtn');
			
			uploadProgress.style.display = 'block';
			uploadBtn.disabled = true;
			uploadBtn.textContent = 'Uploading...';
			
			let completed = 0;
			const total = files.length;
			
			for (let i = 0; i < files.length; i++) {
				const file = files[i];
				const formData = new FormData();
				formData.append('image', file);
				
				try {
					uploadStatus.textContent = 'Uploading ' + file.name + '...';
					
					const response = await fetch('/api/v1/images/upload', {
						method: 'POST',
						headers: {
							'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
						},
						body: formData
					});
					
					if (response.ok) {
						completed++;
						const percent = Math.round((completed / total) * 100);
						uploadPercent.textContent = percent + '%';
						progressBar.style.width = percent + '%';
					} else {
						console.error('Upload failed for', file.name);
					}
				} catch (error) {
					console.error('Upload error for', file.name, error);
				}
			}
			
			// Reset form
			uploadProgress.style.display = 'none';
			uploadBtn.disabled = false;
			uploadBtn.textContent = '⬆️ Upload Files';
			document.getElementById('mediaForm').reset();
			document.getElementById('dropContent').innerHTML = 
				'<div style="font-size: 3rem; margin-bottom: 1rem;">📁</div>' +
				'<button type="button" onclick="document.getElementById(\'mediaFile\').click()" class="action-button">Choose Images</button>' +
				'<p style="margin-top: 1rem; color: #6b7280;">Drag and drop images here or click to browse</p>' +
				'<p style="font-size: 0.8rem; color: #9ca3af;">Supports: JPEG, PNG, WebP • Auto-generates: AVIF, WebP, multiple sizes</p>';
			
			// Reload media gallery
			setTimeout(() => {
				loadMedia();
				alert('Upload completed! ' + completed + ' of ' + total + ' files uploaded successfully.');
			}, 1000);
		}

		// Load media from API
		function loadMedia() {
			fetch('/api/v1/admin/content/media', {
				headers: {
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				}
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) {
					document.getElementById('mediaGallery').innerHTML = 
						'<div style="color: #ef4444; padding: 1rem;">Error loading media: ' + data.message + '</div>';
					return;
				}
				
				const media = data.data.media || [];
				window.currentMedia = media; // Store globally for copy menu
				displayMedia(media);
			})
			.catch(error => {
				document.getElementById('mediaGallery').innerHTML = 
					'<div style="color: #ef4444; padding: 1rem;">Network error loading media</div>';
			});
		}

		// Display media in gallery
		function displayMedia(media) {
			const gallery = document.getElementById('mediaGallery');
			
			if (media.length === 0) {
				gallery.innerHTML = '<div style="text-align: center; padding: 2rem; color: #6b7280;">No media files found. Upload some images to get started!</div>';
				return;
			}
			
			// Get current view size (default to medium)
			const viewSize = localStorage.getItem('mediaViewSize') || 'medium';
			const sizes = {
				small: { width: '150px', height: '120px', cols: 'repeat(auto-fill, minmax(150px, 1fr))' },
				medium: { width: '200px', height: '160px', cols: 'repeat(auto-fill, minmax(200px, 1fr))' },
				large: { width: '300px', height: '240px', cols: 'repeat(auto-fill, minmax(300px, 1fr))' }
			};
			
			// Add view size controls
			let html = '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; padding: 1rem; background: #f9fafb; border-radius: 8px;">' +
				'<div>' +
				'<span style="font-weight: 600; margin-right: 1rem;">View Size:</span>' +
				'<button onclick="setViewSize(\'small\')" class="action-button" style="margin-right: 0.5rem; padding: 0.5rem 1rem; ' + (viewSize === 'small' ? 'background-color: #3b82f6; color: white;' : '') + '">Small</button>' +
				'<button onclick="setViewSize(\'medium\')" class="action-button" style="margin-right: 0.5rem; padding: 0.5rem 1rem; ' + (viewSize === 'medium' ? 'background-color: #3b82f6; color: white;' : '') + '">Medium</button>' +
				'<button onclick="setViewSize(\'large\')" class="action-button" style="padding: 0.5rem 1rem; ' + (viewSize === 'large' ? 'background-color: #3b82f6; color: white;' : '') + '">Large</button>' +
				'</div>' +
				'<div style="color: #6b7280; font-size: 0.9rem;">' + media.length + ' images</div>' +
				'</div>';
			
			html += '<div style="display: grid; grid-template-columns: ' + sizes[viewSize].cols + '; gap: 1rem;">';
			
			media.forEach(item => {
				const sizeText = formatFileSize(item.file_size);
				const dimensionsText = item.width + 'x' + item.height;
				const itemId = String(item.id); // Convert to string to preserve precision
				
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
				
				html += '<div style="border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; background: white; transition: transform 0.2s, box-shadow 0.2s;" onmouseover="this.style.transform=\'scale(1.02)\'; this.style.boxShadow=\'0 4px 12px rgba(0,0,0,0.15)\'" onmouseout="this.style.transform=\'scale(1)\'; this.style.boxShadow=\'none\'">' +
					'<div data-media-id="' + itemId + '" class="media-image" style="height: ' + sizes[viewSize].height + '; background-color: #f3f4f6; display: flex; align-items: center; justify-content: center; cursor: pointer; position: relative;">' +
					'<img src="' + thumbnailUrl + '" alt="' + (item.alt_text || item.filename) + '" style="max-width: 100%; max-height: 100%; object-fit: cover; border-radius: 4px;" onerror="handleImageError(this, \'' + item.original_url + '\')">' +
					'<div style="position: absolute; top: 0.5rem; right: 0.5rem; background: rgba(0,0,0,0.7); color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.7rem;">' + item.mime_type.split('/')[1].toUpperCase() + '</div>' +
					'</div>' +
					'<div style="padding: ' + (viewSize === 'small' ? '0.5rem' : '0.75rem') + ';">' +
					'<p style="font-weight: 600; margin-bottom: 0.25rem; font-size: ' + (viewSize === 'small' ? '0.8rem' : '0.9rem') + '; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + item.filename + '">' + item.filename + '</p>' +
					'<p style="font-size: 0.7rem; color: #6b7280; margin-bottom: 0.5rem;">' + sizeText + ' • ' + dimensionsText + '</p>' +
					'<div style="display: flex; gap: 0.25rem; flex-wrap: wrap;">' +
					'<button data-copy-id="' + itemId + '" class="copy-button action-button" style="padding: 0.4rem 0.6rem; font-size: 0.7rem; flex: 1; min-width: 60px;">📋 Copy</button>' +
					'<button data-delete-id="' + itemId + '" class="delete-button action-button" style="padding: 0.4rem 0.6rem; font-size: 0.7rem; background-color: #ef4444; color: white;">🗑️</button>' +
					'</div>' +
					'</div>' +
					'</div>';
			});
			
			html += '</div>';
			gallery.innerHTML = html;
			
			// Remove any existing event listeners by cloning elements (this removes all listeners)
			document.querySelectorAll('.copy-button, .delete-button, .media-image').forEach(element => {
				const newElement = element.cloneNode(true);
				element.parentNode.replaceChild(newElement, element);
			});
			
			// Add event listeners for copy and delete buttons
			document.querySelectorAll('.copy-button').forEach(button => {
				button.addEventListener('click', function(e) {
					e.stopPropagation();
					const id = this.getAttribute('data-copy-id');
					showCopyMenuFromButton(id, this);
				});
			});
			
			document.querySelectorAll('.delete-button').forEach(button => {
				button.addEventListener('click', function(e) {
					e.stopPropagation();
					const id = this.getAttribute('data-delete-id');
					deleteMedia(id);
				});
			});
			
			document.querySelectorAll('.media-image').forEach(imageDiv => {
				imageDiv.addEventListener('click', function(e) {
					const id = this.getAttribute('data-media-id');
					showMediaDetails(id);
				});
			});
		}

		// Utility functions

		function handleImageError(img, originalUrl) {
			if (!img.hasAttribute('data-failed')) {
				img.setAttribute('data-failed', 'true');
				img.src = originalUrl;
			} else {
				img.style.display = 'none';
				img.parentNode.innerHTML = '<div style="display: flex; align-items: center; justify-content: center; height: 100%; color: #9ca3af; font-size: 0.8rem;">📷 Image not found</div>';
			}
		}

		function copyUrl(url) {
			navigator.clipboard.writeText(url).then(() => {
				// Show a nice toast notification instead of alert
				const toast = document.createElement('div');
				toast.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #10b981; color: white; padding: 1rem 1.5rem; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.15); z-index: 2000; font-weight: 500;';
				toast.textContent = '✅ URL copied to clipboard!';
				document.body.appendChild(toast);
				
				setTimeout(() => {
					toast.style.opacity = '0';
					toast.style.transition = 'opacity 0.3s';
					setTimeout(() => document.body.removeChild(toast), 300);
				}, 2000);
			}).catch(() => {
				alert('Failed to copy URL. Please copy manually: ' + url);
			});
		}

		function deleteMedia(id) {
			if (confirm('Are you sure you want to delete this media file? This action cannot be undone.')) {
				// Use string ID to preserve precision
				fetch('/api/v1/admin/content/media/' + String(id), {
					method: 'DELETE',
					headers: {
						'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
					}
				})
				.then(response => response.json())
				.then(data => {
					if (data.error) {
						alert('Error deleting media: ' + data.message);
					} else {
						alert('Media deleted successfully!');
						loadMedia();
					}
				})
				.catch(error => {
					alert('Network error: ' + error.message);
				});
			}
		}

		function setViewSize(size) {
			localStorage.setItem('mediaViewSize', size);
			loadMedia(); // Reload to apply new size
		}

		function showCopyMenuFromButton(id, buttonElement) {
			// Use viewport coordinates (not page coordinates)
			const rect = buttonElement.getBoundingClientRect();
			
			// Position to the right of button
			let x = rect.right + 10;
			let y = rect.top + 10; // Use viewport coordinates, not page coordinates
			
			// If too far right, position to the left of button
			if (x + 200 > window.innerWidth) {
				x = rect.left - 210;
			}
			
			// If too far down, position above
			if (y + 150 > window.innerHeight) {
				y = rect.top - 150;
			}
			
			// Ensure menu stays in viewport
			if (y < 10) y = 10;
			if (x < 10) x = 10;
			
			const fakeEvent = {
				pageX: x,
				pageY: y,
				stopPropagation: function(){}
			};
			
			showCopyMenu(id, fakeEvent);
		}

		function showCopyMenu(id, event) {
			// Handle missing event object
			if (event && event.stopPropagation) {
				event.stopPropagation();
			}
			
			// Close any existing menu first
			closeMenu();
			
			// Find the media item
			const mediaItem = window.currentMedia.find(item => String(item.id) === String(id));
			if (!mediaItem) return;
			
			// Create copy menu
			const menu = document.createElement('div');
			
			// Use event coordinates if available, otherwise use default position
			const x = (event && event.pageX) ? event.pageX : 100;
			let y = (event && event.pageY) ? event.pageY : 100;
			
			// Calculate available space below the menu position
			const availableHeight = window.innerHeight - y - 20; // 20px margin from bottom
			const estimatedMenuHeight = (mediaItem.variants ? mediaItem.variants.length * 40 : 0) + 100; // Rough estimate
			
			// Determine if menu needs scrolling
			const needsScrolling = estimatedMenuHeight > availableHeight && availableHeight < 300;
			const maxHeight = needsScrolling ? Math.max(150, availableHeight) : 'auto';
			
			// If menu would be too tall and there's more space above, position above
			if (needsScrolling && y > window.innerHeight / 2) {
				y = Math.max(20, y - Math.min(300, estimatedMenuHeight));
			}
			
			// Set menu styles with conditional scrolling
			menu.style.cssText = 'position: fixed; background: white; border: 1px solid #d1d5db; border-radius: 8px; padding: 0; box-shadow: 0 4px 12px rgba(0,0,0,0.15); z-index: 1000; min-width: 200px; max-width: 300px;' + 
				(needsScrolling ? 'max-height: ' + maxHeight + 'px; overflow-y: auto; overflow-x: hidden;' : '');
			
			menu.style.left = x + 'px';
			menu.style.top = y + 'px';
			
			let menuHtml = '<div style="font-weight: 600; margin-bottom: 0.5rem; padding: 0.5rem 0.5rem 0.5rem 0.5rem; border-bottom: 1px solid #e5e7eb; position: sticky; top: 0; background: white; z-index: 1;">Copy URL</div>';
			menuHtml += '<div style="padding: 0 0.5rem 0.5rem 0.5rem;">';
			
			// Original URL
			menuHtml += '<button onclick="copyUrl(\'' + mediaItem.original_url + '\'); closeMenu()" style="display: block; width: 100%; text-align: left; padding: 0.5rem; border: none; background: none; cursor: pointer; border-radius: 4px; margin-bottom: 2px;" onmouseover="this.style.backgroundColor=\'#f3f4f6\'" onmouseout="this.style.backgroundColor=\'transparent\'">📄 Original (' + mediaItem.width + 'x' + mediaItem.height + ')</button>';
			
			// Variants
			if (mediaItem.variants && mediaItem.variants.length > 0) {
				const sizes = ['thumbnail', 'small', 'medium', 'large'];
				sizes.forEach(size => {
					const variants = mediaItem.variants.filter(v => v.size === size);
					if (variants.length > 0) {
						variants.forEach(variant => {
							const formatIcon = variant.format === 'webp' ? '🖼️' : variant.format === 'avif' ? '🎨' : '📷';
							menuHtml += '<button onclick="copyUrl(\'' + variant.url + '\'); closeMenu()" style="display: block; width: 100%; text-align: left; padding: 0.5rem; border: none; background: none; cursor: pointer; border-radius: 4px; margin-bottom: 2px;" onmouseover="this.style.backgroundColor=\'#f3f4f6\'" onmouseout="this.style.backgroundColor=\'transparent\'">' + formatIcon + ' ' + size.charAt(0).toUpperCase() + size.slice(1) + ' ' + variant.format.toUpperCase() + ' (' + variant.width + 'x' + variant.height + ')</button>';
						});
					}
				});
			}
			
			menuHtml += '</div>'; // Close the scrollable content div
			menu.innerHTML = menuHtml;
			document.body.appendChild(menu);
			
			// Close menu when clicking outside
			setTimeout(() => {
				document.addEventListener('click', closeMenu);
			}, 100);
			
			window.currentCopyMenu = menu;
		}

		function closeMenu() {
			if (window.currentCopyMenu) {
				document.body.removeChild(window.currentCopyMenu);
				window.currentCopyMenu = null;
				document.removeEventListener('click', closeMenu);
			}
		}

		function showMediaDetails(id) {
			const mediaItem = window.currentMedia.find(item => String(item.id) === String(id));
			if (!mediaItem) return;
			
			alert('Media Details:\\n\\n' +
				'Filename: ' + mediaItem.filename + '\\n' +
				'Size: ' + formatFileSize(mediaItem.file_size) + '\\n' +
				'Dimensions: ' + mediaItem.width + 'x' + mediaItem.height + '\\n' +
				'Type: ' + mediaItem.mime_type + '\\n' +
				'Variants: ' + (mediaItem.variants ? mediaItem.variants.length : 0) + '\\n' +
				'Created: ' + new Date(mediaItem.created_at).toLocaleString()
			);
		}

		function closeModal() {
			document.getElementById('mediaModal').style.display = 'none';
		}
	</script>`
	
	s.renderAdminPage(c, title, "content", content)
}


