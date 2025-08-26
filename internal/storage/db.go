package storage

import (
	"log"

	. "go-chat/pkg/chat"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)
const (
	DBPath = "gochat.db"
)

func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(DBPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&User{},
		&RefreshToken{},
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
		if err == nil {
			continue
		}

		log.Printf("failed to find role %s: %v", role.Name, err)
		log.Printf("inserting role %s", role.Name)

		if err := db.Create(&role).Error; err != nil {
			log.Printf("failed to insert role %s: %v", role.Name, err)
		}
	}
}
