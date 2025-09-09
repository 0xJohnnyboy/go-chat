package message

import (
	"errors"

	. "go-chat/pkg/chat"
	"gorm.io/gorm"
)

type MessageService struct {
	db *gorm.DB
}

func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

func (s *MessageService) GetChannelMessages(userID, channelID string, limit, offset int, beforeID string) ([]Message, int64, error) {
	// Check if channel exists
	var channel Channel
	if err := s.db.First(&channel, "id = ?", channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("channel not found")
		}
		return nil, 0, err
	}

	// Check if user is a member of the channel
	var userChannel UserChannel
	if err := s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("you are not a member of this channel")
		}
		return nil, 0, err
	}

	// Build query for messages
	query := s.db.Preload("User").Where("channel_id = ?", channelID)

	// Add before filter if specified
	if beforeID != "" {
		var beforeMessage Message
		if err := s.db.First(&beforeMessage, "id = ?", beforeID).Error; err == nil {
			query = query.Where("created_at < ?", beforeMessage.CreatedAt)
		}
	}

	// Get total count
	var total int64
	if err := query.Model(&Message{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get messages with pagination, ordered by most recent first
	var messages []Message
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}

	// Reverse the order to show oldest first (chronological order)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, total, nil
}

func (s *MessageService) CreateMessage(userID, channelID, content string) (*Message, error) {
	// Check if user is a member of the channel
	var userChannel UserChannel
	if err := s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("you are not a member of this channel")
		}
		return nil, err
	}

	// Create message
	message := Message{
		Content:   content,
		UserID:    userID,
		ChannelID: channelID,
	}

	if err := s.db.Create(&message).Error; err != nil {
		return nil, err
	}

	// Load user data
	if err := s.db.Preload("User").First(&message, message.ID).Error; err != nil {
		return nil, err
	}

	return &message, nil
}