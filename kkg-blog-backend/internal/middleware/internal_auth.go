package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const InternalAuthHeader = "X-Internal-Auth"

func InternalAuth(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(expectedToken)
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    503,
				"message": "internal auth token is not configured",
			})
			return
		}
		actual := strings.TrimSpace(c.GetHeader(InternalAuthHeader))
		if actual == "" || actual != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid internal auth token",
			})
			return
		}
		c.Next()
	}
}
