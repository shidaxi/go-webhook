package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/metrics"
)

// Metrics returns a Gin middleware that records Prometheus HTTP metrics.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		metrics.RequestsTotal.WithLabelValues(status, path).Inc()
		metrics.RequestDuration.WithLabelValues(path).Observe(duration)
	}
}
