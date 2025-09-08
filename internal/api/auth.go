package api

import (
	"fmt"

	. "go-chat/internal/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandlers struct {
	authService *AuthService
}

func NewHandlers(db *gorm.DB) *AuthHandlers {
	return &AuthHandlers{
		authService: NewAuthService(db),
	}
}

type UserRegisterInput struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Password string `json:"password" binding:"required" example:"securePassword123"`
}

type UserResponse struct {
	ID       string `json:"id" example:"a1b2c3d4"`
	Username string `json:"username" example:"john_doe"`
}

type AuthResponse struct {
	Message string       `json:"message" example:"Register successful"`
	User    UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"username cannot be empty"`
}

// RegisterHandler registers a new user
// @Summary Register a new user
// @Description Register a new user with username and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body UserRegisterInput true "Registration request"
// @Success 200 {object} AuthResponse "User registered successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /register [post]
func (h *AuthHandlers) RegisterHandler(c *gin.Context) {
	var input UserRegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	user, err := h.authService.Register(input.Username, input.Password)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	token, err := GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": "User created but token generation failed"})
		return
	}

	refreshToken, err := h.authService.CreateRefreshToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "User created but refresh token generation failed"})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", true, true)
	c.SetCookie("refresh_token", refreshToken, 3600*24*7, "/", "", true, true)

	c.JSON(200, gin.H{
		"message": "Register successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

type UserLoginInput struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Password string `json:"password" binding:"required" example:"securePassword123"`
}

// LoginHandler authenticates a user
// @Summary Login user
// @Description Authenticate user with username and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body UserLoginInput true "Login request"
// @Success 200 {object} AuthResponse "User logged in successfully"
// @Failure 400 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /login [post]
func (h *AuthHandlers) LoginHandler(c *gin.Context) {
	var input UserLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	user, err := h.authService.Login(input.Username, input.Password)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	refreshToken, err := h.authService.CreateRefreshToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "User created but refresh token generation failed"})
		return
	}
	token, err := GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": "Token generation failed"})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", true, true)
	c.SetCookie("refresh_token", refreshToken, 3600*24*7, "/", "", true, true)

	c.JSON(200, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

type MessageResponse struct {
	Message string `json:"message" example:"Logged out"`
}

// LogoutHandler logs out the user
// @Summary Logout user
// @Description Logout user and clear authentication cookies
// @Tags Authentication
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} MessageResponse "User logged out successfully"
// @Router /api/logout [post]
func (h *AuthHandlers) LogoutHandler(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		h.authService.RevokeRefreshToken(refreshToken)
	}

	c.SetCookie("token", "", -1, "/", "", true, true)
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	c.JSON(200, gin.H{"message": "Logged out"})
}

// RefreshTokenHandler refreshes the JWT token
// @Summary Refresh JWT token
// @Description Refresh JWT token using refresh token from cookie
// @Tags Authentication
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} MessageResponse "Token refreshed successfully"
// @Failure 401 {object} ErrorResponse "Invalid or missing refresh token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/refresh_token [post]
func (h *AuthHandlers) RefreshTokenHandler(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")

	fmt.Println(refreshToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "No refresh token"})
		return
	}

	user, err := h.authService.ValidateRefreshToken(refreshToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid refresh token"})
		return
	}

	newJWT, err := GenerateToken(user.ID, user.Username)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	c.SetCookie("token", newJWT, 3600*24, "/", "", true, true)
	c.JSON(200, gin.H{"message": "Token refreshed"})
}
