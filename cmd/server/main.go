package main

import (
	. "go-chat/internal/db"
	"go-chat/internal/server"
	"log"
)

func main() {
	_, err := InitDB("gochat.db")
	if err != nil {
		log.Fatal(err)
	}

	server.Run()
}
