package http

import (
	"auth-service/pkg/response"

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
