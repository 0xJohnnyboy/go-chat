package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(path string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&User{},
		&UserIP{},
		&Channel{},
		&Role{},
		&UserChannel{},
	)

	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	return db
}
