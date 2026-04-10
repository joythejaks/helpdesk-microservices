package main

import (
	"log"
	"os"

	"ticket-service/internal/delivery/http"
	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/repository"
	"ticket-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	port := os.Getenv("APP_PORT")
	rabbitURL := os.Getenv("RABBITMQ_URL")

	// DB
	db, err := repository.NewPostgresDB()
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&domain.Ticket{})

	// repo & usecase
	repo := repository.NewTicketRepository(db)
	usecase := usecase.NewTicketUsecase(repo)

	// 🔥 RabbitMQ (NO CRASH)
	publisher, _ := messaging.NewPublisher(rabbitURL)

	handler := http.NewTicketHandler(usecase, publisher)

	r := gin.Default()

	r.POST("/tickets", handler.Create)

	log.Println("🚀 Ticket service running on", port)
	r.Run(":" + port)
}
