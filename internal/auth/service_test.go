package auth

import (
	"testing"
	"time"

	. "go-chat/pkg/chat"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &RefreshToken{}, &Role{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestAuthService_Register(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid registration",
			username:    "testuser",
			password:    "testpassword",
			expectError: false,
		},
		{
			name:        "empty username",
			username:    "",
			password:    "testpassword",
			expectError: true,
			errorMsg:    "username cannot be empty",
		},
		{
			name:        "empty password",
			username:    "testuser",
			password:    "",
			expectError: true,
			errorMsg:    "password cannot be empty",
		},
		{
			name:        "second valid user",
			username:    "testuser2",
			password:    "testpassword2",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Register(tt.username, tt.password)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user == nil {
				t.Error("Expected user to be created")
				return
			}

			if user.Username != tt.username {
				t.Errorf("Expected username '%s', got '%s'", tt.username, user.Username)
			}

			if user.Password == tt.password {
				t.Error("Password should be hashed, not stored in plain text")
			}

			if user.ID == "" {
				t.Error("User ID should be generated")
			}
		})
	}

	// Test duplicate username separately
	t.Run("duplicate username", func(t *testing.T) {
		// Try to register with same username as first test
		_, err := service.Register("testuser", "differentpassword")
		if err == nil {
			t.Error("Expected error for duplicate username")
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// Create a test user
	user, err := service.Register("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
	}{
		{
			name:        "valid login",
			username:    "testuser",
			password:    "testpassword",
			expectError: false,
		},
		{
			name:        "invalid username",
			username:    "nonexistent",
			password:    "testpassword",
			expectError: true,
		},
		{
			name:        "invalid password",
			username:    "testuser",
			password:    "wrongpassword",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginUser, err := service.Login(tt.username, tt.password)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if loginUser == nil {
				t.Error("Expected user to be returned")
				return
			}

			if loginUser.ID != user.ID {
				t.Errorf("Expected user ID '%s', got '%s'", user.ID, loginUser.ID)
			}

			if loginUser.Username != tt.username {
				t.Errorf("Expected username '%s', got '%s'", tt.username, loginUser.Username)
			}
		})
	}
}

func TestAuthService_CreateRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// Create a test user
	user, err := service.Register("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	token, err := service.CreateRefreshToken(user.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Verify token was stored in database
	var refreshToken RefreshToken
	err = db.Where("user_id = ?", user.ID).First(&refreshToken).Error
	if err != nil {
		t.Errorf("Token not found in database: %v", err)
		return
	}

	if refreshToken.ExpiresAt <= time.Now().Unix() {
		t.Error("Token should not be expired immediately after creation")
	}

	// Test creating multiple tokens for same user
	token2, err := service.CreateRefreshToken(user.ID)
	if err != nil {
		t.Errorf("Unexpected error creating second token: %v", err)
		return
	}

	if token == token2 {
		t.Error("Multiple tokens should be different")
	}
}

func TestAuthService_ValidateRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// Create a test user
	user, err := service.Register("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a refresh token
	token, err := service.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("Failed to create refresh token: %v", err)
	}

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid token",
			token:       token,
			expectError: false,
		},
		{
			name:        "invalid token",
			token:       "invalid-token",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedUser, err := service.ValidateRefreshToken(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if validatedUser == nil {
				t.Error("Expected user to be returned")
				return
			}

			if validatedUser.ID != user.ID {
				t.Errorf("Expected user ID '%s', got '%s'", user.ID, validatedUser.ID)
			}
		})
	}
}

func TestAuthService_RevokeRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// Create a test user
	user, err := service.Register("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a refresh token
	token, err := service.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("Failed to create refresh token: %v", err)
	}

	// Verify token works before revocation
	_, err = service.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("Token should be valid before revocation: %v", err)
	}

	// Revoke the token
	err = service.RevokeRefreshToken(token)
	if err != nil {
		t.Errorf("Unexpected error revoking token: %v", err)
	}

	// Verify token no longer works after revocation
	_, err = service.ValidateRefreshToken(token)
	if err == nil {
		t.Error("Token should be invalid after revocation")
	}

	// Test revoking non-existent token (should not error)
	err = service.RevokeRefreshToken("non-existent-token")
	if err != nil {
		t.Errorf("Revoking non-existent token should not error: %v", err)
	}
}