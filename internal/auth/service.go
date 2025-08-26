package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	. "go-chat/pkg/chat"
	. "go-chat/internal/utils"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(username, password string) (*User, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}

	hashedPassword, err := HashString(password)

	if err != nil {
		return nil, err
	}

	user := User{
		Username: username,
		Password: hashedPassword,
	}

	return &user, s.db.Create(&user).Error
}

func (s *AuthService) Login(username, password string) (*User, error) {
	var user User

	err := s.db.Where("username = ?", username).First(&user).Error

	if err != nil {
		return nil, err
	}

	if !VerifyHashedString(password, user.Password) {
		return nil, errors.New("invalid password")
	}

	return &user, nil
}

func (s *AuthService) CreateRefreshToken(userID string) (string, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	hash, err := HashString(token)
	if err != nil {
		return "", err
	}

	refreshToken := RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	if err := s.db.Create(&refreshToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) ValidateRefreshToken(token string) (*User, error) {
	var refreshTokens []RefreshToken
	if err := s.db.Where("expires_at > ?", time.Now().Unix()).Find(&refreshTokens).Error; err != nil {
		fmt.Println(err, "error in retrieval")
		return nil, err
	}

	for _, rt := range refreshTokens {
		isValid := VerifyHashedString(token, rt.TokenHash)
		fmt.Println(rt.TokenHash, token, isValid)
		if isValid {
			var user User
			fmt.Println(rt.UserID)
			if err := s.db.Where("id = ?", rt.UserID).First(&user).Error; err != nil {
				return nil, err
			}
			go s.db.Delete(&RefreshToken{}, "user_id = ? AND expires_at < ?", rt.UserID, time.Now())
			return &user, nil
		}
	}

	return nil, errors.New("invalid refresh token")
}

func (s *AuthService) RevokeRefreshToken(token string) error {
	var refreshTokens []RefreshToken
	s.db.Where("expires_at > ?", time.Now().Unix()).Find(&refreshTokens)

	for _, rt := range refreshTokens {
		if VerifyHashedString(token, rt.TokenHash) {
			return s.db.Delete(&rt).Error
		}
	}
	return nil
}
