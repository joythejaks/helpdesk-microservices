package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort          string
	RabbitMQURL      string
	JWTSecret        string
	AllowedOrigins   []string
	WSRateLimitRPS   float64
	WSRateLimitBurst float64
	MaxWSConnections int
	DBHost           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPort           string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:          os.Getenv("APP_PORT"),
		RabbitMQURL:      os.Getenv("RABBITMQ_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AllowedOrigins:   parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
		WSRateLimitRPS:   parseFloatOrDefault(os.Getenv("WS_RATE_LIMIT_RPS"), 5),
		WSRateLimitBurst: parseFloatOrDefault(os.Getenv("WS_RATE_LIMIT_BURST"), 10),
		MaxWSConnections: parseIntOrDefault(os.Getenv("MAX_WS_CONNECTIONS"), 1000),
		DBHost:           os.Getenv("DB_HOST"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBPort:           os.Getenv("DB_PORT"),
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

// parseOrigins splits a comma-separated ALLOWED_ORIGINS value. Falls back to
// "*" (with a warning) so local/dev setups keep working without extra config.
func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		log.Println("WARNING: ALLOWED_ORIGINS not set, defaulting to '*' (not safe for production)")
		return []string{"*"}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	return origins
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

func parseIntOrDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
