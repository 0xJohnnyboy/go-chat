package api

import (
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

func setupSearchTestDB(t *testing.T) *gorm.DB {
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

func TestSearchHandlers_SearchUsersHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test users
	user1 := &User{Username: "john_doe", Password: hashPasswordForSearch("password123")}
	user2 := &User{Username: "jane_smith", Password: hashPasswordForSearch("password123")}
	user3 := &User{Username: "alice_johnson", Password: hashPasswordForSearch("password123")}
	searcher := &User{Username: "searcher", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)
	require.NoError(t, db.Create(user3).Error)
	require.NoError(t, db.Create(searcher).Error)
	
	// Setup handler
	sh := NewSearchHandlers(db)
	router := gin.New()
	am := auth.NewAuthMiddleware()
	router.GET("/api/search/users", am.RequireAuth(), sh.SearchUsersHandler)
	
	// Create request
	req := httptest.NewRequest("GET", "/api/search/users?q=john", nil)
	
	// Set auth context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", searcher.ID)
	
	// Call handler
	sh.SearchUsersHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	users, ok := response["users"].([]interface{})
	require.True(t, ok)
	assert.Len(t, users, 2) // john_doe and alice_johnson should match
	
	// Check first user
	firstUser := users[0].(map[string]interface{})
	assert.Contains(t, []string{"john_doe", "alice_johnson"}, firstUser["username"])
}

func TestSearchHandlers_SearchChannelsHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(user).Error)
	
	// Create test channels
	channel1 := &Channel{
		Name:      "general-chat",
		IsVisible: true,
		OwnerID:   user.ID,
	}
	channel2 := &Channel{
		Name:      "tech-discussion",
		IsVisible: true,
		OwnerID:   user.ID,
	}
	channel3 := &Channel{
		Name:      "random",
		IsVisible: false, // Hidden channel
		OwnerID:   user.ID,
	}
	require.NoError(t, db.Create(channel1).Error)
	require.NoError(t, db.Create(channel2).Error)
	require.NoError(t, db.Create(channel3).Error)
	
	// Setup handler
	sh := NewSearchHandlers(db)
	
	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/search/channels?q=chat", nil)
	c.Set("user_id", user.ID)
	
	// Call handler
	sh.SearchChannelsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	channels, ok := response["channels"].([]interface{})
	require.True(t, ok)
	assert.Len(t, channels, 1) // Only general-chat should match (visible channels only)
	
	// Check channel
	firstChannel := channels[0].(map[string]interface{})
	assert.Equal(t, "general-chat", firstChannel["name"])
}

func TestSearchHandlers_SearchMessagesHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test users
	user1 := &User{Username: "user1", Password: hashPasswordForSearch("password123")}
	user2 := &User{Username: "user2", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)
	
	// Create role
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel with history enabled
	channel := &Channel{
		Name:        "test-channel",
		IsVisible:   true,
		OwnerID:     user1.ID,
		LoggingDays: 30, // History enabled
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
		{Content: "This is a test message", UserID: user2.ID, ChannelID: channel.ID},
		{Content: "Testing search functionality", UserID: user1.ID, ChannelID: channel.ID},
		{Content: "Random content here", UserID: user2.ID, ChannelID: channel.ID},
	}
	for _, msg := range messages {
		require.NoError(t, db.Create(msg).Error)
	}
	
	// Setup handler
	sh := NewSearchHandlers(db)
	
	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/search/messages?q=test&channel_id=%s", channel.ID), nil)
	c.Set("user_id", user1.ID)
	
	// Call handler
	sh.SearchMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	messages_data, ok := response["messages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, messages_data, 2) // "test message" and "Testing search" should match
}

func TestSearchHandlers_SearchMessagesHandler_NotChannelMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForSearch("password123")}
	nonMember := &User{Username: "nonmember", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(nonMember).Error)
	
	// Create channel with history enabled
	channel := &Channel{
		Name:        "private-channel",
		IsVisible:   true,
		OwnerID:     owner.ID,
		LoggingDays: 30, // History enabled
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Setup handler
	sh := NewSearchHandlers(db)
	
	// Create request from non-member
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/search/messages?q=test&channel_id=%s", channel.ID), nil)
	c.Set("user_id", nonMember.ID)
	
	// Call handler
	sh.SearchMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "You are not a member of this channel", response["error"])
}

func TestSearchHandlers_SearchMessagesHandler_HistoryDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForSearch("password123")}
	member := &User{Username: "member", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(member).Error)
	
	// Create role
	memberRole := &Role{Name: "Member"}
	require.NoError(t, db.Create(memberRole).Error)
	
	// Create channel with history disabled (LoggingDays = 0)
	channel := &Channel{
		Name:        "no-history-channel",
		IsVisible:   true,
		OwnerID:     owner.ID,
		LoggingDays: 0, // History disabled
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Add member to channel
	userChannel := &UserChannel{
		UserID:    member.ID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}
	require.NoError(t, db.Create(userChannel).Error)
	
	// Setup handler
	sh := NewSearchHandlers(db)
	
	// Create request from member
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/search/messages?q=test&channel_id=%s", channel.ID), nil)
	c.Set("user_id", member.ID)
	
	// Call handler
	sh.SearchMessagesHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Message history is disabled for this channel", response["error"])
}

func TestSearchHandlers_SearchUsersHandler_EmptyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupSearchTestDB(t)
	
	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForSearch("password123")}
	require.NoError(t, db.Create(user).Error)
	
	// Setup handler
	sh := NewSearchHandlers(db)
	
	// Create request with empty query
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/search/users?q=", nil)
	c.Set("user_id", user.ID)
	
	// Call handler
	sh.SearchUsersHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Search query is required", response["error"])
}

func hashPasswordForSearch(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}