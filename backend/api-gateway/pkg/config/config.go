package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort          string
	JWTSecret        string
	AuthServiceURL   string
	TicketServiceURL string
	AllowedOrigins   []string
	RateLimitRPS     float64
	RateLimitBurst   float64
	InternalSecret   string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:          os.Getenv("APP_PORT"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AuthServiceURL:   os.Getenv("AUTH_SERVICE_URL"),
		TicketServiceURL: os.Getenv("TICKET_SERVICE_URL"),
		AllowedOrigins:   parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
		RateLimitRPS:     parseFloatOrDefault(os.Getenv("RATE_LIMIT_RPS"), 10),
		RateLimitBurst:   parseFloatOrDefault(os.Getenv("RATE_LIMIT_BURST"), 20),
		InternalSecret:   os.Getenv("INTERNAL_SHARED_SECRET"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if AppConfig.InternalSecret == "" {
		log.Fatal("INTERNAL_SHARED_SECRET is required")
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
