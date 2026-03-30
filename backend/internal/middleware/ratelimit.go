package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

// RateLimit limits requests per user (or IP if unauthenticated) using Redis.
func RateLimit(redisClient *pkgredis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:ip:" + c.ClientIP()
		if userID := c.GetString("user_id"); userID != "" {
			key = "rl:user:" + userID
		}

		allowed, err := redisClient.CheckRateLimit(c.Request.Context(), key, maxRequests, window)
		if err != nil {
			log.Printf("rate limit redis error: %v", err)
			// Redis 故障時放行，避免整個服務不可用
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "請求過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit limits requests per IP address using Redis.
func IPRateLimit(redisClient *pkgredis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:ip:" + c.ClientIP()

		allowed, err := redisClient.CheckRateLimit(c.Request.Context(), key, maxRequests, window)
		if err != nil {
			log.Printf("rate limit redis error: %v", err)
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "請求過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		c.Next()
	}
}
