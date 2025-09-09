package api

import (
	"net/http"
	"strconv"
	"strings"

	s "go-chat/internal/search"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchHandlers struct {
	service *s.SearchService
}

func NewSearchHandlers(db *gorm.DB) *SearchHandlers {
	return &SearchHandlers{
		service: s.NewSearchService(db),
	}
}

type UserSearchResult struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type UsersSearchResponse struct {
	Users []UserSearchResult `json:"users"`
	Total int64              `json:"total"`
}

type ChannelSearchResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsVisible bool   `json:"is_visible"`
	Owner     struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"owner"`
}

type ChannelsSearchResponse struct {
	Channels []ChannelSearchResult `json:"channels"`
	Total    int64                 `json:"total"`
}

type MessageSearchResult struct {
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

type MessagesSearchResponse struct {
	Messages []MessageSearchResult `json:"messages"`
	Total    int64                 `json:"total"`
}

// SearchUsersHandler searches for users by username
// @Summary Search users
// @Description Search for users by username (partial matching)
// @Tags Search
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param q query string true "Search query (minimum 2 characters)"
// @Param limit query int false "Number of results to return (default: 20, max: 50)"
// @Success 200 {object} UsersSearchResponse "Users found"
// @Failure 400 {object} ErrorResponse "Bad request - invalid query"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/search/users [get]
func (h *SearchHandlers) SearchUsersHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Search users
	users, total, err := h.service.SearchUsers(userID.(string), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
		return
	}

	// Convert to response format
	var userResults []UserSearchResult
	for _, user := range users {
		userResults = append(userResults, UserSearchResult{
			ID:       user.ID,
			Username: user.Username,
		})
	}

	response := UsersSearchResponse{
		Users: userResults,
		Total: total,
	}

	c.JSON(http.StatusOK, response)
}

// SearchChannelsHandler searches for channels by name
// @Summary Search channels
// @Description Search for visible channels by name (partial matching)
// @Tags Search
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param q query string true "Search query (minimum 2 characters)"
// @Param limit query int false "Number of results to return (default: 20, max: 50)"
// @Success 200 {object} ChannelsSearchResponse "Channels found"
// @Failure 400 {object} ErrorResponse "Bad request - invalid query"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /api/search/channels [get]
func (h *SearchHandlers) SearchChannelsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Search channels
	channels, total, err := h.service.SearchChannels(userID.(string), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search channels"})
		return
	}

	// Convert to response format
	var channelResults []ChannelSearchResult
	for _, channel := range channels {
		channelResult := ChannelSearchResult{
			ID:        channel.ID,
			Name:      channel.Name,
			IsVisible: channel.IsVisible,
		}
		channelResult.Owner.ID = channel.Owner.ID
		channelResult.Owner.Username = channel.Owner.Username
		channelResults = append(channelResults, channelResult)
	}

	response := ChannelsSearchResponse{
		Channels: channelResults,
		Total:    total,
	}

	c.JSON(http.StatusOK, response)
}

// SearchMessagesHandler searches for messages in a channel
// @Summary Search messages
// @Description Search for messages within a specific channel (only for channel members)
// @Tags Search
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param q query string true "Search query (minimum 2 characters)"
// @Param channel_id query string true "Channel ID to search within"
// @Param limit query int false "Number of results to return (default: 20, max: 50)"
// @Success 200 {object} MessagesSearchResponse "Messages found"
// @Failure 400 {object} ErrorResponse "Bad request - invalid query or channel_id"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "You are not a member of this channel"
// @Failure 404 {object} ErrorResponse "Channel not found"
// @Router /api/search/messages [get]
func (h *SearchHandlers) SearchMessagesHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	channelID := strings.TrimSpace(c.Query("channel_id"))
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel ID is required"})
		return
	}

	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Search messages
	messages, total, err := h.service.SearchMessages(userID.(string), channelID, query, limit)
	if err != nil {
		if err.Error() == "channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if err.Error() == "message history is disabled for this channel" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Message history is disabled for this channel"})
			return
		}
		if err.Error() == "you are not a member of this channel" {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this channel"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search messages"})
		return
	}

	// Convert to response format
	var messageResults []MessageSearchResult
	for _, message := range messages {
		messageResult := MessageSearchResult{
			ID:        message.ID,
			Content:   message.Content,
			UserID:    message.UserID,
			ChannelID: message.ChannelID,
			CreatedAt: message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		messageResult.User.ID = message.User.ID
		messageResult.User.Username = message.User.Username
		messageResults = append(messageResults, messageResult)
	}

	response := MessagesSearchResponse{
		Messages: messageResults,
		Total:    total,
	}

	c.JSON(http.StatusOK, response)
}