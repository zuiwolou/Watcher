package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

// InternalToken guards service-to-service endpoints (mqtt -> webhttp evaluate).
func InternalToken() gin.HandlerFunc {
	expected := os.Getenv("WATCHER_INTERNAL_TOKEN")
	if expected == "" {
		expected = "dev-internal-token"
	}
	return func(c *gin.Context) {
		got := c.GetHeader("X-Internal-Token")
		if got == "" || got != expected {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid internal token")
			c.Abort()
			return
		}
		c.Next()
	}
}
