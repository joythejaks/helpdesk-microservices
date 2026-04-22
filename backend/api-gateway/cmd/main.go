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

	// =======================
	// VALIDATE URLS AT STARTUP
	// =======================
	authURL, err := url.Parse(config.AppConfig.AuthServiceURL)
	if err != nil || authURL.Host == "" {
		logger.Log.Fatal("invalid AUTH_SERVICE_URL: ", config.AppConfig.AuthServiceURL)
	}
	ticketURL, err := url.Parse(config.AppConfig.TicketServiceURL)
	if err != nil || ticketURL.Host == "" {
		logger.Log.Fatal("invalid TICKET_SERVICE_URL: ", config.AppConfig.TicketServiceURL)
	}

	r := gin.Default()

	// =======================
	// CORS
	// =======================
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// =======================
	// FIX REDIRECT LOOP
	// =======================
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// =======================
	// HEALTH
	// =======================
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, "ok")
	})

	// =======================
	// AUTH (PUBLIC)
	// =======================
	r.Any("/auth/login", proxyTrim("/auth", authURL))
	r.Any("/auth/register", proxyTrim("/auth", authURL))
	r.Any("/auth/refresh", proxyTrim("/auth", authURL))

	// =======================
	// AUTH (PROTECTED)
	// =======================
	r.POST("/auth/logout",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)

	// =======================
	// TICKETS (ROOT)
	// =======================
	r.Any("/tickets",
		authMiddleware(secret),
		proxyTo(ticketURL),
	)

	// =======================
	// TICKETS (NESTED)
	// =======================
	r.Any("/tickets/*path",
		authMiddleware(secret),
		proxyTo(ticketURL),
	)

	logger.Log.Info("api-gateway running on port " + config.AppConfig.AppPort)
	r.Run(":" + config.AppConfig.AppPort)
}

// proxyTo reverse-proxies to a pre-parsed target URL, keeping the request path.
func proxyTo(target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		logger.Log.WithFields(logrus.Fields{
			"path":   c.Request.URL.Path,
			"target": target.Host,
		}).Info("proxy request")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// proxyTrim reverse-proxies to target, stripping the given prefix from the path.
func proxyTrim(prefix string, target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		path := c.Request.URL.Path[len(prefix):]
		if path == "" {
			path = "/"
		}
		c.Request.URL.Path = path

		logger.Log.WithFields(logrus.Fields{
			"path":   path,
			"target": target.Host,
		}).Info("proxy auth")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// authMiddleware validates the Bearer JWT and injects X-User-ID / X-User-ROLE headers.
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
			// Cegah algorithm confusion attack — hanya izinkan HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
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

		c.Request.Header.Set("X-User-ID", userID)
		c.Request.Header.Set("X-User-ROLE", role)

		logger.Log.WithFields(logrus.Fields{
			"user_id": userID,
			"role":    role,
		}).Info("authenticated request")

		c.Next()
	}
}
