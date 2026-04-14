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

	// 🔥 TAMBAH refresh token migration
	db.AutoMigrate(&domain.User{}, &domain.RefreshToken{})

	repo := repository.NewUserRepository(db)
	usecase := usecase.NewAuthUsecase(repo)

	// 🔥 TAMBAH refresh repo
	refreshRepo := repository.NewRefreshTokenRepository(db)

	// 🔥 UPDATE constructor
	handler := http.NewAuthHandler(usecase, refreshRepo, jwtSecret)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, "ok")
	})

	http.RegisterRoutes(r, handler)

	logger.Log.Info("auth-service running on port " + port)
	r.Run(":" + port)
}
