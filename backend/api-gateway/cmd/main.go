package main

import (
	"fmt"
	"net/http/httputil"
	"net/url"

	"api-gateway/pkg/config"
	"api-gateway/pkg/logger"
	"api-gateway/pkg/response"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()

	config.Load()
	logger.Init("api-gateway")

	secret := []byte(config.AppConfig.JWTSecret)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// 🔥 FIX REDIRECT LOOP
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// =======================
	// HEALTH
	// =======================
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, "ok")
	})

	// =======================
	// AUTH (TRIM PREFIX)
	// =======================
	r.Any("/auth/*path", proxyTrim("/auth", config.AppConfig.AuthServiceURL))

	// =======================
	// TICKETS (ROOT FIX)
	// =======================
	r.Any("/tickets",
		authMiddleware(secret),
		proxy(config.AppConfig.TicketServiceURL),
	)

	// =======================
	// TICKETS (NESTED)
	// =======================
	r.Any("/tickets/*path",
		authMiddleware(secret),
		proxy(config.AppConfig.TicketServiceURL),
	)

	logger.Log.Info("api-gateway running on port " + config.AppConfig.AppPort)

	r.Run(":" + config.AppConfig.AppPort)
}

//
// =======================
// 🔥 PROXY (KEEP PREFIX)
// =======================
//

func proxy(target string) gin.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	return func(c *gin.Context) {

		logger.Log.WithFields(logrus.Fields{
			"path":   c.Request.URL.Path,
			"target": target,
		}).Info("proxy ticket")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

//
// =======================
// 🔥 PROXY AUTH (TRIM)
// =======================
//

func proxyTrim(prefix, target string) gin.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	return func(c *gin.Context) {

		path := c.Request.URL.Path[len(prefix):]

		if path == "" {
			path = "/"
		}

		c.Request.URL.Path = path

		logger.Log.WithFields(logrus.Fields{
			"path":   path,
			"target": target,
		}).Info("proxy auth")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

//
// =======================
// 🔥 JWT MIDDLEWARE
// =======================
//

func authMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			response.Error(c, 401, "missing token", "UNAUTHORIZED")
			c.Abort()
			return
		}

		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})

		if err != nil || !token.Valid {
			logger.Log.WithError(err).Warn("invalid token")

			response.Error(c, 401, "invalid token", "UNAUTHORIZED")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, 401, "invalid token claims", "UNAUTHORIZED")
			c.Abort()
			return
		}

		userID := fmt.Sprintf("%v", claims["user_id"])
		role := fmt.Sprintf("%v", claims["role"])

		// inject ke downstream service
		c.Request.Header.Set("X-User-ID", userID)
		c.Request.Header.Set("X-User-ROLE", role)

		logger.Log.WithFields(logrus.Fields{
			"user_id": userID,
			"role":    role,
		}).Info("authenticated request")

		c.Next()
	}
}
