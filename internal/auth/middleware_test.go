package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestMain(m *testing.M) {
	// Set test secret for JWT tokens
	os.Setenv("APP_SECRET", "test-secret-key-for-testing")
	gin.SetMode(gin.TestMode)
	
	code := m.Run()
	
	// Clean up
	os.Unsetenv("APP_SECRET")
	os.Exit(code)
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		username string
		wantErr  bool
	}{
		{
			name:     "valid token generation",
			userID:   "user123",
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "empty userID",
			userID:   "",
			username: "testuser",
			wantErr:  false, // Should still work
		},
		{
			name:     "empty username",
			userID:   "user123",
			username: "",
			wantErr:  false, // Should still work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.username)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateToken() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateToken() unexpected error: %v", err)
				return
			}

			if token == "" {
				t.Errorf("GenerateToken() returned empty token")
			}

			// Verify token can be parsed
			parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				return []byte("test-secret-key-for-testing"), nil
			})

			if err != nil {
				t.Errorf("Generated token cannot be parsed: %v", err)
				return
			}

			if !parsedToken.Valid {
				t.Errorf("Generated token is not valid")
				return
			}

			claims, ok := parsedToken.Claims.(jwt.MapClaims)
			if !ok {
				t.Errorf("Token claims are not MapClaims")
				return
			}

			if claims["user_id"] != tt.userID {
				t.Errorf("Expected user_id '%s', got '%v'", tt.userID, claims["user_id"])
			}

			if claims["username"] != tt.username {
				t.Errorf("Expected username '%s', got '%v'", tt.username, claims["username"])
			}

			// Check expiration time
			exp, ok := claims["exp"].(float64)
			if !ok {
				t.Errorf("Token expiration time not found or not float64")
				return
			}

			if exp <= float64(time.Now().Unix()) {
				t.Errorf("Token should not be expired immediately after generation")
			}

			// Check issued at time
			iat, ok := claims["iat"].(float64)
			if !ok {
				t.Errorf("Token issued at time not found or not float64")
				return
			}

			if iat > float64(time.Now().Unix()) {
				t.Errorf("Token issued at time should not be in the future")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	// Generate a test token
	userID := "test123"
	username := "testuser"
	validToken, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Generate an expired token
	expiredClaims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte("test-secret-key-for-testing"))

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		checkClaims func(t *testing.T, claims jwt.MapClaims)
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			wantErr:     false,
			checkClaims: func(t *testing.T, claims jwt.MapClaims) {
				if claims["user_id"] != userID {
					t.Errorf("Expected user_id '%s', got '%v'", userID, claims["user_id"])
				}
				if claims["username"] != username {
					t.Errorf("Expected username '%s', got '%v'", username, claims["username"])
				}
			},
		},
		{
			name:        "expired token",
			tokenString: expiredTokenString,
			wantErr:     true,
		},
		{
			name:        "invalid token format",
			tokenString: "invalid.token.format",
			wantErr:     true,
		},
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     true,
		},
		{
			name:        "token with wrong secret",
			tokenString: func() string {
				claims := jwt.MapClaims{
					"user_id":  userID,
					"username": username,
					"exp":      time.Now().Add(24 * time.Hour).Unix(),
					"iat":      time.Now().Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				return tokenString
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.tokenString)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateToken() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateToken() unexpected error: %v", err)
				return
			}

			if claims == nil {
				t.Errorf("ValidateToken() returned nil claims")
				return
			}

			if tt.checkClaims != nil {
				tt.checkClaims(t, claims)
			}
		})
	}
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	middleware := NewAuthMiddleware()
	
	// Generate a valid token
	validToken, err := GenerateToken("123", "testuser")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		expectedStatus int
		checkContext   func(t *testing.T, c *gin.Context)
	}{
		{
			name: "valid token in cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: validToken,
				})
			},
			expectedStatus: 200,
			checkContext: func(t *testing.T, c *gin.Context) {
				userID, exists := c.Get("user_id")
				if !exists {
					t.Errorf("Expected user_id to be set in context")
					return
				}
				
				if userID != "123" {
					t.Errorf("Expected user_id '123', got %v", userID)
				}

				username, exists := c.Get("username")
				if !exists {
					t.Errorf("Expected username to be set in context")
					return
				}
				
				if username != "testuser" {
					t.Errorf("Expected username 'testuser', got %v", username)
				}
			},
		},
		{
			name: "missing token cookie",
			setupRequest: func(req *http.Request) {
				// Don't add any cookie
			},
			expectedStatus: 401,
		},
		{
			name: "invalid token in cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: "invalid-token",
				})
			},
			expectedStatus: 401,
		},
		{
			name: "empty token in cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: "",
				})
			},
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}
			c.Request = req

			// Create a test handler that will run after middleware
			var contextChecked bool
			testHandler := func(c *gin.Context) {
				if tt.checkContext != nil {
					tt.checkContext(t, c)
				}
				contextChecked = true
				c.JSON(200, gin.H{"message": "success"})
			}

			// Create a handler chain
			handlers := []gin.HandlerFunc{middleware.RequireAuth(), gin.HandlerFunc(testHandler)}
			
			// Run the middleware chain
			for _, handler := range handlers {
				if c.IsAborted() {
					break
				}
				handler(c)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// For successful cases, check that context was actually checked
			if tt.expectedStatus == 200 && !contextChecked {
				t.Errorf("Test handler was not called for successful case")
			}

			// For error cases, check that middleware aborted and context was not checked
			if tt.expectedStatus != 200 {
				if !c.IsAborted() {
					t.Errorf("Expected middleware to abort for error case")
				}
				if contextChecked {
					t.Errorf("Test handler should not be called for error case")
				}
			}
		})
	}
}

func TestAuthMiddleware_Integration(t *testing.T) {
	// Test the middleware in a real Gin router
	router := gin.New()
	middleware := NewAuthMiddleware()
	
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		
		c.JSON(200, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	// Generate a valid token
	validToken, err := GenerateToken("456", "integrationuser")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Test with valid token
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: validToken,
	})
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test without token
	req2, _ := http.NewRequest("GET", "/protected", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != 401 {
		t.Errorf("Expected status 401, got %d", w2.Code)
	}
}