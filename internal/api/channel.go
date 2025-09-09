package api

import (
	"net/http"
	"time"

	c "go-chat/internal/channel"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelHandlers struct {
	service *c.ChannelService
}

func NewChannelHandlers(db *gorm.DB) *ChannelHandlers {
	return &ChannelHandlers{
		service: c.NewChannelService(db),
	}
}

type CreateChannelRequest struct {
	Name      string  `json:"name" binding:"required" example:"general"`
	Password  *string `json:"password,omitempty" example:"secretpass"`
	IsVisible bool    `json:"is_visible" example:"true"`
}

type JoinChannelRequest struct {
	Password *string `json:"password,omitempty" example:"secretpass"`
}

type ChannelResponse struct {
	Channel ChannelInfo `json:"channel"`
}

// CreateChannelHandler creates a new channel
// @Summary Create a new channel
// @Description Create a new channel with optional password protection
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body CreateChannelRequest true "Create channel request"
// @Success 201 {object} ChannelResponse "Channel created successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/channels [post]
func (h *ChannelHandlers) CreateChannelHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel, err := h.service.CreateChannel(userID.(string), req.Name, req.Password, req.IsVisible)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"channel": gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"owner_id":   channel.OwnerID,
		},
	})
}

// GetChannelsHandler gets all visible channels
// @Summary Get all visible channels
// @Description Get a list of all publicly visible channels
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} ChannelsResponse "List of visible channels"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/channels [get]
func (h *ChannelHandlers) GetChannelsHandler(c *gin.Context) {
	channels, err := h.service.GetVisibleChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
		return
	}

	var channelList []gin.H
	for _, channel := range channels {
		channelList = append(channelList, gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"owner": gin.H{
				"id":       channel.Owner.ID,
				"username": channel.Owner.Username,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"channels": channelList})
}

// GetUserChannelsHandler gets user's channels
// @Summary Get user's channels
// @Description Get all channels the authenticated user has joined
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} ChannelsResponse "List of user's channels"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/channels/me [get]
func (h *ChannelHandlers) GetUserChannelsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channels, err := h.service.GetUserChannels(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user channels"})
		return
	}

	var channelList []gin.H
	for _, channel := range channels {
		channelList = append(channelList, gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"owner": gin.H{
				"id":       channel.Owner.ID,
				"username": channel.Owner.Username,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"channels": channelList})
}

// GetChannelHandler gets a specific channel
// @Summary Get channel details
// @Description Get detailed information about a specific channel
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} ChannelResponse "Channel details"
// @Failure 400 {object} ErrorResponse "Channel ID required"
// @Failure 404 {object} ErrorResponse "Channel not found"
// @Router /api/channels/{id} [get]
func (h *ChannelHandlers) GetChannelHandler(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	channel, err := h.service.GetChannel(channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel": gin.H{
			"id":         channel.ID,
			"name":       channel.Name,
			"is_visible": channel.IsVisible,
			"owner": gin.H{
				"id":       channel.Owner.ID,
				"username": channel.Owner.Username,
			},
		},
	})
}

// JoinChannelHandler joins a channel
// @Summary Join a channel
// @Description Join a channel, optionally providing password for protected channels
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param request body JoinChannelRequest true "Join channel request"
// @Success 200 {object} MessageResponse "Successfully joined channel"
// @Failure 400 {object} ErrorResponse "Bad request or incorrect password"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/channels/{id}/join [post]
func (h *ChannelHandlers) JoinChannelHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	var req JoinChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.JoinChannel(userID.(string), channelID, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined channel"})
}

// LeaveChannelHandler leaves a channel
// @Summary Leave a channel
// @Description Leave a channel that the user has previously joined
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} MessageResponse "Successfully left channel"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/channels/{id}/leave [delete]
func (h *ChannelHandlers) LeaveChannelHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	err := h.service.LeaveChannel(userID.(string), channelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully left channel"})
}

// DeleteChannelHandler deletes a channel
// @Summary Delete a channel
// @Description Delete a channel (only channel owner can delete)
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} MessageResponse "Channel deleted successfully"
// @Failure 400 {object} ErrorResponse "Bad request or not authorized"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/channels/{id} [delete]
func (h *ChannelHandlers) DeleteChannelHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	err := h.service.DeleteChannel(userID.(string), channelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

type UserInfo struct {
	ID       string `json:"id" example:"a1b2c3d4"`
	Username string `json:"username" example:"john_doe"`
}

type UsersResponse struct {
	Users []UserInfo `json:"users"`
}

// GetChannelUsersHandler gets channel users
// @Summary Get channel users
// @Description Get a list of all users in a specific channel
// @Tags Channels
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} UsersResponse "List of channel users"
// @Failure 400 {object} ErrorResponse "Channel ID required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/channels/{id}/users [get]
func (h *ChannelHandlers) GetChannelUsersHandler(c *gin.Context) {
	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	users, err := h.service.GetChannelUsers(channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel users"})
		return
	}

	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"id":       user.ID,
			"username": user.Username,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": userList})
}

// Channel Administration Handlers

type BanUserRequest struct {
	UserID string `json:"user_id" binding:"required" example:"a1b2c3d4"`
	Reason string `json:"reason" example:"spam"`
}

type TempBanUserRequest struct {
	UserID   string `json:"user_id" binding:"required" example:"a1b2c3d4"`
	Reason   string `json:"reason" example:"timeout"`
	Duration string `json:"duration" binding:"required" example:"24h"` // e.g., "24h", "30m"
}

// BanUserHandler permanently bans a user from a channel
// @Summary Ban user from channel
// @Description Permanently ban a user from a channel (only channel owner can ban)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param request body BanUserRequest true "Ban user request"
// @Success 200 {object} MessageResponse "User banned successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owner can ban users"
// @Router /api/channels/{id}/ban [post]
func (h *ChannelHandlers) BanUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	var req BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.BanUser(userID.(string), req.UserID, channelID, req.Reason)
	if err != nil {
		if err.Error() == "only channel owner can ban users" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User banned successfully"})
}

// TempBanUserHandler temporarily bans a user from a channel
// @Summary Temporarily ban user from channel
// @Description Temporarily ban a user from a channel for a specified duration (only channel owner can ban)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param request body TempBanUserRequest true "Temporary ban user request"
// @Success 200 {object} MessageResponse "User temporarily banned successfully"
// @Failure 400 {object} ErrorResponse "Bad request or invalid duration format"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owner can ban users"
// @Router /api/channels/{id}/tempban [post]
func (h *ChannelHandlers) TempBanUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	var req TempBanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse duration
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration format"})
		return
	}

	err = h.service.TempBanUser(userID.(string), req.UserID, channelID, req.Reason, duration)
	if err != nil {
		if err.Error() == "only channel owner can ban users" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User temporarily banned successfully"})
}

// UnbanUserHandler unbans a user from a channel
// @Summary Unban user from channel
// @Description Remove a ban from a user, allowing them to rejoin the channel (only channel owner can unban)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param userId path string true "User ID to unban"
// @Success 200 {object} MessageResponse "User unbanned successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owner can unban users"
// @Router /api/channels/{id}/ban/{userId} [delete]
func (h *ChannelHandlers) UnbanUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	err := h.service.UnbanUser(userID.(string), targetUserID, channelID)
	if err != nil {
		if err.Error() == "only channel owner can unban users" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unbanned successfully"})
}

type BanInfo struct {
	ID        uint      `json:"id" example:"1"`
	UserID    string    `json:"user_id" example:"a1b2c3d4"`
	Reason    string    `json:"reason" example:"spam"`
	BannedAt  string    `json:"banned_at" example:"2023-01-01T00:00:00Z"`
	ExpiresAt *string   `json:"expires_at" example:"2023-01-02T00:00:00Z"`
	IsActive  bool      `json:"is_active" example:"true"`
	User      UserInfo  `json:"user"`
	BannedBy  UserInfo  `json:"banned_by"`
}

type BansResponse struct {
	Bans []BanInfo `json:"bans"`
}

// GetChannelBansHandler gets all bans for a channel
// @Summary Get channel bans
// @Description Get a list of all active and inactive bans for a channel (only channel owner can view)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} BansResponse "List of channel bans"
// @Failure 400 {object} ErrorResponse "Channel ID required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owner can view bans"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/channels/{id}/bans [get]
func (h *ChannelHandlers) GetChannelBansHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID required"})
		return
	}

	bans, err := h.service.GetChannelBans(userID.(string), channelID)
	if err != nil {
		if err.Error() == "only channel owner can view bans" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bans"})
		}
		return
	}

	var banList []gin.H
	for _, ban := range bans {
		banData := gin.H{
			"id":         ban.ID,
			"user_id":    ban.UserID,
			"reason":     ban.Reason,
			"banned_at":  ban.CreatedAt,
			"expires_at": ban.ExpiresAt,
			"is_active":  ban.IsActive,
			"user": gin.H{
				"id":       ban.User.ID,
				"username": ban.User.Username,
			},
			"banned_by": gin.H{
				"id":       ban.BannedByUser.ID,
				"username": ban.BannedByUser.Username,
			},
		}
		banList = append(banList, banData)
	}

	c.JSON(http.StatusOK, gin.H{"bans": banList})
}

type RoleUpdateRequest struct {
	UserID string `json:"user_id" binding:"required" example:"abc12345"`
	Role   string `json:"role" binding:"required" example:"Moderator"`
}

// PromoteUserHandler promotes a user in a channel
// @Summary Promote user in channel
// @Description Promote a user to a higher role in the channel (only channel owners can promote)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param request body RoleUpdateRequest true "Role update request"
// @Success 200 {object} MessageResponse "User promoted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owners can promote users"
// @Failure 404 {object} ErrorResponse "Channel or user not found"
// @Router /api/channels/{id}/promote [post]
func (h *ChannelHandlers) PromoteUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID is required"})
		return
	}

	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.PromoteUser(userID.(string), channelID, req.UserID, req.Role)
	if err != nil {
		if err.Error() == "channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if err.Error() == "only channel owners can promote users" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only channel owners can promote users"})
			return
		}
		if err.Error() == "user not found in channel" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found in channel"})
			return
		}
		if err.Error() == "role not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User promoted successfully"})
}

// DemoteUserHandler demotes a user in a channel
// @Summary Demote user in channel
// @Description Demote a user to a lower role in the channel (only channel owners can demote)
// @Tags Channel Administration
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param request body RoleUpdateRequest true "Role update request"
// @Success 200 {object} MessageResponse "User demoted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owners can demote users"
// @Failure 404 {object} ErrorResponse "Channel or user not found"
// @Router /api/channels/{id}/demote [post]
func (h *ChannelHandlers) DemoteUserHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID is required"})
		return
	}

	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.DemoteUser(userID.(string), channelID, req.UserID, req.Role)
	if err != nil {
		if err.Error() == "channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if err.Error() == "only channel owners can demote users" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only channel owners can demote users"})
			return
		}
		if err.Error() == "user not found in channel" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found in channel"})
			return
		}
		if err.Error() == "role not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User demoted successfully"})
}