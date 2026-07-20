package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	// Self-contained healthcheck mode, used by the container's HEALTHCHECK
	// instruction since the distroless runtime image has no shell/wget.
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		os.Exit(runSelfHealthcheck())
	}

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

	rateLimiter := NewRateLimiter(config.AppConfig.RateLimitRPS, config.AppConfig.RateLimitBurst)

	r := gin.Default()

	// =======================
	// GLOBAL MIDDLEWARE
	// =======================
	r.Use(requestIDMiddleware())
	r.Use(internalSecretMiddleware(config.AppConfig.InternalSecret))
	r.Use(rateLimitMiddleware(rateLimiter))
	r.Use(cors.New(cors.Config{
		AllowOrigins: config.AppConfig.AllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},
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
	r.POST("/auth/admin/staff",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)
	r.GET("/auth/admin/agents",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)
	r.GET("/auth/me",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)
	r.PATCH("/auth/me",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)
	r.PATCH("/auth/me/availability",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)
	r.POST("/auth/change-password",
		authMiddleware(secret),
		proxyTrim("/auth", authURL),
	)

	// =======================
	// REPORTS (ADMIN, PROTECTED)
	// =======================
	r.GET("/reports/summary", authMiddleware(secret), proxyTo(ticketURL))
	r.GET("/reports/agents", authMiddleware(secret), proxyTo(ticketURL))
	r.GET("/reports/critical-trends", authMiddleware(secret), proxyTo(ticketURL))
	r.GET("/reports/queue-size", authMiddleware(secret), proxyTo(ticketURL))

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

	runWithGracefulShutdown(r, config.AppConfig.AppPort)
}

// runWithGracefulShutdown starts the HTTP server and blocks until SIGINT/SIGTERM,
// then drains in-flight requests before exiting.
func runWithGracefulShutdown(handler http.Handler, port string) {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		logger.Log.Info("api-gateway running on port " + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("listen: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("server forced to shutdown: ", err)
	}

	logger.Log.Info("server exited")
}

// newProxyTransport bounds how long the gateway will wait on a slow/unreachable
// downstream service, so a stuck backend can't hang gateway connections forever.
func newProxyTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
}

// proxyErrorHandler returns a JSON 502 instead of letting the proxy fall back
// to its default plain-text error when the upstream is unreachable or times out.
func proxyErrorHandler(target *url.URL) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, err error) {
		logger.Log.WithError(err).WithFields(logrus.Fields{
			"path":   req.URL.Path,
			"target": target.Host,
		}).Error("proxy error")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"success":false,"message":"upstream service unavailable","error":"BAD_GATEWAY"}`))
	}
}

// proxyTo reverse-proxies to a pre-parsed target URL, keeping the request path.
func proxyTo(target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = newProxyTransport()
	proxy.ErrorHandler = proxyErrorHandler(target)

	return func(c *gin.Context) {
		logger.WithTraceId(c.GetString("request_id")).WithFields(logrus.Fields{
			"path":   c.Request.URL.Path,
			"target": target.Host,
		}).Info("proxy request")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// proxyTrim reverse-proxies to target, stripping the given prefix from the path.
func proxyTrim(prefix string, target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = newProxyTransport()
	proxy.ErrorHandler = proxyErrorHandler(target)

	return func(c *gin.Context) {
		path := c.Request.URL.Path[len(prefix):]
		if path == "" {
			path = "/"
		}
		c.Request.URL.Path = path

		logger.WithTraceId(c.GetString("request_id")).WithFields(logrus.Fields{
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

		userIDClaim, hasUserID := claims["user_id"]
		roleClaim, hasRole := claims["role"]
		if !hasUserID || !hasRole || userIDClaim == nil || roleClaim == nil {
			response.Error(c, 401, "invalid token claims", "UNAUTHORIZED")
			c.Abort()
			return
		}

		userID := fmt.Sprintf("%v", userIDClaim)
		role := fmt.Sprintf("%v", roleClaim)
		if userID == "" || role == "" {
			response.Error(c, 401, "invalid token claims", "UNAUTHORIZED")
			c.Abort()
			return
		}

		c.Request.Header.Set("X-User-ID", userID)
		c.Request.Header.Set("X-User-ROLE", role)

		logger.WithTraceId(c.GetString("request_id")).WithFields(logrus.Fields{
			"user_id": userID,
			"role":    role,
		}).Info("authenticated request")

		c.Next()
	}
}
