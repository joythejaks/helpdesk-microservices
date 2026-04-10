package main

import (
	"log"
	"os"

	"auth-service/internal/delivery/http"

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

	r := gin.Default()

	http.RegisterRoutes(r)

	r.Run(":" + port)
}
