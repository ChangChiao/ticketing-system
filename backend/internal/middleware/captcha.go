package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResponse struct {
	Success bool `json:"success"`
}

// CaptchaVerify validates a Cloudflare Turnstile token sent in the request body or header.
func CaptchaVerify(secretKey string) gin.HandlerFunc {
	client := &http.Client{Timeout: 10 * time.Second}

	return func(c *gin.Context) {
		// Skip CAPTCHA if no secret key configured (dev environment)
		if secretKey == "" {
			c.Next()
			return
		}

		captchaToken := c.GetHeader("X-Captcha-Token")
		if captchaToken == "" {
			// Try reading from JSON body field
			var body struct {
				CaptchaToken string `json:"captcha_token"`
			}
			if err := c.ShouldBindJSON(&body); err == nil {
				captchaToken = body.CaptchaToken
			}
		}

		if captchaToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "請完成人機驗證"})
			c.Abort()
			return
		}

		resp, err := client.PostForm(turnstileVerifyURL, url.Values{
			"secret":   {secretKey},
			"response": {captchaToken},
			"remoteip": {c.ClientIP()},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "驗證服務暫時無法使用"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "驗證服務回應異常"})
			c.Abort()
			return
		}

		var result turnstileResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "驗證結果解析失敗"})
			c.Abort()
			return
		}

		if !result.Success {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("人機驗證失敗，請重新驗證")})
			c.Abort()
			return
		}

		c.Next()
	}
}
