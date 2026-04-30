package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestSignature validates an HMAC-SHA256 signature on protected requests.
// The frontend generates: HMAC-SHA256(secret, method + path + timestamp)
// Headers: X-Request-Signature, X-Request-Timestamp
func RequestSignature(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if no secret configured (dev environment)
		if secret == "" {
			c.Next()
			return
		}

		signature := c.GetHeader("X-Request-Signature")
		timestamp := c.GetHeader("X-Request-Timestamp")

		if signature == "" || timestamp == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "缺少請求簽名"})
			c.Abort()
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "請求時間戳格式錯誤"})
			c.Abort()
			return
		}
		requestTime := time.UnixMilli(ts)
		if time.Since(requestTime) > 5*time.Minute || time.Until(requestTime) > 1*time.Minute {
			c.JSON(http.StatusForbidden, gin.H{"error": "請求簽名已過期"})
			c.Abort()
			return
		}

		// Construct the message: method + path + timestamp
		message := c.Request.Method + c.Request.URL.Path + timestamp

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(message))
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expected)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "請求簽名驗證失敗"})
			c.Abort()
			return
		}

		c.Next()
	}
}
