package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DeviceFingerprintLimit limits queue entries per device fingerprint.
// The fingerprint is sent via X-Device-Fingerprint header from the frontend.
func DeviceFingerprintLimit(maxEntries int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		fingerprint := c.GetHeader("X-Device-Fingerprint")
		if fingerprint == "" {
			// No fingerprint provided — allow but don't track
			c.Next()
			return
		}

		key := "fp:" + fingerprint

		limiter.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-window)

		reqs := limiter.requests[key]
		cleaned := make([]time.Time, 0, len(reqs))
		for _, t := range reqs {
			if t.After(windowStart) {
				cleaned = append(cleaned, t)
			}
		}

		if len(cleaned) >= maxEntries {
			limiter.mu.Unlock()
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "同一裝置操作過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		cleaned = append(cleaned, now)
		limiter.requests[key] = cleaned
		limiter.mu.Unlock()

		c.Next()
	}
}
