package auth

import (
	"testing"
	"time"
)

func TestIssueAndParseAccess(t *testing.T) {
	m := NewManager("test-secret", 30*time.Minute, 168*time.Hour)
	tok, exp, err := m.IssueAccess("1", "admin", []string{"material:view"}, []string{"WH01"}, []string{"admin", "portal"})
	if err != nil {
		t.Fatal(err)
	}
	if exp <= 0 {
		t.Fatalf("expiresIn=%d", exp)
	}
	claims, err := m.ParseAccess(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "1" || claims.Username != "admin" {
		t.Fatalf("claims=%+v", claims)
	}
	if len(claims.Clients) != 2 || claims.Clients[0] != "admin" {
		t.Fatalf("clients=%v", claims.Clients)
	}
}

func TestParseAccessRejectsTampered(t *testing.T) {
	m := NewManager("test-secret", time.Minute, time.Hour)
	tok, _, err := m.IssueAccess("1", "u", nil, nil, []string{"app"})
	if err != nil {
		t.Fatal(err)
	}
	bad := tok + "x"
	if _, err := m.ParseAccess(bad); err == nil {
		t.Fatal("expected parse error")
	}
	other := NewManager("other-secret", time.Minute, time.Hour)
	if _, err := other.ParseAccess(tok); err == nil {
		t.Fatal("expected secret mismatch error")
	}
}
