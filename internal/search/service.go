package search

import (
	"errors"
	"strings"

	. "go-chat/pkg/chat"
	"gorm.io/gorm"
)

type SearchService struct {
	db *gorm.DB
}

func NewSearchService(db *gorm.DB) *SearchService {
	return &SearchService{db: db}
}

func (s *SearchService) SearchUsers(searcherID, query string, limit int) ([]User, int64, error) {
	// Clean query for SQL LIKE
	likeQuery := "%" + strings.ToLower(query) + "%"

	// Count total matching users (excluding the searcher)
	var total int64
	countQuery := s.db.Model(&User{}).Where("LOWER(username) LIKE ? AND id != ?", likeQuery, searcherID)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Find matching users (excluding the searcher)
	var users []User
	searchQuery := s.db.Where("LOWER(username) LIKE ? AND id != ?", likeQuery, searcherID).
		Order("username ASC").
		Limit(limit)

	if err := searchQuery.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *SearchService) SearchChannels(searcherID, query string, limit int) ([]Channel, int64, error) {
	// Clean query for SQL LIKE
	likeQuery := "%" + strings.ToLower(query) + "%"

	// Count total matching visible channels
	var total int64
	countQuery := s.db.Model(&Channel{}).Where("LOWER(name) LIKE ? AND is_visible = ?", likeQuery, true)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Find matching visible channels with owner information
	var channels []Channel
	searchQuery := s.db.Preload("Owner").
		Where("LOWER(name) LIKE ? AND is_visible = ?", likeQuery, true).
		Order("name ASC").
		Limit(limit)

	if err := searchQuery.Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

func (s *SearchService) SearchMessages(searcherID, channelID, query string, limit int) ([]Message, int64, error) {
	// Check if channel exists
	var channel Channel
	if err := s.db.First(&channel, "id = ?", channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("channel not found")
		}
		return nil, 0, err
	}

	// Check if channel has message history enabled
	if channel.LoggingDays == 0 {
		return nil, 0, errors.New("message history is disabled for this channel")
	}

	// Check if user is a member of the channel
	var userChannel UserChannel
	if err := s.db.Where("user_id = ? AND channel_id = ?", searcherID, channelID).First(&userChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("you are not a member of this channel")
		}
		return nil, 0, err
	}

	// Clean query for SQL LIKE
	likeQuery := "%" + strings.ToLower(query) + "%"

	// Count total matching messages in the channel
	var total int64
	countQuery := s.db.Model(&Message{}).Where("channel_id = ? AND LOWER(content) LIKE ?", channelID, likeQuery)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Find matching messages with user information
	var messages []Message
	searchQuery := s.db.Preload("User").
		Where("channel_id = ? AND LOWER(content) LIKE ?", channelID, likeQuery).
		Order("created_at DESC").
		Limit(limit)

	if err := searchQuery.Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}