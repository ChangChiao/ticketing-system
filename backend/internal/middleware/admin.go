package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "管理功能尚未設定"})
			c.Abort()
			return
		}

		provided := c.GetHeader("X-Admin-Token")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "管理權限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}
