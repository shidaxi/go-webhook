package server

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/handler"
	"github.com/shidaxi/go-webhook/internal/middleware"
)

// NewAdminEngine creates the admin Gin engine with operational routes.
func NewAdminEngine(store *engine.RuleStore, cfg config.AppConfig) *gin.Engine {
	r := gin.New()

	// Lightweight middleware: Recovery → Logging only (no auth)
	r.Use(gin.Recovery())
	r.Use(middleware.Logging())

	ah := handler.NewAdminHandler(store, cfg)
	r.GET("/admin/healthz", ah.Healthz)
	r.GET("/admin/rules", ah.Rules)
	r.GET("/admin/config", ah.Config)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}
