package metricsx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Register mounts GET /metrics (Prometheus default registry).
func Register(r gin.IRoutes) {
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// Handler returns raw http.Handler for non-Gin servers.
func Handler() http.Handler {
	return promhttp.Handler()
}
