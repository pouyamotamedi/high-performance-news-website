package server

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) renderManageTags(c *gin.Context) {
	title := "Manage Tags"
	content := `<div class="dashboard-card">
		<div class="card-title">🏷️ Tag Management</div>
		<div style="margin-bottom: 2rem;">
			<h3>Add New Tag</h3>
			<form id="tagForm" style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-bottom: 2rem;">
				<div>
					<label for="tagName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Tag Name *</label>
					<input type="text" id="tagName" name="name" required
						   style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
				</div>
				<div>
					<label for="tagSlug" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">URL Slug *</label>
					<input type="text" id="tagSlug" name="slug" required
						   style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
				</div>
				<div style="grid-column: 1 / -1;">
					<label for="tagDescription" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Description</label>
					<textarea id="tagDescription" name="description" rows="2"
							  style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
							  placeholder="Brief description of this tag"></textarea>
				</div>
				<div style="grid-column: 1 / -1;">
					<label for="tagKeywords" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Keywords (comma-separated)</label>
					<input type="text" id="tagKeywords" name="keywords"
						   style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
						   placeholder="keyword1, keyword2, keyword3">
				</div>
				<div>
					<label for="tagColor" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Color</label>
					<input type="color" id="tagColor" name="color" value="#3b82f6"
						   style="width: 100%; height: 3rem; border: 1px solid #d1d5db; border-radius: 6px;">
				</div>
				<div>
					<label for="tagLanguage" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Language</label>
					<select id="tagLanguage" name="language_code"
							style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
						<option value="en">🇬🇧 English</option>
						<option value="de">🇩🇪 Deutsch</option>
						<option value="fr">🇫🇷 Français</option>
						<option value="es">🇪🇸 Español</option>
						<option value="ar">🇸🇦 العربية</option>
					</select>
				</div>
				<div>
					<label for="tagTranslationGroupId" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Translation Of</label>
					<select id="tagTranslationGroupId" name="translation_group_id"
							style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
						<option value="">New tag (no translation)</option>
					</select>
					<div style="font-size: 0.8rem; color: #6b7280; margin-top: 0.25rem;">
						💡 Select an existing tag to create a translation
					</div>
				</div>
				<div style="grid-column: 1 / -1; display: flex; align-items: end;">
					<button type="submit" class="action-button" style="width: 100%;">➕ Add Tag</button>
				</div>
			</form>
		</div>
		
		<!-- CSV Import Section -->
		<div style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; background: #f9fafb;">
			<h3>📥 Import Tags from CSV</h3>
			<p style="color: #6b7280; margin-bottom: 1rem;">Upload a CSV file with columns: name, slug, description, keywords, color</p>
			<div style="display: flex; gap: 1rem; align-items: end;">
				<div style="flex: 1;">
					<input type="file" id="csvFile" accept=".csv" style="width: 100%; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
				</div>
				<button type="button" onclick="importCSV()" class="action-button">📥 Import CSV</button>
				<a href="/static/samples/tags-sample.csv" class="action-button" style="background-color: #6b7280;">📄 Sample CSV</a>
			</div>
		</div>

		<h3>Existing Tags</h3>
		
		<!-- Bulk Actions -->
		<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
			<div style="display: flex; gap: 1rem; align-items: center;">
				<label style="display: flex; align-items: center; gap: 0.5rem;">
					<input type="checkbox" id="selectAllTags" onchange="toggleSelectAll()">
					<span>Select All</span>
				</label>
				<button type="button" id="bulkDeleteBtn" onclick="bulkDeleteTags()" 
						class="action-button" style="background-color: #ef4444; display: none;">
					🗑️ Delete Selected
				</button>
			</div>
			<div style="color: #6b7280; font-size: 0.9rem;">
				<span id="selectedCount">0</span> selected
			</div>
		</div>
		
		<div id="tagsList">
			<div style="text-align: center; padding: 2rem; color: #6b7280;">Loading tags...</div>
		</div>
		
		<div style="margin-top: 2rem;">
			<a href="/admin/content" class="action-button" style="background-color: #6b7280;">← Back to Content</a>
		</div>
	</div>

	<script>
		// Load tags when page loads
		document.addEventListener('DOMContentLoaded', function() {
			loadTags();
			loadTagTranslationGroups();
		});

		// Auto-generate slug from name - supports multilingual characters
		document.getElementById('tagName').addEventListener('input', function() {
			const name = this.value;
			const lang = document.getElementById('tagLanguage').value;
			let slug;
			
			if (lang === 'en') {
				// For English: only allow a-z, 0-9, and hyphens
				slug = name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
			} else {
				// For other languages: allow Unicode letters, numbers, and hyphens
				slug = name.toLowerCase()
					.replace(/\s+/g, '-')           // Replace spaces with hyphens
					.replace(/[^\p{L}\p{N}-]/gu, '') // Keep Unicode letters, numbers, hyphens
					.replace(/-+/g, '-')            // Replace multiple hyphens with single
					.replace(/^-|-$/g, '');         // Remove leading/trailing hyphens
			}
			document.getElementById('tagSlug').value = slug;
		});

		// Handle manual slug input - supports multilingual
		document.getElementById('tagSlug').addEventListener('input', function() {
			const lang = document.getElementById('tagLanguage').value;
			if (lang === 'en') {
				this.value = this.value.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
			} else {
				this.value = this.value.toLowerCase()
					.replace(/\s+/g, '-')
					.replace(/[^\p{L}\p{N}-]/gu, '')
					.replace(/-+/g, '-');
			}
		});

		// Filter translation groups when language changes
		document.getElementById('tagLanguage').addEventListener('change', function() {
			filterTagTranslationGroups(this.value);
			// Re-generate slug with new language rules
			const nameInput = document.getElementById('tagName');
			if (nameInput.value) {
				nameInput.dispatchEvent(new Event('input'));
			}
		});

		function loadTagTranslationGroups() {
			fetch('/api/v1/admin/content/tags', {
				headers: {
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				}
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) return;
				
				const tags = data.data.tags || [];
				window.allTags = tags;
				filterTagTranslationGroups(document.getElementById('tagLanguage').value);
			})
			.catch(error => console.error('Error loading translation groups:', error));
		}

		function filterTagTranslationGroups(selectedLang) {
			const select = document.getElementById('tagTranslationGroupId');
			select.innerHTML = '<option value="">New tag (no translation)</option>';
			
			if (!window.allTags || window.allTags.length === 0) return;
			
			// Group tags by translation_group_id
			const groups = {};
			window.allTags.forEach(tag => {
				const groupId = tag.translation_group_id || tag.id;
				if (!groups[groupId]) {
					groups[groupId] = [];
				}
				groups[groupId].push(tag);
			});
			
			// Show tags that don't have a translation in the selected language
			Object.keys(groups).forEach(groupId => {
				const groupTags = groups[groupId];
				const hasSelectedLang = groupTags.some(t => t.language_code === selectedLang);
				
				// Only show if this group doesn't have the selected language yet
				if (!hasSelectedLang) {
					// Find the primary tag (usually English or first one)
					const primary = groupTags.find(t => t.language_code === 'en') || groupTags[0];
					const langFlags = groupTags.map(t => {
						return t.language_code === 'de' ? '🇩🇪' : 
							   t.language_code === 'fr' ? '🇫🇷' :
							   t.language_code === 'es' ? '🇪🇸' :
							   t.language_code === 'ar' ? '🇸🇦' : '🇬🇧';
					}).join(' ');
					
					const option = document.createElement('option');
					option.value = primary.translation_group_id || primary.id;
					option.textContent = primary.name + ' (' + langFlags + ')';
					select.appendChild(option);
				}
			});
			
			// If no options were added (all tags have the selected language), show a message
			if (select.options.length === 1) {
				const option = document.createElement('option');
				option.value = '';
				option.textContent = '-- All tags already have ' + selectedLang.toUpperCase() + ' translation --';
				option.disabled = true;
				select.appendChild(option);
			}
		}

		function loadTags() {
			fetch('/api/v1/admin/content/tags', {
				headers: {
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				}
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) {
					document.getElementById('tagsList').innerHTML = 
						'<div style="color: #ef4444; padding: 1rem;">Error loading tags: ' + data.message + '</div>';
					return;
				}
				
				const tags = data.data.tags || [];
				let html = '';
				
				tags.forEach(tag => {
					const langFlag = tag.language_code === 'de' ? '🇩🇪' : 
									tag.language_code === 'fr' ? '🇫🇷' :
									tag.language_code === 'es' ? '🇪🇸' :
									tag.language_code === 'ar' ? '🇸🇦' : '🇬🇧';
					const colorDot = '<span style="display: inline-block; width: 12px; height: 12px; background-color: ' + 
									(tag.color || '#3b82f6') + '; border-radius: 50%; margin-right: 0.5rem;"></span>';
					
					// Find translation siblings
					let translationInfo = '';
					if (tag.translation_group_id) {
						const siblings = tags.filter(t => 
							t.translation_group_id === tag.translation_group_id && t.id !== tag.id
						);
						if (siblings.length > 0) {
							const siblingFlags = siblings.map(s => {
								return s.language_code === 'de' ? '🇩🇪' : 
									   s.language_code === 'fr' ? '🇫🇷' :
									   s.language_code === 'es' ? '🇪🇸' :
									   s.language_code === 'ar' ? '🇸🇦' : '🇬🇧';
							}).join(' ');
							translationInfo = '<br><span style="color: #10b981; font-size: 0.75rem;">🔗 Translations: ' + siblingFlags + '</span>';
						}
					}
					
					html += '<div style="display: flex; justify-content: space-between; align-items: center; padding: 1rem; border: 1px solid #e5e7eb; border-radius: 6px; margin-bottom: 0.5rem;">' +
						'<div style="display: flex; align-items: center; gap: 1rem;">' +
						'<input type="checkbox" class="tag-checkbox" value="' + tag.id + '" onchange="updateBulkActions()">' +
						'<div>' + colorDot + '<strong>' + tag.name + '</strong> ' + langFlag + ' <span style="color: #6b7280;">[ID: ' + tag.id + '] (' + tag.slug + ')</span><br>' +
						'<small>' + (tag.description || 'No description') + 
						(tag.keywords && tag.keywords.length > 0 ? '<br><strong>Keywords:</strong> ' + tag.keywords.join(', ') : '') + '</small>' +
						translationInfo +
						'<br><span style="font-size: 0.75rem; color: #9ca3af;">Lang: ' + (tag.language_code || 'en') + '</span></div>' +
						'</div>' +
						'<div><button class="action-button" style="padding: 0.5rem 1rem; font-size: 0.8rem;" onclick="editTag(' + tag.id + ')">✏️ Edit</button>' +
						'<button class="action-button" style="padding: 0.5rem 1rem; font-size: 0.8rem; background-color: #ef4444;" onclick="deleteTag(' + tag.id + ')">🗑️ Delete</button></div>' +
						'</div>';
				});
				
				if (html === '') {
					html = '<div style="text-align: center; padding: 2rem; color: #6b7280;">No tags found</div>';
				}
				
				document.getElementById('tagsList').innerHTML = html;
			})
			.catch(error => {
				document.getElementById('tagsList').innerHTML = 
					'<div style="color: #ef4444; padding: 1rem;">Network error loading tags</div>';
			});
		}

		// Create/Update tag form submission
		document.getElementById('tagForm').addEventListener('submit', function(e) {
			e.preventDefault();
			
			const tagName = document.getElementById('tagName').value.trim();
			const tagSlug = document.getElementById('tagSlug').value.trim();
			const tagDescription = document.getElementById('tagDescription').value.trim();
			const tagKeywords = document.getElementById('tagKeywords').value.trim();
			const keywordsArray = tagKeywords ? tagKeywords.split(',').map(k => k.trim()).filter(k => k) : [];
			const tagColor = document.getElementById('tagColor').value;
			const tagLanguage = document.getElementById('tagLanguage').value;
			const translationGroupId = document.getElementById('tagTranslationGroupId').value;
			
			if (!tagName || !tagSlug) {
				alert('Please fill in both tag name and slug');
				return;
			}
			
			const submitBtn = document.querySelector('#tagForm button[type="submit"]');
			const isEditing = submitBtn.innerHTML.includes('Update');
			const tagId = submitBtn.dataset.editId;
			
			const url = isEditing ? '/api/v1/admin/content/tags/' + tagId : '/api/v1/admin/content/tags';
			const method = isEditing ? 'PUT' : 'POST';
			
			fetch(url, {
				method: method,
				headers: {
					'Content-Type': 'application/json',
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				},
				body: JSON.stringify({
					name: tagName,
					slug: tagSlug,
					description: tagDescription,
					keywords: keywordsArray,
					color: tagColor,
					language_code: tagLanguage,
					translation_group_id: translationGroupId ? parseInt(translationGroupId) : null
				})
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) {
					alert('Error: ' + data.message);
				} else {
					alert(isEditing ? 'Tag updated successfully!' : 'Tag created successfully!');
					resetForm();
					loadTags();
					loadTagTranslationGroups();
				}
			})
			.catch(error => {
				alert('Network error: ' + error.message);
			});
		});

		function editTag(id) {
			fetch('/api/v1/admin/content/tags', {
				headers: {
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				}
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) {
					alert('Error loading tag data: ' + data.message);
					return;
				}
				
				const tags = data.data.tags || [];
				const tag = tags.find(t => t.id === id);
				
				if (!tag) {
					alert('Tag not found');
					return;
				}
				
				// Populate form
				document.getElementById('tagName').value = tag.name;
				document.getElementById('tagSlug').value = tag.slug;
				document.getElementById('tagDescription').value = tag.description || '';
				document.getElementById('tagKeywords').value = tag.keywords ? tag.keywords.join(', ') : '';
				document.getElementById('tagColor').value = tag.color || '#3b82f6';
				document.getElementById('tagLanguage').value = tag.language_code || 'en';
				document.getElementById('tagTranslationGroupId').value = tag.translation_group_id || '';
				
				// Change to edit mode
				const submitBtn = document.querySelector('#tagForm button[type="submit"]');
				submitBtn.innerHTML = '✏️ Update Tag';
				submitBtn.dataset.editId = id;
				
				// Scroll to form
				document.getElementById('tagForm').scrollIntoView({ behavior: 'smooth' });
			})
			.catch(error => {
				alert('Network error: ' + error.message);
			});
		}

		function deleteTag(id) {
			if (confirm('Are you sure you want to delete this tag?')) {
				fetch('/api/v1/admin/content/tags/' + id, {
					method: 'DELETE',
					headers: {
						'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
					}
				})
				.then(response => response.json())
				.then(data => {
					if (data.error) {
						alert('Error deleting tag: ' + data.message);
					} else {
						alert('Tag deleted successfully!');
						loadTags();
					}
				})
				.catch(error => {
					alert('Network error: ' + error.message);
				});
			}
		}

		function resetForm() {
			document.getElementById('tagForm').reset();
			document.getElementById('tagColor').value = '#3b82f6';
			document.getElementById('tagLanguage').value = 'en';
			document.getElementById('tagTranslationGroupId').value = '';
			const submitBtn = document.querySelector('#tagForm button[type="submit"]');
			submitBtn.innerHTML = '➕ Add Tag';
			delete submitBtn.dataset.editId;
		}

		// Bulk operations functions
		function toggleSelectAll() {
			const selectAll = document.getElementById('selectAllTags');
			const checkboxes = document.querySelectorAll('.tag-checkbox');
			checkboxes.forEach(cb => cb.checked = selectAll.checked);
			updateBulkActions();
		}

		function updateBulkActions() {
			const checkboxes = document.querySelectorAll('.tag-checkbox:checked');
			const count = checkboxes.length;
			document.getElementById('selectedCount').textContent = count;
			document.getElementById('bulkDeleteBtn').style.display = count > 0 ? 'block' : 'none';
		}

		function bulkDeleteTags() {
			const checkboxes = document.querySelectorAll('.tag-checkbox:checked');
			const tagIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
			
			if (tagIds.length === 0) {
				alert('Please select tags to delete');
				return;
			}

			if (!confirm('Are you sure you want to delete ' + tagIds.length + ' selected tags?')) {
				return;
			}

			fetch('/api/v1/admin/content/tags/bulk', {
				method: 'DELETE',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': 'Bearer ' + localStorage.getItem('auth_token')
				},
				body: JSON.stringify({ tag_ids: tagIds })
			})
			.then(response => response.json())
			.then(data => {
				if (data.error) {
					alert('Error: ' + data.message);
					return;
				}
				
				alert('Successfully deleted ' + data.data.deleted_count + ' tags');
				loadTags(); // Reload the list
				document.getElementById('selectAllTags').checked = false;
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

			fetch('/api/v1/admin/content/tags/import', {
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
				loadTags(); // Reload the list
				fileInput.value = ''; // Clear file input
			})
			.catch(error => {
				alert('Network error: ' + error.message);
			});
		}
	</script>`
	
	s.renderAdminPage(c, title, "content", content)
}

//Enhanced Media Library Interface
