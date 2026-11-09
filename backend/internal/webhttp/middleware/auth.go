package middleware

import (
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"watcher/backend/pkg/auth"
	"watcher/backend/pkg/authz"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

const (
	CtxUserID       = "user_id"
	CtxUsername     = "username"
	CtxPermissions  = "permissions"
	CtxWarehouseIDs = "warehouse_ids"
	CtxClients      = "clients"
	CtxRequestID    = "request_id"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(CtxRequestID, rid)
		c.Writer.Header().Set("X-Request-Id", rid)
		c.Next()
	}
}

func JWT(mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		h := c.GetHeader("Authorization")
		if strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		// WSS / 部分客户端可通过 query 传 access token
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			token = c.Query("access_token")
		}
		if token == "" {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "missing bearer token")
			c.Abort()
			return
		}
		claims, err := mgr.ParseAccess(token)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid token")
			c.Abort()
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxPermissions, claims.Permissions)
		c.Set(CtxWarehouseIDs, claims.WarehouseIDs)
		c.Set(CtxClients, claims.Clients)
		c.Next()
	}
}

func clientAllowed(claimClients []string, want string) bool {
	for _, x := range claimClients {
		if x == want {
			return true
		}
	}
	return false
}

// RequireClient enforces JWT claim clients contains the required client.
// When X-Client is present it must also match and be in claims.
func RequireClient(client string) gin.HandlerFunc {
	return RequireAnyClient(client)
}

// RequireAnyClient allows any of the listed clients (admin|portal|app).
func RequireAnyClient(allowed ...string) gin.HandlerFunc {
	set := map[string]struct{}{}
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	return func(c *gin.Context) {
		v, ok := c.Get(CtxClients)
		if !ok {
			response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "no client access")
			c.Abort()
			return
		}
		clients, _ := v.([]string)
		hdr := strings.TrimSpace(c.GetHeader("X-Client"))
		if hdr != "" {
			if _, ok := set[hdr]; !ok {
				response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "X-Client not permitted on this route: "+hdr)
				c.Abort()
				return
			}
			if !clientAllowed(clients, hdr) {
				response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "X-Client not in token: "+hdr)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		for _, a := range allowed {
			if clientAllowed(clients, a) {
				c.Next()
				return
			}
		}
		response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "client not allowed")
		c.Abort()
	}
}

func RequirePerm(e *casbin.Enforcer, obj, act string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Get(CtxUserID)
		userID, _ := uid.(string)
		ok, err := authz.EnforceUser(e, userID, obj, act)
		if err != nil || !ok {
			response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "permission denied: "+obj+":"+act)
			c.Abort()
			return
		}
		c.Next()
	}
}
