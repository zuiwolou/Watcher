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
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "missing bearer token")
			c.Abort()
			return
		}
		claims, err := mgr.ParseAccess(strings.TrimPrefix(h, "Bearer "))
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

func RequireClient(client string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(CtxClients)
		if !ok {
			response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "no client access")
			c.Abort()
			return
		}
		clients, _ := v.([]string)
		for _, x := range clients {
			if x == client {
				c.Next()
				return
			}
		}
		// also accept header override check against claim list
		hdr := c.GetHeader("X-Client")
		if hdr == client {
			for _, x := range clients {
				if x == hdr {
					c.Next()
					return
				}
			}
		}
		response.Fail(c, http.StatusForbidden, errcode.AuthDenied, "client not allowed: "+client)
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
