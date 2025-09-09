package audit

import (
	"encoding/json"
	"time"

	. "go-chat/pkg/chat"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Action constants for audit logging
const (
	ActionCreateChannel = "CREATE_CHANNEL"
	ActionDeleteChannel = "DELETE_CHANNEL"
	ActionBanUser       = "BAN_USER"
	ActionTempBanUser   = "TEMP_BAN_USER"
	ActionUnbanUser     = "UNBAN_USER"
	ActionPromoteUser   = "PROMOTE_USER"
	ActionDemoteUser    = "DEMOTE_USER"
	ActionJoinChannel   = "JOIN_CHANNEL"
	ActionLeaveChannel  = "LEAVE_CHANNEL"
)

type AuditMetadata struct {
	Reason    string    `json:"reason,omitempty"`
	Duration  string    `json:"duration,omitempty"`
	OldRole   string    `json:"old_role,omitempty"`
	NewRole   string    `json:"new_role,omitempty"`
	ExpiresAt *string   `json:"expires_at,omitempty"`
	IsTemp    bool      `json:"is_temp,omitempty"`
	Password  bool      `json:"password_protected,omitempty"`
}

// LogChannelCreation logs when a channel is created
func (s *AuditService) LogChannelCreation(actorID, channelID, channelName string, isVisible bool, hasPassword bool) error {
	metadata := AuditMetadata{
		Password: hasPassword,
	}
	metadataJSON, _ := json.Marshal(metadata)

	auditLog := AuditLog{
		Action:      ActionCreateChannel,
		ActorID:     actorID,
		ChannelID:   &channelID,
		Description: "Created channel '" + channelName + "'",
		Metadata:    string(metadataJSON),
	}

	return s.db.Create(&auditLog).Error
}

// LogChannelDeletion logs when a channel is deleted
func (s *AuditService) LogChannelDeletion(actorID, channelID, channelName string) error {
	auditLog := AuditLog{
		Action:      ActionDeleteChannel,
		ActorID:     actorID,
		ChannelID:   &channelID,
		Description: "Deleted channel '" + channelName + "'",
		Metadata:    "{}",
	}

	return s.db.Create(&auditLog).Error
}

// LogUserBan logs when a user is banned (permanently or temporarily)
func (s *AuditService) LogUserBan(actorID, targetID, channelID, reason string, isTemp bool, expiresAt *time.Time) error {
	action := ActionBanUser
	description := "Permanently banned user"
	
	if isTemp {
		action = ActionTempBanUser
		description = "Temporarily banned user"
	}

	metadata := AuditMetadata{
		Reason: reason,
		IsTemp: isTemp,
	}
	
	if expiresAt != nil {
		expiresAtStr := expiresAt.Format(time.RFC3339)
		metadata.ExpiresAt = &expiresAtStr
		metadata.Duration = time.Until(*expiresAt).String()
	}

	metadataJSON, _ := json.Marshal(metadata)

	auditLog := AuditLog{
		Action:      action,
		ActorID:     actorID,
		TargetID:    &targetID,
		ChannelID:   &channelID,
		Description: description,
		Metadata:    string(metadataJSON),
	}

	return s.db.Create(&auditLog).Error
}

// LogUserUnban logs when a user is unbanned
func (s *AuditService) LogUserUnban(actorID, targetID, channelID string) error {
	auditLog := AuditLog{
		Action:      ActionUnbanUser,
		ActorID:     actorID,
		TargetID:    &targetID,
		ChannelID:   &channelID,
		Description: "Unbanned user",
		Metadata:    "{}",
	}

	return s.db.Create(&auditLog).Error
}

// LogUserRoleChange logs when a user's role is changed (promote/demote)
func (s *AuditService) LogUserRoleChange(actorID, targetID, channelID, oldRole, newRole string, isPromotion bool) error {
	action := ActionPromoteUser
	description := "Promoted user"
	
	if !isPromotion {
		action = ActionDemoteUser
		description = "Demoted user"
	}

	metadata := AuditMetadata{
		OldRole: oldRole,
		NewRole: newRole,
	}
	metadataJSON, _ := json.Marshal(metadata)

	auditLog := AuditLog{
		Action:      action,
		ActorID:     actorID,
		TargetID:    &targetID,
		ChannelID:   &channelID,
		Description: description + " from " + oldRole + " to " + newRole,
		Metadata:    string(metadataJSON),
	}

	return s.db.Create(&auditLog).Error
}

// LogChannelJoin logs when a user joins a channel
func (s *AuditService) LogChannelJoin(userID, channelID, channelName string) error {
	auditLog := AuditLog{
		Action:      ActionJoinChannel,
		ActorID:     userID,
		ChannelID:   &channelID,
		Description: "Joined channel '" + channelName + "'",
		Metadata:    "{}",
	}

	return s.db.Create(&auditLog).Error
}

// LogChannelLeave logs when a user leaves a channel
func (s *AuditService) LogChannelLeave(userID, channelID, channelName string) error {
	auditLog := AuditLog{
		Action:      ActionLeaveChannel,
		ActorID:     userID,
		ChannelID:   &channelID,
		Description: "Left channel '" + channelName + "'",
		Metadata:    "{}",
	}

	return s.db.Create(&auditLog).Error
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (s *AuditService) GetAuditLogs(channelID *string, actorID *string, action *string, limit, offset int) ([]AuditLog, int64, error) {
	query := s.db.Model(&AuditLog{}).
		Preload("Actor").
		Preload("Target").
		Preload("Channel")

	// Apply filters
	if channelID != nil {
		query = query.Where("channel_id = ?", *channelID)
	}
	if actorID != nil {
		query = query.Where("actor_id = ?", *actorID)
	}
	if action != nil {
		query = query.Where("action = ?", *action)
	}

	// Get total count
	var total int64
	countQuery := *query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get logs with pagination
	var logs []AuditLog
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// GetChannelAuditLogs retrieves audit logs for a specific channel (owner only)
func (s *AuditService) GetChannelAuditLogs(requestorID, channelID string, limit, offset int) ([]AuditLog, int64, error) {
	// First check if requestor is channel owner
	var channel Channel
	if err := s.db.Where("id = ?", channelID).First(&channel).Error; err != nil {
		return nil, 0, err
	}

	if channel.OwnerID != requestorID {
		return nil, 0, gorm.ErrRecordNotFound // Or custom error
	}

	return s.GetAuditLogs(&channelID, nil, nil, limit, offset)
}