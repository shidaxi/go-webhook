package server

import (
	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/handler"
	"github.com/shidaxi/go-webhook/internal/middleware"
)

// NewWebhookEngine creates the business Gin engine with webhook routes.
func NewWebhookEngine(store *engine.RuleStore, authToken string) *gin.Engine {
	r := gin.New()

	// Middleware chain: Recovery → Logging → Metrics → Auth
	r.Use(gin.Recovery())
	r.Use(middleware.Logging())
	r.Use(middleware.Metrics())
	r.Use(middleware.Auth(authToken))

	wh := handler.NewWebhookHandler(store)
	r.POST("/webhook", wh.Handle)

	return r
}
