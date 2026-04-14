package main

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	secret := []byte(os.Getenv("JWT_SECRET"))

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.Any("/auth/*path", proxy("/auth", os.Getenv("AUTH_SERVICE_URL")))

	r.Any("/tickets/*path",
		authMiddleware(secret),
		proxy("/tickets", os.Getenv("TICKET_SERVICE_URL")),
	)

	r.Run(":" + os.Getenv("APP_PORT"))
}

func proxy(prefix, target string) gin.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, prefix)

		// 🔥 HANDLE ROOT CASE
		if path == "" || path == "/" {
			path = prefix // jadi "/tickets"
		}

		c.Request.URL.Path = path
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := strings.TrimPrefix(
			c.GetHeader("Authorization"),
			"Bearer ",
		)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", claims["user_id"]))
		c.Request.Header.Set("X-User-ROLE", fmt.Sprintf("%v", claims["role"]))

		c.Next()
	}
}
