package api

import (
	s "go-chat/internal/storage"
	"github.com/gin-gonic/gin"
)

var certFile = "cert.pem"
var keyFile = "key.pem"

func Serve(port string) error {
	r := gin.Default()

	db, err := s.Connect()

	if err != nil {
		panic(err)
	}

	router := NewRouter(db)
	router.RegisterRoutes(r)

	return r.RunTLS(port, certFile, keyFile)
}

