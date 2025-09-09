package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-chat/internal/auth"
	. "go-chat/pkg/chat"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMessageTestDB(t *testing.T) *gorm.DB {
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

func TestMessageHandlers_GetChannelMessagesHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupMessageTestDB(t)
	
	// Create test users
	user1 := &User{Username: "user1", Password: hashPasswordForTest("password123")}
	user2 := &User{Username: "user2", Password: hashPasswordForTest("password123")}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)
	
	// Create roles
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   user1.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Add users to channel
	userChannel1 := &UserChannel{
		UserID:    user1.ID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}
	userChannel2 := &UserChannel{
		UserID:    user2.ID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}
	require.NoError(t, db.Create(userChannel1).Error)
	require.NoError(t, db.Create(userChannel2).Error)
	
	// Create test messages
	messages := []*Message{
		{Content: "Hello everyone!", UserID: user1.ID, ChannelID: channel.ID},
		{Content: "Hi there!", UserID: user2.ID, ChannelID: channel.ID},
		{Content: "How are you?", UserID: user1.ID, ChannelID: channel.ID},
	}
	for _, msg := range messages {
		require.NoError(t, db.Create(msg).Error)
	}
	
	// Setup handler
	mh := NewMessageHandlers(db)
	router := gin.New()
	am := auth.NewAuthMiddleware()
	router.GET("/api/channels/:id/messages", am.RequireAuth(), mh.GetChannelMessagesHandler)
	
	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/messages", channel.ID), nil)
	
	// Set auth context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", user1.ID)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	mh.GetChannelMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	messages_data, ok := response["messages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, messages_data, 3)
	
	// Check first message
	firstMsg := messages_data[0].(map[string]interface{})
	assert.Equal(t, "Hello everyone!", firstMsg["content"])
	assert.Equal(t, user1.ID, firstMsg["user_id"])
}

func TestMessageHandlers_GetChannelMessagesHandler_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupMessageTestDB(t)
	
	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForTest("password123")}
	require.NoError(t, db.Create(user).Error)
	
	// Create role
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   user.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Add user to channel
	userChannel := &UserChannel{
		UserID:    user.ID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}
	require.NoError(t, db.Create(userChannel).Error)
	
	// Create 15 test messages
	for i := 0; i < 15; i++ {
		msg := &Message{
			Content:   fmt.Sprintf("Message %d", i+1),
			UserID:    user.ID,
			ChannelID: channel.ID,
		}
		require.NoError(t, db.Create(msg).Error)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}
	
	// Setup handler
	mh := NewMessageHandlers(db)
	
	// Test pagination with limit=10
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/messages?limit=10", channel.ID), nil)
	c.Set("user_id", user.ID)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	mh.GetChannelMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	messages_data, ok := response["messages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, messages_data, 10) // Should return only 10 messages
}

func TestMessageHandlers_GetChannelMessagesHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupMessageTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForTest("password123")}
	nonMember := &User{Username: "nonmember", Password: hashPasswordForTest("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(nonMember).Error)
	
	// Create channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Setup handler
	mh := NewMessageHandlers(db)
	
	// Create request from non-member
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/messages", channel.ID), nil)
	c.Set("user_id", nonMember.ID)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	
	// Call handler
	mh.GetChannelMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "You are not a member of this channel", response["error"])
}

func hashPasswordForTest(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}