package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort     string
	RabbitMQURL string
	DBHost      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBPort      string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:     os.Getenv("APP_PORT"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
		DBHost:      os.Getenv("DB_HOST"),
		DBUser:      os.Getenv("DB_USER"),
		DBPassword:  os.Getenv("DB_PASSWORD"),
		DBName:      os.Getenv("DB_NAME"),
		DBPort:      os.Getenv("DB_PORT"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.DBHost == "" {
		log.Fatal("DB_HOST is required")
	}
}
