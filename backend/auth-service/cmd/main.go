package main

import (
	"log"
	"os"

	"auth-service/internal/delivery/http"
	"auth-service/internal/domain"
	"auth-service/internal/repository"
	"auth-service/internal/usecase"
	"auth-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET required")
	}

	logger.Init("auth-service")

	port := os.Getenv("APP_PORT")
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	db, _ := repository.NewPostgresDB()
	db.AutoMigrate(&domain.User{})

	repo := repository.NewUserRepository(db)
	usecase := usecase.NewAuthUsecase(repo)

	handler := http.NewAuthHandler(usecase, jwtSecret)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	http.RegisterRoutes(r, handler)

	r.Run(":" + port)
}
