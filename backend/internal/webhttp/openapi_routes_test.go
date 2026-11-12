package webhttp_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 契约抽测：OpenAPI 中若干关键路径须在 webhttp 入口注册（防路由漂移）。
func TestOpenAPICriticalPathsRegisteredInMain(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	openapi := filepath.Join(root, "contracts", "openapi", "openapi.yaml")
	mainGo := filepath.Join(root, "backend", "cmd", "webhttp", "main.go")
	oa, err := os.ReadFile(openapi)
	if err != nil {
		t.Fatal(err)
	}
	mainSrc, err := os.ReadFile(mainGo)
	if err != nil {
		t.Fatal(err)
	}
	mainStr := string(mainSrc)
	oaStr := string(oa)

	// path fragment in OpenAPI → substring expected in main.go route registration
	checks := []struct {
		openapiHint string
		mainHint    string
	}{
		{"/auth/login", "/auth/login"},
		{"/materials", "/materials"},
		{"/unique-codes", "/unique-codes"},
		{"/alarms", "/alarms"},
		{"/geofences", "/geofences"},
		{"/transport", "/transport"},
		{"/inspection", "/inspection"},
	}
	for _, c := range checks {
		if !strings.Contains(oaStr, c.openapiHint) {
			t.Fatalf("openapi missing %s", c.openapiHint)
		}
		if !strings.Contains(mainStr, c.mainHint) {
			t.Fatalf("webhttp main missing route registration for %s", c.mainHint)
		}
	}
}
