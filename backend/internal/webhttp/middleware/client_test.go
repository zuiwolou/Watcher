package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAnyClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/biz", func(c *gin.Context) {
		c.Set(CtxClients, []string{"portal", "app"})
		c.Next()
	}, RequireAnyClient("admin", "portal", "app"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	r.GET("/admin", func(c *gin.Context) {
		c.Set(CtxClients, []string{"portal"})
		c.Next()
	}, RequireClient("admin"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/biz", nil)
	req.Header.Set("X-Client", "portal")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("portal on biz want 200 got %d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req2.Header.Set("X-Client", "portal")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("portal on admin want 403 got %d", w2.Code)
	}
}
