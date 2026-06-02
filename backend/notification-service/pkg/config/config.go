package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort     string
	RabbitMQURL string
	JWTSecret   string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:     os.Getenv("APP_PORT"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
}
