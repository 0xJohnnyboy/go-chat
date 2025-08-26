package api

import (
	a "go-chat/internal/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	ah *AuthHandlers
	am *a.AuthMiddleware
}

func NewRouter(db *gorm.DB) *Router {
	return &Router{
		ah: NewHandlers(db),
		am: a.NewAuthMiddleware(),
	}
}

func (r *Router) RegisterRoutes(router *gin.Engine) {
	{
		unprotected := router.Group("/")
		unprotected.GET("/hc", HealthCheckHandler)
		unprotected.POST("/register", r.ah.RegisterHandler)
		unprotected.POST("/login", r.ah.LoginHandler)
	}

	{
		protected := router.Group("/api")
		protected.Use(r.am.RequireAuth())
		protected.POST("/logout", r.ah.LogoutHandler)
		protected.POST("/refresh_token", r.ah.RefreshTokenHandler)
	}
}

func HealthCheckHandler(c *gin.Context) {
	c.String(200, "Running")
}
