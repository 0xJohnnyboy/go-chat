package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "go-chat/pkg/chat"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &RefreshToken{}, &Role{}, &Channel{}, &UserChannel{}, &UserBan{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	
	db := setupTestDB(t)
	handlers := NewHandlers(db)
	
	r := gin.New()
	r.POST("/register", handlers.RegisterHandler)
	r.POST("/login", handlers.LoginHandler)
	r.POST("/logout", handlers.LogoutHandler)
	r.POST("/refresh_token", handlers.RefreshTokenHandler)
	
	return r, db
}

func TestAuthHandlers_RegisterHandler(t *testing.T) {
	router, _ := setupRouter(t)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid registration",
			requestBody: UserRegisterInput{
				Username: "testuser",
				Password: "testpassword",
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "Register successful" {
					t.Errorf("Expected success message, got: %v", response["message"])
				}
				
				user, ok := response["user"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected user object in response")
					return
				}
				
				if user["username"] != "testuser" {
					t.Errorf("Expected username 'testuser', got: %v", user["username"])
				}
				
				if user["id"] == nil || user["id"] == "" {
					t.Errorf("Expected user ID to be set")
				}
			},
		},
		{
			name: "empty username",
			requestBody: UserRegisterInput{
				Username: "",
				Password: "testpassword",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "username cannot be empty" {
					t.Errorf("Expected empty username error, got: %v", response["error"])
				}
			},
		},
		{
			name: "empty password",
			requestBody: UserRegisterInput{
				Username: "testuser",
				Password: "",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "password cannot be empty" {
					t.Errorf("Expected empty password error, got: %v", response["error"])
				}
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] == nil {
					t.Errorf("Expected error for invalid JSON")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error
			
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}

			// Check cookies are set for successful registration
			if tt.expectedStatus == 200 {
				cookies := w.Result().Cookies()
				var tokenCookie, refreshCookie *http.Cookie
				
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						tokenCookie = cookie
					}
					if cookie.Name == "refresh_token" {
						refreshCookie = cookie
					}
				}
				
				if tokenCookie == nil {
					t.Errorf("Expected token cookie to be set")
				}
				if refreshCookie == nil {
					t.Errorf("Expected refresh_token cookie to be set")
				}
			}
		})
	}
}

func TestAuthHandlers_LoginHandler(t *testing.T) {
	router, _ := setupRouter(t)

	// Create a test user first
	registerReq := UserRegisterInput{
		Username: "testuser",
		Password: "testpassword",
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create test user: %d", w.Code)
	}

	tests := []struct {
		name           string
		requestBody    UserLoginInput
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid login",
			requestBody: UserLoginInput{
				Username: "testuser",
				Password: "testpassword",
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "Login successful" {
					t.Errorf("Expected success message, got: %v", response["message"])
				}
			},
		},
		{
			name: "invalid username",
			requestBody: UserLoginInput{
				Username: "nonexistent",
				Password: "testpassword",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] == nil {
					t.Errorf("Expected error for invalid username")
				}
			},
		},
		{
			name: "invalid password",
			requestBody: UserLoginInput{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "invalid password" {
					t.Errorf("Expected invalid password error, got: %v", response["error"])
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

			req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

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

func TestAuthHandlers_LogoutHandler(t *testing.T) {
	router, _ := setupRouter(t)

	req, err := http.NewRequest("POST", "/logout", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
		return
	}

	if response["message"] != "Logged out" {
		t.Errorf("Expected logout message, got: %v", response["message"])
	}

	// Check that cookies are cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "token" || cookie.Name == "refresh_token" {
			if cookie.MaxAge != -1 {
				t.Errorf("Expected cookie %s to be cleared (MaxAge = -1), got MaxAge = %d", cookie.Name, cookie.MaxAge)
			}
		}
	}
}

func TestAuthHandlers_RefreshTokenHandler(t *testing.T) {
	router, _ := setupRouter(t)

	// Create and login a test user to get refresh token
	registerReq := UserRegisterInput{
		Username: "testuser",
		Password: "testpassword",
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create test user: %d", w.Code)
	}

	// Extract refresh token from cookies
	var refreshToken string
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshToken = cookie.Value
			break
		}
	}

	if refreshToken == "" {
		t.Fatalf("No refresh token found in registration response")
	}

	tests := []struct {
		name           string
		setupCookie    func(req *http.Request)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid refresh token",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: refreshToken,
				})
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["message"] != "Token refreshed" {
					t.Errorf("Expected refresh message, got: %v", response["message"])
				}
			},
		},
		{
			name: "no refresh token",
			setupCookie: func(req *http.Request) {
				// Don't add any cookie
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "No refresh token" {
					t.Errorf("Expected no refresh token error, got: %v", response["error"])
				}
			},
		},
		{
			name: "invalid refresh token",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: "invalid-token",
				})
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				
				if response["error"] != "Invalid refresh token" {
					t.Errorf("Expected invalid refresh token error, got: %v", response["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/refresh_token", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.setupCookie != nil {
				tt.setupCookie(req)
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