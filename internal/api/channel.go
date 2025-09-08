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
	Name      string  `json:"name" binding:"required"`
	Password  *string `json:"password,omitempty"`
	IsVisible bool    `json:"is_visible"`
}

type JoinChannelRequest struct {
	Password *string `json:"password,omitempty"`
}

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
	UserID string `json:"user_id" binding:"required"`
	Reason string `json:"reason"`
}

type TempBanUserRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Reason   string `json:"reason"`
	Duration string `json:"duration" binding:"required"` // e.g., "24h", "30m"
}

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