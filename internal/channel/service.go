package channel

import (
	"errors"
	"time"

	a "go-chat/internal/audit"
	. "go-chat/internal/utils"
	. "go-chat/pkg/chat"
	"gorm.io/gorm"
)

type ChannelService struct {
	db           *gorm.DB
	auditService *a.AuditService
}

func NewChannelService(db *gorm.DB) *ChannelService {
	return &ChannelService{
		db:           db,
		auditService: a.NewAuditService(db),
	}
}

func (s *ChannelService) CreateChannel(ownerID, name string, password *string, isVisible bool) (*Channel, error) {
	if name == "" {
		return nil, errors.New("channel name cannot be empty")
	}

	var hashedPassword *string
	if password != nil && *password != "" {
		hash, err := HashString(*password)
		if err != nil {
			return nil, err
		}
		hashedPassword = &hash
	}

	channel := Channel{
		Name:        name,
		OwnerID:     ownerID,
		Password:    hashedPassword,
		IsVisible:   isVisible,
		LoggingDays: 30, // default 30 days
	}

	if err := s.db.Create(&channel).Error; err != nil {
		return nil, err
	}

	// Add owner as administrator of the channel
	memberRole, err := s.getOrCreateRole("Administrator")
	if err != nil {
		return nil, err
	}

	userChannel := UserChannel{
		UserID:    ownerID,
		ChannelID: channel.ID,
		RoleID:    &memberRole.ID,
	}

	if err := s.db.Create(&userChannel).Error; err != nil {
		return nil, err
	}

	// Log channel creation
	hasPassword := password != nil && *password != ""
	if err := s.auditService.LogChannelCreation(ownerID, channel.ID, channel.Name, isVisible, hasPassword); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return &channel, nil
}

func (s *ChannelService) GetVisibleChannels() ([]Channel, error) {
	var channels []Channel
	err := s.db.Where("is_visible = ?", true).Preload("Owner").Find(&channels).Error
	return channels, err
}

func (s *ChannelService) GetUserChannels(userID string) ([]Channel, error) {
	var channels []Channel
	err := s.db.Joins("JOIN user_channels ON channels.id = user_channels.channel_id").
		Where("user_channels.user_id = ?", userID).
		Preload("Owner").
		Find(&channels).Error
	return channels, err
}

func (s *ChannelService) GetChannel(channelID string) (*Channel, error) {
	var channel Channel
	err := s.db.Preload("Owner").First(&channel, "id = ?", channelID).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *ChannelService) JoinChannel(userID, channelID string, password *string) error {
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	// Check if user is already in channel
	var existing UserChannel
	err = s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&existing).Error
	if err == nil {
		return errors.New("user already in channel")
	}

	// Check password if channel is password protected
	if channel.Password != nil {
		if password == nil || *password == "" {
			return errors.New("password required for this channel")
		}
		if !VerifyHashedString(*password, *channel.Password) {
			return errors.New("invalid password")
		}
	}

	// Get default member role
	memberRole, err := s.getOrCreateRole("Member")
	if err != nil {
		return err
	}

	userChannel := UserChannel{
		UserID:    userID,
		ChannelID: channelID,
		RoleID:    &memberRole.ID,
	}

	if err := s.db.Create(&userChannel).Error; err != nil {
		return err
	}

	// Log channel join
	if err := s.auditService.LogChannelJoin(userID, channelID, channel.Name); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) LeaveChannel(userID, channelID string) error {
	// Don't allow owner to leave their own channel
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}
	if channel.OwnerID == userID {
		return errors.New("channel owner cannot leave channel")
	}

	if err := s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).Delete(&UserChannel{}).Error; err != nil {
		return err
	}

	// Log channel leave
	if err := s.auditService.LogChannelLeave(userID, channelID, channel.Name); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) DeleteChannel(userID, channelID string) error {
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	if channel.OwnerID != userID {
		return errors.New("only channel owner can delete channel")
	}

	channelName := channel.Name // Store name before deletion

	// Delete all user-channel relationships first
	if err := s.db.Where("channel_id = ?", channelID).Delete(&UserChannel{}).Error; err != nil {
		return err
	}

	// Delete the channel
	if err := s.db.Delete(&Channel{}, "id = ?", channelID).Error; err != nil {
		return err
	}

	// Log channel deletion
	if err := s.auditService.LogChannelDeletion(userID, channelID, channelName); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) GetChannelUsers(channelID string) ([]User, error) {
	var users []User
	err := s.db.Joins("JOIN user_channels ON users.id = user_channels.user_id").
		Where("user_channels.channel_id = ?", channelID).
		Find(&users).Error
	return users, err
}

func (s *ChannelService) BanUser(adminID, userID, channelID, reason string) error {
	// Check if admin is the channel owner or has admin privileges
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	if channel.OwnerID != adminID {
		// TODO: Add role-based permission check for administrators/moderators
		return errors.New("only channel owner can ban users")
	}

	// Cannot ban yourself
	if adminID == userID {
		return errors.New("cannot ban yourself")
	}

	// Cannot ban the channel owner
	if channel.OwnerID == userID {
		return errors.New("cannot ban channel owner")
	}

	// Check if user is in the channel
	var userChannel UserChannel
	err = s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userChannel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user is not in this channel")
		}
		return err
	}

	// Check if user is already banned
	var existingBan UserBan
	err = s.db.Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).First(&existingBan).Error
	if err == nil {
		return errors.New("user is already banned")
	}

	// Create ban record
	ban := UserBan{
		UserID:    userID,
		ChannelID: channelID,
		BannedBy:  adminID,
		Reason:    reason,
		ExpiresAt: nil, // Permanent ban
		IsActive:  true,
	}

	if err := s.db.Create(&ban).Error; err != nil {
		return err
	}

	// Remove user from channel
	if err := s.db.Delete(&userChannel).Error; err != nil {
		return err
	}

	// Log user ban
	if err := s.auditService.LogUserBan(adminID, userID, channelID, reason, false, nil); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) TempBanUser(adminID, userID, channelID, reason string, duration time.Duration) error {
	// Check if admin is the channel owner or has admin privileges
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	if channel.OwnerID != adminID {
		// TODO: Add role-based permission check for administrators/moderators
		return errors.New("only channel owner can ban users")
	}

	// Cannot ban yourself
	if adminID == userID {
		return errors.New("cannot ban yourself")
	}

	// Cannot ban the channel owner
	if channel.OwnerID == userID {
		return errors.New("cannot ban channel owner")
	}

	// Check if user is in the channel
	var userChannel UserChannel
	err = s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userChannel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user is not in this channel")
		}
		return err
	}

	// Check if user is already banned
	var existingBan UserBan
	err = s.db.Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).First(&existingBan).Error
	if err == nil {
		return errors.New("user is already banned")
	}

	// Create temporary ban record
	expiresAt := time.Now().Add(duration)
	ban := UserBan{
		UserID:    userID,
		ChannelID: channelID,
		BannedBy:  adminID,
		Reason:    reason,
		ExpiresAt: &expiresAt,
		IsActive:  true,
	}

	if err := s.db.Create(&ban).Error; err != nil {
		return err
	}

	// Remove user from channel
	if err := s.db.Delete(&userChannel).Error; err != nil {
		return err
	}

	// Log temporary user ban
	if err := s.auditService.LogUserBan(adminID, userID, channelID, reason, true, &expiresAt); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) UnbanUser(adminID, userID, channelID string) error {
	// Check if admin is the channel owner or has admin privileges
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	if channel.OwnerID != adminID {
		// TODO: Add role-based permission check for administrators/moderators
		return errors.New("only channel owner can unban users")
	}

	// Find active ban
	var ban UserBan
	err = s.db.Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).First(&ban).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user is not banned")
		}
		return err
	}

	// Deactivate the ban
	ban.IsActive = false
	if err := s.db.Save(&ban).Error; err != nil {
		return err
	}

	// Log user unban
	if err := s.auditService.LogUserUnban(adminID, userID, channelID); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) IsUserBanned(userID, channelID string) (bool, error) {
	var ban UserBan
	err := s.db.Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).First(&ban).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	// Check if temporary ban has expired
	if ban.ExpiresAt != nil && time.Now().After(*ban.ExpiresAt) {
		// Automatically expire the ban
		ban.IsActive = false
		s.db.Save(&ban)
		return false, nil
	}

	return true, nil
}

func (s *ChannelService) GetChannelBans(adminID, channelID string) ([]UserBan, error) {
	// Check if admin is the channel owner or has admin privileges
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return nil, err
	}

	if channel.OwnerID != adminID {
		// TODO: Add role-based permission check for administrators/moderators
		return nil, errors.New("only channel owner can view bans")
	}

	var bans []UserBan
	err = s.db.Preload("User").Preload("BannedByUser").Where("channel_id = ? AND is_active = ?", channelID, true).Find(&bans).Error
	return bans, err
}

func (s *ChannelService) PromoteUser(requesterID, channelID, targetUserID, roleName string) error {
	// Check if channel exists
	var channel Channel
	if err := s.db.First(&channel, "id = ?", channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("channel not found")
		}
		return err
	}

	// Check if requester is the channel owner
	if channel.OwnerID != requesterID {
		return errors.New("only channel owners can promote users")
	}

	// Check if target user exists in the channel
	var userChannel UserChannel
	if err := s.db.Preload("Role").Where("user_id = ? AND channel_id = ?", targetUserID, channelID).First(&userChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found in channel")
		}
		return err
	}

	// Get current role name for audit logging
	oldRoleName := "Member" // default
	if userChannel.RoleID != nil {
		oldRoleName = userChannel.Role.Name
	}

	// Get the target role
	role, err := s.getOrCreateRole(roleName)
	if err != nil {
		return err
	}

	// Update user role
	userChannel.RoleID = &role.ID
	if err := s.db.Save(&userChannel).Error; err != nil {
		return err
	}

	// Log user promotion
	if err := s.auditService.LogUserRoleChange(requesterID, targetUserID, channelID, oldRoleName, roleName, true); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) DemoteUser(requesterID, channelID, targetUserID, roleName string) error {
	// Check if channel exists
	var channel Channel
	if err := s.db.First(&channel, "id = ?", channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("channel not found")
		}
		return err
	}

	// Check if requester is the channel owner
	if channel.OwnerID != requesterID {
		return errors.New("only channel owners can demote users")
	}

	// Check if target user exists in the channel
	var userChannel UserChannel
	if err := s.db.Preload("Role").Where("user_id = ? AND channel_id = ?", targetUserID, channelID).First(&userChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found in channel")
		}
		return err
	}

	// Get current role name for audit logging
	oldRoleName := "Member" // default
	if userChannel.RoleID != nil {
		oldRoleName = userChannel.Role.Name
	}

	// Get the target role
	role, err := s.getOrCreateRole(roleName)
	if err != nil {
		return err
	}

	// Update user role
	userChannel.RoleID = &role.ID
	if err := s.db.Save(&userChannel).Error; err != nil {
		return err
	}

	// Log user demotion
	if err := s.auditService.LogUserRoleChange(requesterID, targetUserID, channelID, oldRoleName, roleName, false); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

func (s *ChannelService) getOrCreateRole(roleName string) (*Role, error) {
	var role Role
	err := s.db.Where("name = ?", roleName).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			role = Role{Name: roleName}
			if err := s.db.Create(&role).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &role, nil
}

