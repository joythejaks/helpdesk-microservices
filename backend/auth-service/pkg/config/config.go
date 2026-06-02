package config

import (
	"log"
	"os"
)

type Config struct {
	AppPort    string
	JWTSecret  string
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:    os.Getenv("APP_PORT"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		DBHost:     os.Getenv("DB_HOST"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBPort:     os.Getenv("DB_PORT"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if AppConfig.DBHost == "" {
		log.Fatal("DB_HOST is required")
	}
}
