package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/internal/webhttp/middleware"
)

func writeAudit(c *gin.Context, db *gorm.DB, action, resourceType, resourceID string, detail any) {
	if db == nil || c == nil {
		return
	}
	uid := mustUserID(c)
	client := c.GetHeader("X-Client")
	if client == "" {
		if v, ok := c.Get(middleware.CtxClients); ok {
			if arr, ok := v.([]string); ok && len(arr) > 0 {
				client = arr[0]
			}
		}
	}
	iam.WriteAudit(db, uid, action, resourceType, resourceID, client, c.ClientIP(), detail)
}
