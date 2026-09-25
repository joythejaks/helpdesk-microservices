package main

import (
	"context"
	"errors"
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
	"api-gateway/pkg/metrics"
	"api-gateway/pkg/response"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker/v2"
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

	// Shared across replicas via Redis when REDIS_URL is set; otherwise an
	// in-memory limiter, where N replicas would mean N x the limit.
	rateLimiter := newLimiter(config.AppConfig.RedisURL, "rl:gateway:", config.AppConfig.RateLimitRPS, config.AppConfig.RateLimitBurst)

	// One breaker per upstream, shared across every route proxying to that
	// upstream — fails fast (503) once a downstream is reliably down,
	// instead of every request waiting out the full proxy timeout.
	authBreaker := newUpstreamBreaker("auth-service")
	ticketBreaker := newUpstreamBreaker("ticket-service")

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
	r.Use(metrics.GinMiddleware())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
	r.Any("/auth/login", proxyTrim("/auth", authURL, authBreaker))
	r.Any("/auth/register", proxyTrim("/auth", authURL, authBreaker))
	r.Any("/auth/refresh", proxyTrim("/auth", authURL, authBreaker))

	// =======================
	// AUTH (PROTECTED)
	// =======================
	r.POST("/auth/logout",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.POST("/auth/admin/staff",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.GET("/auth/admin/agents",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.GET("/auth/me",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.PATCH("/auth/me",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.PATCH("/auth/me/availability",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)
	r.POST("/auth/change-password",
		authMiddleware(secret),
		proxyTrim("/auth", authURL, authBreaker),
	)

	// =======================
	// REPORTS (ADMIN, PROTECTED)
	// =======================
	r.GET("/reports/summary", authMiddleware(secret), proxyTo(ticketURL, ticketBreaker))
	r.GET("/reports/agents", authMiddleware(secret), proxyTo(ticketURL, ticketBreaker))
	r.GET("/reports/critical-trends", authMiddleware(secret), proxyTo(ticketURL, ticketBreaker))
	r.GET("/reports/queue-size", authMiddleware(secret), proxyTo(ticketURL, ticketBreaker))

	// =======================
	// TICKETS (ROOT)
	// =======================
	r.Any("/tickets",
		authMiddleware(secret),
		proxyTo(ticketURL, ticketBreaker),
	)

	// =======================
	// TICKETS (NESTED)
	// =======================
	r.Any("/tickets/*path",
		authMiddleware(secret),
		proxyTo(ticketURL, ticketBreaker),
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
// to its default plain-text error when the upstream is unreachable or times
// out — or a JSON 503 when the circuit breaker is the reason the request
// never reached the upstream at all, so callers can tell "briefly down" apart
// from "we've stopped even trying."
func proxyErrorHandler(target *url.URL) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, err error) {
		logger.Log.WithError(err).WithFields(logrus.Fields{
			"path":   req.URL.Path,
			"target": target.Host,
		}).Error("proxy error")

		status := http.StatusBadGateway
		body := `{"success":false,"message":"upstream service unavailable","error":"BAD_GATEWAY"}`
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			status = http.StatusServiceUnavailable
			body = `{"success":false,"message":"upstream service temporarily unavailable","error":"CIRCUIT_OPEN"}`
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

// newUpstreamBreaker builds one circuit breaker for an upstream host, meant
// to be constructed once in main() and shared across every route proxying
// to that host — trips (fails fast with 503) once that downstream is
// reliably failing, instead of every request separately waiting out the
// full proxy timeout.
func newUpstreamBreaker(name string) *gobreaker.CircuitBreaker[*http.Response] {
	return gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
		Name: name,
		// A client giving up says nothing about the upstream (see
		// clientGoneError), so it neither counts as a failure nor as a
		// success.
		IsExcluded: func(err error) bool {
			var gone *clientGoneError
			return errors.As(err, &gone)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Log.Warnf("circuit breaker %s: %s -> %s", name, from, to)
		},
	})
}

// breakerTransport wraps a RoundTripper with a circuit breaker. Safe by
// construction: http.RoundTripper.RoundTrip only returns a non-nil error for
// actual transport failures (connection refused, timeout, DNS) — never for
// an ordinary non-2xx HTTP response — so the breaker trips only on real
// downstream connectivity failure, not the upstream's own 4xx/5xx responses.
type breakerTransport struct {
	breaker *gobreaker.CircuitBreaker[*http.Response]
	next    http.RoundTripper
}

// clientGoneError marks a transport error caused by the client giving up (it
// hung up, or its own deadline passed) rather than by the upstream failing.
// Without this, RoundTrip's `context canceled` was counted as an upstream
// failure, so a handful of abandoned requests opened the circuit and every
// other client got 503 for the whole open period — anyone could do that on
// purpose by starting requests and dropping the connection.
type clientGoneError struct{ err error }

func (e *clientGoneError) Error() string { return e.err.Error() }
func (e *clientGoneError) Unwrap() error { return e.err }

func (t *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.breaker.Execute(func() (*http.Response, error) {
		resp, err := t.next.RoundTrip(req)
		// The request's own context is done, so this isn't the upstream's
		// doing. The transport's own timeouts (dial, response headers) don't
		// touch the request context, so a genuinely slow or dead upstream
		// still counts.
		if err != nil && req.Context().Err() != nil {
			return nil, &clientGoneError{err}
		}
		return resp, err
	})

	// Hand the reverse proxy the original error, not the marker.
	var gone *clientGoneError
	if errors.As(err, &gone) {
		return nil, gone.err
	}
	return resp, err
}

// proxyTo reverse-proxies to a pre-parsed target URL, keeping the request path.
func proxyTo(target *url.URL, breaker *gobreaker.CircuitBreaker[*http.Response]) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &breakerTransport{breaker: breaker, next: newProxyTransport()}
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
func proxyTrim(prefix string, target *url.URL, breaker *gobreaker.CircuitBreaker[*http.Response]) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &breakerTransport{breaker: breaker, next: newProxyTransport()}
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
