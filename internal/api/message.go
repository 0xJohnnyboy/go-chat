package api

import (
	"net/http"
	"strconv"

	m "go-chat/internal/message"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageHandlers struct {
	service *m.MessageService
}

func NewMessageHandlers(db *gorm.DB) *MessageHandlers {
	return &MessageHandlers{
		service: m.NewMessageService(db),
	}
}

type MessageInfo struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	CreatedAt string `json:"created_at"`
	User      struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

type MessagesResponse struct {
	Messages []MessageInfo `json:"messages"`
	HasMore  bool          `json:"has_more,omitempty"`
	Total    int64         `json:"total,omitempty"`
}

// GetChannelMessagesHandler retrieves message history for a channel
// @Summary Get channel message history
// @Description Get paginated message history for a channel (only for channel members)
// @Tags Messages
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param limit query int false "Number of messages to retrieve (default: 50, max: 100)"
// @Param offset query int false "Number of messages to skip (default: 0)"
// @Param before query string false "Get messages before this message ID"
// @Success 200 {object} MessagesResponse "Messages retrieved successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "You are not a member of this channel"
// @Failure 404 {object} ErrorResponse "Channel not found"
// @Router /api/channels/{id}/messages [get]
func (h *MessageHandlers) GetChannelMessagesHandler(c *gin.Context) {
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

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	beforeID := c.Query("before")

	// Get messages
	messages, total, err := h.service.GetChannelMessages(userID.(string), channelID, limit, offset, beforeID)
	if err != nil {
		if err.Error() == "channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if err.Error() == "you are not a member of this channel" {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this channel"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	// Convert to response format
	var messageResponses []MessageInfo
	for _, msg := range messages {
		msgResponse := MessageInfo{
			ID:        msg.ID,
			Content:   msg.Content,
			UserID:    msg.UserID,
			ChannelID: msg.ChannelID,
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		msgResponse.User.ID = msg.User.ID
		msgResponse.User.Username = msg.User.Username
		messageResponses = append(messageResponses, msgResponse)
	}

	response := MessagesResponse{
		Messages: messageResponses,
		Total:    total,
		HasMore:  int64(offset+limit) < total,
	}

	c.JSON(http.StatusOK, response)
}