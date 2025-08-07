package main

import (
	"log"
	. "go-chat/internal/db"
	"go-chat/internal/server"
)

func main() {
	_, err := InitDB("gochat.db")
	if err != nil {
		log.Fatal(err)
	}

	server.Run()
}
