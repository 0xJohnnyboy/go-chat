package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/internal/auth"
	. "go-chat/pkg/chat"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRoleTestDB(t *testing.T) *gorm.DB {
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

func TestChannelHandlers_PromoteUserHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupRoleTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPassword("password123")}
	member := &User{Username: "member", Password: hashPassword("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(member).Error)
	
	// Create roles
	adminRole := &Role{Name: "Administrator"}
	modRole := &Role{Name: "Moderator"}
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(adminRole).Error)
	require.NoError(t, db.Create(modRole).Error)
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Add users to channel
	ownerChannel := &UserChannel{
		UserID:    owner.ID,
		ChannelID: channel.ID,
		RoleID:    &adminRole.ID,
	}
	memberChannel := &UserChannel{
		UserID:    member.ID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}
	require.NoError(t, db.Create(ownerChannel).Error)
	require.NoError(t, db.Create(memberChannel).Error)
	
	// Setup handler
	ch := NewChannelHandlers(db)
	router := gin.New()
	am := auth.NewAuthMiddleware()
	router.POST("/api/channels/:id/promote", am.RequireAuth(), ch.PromoteUserHandler)
	
	// Create request
	reqBody := map[string]interface{}{
		"user_id": member.ID,
		"role":    "Moderator",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/channels/%s/promote", channel.ID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Set auth context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", owner.ID)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	ch.PromoteUserHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User promoted successfully", response["message"])
	
	// Verify database update
	var updatedMember UserChannel
	err = db.Preload("Role").Where("user_id = ? AND channel_id = ?", member.ID, channel.ID).First(&updatedMember).Error
	require.NoError(t, err)
	assert.Equal(t, "Moderator", updatedMember.Role.Name)
}

func TestChannelHandlers_PromoteUserHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupRoleTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPassword("password123")}
	member := &User{Username: "member", Password: hashPassword("password123")}
	nonOwner := &User{Username: "nonowner", Password: hashPassword("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(member).Error)
	require.NoError(t, db.Create(nonOwner).Error)
	
	// Create roles
	adminRole := &Role{Name: "Administrator"}
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(adminRole).Error)
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Setup handler
	ch := NewChannelHandlers(db)
	
	// Create request
	reqBody := map[string]interface{}{
		"user_id": member.ID,
		"role":    "Moderator",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/channels/%s/promote", channel.ID), bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", nonOwner.ID) // Non-owner trying to promote
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	ch.PromoteUserHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Only channel owners can promote users", response["error"])
}

func TestChannelHandlers_DemoteUserHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupRoleTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPassword("password123")}
	moderator := &User{Username: "moderator", Password: hashPassword("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(moderator).Error)
	
	// Create roles
	adminRole := &Role{Name: "Administrator"}
	modRole := &Role{Name: "Moderator"}
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(adminRole).Error)
	require.NoError(t, db.Create(modRole).Error)
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Add users to channel
	ownerChannel := &UserChannel{
		UserID:    owner.ID,
		ChannelID: channel.ID,
		RoleID:    &adminRole.ID,
	}
	modChannel := &UserChannel{
		UserID:    moderator.ID,
		ChannelID: channel.ID,
		RoleID:    &modRole.ID,
	}
	require.NoError(t, db.Create(ownerChannel).Error)
	require.NoError(t, db.Create(modChannel).Error)
	
	// Setup handler
	ch := NewChannelHandlers(db)
	
	// Create request
	reqBody := map[string]interface{}{
		"user_id": moderator.ID,
		"role":    "Member",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/channels/%s/demote", channel.ID), bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", owner.ID)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	ch.DemoteUserHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User demoted successfully", response["message"])
	
	// Verify database update
	var updatedMod UserChannel
	err = db.Preload("Role").Where("user_id = ? AND channel_id = ?", moderator.ID, channel.ID).First(&updatedMod).Error
	require.NoError(t, err)
	assert.Equal(t, "Member", updatedMod.Role.Name)
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}