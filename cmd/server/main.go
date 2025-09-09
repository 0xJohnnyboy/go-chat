// Package server provides the main entry point for the go-chat server
// @title Go Chat API
// @version 1.0
// @description A real-time chat server with JWT authentication, channel management, and WebSocket support
// @termsOfService http://swagger.io/terms/

// @contact.name Go Chat API Support
// @contact.url http://github.com/0xJohnnyboy/go-chat
// @contact.email contact@theolambert.com

// @license.name AGPL 3.0
// @license.url https://github.com/0xJohnnyboy/go-chat/blob/main/LICENSE.md 

// @host localhost:9876
// @BasePath /
// @schemes https

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name token
// @description JWT token stored in HTTP-only cookie
package main

import (
	"fmt"
	"os"

	api "go-chat/internal/api"
)

var port = ":9876"

func main() {
	if os.Args[0] == "-p" || os.Args[0] == "--port" {
		port = fmt.Sprintf(":%v", os.Args[1])
	}

	api.Serve(port)
}
