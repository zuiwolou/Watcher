package iam

import (
	"encoding/json"

	"gorm.io/gorm"
)

// WriteAudit appends an audit log row (best-effort, ignores write errors).
func WriteAudit(db *gorm.DB, actorID uint64, action, resourceType, resourceID, client, ip string, detail any) {
	if db == nil {
		return
	}
	detailJSON := ""
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailJSON = string(b)
		}
	}
	_ = db.Create(&AuditLog{
		ActorID: actorID, Action: action, ResourceType: resourceType,
		ResourceID: resourceID, Client: client, IP: ip, DetailJSON: detailJSON,
	}).Error
}
