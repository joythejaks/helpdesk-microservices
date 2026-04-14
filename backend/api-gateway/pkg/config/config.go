package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort          string
	JWTSecret        string
	AuthServiceURL   string
	TicketServiceURL string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:          os.Getenv("APP_PORT"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AuthServiceURL:   os.Getenv("AUTH_SERVICE_URL"),
		TicketServiceURL: os.Getenv("TICKET_SERVICE_URL"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
}
