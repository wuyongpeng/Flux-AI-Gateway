package middleware

import (
	"flux-ai-gateway/internal/limiter"
	"flux-ai-gateway/internal/metrics"

	"github.com/gin-gonic/gin"
)

// AuthAndLimitMiddleware handles identifying the user and applying rate limits.
func AuthAndLimitMiddleware(rl *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(401, gin.H{"error": "missing X-User-ID"})
			c.Abort()
			return
		}

		// Mock mechanism to determine tier
		tier := limiter.Normal
		if userID == "admin" || userID == "vip-user" {
			tier = limiter.VIP
		}

		if !rl.Allow(c.Request.Context(), userID, tier) {
			metrics.Error429Total.WithLabelValues(userID).Inc()
			c.JSON(429, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		// Pass the UserID down the context so metrics/proxy can use it
		c.Set("UserID", userID)

		c.Next()
	}
}
