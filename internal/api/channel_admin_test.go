package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	c "go-chat/internal/channel"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupChannelAdminRouter(t *testing.T) (*gin.Engine, *gorm.DB, *AuthHandlers, *ChannelHandlers) {
	gin.SetMode(gin.TestMode)
	
	db := setupTestDB(t)
	router := NewRouter(db)
	
	r := gin.New()
	router.RegisterRoutes(r)
	
	return r, db, router.ah, router.ch
}

func createTestUserWithAuth(t *testing.T, router *gin.Engine, username, password string) (string, string) {
	// Register user
	registerReq := UserRegisterInput{
		Username: username,
		Password: password,
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create test user %s: %d", username, w.Code)
	}

	// Extract token from cookies
	var token string
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "token" {
			token = cookie.Value
			break
		}
	}

	if token == "" {
		t.Fatalf("No token found for user %s", username)
	}

	// Extract user ID from response
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	user := response["user"].(map[string]interface{})
	userID := user["id"].(string)

	return userID, token
}

func TestChannelHandlers_BanUserHandler(t *testing.T) {
	router, db, _, _ := setupChannelAdminRouter(t)

	// Create test users
	ownerID, ownerToken := createTestUserWithAuth(t, router, "owner", "password")
	userID, _ := createTestUserWithAuth(t, router, "user", "password")
	_, nonOwnerToken := createTestUserWithAuth(t, router, "nonowner", "password")

	// Create channel and add user
	channelService := c.NewChannelService(db)
	channel, err := channelService.CreateChannel(ownerID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	err = channelService.JoinChannel(userID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	type BanUserRequest struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	}

	tests := []struct {
		name           string
		channelID      string
		token          string
		requestBody    BanUserRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:      "owner bans user successfully",
			channelID: channel.ID,
			token:     ownerToken,
			requestBody: BanUserRequest{
				UserID: userID,
				Reason: "spam",
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "User banned successfully" {
					t.Errorf("Expected success message, got: %v", response["message"])
				}
			},
		},
		{
			name:      "non-owner tries to ban user",
			channelID: channel.ID,
			token:     nonOwnerToken,
			requestBody: BanUserRequest{
				UserID: ownerID,
				Reason: "test",
			},
			expectedStatus: 403,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "only channel owner can ban users" {
					t.Errorf("Expected permission error, got: %v", response["error"])
				}
			},
		},
		{
			name:      "ban with missing user_id",
			channelID: channel.ID,
			token:     ownerToken,
			requestBody: BanUserRequest{
				Reason: "test",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] == nil {
					t.Errorf("Expected validation error for missing user_id")
				}
			},
		},
		{
			name:           "unauthenticated request",
			channelID:      channel.ID,
			token:          "",
			requestBody:    BanUserRequest{UserID: userID, Reason: "test"},
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest("POST", "/api/channels/"+tt.channelID+"/ban", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: tt.token,
				})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestChannelHandlers_TempBanUserHandler(t *testing.T) {
	router, db, _, _ := setupChannelAdminRouter(t)

	// Create test users
	ownerID, ownerToken := createTestUserWithAuth(t, router, "owner", "password")
	userID, _ := createTestUserWithAuth(t, router, "user", "password")

	// Create channel and add user
	channelService := c.NewChannelService(db)
	channel, err := channelService.CreateChannel(ownerID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	err = channelService.JoinChannel(userID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	type TempBanUserRequest struct {
		UserID   string `json:"user_id"`
		Reason   string `json:"reason"`
		Duration string `json:"duration"` // e.g., "24h", "30m"
	}

	tests := []struct {
		name           string
		channelID      string
		token          string
		requestBody    TempBanUserRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:      "owner temp bans user successfully",
			channelID: channel.ID,
			token:     ownerToken,
			requestBody: TempBanUserRequest{
				UserID:   userID,
				Reason:   "timeout",
				Duration: "24h",
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "User temporarily banned successfully" {
					t.Errorf("Expected success message, got: %v", response["message"])
				}
			},
		},
		{
			name:      "invalid duration format",
			channelID: channel.ID,
			token:     ownerToken,
			requestBody: TempBanUserRequest{
				UserID:   userID,
				Reason:   "test",
				Duration: "invalid",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] == nil {
					t.Errorf("Expected error for invalid duration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest("POST", "/api/channels/"+tt.channelID+"/tempban", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: tt.token,
				})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestChannelHandlers_UnbanUserHandler(t *testing.T) {
	router, db, _, _ := setupChannelAdminRouter(t)

	// Create test users
	ownerID, ownerToken := createTestUserWithAuth(t, router, "owner", "password")
	userID, _ := createTestUserWithAuth(t, router, "user", "password")

	// Create channel, add user, and ban them
	channelService := c.NewChannelService(db)
	channel, err := channelService.CreateChannel(ownerID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	err = channelService.JoinChannel(userID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	err = channelService.BanUser(ownerID, userID, channel.ID, "test ban")
	if err != nil {
		t.Fatalf("Failed to ban user: %v", err)
	}

	tests := []struct {
		name           string
		channelID      string
		userID         string
		token          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "owner unbans user successfully",
			channelID:      channel.ID,
			userID:         userID,
			token:          ownerToken,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "User unbanned successfully" {
					t.Errorf("Expected success message, got: %v", response["message"])
				}
			},
		},
		{
			name:           "unban non-existent user",
			channelID:      channel.ID,
			userID:         "nonexistent",
			token:          ownerToken,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/channels/"+tt.channelID+"/ban/"+tt.userID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: tt.token,
				})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestChannelHandlers_GetChannelBansHandler(t *testing.T) {
	router, db, _, _ := setupChannelAdminRouter(t)

	// Create test users
	ownerID, ownerToken := createTestUserWithAuth(t, router, "owner", "password")
	user1ID, _ := createTestUserWithAuth(t, router, "user1", "password")
	user2ID, _ := createTestUserWithAuth(t, router, "user2", "password")
	_, nonOwnerToken := createTestUserWithAuth(t, router, "nonowner", "password")

	// Create channel and add users
	channelService := c.NewChannelService(db)
	channel, err := channelService.CreateChannel(ownerID, "testchannel", nil, true)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	err = channelService.JoinChannel(user1ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user1 to channel: %v", err)
	}

	err = channelService.JoinChannel(user2ID, channel.ID, nil)
	if err != nil {
		t.Fatalf("Failed to add user2 to channel: %v", err)
	}

	// Ban users
	err = channelService.BanUser(ownerID, user1ID, channel.ID, "spam")
	if err != nil {
		t.Fatalf("Failed to ban user1: %v", err)
	}

	err = channelService.TempBanUser(ownerID, user2ID, channel.ID, "timeout", 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to temp ban user2: %v", err)
	}

	tests := []struct {
		name           string
		channelID      string
		token          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "owner gets channel bans",
			channelID:      channel.ID,
			token:          ownerToken,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				bans, ok := response["bans"].([]interface{})
				if !ok {
					t.Errorf("Expected bans array in response")
					return
				}
				
				if len(bans) != 2 {
					t.Errorf("Expected 2 bans, got %d", len(bans))
				}
			},
		},
		{
			name:           "non-owner tries to get bans",
			channelID:      channel.ID,
			token:          nonOwnerToken,
			expectedStatus: 403,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "only channel owner can view bans" {
					t.Errorf("Expected permission error, got: %v", response["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/channels/"+tt.channelID+"/bans", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.token != "" {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: tt.token,
				})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}