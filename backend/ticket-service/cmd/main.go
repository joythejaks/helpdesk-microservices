package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	delivery "ticket-service/internal/delivery/http"
	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/migrations"
	"ticket-service/internal/repository"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/config"
	"ticket-service/pkg/logger"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
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
	logger.Init("ticket-service")

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

	if err := migrations.Run(sqlDB, "tickets"); err != nil {
		logger.Log.Fatal("failed to run migrations:", err)
	}

	// repo & usecase
	repo := repository.NewTicketRepository(db)
	ticketUsecase := usecase.NewTicketUsecase(repo)

	reportRepo := repository.NewReportRepository(db)
	reportUsecase := usecase.NewReportUsecase(reportRepo)

	commentRepo := repository.NewCommentRepository(db)
	commentUsecase := usecase.NewCommentUsecase(commentRepo, ticketUsecase)

	attachmentRepo := repository.NewAttachmentRepository(db)
	attachmentUsecase := usecase.NewAttachmentUsecase(attachmentRepo, ticketUsecase)

	// RabbitMQ
	publisher, err := messaging.NewPublisher(rabbitURL)
	if err != nil {
		logger.Log.Warn("rabbitmq not ready:", err)
	}

	handler := delivery.NewTicketHandler(ticketUsecase, publisher)
	reportHandler := delivery.NewReportHandler(reportUsecase)
	commentHandler := delivery.NewCommentHandler(commentUsecase, publisher)
	attachmentHandler := delivery.NewAttachmentHandler(attachmentUsecase, publisher)

	r := gin.Default()

	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// health — intentionally outside the internal-secret gate so the
	// container's own HEALTHCHECK (calling itself over localhost) still works.
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if sqlDB.PingContext(ctx) != nil {
			response.Error(c, http.StatusServiceUnavailable, "database disconnected", "unavailable")
			return
		}
		response.Success(c, "ok")
	})

	// Business routes only reachable via the API gateway (proven by
	// X-Internal-Secret) — closes off calling this service directly and
	// spoofing X-User-ID/X-User-ROLE.
	internalOnly := r.Group("/")
	internalOnly.Use(delivery.InternalOnlyMiddleware(config.AppConfig.InternalSecret))
	{
		internalOnly.POST("/tickets", handler.Create)
		internalOnly.GET("/tickets", handler.GetTickets)
		internalOnly.GET("/tickets/:id", handler.GetByID)
		internalOnly.GET("/tickets/:id/history", handler.GetHistory)
		internalOnly.PATCH("/tickets/:id/assign", handler.Assign)
		internalOnly.PATCH("/tickets/:id/status", handler.UpdateStatus)
		internalOnly.POST("/tickets/:id/comments", commentHandler.Create)
		internalOnly.GET("/tickets/:id/comments", commentHandler.List)
		internalOnly.POST("/tickets/:id/attachments", attachmentHandler.Create)
		internalOnly.GET("/tickets/:id/attachments", attachmentHandler.List)
		internalOnly.GET("/tickets/:id/attachments/:attachmentId", attachmentHandler.Download)

		internalOnly.GET("/reports/summary", reportHandler.Summary)
		internalOnly.GET("/reports/agents", reportHandler.AgentPerformance)
		internalOnly.GET("/reports/critical-trends", reportHandler.CriticalTrend)
		internalOnly.GET("/reports/queue-size", reportHandler.QueueSize)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		logger.Log.Info("ticket-service running on port " + port)
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

	sqlDB.Close()
	logger.Log.Info("server exited")
}

func runSelfHealthcheck() int {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8082"
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
