package api

import (
	a "go-chat/internal/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	ah *AuthHandlers
	ch *ChannelHandlers
	am *a.AuthMiddleware
}

func NewRouter(db *gorm.DB) *Router {
	return &Router{
		ah: NewHandlers(db),
		ch: NewChannelHandlers(db),
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

		// Channel endpoints
		protected.POST("/channels", r.ch.CreateChannelHandler)
		protected.GET("/channels", r.ch.GetChannelsHandler)
		protected.GET("/channels/me", r.ch.GetUserChannelsHandler)
		protected.GET("/channels/:id", r.ch.GetChannelHandler)
		protected.POST("/channels/:id/join", r.ch.JoinChannelHandler)
		protected.DELETE("/channels/:id/leave", r.ch.LeaveChannelHandler)
		protected.DELETE("/channels/:id", r.ch.DeleteChannelHandler)
		protected.GET("/channels/:id/users", r.ch.GetChannelUsersHandler)
		
		// Channel administration endpoints
		protected.POST("/channels/:id/ban", r.ch.BanUserHandler)
		protected.POST("/channels/:id/tempban", r.ch.TempBanUserHandler)
		protected.DELETE("/channels/:id/ban/:userId", r.ch.UnbanUserHandler)
		protected.GET("/channels/:id/bans", r.ch.GetChannelBansHandler)
	}
}

func HealthCheckHandler(c *gin.Context) {
	c.String(200, "Running")
}
