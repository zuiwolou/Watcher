package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInternalToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prev := os.Getenv("WATCHER_INTERNAL_TOKEN")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("WATCHER_INTERNAL_TOKEN")
		} else {
			_ = os.Setenv("WATCHER_INTERNAL_TOKEN", prev)
		}
	})
	_ = os.Setenv("WATCHER_INTERNAL_TOKEN", "secret-token")

	r := gin.New()
	r.GET("/internal/ping", InternalToken(), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without token, got %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/internal/ping", nil)
	req2.Header.Set("X-Internal-Token", "secret-token")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("want 200 with token, got %d", w2.Code)
	}
}
