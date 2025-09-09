package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	a "go-chat/internal/audit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditHandlers struct {
	service *a.AuditService
}

func NewAuditHandlers(db *gorm.DB) *AuditHandlers {
	return &AuditHandlers{
		service: a.NewAuditService(db),
	}
}

type AuditLogResponse struct {
	ID          uint                   `json:"id" example:"1"`
	Action      string                 `json:"action" example:"BAN_USER"`
	ActorID     string                 `json:"actor_id" example:"abc12345"`
	TargetID    *string                `json:"target_id" example:"def67890"`
	ChannelID   *string                `json:"channel_id" example:"xyz123"`
	Description string                 `json:"description" example:"Banned user from channel"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   string                 `json:"created_at" example:"2023-01-01T00:00:00Z"`
	Actor       struct {
		ID       string `json:"id" example:"abc12345"`
		Username string `json:"username" example:"admin_user"`
	} `json:"actor"`
	Target *struct {
		ID       string `json:"id" example:"def67890"`
		Username string `json:"username" example:"banned_user"`
	} `json:"target,omitempty"`
	Channel *struct {
		ID   string `json:"id" example:"xyz123"`
		Name string `json:"name" example:"general"`
	} `json:"channel,omitempty"`
}

type AuditLogsResponse struct {
	Logs  []AuditLogResponse `json:"logs"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// GetChannelAuditLogsHandler gets audit logs for a specific channel
// @Summary Get channel audit logs
// @Description Get audit logs for a specific channel (only channel owners can view)
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "Channel ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of results per page (default: 20, max: 100)"
// @Success 200 {object} AuditLogsResponse "Audit logs retrieved successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Only channel owners can view audit logs"
// @Failure 404 {object} ErrorResponse "Channel not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/channels/{id}/audit [get]
func (h *AuditHandlers) GetChannelAuditLogsHandler(c *gin.Context) {
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

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Get audit logs for the channel
	logs, total, err := h.service.GetChannelAuditLogs(userID.(string), channelID, limit, offset)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	// Convert to response format
	var auditLogs []AuditLogResponse
	for _, log := range logs {
		auditLog := AuditLogResponse{
			ID:          log.ID,
			Action:      log.Action,
			ActorID:     log.ActorID,
			TargetID:    log.TargetID,
			ChannelID:   log.ChannelID,
			Description: log.Description,
			CreatedAt:   log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Parse metadata JSON
		if log.Metadata != "" {
			metadata := make(map[string]interface{})
			// Simple JSON unmarshaling - in production might want more robust parsing
			if err := parseMetadataJSON(log.Metadata, &metadata); err == nil {
				auditLog.Metadata = metadata
			} else {
				auditLog.Metadata = map[string]interface{}{}
			}
		} else {
			auditLog.Metadata = map[string]interface{}{}
		}

		// Set actor information
		auditLog.Actor.ID = log.Actor.ID
		auditLog.Actor.Username = log.Actor.Username

		// Set target information if exists
		if log.Target != nil {
			auditLog.Target = &struct {
				ID       string `json:"id" example:"def67890"`
				Username string `json:"username" example:"banned_user"`
			}{
				ID:       log.Target.ID,
				Username: log.Target.Username,
			}
		}

		// Set channel information if exists
		if log.Channel != nil {
			auditLog.Channel = &struct {
				ID   string `json:"id" example:"xyz123"`
				Name string `json:"name" example:"general"`
			}{
				ID:   log.Channel.ID,
				Name: log.Channel.Name,
			}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	response := AuditLogsResponse{
		Logs:  auditLogs,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	c.JSON(http.StatusOK, response)
}

// GetAuditLogsHandler gets audit logs with filtering options (admin only)
// @Summary Get audit logs with filtering
// @Description Get audit logs with optional filtering (system admin only)
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param channel_id query string false "Filter by channel ID"
// @Param actor_id query string false "Filter by actor ID"
// @Param action query string false "Filter by action type"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of results per page (default: 20, max: 100)"
// @Success 200 {object} AuditLogsResponse "Audit logs retrieved successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/audit [get]
func (h *AuditHandlers) GetAuditLogsHandler(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// TODO: Add admin role check when role system is implemented
	// For now, this endpoint is available to all authenticated users
	// In production, you would want to check if user has admin role

	// Parse filter parameters
	channelID := c.Query("channel_id")
	actorID := c.Query("actor_id")
	action := c.Query("action")

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Prepare filter pointers
	var channelFilter, actorFilter, actionFilter *string
	if channelID != "" {
		channelFilter = &channelID
	}
	if actorID != "" {
		actorFilter = &actorID
	}
	if action != "" {
		actionFilter = &action
	}

	// Get audit logs with filters
	logs, total, err := h.service.GetAuditLogs(channelFilter, actorFilter, actionFilter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	// Convert to response format (similar to channel audit logs)
	var auditLogs []AuditLogResponse
	for _, log := range logs {
		auditLog := AuditLogResponse{
			ID:          log.ID,
			Action:      log.Action,
			ActorID:     log.ActorID,
			TargetID:    log.TargetID,
			ChannelID:   log.ChannelID,
			Description: log.Description,
			CreatedAt:   log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Parse metadata JSON
		if log.Metadata != "" {
			metadata := make(map[string]interface{})
			if err := parseMetadataJSON(log.Metadata, &metadata); err == nil {
				auditLog.Metadata = metadata
			} else {
				auditLog.Metadata = map[string]interface{}{}
			}
		} else {
			auditLog.Metadata = map[string]interface{}{}
		}

		// Set actor information
		auditLog.Actor.ID = log.Actor.ID
		auditLog.Actor.Username = log.Actor.Username

		// Set target information if exists
		if log.Target != nil {
			auditLog.Target = &struct {
				ID       string `json:"id" example:"def67890"`
				Username string `json:"username" example:"banned_user"`
			}{
				ID:       log.Target.ID,
				Username: log.Target.Username,
			}
		}

		// Set channel information if exists
		if log.Channel != nil {
			auditLog.Channel = &struct {
				ID   string `json:"id" example:"xyz123"`
				Name string `json:"name" example:"general"`
			}{
				ID:   log.Channel.ID,
				Name: log.Channel.Name,
			}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	response := AuditLogsResponse{
		Logs:  auditLogs,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	c.JSON(http.StatusOK, response)
}

// Simple JSON metadata parser
func parseMetadataJSON(jsonStr string, metadata *map[string]interface{}) error {
	return json.Unmarshal([]byte(jsonStr), metadata)
}