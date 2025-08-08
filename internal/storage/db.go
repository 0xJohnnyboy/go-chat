package storage

import (
	"log"
	"errors"

	. "go-chat/pkg/chat"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&User{},
		&UserIP{},
		&Channel{},
		&Role{},
		&UserChannel{},
	)

	if err != nil {
		return nil, err
	}

	seedRoles(db)

	return db, nil
}

func seedRoles(db *gorm.DB) {
	roles := []Role{
		{Name: "Administrator"},
		{Name: "Moderator"},
		{Name: "Member"},
		{Name: "Guest"},
	}

	for _, role := range roles {
		var existing Role
		err := db.First(&existing, "name = ?", role.Name).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&role).Error; err != nil {
				log.Printf("failed to insert role %s: %v", role.Name, err)
			}
		}
	}
}
