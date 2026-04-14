package main

import (
	"auth-service/internal/delivery/http"
	"auth-service/internal/domain"
	"auth-service/internal/repository"
	"auth-service/internal/usecase"
	"auth-service/pkg/config"
	"auth-service/pkg/logger"
	"auth-service/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.Load()

	port := config.AppConfig.AppPort
	jwtSecret := []byte(config.AppConfig.JWTSecret)

	logger.Init("auth-service")

	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Log.Fatal("failed to connect database:", err)
	}
	db.AutoMigrate(&domain.User{})

	repo := repository.NewUserRepository(db)
	usecase := usecase.NewAuthUsecase(repo)

	handler := http.NewAuthHandler(usecase, jwtSecret)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, "ok")
	})

	http.RegisterRoutes(r, handler)

	logger.Log.Info("auth-service running on port " + port)
	r.Run(":" + port)
}
