package authz

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const defaultModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("casbin adapter: %w", err)
	}
	m, err := model.NewModelFromString(defaultModel)
	if err != nil {
		return nil, err
	}
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, err
	}
	return e, nil
}

// EnforceUser checks permission for a user subject "user:<id>" via role bindings.
func EnforceUser(e *casbin.Enforcer, userID, obj, act string) (bool, error) {
	return e.Enforce(fmt.Sprintf("user:%s", userID), obj, act)
}

func AddRoleForUser(e *casbin.Enforcer, userID, roleCode string) (bool, error) {
	return e.AddGroupingPolicy(fmt.Sprintf("user:%s", userID), fmt.Sprintf("role:%s", roleCode))
}

func AddRolePermission(e *casbin.Enforcer, roleCode, obj, act string) (bool, error) {
	return e.AddPolicy(fmt.Sprintf("role:%s", roleCode), obj, act)
}
