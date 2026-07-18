package http

import (
	"crypto/subtle"

	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

// InternalOnlyMiddleware rejects any request that doesn't carry the shared
// secret the API gateway attaches to every proxied request — closes off
// calling ticket-service directly and spoofing X-User-ID/X-User-ROLE,
// bypassing the gateway's JWT validation.
func InternalOnlyMiddleware(secret string) gin.HandlerFunc {
	secretBytes := []byte(secret)

	return func(c *gin.Context) {
		got := c.GetHeader("X-Internal-Secret")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), secretBytes) != 1 {
			response.Error(c, 403, "forbidden", "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}
