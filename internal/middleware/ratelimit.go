package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	RequestsPerSecond int           // Number of requests per second allowed
	BurstSize         int           // Maximum burst size
	CleanupInterval   time.Duration // How often to clean up old limiters
}

// IPRateLimiter manages rate limiters for different IP addresses
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   RateLimitConfig
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(config RateLimitConfig) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
	
	// Start cleanup goroutine
	go limiter.cleanupRoutine()
	
	return limiter
}

// GetLimiter returns the rate limiter for a specific IP
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(i.config.RequestsPerSecond), i.config.BurstSize)
		i.limiters[ip] = limiter
	}
	
	return limiter
}

// cleanupRoutine periodically removes unused limiters to prevent memory leaks
func (i *IPRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(i.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		i.mu.Lock()
		for ip, limiter := range i.limiters {
			// Remove limiters that haven't been used recently
			if limiter.Tokens() == float64(i.config.BurstSize) {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

// getClientIP extracts the real client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first (for proxies)
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		ip := forwarded
		for idx := 0; idx < len(forwarded); idx++ {
			if forwarded[idx] == ',' {
				ip = forwarded[:idx]
				break
			}
		}
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			return ip
		}
	}
	
	// Check X-Real-IP header
	realIP := c.GetHeader("X-Real-IP")
	if realIP != "" {
		if parsedIP := net.ParseIP(realIP); parsedIP != nil {
			return realIP
		}
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		rateLimiter := limiter.GetLimiter(clientIP)
		
		if !rateLimiter.Allow() {
			c.Header("Retry-After", "1") // Suggest retry after 1 second
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please slow down.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// Predefined rate limit configurations for different endpoint types
var (
	// StrictRateLimit for authentication endpoints (stricter limits)
	StrictRateLimit = RateLimitConfig{
		RequestsPerSecond: 5,             // 5 requests per second
		BurstSize:         10,            // Allow burst of 10
		CleanupInterval:   5 * time.Minute,
	}
	
	// StandardRateLimit for general API endpoints
	StandardRateLimit = RateLimitConfig{
		RequestsPerSecond: 30,            // 30 requests per second
		BurstSize:         50,            // Allow burst of 50
		CleanupInterval:   5 * time.Minute,
	}
	
	// LenientRateLimit for read-only endpoints
	LenientRateLimit = RateLimitConfig{
		RequestsPerSecond: 100,           // 100 requests per second
		BurstSize:         200,           // Allow burst of 200
		CleanupInterval:   5 * time.Minute,
	}
)