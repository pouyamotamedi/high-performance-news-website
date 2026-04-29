package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// renderAdminUsers renders the user management admin page with real data
func (s *Server) renderAdminUsers(c *gin.Context) {
	title := "User Management"
	
	// Get user statistics from the database
	userStats := s.getUserStatistics()
	
	usersContent := fmt.Sprintf(`
        <div class="dashboard-card">
            <div class="card-title">👥 User Statistics</div>
            <div>
                <p>Total Users: <span class="metric">%d</span></p>
                <p>Active Users: <span class="metric">%d</span></p>
                <p>Admin Users: <span class="metric">%d</span></p>
                <p>Editor Users: <span class="metric">%d</span></p>
                <p>Reporter Users: <span class="metric">%d</span></p>
                <p>Contributor Users: <span class="metric">%d</span></p>
            </div>
        </div>

        <div class="dashboard-card">
            <div class="card-title">User Management Actions</div>
            <div style="display: flex; flex-wrap: wrap; gap: 0.5rem;">
                <a href="/admin/users/create" class="action-button">➕ Add New User</a>
                <a href="/admin/users/list" class="action-button">📋 List All Users</a>
                <a href="/admin/users/roles" class="action-button">🔐 Manage Roles</a>
                <a href="/admin/users/export" class="action-button">📊 Export Users</a>
            </div>
        </div>

        <div class="dashboard-card">
            <div class="card-title">Recent Users</div>
            <div id="recentUsers">
                <p>Loading recent users...</p>
            </div>
        </div>

        <script>
            // Load recent users on page load
            document.addEventListener('DOMContentLoaded', function() {
                loadRecentUsers();
            });

            async function loadRecentUsers() {
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users?limit=5&sort=created_at&order=desc', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    if (response.ok) {
                        const data = await response.json();
                        displayRecentUsers(data.data || data.users || []);
                    } else {
                        document.getElementById('recentUsers').innerHTML = '<p>Error loading recent users</p>';
                    }
                } catch (error) {
                    console.error('Error loading recent users:', error);
                    document.getElementById('recentUsers').innerHTML = '<p>Error loading recent users</p>';
                }
            }

            function displayRecentUsers(users) {
                const container = document.getElementById('recentUsers');
                if (users.length === 0) {
                    container.innerHTML = '<p>No users found</p>';
                    return;
                }

                const usersList = users.map(user => {
                    const createdAt = new Date(user.created_at).toLocaleDateString();
                    const roleIcon = getRoleIcon(user.role);
                    return '<p>' + roleIcon + ' ' + user.username + ' (' + user.role + ') - ' + createdAt + '</p>';
                }).join('');

                container.innerHTML = usersList;
            }

            function getRoleIcon(role) {
                switch(role) {
                    case 'admin': return '👑';
                    case 'editor': return '✏️';
                    case 'reporter': return '📰';
                    case 'contributor': return '✍️';
                    default: return '👤';
                }
            }
        </script>`,
		userStats.Total, userStats.Active, userStats.AdminCount, 
		userStats.EditorCount, userStats.ReporterCount, userStats.ContributorCount)

	s.renderAdminPage(c, title, "users", usersContent)
}

// UserStatistics represents user statistics for the admin panel
type UserStatistics struct {
	Total           int
	Active          int
	AdminCount      int
	EditorCount     int
	ReporterCount   int
	ContributorCount int
}

// getUserStatistics retrieves user statistics from the database
func (s *Server) getUserStatistics() UserStatistics {
	stats := UserStatistics{}
	
	// If we have a database connection, get real statistics
	if s.db != nil {
		// Get total users
		err := s.db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.Total)
		if err != nil {
			stats.Total = 0
		}
		
		// Get active users
		err = s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&stats.Active)
		if err != nil {
			stats.Active = 0
		}
		
		// Get role counts
		err = s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&stats.AdminCount)
		if err != nil {
			stats.AdminCount = 0
		}
		
		err = s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'editor'").Scan(&stats.EditorCount)
		if err != nil {
			stats.EditorCount = 0
		}
		
		err = s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'reporter'").Scan(&stats.ReporterCount)
		if err != nil {
			stats.ReporterCount = 0
		}
		
		err = s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'contributor'").Scan(&stats.ContributorCount)
		if err != nil {
			stats.ContributorCount = 0
		}
	} else {
		// Fallback to mock data if no database
		stats.Total = 15
		stats.Active = 12
		stats.AdminCount = 2
		stats.EditorCount = 3
		stats.ReporterCount = 6
		stats.ContributorCount = 4
	}
	
	return stats
}

// User Management Handlers
func (s *Server) renderCreateUser(c *gin.Context) {
	title := "Create User"
	content := `
        <div class="dashboard-card">
            <div class="card-title">👤 Create New User</div>
            <div id="messageContainer" style="margin-bottom: 1rem;"></div>
            <form id="userForm" style="display: flex; flex-direction: column; gap: 1rem;">
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                    <div>
                        <label for="username" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Username *</label>
                        <input type="text" id="username" name="username" required minlength="3" maxlength="50"
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Enter username">
                    </div>
                    <div>
                        <label for="email" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Email *</label>
                        <input type="email" id="email" name="email" required maxlength="255"
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Enter email address">
                    </div>
                </div>
                
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                    <div>
                        <label for="firstName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">First Name</label>
                        <input type="text" id="firstName" name="firstName" maxlength="100"
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Enter first name">
                    </div>
                    <div>
                        <label for="lastName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Last Name</label>
                        <input type="text" id="lastName" name="lastName" maxlength="100"
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Enter last name">
                    </div>
                </div>
                
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                    <div>
                        <label for="role" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Role *</label>
                        <select id="role" name="role" required 
                                style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">
                            <option value="">Select Role</option>
                            <option value="admin">👑 Admin - Full system access</option>
                            <option value="editor">✏️ Editor - Content management + publishing</option>
                            <option value="reporter">📰 Reporter - Content creation + editing</option>
                            <option value="contributor">✍️ Contributor - Content creation only</option>
                        </select>
                    </div>
                    <div>
                        <label for="password" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Password *</label>
                        <input type="password" id="password" name="password" required minlength="8" maxlength="128"
                               style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;"
                               placeholder="Enter password (min 8 chars)">
                    </div>
                </div>
                
                <div>
                    <label for="bio" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Bio (Optional)</label>
                    <textarea id="bio" name="bio" maxlength="1000" rows="3"
                              style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; resize: vertical;"
                              placeholder="Enter user bio (optional)"></textarea>
                </div>
                
                <div style="display: flex; gap: 1rem; margin-top: 1rem;">
                    <button type="submit" class="action-button" id="submitBtn">👤 Create User</button>
                    <a href="/admin/users" class="action-button" style="background-color: #6b7280;">← Back to Users</a>
                </div>
            </form>
        </div>
        
        <script>
            document.getElementById('userForm').addEventListener('submit', async function(e) {
                e.preventDefault();
                
                const submitBtn = document.getElementById('submitBtn');
                const messageContainer = document.getElementById('messageContainer');
                
                // Disable submit button
                submitBtn.disabled = true;
                submitBtn.textContent = '⏳ Creating User...';
                
                // Clear previous messages
                messageContainer.innerHTML = '';
                
                // Get form data
                const formData = new FormData(e.target);
                const userData = {
                    username: formData.get('username'),
                    email: formData.get('email'),
                    password: formData.get('password'),
                    role: formData.get('role'),
                    first_name: formData.get('firstName'),
                    last_name: formData.get('lastName'),
                    bio: formData.get('bio')
                };
                
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users', {
                        method: 'POST',
                        headers: {
                            'Authorization': 'Bearer ' + token,
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(userData)
                    });
                    
                    const result = await response.json();
                    
                    if (response.ok) {
                        // Success
                        messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #d1fae5; border: 1px solid #10b981; border-radius: 6px; color: #065f46;">✅ User created successfully! Username: ' + result.username + '</div>';
                        
                        // Reset form
                        e.target.reset();
                        
                        // Redirect after 2 seconds
                        setTimeout(() => {
                            window.location.href = '/admin/users';
                        }, 2000);
                    } else {
                        // Error
                        const errorMsg = result.error || 'Failed to create user';
                        messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #fee2e2; border: 1px solid #ef4444; border-radius: 6px; color: #991b1b;">❌ Error: ' + errorMsg + '</div>';
                    }
                } catch (error) {
                    console.error('Error creating user:', error);
                    messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #fee2e2; border: 1px solid #ef4444; border-radius: 6px; color: #991b1b;">❌ Network error. Please try again.</div>';
                } finally {
                    // Re-enable submit button
                    submitBtn.disabled = false;
                    submitBtn.textContent = '👤 Create User';
                }
            });
        </script>`
	s.renderAdminPage(c, title, "users", content)
}

func (s *Server) renderUserList(c *gin.Context) {
	title := "User List"
	content := `
        <div class="dashboard-card">
            <div class="card-title">📋 All Users</div>
            <div style="margin-bottom: 1rem; display: flex; gap: 1rem; align-items: center;">
                <input type="text" id="searchInput" placeholder="Search users..." 
                       style="flex: 1; padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                <select id="roleFilter" style="padding: 0.5rem; border: 1px solid #d1d5db; border-radius: 6px;">
                    <option value="">All Roles</option>
                    <option value="admin">Admin</option>
                    <option value="editor">Editor</option>
                    <option value="reporter">Reporter</option>
                    <option value="contributor">Contributor</option>
                </select>
                <button onclick="loadUsers()" class="action-button">🔍 Search</button>
            </div>
            
            <div id="usersTable">
                <p>Loading users...</p>
            </div>
            
            <div style="margin-top: 1rem;">
                <a href="/admin/users" class="action-button" style="background-color: #6b7280;">← Back to Users</a>
                <a href="/admin/users/create" class="action-button">➕ Add New User</a>
            </div>
        </div>
        
        <script>
            document.addEventListener('DOMContentLoaded', function() {
                loadUsers();
            });
            
            async function loadUsers() {
                const token = localStorage.getItem('auth_token');
                const search = document.getElementById('searchInput').value;
                const role = document.getElementById('roleFilter').value;
                
                let url = '/api/v1/admin-panel/users?limit=50';
                if (search) url += '&search=' + encodeURIComponent(search);
                if (role) url += '&role=' + encodeURIComponent(role);
                
                try {
                    const response = await fetch(url, {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    if (response.ok) {
                        const data = await response.json();
                        displayUsers(data.data || data.users || []);
                    } else {
                        document.getElementById('usersTable').innerHTML = '<p>Error loading users</p>';
                    }
                } catch (error) {
                    console.error('Error loading users:', error);
                    document.getElementById('usersTable').innerHTML = '<p>Error loading users</p>';
                }
            }
            
            function displayUsers(users) {
                const container = document.getElementById('usersTable');
                
                if (users.length === 0) {
                    container.innerHTML = '<p>No users found</p>';
                    return;
                }
                
                let table = '<table style="width: 100%; border-collapse: collapse; margin-top: 1rem;">';
                table += '<thead><tr style="background-color: #f9fafb; border-bottom: 1px solid #e5e7eb;">';
                table += '<th style="padding: 0.75rem; text-align: left;">User</th>';
                table += '<th style="padding: 0.75rem; text-align: left;">Role</th>';
                table += '<th style="padding: 0.75rem; text-align: left;">Status</th>';
                table += '<th style="padding: 0.75rem; text-align: left;">Created</th>';
                table += '<th style="padding: 0.75rem; text-align: left;">Actions</th>';
                table += '</tr></thead><tbody>';
                
                users.forEach(user => {
                    const roleIcon = getRoleIcon(user.role);
                    const statusBadge = user.is_active ? 
                        '<span style="background-color: #d1fae5; color: #065f46; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.875rem;">Active</span>' :
                        '<span style="background-color: #fee2e2; color: #991b1b; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.875rem;">Inactive</span>';
                    
                    const createdAt = new Date(user.created_at).toLocaleDateString();
                    
                    table += '<tr style="border-bottom: 1px solid #e5e7eb;">';
                    table += '<td style="padding: 0.75rem;"><div><strong>' + user.username + '</strong><br><small>' + user.email + '</small></div></td>';
                    table += '<td style="padding: 0.75rem;">' + roleIcon + ' ' + user.role + '</td>';
                    table += '<td style="padding: 0.75rem;">' + statusBadge + '</td>';
                    table += '<td style="padding: 0.75rem;">' + createdAt + '</td>';
                    table += '<td style="padding: 0.75rem;"><button onclick="editUser(' + user.id + ')" class="action-button" style="font-size: 0.875rem; padding: 0.25rem 0.5rem;">✏️ Edit</button></td>';
                    table += '</tr>';
                });
                
                table += '</tbody></table>';
                container.innerHTML = table;
            }
            
            function getRoleIcon(role) {
                switch(role) {
                    case 'admin': return '👑';
                    case 'editor': return '✏️';
                    case 'reporter': return '📰';
                    case 'contributor': return '✍️';
                    default: return '👤';
                }
            }
            
            function editUser(userId) {
                window.location.href = '/admin/users/edit/' + userId;
            }
            
            // Add search on enter key
            document.getElementById('searchInput').addEventListener('keypress', function(e) {
                if (e.key === 'Enter') {
                    loadUsers();
                }
            });
        </script>`
	s.renderAdminPage(c, title, "users", content)
}

func (s *Server) renderManageRoles(c *gin.Context) {
	title := "Manage Roles"
	content := `
        <div class="dashboard-card">
            <div class="card-title">🔐 Role Management</div>
            <p style="margin-bottom: 2rem;">Manage user roles and their permissions in the system.</p>
            
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem;">
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem;">
                    <h3 style="margin: 0 0 1rem 0; color: #1f2937;">👑 Admin</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Full system access and control</p>
                    <div style="font-size: 0.875rem;">
                        <strong>Permissions:</strong>
                        <ul style="margin: 0.5rem 0; padding-left: 1.5rem;">
                            <li>Create, read, update, delete content</li>
                            <li>Manage users and roles</li>
                            <li>System administration</li>
                            <li>All editor and reporter permissions</li>
                        </ul>
                    </div>
                </div>
                
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem;">
                    <h3 style="margin: 0 0 1rem 0; color: #1f2937;">✏️ Editor</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Content management and publishing</p>
                    <div style="font-size: 0.875rem;">
                        <strong>Permissions:</strong>
                        <ul style="margin: 0.5rem 0; padding-left: 1.5rem;">
                            <li>Create, read, update, delete content</li>
                            <li>Publish and moderate content</li>
                            <li>Manage reporters and contributors</li>
                        </ul>
                    </div>
                </div>
                
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem;">
                    <h3 style="margin: 0 0 1rem 0; color: #1f2937;">📰 Reporter</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Content creation and editing</p>
                    <div style="font-size: 0.875rem;">
                        <strong>Permissions:</strong>
                        <ul style="margin: 0.5rem 0; padding-left: 1.5rem;">
                            <li>Create new content</li>
                            <li>Read all content</li>
                            <li>Update own content</li>
                        </ul>
                    </div>
                </div>
                
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem;">
                    <h3 style="margin: 0 0 1rem 0; color: #1f2937;">✍️ Contributor</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Basic content creation</p>
                    <div style="font-size: 0.875rem;">
                        <strong>Permissions:</strong>
                        <ul style="margin: 0.5rem 0; padding-left: 1.5rem;">
                            <li>Create new content</li>
                            <li>Read published content</li>
                        </ul>
                    </div>
                </div>
            </div>
            
            <div style="margin-top: 2rem;">
                <a href="/admin/users" class="action-button" style="background-color: #6b7280;">← Back to Users</a>
            </div>
        </div>`
	s.renderAdminPage(c, title, "users", content)
}

func (s *Server) renderEditUser(c *gin.Context) {
	title := "Edit User"
	userID := c.Param("id")
	
	content := fmt.Sprintf(`
        <div class="dashboard-card">
            <div class="card-title">✏️ Edit User</div>
            <div id="messageContainer" style="margin-bottom: 1rem;"></div>
            <div id="userFormContainer">
                <p>Loading user data...</p>
            </div>
            
            <div style="margin-top: 1rem;">
                <a href="/admin/users/list" class="action-button" style="background-color: #6b7280;">← Back to User List</a>
            </div>
        </div>
        
        <script>
            const userId = %s;
            
            document.addEventListener('DOMContentLoaded', function() {
                loadUserData();
            });
            
            async function loadUserData() {
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users/' + userId, {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    if (response.ok) {
                        const data = await response.json();
                        const user = data.data || data;
                        displayEditForm(user);
                    } else {
                        document.getElementById('userFormContainer').innerHTML = '<p>Error loading user data</p>';
                    }
                } catch (error) {
                    console.error('Error loading user:', error);
                    document.getElementById('userFormContainer').innerHTML = '<p>Error loading user data</p>';
                }
            }
            
            function displayEditForm(user) {
                const container = document.getElementById('userFormContainer');
                container.innerHTML = '<form id="editUserForm" style="display: flex; flex-direction: column; gap: 1rem;">' +
                    '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">' +
                        '<div>' +
                            '<label for="username" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Username *</label>' +
                            '<input type="text" id="username" name="username" required minlength="3" maxlength="50" value="' + user.username + '"' +
                                   'style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                        '</div>' +
                        '<div>' +
                            '<label for="email" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Email *</label>' +
                            '<input type="email" id="email" name="email" required maxlength="255" value="' + user.email + '"' +
                                   'style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                        '</div>' +
                    '</div>' +
                    '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">' +
                        '<div>' +
                            '<label for="firstName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">First Name</label>' +
                            '<input type="text" id="firstName" name="firstName" maxlength="100" value="' + (user.first_name || '') + '"' +
                                   'style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                        '</div>' +
                        '<div>' +
                            '<label for="lastName" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Last Name</label>' +
                            '<input type="text" id="lastName" name="lastName" maxlength="100" value="' + (user.last_name || '') + '"' +
                                   'style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                        '</div>' +
                    '</div>' +
                    '<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">' +
                        '<div>' +
                            '<label for="role" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Role *</label>' +
                            '<select id="role" name="role" required style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                                '<option value="admin"' + (user.role === 'admin' ? ' selected' : '') + '>👑 Admin</option>' +
                                '<option value="editor"' + (user.role === 'editor' ? ' selected' : '') + '>✏️ Editor</option>' +
                                '<option value="reporter"' + (user.role === 'reporter' ? ' selected' : '') + '>📰 Reporter</option>' +
                                '<option value="contributor"' + (user.role === 'contributor' ? ' selected' : '') + '>✍️ Contributor</option>' +
                            '</select>' +
                        '</div>' +
                        '<div>' +
                            '<label for="isActive" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Status</label>' +
                            '<select id="isActive" name="isActive" style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px;">' +
                                '<option value="true"' + (user.is_active ? ' selected' : '') + '>Active</option>' +
                                '<option value="false"' + (!user.is_active ? ' selected' : '') + '>Inactive</option>' +
                            '</select>' +
                        '</div>' +
                    '</div>' +
                    '<div>' +
                        '<label for="bio" style="display: block; margin-bottom: 0.5rem; font-weight: 600;">Bio</label>' +
                        '<textarea id="bio" name="bio" maxlength="1000" rows="3"' +
                                  'style="width: 100%%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 6px; resize: vertical;">' + (user.bio || '') + '</textarea>' +
                    '</div>' +
                    '<div style="display: flex; gap: 1rem; margin-top: 1rem;">' +
                        '<button type="submit" class="action-button" id="updateBtn">💾 Update User</button>' +
                        '<button type="button" onclick="deleteUser()" class="action-button" style="background-color: #ef4444;">🗑️ Delete User</button>' +
                    '</div>' +
                '</form>';
                
                // Add form submit handler
                document.getElementById('editUserForm').addEventListener('submit', updateUser);
            }
            
            async function updateUser(e) {
                e.preventDefault();
                
                const updateBtn = document.getElementById('updateBtn');
                const messageContainer = document.getElementById('messageContainer');
                
                updateBtn.disabled = true;
                updateBtn.textContent = '⏳ Updating...';
                messageContainer.innerHTML = '';
                
                const formData = new FormData(e.target);
                const userData = {
                    username: formData.get('username'),
                    email: formData.get('email'),
                    role: formData.get('role'),
                    first_name: formData.get('firstName'),
                    last_name: formData.get('lastName'),
                    bio: formData.get('bio'),
                    is_active: formData.get('isActive') === 'true'
                };
                
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users/' + userId, {
                        method: 'PUT',
                        headers: { 
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + token
                        },
                        body: JSON.stringify(userData)
                    });
                    
                    const result = await response.json();
                    
                    if (response.ok) {
                        messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #d1fae5; border: 1px solid #10b981; border-radius: 6px; color: #065f46;">✅ User updated successfully!</div>';
                    } else {
                        messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #fee2e2; border: 1px solid #ef4444; border-radius: 6px; color: #991b1b;">❌ Error: ' + (result.error || 'Failed to update user') + '</div>';
                    }
                } catch (error) {
                    messageContainer.innerHTML = '<div style="padding: 1rem; background-color: #fee2e2; border: 1px solid #ef4444; border-radius: 6px; color: #991b1b;">❌ Network error. Please try again.</div>';
                } finally {
                    updateBtn.disabled = false;
                    updateBtn.textContent = '💾 Update User';
                }
            }
            
            async function deleteUser() {
                if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
                    return;
                }
                
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users/' + userId, {
                        method: 'DELETE',
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    
                    if (response.ok) {
                        alert('User deleted successfully');
                        window.location.href = '/admin/users/list';
                    } else {
                        const result = await response.json();
                        alert('Error deleting user: ' + (result.error || 'Unknown error'));
                    }
                } catch (error) {
                    alert('Network error. Please try again.');
                }
            }
        </script>`, userID)
	
	s.renderAdminPage(c, title, "users", content)
}

func (s *Server) renderExportUsers(c *gin.Context) {
	title := "Export Users"
	content := `
        <div class="dashboard-card">
            <div class="card-title">📊 Export Users</div>
            <p style="margin-bottom: 2rem;">Export user data in various formats for reporting and analysis.</p>
            
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem; margin-bottom: 2rem;">
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem; text-align: center;">
                    <h3 style="margin: 0 0 1rem 0;">📄 CSV Export</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Export all users to CSV format</p>
                    <button onclick="exportUsers('csv')" class="action-button">📄 Export CSV</button>
                </div>
                
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem; text-align: center;">
                    <h3 style="margin: 0 0 1rem 0;">📊 Excel Export</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Export all users to Excel format</p>
                    <button onclick="exportUsers('xlsx')" class="action-button">📊 Export Excel</button>
                </div>
                
                <div style="border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem; text-align: center;">
                    <h3 style="margin: 0 0 1rem 0;">🔧 JSON Export</h3>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Export all users to JSON format</p>
                    <button onclick="exportUsers('json')" class="action-button">🔧 Export JSON</button>
                </div>
            </div>
            
            <div style="background-color: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; padding: 1rem;">
                <h4 style="margin: 0 0 1rem 0;">Export Options</h4>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                    <label style="display: flex; align-items: center; gap: 0.5rem;">
                        <input type="checkbox" id="includeInactive" checked>
                        Include inactive users
                    </label>
                    <label style="display: flex; align-items: center; gap: 0.5rem;">
                        <input type="checkbox" id="includePasswords">
                        Include password hashes (Admin only)
                    </label>
                    <label style="display: flex; align-items: center; gap: 0.5rem;">
                        <input type="checkbox" id="includeTimestamps" checked>
                        Include timestamps
                    </label>
                </div>
            </div>
            
            <div style="margin-top: 2rem;">
                <a href="/admin/users" class="action-button" style="background-color: #6b7280;">← Back to Users</a>
            </div>
        </div>
        
        <script>
            async function exportUsers(format) {
                const includeInactive = document.getElementById('includeInactive').checked;
                const includePasswords = document.getElementById('includePasswords').checked;
                const includeTimestamps = document.getElementById('includeTimestamps').checked;
                
                const params = new URLSearchParams({
                    format: format,
                    include_inactive: includeInactive,
                    include_passwords: includePasswords,
                    include_timestamps: includeTimestamps
                });
                
                try {
                    const token = localStorage.getItem('auth_token');
                    const response = await fetch('/api/v1/admin-panel/users/export?' + params.toString(), {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    
                    if (response.ok) {
                        const blob = await response.blob();
                        const url = window.URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = 'users_export_' + new Date().toISOString().split('T')[0] + '.' + format;
                        document.body.appendChild(a);
                        a.click();
                        document.body.removeChild(a);
                        window.URL.revokeObjectURL(url);
                    } else {
                        alert('Export failed. Please try again.');
                    }
                } catch (error) {
                    console.error('Export error:', error);
                    alert('Export failed. Please try again.');
                }
            }
        </script>`
	s.renderAdminPage(c, title, "users", content)
}