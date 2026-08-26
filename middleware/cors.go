package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that sets Cross-Origin Resource Sharing headers.
func CORS(allowedOrigins []string) gin.HandlerFunc {

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if the request origin is allowed
		if allowsAllOrigins(allowedOrigins) && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		} else if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Visitor-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours preflight cache

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func allowsAllOrigins(allowed []string) bool {
	for _, o := range allowed {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}

// isOriginAllowed checks if the given origin is in the allowed list.
func isOriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, o := range allowed {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}
