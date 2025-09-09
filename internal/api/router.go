package api

import (
	a "go-chat/internal/auth"
	"go-chat/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	ah *AuthHandlers
	ch *ChannelHandlers
	uh *UserHandlers
	mh *MessageHandlers
	sh *SearchHandlers
	am *a.AuthMiddleware
	// Rate limiters for different endpoint types
	authRateLimit     *middleware.IPRateLimiter
	generalRateLimit  *middleware.IPRateLimiter
	readOnlyRateLimit *middleware.IPRateLimiter
}

func NewRouter(db *gorm.DB) *Router {
	return &Router{
		ah: NewHandlers(db),
		ch: NewChannelHandlers(db),
		uh: NewUserHandlers(db),
		mh: NewMessageHandlers(db),
		sh: NewSearchHandlers(db),
		am: a.NewAuthMiddleware(),
		// Initialize rate limiters with different configurations
		authRateLimit:     middleware.NewIPRateLimiter(middleware.StrictRateLimit),
		generalRateLimit:  middleware.NewIPRateLimiter(middleware.StandardRateLimit),
		readOnlyRateLimit: middleware.NewIPRateLimiter(middleware.LenientRateLimit),
	}
}

func (r *Router) RegisterRoutes(router *gin.Engine) {
	{
		// Health check with lenient rate limiting
		health := router.Group("/")
		health.Use(middleware.RateLimitMiddleware(r.readOnlyRateLimit))
		health.GET("/hc", HealthCheckHandler)
	}
	
	{
		// Authentication endpoints with strict rate limiting
		auth := router.Group("/")
		auth.Use(middleware.RateLimitMiddleware(r.authRateLimit))
		auth.POST("/register", r.ah.RegisterHandler)
		auth.POST("/login", r.ah.LoginHandler)
	}

	{
		// Authentication-related protected endpoints with strict rate limiting
		authProtected := router.Group("/api")
		authProtected.Use(r.am.RequireAuth())
		authProtected.Use(middleware.RateLimitMiddleware(r.authRateLimit))
		authProtected.POST("/logout", r.ah.LogoutHandler)
		authProtected.POST("/refresh_token", r.ah.RefreshTokenHandler)
	}
	
	{
		// Read-only endpoints with lenient rate limiting
		readOnly := router.Group("/api")
		readOnly.Use(r.am.RequireAuth())
		readOnly.Use(middleware.RateLimitMiddleware(r.readOnlyRateLimit))
		readOnly.GET("/user/channels/owned", r.uh.GetOwnedChannelsHandler)
		readOnly.GET("/user/channels/joined", r.uh.GetJoinedChannelsHandler)
		readOnly.GET("/channels", r.ch.GetChannelsHandler)
		readOnly.GET("/channels/me", r.ch.GetUserChannelsHandler)
		readOnly.GET("/channels/:id", r.ch.GetChannelHandler)
		readOnly.GET("/channels/:id/users", r.ch.GetChannelUsersHandler)
		readOnly.GET("/channels/:id/bans", r.ch.GetChannelBansHandler)
		readOnly.GET("/channels/:id/messages", r.mh.GetChannelMessagesHandler)
		readOnly.GET("/search/users", r.sh.SearchUsersHandler)
		readOnly.GET("/search/channels", r.sh.SearchChannelsHandler)
		readOnly.GET("/search/messages", r.sh.SearchMessagesHandler)
	}
	
	{
		// General API endpoints with standard rate limiting
		protected := router.Group("/api")
		protected.Use(r.am.RequireAuth())
		protected.Use(middleware.RateLimitMiddleware(r.generalRateLimit))
		
		// User endpoints
		protected.PATCH("/user", r.uh.UpdateUserHandler)
		protected.DELETE("/user", r.uh.DeleteUserHandler)

		// Channel endpoints
		protected.POST("/channels", r.ch.CreateChannelHandler)
		protected.POST("/channels/:id/join", r.ch.JoinChannelHandler)
		protected.DELETE("/channels/:id/leave", r.ch.LeaveChannelHandler)
		protected.DELETE("/channels/:id", r.ch.DeleteChannelHandler)
		
		// Channel administration endpoints
		protected.POST("/channels/:id/ban", r.ch.BanUserHandler)
		protected.POST("/channels/:id/tempban", r.ch.TempBanUserHandler)
		protected.DELETE("/channels/:id/ban/:userId", r.ch.UnbanUserHandler)
		protected.POST("/channels/:id/promote", r.ch.PromoteUserHandler)
		protected.POST("/channels/:id/demote", r.ch.DemoteUserHandler)
	}
}

// HealthCheckHandler checks if the server is running
// @Summary Health check
// @Description Check if the server is running and responsive
// @Tags System
// @Produce plain
// @Success 200 {string} string "Running"
// @Router /hc [get]
func HealthCheckHandler(c *gin.Context) {
	c.String(200, "Running")
}
