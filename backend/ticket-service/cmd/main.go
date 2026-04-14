package main

import (
	"log"
	"os"

	"ticket-service/internal/delivery/http"
	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/repository"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	port := os.Getenv("APP_PORT")
	rabbitURL := os.Getenv("RABBITMQ_URL")

	// 🔥 INIT LOGGER
	logger.Init("ticket-service")

	// DB
	db, err := repository.NewPostgresDB()
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&domain.Ticket{})

	// repo & usecase
	repo := repository.NewTicketRepository(db)
	usecase := usecase.NewTicketUsecase(repo)

	// RabbitMQ
	publisher, _ := messaging.NewPublisher(rabbitURL)

	handler := http.NewTicketHandler(usecase, publisher)

	r := gin.Default()

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// endpoint
	r.POST("/tickets", handler.Create)

	log.Println("🚀 Ticket service running on", port)
	r.Run(":" + port)
}
