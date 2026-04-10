package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("secret")

func main() {
	r := gin.Default()

	// public route
	r.Any("/auth/*path", proxy("/auth", "http://auth-service:8081"))

	// protected route
	r.Any("/tickets/*path", authMiddleware(), proxy("/tickets", "http://ticket-service:8082"))

	log.Println("🚀 API Gateway running on :8080")
	r.Run(":8080")
}

func proxy(prefix string, target string) gin.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	return func(c *gin.Context) {
		// strip prefix
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, prefix)

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}
