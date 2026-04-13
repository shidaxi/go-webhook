package middleware

import (
	"github.com/gin-gonic/gin"
)

// Metrics returns a Gin middleware that records Prometheus HTTP metrics.
// Placeholder — actual Prometheus recording will be added in Phase 4.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
