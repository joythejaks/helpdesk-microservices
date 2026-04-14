package main

import (
	"ticket-service/internal/delivery/http"
	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/repository"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/config"
	"ticket-service/pkg/logger"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.Load()
	logger.Init("ticket-service")

	port := config.AppConfig.AppPort
	rabbitURL := config.AppConfig.RabbitMQURL

	// DB
	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Log.Fatal("failed to connect DB:", err)
	}

	db.AutoMigrate(&domain.Ticket{})

	// repo & usecase
	repo := repository.NewTicketRepository(db)
	usecase := usecase.NewTicketUsecase(repo)

	// RabbitMQ
	publisher, err := messaging.NewPublisher(rabbitURL)
	if err != nil {
		logger.Log.Warn("rabbitmq not ready:", err)
	}

	handler := http.NewTicketHandler(usecase, publisher)

	r := gin.Default()

	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// health
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, "ok")
	})

	// endpoint
	r.POST("/tickets", handler.Create)
	r.GET("/tickets", handler.GetTickets)
	r.GET("/tickets/:id", handler.GetByID)

	logger.Log.Info("ticket-service running on port " + port)

	r.Run(":" + port)
}
