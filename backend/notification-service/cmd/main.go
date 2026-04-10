package main

import (
	"log"
	"net/http"
	"os"

	"notification-service/internal/consumer"
	"notification-service/internal/delivery/ws"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	port := os.Getenv("APP_PORT")
	rabbitURL := os.Getenv("RABBITMQ_URL")

	// 🔥 start consumer TANPA crash
	consumer.StartConsumer(rabbitURL)

	// websocket
	http.HandleFunc("/ws", ws.HandleConnections)

	go ws.HandleMessages()

	log.Println("🚀 Notification service running on", port)
	http.ListenAndServe(":"+port, nil)
}
