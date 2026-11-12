package authz

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCasbinRolePermission(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:casbin_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewEnforcer(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddRolePermission(e, "admin", "device", "view"); err != nil {
		t.Fatal(err)
	}
	if _, err := AddRoleForUser(e, "1", "admin"); err != nil {
		t.Fatal(err)
	}
	ok, err := EnforceUser(e, "1", "device", "view")
	if err != nil || !ok {
		t.Fatalf("admin should allow device:view, ok=%v err=%v", ok, err)
	}
	ok, err = EnforceUser(e, "1", "device", "delete")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("admin should not have device:delete unless granted")
	}
	ok, err = EnforceUser(e, "99", "device", "view")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("user without role must be denied")
	}
}
