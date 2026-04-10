package main

import (
	"log"
	"os"

	"auth-service/internal/delivery/http"
	"auth-service/internal/domain"
	"auth-service/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	// ✅ CONNECT DB DULU
	db, err := repository.NewPostgresDB()
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	log.Println("Database connected")

	// ✅ MIGRATION
	err = db.AutoMigrate(&domain.User{})
	if err != nil {
		log.Fatal("migration failed:", err)
	}

	log.Println("Migration done")

	// baru start server
	r := gin.Default()
	http.RegisterRoutes(r)

	r.Run(":" + port)
}
