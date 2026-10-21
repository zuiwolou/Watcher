package iam

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"watcher/backend/pkg/authz"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{}, &Role{}, &UserRole{}, &UserClientAccess{}, &UserScope{},
		&RefreshToken{}, &AuditLog{},
	)
}

func SeedAdmin(db *gorm.DB, e *casbin.Enforcer, username, password, warehouseID string) error {
	var role Role
	if err := db.Where("code = ?", "admin").First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			role = Role{Code: "admin", Name: "系统管理员"}
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	for _, p := range Catalog {
		if _, err := authz.AddRolePermission(e, "admin", p[0], p[1]); err != nil {
			return fmt.Errorf("casbin policy: %w", err)
		}
	}

	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user = User{
			Username:           username,
			PasswordHash:       string(hash),
			DisplayName:        "系统管理员",
			Status:             1,
			MustChangePassword: false,
		}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	_ = db.Where("user_id = ? AND role_id = ?", user.ID, role.ID).FirstOrCreate(&UserRole{UserID: user.ID, RoleID: role.ID})
	for _, c := range []string{"portal", "admin", "app"} {
		_ = db.Where("user_id = ? AND client = ?", user.ID, c).
			FirstOrCreate(&UserClientAccess{UserID: user.ID, Client: c})
	}
	_ = db.Where("user_id = ? AND warehouse_id = ?", user.ID, warehouseID).
		FirstOrCreate(&UserScope{UserID: user.ID, WarehouseID: warehouseID})

	if _, err := authz.AddRoleForUser(e, fmt.Sprintf("%d", user.ID), "admin"); err != nil {
		return err
	}

	// readonly role for LDAP auto-provision
	var ro Role
	if err := db.Where("code = ?", "readonly").First(&ro).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ro = Role{Code: "readonly", Name: "只读观察员"}
			if err := db.Create(&ro).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	for _, p := range [][2]string{
		{"portal", "access"}, {"app", "access"},
		{"material", "view"}, {"alarm", "view"}, {"monitor", "view"},
		{"track", "view"}, {"geofence", "view"}, {"inventory", "view"}, {"inspection", "view"}, {"warehouse", "view"},
	} {
		_, _ = authz.AddRolePermission(e, "readonly", p[0], p[1])
	}

	return e.SavePolicy()
}

func ListPermissions(e *casbin.Enforcer, userID uint64) ([]string, error) {
	sub := fmt.Sprintf("user:%d", userID)
	roles, err := e.GetRolesForUser(sub)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, role := range roles {
		policies, err := e.GetFilteredPolicy(0, role)
		if err != nil {
			return nil, err
		}
		for _, p := range policies {
			if len(p) >= 3 {
				set[p[1]+":"+p[2]] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out, nil
}
