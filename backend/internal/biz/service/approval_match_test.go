package service

import "testing"

func TestActorMatched(t *testing.T) {
	lv := &ApprovalLevel{Level: 1, RoleCodes: []string{"admin"}, UserIDs: []uint64{9}}
	if !actorMatched(lv, 9, nil) {
		t.Fatal("user id match")
	}
	if actorMatched(lv, 1, nil) {
		t.Fatal("no role fn should not match by role")
	}
	has := func(c string) bool { return c == "admin" }
	if !actorMatched(lv, 1, has) {
		t.Fatal("role match")
	}
	empty := &ApprovalLevel{Level: 1}
	if actorMatched(empty, 1, has) {
		t.Fatal("empty level should not match anyone")
	}
}

func TestStrategyHelpers(t *testing.T) {
	if !strategyAll(&ApprovalLevel{Strategy: "all"}) {
		t.Fatal("all")
	}
	if strategyAll(&ApprovalLevel{Strategy: "any"}) {
		t.Fatal("any")
	}
	if requiredApprovers(&ApprovalLevel{UserIDs: []uint64{1, 2, 3}}) != 3 {
		t.Fatal("required")
	}
}
