package main

import "go-chat/internal/server"

func main() {
	_ := db.InitDB("gochat.db")
	server.Run()
}
