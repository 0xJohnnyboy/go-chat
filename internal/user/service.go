package user

import (
	"errors"
	"fmt"

	"go-chat/pkg/chat"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" example:"new_username"`
	Password *string `json:"password,omitempty" example:"newPassword123"`
}

func (s *UserService) UpdateUser(userID string, req UpdateUserRequest) (*chat.User, error) {
	var user chat.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Username != nil {
		// Check if username already exists
		var existingUser chat.User
		result := s.db.First(&existingUser, "username = ? AND id != ?", *req.Username, userID)
		if result.Error == nil {
			return nil, errors.New("username already exists")
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check username uniqueness: %w", result.Error)
		}
		updates["username"] = *req.Username
	}

	if req.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		updates["password"] = string(hashedPassword)
	}

	if len(updates) == 0 {
		return &user, nil // No updates requested
	}

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Reload user to get updated data
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload user: %w", err)
	}

	return &user, nil
}

func (s *UserService) DeleteUser(userID string) error {
	var user chat.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Soft delete the user (GORM will handle setting DeletedAt)
	if err := s.db.Delete(&user).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Clean up refresh tokens
	if err := s.db.Where("user_id = ?", userID).Delete(&chat.RefreshToken{}).Error; err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to clean up refresh tokens for user %s: %v\n", userID, err)
	}

	return nil
}

func (s *UserService) GetOwnedChannels(userID string) ([]chat.Channel, error) {
	var channels []chat.Channel
	err := s.db.Preload("Owner").Where("owner_id = ?", userID).Find(&channels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get owned channels: %w", err)
	}
	return channels, nil
}

func (s *UserService) GetJoinedChannels(userID string) ([]chat.Channel, error) {
	var channels []chat.Channel
	
	err := s.db.
		Preload("Owner").
		Joins("JOIN user_channels ON channels.id = user_channels.channel_id").
		Where("user_channels.user_id = ?", userID).
		Find(&channels).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get joined channels: %w", err)
	}
	
	return channels, nil
}