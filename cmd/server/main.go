package server

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
