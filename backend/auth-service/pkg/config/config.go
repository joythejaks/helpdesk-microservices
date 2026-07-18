package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppPort        string
	JWTSecret      string
	DBHost         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBPort         string
	InternalSecret string
	EnableSwagger  bool
	AuthRateLimitRPS   float64
	AuthRateLimitBurst float64
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:            os.Getenv("APP_PORT"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		DBHost:             os.Getenv("DB_HOST"),
		DBUser:             os.Getenv("DB_USER"),
		DBPassword:         os.Getenv("DB_PASSWORD"),
		DBName:             os.Getenv("DB_NAME"),
		DBPort:             os.Getenv("DB_PORT"),
		InternalSecret:     os.Getenv("INTERNAL_SHARED_SECRET"),
		EnableSwagger:      os.Getenv("ENABLE_SWAGGER") == "true",
		AuthRateLimitRPS:   parseFloatOrDefault(os.Getenv("AUTH_RATE_LIMIT_RPS"), 5),
		AuthRateLimitBurst: parseFloatOrDefault(os.Getenv("AUTH_RATE_LIMIT_BURST"), 10),
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

	if AppConfig.InternalSecret == "" {
		log.Fatal("INTERNAL_SHARED_SECRET is required")
	}
}

func parseFloatOrDefault(raw string, def float64) float64 {
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
