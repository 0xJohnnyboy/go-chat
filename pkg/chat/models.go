package chat

import (
	"time"
	"gorm.io/gorm"
	nanoid "github.com/matoous/go-nanoid/v2"
)

type User struct {
	ID string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name string `gorm:"not null"`
	Password *string
	IsGuest bool `gorm:"default:false"`

	IPs []UserIP `gorm:"constraint:OnDelete:SET NULL"`
	UserChannels []UserChannel
}

type Channel struct {
	ID string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name string `gorm:"uniqueIndex;not null"`
	IsVisible bool
	Password *string
	LoggingDays uint

	OwnerID string
	Owner   User `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	UserChannels []UserChannel
}

type UserIP struct {
	gorm.Model
    UserID    string `gorm:"not null"`
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
    IP        string
}

type UserChannel struct {
	gorm.Model

    UserID    string `gorm:"not null"`
    ChannelID string `gorm:"not null"`
	RoleID    *string

	User User `gorm:"foreignKey:UserID"`
	Channel Channel `gorm:"foreignKey:ChannelID"`
	Role Role `gorm:"foreignKey:RoleID"`
}

type Role struct {
	gorm.Model
	Name string `gorm:"uniqueIndex;not null"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID, err =  nanoid.New(8)
	return err
}

func (c *Channel) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err =  nanoid.New(6)
	return err
}
