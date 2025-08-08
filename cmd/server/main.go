package server

import (
	. "go-chat/internal/storage"
	"log"
)

func main() {
	db, err := GetDB()
	if err != nil {
		log.Fatal(err)
	}
}
