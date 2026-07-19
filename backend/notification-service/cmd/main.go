package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"notification-service/internal/consumer"
	delivery "notification-service/internal/delivery/http"
	"notification-service/internal/delivery/ws"
	"notification-service/internal/domain"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"
	"notification-service/pkg/config"
	"notification-service/pkg/logger"
	"notification-service/pkg/response"

	"github.com/joho/godotenv"
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
	db.AutoMigrate(&domain.Notification{})

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

	// REST notifications API — self-authenticated (Authorization: Bearer),
	// same trust model as /ws since this service was never put behind the
	// gateway (the reverse proxy doesn't handle WS upgrades).
	mux.HandleFunc("GET /notifications", notificationHandler.List)
	mux.HandleFunc("PATCH /notifications/read-all", notificationHandler.MarkAllRead)
	mux.HandleFunc("PATCH /notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(r.PathValue("id"), 10, 0)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid notification id", "BAD_REQUEST")
			return
		}
		notificationHandler.MarkRead(w, r, uint(id))
	})

	go ws.HandleMessages()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
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
