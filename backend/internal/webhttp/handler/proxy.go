package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// StreamProxy upgrades after JWT middleware and reverse-proxies to push service.
func StreamProxy(pushBase string) gin.HandlerFunc {
	u, err := url.Parse(pushBase)
	if err != nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusBadGateway, gin.H{"code": "UPSTREAM_UNAVAILABLE", "message": "bad push url"})
		}
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	return func(c *gin.Context) {
		c.Request.URL.Path = "/v1/stream"
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func HistoryProxy(historyBase, path string) gin.HandlerFunc {
	u, err := url.Parse(historyBase)
	if err != nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusBadGateway, gin.H{"code": "UPSTREAM_UNAVAILABLE"})
		}
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	return func(c *gin.Context) {
		c.Request.URL.Path = path
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
