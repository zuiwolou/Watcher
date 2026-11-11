package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

const CtxOpenClientID = "open_client_id"

// OpenAPIAuth validates independent regulatory credentials (not business JWT).
// Accepts Basic Auth (clientId:secret) or headers X-Open-Client-Id / X-Open-Client-Secret.
func OpenAPIAuth(biz *bizsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, secret := c.GetHeader("X-Open-Client-Id"), c.GetHeader("X-Open-Client-Secret")
		if clientID == "" || secret == "" {
			if authz := c.GetHeader("Authorization"); strings.HasPrefix(authz, "Basic ") {
				raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authz, "Basic "))
				if err == nil {
					parts := strings.SplitN(string(raw), ":", 2)
					if len(parts) == 2 {
						clientID, secret = parts[0], parts[1]
					}
				}
			}
		}
		if clientID == "" || secret == "" {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "open credentials required")
			c.Abort()
			return
		}
		cred, err := biz.ValidateOpenCredential(c.Request.Context(), clientID, secret)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid open credentials")
			c.Abort()
			return
		}
		c.Set(CtxOpenClientID, cred.ClientID)
		c.Next()
	}
}
