package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort     string
	RabbitMQURL string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:     os.Getenv("APP_PORT"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}
}
