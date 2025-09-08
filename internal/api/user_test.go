package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/internal/auth"
	. "go-chat/pkg/chat"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupUserTest() (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	
	db := setupTestDB(&testing.T{})
	
	router := gin.New()
	
	r := NewRouter(db)
	r.RegisterRoutes(router)
	
	return router, db
}

func createTestUserForUserTests(db *gorm.DB, username, password string) *User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &User{
		Username: username,
		Password: string(hashedPassword),
	}
	db.Create(user)
	return user
}

func createTestChannelForUserTests(db *gorm.DB, user *User, name string, isVisible bool) *Channel {
	channel := &Channel{
		Name:      name,
		IsVisible: isVisible,
		OwnerID:   user.ID,
	}
	db.Create(channel)
	return channel
}

func joinUserToChannel(db *gorm.DB, user *User, channel *Channel) {
	userChannel := &UserChannel{
		UserID:    user.ID,
		ChannelID: channel.ID,
	}
	db.Create(userChannel)
}

func getAuthTokenForUser(user *User) (string, error) {
	return auth.GenerateToken(user.ID, user.Username)
}

func TestUpdateUserEndpoint(t *testing.T) {
	router, db := setupUserTest()

	t.Run("should update user username successfully", func(t *testing.T) {
		user := createTestUserForUserTests(db, "testuser", "password123")
		token, _ := getAuthTokenForUser(user)

		updateData := map[string]interface{}{
			"username": "newusername",
		}
		jsonData, _ := json.Marshal(updateData)

		req := httptest.NewRequest("PATCH", "/api/user", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "User updated successfully", response["message"])
		assert.Equal(t, "newusername", response["user"].(map[string]interface{})["username"])
	})

	t.Run("should update user password successfully", func(t *testing.T) {
		user := createTestUserForUserTests(db, "testuser2", "oldpassword")
		token, _ := getAuthTokenForUser(user)

		updateData := map[string]interface{}{
			"password": "newpassword123",
		}
		jsonData, _ := json.Marshal(updateData)

		req := httptest.NewRequest("PATCH", "/api/user", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "User updated successfully", response["message"])
	})

	t.Run("should fail with duplicate username", func(t *testing.T) {
		_ = createTestUserForUserTests(db, "existing", "password123")
		user2 := createTestUserForUserTests(db, "user2", "password123")
		token, _ := getAuthTokenForUser(user2)

		updateData := map[string]interface{}{
			"username": "existing",
		}
		jsonData, _ := json.Marshal(updateData)

		req := httptest.NewRequest("PATCH", "/api/user", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should require authentication", func(t *testing.T) {
		updateData := map[string]interface{}{
			"username": "newname",
		}
		jsonData, _ := json.Marshal(updateData)

		req := httptest.NewRequest("PATCH", "/api/user", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestDeleteUserEndpoint(t *testing.T) {
	router, db := setupUserTest()

	t.Run("should delete user account successfully", func(t *testing.T) {
		user := createTestUserForUserTests(db, "tobedeleted", "password123")
		token, _ := getAuthTokenForUser(user)

		req := httptest.NewRequest("DELETE", "/api/user", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Account deleted successfully", response["message"])

		// Verify user is soft deleted
		var deletedUser User
		result := db.Unscoped().First(&deletedUser, "id = ?", user.ID)
		assert.NoError(t, result.Error)
		assert.NotNil(t, deletedUser.DeletedAt)
	})

	t.Run("should require authentication", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/user", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetOwnedChannelsEndpoint(t *testing.T) {
	router, db := setupUserTest()

	t.Run("should return owned channels", func(t *testing.T) {
		user := createTestUserForUserTests(db, "channelowner", "password123")
		channel1 := createTestChannelForUserTests(db, user, "owned-channel-1", true)
		channel2 := createTestChannelForUserTests(db, user, "owned-channel-2", false)
		
		// Create a channel owned by someone else to ensure it's not returned
		otherUser := createTestUserForUserTests(db, "otheruser", "password123")
		createTestChannelForUserTests(db, otherUser, "not-owned", true)

		token, _ := getAuthTokenForUser(user)

		req := httptest.NewRequest("GET", "/api/user/channels/owned", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		channels := response["channels"].([]interface{})
		assert.Len(t, channels, 2)

		// Verify the channels are the ones owned by the user
		channelNames := make([]string, len(channels))
		for i, ch := range channels {
			channelData := ch.(map[string]interface{})
			channelNames[i] = channelData["name"].(string)
		}
		assert.Contains(t, channelNames, channel1.Name)
		assert.Contains(t, channelNames, channel2.Name)
	})

	t.Run("should return empty array when no owned channels", func(t *testing.T) {
		user := createTestUserForUserTests(db, "noowner", "password123")
		token, _ := getAuthTokenForUser(user)

		req := httptest.NewRequest("GET", "/api/user/channels/owned", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		channels, exists := response["channels"]
		assert.True(t, exists)
		if channelArray, ok := channels.([]interface{}); ok {
			assert.Len(t, channelArray, 0)
		} else {
			assert.Nil(t, channels)
		}
	})

	t.Run("should require authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/channels/owned", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetJoinedChannelsEndpoint(t *testing.T) {
	router, db := setupUserTest()

	t.Run("should return joined channels", func(t *testing.T) {
		user := createTestUserForUserTests(db, "joiner", "password123")
		owner := createTestUserForUserTests(db, "owner", "password123")
		
		channel1 := createTestChannelForUserTests(db, owner, "joined-channel-1", true)
		channel2 := createTestChannelForUserTests(db, owner, "joined-channel-2", false)
		
		// Join user to these channels
		joinUserToChannel(db, user, channel1)
		joinUserToChannel(db, user, channel2)

		// Create a channel the user hasn't joined
		createTestChannelForUserTests(db, owner, "not-joined", true)

		token, _ := getAuthTokenForUser(user)

		req := httptest.NewRequest("GET", "/api/user/channels/joined", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		channels := response["channels"].([]interface{})
		assert.Len(t, channels, 2)

		// Verify the channels are the ones the user joined
		channelNames := make([]string, len(channels))
		for i, ch := range channels {
			channelData := ch.(map[string]interface{})
			channelNames[i] = channelData["name"].(string)
		}
		assert.Contains(t, channelNames, channel1.Name)
		assert.Contains(t, channelNames, channel2.Name)
	})

	t.Run("should return empty array when no joined channels", func(t *testing.T) {
		user := createTestUserForUserTests(db, "nojoiner", "password123")
		token, _ := getAuthTokenForUser(user)

		req := httptest.NewRequest("GET", "/api/user/channels/joined", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		channels, exists := response["channels"]
		assert.True(t, exists)
		if channelArray, ok := channels.([]interface{}); ok {
			assert.Len(t, channelArray, 0)
		} else {
			assert.Nil(t, channels)
		}
	})

	t.Run("should require authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/channels/joined", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}