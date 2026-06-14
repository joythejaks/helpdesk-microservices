package http

import (
	"auth-service/pkg/response"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware mengecek apakah user memiliki role yang diizinkan
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetHeader("X-User-Role") // Diisi oleh API Gateway setelah validasi JWT

		for _, role := range allowedRoles {
			if role == userRole {
				c.Next()
				return
			}
		}

		response.Error(c, 403, "forbidden: insufficient permissions", "forbidden")
		c.Abort()
	}
}

// TraceMiddleware memastikan setiap request memiliki Trace ID untuk logging
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set("TraceID", traceID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}
