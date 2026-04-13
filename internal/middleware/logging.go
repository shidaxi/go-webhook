package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/logger"
	"go.uber.org/zap"
)

// Logging returns a Gin middleware that logs each request using Zap.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		logger.L().Info("request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("request_id", c.GetHeader("X-Request-ID")),
		)
	}
}
