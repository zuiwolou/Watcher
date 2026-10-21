package iam

import "time"

type User struct {
	ID                  uint64    `gorm:"primaryKey"`
	Username            string    `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash        string    `gorm:"size:255"`
	DisplayName         string    `gorm:"size:64"`
	Mobile              string    `gorm:"size:32"`
	Dept                string    `gorm:"size:128"`
	Status              int8      `gorm:"default:1"` // 1 enable
	MustChangePassword  bool      `gorm:"default:false"`
	LDAPDN              string    `gorm:"size:512;column:ldap_dn"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (User) TableName() string { return "users" }

type Role struct {
	ID        uint64 `gorm:"primaryKey"`
	Code      string `gorm:"size:64;uniqueIndex;not null"`
	Name      string `gorm:"size:64"`
	Remark    string `gorm:"size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Role) TableName() string { return "roles" }

type UserRole struct {
	UserID uint64 `gorm:"primaryKey"`
	RoleID uint64 `gorm:"primaryKey"`
}

func (UserRole) TableName() string { return "user_roles" }

type UserClientAccess struct {
	UserID uint64 `gorm:"primaryKey"`
	Client string `gorm:"size:16;primaryKey"` // portal|admin|app
}

func (UserClientAccess) TableName() string { return "user_client_access" }

type UserScope struct {
	UserID      uint64 `gorm:"primaryKey"`
	WarehouseID string `gorm:"size:64;primaryKey"`
}

func (UserScope) TableName() string { return "user_scopes" }

type RefreshToken struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"index;not null"`
	TokenHash string `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time
	Revoked   bool `gorm:"default:false"`
	CreatedAt time.Time
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

type AuditLog struct {
	ID           uint64 `gorm:"primaryKey"`
	ActorID      uint64 `gorm:"index"`
	Action       string `gorm:"size:64"`
	ResourceType string `gorm:"size:64"`
	ResourceID   string `gorm:"size:64"`
	Client       string `gorm:"size:16"`
	IP           string `gorm:"size:64"`
	DetailJSON   string `gorm:"type:text"`
	CreatedAt    time.Time
}

func (AuditLog) TableName() string { return "audit_logs" }
