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

	// 🔥 start consumer (non-blocking)
	consumer.StartConsumer(rabbitURL)

	// websocket
	http.HandleFunc("/ws", ws.HandleConnections)

	go ws.HandleMessages()

	// ✅ health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	logger.Log.Info("notification-service running on port " + port)

	http.ListenAndServe(":"+port, nil)
}
