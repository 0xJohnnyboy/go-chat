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

func hashPasswordForAuditTest(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestAuditHandlers_GetChannelAuditLogsHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForAuditTest("password123")}
	member := &User{Username: "member", Password: hashPasswordForAuditTest("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(member).Error)
	
	// Create test channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Create audit logs
	logs := []AuditLog{
		{
			Action:      "CREATE_CHANNEL",
			ActorID:     owner.ID,
			ChannelID:   &channel.ID,
			Description: "Created channel",
			Metadata:    "{}",
		},
		{
			Action:      "JOIN_CHANNEL",
			ActorID:     member.ID,
			ChannelID:   &channel.ID,
			Description: "Joined channel",
			Metadata:    "{}",
		},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(&log).Error)
	}
	
	// Setup handler
	ah := NewAuditHandlers(db)
	router := gin.New()
	am := auth.NewAuthMiddleware()
	router.GET("/api/channels/:id/audit", am.RequireAuth(), ah.GetChannelAuditLogsHandler)
	
	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/audit", channel.ID), nil)
	
	// Set auth context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	c.Set("user_id", owner.ID)
	
	// Call handler
	ah.GetChannelAuditLogsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response AuditLogsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, int64(2), response.Total)
	assert.Len(t, response.Logs, 2)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.Limit)
	
	// Check first log
	firstLog := response.Logs[0]
	assert.Equal(t, "JOIN_CHANNEL", firstLog.Action) // Should be newest first
	assert.Equal(t, member.ID, firstLog.ActorID)
	assert.Equal(t, channel.ID, *firstLog.ChannelID)
	assert.Equal(t, "Joined channel", firstLog.Description)
	assert.Equal(t, member.Username, firstLog.Actor.Username)
}

func TestAuditHandlers_GetChannelAuditLogsHandler_NonOwnerAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Create test users
	owner := &User{Username: "owner", Password: hashPasswordForAuditTest("password123")}
	nonOwner := &User{Username: "nonowner", Password: hashPasswordForAuditTest("password123")}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(nonOwner).Error)
	
	// Create test channel
	channel := &Channel{
		Name:      "private-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Setup handler
	ah := NewAuditHandlers(db)
	
	// Create request from non-owner
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/audit", channel.ID), nil)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	c.Set("user_id", nonOwner.ID)
	
	// Call handler
	ah.GetChannelAuditLogsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Channel not found or access denied", response["error"])
}

func TestAuditHandlers_GetChannelAuditLogsHandler_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Create test user
	owner := &User{Username: "owner", Password: hashPasswordForAuditTest("password123")}
	require.NoError(t, db.Create(owner).Error)
	
	// Create test channel
	channel := &Channel{
		Name:      "test-channel",
		IsVisible: true,
		OwnerID:   owner.ID,
	}
	require.NoError(t, db.Create(channel).Error)
	
	// Create multiple audit logs
	for i := 0; i < 5; i++ {
		log := AuditLog{
			Action:      "TEST_ACTION",
			ActorID:     owner.ID,
			ChannelID:   &channel.ID,
			Description: fmt.Sprintf("Test action %d", i),
			Metadata:    "{}",
		}
		require.NoError(t, db.Create(&log).Error)
	}
	
	// Setup handler
	ah := NewAuditHandlers(db)
	
	// Create request with pagination
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/channels/%s/audit?page=1&limit=3", channel.ID), nil)
	c.Params = gin.Params{{Key: "id", Value: channel.ID}}
	c.Set("user_id", owner.ID)
	
	// Call handler
	ah.GetChannelAuditLogsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response AuditLogsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, int64(5), response.Total)
	assert.Len(t, response.Logs, 3)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 3, response.Limit)
}

func TestAuditHandlers_GetAuditLogsHandler_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Create test users
	user1 := &User{Username: "user1", Password: hashPasswordForAuditTest("password123")}
	user2 := &User{Username: "user2", Password: hashPasswordForAuditTest("password123")}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)
	
	// Create test channels
	channel1 := &Channel{Name: "channel1", IsVisible: true, OwnerID: user1.ID}
	channel2 := &Channel{Name: "channel2", IsVisible: true, OwnerID: user1.ID}
	require.NoError(t, db.Create(channel1).Error)
	require.NoError(t, db.Create(channel2).Error)
	
	// Create audit logs for different channels and users
	logs := []AuditLog{
		{Action: "CREATE_CHANNEL", ActorID: user1.ID, ChannelID: &channel1.ID, Description: "Created channel1", Metadata: "{}"},
		{Action: "CREATE_CHANNEL", ActorID: user1.ID, ChannelID: &channel2.ID, Description: "Created channel2", Metadata: "{}"},
		{Action: "JOIN_CHANNEL", ActorID: user2.ID, ChannelID: &channel1.ID, Description: "Joined channel1", Metadata: "{}"},
		{Action: "LEAVE_CHANNEL", ActorID: user2.ID, ChannelID: &channel1.ID, Description: "Left channel1", Metadata: "{}"},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(&log).Error)
	}
	
	// Setup handler
	ah := NewAuditHandlers(db)
	
	// Test channel filter
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/audit?channel_id=%s", channel1.ID), nil)
	c.Set("user_id", user1.ID)
	
	ah.GetAuditLogsHandler(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response AuditLogsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, int64(3), response.Total) // 3 logs for channel1
	assert.Len(t, response.Logs, 3)
	
	// Test action filter
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/audit?action=CREATE_CHANNEL", nil)
	c.Set("user_id", user1.ID)
	
	ah.GetAuditLogsHandler(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, int64(2), response.Total) // 2 CREATE_CHANNEL logs
	assert.Len(t, response.Logs, 2)
}

func TestAuditHandlers_GetAuditLogsHandler_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Setup handler
	ah := NewAuditHandlers(db)
	
	// Create request without auth
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/audit", nil)
	
	// Call handler
	ah.GetAuditLogsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not authenticated", response["error"])
}

func TestAuditHandlers_GetChannelAuditLogsHandler_MissingChannelID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	db := setupAuditTestDB(t)
	
	// Create test user
	user := &User{Username: "testuser", Password: hashPasswordForAuditTest("password123")}
	require.NoError(t, db.Create(user).Error)
	
	// Setup handler
	ah := NewAuditHandlers(db)
	
	// Create request without channel ID
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/channels//audit", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	c.Set("user_id", user.ID)
	
	// Call handler
	ah.GetChannelAuditLogsHandler(c)
	
	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Channel ID required", response["error"])
}