package main

import (
	"net/http"

	"notification-service/internal/consumer"
	"notification-service/internal/delivery/ws"
	"notification-service/pkg/config"
	"notification-service/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.Load()
	logger.Init("notification-service")

	port := config.AppConfig.AppPort
	rabbitURL := config.AppConfig.RabbitMQURL

	// start consumer (non-blocking, auto-reconnect)
	consumer.StartConsumer(rabbitURL)

	// custom mux — hindari register ke DefaultServeMux global
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", ws.HandleConnections)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	go ws.HandleMessages()

	logger.Log.Info("notification-service running on port " + port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Log.Fatal("server error: ", err)
	}
}
