package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

// DeviceFingerprintLimit limits queue entries per device fingerprint using Redis.
// The fingerprint is sent via X-Device-Fingerprint header from the frontend.
func DeviceFingerprintLimit(redisClient *pkgredis.Client, maxEntries int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		fingerprint := c.GetHeader("X-Device-Fingerprint")
		if fingerprint == "" {
			// No fingerprint provided — allow but don't track
			c.Next()
			return
		}

		key := "rl:fp:" + fingerprint

		allowed, err := redisClient.CheckRateLimit(c.Request.Context(), key, maxEntries, window)
		if err != nil {
			log.Printf("fingerprint rate limit redis error: %v", err)
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "同一裝置操作過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		c.Next()
	}
}
