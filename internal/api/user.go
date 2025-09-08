package api

import (
	"net/http"

	u "go-chat/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandlers struct {
	service *u.UserService
}

func NewUserHandlers(db *gorm.DB) *UserHandlers {
	return &UserHandlers{
		service: u.NewUserService(db),
	}
}

type UpdateUserResponse struct {
	Message string       `json:"message" example:"User updated successfully"`
	User    UserResponse `json:"user"`
}

// UpdateUserHandler updates user information
// @Summary Update user information
// @Description Update user username and/or password
// @Tags User Management
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body user.UpdateUserRequest true "Update user request"
// @Success 200 {object} UpdateUserResponse "User updated successfully"
// @Failure 400 {object} ErrorResponse "Bad request or username already exists"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/user [patch]
func (h *UserHandlers) UpdateUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req u.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateUser(userID.(string), req)
	if err != nil {
		if err.Error() == "username already exists" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// DeleteUserHandler deletes user account
// @Summary Delete user account
// @Description Soft delete user account and clear authentication cookies
// @Tags User Management
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} MessageResponse "Account deleted successfully"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/user [delete]
func (h *UserHandlers) DeleteUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := h.service.DeleteUser(userID.(string))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		}
		return
	}

	// Clear auth cookies
	c.SetCookie("token", "", -1, "/", "", true, true)
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

type ChannelOwner struct {
	ID       string `json:"id" example:"a1b2c3d4"`
	Username string `json:"username" example:"john_doe"`
}

type ChannelInfo struct {
	ID        string       `json:"id" example:"ch123"`
	Name      string       `json:"name" example:"general"`
	IsVisible bool         `json:"is_visible" example:"true"`
	CreatedAt string       `json:"created_at" example:"2023-01-01T00:00:00Z"`
	Owner     ChannelOwner `json:"owner"`
}

type ChannelsResponse struct {
	Channels []ChannelInfo `json:"channels"`
}

// GetOwnedChannelsHandler gets channels owned by user
// @Summary Get owned channels
// @Description Get all channels owned by the authenticated user
// @Tags User Management
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} ChannelsResponse "List of owned channels"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/user/channels/owned [get]
func (h *UserHandlers) GetOwnedChannelsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channels, err := h.service.GetOwnedChannels(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch owned channels"})
		return
	}

	var channelList []gin.H
	for _, channel := range channels {
		channelList = append(channelList, gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"created_at": channel.CreatedAt,
			"owner": gin.H{
				"id":       channel.Owner.ID,
				"username": channel.Owner.Username,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"channels": channelList})
}

// GetJoinedChannelsHandler gets channels joined by user
// @Summary Get joined channels
// @Description Get all channels the authenticated user has joined
// @Tags User Management
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} ChannelsResponse "List of joined channels"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/user/channels/joined [get]
func (h *UserHandlers) GetJoinedChannelsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channels, err := h.service.GetJoinedChannels(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch joined channels"})
		return
	}

	var channelList []gin.H
	for _, channel := range channels {
		channelList = append(channelList, gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"created_at": channel.CreatedAt,
			"owner": gin.H{
				"id":       channel.Owner.ID,
				"username": channel.Owner.Username,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"channels": channelList})
}