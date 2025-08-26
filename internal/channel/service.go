package channel

import (
	"errors"

	. "go-chat/internal/utils"
	. "go-chat/pkg/chat"
	"gorm.io/gorm"
)

type ChannelService struct {
	db *gorm.DB
}

func NewChannelService(db *gorm.DB) *ChannelService {
	return &ChannelService{db: db}
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

	return s.db.Create(&userChannel).Error
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

	return s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).Delete(&UserChannel{}).Error
}

func (s *ChannelService) DeleteChannel(userID, channelID string) error {
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return err
	}

	if channel.OwnerID != userID {
		return errors.New("only channel owner can delete channel")
	}

	// Delete all user-channel relationships first
	if err := s.db.Where("channel_id = ?", channelID).Delete(&UserChannel{}).Error; err != nil {
		return err
	}

	// Delete the channel
	return s.db.Delete(&Channel{}, "id = ?", channelID).Error
}

func (s *ChannelService) GetChannelUsers(channelID string) ([]User, error) {
	var users []User
	err := s.db.Joins("JOIN user_channels ON users.id = user_channels.user_id").
		Where("user_channels.channel_id = ?", channelID).
		Find(&users).Error
	return users, err
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

