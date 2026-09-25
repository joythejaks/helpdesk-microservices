package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppPort              string
	RabbitMQURL          string
	RabbitQueueType      string
	DBHost               string
	DBUser               string
	DBPassword           string
	DBName               string
	DBPort               string
	InternalSecret       string
	TicketRateLimitRPS   float64
	TicketRateLimitBurst float64
	RedisURL             string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:              os.Getenv("APP_PORT"),
		RabbitMQURL:          os.Getenv("RABBITMQ_URL"),
		RabbitQueueType:      queueTypeOrDefault(os.Getenv("RABBITMQ_QUEUE_TYPE")),
		DBHost:               os.Getenv("DB_HOST"),
		DBUser:               os.Getenv("DB_USER"),
		DBPassword:           os.Getenv("DB_PASSWORD"),
		DBName:               os.Getenv("DB_NAME"),
		DBPort:               os.Getenv("DB_PORT"),
		InternalSecret:       os.Getenv("INTERNAL_SHARED_SECRET"),
		TicketRateLimitRPS:   parseFloatOrDefault(os.Getenv("TICKET_RATE_LIMIT_RPS"), 5),
		TicketRateLimitBurst: parseFloatOrDefault(os.Getenv("TICKET_RATE_LIMIT_BURST"), 10),
		RedisURL:             os.Getenv("REDIS_URL"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.DBHost == "" {
		log.Fatal("DB_HOST is required")
	}

	if AppConfig.InternalSecret == "" {
		log.Fatal("INTERNAL_SHARED_SECRET is required")
	}
}

// queueTypeOrDefault defaults to classic, which is what existing queues (and
// docker compose) use; quorum is opted into with RABBITMQ_QUEUE_TYPE=quorum.
// Validation happens in main so config stays free of the messaging package.
func queueTypeOrDefault(raw string) string {
	if raw == "" {
		return "classic"
	}
	return raw
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
