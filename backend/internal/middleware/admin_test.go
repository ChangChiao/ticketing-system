package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminToken_AllowsMatchingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AdminToken("secret-token"))
	router.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-Admin-Token", "secret-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminToken_RejectsMissingOrWrongToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "wrong", token: "bad-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AdminToken("secret-token"))
			router.GET("/admin", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.token != "" {
				req.Header.Set("X-Admin-Token", tt.token)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminToken_RejectsWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AdminToken(""))
	router.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-Admin-Token", "secret-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}
