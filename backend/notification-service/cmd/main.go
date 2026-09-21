package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"notification-service/internal/consumer"
	delivery "notification-service/internal/delivery/http"
	"notification-service/internal/delivery/ws"
	"notification-service/internal/migrations"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"
	"notification-service/pkg/config"
	"notification-service/pkg/logger"
	"notification-service/pkg/metrics"
	"notification-service/pkg/response"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Self-contained healthcheck mode for the container's HEALTHCHECK
	// instruction (distroless runtime image has no shell/wget).
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		os.Exit(runSelfHealthcheck())
	}

	godotenv.Load()

	config.Load()
	logger.Init("notification-service")

	port := config.AppConfig.AppPort
	rabbitURL := config.AppConfig.RabbitMQURL

	// DB
	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Log.Fatal("failed to connect DB:", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatal("failed to get database instance:", err)
	}
	if err := migrations.Run(sqlDB, "notifications"); err != nil {
		logger.Log.Fatal("failed to run migrations:", err)
	}

	notificationRepo := repository.NewNotificationRepository(db)
	notificationUsecase := usecase.NewNotificationUsecase(notificationRepo)
	notificationHandler := delivery.NewNotificationHandler(notificationUsecase)

	ws.Init([]byte(config.AppConfig.JWTSecret), config.AppConfig.AllowedOrigins, config.AppConfig.MaxWSConnections)
	delivery.Init([]byte(config.AppConfig.JWTSecret))

	// start consumer (non-blocking, auto-reconnect)
	consumer.StartConsumer(rabbitURL, notificationUsecase)

	// custom mux — hindari register ke DefaultServeMux global
	mux := http.NewServeMux()

	wsLimiter := ws.NewRateLimiter(config.AppConfig.WSRateLimitRPS, config.AppConfig.WSRateLimitBurst)
	mux.HandleFunc("/ws", ws.RateLimit(wsLimiter, ws.HandleConnections))

	// /health is readiness (DB + RabbitMQ checked); /healthz is liveness
	// (unconditional 200) — Kubernetes should restart the pod on the
	// latter, not on a slow dependency reconnect the former would fail.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if !consumer.IsConnected() || sqlDB.PingContext(ctx) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("dependency disconnected"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, "ok")
	})

	mux.Handle("/metrics", promhttp.Handler())

	// REST notifications API — self-authenticated (Authorization: Bearer),
	// same trust model as /ws since this service was never put behind the
	// gateway (the reverse proxy doesn't handle WS upgrades).
	mux.HandleFunc("GET /notifications", metrics.Wrap("GET /notifications", notificationHandler.List))
	mux.HandleFunc("PATCH /notifications/read-all", metrics.Wrap("PATCH /notifications/read-all", notificationHandler.MarkAllRead))
	mux.HandleFunc("PATCH /notifications/{id}/read", metrics.Wrap("PATCH /notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(r.PathValue("id"), 10, 0)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid notification id", "BAD_REQUEST")
			return
		}
		notificationHandler.MarkRead(w, r, uint(id))
	}))

	go ws.HandleMessages()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: requestIDMiddleware(mux),
	}

	go func() {
		logger.Log.Info("notification-service running on port " + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("server error: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down server...")

	// WS connections are hijacked from the HTTP server once upgraded, so
	// Shutdown alone won't drain them — close them explicitly.
	ws.CloseAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("server forced to shutdown: ", err)
	}

	sqlDB.Close()
	logger.Log.Info("server exited")
}

// requestIDMiddleware reads the X-Request-ID header api-gateway already
// generates/forwards (or generates one, for requests that reach this
// service directly, e.g. the WebSocket upgrade), so every request can be
// correlated across services — this service had no such middleware before,
// and has no gin/other middleware stack to insert into, so it wraps the
// whole mux once instead of per-route.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		r.Header.Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func runSelfHealthcheck() int {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8083"
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
