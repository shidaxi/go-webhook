package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Auth returns a Gin middleware that validates requests using a bearer token.
// If token is empty, the middleware is a no-op (auth disabled).
func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth != "Bearer "+token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		c.Next()
	}
}
