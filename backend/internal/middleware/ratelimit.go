package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
}

var limiter = &rateLimiter{
	requests: make(map[string][]time.Time),
}

func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID := c.GetString("user_id"); userID != "" {
			key = "user:" + userID
		}

		limiter.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-window)

		// Clean old entries
		reqs := limiter.requests[key]
		cleaned := make([]time.Time, 0, len(reqs))
		for _, t := range reqs {
			if t.After(windowStart) {
				cleaned = append(cleaned, t)
			}
		}

		if len(cleaned) >= maxRequests {
			limiter.mu.Unlock()
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "請求過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		cleaned = append(cleaned, now)
		limiter.requests[key] = cleaned
		limiter.mu.Unlock()

		c.Next()
	}
}

// IPRateLimit limits requests per IP specifically
func IPRateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()

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

		if len(cleaned) >= maxRequests {
			limiter.mu.Unlock()
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "請求過於頻繁，請稍後再試"})
			c.Abort()
			return
		}

		cleaned = append(cleaned, now)
		limiter.requests[key] = cleaned
		limiter.mu.Unlock()

		c.Next()
	}
}
