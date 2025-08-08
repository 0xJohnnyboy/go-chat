package http

import (
    "fmt"
    "github.com/gin-gonic/gin"
)

func Serve(port int) error {
    r := gin.Default()
    RegisterRoutes(r)
    return r.Run(fmt.Sprintf(":%d", port))
}

func RegisterRoutes(r *gin.Engine){
}


