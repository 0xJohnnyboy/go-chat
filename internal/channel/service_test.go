package channel

import (
	"testing"

	. "go-chat/pkg/chat"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &Channel{}, &UserChannel{}, &Role{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create default roles
	roles := []Role{
		{Name: "Administrator"},
		{Name: "Moderator"},
		{Name: "Member"},
		{Name: "Guest"},
	}
	for _, role := range roles {
		db.Create(&role)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username string) *User {
	user := User{Username: username, Password: "hashedpassword"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return &user
}

func TestChannelService_CreateChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	user := createTestUser(t, db, "testuser")

	tests := []struct {
		name        string
		ownerID     string
		channelName string
		password    *string
		isVisible   bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid channel creation",
			ownerID:     user.ID,
			channelName: "testchannel",
			password:    nil,
			isVisible:   true,
			expectError: false,
		},
		{
			name:        "channel with password",
			ownerID:     user.ID,
			channelName: "privatechannel",
			password:    stringPtr("secret123"),
			isVisible:   false,
			expectError: false,
		},
		{
			name:        "empty channel name",
			ownerID:     user.ID,
			channelName: "",
			password:    nil,
			isVisible:   true,
			expectError: true,
			errorMsg:    "channel name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := service.CreateChannel(tt.ownerID, tt.channelName, tt.password, tt.isVisible)

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

			if channel == nil {
				t.Error("Expected channel to be created")
				return
			}

			if channel.Name != tt.channelName {
				t.Errorf("Expected channel name '%s', got '%s'", tt.channelName, channel.Name)
			}

			if channel.OwnerID != tt.ownerID {
				t.Errorf("Expected owner ID '%s', got '%s'", tt.ownerID, channel.OwnerID)
			}

			if channel.IsVisible != tt.isVisible {
				t.Errorf("Expected visibility %v, got %v", tt.isVisible, channel.IsVisible)
			}

			// Verify owner is added as administrator
			var userChannel UserChannel
			err = db.Preload("Role").Where("user_id = ? AND channel_id = ?", tt.ownerID, channel.ID).First(&userChannel).Error
			if err != nil {
				t.Errorf("Owner should be added to channel: %v", err)
				return
			}

			if userChannel.Role.Name != "Administrator" {
				t.Errorf("Owner should have Administrator role, got '%s'", userChannel.Role.Name)
			}
		})
	}
}

func TestChannelService_GetVisibleChannels(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	user := createTestUser(t, db, "testuser")

	// Create visible and invisible channels
	visibleChannel, err := service.CreateChannel(user.ID, "visible", nil, true)
	if err != nil {
		t.Fatalf("Failed to create visible channel: %v", err)
	}

	_, err = service.CreateChannel(user.ID, "invisible", nil, false)
	if err != nil {
		t.Fatalf("Failed to create invisible channel: %v", err)
	}

	channels, err := service.GetVisibleChannels()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(channels) != 1 {
		t.Errorf("Expected 1 visible channel, got %d", len(channels))
		return
	}

	if channels[0].ID != visibleChannel.ID {
		t.Errorf("Expected visible channel ID '%s', got '%s'", visibleChannel.ID, channels[0].ID)
	}
}

func TestChannelService_GetUserChannels(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	user1 := createTestUser(t, db, "user1")
	user2 := createTestUser(t, db, "user2")

	// Create channels for different users
	channel1, err := service.CreateChannel(user1.ID, "channel1", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel1: %v", err)
	}

	_, err = service.CreateChannel(user2.ID, "channel2", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel2: %v", err)
	}

	channels, err := service.GetUserChannels(user1.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(channels) != 1 {
		t.Errorf("Expected 1 channel for user1, got %d", len(channels))
		return
	}

	if channels[0].ID != channel1.ID {
		t.Errorf("Expected channel ID '%s', got '%s'", channel1.ID, channels[0].ID)
	}
}

func TestChannelService_JoinChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	owner := createTestUser(t, db, "owner")

	// Create channels with different configurations
	publicChannel, err := service.CreateChannel(owner.ID, "public", nil, true)
	if err != nil {
		t.Fatalf("Failed to create public channel: %v", err)
	}

	privateChannel, err := service.CreateChannel(owner.ID, "private", stringPtr("secret"), true)
	if err != nil {
		t.Fatalf("Failed to create private channel: %v", err)
	}

	tests := []struct {
		name        string
		setupUser   func(t *testing.T) *User
		channelID   string
		password    *string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "join public channel",
			setupUser:   func(t *testing.T) *User { return createTestUser(t, db, "user1") },
			channelID:   publicChannel.ID,
			password:    nil,
			expectError: false,
		},
		{
			name:        "join private channel with correct password",
			setupUser:   func(t *testing.T) *User { return createTestUser(t, db, "user2") },
			channelID:   privateChannel.ID,
			password:    stringPtr("secret"),
			expectError: false,
		},
		{
			name:        "join private channel with wrong password",
			setupUser:   func(t *testing.T) *User { return createTestUser(t, db, "user3") },
			channelID:   privateChannel.ID,
			password:    stringPtr("wrong"),
			expectError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "join private channel without password",
			setupUser:   func(t *testing.T) *User { return createTestUser(t, db, "user4") },
			channelID:   privateChannel.ID,
			password:    nil,
			expectError: true,
			errorMsg:    "password required for this channel",
		},
		{
			name:        "join non-existent channel",
			setupUser:   func(t *testing.T) *User { return createTestUser(t, db, "user5") },
			channelID:   "nonexistent",
			password:    nil,
			expectError: true,
		},
	}

	// Test joining same channel twice
	t.Run("join same channel twice", func(t *testing.T) {
		user := createTestUser(t, db, "duplicateuser")
		
		// First join should succeed
		err := service.JoinChannel(user.ID, publicChannel.ID, nil)
		if err != nil {
			t.Errorf("First join should succeed: %v", err)
			return
		}
		
		// Second join should fail
		err = service.JoinChannel(user.ID, publicChannel.ID, nil)
		if err == nil {
			t.Errorf("Expected error for duplicate join")
			return
		}
		
		if err.Error() != "user already in channel" {
			t.Errorf("Expected 'user already in channel', got '%s'", err.Error())
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.setupUser(t)
			err := service.JoinChannel(user.ID, tt.channelID, tt.password)

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

			// Verify user was added to channel with Member role
			var userChannel UserChannel
			err = db.Preload("Role").Where("user_id = ? AND channel_id = ?", user.ID, tt.channelID).First(&userChannel).Error
			if err != nil {
				t.Errorf("User should be added to channel: %v", err)
				return
			}

			if userChannel.Role.Name != "Member" {
				t.Errorf("User should have Member role, got '%s'", userChannel.Role.Name)
			}
		})
	}
}

func TestChannelService_LeaveChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	owner := createTestUser(t, db, "owner")
	user := createTestUser(t, db, "user")

	channel, err := service.CreateChannel(owner.ID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Add user to channel
	err = service.JoinChannel(user.ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	tests := []struct {
		name        string
		userID      string
		channelID   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "user leaves channel",
			userID:      user.ID,
			channelID:   channel.ID,
			expectError: false,
		},
		{
			name:        "owner tries to leave own channel",
			userID:      owner.ID,
			channelID:   channel.ID,
			expectError: true,
			errorMsg:    "channel owner cannot leave channel",
		},
		{
			name:        "leave non-existent channel",
			userID:      user.ID,
			channelID:   "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.LeaveChannel(tt.userID, tt.channelID)

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

			// Verify user was removed from channel
			var count int64
			db.Model(&UserChannel{}).Where("user_id = ? AND channel_id = ?", tt.userID, tt.channelID).Count(&count)
			if count != 0 {
				t.Errorf("User should be removed from channel")
			}
		})
	}
}

func TestChannelService_DeleteChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	owner := createTestUser(t, db, "owner")
	user := createTestUser(t, db, "user")

	channel, err := service.CreateChannel(owner.ID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Add user to channel
	err = service.JoinChannel(user.ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	tests := []struct {
		name        string
		userID      string
		channelID   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "non-owner tries to delete channel",
			userID:      user.ID,
			channelID:   channel.ID,
			expectError: true,
			errorMsg:    "only channel owner can delete channel",
		},
		{
			name:        "owner deletes channel",
			userID:      owner.ID,
			channelID:   channel.ID,
			expectError: false,
		},
		{
			name:        "delete non-existent channel",
			userID:      owner.ID,
			channelID:   "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteChannel(tt.userID, tt.channelID)

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

			// Verify channel and user relationships were deleted
			var channelCount, userChannelCount int64
			db.Model(&Channel{}).Where("id = ?", tt.channelID).Count(&channelCount)
			db.Model(&UserChannel{}).Where("channel_id = ?", tt.channelID).Count(&userChannelCount)

			if channelCount != 0 {
				t.Errorf("Channel should be deleted")
			}
			if userChannelCount != 0 {
				t.Errorf("User-channel relationships should be deleted")
			}
		})
	}
}

func TestChannelService_GetChannelUsers(t *testing.T) {
	db := setupTestDB(t)
	service := NewChannelService(db)
	owner := createTestUser(t, db, "owner")
	user1 := createTestUser(t, db, "user1")
	user2 := createTestUser(t, db, "user2")

	channel, err := service.CreateChannel(owner.ID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Add users to channel
	err = service.JoinChannel(user1.ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user1 to channel: %v", err)
	}

	err = service.JoinChannel(user2.ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user2 to channel: %v", err)
	}

	users, err := service.GetChannelUsers(channel.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Should have 3 users: owner + 2 members
	if len(users) != 3 {
		t.Errorf("Expected 3 users in channel, got %d", len(users))
		return
	}

	userIDs := make(map[string]bool)
	for _, u := range users {
		userIDs[u.ID] = true
	}

	expectedUsers := []string{owner.ID, user1.ID, user2.ID}
	for _, expectedID := range expectedUsers {
		if !userIDs[expectedID] {
			t.Errorf("Expected user %s to be in channel", expectedID)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}