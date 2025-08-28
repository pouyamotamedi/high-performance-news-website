package models

import (
	"testing"
)

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			user: User{
				Username:  "testuser",
				Email:     "test@example.com",
				Role:      RoleReporter,
				FirstName: "Test",
				LastName:  "User",
				Bio:       "Test bio",
				Avatar:    "https://example.com/avatar.jpg",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			user: User{
				Email: "test@example.com",
				Role:  RoleReporter,
			},
			wantErr: true,
			errMsg:  "username is required",
		},
		{
			name: "username too short",
			user: User{
				Username: "ab",
				Email:    "test@example.com",
				Role:     RoleReporter,
			},
			wantErr: true,
			errMsg:  "username must be at least 3 characters",
		},
		{
			name: "username too long",
			user: User{
				Username: string(make([]byte, 51)), // 51 characters
				Email:    "test@example.com",
				Role:     RoleReporter,
			},
			wantErr: true,
			errMsg:  "username must be less than 50 characters",
		},
		{
			name: "invalid username characters",
			user: User{
				Username: "test-user",
				Email:    "test@example.com",
				Role:     RoleReporter,
			},
			wantErr: true,
			errMsg:  "username can only contain letters, numbers, and underscores",
		},
		{
			name: "missing email",
			user: User{
				Username: "testuser",
				Role:     RoleReporter,
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "invalid email format",
			user: User{
				Username: "testuser",
				Email:    "invalid-email",
				Role:     RoleReporter,
			},
			wantErr: true,
			errMsg:  "email format is invalid",
		},
		{
			name: "invalid role",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     "invalid",
			},
			wantErr: true,
			errMsg:  "role must be one of: admin, editor, reporter, contributor",
		},
		{
			name: "first name too long",
			user: User{
				Username:  "testuser",
				Email:     "test@example.com",
				Role:      RoleReporter,
				FirstName: string(make([]byte, 101)), // 101 characters
			},
			wantErr: true,
			errMsg:  "first_name must be less than 100 characters",
		},
		{
			name: "bio too long",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     RoleReporter,
				Bio:      string(make([]byte, 1001)), // 1001 characters
			},
			wantErr: true,
			errMsg:  "bio must be less than 1000 characters",
		},
		{
			name: "invalid avatar URL",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     RoleReporter,
				Avatar:   "not-a-url",
			},
			wantErr: true,
			errMsg:  "avatar must be a valid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUser(&tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateUser() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{
			name:     "valid username with letters",
			username: "testuser",
			want:     true,
		},
		{
			name:     "valid username with numbers",
			username: "testuser123",
			want:     true,
		},
		{
			name:     "valid username with underscores",
			username: "test_user",
			want:     true,
		},
		{
			name:     "invalid username with hyphens",
			username: "test-user",
			want:     false,
		},
		{
			name:     "invalid username with spaces",
			username: "test user",
			want:     false,
		},
		{
			name:     "invalid username with special characters",
			username: "test@user",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUsername(tt.username)
			if got != tt.want {
				t.Errorf("IsValidUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid email",
			email: "test@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "test@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "test+tag@example.com",
			want:  true,
		},
		{
			name:  "invalid email without @",
			email: "testexample.com",
			want:  false,
		},
		{
			name:  "invalid email without domain",
			email: "test@",
			want:  false,
		},
		{
			name:  "invalid email without TLD",
			email: "test@example",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidEmail(tt.email)
			if got != tt.want {
				t.Errorf("IsValidEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name string
		role UserRole
		want bool
	}{
		{
			name: "valid admin role",
			role: RoleAdmin,
			want: true,
		},
		{
			name: "valid editor role",
			role: RoleEditor,
			want: true,
		},
		{
			name: "valid reporter role",
			role: RoleReporter,
			want: true,
		},
		{
			name: "valid contributor role",
			role: RoleContributor,
			want: true,
		},
		{
			name: "invalid role",
			role: "invalid",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidRole(tt.role)
			if got != tt.want {
				t.Errorf("IsValidRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		role UserRole
		want []string
	}{
		{
			name: "admin permissions",
			role: RoleAdmin,
			want: []string{"create", "read", "update", "delete", "manage_users", "manage_system"},
		},
		{
			name: "editor permissions",
			role: RoleEditor,
			want: []string{"create", "read", "update", "delete", "publish", "moderate"},
		},
		{
			name: "reporter permissions",
			role: RoleReporter,
			want: []string{"create", "read", "update"},
		},
		{
			name: "contributor permissions",
			role: RoleContributor,
			want: []string{"create", "read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRolePermissions(tt.role)
			if len(got) != len(tt.want) {
				t.Errorf("GetRolePermissions() = %v, want %v", got, tt.want)
				return
			}
			for i, perm := range got {
				if perm != tt.want[i] {
					t.Errorf("GetRolePermissions() = %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

func TestUserHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		user       User
		permission string
		want       bool
	}{
		{
			name: "admin has manage_users permission",
			user: User{Role: RoleAdmin},
			permission: "manage_users",
			want: true,
		},
		{
			name: "editor has publish permission",
			user: User{Role: RoleEditor},
			permission: "publish",
			want: true,
		},
		{
			name: "reporter does not have delete permission",
			user: User{Role: RoleReporter},
			permission: "delete",
			want: false,
		},
		{
			name: "contributor has read permission",
			user: User{Role: RoleContributor},
			permission: "read",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.HasPermission(tt.permission)
			if got != tt.want {
				t.Errorf("User.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserCanManageUser(t *testing.T) {
	admin := &User{ID: 1, Role: RoleAdmin}
	editor := &User{ID: 2, Role: RoleEditor}
	reporter := &User{ID: 3, Role: RoleReporter}
	contributor := &User{ID: 4, Role: RoleContributor}

	tests := []struct {
		name       string
		user       *User
		targetUser *User
		want       bool
	}{
		{
			name:       "admin can manage editor",
			user:       admin,
			targetUser: editor,
			want:       true,
		},
		{
			name:       "editor can manage reporter",
			user:       editor,
			targetUser: reporter,
			want:       true,
		},
		{
			name:       "editor can manage contributor",
			user:       editor,
			targetUser: contributor,
			want:       true,
		},
		{
			name:       "reporter cannot manage editor",
			user:       reporter,
			targetUser: editor,
			want:       false,
		},
		{
			name:       "user can manage themselves",
			user:       reporter,
			targetUser: reporter,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.CanManageUser(tt.targetUser)
			if got != tt.want {
				t.Errorf("User.CanManageUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserPrepareForDB(t *testing.T) {
	user := &User{
		Username:  "  TestUser  ",
		Email:     "  TEST@EXAMPLE.COM  ",
		FirstName: "  John  ",
		LastName:  "  Doe  ",
		Bio:       "  Test bio  ",
		Role:      "",
	}

	user.PrepareForDB()

	if user.Username != "testuser" {
		t.Errorf("PrepareForDB() username = %v, want %v", user.Username, "testuser")
	}
	if user.Email != "test@example.com" {
		t.Errorf("PrepareForDB() email = %v, want %v", user.Email, "test@example.com")
	}
	if user.FirstName != "John" {
		t.Errorf("PrepareForDB() first_name = %v, want %v", user.FirstName, "John")
	}
	if user.Role != RoleContributor {
		t.Errorf("PrepareForDB() role = %v, want %v", user.Role, RoleContributor)
	}
}

func TestUserGetFullName(t *testing.T) {
	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "user with first and last name",
			user: User{
				Username:  "testuser",
				FirstName: "John",
				LastName:  "Doe",
			},
			want: "John Doe",
		},
		{
			name: "user with only first name",
			user: User{
				Username:  "testuser",
				FirstName: "John",
			},
			want: "John",
		},
		{
			name: "user with no names",
			user: User{
				Username: "testuser",
			},
			want: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.GetFullName()
			if got != tt.want {
				t.Errorf("User.GetFullName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "Password123!",
			wantErr:  false,
		},
		{
			name:     "password too short",
			password: "Pass1!",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:     "password too long",
			password: string(make([]byte, 129)),
			wantErr:  true,
			errMsg:   "password must be less than 128 characters",
		},
		{
			name:     "password without uppercase",
			password: "password123!",
			wantErr:  true,
			errMsg:   "password must contain at least one uppercase letter",
		},
		{
			name:     "password without lowercase",
			password: "PASSWORD123!",
			wantErr:  true,
			errMsg:   "password must contain at least one lowercase letter",
		},
		{
			name:     "password without number",
			password: "Password!",
			wantErr:  true,
			errMsg:   "password must contain at least one number",
		},
		{
			name:     "password without special character",
			password: "Password123",
			wantErr:  true,
			errMsg:   "password must contain at least one special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("IsValidPassword() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}