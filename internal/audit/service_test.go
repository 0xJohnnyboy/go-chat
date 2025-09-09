package audit

import (
	"encoding/json"
	"testing"
	"time"

	. "go-chat/pkg/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &RefreshToken{}, &Role{}, &Channel{}, &UserChannel{}, &UserBan{}, &Message{}, &AuditLog{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func hashPasswordForAudit(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestAuditService_LogChannelCreation(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user and channel
	user := &User{Username: "testuser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   user.ID,
	}
	require.NoError(t, db.Create(channel).Error)

	// Test logging channel creation
	err := service.LogChannelCreation(user.ID, channel.ID, channel.Name, true, false)
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Where("action = ?", ActionCreateChannel).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionCreateChannel, auditLog.Action)
	assert.Equal(t, user.ID, auditLog.ActorID)
	assert.Equal(t, channel.ID, *auditLog.ChannelID)
	assert.Equal(t, "Created channel 'test-channel'", auditLog.Description)
	assert.Equal(t, user.Username, auditLog.Actor.Username)

	// Check metadata
	var metadata AuditMetadata
	err = json.Unmarshal([]byte(auditLog.Metadata), &metadata)
	require.NoError(t, err)
	assert.False(t, metadata.Password)
}

func TestAuditService_LogChannelCreation_WithPassword(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user and channel
	user := &User{Username: "testuser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	channel := &Channel{
		Name:      "secret-channel",
		IsVisible: false,
		OwnerID:   user.ID,
	}
	require.NoError(t, db.Create(channel).Error)

	// Test logging channel creation with password
	err := service.LogChannelCreation(user.ID, channel.ID, channel.Name, false, true)
	require.NoError(t, err)

	// Verify audit log metadata includes password flag
	var auditLog AuditLog
	err = db.Where("action = ?", ActionCreateChannel).First(&auditLog).Error
	require.NoError(t, err)

	var metadata AuditMetadata
	err = json.Unmarshal([]byte(auditLog.Metadata), &metadata)
	require.NoError(t, err)
	assert.True(t, metadata.Password)
}

func TestAuditService_LogChannelDeletion(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	// Test logging channel deletion
	err := service.LogChannelDeletion(user.ID, "test123", "deleted-channel")
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Where("action = ?", ActionDeleteChannel).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionDeleteChannel, auditLog.Action)
	assert.Equal(t, user.ID, auditLog.ActorID)
	assert.Equal(t, "test123", *auditLog.ChannelID)
	assert.Equal(t, "Deleted channel 'deleted-channel'", auditLog.Description)
}

func TestAuditService_LogUserBan_Permanent(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	admin := &User{Username: "admin", Password: hashPasswordForAudit("password123")}
	bannedUser := &User{Username: "baduser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(bannedUser).Error)

	// Test logging permanent ban
	err := service.LogUserBan(admin.ID, bannedUser.ID, "chan123", "spam", false, nil)
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Preload("Target").Where("action = ?", ActionBanUser).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionBanUser, auditLog.Action)
	assert.Equal(t, admin.ID, auditLog.ActorID)
	assert.Equal(t, bannedUser.ID, *auditLog.TargetID)
	assert.Equal(t, "chan123", *auditLog.ChannelID)
	assert.Equal(t, "Permanently banned user", auditLog.Description)

	// Check metadata
	var metadata AuditMetadata
	err = json.Unmarshal([]byte(auditLog.Metadata), &metadata)
	require.NoError(t, err)
	assert.Equal(t, "spam", metadata.Reason)
	assert.False(t, metadata.IsTemp)
	assert.Nil(t, metadata.ExpiresAt)
}

func TestAuditService_LogUserBan_Temporary(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	admin := &User{Username: "admin", Password: hashPasswordForAudit("password123")}
	bannedUser := &User{Username: "timeoutuser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(bannedUser).Error)

	// Test logging temporary ban
	expiresAt := time.Now().Add(24 * time.Hour)
	err := service.LogUserBan(admin.ID, bannedUser.ID, "chan123", "timeout", true, &expiresAt)
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Preload("Target").Where("action = ?", ActionTempBanUser).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionTempBanUser, auditLog.Action)
	assert.Equal(t, "Temporarily banned user", auditLog.Description)

	// Check metadata
	var metadata AuditMetadata
	err = json.Unmarshal([]byte(auditLog.Metadata), &metadata)
	require.NoError(t, err)
	assert.Equal(t, "timeout", metadata.Reason)
	assert.True(t, metadata.IsTemp)
	assert.NotNil(t, metadata.ExpiresAt)
	assert.NotEmpty(t, metadata.Duration)
}

func TestAuditService_LogUserUnban(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	admin := &User{Username: "admin", Password: hashPasswordForAudit("password123")}
	unbannedUser := &User{Username: "freeduser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(unbannedUser).Error)

	// Test logging unban
	err := service.LogUserUnban(admin.ID, unbannedUser.ID, "chan123")
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Preload("Target").Where("action = ?", ActionUnbanUser).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionUnbanUser, auditLog.Action)
	assert.Equal(t, admin.ID, auditLog.ActorID)
	assert.Equal(t, unbannedUser.ID, *auditLog.TargetID)
	assert.Equal(t, "Unbanned user", auditLog.Description)
}

func TestAuditService_LogUserRoleChange_Promotion(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	admin := &User{Username: "admin", Password: hashPasswordForAudit("password123")}
	promotedUser := &User{Username: "newmod", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(promotedUser).Error)

	// Test logging promotion
	err := service.LogUserRoleChange(admin.ID, promotedUser.ID, "chan123", "Member", "Moderator", true)
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Preload("Target").Where("action = ?", ActionPromoteUser).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionPromoteUser, auditLog.Action)
	assert.Equal(t, "Promoted user from Member to Moderator", auditLog.Description)

	// Check metadata
	var metadata AuditMetadata
	err = json.Unmarshal([]byte(auditLog.Metadata), &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Member", metadata.OldRole)
	assert.Equal(t, "Moderator", metadata.NewRole)
}

func TestAuditService_LogUserRoleChange_Demotion(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	admin := &User{Username: "admin", Password: hashPasswordForAudit("password123")}
	demotedUser := &User{Username: "exmod", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(demotedUser).Error)

	// Test logging demotion
	err := service.LogUserRoleChange(admin.ID, demotedUser.ID, "chan123", "Moderator", "Member", false)
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Preload("Target").Where("action = ?", ActionDemoteUser).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionDemoteUser, auditLog.Action)
	assert.Equal(t, "Demoted user from Moderator to Member", auditLog.Description)
}

func TestAuditService_LogChannelJoin(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user
	user := &User{Username: "joiner", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	// Test logging channel join
	err := service.LogChannelJoin(user.ID, "chan123", "welcome-channel")
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Where("action = ?", ActionJoinChannel).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionJoinChannel, auditLog.Action)
	assert.Equal(t, user.ID, auditLog.ActorID)
	assert.Equal(t, "chan123", *auditLog.ChannelID)
	assert.Equal(t, "Joined channel 'welcome-channel'", auditLog.Description)
}

func TestAuditService_LogChannelLeave(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user
	user := &User{Username: "leaver", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	// Test logging channel leave
	err := service.LogChannelLeave(user.ID, "chan123", "goodbye-channel")
	require.NoError(t, err)

	// Verify audit log was created
	var auditLog AuditLog
	err = db.Preload("Actor").Where("action = ?", ActionLeaveChannel).First(&auditLog).Error
	require.NoError(t, err)

	assert.Equal(t, ActionLeaveChannel, auditLog.Action)
	assert.Equal(t, user.ID, auditLog.ActorID)
	assert.Equal(t, "chan123", *auditLog.ChannelID)
	assert.Equal(t, "Left channel 'goodbye-channel'", auditLog.Description)
}

func TestAuditService_GetAuditLogs_WithPagination(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user).Error)

	// Create multiple audit logs
	for i := 0; i < 5; i++ {
		err := service.LogChannelJoin(user.ID, "chan123", "test-channel")
		require.NoError(t, err)
	}

	// Test pagination
	logs, total, err := service.GetAuditLogs(nil, nil, nil, 3, 0)
	require.NoError(t, err)

	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 3)

	// Test second page
	logs, total, err = service.GetAuditLogs(nil, nil, nil, 3, 3)
	require.NoError(t, err)

	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 2)
}

func TestAuditService_GetAuditLogs_WithFilters(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	user1 := &User{Username: "user1", Password: hashPasswordForAudit("password123")}
	user2 := &User{Username: "user2", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)

	// Create audit logs for different users and channels
	err := service.LogChannelJoin(user1.ID, "chan1", "channel-1")
	require.NoError(t, err)
	err = service.LogChannelJoin(user2.ID, "chan1", "channel-1")
	require.NoError(t, err)
	err = service.LogChannelJoin(user1.ID, "chan2", "channel-2")
	require.NoError(t, err)
	err = service.LogChannelLeave(user1.ID, "chan1", "channel-1")
	require.NoError(t, err)

	// Test channel filter
	channelID := "chan1"
	logs, total, err := service.GetAuditLogs(&channelID, nil, nil, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)

	// Test actor filter
	logs, total, err = service.GetAuditLogs(nil, &user1.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)

	// Test action filter
	action := ActionJoinChannel
	logs, total, err = service.GetAuditLogs(nil, nil, &action, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)

	// Test combined filters
	logs, total, err = service.GetAuditLogs(&channelID, &user1.ID, &action, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
}

func TestAuditService_GetChannelAuditLogs_OwnerOnly(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)

	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForAudit("password123")}
	nonOwner := &User{Username: "user", Password: hashPasswordForAudit("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(nonOwner).Error)

	// Create test channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)

	// Create audit log
	err := service.LogChannelJoin(nonOwner.ID, channel.ID, channel.Name)
	require.NoError(t, err)

	// Test owner can access logs
	logs, total, err := service.GetChannelAuditLogs(owner.ID, channel.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)

	// Test non-owner cannot access logs
	_, _, err = service.GetChannelAuditLogs(nonOwner.ID, channel.ID, 10, 0)
	require.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}