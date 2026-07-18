package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/consumer"
	"notification-service/internal/delivery/ws"
	"notification-service/pkg/config"
	"notification-service/pkg/logger"

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

	ws.Init([]byte(config.AppConfig.JWTSecret), config.AppConfig.AllowedOrigins, config.AppConfig.MaxWSConnections)

	// start consumer (non-blocking, auto-reconnect)
	consumer.StartConsumer(rabbitURL)

	// custom mux — hindari register ke DefaultServeMux global
	mux := http.NewServeMux()

	wsLimiter := ws.NewRateLimiter(config.AppConfig.WSRateLimitRPS, config.AppConfig.WSRateLimitBurst)
	mux.HandleFunc("/ws", ws.RateLimit(wsLimiter, ws.HandleConnections))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !consumer.IsConnected() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("rabbitmq disconnected"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
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
